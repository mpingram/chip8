package main

import (
	"fmt"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"io/ioutil"
	"log"
	_ "os"
	"runtime"
	"strings"
)

func init() {
	// openGL requires this to render properly
	runtime.LockOSThread()
}

func main() {

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
	log.Println(version)

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
		// top-left of screen
		-1.0, 1.0, 0.0,
		// bottom-left
		-1.0, -1.0, 0.0,
		// bottom-right
		1.0, -1.0, 0.0,

		// top-right
		1.0, 1.0, 0.0,
		// top-left
		-1.0, 1.0, 0.0,
		// bottom-right
		1.0, -1.0, 0.0,
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
		0,        // configure the 0th vertex attribute (in our case, 'location')
		3,        // each vertex attribute is made of three components (in our case, xyz coordinates)
		gl.FLOAT, // each component is a 32bit float
		false,    // there are no delimiters between each ser of components in the array (array is tightly packed)
		3*4,      // the span of bytes of one vertex attribute is 3 float32s, each float32 is 4 bytes
		nil,      // the offset of the first vertex attribute in the array is zero. For some reason, this requires a void pointer cast, represented in go-gl as nil.
	)
	gl.EnableVertexAttribArray(0)

	// =====================================

	//romPath := "./roms/Pong (1 player).ch8"
	//rom, err := ioutil.ReadFile(romPath)
	//if err != nil {
	//		panic(err)
	//	}

	//c8 := new(Chip8)
	//c8.Load(rom)
	for !window.ShouldClose() {

		//c8.Step()

		gl.ClearColor(0.1, 0.2, 0.1, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// use our shader program from earlier
		gl.UseProgram(shaderProgram)
		// bind the vertex array object we configured earlier
		// containing our screen-sized rectangle
		gl.BindVertexArray(vao)
		// draw the vertices in our vertex array object as triangles
		numTriangles := int32(len(vertices) / 3)
		gl.DrawArrays(gl.TRIANGLES, 0, numTriangles)

		// render screen
		window.SwapBuffers()
		// -------------------------------

		// 'handle' errors
		if err := gl.GetError(); err != gl.NO_ERROR {
			panic(err)
		}
	}

	//defer c8.Log.WriteTo(os.Stdout)
}

func compileShader(sourceBytes []byte, shaderType uint32) (uint32, error) {
	sourceStr := string(sourceBytes)
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
		w.SetShouldClose(true)
	}
	if w.GetKey(glfw.KeyQ) == glfw.Press {
		c.KeyDown(0x4)
	}
}
