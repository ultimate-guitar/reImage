package main

import (
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
	reQuality        int
	reCompression    int
	reFormat         bimg.ImageType
}

var httpTransport = &http.Transport{
	MaxIdleConns:        httpClientMaxIdleConns,
	IdleConnTimeout:     httpClientIdleConnTimeout,
	MaxIdleConnsPerHost: httpClientMaxIdleConnsPerHost,
}
var httpClient = &http.Client{Transport: httpTransport, Timeout: httpClientImageDownloadTimeout}

func getResizeHandler(ctx *fasthttp.RequestCtx) {
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
	params.imageUrl = fasthttp.URI{}
	params.imageUrl.SetQueryStringBytes(ctx.URI().QueryString())
	sourceHeader := string(ctx.Request.Header.Peek(resizeHeaderNameSource))
	if (sourceHeader == "") && ctx.IsGet()  {
		return fmt.Errorf("empty '%s' header on GET request", resizeHeaderNameSource)
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

	// Parse Quality Header and Args
	{
		if header := string(ctx.Request.Header.Peek(resizeHeaderNameQuality)[:]); header != "" {
			quality, err := strconv.Atoi(header)
			if (err != nil) || quality < 0 || quality > 100 {
				return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameQuality, header)
			}
			params.reQuality = quality
		} else {
			params.reQuality = resizeHeaderDefaultQuality
		}

		qualityArg := ""
		if ctx.QueryArgs().Has("qlt") {
			qualityArg = string(ctx.QueryArgs().Peek("qlt")[:])
		} else if ctx.PostArgs().Has("qlt") {
			qualityArg = string(ctx.PostArgs().Peek("qlt")[:])
		}

		if qualityArg != "" {
			quality, err := strconv.Atoi(qualityArg)
			if err != nil || quality < 0 || quality > 100 {
				return fmt.Errorf("wrong arg 'qlt' value: '%s'", qualityArg)
			}
			params.reQuality = quality
		}
	}

	// Parse Compression Header and Args
	{
		if header := string(ctx.Request.Header.Peek(resizeHeaderNameCompression)[:]); header != "" {
			compression, err := strconv.Atoi(header)
			if (err != nil) || compression < 0 || compression > 9 {
				return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameCompression, header)
			}
			params.reCompression = compression
		} else {
			params.reCompression = resizeHeaderDefaultCompression
		}

		commpressionArg := ""
		if ctx.QueryArgs().Has("cmp") {
			commpressionArg = string(ctx.QueryArgs().Peek("cmp")[:])
		} else if ctx.PostArgs().Has("cmp") {
			commpressionArg = string(ctx.PostArgs().Peek("cmp")[:])
		}

		if commpressionArg != "" {
			compression, err := strconv.Atoi(commpressionArg)
			if err != nil || compression < 0 || compression > 9 {
				return fmt.Errorf("wrong arg 'cmp' value: '%s'", commpressionArg)
			}
			params.reCompression = compression
		}
	}
	// Parse Format Args
	{
		formatArgs := ""

		if ctx.QueryArgs().Has("fmt") {
			formatArgs = string(ctx.QueryArgs().Peek("fmt")[:])
		} else if ctx.PostArgs().Has("fmt") {
			formatArgs = string(ctx.PostArgs().Peek("fmt")[:])
		}

		if formatArgs != "" {
			switch strings.ToLower(formatArgs) {
			case "jpeg": params.reFormat = bimg.JPEG
			case "jpg": params.reFormat = bimg.JPEG
			case "png": params.reFormat = bimg.PNG
			case "webp": params.reFormat = bimg.WEBP
			case "tiff": params.reFormat = bimg.TIFF
			default:
				return fmt.Errorf("wrong arg 'fmt' value: '%s'", formatArgs)
			}
		}
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
	res, err := httpClient.Get(params.imageUrl.String())
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
