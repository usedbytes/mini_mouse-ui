package main

import (
	"encoding/gob"
	"fmt"
	"flag"
	"image"
	"log"
	"time"

	"github.com/ungerik/go-cairo"
	"github.com/veandco/go-sdl2/sdl"

	"github.com/usedbytes/mini_mouse/ui/conn"
	"github.com/usedbytes/mini_mouse/ui/module"
)

var bench bool
var addr string

func init() {
	gob.Register(&image.NRGBA{})
	gob.Register(&image.Gray{})
	gob.Register(&image.YCbCr{})

	const (
		defaultAddr = "minimouse.local:1234"
		usageAddr   = "Remote address"

		defaultBench = false
		usageBench   = "Measure drawing time"

	)

	flag.StringVar(&addr, "a", defaultAddr, usageAddr)
	flag.BoolVar(&bench, "b", defaultBench, usageBench)
}

func main() {
	flag.Parse()

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

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

	c, err := conn.NewConn(addr)
	if err != nil {
		fmt.Println("Couldn't connect to server", addr);
	}

	rover, err := module.NewRover(c)
	if err != nil {
		panic(err)
	}

	cam := module.NewCamera(c)

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
					log.Println("Snapshot")
					cam.Snapshot()
				}
			}
		}

		now := time.Now()

		cam.Update()
		rover.Update()

		cairoSurface.Save()
		rover.Draw(cairoSurface, image.Rect(50, 50, 550, 550))
		cairoSurface.Restore()

		cairoSurface.Save()
		cam.Draw(cairoSurface, image.Rect(600, 50, 1100, 550))
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
