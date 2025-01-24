package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/celestiaorg/knuu/internal/api/v1/handlers"
	"github.com/celestiaorg/knuu/internal/api/v1/middleware"
	"github.com/celestiaorg/knuu/internal/api/v1/services"
	"github.com/celestiaorg/knuu/internal/database/models"
	"github.com/celestiaorg/knuu/internal/database/repos"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"gorm.io/gorm"
)

const (
	defaultPort    = 8080
	defaultLogMode = gin.ReleaseMode
)

type apiCleanup struct {
	testService *services.TestService
}

type API struct {
	router  *gin.Engine
	server  *http.Server
	cleanup apiCleanup
}

type Options struct {
	Port          int
	LogMode       string // gin.DebugMode, gin.ReleaseMode(default), gin.TestMode
	OriginAllowed string
	SecretKey     string

	AdminUser string // default admin username
	AdminPass string // default admin password

	TestServiceOptions services.TestServiceOptions
}

func New(ctx context.Context, db *gorm.DB, opts Options) (*API, error) {
	opts = setDefaults(opts)
	gin.SetMode(opts.LogMode)

	rt := gin.Default()

	auth := middleware.NewAuth(opts.SecretKey)
	uh, err := getUserHandler(ctx, opts, db, auth)
	if err != nil {
		return nil, err
	}

	public := rt.Group("/")
	{
		public.POST(pathsUserLogin, uh.Login)
	}

	testService, err := services.NewTestService(ctx, repos.NewTestRepository(db), opts.TestServiceOptions)
	if err != nil {
		return nil, err
	}

	protected := rt.Group("/", auth.AuthMiddleware())
	{
		protected.POST(pathsUserRegister, auth.RequireRole(models.RoleAdmin), uh.Register)

		th := handlers.NewTestHandler(testService)
		protected.POST(pathsTests, th.CreateTest)
		// protected.GET(pathsTestDetails, th.GetTestDetails)
		// protected.GET(pathsTestInstances, th.GetInstances)
		protected.GET(pathsTestInstanceDetails, th.GetInstance)
		protected.GET(pathsTestInstanceStatus, th.GetInstanceStatus)
		protected.POST(pathsTestInstanceDetails, th.CreateInstance) // Need to do something about updating an instance
		// protected.POST(pathsTestInstanceExecute, th.ExecuteInstance)
	}

	_ = protected

	a := &API{
		router: rt,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", opts.Port),
			Handler: handleOrigin(rt, opts.OriginAllowed),
		},
		cleanup: apiCleanup{
			testService: testService,
		},
	}

	if opts.LogMode != gin.ReleaseMode {
		public.GET("/", a.IndexPage)
	}

	return a, nil
}

func (a *API) Start() error {
	fmt.Printf("Starting API server in %s mode on %s\n", gin.Mode(), a.server.Addr)
	return a.server.ListenAndServe()
}

func (a *API) Stop(ctx context.Context) error {
	fmt.Println("Shutting down API server")
	if a.cleanup.testService != nil {
		err := a.cleanup.testService.Shutdown(ctx)
		if err != nil {
			return err
		}
	}
	return a.server.Close()
}

func setDefaults(opts Options) Options {
	if opts.Port == 0 {
		opts.Port = defaultPort
	}

	if opts.LogMode == "" {
		opts.LogMode = defaultLogMode
	}

	if opts.SecretKey == "" {
		opts.SecretKey = "secret"
	}

	return opts
}

func handleOrigin(router *gin.Engine, originAllowed string) http.Handler {
	if originAllowed == "" {
		return router
	}

	headersOk := []string{"X-Requested-With", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "X-CSRF-Token"}
	originsOk := []string{originAllowed}
	methodsOk := []string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}

	return cors.New(cors.Options{
		AllowedHeaders: headersOk,
		AllowedOrigins: originsOk,
		AllowedMethods: methodsOk,
	}).Handler(router)
}

func getUserHandler(ctx context.Context, opts Options, db *gorm.DB, auth *middleware.Auth) (*handlers.UserHandler, error) {
	us, err := services.NewUserService(ctx, opts.AdminUser, opts.AdminPass, repos.NewUserRepository(db))
	if err != nil {
		return nil, err
	}

	return handlers.NewUserHandler(us, auth), nil
}
