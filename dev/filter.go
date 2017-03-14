package dev

import (
	"regexp"
	"strconv"
)

// FilterTable stores line filters for custom line-by-line processing of configuration.
type FilterTable struct {
	table map[string]FilterFunc
	re1   *regexp.Regexp
	re2   *regexp.Regexp
	re3   *regexp.Regexp
	re4   *regexp.Regexp
}

// FilterFunc is a helper function type for line filters.
type FilterFunc func(hasPrintf, bool, *FilterTable, []byte, int) []byte

// NewFilterTable creates a filter table.
func NewFilterTable(logger hasPrintf) *FilterTable {
	t := &FilterTable{
		table: map[string]FilterFunc{},
		re1:   regexp.MustCompile(`^\w{3}\s\w{3}\s\d{1,2}\s`), // Thu Feb 11 15:45:43.545 BRST
		re2:   regexp.MustCompile(`^Building`),                // Building configuration...
		re3:   regexp.MustCompile(`^!! Last`),                 // !! Last configuration change at Tue Jan 26 16:40:46 2016 by user
		re4:   regexp.MustCompile(`^\w+ uptime is `),          // asr9010 uptime is 9 years, 2 weeks, 5 days, 20 hours, 3 minutes
	}
	registerFilters(logger, t.table)
	return t
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
Thu Feb 11 15:45:43.545 BRST
Building configuration...
!! IOS XR Configuration 5.1.3
!! Last configuration change at Tue Jan 26 16:40:46 2016 by user
asr9010 uptime is 9 years, 2 weeks, 5 days, 20 hours, 3 minutes
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
	if table.re4.Match(line) {
		if debug {
			logger.Printf("filterIOSXR: drop: [%s]", string(line))
		}
		return []byte{}
	}

	return line
}
