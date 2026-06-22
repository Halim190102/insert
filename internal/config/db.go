package config

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"time"

	_ "github.com/sijms/go-ora/v2"
	"golang.org/x/crypto/ssh"
)

func createSSHTunnel(cfg *ENVConfig) error {
	sshConfig := &ssh.ClientConfig{
		User: cfg.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.SSHPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial(
		"tcp",
		fmt.Sprintf("%s:%s", cfg.SSHHost, cfg.SSHPort),
		sshConfig,
	)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:11521")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				continue
			}

			go func() {
				remoteConn, err := client.Dial(
					"tcp",
					fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
				)
				if err != nil {
					localConn.Close()
					return
				}

				go func() {
					defer localConn.Close()
					defer remoteConn.Close()
					io.Copy(remoteConn, localConn)
				}()

				go func() {
					defer localConn.Close()
					defer remoteConn.Close()
					io.Copy(localConn, remoteConn)
				}()
			}()
		}
	}()

	log.Println("✅ SSH Tunnel established on localhost:11521")
	return nil
}

func ConnectOracle(cfg *ENVConfig) *sql.DB {
	if err := createSSHTunnel(cfg); err != nil {
		log.Fatalf("❌ SSH tunnel failed: %v", err)
	}

	// escape password jika mengandung karakter spesial
	password := url.QueryEscape(cfg.DBPass)

	dsn := fmt.Sprintf(
		"oracle://%s:%s@127.0.0.1:11521/%s",
		cfg.DBUser,
		password,
		cfg.DBService,
	)

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		log.Fatalf("❌ Oracle connection failed: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Oracle ping failed: %v", err)
	}

	log.Println("✅ Oracle connected successfully")

	return db
}