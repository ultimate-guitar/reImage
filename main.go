package main

import (
	"github.com/namsral/flag"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"log"
	"time"
	"image/png"

	//Profiling
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

var (
	cfgListen      string
	cfgListenDebug string
)

// Resize Headers
const (
	resizeHeaderNameSource         = "x-resize-base"
	resizeHeaderNameSchema         = "x-resize-scheme"
	resizeHeaderDefaultSchema      = "https"
	resizeHeaderNameQuality        = "x-resize-quality"
	resizeHeaderDefaultQuality     = 80
	resizeHeaderNameCopression     = "x-resize-compression"
	resizeHeaderDefaultCompression = 6
	maxConcurrencyRequests         = 2048
	imageDownloadTimeout           = 20 * time.Second
	requestReadTimeout             = 10 * time.Second
	responseWriteTimeout           = 20 * time.Second
	resizePngSpeed                 = 1
	resizePngCompression           = png.BestCompression
)

func main() {
	parseFlags()

	if cfgListenDebug != "" {
		runtime.SetBlockProfileRate(1)
		go func() {
			log.Println(http.ListenAndServe(cfgListenDebug, nil))
		}()
	}

	listen, err := reuseport.Listen("tcp4", cfgListen)
	if err != nil {
		log.Fatalf("Error in reuseport listener: %s", err)
	}
	server := &fasthttp.Server{
		Handler:          resizeHandler,
		DisableKeepalive: true,
		GetOnly:          true,
		Concurrency:      maxConcurrencyRequests,
		ReadTimeout:      requestReadTimeout,
		WriteTimeout:     responseWriteTimeout,
	}

	log.Printf("Server started on %s\n", cfgListen)
	if err := server.Serve(listen); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func parseFlags() {
	flag.StringVar(&cfgListen, "CFG_LISTEN", "127.0.0.1:7075", "Listen interface and port")
	flag.StringVar(&cfgListenDebug, "CFG_DEBUG", "", "Listen interface and port for debug")
	flag.Parse()
}
