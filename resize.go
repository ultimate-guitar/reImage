package main

import (
	"fmt"
	"github.com/larrabee/go-imagequant"
	"github.com/valyala/fasthttp"
	"gopkg.in/h2non/bimg.v1"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type requestParams struct {
	imageUrl         fasthttp.URI
	imageOriginBody  []byte
	imageResizedBody []byte
	imageContentType string
	reWidth          int
	reHeight         int
	reQuality        int
	reCompression    int
}

func resizeHandler(ctx *fasthttp.RequestCtx) {
	params := requestParams{}
	if err := requestParser(ctx, &params); err != nil {
		log.Printf("Can not parse requested url: '%s', err: %s", ctx.URI(), err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	if code, err := getSourceImage(&params); err != nil {
		log.Printf("Can not get source image: '%s', err: %s", params.imageUrl.String(), err)
		ctx.SetStatusCode(code)
		return
	}
	if err := resizeImage(&params); err != nil {
		log.Printf("Can not resize image: '%s', err: %s", params.imageUrl.String(), err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetBody(params.imageResizedBody)
	ctx.SetContentType(params.imageContentType)
	ctx.SetStatusCode(fasthttp.StatusOK)
	//debug.FreeOSMemory()
	return
}

func requestParser(ctx *fasthttp.RequestCtx, params *requestParams) (err error) {
	params.imageUrl = fasthttp.URI{}
	params.imageUrl.SetQueryStringBytes(ctx.URI().QueryString())
	sourceHeader := string(ctx.Request.Header.Peek(resizeHeaderNameSource))
	if sourceHeader == "" {
		return fmt.Errorf("empty '%s' header", resizeHeaderNameSource)
	}
	params.imageUrl.SetHost(sourceHeader)

	switch schemaHeader := string(ctx.Request.Header.Peek(resizeHeaderNameSchema)); schemaHeader {
	case "":
		params.imageUrl.SetScheme(resizeHeaderDefaultSchema)
	case "https":
		params.imageUrl.SetScheme("https")
	case "http":
		params.imageUrl.SetScheme("http")
	default:
		return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameSchema, schemaHeader)
	}

	// Parse Quality Header
	if header := string(ctx.Request.Header.Peek(resizeHeaderNameQuality)); header == "" {
		params.reQuality = resizeHeaderDefaultQuality
	} else {
		quality, err := strconv.Atoi(header)
		if (err != nil) || quality < 0 {
			return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameQuality, header)
		}
		params.reQuality = quality
	}

	// Parse Compression Header
	if header := string(ctx.Request.Header.Peek(resizeHeaderNameCopression)); header == "" {
		params.reCompression = resizeHeaderDefaultCompression
	} else {
		compression, err := strconv.Atoi(header)
		if (err != nil) || compression < 0 {
			return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameCopression, header)
		}
		params.reCompression = compression
	}

	// Parse Request uri for resize params
	{
		splitedPath := strings.Split(string(ctx.URI().Path()), "@")
		resizeArgs := strings.Split(strings.ToLower(splitedPath[len(splitedPath)-1]), "x")
		params.imageUrl.SetPath(strings.Join(splitedPath[:len(splitedPath)-1], "@"))
		if resizeArgs[0] != "" {
			if params.reWidth, err = strconv.Atoi(resizeArgs[0]); err != nil {
				return fmt.Errorf("reWidth value '%s' parsing error: %s", resizeArgs[0], err)
			}
		}
		if (len(resizeArgs) >= 2) && (resizeArgs[1] != "") {
			if params.reHeight, err = strconv.Atoi(resizeArgs[1]); err != nil {
				return fmt.Errorf("reHeight value '%s' parsing error: %s", resizeArgs[1], err)
			}
		}

		if (params.reWidth == 0) && (params.reHeight == 0) {
			return fmt.Errorf("both reWidth and reHeight have zero value")
		} else if params.reWidth < 0 {
			return fmt.Errorf("reWidth have negative value")
		} else if params.reHeight < 0 {
			return fmt.Errorf("reHeight have negative value")
		}
	}
	return nil
}

func getSourceImage(params *requestParams) (code int, err error) {
	transport := &http.Transport{DisableKeepAlives: false}
	client := &http.Client{Transport: transport, Timeout: imageDownloadTimeout}
	res, err := client.Get(params.imageUrl.String())
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return fasthttp.StatusInternalServerError, err
	}

	if res.StatusCode != fasthttp.StatusOK {
		return res.StatusCode, fmt.Errorf("status code %d != 200", res.StatusCode)
	}
	params.imageOriginBody = make([]byte, res.ContentLength)
	_, err = io.ReadFull(res.Body, params.imageOriginBody)
	if err != nil {
		return fasthttp.StatusInternalServerError, err
	}
	params.imageContentType = res.Header.Get("content-type")
	return res.StatusCode, nil
}

func resizeImage(params *requestParams) (err error) {
	options := bimg.Options{
		Width:         params.reWidth,
		Height:        params.reHeight,
		Quality:       params.reQuality,
		Compression:   params.reCompression,
		Interpolator:  bimg.Nohalo,
		StripMetadata: true,
		NoProfile:     true,
		Embed:         true,
		Trim:          true,
	}

	bimg.VipsCacheSetMax(0)
	image := bimg.NewImage(params.imageOriginBody)
	params.imageResizedBody, err = image.Process(options)
	if err != nil {
		return err
	}
	if image.Type() == "png" {
		if err := optimizePng(params); err != nil {
			log.Printf("Can not optimize png image: '%s', err: %s", params.imageUrl.String(), err)
		}
	}
	return nil
}

func optimizePng(params *requestParams) (err error) {
	image, err := imagequant.Crush(params.imageResizedBody, resizePngSpeed, resizePngCompression)
	if err != nil {
		return err
	}
	params.imageResizedBody = image
	return nil
}
