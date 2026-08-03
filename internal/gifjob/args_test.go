package gifjob

import (
	"reflect"
	"strings"
	"testing"
)

func TestVideoArgsGIF(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 1000, EndMS: 3500, FPS: 15, Width: 480, Loop: LoopForever, Format: FormatGIF, Quality: Quality{MaxColors: 128, Dither: DitherSierra}}
	passes := VideoArgs(c, "pal.png", "out.gif")
	if len(passes) != 2 {
		t.Fatalf("GIF should be 2 passes, got %d", len(passes))
	}
	want1 := []string{
		"-y", "-ss", "1.000", "-t", "2.500", "-i", "in.mp4",
		"-filter_complex", "[0:v]fps=15,scale=480:-2:flags=lanczos[v];[v]palettegen=max_colors=128:stats_mode=diff[p]",
		"-map", "[p]", "pal.png",
	}
	if !reflect.DeepEqual(passes[0], want1) {
		t.Errorf("pass1 =\n%v\nwant\n%v", passes[0], want1)
	}
	want2 := []string{
		"-y", "-ss", "1.000", "-t", "2.500", "-i", "in.mp4", "-i", "pal.png",
		"-filter_complex", "[0:v]fps=15,scale=480:-2:flags=lanczos[v];[v][1:v]paletteuse=dither=sierra2_4a[o]",
		"-map", "[o]", "-loop", "0", "out.gif",
	}
	if !reflect.DeepEqual(passes[1], want2) {
		t.Errorf("pass2 =\n%v\nwant\n%v", passes[1], want2)
	}
}

func TestVideoArgsWebP(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 1000, FPS: 10, Width: 160, Loop: LoopForever, Format: FormatWebP, Quality: Quality{MaxColors: 256, Dither: DitherSierra, WebPQuality: 80}}
	passes := VideoArgs(c, "pal.png", "out.webp")
	if len(passes) != 1 {
		t.Fatalf("WebP should be 1 pass, got %d", len(passes))
	}
	want := []string{
		"-y", "-ss", "0.000", "-t", "1.000", "-i", "in.mp4",
		"-filter_complex", "[0:v]fps=10,scale=160:-2:flags=lanczos[v]", "-map", "[v]",
		"-c:v", "libwebp_anim", "-loop", "0", "-q:v", "80", "out.webp",
	}
	if !reflect.DeepEqual(passes[0], want) {
		t.Errorf("webp pass =\n%v\nwant\n%v", passes[0], want)
	}
}

func TestVideoArgsAPNGLoopOnce(t *testing.T) {
	c := VideoConfig{Input: "in.mp4", StartMS: 0, EndMS: 1000, FPS: 10, Width: 160, Loop: LoopOnce, Format: FormatAPNG, Quality: DefaultQuality()}
	passes := VideoArgs(c, "pal.png", "out.png")
	if len(passes) != 1 {
		t.Fatalf("APNG should be 1 pass, got %d", len(passes))
	}
	want := []string{
		"-y", "-ss", "0.000", "-t", "1.000", "-i", "in.mp4",
		"-filter_complex", "[0:v]fps=10,scale=160:-2:flags=lanczos[v]", "-map", "[v]",
		"-f", "apng", "-plays", "1", "out.png", // once -> plays 1
	}
	if !reflect.DeepEqual(passes[0], want) {
		t.Errorf("apng pass =\n%v\nwant\n%v", passes[0], want)
	}
}

func TestImagesArgsAndConcatList(t *testing.T) {
	c := ImagesConfig{Inputs: []string{"a.png", "b.png"}, FrameMS: 100, Width: 400, Loop: 3, Quality: Quality{MaxColors: 256, Dither: DitherSierra}}
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

func TestWriteConcatListEscapesQuotes(t *testing.T) {
	inputs := []string{"a'b.png", "c'd.png"}
	var b strings.Builder
	if err := WriteConcatList(&b, inputs, 100); err != nil {
		t.Fatal(err)
	}
	want := "file 'a'\\''b.png'\nduration 0.100\nfile 'c'\\''d.png'\nduration 0.100\nfile 'c'\\''d.png'\n"
	if b.String() != want {
		t.Errorf("concat list with quotes =\n%q\nwant\n%q", b.String(), want)
	}
}

func TestNormalizeArgs(t *testing.T) {
	got := NormalizeArgs("in.png", "out.png", 400, 300)
	want := []string{
		"-y", "-i", "in.png",
		"-vf", "scale=400:300:force_original_aspect_ratio=decrease,pad=400:300:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1",
		"out.png",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NormalizeArgs =\n%v\nwant\n%v", got, want)
	}
}

func TestLoopArgs(t *testing.T) {
	cases := []struct {
		f    Format
		loop LoopMode
		flag string
		val  string
	}{
		{FormatGIF, LoopForever, "-loop", "0"},
		{FormatGIF, LoopOnce, "-loop", "-1"},
		{FormatGIF, LoopMode(5), "-loop", "5"},
		{FormatWebP, LoopForever, "-loop", "0"},
		{FormatWebP, LoopOnce, "-loop", "1"}, // once is 1 for webp, not -1
		{FormatWebP, LoopMode(5), "-loop", "5"},
		{FormatAPNG, LoopForever, "-plays", "0"},
		{FormatAPNG, LoopOnce, "-plays", "1"},
		{FormatAPNG, LoopMode(3), "-plays", "3"},
	}
	for _, c := range cases {
		flag, val := loopArgs(c.f, c.loop)
		if flag != c.flag || val != c.val {
			t.Errorf("loopArgs(%q,%d) = %q %q, want %q %q", c.f, int(c.loop), flag, val, c.flag, c.val)
		}
	}
}

func TestPaletteTails(t *testing.T) {
	q := Quality{MaxColors: 128, Dither: DitherBayer}
	if got := palettegenTail(q); got != "palettegen=max_colors=128:stats_mode=diff" {
		t.Errorf("palettegenTail = %q", got)
	}
	if got := paletteuseTail(q); got != "paletteuse=dither=bayer" {
		t.Errorf("paletteuseTail = %q", got)
	}
}

func TestFilterChainAndGraph(t *testing.T) {
	// Plain: fps + scale only.
	c := VideoConfig{FPS: 15, Width: 480}
	if got := filterChain(c); got != "fps=15,scale=480:-2:flags=lanczos" {
		t.Errorf("plain chain = %q", got)
	}
	if got := videoGraph(c); got != "[0:v]fps=15,scale=480:-2:flags=lanczos[v]" {
		t.Errorf("plain graph = %q", got)
	}
	// Everything on: crop(1:1) + fps + scale + speed 2x + reverse, plus boomerang.
	c2 := VideoConfig{FPS: 12, Width: 200, Aspect: AspectSquare, Speed: 2.0, Reverse: true, Boomerang: true}
	wantChain := "crop='min(iw,ih)':'min(iw,ih)',fps=12,scale=200:-2:flags=lanczos,setpts=0.5000*PTS,reverse"
	if got := filterChain(c2); got != wantChain {
		t.Errorf("full chain =\n%q\nwant\n%q", got, wantChain)
	}
	wantGraph := "[0:v]" + wantChain + ",split[a][b];[b]reverse[r];[a][r]concat=n=2:v=1[v]"
	if got := videoGraph(c2); got != wantGraph {
		t.Errorf("boomerang graph =\n%q\nwant\n%q", got, wantGraph)
	}
	// Speed 1.0 and speed 0 both omit setpts.
	if setptsExpr(1.0) != "" || setptsExpr(0) != "" {
		t.Errorf("setpts for 1.0/0 should be empty")
	}
	if got := setptsExpr(0.5); got != "setpts=2.0000*PTS" {
		t.Errorf("setpts(0.5) = %q, want setpts=2.0000*PTS", got)
	}
	// Aspect crop expressions.
	if cropExpr(AspectFree) != "" {
		t.Error("free aspect must produce no crop")
	}
	if got := cropExpr(AspectWide); got != "crop='min(iw,ih*16/9)':'min(ih,iw*9/16)'" {
		t.Errorf("16:9 crop = %q", got)
	}
	if got := cropExpr(AspectTall); got != "crop='min(iw,ih*9/16)':'min(ih,iw*16/9)'" {
		t.Errorf("9:16 crop = %q", got)
	}
}
