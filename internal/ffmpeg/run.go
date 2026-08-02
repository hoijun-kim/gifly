package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Progress is one update from ffmpeg's -progress stream. OutTimeMS is the output
// timestamp in milliseconds (ffmpeg reports microseconds as out_time_ms - a
// long-standing misnomer - so it is divided by 1000 here). Done is set on the
// terminal block.
type Progress struct {
	OutTimeMS int64
	Frame     int
	Done      bool
}

// parseProgressLine folds one "key=value" line into p. It returns true when the
// line closes a progress block ("progress=continue" or "progress=end"), at which
// point p holds a complete update; Done reflects whether it was the end block.
func parseProgressLine(line string, p *Progress) bool {
	k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
	if !ok {
		return false
	}
	switch k {
	case "frame":
		if n, err := strconv.Atoi(v); err == nil {
			p.Frame = n
		}
	case "out_time_ms": // ffmpeg reports microseconds under this key
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			p.OutTimeMS = n / 1000
		}
	case "progress":
		p.Done = v == "end"
		return true
	}
	return false
}

// Run executes bin with gifly's standard global flags followed by args (see
// runArgs for the streaming/error behavior).
func Run(ctx context.Context, bin string, args []string, onProgress func(Progress)) error {
	full := append([]string{"-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats"}, args...)
	return runArgs(ctx, bin, full, onProgress)
}

// runArgs runs bin with exactly args (no flags prepended), parses the -progress
// stream on stdout into onProgress, kills the process on ctx cancel, and on a
// non-zero exit returns an error carrying the tail of stderr. Run adds the
// global flags; tests call runArgs directly with a test-binary re-exec, which
// must not receive those flags.
func runArgs(ctx context.Context, bin string, args []string, onProgress func(Progress)) error {
	cmd := exec.CommandContext(ctx, bin, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ffmpeg: %w", err)
	}

	var p Progress
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		if parseProgressLine(sc.Text(), &p) && onProgress != nil {
			onProgress(p)
		}
	}

	if err := cmd.Wait(); err != nil {
		tail := strings.TrimSpace(stderr.String())
		if len(tail) > 500 {
			tail = tail[len(tail)-500:]
		}
		if tail == "" {
			return fmt.Errorf("ffmpeg failed: %w", err)
		}
		return fmt.Errorf("ffmpeg failed: %w: %s", err, tail)
	}
	return nil
}

// RunnerFunc adapts the package-level Run into an interface value the gifjob
// orchestrator can accept, so tests can substitute a fake that records argv and
// never launches a process.
type RunnerFunc func(ctx context.Context, bin string, args []string, onProgress func(Progress)) error

func (f RunnerFunc) Run(ctx context.Context, bin string, args []string, onProgress func(Progress)) error {
	return f(ctx, bin, args, onProgress)
}
