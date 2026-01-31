package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"embed"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alexj212/consolekit"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

//go:embed scripts/*.run
var Data embed.FS

var (
	BuildDate    string
	LatestCommit string
	Version      string
)

func main() {
	// Create command executor
	customizer := func(exec *consolekit.CommandExecutor) error {
		exec.Scripts = Data
		exec.AddBuiltinCommands()
		exec.AddCommands(consolekit.AddRun(exec, Data))

		// Add version command
		var verCmd = &cobra.Command{
			Use:     "version",
			Aliases: []string{"v", "ver"},
			Short:   "Show version info",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Printf("SSH Server Example\n")
				cmd.Printf("BuildDate    : %s\n", BuildDate)
				cmd.Printf("LatestCommit : %s\n", LatestCommit)
				cmd.Printf("Version      : %s\n", Version)
			},
		}
		exec.AddCommands(func(rootCmd *cobra.Command) {
			rootCmd.AddCommand(verCmd)
		})

		return nil
	}

	executor, err := consolekit.NewCommandExecutor("ssh-server", customizer)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Load or generate host key
	hostKey, err := loadOrGenerateHostKey("ssh_host_key")
	if err != nil {
		log.Fatalf("Failed to load host key: %v", err)
	}

	// Create SSH handler
	sshHandler := consolekit.NewSSHHandler(executor, ":2222", hostKey)

	// Configure authentication
	sshHandler.SetAuthConfig(&consolekit.SSHAuthConfig{
		// Password authentication
		PasswordAuth: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			// Simple password check - in production, use proper authentication
			validUsers := map[string]string{
				"admin":     "secret123",
				"developer": "dev456",
				"guest":     "guest",
			}

			if expectedPass, ok := validUsers[conn.User()]; ok {
				if string(password) == expectedPass {
					log.Printf("User %s authenticated via password from %s\n",
						conn.User(), conn.RemoteAddr())
					return &ssh.Permissions{
						Extensions: map[string]string{
							"user": conn.User(),
						},
					}, nil
				}
			}

			log.Printf("Failed authentication attempt for user %s from %s\n",
				conn.User(), conn.RemoteAddr())
			return nil, fmt.Errorf("invalid credentials")
		},

		// Public key authentication (if ~/.ssh/authorized_keys exists)
		PublicKeyAuth: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Load authorized keys
			authorizedKeys, err := loadAuthorizedKeys()
			if err != nil {
				return nil, fmt.Errorf("no authorized keys")
			}

			// Check if key is authorized
			keyData := string(key.Marshal())
			if _, ok := authorizedKeys[keyData]; ok {
				log.Printf("User %s authenticated via public key from %s\n",
					conn.User(), conn.RemoteAddr())
				return &ssh.Permissions{
					Extensions: map[string]string{
						"user":    conn.User(),
						"pubkey":  "true",
						"keytype": key.Type(),
					},
				}, nil
			}

			return nil, fmt.Errorf("public key not authorized")
		},
	})

	// Optional: Configure command filtering
	// Deny dangerous commands for guest user
	config := &consolekit.TransportConfig{
		Executor: executor,
		DeniedCommands: []string{
			"osexec",  // Don't allow OS command execution
			"clip",    // Clipboard access
			"paste",   // Clipboard access
		},
	}
	sshHandler.SetTransportConfig(config)

	// Print connection info
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ConsoleKit SSH Server Example                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("SSH server listening on port 2222")
	fmt.Println()
	fmt.Println("Connect using:")
	fmt.Println("  ssh admin@localhost -p 2222")
	fmt.Println("  Password: secret123")
	fmt.Println()
	fmt.Println("Available users:")
	fmt.Println("  admin     - Password: secret123 (full access)")
	fmt.Println("  developer - Password: dev456     (full access)")
	fmt.Println("  guest     - Password: guest      (restricted)")
	fmt.Println()
	fmt.Println("Single command execution:")
	fmt.Println("  ssh admin@localhost -p 2222 'print \"Hello from SSH\"'")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()

	// Start SSH server (blocking)
	if err := sshHandler.Start(); err != nil {
		log.Fatalf("SSH server error: %v", err)
	}
}

// loadOrGenerateHostKey loads existing host key or generates a new one
func loadOrGenerateHostKey(keyPath string) (ssh.Signer, error) {
	// Try to load existing key
	if _, err := os.Stat(keyPath); err == nil {
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read host key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host key: %w", err)
		}

		log.Printf("Loaded existing host key from %s\n", keyPath)
		return signer, nil
	}

	// Generate new key
	log.Printf("Generating new RSA host key...\n")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Save private key
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	if err := pem.Encode(keyFile, privateKeyPEM); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}

	log.Printf("Saved new host key to %s\n", keyPath)

	// Create signer
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return signer, nil
}

// loadAuthorizedKeys loads SSH authorized keys from ~/.ssh/authorized_keys
func loadAuthorizedKeys() (map[string]struct{}, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	authKeysPath := filepath.Join(homeDir, ".ssh", "authorized_keys")
	authKeysData, err := os.ReadFile(authKeysPath)
	if err != nil {
		return nil, err
	}

	authorizedKeys := make(map[string]struct{})
	for len(authKeysData) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authKeysData)
		if err != nil {
			break
		}
		authorizedKeys[string(pubKey.Marshal())] = struct{}{}
		authKeysData = rest
	}

	return authorizedKeys, nil
}
