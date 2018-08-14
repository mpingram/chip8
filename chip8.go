package main

import (
	"fmt"
	"io/ioutil"
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
	// address register
	i uint16
	// data registers
	v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, vA, vB, vC, vD, vF byte
	// delay and sound timers.
	// Both delay and sound timers are registers that are decremented at 60hz once set.
	dt byte
	st byte
	// program counter
	pc uint16
	// stack pointer
	sp byte

	stack  [16]uint16
	memory [4096]byte
}

const NULL_OPCODE uint16 = 0x0000

func (c *Chip8) Run(program []byte) {

	// load program into memory
	var programStartAddr int = 0x200
	for i, b := range program {
		c.memory[programStartAddr+i] = b
	}

	// set chip's address register to start of program
	c.i = uint16(programStartAddr)

	// run the program, i.e. iterate through the program's opcodes and execute them
	for opcode := c.nextOpcode(); opcode != NULL_OPCODE; opcode = c.nextOpcode() {
		c.pc += 1
		fmt.Printf("%04x\t", opcode)
		c.exec(opcode)
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
			fmt.Println("CLS")

		// 00EE: RET (return)
		case 0x00ee:
			fmt.Println("RET")

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}

	case 0x1:
		// 1nnn: JP (jump) addr
		addr := opcode & 0x0fff
		fmt.Printf("JP %03x\n", addr)

	case 0x2:
		// 2nnn: CALL addr
		addr := opcode & 0x0fff
		fmt.Printf("CALL %03x\n", addr)

	case 0x3:
		// 3xkk: SE Vx byte (skip if equal)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		fmt.Printf("SE V%x %02x\n", x, kk)

	case 0x4:
		// 4xkk: SNE Vx byte (skip if not equal)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		fmt.Printf("SNE V%x %02x\n", x, kk)

	case 0x5:
		// 5xy0: SE Vx Vy (skip if equal)
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		fmt.Printf("SE V%x V%x\n", x, y)

	case 0x6:
		// 6xkk: LD Vx byte (load value to register)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		fmt.Printf("LD V%x %02x\n", x, kk)

	case 0x7:
		// 7xkk: ADD Vx byte (add value to register)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		fmt.Printf("ADD V%x %02x\n", x, kk)

	case 0x8:
		switch last := opcode & 0x000f; last {

		// 8xy0: LD Vx Vy (clone register)
		case 0x0:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("LD V%x V%x\n", x, y)

		// 8xy1: OR Vx Vy (or Vx Vy, assign result to Vx)
		case 0x1:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("OR V%x V%x\n", x, y)

		// 8xy2: AND Vx Vy (and Vx Vy, assign result to Vx)
		case 0x2:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("OR V%x V%x\n", x, y)

		// 8xy3: XOR Vx Vy (or Vx Vy, assign result to Vx)
		case 0x3:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("XOR V%x V%x\n", x, y)

		// 8xy4: ADD Vx Vy (add Vx Vy, assign result to Vx)
		case 0x4:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("ADD V%x V%x\n", x, y)

		// 8xy5: SUB Vx Vy (sub Vx Vy, assign result to Vx)
		case 0x5:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("SUB V%x V%x\n", x, y)

		// 8xy6: SHR Vx Vy (set VF=1 if the lowest bit of Vx is 1 otherwise set VF=0, then right shift Vx by 1)
		case 0x6:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("SHR V%x V%x\n", x, y)

		// 8xy7: SUBN Vx Vy (set VF=1 if Vy > Vx otherwise set VF=0, sub Vx Vy, assign result to Vx)
		case 0x7:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("SUBN V%x V%x\n", x, y)

		// 8xyE: SHR Vx Vy (set VF=1 if the highest bit of Vx is 1 otherwise set VF=0, then left shift Vx by 1)
		case 0xE:
			x := opcode & 0x0f00 >> 8
			y := opcode & 0x00f0 >> 4
			fmt.Printf("SHR V%x V%x\n", x, y)

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}

	case 0x9:
		// 9xy0: SNE Vx Vy (skip next opcode if Vx != Vy)
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		fmt.Printf("SNE V%x V%x\n", x, y)

	case 0xA:
		// Annn: LD I addr (set I=nnn)
		addr := opcode & 0x0fff
		fmt.Printf("LD I %03x\n", addr)

	case 0xB:
		// Bnnn: JP V0 addr (jump to address nnn + v0, set PC=nnn + v0)
		addr := opcode & 0x0fff
		fmt.Printf("JP V0 %03x\n", addr)

	case 0xC:
		// Cxkk: RND Vx byte (Vx = random byte and kk)
		x := opcode & 0x0f00 >> 8
		kk := opcode & 0x00ff
		fmt.Printf("RND V%x %02x\n", x, kk)

	case 0xD:
		// Dxyn: DRW Vx Vy n (display n-byte sprite located at I at coordinates Vx,Vy, set VF=collision [if sprite is drawn on top of any active pixels])
		x := opcode & 0x0f00 >> 8
		y := opcode & 0x00f0 >> 4
		n := opcode & 0x000f
		fmt.Printf("DRW V%x V%x %x\n", x, y, n)

	case 0xE:
		switch lastTwo := opcode & 0x0ff; lastTwo {
		// Ex9E: SKP Vx (skip next instruction if key with the value of Vx is currently pressed)
		case 0x9E:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("SKP V%x\n", x)

		// ExA1: SKNP Vx (skip next instruction if key with the value of Vx is currently not pressed)
		case 0xA1:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("SKNP V%x\n", x)

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}
	case 0xF:
		switch lastTwo := opcode & 0x0ff; lastTwo {

		// Fx07: LD Vx DT (set Vx=DT)
		case 0x07:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("LD V%x DT\n", x)

		// Fx0A: LD Vx K (wait for key press, store value of key press in Vx)
		case 0x0a:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("LD V%x K\n", x)

		// Fx15: LD DT Vx (set DT=Vx)
		case 0x15:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("LD DT V%x\n", x)

		// Fx18: LD ST Vx (set ST=Vx)
		case 0x18:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("LD ST V%x\n", x)

		// Fx1E: ADD I Vx (set I=I+Vx)
		case 0x1E:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("ADD I V%x\n", x)

		// Fx29: LD F Vx (set I=memory address of sprite corresponding to digit in Vx)
		case 0x29:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("LD F V%x\n", x)

		// Fx33: LD B Vx (store binary converted decimal [BCD] representation of number in Vx in memory locations I(hundreds place), I+1(tens place), I+2(ones place)
		case 0x33:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("LD B V%x\n", x)

		// Fx55: LD I Vx (store registers V0 through Vx in memory starting at I)
		case 0x55:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("LD I V%x\n", x)

		// Fx65: LD Vx I (read values in memory starting at I into registers V0 through Vx)
		case 0x65:
			x := opcode & 0x0f00 >> 8
			fmt.Printf("LD V%x I\n", x)

		default:
			panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
		}

	default:
		panic(fmt.Sprintf("Unrecognized opcode: %04x", opcode))
	}

}

func (c *Chip8) nextOpcode() uint16 {
	// skip the address pointer over the previous opcodes
	c.i += 2
	// the opcode we want to read is the next two bytes,
	// stored big-endian.
	high := c.memory[c.i]
	low := c.memory[c.i+1]
	// combine bytes as one uint16,
	// keeping the big-endian representation
	opcode := (uint16(high) << 8) | uint16(low)
	return opcode
}
