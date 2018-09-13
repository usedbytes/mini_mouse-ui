package main

import (
	"fmt"
	"math"
	"net/rpc"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/ungerik/go-cairo"
	"time"

	"image"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

var bench bool

var roverSprite *cairo.Surface
func drawRover(cairoSurface *cairo.Surface, heading float64) *cairo.Pattern {
	if roverSprite == nil {
		var status cairo.Status
		roverSprite, status = cairo.NewSurfaceFromPNG("outline.png")
		if status != cairo.STATUS_SUCCESS {
			panic("erk")
		}
	}

	// Draw to a sub-window (500x500)
	cairoSurface.PushGroup()
	wwidth := float64(500)
	wheight := float64(500)
	width := float64(roverSprite.GetWidth())
	height := float64(roverSprite.GetHeight())

	// sub-window background/outline
	cairoSurface.Rectangle(0, 0, wwidth, wheight)
	cairoSurface.SetSourceRGB(0.3, 0.3, 0.3)
	cairoSurface.FillPreserve()
	cairoSurface.SetSourceRGB(1.0, 0.0, 0.0)
	cairoSurface.Stroke()

	// Draw texture in middle of sub-window, rotated
	cairoSurface.Translate(float64(wwidth / 2), float64(wheight / 2))
	cairoSurface.Rotate(heading)
	cairoSurface.SetSourceSurface(roverSprite, -width / 2, -height / 2)
	cairoSurface.Rectangle(-width / 2, -height / 2, width, height)
	cairoSurface.Fill()

	// Grab the sub-window pattern
	return cairoSurface.PopGroup()
}

func drawPlot(surf *cairo.Surface) *cairo.Pattern {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	l, err := plotter.NewLine(plotter.XYs{{0, 0}, {1, 1}, {2, 2}})
	if err != nil {
		panic(err)
	}
	p.Add(l)

	// Draw the plot to an in-memory image.
	dest := image.NewRGBA(image.Rect(0, 0, 500, 500))
	c := vgimg.NewWith(vgimg.UseImage(dest))
	p.Draw(draw.New(c))

	outSurf := cairo.NewSurfaceFromImage(dest)
	defer outSurf.Destroy()
	return cairo.NewPatternForSurface(outSurf)
}

func main() {
	fmt.Println("Mini Mouse UI")
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	//devname := "tcp:minimouse.local:1234"
	devname := "tcp:localhost:1234"
	c, err := rpc.DialHTTP("tcp", devname[len("tcp:"):])
	if err != nil {
		fmt.Println("Couldn't connect to server");
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
	cairoSurface.SelectFontFace("serif", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
	cairoSurface.SetFontSize(32.0)

	tick := time.NewTicker(16 * time.Millisecond)

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
			var vec []float64
			err = c.Call("Telem.GetEuler", true, &vec)
			if err != nil {
				fmt.Println("Error reading vector:", err)

			} else {
				rot = vec[0] * math.Pi / 180.0
			}
		}

		now := time.Now()

		// Clear the whole window (black)
		cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
		cairoSurface.SetSourceRGB(0.0, 0.0, 0.0)
		cairoSurface.Fill()

		grad := cairo.NewPatternLinear(cairo.Linear{0, 0, float64(windowW) / 2, float64(windowH) / 2})
		grad.SetExtend(cairo.EXTEND_REFLECT)
		grad.AddColorStopRGB(0, 0, 1.0, 0)
		grad.AddColorStopRGB(1.0, 0, 0, 1.0)
		cairoSurface.SetSource(grad)
		grad.Destroy()

		cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
		cairoSurface.Fill()

		pattern := drawRover(cairoSurface, rot)
		// Position the sub-window
		translate := cairo.Matrix{}
		translate.InitTranslate(50, 50)
		translate.Invert()
		pattern.SetMatrix(translate)
		cairoSurface.SetSource(pattern)
		pattern.Destroy()
		cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
		cairoSurface.Fill()

		pattern = drawPlot(cairoSurface)
		translate = cairo.Matrix{}
		translate.InitTranslate(600, 50)
		translate.Invert()
		pattern.SetMatrix(translate)
		cairoSurface.SetSource(pattern)
		pattern.Destroy()
		cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
		cairoSurface.Fill()

		// Finally draw to the screen
		cairoSurface.Flush()
		window.UpdateSurface()

		if bench {
			fmt.Println(time.Since(now))
			now = time.Now()
		}
	}
}
