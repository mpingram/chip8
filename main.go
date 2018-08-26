package main

import (
	"fmt"
	"github.com/go-gl/gl/v3.2-compatibility/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
)

func init() {
	// openGL requires this to render properly
	runtime.LockOSThread()
}

func main() {

	romPath := "./roms/Pong (1 player).ch8"
	rom, err := ioutil.ReadFile(romPath)
	if err != nil {
		panic(err)
	}

	c8 := new(Chip8)
	c8.Load(rom)

	vertexShaderSource, err := ioutil.ReadFile("./c8.vert.glsl")
	if err != nil {
		panic(err)
	}
	fragShaderSource, err := ioutil.ReadFile("./c8.frag.glsl")
	if err != nil {
		panic(err)
	}

	err = glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	window, err := glfw.CreateWindow(640, 320, "Chip-8", nil, nil)
	if err != nil {
		panic(err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window.MakeContextCurrent()

	// opengl initialization
	// ====================================
	err = gl.Init()
	if err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Printf("OpenGL version: %s\n", version)

	gl.Viewport(0, 0, 640, 320)

	// link vertex and fragment shaders into shader program
	// and use it for rendering
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragShader, err := compileShader(fragShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}
	shaderProgram := gl.CreateProgram()
	gl.AttachShader(shaderProgram, vertexShader)
	gl.AttachShader(shaderProgram, fragShader)
	gl.LinkProgram(shaderProgram)
	// check for linking errors
	var status int32
	gl.GetProgramiv(shaderProgram, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		logLength := int32(512)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(shaderProgram, logLength, nil, gl.Str(log))
		panic(log)
	}

	// free our shaders once we've linked them into a shader program
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragShader)

	// intialize a vertex buffer object -- this will hold all of the vertices
	// we're rendering.
	var vbo uint32
	gl.GenBuffers(1, &vbo)

	// initialize a vertex array object -- this is a convenience object that
	// stores information about how the currently bound vertex buffer object is
	// configured. The information that the vertex array object stores is the
	// info about the shape of the VBO that we set in gl.VertexAttribPointer below.
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	// set our new vertex array object as the active VAO
	gl.BindVertexArray(vao)

	// set our new vertex buffer object as the active VBO; now things we configure
	// in gl.VertexAttribPointer will be stored to the bound VAO for later use.
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

	// create our vertices. These will be a simple rect (2 triangles)
	// covering the 2 screens -- a canvas for us to render our screen 'texture'
	// to.
	vertices := []float32{
		// x 	y		z  		s   t
		// -------------------------
		-1.0, 1.0, 0.0, -1.0, 1.0, // top-left
		-1.0, -1.0, 0.0, -1.0, -1.0, // bottom-left
		1.0, -1.0, 0.0, 1.0, -1.0, // bottom-right
		1.0, 1.0, 0.0, 1.0, 1.0, // top-right
		-1.0, 1.0, 0.0, -1.0, 1.0, // top-left
		1.0, -1.0, 0.0, 1.0, -1.0, // bottom-right
	}

	// load our vertices into our vertex buffer object
	gl.BufferData(
		gl.ARRAY_BUFFER,  // load into the current array buffer
		4*len(vertices),  // total number of bytes in the array to be loaded (each float32 is 4 bytes wide)
		gl.Ptr(vertices), // openGL pointer to the array of vertices
		gl.STATIC_DRAW,   // hint to openGL that we won't be changing these vertices often at all
	)
	// tell openGL about the shape of our vertex buffer
	gl.VertexAttribPointer(
		0,        // configure the vertex attribute with id 0 (location)
		3,        // each vertex attribute is made of three components (in this case, xyz coordinates)
		gl.FLOAT, // each component is a 32bit float
		false,    // there are no delimiters between each ser of components in the array (array is tightly packed)
		5*4,      // the span of bytes of one vertex attribute is 5 float32s (3 for location attrib, 2 for texel coordinate attrib). Each float32 is 4 bytes.
		nil,      // the offset of the first vertex attribute in the array is zero. For some reason, this requires a void pointer cast, represented in go-gl as nil.
	)
	gl.VertexAttribPointer(
		1,                 // configure the vertex attribute with id 1 (texture coordinates)
		2,                 // each vertex attribute is made of two components (in this case, st texture coordinates)
		gl.FLOAT,          // each component is a 32bit float
		false,             // there are no delimiters between each ser of components in the array (array is tightly packed)
		5*4,               // the span of bytes of one vertex attribute is 5 float32s (3 for location attrib, 2 for texel coordinate attrib). Each float32 is 4 bytes.
		gl.PtrOffset(3*4), // the offset of the first vertex attribute in the array is zero. For some reason, this requires a void pointer cast, represented in go-gl as nil.
	)
	gl.EnableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)

	// create our screen texture
	var texture1 uint32
	gl.GenTextures(1, &texture1)
	gl.BindTexture(gl.TEXTURE_2D, texture1)
	// clamp texture to border: do not render texels outside of texture coordinate area
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	// use nearest neighbor texture filtering when zooming up; it's pixel graphics, let's keep it blocky
	// when zooming down, use bilinear filtering
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	// create texture data from initial Chip8 screen
	texData := toTextureData(c8.Screen)
	texWidth := int32(64)
	texHeight := int32(32)
	gl.TexImage2D(
		gl.TEXTURE_2D, // target the 2D texture
		0,             // mipmap level 0
		gl.R8,         // internal texture format; a subtype of the texture format parameter above. gl.R8 is A single 'red' channel represented by one byte.
		texWidth,
		texHeight,
		0,                // fun fact, this parameter apparently is 'border', and it must always be 0 or else!
		gl.RED,           // texture format; gl.RED is a single red channel.
		gl.UNSIGNED_BYTE, // type; gl.R8 requires an unsigned byte. Some other internal texture formats can have different types, that's why this is here.
		gl.Ptr(texData),  // last but not least, the pixel data of the texture
	)

	// set our texture uniform in our shader to our texture (NOTE why 0 and not texture id?)
	gl.UseProgram(shaderProgram)
	texUniform := gl.GetUniformLocation(shaderProgram, gl.Str("texture1\000"))
	gl.Uniform1i(texUniform, 0)
	// =====================================

	for !window.ShouldClose() {

		c8.Step()

		gl.ClearColor(0.1, 0.2, 0.1, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// use our shader program from earlier
		gl.UseProgram(shaderProgram)
		// get the next frame's texture data
		texData = toTextureData(c8.Screen)
		// replace the current texture with new texture
		gl.TexSubImage2D(
			gl.TEXTURE_2D,
			0,                // mipmap level 0
			0,                // x offset
			0,                // y offset
			64,               // width
			32,               // height
			gl.RED,           // format
			gl.UNSIGNED_BYTE, // type,
			gl.Ptr(texData),  // data
		)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture1)

		// containing our screen-sized rectangle
		gl.BindVertexArray(vao)
		// draw the vertices in our vertex array object as triangles
		numTriangles := int32(len(vertices) / 3)
		gl.DrawArrays(gl.TRIANGLES, 0, numTriangles)

		glfw.PollEvents()
		processInput(window, c8)
		// render screen
		window.SwapBuffers()
		// -------------------------------

		// 'handle' errors
		if err := gl.GetError(); err != gl.NO_ERROR {
			panic(err)
		}
	}

	defer c8.Log.WriteTo(os.Stdout)
}

func toTextureData(screen [32][64]bool) []byte {
	texData := []byte{}
	for col, _ := range screen {
		for row, _ := range screen[col] {
			var pixel byte
			if screen[col][row] == true {
				pixel = 0xFF
			} else {
				pixel = 0x00
			}
			texData = append(texData, pixel)
		}
	}
	return texData
}

func compileShader(sourceBytes []byte, shaderType uint32) (uint32, error) {
	sourceStr := string(sourceBytes) + string('\000')
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(sourceStr)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile shader:\n%v\n%v", sourceStr, log)
	}

	return shader, nil
}

func processInput(w *glfw.Window, c *Chip8) {
	if w.GetKey(glfw.KeyEscape) == glfw.Press {
		fmt.Println("Esc")
		w.SetShouldClose(true)
	}
	if w.GetKey(glfw.KeyQ) == glfw.Press {
		c.KeyDown(0x4)
	}
}
