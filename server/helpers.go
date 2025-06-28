package server

import (
	"archive/zip"
	"io"

	crtarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

func helperZipFileOpener(zf *zip.File) crtarball.Opener {
	return func() (io.ReadCloser, error) {
		return zf.Open()
	}
}

func must(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}
