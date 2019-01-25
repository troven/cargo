package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jawher/mow.cli"
	log "github.com/sirupsen/logrus"
)

var app = cli.App("cargo", "Cargo is a simple static site builder.")
var logLevel = app.Int("l log-level", 4, "Sets the log level [0 = no log, 5 = debug].")
var agreeAll = app.BoolOpt("y yes", false, "Agree to all prompts automatically.")
var delimiters = app.StringOpt("delimiters", "{{,}}", "Comma-seprated delimiters to scan in templates, left and right.")
var prefix = app.StringOpt("prefix", "_", "Prefix in filenames to specify singluar templates.")
var contextSources = app.StringsOpt("context", nil,
	"Specify multiple context sources in format Name=<yaml/json file> (e.g. Values=helm-chart-values.yaml")

var srcDir = app.StringArg("SRC", "", "Specify source files dir for your site.")
var dstDir = app.StringArg("DST", "published", "Specify destination dir for your site publication.")

func main() {
	app.Spec = "SRC [DST]"
	app.Before = func() {
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
	loader, err := NewTemplateLoader(*srcDir, &TemplateLoaderOptions{
		ModePrefix: *prefix,
		LeftDelim:  delimsParsed[0],
		RightDelim: delimsParsed[1],
	})
	if err != nil {
		log.Fatalln("[ERR]", err)
	}
	for _, source := range *contextSources {
		parts := strings.Split(source, "=")
	}

	actionQueue := NewQueue(
		NewDirAction(*dstDir),
	)
	c := NewTemplateContext()

	// if ctx.RepoEnabled {
	// 	actionQueue = append(actionQueue,
	// 		CreateNewFileAction(filepath.Join(basePath, filePrefix+"data.go"), ctx.RenderInto(dataTemplate)),
	// 	)
	// }

	fmt.Println(actionQueue.Description())
	agree := *agreeAll
	if !agree {
		agree = cliConfirm("Are you sure to apply these changes?")
		if !agree {
			log.Println("Action cancelled.")
			return
		}
	}
	ts := time.Now()
	if !actionQueue.Exec() {
		log.Println("Failed in", time.Since(ts))
		return
	}
	log.Println("Done in", time.Since(ts))
}
