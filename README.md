# deploy_ftp

🚀 `deploy_ftp` is a minimal CLI utility written in Go that uploads static files to an FTP server using a declarative JSON-based upload map and `.env` configuration.

It's meant for simple local deploys.

---

## 📦 Build

    go build -o deploy_ftp ./cmd/deploy_ftp

## ▶️ Run

    go run ./cmd/deploy_ftp

    go run ./cmd/deploy_ftp [--verify] [--dry-run] [--env-path PATH] [--map PATH]

Or after building:

    ./deploy_ftp

### 📄 Flags

| Flag         | Description                                                                                                 |
|--------------|-------------------------------------------------------------------------------------------------------------|
| `--verify`   | Verifies FTP connection and login; no upload is performed.                                                  |
| `--dry-run`  | Simulates the upload: logs which files would be uploaded and which directories would be created.            |
| `--env-path` | Custom path to your `.env` file (default: `./.env`).                                                        |
| `--map`      | Custom path to your `upload_map.json` file (default: `./upload_map.json`).                                  |
| `--version`  | Display version info                                                                                        |

### Examples

✅ Verify FTP connection:

    go run ./cmd/deploy_ftp --verify

🔍 Test what would be uploaded:

    go run ./cmd/deploy_ftp --dry-run

🔍 Dry-run with custom config paths:

    go run ./cmd/deploy_ftp --dry-run --env-path ./configs/my.env --map ./configs/my_upload_map.json

⬆️ Perform actual upload:

    go run ./cmd/deploy_ftp

## ⚙️ Configuration

This tool expects two files in the current directory:

.env: contains your FTP credentials

upload_map.json: maps local → remote paths

You can also provide custom paths:

    ./deploy_ftp --env-path=../secrets/my.env --map=../config/upload.json

### Sample .env

    FTP_HOST=ftp.example.com
    FTP_PORT=2121
    FTP_USER=yourusername
    FTP_PASS=yourpassword
    USE_TLS=true

### Sample upload_map.json

    [
    { "local": "public/index.html", "remote": "/public_html/index.html" },
    { "local": "public/styles.css", "remote": "/public_html/styles.css" }
    ]

### ✅ Verify Connection

Test your FTP setup without uploading any files:

    ./deploy_ftp --verify

## 🛠 Deploy

Upload all mapped files to the configured FTP server:

    ./deploy_ftp

If a file is missing locally, it is skipped with a warning.

If the remote directory does not exist, it will be created (TODO).

## 🔧 Maintenance

To update dependencies:

    go get -u ./...
    go mod tidy

To build cross-platform binaries:

    GOOS=linux GOARCH=amd64 go build -o deploy_ftp_linux ./cmd/deploy_ftp
    GOOS=windows GOARCH=amd64 go build -o deploy_ftp.exe ./cmd/deploy_ftp

