package utils

import (
	"fmt"
	"os"
)

func ErrorExit(message string, args ...any)  {
  fmt.Fprintf(os.Stderr, message, args...)
  os.Exit(1)
}
