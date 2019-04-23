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
	"github.com/troven/cargo/version"
)

var app = cli.App("cargo", "Cargo builds things.")

func main() {
	app.Command("run", "The Cargo run operation moves source files to the destination folder, "+
		"processing the template files it encounters.", runCmd)
	app.Command("version", "Prints the version", versionCmd)
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func runCmd(cmd *cli.Cmd) {
	logLevel := cmd.IntOpt("l log-level", 3, "Sets the log level [0 = no log, 5 = debug].")
	dryRun := cmd.BoolOpt("d dry-run", false, "Do not modify filesystem, only print planned actions.")
	delimiters := cmd.StringOpt("delimiters", "{{,}}", "Comma-seprated delimiters to scan in templates, left and right.")
	modePrefix := cmd.StringOpt("prefix", "_", "Prefix in filenames to specify singular templates.")
	contextSources := cmd.StringsOpt("c context", nil,
		"Specify multiple context sources in format Name=<yaml/json file> (e.g. Values=helm-chart-values.yaml)")

	srcDir := cmd.StringArg("SRC", "cargo/", "Specify source files dir for your site.")
	dstDir := cmd.StringArg("DST", "build/", "Specify destination dir for your site publication.")

	cmd.Spec = "[OPTIONS] SRC [DST]"
	cmd.Before = func() {
		if isDebug(logLevel) {
			log.SetReportCaller(true)
		}
		log.SetLevel(log.Level(*logLevel))
	}
	cmd.Action = func() {
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
				ModePrefix: *modePrefix,
				LeftDelim:  delimsParsed[0],
				RightDelim: delimsParsed[1],
			})
		if err != nil {
			log.Fatalln(err)
		}
		rootContext := NewTemplateContext()
		var hasGlobal bool
		for _, source := range *contextSources {
			parts := strings.Split(source, "=")
			if len(parts) != 2 {
				data, err := ioutil.ReadFile(parts[0])
				if err == nil {
					if err == rootContext.LoadGlobalFromYAML(data) {
						hasGlobal = true
						continue
					}
				}
				err = fmt.Errorf("incorrect context source specification: %s", source)
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
		if !hasGlobal {
			data, err := ioutil.ReadFile("cargo.yaml")
			if err == nil {
				if err == rootContext.LoadGlobalFromYAML(data) {
					hasGlobal = true
				}
			}
		}
		if isDebug(logLevel) {
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
			verbatimActions = append(verbatimActions, CopyFileAction(*dstDir, target, source))
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
			} else if isEmptyOrWhitespace(contents) {
				return nil
			}
			target := filepath.Join(*dstDir, relativePath)
			target = removeModePrefix(target, *modePrefix)
			if info, err := os.Stat(target); os.IsNotExist(err) {
				singularActions = append(singularActions, CreateNewFileAction(*dstDir, target, contents))
			} else if info.IsDir() {
				log.Fatalln("target is a directory:", target)
			} else {
				singularActions = append(singularActions, OverwriteFileAction(*dstDir, target, contents))
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
					target = removeModePrefix(target, *modePrefix)
					if tpl == nil {
						collectionActions = append(collectionActions, CopyFileAction(*dstDir, target, source))
						continue
					}
					contents, err := renderTemplate(tpl, source, currentContext)
					if err != nil {
						err = fmt.Errorf("template rendering failed for %s: %v", relativePath, err)
						log.Fatalln(err)
					} else if isEmptyOrWhitespace(contents) {
						continue
					}
					if info, err := os.Stat(target); os.IsNotExist(err) {
						collectionActions = append(collectionActions, CreateNewFileAction(*dstDir, target, contents))
					} else if info.IsDir() {
						log.Fatalln("target is a directory:", target)
					} else {
						collectionActions = append(collectionActions, OverwriteFileAction(*dstDir, target, contents))
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
			NewDirAction(*dstDir, *dstDir),
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
}

func removeModePrefix(path, modePrefix string) string {
	name := filepath.Base(path)
	dir := filepath.Dir(path)
	name = strings.TrimPrefix(name, modePrefix)
	return filepath.Join(dir, name)
}

func versionCmd(cmd *cli.Cmd) {
	cmd.Action = func() {
		ver := fmt.Sprintf("cargo %s", version.Version)
		if len(version.GitCommit) > 0 {
			ver = fmt.Sprintf("cargo %s (commit %s)", version.Version, version.GitCommit)
		}
		fmt.Println(ver)
	}
}

func isDebug(logLevel *int) bool {
	return *logLevel >= 5
}

func isEmptyOrWhitespace(contents []byte) bool {
	if len(contents) == 0 {
		return true
	} else if len(contents) > 512 {
		// we set a reasonable bounds to file size
		return false
	}
	for _, r := range contents {
		if r != '\n' && r != '\r' && r != ' ' {
			return false
		}
	}
	return true
}

func renderTemplate(tpl *template.Template, source string, context TemplateContext) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := tpl.ExecuteTemplate(buf, filepath.Base(source), context); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	return data, nil
}
