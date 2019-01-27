## First time using Go

#### For OS X:

Setup your machine for Go development - it's easier that way.

```
export GOPATH="${HOME}/.go"
export GOROOT="$(brew --prefix golang)/libexec"
export PATH="$PATH:${GOPATH}/bin:${GOROOT}/bin"
test -d "${GOPATH}" || mkdir "${GOPATH}"
test -d "${GOPATH}/src/github.com" || mkdir -p "${GOPATH}/src/github.com"

brew install go
go get golang.org/x/tools/cmd/godoc
go get github.com/golang/lint/golint
```

### 1) Download from internet

go get -u github.com/troven/cargo/cmd/cargo

### 2) Build from source

mkdir -p $GOPATH/github.com/troven/
cd $GOPATH/github.com/troven/
git clone git@github.com:troven/cargo.git
go build .
cp ./cargo /usr/local/bin





