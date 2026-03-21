package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Andrelbmachado/media2ascii/convert"
	termaccess "github.com/Andrelbmachado/media2ascii/terminal"
)

const (
	minQuality     = 0
	maxQuality     = 100
	defaultQuality = 100
	defaultFPS     = 8.0
)

var convertDefaultOptions = convert.DefaultOptions

type cliConfig struct {
	file    string
	quality int
	fps     float64
	colored bool
	isVideo bool
}

type videoPlaybackSettings struct {
	fps     float64
	quality int
	colored bool
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
		if err := playVideoAsASCII(converter, cfg); err != nil {
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
//	media2ascii <arquivo> [qualidade] [fps]
func parseArgs(args []string) (cliConfig, error) {
	cfg := cliConfig{
		quality: defaultQuality,
		fps:     defaultFPS,
		colored: true,
	}

	if len(args) == 0 {
		return cfg, errors.New("informe o arquivo de imagem ou vídeo")
	}
	cfg.file = args[0]
	cfg.isVideo = isVideoFile(cfg.file)

	if len(args) >= 2 {
		q, err := strconv.Atoi(args[1])
		if err != nil || q < minQuality || q > maxQuality {
			return cfg, fmt.Errorf("qualidade inválida '%s': use um número entre 0 e 100", args[1])
		}
		cfg.quality = q
	}

	if len(args) >= 3 {
		f, err := strconv.ParseFloat(args[2], 64)
		if err != nil || f <= 0 {
			return cfg, fmt.Errorf("fps inválido '%s': use um número maior que 0", args[2])
		}
		cfg.fps = f
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

func playVideoAsASCII(converter *convert.ImageConverter, cfg cliConfig) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return errors.New(missingToolGuidance("ffmpeg"))
	}

	settings := videoPlaybackSettings{
		fps:     cfg.fps,
		quality: cfg.quality,
		colored: cfg.colored,
	}

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)
	defer signal.Stop(interruptChannel)

	fmt.Fprintln(os.Stdout, "Iniciando conversão... Pressione Ctrl+C para parar.")

	for {
		opts := buildConvertOptions(cfg)
		screenWidth, screenHeight := getTerminalSizeFallback(120, 40)
		applyVideoSettingsForScreen(opts, settings, screenWidth, screenHeight)

		frameCh, err := streamVideoFrames(cfg.file, settings.fps, opts)
		if err != nil {
			return err
		}

		frameInterval := time.Duration(float64(time.Second) / settings.fps)
		interrupted := playFrames(frameCh, frameInterval, os.Stdout, interruptChannel, screenWidth, screenHeight)

		if interrupted {
			fmt.Fprintln(os.Stdout, "\nPlayback interrompido.")
			break
		}

		newSettings, replay, quit := askPlaybackAction(os.Stdin, os.Stdout, settings)
		if quit {
			break
		}
		if replay {
			settings = newSettings
		}
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

func playFrames(frameCh <-chan string, interval time.Duration, out io.Writer, interrupt <-chan os.Signal, screenWidth, screenHeight int) bool {
	buf := bufio.NewWriterSize(out, 4<<20)
	for frame := range frameCh {
		select {
		case <-interrupt:
			return true
		default:
		}

		// Trim trailing newlines: if the frame fills the terminal height, the
		// final \n scrolls the terminal one line each frame, causing the video
		// to drift off-screen over time.
		frame = strings.TrimRight(frame, "\n")

		buf.WriteString("\033[H\033[2J")
		buf.WriteString(frame)
		buf.Flush()

		timer := time.NewTimer(interval)
		select {
		case <-interrupt:
			timer.Stop()
			return true
		case <-timer.C:
		}
	}
	return false
}

func applyVideoSettingsForScreen(options *convert.Options, settings videoPlaybackSettings, screenWidth int, screenHeight int) {
	options.Colored = settings.colored
	options.Reversed = false
	options.Ratio = 1
	options.FitScreen = true
	options.StretchedScreen = false
	options.FixedWidth = -1
	options.FixedHeight = -1
}

func getTerminalSizeFallback(defaultWidth int, defaultHeight int) (int, int) {
	accessor := termaccess.NewTerminalAccessor()
	width, height, err := accessor.ScreenSize()
	if err != nil || width <= 0 || height <= 0 {
		return defaultWidth, defaultHeight
	}
	return width, height
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func askPlaybackAction(in io.Reader, out io.Writer, current videoPlaybackSettings) (videoPlaybackSettings, bool, bool) {
	reader := bufio.NewReader(in)
	for {
		fmt.Fprintf(out,
			"\nENTER=replay | sair=stop | tela | fps <n> | qualidade <0-100> | cores <BW|Color> (atual: fps=%.2f, qualidade=%d, cores=%s): ",
			current.fps,
			current.quality,
			colorModeLabel(current.colored),
		)
		line, err := reader.ReadString('\n')
		if err != nil {
			return current, false, true
		}

		if isShowScreenSizeCommand(line) {
			screenWidth, screenHeight := getTerminalSizeFallback(120, 40)
			fmt.Fprintf(out, "tela atual: %dx%d\n", screenWidth, screenHeight)
			continue
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
	if lower == "sair" || lower == "quit" || lower == "exit" || lower == "stop" {
		return current, false, true, nil
	}

	if strings.HasPrefix(lower, "fps") {
		parts := strings.Fields(lower)
		if len(parts) != 2 {
			return current, false, false, errors.New("use: fps <numero>")
		}
		fpsValue, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || fpsValue <= 0 {
			return current, false, false, errors.New("fps invalido: use numero maior que 0")
		}
		updated := current
		updated.fps = fpsValue
		return updated, true, false, nil
	}

	if strings.HasPrefix(lower, "qualidade") {
		parts := strings.Fields(lower)
		if len(parts) != 2 {
			return current, false, false, errors.New("use: qualidade <0-100>")
		}
		qualityValue, err := strconv.Atoi(parts[1])
		if err != nil || qualityValue < minQuality || qualityValue > maxQuality {
			return current, false, false, errors.New("qualidade invalida: use inteiro entre 0 e 100")
		}
		updated := current
		updated.quality = qualityValue
		return updated, true, false, nil
	}

	if strings.HasPrefix(lower, "cores") {
		value := strings.TrimSpace(trimmed[len("cores"):])
		valueLower := strings.ToLower(value)
		if valueLower == "" {
			return current, false, false, errors.New("use: cores BW ou cores Color")
		}
		updated := current
		if strings.Contains(valueLower, "bw") || strings.Contains(valueLower, "black") || strings.Contains(valueLower, "preto") {
			updated.colored = false
			return updated, true, false, nil
		}
		if strings.Contains(valueLower, "color") || strings.Contains(valueLower, "cor") {
			updated.colored = true
			return updated, true, false, nil
		}
		return current, false, false, errors.New("valor de cores invalido: use BW ou Color")
	}

	return current, false, false, errors.New("comando invalido. Use ENTER, sair, tela, fps <n>, qualidade <0-100>, cores <BW|Color>")
}

func isShowScreenSizeCommand(input string) bool {
	value := strings.ToLower(strings.TrimSpace(input))
	return value == "tela" || value == "size"
}

func colorModeLabel(colored bool) string {
	if colored {
		return "Color"
	}
	return "BW"
}

func missingToolGuidance(tool string) string {
	switch tool {
	case "ffmpeg":
		return "ffmpeg não encontrado no PATH. Instale com: brew install ffmpeg (macOS) ou apt install ffmpeg (Linux)"
	default:
		return tool + " não está disponível no PATH"
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `media2ascii - Converta imagens e vídeos em arte ASCII

Uso:
  media2ascii <arquivo> [qualidade] [fps]

Exemplos:
  media2ascii video.mp4
  media2ascii video.mp4 70 10
  media2ascii video.mp4 50 25
  media2ascii imagem.jpg
  media2ascii imagem.jpg 90

Parâmetros:
  qualidade  Número de 0 (mínima) a 100 (máxima). Padrão: 70
  fps        Frames por segundo para vídeo. Padrão: 8

Requisitos para vídeo:
  ffmpeg deve estar instalado (brew install ffmpeg)

Durante a reprodução:
  Ctrl+C             Parar imediatamente
  ENTER              Replay
  fps <n>            Alterar FPS
  qualidade <0-100>  Alterar qualidade
  cores BW|Color     Alterar modo de cores
  sair               Sair`)
}
