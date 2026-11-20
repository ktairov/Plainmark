// Command buildapp reserves the next Plainmark version and runs MyGo.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

const configPath = "mygo.json"

var versionLine = regexp.MustCompile(`(?m)^[\t ]*"version"[\t ]*:[\t ]*"(\d+\.\d+\.(\d+))"[\t ]*,?[\t ]*\r?$`)

func main() {
	if err := build(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Build:", err)
		os.Exit(1)
	}
}

func build(args []string) error {
	const lockPath = ".build-number.lock"
	if err := os.Mkdir(lockPath, 0700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("another build holds %s", lockPath)
		}
		return err
	}
	defer os.Remove(lockPath)

	config, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var current struct{ Version string }
	if err := json.Unmarshal(config, &current); err != nil {
		return err
	}
	match := versionLine.FindSubmatchIndex(config)
	if match == nil || string(config[match[2]:match[3]]) != current.Version {
		return fmt.Errorf("%s must contain a numeric major.minor.build version", configPath)
	}
	firstDigit, end := match[4], match[5]
	number, err := strconv.ParseUint(string(config[firstDigit:end]), 10, 32)
	if err != nil || number == 1<<32-1 {
		return fmt.Errorf("build number out of range in %s", configPath)
	}
	next := strconv.FormatUint(number+1, 10)
	updated := string(config[:firstDigit]) + next + string(config[end:])
	// Reserve before invoking MyGo. A partial multi-platform build may already
	// have produced an artifact, so a failed attempt cannot reuse the number.
	if err := os.WriteFile(lockPath+"/mygo.json", []byte(updated), 0644); err != nil {
		return err
	}
	defer os.Remove(lockPath + "/mygo.json")
	if err := os.Rename(lockPath+"/mygo.json", configPath); err != nil {
		return err
	}
	fmt.Printf("Building Plainmark %s%s\n", config[match[2]:firstDigit], next)

	cmd := exec.Command("go", append([]string{"run", "github.com/egoist/mygo/cmd/mygo@v0.7.4", "build"}, args...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
