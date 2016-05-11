package dev

import (
	"regexp"
	"strconv"
)

type FilterTable struct {
	table map[string]FilterFunc
	re1   *regexp.Regexp
	re2   *regexp.Regexp
	re3   *regexp.Regexp
}

type FilterFunc func(hasPrintf, bool, *FilterTable, []byte, int) []byte

func NewFilterTable(logger hasPrintf) *FilterTable {
	t := &FilterTable{
		table: map[string]FilterFunc{},
		re1:   reCompile(`^\w{3}\s\w{3}\s\d{1,2}\s`), // Fri Feb 11 15:45:43.545 BRST
		re2:   reCompile(`^Building`),                // Building configuration...
		re3:   reCompile(`!! Last`),                  // !! Last configuration change at Wed Oct 26 16:40:46 2016 by user
	}
	registerFilters(logger, t.table)
	return t
}

func reCompile(s string) *regexp.Regexp {
	re, err := regexp.Compile(s)
	if err != nil {
		panic(err)
	}
	return re
}

func register(logger hasPrintf, table map[string]FilterFunc, name string, f FilterFunc) {
	logger.Printf("line filter registered: '%s'", name)
	table[name] = f
}

func registerFilters(logger hasPrintf, table map[string]FilterFunc) {
	register(logger, table, "iosxr", filterIOSXR)
	register(logger, table, "noop", filterNoop)
	register(logger, table, "drop", filterDrop)
	register(logger, table, "count_lines", filterCountLines)
}

func filterDrop(logger hasPrintf, debug bool, table *FilterTable, line []byte, lineNum int) []byte {
	return []byte{}
}

func filterNoop(logger hasPrintf, debug bool, table *FilterTable, line []byte, lineNum int) []byte {
	return line
}

func filterCountLines(logger hasPrintf, debug bool, table *FilterTable, line []byte, lineNum int) []byte {
	line = append([]byte(strconv.Itoa(lineNum)+": "), line...)
	return line
}

/*
Fri Feb 11 15:45:43.545 BRST
Building configuration...
!! IOS XR Configuration 5.1.3
!! Last configuration change at Wed Oct 26 16:40:46 2016 by user
*/
func filterIOSXR(logger hasPrintf, debug bool, table *FilterTable, line []byte, lineNum int) []byte {

	if table.re1.Match(line) {
		if debug {
			logger.Printf("filterIOSXR: drop: [%s]", string(line))
		}
		return []byte{}
	}
	if table.re2.Match(line) {
		if debug {
			logger.Printf("filterIOSXR: drop: [%s]", string(line))
		}
		return []byte{}
	}
	if table.re3.Match(line) {
		if debug {
			logger.Printf("filterIOSXR: drop: [%s]", string(line))
		}
		return []byte{}
	}

	return line
}
