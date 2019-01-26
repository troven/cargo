package main

import (
	"bytes"
	"encoding/json"
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
var contextSources = app.StringsOpt("c context", nil,
	"Specify multiple context sources in format Name=<yaml/json file> (e.g. Values=helm-chart-values.yaml)")

var srcDir = app.StringArg("SRC", "", "Specify source files dir for your site.")
var dstDir = app.StringArg("DST", "published/", "Specify destination dir for your site publication.")

func main() {
	app.Spec = "[OPTIONS] SRC [DST]"
	app.Before = func() {
		if isDebug() {
			log.SetReportCaller(true)
		}
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
	rootContext := NewTemplateContext()
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
			if err := rootContext.LoadFromJSON(name, data); err != nil {
				err = fmt.Errorf("error loading %s: %v", path, err)
				log.Fatalln(err)
			}
		} else if sourceExt == ".yaml" || sourceExt == ".yml" {
			if err := rootContext.LoadFromYAML(name, data); err != nil {
				err = fmt.Errorf("error loading %s: %v", path, err)
				log.Fatalln(err)
			}
		} else {
			err := fmt.Errorf("unsupported Context source format: %s", sourceExt)
			log.Fatalln(err)
		}
	}
	if isDebug() {
		v, _ := json.MarshalIndent(rootContext, "", "\t")
		log.Debugln("Context:", string(v))
	}
	if err := rootContext.LoadEnvVars(); err != nil {
		log.Fatalln(err)
	}
	if err := rootContext.LoadOsVars(); err != nil {
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
		contents, err := renderTemplate(tpl, source, rootContext)
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

	var collectionActions Queue
	if err := loader.RenderEachTemplate(TemplateModeCollection,
		func(tpl *template.Template, source string) error {
			outputs, err := loader.RenderFilepath(rootContext, source)
			if err != nil {
				return err
			}
			for output, currentContext := range outputs {
				relativePath := strings.TrimPrefix(source, srcAbsPath)
				relativeOutput := strings.TrimPrefix(output, srcAbsPath)
				target := filepath.Join(*dstDir, relativeOutput)
				if tpl == nil {
					collectionActions = append(collectionActions, CopyFileAction(target, source))
					continue
				}
				contents, err := renderTemplate(tpl, source, currentContext)
				if err != nil {
					err = fmt.Errorf("template rendering failed for %s: %v", relativePath, err)
					log.Fatalln(err)
				}
				if info, err := os.Stat(target); os.IsNotExist(err) {
					collectionActions = append(collectionActions, CreateNewFileAction(target, contents))
				} else if info.IsDir() {
					log.Fatalln("target is a directory:", target)
				} else {
					collectionActions = append(collectionActions, OverwriteFileAction(target, contents))
				}
			}
			return nil
		}); err != nil {
		err = fmt.Errorf("tempate validation failed: %v", err)
		log.Fatalln(err)
	}
	if *dryRun {
		fmt.Println(collectionActions.Description("Collection Templates"))
	}

	if *dryRun {
		return
	}
	actionQueue := NewQueue(
		NewDirAction(*dstDir),
	)
	actionQueue = append(actionQueue, verbatimActions...)
	actionQueue = append(actionQueue, singularActions...)
	actionQueue = append(actionQueue, collectionActions...)
	ts := time.Now()
	if !actionQueue.Exec() {
		log.Fatalln("failed in", time.Since(ts))
	}
	log.Infoln("done in", time.Since(ts))
}

func isDebug() bool {
	return *logLevel >= 5
}

func renderTemplate(tpl *template.Template, source string, context TemplateContext) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := tpl.ExecuteTemplate(buf, filepath.Base(source), context); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	return data, nil
}
