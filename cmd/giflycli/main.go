// Command giflycli drives a GIF conversion from the command line - the engine's
// end-to-end harness before any GUI exists, and the way the manual gate below is
// run against a real ffmpeg.
//
//	giflycli video -i in.mp4 -o out.gif -ss 1000 -to 3500 -fps 15 -w 480 [-loop forever] [-colors 256] [-nodither]
//	giflycli images -o out.gif -ms 100 -w 400 [-loop forever] a.png b.png c.png
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

func quality(colors int, noDither bool) gifjob.Quality {
	return gifjob.Quality{MaxColors: colors, Dither: !noDither}
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
		if err := fs.Parse(argv); err != nil {
			return err
		}
		loop, err := parseLoop(*loopS)
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
		c := gifjob.VideoConfig{Input: *in, StartMS: *ss, EndMS: end, FPS: *fps, Width: width, Loop: loop, Quality: quality(*colors, *noDither)}
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
		loopS := fs.String("loop", "forever", "forever | once | N")
		colors := fs.Int("colors", 256, "palette colors 2..256")
		noDither := fs.Bool("nodither", false, "disable dithering")
		if err := fs.Parse(argv); err != nil {
			return err
		}
		loop, err := parseLoop(*loopS)
		if err != nil {
			return err
		}
		c := gifjob.ImagesConfig{Inputs: fs.Args(), FrameMS: *ms, Width: *w, Loop: loop, Quality: quality(*colors, *noDither)}
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
