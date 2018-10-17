package main

import (
	"image"
	"image/color"

	"github.com/ungerik/go-cairo"
	"github.com/usedbytes/mini_mouse/ui/vgcairo"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type Plot struct {
	p *plot.Plot
	l *plotter.Line
}

func NewPlot() (*Plot, error) {
	p := Plot{ }

	var err error
	p.p, err = plot.New()
	if err != nil {
		return nil, err
	}

	p.l, err = plotter.NewLine(plotter.XYs{{0, 0}, {1, 2}, {2, 2}})
	if err != nil {
		return nil, err
	}

	p.l.LineStyle = draw.LineStyle{
		Color: color.RGBA{ 0xff, 0, 0, 0xff },
		Width: 4,
		Dashes: []vg.Length{5, 5},
	}
	p.p.Add(p.l)

	return &p, nil
}

func (p *Plot) Draw(into *cairo.Surface, at image.Rectangle) {
	w, h := float64(at.Max.X - at.Min.X), float64(at.Max.Y - at.Min.Y)

	canvas := vgcairo.New(vg.Length(w), vg.Length(h))
	p.p.Draw(draw.New(canvas))

	into.Translate(float64(at.Min.X), float64(at.Min.Y))
	into.Rectangle(0, 0, w, h)
	into.Clip()

	into.SetSourceSurface(canvas.Surface(), 0, 0)
	into.Paint()

	canvas.Destroy()
}
