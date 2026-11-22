# deploy_ftp

🚀 `deploy_ftp` is a minimal CLI utility written in Go that uploads static files to an FTP server using a declarative JSON-based upload map and `.env` configuration.

It's meant for simple local deploys.

---

## 📦 Build

    go build -o deploy_ftp ./cmd/deploy_ftp

## 📥 Install

    go install ./cmd/deploy_ftp
    deploy_ftp -v

When installed from a Git checkout, `-v` shows the module version Go embeds in the binary. A tagged commit shows its tag. Other commits show a Go pseudo-version containing the commit ID. Uncommitted changes add a `+dirty` suffix.

## 🧰 Setup

    cp upload_map.example.json upload_map.json

Map the local and remote file paths

    cp .env.sample .env

Change the credentials

## ▶️ Run

If the mappings are in **upload_map.json** and the credentials are in the **.env** file, no options are needed.

    go run ./cmd/deploy_ftp

    go run ./cmd/deploy_ftp [--verify] [--dry-run] [-debug] [--env-path PATH] [--map PATH]

Or after building:

    ./deploy_ftp

### 📄 Flags

| Flag         | Description                                                                                                 |
|--------------|-------------------------------------------------------------------------------------------------------------|
| `--verify`   | Verifies FTP connection and login; no upload is performed.                                                  |
| `--dry-run`  | Simulates the upload: logs which files would be uploaded and which directories would be created.            |
| `-debug`     | Logs FTP commands and replies to stderr with credentials redacted.                                          |
| `--env-path` | Custom path to your `.env` file (default: `./.env`).                                                        |
| `--map`      | Custom path to your `upload_map.json` file (default: `./upload_map.json`).                                  |
| `-v`         | Display version info                                                                                        |
| `-h`         | Display help/usage info                                                                                     |


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
    # FTP_TLS_MAX_VERSION=1.2

The client uses explicit FTPS. If a server repeatedly returns `451 Error during read from data connection` during uploads, try setting `FTP_TLS_MAX_VERSION=1.2` in that deployment's `.env`. Leave it unset for Go's default TLS negotiation. Use `-debug` to see the negotiated TLS version.

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

The uploader tries to create missing remote directories. After a temporary FTP failure, it retries the file from the beginning up to twice, after 250 ms and 500 ms.

The uploader processes the rest of the map after a file fails. It lists failed files at the end and exits with status 1 if any file could not be uploaded.

Use `./deploy_ftp -debug` to trace FTP commands and replies on stderr.

## 🔧 Maintenance

To update dependencies:

    go get -u ./...
    go mod tidy

To build cross-platform binaries:

    GOOS=linux GOARCH=amd64 go build -o deploy_ftp_linux ./cmd/deploy_ftp
    GOOS=windows GOARCH=amd64 go build -o deploy_ftp.exe ./cmd/deploy_ftp

## Release on GitHub

Run this from the repository root on `main`. Choose an unused version tag. This example uses `v1.1.0`.

```sh
git status --short
go test ./...
```

Stage the files for the release and review the staged changes before committing.

```sh
git add -A
git diff --cached
```

```sh
git commit -m "Prepare deploy_ftp v1.1.0"
git tag -a v1.1.0 -m "deploy_ftp v1.1.0"
git push origin main v1.1.0
```

Pushing the tag triggers GitHub Actions to build Linux and Windows `amd64` ZIP archives and attach them to the GitHub release.

Install the tagged version and check the version printed by the binary.

```sh
go install github.com/raulviigipuu/deploy_ftp/cmd/deploy_ftp@v1.1.0
deploy_ftp -v
```
