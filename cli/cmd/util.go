package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pkg/errors"
)

var (
	red   = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
)

func isGitClean() (bool, error) {
	output, err := run("git status --porcelain", "failed to get git status")
	return len(output) == 0, err
}

func getGitHash() (string, error) {
	output, err := run("git rev-parse --short HEAD", "failed to get git hash")
	return strings.TrimSpace(output), err
}

func run(shellCommand, failure string) (string, error) {
	args := strings.Split(shellCommand, " ")
	c := exec.Command(args[0], args[1:]...)

	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out

	err := c.Run()
	output := out.String()

	if err != nil {
		var b strings.Builder
		b.WriteString(failure)
		b.WriteString("\n\n")
		b.WriteString("running:\n    ")
		b.WriteString(shellCommand)
		b.WriteString("\n\noutput:\n    ")
		b.WriteString(strings.ReplaceAll(output, "\n", "\n    "))

		if _, ok := err.(*exec.ExitError); ok {
			err = errors.New(b.String())
		} else {
			err = errors.WithMessage(err, b.String())
		}
	}

	return output, err
}

func wait(message string, f func() (string, error)) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Writer = os.Stdout
	s.Prefix = message + ": "
	s.Start()

	finished, err := f()
	if err == nil {
		s.FinalMSG = fmt.Sprintf("%s: %s\n", message, green("✓"))
		s.Stop()
		if finished != "" {
			fmt.Print(finished)
		}
	} else {
		s.FinalMSG = fmt.Sprintf("%s: %s\n", message, red("✗"))
		s.Stop()
		fmt.Printf("    %v\n", err)
		os.Exit(1)
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s %v\n", red("ERROR:"), err)
	os.Exit(1)
}

func fatalIf(err error, defers ...func()) {
	if err != nil {
		for _, f := range defers {
			f()
		}
		fatal(err)
	}
}
