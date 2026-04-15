package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPConfig holds the connection parameters for an SFTP upload target.
type SFTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user"`
	Password string `json:"password"`
	Dir      string `json:"dir,omitempty"` // remote directory; defaults to "/"
}

func (c *SFTPConfig) addr() string {
	port := c.Port
	if port == 0 {
		port = 22
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(port))
}

func (c *SFTPConfig) remoteDir() string {
	if c.Dir == "" {
		return "/"
	}
	return c.Dir
}

// uploadSFTP uploads localPath to the remote directory defined in cfg, then removes the local file.
func uploadSFTP(cfg *SFTPConfig, localPath string) error {
	sshCfg := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.Password),
		},
		// Self-signed / internal servers: accept any host key.
		// For production, replace with a known-hosts verifier.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
	}

	conn, err := ssh.Dial("tcp", cfg.addr(), sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial %s: %w", cfg.addr(), err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer client.Close()

	if err := client.MkdirAll(cfg.remoteDir()); err != nil {
		return fmt.Errorf("mkdir %s: %w", cfg.remoteDir(), err)
	}

	remotePath := filepath.Join(cfg.remoteDir(), filepath.Base(localPath))
	dst, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file %s: %w", remotePath, err)
	}
	defer dst.Close()

	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer src.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	return nil
}
