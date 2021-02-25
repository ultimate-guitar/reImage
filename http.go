package main

import (
	hex2 "encoding/hex"
	"fmt"
	"github.com/h2non/bimg"
	"github.com/labstack/echo/v4"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type requestParams struct {
	imageUrl         *url.URL
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

func healthHandler(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

func getResizeHandler(c echo.Context) error {
	params := requestParams{}
	if err := requestParser(c.Request(), &params); err != nil {
		c.Logger().Errorf("Can not parse requested url: '%s', err: %s", c.Request().URL.String(), err)
		return c.String(http.StatusBadRequest, "")
	}
	if code, err := getSourceImage(&params); err != nil {
		c.Logger().Errorf("Can not get source image: '%s', err: %s", params.imageUrl.String(), err)
		return c.String(code, "")
	}

	if config.SkipEmptyImages && len(params.imageBody) == 0 {
		c.Logger().Warnf("Empty images skipped: %s", params.imageUrl.String())
		return c.Blob(http.StatusOK, params.imageContentType, nil)
	}

	if err := resizeImage(&params); err != nil {
		c.Logger().Errorf("Can not resize image: '%s', err: %s", params.imageUrl.String(), err)
		return c.String(http.StatusInternalServerError, "")
	}

	return c.Blob(http.StatusOK, params.imageContentType, params.imageBody)
}

func postResizeHandler(c echo.Context) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		c.Logger().Errorf("Failed to read POST body, err: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}
	params := requestParams{imageBody: body}
	if err := requestParser(c.Request(), &params); err != nil {
		c.Logger().Errorf("Can not parse requested url: '%s', err: %s", c.Request().URL.String(), err)
		return c.String(http.StatusBadRequest, "")
	}

	if err := resizeImage(&params); err != nil {
		c.Logger().Errorf("Can not resize image: '%s', err: %s", params.imageUrl.String(), err)
		return c.String(http.StatusInternalServerError, "")
	}

	return c.Blob(http.StatusOK, params.imageContentType, params.imageBody)
}

func requestParser(req *http.Request, params *requestParams) (err error) {
	params.imageUrl, _ = url.Parse(req.URL.String())
	args := params.imageUrl.Query()
	for _, arg := range []string{"qlt", "cmp", "fmt", "crop", "bgclr"} {
		args.Del(arg)
	}
	params.imageUrl.RawQuery = args.Encode()

	if sourceHeader := req.Header.Get(resizeHeaderNameSource); req.Method == "GET" && sourceHeader == "" {
		return fmt.Errorf("empty '%s' header on GET request", resizeHeaderNameSource)
	} else {
		params.imageUrl.Host = sourceHeader
	}

	switch schemaHeader := strings.ToLower(req.Header.Get(resizeHeaderNameSchema)); schemaHeader {
	case "":
		params.imageUrl.Scheme = resizeHeaderDefaultSchema
	case "https", "http":
		params.imageUrl.Scheme = schemaHeader
	default:
		return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameSchema, schemaHeader)
	}

	// Parse Quality Header and Args
	if arg := req.URL.Query().Get("qlt"); arg != "" {
		quality, err := strconv.Atoi(arg)
		if err != nil || quality < 0 || quality > 100 {
			return fmt.Errorf("wrong arg 'qlt' value: '%s'", arg)
		}
		params.quality = quality
	} else if header := req.Header.Get(resizeHeaderNameQuality); header != "" {
		quality, err := strconv.Atoi(header)
		if (err != nil) || quality < 0 || quality > 100 {
			return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameQuality, header)
		}
		params.quality = quality
	} else {
		params.quality = resizeHeaderDefaultQuality
	}

	// Parse Compression Header and Args
	if arg := req.URL.Query().Get("cmp"); arg != "" {
		compression, err := strconv.Atoi(arg)
		if err != nil || compression < 0 || compression > 9 {
			return fmt.Errorf("wrong arg 'cmp' value: '%s'", arg)
		}
		params.compression = compression
	} else if header := req.Header.Get(resizeHeaderNameCompression); header != "" {
		compression, err := strconv.Atoi(header)
		if (err != nil) || compression < 0 || compression > 9 {
			return fmt.Errorf("wrong '%s' header value: '%s'", resizeHeaderNameCompression, header)
		}
		params.compression = compression
	} else {
		params.compression = resizeHeaderDefaultCompression
	}

	// Parse Format Args
	if arg := req.URL.Query().Get("fmt"); arg != "" {
		switch strings.ToLower(arg) {
		case "jpeg", "jpg":
			params.format = bimg.JPEG
		case "png":
			params.format = bimg.PNG
		case "webp":
			params.format = bimg.WEBP
		case "tiff":
			params.format = bimg.TIFF
		default:
			return fmt.Errorf("wrong arg 'fmt' value: '%s'", arg)
		}
	}

	// Parse Crop args
	if arg := req.URL.Query().Get("crop"); arg != "" {
		arg, err := strconv.ParseBool(arg)
		if err != nil {
			return fmt.Errorf("wrong arg 'crop' value: '%s'", arg)
		}
		params.crop = arg
	}

	// Parse Background color Args
	if arg := req.URL.Query().Get("bgclr"); arg != "" {
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
		splitedPath := strings.Split(string(req.URL.Path), "@")
		resizeArgs := strings.Split(strings.ToLower(splitedPath[len(splitedPath)-1]), "x")
		params.imageUrl.Path = strings.Join(splitedPath[:len(splitedPath)-1], "@")
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
		return http.StatusInternalServerError, err
	}

	req.Header.Set("User-Agent", httpUserAgent)
	res, err := httpClient.Do(req)
	if res != nil {
		defer res.Body.Close()
		defer io.Copy(ioutil.Discard, res.Body)
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if res.StatusCode != http.StatusOK {
		return res.StatusCode, fmt.Errorf("status code %d != %d", res.StatusCode, http.StatusOK)
	}

	params.imageBody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	params.imageContentType = res.Header.Get("content-type")
	return res.StatusCode, nil
}
