package utils

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func SoftDelete(path, softDeleteDir string) error {
	var res error

	switch softDeleteDir {
	default:
		target := filepath.Join(softDeleteDir, filepath.Base(path))
		err := os.Rename(path, target)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			res = err
			if strings.Contains(err.Error(), "invalid cross-device link") {
				res = copy(path, target)
			}
		}
		fallthrough

	case "":
		err := os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			res = err
		}
	}

	return res
}

func copy(from, to string) error {
	src, err := os.OpenFile(from, os.O_RDONLY, 0640)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
