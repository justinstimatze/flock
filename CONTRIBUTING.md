# Contributing to flock

Thanks for wanting to make the geese louder.

## Dev setup

flock is pure Go with two small dependencies. Build and run from a clone:

```sh
git clone https://github.com/justinstimatze/flock && cd flock
go build -o flock . && ./flock          # Ctrl-C to stop
```

You need to be in the `input` group (see the README) and have a PipeWire/Pulse
audio server running.

## Checks (CI runs these on every push/PR)

```sh
gofmt -l .        # must print nothing
go vet ./...
go build ./...
```

## Adding honks

- Drop `.wav` files into `honks/<pack>/` (16-bit PCM mono/stereo; ffmpeg writes
  this by default).
- Prefer clips of similar length and volume — a flock should be even.
- The built-in packs are compiled into the binary with `go:embed`, so **rebuild**
  (`go build -o flock .`) to pick up changes. (You can also point flock at any
  folder without rebuilding: `./flock ~/my-honks/`.)
- Regenerate the bundled packs with `tools/slice_goose.sh` (real, from the
  committed CC0 source) or `tools/generate_honks.sh` (synth).
- Only contribute audio you have the right to share. CC0 / public domain is
  ideal; record the source and license in `CREDITS.md`.

## Hands-free testing

flock reads `/dev/input`, so injected keystrokes must originate at the kernel
layer to be seen. Inject with `evemu` or `ydotool` (both use `uinput`), **not**
`xdotool`/`wtype` (those sit above evdev and won't trigger it):

```sh
sudo ydotool key 30:1 30:0     # 'a' down then up
```

## Pull requests

Branch off `main`, keep changes focused, and make sure CI is green. Be a silly
goose, but a tidy one.
