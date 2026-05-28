#!/usr/bin/env bash
# slice_goose.sh — (re)generate honks/goose/ from the committed CC0 source.
#
# Fully reproducible from a clean clone: reads assets/geese-cc0.flac, band-limits
# to the goose band, spectrally subtracts the steady wind (profiled from the
# wind-only lead-in before the flurry), brings the (very quiet) recording up to a
# usable level, then cuts ten hand-picked clips from the strong middle of the
# flurry, de-clicked and peak-normalized. Requires ffmpeg.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$REPO/assets/geese-cc0.flac"
OUT="$REPO/honks/goose"
LEN=0.70       # clip length (a snappy honk per keypress)
# Hand-picked clip starts (seconds), chosen as the ten loudest, most
# honk-dominated 0.70s windows in the dense flurry (~13.3-16.8s). The grid
# approach pulled later clips from the weakening tail (>17s) where the geese
# settle and only low-frequency handling thumps remain; these explicit starts
# stay in the strong region, keeping the per-clip loudness spread to ~3 dB.
STARTS=(13.30 13.70 14.00 14.30 14.60 15.00 15.50 16.10 16.50 16.80)

[ -f "$SRC" ] || { echo "slice_goose: missing $SRC" >&2; exit 1; }
mkdir -p "$OUT"
rm -f "$OUT"/*.wav

maxvol() { # read max_volume (dB) from an ffmpeg volumedetect pass
	ffmpeg -hide_banner -nostats -i "$1" -af volumedetect -f null - 2>&1 \
		| awk '/max_volume:/ {print $(NF-1)}'
}

# 1. band-limit to the goose band, then gain the peak up to -1 dBFS
band="$(mktemp --suffix=.wav)"
trap 'rm -f "$band"' EXIT
pk=$(ffmpeg -hide_banner -nostats -i "$SRC" \
	-af "highpass=f=300,lowpass=f=5000,volumedetect" -f null - 2>&1 \
	| awk '/max_volume:/ {print $(NF-1)}')
gain=$(python3 -c "print(f'{-1.0-($pk):.1f}')")
# After band-limiting and gain, spectrally subtract the steady wind: profile it
# from the wind-only lead-in (2-6s, before the flurry at ~13s), then afftdn
# subtracts that spectrum while sparing the harmonic honk transient; anlmdn mops
# up residual broadband hiss. The highpass sits at 300Hz (not 250) to also trim
# low-frequency mic-handling rumble. afftdn is kept gentle (nr=12, static profile
# tn=0): heavier reduction (nr=24, tracking) carved "musical noise" out of the
# gaps between the honk's harmonics, which read as a gritty mic distortion.
ffmpeg -y -loglevel error -i "$SRC" -af "\
highpass=f=300,lowpass=f=5000,volume=${gain}dB,\
asendcmd=2.0 afftdn sample_noise start,\
asendcmd=6.0 afftdn sample_noise end,\
afftdn=nr=12:nf=-38:tn=0,\
anlmdn=s=0.0005:p=0.004:r=0.006" \
	-ar 44100 -ac 1 "$band"

# 2. cut each hand-picked window, de-click with short fades, peak-normalize to -1 dB
fo=$(python3 -c "print(f'{$LEN-0.06:.3f}')")
for i in "${!STARTS[@]}"; do
	ss="${STARTS[$i]}"
	raw="$(mktemp --suffix=.wav)"
	ffmpeg -y -loglevel error -ss "$ss" -t "$LEN" -i "$band" \
		-af "afade=t=in:d=0.015,afade=t=out:st=${fo}:d=0.06" -ar 44100 -ac 1 "$raw"
	cpk=$(maxvol "$raw")
	cg=$(python3 -c "print(f'{-1.0-($cpk):.1f}')")
	ffmpeg -y -loglevel error -i "$raw" -af "volume=${cg}dB" \
		-ar 44100 -ac 1 "$(printf '%s/honk%02d.wav' "$OUT" "$i")"
	rm -f "$raw"
done

echo "slice_goose: wrote $(find "$OUT" -name '*.wav' | wc -l) honks to $OUT"
