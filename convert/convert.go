// Package convert can convert a image to ascii string or matrix
package convert

import (
	"bytes"
	"github.com/Andrelbmachado/media2ascii/v2/ascii"
	"github.com/rwcarlsen/goexif/exif"
	"image"
	"image/color"
	// Support decode jpeg image
	_ "image/jpeg"
	// Support deocde the png image
	_ "image/png"
	"log"
	"os"
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

// OpenImageFile opens an image file and returns an image.Image with EXIF
// orientation already applied, so the caller always gets correctly-rotated pixels.
func OpenImageFile(imageFilename string) (image.Image, error) {
	// Read the whole file once; we need the bytes for both EXIF and decoding.
	data, err := os.ReadFile(imageFilename)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Apply EXIF orientation so portrait stays portrait, etc.
	orientation := readExifOrientation(data)
	return applyExifOrientation(img, orientation), nil
}

// readExifOrientation uses goexif to reliably read the EXIF orientation tag
// from raw image bytes. Returns 1 (no-op) when there is no tag or on any error.
func readExifOrientation(data []byte) int {
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return 1
	}
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return 1
	}
	val, err := tag.Int(0)
	if err != nil {
		return 1
	}
	return val
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
