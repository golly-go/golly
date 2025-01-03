package utils

import (
	"runtime"
	"strconv"
	"strings"
)

var gollySourceDir = "github.com/golly-go/"

// FileWithLineNum return the file name and line number of the current file
// func FileWithLineNum() string {
// 	// the second caller usually from gorm internal, so set i start from 2
// 	for i := 2; i < 15; i++ {
// 		_, file, line, ok := runtime.Caller(i)
// 		if ok && ((!strings.HasPrefix(file, gollySourceDir) && !strings.Contains(file, "gorm.io")) || strings.HasSuffix(file, "_test.go")) {
// 			return file + ":" + strconv.FormatInt(int64(line), 10)
// 		}
// 	}

// 	return ""
// }

// FileWithLineNum return the file name and line number of the current file

// FileWithLineNum returns the file name and line number of the current file
func FileWithLineNum() string {
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)

		// Exclude files from golly source directory except for test files
		if ok && !(strings.HasPrefix(file, gollySourceDir) && !strings.HasSuffix(file, "_test.go")) {
			return file + ":" + strconv.Itoa(line)
		}
	}
	return "unknown"
}
