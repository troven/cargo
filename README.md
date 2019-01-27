## cargo by troven

<img src="docs/cargo.png" width="300px" />

Cargo is a tiny static file generator written in golang for elegance, efficiency and relevance. 

It works by processing a nested folder of files and generating new files and folders.

### Installation

```
go get -u github.com/troven/cargo/cmd/cargo
```

Doh! Not a golang ninja? [start here](GO_NOOBS.md)

### Usage

```
$ cargo -h

Usage: cargo [COMMAND] [OPTIONS] SRC [DST]

Commands:

  run                Process each file/template in SRC and write them to DST

Arguments:
  SRC                Specify source files dir for your site.
  DST                Specify destination dir for your site publication. (default "published/")

Options:
  -l, --log-level    Sets the log level [0 = no log, 5 = debug]. (default 4)
  -d, --dry-run      Do not modify filesystem, only print planned actions.
      --delimiters   Comma-seprated delimiters to scan in templates, left and right. (default "{{,}}")
      --prefix       Prefix in filenames to specify singluar templates. (default "_")
  -c, --context      Specify multiple context sources in format Name=<yaml/json file> (e.g. Values=helm-chart-values.yaml)
```

### Examples

See the [test](/test) directory for a worked example:

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

### Run test cases
Use `make test` to check everything is working smoothly:

```
$ make test
cargo run\
        --context App=test/app_context.json \
        --context Friends=test/friends_context.yaml \
        test/ published/

INFO[0000] action#1: new dir [dst] if not exists
INFO[0000] action#2: copy file [dst]/app_context.json
INFO[0000] action#3: copy file [dst]/friends_context.yaml
INFO[0000] action#4: copy file [dst]/verbatim.jpg
INFO[0000] action#5: new file [dst]/_templated.md size=280 B (no overwrite)
INFO[0000] action#6: copy file [dst]/subfolder/go1.11.2.png
INFO[0000] action#7: new file [dst]/subfolder2/file.txt size=51 B (no overwrite)
INFO[0000] action#8: new file [dst]/Maxim_26.txt size=41 B (no overwrite)
INFO[0000] action#9: new file [dst]/Ivan_25.txt size=40 B (no overwrite)
INFO[0000] action#10: new file [dst]/friends.txt size=29 B (no overwrite)
INFO[0000] done in 3.763994ms
```

Documentation is WIP.
