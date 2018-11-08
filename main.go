package main

import (
	"fmt"
	"image"
	"time"


	"github.com/ungerik/go-cairo"
	"github.com/veandco/go-sdl2/sdl"
)

var bench bool = true

func main() {
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

	rover, err := NewRover()
	if err != nil {
		panic(err)
	}

	plot, err := NewPlot()
	if err != nil {
		panic(err)
	}

	running := true
	tick := time.NewTicker(16 * time.Millisecond)
	for running {
		<-tick.C
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break
			}
		}

		now := time.Now()

		cairoSurface.Save()
		rover.Draw(cairoSurface, image.Rect(50, 50, 550, 550))
		cairoSurface.Restore()

		cairoSurface.Save()
		plot.Draw(cairoSurface, image.Rect(600, 50, 1100, 550))
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
