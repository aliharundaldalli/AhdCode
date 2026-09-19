package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// The view file the runtime writes for one show() or Surface.save:
//
//	{"version":2,"mode":"chart","title":...,"preview":...,"renderer":...,"dialog":...,"request":{...}}
//	{"version":2,"mode":"surface","title":...,"dialog":...,"surface":{...}}
//	{"version":2,"mode":"render","surface":{...},"output":...}
//
// request is the chart's own ahdplot request (its output path is set only
// when Save runs); renderer and dialog are the absolute paths of the bundled
// ahdplot and ahdgui helpers, or empty when they are not installed, which
// only disables Save.

const (
	specVersion  = 2
	modeChart    = "chart"
	modeSurface  = "surface"
	modeRender   = "render"
	maxSpecBytes = 64 << 20
	saveTimeout  = 60 * time.Second
)

type viewSpec struct {
	Version  int             `json:"version"`
	Mode     string          `json:"mode"`
	Title    string          `json:"title,omitempty"`
	Preview  string          `json:"preview,omitempty"`
	Renderer string          `json:"renderer,omitempty"`
	Dialog   string          `json:"dialog,omitempty"`
	Request  json.RawMessage `json:"request,omitempty"`
	Surface  *surfaceSpec    `json:"surface,omitempty"`
	Output   string          `json:"output,omitempty"`
}

func validPath(path string) bool {
	return path != "" && len(path) <= maxPathBytes && utf8.ValidString(path) && !strings.ContainsRune(path, 0)
}

// readSpec reads the view file completely, deletes it, and checks it.
func readSpec(path string) (viewSpec, error) {
	if !validPath(path) {
		return viewSpec{}, errors.New("invalid view file path")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return viewSpec{}, errors.New("the view file could not be read")
	}
	if info.Size() > maxSpecBytes {
		_ = os.Remove(path)
		return viewSpec{}, errors.New("the view file is too large")
	}
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil {
		return viewSpec{}, errors.New("the view file could not be read")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var spec viewSpec
	if decoder.Decode(&spec) != nil {
		return viewSpec{}, errors.New("the view file is not valid")
	}
	return spec, spec.validate()
}

func (spec viewSpec) validate() error {
	if spec.Version != specVersion {
		return fmt.Errorf("view file version %d is not supported", spec.Version)
	}
	if !utf8.ValidString(spec.Title) || utf8.RuneCountInString(spec.Title) > maxTitleRunes || strings.ContainsRune(spec.Title, 0) {
		return errors.New("invalid window title")
	}
	for _, helper := range []string{spec.Renderer, spec.Dialog} {
		if helper != "" && (!validPath(helper) || !filepath.IsAbs(helper)) {
			return errors.New("invalid helper path")
		}
	}
	switch spec.Mode {
	case modeChart:
		if !validPath(spec.Preview) {
			return errors.New("invalid preview path")
		}
		if len(spec.Request) == 0 || spec.Request[0] != '{' {
			return errors.New("the view file has no chart request")
		}
		return nil
	case modeSurface, modeRender:
		if spec.Surface == nil {
			return errors.New("the view file has no Surface")
		}
		if spec.Mode == modeRender && (!validPath(spec.Output) || !filepath.IsAbs(spec.Output) || !strings.EqualFold(filepath.Ext(spec.Output), ".png")) {
			return errors.New("a Surface is saved to an absolute .png path")
		}
		return spec.Surface.validate()
	}
	return errors.New("unknown view mode " + spec.Mode)
}

// ---- Save --------------------------------------------------------------------

// chooseSavePath asks the ahdgui helper for a save dialog; "" means the user
// cancelled.
func chooseSavePath(dialog, suggested string, extensions []string) (string, error) {
	if dialog == "" {
		return "", errors.New("Save needs AhdCode's GUI helper (ahdgui), which is not installed")
	}
	request, _ := json.Marshal(map[string]any{"op": "dialog", "version": 3, "kind": "saveFile",
		"title": "Save", "name": suggested, "extensions": extensions})
	command := exec.Command(dialog)
	command.Stdin = bytes.NewReader(append(request, '\n'))
	output, err := command.StdoutPipe()
	if err != nil {
		return "", errors.New("the save dialog could not be shown")
	}
	if err := command.Start(); err != nil {
		return "", errors.New("the save dialog could not be shown")
	}
	line, _ := bufio.NewReaderSize(output, 64<<10).ReadBytes('\n')
	_, _ = io.Copy(io.Discard, output)
	_ = command.Wait()
	var reply struct {
		OK        bool     `json:"ok"`
		Error     string   `json:"error"`
		Paths     []string `json:"paths"`
		Cancelled bool     `json:"cancelled"`
	}
	if json.Unmarshal(line, &reply) != nil || (!reply.OK && reply.Error == "") {
		return "", errors.New("the save dialog could not be shown")
	}
	if !reply.OK {
		return "", errors.New("the save dialog could not be shown: " + reply.Error)
	}
	if reply.Cancelled || len(reply.Paths) != 1 {
		return "", nil
	}
	if !validPath(reply.Paths[0]) || !filepath.IsAbs(reply.Paths[0]) {
		return "", errors.New("the save dialog returned an invalid path")
	}
	return reply.Paths[0], nil
}

// saveChart runs the ahdplot renderer on the chart's own request with the
// chosen path, so the file is exactly what Chart.save or Figure.save writes;
// the view's zoom and turn play no part.
func saveChart(spec viewSpec, path string) error {
	if spec.Renderer == "" {
		return errors.New("Save needs AhdCode's Plot renderer (ahdplot), which is not installed")
	}
	var request map[string]json.RawMessage
	if json.Unmarshal(spec.Request, &request) != nil {
		return errors.New("the chart request is not valid")
	}
	destination, _ := json.Marshal(path)
	request["output_path"] = destination
	encoded, _ := json.Marshal(request)
	file, err := os.CreateTemp("", "ahdplotview-save-*.json")
	if err != nil {
		return errors.New("the chart could not be saved: no temporary file")
	}
	defer os.Remove(file.Name())
	_, err = file.Write(encoded)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return errors.New("the chart could not be saved: no temporary file")
	}
	ctx, cancel := context.WithTimeout(context.Background(), saveTimeout)
	defer cancel()
	output, runErr := exec.CommandContext(ctx, spec.Renderer, file.Name()).Output()
	var response struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(output, &response)
	if runErr != nil || !response.OK {
		if response.Message == "" {
			response.Message = "the renderer stopped"
		}
		return errors.New("the chart could not be saved: " + response.Message)
	}
	return nil
}

// saveSurface writes a Surface's canonical PNG: the initial camera, exactly
// Surface.save's bytes. It is written beside the destination and renamed, so
// a failed save never leaves half a file.
func saveSurface(surface surfaceSpec, path string) error {
	if !strings.EqualFold(filepath.Ext(path), ".png") {
		return errors.New("a Surface is saved as .png")
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, newSurfaceRenderer().renderCanonical(surface)); err != nil {
		return errors.New("the Surface could not be encoded")
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".ahdcode-surface-*.png")
	if err != nil {
		return fmt.Errorf("the Surface could not be saved to %s", path)
	}
	_, err = temporary.Write(buffer.Bytes())
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(temporary.Name(), 0o644)
	}
	if err == nil {
		err = os.Rename(temporary.Name(), path)
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
		return fmt.Errorf("the Surface could not be saved to %s", path)
	}
	return nil
}

// renderSurfaceFile is render mode: Surface.save without a window.
func renderSurfaceFile(spec viewSpec, out io.Writer) error {
	if err := saveSurface(*spec.Surface, spec.Output); err != nil {
		return err
	}
	answer(out, reply{Ready: true})
	return nil
}
