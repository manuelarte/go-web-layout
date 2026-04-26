//go:build mage

//mage:multiline

// Set the general description you want to have displayed with mage -l here.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

const (
	AppName = "GoWebLayout"
	Pkg     = "github.com/manuelarte/go-web-layout"
	InfoPkg = Pkg + "/internal/info"
)

type LdFlags struct {
	Branch    string
	BuildTime string
	CommitID  string
	Version   string
}

// A build step that requires additional params, or platform specific steps for example
func Build() error {
	mg.Deps(Generate, Tidy)
	fmt.Println("Building...")
	ldFlags, err := flags()
	if err != nil {
		return err
	}

	ldflagsArg := fmt.Sprintf(
		"-X %s.Branch=%s -X %s.BuildTime=%s -X %s.CommitID=%s -X %s.Version=%s",
		InfoPkg, ldFlags.Branch, InfoPkg, ldFlags.BuildTime, InfoPkg, ldFlags.CommitID, InfoPkg, ldFlags.Version,
	)

	cmd := exec.Command(
		"go",
		"build",
		"-ldflags", ldflagsArg,
		"-o", AppName,
		"./cmd/go-web-layout/.",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	// Explicitly set executable permissions (0755)
	return os.Chmod(AppName, 0o755)
}

// Run
func Run() error {
	mg.Deps(Build)
	fmt.Println("Running...")

	// Safety check: ensure AppName is not a directory
	info, err := os.Stat(AppName)
	if err == nil && info.IsDir() {
		return fmt.Errorf("'%s' is a directory, not an executable. Check for naming conflicts", AppName)
	}

	// Use absolute path or clean relative path for execution
	binPath := "./" + AppName
	cmd := exec.Command(binPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Docker build image
func DockerBuild() error {
	fmt.Println("Building Docker image...")
	ldFlags, err := flags()
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"docker",
		"build",
		"--build-arg", "BRANCH="+ldFlags.Branch,
		"--build-arg", "BUILD_TIME="+ldFlags.BuildTime,
		"--build-arg", "COMMIT_ID="+ldFlags.CommitID,
		"--build-arg", "APP_VERSION=Docker"+ldFlags.Version,
		"-t", Pkg,
		".",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Docker run
func DockerRun() error {
	fmt.Println("Running Docker image...")
	mg.Deps(DockerBuild)
	cmd := exec.Command("docker", "run", "-p", "3001:3001", "-p", "3002:3002", Pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Update all deps.
func UpdateDeps() error {
	fmt.Println("Installing Deps...")
	cmd := exec.Command("go", "get", "-u")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Generate() error {
	fmt.Println("Generating code")
	cmd := exec.Command("go", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Println("Code generated")
	return nil
}

func Format() error {
	fmt.Println("Formatting...")
	cmd := exec.Command("golangci-lint", "fmt")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("Keep-sorted golangci.yml")
	cmd = exec.Command("keep-sorted", "./.golangci.yml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("buf format -w")
	cmd = exec.Command("buf", "format", "-w")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("Formatting complete")
	return nil
}

func Lint() error {
	mg.Deps(Format)
	fmt.Println("Linting...")
	fmt.Println("Running golangci-lint with --fix to automatically fix issues where possible")
	cmd := exec.Command("golangci-lint", "run", "--fix", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("buf lint")
	cmd = exec.Command("buf", "lint")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("Spectral lint")
	cmd = exec.Command("spectral", "lint", "./resources/openapi.yml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("Running hadolint")
	// Run hadolint via docker to avoid local binary issues (Access violation errors)
	cmdHadolint := exec.Command("docker", "run", "--rm", "-i", "hadolint/hadolint", "hadolint", "-")

	dockerfile, err := os.Open("Dockerfile")
	if err != nil {
		return fmt.Errorf("failed to open Dockerfile: %w", err)
	}
	defer dockerfile.Close()

	cmdHadolint.Stdin = dockerfile
	cmdHadolint.Stdout = os.Stdout
	cmdHadolint.Stderr = os.Stderr
	if err := cmdHadolint.Run(); err != nil {
		fmt.Printf("hadolint found issues: %v\n", err)
	}

	fmt.Println("Linting complete")
	return nil
}

// Test the project
func Test() error {
	fmt.Println("Testing...")
	cmd := exec.Command("go", "test", "--cover", "-timeout=300s", "-parallel=16", "-v", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Clean up after yourself
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll(AppName)
}

// Tidy
func Tidy() error {
	fmt.Println("Tidying...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// flags used to build the app
func flags() (LdFlags, error) {
	branchOut, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return LdFlags{}, err
	}
	buildTime := time.Now().Format("2006-01-02T15:04:05Z07:00")
	commitID, _ := exec.Command("git", "rev-list", "-1", "HEAD").Output()

	return LdFlags{
		Branch:    strings.TrimSpace(string(branchOut)),
		BuildTime: buildTime,
		CommitID:  strings.TrimSpace(string(commitID)),
		Version:   "LOCAL",
	}, nil
}

func Tools() error {
	fmt.Println("Installing tools...")
	cmd := exec.Command("go", "get", "-tool", "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	toInstall := []string{
		// keep-sorted start
		"github.com/bufbuild/buf/cmd/buf@v1.68.4",
		"github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4",
		"github.com/google/keep-sorted@v0.7.1",
		"github.com/sqlc-dev/sqlc/cmd/sqlc@latest",
		"go.uber.org/mock/mockgen@latest",
		// keep-sorted end
	}
	for _, tool := range toInstall {
		cmd := exec.Command("go", "install", tool)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
