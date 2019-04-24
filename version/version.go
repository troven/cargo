package version

import (
	"fmt"
	"runtime"
)

var GitCommit string

const Version = "0.2.1"

var BuildDate = ""

var GoVersion = runtime.Version()

var OSArch = fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
