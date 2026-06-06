package main

import "os"

func main() {
	if len(os.Args) <= 1 {
		// No arguments → launch GUI
		runGUI()
		return
	}
	// Arguments present → CLI mode (cobra)
	Execute()
}
