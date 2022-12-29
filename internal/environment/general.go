package environment

import (
	"bytes"
	"os/exec"
	"strings"
)

func ThisOSWindows() bool {

	var stderr bytes.Buffer
	defer stderr.Reset()

	var out bytes.Buffer
	defer out.Reset()

	cmd := exec.Command("cmd", "ver")
	cmd.Stdin = strings.NewReader("some input")
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return false
	}
	myOS := out.String()
	if strings.Contains(myOS, "Microsoft Windows") {
		return true
	}
	return false
}

func ParsCfgByte(res []byte) bytes.Buffer {

	var out bytes.Buffer
	configLines := strings.Split(string(res), "\n")
	for i := 0; i < len(configLines); i++ {

		if configLines[i] != "" {
			var strs string
			splitStr := strings.SplitAfterN(configLines[i], "// ", -1)
			if len(splitStr) != 0 {
				strs = strings.Replace(splitStr[0], "// ", "\n", -1)
				out.WriteString(strs)
			}
		}
	}
	return out
}
