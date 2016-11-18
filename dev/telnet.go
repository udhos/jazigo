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

func telnetNegotiation(b []byte, n int, t transp) (int, error) {

	timeout := 5 * time.Second // FIXME??
	hitNeg := false

	for {
		if n < 3 {
			break
		}
		if b[0] != 255 {
			break // not IAC
		}
		if b[1] == 253 {
			// do
			opt := b[2]
			t.SetWriteDeadline(time.Now().Add(timeout)) // FIXME: handle error
			t.Write([]byte{255, 252, opt})              // IAC WONT opt - FIXME: handle error
			n = shift(b, n, 3)
			hitNeg = true
			continue
		}
		if b[1] == 251 {
			// will
			opt := b[2]
			t.SetWriteDeadline(time.Now().Add(timeout)) // FIXME: handle error
			t.Write([]byte{255, 254, opt})              // IAC DONT opt - FIXME: handle error
			n = shift(b, n, 3)
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
