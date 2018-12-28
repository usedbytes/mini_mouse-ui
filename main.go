package main

import (
	"encoding/gob"
	"fmt"
	"flag"
	"image"
	"image/png"
	"log"
	"net/rpc"
	"os"
	"time"

	"github.com/ungerik/go-cairo"
	"github.com/veandco/go-sdl2/sdl"

	"github.com/usedbytes/mini_mouse/bot/plan/line/algo"
)

var bench bool = false
var addr string

type Pose struct {
	X, Y float64
	Heading float64
}


func saveCapture(img image.Image) error {
	filename := fmt.Sprintf("captures/capture-%s.png", time.Now().Format("2006-01-02-030405"))
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

func init() {
	gob.Register(&image.NRGBA{})
	gob.Register(&image.Gray{})

	const (
		defaultAddr = "minimouse.local:1234"
		usageAddr   = "Remote address"

	)

	flag.StringVar(&addr, "a", defaultAddr, usageAddr)
}

func main() {
	flag.Parse()

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	c, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		fmt.Println("Couldn't connect to server", addr);
	}

	windowW := 1150
	windowH := 600

	window, err := sdl.CreateWindow("Mini Mouse", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(windowW), int32(windowH), sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	sdlSurface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}

	cairoSurface := cairo.NewSurfaceFromData(sdlSurface.Data(), cairo.FORMAT_ARGB32, int(sdlSurface.W), int(sdlSurface.H), int(sdlSurface.Pitch));

	grad := cairo.NewPatternLinear(cairo.Linear{0, 0, float64(windowW) / 2, float64(windowH) / 2})
	grad.SetExtend(cairo.EXTEND_REFLECT)
	grad.AddColorStopRGB(0, 0, 1.0, 0)
	grad.AddColorStopRGB(1.0, 0, 0, 1.0)
	cairoSurface.SetSource(grad)
	grad.Destroy()
	cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
	cairoSurface.Fill()

	rover, err := NewRover()
	if err != nil {
		panic(err)
	}

	iw := NewImageWidget()

	reconnect := false

	var img image.Image
	var pose Pose
	running := true
	tick := time.NewTicker(16 * time.Millisecond)
	for running {
		<-tick.C
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch ev := event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break
			case *sdl.KeyboardEvent:
				if ev.Keysym.Sym == 's' && ev.State == 0 {
					log.Println("Screenshot")
					if img != nil {
						saveCapture(img)
					}
				}
			}
		}

		if reconnect {
			c, err = rpc.DialHTTP("tcp", addr)
			if err != nil {
				fmt.Println("Couldn't connect to server");
			} else {
				reconnect = false
			}
		}

		now := time.Now()

		if c != nil {
			err = c.Call("Telem.GetFrame", true, &img)
			if err != nil {
				fmt.Println("Error reading image:", err)
				c = nil
				reconnect = true
			} else if img != nil {
				_ = algo.FindLine(img.(*image.Gray))
				iw.SetImage(img)
			}
		}

		if c != nil {
			err = c.Call("Telem.GetPose", true, &pose)
			if err != nil {
				fmt.Println("Error reading pose:", err)
				c = nil
				reconnect = true
			}
		}

		cairoSurface.Save()
		rover.SetHeading(pose.Heading)
		rover.Draw(cairoSurface, image.Rect(50, 50, 550, 550))
		cairoSurface.Restore()

		cairoSurface.Save()
		iw.Draw(cairoSurface, image.Rect(600, 50, 1100, 550))
		cairoSurface.Restore()

		// Finally draw to the screen
		cairoSurface.Flush()
		window.UpdateSurface()

		if bench {
			fmt.Printf("                              \r")
			fmt.Printf("%v\r", time.Since(now))
		}
	}
}
