package main

import (
	"fmt"
	"github.com/go-gl/glfw/v3.2/glfw"
)

type GLFWKeyboardInput struct {
	window *glfw.Window
}

func NewGLFWKeyboardInput(window *glfw.Window) *GLFWKeyboardInput {
	return &GLFWKeyboardInput{window}
}

func (input *GLFWKeyboardInput) Poll() KeyState {
	if input.window == nil {
		panic("Poll() called before AttachWindow")
	}

	glfw.PollEvents()

	// FIXME keep?
	if input.window.GetKey(glfw.KeyEscape) == glfw.Press {
		input.window.SetShouldClose(true)
	}

	k := KeyState{}
	// populate keystate
	if input.window.GetKey(glfw.KeyQ) == glfw.Press {
		fmt.Println("Q")
		k[0x1] = true
	}
	if input.window.GetKey(glfw.KeyW) == glfw.Press {
		fmt.Println("W")
		k[0x2] = true
	}
	if input.window.GetKey(glfw.KeyE) == glfw.Press {
		k[0x3] = true
	}
	if input.window.GetKey(glfw.KeyR) == glfw.Press {
		k[0xc] = true
	}

	if input.window.GetKey(glfw.KeyA) == glfw.Press {
		k[0x4] = true
	}
	if input.window.GetKey(glfw.KeyS) == glfw.Press {
		k[0x5] = true
	}
	if input.window.GetKey(glfw.KeyD) == glfw.Press {
		k[0x6] = true
	}
	if input.window.GetKey(glfw.KeyF) == glfw.Press {
		k[0xD] = true
	}

	if input.window.GetKey(glfw.KeyZ) == glfw.Press {
		k[0x7] = true
	}
	if input.window.GetKey(glfw.KeyX) == glfw.Press {
		k[0x8] = true
	}
	if input.window.GetKey(glfw.KeyC) == glfw.Press {
		k[0x9] = true
	}
	if input.window.GetKey(glfw.KeyV) == glfw.Press {
		k[0xE] = true
	}

	if input.window.GetKey(glfw.Key1) == glfw.Press {
		k[0xA] = true
	}
	if input.window.GetKey(glfw.Key2) == glfw.Press {
		k[0x0] = true
	}
	if input.window.GetKey(glfw.Key3) == glfw.Press {
		k[0xB] = true
	}
	if input.window.GetKey(glfw.Key4) == glfw.Press {
		k[0xF] = true
	}

	return k
}
