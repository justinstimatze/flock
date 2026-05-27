# 🪿 flock

```
              🪿
            🪿
          🪿       🪿
        🪿            🪿        a flock of geese honks on every keypress
          🪿       🪿
            🪿
              🪿
```

![code: MIT](https://img.shields.io/badge/code-MIT-blue)
![sounds: CC0](https://img.shields.io/badge/sounds-CC0-green)
![habitat: Linux (Wayland · X11 · TTY)](https://img.shields.io/badge/habitat-Wayland%20%C2%B7%20X11%20%C2%B7%20TTY-orange)
![CI](https://github.com/justinstimatze/flock/actions/workflows/ci.yml/badge.svg)

> *We join our subject at dawn, in the dim glow of an idle terminal. It has not
> yet pressed a key. It does not yet know what it has done.* — [narrator, hushed]

`flock` is a small wild animal that lives in your Linux machine and **honks every
time you press a key** — in every application, all at once, indefinitely (or
until you `Ctrl-C`). Press one key and a single goose calls. Type a sentence and
a whole flock answers, in overlapping, never-quite-repeating honks. It is a
single static Go binary with the honks living inside it.

## Field Notes — why it honks *everywhere*

The Keyboard Goose (*Anser clavis*) nests **beneath the display server**, at the
kernel's input layer (`/dev/input/event*`). Every keystroke must wade past it
before reaching any application — which is why no window, browser, or compositor
can shoo it away. It is *also* why you cannot fake a keystroke to it from above:
a browser or `xdotool` calls out from the wrong altitude, and the goose,
unbothered, simply does not answer. (Most rival species hook the desktop session
instead, and so fall silent on a backgrounded Wayland window. Not this one.)

## Habitat & Range

A hardy bird, thriving across the Linux wetlands:

- **Range:** Wayland, X11, and the bare TTY alike — it nests below all three.
- **Forage:** Go 1.23+ to build the binary (pure Go, only two small deps; no CGO).
- **Watering hole:** a **PulseAudio or PipeWire** server for sound. Stock Ubuntu
  and Fedora *desktops* ship one (`pipewire-pulse`) by default; minimal/headless
  installs need `pipewire-pulse` (or `pulseaudio`) added. flock is silent without one.
- **Banding:** you must be in the `input` group to observe it — no root; geese
  distrust authority:
  ```sh
  sudo usermod -aG input "$USER"   # then log out and back in
  ```
  This is a real, permanent grant — see [SECURITY.md](SECURITY.md) before you do it.

## Attracting a Flock — install

Lure the species to your machine and raise one:
```sh
git clone https://github.com/justinstimatze/flock
cd flock
go build -o flock .
./flock                # Ctrl-C to release it
```

Establish a permanent breeding population (a systemd *user* service — no root):
```sh
./install.sh               # builds, installs the binary, returns at every login
./install.sh --uninstall   # gently relocates the flock elsewhere
```

> The installed service is sandboxed (no network, read-only `$HOME`, restricted
> syscalls). That intentionally blocks **remote/TCP audio sinks** — flock plays
> only to your local PipeWire/Pulse socket.

## Observed Behavior — usage

```sh
flock [PACK]
```

| Incantation | Resulting wildlife |
|---|---|
| `./flock` | the wild Canada-goose flock (default) |
| `./flock goose-synth` | a captive-bred synthetic flock |
| `./flock ~/my-honks/` | a flock of your own raising (any folder of `.wav`s) |
| `HONK_DEBUG=1 ./flock` | the goose announces each call by name |

It honks only on **key-down** (a held key will not stampede), and a hard cap of
**12 simultaneous geese** guarantees that even the fastest typist summons a
flock — never a biblical plague. New keyboards plugged in mid-session are adopted
automatically; unplugged ones are released without a fuss.

## Plumage Variation — your own packs

The built-in packs are baked into the binary (`go:embed`). To raise a new
*built-in* subspecies, drop `.wav` files into `honks/<name>/` and rebuild
(`go build -o flock .`). Or skip the rebuild entirely and point flock at any
folder: `./flock ~/my-honks/`. A random clip sounds per press (never the same
twice in a row), so even a handful of honks reads as a full, restless flock.

## Tracking the flock — testing

Quickest check that audio works — no `input` group or key injection needed:
```sh
./flock --play goose   # plays every honk in the pack once, then exits
```

To exercise the live key-triggered path hands-free, you must inject keystrokes at
the *kernel* layer — with `evemu` or `ydotool` (both use `uinput`). Tools that
inject higher up the stack (`xdotool`, `wtype`, a browser) won't trigger it, for
the same reason it honks in every app: it sits underneath them all.

## Diet

The Keyboard Goose feeds exclusively on **keystrokes**, which pass through it
constantly. Crucially, **it does not hoard them.** It tastes only whether an
event was a *press* — enough to decide whether to honk — and forgets everything
else at once. It opens no files, makes no network connections (the service
enforces this at the kernel level), and never demands root. It is a honking
machine, not a keylogger; see [SECURITY.md](SECURITY.md), and read `main.go`
yourself (~340 lines, plus a small WAV decoder).

## Related Species — prior art & influences

The Keyboard Goose did not evolve in isolation. Its nearest relatives:

- **[bucklespring](https://github.com/zevv/bucklespring)** (GPL-2.0) — the
  ancestral form: an evdev daemon calling a fixed sound *per key*. flock shares
  its skeleton but calls *at random* (geese are not so predictable).
- **[wayvibes](https://github.com/sahaj-b/wayvibes)** — the closest living
  Wayland-native cousin (libevdev + Mechvibes packs). We grew our own listener
  rather than domesticate it (it carries no license tag).
- **[Mechvibes](https://mechvibes.com)** — a widespread soundpack species; its
  community **GOOSE** plumage is this joke's most direct ancestor. (iohook-bound,
  so it loses its voice in the Wayland background — the very gap flock fills.)
- **[Klack](https://tryklack.com)** (macOS) — bearer of the *variety* gene: it
  randomizes pitch so no two calls match. flock expresses the same trait via a
  honk **pool**.
- **[QuackBoard](https://github.com/jlam55555/quackboard)** — a duck. Close, but
  a duck.
- **Untitled Goose Game** (House House) — the celebrity of the clade. Its calls
  are copyrighted and **appear nowhere here**; flock honks with wild,
  public-domain geese only.

(`rustyvibes` and `MechVibes` were also observed, but they call from the
X11/iohook canopy and fall silent on Wayland. flock's evdev burrow does not.)

## Conservation & Provenance — credits + license

- **Wild goose audio** (`honks/goose/`) was recorded from *"Low flyover by five
  Canadian Geese"* by Extemporalist
  ([Wikimedia Commons](https://commons.wikimedia.org/wiki/File:Low_flyover_by_five_Canadian_Geese.flac),
  **CC0 1.0 / public domain**) — band-limited, normalized, and cut into 10 clips.
  See [`tools/slice_goose.sh`](tools/slice_goose.sh) and [CREDITS.md](CREDITS.md).
- **Code:** MIT — [LICENSE](LICENSE). **Sounds:** CC0. Honk freely.

<details>
<summary>📼 Unabridged field recording (4.2 hours, abridged — turn your volume down)</summary>

honk honk honk honk honk honk honk honk honk honk honk honk honk honk honk honk
honk honk honk honk honk honk honk honk honk honk honk honk honk honk honk honk
honk honk honk honk honk honk honk honk **HONK** honk honk honk honk honk honk
honk honk honk honk honk honk honk honk honk honk honk honk honk honk honk honk
honk honk honk honk honk honk honk honk honk honk honk honk honk honk honk honk
honk honk honk honk honk honk honk *(a distant honk)* honk honk honk honk honk

</details>

---

> Your coworkers will not understand. Your keyboard will sound like a wetland at
> dawn. Install it anyway, and **be a silly goose.** 🪿
