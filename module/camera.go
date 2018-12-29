package module

import (
	"fmt"
	"image"
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

func (c *Camera) Update() {
	err := c.conn.Call("Telem.GetFrame", true, &c.img)
	if err != nil {
		fmt.Println("Error reading image:", err)
	} else if c.img != nil {
		//_ = algo.FindLine(c.img.(*image.Gray))
		switch v := c.img.(type) {
		/*
		case *image.YCbCr:
			c.iw.SetImage(cv.NewRawYCbCr(v))
		*/
		default:
			_ = v
			c.iw.SetImage(c.img)
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
