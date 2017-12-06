package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/saljam/mjpeg"
	"gocv.io/x/gocv"
)

var (
	deviceID int
	err      error
	webcam   *gocv.VideoCapture
	img      gocv.Mat

	stream *mjpeg.Stream
)

func capture() {
	for {
		if ok := webcam.Read(img); !ok {
			fmt.Printf("cannot read device: %d\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}
		buf, _ := gocv.IMEncode(".jpg", img)
		stream.UpdateJPEG(buf)
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("How to run:\n\tmjpeg-streamer [camera ID] [host:port]")
		return
	}

	// parse args
	deviceID, _ = strconv.Atoi(os.Args[1])
	host := os.Args[2]

	// open webcam
	webcam, err = gocv.VideoCaptureDevice(deviceID)
	if err != nil {
		fmt.Printf("error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	// prepare image matrix
	img = gocv.NewMat()
	defer img.Close()

	// create stream
	stream = mjpeg.NewStream()

	go capture()

	http.Handle("/", stream)
	log.Fatal(http.ListenAndServe(host, nil))
}
