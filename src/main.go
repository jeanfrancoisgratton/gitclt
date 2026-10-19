package main

import (
	"fmt"
	"gitclt/cmd"
	"os"
	"path/filepath"
)

func main() {
	var err error
	CurrentWorkingDir := ""
	// Whatever happens, we need to preserve the current pwd, and restore it on exit, however the software exits
	if CurrentWorkingDir, err = os.Getwd(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// We need to create a configuration directory. This is a per-user config dir
	if err = os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".config", "JFG", "gitclt"), os.ModePerm); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cmd.Execute()

	// Software execution is complete, let's get the hell outta here
	_ = os.Chdir(CurrentWorkingDir)
}
