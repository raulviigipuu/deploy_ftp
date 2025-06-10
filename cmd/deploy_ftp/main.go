package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/raulviigipuu/deploy_ftp/internal/config"
	"github.com/raulviigipuu/deploy_ftp/internal/logx"
	"github.com/raulviigipuu/deploy_ftp/internal/uploader"
)

var Version = "dev" // default if not overridden at build time

func main() {

	logx.Init(nil)

	envPath := flag.String("env-path", "", "Path to .env file (default: ./.env)")
	mapPath := flag.String("map", "", "Path to upload_map.json (default: ./upload_map.json)")
	verifyOnly := flag.Bool("verify", false, "Only test FTP connection, do not upload")
	dryRun := flag.Bool("dry-run", false, "Simulate FTP upload without transferring files")
	versionFlag := flag.Bool("v", false, "Show version and exit")
	helpFlag := flag.Bool("h", false, "Show help and exit")

	flag.Parse()

	// Help
	if *helpFlag {
		printHelp()
		return
	}

	// Version
	if *versionFlag {
		logx.Info(fmt.Sprintf("deploy_ftp version: %s", Version))
		return
	}

	// Conf
	cfg, err := config.Load(*envPath, *mapPath)
	if err != nil {
		logx.FatalErr(err)
	}

	logx.Info("✅ Configuration loaded successfully:")
	logx.Info(fmt.Sprintf("FTP Host: %s", cfg.FTPHost))
	logx.Info(fmt.Sprintf("TLS Enabled: %v", cfg.UseTLS))
	logx.Info(fmt.Sprintf("Dry-run: %v", *dryRun))
	logx.Info(fmt.Sprintf("Verify: %v", *verifyOnly))

	if *verifyOnly {
		if err := uploader.TestConnection(cfg); err != nil {
			logx.FatalErr(err)
		}
		logx.Info("✅ FTP connection successful!")
		return
	}

	// Files
	logx.Info("📦 Upload plan:")
	missing := 0
	for _, entry := range cfg.Map {
		if _, err := os.Stat(entry.Local); err == nil {
			logx.Info(fmt.Sprintf("⬆️ Ready: %s → %s", entry.Local, entry.Remote))
		} else {
			logx.Error(fmt.Sprintf("❌ Missing local file: %s", entry.Local))
			missing++
		}
	}
	if missing > 0 {
		logx.Info(fmt.Sprintf("⚠️ %d file(s) missing locally", missing))
	}

	// Upload
	if err := uploader.UploadAll(cfg, *dryRun); err != nil {
		logx.Fatal("❌ " + err.Error())
	}
}

func printHelp() {
	fmt.Println("deploy_ftp - Minimal FTP upload utility")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  deploy_ftp [options]")
	fmt.Println()
	fmt.Println("Assuming mappings are in upload_map.json and credentials in .env, no options are needed, simply execute.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -env-path string   Path to .env file (default: ./.env)")
	fmt.Println("  -map string        Path to upload_map.json (default: ./upload_map.json)")
	fmt.Println("  -verify            Only test FTP connection, do not upload")
	fmt.Println("  -dry-run           Simulate FTP upload without transferring files")
	fmt.Println("  -v           Show version and exit")
	fmt.Println("  -h                 Show this help and exit")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  deploy_ftp -verify")
	fmt.Println("  deploy_ftp -v")
	fmt.Println("  deploy_ftp -env-path ./my.env -map ./my_upload_map.json -dry-run")
}
