package api

import (
	"fmt"

	"github.com/celestiaorg/knuu/internal/api/v1"
	"github.com/celestiaorg/knuu/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	apiCmdName = "api"

	flagPort     = "port"
	flagLogLevel = "log-level"

	flagDBHost     = "db.host"
	flagDBUser     = "db.user"
	flagDBPassword = "db.password"
	flagDBName     = "db.name"
	flagDBPort     = "db.port"

	flagSecretKey = "secret-key"
	flagAdminUser = "admin-user"
	flagAdminPass = "admin-pass"

	defaultPort     = 8080
	defaultLogLevel = gin.ReleaseMode

	defaultDBHost     = "localhost"
	defaultDBUser     = "postgres"
	defaultDBPassword = "postgres"
	defaultDBName     = "postgres"
	defaultDBPort     = 5432

	defaultSecretKey = "secret"
	defaultAdminUser = "admin"
	defaultAdminPass = "admin"
)

func NewAPICmd() *cobra.Command {
	apiCmd := &cobra.Command{
		Use:   apiCmdName,
		Short: "Start the Knuu API server",
		Long:  "Start the API server to manage tests, tokens, and users.",
		RunE:  runAPIServer,
	}

	apiCmd.Flags().IntP(flagPort, "p", defaultPort, "Port to run the API server on")
	apiCmd.Flags().StringP(flagLogLevel, "l", defaultLogLevel, "Log level: debug | release | test")

	apiCmd.Flags().StringP(flagDBHost, "d", defaultDBHost, "Postgres database host")
	apiCmd.Flags().StringP(flagDBUser, "", defaultDBUser, "Postgres database user")
	apiCmd.Flags().StringP(flagDBPassword, "", defaultDBPassword, "Postgres database password")
	apiCmd.Flags().StringP(flagDBName, "", defaultDBName, "Postgres database name")
	apiCmd.Flags().IntP(flagDBPort, "", defaultDBPort, "Postgres database port")

	apiCmd.Flags().StringP(flagSecretKey, "", defaultSecretKey, "JWT secret key")
	apiCmd.Flags().StringP(flagAdminUser, "", defaultAdminUser, "Admin username")
	apiCmd.Flags().StringP(flagAdminPass, "", defaultAdminPass, "Admin password")

	return apiCmd
}

func runAPIServer(cmd *cobra.Command, args []string) error {
	port, err := cmd.Flags().GetInt(flagPort)
	if err != nil {
		return fmt.Errorf("failed to get port: %v", err)
	}

	logLevel, err := cmd.Flags().GetString(flagLogLevel)
	if err != nil {
		return fmt.Errorf("failed to get log level: %v", err)
	}

	dbOpts, err := getDBOptions(cmd.Flags())
	if err != nil {
		return fmt.Errorf("failed to get database options: %v", err)
	}

	db, err := database.New(dbOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	secretKey, err := cmd.Flags().GetString(flagSecretKey)
	if err != nil {
		return fmt.Errorf("failed to get secret key: %v", err)
	}

	adminUser, err := cmd.Flags().GetString(flagAdminUser)
	if err != nil {
		return fmt.Errorf("failed to get admin user: %v", err)
	}

	adminPass, err := cmd.Flags().GetString(flagAdminPass)
	if err != nil {
		return fmt.Errorf("failed to get admin password: %v", err)
	}

	apiServer := api.New(db, api.Options{
		Port:      port,
		LogMode:   logLevel,
		SecretKey: secretKey,
		AdminUser: adminUser,
		AdminPass: adminPass,
	})

	return apiServer.Start()
}

func getDBOptions(flags *pflag.FlagSet) (database.Options, error) {
	dbHost, err := flags.GetString(flagDBHost)
	if err != nil {
		return database.Options{}, fmt.Errorf("failed to get database host: %v", err)
	}

	dbUser, err := flags.GetString(flagDBUser)
	if err != nil {
		return database.Options{}, fmt.Errorf("failed to get database user: %v", err)
	}

	dbPassword, err := flags.GetString(flagDBPassword)
	if err != nil {
		return database.Options{}, fmt.Errorf("failed to get database password: %v", err)
	}

	dbName, err := flags.GetString(flagDBName)
	if err != nil {
		return database.Options{}, fmt.Errorf("failed to get database name: %v", err)
	}

	dbPort, err := flags.GetInt(flagDBPort)
	if err != nil {
		return database.Options{}, fmt.Errorf("failed to get database port: %v", err)
	}

	return database.Options{
		Host:     dbHost,
		User:     dbUser,
		Password: dbPassword,
		DBName:   dbName,
		Port:     dbPort,
	}, nil
}
