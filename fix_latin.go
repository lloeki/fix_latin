package main

import (
	"io"
	"io/ioutil"
	"os"
	"unicode/utf8"
)

type Encoding int

const (
	ISO_8859_1 Encoding = iota
	// TODO: ISO_8859_15
	CP1252
)

type Fixer struct {
	allowControl bool
	handleCP1252 bool
}

func AllowControl(f *Fixer) error {
	f.allowControl = true
	return nil
}

func Assume(e Encoding) func(*Fixer) error {
	return func(f *Fixer) error {
		f.handleCP1252 = e == CP1252
		return nil
	}
}

func Fix(r io.Reader, w io.Writer, options ...func(*Fixer) error) {
	f := &Fixer{}

	for _, option := range options {
		err := option(f)
		if err != nil {
			panic("invalid option")
		}
	}
	allow_control := f.allowControl
	handle_cp1252 := f.handleCP1252

	input, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	output := make([]byte, 0)

	for len(input) > 0 {
		// UTF-8 "self" / ASCII
		if input[0] < utf8.RuneSelf {
			output = append(output, input[0])
			input = input[1:]
			continue
		}

		// UTF-8 size 2
		if input[0] >= 0xC0 && input[0] <= 0xDF &&
			input[1] >= 0x80 && input[1] <= 0xBF {
			output = append(output, input[0])
			output = append(output, input[1])
			input = input[2:]
			continue
		}

		// UTF-8 size 3
		if input[0] >= 0xE0 && input[0] <= 0xEF &&
			input[1] >= 0x80 && input[1] <= 0xBF &&
			input[2] >= 0x80 && input[2] <= 0xBF {
			output = append(output, input[0])
			output = append(output, input[1])
			output = append(output, input[2])
			input = input[3:]
			continue
		}

		// UTF-8 size 4
		if input[0] >= 0xF0 && input[0] <= 0xF7 &&
			input[1] >= 0x80 && input[1] <= 0xBF &&
			input[2] >= 0x80 && input[2] <= 0xBF &&
			input[3] >= 0x80 && input[3] <= 0xBF {
			output = append(output, input[0])
			output = append(output, input[1])
			output = append(output, input[2])
			output = append(output, input[3])
			input = input[4:]
			continue
		}

		// UTF-8 size 5
		if input[0] >= 0xF8 && input[0] <= 0xFB &&
			input[1] >= 0x80 && input[1] <= 0xBF &&
			input[2] >= 0x80 && input[2] <= 0xBF &&
			input[3] >= 0x80 && input[3] <= 0xBF &&
			input[4] >= 0x80 && input[4] <= 0xBF {
			output = append(output, input[0])
			output = append(output, input[1])
			output = append(output, input[2])
			output = append(output, input[3])
			output = append(output, input[4])
			input = input[5:]
			continue
		}

		// CP1252
		if handle_cp1252 {
			// does not define 0x81, 0x8D, 0x8F, 0x90, 09D
			cp1252 := map[byte][]byte{
				0x80: {0xE2, 0x82, 0xAC}, // EURO SIGN
				0x82: {0xE2, 0x80, 0x9A}, // SINGLE LOW-9 QUOTATION MARK
				0x83: {0xC6, 0x92},       // LATIN SMALL LETTER F WITH HOOK
				0x84: {0xE2, 0x80, 0x9E}, // DOUBLE LOW-9 QUOTATION MARK
				0x85: {0xE2, 0x80, 0xA6}, // HORIZONTAL ELLIPSIS
				0x86: {0xE2, 0x80, 0xA0}, // DAGGER
				0x87: {0xE2, 0x80, 0xA1}, // DOUBLE DAGGER
				0x88: {0xCB, 0x86},       // MODIFIER LETTER CIRCUMFLEX ACCENT
				0x89: {0xE2, 0x80, 0xB0}, // PER MILLE SIGN
				0x8A: {0xC5, 0xA0},       // LATIN CAPITAL LETTER S WITH CARON
				0x8B: {0xE2, 0x80, 0xB9}, // SINGLE LEFT-POINTING ANGLE QUOTATION MARK
				0x8C: {0xC5, 0x92},       // LATIN CAPITAL LIGATURE OE
				0x8E: {0xC5, 0xBD},       // LATIN CAPITAL LETTER Z WITH CARON
				0x91: {0xE2, 0x80, 0x98}, // LEFT SINGLE QUOTATION MARK
				0x92: {0xE2, 0x80, 0x99}, // RIGHT SINGLE QUOTATION MARK
				0x93: {0xE2, 0x80, 0x9C}, // LEFT DOUBLE QUOTATION MARK
				0x94: {0xE2, 0x80, 0x9D}, // RIGHT DOUBLE QUOTATION MARK
				0x95: {0xE2, 0x80, 0xA2}, // BULLET
				0x96: {0xE2, 0x80, 0x93}, // EN DASH
				0x97: {0xE2, 0x80, 0x94}, // EM DASH
				0x98: {0xCB, 0x9C},       // SMALL TILDE
				0x99: {0xE2, 0x84, 0xA2}, // TRADE MARK SIGN
				0x9A: {0xC5, 0xA1},       // LATIN SMALL LETTER S WITH CARON
				0x9B: {0xE2, 0x80, 0xBA}, // SINGLE RIGHT-POINTING ANGLE QUOTATION MARK
				0x9C: {0xC5, 0x93},       // LATIN SMALL LIGATURE OE
				0x9E: {0xC5, 0xBE},       // LATIN SMALL LETTER Z WITH CARON
				0x9F: {0xC5, 0xB8},       // LATIN CAPITAL LETTER Y WITH DIAERESIS
			}
			if bytes, ok := cp1252[input[0]]; ok {
				for _, b := range bytes {
					output = append(output, b)
				}
				input = input[1:]
				continue
			}
		}

		// ISO-8859-1 high-order control chars
		if !allow_control && input[0] >= 0x80 && input[0] <= 0x9F {
			panic("control char")
			continue
		}

		// convert ISO-8859-1 to UTF-8
		if input[0] >= 0x80 && input[0] <= 0xFF {
			bytes := []byte(string(rune(input[0])))
			for _, b := range bytes {
				output = append(output, b)
			}
			input = input[1:]
			continue
		}

		panic("unhandled char")
	}

	w.Write(output)
}

func main() {
	Fix(os.Stdin, os.Stdout, AllowControl, Assume(CP1252))
}
