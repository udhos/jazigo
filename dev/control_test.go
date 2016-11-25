package dev

import (
	"bytes"
	"testing"
)

type controlLogger struct {
	*testing.T
}

func (t *controlLogger) Printf(format string, v ...interface{}) {
	t.Logf("controlLogger: "+format, v...)
}

func TestControl1(t *testing.T) {
	logger := &controlLogger{t}
	debug := false

	empty := []byte{}
	crlf := []byte{CR, LF}
	four := []byte("1234")
	five := []byte("12345")
	oneBS := []byte{BS}
	oneCR := []byte{CR}
	fiveBS := append([]byte("12345"), BS)
	fiveCR := append([]byte("12345"), CR)
	middleBS := []byte{'1', '2', '3', BS, '4', '5'}
	middleCR := []byte{'1', '2', '3', CR, '4', '5'}

	control(t, debug, logger, "empty", empty, empty, empty, empty)
	control(t, debug, logger, "bufCRLF", crlf, empty, crlf, empty)
	control(t, debug, logger, "suffixCRLF", empty, crlf, empty, crlf)
	control(t, debug, logger, "bothCRLF", crlf, crlf, crlf, crlf)
	control(t, debug, logger, "no-control", five, five, five, five)

	control(t, debug, logger, "suffix-BS1", empty, oneBS, empty, empty)
	control(t, debug, logger, "suffix-BS2", five, oneBS, four, empty)
	control(t, debug, logger, "suffix-fiveBS1", empty, fiveBS, empty, four)
	control(t, debug, logger, "suffix-fiveBS2", five, fiveBS, five, four)
	control(t, debug, logger, "suffix-middleBS1", empty, middleBS, empty, []byte("1245"))
	control(t, debug, logger, "suffix-middleBS2", five, middleBS, five, []byte("1245"))

	control(t, debug, logger, "suffix-CR1", empty, oneCR, empty, empty)
	control(t, debug, logger, "suffix-CR2", five, oneCR, empty, empty)
	control(t, debug, logger, "suffix-fiveCR1", empty, fiveCR, empty, empty)
	control(t, debug, logger, "suffix-fiveCR2", five, fiveCR, empty, empty)
	control(t, debug, logger, "suffix-middleCR1", empty, middleCR, empty, []byte("45"))
	control(t, debug, logger, "suffix-middleCR2", five, middleCR, empty, []byte("45"))
}

func control(t *testing.T, debug bool, logger hasPrintf, label string, inputBuf, inputSuffix, expectedBuf, expectedSuffix []byte) {
	buf := clone(inputBuf)
	suffix := clone(inputSuffix)

	gotBuf, gotSuffix := removeControlChars(logger, debug, buf, suffix)

	if !bytes.Equal(gotBuf, expectedBuf) {
		t.Errorf("%s: buf mismatch: got=%q wanted=%q", label, gotBuf, expectedBuf)
	}

	if !bytes.Equal(gotSuffix, expectedSuffix) {
		t.Errorf("%s: suffix mismatch: got=%q wanted=%q", label, gotSuffix, expectedSuffix)
	}
}

func clone(a []byte) []byte {
	b := make([]byte, len(a))
	copy(b, a)
	return b
}
