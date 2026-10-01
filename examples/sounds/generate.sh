#!/bin/sh
# Generates the example sounds. Needs ffmpeg and espeak-ng.
set -e
cd "$(dirname "$0")"
Q="-loglevel error -y"

ffmpeg $Q -f lavfi -i "aevalsrc=0.6*sin(2*PI*880*t)*min(1\,(0.15-t)*40):s=16000:d=0.15" \
	-ac 1 -c:a pcm_s16le beep.wav

ffmpeg $Q -f lavfi -i "aevalsrc=0.8*sin(2*PI*2000*t)*exp(-t*400):s=8000:d=0.02" \
	-ac 1 -c:a pcm_u8 click.wav

espeak-ng -w speech_raw.wav "Hello from Tiny Go. This is tiny player."
ffmpeg $Q -i speech_raw.wav -ac 1 -ar 16000 -c:a adpcm_ima_wav speech.wav
rm speech_raw.wav

ffmpeg $Q -f lavfi -i "aevalsrc=\
0.4*sin(2*PI*(262*between(mod(t\,2)\,0\,0.5)+330*between(mod(t\,2)\,0.5\,1)+392*between(mod(t\,2)\,1\,1.5)+523*between(mod(t\,2)\,1.5\,2))*t)*(1-mod(t\,0.5)*1.6)|\
0.4*sin(2*PI*131*t):s=44100:d=2" \
	-c:a adpcm_ima_wav music.wav
