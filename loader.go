package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

type TemplateLoader struct {
	opts      *TemplateLoaderOptions
	sources   [TemplateMode][]string
	templates map[TemplateMode]*template.Template

	// filepathTplRx contains a precompiled Rx for replacing template tags
	// in file paths, token delims must be quoted before compiling such Rx.
	filepathTplRx *regexp.Regexp
}

type TemplateMode string

const (
	TemplateModeSingle     TemplateMode = "single"
	TemplateModeCollection TemplateMode = "collection"
)

type TemplateLoaderOptions struct {
	LeftDelim  string
	RightDelim string
	ModePrefix string
}

func checkTemplateLoaderOptions(opts *TemplateLoaderOptions) *TemplateLoaderOptions {
	if opts == nil {
		opts = new(TemplateLoaderOptions)
	}
	if len(opts.LeftDelim) == 0 {
		opts.LeftDelim = "{{"
	}
	if len(opts.RightDelim) == 0 {
		opts.RightDelim = "}}"
	}
	if len(opts.ModePrefix) == 0 {
		opts.ModePrefix = "_"
	}
	return opts
}

// NewTemplateLoader returns a new template loader with all files stat'd and
// categorized into rendiring modes [single, collection] based on name prefix.
func NewTemplateLoader(paths []string, opts *TemplateLoaderOptions) (*TemplateLoader, error) {
	loader := &TemplateLoader{
		opts:      checkTemplateLoaderOptions(opts),
		sources:   make([TemplateMode][]string, 2),
		templates: make(map[TemplateMode]*template.Template, 2),
	}
	seen := make(map[string]struct{})
	for _, path := range paths {
		fullPath, err := filepath.Abs()
		if err != nil {
			log.WithFields(log.Fields{
				"Path", path,
			}).Warningln("unable to convert path to absolute, skipping")
			continue
		}
		if info, err := os.Stat(fullPath); !info.IsDir() {
			if _, ok := seen[name]; ok {
				return nil
			} else {
				seen[name] = struct{}{}
			}
			loader.addFileSource(fullPath)
			continue
		}
		if err := filepath.Walk(fullPath, func(name string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if _, ok := seen[name]; ok {
				return nil
			} else {
				seen[name] = struct{}{}
			}
			loader.addFileSource(name)
			return nil
		}); err != nil {
			log.WithFields(log.Fields{
				"Path", fullPath,
			}).Warningln("unable to walk down the path, skipping")
			continue
		}
	}
	sort.Strings(loader.sources[TemplateModeSingle])
	sort.Strings(loader.sources[TemplateModeCollection])

	if tpl, err := template.ParseFiles(loader.sources[TemplateModeSingle]...); err != nil {
		err = fmt.Errorf("template parse error: %v", err)
		return nil, err
	} else {
		loader.templates[TemplateModeSingle] = tpl
	}
	if tpl, err := template.ParseFiles(loader.sources[TemplateModeCollection]...); err != nil {
		err = fmt.Errorf("template parse error: %v", err)
		return nil, err
	} else {
		loader.templates[TemplateModeCollection] = tpl
	}

	loader.filepathTplRx = regexp.MustCompile(
		regexp.QuoteMeta(loader.opts.LeftDelim) +
			`\s*(?P<field>\.?[a-zA-Z0-9_.]+)\s*` +
			regexp.QuoteMeta(loader.opts.RightDelim),
	)

	return loader, nil
}

func (l *TemplateLoader) addFileSource(path string) {
	name := filepath.Base(path)
	dir := filepath.Dir(path)
	mode := TemplateCollection
	if strings.HasPrefix(name, l.opts.ModePrefix) {
		path = filepath.Join(dir, name)
		mode = TemplateSingle
	}
	l.sources[mode] = append(l.sources[mode], path)
}

func (l *TemplateLoader) RenderFilepath(c TemplateContext, pathTemplate string) (string, error) {
	var err error
	var collectionField string
	result := l.filepathTplRx.ReplaceAllStringFunc(pathTemplate, func(field string) string {
		field = strings.TrimPrefix(field, ".")
		if itemsLength := c.LengthOf(field); itemsLength > 0 {
			if len(collectionField) > 0 {
				err = fmt.Errorf("multiple collections are not expected at the same time, first was: %s", collectionField)
				return ""
			} else {
				collectionField = field
			}
			current, ok := c["Current"]
			if !ok {
				err = errors.New("field points to a collection, but current item is not set")
				return ""
			}
			return fmt.Sprintf("%v", current)
		}
		v, ok := c.Item(field)
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

type RenderFunc func(tpl *template.Template, source string) error

func (l *TemplateLoader) RenderEachTemplate(mode TemplateMode, fn RenderFunc) error {
	for _, source := range l.sources[mode] {
		if err := fn(l.templates[mode], source); err != nil {
			return err
		}
	}
	return nil
	// buf := new(bytes.Buffer)
	// if err := tpl.ExecuteTemplate(buf, v); err != nil {
	// 	err = fmt.Errorf("RenderTemplate: execution error: %v", err)
	// 	return nil, err
	// }
	// return buf.Bytes(), nil
}
