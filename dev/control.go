package dev

import (
	"bytes"
	"unicode"
)

func removeControlChars(logger hasPrintf, debug bool, buf, suffix []byte) ([]byte, []byte) {

	for i := 0; i < len(suffix); i++ {
		b := suffix[i]
		switch {
		case b == LF: // do nothing, otherwise it would be killed as other control
		case b == CR:
			next := i + 1
			if next < len(suffix) {
				if suffix[next] == 0 {
					suffix[next] = LF // handle CR NULL as CR LF
				}
				if suffix[next] == LF {
					continue
				}
			}

			// sole CR: perform carriage return

			j := bytes.LastIndexByte(suffix[:i], LF)
			if j < 0 {
				// suffix: previous LF not found: search on buf

				j = bytes.LastIndexByte(buf, LF)
				if j < 0 {
					// buf: previous LF not found: shift all
					buf = buf[:0]
				} else {
					// buf: previous LF found: shift
					buf = buf[:j]
				}

				suffix = suffix[next:] // shift suffix
				i = -1                 // handle suffix from start

				continue
			}

			// suffix: previous LF found: shift over it
			if j > 0 {
				// LF is not first char on suffix: remove possible CR from suffix
				if suffix[j-1] == CR {
					j-- // kill CR LF from suffix
				}
			} else {
				// LF is first char on suffix: remove possible CR from buf
				bufSize := len(buf)
				if bufSize > 0 {
					last := bufSize - 1
					if buf[last] == CR {
						buf = buf[:last] // kill CR (last byte) from buf
					}
				}
			}
			suffix = append(suffix[:j], suffix[i+1:]...) // shift over previous [CR] LF
			i = j - 1                                    // handle j again
		case b == BS:
			// perform backspace: remove two chars
			if i > 0 {
				// remove X,BS from suffix
				suffix = append(suffix[:i-1], suffix[i+1:]...) // cut bytes at i-1 and i
				i -= 2                                         // handle i again
				continue
			}

			// remove X from buf, BS from suffix
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
			suffix = suffix[1:]
			i = -1 // handle prefix from start

		case b == 27:

			if j := i + 1; j < len(suffix) {
				switch suffix[j] {
				case '[':

					if size, ok := prefixNumberM(suffix[j+1:]); ok {
						// remove N control chars: ESC [ d d d m
						//                         i   j <----->
						//                               size
						k := j + size
						suffix = append(suffix[:i], suffix[k+1:]...) // cut bytes i..k
						i--                                          // handle i again
						continue
					}

					if k := j + 1; k < len(suffix) {
						switch suffix[k] {
						case 'A', 'B', 'C', 'D', 'J', 'K':
							// remove 3 control chars: ESC [ x
							suffix = append(suffix[:i], suffix[k+1:]...) // cut bytes i..k
							i--                                          // handle i again
						case '1', '3', '4':
							if l := k + 1; l < len(suffix) {
								switch suffix[l] {
								case '~':
									// remove 4 control chars: ESC [ x y
									suffix = append(suffix[:i], suffix[k+1:]...) // cut bytes i..k
									i--                                          // handle i again
								default:
									logger.Printf("unknown 4-char escape: %q", suffix[i:])
								}
							}
						default:
							logger.Printf("unknown 3-char escape: %q", suffix[i:])

						}
					}
				default:
					logger.Printf("unknown 2-char escape: %q", suffix[i:])

				}
			}

		case b < 32: // other control
			// remove the single control char
			suffix = append(suffix[:i], suffix[i+1:]...) // cut byte at i
			i--                                          // handle i again
		}
	}

	return buf, suffix
}

func prefixNumberM(s []byte) (int, bool) {

	foundDigit := false

	for i, c := range s {
		switch {
		case c == 'm':
			if foundDigit {
				return i + 1, true
			}
			return 0, false
		case unicode.IsDigit(rune(c)):
			foundDigit = true
		default:
			return 0, false
		}
	}

	return 0, false
}
