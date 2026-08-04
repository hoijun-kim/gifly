gifly - turn a video or a set of images into a GIF, WebP or APNG.

Getting started
---------------
1. Unzip this folder anywhere.
2. Run gifly.exe.

Keep the ffmpeg/ folder next to gifly.exe - the app runs those bundled
ffmpeg.exe/ffprobe.exe to do the actual encoding. If you prefer a single file
with no folder, use the standalone build instead
(gifly-<version>-standalone-win64.zip).

What's here
-----------
  gifly.exe             the app
  ffmpeg/               bundled FFmpeg (ffmpeg.exe, ffprobe.exe) - keep beside gifly.exe
  LICENSE               gifly license (PolyForm Noncommercial 1.0.0)
  NOTICE.txt            third-party notices (FFmpeg)
  THIRD-PARTY-LICENSES/ FFmpeg license (LGPL-3.0)

Using your own FFmpeg
---------------------
Set the GIFLY_FFMPEG_DIR environment variable to a folder containing
ffmpeg.exe and ffprobe.exe, or replace the files in the ffmpeg/ folder with
another LGPL-compatible FFmpeg build.
