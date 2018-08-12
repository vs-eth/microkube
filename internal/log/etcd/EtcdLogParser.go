//go:generate ldetool generate --package etcd --go-string logs.lde

package etcd

import (
	"github.com/uubk/microkube/internal/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ETCDLogParser struct {
	log.BaseLogParser
}

func NewETCDLogParser() (*ETCDLogParser) {
	obj := ETCDLogParser{}
	obj.BaseLogParser = *log.NewBaseLogParser(obj.handleLine)
	return &obj
}

func (h *ETCDLogParser) handleLine(lineStr string) error {
	line := ETCDLogLine{}
	ok, err := line.Extract(lineStr)
	if err != nil {
		return errors.Wrap(err, "Couldn't decode line '"+lineStr+"'!")
	}
	if !ok {
		return errors.New("Couldn't decode line '"+ lineStr+ "'")
	}

	entry := logrus.WithFields(logrus.Fields{
		"app": "etcd",
		"component": string(line.Component),
	})

	switch line.Severity {
	case "I":
		entry.Info(line.Message)
	case "E":
		entry.Error(line.Message)
	case "W":
		entry.Warning(line.Message)
	case "D":
		entry.Debug(line.Message)
	case "N": // Notice is handled as info...
		entry.Info(line.Message)
	default:
		logrus.WithFields(logrus.Fields{
			"component": "EtcdLogParser",
			"app": "microkube",
			"level": line.Severity,
		}).Warn("Unknown severity level in etcd log parser")
		entry.Warn(line.Message)
	}

	return nil
}