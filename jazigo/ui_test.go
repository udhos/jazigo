package main

import (
	"testing"
)

func TestSplitBufLines(t *testing.T) {
	split(t, "", 0)
	split(t, "x", 1)
	split(t, "\n", 1)
	split(t, "x\n", 1)
	split(t, "\nx", 2)
	split(t, "x\nx", 2)
	split(t, "\n\n", 2)
	split(t, "x\n\n", 2)
	split(t, "\nx\n", 2)
	split(t, "\n\nx", 3)
	split(t, "\n\nx\n", 3)
	split(t, "\n\n\n", 3)
}

func split(t *testing.T, input string, wantLineCount int) {
	result := splitBufLines([]byte(input))
	count := len(result)
	if count != wantLineCount {
		t.Errorf("splitBufLines: input=%v expected=%d got=%d", input, wantLineCount, count)
	}
}
