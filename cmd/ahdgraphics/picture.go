package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	_ "image/jpeg" // registers the JPEG decoder
	"image/png"
	"io/fs"
	"os"
	"sync"
	"time"
)

// An image request names a PNG or JPEG file. The helper reads it once and
// keeps the decoded pixels, so a program that redraws the same picture on
// every timer tick decodes it only the first time. A file that changed on
// disk (by modification time or size) is read again.

const (
	// maxPictureBytes bounds the file read for one image.
	maxPictureBytes = 64 << 20
	// maxPictureSide and maxPicturePixels bound the decoded picture.
	maxPictureSide   = 8192
	maxPicturePixels = 32 << 20
	// maxPictures bounds the decoded files one Canvas keeps.
	maxPictures = 256
)

type picture struct {
	pixels   image.Image
	modified time.Time
	size     int64

	encodeOnce sync.Once
	dataURI    string
}

// picture returns the decoded file at path, reading it if it is new or has
// changed. The path is absolute; the runtime resolved it.
func (m *model) picture(path string) (*picture, error) {
	if path == "" || len(path) > maxPathBytes {
		return nil, errors.New("invalid image path")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, pictureProblem(err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("could not read the image: it is not a file")
	}
	if info.Size() > maxPictureBytes {
		return nil, errors.New("the image file is larger than 64 MiB")
	}
	m.mu.Lock()
	cached := m.pictures[path]
	m.mu.Unlock()
	if cached != nil && cached.size == info.Size() && cached.modified.Equal(info.ModTime()) {
		return cached, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, pictureProblem(err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		return nil, errors.New("the file is not a PNG or JPEG image")
	}
	if config.Width < 1 || config.Height < 1 || config.Width > maxPictureSide || config.Height > maxPictureSide ||
		config.Width*config.Height > maxPicturePixels {
		return nil, errors.New("the image is too large; each side may be at most 8192 pixels")
	}
	pixels, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("the image file is damaged and could not be decoded")
	}
	loaded := &picture{pixels: pixels, modified: info.ModTime(), size: info.Size()}
	m.mu.Lock()
	if m.pictures == nil {
		m.pictures = make(map[string]*picture)
	}
	if _, replacing := m.pictures[path]; !replacing && len(m.pictures) >= maxPictures {
		// Commands already drawn keep their own reference; only the lookup
		// table is emptied.
		m.pictures = make(map[string]*picture)
	}
	m.pictures[path] = loaded
	m.mu.Unlock()
	return loaded, nil
}

func pictureProblem(err error) error {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		return errors.New("could not read the image: " + pathError.Err.Error())
	}
	return errors.New("could not read the image")
}

// embedded returns the picture as a PNG data URI for the SVG export.
func (p *picture) embedded() string {
	p.encodeOnce.Do(func() {
		var buffer bytes.Buffer
		if png.Encode(&buffer, p.pixels) == nil {
			p.dataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
		}
	})
	return p.dataURI
}
