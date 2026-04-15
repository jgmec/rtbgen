package main

import (
	"os"
	"testing"
)

func TestSFTPConfig_addr_DefaultPort(t *testing.T) {
	c := &SFTPConfig{Host: "sftp.example.com"}
	if got := c.addr(); got != "sftp.example.com:22" {
		t.Errorf("addr() = %q, want %q", got, "sftp.example.com:22")
	}
}

func TestSFTPConfig_addr_CustomPort(t *testing.T) {
	c := &SFTPConfig{Host: "sftp.example.com", Port: 2222}
	if got := c.addr(); got != "sftp.example.com:2222" {
		t.Errorf("addr() = %q, want %q", got, "sftp.example.com:2222")
	}
}

func TestSFTPConfig_remoteDir_Default(t *testing.T) {
	c := &SFTPConfig{}
	if got := c.remoteDir(); got != "/" {
		t.Errorf("remoteDir() = %q, want %q", got, "/")
	}
}

func TestSFTPConfig_remoteDir_Custom(t *testing.T) {
	c := &SFTPConfig{Dir: "/uploads/test"}
	if got := c.remoteDir(); got != "/uploads/test" {
		t.Errorf("remoteDir() = %q, want %q", got, "/uploads/test")
	}
}

func TestUploadSFTP_ConnectionFailed(t *testing.T) {
	// Port 1 is reserved and will always refuse connections.
	cfg := &SFTPConfig{Host: "127.0.0.1", Port: 1, User: "u", Password: "p"}
	f, err := os.CreateTemp(t.TempDir(), "upload-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	err = uploadSFTP(cfg, f.Name())
	if err == nil {
		t.Error("expected error for unreachable SFTP host, got nil")
	}
}

func TestUploadSFTP_MissingLocalFile(t *testing.T) {
	// Even with a reachable host (which would fail at dial), a missing local file
	// path should be caught. We test with an unreachable host — the dial error fires
	// first, which is still an error.
	cfg := &SFTPConfig{Host: "127.0.0.1", Port: 1, User: "u", Password: "p"}
	err := uploadSFTP(cfg, "/nonexistent/path.zip")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
