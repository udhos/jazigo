package dev

import (
	"time"
)

const (
	cmdWill = 251
	cmdWont = 252
	cmdDo   = 253
	cmdDont = 254
	cmdIAC  = 255

	optEcho           = 1
	optSupressGoAhead = 3
	optLinemode       = 34
)

func shift(b []byte, size, offset int) int {
	copy(b, b[offset:size])
	return size - offset
}

type telnetNegotiationOnly struct{}

var telnetNegOnly = telnetNegotiationOnly{}

func (e telnetNegotiationOnly) Error() string {
	return "telnetNegotiationOnlyError"
}

func telnetNegotiation(buf []byte, n int, t transp, logger hasPrintf, debug bool) (int, error) {

	timeout := 5 * time.Second // FIXME??
	hitNeg := false

	for {
		if n < 3 {
			break
		}
		if buf[0] != cmdIAC {
			break // not IAC
		}
		if debug {
			logger.Printf("telnetNegotiation: debug: FOUND telnet IAC")
		}
		b1 := buf[1]
		switch b1 {
		case cmdDo, cmdDont:
			opt := buf[2]
			t.SetWriteDeadline(time.Now().Add(timeout)) // FIXME: handle error
			t.Write([]byte{cmdIAC, cmdWont, opt})       // IAC WONT opt - FIXME: handle error
			n = shift(buf, n, 3)
			hitNeg = true
			continue
		case cmdWill, cmdWont:
			opt := buf[2]
			t.SetWriteDeadline(time.Now().Add(timeout)) // FIXME: handle error
			t.Write([]byte{cmdIAC, cmdDont, opt})       // IAC DONT opt - FIXME: handle error
			n = shift(buf, n, 3)
			hitNeg = true
			continue
		}
		break
	}

	if n == 0 && hitNeg {
		return 0, telnetNegOnly
	}

	return n, nil
}
