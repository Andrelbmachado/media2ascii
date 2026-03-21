package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Andrelbmachado/media2ascii/convert"
	termaccess "github.com/Andrelbmachado/media2ascii/terminal"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"golang.org/x/term"
)

const (
	minQuality     = 0
	maxQuality     = 100
	defaultQuality = 100
	defaultFPS     = 30.0
)

var ansiSequenceRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var convertDefaultOptions = convert.DefaultOptions

type cliConfig struct {
	file    string
	quality int
	fps     float64
	colored bool
	isVideo bool
	export  bool
	noAudio bool
}

type videoPlaybackSettings struct {
	fps     float64
	quality int
	colored bool
	audio   bool
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cfg, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erro:", err)
		fmt.Fprintln(os.Stderr)
		usage()
		os.Exit(1)
	}

	converter := convert.NewImageConverter()

	if cfg.isVideo {
		if cfg.export {
			if err := exportVideoAsASCII(cfg); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			return
		}
		if err := playVideoAsASCII(cfg); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		return
	}

	opts := buildConvertOptions(cfg)
	fmt.Print(converter.ImageFile2ASCIIString(cfg.file, opts))
}

// parseArgs parses positional arguments:
//
//	media2ascii <arquivo> [qualidade] [fps] [cores] [audio] [--export]
func parseArgs(args []string) (cliConfig, error) {
	cfg := cliConfig{
		quality: defaultQuality,
		fps:     defaultFPS,
		colored: true,
	}

	// Extract flags from any position
	filtered := args[:0]
	for _, arg := range args {
		switch arg {
		case "--export", "-export":
			cfg.export = true
		case "--no-audio", "-no-audio", "--mute", "-mute":
			cfg.noAudio = true
		default:
			filtered = append(filtered, arg)
		}
	}
	args = filtered

	if len(args) == 0 {
		return cfg, errors.New("informe o arquivo de imagem ou vídeo")
	}
	cfg.file = args[0]
	cfg.isVideo = isVideoFile(cfg.file)

	// args[1] = qualidade
	if len(args) >= 2 {
		q, err := strconv.Atoi(args[1])
		if err != nil || q < minQuality || q > maxQuality {
			return cfg, fmt.Errorf("qualidade inválida '%s': use um número entre 0 e 100", args[1])
		}
		cfg.quality = q
	}

	// args[2] = fps
	if len(args) >= 3 {
		f, err := strconv.ParseFloat(args[2], 64)
		if err != nil || f <= 0 {
			return cfg, fmt.Errorf("fps inválido '%s': use um número maior que 0", args[2])
		}
		cfg.fps = f
	}

	// args[3] = cores (Color/BW)
	if len(args) >= 4 {
		switch strings.ToLower(args[3]) {
		case "color", "colorido", "cor":
			cfg.colored = true
		case "bw", "pb", "preto":
			cfg.colored = false
		default:
			return cfg, fmt.Errorf("cores inválido '%s': use Color ou BW", args[3])
		}
	}

	// args[4] = audio (ON/OFF)
	if len(args) >= 5 {
		switch strings.ToLower(args[4]) {
		case "on", "sim", "yes":
			cfg.noAudio = false
		case "off", "nao", "não", "no":
			cfg.noAudio = true
		default:
			return cfg, fmt.Errorf("audio inválido '%s': use ON ou OFF", args[4])
		}
	}

	return cfg, nil
}

func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm", ".m4v", ".mpeg", ".mpg", ".3gp":
		return true
	}
	return false
}

func buildConvertOptions(cfg cliConfig) *convert.Options {
	return &convert.Options{
		Ratio:           convertDefaultOptions.Ratio,
		FixedWidth:      convertDefaultOptions.FixedWidth,
		FixedHeight:     convertDefaultOptions.FixedHeight,
		FitScreen:       convertDefaultOptions.FitScreen,
		StretchedScreen: convertDefaultOptions.StretchedScreen,
		Colored:         cfg.colored,
		Reversed:        false,
	}
}

func playVideoAsASCII(cfg cliConfig) error {
	ffmpegBin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return errors.New(missingToolGuidance("ffmpeg"))
	}

	settings := videoPlaybackSettings{
		fps:     cfg.fps,
		quality: cfg.quality,
		colored: cfg.colored,
		audio:   !cfg.noAudio,
	}

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)
	defer signal.Stop(interruptChannel)

	// Raw mode: detect spacebar without requiring Enter.
	stdinFd := int(os.Stdin.Fd())
	var rawState *term.State
	setupRaw := func() {
		rawState, _ = term.MakeRaw(stdinFd)
	}
	restoreNormal := func() {
		if rawState != nil {
			term.Restore(stdinFd, rawState)
			rawState = nil
		}
	}
	defer restoreNormal()
	setupRaw()

	// Keypress goroutine: space → pause, Ctrl+C / q → interrupt.
	pauseCh := make(chan struct{}, 1)
	var listenKeys int32 = 1
	go func() {
		buf := make([]byte, 1)
		for {
			if _, err := os.Stdin.Read(buf); err != nil {
				return
			}
			if atomic.LoadInt32(&listenKeys) == 0 {
				continue
			}
			switch buf[0] {
			case ' ':
				select {
				case pauseCh <- struct{}{}:
				default:
				}
			case 3, 'q', 'Q': // Ctrl+C or Q
				interruptChannel <- os.Interrupt
			}
		}
	}()

	fmt.Fprintln(os.Stdout, "Iniciando conversão... Pressione Ctrl+C para parar. ESPAÇO para pausar.")

	for {
		opts := buildConvertOptions(cfg)
		screenWidth, screenHeight := getTerminalSizeFallback(120, 40)
		applyVideoSettingsForScreen(opts, settings, screenWidth, screenHeight)

		// Prepare audio in background.
		audioCh := make(chan *audioPlayer, 1)
		if !settings.audio {
			audioCh <- nil
		} else {
			go func() { audioCh <- newAudioPlayer(cfg.file, ffmpegBin) }()
		}

		frameCh, err := streamVideoFrames(cfg.file, settings.fps, opts)
		if err != nil {
			return err
		}

		var audio *audioPlayer
		select {
		case audio = <-audioCh:
		case <-time.After(15 * time.Second):
		}
		if audio != nil {
			audio.start()
		}

		frameInterval := time.Duration(float64(time.Second) / settings.fps)

		// Inner loop: play → pause → resume, until video ends or user quits.
		done := false
		for !done {
			result := playFrames(frameCh, frameInterval, os.Stdout, interruptChannel, pauseCh, screenWidth, screenHeight)

			switch result {
			case playResultInterrupted:
				if audio != nil {
					audio.stop()
				}
				restoreNormal()
				fmt.Fprintln(os.Stdout, "\nPlayback interrompido.")
				return nil

			case playResultPaused:
				// Keep current frame on screen; show menu below.
				atomic.StoreInt32(&listenKeys, 0)
				restoreNormal()
				fmt.Fprintln(os.Stdout, "")
				newSettings, keepGoing, quit := askPlaybackAction(os.Stdin, os.Stdout, settings)
				if quit {
					if audio != nil {
						audio.stop()
					}
					// Drain remaining frames so goroutines can exit.
					go func() {
						for range frameCh {
						}
					}()
					return nil
				}
				// Apply audio change immediately if toggled.
				if newSettings.audio != settings.audio {
					if audio != nil {
						audio.stop()
						audio = nil
					}
					if newSettings.audio {
						a := newAudioPlayer(cfg.file, ffmpegBin)
						if a != nil {
							a.start()
						}
						audio = a
					}
				}
				settings = newSettings
				_ = keepGoing
				setupRaw()
				atomic.StoreInt32(&listenKeys, 1)
				// Continue consuming remaining frames from same channel.

			case playResultFinished:
				done = true
			}
		}

		if audio != nil {
			audio.stop()
		}

		// End-of-video menu.
		atomic.StoreInt32(&listenKeys, 0)
		restoreNormal()
		newSettings, replay, quit := askPlaybackAction(os.Stdin, os.Stdout, settings)
		if quit {
			break
		}
		if replay {
			settings = newSettings
		}
		setupRaw()
		atomic.StoreInt32(&listenKeys, 1)
	}

	return nil
}

type frameJob struct {
	index int
	data  []byte
}

type frameResult struct {
	index int
	ascii string
}

func streamVideoFrames(videoFile string, fps float64, options *convert.Options) (<-chan string, error) {
	ffmpegBin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, errors.New(missingToolGuidance("ffmpeg"))
	}

	cmd := exec.Command(ffmpegBin,
		"-hide_banner", "-loglevel", "error",
		"-i", videoFile,
		"-vf", fmt.Sprintf("fps=%g", fps),
		"-f", "image2pipe",
		"-vcodec", "png",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("falha ao iniciar ffmpeg: %w", err)
	}

	numWorkers := runtime.NumCPU()
	bufferSize := numWorkers * 4

	jobCh := make(chan frameJob, numWorkers)
	resultCh := make(chan frameResult, bufferSize)
	orderedCh := make(chan string, bufferSize)

	// Worker pool: decode PNG bytes + convert to ASCII in parallel
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localConverter := convert.NewImageConverter()
			for job := range jobCh {
				img, decErr := png.Decode(bytes.NewReader(job.data))
				if decErr != nil {
					continue
				}
				asciiStr := localConverter.Image2ASCIIString(img, options)
				resultCh <- frameResult{job.index, asciiStr}
			}
		}()
	}

	// Reader: extract individual PNG frames from ffmpeg pipe stream
	go func() {
		defer close(jobCh)
		index := 0
		reader := bufio.NewReaderSize(stdout, 1<<20)
		for {
			pngBytes, readErr := readOnePNG(reader)
			if readErr != nil {
				break
			}
			jobCh <- frameJob{index, pngBytes}
			index++
		}
		_ = cmd.Wait()
	}()

	// Close resultCh after all workers finish
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Reorder frames to maintain correct display order
	go func() {
		defer close(orderedCh)
		pending := make(map[int]string)
		nextIndex := 0
		for r := range resultCh {
			pending[r.index] = r.ascii
			for {
				if asciiStr, ok := pending[nextIndex]; ok {
					orderedCh <- asciiStr
					delete(pending, nextIndex)
					nextIndex++
				} else {
					break
				}
			}
		}
		// Flush any remaining frames in order
		for {
			if asciiStr, ok := pending[nextIndex]; ok {
				orderedCh <- asciiStr
				delete(pending, nextIndex)
				nextIndex++
			} else {
				break
			}
		}
	}()

	return orderedCh, nil
}

var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// readOnePNG reads exactly one PNG image from r by parsing PNG chunks.
func readOnePNG(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer

	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		return nil, err
	}
	if !bytes.Equal(sig, pngSignature) {
		return nil, errors.New("invalid PNG signature")
	}
	buf.Write(sig)

	for {
		header := make([]byte, 8)
		if _, err := io.ReadFull(r, header); err != nil {
			return nil, err
		}
		dataLen := binary.BigEndian.Uint32(header[:4])
		buf.Write(header)

		rest := make([]byte, int(dataLen)+4)
		if _, err := io.ReadFull(r, rest); err != nil {
			return nil, err
		}
		buf.Write(rest)

		if string(header[4:8]) == "IEND" {
			break
		}
	}

	return buf.Bytes(), nil
}

type playResult int

const (
	playResultFinished    playResult = iota
	playResultInterrupted            // Ctrl+C
	playResultPaused                 // spacebar
)

func playFrames(frameCh <-chan string, interval time.Duration, out io.Writer, interrupt <-chan os.Signal, pause <-chan struct{}, screenWidth, screenHeight int) playResult {
	for frame := range frameCh {
		select {
		case <-interrupt:
			return playResultInterrupted
		case <-pause:
			return playResultPaused
		default:
		}

		centeredFrame := centerASCIIFrame(frame, screenWidth, screenHeight)
		fmt.Fprint(out, "\033[H\033[2J")
		fmt.Fprint(out, centeredFrame)

		timer := time.NewTimer(interval)
		select {
		case <-interrupt:
			timer.Stop()
			return playResultInterrupted
		case <-pause:
			timer.Stop()
			return playResultPaused
		case <-timer.C:
		}
	}
	return playResultFinished
}

func applyVideoSettingsForScreen(options *convert.Options, settings videoPlaybackSettings, screenWidth int, screenHeight int) {
	options.Colored = settings.colored
	options.Reversed = false
	options.Ratio = 1
	options.FitScreen = false
	options.StretchedScreen = false

	qualityRatio := float64(settings.quality) / 100
	options.FixedWidth = scaleBetween(4, maxInt(4, screenWidth-2), qualityRatio)
	options.FixedHeight = scaleBetween(2, maxInt(2, screenHeight-2), qualityRatio)
}

func getTerminalSizeFallback(defaultWidth int, defaultHeight int) (int, int) {
	accessor := termaccess.NewTerminalAccessor()
	width, height, err := accessor.ScreenSize()
	if err != nil || width <= 0 || height <= 0 {
		return defaultWidth, defaultHeight
	}
	return width, height
}

func scaleBetween(minimum int, maximum int, ratio float64) int {
	if maximum <= minimum {
		return minimum
	}
	clamped := math.Min(math.Max(ratio, 0), 1)
	return minimum + int(math.Round(float64(maximum-minimum)*clamped))
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func centerASCIIFrame(frameASCII string, screenWidth int, screenHeight int) string {
	trimmed := strings.TrimRight(frameASCII, "\n")
	frameLines := []string{""}
	if trimmed != "" {
		frameLines = strings.Split(trimmed, "\n")
	}

	frameHeight := len(frameLines)
	padTop := maxInt((screenHeight-frameHeight)/2, 0)
	padBottom := maxInt(screenHeight-padTop-frameHeight, 0)

	var builder strings.Builder
	for i := 0; i < padTop; i++ {
		builder.WriteString("\n")
	}

	for _, line := range frameLines {
		lineWidth := visibleWidth(line)
		padLeft := maxInt((screenWidth-lineWidth)/2, 0)
		padRight := maxInt(screenWidth-padLeft-lineWidth, 0)
		builder.WriteString(strings.Repeat(" ", padLeft))
		builder.WriteString(line)
		builder.WriteString(strings.Repeat(" ", padRight))
		builder.WriteString("\n")
	}

	for i := 0; i < padBottom; i++ {
		builder.WriteString("\n")
	}

	result := builder.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result
}

func visibleWidth(value string) int {
	plain := ansiSequenceRegexp.ReplaceAllString(value, "")
	return len([]rune(plain))
}

func askPlaybackAction(in io.Reader, out io.Writer, current videoPlaybackSettings) (videoPlaybackSettings, bool, bool) {
	reader := bufio.NewReader(in)
	for {
		fmt.Fprintf(out,
			"\nReplay = ENTER | Sair = Q | Qualidade = 0->100 | FPS = 0->30 | Cores = BW or Color | Audio = ON or OFF  [qualidade=%d fps=%.0f cores=%s audio=%s]: ",
			current.quality,
			current.fps,
			colorModeLabel(current.colored),
			audioLabel(current.audio),
		)
		line, err := reader.ReadString('\n')
		if err != nil {
			return current, false, true
		}

		next, replay, quit, parseErr := parsePlaybackCommand(line, current)
		if parseErr != nil {
			fmt.Fprintf(out, "%s\n", parseErr.Error())
			continue
		}

		if replay {
			return next, true, false
		}
		if quit {
			return current, false, true
		}
	}
}

func parsePlaybackCommand(input string, current videoPlaybackSettings) (videoPlaybackSettings, bool, bool, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return current, true, false, nil
	}

	lower := strings.ToLower(trimmed)

	if lower == "q" || lower == "quit" || lower == "exit" || lower == "sair" || lower == "stop" {
		return current, false, true, nil
	}

	if strings.HasPrefix(lower, "qualidade") {
		parts := strings.Fields(lower)
		if len(parts) != 2 {
			return current, false, false, errors.New("use: qualidade <0-100>")
		}
		v, err := strconv.Atoi(parts[1])
		if err != nil || v < minQuality || v > maxQuality {
			return current, false, false, errors.New("qualidade inválida: use inteiro entre 0 e 100")
		}
		updated := current
		updated.quality = v
		return updated, true, false, nil
	}

	if strings.HasPrefix(lower, "fps") {
		parts := strings.Fields(lower)
		if len(parts) != 2 {
			return current, false, false, errors.New("use: fps <numero>")
		}
		v, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || v <= 0 {
			return current, false, false, errors.New("fps inválido: use número maior que 0")
		}
		updated := current
		updated.fps = v
		return updated, true, false, nil
	}

	if strings.HasPrefix(lower, "cores") {
		val := strings.ToLower(strings.TrimSpace(trimmed[len("cores"):]))
		updated := current
		switch {
		case strings.Contains(val, "bw") || strings.Contains(val, "pb"):
			updated.colored = false
		case strings.Contains(val, "color") || strings.Contains(val, "cor"):
			updated.colored = true
		default:
			return current, false, false, errors.New("use: cores BW ou cores Color")
		}
		return updated, true, false, nil
	}

	if strings.HasPrefix(lower, "audio") {
		val := strings.ToLower(strings.TrimSpace(trimmed[len("audio"):]))
		updated := current
		switch val {
		case "on", "sim", "yes":
			updated.audio = true
		case "off", "nao", "não", "no":
			updated.audio = false
		default:
			return current, false, false, errors.New("use: audio ON ou audio OFF")
		}
		return updated, true, false, nil
	}

	return current, false, false, errors.New("comando inválido. Use ENTER, Q, qualidade <n>, fps <n>, cores <BW|Color>, audio <ON|OFF>")
}

func colorModeLabel(colored bool) string {
	if colored {
		return "Color"
	}
	return "BW"
}

func audioLabel(audio bool) string {
	if audio {
		return "ON"
	}
	return "OFF"
}

func missingToolGuidance(tool string) string {
	switch tool {
	case "ffmpeg":
		return "ffmpeg não encontrado no PATH. Instale com: brew install ffmpeg (macOS) ou apt install ffmpeg (Linux)"
	default:
		return tool + " não está disponível no PATH"
	}
}

const (
	fontCharW   = 7  // basicfont.Face7x13 char width
	fontCharH   = 13 // basicfont.Face7x13 char height
	fontAscent  = 11 // pixels above baseline
	renderScale = 2  // upscale factor for readability
)

// coloredChar holds a rune and its foreground color parsed from ANSI codes.
type coloredChar struct {
	ch  rune
	col color.RGBA
}

var ansiEscRegexp = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// parseColoredLine splits a line with ANSI color codes into colored characters.
func parseColoredLine(line string) []coloredChar {
	white := color.RGBA{255, 255, 255, 255}
	current := white
	var result []coloredChar
	for len(line) > 0 {
		loc := ansiEscRegexp.FindStringIndex(line)
		if loc == nil {
			for _, ch := range line {
				result = append(result, coloredChar{ch, current})
			}
			break
		}
		for _, ch := range line[:loc[0]] {
			result = append(result, coloredChar{ch, current})
		}
		match := ansiEscRegexp.FindStringSubmatch(line[loc[0]:loc[1]])
		if len(match) > 1 {
			current = parseANSICode(match[1], current)
		}
		line = line[loc[1]:]
	}
	return result
}

// parseANSICode updates the current color based on ANSI escape code content.
func parseANSICode(code string, current color.RGBA) color.RGBA {
	white := color.RGBA{255, 255, 255, 255}
	if code == "" || code == "0" {
		return white
	}
	parts := strings.Split(code, ";")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err == nil {
			nums = append(nums, n)
		}
	}
	for i := 0; i < len(nums); i++ {
		switch nums[i] {
		case 0:
			return white
		case 38:
			if i+4 < len(nums) && nums[i+1] == 2 {
				return color.RGBA{R: uint8(nums[i+2]), G: uint8(nums[i+3]), B: uint8(nums[i+4]), A: 255}
			}
		}
	}
	return current
}

// renderFrameToImage converts an ASCII frame (with ANSI colors) to an RGBA image.
func renderFrameToImage(asciiFrame string) *image.RGBA {
	lines := strings.Split(strings.TrimRight(asciiFrame, "\n"), "\n")
	parsed := make([][]coloredChar, len(lines))
	maxCols := 0
	for i, line := range lines {
		parsed[i] = parseColoredLine(line)
		if len(parsed[i]) > maxCols {
			maxCols = len(parsed[i])
		}
	}
	if maxCols == 0 {
		maxCols = 1
	}
	numLines := len(lines)
	if numLines == 0 {
		numLines = 1
	}

	baseW := maxCols * fontCharW
	baseH := numLines * fontCharH
	base := image.NewRGBA(image.Rect(0, 0, baseW, baseH))
	draw.Draw(base, base.Bounds(), image.Black, image.Point{}, draw.Src)

	face := basicfont.Face7x13
	for lineIdx, chars := range parsed {
		x := 0
		baseline := lineIdx*fontCharH + fontAscent
		for _, cc := range chars {
			d := &font.Drawer{
				Dst:  base,
				Src:  image.NewUniform(cc.col),
				Face: face,
				Dot:  fixed.P(x, baseline),
			}
			d.DrawString(string(cc.ch))
			x += fontCharW
		}
	}

	// Scale up for readability
	scaledW := baseW * renderScale
	scaledH := baseH * renderScale
	scaled := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))
	for y := 0; y < scaledH; y++ {
		for x := 0; x < scaledW; x++ {
			scaled.Set(x, y, base.At(x/renderScale, y/renderScale))
		}
	}
	return scaled
}

func savePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func createVideoFromFrames(framesDir, outputFile string, fps float64) error {
	ffmpegBin, _ := exec.LookPath("ffmpeg")
	inputPattern := filepath.Join(framesDir, "frame_%06d.png")
	cmd := exec.Command(ffmpegBin,
		"-y",
		"-framerate", fmt.Sprintf("%g", fps),
		"-i", inputPattern,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "fast",
		outputFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg erro: %s", string(out))
	}
	return nil
}

func exportVideoAsASCII(cfg cliConfig) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return errors.New(missingToolGuidance("ffmpeg"))
	}

	videoDir := filepath.Dir(cfg.file)
	videoBase := strings.TrimSuffix(filepath.Base(cfg.file), filepath.Ext(cfg.file))
	framesDir := filepath.Join(videoDir, videoBase+"_ascii_frames")
	outputMP4 := filepath.Join(videoDir, videoBase+"_ascii.mp4")

	if err := os.MkdirAll(framesDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório: %w", err)
	}

	opts := buildConvertOptions(cfg)
	settings := videoPlaybackSettings{fps: cfg.fps, quality: cfg.quality, colored: true}
	// Use a fixed virtual screen size for consistent export resolution
	applyVideoSettingsForScreen(opts, settings, 200, 60)

	frameCh, err := streamVideoFrames(cfg.file, settings.fps, opts)
	if err != nil {
		return err
	}

	fmt.Println("Exportando frames ASCII...")
	frameIndex := 0
	for frame := range frameCh {
		// Save text frame
		txtPath := filepath.Join(framesDir, fmt.Sprintf("frame_%06d.txt", frameIndex))
		_ = os.WriteFile(txtPath, []byte(frame), 0644)

		// Render and save PNG
		img := renderFrameToImage(frame)
		pngPath := filepath.Join(framesDir, fmt.Sprintf("frame_%06d.png", frameIndex))
		if err := savePNG(img, pngPath); err != nil {
			return err
		}

		frameIndex++
		fmt.Printf("\rFrames processados: %d", frameIndex)
	}
	fmt.Printf("\rFrames processados: %d\n", frameIndex)

	if frameIndex == 0 {
		return errors.New("nenhum frame gerado")
	}

	fmt.Println("Gerando MP4...")
	if err := createVideoFromFrames(framesDir, outputMP4, cfg.fps); err != nil {
		return err
	}

	fmt.Printf("Vídeo ASCII salvo em: %s\n", outputMP4)
	fmt.Printf("Frames de texto salvos em: %s\n", framesDir)
	return nil
}

// audioPlayer manages background audio playback during ASCII video rendering.
type audioPlayer struct {
	cmds    []*exec.Cmd
	tmpFile string
}

// newAudioPlayer prepares audio for videoFile. Returns nil if the platform is
// unsupported or the video has no audio track (fails silently).
func newAudioPlayer(videoFile, ffmpegBin string) *audioPlayer {
	switch runtime.GOOS {
	case "darwin":
		return newDarwinAudioPlayer(videoFile, ffmpegBin)
	case "windows":
		return newWindowsAudioPlayer(videoFile, ffmpegBin)
	}
	return nil
}

// newDarwinAudioPlayer extracts audio to a temp WAV file using ffmpeg,
// then plays it via afplay (built-in macOS audio player).
func newDarwinAudioPlayer(videoFile, ffmpegBin string) *audioPlayer {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("m2a_audio_%d.wav", os.Getpid()))
	extractCmd := exec.Command(ffmpegBin,
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", videoFile,
		"-vn", "-acodec", "pcm_s16le", "-ar", "44100", "-ac", "2",
		tmpFile,
	)
	if err := extractCmd.Run(); err != nil {
		return nil
	}
	afplayCmd := exec.Command("afplay", tmpFile)
	return &audioPlayer{cmds: []*exec.Cmd{afplayCmd}, tmpFile: tmpFile}
}

// newWindowsAudioPlayer extracts audio to a temp WAV file using ffmpeg,
// then plays it via PowerShell's System.Media.SoundPlayer. Blocks during extraction.
func newWindowsAudioPlayer(videoFile, ffmpegBin string) *audioPlayer {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("m2a_audio_%d.wav", os.Getpid()))
	extractCmd := exec.Command(ffmpegBin,
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", videoFile,
		"-vn", "-acodec", "pcm_s16le", "-ar", "44100", "-ac", "2",
		tmpFile,
	)
	if err := extractCmd.Run(); err != nil {
		return nil // no audio track or unsupported format
	}
	psCmd := exec.Command("powershell", "-NonInteractive", "-Command",
		fmt.Sprintf(
			`$p=[System.Media.SoundPlayer]::new('%s');$p.Play();while($true){Start-Sleep 1}`,
			tmpFile,
		),
	)
	return &audioPlayer{cmds: []*exec.Cmd{psCmd}, tmpFile: tmpFile}
}

func (p *audioPlayer) start() {
	for _, cmd := range p.cmds {
		_ = cmd.Start()
	}
}

func (p *audioPlayer) stop() {
	for _, cmd := range p.cmds {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
	for _, cmd := range p.cmds {
		if cmd.Process != nil {
			_ = cmd.Wait()
		}
	}
	if p.tmpFile != "" {
		os.Remove(p.tmpFile)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `media2ascii - Converta imagens e vídeos em arte ASCII

Uso:
  media2ascii <arquivo> [qualidade] [fps] [cores] [audio] [--export]

Exemplos:
  media2ascii video.mp4
  media2ascii video.mp4 100 30
  media2ascii video.mp4 100 30 Color ON
  media2ascii video.mp4 100 30 BW OFF --export
  media2ascii imagem.jpg

Parâmetros:
  qualidade  0 a 100. Padrão: 100
  fps        Frames por segundo. Padrão: 30
  cores      Color ou BW. Padrão: Color
  audio      ON ou OFF. Padrão: ON
  --export   Salva os frames em texto e gera um MP4 com a arte ASCII
  --no-audio Atalho para audio OFF

Durante a reprodução:
  ENTER                  Replay
  Q                      Sair
  qualidade <0-100>      Alterar qualidade
  fps <0-30>             Alterar FPS
  cores <BW|Color>       Alterar cores
  audio <ON|OFF>         Ligar/desligar áudio
  Ctrl+C                 Parar imediatamente`)
}
