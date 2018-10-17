package widget

import (
	"image"

	"github.com/ungerik/go-cairo"
)

type Drawer interface {
	Draw(into *cairo.Surface, at image.Rect)
}
