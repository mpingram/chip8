package cpu

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// A KeyCode is a number that represents a key on the Chip-8 hexadecimal keyboard.
// Only the numbers 0 through 16 (0x0 through 0xF) are valid KeyCodes.
// KeyCode 0 indicates 'No Keypress', ie that no key is currently pressed.
type KeyCode byte

// KeyNone indicates 'No Keypresss', ie that no key is currently pressed.
const (
	KeyNone KeyCode = 0x00
	Key1    KeyCode = 0x01
	Key2    KeyCode = 0x02
	Key3    KeyCode = 0x03
	Key4    KeyCode = 0x04
	Key5    KeyCode = 0x05
	Key6    KeyCode = 0x06
	Key7    KeyCode = 0x07
	Key8    KeyCode = 0x08
	Key9    KeyCode = 0x09
	KeyA    KeyCode = 0x0a
	KeyB    KeyCode = 0x0b
	KeyC    KeyCode = 0x0c
	KeyD    KeyCode = 0x0d
	KeyF    KeyCode = 0x0f
)

// Chip8 represents an emulated Chip-8 CPU. Not that the Chip-8 was ever a real physical
// computer with a CPU, but writing this emulator has taught me that sometimes it's fun to pretend.
//
// To run a program or a game on the Chip-8, first connect your keyboard (Chip8.ConnectKeyboard) and
// your speaker (Chip8.ConnectSpeaker). Unless you enjoy interacting with computers in a
// more philosophical way, you'll also want to set up some kind of screen to display the contents
// of the Chip-8's video memory. You can do this by using a video adapter that reads the video
// memory directly using Chip8.ReadVideoMemory(). One or two video adapters (OpenGL and [soon] WebGL)
// come out of the box in this repository -- or feel free to write your own!
//
// Lastly, go ahead run your program with Chip8.Run(programSource).
//
// If you do have that philosophical inclination and you want to poke around at the inner workings
// of the Chip8, there are a bunch of methods that give you far greater control over the fine-grained
// operation of the chip, letting you start and stop the CPU, inspect its state, and execute a single
// instruction at a time.
type Chip8 struct {
	// program counter
	pc uint16
	// address register
	i uint16
	// data registers
	v [16]byte
	// delay and sound timers.
	// Both delay and sound timers are registers that are decremented at 60hz once set.
	dt byte
	st byte

	// stack pointer
	sp     uint16
	memory [4096]byte

	Log    bytes.Buffer
	logger *log.Logger

	speed         int
	clock         *time.Ticker
	isStoppedFlag bool

	speaker  Speaker
	input    Keyboard
	videoOut chan<- [256]byte
}

// NewChip8 returns an initialized Chip8, ready to run
// any programs you can get your hands on.
// The Chip8 it returns will be in a stopped state and
// has no program loaded into memory -- call the
// Chip8.Run(program) method to run a program.
//
// It is probably confusing that NewChip8 asks for a keyboard
// and a speaker but no display device. This implementation
// of the Chip8 interpreter takes a very lazy approach to displaying
// the screen: it provides direct read-only access to its video memory.
// "If these kids want to see the screen, they can read the hex or get
// off my lawn", this implementation says.
func NewChip8(keyboard Keyboard, speaker Speaker, videoOut chan<- [256]byte) *Chip8 {
	c := new(Chip8)
	c.reset()
	c.input = keyboard
	c.speaker = speaker
	c.videoOut = videoOut
	return c
}

// Run loads a program into memory and executes it.
//
// This is the simplest way to run a program on the Chip8 CPU. Make sure that you
// have set up a display set up to read the Chip8's video memory, or else you'll
// see a black screen, just like if you forgot to plug in your TV in Real Life.
func (c *Chip8) Run(program []byte) error {
	c.reset()
	err := c.load(program)
	if err != nil {
		return err
	}
	c.Resume()
	return nil
}

// TODO add 'Compatibility mode'? for that one instruction that gets implemented in one
// of two ways

// The Keyboard inerface represents the Chip8 keyboard input.
// It exposes a single Poll() method, which returns
// the keycode of the key that is currently being pressed,
// or the KeyCode KeyNone indicating no key is being pressed.
//
// The Chip8 cpu polls the keyboard for keypresses on its own time.
// As I understand it, this mirrors how the COSMAC VIP (the original computer that the
// Chip-8 interpreter was written for) handled input -- by polling,
// instead of by interrupts. As I understand it! This is new territory for me.
// I didn't even know what an interrupt was until yesterday.
type Keyboard interface {
	Poll() KeyCode
}

// The Speaker interface represents the Chip8 speaker, which acts as a simple
// buzzer -- the Chip8 doesn't specify the frequency of the sound, only its
// duration. So it could totally be a fart noise if you want.
// Whatever the sound, the Chip8 CPU will call the StartSound() method when
// the Speaker should begin playing and will call the StopSound() method when
// the sound should end.
type Speaker interface {
	StartSound()
	StopSound()
}

// Chip8State represents a read-only snapshot of the internal state of the Chip-8 CPU and RAM.
//
// It copies the stack and video memory into their own struct fields, even though
// under the hood both the stack and the video memory live in the same 'Memory' byte array
// with everything else.
// This is for convenience, so you don't have to know what the offsets are to figure out
// where the screen and stack start. (If you were curious, the offsets are 0xEA0 for the stack
// and 0xF00 for the video memory.)
type Chip8State struct {
	PC            uint16
	I             uint16
	V             [16]byte
	DT            byte
	ST            byte
	Memory        [4096]byte
	Stack         []byte
	VideoMemory   []byte
	MemoryDiagram string
	Speed         int
}

// ReadVideoMemory returns an slice of 256 bytes that represent the 64x32px Chip8 screen
// in an unfriendly, error-prone, low-level, and semi-authentic way.
//
// Every group of 8 bytes in the array represents one 64px row of the screen.
// (Each bit represents a pixel; there are 8 bytes in a row, so that's 8*8 = 64 pixels.)
// There are 32 rows of these 8 bytes, so 32*8 = 256 bytes total.
//
// The memory is laid out such that each set of 8 bytes (totaling 64 bits/pixels)
// represents one screen row. There are 32 rows, so 32 * 8 = 256 bytes of video memory.
//
// The chip8 screen uses a top-left coordinate space: the first byte of video memory
// represents the 8 pixels on the top-left of the screen. Then the highest bit of that first byte
// represents the top-left pixel, or coordinate 0,0.
//
// If you're wondering, this is intended to be similar to the setup of the COSMAC VIP,
// the original computer that the first Chip-8 emulator was written for, although it
// wasn't emulating anything except itself back then, I guess.
// On the VIP, the video card had direct memory access to the section of the RAM that
// contains the 256b of video memory that represents the screen: 60 times a second,
// the video card read the video memory and converted it to electrical signals that the
// CRT TV it was connected to could display. I think! I've never even looked at any of
// these computers except online.
func (c *Chip8) refreshScreen() {
	var screen [256]byte
	copy(screen, c.memory[videoMemoryAddress:highestMemoryAddress]
	c.videoOut <- screen
}

// load takes a Chip8 program as input and loads the program into the Chip8 memory.
func (c *Chip8) load(program []byte) error {
	// load program into memory
	var programStartAddr = 0x200
	for i, b := range program {
		c.memory[programStartAddr+i] = b
	}
	return nil
}

// reset clears the Chip8 memory and resets the Chip8 cpu to its starting state.
// After the reset method is called, the Chip8 will be in a paused state and will have
// no program loaded.
//
// Any input or speaker connected to the Chip8 remains connected to the Chip8 instance
// after reset() is called.
func (c *Chip8) reset() {
	// set all properties of Chip8 struct to default values
	c.i = 0x00
	c.v = [16]byte{}
	c.dt = 0x00
	c.st = 0x00
	c.sp = stackAddress
	c.memory = [4096]byte{}

	c.speed = 60                               // number of instructions to execute per second
	c.clock = time.NewTicker(time.Second / 60) // Tick 60 times per second
	c.Log = bytes.Buffer{}

	// Chip8 begins life in stopped state.
	c.isStoppedFlag = true

	// instantiate Chip8 logger.
	c.logger = log.New(&c.Log, "chip8:", log.Ltime|log.Lmicroseconds)

	// set program counter to start of program memory
	c.pc = 0x200

	// set decimal digits in memory location
	loadFontSprites(&c.memory, 0x0)
}

// Resume puts the Chip8 back into a running state after the Chip8 has
// been halted (by calling -- you guessed it -- Halt()).
// If the Chip8 is in a running state, calls to Resume have no effect.
func (c *Chip8) Resume() {
	// Only begin the CPU loop if Chip8 CPU is currently stopped.
	if !c.IsRunning() {
		c.isStoppedFlag = false
		// While the Chip8 is in 'running' state,
		// Run the CPU loop. Exit the loop once
		// the Chip8 exits running state.
		for c.IsRunning() {
			// wait for the clock to tick
			<-c.clock.C
			// decode and execute the next instruction
			c.cycle()
			// TODO CONSIDER add 'err' and/or 'finished' here,
			// to make sequence of control simpler.
		}
	}
}

// Halt pauses a running Chip8 CPU after the currently executing instruction finishes.
// To resume a stopped Chip8, call its Resume() method.
// While the CPU is in a stopped state, further calls to Stop have no effect.
func (c *Chip8) Halt() {
	if c.IsRunning() {
		c.isStoppedFlag = true
	}
}

// IsRunning returns true if the Chip8 CPU is in a running state
// and false if the Chip8 CPU is in a halted state.
func (c *Chip8) IsRunning() bool {
	return !c.isStoppedFlag
}

// Step executes the next instruction in its entirety and then pauses the Chip8 CPU.
//
// The behavior of Step changes slightly depending on the state of the Chip8 CPU.
// If the Chip8 CPU is currently running, Step lets the CPU finish executing the instruction
// it is currently processing and then pauses the CPU. Then Step executes the next program
// instruction and pauses the Chip8 CPU.
// If the Chip8 CPU is currently paused, Step executes the next program instruction and pauses
// the Chip8 CPU.
func (c *Chip8) Step() {
	// stop CPU if currently running
	if c.IsRunning() {
		c.Halt()
	}
	// do one cycle
	c.cycle()
}

func (c *Chip8) cycle() {

	// decrement delay timer
	if c.dt > 0 {
		c.dt--
	}
	// decrement sound timer
	if c.st > 0 {
		c.st--
		// tell the speaker to stop playing if we reached
		// the end of the sound timer on this cycle.
		if c.st == 0 {
			c.speaker.StopSound()
		}
	}
	// if haven't reached end of program,
	// execute next instruction in program.
	opcode := c.readOpcode(c.pc)
	if opcode == eofInstruction {
		c.Halt()
	} else {
		// exec will handle incrementing and/or moving the program counter.
		c.exec(opcode)
	}
}

// Snapshot returns a static copy of the Chip8 CPU at the moment the method is called.
func (c *Chip8) Snapshot() Chip8State {
	return Chip8State{
		PC:            c.pc,
		I:             c.i,
		V:             c.v,
		DT:            c.dt,
		ST:            c.st,
		Stack:         c.memory[stackAddress:c.sp],
		Memory:        c.memory,
		MemoryDiagram: "FIXME:NotImplemented",
		VideoMemory:   c.memory[videoMemoryAddress:highestMemoryAddress],
		Speed:         c.speed}
}

const stackAddress uint16 = 0xEA0
const videoMemoryAddress uint16 = 0xF00
const highestMemoryAddress uint16 = 0xFFF
const eofInstruction = 0x0000

func loadFontSprites(memory *[4096]byte, startAddress int) {
	fontSpriteData := [16 * 5]byte{
		0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
		0x20, 0x60, 0x20, 0x20, 0x70, // 1
		0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
		0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
		0x90, 0x90, 0xF0, 0x10, 0x10, // 4
		0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
		0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
		0xF0, 0x10, 0x20, 0x40, 0x40, // 7
		0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
		0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
		0xF0, 0x90, 0xF0, 0x90, 0x90, // A
		0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
		0xF0, 0x80, 0x80, 0x80, 0xF0, // C
		0xE0, 0x90, 0x90, 0x90, 0xE0, // D
		0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
		0xF0, 0x80, 0xF0, 0x80, 0x80, // F
	}
	totalBytes := 16 * 5 // 16 glyphs, 5 bytes wide
	for i := startAddress; i < startAddress+totalBytes; i++ {
		offset := startAddress + i
		memory[offset] = fontSpriteData[i]
	}
}

// drawSprite draws the sprite to the specified coordinates on the screen.
//
// The x and y arguments are the sprite's target top-left screen coordinates.
// If x or y are outside the visible area of the screen, drawSprite wraps the
// coordinates around so they are safely in screen space.
//
// If pixels of the sprite are drawn outside the visible area of the screen,
// those pixels are wrapped around to the opposite edge.
//
// drawSprite returns true if the sprite was drawn on top of any other pixels
// already on the screen; otherwise it returns false.
func (c *Chip8) drawSprite(sprite []byte, x, y byte) bool {
	/*
	* I can't count how many times I've misunderstood this algorithm, so I've guzzled some coffee
	* and written out exactly how and why it works.
	*
	* LAYOUT OF VIDEO MEMORY
	* The video memory in this chip8 implementation is an array of 256 bytes,
	* in which each bit represents a pixel on the 64x32px screen.

	* The memory is laid out such that each set of 8 bytes (totaling 64 bits/pixels)
	* represents one screen row. There are 32 rows, so 32 * 8 = 256 bytes of video memory.
	*
	* The chip8 screen uses a top-left coordinate space: the first byte of video memory
	* represents the 8 pixels on the top-left of the screen. Then the highest bit of that first byte
	* represents the top-left pixel, or coordinate 0,0.
	*
	* HOW WE WRITE A SPRITE TO VIDEO MEMORY
	* Let's say we need to write a sprite that is 1 row tall to x=35 y=3. I'm going to
	* refer to the single row of the sprite as the 'sprite byte', because it is just one byte
	* wide and it rhymes.
	*
	* We need to write the sprite byte to the fourth screen row from bits 35 to 42 inclusive.
	* But we can only write whole bytes of video memory, not single bits. If the x coordinate
	* falls between two bytes, (as this one does) we will need to split our sprite byte into
	* two bytes that align with the video memory using this formula:
	* 	spriteLeftByte = spriteByte >> x%8 				(if x=35,spriteByte >> 3)
	* 	spriteRightByte = spriteByte << 8-(x%8) 		(if x=35, spriteByte << 5)
	*
	* This diagram might make more sense:
	*
	*			pixel 35      42
	*				   |       |
	*              |___01010|101_____|         		   		<- sprite byte from bits 35 to 42
	*              |00001010|10100000|        |  	   ||	<- spriteLeftByte and spriteRightByte
	*      00000000|00001111|00001111|00000000|00000000||	<- screen row
	*      00000000|00001101|10100111|00000000|00000000||	<- new screen row (XOR leftByte and rightByte with bytes 5 and 6)
	*     ---------+--------+--------+--------+--------||
	*     |   3    |    4   |    5   |    6   |    7   || 	<- bytes (0-indexed)
	*     |24    31|32    39|40	   47|48    55|56	 63||	<- pixels (0-indexed)
	*
	*
	* Now we just need to XOR spriteLeftByte and spriteRightByte with the correct bytes in video memory.
	* screenLeftByte is the (x//8) == 4th byte on this row, and screenRightByte is the
	* ([x//8] + 1) % 8== 5th byte on this screen row.
	* NOTE the modulo 8 wraps screenRightByte around to byte 0 on this row if the sprite overflowed past
	* the edge of the screen.
	*
	* So we know that screenLeftByte is the 4th byte and screenRightByte is the 5th byte on this screen row.
	* Now we need to figure out where in video memory this screen row is. Because each row is 8 bytes wide,
	* row y begins at offset (y*8) in video memory. Therefore, screenLeftByte is the (y*8)+5 == 29th byte and
	* screenRightByte is the 30th byte in video memory. We now XOR spriteLeftByte with screenLeftByte and
	* spriteRightByte with screenRightByte and we've successfully written the sprite to video memory.
	*
	* If is 0 or evenly divisible by 8 (if x%8 == 0), then the spriteByte is already byte-aligned
	* with the screen row and we have a simpler case: we don't need to modify spriteByte and we only need to
	* write one byte to video memory, which we'll call screenByte. As above, we know screenByte is the
	* (y*8) + (x//8) th byte in video memory. We can simply XOR spriteByte with screenByte and we're done.
	*
	* To write a sprite that is more than one pixel tall (what will they think up next??), we just repeat
	* this procedure once for each row, just increasing the value of y for each row.
	 */

	// Wrap the coordinates around so that they land inside screen space.
	var screenW, screenH byte
	screenW = 64
	screenH = 32
	x = x % screenW
	y = y % screenH

	// Write the sprite to video memory. If a sprite pixel is
	// written over an active screen pixel, turn that pixel off
	// (invert it) and set the 'occluded' flag to true.
	var occluded = false
	for i, spriteByte := range sprite {
		xOffset := uint16(x / 8)
		yOffset := uint16((y + byte(i)) * 8)
		if isByteAligned := x%8 == 0; isByteAligned {
			offset := videoMemoryAddress + yOffset + xOffset
			screenByte := c.memory[offset]
			// if spriteByte and screenByte have an active pixel in the same place,
			// spriteByte occluded an active pixel.
			occluded = spriteByte&screenByte != 0
			c.memory[offset] = spriteByte ^ screenByte

		} else {
			spriteLeftByte := spriteByte >> x % 8
			spriteRightByte := spriteByte << (8 - (x % 8))

			leftOffset := videoMemoryAddress + yOffset + xOffset
			rightOffset := videoMemoryAddress + yOffset + ((xOffset + 1) % 8)
			screenLeftByte := c.memory[leftOffset]
			screenRightByte := c.memory[rightOffset]
			// if spriteByte and screenByte have an active pixel in the same place,
			// spriteByte occluded an active pixel.
			occluded = spriteLeftByte&screenLeftByte != 0 ||
				spriteRightByte&screenRightByte != 0
			c.memory[leftOffset] = spriteLeftByte ^ screenLeftByte
			c.memory[rightOffset] = spriteRightByte ^ screenRightByte
		}
	}

	return occluded
}

func (c *Chip8) stackPush(addr uint16) {
	high := byte(addr >> 8)
	low := byte(addr & 0x00FF)
	c.memory[c.sp] = high
	c.memory[c.sp+1] = low
	c.sp += 2
}

func (c *Chip8) stackPop() uint16 {
	high := c.memory[c.sp]
	low := c.memory[c.sp+1]
	if c.sp > stackAddress {
		c.sp -= 2
	}
	return uint16(high)<<8 | uint16(low)
}

func (c *Chip8) exec(opcode uint16) {

	// key:
	// ------
	// nnn - low 12 bits of opcode
	// n - low 4 bits of opcode
	// x - low 4 bits of opcode's high byte
	// y - low 4 bits of opcode's low byte
	// kk - opcode's low byte

	switch first := (opcode & 0xf000) >> 12; first {

	case 0x0:
		switch opcode {
		// 00E0: CLS (clear)
		case 0x00e0:
			c.logger.Printf("%04x: CLS", opcode)
			// zero out all bytes in video memory
			for i := videoMemoryAddress; i <= highestMemoryAddress; i++ {
				c.memory[i] = 0x0
			}
			c.pc += 2

		// 00EE: RET (return)
		case 0x00ee:
			c.logger.Printf("%04x: RET", opcode)
			c.pc = c.stackPop()
			// we've gone back to the location of the original CALL instruction;
			// proceed past it to the next instruction.
			c.pc += 2

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}

	// 1nnn: JP (jump) addr
	case 0x1:
		addr := opcode & 0x0fff
		c.logger.Printf("%04x: JP %03x\n", opcode, addr)
		c.pc = addr

	// 2nnn: CALL addr
	case 0x2:
		addr := opcode & 0x0fff
		c.logger.Printf("%04x: CALL %03x\n", opcode, addr)
		c.stackPush(c.pc)
		c.pc = addr

	// 3xkk: SE Vx byte (skip if equal)
	case 0x3:
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		c.logger.Printf("%04x: SE V%x %02x\n", opcode, x, kk)
		if c.v[x] == byte(kk) {
			c.pc += 2
		}
		c.pc += 2

	// 4xkk: SNE Vx byte (skip if not equal)
	case 0x4:
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		c.logger.Printf("%04x: SNE V%x %02x\n", opcode, x, kk)
		if c.v[x] != byte(kk) {
			c.pc += 2
		}
		c.pc += 2

	// 5xy0: SE Vx Vy (skip if equal)
	case 0x5:
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		c.logger.Printf("%04x: SE V%x V%x\n", opcode, x, y)
		if c.v[x] == c.v[y] {
			c.pc += 2
		}
		c.pc += 2

	// 6xkk: LD Vx byte (load value to register)
	case 0x6:
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		c.logger.Printf("%04x: LD V%x %02x\n", opcode, x, kk)
		c.v[x] = byte(kk)
		c.pc += 2

	// 7xkk: ADD Vx byte (add value to register)
	case 0x7:
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		c.logger.Printf("%04x: ADD V%x %02x\n", opcode, x, kk)
		c.v[x] = c.v[x] + byte(kk)
		c.pc += 2

	case 0x8:
		switch last := opcode & 0x000f; last {

		// 8xy0: LD Vx Vy (clone register)
		case 0x0:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: LD V%x V%x\n", opcode, x, y)
			c.v[x] = c.v[y]
			c.pc += 2

		// 8xy1: OR Vx Vy (or Vx Vy, assign result to Vx)
		case 0x1:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: OR V%x V%x\n", opcode, x, y)
			c.v[x] = c.v[x] | c.v[y]
			c.pc += 2

		// 8xy2: AND Vx Vy (and Vx Vy, assign result to Vx)
		case 0x2:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: OR V%x V%x\n", opcode, x, y)
			c.v[x] = c.v[x] & c.v[y]
			c.pc += 2

		// 8xy3: XOR Vx Vy (or Vx Vy, assign result to Vx)
		case 0x3:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: XOR V%x V%x\n", opcode, x, y)
			c.v[x] = c.v[x] ^ c.v[y]
			c.pc += 2

		// 8xy4: ADD Vx Vy (add Vx Vy, assign result to Vx, set Vf if carry)
		case 0x4:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: ADD V%x V%x\n", opcode, x, y)
			if (x + y) > 255 {
				c.v[0xf] = 1
			}
			c.v[x] = c.v[x] + c.v[y]
			c.pc += 2

		// 8xy5: SUB Vx Vy (sub Vx Vy, assign result to Vx)
		case 0x5:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: SUB V%x V%x\n", opcode, x, y)
			c.v[x] = c.v[x] - c.v[y]
			c.pc += 2

		// 8xy6: SHR Vx Vy (set VF=1 if the lowest bit of Vx is 1 otherwise set VF=0, then right shift Vx by 1)
		case 0x6:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: SHR V%x V%x\n", opcode, x, y)
			c.v[0xf] = c.v[x] & 0x01
			c.v[x] = c.v[x] >> 1
			c.pc += 2

		// 8xy7: SUBN Vx Vy (set VF=1 if Vy > Vx otherwise set VF=0, sub Vx Vy, assign result to Vx)
		case 0x7:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: SUBN V%x V%x\n", opcode, x, y)
			if c.v[y] > c.v[x] {
				c.v[0xf] = 1
			} else {
				c.v[0xf] = 0
			}
			c.v[x] = c.v[x] - c.v[y]
			c.pc += 2

		// 8xyE: SHL Vx Vy (set VF=1 if the highest bit of Vx is 1 otherwise set VF=0, then left shift Vx by 1)
		case 0xE:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.logger.Printf("%04x: SHL V%x V%x\n", opcode, x, y)
			c.v[0xf] = c.v[x] & 0x80 // 128 in decimal, 1000 0000 in binary
			c.v[x] = c.v[x] << 1
			c.pc += 2

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}

	// 9xy0: SNE Vx Vy (skip next opcode if Vx != Vy)
	case 0x9:
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		c.logger.Printf("%04x: SNE V%x V%x\n", opcode, x, y)
		if c.v[x] != c.v[y] {
			c.pc += 2
		}
		c.pc += 2

	// Annn: LD I addr (set I=nnn)
	case 0xA:
		addr := opcode & 0x0fff
		c.logger.Printf("%04x: LD I %03x\n", opcode, addr)
		c.i = addr
		c.pc += 2

	// Bnnn: JP V0 addr (jump to address nnn + v0, set PC=nnn + v0)
	case 0xB:
		addr := opcode & 0x0fff
		c.logger.Printf("%04x: JP V0 %03x\n", opcode, addr)
		c.pc = addr + uint16(c.v[0])

	// Cxkk: RND Vx byte (Vx = random byte and kk)
	case 0xC:
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		c.logger.Printf("%04x: RND V%x %02x\n", opcode, x, kk)
		// Read is exported function from math/rand -- loads random bytes into passed array.
		rnd := byte(rand.Intn(256))
		c.v[x] = rnd & byte(kk)
		c.pc += 2

	// Dxyn: DRW Vx Vy n (display n-byte sprite located at I at coordinates Vx,Vy, set VF=collision [if sprite is drawn on top of any active pixels])
	case 0xD:
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		n := opcode & 0x000f
		c.logger.Printf("%04x: DRW V%x V%x %x\n", opcode, x, y, n)
		sprite := make([]byte, 0, 16)
		for i := c.i; i < c.i+n; i++ {
			sprite = append(sprite, c.memory[i])
		}
		occluded := c.drawSprite(sprite, c.v[x], c.v[y])
		if occluded {
			fmt.Printf("\n\n\n\nOccluded my duded\n\n\n\n")
			c.v[0xf] = 1
		} else {
			c.v[0xf] = 0
		}
		c.pc += 2

	case 0xE:
		switch lastTwo := opcode & 0x0ff; lastTwo {
		// Ex9E: SKP Vx (skip next instruction if key with the value of Vx is currently pressed)
		case 0x9E:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: SKP V%x\n", opcode, x)
			key := KeyCode(c.v[x])
			if c.input.Poll() == key {
				c.pc += 2
			}
			c.pc += 2

		// ExA1: SKNP Vx (skip next instruction if key with the value of Vx is currently not pressed)
		case 0xA1:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: SKNP V%x\n", opcode, x)
			key := KeyCode(c.v[x])
			if c.input.Poll() != key {
				c.pc += 2
			}
			c.pc += 2

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}
	case 0xF:
		switch lastTwo := opcode & 0x0ff; lastTwo {

		// Fx07: LD Vx DT (set Vx=DT)
		case 0x07:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: LD V%x DT\n", opcode, x)
			c.v[x] = c.dt
			c.pc += 2

		// Fx0A: LD Vx K (wait for key press, store value of key press in Vx)
		case 0x0a:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: LD V%x K\n", opcode, x)
			if key := c.input.Poll(); key != KeyNone {
				c.v[x] = byte(key)
				c.pc += 2
			}
			// if no key is pressed, do NOT advance the
			// program counter -- execute this same instruction next cycle.
			// This effectively halts the interpreter until a key is pressed.

		// Fx15: LD DT Vx (set DT=Vx)
		case 0x15:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: LD DT V%x\n", opcode, x)
			c.dt = c.v[x]
			c.pc += 2

		// Fx18: LD ST Vx (set ST=Vx)
		case 0x18:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: LD ST V%x\n", opcode, x)
			c.st = c.v[x]
			// tell the speaker to start making noise
			c.speaker.StartSound()
			c.pc += 2

		// Fx1E: ADD I Vx (set I=I+Vx)
		case 0x1E:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: ADD I V%x\n", opcode, x)
			c.i = c.i + uint16(c.v[x])
			c.pc += 2

		// Fx29: LD F Vx (set I=memory address of sprite corresponding to digit in Vx)
		case 0x29:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: LD F V%x\n", opcode, x)
			digit := c.v[x]
			// each sprite corresponds to one digit and is five bytes wide,
			// and digits are stored in increasing order. So the sprite for '5'
			// will start at five sets of bytes away from the starting address.
			fontSpritesStartAddress := 0x00
			spriteWidth := 5
			offset := digit * byte(spriteWidth)
			c.i = uint16(fontSpritesStartAddress) + uint16(offset)
			c.pc += 2

		// Fx33: LD B Vx (store binary converted decimal [BCD] representation of number in Vx in memory locations I(hundreds place), I+1(tens place), I+2(ones place)
		case 0x33:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: LD B V%x\n", opcode, x)
			// TODO implement
			c.pc += 2

		// Fx55: LD I Vx (store registers V0 through Vx in memory starting at I)
		case 0x55:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: LD I V%x\n", opcode, x)
			for i := uint16(0); i < x; i++ {
				c.memory[c.i+i] = c.v[i]
			}
			c.pc += 2

		// Fx65: LD Vx I (read values in memory starting at I into registers V0 through Vx)
		case 0x65:
			x := opcode & 0x0f00 >> 8
			c.logger.Printf("%04x: LD V%x I\n", opcode, x)
			for i := uint16(0); i < x; i++ {
				c.v[i] = c.memory[c.i+i]
			}
			c.pc += 2

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}

	default:
		panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
	}

}

func (c *Chip8) readOpcode(addr uint16) uint16 {
	// the opcode we want to read is the next two bytes,
	// stored big-endian.
	high := c.memory[addr]
	low := c.memory[addr+1]
	// combine bytes as one uint16,
	// keeping the big-endian representation
	opcode := (uint16(high) << 8) | uint16(low)
	return opcode
}
