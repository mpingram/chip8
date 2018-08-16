package main

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"math/rand"
	"os"
)

func main() {
	var romPath string
	if len(os.Args) < 2 {
		//panic('Require path to Chip8 program ROM as first argument')
		// FIXME development only
		romPath = "./roms/Pong (1 player).ch8"
	} else {
		romPath = os.Args[1]
	}
	rom, err := ioutil.ReadFile(romPath)
	if err != nil {
		panic(err)
	}

	c8 := new(Chip8)
	c8.Run(rom)
}

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
	sp byte

	stack  []uint16
	memory [4096]byte

	// screen is 38x64 px; this 'clever hack' represents
	// each row as a 64-bit binary number. Let's see if I'll
	// regret this.
	screen [38]uint64

	// keyState stores state of keyboard as a bitfield.
	// Keys map to the number of their corresponding bit in descending order:
	// F E D C B A 9 8 7 6 5 4 3 2 1 0
	keyState uint16

	DEBUG_INSTRS []string
}

const NULL_OPCODE uint16 = 0x0000

func (c *Chip8) Run(program []byte) {

	// FIXME move elsewhere
	// -----------------
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)
	defer c.clearDisplay()
	// ------------------

	// load program into memory
	var programStartAddr int = 0x200
	for i, b := range program {
		c.memory[programStartAddr+i] = b
	}

	// set chip's program counter to start of program
	c.pc = uint16(programStartAddr)

	// run the program, i.e. iterate through the program's opcodes and execute them
	for opcode := c.readOpcode(c.pc); opcode != NULL_OPCODE; opcode = c.readOpcode(c.pc) {
		c.pc += 2
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("%04x\t", opcode))
		c.exec(opcode)
		c.refreshDisplay()
	}
}

func (c *Chip8) refreshDisplay() {
	// draw screen onto display
	for i := 0; i < 38; i++ {
		fmt.Printf("%064b\n\r", c.screen[i])
	}
	// return cursor to origin
	fmt.Print("\033[1;1H")
}

func (c *Chip8) clearDisplay() {
	fmt.Print("\033[2J")
}

/**
* drawSprite draws the sprite to the specified coordinates (top-left origin)
* It returns true if the sprite occluded any other pixels already on the screen,
* otherwise false.
 */
func (c *Chip8) drawSprite(sprite []byte, x, y uint16) bool {
	var occluded bool = false
	for i := int(x); i <= int(x)+len(sprite); i++ {
		row := c.screen[i]
		// shift sprite into position on screen by
		// moving it at most 56 pixels left (if y=0),
		// so that the sprite is the top byte of
		// the int64 screen row.
		spriteRow := uint64(sprite[i]) << uint64(56-y)
		if row&spriteRow != 0 {
			occluded = true
		}
		c.screen[i] = row ^ spriteRow
	}
	return occluded
}

func (c *Chip8) keydown(key byte) {
	// take only the bottom four bits
	// (NOTE is this equivalent to modulo 16?)
	k := key & 0x0f
	c.keyState = c.keyState | (1 << k)
}

func (c *Chip8) keyup(key byte) {
	k := key & 0x0f
	c.keyState = c.keyState & ^(1 << k)
}

func (c *Chip8) keyIsPressed(key byte) bool {
	k := key & 0x0f
	if c.keyState&(1<<k) != 0 {
		return true
	} else {
		return false
	}
}

func (c *Chip8) clearScr() {
	for i, _ := range c.screen {
		c.screen[i] = 0
	}
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
			c.clearScr()
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintln("CLS"))

		// 00EE: RET (return)
		case 0x00ee:
			c.pc = c.stack[c.sp]
			c.sp -= 1
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintln("RET"))

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}

	case 0x1:
		// 1nnn: JP (jump) addr
		addr := opcode & 0x0fff
		c.pc = addr
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("JP %03x\n", addr))

	case 0x2:
		// 2nnn: CALL addr
		addr := opcode & 0x0fff
		c.stack = append(c.stack, c.pc)
		c.sp += 1
		c.pc = addr
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("CALL %03x\n", addr))

	case 0x3:
		// 3xkk: SE Vx byte (skip if equal)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		if c.v[x] == byte(kk) {
			c.pc += 2
		}
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SE V%x %02x\n", x, kk))

	case 0x4:
		// 4xkk: SNE Vx byte (skip if not equal)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		if c.v[x] != byte(kk) {
			c.pc += 2
		}
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SNE V%x %02x\n", x, kk))

	case 0x5:
		// 5xy0: SE Vx Vy (skip if equal)
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		if c.v[x] == c.v[y] {
			c.pc += 2
		}
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SE V%x V%x\n", x, y))

	case 0x6:
		// 6xkk: LD Vx byte (load value to register)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		c.v[x] = byte(kk)
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD V%x %02x\n", x, kk))

	case 0x7:
		// 7xkk: ADD Vx byte (add value to register)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		c.v[x] = c.v[x] + byte(kk)
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("ADD V%x %02x\n", x, kk))

	case 0x8:
		switch last := opcode & 0x000f; last {

		// 8xy0: LD Vx Vy (clone register)
		case 0x0:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.v[x] = c.v[y]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD V%x V%x\n", x, y))

		// 8xy1: OR Vx Vy (or Vx Vy, assign result to Vx)
		case 0x1:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.v[x] = c.v[x] | c.v[y]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("OR V%x V%x\n", x, y))

		// 8xy2: AND Vx Vy (and Vx Vy, assign result to Vx)
		case 0x2:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.v[x] = c.v[x] & c.v[y]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("OR V%x V%x\n", x, y))

		// 8xy3: XOR Vx Vy (or Vx Vy, assign result to Vx)
		case 0x3:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.v[x] = c.v[x] ^ c.v[y]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("XOR V%x V%x\n", x, y))

		// 8xy4: ADD Vx Vy (add Vx Vy, assign result to Vx)
		case 0x4:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.v[x] = c.v[x] + c.v[y]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("ADD V%x V%x\n", x, y))

		// 8xy5: SUB Vx Vy (sub Vx Vy, assign result to Vx)
		case 0x5:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.v[x] = c.v[x] - c.v[y]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SUB V%x V%x\n", x, y))

		// 8xy6: SHR Vx Vy (set VF=1 if the lowest bit of Vx is 1 otherwise set VF=0, then right shift Vx by 1)
		case 0x6:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.v[0xf] = c.v[x] & 0x01
			c.v[x] = c.v[x] >> 1
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SHR V%x V%x\n", x, y))

		// 8xy7: SUBN Vx Vy (set VF=1 if Vy > Vx otherwise set VF=0, sub Vx Vy, assign result to Vx)
		case 0x7:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			if c.v[y] > c.v[x] {
				c.v[0xf] = 1
			} else {
				c.v[0xf] = 0
			}
			c.v[x] = c.v[x] - c.v[y]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SUBN V%x V%x\n", x, y))

		// 8xyE: SHL Vx Vy (set VF=1 if the highest bit of Vx is 1 otherwise set VF=0, then left shift Vx by 1)
		case 0xE:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			c.v[0xf] = c.v[x] & 0x80 // 128 in decimal, 1000 0000 in binary
			c.v[x] = c.v[x] << 1
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SHL V%x V%x\n", x, y))

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}

	case 0x9:
		// 9xy0: SNE Vx Vy (skip next opcode if Vx != Vy)
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		if c.v[x] != c.v[y] {
			c.pc += 2
		}
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SNE V%x V%x\n", x, y))

	case 0xA:
		// Annn: LD I addr (set I=nnn)
		addr := opcode & 0x0fff
		c.i = addr
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD I %03x\n", addr))

	case 0xB:
		// Bnnn: JP V0 addr (jump to address nnn + v0, set PC=nnn + v0)
		addr := opcode & 0x0fff
		c.pc = addr + uint16(c.v[0])
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("JP V0 %03x\n", addr))

	case 0xC:
		// Cxkk: RND Vx byte (Vx = random byte and kk)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		// Read is exported function from math/rand -- loads random bytes into passed array.
		rnd := byte(rand.Intn(256))
		c.v[x] = rnd & byte(kk)
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("RND V%x %02x\n", x, kk))

	case 0xD:
		// Dxyn: DRW Vx Vy n (display n-byte sprite located at I at coordinates Vx,Vy, set VF=collision [if sprite is drawn on top of any active pixels])
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		n := opcode & 0x000f
		sprite := make([]byte, 16, 16)
		for i := c.i; i < c.i+n; i++ {
			sprite = append(sprite, c.memory[i])
		}
		occluded := c.drawSprite(sprite, x, y)
		if occluded {
			c.v[0xf] = 1
		} else {
			c.v[0xf] = 0
		}
		c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("DRW V%x V%x %x\n", x, y, n))

	case 0xE:
		switch lastTwo := opcode & 0x0ff; lastTwo {
		// Ex9E: SKP Vx (skip next instruction if key with the value of Vx is currently pressed)
		case 0x9E:
			x := opcode & 0x0f00 >> 8
			if c.keyIsPressed(c.v[x]) {
				c.pc += 2
			}
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SKP V%x\n", x))

		// ExA1: SKNP Vx (skip next instruction if key with the value of Vx is currently not pressed)
		case 0xA1:
			x := opcode & 0x0f00 >> 8
			if !c.keyIsPressed(c.v[x]) {
				c.pc += 2
			}
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("SKNP V%x\n", x))

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}
	case 0xF:
		switch lastTwo := opcode & 0x0ff; lastTwo {

		// Fx07: LD Vx DT (set Vx=DT)
		case 0x07:
			x := opcode & 0x0f00 >> 8
			c.v[x] = c.dt
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD V%x DT\n", x))

		// Fx0A: LD Vx K (wait for key press, store value of key press in Vx)
		case 0x0a:
			x := opcode & 0x0f00 >> 8
			// TODO implement
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD V%x K\n", x))

		// Fx15: LD DT Vx (set DT=Vx)
		case 0x15:
			x := opcode & 0x0f00 >> 8
			c.dt = c.v[x]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD DT V%x\n", x))

		// Fx18: LD ST Vx (set ST=Vx)
		case 0x18:
			x := opcode & 0x0f00 >> 8
			c.st = c.v[x]
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD ST V%x\n", x))

		// Fx1E: ADD I Vx (set I=I+Vx)
		case 0x1E:
			x := opcode & 0x0f00 >> 8
			c.i = c.i + uint16(c.v[x])
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("ADD I V%x\n", x))

		// Fx29: LD F Vx (set I=memory address of sprite corresponding to digit in Vx)
		case 0x29:
			x := opcode & 0x0f00 >> 8
			// TODO implement
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD F V%x\n", x))

		// Fx33: LD B Vx (store binary converted decimal [BCD] representation of number in Vx in memory locations I(hundreds place), I+1(tens place), I+2(ones place)
		case 0x33:
			x := opcode & 0x0f00 >> 8
			// TODO implement
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD B V%x\n", x))

		// Fx55: LD I Vx (store registers V0 through Vx in memory starting at I)
		case 0x55:
			x := opcode & 0x0f00 >> 8
			for i := uint16(0); i < x; i++ {
				c.memory[c.i+i] = c.v[i]
			}
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD I V%x\n", x))

		// Fx65: LD Vx I (read values in memory starting at I into registers V0 through Vx)
		case 0x65:
			x := opcode & 0x0f00 >> 8
			for i := uint16(0); i < x; i++ {
				c.v[i] = c.memory[c.i+i]
			}
			c.DEBUG_INSTRS = append(c.DEBUG_INSTRS, fmt.Sprintf("LD V%x I\n", x))

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
	low := c.memory[addr]
	// combine bytes as one uint16,
	// keeping the big-endian representation
	opcode := (uint16(high) << 8) | uint16(low)
	return opcode
}
