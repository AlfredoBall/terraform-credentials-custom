package log

import (
	"fmt"
	"os"
)

var green = "\033[32m"
var reset = "\033[0m"

func Info(msg string) {
	fmt.Fprintln(os.Stderr, green+"[tfcred] "+msg+reset)
}

func Err(msg string) {
	fmt.Fprintln(os.Stderr, "[tfcred][error] "+msg)
}
