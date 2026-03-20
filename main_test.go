package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseArgsNoFile(t *testing.T) {
	_, err := parseArgs([]string{})
	if err == nil {
		t.Error("expected error when no args provided")
	}
}

func TestParseArgsImageFile(t *testing.T) {
	cfg, err := parseArgs([]string{"photo.jpg"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.file != "photo.jpg" {
		t.Errorf("expected file=photo.jpg, got %s", cfg.file)
	}
	if cfg.isVideo {
		t.Error("jpg should not be detected as video")
	}
	if cfg.quality != defaultQuality {
		t.Errorf("expected default quality %d, got %d", defaultQuality, cfg.quality)
	}
	if !cfg.colored {
		t.Error("expected colored=true by default")
	}
}

func TestParseArgsVideoFile(t *testing.T) {
	cfg, err := parseArgs([]string{"video.mp4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.isVideo {
		t.Error("mp4 should be detected as video")
	}
}

func TestParseArgsAllParams(t *testing.T) {
	cfg, err := parseArgs([]string{"video.mp4", "50", "15"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.quality != 50 {
		t.Errorf("expected quality=50, got %d", cfg.quality)
	}
	if cfg.fps != 15 {
		t.Errorf("expected fps=15, got %f", cfg.fps)
	}
}

func TestParseArgsQualityOnly(t *testing.T) {
	cfg, err := parseArgs([]string{"video.mp4", "80"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.quality != 80 {
		t.Errorf("expected quality=80, got %d", cfg.quality)
	}
	if cfg.fps != defaultFPS {
		t.Errorf("expected default fps, got %f", cfg.fps)
	}
}

func TestParseArgsInvalidQuality(t *testing.T) {
	_, err := parseArgs([]string{"video.mp4", "101"})
	if err == nil {
		t.Error("expected error for quality > 100")
	}
}

func TestParseArgsInvalidFPS(t *testing.T) {
	_, err := parseArgs([]string{"video.mp4", "70", "0"})
	if err == nil {
		t.Error("expected error for fps=0")
	}
}

func TestParseArgsInvalidQualityString(t *testing.T) {
	_, err := parseArgs([]string{"video.mp4", "abc"})
	if err == nil {
		t.Error("expected error for non-numeric quality")
	}
}

func TestIsVideoFile(t *testing.T) {
	videoFiles := []string{"a.mp4", "b.avi", "c.mov", "d.mkv", "e.webm", "f.flv"}
	for _, f := range videoFiles {
		if !isVideoFile(f) {
			t.Errorf("expected %s to be detected as video", f)
		}
	}
	imageFiles := []string{"a.jpg", "b.png", "c.jpeg", "d.gif"}
	for _, f := range imageFiles {
		if isVideoFile(f) {
			t.Errorf("expected %s NOT to be detected as video", f)
		}
	}
}

func TestAskPlaybackAction(t *testing.T) {
	base := videoPlaybackSettings{fps: 8, quality: 70, colored: true}
	out := &bytes.Buffer{}
	next, replay, quit := askPlaybackAction(strings.NewReader("fps 12\n"), out, base)
	if !replay {
		t.Error("expected replay=true")
	}
	if quit {
		t.Error("expected quit=false")
	}
	if next.fps != 12.0 {
		t.Errorf("expected fps=12, got %f", next.fps)
	}
}

func TestAskPlaybackActionShowScreenSize(t *testing.T) {
	base := videoPlaybackSettings{fps: 8, quality: 70, colored: true}
	out := &bytes.Buffer{}
	next, replay, quit := askPlaybackAction(strings.NewReader("tela\nsair\n"), out, base)
	if replay {
		t.Error("expected replay=false")
	}
	if !quit {
		t.Error("expected quit=true")
	}
	if next != base {
		t.Error("expected settings unchanged")
	}
	if !strings.Contains(out.String(), "tela atual:") {
		t.Error("expected screen size output")
	}
}

func TestParsePlaybackCommand(t *testing.T) {
	base := videoPlaybackSettings{fps: 8, quality: 70, colored: true}

	next, replay, quit, err := parsePlaybackCommand("\n", base)
	if err != nil || !replay || quit || next != base {
		t.Error("empty input should replay with same settings")
	}

	next, replay, quit, err = parsePlaybackCommand("fps 12\n", base)
	if err != nil || !replay || quit || next.fps != 12.0 {
		t.Error("fps command failed")
	}

	next, replay, quit, err = parsePlaybackCommand("qualidade 0\n", base)
	if err != nil || !replay || next.quality != 0 {
		t.Error("qualidade command failed")
	}

	next, replay, quit, err = parsePlaybackCommand("cores bw\n", base)
	if err != nil || !replay || next.colored {
		t.Error("cores bw command failed")
	}

	next, replay, quit, err = parsePlaybackCommand("cores Color Color Color\n", base)
	if err != nil || !replay || !next.colored {
		t.Error("cores Color command failed")
	}

	next, replay, quit, err = parsePlaybackCommand("sair\n", base)
	if err != nil || replay || !quit || next != base {
		t.Error("sair command failed")
	}

	_, _, _, err = parsePlaybackCommand("qualidade 101\n", base)
	if err == nil {
		t.Error("expected error for quality > 100")
	}

	_, _, _, err = parsePlaybackCommand("fps 0\n", base)
	if err == nil {
		t.Error("expected error for fps=0")
	}
}

func TestScaleBetween(t *testing.T) {
	if scaleBetween(4, 100, 0) != 4 {
		t.Error("scaleBetween min failed")
	}
	if scaleBetween(4, 100, 1) != 100 {
		t.Error("scaleBetween max failed")
	}
	if scaleBetween(4, 100, 0.5) != 52 {
		t.Error("scaleBetween 0.5 failed")
	}
	if scaleBetween(4, 100, -1) != 4 {
		t.Error("scaleBetween below min failed")
	}
	if scaleBetween(4, 100, 2) != 100 {
		t.Error("scaleBetween above max failed")
	}
}

func TestApplyVideoSettingsForScreen(t *testing.T) {
	opt := convertDefaultOptions
	settings := videoPlaybackSettings{fps: 8, quality: 0, colored: false}
	applyVideoSettingsForScreen(&opt, settings, 120, 40)
	if opt.FixedWidth != 4 {
		t.Errorf("expected FixedWidth=4, got %d", opt.FixedWidth)
	}
	if opt.FixedHeight != 2 {
		t.Errorf("expected FixedHeight=2, got %d", opt.FixedHeight)
	}
	if opt.Colored {
		t.Error("expected Colored=false")
	}

	settings.quality = 100
	settings.colored = true
	applyVideoSettingsForScreen(&opt, settings, 120, 40)
	if opt.FixedWidth != 120 {
		t.Errorf("expected FixedWidth=120, got %d", opt.FixedWidth)
	}
	if opt.FixedHeight != 40 {
		t.Errorf("expected FixedHeight=40, got %d", opt.FixedHeight)
	}
	if !opt.Colored {
		t.Error("expected Colored=true")
	}
}

func TestCenterASCIIFrame(t *testing.T) {
	centered := centerASCIIFrame("AA\nBB\n", 6, 4)
	if centered != "\n  AA  \n  BB  \n\n" {
		t.Errorf("unexpected centering: %q", centered)
	}
}

func TestVisibleWidth(t *testing.T) {
	if visibleWidth("abc") != 3 {
		t.Error("plain text width failed")
	}
	if visibleWidth("\u001b[38;2;255;0;0ma\u001b[0m") != 1 {
		t.Error("ANSI-colored text width failed")
	}
}

func TestIsShowScreenSizeCommand(t *testing.T) {
	if !isShowScreenSizeCommand("tela") {
		t.Error("expected 'tela' to be screen size command")
	}
	if !isShowScreenSizeCommand("size") {
		t.Error("expected 'size' to be screen size command")
	}
	if isShowScreenSizeCommand("fps 10") {
		t.Error("expected 'fps 10' NOT to be screen size command")
	}
}
