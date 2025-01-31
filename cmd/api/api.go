package api

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/celestiaorg/knuu/internal/api/v1"
	"github.com/celestiaorg/knuu/internal/api/v1/services"
	"github.com/celestiaorg/knuu/internal/database"
)

const (
	apiCmdName = "api"

	flagPort        = "port"
	flagAPILogLevel = "log-level"

	flagDBHost     = "db.host"
	flagDBUser     = "db.user"
	flagDBPassword = "db.password"
	flagDBName     = "db.name"
	flagDBPort     = "db.port"

	flagSecretKey = "secret-key"
	flagAdminUser = "admin-user"
	flagAdminPass = "admin-pass"

	flagTestsLogsPath = "tests-logs-path"

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

	defaultLogsPath = services.DefaultTestLogsPath
)

func NewAPICmd() *cobra.Command {
	apiCmd := &cobra.Command{
		Use:   apiCmdName,
		Short: "Start the Knuu API server",
		Long:  "Start the API server to manage tests, tokens, and users.",
		RunE:  runAPIServer,
	}

	apiCmd.Flags().IntP(flagPort, "p", defaultPort, "Port to run the API server on")
	apiCmd.Flags().StringP(flagAPILogLevel, "l", defaultLogLevel, "Log level: debug | release | test")

	apiCmd.Flags().StringP(flagDBHost, "d", defaultDBHost, "Postgres database host")
	apiCmd.Flags().StringP(flagDBUser, "", defaultDBUser, "Postgres database user")
	apiCmd.Flags().StringP(flagDBPassword, "", defaultDBPassword, "Postgres database password")
	apiCmd.Flags().StringP(flagDBName, "", defaultDBName, "Postgres database name")
	apiCmd.Flags().IntP(flagDBPort, "", defaultDBPort, "Postgres database port")

	apiCmd.Flags().StringP(flagSecretKey, "", defaultSecretKey, "JWT secret key")
	apiCmd.Flags().StringP(flagAdminUser, "", defaultAdminUser, "Admin username")
	apiCmd.Flags().StringP(flagAdminPass, "", defaultAdminPass, "Admin password")

	apiCmd.Flags().StringP(flagTestsLogsPath, "", defaultLogsPath, "Directory to store logs of the tests")

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

	handleShutdown(apiServer, db, apiOpts.Logger)

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

	apiLogLevel, err := flags.GetString(flagAPILogLevel)
	if err != nil {
		return database.Options{}, fmt.Errorf("failed to get API log level: %v", err)
	}

	var dbLogLevel logger.LogLevel
	switch apiLogLevel {
	case gin.DebugMode:
		dbLogLevel = logger.Info
	case gin.ReleaseMode:
		dbLogLevel = logger.Error
	case gin.TestMode:
		dbLogLevel = logger.Info
	}

	return database.Options{
		Host:     dbHost,
		User:     dbUser,
		Password: dbPassword,
		DBName:   dbName,
		Port:     dbPort,
		LogLevel: dbLogLevel,
	}, nil
}

func getAPIOptions(flags *pflag.FlagSet) (api.Options, error) {
	port, err := flags.GetInt(flagPort)
	if err != nil {
		return api.Options{}, fmt.Errorf("failed to get port: %v", err)
	}

	apiLogLevel, err := flags.GetString(flagAPILogLevel)
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

	testsLogsPath, err := flags.GetString(flagTestsLogsPath)
	if err != nil {
		return api.Options{}, fmt.Errorf("failed to get tests logs path: %v", err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	switch apiLogLevel {
	case gin.DebugMode:
		logger.SetLevel(logrus.DebugLevel)
	case gin.ReleaseMode:
		logger.SetLevel(logrus.ErrorLevel)
	case gin.TestMode:
		logger.SetLevel(logrus.InfoLevel)
	}

	return api.Options{
		Port:       port,
		APILogMode: apiLogLevel, // gin logger (HTTP request level)
		SecretKey:  secretKey,
		AdminUser:  adminUser,
		AdminPass:  adminPass,
		Logger:     logger, // handler (application level logger)
		TestServiceOptions: services.TestServiceOptions{
			TestsLogsPath: testsLogsPath, // directory to store logs of each test
			Logger:        logger,
		},
	}, nil
}

func handleShutdown(apiServer *api.API, db *gorm.DB, logger *logrus.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf("failed to get sql db: %v", err)
	}

	go func() {
		sig := <-quit
		logger.Infof("Received signal: %v. Shutting down gracefully...", sig)
		if err := sqlDB.Close(); err != nil {
			logger.Errorf("failed to close sql db: %v", err)
		}
		logger.Info("DB connection closed")
		if err := apiServer.Stop(context.Background()); err != nil {
			logger.Errorf("failed to stop api server: %v", err)
		}
		logger.Info("API server stopped")
		os.Exit(0)
	}()
}
