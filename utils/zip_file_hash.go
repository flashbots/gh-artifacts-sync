package utils

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

// ZipMd5 returns base64 encoded MD5 hash of a file in zip archive
func ZipMd5(f *zip.File) (string, error) {
	stream, err := f.Open()
	if err != nil {
		return "", err
	}
	defer stream.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, stream); err != nil {
		return "", err
	}

	res := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	return res, nil
}

// ZipSha256 returns base64 encoded SHA256 hash of a file in zip archive
func ZipSha256(f *zip.File) (string, error) {
	stream, err := f.Open()
	if err != nil {
		return "", err
	}
	defer stream.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, stream); err != nil {
		return "", err
	}

	res := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	return res, nil
}
