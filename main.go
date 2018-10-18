package main

import (
	_ "fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/mpingram/chip8/cpu"
)

func init() {
	// openGL requires this to render properly
	runtime.LockOSThread()
}

func main() {

	err := glfw.Init()
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

	renderer := NewOpenGLRenderer(window)
	input := NewGLFWKeyboardInput(window)

	romPath := "./roms/Pong (1 player).ch8"
	rom, err := ioutil.ReadFile(romPath)
	if err != nil {
		panic(err)
	}

	c8 := new(cpu.Chip8)
	c8.AttachDisplay(renderer)
	c8.AttachInput(input)
	c8.Run(rom)

	defer c8.Log.WriteTo(os.Stdout)

}
