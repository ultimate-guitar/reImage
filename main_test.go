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

func TestResizeImage(t *testing.T) {
	//runProfiler()
	var testSetGood = []testImage{
		{requestParams{reWidth: 1280, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "jpeg", "bird_1920x1279.jpg"}},
		{requestParams{reWidth: 1280, reHeight: 0, quality: 100, compression: 9}, []string{"samples", "jpeg", "bird_1920x1279.jpg"}},
		{requestParams{reWidth: 0, reHeight: 1488, quality: 80, compression: 6}, []string{"samples", "jpeg", "bird_4466x2977.jpg"}},
		{requestParams{reWidth: 0, reHeight: 1488, quality: 60, compression: 6}, []string{"samples", "jpeg", "bird_4466x2977.jpg"}},
		{requestParams{reWidth: 427, reHeight: 284, quality: 80, compression: 6}, []string{"samples", "jpeg", "clock_1280x853.jpg"}},
		{requestParams{reWidth: 0, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "jpeg", "clock_6000x4000.jpg"}},
		{requestParams{reWidth: 0, reHeight: 0, quality: 80, compression: 9}, []string{"samples", "jpeg", "clock_6000x4000.jpg"}},
		{requestParams{reWidth: 720, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "jpeg", "fireworks_1920x1280.jpg"}},
		{requestParams{reWidth: 128, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "jpeg", "fireworks_640x426.jpg"}},
		{requestParams{reWidth: 1600, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "jpeg", "owl_2048x1500.jpg"}},
		{requestParams{reWidth: 575, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "jpeg", "owl_640x468.jpg"}},
		{requestParams{reWidth: 1750, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "png", "cc_705x453.png"}},
		{requestParams{reWidth: 0, reHeight: 1080, quality: 80, compression: 6}, []string{"samples", "png", "istanbul_3993x2311.png"}},
		{requestParams{reWidth: 500, reHeight: 500, quality: 80, compression: 6}, []string{"samples", "png", "penguin_1138x2378.png"}},
		{requestParams{reWidth: 50, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "png", "penguin_380x793.png"}},
		{requestParams{reWidth: 640, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "png", "wine_2400x2400.png"}},
		{requestParams{reWidth: 640, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "png", "wine_800x800.png"}},
	}

	fmt.Println("Test good images:")
	for _, image := range testSetGood {
		var err error
		destPathA := make([]string, len(image.path))
		copy(destPathA, image.path)
		destPathA[0] = "results"
		destPathA[2] = fmt.Sprintf("%s_to_%dx%d_q%d_c%d.%s", strings.Split(destPathA[2], ".")[0], image.params.reWidth, image.params.reHeight, image.params.quality, image.params.compression, strings.Split(destPathA[2], ".")[1])
		sourcePath := path.Join(image.path...)
		destPath := path.Join(destPathA...)
		if err := os.MkdirAll(path.Dir(destPath), os.ModePerm); err != nil {
			t.Errorf("mkdir error on dir: %s, err: %s", path.Dir(destPath), err)
		}

		fmt.Printf("Resize image\t%-50s\tTO\t%-50s\n", sourcePath, destPath)

		if image.params.imageBody, err = ioutil.ReadFile(sourcePath); err != nil {
			t.Errorf("IO error on file: %s, err: %s", sourcePath, err)
		}

		if err := resize(&image.params); err != nil {
			t.Errorf("cannot resize image: %s, err: %s", sourcePath, err)
		}

		if err := ioutil.WriteFile(destPath, image.params.imageBody, 0644); err != nil {
			t.Errorf("IO error on file: %s, err: %s", destPath, err)
		}
	}

	var testSetBroken = []testImage{
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-1.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-2.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-3.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-4.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-5.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-6.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-7.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-8.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-9.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-10.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "corrupted-11.jpg"}},
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xc1n0g08.png"}}, //color type 1
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xc9n2c08.png"}}, //color type 9
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xcrn0g04.png"}}, //added cr bytes
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xd0n2c08.png"}}, //bit-depth 0
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xd3n2c08.png"}}, //bit-depth 3
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xd9n2c08.png"}}, //bit-depth 99
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xdtn0g01.png"}}, //missing IDAT chunk
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xhdn0g08.png"}}, //incorrect IHDR checksum
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xlfn0g04.png"}}, //added lf bytes
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xs1n0g01.png"}}, //signature byte 1 MSBit reset to zero
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xs2n0g01.png"}}, //signature byte 2 is a 'Q'
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xs4n0g01.png"}}, //signature byte 4 lowercase
		{requestParams{reWidth: 10, reHeight: 0, quality: 80, compression: 6}, []string{"samples", "broken", "xs7n0g01.png"}}, //7th byte a space instead of control-Z
	}

	fmt.Println("\n\nTest bad images:")
	for _, image := range testSetBroken {
		var err error
		sourcePath := path.Join(image.path...)
		fmt.Printf("Trying to resize image\t%-50s\n", sourcePath)

		if image.params.imageBody, err = ioutil.ReadFile(sourcePath); err != nil {
			t.Errorf("IO error on file: %s, err: %s", sourcePath, err)
		}

		if err := resize(&image.params); err == nil {
			t.Errorf("broken image resized without error: %s, err: %s", sourcePath, err)
		}
	}
}
