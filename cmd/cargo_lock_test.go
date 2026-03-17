package cmd

import (
	"runtime"
	"testing"
)

// TestKillEndlessExeNoopOnLinux verifies killEndlessExe is a no-op on non-Windows
// platforms. This test would fail to compile if the function were removed.
func TestKillEndlessExeNoopOnLinux(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping Linux no-op test on Windows")
	}
	// must not panic or block; on Linux this returns immediately
	killEndlessExe()
}

// TestKillEndlessExeWindowsNoProcess verifies killEndlessExe is a no-op when
// endless.exe is not running on Windows (taskkill exits with non-zero, no sleep).
func TestKillEndlessExeWindowsNoProcess(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	// endless.exe is not expected to be running in CI; must not error or panic
	killEndlessExe()
}
