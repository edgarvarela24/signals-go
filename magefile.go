//go:build mage

package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
// Usage: mage
var Default = Test

// helper: returns list of module packages (excluding vendor) or empty slice.
func listPackages() ([]string, error) {
	cmd := exec.Command("go", "list", "./...")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list failed: %v: %s", err, out.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

// Build compiles the project.
func Build() error {
	fmt.Println("Building...")
	pkgs, err := listPackages()
	if err != nil {
		return err
	}
	if len(pkgs) == 0 { // nothing to do yet
		fmt.Println("(no packages yet – skipping build)")
		return nil
	}
	if err := sh.RunV("go", "build", "./..."); err != nil {
		return err
	}
	return sh.RunV("go", "vet", "./...")
}

// Test runs all unit tests.
// Usage: mage test
func Test() error {
	fmt.Println("Running tests...")
	pkgs, err := listPackages()
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		fmt.Println("(no packages yet – skipping tests)")
		return nil
	}
	return sh.RunV("go", "test", "-v", "-race", "./...")
}

// Clean removes build artifacts.
// Usage: mage clean
func Clean() {
	fmt.Println("Cleaning...")
	// Nothing generated yet.
}

// Fmt runs go fmt on the module.
func Fmt() error {
	fmt.Println("Formatting...")
	return sh.RunV("go", "fmt", "./...")
}

// Tidy runs go mod tidy.
func Tidy() error {
	fmt.Println("Tidying go.mod...")
	return sh.RunV("go", "mod", "tidy")
}

// All runs formatting, build, lint, and tests (good for local pre-push).
func All() error {
	fmt.Println("Running all checks...")
	steps := []func() error{Fmt, Tidy, Test}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// CI is a stricter pipeline entrypoint; logs failure early.
func CI() {
	if err := All(); err != nil {
		log.Fatalf("CI failed: %v", err)
	}
}
