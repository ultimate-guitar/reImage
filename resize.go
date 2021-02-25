package main

import (
	"fmt"
	"github.com/h2non/bimg"
	"github.com/ultimate-guitar/go-imagequant"
	"image/png"
	"log"
)

const (
	ContentTypeJPEG = "image/jpeg"
	ContentTypePNG  = "image/png"
	ContentTypeWEBP = "image/webp"
	ContentTypeTIFF = "image/tiff"
	ContentTypeGIF  = "image/gif"
)

func resizeImage(params *requestParams) (err error) {
	bimg.VipsCacheSetMax(resizeLibVipsCacheSize)
	image := bimg.NewImage(params.imageBody)

	options := bimg.Options{
		Width:         params.reWidth,
		Height:        params.reHeight,
		Quality:       params.quality,
		Interpolator:  resizeLibVipsInterpolator,
		StripMetadata: true,
		NoProfile:     true,
		Embed:         true,
		Type:          params.format,
		Crop:          params.crop,
		Background:    params.bgColor,
		Extend:        bimg.ExtendBackground,
		Compression:   params.compression,
	}


	pngOptimisationNeeded := false
	// Set content type and convert options based on output image type
	switch options.Type {
	case bimg.JPEG:
		params.imageContentType = ContentTypeJPEG
	case bimg.PNG:
		params.imageContentType = ContentTypePNG
		options.Compression = 0 // Image will be compressed later, on optimization step
		pngOptimisationNeeded = true
	case bimg.WEBP:
		params.imageContentType = ContentTypeWEBP
	case bimg.TIFF:
		params.imageContentType = ContentTypeTIFF
	case bimg.UNKNOWN:
		switch bimg.DetermineImageType(params.imageBody) {
		case bimg.JPEG:
			params.imageContentType = ContentTypeJPEG
		case bimg.PNG:
			params.imageContentType = ContentTypePNG
			options.Compression = 0 // Image will be compressed later, on optimization step
			pngOptimisationNeeded = true
		case bimg.WEBP:
			params.imageContentType = ContentTypeWEBP
		case bimg.TIFF:
			params.imageContentType = ContentTypeTIFF
		case bimg.GIF:
			params.imageContentType = ContentTypeJPEG
			options.Type = bimg.JPEG
		}
	}

	params.imageBody, err = image.Process(options)
	if err != nil {
		return err
	}

	if pngOptimisationNeeded {
		if err := optimizePng(params); err != nil {
			log.Printf("Can not optimize png image: '%s', err: %s", params.imageUrl.String(), err)
		}
	}

	return nil
}

func optimizePng(params *requestParams) (err error) {
	compression, err := zlibCompressionLevelToPNG(params.compression)
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
