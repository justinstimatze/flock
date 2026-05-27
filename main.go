// flock — a flock of geese honks on every keypress, system-wide, via raw evdev.
//
// It reads /dev/input/event* (the kernel input layer, below the display server),
// so it honks in every app on Wayland, X11 and the bare TTY alike — and is why
// no single app can fake or suppress it. Audio is played through PipeWire/Pulse
// in pure Go (no paplay subprocess). You must be in the `input` group; no root.
package main

import (
	"embed"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/jfreymuth/pulse"
	"golang.org/x/sys/unix"
)

//go:embed all:honks
var honksFS embed.FS

const (
	evKey     = 0x01 // EV_KEY
	keyDown   = 1    // value: 0=up, 1=down, 2=autorepeat -> honk on the press only
	keyA      = 30
	keyZ      = 44
	keySpace  = 57
	keyMax    = 0x2ff
	maxVoices = 12 // concurrency cap so fast typing can't run away
)

// inputEvent mirrors the kernel struct input_event. unix.Timeval is arch-correct
// per platform, so this layout is right on 64-bit, 32-bit, and y2038 builds alike.
type inputEvent struct {
	Time  unix.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

var evSize = int(unsafe.Sizeof(inputEvent{}))

// EVIOCGBIT(EV_KEY): asm-generic _IOC encoding (correct on amd64/arm64/riscv64
// and every other asm-generic arch; the legacy alpha/mips/ppc/sparc layouts are
// unsupported — desktops don't run there).
const (
	iocTypeShift = 8
	iocSizeShift = iocTypeShift + 8
	iocDirShift  = iocSizeShift + 14
	iocRead      = 2
)

func eviocgbit(ev, length uintptr) uintptr {
	return (iocRead << iocDirShift) | (length << iocSizeShift) | (uintptr('E') << iocTypeShift) | (0x20 + ev)
}

func testBit(bits []byte, n int) bool {
	return n/8 < len(bits) && bits[n/8]&(1<<(uint(n)%8)) != 0
}

// isKeyboard returns true only for a real typing keyboard — it must expose the
// alphabetic block and the spacebar. This rejects power buttons, lid switches,
// and vendor hotkey pseudo-devices that would otherwise double-honk every press.
func isKeyboard(fd int) bool {
	bits := make([]byte, keyMax/8+1)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd),
		eviocgbit(evKey, uintptr(len(bits))), uintptr(unsafe.Pointer(&bits[0])))
	if errno != 0 {
		return false
	}
	return testBit(bits, keyA) && testBit(bits, keyZ) && testBit(bits, keySpace)
}

func discoverKeyboards() map[string]int {
	found := map[string]int{}
	paths, _ := filepath.Glob("/dev/input/event*")
	for _, p := range paths {
		fd, err := unix.Open(p, unix.O_RDONLY, 0)
		if err != nil {
			continue
		}
		if isKeyboard(fd) {
			found[p] = fd
		} else {
			unix.Close(fd)
		}
	}
	return found
}

// watch reads one keyboard, forwarding key-down presses, until the device goes
// away (unplug). Reading into a []inputEvent backing array keeps the bytes both
// aligned and native-endian, so no manual struct unpacking is needed.
func watch(fd int, p string, presses chan<- struct{}, gone chan<- string) {
	events := make([]inputEvent, 64)
	raw := unsafe.Slice((*byte)(unsafe.Pointer(&events[0])), len(events)*evSize)
	for {
		n, err := unix.Read(fd, raw)
		if err == unix.EINTR {
			continue
		}
		if err != nil || n <= 0 {
			break
		}
		for i := 0; i < n/evSize; i++ {
			if events[i].Type == evKey && events[i].Value == keyDown {
				presses <- struct{}{}
			}
		}
	}
	unix.Close(fd)
	gone <- p
}

type honk struct {
	name     string
	samples  []int16
	rate     int
	channels int
}

// loadPack loads a sound pack by name (embedded in the binary) or from a
// directory path on disk (for custom packs without rebuilding).
func loadPack(name string) ([]honk, error) {
	var names []string
	var read func(string) ([]byte, error)

	if st, err := os.Stat(name); err == nil && st.IsDir() {
		entries, _ := os.ReadDir(name)
		for _, e := range entries {
			names = append(names, e.Name())
		}
		read = func(f string) ([]byte, error) { return os.ReadFile(filepath.Join(name, f)) }
	} else {
		dir := path.Join("honks", name)
		entries, err := honksFS.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("unknown pack %q", name)
		}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		read = func(f string) ([]byte, error) { return honksFS.ReadFile(path.Join(dir, f)) }
	}

	var honks []honk
	for _, n := range names {
		if !strings.HasSuffix(n, ".wav") {
			continue
		}
		b, err := read(n)
		if err != nil {
			continue
		}
		s, rate, ch, err := decodeWAV(b)
		if err != nil {
			continue
		}
		honks = append(honks, honk{name: n, samples: s, rate: rate, channels: ch})
	}
	sort.Slice(honks, func(i, j int) bool { return honks[i].name < honks[j].name })
	if len(honks) == 0 {
		return nil, fmt.Errorf("no .wav files in pack %q", name)
	}
	return honks, nil
}

// newHonkStream builds a one-shot playback stream for a honk. The reader feeds
// the decoded PCM and returns EndOfData when exhausted, so the stream self-stops.
func newHonkStream(client *pulse.Client, h honk) (*pulse.PlaybackStream, error) {
	pos := 0
	reader := pulse.Int16Reader(func(out []int16) (int, error) {
		if pos >= len(h.samples) {
			return 0, pulse.EndOfData
		}
		n := copy(out, h.samples[pos:])
		pos += n
		return n, nil
	})
	// PlaybackLatency is required: without a buffer target, Start() blocks forever
	// waiting for the server to request data.
	opts := []pulse.PlaybackOption{
		pulse.PlaybackSampleRate(h.rate),
		pulse.PlaybackLatency(0.1),
		pulse.PlaybackMediaName("honk"),
	}
	if h.channels >= 2 {
		opts = append(opts, pulse.PlaybackStereo)
	} else {
		opts = append(opts, pulse.PlaybackMono)
	}
	return client.NewPlayback(reader, opts...)
}

func main() {
	args := os.Args[1:]
	play := false
	if len(args) > 0 && args[0] == "--play" {
		play = true // audio smoke test: play every honk once, then exit
		args = args[1:]
	}
	pack := "goose"
	if len(args) > 0 {
		pack = args[0]
	}
	honks, err := loadPack(pack)
	if err != nil {
		fmt.Fprintln(os.Stderr, "flock:", err)
		os.Exit(1)
	}

	client, err := pulse.NewClient(pulse.ClientApplicationName("flock"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "flock: cannot reach an audio server (PipeWire/PulseAudio):", err)
		os.Exit(1)
	}
	defer client.Close()

	if play {
		for _, h := range honks {
			fmt.Fprintln(os.Stderr, "flock: playing", h.name)
			s, err := newHonkStream(client, h)
			if err != nil {
				fmt.Fprintln(os.Stderr, "flock: playback error:", err)
				continue
			}
			s.Start()
			s.Drain()
			s.Close()
		}
		return
	}

	debug := os.Getenv("HONK_DEBUG") != ""
	presses := make(chan struct{}, 256)
	gone := make(chan string, 64)
	devices := map[string]int{}
	start := func(p string, fd int) { devices[p] = fd; go watch(fd, p, presses, gone) }

	for p, fd := range discoverKeyboards() {
		start(p, fd)
	}
	if len(devices) == 0 {
		fmt.Fprintln(os.Stderr, "flock: no keyboards found (are you in the 'input' group?)")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "flock: %d honks on %d keyboard(s). Ctrl-C to stop.\n", len(honks), len(devices))
	if debug {
		for p := range devices {
			fmt.Fprintln(os.Stderr, "flock: watching", p)
		}
	}

	// One goroutine owns the audio client and the live "voices", so there is zero
	// concurrent access to the Pulse connection. The server mixes the flock for us.
	type voice struct {
		stream *pulse.PlaybackStream
		end    time.Time
	}
	var voices []voice
	reap := func() {
		now, kept := time.Now(), voices[:0]
		for _, v := range voices {
			if now.After(v.end) {
				v.stream.Close()
			} else {
				kept = append(kept, v)
			}
		}
		voices = kept
	}

	last := -1
	cleanup := time.NewTicker(50 * time.Millisecond)
	rescan := time.NewTicker(3 * time.Second)
	defer cleanup.Stop()
	defer rescan.Stop()

	for {
		select {
		case <-presses:
			reap()
			if len(voices) >= maxVoices {
				continue // flock already at full throat
			}
			i := rand.Intn(len(honks))
			for len(honks) > 1 && i == last {
				i = rand.Intn(len(honks))
			}
			last = i
			h := honks[i]
			if debug {
				fmt.Fprintln(os.Stderr, "honk! ->", h.name)
			}
			stream, err := newHonkStream(client, h)
			if err != nil {
				if debug {
					fmt.Fprintln(os.Stderr, "flock: playback error:", err)
				}
				continue
			}
			stream.Start()
			frames := len(h.samples) / max(h.channels, 1)
			dur := time.Duration(frames) * time.Second / time.Duration(h.rate)
			voices = append(voices, voice{stream, time.Now().Add(dur + 150*time.Millisecond)})

		case <-cleanup.C:
			reap()

		case p := <-gone:
			delete(devices, p)
			if debug {
				fmt.Fprintln(os.Stderr, "flock:", p, "went away")
			}

		case <-rescan.C:
			for p, fd := range discoverKeyboards() {
				if _, ok := devices[p]; ok {
					unix.Close(fd) // already watching it
					continue
				}
				if debug {
					fmt.Fprintln(os.Stderr, "flock: new keyboard", p)
				}
				start(p, fd)
			}
		}
	}
}
