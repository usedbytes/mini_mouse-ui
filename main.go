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
	size int

	iw *ImageWidget
}

func NewDCTriangle(glctx gl.Context, size int) *DCTriangle {
	var err error
	tri := &DCTriangle{ size: size, ctx: glctx, }


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
	varying mediump vec2 v_TexCoord;
	uniform sampler2D tex;
	void main()
	{
		mediump vec4 rgb = texture2D(tex, v_TexCoord);
		gl_FragColor = vec4(rgb.b, rgb.g, rgb.r, 1.0);
	}
	`

	program, err := glutil.CreateProgram(glctx, vertexSrc, fragmentSrc)
	if err != nil {
		log.Fatalf("Couldn't build program %v", err)
	}

	tri.dc = NewDrawcall(glctx, program)
	tri.dc.SetViewport(image.Rect(0, 0, size, size))

	tri.fbo = NewFramebuffer(glctx, size, size, gl.RGBA)

	tri.dc.SetFBO(tri.fbo.Framebuffer)

	vData := f32.Bytes(binary.LittleEndian,
		 0.0, -0.5, 0.5, 0.0,
		-0.5,  0.5, 0.0, 1.0,
		 0.5,  0.5, 1.0, 1.0,
	)
	tri.dc.SetVertexData(vData)
	tri.dc.SetIndices([]uint16{0, 1, 2})
	tri.dc.SetAttribute("position", 2, 4, 0)
	tri.dc.SetAttribute("tc", 2, 4, 2)

	infile, err := os.Open("bb.png")
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

	t.dc.Draw()
	t.dc.ctx.Finish()

	im := t.fbo.GetImage(t.ctx)
	t.iw.SetImage(im)
	t.iw.Draw(into, at)
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

	tri := NewDCTriangle(glctx, 500)

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
		tri.Draw(cairoSurface, image.Rect(250, 50, 750, 550))
		cairoSurface.Restore()

		// Finally draw to the screen
		cairoSurface.Flush()
		window.UpdateSurface()

		if bench {
			fmt.Printf("                              \r")
			fmt.Printf("%v\r", time.Since(now))
		}
	}
}
