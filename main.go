package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/ungerik/go-cairo"
	"time"

	"image"

)

var bench bool

func main() {
	fmt.Println("Mini Mouse UI")
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

	rover, err := NewRover()
	if err != nil {
		panic(err)
	}

	plot, err := NewPlot()
	if err != nil {
		panic(err)
	}

	cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
	cairoSurface.Fill()

	tick := time.NewTicker(16 * time.Millisecond)

	running := true
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
			fmt.Println(time.Since(now))
			now = time.Now()
		}
	}
}
