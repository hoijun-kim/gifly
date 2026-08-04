#!/usr/bin/env bash
# Build gifly's Windows release artifacts:
#   A) gifly-<ver>-win64.zip            - gifly.exe + a sidecar ffmpeg/ folder
#   B) gifly-<ver>-standalone-win64.zip - a single gifly.exe with ffmpeg inside
# plus SHA256SUMS.txt. Run from anywhere; needs the Wails CLI and a source
# FFmpeg (ffmpeg.exe + ffprobe.exe + its LICENSE.txt).
#
# Usage: scripts/package.sh [version]   (version defaults to 0.1.0)
#   GIFLY_FFMPEG_DIR overrides where the source ffmpeg.exe/ffprobe.exe live.
set -euo pipefail

VER="${1:-0.1.0}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

FFSRC="${GIFLY_FFMPEG_DIR:-C:/Users/hoijun/ffmpeg-dev/ffmpeg-master-latest-win64-lgpl/bin}"
FFLICENSE="$(dirname "$FFSRC")/LICENSE.txt"
for f in "$FFSRC/ffmpeg.exe" "$FFSRC/ffprobe.exe" "$FFLICENSE"; do
  [ -f "$f" ] || { echo "package: missing $f (set GIFLY_FFMPEG_DIR)"; exit 1; }
done

DIST="$ROOT/dist"
rm -rf "$DIST"; mkdir -p "$DIST"

# keep the tracked third-party license copy fresh.
mkdir -p THIRD-PARTY-LICENSES
cp "$FFLICENSE" THIRD-PARTY-LICENSES/ffmpeg-LICENSE.txt

zipdir() { # zipdir <staging-dir> <zip-basename>   (zip lands beside the staging dir)
  ( cd "$(dirname "$1")" && powershell -NoProfile -Command \
    "Compress-Archive -Path '$(basename "$1")/*' -DestinationPath '$2' -Force" )
}

echo "== A: zip build (sidecar ffmpeg) =="
wails build -tags production >/dev/null
git checkout -- frontend/dist/.gitkeep 2>/dev/null || true
A="$DIST/gifly-$VER-win64"
mkdir -p "$A/ffmpeg" "$A/THIRD-PARTY-LICENSES"
cp build/bin/gifly.exe "$A/"
cp "$FFSRC/ffmpeg.exe" "$FFSRC/ffprobe.exe" "$A/ffmpeg/"
cp LICENSE NOTICE.txt "$A/"
cp THIRD-PARTY-LICENSES/ffmpeg-LICENSE.txt "$A/THIRD-PARTY-LICENSES/"
cp packaging/README-zip.txt "$A/README.txt"
zipdir "$A" "gifly-$VER-win64.zip"

echo "== B: standalone single exe (embedded ffmpeg) =="
mkdir -p internal/ffmpeg/bin
cp "$FFSRC/ffmpeg.exe" "$FFSRC/ffprobe.exe" internal/ffmpeg/bin/
wails build -tags "production embedffmpeg" >/dev/null
git checkout -- frontend/dist/.gitkeep 2>/dev/null || true
B="$DIST/gifly-$VER-standalone-win64"
mkdir -p "$B/THIRD-PARTY-LICENSES"
cp build/bin/gifly.exe "$B/gifly.exe"
cp LICENSE NOTICE.txt "$B/"
cp THIRD-PARTY-LICENSES/ffmpeg-LICENSE.txt "$B/THIRD-PARTY-LICENSES/"
cp packaging/README-standalone.txt "$B/README.txt"
zipdir "$B" "gifly-$VER-standalone-win64.zip"
rm -rf internal/ffmpeg/bin

# leave build/bin holding the normal (small) exe, not the standalone one.
wails build -tags production >/dev/null
git checkout -- frontend/dist/.gitkeep 2>/dev/null || true

echo "== checksums =="
( cd "$DIST" && sha256sum *.zip > SHA256SUMS.txt )

echo "== done =="
( cd "$DIST" && ls -la *.zip SHA256SUMS.txt )
