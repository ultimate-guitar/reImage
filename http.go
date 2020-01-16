package main

import (
	hex2 "encoding/hex"
	"fmt"
	"github.com/valyala/fasthttp"
	"gopkg.in/h2non/bimg.v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type requestParams struct {
	imageUrl         fasthttp.URI
	imageBody        []byte
	imageContentType string
	reWidth          int
	reHeight         int
	quality          int
	compression      int
	format           bimg.ImageType
	crop             bool
	bgColor          bimg.Color
}

func getResizeHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Request.RequestURI()) == "/health" {
		ctx.SetBody([]byte("OK"))
		ctx.SetStatusCode(fasthttp.StatusOK)
		return
	}

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

	if config.SkipEmptyImages && len(params.imageBody) == 0 {
		log.Printf("Empty images skipped: %s", params.imageUrl.String())
		ctx.SetContentType(params.imageContentType)
		ctx.SetStatusCode(fasthttp.StatusOK)
		return
	}

	if err := resizeImage(&params); err != nil {
		log.Printf("Can not resize image: '%s', err: %s", params.imageUrl.String(), err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetBody(params.imageBody)
	ctx.SetContentType(params.imageContentType)
	ctx.SetStatusCode(fasthttp.StatusOK)
	return
}

func postResizeHandler(ctx *fasthttp.RequestCtx) {
	params := requestParams{imageBody: ctx.PostBody()}
	if err := requestParser(ctx, &params); err != nil {
		log.Printf("Can not parse requested url: '%s', err: %s", ctx.URI(), err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	if err := resizeImage(&params); err != nil {
		log.Printf("Can not resize image: '%s', err: %s", params.imageUrl.String(), err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetBody(params.imageBody)
	ctx.SetContentType(params.imageContentType)
	ctx.SetStatusCode(fasthttp.StatusOK)
	return
}

func requestParser(ctx *fasthttp.RequestCtx, params *requestParams) (err error) {
	ctx.URI().CopyTo(&params.imageUrl)
	for _, arg := range []string{"qlt", "cmp", "fmt", "crop", "bgclr"} {
		params.imageUrl.QueryArgs().Del(arg)
	}

	if sourceHeader := string(ctx.Request.Header.Peek(resizeHeaderNameSource)); ctx.IsGet() && sourceHeader == "" {
		return fmt.Errorf("empty '%s' header on GET request", resizeHeaderNameSource)
	} else {
		params.imageUrl.SetHost(sourceHeader)
	}

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

	// Parse Quality Header and Args
	if ctx.URI().QueryArgs().Has("qlt") {
		arg := string(ctx.QueryArgs().Peek("qlt")[:])
		quality, err := strconv.Atoi(arg)
		if err != nil || quality < 0 || quality > 100 {
			return fmt.Errorf("wrong arg 'qlt' value: '%s'", arg)
		}
		params.quality = quality
	} else if header := string(ctx.Request.Header.Peek(resizeHeaderNameQuality)[:]); header != "" {
		quality, err := strconv.Atoi(header)
		if (err != nil) || quality < 0 || quality > 100 {
			return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameQuality, header)
		}
		params.quality = quality
	} else {
		params.quality = resizeHeaderDefaultQuality
	}

	// Parse Compression Header and Args
	if ctx.URI().QueryArgs().Has("cmp") {
		arg := string(ctx.QueryArgs().Peek("cmp")[:])
		compression, err := strconv.Atoi(arg)
		if err != nil || compression < 0 || compression > 9 {
			return fmt.Errorf("wrong arg 'cmp' value: '%s'", arg)
		}
		params.compression = compression
	} else if header := string(ctx.Request.Header.Peek(resizeHeaderNameCompression)[:]); header != "" {
		compression, err := strconv.Atoi(header)
		if (err != nil) || compression < 0 || compression > 9 {
			return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameCompression, header)
		}
		params.compression = compression
	} else {
		params.compression = resizeHeaderDefaultCompression
	}

	// Parse Format Args
	if ctx.QueryArgs().Has("fmt") {
		formatArg := string(ctx.QueryArgs().Peek("fmt")[:])
		switch strings.ToLower(formatArg) {
		case "jpeg", "jpg":
			params.format = bimg.JPEG
		case "png":
			params.format = bimg.PNG
		case "webp":
			params.format = bimg.WEBP
		case "tiff":
			params.format = bimg.TIFF
		default:
			return fmt.Errorf("wrong arg 'fmt' value: '%s'", formatArg)
		}
	}

	// Parse Crop args
	if ctx.QueryArgs().Has("crop") {
		cropArg := string(ctx.QueryArgs().Peek("crop")[:])
		arg, err := strconv.ParseBool(cropArg)
		if err != nil {
			return fmt.Errorf("wrong arg 'crop' value: '%s'", cropArg)
		}
		params.crop = arg
	}

	// Parse Background color Args
	if ctx.QueryArgs().Has("bgclr") {
		arg := strings.ToLower(string(ctx.QueryArgs().Peek("bgclr")[:]))
		hex, err := hex2.DecodeString(arg)
		if err != nil {
			return fmt.Errorf("wrong arg 'bgclr' value: '%s'", arg)
		}
		if len(hex) != 3 {
			return fmt.Errorf("wrong arg 'bgclr' value: '%s'", arg)
		}
		params.bgColor = bimg.Color{R: hex[0], G: hex[1], B: hex[2]}
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

		if params.reWidth < 0 {
			return fmt.Errorf("reWidth have negative value")
		} else if params.reHeight < 0 {
			return fmt.Errorf("reHeight have negative value")
		}
	}
	return nil
}

func getSourceImage(params *requestParams) (code int, err error) {
	req, err := http.NewRequest("GET", params.imageUrl.String(), nil)
	if err != nil {
		return fasthttp.StatusInternalServerError, err
	}

	req.Header.Set("User-Agent", httpUserAgent)
	res, err := httpClient.Do(req)
	if res != nil {
		defer res.Body.Close()
		defer io.Copy(ioutil.Discard, res.Body)
	}
	if err != nil {
		return fasthttp.StatusInternalServerError, err
	}

	if res.StatusCode != fasthttp.StatusOK {
		return res.StatusCode, fmt.Errorf("status code %d != %d", res.StatusCode, fasthttp.StatusOK)
	}

	params.imageBody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return fasthttp.StatusInternalServerError, err
	}
	params.imageContentType = res.Header.Get("content-type")
	return res.StatusCode, nil
}
