package api

import (
	"context"
	"fmt"

	"github.com/celestiaorg/knuu/internal/api/v1"
	"github.com/celestiaorg/knuu/internal/api/v1/services"
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

	flagLogsPath = "logs-path"

	defaultPort     = 8080
	defaultLogLevel = gin.ReleaseMode

	defaultDBHost     = database.DefaultHost
	defaultDBUser     = database.DefaultUser
	defaultDBPassword = database.DefaultPassword
	defaultDBName     = database.DefaultDBName
	defaultDBPort     = database.DefaultPort

	defaultSecretKey = "secret"
	defaultAdminUser = "admin"
	defaultAdminPass = "admin"

	defaultLogsPath = services.DefaultLogsPath
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

	apiCmd.Flags().StringP(flagLogsPath, "", defaultLogsPath, "Path to store logs")

	return apiCmd
}

func runAPIServer(cmd *cobra.Command, args []string) error {
	dbOpts, err := getDBOptions(cmd.Flags())
	if err != nil {
		return fmt.Errorf("failed to get database options: %v", err)
	}

	db, err := database.New(dbOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	apiOpts, err := getAPIOptions(cmd.Flags())
	if err != nil {
		return fmt.Errorf("failed to get API options: %v", err)
	}

	apiServer, err := api.New(context.Background(), db, apiOpts)
	if err != nil {
		return fmt.Errorf("failed to create API server: %v", err)
	}

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

func getAPIOptions(flags *pflag.FlagSet) (api.Options, error) {
	port, err := flags.GetInt(flagPort)
	if err != nil {
		return api.Options{}, fmt.Errorf("failed to get port: %v", err)
	}

	logLevel, err := flags.GetString(flagLogLevel)
	if err != nil {
		return api.Options{}, fmt.Errorf("failed to get log level: %v", err)
	}

	secretKey, err := flags.GetString(flagSecretKey)
	if err != nil {
		return api.Options{}, fmt.Errorf("failed to get secret key: %v", err)
	}

	adminUser, err := flags.GetString(flagAdminUser)
	if err != nil {
		return api.Options{}, fmt.Errorf("failed to get admin user: %v", err)
	}

	adminPass, err := flags.GetString(flagAdminPass)
	if err != nil {
		return api.Options{}, fmt.Errorf("failed to get admin password: %v", err)
	}

	logsPath, err := flags.GetString(flagLogsPath)
	if err != nil {
		return api.Options{}, fmt.Errorf("failed to get logs path: %v", err)
	}

	return api.Options{
		Port:      port,
		LogMode:   logLevel,
		SecretKey: secretKey,
		AdminUser: adminUser,
		AdminPass: adminPass,
		TestServiceOptions: services.TestServiceOptions{
			LogsPath: logsPath,
		},
	}, nil
}
