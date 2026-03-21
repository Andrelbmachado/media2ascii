package convert

import (
	"github.com/nfnt/resize"
	"github.com/Andrelbmachado/media2ascii/terminal"
	"image"
	"log"
	"math"
)

// NewResizeHandler create a new resize handler
func NewResizeHandler() ResizeHandler {
	handler := &ImageResizeHandler{
		terminal: terminal.NewTerminalAccessor(),
	}

	initResizeResolver(handler)
	return handler
}

// initResizeResolver register the size resolvers
func initResizeResolver(handler *ImageResizeHandler) {
	sizeResolvers := make([]imageSizeResolver, 0, 5)
	// fixed height or width resolver
	sizeResolvers = append(sizeResolvers, imageSizeResolver{
		match: func(options *Options) bool {
			return options.FixedWidth != -1 || options.FixedHeight != -1
		},
		compute: func(sz image.Rectangle, options *Options, handler *ImageResizeHandler) (width, height int, err error) {
			charW := handler.terminal.CharWidth()
			hasW := options.FixedWidth != -1
			hasH := options.FixedHeight != -1

			if hasW && hasH {
				// Both set: treat as max bounds, fit proportionally (maintain aspect ratio)
				hFromW := int(math.Round(float64(sz.Max.Y) * float64(options.FixedWidth) / float64(sz.Max.X) * charW))
				if hFromW <= options.FixedHeight {
					// Width is the limiting dimension
					width = options.FixedWidth
					height = hFromW
					if height < 1 {
						height = 1
					}
				} else {
					// Height is the limiting dimension
					height = options.FixedHeight
					if charW > 0 {
						width = int(math.Round(float64(sz.Max.X) * float64(options.FixedHeight) / float64(sz.Max.Y) / charW))
					} else {
						width = sz.Max.X
					}
					if width < 1 {
						width = 1
					}
				}
			} else if hasW {
				// Width only: compute char-aspect-corrected height
				width = options.FixedWidth
				height = int(math.Round(float64(sz.Max.Y) * float64(options.FixedWidth) / float64(sz.Max.X) * charW))
				if height < 1 {
					height = 1
				}
			} else {
				// Height only: compute char-aspect-corrected width
				height = options.FixedHeight
				if charW > 0 {
					width = int(math.Round(float64(sz.Max.X) * float64(options.FixedHeight) / float64(sz.Max.Y) / charW))
				} else {
					width = sz.Max.X
				}
				if width < 1 {
					width = 1
				}
			}
			return
		},
	})
	// scaled by ratio
	sizeResolvers = append(sizeResolvers, imageSizeResolver{
		match: func(options *Options) bool {
			return options.Ratio != 1
		},
		compute: func(sz image.Rectangle, options *Options, handler *ImageResizeHandler) (width, height int, err error) {
			ratio := options.Ratio
			width = handler.ScaleWidthByRatio(float64(sz.Max.X), ratio)
			height = handler.ScaleHeightByRatio(float64(sz.Max.Y), ratio)
			return
		},
	})
	// scaled by stretched screen
	sizeResolvers = append(sizeResolvers, imageSizeResolver{
		match: func(options *Options) bool {
			return options.StretchedScreen
		},
		compute: func(sz image.Rectangle, options *Options, handler *ImageResizeHandler) (width, height int, err error) {
			return handler.terminal.ScreenSize()
		},
	})
	// scaled by fit the screen
	sizeResolvers = append(sizeResolvers, imageSizeResolver{
		match: func(options *Options) bool {
			return options.FitScreen
		},
		compute: func(sz image.Rectangle, options *Options, handler *ImageResizeHandler) (width, height int, err error) {
			return handler.CalcProportionalFittingScreenSize(sz)
		},
	})
	// default size
	sizeResolvers = append(sizeResolvers, imageSizeResolver{
		match: func(options *Options) bool {
			return true
		},
		compute: func(sz image.Rectangle, options *Options, handler *ImageResizeHandler) (width, height int, err error) {
			return sz.Max.X, sz.Max.Y, nil
		},
	})

	handler.imageSizeResolvers = sizeResolvers
}

// ResizeHandler define the operation to resize a image
type ResizeHandler interface {
	ScaleImage(image image.Image, options *Options) (newImage image.Image)
}

// ImageResizeHandler implement the ResizeHandler interface and
// responsible for image resizing
type ImageResizeHandler struct {
	terminal           terminal.Terminal
	imageSizeResolvers []imageSizeResolver
}

// imageSizeResolver to resolve the image size from the options
type imageSizeResolver struct {
	match   func(options *Options) bool
	compute func(sz image.Rectangle, options *Options, handler *ImageResizeHandler) (width, height int, err error)
}

// ScaleImage resize the convert to expected size base on the convert options
func (handler *ImageResizeHandler) ScaleImage(image image.Image, options *Options) (newImage image.Image) {
	sz := image.Bounds()
	newWidth, newHeight, err := handler.resolveSize(sz, options)
	if err != nil {
		log.Fatal(err)
	}

	newImage = resize.Resize(uint(newWidth), uint(newHeight), image, resize.Lanczos3)
	return
}

// resolveSize resolve the image size
func (handler *ImageResizeHandler) resolveSize(sz image.Rectangle, options *Options) (width, height int, err error) {
	for _, resolver := range handler.imageSizeResolvers {
		if resolver.match(options) {
			return resolver.compute(sz, options, handler)
		}
	}
	return sz.Max.X, sz.Max.Y, nil
}

// CalcProportionalFittingScreenSize proportional scale the image
// so that the terminal can just show the picture.
func (handler *ImageResizeHandler) CalcProportionalFittingScreenSize(sz image.Rectangle) (newWidth, newHeight int, err error) {
	screenWidth, screenHeight, err := handler.terminal.ScreenSize()
	if err != nil {
		log.Fatal(nil)
	}
	newWidth, newHeight = handler.CalcFitSize(
		float64(screenWidth),
		float64(screenHeight),
		float64(sz.Max.X),
		float64(sz.Max.Y))
	return
}

// CalcFitSizeRatio through the given length and width,
// the computation can match the optimal scaling ratio of the length and width.
// In other words, it is able to give a given size rectangle to contain pictures
// Either match the width first, then scale the length equally,
// or match the length first, then scale the height equally.
// More detail please check the example
func (handler *ImageResizeHandler) CalcFitSizeRatio(width, height, imageWidth, imageHeight float64) (ratio float64) {
	ratio = 1.0
	// try to fit the height
	ratio = height / imageHeight
	scaledWidth := imageWidth * ratio / handler.terminal.CharWidth()
	if scaledWidth < width {
		return ratio / handler.terminal.CharWidth()
	}

	// try to fit the width
	ratio = width / imageWidth
	return ratio
}

// CalcFitSize through the given length and width ,
// Calculation is able to match the length and width of
// the specified size, and is proportional scaling.
func (handler *ImageResizeHandler) CalcFitSize(width, height, toBeFitWidth, toBeFitHeight float64) (fitWidth, fitHeight int) {
	ratio := handler.CalcFitSizeRatio(width, height, toBeFitWidth, toBeFitHeight)
	fitWidth = handler.ScaleWidthByRatio(toBeFitWidth, ratio)
	fitHeight = handler.ScaleHeightByRatio(toBeFitHeight, ratio)
	return
}

// ScaleWidthByRatio scaled the width by ratio
func (handler *ImageResizeHandler) ScaleWidthByRatio(width float64, ratio float64) int {
	return int(width * ratio)
}

// ScaleHeightByRatio scaled the height by ratio
func (handler *ImageResizeHandler) ScaleHeightByRatio(height float64, ratio float64) int {
	return int(height * ratio * handler.terminal.CharWidth())
}
