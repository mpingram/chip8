package main

import (
	"io/ioutil"
	"os"
	"runtime"
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

	c8 := NewChip8()
	c8.load(rom)
	for window.ShouldClose() {
		c8.Step()
		render(c8.Screen)
		// if key is pressed c8.KeyDown(key)
	}
}

type ControllerState struct {
	key1 bool
	key2 bool
	key3 bool
	key4 bool
	key5 bool
	key6 bool
	key7 bool
	key8 bool
	key9 bool
	keyA bool
	keyB bool
	keyC bool
	keyD bool
	keyE bool
	keyF bool
}

type Chip8Display interface {
	Render(screen [32]uint64)
}

type Chip8OpenGLDisplay struct {
}

func (a *Chip8OpenGLDisplay) Render(screen [32]uint64) {
	// convert screen to texture format
	// update current texture
	// re-render
	// poll inputs
}

func (a *Chip8OpenGLAdapter) done() {

}
