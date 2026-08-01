package gifjob

import (
	"reflect"
	"strings"
	"testing"
)

func TestVideoArgsTwoPass(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 1000, EndMS: 3500, FPS: 15, Width: 480, Loop: LoopForever, Quality: Quality{MaxColors: 128, Dither: true}}
	p1, p2 := VideoArgs(c, "pal.png", "out.gif")

	want1 := []string{
		"-y", "-ss", "1.000", "-i", "in.mp4", "-t", "2.500",
		"-vf", "fps=15,scale=480:-2:flags=lanczos,palettegen=max_colors=128:stats_mode=diff",
		"pal.png",
	}
	if !reflect.DeepEqual(p1, want1) {
		t.Errorf("pass1 =\n%v\nwant\n%v", p1, want1)
	}
	want2 := []string{
		"-y", "-ss", "1.000", "-i", "in.mp4", "-t", "2.500", "-i", "pal.png",
		"-lavfi", "fps=15,scale=480:-2:flags=lanczos[x];[x][1:v]paletteuse=dither=sierra2_4a",
		"-loop", "0", "out.gif",
	}
	if !reflect.DeepEqual(p2, want2) {
		t.Errorf("pass2 =\n%v\nwant\n%v", p2, want2)
	}
}

func TestVideoArgsDitherOffAndLoopOnce(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 1000, FPS: 10, Width: 320, Loop: LoopOnce, Quality: Quality{MaxColors: 256, Dither: false}}
	_, p2 := VideoArgs(c, "pal.png", "out.gif")
	joined := strings.Join(p2, " ")
	if !strings.Contains(joined, "paletteuse=dither=none") {
		t.Errorf("dither off should produce dither=none: %s", joined)
	}
	if !strings.Contains(joined, "-loop -1") {
		t.Errorf("LoopOnce should produce -loop -1: %s", joined)
	}
}

func TestImagesArgsAndConcatList(t *testing.T) {
	c := ImagesConfig{Inputs: []string{"a.png", "b.png"}, FrameMS: 100, Width: 400, Loop: 3, Quality: Quality{MaxColors: 256, Dither: true}}
	p1, p2 := ImagesArgs(c, "list.txt", "pal.png", "out.gif")

	want1 := []string{
		"-y", "-f", "concat", "-safe", "0", "-i", "list.txt",
		"-vf", "scale=400:-2:flags=lanczos,palettegen=max_colors=256:stats_mode=diff",
		"pal.png",
	}
	if !reflect.DeepEqual(p1, want1) {
		t.Errorf("images pass1 =\n%v\nwant\n%v", p1, want1)
	}
	want2 := []string{
		"-y", "-f", "concat", "-safe", "0", "-i", "list.txt", "-i", "pal.png",
		"-lavfi", "scale=400:-2:flags=lanczos[x];[x][1:v]paletteuse=dither=sierra2_4a",
		"-loop", "3", "out.gif",
	}
	if !reflect.DeepEqual(p2, want2) {
		t.Errorf("images pass2 =\n%v\nwant\n%v", p2, want2)
	}

	var b strings.Builder
	if err := WriteConcatList(&b, c.Inputs, c.FrameMS); err != nil {
		t.Fatal(err)
	}
	want := "file 'a.png'\nduration 0.100\nfile 'b.png'\nduration 0.100\nfile 'b.png'\n"
	if b.String() != want {
		t.Errorf("concat list =\n%q\nwant\n%q", b.String(), want)
	}
}
