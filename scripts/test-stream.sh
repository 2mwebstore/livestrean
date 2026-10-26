#!/usr/bin/env bash
# Streams a test pattern with a beep to a channel, the way OBS does it:
#   Server     = rtmp://<host>/live
#   Stream Key = <slug>?user=obs&pass=<stream key>
#
# Usage: ./scripts/test-stream.sh <channel-slug> <stream-key>
set -euo pipefail

if [ $# -ne 2 ]; then
  echo "Usage: $0 <channel-slug> <stream-key>" >&2
  exit 1
fi
SLUG="$1"
KEY="$2"

echo "Streaming test pattern to channel '$SLUG' (Ctrl+C to stop)"
exec docker run --rm --init --add-host=host.docker.internal:host-gateway \
  --entrypoint ffmpeg bluenviron/mediamtx:1.21.2-ffmpeg \
  -hide_banner -loglevel warning -re \
  -f lavfi -i "testsrc2=size=1280x720:rate=30" \
  -f lavfi -i "sine=frequency=440:sample_rate=48000" \
  -c:v libx264 -preset veryfast -tune zerolatency -pix_fmt yuv420p -g 30 -b:v 2500k \
  -c:a aac -b:a 128k \
  -f flv -rtmp_app live -rtmp_playpath "${SLUG}?user=obs&pass=${KEY}" \
  "rtmp://host.docker.internal:1935/live"
