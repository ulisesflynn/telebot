package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/saljam/mjpeg"
	"gobot.io/x/gobot"
	g "gobot.io/x/gobot/platforms/dexter/gopigo3"
	"gobot.io/x/gobot/platforms/raspi"
	"gocv.io/x/gocv"
)

const (
	MinimumArea = 3000
)

var (
	deviceID int
	err      error
	webcam   *gocv.VideoCapture
	img      gocv.Mat
	stream   *mjpeg.Stream
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("How to run:\n\tmjpeg-streamer [camera ID] [host:port]")
		return
	}

	raspiAdaptor := raspi.NewAdaptor()
	gopigo3 := g.NewDriver(raspiAdaptor)
	robot := gobot.NewRobot("gopigo3", []gobot.Connection{raspiAdaptor},
		[]gobot.Device{gopigo3},
	)
	robot.Start()

	// parse args
	deviceID, _ = strconv.Atoi(os.Args[1])
	host := os.Args[2]
	xmlFile := os.Args[3]

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

	imgDelta := gocv.NewMat()
	defer imgDelta.Close()

	imgThresh := gocv.NewMat()
	defer imgThresh.Close()

	mog2 := gocv.NewBackgroundSubtractorMOG2()
	defer mog2.Close()

	fmt.Printf("Start reading camera device: %v\n", deviceID)

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	classifier.Load(xmlFile)
	status := "Ready"
	go func() {
		for {
			if ok := webcam.Read(img); !ok {
				fmt.Printf("Error cannot read device %d\n", deviceID)
				return
			}
			if img.Empty() {
				continue
			}

			status = "Ready"
			statusColor := color.RGBA{0, 255, 0, 0}

			// first phase of cleaning up image, obtain foreground only
			mog2.Apply(img, imgDelta)

			// remaining cleanup of the image to use for finding contours
			gocv.Threshold(imgDelta, imgThresh, 25, 255, gocv.ThresholdBinary)
			kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
			gocv.Dilate(imgThresh, imgThresh, kernel)

			contours := gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
			for _, c := range contours {
				area := gocv.ContourArea(c)
				if area < MinimumArea {
					continue
				}

				status = "Motion detected"
				statusColor = color.RGBA{255, 0, 0, 0}
				rect := gocv.BoundingRect(c)
				gocv.Rectangle(img, rect, color.RGBA{255, 0, 0, 0}, 2)
				gopigo3.SetLED(g.LED_EYE_RIGHT, 0xFF, 0x00, 0x00)
			}
			gopigo3.SetLED(g.LED_EYE_RIGHT, 0x00, 0x00, 0x00)
			gocv.PutText(img, status, image.Pt(10, 20), gocv.FontHersheyPlain, 1.2, statusColor, 2)

			buf, _ := gocv.IMEncode(".jpg", img)
			stream.UpdateJPEG(buf)
		}
	}()

	// create stream
	stream = mjpeg.NewStream()

	//go capture()

	http.Handle("/", stream)
	log.Fatal(http.ListenAndServe(host, nil))
}
