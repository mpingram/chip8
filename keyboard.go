package main

import (
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

	// copied from cpu; this is where it should go tho
	// META CONTROL LOOP
	// ===
	// read and execute control key inputs
	/*
		switch c.input.Poll() {
		case PAUSE_KEY:
			c.Pause()
		case UNPAUSE_KEY:
			c.Start()
		case POWEROFF_KEY:
			c.shouldCloseFlag = true
		case STEP_KEY
			// only step forward a paused CPU to avoid a double-step
			if c.IsPaused() {
				c.Step()
			}
		case DUMPSTATE_KEY
			c.logger.Print(c)
		}
		// ===
	*/

	k := KeyState{}
	// Power Off key
	if input.window.GetKey(glfw.KeyEscape) == glfw.Press {
		k[0x10] = true
		input.window.SetShouldClose(true)
	}
	// Pause key
	if input.window.GetKey(glfw.KeyP) == glfw.Press {
		k[0x11] = true
	}
	// Unpause key
	if input.window.GetKey(glfw.KeyLeftBracket) == glfw.Press {
		k[0x12] = true
	}
	// Step key
	if input.window.GetKey(glfw.KeyRightBracket) == glfw.Press {
		k[0x13] = true
	}
	// Dump state key
	if input.window.GetKey(glfw.KeyO) == glfw.Press {
		k[0x14] = true
	}

	if input.window.GetKey(glfw.KeyQ) == glfw.Press {
		k[0x1] = true
	}
	if input.window.GetKey(glfw.KeyW) == glfw.Press {
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
