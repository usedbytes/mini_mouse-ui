package main

import (
	"bytes"
	"encoding/binary"
	"image"

	"golang.org/x/mobile/gl"
)

type Attribute struct {
	gl.Attrib

	Size, Stride, Offset int
	Type gl.Enum
	Normalized bool
}

type Drawcall struct {
	ctx gl.Context
	Program gl.Program
	Textures map[string]gl.Texture
	Buffers map[gl.Enum]gl.Buffer
	Uniforms map[string]gl.Uniform
	Attributes map[string]Attribute
	Viewport image.Rectangle
	nIndices int

	FBO gl.Framebuffer
}

func NewDrawcall(ctx gl.Context, shader gl.Program) *Drawcall {
	dc := &Drawcall{
		ctx: ctx,
		Program: shader,
		Textures: make(map[string]gl.Texture),
		Buffers: make(map[gl.Enum]gl.Buffer),
		Uniforms: make(map[string]gl.Uniform),
		Attributes: make(map[string]Attribute),
	}

	dc.Buffers[gl.ARRAY_BUFFER] = ctx.CreateBuffer()
	dc.Buffers[gl.ELEMENT_ARRAY_BUFFER] = ctx.CreateBuffer()

	return dc
}

func (dc *Drawcall) SetAttribute(name string, size, stride, offset int) {
	dc.ctx.UseProgram(dc.Program)

	a, ok := dc.Attributes[name]
	if !ok {
		a = Attribute{ }
		a.Attrib = dc.ctx.GetAttribLocation(dc.Program, name)
	}
	a.Size = size
	a.Stride = stride
	a.Offset = offset
	a.Type = gl.FLOAT

	dc.Attributes[name] = a
}

func (dc *Drawcall) setBufferData(target gl.Enum, data []byte) {
	dc.ctx.UseProgram(dc.Program)
	dc.ctx.BindBuffer(target, dc.Buffers[target])
	dc.ctx.BufferData(target, data, gl.STATIC_DRAW)
}

func (dc *Drawcall) SetVertexData(data []byte) {
	dc.setBufferData(gl.ARRAY_BUFFER, data)
}

func (dc *Drawcall) SetIndices(data []uint16) {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, data)
	dc.setBufferData(gl.ELEMENT_ARRAY_BUFFER, buf.Bytes())
	dc.nIndices = len(data)
}

func (dc *Drawcall) SetTexture(name string, tex gl.Texture) {
	dc.ctx.UseProgram(dc.Program)

	_, ok := dc.Uniforms[name]
	if !ok {
		dc.Uniforms[name] = dc.ctx.GetUniformLocation(dc.Program, name)
	}
	dc.Textures[name] = tex
}

func (dc *Drawcall) lookupUniform(name string) gl.Uniform {
	dc.ctx.UseProgram(dc.Program)

	u, ok := dc.Uniforms[name]
	if !ok {
		u = dc.ctx.GetUniformLocation(dc.Program, name)
		dc.Uniforms[name] = u
	}

	return u
}

func (dc *Drawcall) SetUniformi(name string, data []int32) {
	dc.ctx.UseProgram(dc.Program)

	u := dc.lookupUniform(name)

	switch len(data) {
		case 1:
			dc.ctx.Uniform1iv(u, data)
		case 2:
			dc.ctx.Uniform2iv(u, data)
		case 3:
			dc.ctx.Uniform3iv(u, data)
		case 4:
			dc.ctx.Uniform4iv(u, data)
	}
}

func (dc *Drawcall) SetUniformf(name string, data []float32) {
	dc.ctx.UseProgram(dc.Program)

	u := dc.lookupUniform(name)

	switch len(data) {
		case 1:
			dc.ctx.Uniform1fv(u, data)
		case 2:
			dc.ctx.Uniform2fv(u, data)
		case 3:
			dc.ctx.Uniform3fv(u, data)
		case 4:
			dc.ctx.Uniform4fv(u, data)
	}
}

func (dc *Drawcall) SetFBO(fbo gl.Framebuffer) {
	dc.FBO = fbo
}

func (dc *Drawcall) SetViewport(vp image.Rectangle) {
	dc.Viewport = vp
}

func (dc *Drawcall) Draw() {
	dc.ctx.UseProgram(dc.Program)

	dc.ctx.BindFramebuffer(gl.FRAMEBUFFER, dc.FBO)

	dc.ctx.Viewport(dc.Viewport.Min.X, dc.Viewport.Min.Y, dc.Viewport.Dx(), dc.Viewport.Dy())

	i := 0
	for k, v := range dc.Textures {
		dc.ctx.Uniform1i(dc.Uniforms[k], i)
		dc.ctx.ActiveTexture(gl.Enum(gl.TEXTURE0 + i))
		dc.ctx.BindTexture(gl.TEXTURE_2D, v)
		i++
	}

	for k, v := range dc.Buffers {
		dc.ctx.BindBuffer(k, v)
	}

	for _, v := range dc.Attributes {
		dc.ctx.VertexAttribPointer(v.Attrib, v.Size, v.Type, v.Normalized, v.Stride, v.Offset)
		dc.ctx.EnableVertexAttribArray(v.Attrib)
	}

	dc.ctx.DrawElements(gl.TRIANGLE_STRIP, dc.nIndices, gl.UNSIGNED_SHORT, 0)

	for _, v := range dc.Attributes {
		dc.ctx.DisableVertexAttribArray(v.Attrib)
	}
}

