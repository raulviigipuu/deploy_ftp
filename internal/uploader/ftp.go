package uploader

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/raulviigipuu/deploy_ftp/internal/config"
	"github.com/raulviigipuu/deploy_ftp/internal/logx"
)

func TestConnection(cfg *config.Config, debug bool) error {
	log := newFTPLog(cfg, debug)
	client, err := dialFTP(cfg, log)
	if err != nil {
		return fmt.Errorf("connection failed: %s", log.redact(err.Error()))
	}
	defer client.Quit()

	if _, err := client.List("/"); err != nil {
		return fmt.Errorf("could not list root directory: %s", log.redact(err.Error()))
	}
	return nil
}

func UploadAll(cfg *config.Config, dryRun, debug bool) error {
	log := newFTPLog(cfg, debug)
	if dryRun {
		return planUpload(cfg, log)
	}

	upload := ftpUpload{cfg: cfg, log: log}
	defer upload.close()

	var failures []string
	for _, entry := range cfg.Map {
		remotePath := normalizeRemotePath(entry.Remote)
		logx.Info(fmt.Sprintf("📁 Uploading: %s → %s", entry.Local, remotePath))
		if err := upload.storeWithRetry(entry.Local, remotePath); err != nil {
			failures = append(failures, fmt.Sprintf("  %s → %s: %s", entry.Local, remotePath, log.redact(err.Error())))
			continue
		}
		logx.Info("✅ Upload successful")
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to upload %d file(s):\n%s", len(failures), strings.Join(failures, "\n"))
	}
	return nil
}

type ftpUpload struct {
	cfg             *config.Config
	log             ftpLog
	client          *ftp.ServerConn
	sizeUnsupported bool
}

func (u *ftpUpload) close() {
	if u.client != nil {
		u.client.Quit()
		u.client = nil
		u.sizeUnsupported = false
	}
}

func (u *ftpUpload) storeWithRetry(localPath, remotePath string) error {
	const maxAttempts = 3
	for attempt := 1; ; attempt++ {
		err := u.store(localPath, remotePath)
		if err == nil || attempt == maxAttempts || !temporaryTransferError(err) {
			return err
		}

		delay := time.Duration(attempt) * 250 * time.Millisecond
		logx.Info(fmt.Sprintf("⚠️ Retrying %s in %s (attempt %d/%d): %s",
			remotePath, delay, attempt+1, maxAttempts, u.log.redact(err.Error())))
		time.Sleep(delay)
	}
}

func (u *ftpUpload) store(localPath, remotePath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()

	if u.client == nil {
		u.client, err = dialFTP(u.cfg, u.log)
		if err != nil {
			return fmt.Errorf("connect to FTP server: %w", err)
		}
	}

	if err := ensureRemoteDirStructure(u.client, path.Dir(remotePath)); err != nil {
		u.close()
		return err
	}

	source := &uploadReader{file: file}
	if err := storeFile(u.client, remotePath, source); err != nil {
		// A failed transfer can leave unread replies on the control connection.
		u.close()
		if source.err != nil {
			return fmt.Errorf("read local file: %w", source.err)
		}
		return fmt.Errorf("store file: %w", err)
	}

	if err := u.verifySize(remotePath, source.bytesRead); err != nil {
		u.close()
		return fmt.Errorf("verify remote size: %w", err)
	}
	return nil
}

func storeFile(client *ftp.ServerConn, remotePath string, source io.Reader) error {
	err := client.Stor(remotePath, source)
	// STOR also allows 250 as the final reply. The client expects 226 and
	// returns joined errors. Accept 250 when every other step succeeded.
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		if len(causes) == 1 && ftpReplyCode(causes[0]) == ftp.StatusRequestedFileActionOK {
			return nil
		}
	}
	return err
}

var errSizeMismatch = errors.New("uploaded size mismatch")

func (u *ftpUpload) verifySize(remotePath string, sent int64) error {
	if u.sizeUnsupported {
		return nil
	}

	remoteSize, err := u.client.FileSize(remotePath)
	switch ftpReplyCode(err) {
	case ftp.StatusBadCommand, ftp.StatusNotImplemented, ftp.StatusNotImplementedParameter:
		u.sizeUnsupported = true
		return nil
	}
	if err != nil {
		return err
	}
	if remoteSize != sent {
		return fmt.Errorf("%w: sent %d bytes, server reports %d", errSizeMismatch, sent, remoteSize)
	}
	return nil
}

type uploadReader struct {
	file      *os.File
	bytesRead int64
	err       error
}

func (r *uploadReader) Read(p []byte) (int, error) {
	n, err := r.file.Read(p)
	r.bytesRead += int64(n)
	if err != nil && err != io.EOF {
		r.err = err
	}
	return n, err
}

func ftpReplyCode(err error) int {
	var reply *textproto.Error
	if errors.As(err, &reply) {
		return reply.Code
	}
	return 0
}

func temporaryTransferError(err error) bool {
	if code := ftpReplyCode(err); code != 0 {
		return code/100 == 4
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsTemporary || dnsErr.IsTimeout
	}
	var netErr net.Error
	return errors.Is(err, errSizeMismatch) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		(errors.As(err, &netErr) && netErr.Timeout())
}

const ftpTimeout = 10 * time.Second

func dialFTP(cfg *config.Config, log ftpLog) (*ftp.ServerConn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // allow self-signed certs
		MaxVersion:         cfg.TLSMaxVersion,
	}
	if log.debug {
		tlsConfig.VerifyConnection = func(state tls.ConnectionState) error {
			fmt.Fprintln(os.Stderr, "ftp: negotiated", tls.VersionName(state.Version))
			return nil
		}
	}
	dialer := net.Dialer{Timeout: ftpTimeout}
	control := true
	options := []ftp.DialOption{
		ftp.DialWithExplicitTLS(tlsConfig),
		ftp.DialWithTrustPasvIP(true),
		ftp.DialWithDialFunc(func(network, address string) (net.Conn, error) {
			conn, err := dialer.Dial(network, address)
			if err != nil {
				return nil, err
			}
			socket := &ftpSocket{Conn: conn}
			if control {
				control = false
				return socket, nil // the client upgrades this socket after AUTH TLS
			}
			// With a custom dialer, the caller wraps data sockets in TLS.
			return tls.Client(socket, tlsConfig), nil
		}),
	}
	if log.debug {
		options = append(options, ftp.DialWithDebugOutput(&ftpTrace{output: os.Stderr, log: log}))
	}

	address := net.JoinHostPort(cfg.FTPHost, strconv.Itoa(cfg.FTPPort))
	client, err := ftp.Dial(address, options...)
	if err != nil {
		return nil, err
	}
	if err := client.Login(cfg.FTPUser, cfg.FTPPass); err != nil {
		client.Quit()
		return nil, err
	}
	return client, nil
}

// Keep control and data timeouts relative to each I/O operation.
type ftpSocket struct {
	net.Conn
}

func (c *ftpSocket) Read(p []byte) (int, error) {
	if err := c.SetReadDeadline(time.Now().Add(ftpTimeout)); err != nil {
		return 0, err
	}
	return c.Conn.Read(p)
}

func (c *ftpSocket) Write(p []byte) (int, error) {
	if err := c.SetWriteDeadline(time.Now().Add(ftpTimeout)); err != nil {
		return 0, err
	}
	return c.Conn.Write(p)
}

type ftpLog struct {
	debug   bool
	secrets *strings.Replacer
}

func newFTPLog(cfg *config.Config, debug bool) ftpLog {
	// Prefer a full credential match when one credential is a prefix of the other.
	first, second := cfg.FTPUser, cfg.FTPPass
	if len(second) > len(first) {
		first, second = second, first
	}
	var pairs []string
	for _, value := range []string{first, second} {
		if value != "" {
			pairs = append(pairs, value, "[redacted]")
		}
	}
	return ftpLog{debug: debug, secrets: strings.NewReplacer(pairs...)}
}

func (l ftpLog) redact(s string) string {
	return l.secrets.Replace(s)
}

type ftpTrace struct {
	output  io.Writer
	log     ftpLog
	pending bytes.Buffer
}

func (t *ftpTrace) Write(p []byte) (int, error) {
	t.pending.Write(p)
	// The client sends raw chunks, which may split a credential across writes.
	for {
		end := bytes.IndexByte(t.pending.Bytes(), '\n')
		if end < 0 {
			return len(p), nil
		}
		line := strings.TrimRight(string(t.pending.Next(end+1)), "\r\n")
		fmt.Fprintln(t.output, "ftp:", t.log.redact(line))
	}
}

func planUpload(cfg *config.Config, log ftpLog) error {
	client, err := dialFTP(cfg, log)
	if err != nil {
		return fmt.Errorf("FTP connection failed: %s", log.redact(err.Error()))
	}
	defer client.Quit()

	for _, entry := range cfg.Map {
		remotePath := normalizeRemotePath(entry.Remote)
		for _, dir := range remoteDirPrefixes(path.Dir(remotePath)) {
			// All paths are absolute, so changing directory only checks accessibility.
			if err := client.ChangeDir(dir); err != nil {
				if ftpReplyCode(err) != ftp.StatusFileUnavailable {
					return fmt.Errorf("check remote directory %s: %s", dir, log.redact(err.Error()))
				}
				logx.Info(fmt.Sprintf("📂 Dry-run: would create remote directory: %s", dir))
			}
		}
		logx.Info(fmt.Sprintf("🔍 Dry-run: would upload %s → %s", entry.Local, remotePath))
	}
	return nil
}

func ensureRemoteDirStructure(client *ftp.ServerConn, remoteDir string) error {
	for _, dir := range remoteDirPrefixes(remoteDir) {
		err := client.MakeDir(dir)
		if err == nil {
			logx.Info(fmt.Sprintf("📂 Created remote directory: %s", dir))
			continue
		}

		// Existing directories can be reported as 550 or 521.
		if code := ftpReplyCode(err); code == ftp.StatusFileUnavailable || code == 521 {
			continue
		}
		return fmt.Errorf("create remote directory %s: %w", dir, err)
	}
	return nil
}

func normalizeRemotePath(p string) string {
	return strings.TrimSuffix(path.Clean("/"+p), "/")
}

func remoteDirPrefixes(dir string) []string {
	var prefixes []string
	current := "/"
	for _, part := range strings.Split(dir, "/") {
		if part != "" {
			current = path.Join(current, part)
			prefixes = append(prefixes, current)
		}
	}
	return prefixes
}
