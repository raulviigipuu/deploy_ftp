package uploader

import (
	"crypto/tls"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/raulviigipuu/deploy_ftp/internal/config"
	"github.com/secsy/goftp"

	logx "github.com/raulviigipuu/deploy_ftp/internal/logx"
)

// ========================
// Test Connection
// ========================
func TestConnection(cfg *config.Config) error {
	client, err := dialFTP(cfg)
	if err != nil {
		return fmt.Errorf("❌ Connection failed: %v", err)
	}
	defer client.Close()

	// Try listing root directory as a test
	if _, err := client.ReadDir("/"); err != nil {
		return fmt.Errorf("❌ Could not list root directory: %v", err)
	}

	return nil
}

// ========================
// Upload All Files
// ========================
func UploadAll(cfg *config.Config, dryRun bool) error {
	client, err := dialFTP(cfg)
	if err != nil {
		return fmt.Errorf("❌ FTP connection failed: %v", err)
	}
	defer client.Close()

	for _, entry := range cfg.Map {
		remotePath := normalizeRemotePath(entry.Remote)
		logx.Info(fmt.Sprintf("📁 Uploading: %s → %s", entry.Local, remotePath))

		// Determine directories that would be created
		if dryRun {
			dirsToCreate, err := findMissingRemoteDirs(client, path.Dir(remotePath))
			if err != nil {
				logx.Error(fmt.Sprintf("⚠️ Dry-run: Could not evaluate remote dirs for %s: %v", remotePath, err))
				continue
			}
			for _, dir := range dirsToCreate {
				logx.Info(fmt.Sprintf("📂 Dry-run: would create remote directory: %s", dir))
			}
			logx.Info(fmt.Sprintf("🔍 Dry-run: would upload %s → %s", entry.Local, remotePath))
			continue
		}

		// Ensure remote directory structure exists
		if err := ensureRemoteDirStructure(client, path.Dir(remotePath)); err != nil {
			logx.Error(fmt.Sprintf("⚠️ Could not create directory %s: %v", path.Dir(remotePath), err))
			continue
		}

		// Open local file
		file, err := os.Open(entry.Local)
		if err != nil {
			logx.Error(fmt.Sprintf("❌ Failed to open local file: %v", err))
			continue
		}

		// Upload file
		err = client.Store(remotePath, file)
		file.Close()
		if err != nil {
			logx.Error(fmt.Sprintf("❌ Upload failed: %v", err))
			continue
		}

		logx.Info("✅ Upload successful")
	}

	return nil
}

// ========================
// Helper: Connect
// ========================
func dialFTP(cfg *config.Config) (*goftp.Client, error) {
	config := goftp.Config{
		User:               cfg.FTPUser,
		Password:           cfg.FTPPass,
		TLSConfig:          &tls.Config{InsecureSkipVerify: true}, // allow self-signed certs
		TLSMode:            goftp.TLSExplicit,
		ConnectionsPerHost: 1,
		Timeout:            10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", cfg.FTPHost, cfg.FTPPort)
	return goftp.DialConfig(config, address)
}

// ========================
// Helper: Ensures that remote dirs exists
// ========================
func ensureRemoteDirStructure(client *goftp.Client, remoteDir string) error {
	parts := strings.Split(remoteDir, "/")
	curr := "/"
	for _, part := range parts {
		if part == "" {
			continue
		}
		curr = path.Join(curr, part)
		_, err := client.Mkdir(curr)
		if err == nil {
			logx.Info(fmt.Sprintf("📂 Created remote directory: %s", curr))
		}
	}
	return nil
}

// ========================
// Helper: Some insurance for path format
// ========================
func normalizeRemotePath(p string) string {
	p = path.Clean("/" + p)           // Clean handles ., .., and redundant slashes
	return strings.TrimSuffix(p, "/") // remove trailing slash
}

// ========================
// Helper: Check the remote dirs
// ========================
func findMissingRemoteDirs(client *goftp.Client, targetDir string) ([]string, error) {
	var missing []string
	currPath := "/"

	parts := strings.Split(path.Clean("/"+targetDir), "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		currPath = path.Join(currPath, part)
		_, err := client.Stat(currPath)
		if err != nil {
			missing = append(missing, currPath)
		}
	}

	return missing, nil
}
