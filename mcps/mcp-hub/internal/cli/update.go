package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// CurrentVersion is set from main.go
var CurrentVersion string

// UpdateCmd is the update subcommand that updates hub to the latest version
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update mh to the latest version",
	Long: `Check for updates and upgrade mh to the latest version.

Examples:
  # Check and update to latest
  mh update

  # Check only (dry run)
  mh update --check`,
	RunE: runUpdate,
}

func init() {
	UpdateCmd.Flags().Bool("check", false, "only check for updates, don't install")
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")

	// Fetch all releases and filter by name containing "Hub"
	latest, err := getLatestHubRelease()
	if err != nil {
		return err
	}

	if latest == nil {
		fmt.Println("No Hub releases found")
		return nil
	}

	// Compare versions
	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	currentVersion := strings.TrimPrefix(CurrentVersion, "v")

	if currentVersion == "dev" {
		fmt.Printf("Development version, latest available: %s\n", latest.TagName)
		if checkOnly {
			return nil
		}
	} else if !isNewerVersion(latestVersion, currentVersion) {
		fmt.Printf("Already up to date (%s)\n", CurrentVersion)
		return nil
	} else {
		fmt.Printf("New version available: %s (current: %s)\n", latest.TagName, CurrentVersion)
		if checkOnly {
			return nil
		}
	}

	// Find matching asset
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	assetName := fmt.Sprintf("hub_%s_%s_%s.%s", latestVersion, runtime.GOOS, runtime.GOARCH, ext)

	var downloadURL string
	for _, a := range latest.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no asset found for %s/%s (looking for %s)", runtime.GOOS, runtime.GOARCH, assetName)
	}

	// Download and replace
	fmt.Printf("Downloading %s...\n", assetName)
	if err := downloadAndReplace(downloadURL, ext); err != nil {
		return err
	}

	fmt.Printf("✓ Successfully updated to %s\n", latest.TagName)
	return nil
}

func getLatestHubRelease() (*ghRelease, error) {
	resp, err := http.Get("https://api.github.com/repos/vaayne/cc-plugins/releases")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	// Find latest release with name starting with "Hub"
	for _, r := range releases {
		if strings.HasPrefix(r.Name, "Hub ") {
			return &r, nil
		}
	}

	return nil, nil
}

// isNewerVersion returns true if latest is newer than current
// Simple semver comparison (major.minor.patch)
func isNewerVersion(latest, current string) bool {
	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for i := range 3 {
		var l, c int
		if i < len(latestParts) {
			fmt.Sscanf(latestParts[i], "%d", &l)
		}
		if i < len(currentParts) {
			fmt.Sscanf(currentParts[i], "%d", &c)
		}
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}
	return false
}

func downloadAndReplace(url, ext string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Read entire response for zip (needs random access) or stream for tar.gz
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read download: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Extract binary
	var binaryData []byte
	binaryName := "hub"
	if runtime.GOOS == "windows" {
		binaryName = "hub.exe"
	}

	if ext == "zip" {
		binaryData, err = extractFromZip(data, binaryName)
	} else {
		binaryData, err = extractFromTarGz(data, binaryName)
	}
	if err != nil {
		return err
	}

	// Write to temp file, then rename (atomic)
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, binaryData, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	// On Windows, rename the old binary first
	if runtime.GOOS == "windows" {
		old := exe + ".old"
		os.Remove(old) // ignore error
		if err := os.Rename(exe, old); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("failed to backup old binary: %w", err)
		}
	}

	if err := os.Rename(tmp, exe); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	return nil
}

func extractFromTarGz(data []byte, binaryName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}
		if hdr.Name == binaryName || strings.HasSuffix(hdr.Name, "/"+binaryName) {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %s not found in archive", binaryName)
}

func extractFromZip(data []byte, binaryName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	for _, f := range zr.File {
		if f.Name == binaryName || strings.HasSuffix(f.Name, "/"+binaryName) {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open zip entry: %w", err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("binary %s not found in archive", binaryName)
}
