package cmd

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path"
)

func EnsureDir(root, subdirectory string, permissions os.FileMode) error {
	dir := path.Join(root, subdirectory)

	// Errors in mkdir are ignored
	err := os.Mkdir(dir, permissions)
	if err == nil {
		log.WithField("dir", dir).Debug("Directory created")
	}

	info, err := os.Stat(dir)
	if err != nil {
		log.WithField("dir", dir).WithError(err).Fatal("Couldn't stat directory")
		return err
	}
	if !info.IsDir() {
		log.WithField("dir", dir).WithError(err).Fatal("Directory is not a directory")
		return err
	}
	return nil
}
