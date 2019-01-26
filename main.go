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
var dstDir = app.StringArg("DST", "published/", "Specify destination dir for your site publication.")

func main() {
	// app.Spec = "SRC [DST]"
	app.Before = func() {
		if *logLevel == 5 {
			log.SetReportCaller(true)
		}
		// log.AddHook(filename.NewHook())
		log.SetLevel(log.Level(*logLevel))
	}
	app.Action = mainCmd
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func mainCmd() {
	delimsParsed := strings.Split(*delimiters, ",")
	if len(delimsParsed) != 2 {
		log.Fatalln("incorrect delimiters specification:", *delimiters)
	}
	srcAbsPath, err := filepath.Abs(*srcDir)
	if err != nil {
		log.Fatalln("cannot get absolute path for:", *srcDir)
	} else if !strings.HasSuffix(srcAbsPath, "/") {
		srcAbsPath += "/"
	}
	loader, err := NewTemplateLoader(
		[]string{*srcDir},
		&TemplateLoaderOptions{
			ModePrefix: *prefix,
			LeftDelim:  delimsParsed[0],
			RightDelim: delimsParsed[1],
		})
	if err != nil {
		log.Fatalln(err)
	}
	c := NewTemplateContext()
	for _, source := range *contextSources {
		parts := strings.Split(source, "=")
		if len(parts) != 2 {
			err := fmt.Errorf("incorrect context source specification: %s", source)
			log.Fatalln(err)
		}
		name := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[1])
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalln(err)
		}
		sourceExt := filepath.Ext(path)
		if sourceExt == ".json" {
			if err := c.LoadFromJSON(name, data); err != nil {
				err = fmt.Errorf("error loading %s: %v", path, err)
				log.Fatalln(err)
			}
		} else if sourceExt == ".yaml" || sourceExt == ".yml" {
			if err := c.LoadFromYAML(name, data); err != nil {
				err = fmt.Errorf("error loading %s: %v", path, err)
				log.Fatalln(err)
			}
		} else {
			err := fmt.Errorf("unsupported Context source format: %s", sourceExt)
			log.Fatalln(err)
		}
	}
	if err := c.LoadEnvVars(); err != nil {
		log.Fatalln(err)
	}
	if err := c.LoadOsVars(); err != nil {
		log.Fatalln(err)
	}

	var verbatimActions Queue
	if err := loader.ForEachSource(TemplateModeVerbatim, func(source string) error {
		relativePath := strings.TrimPrefix(source, srcAbsPath)
		target := filepath.Join(*dstDir, relativePath)
		verbatimActions = append(verbatimActions, CopyFileAction(target, source))
		return nil
	}); err != nil {
		log.Fatalln(err)
	}
	if *dryRun {
		fmt.Println(verbatimActions.Description("Verbatim Files"))
	}

	var singularActions Queue
	if err := loader.RenderEachTemplate(TemplateModeSingle, func(tpl *template.Template, source string) error {
		relativePath := strings.TrimPrefix(source, srcAbsPath)
		contents, err := renderTemplate(tpl, source, c)
		if err != nil {
			err = fmt.Errorf("template rendering failed for %s: %v", relativePath, err)
			log.Fatalln(err)
		}
		target := filepath.Join(*dstDir, relativePath)
		if info, err := os.Stat(target); os.IsNotExist(err) {
			singularActions = append(singularActions, CreateNewFileAction(target, contents))
		} else if info.IsDir() {
			log.Fatalln("target is a directory:", target)
		} else {
			singularActions = append(singularActions, OverwriteFileAction(target, contents))
		}
		return nil
	}); err != nil {
		err = fmt.Errorf("tempate validation failed: %v", err)
		log.Fatalln(err)
	}
	if *dryRun {
		fmt.Println(singularActions.Description("Single Templates"))
	}

	if *dryRun {
		return
	}
	actionQueue := NewQueue(
		NewDirAction(*dstDir),
	)
	actionQueue = append(actionQueue, verbatimActions...)
	actionQueue = append(actionQueue, singularActions...)
	ts := time.Now()
	if !actionQueue.Exec() {
		log.Fatalln("failed in", time.Since(ts))
	}
	log.Infoln("done in", time.Since(ts))
}

func renderTemplate(tpl *template.Template, source string, c TemplateContext) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := tpl.ExecuteTemplate(buf, filepath.Base(source), c); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	return data, nil
}
