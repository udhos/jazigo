package dev

import (
	"regexp"
	"strconv"
)

type FilterTable struct {
	table map[string]FilterFunc
	re1   *regexp.Regexp
}

type FilterFunc func(*FilterTable, []byte, int) []byte

func NewFilterTable(logger hasPrintf) *FilterTable {
	t := &FilterTable{
		table: map[string]FilterFunc{},
		re1:   reCompile(`^Building`),
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

func filterDrop(table *FilterTable, line []byte, lineNum int) []byte {
	return []byte{}
}

func filterNoop(table *FilterTable, line []byte, lineNum int) []byte {
	return line
}

func filterCountLines(table *FilterTable, line []byte, lineNum int) []byte {
	line = append([]byte(strconv.Itoa(lineNum)+": "), line...)
	return line
}

/*
Fri Feb 11 15:45:43.545 BRST
Building configuration...
!! IOS XR Configuration 5.1.3
!! Last configuration change at Wed Oct 26 16:40:46 2016 by user
*/
func filterIOSXR(table *FilterTable, line []byte, lineNum int) []byte {

	if table.re1.Match(line) {
		// drop lines starting with 'Building'
		return []byte{}
	}

	return line
}
