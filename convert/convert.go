// Package convert can convert a image to ascii string or matrix
package convert

import (
	"bytes"
	"encoding/binary"
	"github.com/Andrelbmachado/media2ascii/ascii"
	"image"
	"image/color"
	// Support decode jpeg image
	_ "image/jpeg"
	// Support deocde the png image
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Options to convert the image to ASCII
type Options struct {
	Ratio           float64
	FixedWidth      int
	FixedHeight     int
	FitScreen       bool // only work on terminal
	StretchedScreen bool // only work on terminal
	Colored         bool // only work on terminal
	Reversed        bool
}

// DefaultOptions for convert image
var DefaultOptions = Options{
	Ratio:           1,
	FixedWidth:      -1,
	FixedHeight:     -1,
	FitScreen:       true,
	Colored:         true,
	Reversed:        false,
	StretchedScreen: false,
}

// NewImageConverter create a new image converter
func NewImageConverter() *ImageConverter {
	return &ImageConverter{
		resizeHandler:  NewResizeHandler(),
		pixelConverter: ascii.NewPixelConverter(),
	}
}

// Converter define the convert image basic operations
type Converter interface {
	Image2ASCIIMatrix(image image.Image, imageConvertOptions *Options) []string
	Image2ASCIIString(image image.Image, options *Options) string
	ImageFile2ASCIIMatrix(imageFilename string, option *Options) []string
	ImageFile2ASCIIString(imageFilename string, option *Options) string
	Image2PixelASCIIMatrix(image image.Image, imageConvertOptions *Options) [][]ascii.CharPixel
	ImageFile2PixelASCIIMatrix(image image.Image, imageConvertOptions *Options) [][]ascii.CharPixel
}

// ImageConverter implement the Convert interface, and responsible
// to image conversion
type ImageConverter struct {
	resizeHandler  ResizeHandler
	pixelConverter ascii.PixelConverter
}

// Image2CharPixelMatrix convert a image to a pixel ascii matrix
func (converter *ImageConverter) Image2CharPixelMatrix(image image.Image, imageConvertOptions *Options) [][]ascii.CharPixel {
	newImage := converter.resizeHandler.ScaleImage(image, imageConvertOptions)
	sz := newImage.Bounds()
	newWidth := sz.Max.X
	newHeight := sz.Max.Y
	pixelASCIIs := make([][]ascii.CharPixel, 0, newHeight)
	for i := 0; i < int(newHeight); i++ {
		line := make([]ascii.CharPixel, 0, newWidth)
		for j := 0; j < int(newWidth); j++ {
			pixel := color.NRGBAModel.Convert(newImage.At(j, i))
			// Convert the pixel to ascii char
			pixelConvertOptions := ascii.NewOptions()
			pixelConvertOptions.Colored = imageConvertOptions.Colored
			pixelConvertOptions.Reversed = imageConvertOptions.Reversed
			pixelASCII := converter.pixelConverter.ConvertPixelToPixelASCII(pixel, &pixelConvertOptions)
			line = append(line, pixelASCII)
		}
		pixelASCIIs = append(pixelASCIIs, line)
	}
	return pixelASCIIs
}

// ImageFile2CharPixelMatrix convert a image to a pixel ascii matrix
func (converter *ImageConverter) ImageFile2CharPixelMatrix(imageFilename string, imageConvertOptions *Options) [][]ascii.CharPixel {
	img, err := OpenImageFile(imageFilename)
	if err != nil {
		log.Fatal("open image failed : " + err.Error())
	}
	return converter.Image2CharPixelMatrix(img, imageConvertOptions)
}

// Image2ASCIIMatrix converts a image to ASCII matrix
func (converter *ImageConverter) Image2ASCIIMatrix(image image.Image, imageConvertOptions *Options) []string {
	// Resize the convert first
	newImage := converter.resizeHandler.ScaleImage(image, imageConvertOptions)
	sz := newImage.Bounds()
	newWidth := sz.Max.X
	newHeight := sz.Max.Y
	rawCharValues := make([]string, 0, int(newWidth*newHeight+newWidth))
	for i := 0; i < int(newHeight); i++ {
		for j := 0; j < int(newWidth); j++ {
			pixel := color.NRGBAModel.Convert(newImage.At(j, i))
			// Convert the pixel to ascii char
			pixelConvertOptions := ascii.NewOptions()
			pixelConvertOptions.Colored = imageConvertOptions.Colored
			pixelConvertOptions.Reversed = imageConvertOptions.Reversed
			rawChar := converter.pixelConverter.ConvertPixelToASCII(pixel, &pixelConvertOptions)
			rawCharValues = append(rawCharValues, rawChar)
		}
		rawCharValues = append(rawCharValues, "\n")
	}
	return rawCharValues
}

// Image2ASCIIString converts a image to ascii matrix, and the join the matrix to a string
func (converter *ImageConverter) Image2ASCIIString(image image.Image, options *Options) string {
	convertedPixelASCII := converter.Image2ASCIIMatrix(image, options)
	var buffer bytes.Buffer

	for i := 0; i < len(convertedPixelASCII); i++ {
		buffer.WriteString(convertedPixelASCII[i])
	}
	return buffer.String()
}

// ImageFile2ASCIIMatrix converts a image file to ascii matrix
func (converter *ImageConverter) ImageFile2ASCIIMatrix(imageFilename string, option *Options) []string {
	img, err := OpenImageFile(imageFilename)
	if err != nil {
		log.Fatal("open image failed : " + err.Error())
	}
	return converter.Image2ASCIIMatrix(img, option)
}

// ImageFile2ASCIIString converts a image file to ascii string
func (converter *ImageConverter) ImageFile2ASCIIString(imageFilename string, option *Options) string {
	img, err := OpenImageFile(imageFilename)
	if err != nil {
		log.Fatal("open image failed : " + err.Error())
	}
	return converter.Image2ASCIIString(img, option)
}

// OpenImageFile open a image and return a image object
func OpenImageFile(imageFilename string) (image.Image, error) {
	f, err := os.Open(imageFilename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read EXIF orientation for JPEG files before decoding.
	orientation := 1
	ext := strings.ToLower(filepath.Ext(imageFilename))
	if ext == ".jpg" || ext == ".jpeg" {
		orientation = readExifOrientation(f)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	return applyExifOrientation(img, orientation), nil
}

// readExifOrientation reads the EXIF orientation tag from a JPEG file.
// Returns 1 (no transformation) if not found or on any error.
// Resets the file position to the beginning after reading.
func readExifOrientation(f *os.File) int {
	buf := make([]byte, 65536)
	n, _ := f.Read(buf)
	f.Seek(0, 0)
	data := buf[:n]

	if len(data) < 3 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1 // not a JPEG
	}

	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		i += 2
		// Markers with no length field
		if marker == 0xD8 || marker == 0xD9 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}
		if i+2 > len(data) {
			return 1
		}
		segLen := int(data[i])<<8 | int(data[i+1])
		if segLen < 2 {
			return 1
		}
		segEnd := i + segLen
		if segEnd > len(data) {
			return 1
		}
		segData := data[i+2 : segEnd]
		i = segEnd

		if marker != 0xE1 { // not APP1
			continue
		}
		if len(segData) < 6 || string(segData[:6]) != "Exif\x00\x00" {
			continue
		}

		tiff := segData[6:]
		if len(tiff) < 8 {
			return 1
		}

		var bo binary.ByteOrder
		switch {
		case tiff[0] == 'I' && tiff[1] == 'I':
			bo = binary.LittleEndian
		case tiff[0] == 'M' && tiff[1] == 'M':
			bo = binary.BigEndian
		default:
			return 1
		}
		if bo.Uint16(tiff[2:4]) != 42 {
			return 1
		}

		ifdOffset := int(bo.Uint32(tiff[4:8]))
		if ifdOffset+2 > len(tiff) {
			return 1
		}
		numEntries := int(bo.Uint16(tiff[ifdOffset : ifdOffset+2]))
		base := ifdOffset + 2
		for j := 0; j < numEntries; j++ {
			off := base + j*12
			if off+12 > len(tiff) {
				break
			}
			if bo.Uint16(tiff[off:off+2]) == 0x0112 { // Orientation tag
				return int(bo.Uint16(tiff[off+8 : off+10]))
			}
		}
		return 1
	}
	return 1
}

// applyExifOrientation rotates/flips img according to the EXIF orientation value.
func applyExifOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return flipH(img)
	case 3:
		return rotate180(img)
	case 4:
		return flipV(img)
	case 5:
		return flipH(rotateCW90(img))
	case 6:
		return rotateCW90(img)
	case 7:
		return flipH(rotateCCW90(img))
	case 8:
		return rotateCCW90(img)
	}
	return img
}

func rotateCW90(src image.Image) image.Image {
	b := src.Bounds()
	W := b.Max.X - b.Min.X
	H := b.Max.Y - b.Min.Y
	dst := image.NewNRGBA(image.Rect(0, 0, H, W))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			dst.Set(H-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotateCCW90(src image.Image) image.Image {
	b := src.Bounds()
	W := b.Max.X - b.Min.X
	H := b.Max.Y - b.Min.Y
	dst := image.NewNRGBA(image.Rect(0, 0, H, W))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			dst.Set(y, W-1-x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotate180(src image.Image) image.Image {
	b := src.Bounds()
	W := b.Max.X - b.Min.X
	H := b.Max.Y - b.Min.Y
	dst := image.NewNRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			dst.Set(W-1-x, H-1-y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func flipH(src image.Image) image.Image {
	b := src.Bounds()
	W := b.Max.X - b.Min.X
	H := b.Max.Y - b.Min.Y
	dst := image.NewNRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			dst.Set(W-1-x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func flipV(src image.Image) image.Image {
	b := src.Bounds()
	W := b.Max.X - b.Min.X
	H := b.Max.Y - b.Min.Y
	dst := image.NewNRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			dst.Set(x, H-1-y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
