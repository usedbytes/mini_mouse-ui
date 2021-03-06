package main

import (
	"fmt"
	"image"
	"math"

	"github.com/ungerik/go-cairo"
)

type Rover struct {
	sprite *cairo.Surface
	w, h float64
	theta float64
}

func NewRover() (*Rover, error) {
	r := Rover{}

	sprite, status := cairo.NewSurfaceFromPNG("outline.png")
	if status != cairo.STATUS_SUCCESS {
		return nil, fmt.Errorf("Couldn't load sprite image")
	}
	r.sprite = sprite

	r.w, r.h = float64(r.sprite.GetWidth()), float64(r.sprite.GetHeight())

	return &r, nil
}

func (r *Rover) Draw(into *cairo.Surface, at image.Rectangle) {
	w, h := float64(at.Max.X - at.Min.X), float64(at.Max.Y - at.Min.Y)
	scale := math.Min(w / r.w, h / r.h)

	into.Translate(float64(at.Min.X), float64(at.Min.Y))
	into.Rectangle(0, 0, w, h)
	into.ClipPreserve()
	into.SetSourceRGB(0.3, 0.3, 0.3)
	into.FillPreserve()
	into.SetSourceRGB(1.0, 0.0, 0.0)
	into.Stroke()

	into.Save()
	into.Translate(w / 2, h / 2)
	into.Scale(scale, scale)
	into.Rotate(r.theta)
	into.SetSourceSurface(r.sprite, -r.w / 2, -r.h / 2)
	into.Paint()
	into.Restore()
}

func (r *Rover) SetHeading(degrees float64) {
	r.theta = degrees * math.Pi / 180.0
}
