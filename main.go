package main

import (
	"fmt"
	"math"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/ungerik/go-cairo"
	"github.com/usedbytes/mini_mouse/ui/pose"
	"github.com/usedbytes/bot_matrix/datalink"
	"github.com/usedbytes/bot_matrix/datalink/rpcconn"
	"time"
)

func main() {
	fmt.Println("Mini Mouse UI")
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	devname := "tcp:localhost:5556"
	c, err := rpcconn.NewRPCClient(devname[len("tcp:"):])
	if err != nil {
		fmt.Println("Couldn't connect to server");
	}

	windowW := 800
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
	cairoSurface.SelectFontFace("serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
	cairoSurface.SetFontSize(32.0)

	tick := time.NewTicker(16 * time.Millisecond)

	outline, status := cairo.NewSurfaceFromPNG("outline.png")
	if status != cairo.STATUS_SUCCESS {
		panic("erk")
	}

	rot := float64(0)

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

		if c != nil {
			pkts, err := c.Transact([]datalink.Packet{})
			if err != nil {
				fmt.Println("Error reading datalink")
			}

			for _, p := range pkts {
				if p.Endpoint == pose.Endpoint {
					report := ((&pose.Pose{}).Receive(&p)).(*pose.PoseReport)
					rot = report.Heading * math.Pi / 180.0
				}
			}
		}

		// Clear the whole window (black)
		cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
		cairoSurface.SetSourceRGB(0.0, 0.0, 0.0)
		cairoSurface.Fill()

		// Draw to a sub-window (500x500)
		cairoSurface.PushGroup()
		wwidth := float64(500)
		wheight := float64(500)
		width := float64(outline.GetWidth())
		height := float64(outline.GetHeight())

		// sub-window background/outline
		cairoSurface.Rectangle(0, 0, wwidth, wheight)
		cairoSurface.SetSourceRGB(0.3, 0.3, 0.3)
		cairoSurface.FillPreserve()
		cairoSurface.SetSourceRGB(1.0, 0.0, 0.0)
		cairoSurface.Stroke()

		// Draw texture in middle of sub-window, rotated
		cairoSurface.Translate(float64(wwidth / 2), float64(wheight / 2))
		cairoSurface.Rotate(rot)
		cairoSurface.SetSourceSurface(outline, -width / 2, -height / 2)
		cairoSurface.Rectangle(-width / 2, -height / 2, width, height)
		cairoSurface.Fill()

		// Grab the sub-window pattern
		pattern := cairoSurface.PopGroup()

		// Position the sub-window
		translate := cairo.Matrix{}
		translate.InitTranslate(50, 50)
		translate.Invert()
		pattern.SetMatrix(translate)

		cairoSurface.SetSource(pattern)
		pattern.Destroy()
		cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
		cairoSurface.Fill()

		// Finally draw to the screen
		cairoSurface.Flush()
		window.UpdateSurface()
	}
}
