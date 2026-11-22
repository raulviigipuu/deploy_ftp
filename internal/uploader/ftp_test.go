package uploader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"strings"
	"testing"

	"github.com/raulviigipuu/deploy_ftp/internal/config"
)

func TestTemporaryTransferError(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{"missing reply", fmt.Errorf("store file: %w", errors.Join(io.EOF)), true},
		{"transient FTP reply", &textproto.Error{Code: 426, Msg: "transfer aborted"}, true},
		{"permanent FTP reply after sending data", errors.Join(io.ErrUnexpectedEOF, &textproto.Error{Code: 552, Msg: "quota exceeded"}), false},
		{"size mismatch", fmt.Errorf("%w: expected 10 bytes", errSizeMismatch), true},
		{"local read failure", errors.New("read local file: permission denied"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := temporaryTransferError(tt.err); got != tt.want {
				t.Errorf("temporaryTransferError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestFTPTraceRedactsCredentialsAcrossWrites(t *testing.T) {
	cfg := &config.Config{FTPUser: "ftp-user", FTPPass: "ftp-user-secret"}
	var output bytes.Buffer
	trace := ftpTrace{output: &output, log: newFTPLog(cfg, true)}

	for _, chunk := range []string{
		"USER ftp-", "user\r\nPASS ftp-user-", "secret\r", "\n",
		"230 Echo ftp-user and ftp-user-se", "cret\r\nSTOR /photo.jpg\r\n",
	} {
		if n, err := io.WriteString(&trace, chunk); n != len(chunk) || err != nil {
			t.Fatalf("trace.Write(%q) = %d, %v", chunk, n, err)
		}
	}

	want := "ftp: USER [redacted]\nftp: PASS [redacted]\nftp: 230 Echo [redacted] and [redacted]\nftp: STOR /photo.jpg\n"
	if got := output.String(); strings.Contains(got, cfg.FTPUser) || got != want {
		t.Errorf("unexpected trace output %q", got)
	}
}
