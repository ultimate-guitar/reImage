package main

import (
	"github.com/buaazp/fasthttprouter"
	"github.com/namsral/flag"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"gopkg.in/h2non/bimg.v1"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Listen          string
	SkipEmptyImages bool
}

const (
	resizeHeaderNameSource         = "x-resize-base"
	resizeHeaderNameSchema         = "x-resize-scheme"
	resizeHeaderDefaultSchema      = "https"
	resizeHeaderNameQuality        = "x-resize-quality"
	resizeHeaderDefaultQuality     = 80
	resizeHeaderNameCompression    = "x-resize-compression"
	resizeHeaderDefaultCompression = 6
	httpClientMaxIdleConns         = 512
	httpClientMaxIdleConnsPerHost  = 64
	httpClientMaxConnsPerHost      = 128
	httpClientIdleConnTimeout      = 30 * time.Second
	httpClientImageDownloadTimeout = 30 * time.Second
	serverMaxConcurrencyRequests   = 2048
	serverRequestReadTimeout       = 10 * time.Second
	serverResponseWriteTimeout     = 20 * time.Second
	resizePngSpeed                 = 3
	resizeLibVipsInterpolator      = bimg.Bicubic
	resizeLibVipsCacheSize         = 128 // Operations cache size. Increase it gain high perforce and high memory usage
	httpUserAgent                  = "reImage HTTP Fetcher"
)

func init() {
	parseFlags(config)

	httpTransport := &http.Transport{
		MaxIdleConns:        httpClientMaxIdleConns,
		IdleConnTimeout:     httpClientIdleConnTimeout,
		MaxIdleConnsPerHost: httpClientMaxIdleConnsPerHost,
		MaxConnsPerHost:     httpClientMaxConnsPerHost,
	}
	httpClient = &http.Client{Transport: httpTransport, Timeout: httpClientImageDownloadTimeout}
}

var httpClient *http.Client
var config = &Config{}

func main() {
	listen, err := reuseport.Listen("tcp4", config.Listen)
	if err != nil {
		log.Fatalf("Error in reuseport listener: %s", err)
	}

	router := getRouter()

	server := &fasthttp.Server{
		Handler:          router.Handler,
		DisableKeepalive: true,
		GetOnly:          false,
		Concurrency:      serverMaxConcurrencyRequests,
		ReadTimeout:      serverRequestReadTimeout,
		WriteTimeout:     serverResponseWriteTimeout,
	}

	log.Printf("Server started on %s\n", config.Listen)
	if err := server.Serve(listen); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func getRouter() *fasthttprouter.Router {
	router := fasthttprouter.New()
	router.GET("/*p", getResizeHandler)
	router.POST("/*p", postResizeHandler)
	return router
}

func parseFlags(config *Config) {
	flag.StringVar(&config.Listen, "CFG_LISTEN", "127.0.0.1:7075", "Listen interface and port")
	flag.BoolVar(&config.SkipEmptyImages, "CFG_SKIP_EMPTY_IMAGES", false, "Skip empty images resizing")
	if *flag.Bool("CFG_DISABLE_HTTP2", false, "Disable HTTP2 for image downloader") == true {
		_ = os.Setenv("GODEBUG", os.Getenv("GODEBUG")+"http2client=0")
	}
	flag.Parse()
}
