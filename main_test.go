package main

import (
	"io/ioutil"
	"testing"

	//Profiling
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"runtime"
	"strings"
)

const (
	testListenDebug = "127.0.0.1:6060"
)

func runProfiler() {
	runtime.SetBlockProfileRate(1)
	runtime.MemProfileRate = 1
	go func() {
		fmt.Println(http.ListenAndServe(testListenDebug, nil))
	}()
}

func resize(params *requestParams) error {
	if err := resizeImage(params); err != nil {
		return err
	}
	return nil
}

type testImage struct {
	params requestParams
	path   []string
}

var testSet = []testImage{
	testImage{requestParams{reWidth: 1280, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "jpeg", "bird_1920x1279.jpg"}},
	testImage{requestParams{reWidth: 0, reHeight: 1488, reQuality: 80, reCompression: 6}, []string{"samples", "jpeg", "bird_4466x2977.jpg"}},
	testImage{requestParams{reWidth: 427, reHeight: 284, reQuality: 80, reCompression: 6}, []string{"samples", "jpeg", "clock_1280x853.jpg"}},
	testImage{requestParams{reWidth: 0, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "jpeg", "clock_6000x4000.jpg"}},
	testImage{requestParams{reWidth: 720, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "jpeg", "fireworks_1920x1280.jpg"}},
	testImage{requestParams{reWidth: 128, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "jpeg", "fireworks_640x426.jpg"}},
	testImage{requestParams{reWidth: 1600, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "jpeg", "owl_2048x1500.jpg"}},
	testImage{requestParams{reWidth: 575, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "jpeg", "owl_640x468.jpg"}},
	testImage{requestParams{reWidth: 1750, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "png", "cc_705x453.png"}},
	testImage{requestParams{reWidth: 0, reHeight: 1080, reQuality: 80, reCompression: 6}, []string{"samples", "png", "istanbul_3993x2311.png"}},
	testImage{requestParams{reWidth: 500, reHeight: 500, reQuality: 80, reCompression: 6}, []string{"samples", "png", "penguin_1138x2378.png"}},
	testImage{requestParams{reWidth: 50, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "png", "penguin_380x793.png"}},
	testImage{requestParams{reWidth: 640, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "png", "wine_2400x2400.png"}},
	testImage{requestParams{reWidth: 640, reHeight: 0, reQuality: 80, reCompression: 6}, []string{"samples", "png", "wine_800x800.png"}},
}

func TestResizeImage(t *testing.T) {
	//runProfiler()
	for _, image := range testSet {
		destPathA := make([]string, len(image.path))
		copy(destPathA, image.path)
		destPathA[0] = "results"
		destPathA[2] = fmt.Sprintf("%s_to_%dx%d_q%d_c%d.%s", strings.Split(destPathA[2], ".")[0], image.params.reWidth, image.params.reHeight, image.params.reQuality, image.params.reCompression, strings.Split(destPathA[2], ".")[1])
		sourcePath := path.Join(image.path...)

		exPath, err := os.Getwd()
		if err != nil {
			fmt.Errorf("os.Getwd() err: %s", err)
		}
		destPath := path.Join(exPath, path.Join(destPathA...))
		if err := os.MkdirAll(path.Dir(destPath), os.ModePerm); err != nil {
			fmt.Errorf("Mkdir error on dir: %s, err: %s", path.Dir(destPath), err)
		}

		fmt.Printf("Resize image\t%-50s\tTO\t%-50s\n", sourcePath, destPath)

		if image.params.imageOriginBody, err = ioutil.ReadFile(sourcePath); err != nil {
			fmt.Errorf("IO error on file: %s, err: %s", sourcePath, err)
		}

		if err := resize(&image.params); err != nil {
			fmt.Errorf("Cannot resize image: %s, err: %s", sourcePath, err)
		}

		if err := ioutil.WriteFile(destPath, image.params.imageResizedBody, 0644); err != nil {
			fmt.Errorf("IO error on file: %s, err: %s", destPath, err)
		}
	}
}
