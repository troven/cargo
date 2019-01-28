## cargo by troven

<img src="docs/cargo.png" width="300px" />

Cargo is a tiny static file generator written in golang for elegance, efficiency and embedding. 

It works by processing a nested folder of files and generating new files and folders.

### Installation

If you want to build the executable on your own machine, you need to install Go first. Open [Downloads](https://golang.org/dl/) and get at least Go1.11+ for your platform. Or you can install using `brew` on macOS or using your distro package manager. Prefer most recent versions for better compatibility.

Doh! Not a golang ninja? [start here](GO_NOOBS.md)

After installing Go, you'll need to clone this repo. 

#### Public Branch

If repo is public already, just run this:

```
go get -u github.com/troven/cargo/cmd/cargo
```

#### Private Branch

For a private branch, you'll need to clone it manually:

```
mkdir -p $GOPATH/src/github.com/troven/cargo
cd $GOPATH/src/github.com/troven/cargo
git clone git@github.com:troven/cargo.git .
go get github.com/troven/cargo
```

### Building from source


After running `go get` on the root package name, the `cargo` executable will not be built yet. To build or install it, use the provided script:

```
make install # build and install executable into $GOPATH/bin
make build # simply build the executable into bin/
```

You can set up your OS to run arbitrary commands from `$GOPATH/bin` by exporting that path to `$PATH`:
```
export PATH=$PATH:$GOPATH/bin
```

Or you can always use `make build` to keep the resulting executable relative to current directory. 

### Cargo Run

```
$ cargo run -h

Usage: cargo run [OPTIONS] SRC [DST]

The Cargo run operation moves source files to the destination folder, processing the
template files it encounters.

Arguments:
  SRC                Specify source files dir for your site.
  DST                Specify destination dir for your site publication. (default "published/")

Options:
  -l, --log-level    Sets the log level [0 = no log, 5 = debug]. (default 4)
  -d, --dry-run      Do not modify filesystem, only print planned actions.
      --delimiters   Comma-seprated delimiters to scan in templates, left and right. (default "{{,}}")
      --prefix       Prefix in filenames to specify singular templates. (default "_")
  -c, --context      Specify multiple context sources in format Name=<yaml/json file> (e.g. Values=helm-chart-values.yaml)
```

### Examples

Check out the `[test](/test)` directory for a worked example:

```
test/
├── _templated.md
├── app_context.json
├── friends_context.yaml
├── subfolder
│   └── {{os.RuntimeVersion}}.png
├── verbatim.jpg
├── {{app.path}}.txt
├── {{friends.name}}_{{friends.age}}.txt
└── {{friends}}.txt
```

### 

Start your first Cargo run

```
cargo run --context App=test/app_context.json --context Friends=test/friends_context.yaml test/ published/
```

### How it works?

Most files are copied verbatim. Some are processed. One has a twist ...

There are two types of processed files - "singles" and "collections": Their contents are processed as a golang text template - [read a tutorial](https://blog.gopheracademy.com/advent-2017/using-go-templates/).

You can see working examples in the `./test/` folder. Or take a deeper peek and [learn more here](Templates.md)

Each template is executed by applying them to a data structure - called the Context. 

The files can be any type - including movies, audio, images, software, etc.

The templates can be any text file - HTML, SVG, XML, JSON, source code (like Java & Javascript).

One or more YAML/JSON files can be loaded into the Context from CLI using the `--context` option.

#### Single Templates

Single template files are prefixed with an `_` underscore.

When the template is evaluated, the output file is copied without the prefix.

#### Collection Templates

Collections are identified by special characters in the path name. 

A collection template contain a Collection Path in the file or folder name.

To add another "complication" - there two types of Collection Path - object paths and array paths. 

Both `{{app.path}}`, `{{friends.name}}` match a field within a nested object where as `{{friends}}` matches an array. 

#### Object Collection

Object paths are arguably, the simpliest and most powerful. The key/index of the matched collection (set/array) is interpolated into the output path filename.

This means you can build complex output filenames that depends on the Context.

Of course, folders are created as needed - even when the substituted key contains a path separator. In this case, the final path is computed from the original path with the new path embedded within.

As we iterate over the collection, we add it the `Current` context.

Both `{{app.path}}`, `{{friends.name}}` match a field within a nested object. In this

Mutliple files are generated matching the Collection Path.

```
...
{{app.path}}.txt
{{friends.name}}_{{friends.age}}.txt
{{friends}}.txt
```

So for collections, one file goes in - many may come out. So given the following "friends" Context:

```
cat ./test/friends_context.yaml

- { "Name": "Maxim", "Age": 26}
- { "Name": "Ivan", "Age": 25}
```

you might expect to generate

```
ls -1 ./published/
...
Ivan_25.txt
Maxim_26.txt
...
```

#### Array Collection

These are single files where-in the Current context is set to each item in the array.

#### Creating Templates

Templates are quite simple to create - they are simply text files. Whatever you can express in a text file can be easily converted into a template.

Cargo regularly builds websites and documentation (HTML/CSS), images and diagrams (SVG), data files (XML, JSON), software components (Java & Javascript) and integration tests (BDD/TDD).

When Cargo encounters a template file, it loads the Context then interprets the template. The template instructs Cargo how to translate the contents into the final product. 

You can see lots of examples in the `./test/` folder. Or [learn more here](Templates.md)

#### Environment

We also load `ENV` and `OS` vars into the global context too.

All `ENV` variables are available as `{{ .Env.FOO }}` in the template.

#### Special template twist

If the contents of a templated file resolves to the empty string - then it is not output.

### Makefile Usage

```
$ make help
Management commands for cargo:

Usage:
    make build           Compile the project.
    make install         Compile the project and install cargo to $GOPATH/bin.
    make get-deps        runs dep ensure, mostly used for CI.
    make build-docker    Compile optimized for Docker linux (scratch / alpine).
    make image           Build final docker image with just the go binary inside
    make tag             Tag image created by package with latest, git commit and version
    make test            Run tests on the source code of the project.
    make test-run        Run tests using cargo executable (must be available in $PATH).
    make push            Push tagged images to registry
    make clean           Clean the directory tree.

```

