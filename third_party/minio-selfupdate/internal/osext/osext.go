package osext

import "os"

func Executable() (string, error) { return os.Executable() }
