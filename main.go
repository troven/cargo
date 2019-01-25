package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/jawher/mow.cli"
	"github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"
)

var app = cli.App("cargo", "Cargo is a simple static site builder.")
var logLevel = app.IntOpt("l log-level", 4, "Sets the log level [0 = no log, 5 = debug].")
var dryRun = app.BoolOpt("d dry-run", false, "Do not modify filesystem, only print planned actions.")
var delimiters = app.StringOpt("delimiters", "{{,}}", "Comma-seprated delimiters to scan in templates, left and right.")
var prefix = app.StringOpt("prefix", "_", "Prefix in filenames to specify singluar templates.")
var contextSources = app.StringsOpt("context", nil,
	"Specify multiple context sources in format Name=<yaml/json file> (e.g. Values=helm-chart-values.yaml")

var srcDir = app.StringArg("SRC", "", "Specify source files dir for your site.")
var dstDir = app.StringArg("DST", "published", "Specify destination dir for your site publication.")

func main() {
	app.Spec = "SRC [DST]"
	app.Before = func() {
		log.AddHook(filename.NewHook())
		log.SetLevel(log.Level(*logLevel))
	}
	app.Action = mainCmd
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func mainCmd() {
	delimsParsed := strings.Split(*delimiters, ",")
	if len(delimsParsed) != 0 {
		log.Fatalln("[ERR] incorrect delimiters specification:", *delimiters)
	}
	srcAbsPath, err := filepath.Abs(*srcDir)
	if err != nil {
		log.Fatalln("[ERR] cannot get absolute path for:", *srcDir)
	}
	loader, err := NewTemplateLoader(
		[]string{*srcDir},
		&TemplateLoaderOptions{
			ModePrefix: *prefix,
			LeftDelim:  delimsParsed[0],
			RightDelim: delimsParsed[1],
		})
	if err != nil {
		log.Fatalln("[ERR]", err)
	}
	c := NewTemplateContext()
	for _, source := range *contextSources {
		parts := strings.Split(source, "=")
		if len(parts) != 2 {
			err := fmt.Errorf("incorrect context source specification: %s", source)
			log.Fatalln("[ERR]", err)
		}
		name := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[1])
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalln("[ERR]", err)
		}
		sourceExt := filepath.Ext(path)
		if sourceExt == ".json" {
			if err := c.LoadFromJSON(name, data); err != nil {
				err = fmt.Errorf("error loading %s: %v", path, err)
				log.Fatalln("[ERR]", err)
			}
		} else if sourceExt == ".yaml" || sourceExt == ".yml" {
			if err := c.LoadFromYAML(name, data); err != nil {
				err = fmt.Errorf("error loading %s: %v", path, err)
				log.Fatalln("[ERR]", err)
			}
		} else {
			err := fmt.Errorf("unsupported Context source format: %s", sourceExt)
			log.Fatalln("[ERR]", err)
		}
	}
	if err := c.LoadEnvVars(); err != nil {
		log.Fatalln("[ERR]", err)
	}
	if err := c.LoadOsVars(); err != nil {
		log.Fatalln("[ERR]", err)
	}

	var singularActions []QueueAction
	if err := loader.RenderEachTemplate(TemplateModeSingle, func(tpl *template.Template, source string) error {
		contents, err := renderTemplate(tpl, source, c)
		if err != nil {
			err = fmt.Errorf("template rendering failed for %s: %v", source, err)
			log.Fatalln("[ERR]", err)
		}

		relativePath := strings.TrimSuffix(source, srcAbsPath)
		target := filepath.Join(*dstDir, relativePath)
		if info, err := os.Stat(target); os.IsNotExist(err) {
			singularActions = append(singularActions, CreateNewFileAction(target, contents))
		} else if info.IsDir() {
			log.Fatalln("[ERR] target is dir:", target)
		} else {
			singularActions = append(singularActions, OverwriteFileAction(target, contents))
		}
		return nil
	}); err != nil {
		err = fmt.Errorf("tempate vlaidation failed: %v", err)
		log.Fatalln("[ERR]", err)
	}
	actionQueue := NewQueue(
		NewDirAction(*dstDir),
	)
	actionQueue = append(actionQueue, singularActions...)

	if *dryRun {
		fmt.Println(actionQueue.Description())
		return
	}
	ts := time.Now()
	if !actionQueue.Exec() {
		log.Infoln("failed in", time.Since(ts))
		return
	}
	log.Infoln("done in", time.Since(ts))
}

func renderTemplate(tpl *template.Template, source string, c TemplateContext) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := tpl.ExecuteTemplate(buf, source, c); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	return data, nil
}
