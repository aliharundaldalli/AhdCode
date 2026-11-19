package ahdruntime

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

// HTML templates.
//
// A TemplateSet keeps source text and a shared prototype built only from
// layouts and partials (every name that is not under pages/). Files under
// pages/ are page-private: their {{define}} names never enter the shared
// prototype, so one page cannot override another page's blocks or hijack
// another file's identity. render and layout always Clone the prototype
// before Parse or Execute, so a shared set stays usable after any mix of
// calls, including concurrent ones. A failed render does not mutate the
// prototype.
//
// TemplateData is a JSON tree. Nested lists and objects embed child JSON
// values directly, so encoding size grows with the data, not exponentially.
// Ordinary strings are escaped by html/template. Templates.trusted is the
// only raw-markup kind. Template actions cannot call AhdCode or read files.
//
// Lookup names are slash-separated relative paths, including .html for files
// loaded from a directory. This follows html/template of the Go version that
// built this program. It is not a second template language.

const (
	ahdTemplateMaxFiles  = 256
	ahdTemplateMaxBytes  = 1 << 20
	ahdTemplateMaxDepth  = 32
	ahdTemplateMaxNodes  = 10000
	ahdTemplateMaxOutput = 8 << 20
	ahdTemplateMaxSets   = 1024
	ahdTemplateMaxSegs   = 16
	ahdTemplateMaxName   = 128
)

type ahdTemplateSet struct {
	shared  *template.Template
	sources map[string]string
}

var (
	ahdTemplatesMu   sync.Mutex
	ahdTemplates     = map[string]*ahdTemplateSet{}
	ahdTemplatesNext atomic.Int64
)

func init() {
	AhdRegisterError(AhdClassTemplateError, func(message string) AhdInstance {
		instance := &ahdModuleError{message: message}
		instance.AhdSetClass(AhdClassTemplateError)
		return instance
	})
}

func AhdTemplatesNull() string { return `{"k":"N"}` }

func AhdTemplatesFromBool(value bool) string {
	if value {
		return `{"k":"B","v":true}`
	}
	return `{"k":"B","v":false}`
}

func AhdTemplatesFromInt(value int64) string {
	return `{"k":"I","v":` + strconv.FormatInt(value, 10) + `}`
}

func AhdTemplatesFromReal(class *AhdClass, value float64) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		AhdRaiseClass(class, "template real value is not finite")
	}
	return `{"k":"R","v":` + string(encoded) + `}`
}

func AhdTemplatesFromString(value string) string {
	return `{"k":"S","v":` + ahdTemplateJSONString(value) + `}`
}

func AhdTemplatesTrusted(value string) string {
	return `{"k":"H","v":` + ahdTemplateJSONString(value) + `}`
}

func AhdTemplatesList(texts []string) string {
	class := AhdClassTemplateError
	ahdTemplateCheckChildren(class, texts)
	parts := make([]json.RawMessage, len(texts))
	for index, text := range texts {
		parts[index] = json.RawMessage(text)
	}
	encoded, err := json.Marshal(parts)
	if err != nil {
		AhdRaiseClass(class, "template data is unreadable")
	}
	return `{"k":"L","v":` + string(encoded) + `}`
}

func AhdTemplatesObject(class *AhdClass, keys, texts []string) string {
	if len(keys) != len(texts) {
		AhdRaiseClass(class, "template object is unreadable")
	}
	ahdTemplateCheckChildren(class, texts)
	seen := map[string]struct{}{}
	pairs := make([][2]json.RawMessage, len(keys))
	for index, key := range keys {
		if _, ok := seen[key]; ok {
			AhdRaiseClass(class, "template object key "+ahdHTMLQuote(key)+" is duplicated")
		}
		seen[key] = struct{}{}
		keyJSON, err := json.Marshal(key)
		if err != nil {
			AhdRaiseClass(class, "template object is unreadable")
		}
		pairs[index] = [2]json.RawMessage{json.RawMessage(keyJSON), json.RawMessage(texts[index])}
	}
	encoded, err := json.Marshal(pairs)
	if err != nil {
		AhdRaiseClass(class, "template object is unreadable")
	}
	return `{"k":"O","v":` + string(encoded) + `}`
}

func AhdTemplatesParse(class *AhdClass, name, source string) string {
	if err := ahdTemplateCheckSource(name, source); err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return ahdTemplateStore(class, map[string]string{name: source}, []string{name})
}

func AhdTemplatesSources(class *AhdClass, names, bodies []string) string {
	if len(names) != len(bodies) {
		AhdRaiseClass(class, "template names and bodies must have the same length")
	}
	if len(names) > ahdTemplateMaxFiles {
		AhdRaiseClass(class, fmt.Sprintf("a template set may contain at most %d HTML templates", ahdTemplateMaxFiles))
	}
	sources := map[string]string{}
	order := make([]string, 0, len(names))
	for index, name := range names {
		if err := ahdTemplateCheckSource(name, bodies[index]); err != nil {
			AhdRaiseClass(class, err.Error())
		}
		if _, ok := sources[name]; ok {
			AhdRaiseClass(class, "template "+ahdHTMLQuote(name)+" is listed twice")
		}
		sources[name] = bodies[index]
		order = append(order, name)
	}
	return ahdTemplateStore(class, sources, order)
}

func AhdTemplatesFiles(class *AhdClass, root string) string {
	resolved, err := ahdTemplateRoot(root)
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	sources := map[string]string{}
	var order []string
	walkErr := filepath.WalkDir(resolved, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("could not read the template directory")
		}
		reparse, err := ahdTemplateComponentReparse(path)
		if err != nil {
			return fmt.Errorf("could not read the template directory")
		}
		if reparse {
			if path == resolved {
				return fmt.Errorf("template directory must not be a symbolic link or junction")
			}
			return fmt.Errorf("a template path must not follow a symbolic link or junction")
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
			return nil
		}
		relative, err := filepath.Rel(resolved, path)
		if err != nil || !ahdTemplateInside(relative) {
			return fmt.Errorf("a template path left its directory")
		}
		name := filepath.ToSlash(relative)
		if err := ahdTemplateCheckName(name); err != nil {
			return err
		}
		if len(sources) >= ahdTemplateMaxFiles {
			return fmt.Errorf("a template directory may contain at most %d HTML files", ahdTemplateMaxFiles)
		}
		// WalkDir may descend into a Windows junction that is not a symlink.
		// Reject any reparse point on the path before reading bytes.
		if err := ahdTemplateEnsureContained(path); err != nil {
			return err
		}
		body, err := ahdTemplateReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read template %s", name)
		}
		if len(body) > ahdTemplateMaxBytes {
			return fmt.Errorf("template %s is larger than %d bytes", name, ahdTemplateMaxBytes)
		}
		if !utf8.Valid(body) {
			return fmt.Errorf("template %s is not valid UTF-8", name)
		}
		sources[name] = string(body)
		order = append(order, name)
		return nil
	})
	if walkErr != nil {
		AhdRaiseClass(class, walkErr.Error())
	}
	sort.Strings(order)
	return ahdTemplateStore(class, sources, order)
}

func AhdTemplatesRender(class *AhdClass, handle, name, data string) string {
	set := ahdTemplateLookup(class, handle)
	source, ok := set.sources[name]
	if !ok {
		AhdRaiseClass(class, "template "+ahdHTMLQuote(name)+" is not in this set")
	}
	clone, err := ahdTemplateCloneFor(set, name, source)
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return ahdTemplateExecute(class, clone, name, data)
}

func AhdTemplatesLayout(class *AhdClass, handle, layout, page, data string) string {
	set := ahdTemplateLookup(class, handle)
	if _, ok := set.sources[layout]; !ok {
		AhdRaiseClass(class, "template "+ahdHTMLQuote(layout)+" is not in this set")
	}
	if ahdTemplateIsPage(layout) {
		AhdRaiseClass(class, "template "+ahdHTMLQuote(layout)+" is a page and cannot be used as a layout")
	}
	if !ahdTemplateIsPage(page) {
		AhdRaiseClass(class, "template "+ahdHTMLQuote(page)+" must be under pages/ to be used as a page")
	}
	source, ok := set.sources[page]
	if !ok {
		AhdRaiseClass(class, "template "+ahdHTMLQuote(page)+" is not in this set")
	}
	clone, err := ahdTemplateCloneFor(set, page, source)
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return ahdTemplateExecute(class, clone, layout, data)
}

func ahdTemplateIsPage(name string) bool {
	return strings.HasPrefix(name, "pages/")
}

func ahdTemplateStore(class *AhdClass, sources map[string]string, order []string) string {
	shared, err := ahdTemplateBuildShared(sources, order)
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	ahdTemplatesMu.Lock()
	defer ahdTemplatesMu.Unlock()
	if len(ahdTemplates) >= ahdTemplateMaxSets {
		AhdRaiseClass(class, fmt.Sprintf("at most %d template sets may be loaded in one process", ahdTemplateMaxSets))
	}
	id := strconv.FormatInt(ahdTemplatesNext.Add(1), 10)
	ahdTemplates[id] = &ahdTemplateSet{shared: shared, sources: sources}
	return id
}

func ahdTemplateBuildShared(sources map[string]string, order []string) (*template.Template, error) {
	root := template.New("").Option("missingkey=error")
	claimed := map[string]string{}
	for _, name := range order {
		// Inspect the parser's names, including unquoted identifiers from
		// both define and block actions, before adding any trees to shared.
		probe, err := template.New(name).Parse(sources[name])
		if err != nil {
			return nil, fmt.Errorf("template %s is not valid: %s", ahdHTMLQuote(name), ahdTemplatePublicError(err))
		}
		page := ahdTemplateIsPage(name)
		for _, tmpl := range probe.Templates() {
			defined := tmpl.Name()
			if defined != name {
				_, fileIdentity := sources[defined]
				if defined == "" || fileIdentity || (!page && ahdTemplateIsPage(defined)) {
					return nil, fmt.Errorf("template %s must not redefine %s", ahdHTMLQuote(name), ahdHTMLQuote(defined))
				}
			}
			if page {
				continue
			}
			if owner, ok := claimed[defined]; ok {
				return nil, fmt.Errorf("template %s is defined in both %s and %s", ahdHTMLQuote(defined), ahdHTMLQuote(owner), ahdHTMLQuote(name))
			}
			claimed[defined] = name
		}
		if !page {
			if _, err := root.New(name).Parse(sources[name]); err != nil {
				return nil, fmt.Errorf("template %s is not valid: %s", ahdHTMLQuote(name), ahdTemplatePublicError(err))
			}
		}
	}
	return root, nil
}

func ahdTemplateCloneFor(set *ahdTemplateSet, name, source string) (*template.Template, error) {
	clone, err := set.shared.Clone()
	if err != nil {
		return nil, fmt.Errorf("template %s could not be prepared", ahdHTMLQuote(name))
	}
	if ahdTemplateIsPage(name) {
		if _, err := clone.New(name).Parse(source); err != nil {
			return nil, fmt.Errorf("template %s is not valid: %s", ahdHTMLQuote(name), ahdTemplatePublicError(err))
		}
		return clone, nil
	}
	if clone.Lookup(name) == nil {
		return nil, fmt.Errorf("template %s is not in this set", ahdHTMLQuote(name))
	}
	return clone, nil
}

func ahdTemplateLookup(class *AhdClass, handle string) *ahdTemplateSet {
	ahdTemplatesMu.Lock()
	set := ahdTemplates[handle]
	ahdTemplatesMu.Unlock()
	if set == nil {
		AhdRaiseClass(class, "template set is not loaded")
	}
	return set
}

func ahdTemplateExecute(class *AhdClass, set *template.Template, name, data string) string {
	value, err := ahdTemplateDecode(data, 0, new(int))
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	writer := &ahdTemplateLimitedWriter{remaining: ahdTemplateMaxOutput}
	if err := set.ExecuteTemplate(writer, name, value); err != nil {
		AhdRaiseClass(class, "template "+ahdHTMLQuote(name)+" failed: "+ahdTemplatePublicError(err))
	}
	return writer.buf.String()
}

type ahdTemplateLimitedWriter struct {
	buf       strings.Builder
	remaining int
}

func (writer *ahdTemplateLimitedWriter) Write(piece []byte) (int, error) {
	if len(piece) > writer.remaining {
		return 0, fmt.Errorf("template output is larger than %d bytes", ahdTemplateMaxOutput)
	}
	writer.remaining -= len(piece)
	return writer.buf.Write(piece)
}

func ahdTemplatePublicError(err error) string {
	text := err.Error()
	text = strings.ReplaceAll(text, "\n", " ")
	if len(text) > 300 {
		text = text[:300]
	}
	return text
}

func ahdTemplateRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" || strings.ContainsRune(root, 0) {
		return "", fmt.Errorf("template directory is empty")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("template directory could not be resolved")
	}
	absolute = filepath.Clean(absolute)
	if err := ahdTemplateEnsureContained(absolute); err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("template directory is not a directory")
	}
	reparse, err := ahdTemplateComponentReparse(absolute)
	if err != nil {
		return "", fmt.Errorf("template directory could not be resolved")
	}
	if reparse || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("template directory must not be a symbolic link or junction")
	}
	return absolute, nil
}

func ahdTemplateInside(relative string) bool {
	if relative == "" || relative == "." || filepath.IsAbs(relative) {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func ahdTemplateCheckName(name string) error {
	if name == "" || len(name) > ahdTemplateMaxName || strings.ContainsRune(name, 0) || strings.ContainsAny(name, "\\\r\n") {
		return fmt.Errorf("template name is not a relative path")
	}
	if strings.HasPrefix(name, "/") || strings.Contains(name, ":") {
		return fmt.Errorf("template name is not a relative path")
	}
	if filepath.IsAbs(name) || filepath.IsAbs(filepath.FromSlash(name)) {
		return fmt.Errorf("template name is not a relative path")
	}
	segments := strings.Split(name, "/")
	if len(segments) > ahdTemplateMaxSegs {
		return fmt.Errorf("template name is not a relative path")
	}
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || strings.HasPrefix(segment, ".") {
			return fmt.Errorf("template name is not a relative path")
		}
	}
	return nil
}

func ahdTemplateCheckSource(name, source string) error {
	if err := ahdTemplateCheckName(name); err != nil {
		return err
	}
	if len(source) > ahdTemplateMaxBytes {
		return fmt.Errorf("template %s is larger than %d bytes", name, ahdTemplateMaxBytes)
	}
	if !utf8.ValidString(source) {
		return fmt.Errorf("template %s is not valid UTF-8", name)
	}
	return nil
}

func ahdTemplateJSONString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}

// Check children before encoding the parent. Each TemplateData node
// has one JSON object envelope. Lists and object entries use arrays, and
// scalar payloads and object keys contain no structural tokens.
func ahdTemplateCheckChildren(class *AhdClass, texts []string) {
	if len(texts) >= ahdTemplateMaxNodes {
		AhdRaiseClass(class, "template data is too large")
	}
	nodes := 1 // Include the new parent in the cumulative budget.
	for _, text := range texts {
		if err := ahdTemplateCheckEncoded(text, &nodes); err != nil {
			AhdRaiseClass(class, err.Error())
		}
	}
}

func ahdTemplateCheckEncoded(text string, nodes *int) error {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	open, err := decoder.Token()
	if err != nil || open != json.Delim('{') {
		return fmt.Errorf("template data is unreadable")
	}
	key, err := decoder.Token()
	if err != nil || key != "k" {
		return fmt.Errorf("template data is unreadable")
	}
	kind, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("template data is unreadable")
	}
	switch kind {
	case "N", "B", "I", "R", "S", "H", "L", "O":
	default:
		return fmt.Errorf("template data is unreadable")
	}

	// The child root is already one level below the new parent.
	depth := 1
	if err := ahdTemplateCountNode(depth, nodes); err != nil {
		return err
	}
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("template data is unreadable")
		}
		switch token {
		case json.Delim('{'):
			depth++
			if err := ahdTemplateCountNode(depth, nodes); err != nil {
				return err
			}
		case json.Delim('}'):
			depth--
		}
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("template data is unreadable")
	}
	return nil
}

func ahdTemplateCountNode(depth int, nodes *int) error {
	if depth > ahdTemplateMaxDepth {
		return fmt.Errorf("template data is too deep")
	}
	*nodes++
	if *nodes > ahdTemplateMaxNodes {
		return fmt.Errorf("template data is too large")
	}
	return nil
}

type ahdTemplateNode struct {
	K string          `json:"k"`
	V json.RawMessage `json:"v"`
}

func ahdTemplateDecode(text string, depth int, nodes *int) (any, error) {
	if err := ahdTemplateCountNode(depth, nodes); err != nil {
		return nil, err
	}
	if text == "" {
		return nil, fmt.Errorf("template data is unreadable")
	}
	var node ahdTemplateNode
	if err := json.Unmarshal([]byte(text), &node); err != nil {
		return nil, fmt.Errorf("template data is unreadable")
	}
	switch node.K {
	case "N":
		if len(node.V) != 0 && string(node.V) != "null" {
			return nil, fmt.Errorf("template data is unreadable")
		}
		return nil, nil
	case "B":
		var value bool
		if err := json.Unmarshal(node.V, &value); err != nil {
			return nil, fmt.Errorf("template data is unreadable")
		}
		return value, nil
	case "I":
		var value int64
		if err := json.Unmarshal(node.V, &value); err != nil {
			return nil, fmt.Errorf("template data is unreadable")
		}
		return value, nil
	case "R":
		var value float64
		if err := json.Unmarshal(node.V, &value); err != nil {
			return nil, fmt.Errorf("template data is unreadable")
		}
		return value, nil
	case "S", "H":
		var value string
		if err := json.Unmarshal(node.V, &value); err != nil {
			return nil, fmt.Errorf("template data is unreadable")
		}
		if node.K == "H" {
			return template.HTML(value), nil
		}
		return value, nil
	case "L":
		var items []json.RawMessage
		if err := json.Unmarshal(node.V, &items); err != nil {
			return nil, fmt.Errorf("template data is unreadable")
		}
		decoded := make([]any, len(items))
		for index, item := range items {
			value, err := ahdTemplateDecode(string(item), depth+1, nodes)
			if err != nil {
				return nil, err
			}
			decoded[index] = value
		}
		return decoded, nil
	case "O":
		var pairs [][2]json.RawMessage
		if err := json.Unmarshal(node.V, &pairs); err != nil {
			return nil, fmt.Errorf("template data is unreadable")
		}
		object := map[string]any{}
		for _, pair := range pairs {
			var key string
			if err := json.Unmarshal(pair[0], &key); err != nil {
				return nil, fmt.Errorf("template data is unreadable")
			}
			value, err := ahdTemplateDecode(string(pair[1]), depth+1, nodes)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		return object, nil
	default:
		return nil, fmt.Errorf("template data is unreadable")
	}
}

// ahdTemplateEnsureContained rejects a symlink or junction on any existing
// component of path. A parent replaced after this check is the remaining race.
func ahdTemplateEnsureContained(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("template path cannot be resolved")
	}
	absolute = filepath.Clean(absolute)
	volume := filepath.VolumeName(absolute)
	rest := absolute
	if len(volume) > 0 && len(absolute) >= len(volume) {
		rest = absolute[len(volume):]
	}
	current := volume
	if len(volume) == 2 && volume[1] == ':' {
		current = volume + `\`
	}
	for _, part := range strings.FieldsFunc(rest, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == "" || part == "." {
			continue
		}
		if current == "" || strings.HasSuffix(current, `\`) || strings.HasSuffix(current, `/`) {
			current += part
		} else {
			current += string(os.PathSeparator) + part
		}
		reparse, err := ahdTemplateComponentReparse(current)
		if ahdTemplateMissing(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("template path cannot be read")
		}
		if reparse {
			return fmt.Errorf("a template path must not follow a symbolic link or junction")
		}
	}
	return nil
}

func ahdTemplateReadLimited(reader io.Reader, limit int) ([]byte, error) {
	limited := io.LimitReader(reader, int64(limit)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(body) > limit {
		return nil, fmt.Errorf("template is larger than %d bytes", limit)
	}
	return body, nil
}
