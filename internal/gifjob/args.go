package gifjob

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// secs formats a millisecond count as fixed-point seconds with millisecond
// precision, e.g. 2500 -> "2.500". ffmpeg accepts this for -ss/-t.
func secs(ms int64) string {
	return fmt.Sprintf("%.3f", float64(ms)/1000)
}

func ditherArg(q Quality) string {
	if q.Dither {
		return "sierra2_4a"
	}
	return "none"
}

// scaleChain is the shared scale filter: force width, keep aspect, even height,
// lanczos resampling.
func scaleChain(width int) string {
	return fmt.Sprintf("scale=%d:-2:flags=lanczos", width)
}

func palettegen(width int, q Quality) string {
	return scaleChain(width) + fmt.Sprintf(",palettegen=max_colors=%d:stats_mode=diff", q.MaxColors)
}

func paletteuse(width int, q Quality) string {
	return scaleChain(width) + fmt.Sprintf("[x];[x][1:v]paletteuse=dither=%s", ditherArg(q))
}

// VideoArgs builds the two ffmpeg passes for a video job. Pass 1 writes the
// optimized palette; pass 2 encodes the GIF against it. -ss before -i is a fast
// seek; -t (duration) avoids the -ss/-to origin ambiguity. The config is assumed
// already validated.
func VideoArgs(c VideoConfig, palettePath, outPath string) (pass1, pass2 []string) {
	start := secs(c.StartMS)
	dur := secs(c.EndMS - c.StartMS)
	fps := strconv.Itoa(c.FPS)

	pass1 = []string{
		"-y", "-ss", start, "-t", dur, "-i", c.Input,
		"-vf", "fps=" + fps + "," + palettegen(c.Width, c.Quality),
		palettePath,
	}
	pass2 = []string{
		"-y", "-ss", start, "-t", dur, "-i", c.Input, "-i", palettePath,
		"-lavfi", "fps=" + fps + "," + paletteuse(c.Width, c.Quality),
		"-loop", strconv.Itoa(int(c.Loop)), outPath,
	}
	return pass1, pass2
}

// ImagesArgs builds the two ffmpeg passes for an images job, reading the ordered
// frames through the concat demuxer (frame durations come from listPath, so no
// fps filter is applied).
func ImagesArgs(c ImagesConfig, listPath, palettePath, outPath string) (pass1, pass2 []string) {
	pass1 = []string{
		"-y", "-f", "concat", "-safe", "0", "-i", listPath,
		"-vf", palettegen(c.Width, c.Quality),
		palettePath,
	}
	pass2 = []string{
		"-y", "-f", "concat", "-safe", "0", "-i", listPath, "-i", palettePath,
		"-lavfi", paletteuse(c.Width, c.Quality),
		"-loop", strconv.Itoa(int(c.Loop)), outPath,
	}
	return pass1, pass2
}

// escapeConcatPath escapes single quotes in a path for the ffmpeg concat demuxer.
// Each single quote is replaced by the four-character sequence quote, backslash,
// quote, quote: it closes the single-quoted string, adds a backslash-escaped
// quote, then reopens the string.
func escapeConcatPath(p string) string {
	return strings.ReplaceAll(p, "'", "'\\''")
}

// WriteConcatList writes the concat-demuxer script: each frame with its duration
// in seconds, and the final frame repeated because the concat demuxer ignores
// the last entry's duration (without the repeat, the last frame flashes for one
// output tick instead of frameMS).
func WriteConcatList(w io.Writer, inputs []string, frameMS int) error {
	dur := secs(int64(frameMS))
	for _, in := range inputs {
		if _, err := fmt.Fprintf(w, "file '%s'\nduration %s\n", escapeConcatPath(in), dur); err != nil {
			return err
		}
	}
	if len(inputs) > 0 {
		if _, err := fmt.Fprintf(w, "file '%s'\n", escapeConcatPath(inputs[len(inputs)-1])); err != nil {
			return err
		}
	}
	return nil
}
