## cargo by troven

<img src="cargo.png" width="300px" />

### Run test cases
Use `make test` to check everything is working smoothly:

```
$ make test
cargo run\
        --context test/cargo.yaml \
        --context App=test/app.json \
        --context Friends=test/friends.yaml \
        test/cargo test/build
INFO[0000] action#1: new dir [dst] if not exists
INFO[0000] action#2: copy file [dst]/cargo.png
INFO[0000] action#3: new file [dst]/cargo.md size=135 B (no overwrite)
INFO[0000] action#4: new file [dst]/templated.md size=280 B (no overwrite)
INFO[0000] action#5: copy file [dst]/subfolder/go1.11.2.png
INFO[0000] action#6: new file [dst]/subfolder2/file.txt size=51 B (no overwrite)
INFO[0000] action#7: new file [dst]/Ivan_25.txt size=40 B (no overwrite)
INFO[0000] action#8: new file [dst]/Maxim_26.txt size=41 B (no overwrite)
INFO[0000] action#9: new file [dst]/friends.txt size=29 B (no overwrite)
INFO[0000] done in 2.63111ms
```

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
    make test-write      Run tests using cargo executable, writes files.
    make push            Push tagged images to registry
    make clean           Clean the directory tree.
```

