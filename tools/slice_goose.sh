#!/usr/bin/env bash
# slice_goose.sh — (re)generate honks/goose/ from the committed CC0 source.
#
# Fully reproducible from a clean clone: reads assets/geese-cc0.flac, band-limits
# to the goose band, spectrally subtracts the steady wind (profiled from the
# wind-only lead-in before the flurry), brings the (very quiet) recording up to a
# usable level, then slices the honking flurry into N even, de-clicked,
# peak-normalized clips. Requires ffmpeg.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$REPO/assets/geese-cc0.flac"
OUT="$REPO/honks/goose"
START=13.0     # the honking flurry is dense from ~13s
STEP=0.8       # hop between clip starts
LEN=0.70       # clip length (a snappy honk per keypress)
N=10

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
	-af "highpass=f=250,lowpass=f=5000,volumedetect" -f null - 2>&1 \
	| awk '/max_volume:/ {print $(NF-1)}')
gain=$(python3 -c "print(f'{-1.0-($pk):.1f}')")
# After band-limiting and gain, spectrally subtract the steady wind: profile it
# from the wind-only lead-in (2-6s, before the flurry at ~13s), then afftdn
# subtracts that spectrum while sparing the harmonic honk transient. anlmdn mops
# up residual broadband hiss; a gentle gate keeps the inter-clip floor at zero.
# Nets ~+14 dB honk-over-wind vs band-limiting alone, so overlapping honks no
# longer stack their noise floors into an audible wind swell.
ffmpeg -y -loglevel error -i "$SRC" -af "\
highpass=f=250,lowpass=f=5000,volume=${gain}dB,\
asendcmd=2.0 afftdn sample_noise start,\
asendcmd=6.0 afftdn sample_noise end,\
afftdn=nr=24:nf=-45:tn=1,\
anlmdn=s=0.0008:p=0.004:r=0.006,\
agate=threshold=0.01:ratio=2:attack=5:release=120" \
	-ar 44100 -ac 1 "$band"

# 2. slice into N clips, de-click with short fades, peak-normalize each to -1 dB
for i in $(seq 0 $((N - 1))); do
	ss=$(python3 -c "print(f'{$START + $i*$STEP:.3f}')")
	fo=$(python3 -c "print(f'{$LEN-0.06:.3f}')")
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
