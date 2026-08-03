// Command giflycli drives a GIF conversion from the command line - the engine's
// end-to-end harness before any GUI exists, and the way the manual gate below is
// run against a real ffmpeg.
//
//	giflycli video -i in.mp4 -o out.gif -ss 1000 -to 3500 -fps 15 -w 480 [-loop forever] [-colors 256] [-nodither]
//	    [-format gif|webp|apng] [-aspect free|1:1|16:9|9:16] [-speed 0.25..4] [-reverse] [-boomerang]
//	    [-dither none|bayer|sierra2|floyd] [-q 0..100]
//	giflycli images -o out.gif -ms 100 -w 400 [-h N] [-loop forever] [-colors 256] [-nodither]
//	    [-format gif|webp|apng] [-aspect free|1:1|16:9|9:16] [-speed 0.25..4] [-reverse] [-boomerang]
//	    [-dither none|bayer|sierra2|floyd] [-q 0..100] a.png b.png c.png
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/hoijun-kim/gifly/internal/ffmpeg"
	"github.com/hoijun-kim/gifly/internal/gifjob"
	"github.com/hoijun-kim/gifly/internal/probe"
)

// parseLoop turns the -loop flag into a LoopMode: "forever", "once", or a
// non-negative integer count.
func parseLoop(s string) (gifjob.LoopMode, error) {
	switch s {
	case "forever":
		return gifjob.LoopForever, nil
	case "once":
		return gifjob.LoopOnce, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("loop must be forever, once, or a non-negative count, got %q", s)
	}
	return gifjob.LoopMode(n), nil
}

// imagesHeight picks the images canvas height: a positive -h override, else the
// even height derived from the first frame's aspect at the output width.
func imagesHeight(override, width, firstW, firstH int) int {
	if override > 0 {
		return override
	}
	return gifjob.CanvasHeight(firstW, firstH, width)
}

func parseFormat(s string) (gifjob.Format, error) {
	f := gifjob.Format(s)
	if !f.Valid() {
		return "", fmt.Errorf("format must be gif, webp or apng, got %q", s)
	}
	return f, nil
}

func parseAspect(s string) (gifjob.Aspect, error) {
	if s == "free" || s == "" {
		return gifjob.AspectFree, nil
	}
	a := gifjob.Aspect(s)
	if !a.Valid() {
		return "", fmt.Errorf("aspect must be free, 1:1, 16:9 or 9:16, got %q", s)
	}
	return a, nil
}

func parseDither(s string) (gifjob.DitherMethod, error) {
	switch gifjob.DitherMethod(s) {
	case gifjob.DitherNone, gifjob.DitherBayer, gifjob.DitherSierra, gifjob.DitherFloyd:
		return gifjob.DitherMethod(s), nil
	}
	return "", fmt.Errorf("dither must be none, bayer, sierra2 or floyd, got %q", s)
}

// resolveDither parses -dither and applies the -nodither back-compat override:
// when noDither is set it forces DitherNone regardless of the -dither value.
func resolveDither(s string, noDither bool) (gifjob.DitherMethod, error) {
	if noDither {
		return gifjob.DitherNone, nil
	}
	return parseDither(s)
}

// defaultOut swaps the extension of the default "out.gif" to match the format.
// A caller who set any other output name keeps it verbatim.
func defaultOut(out string, f gifjob.Format) string {
	if out == "out.gif" {
		return "out" + f.Ext()
	}
	return out
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: giflycli <video|images> ...")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "giflycli:", err)
		os.Exit(1)
	}
}

func run(sub string, argv []string) error {
	tools, err := ffmpeg.Tools()
	if err != nil {
		return err
	}
	onProg := func(p ffmpeg.Progress) { fmt.Printf("\r  %d ms encoded", p.OutTimeMS) }

	switch sub {
	case "video":
		fs := flag.NewFlagSet("video", flag.ContinueOnError)
		in := fs.String("i", "", "input video")
		out := fs.String("o", "out.gif", "output gif")
		ss := fs.Int64("ss", 0, "trim start (ms)")
		to := fs.Int64("to", 0, "trim end (ms); 0 = end of video")
		fps := fs.Int("fps", 15, "output fps")
		w := fs.Int("w", 0, "output width; 0 = source width")
		loopS := fs.String("loop", "forever", "forever | once | N")
		colors := fs.Int("colors", 256, "palette colors 2..256")
		noDither := fs.Bool("nodither", false, "disable dithering")
		formatS := fs.String("format", "gif", "output format: gif | webp | apng")
		aspectS := fs.String("aspect", "free", "crop aspect: free | 1:1 | 16:9 | 9:16")
		speed := fs.Float64("speed", 1.0, "playback speed 0.25..4")
		reverse := fs.Bool("reverse", false, "reverse playback")
		boomerang := fs.Bool("boomerang", false, "play forward then backward")
		ditherS := fs.String("dither", "sierra2", "dither: none | bayer | sierra2 | floyd")
		webpQ := fs.Int("q", 75, "webp quality 0..100")
		if err := fs.Parse(argv); err != nil {
			return err
		}
		loop, err := parseLoop(*loopS)
		if err != nil {
			return err
		}
		fmtVal, err := parseFormat(*formatS)
		if err != nil {
			return err
		}
		aspectVal, err := parseAspect(*aspectS)
		if err != nil {
			return err
		}
		ditherVal, err := resolveDither(*ditherS, *noDither)
		if err != nil {
			return err
		}
		m, err := probe.Video(context.Background(), tools.FFprobe, *in)
		if err != nil {
			return err
		}
		end := *to
		if end == 0 {
			end = m.DurationMS
		}
		width := *w
		if width == 0 {
			width = m.Width
		}
		*out = defaultOut(*out, fmtVal)
		q := gifjob.Quality{MaxColors: *colors, Dither: ditherVal, WebPQuality: *webpQ}
		c := gifjob.VideoConfig{
			Input: *in, StartMS: *ss, EndMS: end, FPS: *fps, Width: width, Loop: loop, Quality: q,
			SrcWidth: m.Width, SrcHeight: m.Height, Aspect: aspectVal,
			Speed: *speed, Reverse: *reverse, Boomerang: *boomerang, Format: fmtVal,
		}
		res, err := gifjob.RunVideo(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, *out, onProg)
		if err != nil {
			return err
		}
		fmt.Printf("\rwrote %s  %dx%d  %d bytes\n", res.Path, res.Width, res.Height, res.Bytes)
		return nil

	case "images":
		fs := flag.NewFlagSet("images", flag.ContinueOnError)
		out := fs.String("o", "out.gif", "output gif")
		ms := fs.Int("ms", 100, "per-frame duration (ms)")
		w := fs.Int("w", 480, "output width")
		h := fs.Int("h", 0, "output height; 0 = from the first image")
		loopS := fs.String("loop", "forever", "forever | once | N")
		colors := fs.Int("colors", 256, "palette colors 2..256")
		noDither := fs.Bool("nodither", false, "disable dithering")
		formatS := fs.String("format", "gif", "output format: gif | webp | apng")
		aspectS := fs.String("aspect", "free", "crop aspect: free | 1:1 | 16:9 | 9:16")
		speed := fs.Float64("speed", 1.0, "playback speed 0.25..4")
		reverse := fs.Bool("reverse", false, "reverse playback")
		boomerang := fs.Bool("boomerang", false, "play forward then backward")
		ditherS := fs.String("dither", "sierra2", "dither: none | bayer | sierra2 | floyd")
		webpQ := fs.Int("q", 75, "webp quality 0..100")
		if err := fs.Parse(argv); err != nil {
			return err
		}
		loop, err := parseLoop(*loopS)
		if err != nil {
			return err
		}
		fmtVal, err := parseFormat(*formatS)
		if err != nil {
			return err
		}
		aspectVal, err := parseAspect(*aspectS)
		if err != nil {
			return err
		}
		ditherVal, err := resolveDither(*ditherS, *noDither)
		if err != nil {
			return err
		}
		if len(fs.Args()) == 0 {
			return fmt.Errorf("no input images")
		}
		first, err := probe.Image(fs.Args()[0])
		if err != nil {
			return err
		}
		height := *h
		if height <= 0 {
			height = gifjob.OutputHeight(first.Width, first.Height, aspectVal, *w)
		}
		*out = defaultOut(*out, fmtVal)
		q := gifjob.Quality{MaxColors: *colors, Dither: ditherVal, WebPQuality: *webpQ}
		c := gifjob.ImagesConfig{
			Inputs: fs.Args(), FrameMS: *ms, Width: *w, Height: height, Loop: loop, Quality: q,
			Speed: *speed, Reverse: *reverse, Boomerang: *boomerang, Format: fmtVal,
		}
		res, err := gifjob.RunImages(context.Background(), tools, ffmpeg.RunnerFunc(ffmpeg.Run), c, *out, onProg)
		if err != nil {
			return err
		}
		fmt.Printf("\rwrote %s  %dx%d  %d bytes\n", res.Path, res.Width, res.Height, res.Bytes)
		return nil

	default:
		return fmt.Errorf("unknown subcommand %q (want video or images)", sub)
	}
}
