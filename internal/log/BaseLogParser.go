package log

import (
	"bufio"
	"github.com/pkg/errors"
	"bytes"
	"strings"
)

type LineHandlerFunc func(string) (error)

type BaseLogParser struct {
	buf *bytes.Buffer
	bufReader *bufio.Scanner
	lineHandler LineHandlerFunc
}

func NewBaseLogParser(lineHandler LineHandlerFunc) (*BaseLogParser) {
	obj := &BaseLogParser{
		buf: &bytes.Buffer{},
		lineHandler: lineHandler,
	}
	return obj
}

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