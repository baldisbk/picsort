package main

import (
	"crypto/sha256"
	"encoding/base32"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"golang.org/x/xerrors"
)

func ReadHash(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", xerrors.Errorf("open: %w", err)
	}

	h := sha256.New()
	r := io.TeeReader(f, h)

	if _, err := io.ReadAll(r); err != nil {
		return "", xerrors.Errorf("read: %w", err)
	}

	return base32.StdEncoding.EncodeToString(h.Sum([]byte{})), nil
}

func ReadExif(name string) (string, time.Time, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", time.Time{}, xerrors.Errorf("open: %w", err)
	}

	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(f)
	if err != nil {
		return "", time.Time{}, xerrors.Errorf("decode: %w", err)
	}

	camMake, err := x.Get(exif.Make)
	if err != nil {
		return "", time.Time{}, xerrors.Errorf("get model: %w", err)
	}
	maker := camMake.String()
	if q, e := strconv.Unquote(maker); e == nil {
		maker = q
	}

	camModel, err := x.Get(exif.Model)
	if err != nil {
		return "", time.Time{}, xerrors.Errorf("get model: %w", err)
	}
	model := camModel.String()
	if q, e := strconv.Unquote(model); e == nil {
		model = q
	}

	tm, err := x.DateTime()
	if err != nil {
		return "", time.Time{}, xerrors.Errorf("get ts: %w", err)
	}
	return maker + " " + model, tm, nil
}
