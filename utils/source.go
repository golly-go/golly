package utils

import (
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var gollySourceDir string

func init() {
	_, file, _, _ := runtime.Caller(0)

	gollySourceDir = regexp.MustCompile(`utils.source\.go`).ReplaceAllString(file, "")
}

// FileWithLineNum return the file name and line number of the current file
func FileWithLineNum() string {
	// the second caller usually from gorm internal, so set i start from 2
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && ((!strings.HasPrefix(file, gollySourceDir) && !strings.Contains(file, "gorm.io")) || strings.HasSuffix(file, "_test.go")) {
			return file + ":" + strconv.FormatInt(int64(line), 10)
		}
	}

	return ""
}
