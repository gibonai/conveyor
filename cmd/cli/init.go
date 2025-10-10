/*
Copyright © 2024 Conveyor CI Contributors
*/
package cli

import (
	"fmt"

	"github.com/open-ug/conveyor/internal/init"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Conveyor CI for the first time",
	Long: `Initialize Conveyor CI for the first time by creating necessary configuration files, 
directories, and TLS certificates. This command sets up everything needed to run Conveyor CI.

Examples:
  conveyor init                    # Initialize with default settings
  conveyor init --force            # Overwrite existing files
  conveyor init --auth-enabled     # Enable authentication
  conveyor init --api-port 9090    # Use custom API port
  conveyor init --ca /path/to/ca.crt --private-key /path/to/key.pem --crt /path/to/server.crt
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flag values
		config, _ := cmd.Flags().GetString("config")
		force, _ := cmd.Flags().GetBool("force")
		authEnabled, _ := cmd.Flags().GetBool("auth-enabled")
		apiPort, _ := cmd.Flags().GetInt("api-port")
		natsPort, _ := cmd.Flags().GetInt("nats-port")
		caPath, _ := cmd.Flags().GetString("ca")
		privateKeyPath, _ := cmd.Flags().GetString("private-key")
		certPath, _ := cmd.Flags().GetString("crt")

		// Create init options
		opts := init.Options{
			ConfigPath:     config,
			Force:          force,
			AuthEnabled:    authEnabled,
			APIPort:        apiPort,
			NATSPort:       natsPort,
			CAPath:         caPath,
			PrivateKeyPath: privateKeyPath,
			CertPath:       certPath,
		}

		fmt.Println("Initializing Conveyor CI...")
		
		if err := init.Initialize(opts); err != nil {
			return fmt.Errorf("failed to initialize Conveyor CI: %w", err)
		}

		fmt.Println("✅ Conveyor CI initialized successfully!")
		fmt.Println("You can now start the service with: conveyor up")
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Configuration flags
	initCmd.Flags().StringP("config", "c", "", "Custom config file path (default: conveyor.yml in appropriate directory)")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing files without prompting")
	
	// Authentication flags
	initCmd.Flags().Bool("auth-enabled", false, "Enable authentication in configuration")
	
	// Port configuration flags
	initCmd.Flags().Int("api-port", 8080, "API server port")
	initCmd.Flags().Int("nats-port", 4222, "NATS server port")
	
	// Certificate flags
	initCmd.Flags().String("ca", "", "Path to existing CA certificate (skip generation if provided)")
	initCmd.Flags().String("private-key", "", "Path to existing private key (use with --ca)")
	initCmd.Flags().String("crt", "", "Path to existing server certificate (use with --ca)")
}