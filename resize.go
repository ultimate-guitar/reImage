package main

import (
	"fmt"
	"github.com/ultimate-guitar/go-imagequant"
	"gopkg.in/h2non/bimg.v1"
	"image/png"
	"log"
)

func resizeImage(params *requestParams) (err error) {
	bimg.VipsCacheSetMax(resizeLibVipsCacheSize)
	image := bimg.NewImage(params.imageBody)

	options := bimg.Options{
		Width:         params.reWidth,
		Height:        params.reHeight,
		Quality:       params.reQuality,
		Interpolator:  resizeLibVipsInterpolator,
		StripMetadata: true,
		NoProfile:     true,
		Embed:         true,
		Type:          params.reFormat,
	}

	// Special option for some image types
	if options.Type == bimg.PNG || (options.Type == bimg.UNKNOWN && image.Type() == "png") {
		options.Compression = 0 // Image will be compressed later, on optimization step
	} else {
		options.Compression = params.reCompression
	}

	if image.Type() == "gif" && options.Type == bimg.UNKNOWN {
		options.Type = bimg.JPEG
		params.imageContentType = "image/jpeg"
	}

	// Set content type based on output image type
	switch options.Type {
	case bimg.JPEG:
		params.imageContentType = "image/jpeg"
	case bimg.PNG:
		params.imageContentType = "image/png"
	case bimg.WEBP:
		params.imageContentType = "image/webp"
	case bimg.TIFF:
		params.imageContentType = "image/tiff"
	case bimg.UNKNOWN:
		if image.Type() == "webp" {
			params.imageContentType = "image/webp"
		}
	}

	params.imageBody, err = image.Process(options)
	if err != nil {
		return err
	}

	if options.Type == bimg.PNG || (options.Type == bimg.UNKNOWN && image.Type() == "png") {
		if err := optimizePng(params); err != nil {
			log.Printf("Can not optimize png image: '%s', err: %s", params.imageUrl.String(), err)
		}
	}

	return nil
}

func optimizePng(params *requestParams) (err error) {
	compression, err := zlibCompressionLevelToPNG(params.reCompression)
	if err != nil {
		return err
	}
	image, err := imagequant.Crush(params.imageBody, resizePngSpeed, compression)
	if err != nil {
		return err
	}
	params.imageBody = image
	return nil
}

func zlibCompressionLevelToPNG(zlibLevel int) (png.CompressionLevel, error) {
	switch zlibLevel {
	case 0:
		return png.NoCompression, nil
	case 9:
		return png.BestCompression, nil
	case 1, 2, 3, 4:
		return png.BestSpeed, nil
	case 5, 6, 7, 8:
		return png.DefaultCompression, nil
	default:
		return png.DefaultCompression, fmt.Errorf("wrong zlib compression level: %d", zlibLevel)
	}
}
