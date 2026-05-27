# Security Policy

## Reporting a vulnerability

Email **justin@justinstimatze.com**, or use GitHub's private vulnerability
reporting (the repository's Security tab). Please don't open a public issue for
security problems. I aim to acknowledge within a few days.

## What flock does — and does not do — with your keystrokes

flock reads raw keyboard events from `/dev/input/event*`, so every keypress
passes through the process. It is **not** a keylogger:

- It inspects only each event's **type** and **value** (was this a key *press*?)
  to decide whether to honk. The key's identity/scancode is read into a throwaway
  field and never used — and the honk is chosen at *random*, so the sound doesn't
  even encode which key you hit.
- It opens no files for writing and persists no keystroke data.
- It makes **no network connections**. The systemd unit sets
  `RestrictAddressFamilies=AF_UNIX`, which blocks creating any IP socket at the
  syscall level (seccomp) — kernel-enforced, and (unlike `PrivateNetwork`) it
  works in an unprivileged user service.
- Audio plays through the PipeWire/Pulse socket in-process — no subprocess, no
  shelling out.
- It runs as your normal user — **never root**.

All of this is verifiable in `main.go` (~340 lines) and `wav.go`.

## The `input` group is a real, permanent grant — know what you enable

Running flock requires adding your user to the `input` group:

```sh
sudo usermod -aG input "$USER"
```

This grants **every** process that user runs — not just flock — read access to
**all** `/dev/input` devices: system-wide raw keystrokes, for the life of the
account. It is **not** undone by uninstalling flock. To revoke it:

```sh
sudo gpasswd -d "$USER" input
```

Decide whether that trade-off is acceptable before installing. (`./install.sh
--uninstall` reminds you of this.)

## HONK_DEBUG

With `HONK_DEBUG` set, flock prints the chosen *sound-file name* (never the key)
to stderr, one line per press. Under the systemd service that goes to the journal
(a persistent file), so leave it off for normal use. Contributors: never add the
key code to that line.

## Scope

- Supported: the `main` branch / latest release.
- Out of scope: running as root, or dropping the `input`-group requirement — both
  are intentional design constraints, not vulnerabilities.
