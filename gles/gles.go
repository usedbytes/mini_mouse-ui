package gles

import (
	"encoding/binary"
	"image"
	"log"
	"os"
	"image/png"

	"github.com/ungerik/go-cairo/extimage"

	"golang.org/x/mobile/gl"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"

)

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
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
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
	return f.GetSubImage(glctx, image.Rect(0, 0, f.Width, f.Height))
}

func (f *Framebuffer) GetSubImage(glctx gl.Context, region image.Rectangle) image.Image {

	glctx.BindFramebuffer(gl.FRAMEBUFFER, f.Framebuffer)
	dst := make([]uint8, int(region.Dx() * region.Dy() * 4))
	glctx.ReadPixels(dst, region.Min.X, region.Min.Y, region.Dx(), region.Dy(), gl.RGBA, gl.UNSIGNED_BYTE)

	return &extimage.BGRA{
		Pix: dst,
		Stride: region.Dx() * 4,
		Rect: image.Rect(0, 0, region.Dx(), region.Dy()),
	}
}

type TexturedQuad struct {
	ctx gl.Context
	fbo *Framebuffer
	dc *Drawcall

	width, height int
}

func roundToPower2(x int) int {
	x2 := 1
	for x2 < x {
		x2 *= 2
	}
	return x2
}

func NewTexturedQuad(glctx gl.Context, width, height int) *TexturedQuad {
	var err error
	quad := &TexturedQuad{ width: width, height: height, ctx: glctx, }


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

	quad.dc = NewDrawcall(glctx, program)
	quad.dc.SetViewport(image.Rect(0, 0, width, height))

	quad.fbo = NewFramebuffer(glctx, width * 2, height, gl.RGBA)

	quad.dc.SetFBO(quad.fbo.Framebuffer)

	vData := f32.Bytes(binary.LittleEndian,
		-1.0,  1.0, 0.0, 1.0,
		-1.0, -1.0, 0.0, 0.0,
		 1.0,  1.0, 1.0, 1.0,
		 1.0, -1.0, 1.0, 0.0,
	)
	quad.dc.SetVertexData(vData)
	quad.dc.SetIndices([]uint16{0, 1, 2, 3})
	quad.dc.SetAttribute("position", 2, 4, 0)
	quad.dc.SetAttribute("tc", 2, 4, 2)

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

	quad.dc.SetTexture("tex", tex)

	return quad
}

func (q *TexturedQuad) Draw() {
	q.dc.Draw()
	q.dc.ctx.Finish()
}

func (q *TexturedQuad) GetImage() image.Image {
	return q.fbo.GetImage(q.ctx)
}
