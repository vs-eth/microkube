// Package log contains log parsers for output of subprocesses
package log

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
	"strings"
)

// LineHandlerFunc describes a function that is able to consume a log line
type LineHandlerFunc func(string) error

// Parser is the interface type of all log-parsing classes in this package.
// A Parser is used to handle the output of child processes and re-log it using the logger of the main process,
// unifying all logs into a single log with a coherent structure
type Parser interface {
	// HandleData takes all necessary actions with the provided buffer
	HandleData(data []byte) error
}

// BaseLogParser provides utilities often needed to implement log parsers, that is line reassembly functionality
type BaseLogParser struct {
	buf         *bytes.Buffer
	bufReader   *bufio.Scanner
	lineHandler LineHandlerFunc
}

// NewBaseLogParser creates a new log parser that uses the provided line handler
func NewBaseLogParser(lineHandler LineHandlerFunc) *BaseLogParser {
	obj := &BaseLogParser{
		buf:         &bytes.Buffer{},
		lineHandler: lineHandler,
	}
	return obj
}

// See docs of interface type
func (lp *BaseLogParser) HandleData(data []byte) error {
	lp.buf.Write(data)

	if strings.Contains(lp.buf.String(), "\n") {
		line, err := lp.buf.ReadString('\n')
		if err == nil {
			err := lp.lineHandler(line)
			if err != nil {
				return errors.Wrap(err, "Couldn't decode buffer")
			}
		} else {
			return errors.New("Buffer contained '\\n' but no line could be read?")
		}
	}

	return nil
}
