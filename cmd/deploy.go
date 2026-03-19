package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var skipTest bool

func init() {
	deployCmd.Flags().BoolVar(&skipTest, "skip-test", false, "skip go test step")
	rootCmd.AddCommand(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Build, test, and deploy operator to k3s",
	RunE:  runDeploy,
}

func runCmd(desc, name string, args ...string) error {
	fmt.Printf("=== %s ===\n", desc)
	fmt.Printf("  $ %s %s\n", name, strings.Join(args, " "))
	c := exec.Command(name, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func runCmdEnv(desc string, env []string, name string, args ...string) error {
	fmt.Printf("=== %s ===\n", desc)
	fmt.Printf("  $ %s %s\n", name, strings.Join(args, " "))
	c := exec.Command(name, args...)
	c.Env = append(os.Environ(), env...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found: no go.mod in cwd or any parent directory")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	if err := os.Chdir(repoRoot); err != nil {
		return fmt.Errorf("chdir %s: %w", repoRoot, err)
	}

	// 1. build windows binary
	exe := "k3sc"
	if runtime.GOOS == "windows" {
		exe = "k3sc.exe"
	}
	if err := runCmd("building windows binary", "go", "build", "-o", exe, "."); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	// 2. tests
	if !skipTest {
		if err := runCmd("running tests", "go", "test", "./..."); err != nil {
			return fmt.Errorf("go test: %w", err)
		}
	} else {
		fmt.Println("=== skipping tests ===")
	}

	// 3. cross-compile linux binary
	if err := runCmdEnv("cross-compiling linux binary",
		[]string{"GOOS=linux", "GOARCH=amd64"},
		"go", "build", "-o", filepath.Join("image", "k3sc"), "."); err != nil {
		return fmt.Errorf("linux build: %w", err)
	}

	// 4. build container image via WSL
	mntRoot := strings.ReplaceAll(repoRoot, `\`, `/`)
	// convert C:\code\k3sc -> /mnt/c/code/k3sc
	if len(mntRoot) >= 2 && mntRoot[1] == ':' {
		mntRoot = "/mnt/" + strings.ToLower(mntRoot[:1]) + mntRoot[2:]
	}
	nerdctl := "sudo nerdctl --address /run/k3s/containerd/containerd.sock --namespace k8s.io"
	buildCmd := fmt.Sprintf("cd %s && %s build -t claude-agent:latest image/", mntRoot, nerdctl)
	if err := runCmd("building container image",
		"wsl", "-d", "Ubuntu-24.04", "--", "bash", "-c", buildCmd); err != nil {
		return fmt.Errorf("image build: %w", err)
	}

	// 5. restart operator
	kubectl := "sudo k3s kubectl"
	restartCmd := fmt.Sprintf("%s rollout restart deployment k3sc-operator -n claude-agents", kubectl)
	if err := runCmd("restarting operator",
		"wsl", "-d", "Ubuntu-24.04", "--", "bash", "-c", restartCmd); err != nil {
		return fmt.Errorf("rollout restart: %w", err)
	}

	// 6. wait for rollout
	waitCmd := fmt.Sprintf("%s rollout status deployment k3sc-operator -n claude-agents --timeout=60s", kubectl)
	if err := runCmd("waiting for rollout",
		"wsl", "-d", "Ubuntu-24.04", "--", "bash", "-c", waitCmd); err != nil {
		return fmt.Errorf("rollout status: %w", err)
	}

	fmt.Println("\n=== deploy complete ===")
	return nil
}
