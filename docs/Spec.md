# cargo
Cargo is a simple static site builder

It works by processing a nested folder of files and generating new files and folders.

Most files are copied verbatim. Some are processed. One has a twist ...

There are two types of processed files - "singles" and "collections": Their contents are processed as a golang text template.

Each template is executed by applying them to a data structure - called the Context. using the standard go template pkg. The files are not necessarily HTML - they may be anything textual (SVG, XML, Javascript).

One or more YAML/JSON files can be loaded into the context from CLI (and stdio?).

Single template files are prefixed with a _ underscore.

When the template is evaluated, the output file is copied without the prefix.

Collections ... contain a template expression embedded in the file or folder name.

One file goes in - many come out. for example: "{{friends.name}}-photo.png"

Specifically, the key/index of the matched collection (set/array) is interpolated into the output path filename.

So may write two files - "Viktor-photo.png" and "Anna-photo.png" - if we passed in a suitable context.

Folders are created as needed - take care if the substituted key contains a path separator :-)

As we iterate over the collection, we should add it the Current context.

We should load ENV and OS vars into the global context too.

And our twist ... if the contents of a templated file resolves to the empty string - then it is not output.

#### Command Line Example

The tool will need a simple CLI:

```
cargo ./site ./published
```

with some options:
```
--[Context Name]=<yaml/json file> # Values=helm-chart-values.yaml should render simple Helm templates
--delimiters={{}} # change to a matching set of delimiters for contents template interpolation
--prefix=$ # change the singular prefix to $ instead of _
--dry-run # validate only, don't output
--help
--version
```
For reference, look at:

https://golang.org/pkg/text/template/
https://golang.org/pkg/html/template/#Template.Delims
https://docs.helm.sh/chart_template_guide/#the-chart-template-developer-s-guide

