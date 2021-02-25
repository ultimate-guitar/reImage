package main

import (
	"github.com/alexflint/go-arg"
	"github.com/h2non/bimg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Debug           bool   `arg:"env:CFG_DEBUG" help:"Enable debug logging"`
	Listen          string `arg:"env:CFG_LISTEN" help:"Listen interface and port"`
	SkipEmptyImages bool   `arg:"env:CFG_SKIP_EMPTY_IMAGES" help:"Skip empty images resizing"`
	DisableHttp2    bool   `arg:"env:CFG_DISABLE_HTTP2" help:"Disable HTTP2 for image downloader"`
}

const (
	resizeHeaderNameSource         = "x-resize-base"
	resizeHeaderNameSchema         = "x-resize-scheme"
	resizeHeaderDefaultSchema      = "https"
	resizeHeaderNameQuality        = "x-resize-quality"
	resizeHeaderDefaultQuality     = 80
	resizeHeaderNameCompression    = "x-resize-compression"
	resizeHeaderDefaultCompression = 6
	httpClientMaxIdleConns         = 128
	httpClientMaxIdleConnsPerHost  = 128
	httpClientMaxConnsPerHost      = 128
	httpClientIdleConnTimeout      = 30 * time.Second
	httpClientImageDownloadTimeout = 30 * time.Second
	resizePngSpeed                 = 3
	resizeLibVipsInterpolator      = bimg.Bicubic
	resizeLibVipsCacheSize         = 128 // Operations cache size. Increase it gain high perforce and high memory usage
	httpUserAgent                  = "reImage HTTP Fetcher"
)

var config = parseFlags()
var httpClient *http.Client

func init() {
	httpTransport := &http.Transport{
		MaxIdleConns:        httpClientMaxIdleConns,
		IdleConnTimeout:     httpClientIdleConnTimeout,
		MaxIdleConnsPerHost: httpClientMaxIdleConnsPerHost,
		MaxConnsPerHost:     httpClientMaxConnsPerHost,
	}
	httpClient = &http.Client{Transport: httpTransport, Timeout: httpClientImageDownloadTimeout}
}

func main() {
	e := echo.New()
	e.Use(middleware.Recover())
	e.HideBanner = true

	if config.Debug {
		e.Use(middleware.Logger())
		e.Logger.SetLevel(log.DEBUG)
	}

	e.GET("/health", healthHandler)
	e.GET("/*", getResizeHandler)
	e.POST("/*", postResizeHandler)

	e.Logger.Printf("Server started on %s\n", config.Listen)
	e.Logger.Fatal(e.Start(config.Listen))
}

func parseFlags() *Config {
	config := Config{
		Listen:          "127.0.0.1:7075",
		SkipEmptyImages: false,
		DisableHttp2:    false,
		Debug: false,
	}
	arg.MustParse(&config)
	if config.DisableHttp2 {
		_ = os.Setenv("GODEBUG", os.Getenv("GODEBUG")+"http2client=0")
	}
	return &config
}
