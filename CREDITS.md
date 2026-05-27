# flock — credits & licensing

A flock of geese on every keypress, system-wide, via the Linux evdev layer.

## Sound source — `honks/goose/` (the real flock)
- **"Low flyover by five Canadian Geese"** by *Extemporalist*, via Wikimedia Commons.
- **License: CC0 1.0 Universal** (Public Domain Dedication). No attribution
  required; recorded here for provenance only.
- Source page: https://commons.wikimedia.org/wiki/File:Low_flyover_by_five_Canadian_Geese.flac
- The original recording is committed at `assets/geese-cc0.flac`.
- Processing (`tools/slice_goose.sh`, reproducible from the committed source):
  band-limited 250 Hz–5 kHz to drop wind rumble, gained up from a very quiet
  recording, and sliced from the honking flurry into 10 × 0.70 s clips, each
  peak-normalized to −1 dBFS for consistent length + volume.

## Sound source — `honks/goose-synth/` (offline fallback)
- 100% synthesized with ffmpeg (`tools/generate_honks.sh`). No third-party
  assets, no license constraints. A cartoonish honk, not a real goose.

## Code — `main.go`, `wav.go`, `tools/*.sh`
- Original code, MIT-licensed (see [LICENSE](LICENSE)). Nothing lifted from third
  parties. We deliberately did **not** fork the existing wheels:
  - **wayvibes** — no license (all rights reserved): usable as-is, not forkable.
  - **quackboard** — no license: same.
  - **bucklespring** — GPL-2.0 (forkable) but fixed-one-sound-per-key, needing a
    C patch to randomize.
  A small Go evdev listener was less work *and* fully license-clean, does true
  random-per-press variety, and — via `golang.org/x/sys/unix` — is arch-correct
  where the original Python prototype's hardcoded struct/ioctl layouts were not.

## Dependencies
- [`github.com/jfreymuth/pulse`](https://github.com/jfreymuth/pulse) (MIT) —
  pure-Go PulseAudio/PipeWire client (no CGO, keeps the binary static).
- [`golang.org/x/sys`](https://pkg.go.dev/golang.org/x/sys) (BSD-3-Clause) —
  arch-correct syscalls and types.
