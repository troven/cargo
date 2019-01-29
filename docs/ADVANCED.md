## cargo by troven

<img src="docs/cargo.png" width="300px" />

### Advanced fields

Cargo injects the context with variables it could gather from Environment and OS constants.

Examples of referencing Env:

* {{ .Env.FOO_BAR }} exported as `FOO_BAR`
* {{ .Env.PWD }}
* any env variable actually

Fields exported from OS:

* {{ OS.PathSeparator }}, example: / (or \ on Windows)
* {{ OS.PathListSeparator }}, example: ;
* {{ OS.WorkDir }}, example: /home/user/dev/k8s/configs
* {{ OS.Hostname }}, example: death.star
* {{ OS.Executable }}, example: cargo
* {{ OS.RuntimeVersion }}, example: go1.11.2.png
* {{ OS.Arch }}, example: amd64
* {{ OS.OS }}, example: linux
