#!/usr/bin/env bash
# generate_honks.sh — synthesize the fallback honks/goose-synth/ pack with ffmpeg.
# 100% synthesis, no third-party assets: a buzzy band-limited sawtooth with a
# pitch glide and vibrato reads as a (cartoonish) goose honk. Requires ffmpeg.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$REPO/honks/goose-synth"
SR=44100
mkdir -p "$OUT"
rm -f "$OUT"/*.wav

gen() { ffmpeg -y -loglevel error -f lavfi -i "aevalsrc=$3:s=$SR:d=$2" -ac 1 "$1"; }

#       base  glide  vib_rate vib_depth  dur
PARAMS=(
	"210  0.14   6  0.04  0.34"
	"245  0.10   5  0.03  0.30"
	"280 -0.08   7  0.05  0.28"
	"315  0.18   6  0.04  0.26"
	"360  0.06   8  0.03  0.24"
	"400 -0.10   5  0.06  0.32"
	"440  0.20   9  0.04  0.22"
	"300  0.12   6  0.05  0.38"
	"260 -0.06   7  0.03  0.30"
	"330  0.16   8  0.05  0.25"
)
i=0
for p in "${PARAMS[@]}"; do
	read -r f g vr vd d <<<"$p"
	F="(${f}*(1+${g}*t/${d})*(1+${vd}*sin(2*PI*${vr}*t)))"
	SAW="(sin(2*PI*${F}*t)+sin(2*PI*2*${F}*t)/2+sin(2*PI*3*${F}*t)/3+sin(2*PI*4*${F}*t)/4+sin(2*PI*5*${F}*t)/5)"
	ENV="((1-exp(-t/0.008))*exp(-t/(${d}*0.55)))"
	gen "$(printf '%s/honk%02d.wav' "$OUT" "$i")" "$d" "0.3*${ENV}*${SAW}"
	i=$((i + 1))
done

echo "generate_honks: wrote $(find "$OUT" -name '*.wav' | wc -l) synth honks to $OUT"
