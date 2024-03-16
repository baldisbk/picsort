package main

import (
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
)

var extensions = []string{"jpg", "jpeg", "png", "heic", "webp"}

var NotPictureErr = errors.New("not a picture file")

func ReadHash(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}

	h := sha256.New()
	r := io.TeeReader(f, h)

	if _, err := io.ReadAll(r); err != nil {
		return "", fmt.Errorf("read: %w", err)
	}

	return base32.StdEncoding.EncodeToString(h.Sum([]byte{})), nil
}

func ReadExif(name string) (cam string, ts time.Time, err error) {
	ext := path.Ext(name)
	found := false
	for _, e := range extensions {
		if "."+e == strings.ToLower(ext) {
			found = true
			break
		}
	}
	if !found {
		return "", time.Time{}, NotPictureErr
	}

	f, err := os.Open(name)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("open: %w", err)
	}

	info, err := os.Stat(name)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("stat %q: %w", name, err)
	}

	defer func() {
		if err := recover(); err != nil {
			cam, ts = "Unknown Unknown", info.ModTime()
		}
	}()

	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(f)
	if err != nil {
		return "Unknown Unknown", info.ModTime(), nil
	}

	var maker, model string
	camMake, err := x.Get(exif.Make)
	if err != nil {
		maker = "Unknown"
	} else {
		maker = camMake.String()
		if q, e := strconv.Unquote(maker); e == nil {
			maker = q
		}
	}

	camModel, err := x.Get(exif.Model)
	if err != nil {
		model = "Unknown"
	} else {
		model = camModel.String()
		if q, e := strconv.Unquote(model); e == nil {
			model = q
		}
	}

	tm, err := x.DateTime()
	if err != nil {
		tm = info.ModTime()
	}
	return maker + " " + model, tm, nil
}
