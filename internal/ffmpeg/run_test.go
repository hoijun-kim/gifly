package ffmpeg

import (
	"strings"
	"testing"
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
