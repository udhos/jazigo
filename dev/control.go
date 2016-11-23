package dev

import (
	"bytes"
)

func removeControlChars(logger hasPrintf, debug bool, buf []byte, appendedLen int) []byte {
	bufSize := len(buf)
	minSize := bufSize - appendedLen
	previousWasLF := false

	for i := bufSize - 1; i >= minSize; i-- {
		b := buf[i]
		switch {
		case b == LF:
			previousWasLF = true
			continue
		case b == CR:
			if !previousWasLF {
				// sole CR: perform carriage return
				j := bytes.LastIndexByte(buf[:i], LF)
				if j < 0 {
					// LF not found
					if debug {
						logger.Printf("removeControlChars: cutCR1=[%q]", string(buf[:i+1]))
					}
					buf = append(buf[:0], buf[i+1:]...) // shift all
					i = 0
				} else {
					// LF found: shift over it
					if j > 0 {
						if buf[j-1] == CR {
							j--
						}
					}
					if debug {
						logger.Printf("removeControlChars: cutCR2=[%q]", string(buf[j:i+1]))
					}
					buf = append(buf[:j], buf[i+1:]...) // shift over next [CR] LF
					i = j
				}
			}
		case b == BS:
			// perform backspace: remove two chars
			if i > 0 {
				if debug {
					logger.Printf("removeControlChars: cutBS=[%q]", string(buf[i-1:i+1]))
				}
				buf = append(buf[:i-1], buf[i+1:]...) // cut bytes at i-1 and i
				i--
			}
		case b < 32: // other control
			// remove the single control char
			if debug {
				logger.Printf("removeControlChars: cutOT=[%q]", string(buf[i:i+1]))
			}
			buf = append(buf[:i], buf[i+1:]...) // cut byte at i
		}
		previousWasLF = false
	}

	return buf
}
