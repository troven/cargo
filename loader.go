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
	sources   map[TemplateMode][]string
	templates map[TemplateMode]*template.Template

	// filepathTplRx contains a precompiled Rx for replacing template tags
	// in file paths, token delims must be quoted before compiling such Rx.
	filepathTplRx *regexp.Regexp
}

type TemplateMode string

const (
	TemplateModeSingle     TemplateMode = "single"
	TemplateModeVerbatim   TemplateMode = "verbatim"
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
		sources:   make(map[TemplateMode][]string, 2),
		templates: make(map[TemplateMode]*template.Template, 2),
	}
	seen := make(map[string]struct{})
	for _, path := range paths {
		fullPath, err := filepath.Abs(path)
		if err != nil {
			log.WithFields(log.Fields{
				"Path": path,
			}).Warningln("unable to convert path to absolute, skipping")
			continue
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			log.WithFields(log.Fields{
				"Path":     path,
				"FullPath": fullPath,
			}).Warningln("unable to stat, skipping")
			continue
		}
		if !info.IsDir() {
			if _, ok := seen[fullPath]; ok {
				continue
			} else {
				seen[fullPath] = struct{}{}
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
				"Path": fullPath,
			}).Warningln("unable to walk down the path, skipping")
			continue
		}
	}
	sort.Strings(loader.sources[TemplateModeSingle])
	sort.Strings(loader.sources[TemplateModeVerbatim])
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
	mode := TemplateModeVerbatim
	if l.filepathTplRx.MatchString(path) {
		mode = TemplateModeCollection
	} else if strings.HasPrefix(name, l.opts.ModePrefix) {
		path = filepath.Join(dir, name)
		mode = TemplateModeSingle
	}
	l.sources[mode] = append(l.sources[mode], path)
}

// findCollectionPrefix finds the shortest prefix of a collection referenced in selector.
// It returns two new selectors: collection selector from TemplateContext,
// also field selector for elements in collection. It returns false,
// if no collection prefix found.
func findCollectionPrefix(c TemplateContext, selector string) (string, string, bool) {
	parts := strings.Split(selector, ".")
	prefix := parts[0]
	if _, ok := c.LengthOf(prefix); ok {
		if len(parts) == 1 {
			// only collection specified
			return prefix, "", true
		}
		// collection prefix and field selector specified
		selector := strings.Join(parts[1:], ".")
		return prefix, selector, true
	} else if len(parts) == 1 {
		// only one part specified and it's not a collection prefix
		return "", "", false
	}
	for i := 1; i < len(parts); i++ {
		prefix += "." + parts[i]
		if _, ok := c.LengthOf(prefix); ok {
			if i == len(parts)-1 {
				// was last part — the whole selector is a collection prefix
				return prefix, "", true
			}
			// collection prefix and field selector specified
			selector := strings.Join(parts[i:], ".")
			return prefix, selector, true
		}
	}
	// no collection prefix found in the specified selector
	return "", "", false
}

type collectionCache struct {
	CollectionSelector string
	ItemFieldSelector  string
}

// RenderFilepath yields a map of one or multiple file paths based on path template. If template references a collection,
// the output mapping will have all keys mapped to the corresponding TemplateContext with "Current" field set.
//
// Example: "{{ friends.name }}"" will be mapped as
// ("Alice" => TemplateContext), where TemplateContext.Current is TemplateContext.Friends[0].
// ("Bob" => TemplateContext), where TemplateContext.Current is TemplateContext.Friends[1].
func (l *TemplateLoader) RenderFilepath(
	rootCtx TemplateContext, pathTemplate string) (map[string]TemplateContext, error) {

	var err error
	var collectionSelector string
	var collectionLength int
	cache := make(map[string]collectionCache)

	// replaceWithCurrent is a function that replaces all templated placeholders in pathTemplate,
	// using idx as an offset in the collection, if collection references are used in pathTemplate.
	//
	// The template can only have one collection reference, but multiple collection item field references.
	// For example, it can has {{friends.name}}_{{friends.age}} so the collection "friends" will be traversed once,
	// and on each idx like friends[0], friends[1], ..., friends[idx] these placeholders will be replaced
	// with "name" and "age" field values from the corresponding collection items.
	replaceWithCurrent := func(idx int) (string, TemplateContext) {
		var currentCtx TemplateContext
		path := l.filepathTplRx.ReplaceAllStringFunc(pathTemplate, func(field string) string {
			if cached, ok := cache[field]; ok {
				// already parsed field, on previous iteration, and it's a collection
				if item, found := rootCtx.
					CurrentAt(cached.CollectionSelector, 0).
					CurrentItem(cached.ItemFieldSelector); found {
					return fmt.Sprintf("%v", item)
				}
				return ""
			}
			selector := strings.TrimPrefix(field, ".")
			collection, itemField, ok := findCollectionPrefix(rootCtx, selector)
			if ok {
				// a collection prefix has been found
				if len(collectionSelector) > 0 {
					if collection != collectionSelector {
						err = fmt.Errorf("multiple collections are not expected, first was: %s", collectionSelector)
						return ""
					}
				} else {
					collectionSelector = collection
					collectionLength, _ = rootCtx.LengthOf(collection)
					cache[field] = collectionCache{
						CollectionSelector: collectionSelector,
						ItemFieldSelector:  itemField,
					}
				}
				currentCtx = rootCtx.CurrentAt(collection, idx)
				if item, found := currentCtx.CurrentItem(itemField); found {
					return fmt.Sprintf("%v", item)
				}
				return ""
			}
			if item, ok := rootCtx.Item(field); ok {
				return fmt.Sprintf("%v", item)
			}
			return ""
		})
		if currentCtx == nil {
			return path, rootCtx
		}
		return path, currentCtx
	}
	path, currentCtx := replaceWithCurrent(0)
	resultMap := map[string]TemplateContext{
		path: currentCtx,
	}
	if collectionLength > 1 {
		// we have more items to traverse in collection
		for idx := 1; idx < collectionLength; idx++ {
			path, currentCtx = replaceWithCurrent(idx)
			resultMap[path] = currentCtx
		}
	}
	return resultMap, nil
}

type RenderFunc func(tpl *template.Template, source string) error

func (l *TemplateLoader) RenderEachTemplate(mode TemplateMode, fn RenderFunc) error {
	if mode == TemplateModeVerbatim {
		err := errors.New("wrong mode: verbatim doesn't use template engin")
		return err
	}
	for _, source := range l.sources[mode] {
		if err := fn(l.templates[mode], source); err != nil {
			return err
		}
	}
	return nil
}
