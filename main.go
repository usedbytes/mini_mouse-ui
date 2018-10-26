package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"time"


	"github.com/ungerik/go-cairo"
	"github.com/ungerik/go-cairo/extimage"
	"github.com/usedbytes/hlegl"
	"github.com/usedbytes/hsv"
	"github.com/veandco/go-sdl2/sdl"

	"golang.org/x/mobile/gl"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"
)

var bench bool = true

type Framebuffer struct {
	gl.Framebuffer
	Tex gl.Texture
	Width, Height int
	Format gl.Enum
}

func NewTextureFromImage(glctx gl.Context, img *image.NRGBA) gl.Texture {
	tex := glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, tex)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, img.Bounds().Dx(), img.Bounds().Dy(), gl.RGBA, gl.UNSIGNED_BYTE, img.Pix)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	return tex
}

func NewFramebuffer(glctx gl.Context, width, height int, format gl.Enum) *Framebuffer {
	fbo := Framebuffer{
		Width: width,
		Height: height,
		Format: format,
	}

	fbo.Framebuffer = glctx.CreateFramebuffer()
	fbo.Tex = glctx.CreateTexture()

	glctx.BindFramebuffer(gl.FRAMEBUFFER, fbo.Framebuffer)
	glctx.BindTexture(gl.TEXTURE_2D, fbo.Tex)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, int(format), width, height, format, gl.UNSIGNED_BYTE, nil)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fbo.Tex, 0)

	result := glctx.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if result != gl.FRAMEBUFFER_COMPLETE {
		log.Println("Error: Framebuffer not complete.")
		return nil
	}

	return &fbo
}

func (f *Framebuffer) GetImage(glctx gl.Context) image.Image {

	glctx.BindFramebuffer(gl.FRAMEBUFFER, f.Framebuffer)
	dst := make([]uint8, int(f.Width * f.Height * 4))
	glctx.ReadPixels(dst, 0, 0, f.Width, f.Height, gl.RGBA, gl.UNSIGNED_BYTE)

	return &extimage.BGRA{
		Pix: dst,
		Stride: f.Width * 4,
		Rect: image.Rect(0, 0, f.Width, f.Height),
	}
}

type DCTriangle struct {
	ctx gl.Context
	fbo *Framebuffer
	dc *Drawcall
	color gl.Uniform

	hsv hsv.HSVColor
	width, height int

	iw *ImageWidget
}


func generateMipViewports(contentSize image.Point, xFactor, yFactor float64) []image.Rectangle {
	minx, miny := 0, 0
	w, h := float64(contentSize.X), float64(contentSize.Y)

	xStride, yStride := 0.0, 0.0
	if xFactor <= yFactor {
		xStride = 1.0
	}
	if yFactor < xFactor {
		yStride = 1.0
	}

	ret := make([]image.Rectangle, 0)
	for i := 0; w >= 1 && h >= 1; i++ {
		ret = append(ret, image.Rect(minx, miny, minx + int(w), miny + int(h)))

		minx += int(w * xStride)
		miny += int(h * yStride)
		w *= xFactor
		h *= yFactor
	}

	return ret
}

func rectToUV(rect image.Rectangle, textureSize image.Point) (x1, y1, x2, y2 float32) {
	return float32(rect.Min.X) / float32(textureSize.X),
	       float32(rect.Min.Y) / float32(textureSize.Y),
	       float32(rect.Max.X) / float32(textureSize.X),
	       float32(rect.Max.Y) / float32(textureSize.Y)
}


func roundToPower2(x int) int {
	x2 := 1
	for x2 < x {
		x2 *= 2
	}
	return x2
}

func NewDCTriangle(glctx gl.Context, width, height int) *DCTriangle {
	var err error
	tri := &DCTriangle{ width: width, height: height, ctx: glctx, }


	vertexSrc := `
	#version 100
	attribute vec2 position;
	attribute vec2 tc;
	varying mediump vec2 v_TexCoord;

	void main()
	{
		gl_Position = vec4(position, 0.0, 1.0);
		v_TexCoord = tc;
	}
	`

	fragmentSrc := `
	#version 100
	precision mediump float;
	uniform sampler2D tex;
	varying mediump vec2 v_TexCoord;
	void main()
	{
		vec4 rgb = texture2D(tex, v_TexCoord);

		gl_FragColor = vec4(rgb.b, rgb.g, rgb.r, 1.0);
	}
	`

	program, err := glutil.CreateProgram(glctx, vertexSrc, fragmentSrc)
	if err != nil {
		log.Fatalf("Couldn't build program %v", err)
	}

	tri.dc = NewDrawcall(glctx, program)
	tri.dc.SetViewport(image.Rect(0, 0, width, height))

	tri.fbo = NewFramebuffer(glctx, width * 2, height, gl.RGBA)

	tri.dc.SetFBO(tri.fbo.Framebuffer)

	vData := f32.Bytes(binary.LittleEndian,
		-1.0,  1.0, 0.0, 1.0,
		-1.0, -1.0, 0.0, 0.0,
		 1.0,  1.0, 1.0, 1.0,
		 1.0, -1.0, 1.0, 0.0,
	)
	tri.dc.SetVertexData(vData)
	tri.dc.SetIndices([]uint16{0, 1, 2, 3})
	tri.dc.SetAttribute("position", 2, 4, 0)
	tri.dc.SetAttribute("tc", 2, 4, 2)

	infile, err := os.Open("patch.png")
	if err != nil {
		panic(err)
	}
	defer infile.Close()

	img, err := png.Decode(infile)
	if err != nil {
		panic(err)
	}

	tex := NewTextureFromImage(glctx, img.(*image.NRGBA))

	tri.dc.SetTexture("tex", tex)
	//tri.dc.SetUniformf("step", []float32{-1 / 16.0})

	tri.hsv = hsv.HSVColor{
		0, 255, 255,
	}

	tri.iw = NewImageWidget()

	return tri
}

func (t *DCTriangle) Draw(into *cairo.Surface, at image.Rectangle) {

	r, g, b, _ := t.hsv.RGBA()
	t.dc.SetUniformf("color", []float32{float32(r) / 65535.0, float32(g) / 65535.0, float32(b) / 65535.0, 1})
	t.hsv.H += 1

	rects := generateMipViewports(image.Pt(t.width, t.height), 0.5, 0.5)
	for _, vp := range rects {
		t.dc.SetViewport(vp)
		t.dc.Draw()
	}
	t.dc.ctx.Finish()

	im := t.fbo.GetImage(t.ctx)
	t.iw.SetImage(im)
	t.iw.Draw(into, at)
}

type Recursor struct {
	ctx gl.Context

	oddFBO *Framebuffer
	evenFBO *Framebuffer

	orig gl.Texture

	rects []image.Rectangle

	dc *Drawcall

	width, height int

	iw *ImageWidget
}

func NewRecursor(glctx gl.Context, width, height int) *Recursor {
	var err error
	rec := &Recursor{ width: width, height: height, ctx: glctx, }


	vertexSrc := `
	#version 100
	attribute vec2 position;
	attribute vec2 tc;
	varying mediump vec2 v_TexCoord;

	void main()
	{
		gl_Position = vec4(position, 0.0, 1.0);
		v_TexCoord = tc;
	}
	`

	fragmentSrc := `
	#version 100
	precision mediump float;
	uniform sampler2D tex;
	varying mediump vec2 v_TexCoord;
	void main()
	{
		vec4 rgb = texture2D(tex, v_TexCoord);

		gl_FragColor = vec4(rgb.b, rgb.g, rgb.r, 1.0);
	}
	`

	program, err := glutil.CreateProgram(glctx, vertexSrc, fragmentSrc)
	if err != nil {
		log.Fatalf("Couldn't build program %v", err)
	}

	rec.dc = NewDrawcall(glctx, program)

	xFactor, yFactor := 2, 1

	oddRects := generateMipViewports(image.Pt(width / xFactor, height / yFactor), 1.0 / float64(xFactor * xFactor), 1.0 / float64(yFactor * yFactor))
	max := 0
	for _, r := range oddRects {
		if r.Max.X > max {
			max = r.Max.X
		}
		if r.Max.Y > max {
			max = r.Max.Y
		}
	}
	max = roundToPower2(max)
	rec.oddFBO = NewFramebuffer(glctx, max, max, gl.RGBA)

	evenRects := generateMipViewports(image.Pt(width / (xFactor * xFactor), height / (yFactor * yFactor)), 1.0 / float64(xFactor * xFactor), 1.0 / float64(yFactor * yFactor))
	max = 0
	for _, r := range evenRects {
		if r.Max.X > max {
			max = r.Max.X
		}
		if r.Max.Y > max {
			max = r.Max.Y
		}
	}
	max = roundToPower2(max)
	rec.evenFBO = NewFramebuffer(glctx, max, max, gl.RGBA)

	rec.rects = make([]image.Rectangle, len(oddRects) + len(evenRects))
	//rec.rects[0] = image.Rect(0, 0, width, height)
	for i := 0; i < len(oddRects) * 2; i += 2 {
		rec.rects[i] = oddRects[i / 2]
	}
	for i := 0; i < len(evenRects) * 2; i += 2 {
		rec.rects[i + 1] = evenRects[i / 2]
	}

	for i, r := range rec.rects {
		fmt.Println(i, ": ", r)
	}

	w, h := width, height
	vertices := make([]float32, 0, (len(rec.rects)) * 16)

	// Special case for first pass - full original texture
	vertices = append(vertices,
		-1.0,  1.0, 0.0, 1.0,
		-1.0, -1.0, 0.0, 0.0,
		 1.0,  1.0, 1.0, 1.0,
		 1.0, -1.0, 1.0, 0.0,
	 )

	 for i := 1; i < len(rec.rects); i++ {
		rect := rec.rects[i - 1]
		var srcSize image.Point
		if (i & 1) != 0 {
			srcSize = image.Pt(rec.oddFBO.Width, rec.oddFBO.Height)
		} else {
			srcSize = image.Pt(rec.evenFBO.Width, rec.evenFBO.Height)
		}

		x1, y1, x2, y2 := rectToUV(rect, srcSize)
		w /= xFactor
		h /= yFactor

		vertices = append(vertices,
			-1.0,  1.0, x1, y2,
			-1.0, -1.0, x1, y1,
			 1.0,  1.0, x2, y2,
			 1.0, -1.0, x2, y1,
		 )
	}

	for i := 0; i < len(rec.rects) - 1; i++ {
		fmt.Println(vertices[i * 16 + 0: i * 16 + 0 + 4])
		fmt.Println(vertices[i * 16 + 4: i * 16 + 4 + 4])
		fmt.Println(vertices[i * 16 + 8: i * 16 + 8 + 4])
		fmt.Println(vertices[i * 16 + 12: i * 16 + 12 + 4])
		fmt.Println("---")
	}

	vData := f32.Bytes(binary.LittleEndian, vertices...)
	rec.dc.SetVertexData(vData)
	rec.dc.SetIndices([]uint16{0, 1, 2, 3})
	rec.dc.SetAttribute("position", 2, 4, 0)
	rec.dc.SetAttribute("tc", 2, 4, 2)

	infile, err := os.Open("tex.png")
	if err != nil {
		panic(err)
	}
	defer infile.Close()

	img, err := png.Decode(infile)
	if err != nil {
		panic(err)
	}

	rec.orig = NewTextureFromImage(glctx, img.(*image.NRGBA))

	rec.iw = NewImageWidget()

	return rec
}

func (rec *Recursor) Draw(into *cairo.Surface, at image.Rectangle) {
	w2 := at.Bounds().Dx() / 2
	left := image.Rect(at.Bounds().Min.X, at.Bounds().Min.Y, at.Bounds().Min.X + w2, at.Bounds().Max.Y)
	right := image.Rect(at.Bounds().Min.X + w2, at.Bounds().Min.Y, at.Bounds().Max.X, at.Bounds().Max.Y)

	// Special case:
	rec.dc.SetAttribute("position", 2, 4, 0)
	rec.dc.SetAttribute("tc", 2, 4, 2)
	rec.dc.SetTexture("tex", rec.orig)
	rec.dc.SetViewport(rec.rects[0])
	rec.dc.SetFBO(rec.oddFBO.Framebuffer)
	rec.dc.Draw()

	src := rec.oddFBO
	dst := rec.evenFBO
	for i, r := range rec.rects[1:] {
		rec.dc.SetAttribute("position", 2, 4, (i + 1) * 4 * 4)
		rec.dc.SetAttribute("tc", 2, 4, 2 + (i + 1) * 4 * 4)
		rec.dc.SetTexture("tex", src.Tex)
		rec.dc.SetViewport(r)
		rec.dc.SetFBO(dst.Framebuffer)
		rec.dc.Draw()

		tmp := src
		src = dst
		dst = tmp
	}

	rec.dc.ctx.Finish()

	rec.iw.SetImage(rec.oddFBO.GetImage(rec.ctx))
	rec.iw.Draw(into, left)

	rec.iw.SetImage(rec.evenFBO.GetImage(rec.ctx))
	rec.iw.Draw(into, right)
}

func main() {

	glctx := hlegl.Initialise()
	if glctx == nil {
		panic("Couldn't get GL context")
	}

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

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

	rover, err := NewRover()
	if err != nil {
		panic(err)
	}

	plot, err := NewPlot()
	if err != nil {
		panic(err)
	}

	//tri := NewDCTriangle(glctx, 128, 128)

	rec := NewRecursor(glctx, 128, 128)

	running := true
	tick := time.NewTicker(16 * time.Millisecond)
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

		now := time.Now()

		cairoSurface.Save()
		rover.Draw(cairoSurface, image.Rect(50, 50, 550, 550))
		cairoSurface.Restore()

		cairoSurface.Save()
		plot.Draw(cairoSurface, image.Rect(600, 50, 1100, 550))
		cairoSurface.Restore()

		cairoSurface.Save()
		rec.Draw(cairoSurface, image.Rect(50, 50, 1050, 550))
		cairoSurface.Restore()
		/*
		col := hsv.HSVColor{
			0, 255, 255,
		}
		ret := generateMipViewports(image.Pt(256, 256), 0.3, 0.3)
		for _, rect := range ret {
			r, g, b, _ := col.RGBA()
			cairoSurface.SetSourceRGB(float64(r) / 65535.0, float64(g) / 65535.0, float64(b) / 65535.0)
			cairoSurface.Rectangle(float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()))
			cairoSurface.Fill()
			col.H += 50
			fmt.Println(rect.Dx(), " x ", rect.Dy())
		}
		*/

		// Finally draw to the screen
		cairoSurface.Flush()
		window.UpdateSurface()

		if bench {
			fmt.Printf("                              \r")
			fmt.Printf("%v\r", time.Since(now))
		}
	}
}
