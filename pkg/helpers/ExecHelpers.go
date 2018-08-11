package helpers

import (
	"github.com/pkg/errors"
	"os"
	"path"
)

// Try to find binary 'name'. The following locations are checked in this order:
// * cwd/third_party/name
// * cwd/../../../third_party/name
// * 'appdir'/third_party/name
func FindBinary(name string, appDir string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "couldn't read cwd")
	}
	wd = path.Join(wd, "third_party", name)
	_, err = os.Stat(wd)
	if err == nil {
		return wd, nil
	}
	wd, err = os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "couldn't read cwd")
	}
	wd = path.Dir(path.Dir(path.Dir(wd)))
	wd = path.Join(wd, "third_party", name)
	_, err = os.Stat(wd)
	if err == nil {
		return wd, nil
	}
	wd, err = os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "couldn't read cwd")
	}
	wd = path.Dir(path.Dir(wd))
	wd = path.Join(wd, "third_party", name)
	_, err = os.Stat(wd)
	if err == nil {
		return wd, nil
	}
	wd, err = os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "couldn't read cwd")
	}
	wd = path.Dir(wd)
	wd = path.Join(wd, "third_party", name)
	_, err = os.Stat(wd)
	if err == nil {
		return wd, nil
	}
	wd = path.Join(appDir, "third_party", name)
	_, err = os.Stat(wd)
	return wd, err
}
