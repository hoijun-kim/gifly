gifly (standalone) - turn a video or a set of images into a GIF, WebP or APNG.

This is the single-file build. gifly.exe carries FFmpeg inside it and needs no
ffmpeg/ folder next to it. On first run it writes its copy of ffmpeg.exe and
ffprobe.exe to %LOCALAPPDATA%\gifly\ so later runs start faster; the exe stays
large because it contains them.

Getting started
---------------
Just run gifly.exe.

What's here
-----------
  gifly.exe             the app (FFmpeg included)
  LICENSE               gifly license (PolyForm Noncommercial 1.0.0)
  NOTICE.txt            third-party notices (FFmpeg)
  THIRD-PARTY-LICENSES/ FFmpeg license (LGPL-3.0)

Using your own FFmpeg
---------------------
gifly still checks the GIFLY_FFMPEG_DIR environment variable and your PATH
before its embedded copy, so you can point it at a different LGPL-compatible
FFmpeg build.
