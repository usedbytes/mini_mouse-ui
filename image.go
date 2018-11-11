package main

import (
	"image"
	"math"

	"github.com/ungerik/go-cairo"
)

type ImageWidget struct {
	img image.Image
	surface *cairo.Surface
	w, h float64
}

func NewImageWidget() *ImageWidget {
	iw := ImageWidget{}

	return &iw
}

func (iw *ImageWidget) Draw(into *cairo.Surface, at image.Rectangle) {
	w, h := float64(at.Dx()), float64(at.Dy())

	into.Save()
	defer into.Restore()

	into.Translate(float64(at.Min.X), float64(at.Min.Y))

	into.Rectangle(0, 0, w, h)
	into.SetSourceRGBA(0.0, 0.0, 1.0, 1.0)
	into.ClipPreserve()
	into.Fill()

	if iw.surface != nil {
		scale := math.Min(w / iw.w, h / iw.h)
		into.Save()
		into.Scale(scale, scale)
		into.SetSourceSurface(iw.surface, 0, 0)
		p := into.GetSource()
		p.SetFilter(cairo.CAIRO_FILTER_NEAREST)
		into.Paint()
		into.Restore()
	}
}

func (iw *ImageWidget) SetImage(img image.Image) {
	if iw.surface != nil {
		iw.surface.Destroy()
		iw.surface = nil
	}

	iw.img = img
	if iw.img != nil {
		iw.w, iw.h = float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
		iw.surface = cairo.NewSurfaceFromImage(img)
	}
}
