package vgcairo

import (
	"fmt"
	"image"
	"image/color"

	"gonum.org/v1/plot/vg"
	"github.com/ungerik/go-cairo"
)

type Canvas struct {
	gc    *cairo.Surface
	W, H int
}

func (c *Canvas) flip() {
	c.gc.Translate(0, float64(c.H))
	c.gc.Scale(1, -1)
}

func New(w, h vg.Length) *Canvas {
	c := &Canvas{ gc: cairo.NewSurface(cairo.FORMAT_ARGB32, int(w), int(h)), W: int(w), H: int(h), }
	c.flip()
	return c
}

func (c *Canvas) Surface() *cairo.Surface {
	return c.gc
}

func (c *Canvas) SetLineWidth(w vg.Length) {
	c.gc.SetLineWidth(float64(w))
}

func (c *Canvas) SetLineDash(ds []vg.Length, offs vg.Length) {
	if len(ds) == 0 {
		return
	}

	dashes := make([]float64, len(ds))
	for i, l := range ds {
		dashes[i] = l.Points()
	}

	c.gc.SetDash(dashes, len(dashes), offs.Points())
}

func (c *Canvas) SetColor(clr color.Color) {
	if clr == nil {
		clr = color.Black
	}

	r, g, b, a := clr.RGBA()
	c.gc.SetSourceRGBA(float64(r) / 0xffff, float64(g) / 0xffff, float64(b) / 0xffff, float64(a) / 0xffff)
}

func (c *Canvas) Rotate(t float64) {
	c.gc.Rotate(t)
}

func (c *Canvas) Translate(pt vg.Point) {
	c.gc.Translate(float64(pt.X.Points()) , float64(pt.Y.Points()))
}

func (c *Canvas) Scale(x, y float64) {
	c.gc.Scale(x, y)
}

func (c *Canvas) Push() {
	c.gc.Save()
	c.flip()
}

func (c *Canvas) Pop() {
	c.gc.Restore()
}

func (c *Canvas) Stroke(p vg.Path) {
	c.outline(p)
	c.gc.Stroke()
}

func (c *Canvas) Fill(p vg.Path) {
	c.outline(p)
	c.gc.Fill()
}

func (c *Canvas) outline(p vg.Path) {
	c.gc.NewPath()
	for _, comp := range p {
		switch comp.Type {
		case vg.MoveComp:
			c.gc.MoveTo(comp.Pos.X.Points(), comp.Pos.Y.Points())

		case vg.LineComp:
			c.gc.LineTo(comp.Pos.X.Points(), comp.Pos.Y.Points())

		case vg.ArcComp:
			c.gc.Arc(comp.Pos.X.Points(), comp.Pos.Y.Points(),
				comp.Radius.Points(), comp.Start, comp.Angle)

		case vg.CloseComp:
			c.gc.ClosePath()

		default:
			panic(fmt.Sprintf("Unknown path component: %d", comp.Type))
		}
	}
}

func (c *Canvas) Size() (x, y vg.Length) {
	return vg.Length(c.W), vg.Length(c.H)
}

func (c *Canvas) FillString(font vg.Font, pt vg.Point, str string) {
	c.Push()
	c.gc.SetFontSize(font.Size.Points())
	c.gc.SelectFontFace(font.Name(), cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
	c.gc.MoveTo(pt.X.Points(), -pt.Y.Points() + float64(c.H))
	c.gc.ShowText(str)
	c.Pop()
}

func (c *Canvas) DrawImage(rect vg.Rectangle, img image.Image) {
	// FIXME: This is definitely broken.
	// Axes might be flipped, no scaling is performed.
	// Not sure what the co-ordinates are meant to be.

	surf := cairo.NewSurfaceFromImage(img)
	defer surf.Destroy()
	pattern := cairo.NewPatternForSurface(surf)
	defer pattern.Destroy()

	c.Push()
	c.gc.SetSource(pattern)
	c.gc.Rectangle(0, 0, float64(c.W), float64(c.H))
	c.gc.Fill()
	c.Pop()
}
