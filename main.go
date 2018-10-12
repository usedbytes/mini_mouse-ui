package main

import (
	"fmt"
	"math"
	"net/rpc"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/ungerik/go-cairo"
	"time"

	"gocv.io/x/gocv"

	"github.com/usedbytes/mini_mouse/ui/vgcairo"
	"image"
	"image/color"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"

	"net"

	"github.com/usedbytes/bot_matrix/datalink"
	"github.com/usedbytes/bot_matrix/datalink/netconn"

	"github.com/usedbytes/mini_mouse/ui/imgpkt"
)

var bench bool = true

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
	l, err := plotter.NewLine(plotter.XYs{{0, 0}, {1, 2}, {2, 2}})
	if err != nil {
		panic(err)
	}

	l.LineStyle = draw.LineStyle{
		Color: color.RGBA{ 0xff, 0, 0, 0xff },
		Width: 4,
		Dashes: []vg.Length{5, 5},
	}
	p.Add(l)

	// Draw the plot to an in-memory image.
	c := vgcairo.New(500, 500)
	p.Draw(draw.New(c))

	return cairo.NewPatternForSurface(c.Surface())
}

func expandContrastRowWise(m *gocv.Mat) {
	shape := m.Size()

	for y := 0; y < shape[1]; y++ {
		line := m.Region(image.Rect(0, y, shape[0], y + 1))

		min, max, _, _ := gocv.MinMaxIdx(line)
		factor := 255.0 / float32(max - min)
		if min == max {
			factor = 0.0
		}

		line.SubtractUChar(uint8(min))
		line.MultiplyFloat(factor)
		line.Close()
	}
}

func findMiddles(m *gocv.Mat) [][]float32 {
	shape := m.Size()
	middles := make([][]float32, shape[1])

	for y := 0; y < shape[0]; y++ {
		middles[y] = make([]float32, 0, 2)
		line := m.Region(image.Rect(0, y, shape[1], y + 1))
		dat := line.DataPtrUint8()

		in := false
		start := 0
		for x, v := range dat {
			if v == 0 {
				if in {
					if (x - start >= 2) {
						middles[y] = append(middles[y], float32(start + x) / 2)
					}
					in = false
				}
			} else {
				if !in {
					start = x
					in = true
				}
			}
		}
		if in {
			if (shape[1] - start >= 2) {
				middles[y] = append(middles[y], float32(start + shape[1]) / 2)
			}
		}
		line.Close()
	}
	return middles
}

var cvWin *gocv.Window
func doCv(ip *imgpkt.ImagePacket) [][]image.Point {
	src, err := gocv.NewMatFromBytes(int(ip.Height), int(ip.Width), gocv.MatTypeCV8UC4, ip.Data)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	size := src.Size()

	img := src.Region(image.Rect(0, size[0] / 2, size[1], size[0]))
	size = img.Size()

	gray := gocv.NewMat()
	gocv.CvtColor(img, &gray, gocv.ColorBGRAToGray)

	minSize := image.Point{16, 16}
	hScale := float32(size[1]) / float32(minSize.X)
	vScale := float32(size[0]) / float32(minSize.Y)

	small := gocv.NewMat()
	gocv.Resize(gray, &small, minSize, 0, 0, gocv.InterpolationLinear)
	gray.Close()
	expandContrastRowWise(&small)

	binary := gocv.NewMat()
	gocv.Threshold(small, &binary, 127.0, 255.0, gocv.ThresholdBinary)
	small.Close()

	middles := findMiddles(&binary)
	binary.Close()

	scaledMiddles := make([][]image.Point, len(middles))
	for i, row := range middles {
		scaledMiddles[i] = make([]image.Point, len(row))
		for j, m := range row {
			scaledMiddles[i][j] = image.Pt(int(float32(m) * hScale), int((float32(i) + 0.5) * vScale) + (size[0]))
		}
	}

	img.Close()
	src.Close()

	return scaledMiddles
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func findClosest(points []image.Point, pt image.Point) int {
	min := 99999
	mindx := 0
	for i, p := range points {
		dst := abs(pt.X - p.X)
		if dst < min {
			min = dst
			mindx = i
		}
	}
	return mindx
}

func findLine(w int, scaledMiddles [][]image.Point) []image.Point {
	dx := 0
	linePoints := make([]image.Point, len(scaledMiddles))

	var current int
	for _, row := range scaledMiddles {
		if len(row) == 0 {
			continue
		}
		current = row[0].X
	}

	for i, row := range scaledMiddles {
		if len(row) == 0 {
			continue
		}
		pred := current + dx
		idx := findClosest(row, image.Pt(pred, 0))
		dx = row[idx].X - current
		linePoints[i] = row[idx]
	}

	return linePoints
}

func main() {
	fmt.Println("Mini Mouse UI")
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	cvWin = gocv.NewWindow("Hello")

	//devname := "tcp:minimouse.local:1234"
	devname := "tcp:localhost:1234"
	c, err := rpc.DialHTTP("tcp", devname[len("tcp:"):])
	if err != nil {
		fmt.Println("Couldn't connect to server");
	}

	var t datalink.Transactor
	//nc, err := net.Dial("tcp", "minimouse.local:9876")
	nc, err := net.Dial("tcp", "localhost:9876")
	if err != nil {
		fmt.Println("Couldn't connect to image server");
	} else {
		t = netconn.NewNetconn(nc)
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

		if t != nil {
			rxp, err := t.Transact([]datalink.Packet{})
			if err != nil {
				fmt.Println(err)
			}

			for _, p := range rxp {
				ip, err := imgpkt.UnMarshal(&p)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Printf("Received image %d x %d\n", ip.Width, ip.Height)
					rsurf := cairo.NewSurface(cairo.FORMAT_ARGB32, int(ip.Width), int(ip.Height))
					rsurf.SetData(ip.Data)

					middles := doCv(&ip)

					rsurf.SetSourceRGB(1.0, 0.0, 0.0)
					for _, row := range middles {
						for _, m := range row {
							rsurf.Rectangle(float64(m.X - 2), float64(m.Y - 2), 4, 4)
							rsurf.Fill()
						}
					}

					for i, j := 0, len(middles)-1; i < j; i, j = i+1, j-1 {
						middles[i], middles[j] = middles[j], middles[i]
					}
					line := findLine(int(ip.Width), middles)

					rsurf.SetSourceRGB(0.0, 1.0, 0.0)
					for _, pt := range line {
						rsurf.Rectangle(float64(pt.X - 1), float64(pt.Y - 1), 2, 2)
						rsurf.Fill()
					}
					rsurf.Flush()

					pattern := cairo.NewPatternForSurface(rsurf)
					translate := cairo.Matrix{}
					scalef := float64(500) / float64(ip.Width)
					translate.InitTranslate(600, 50)
					translate.Scale(scalef, scalef)
					translate.Invert()

					pattern.SetMatrix(translate)
					cairoSurface.SetSource(pattern)
					cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
					cairoSurface.Fill()

					pattern.Destroy()
					rsurf.Destroy()
				}
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

		/*
		pattern = drawPlot(cairoSurface)
		translate = cairo.Matrix{}
		translate.InitTranslate(600, 50)
		translate.Invert()
		pattern.SetMatrix(translate)
		cairoSurface.SetSource(pattern)
		pattern.Destroy()
		cairoSurface.Rectangle(0, 0, float64(windowW), float64(windowH))
		cairoSurface.Fill()
		*/

		// Finally draw to the screen
		cairoSurface.Flush()
		window.UpdateSurface()

		if bench {
			fmt.Println(time.Since(now))
			now = time.Now()
		}
	}
}
