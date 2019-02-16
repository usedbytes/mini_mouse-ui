package module

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"time"

	"github.com/ungerik/go-cairo"

	_ "github.com/usedbytes/mini_mouse/bot/plan/line/algo"
	"github.com/usedbytes/mini_mouse/ui/conn"
	"github.com/usedbytes/mini_mouse/ui/widget"
	"github.com/usedbytes/mini_mouse/cv"
)

type Camera struct {
	conn *conn.Conn
	iw *widget.ImageWidget

	img image.Image
}

func NewCamera(conn *conn.Conn) *Camera {
	cam := Camera {
		conn: conn,
		iw: widget.NewImageWidget(),
	}

	return &cam
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

func runAlgorithm(in, out image.Image) {
	diff := cv.DeltaCByCol(in)
	minMax := cv.MinMaxRowwise(diff)
	cv.ExpandContrastRowWise(diff, minMax)
	cv.Threshold(diff, 128)

	left := cv.Tuple{ 0, 0 }
	right := cv.Tuple{ 0, 0 }
	n := 0

	for y := 0; y < diff.Bounds().Dy(); y++ {
		blobs := cv.FindBlobs(diff.Pix[y * diff.Stride:y * diff.Stride + diff.Bounds().Dx()])
		if len(blobs) == 2 {
			left.First += blobs[0].First * 2
			left.Second += blobs[0].Second * 2
			right.First += blobs[1].First * 2
			right.Second += blobs[1].Second * 2
			n++
		}
	}

	if n == 0 {
		return
	}

	left.First /= n
	left.Second /= n
	right.First /= n
	right.Second /= n
	fmt.Println(left, right)

	red := &image.Uniform{color.RGBA{0x80, 0, 0, 0x80}}

	rect := image.Rect(left.First, 0, left.Second, out.Bounds().Dy())
	draw.Draw(out.(draw.Image), rect, red, image.ZP, draw.Over)

	rect = image.Rect(right.First, 0, right.Second, out.Bounds().Dy())
	draw.Draw(out.(draw.Image), rect, red, image.ZP, draw.Over)
}

func (c *Camera) Update() {
	err := c.conn.Call("Telem.GetFrame", true, &c.img)
	if err != nil {
		fmt.Println("Error reading image:", err)
	} else if c.img != nil {

		w, h := c.img.Bounds().Dx(), int(float64(c.img.Bounds().Dy()) * 0.685)
		h = h - (h % 2)

		in := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)

		yi := c.img.(*image.YCbCr)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				p := yi.YCbCrAt(x, y)

				yoff := in.YOffset(x, y)
				coff := in.COffset(x, y)

				in.Y[yoff] = p.Y
				in.Cb[coff] = p.Cb
				in.Cr[coff] = p.Cr
			}
		}

		var mod image.Image = image.NewRGBA(in.Bounds())
		draw.Draw(mod.(draw.Image), in.Bounds(), in, image.ZP, draw.Src)

		runAlgorithm(in, mod)

		//_ = algo.FindLine(c.img.(*image.Gray))
		switch v := c.img.(type) {
		/*
		case *image.YCbCr:
			c.iw.SetImage(cv.NewRawYCbCr(v))
		*/
		default:
			_ = v
			c.iw.SetImage(mod)
		}
	}
}

func (c *Camera) Draw(into *cairo.Surface, at image.Rectangle) {
	c.iw.Draw(into, at)
}

func (c *Camera) Snapshot() error {
	if c.img != nil {
		switch v := c.img.(type) {
		case *image.YCbCr:
			return saveCapture(cv.NewRawYCbCr(v))
		default:
			return saveCapture(c.img)
		}
	}

	return nil
}
