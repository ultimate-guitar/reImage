package main

import (
	"github.com/namsral/flag"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"gopkg.in/h2non/bimg.v1"
	"log"
	"time"
)

var (
	cfgListen      string
)

const (
	resizeHeaderNameSource         = "x-resize-base"
	resizeHeaderNameSchema         = "x-resize-scheme"
	resizeHeaderDefaultSchema      = "https"
	resizeHeaderNameQuality        = "x-resize-quality"
	resizeHeaderDefaultQuality     = 80
	resizeHeaderNameCompression    = "x-resize-compression"
	resizeHeaderDefaultCompression = 6
	httpClientMaxIdleConns         = 512
	httpClientMaxIdleConnsPerHost  = 512
	httpClientIdleConnTimeout      = 120 * time.Second
	httpClientImageDownloadTimeout = 30 * time.Second
	serverMaxConcurrencyRequests   = 2048
	serverRequestReadTimeout       = 10 * time.Second
	serverResponseWriteTimeout     = 20 * time.Second
	resizePngSpeed                 = 2
	resizeLibVipsInterpolator      = bimg.Bicubic
	resizeLibVipsCacheSize         = 128 // Operations cache size. Increase it gain high perforce and high memory usage
)

func main() {
	parseFlags()

	listen, err := reuseport.Listen("tcp4", cfgListen)
	if err != nil {
		log.Fatalf("Error in reuseport listener: %s", err)
	}
	server := &fasthttp.Server{
		Handler:          resizeHandler,
		DisableKeepalive: true,
		GetOnly:          true,
		Concurrency:      serverMaxConcurrencyRequests,
		ReadTimeout:      serverRequestReadTimeout,
		WriteTimeout:     serverResponseWriteTimeout,
	}

	log.Printf("Server started on %s\n", cfgListen)
	if err := server.Serve(listen); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func parseFlags() {
	flag.StringVar(&cfgListen, "CFG_LISTEN", "127.0.0.1:7075", "Listen interface and port")
	flag.Parse()
}
