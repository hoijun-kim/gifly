package ffmpeg

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseProgressStream(t *testing.T) {
	// ffmpeg -progress writes flat key=value lines, one block per update,
	// terminated by a progress=continue (or progress=end for the last).
	stream := strings.Join([]string{
		"frame=12",
		"out_time_ms=500000",
		"progress=continue",
		"frame=48",
		"out_time_ms=2000000",
		"progress=end",
	}, "\n")

	var got []Progress
	var p Progress
	for _, line := range strings.Split(stream, "\n") {
		if parseProgressLine(line, &p) {
			got = append(got, p)
		}
	}

	if len(got) != 2 {
		t.Fatalf("emitted %d progress blocks, want 2: %+v", len(got), got)
	}
	if got[0].Frame != 12 || got[0].OutTimeMS != 500 || got[0].Done {
		t.Errorf("block 0 = %+v, want frame 12, 500ms, not done", got[0])
	}
	if got[1].Frame != 48 || got[1].OutTimeMS != 2000 || !got[1].Done {
		t.Errorf("block 1 = %+v, want frame 48, 2000ms, done", got[1])
	}
}

// TestHelperProcess is not a real test; the tests below re-exec this binary as a
// stand-in for ffmpeg, choosing behavior via GIFLY_HELPER_MODE.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GIFLY_HELPER_WANTED") != "1" {
		return
	}
	switch os.Getenv("GIFLY_HELPER_MODE") {
	case "progress-then-ok":
		os.Stdout.WriteString("frame=5\nout_time_ms=1000000\nprogress=continue\nprogress=end\n")
	case "fail-with-stderr":
		os.Stderr.WriteString("Error: unsupported codec")
		os.Exit(3)
	case "hang":
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}

// helperRun re-execs this test binary as TestHelperProcess and drives it through
// runArgs (NOT Run - the global ffmpeg flags would be rejected as test flags).
func helperRun(ctx context.Context, mode string, onProgress func(Progress)) error {
	os.Setenv("GIFLY_HELPER_WANTED", "1")
	os.Setenv("GIFLY_HELPER_MODE", mode)
	return runArgs(ctx, os.Args[0], []string{"-test.run=TestHelperProcess", "--"}, onProgress)
}

func TestRunReportsProgressAndSucceeds(t *testing.T) {
	var last Progress
	if err := helperRun(context.Background(), "progress-then-ok", func(p Progress) { last = p }); err != nil {
		t.Fatalf("runArgs = %v, want success", err)
	}
	if last.OutTimeMS != 1000 || !last.Done {
		t.Errorf("last progress = %+v, want 1000ms and Done", last)
	}
}

func TestRunSurfacesStderrTailOnFailure(t *testing.T) {
	err := helperRun(context.Background(), "fail-with-stderr", nil)
	if err == nil {
		t.Fatal("runArgs should fail on a non-zero exit")
	}
	if !strings.Contains(err.Error(), "unsupported codec") {
		t.Errorf("error %q does not carry ffmpeg's stderr tail", err.Error())
	}
}

func TestRunCancelKillsProcess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- helperRun(ctx, "hang", nil) }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a cancelled run should return a non-nil error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after ctx cancel; the process was not killed")
	}
}
