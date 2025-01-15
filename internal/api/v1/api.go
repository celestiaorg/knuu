package api

import (
	"fmt"
	"net/http"

	"github.com/celestiaorg/knuu/internal/api/v1/handlers"
	"github.com/celestiaorg/knuu/internal/api/v1/middleware"
	"github.com/celestiaorg/knuu/internal/api/v1/services"
	"github.com/celestiaorg/knuu/internal/database/repos"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"gorm.io/gorm"
)

const (
	defaultPort    = 8080
	defaultLogMode = gin.ReleaseMode
)

type API struct {
	router *gin.Engine
	server *http.Server
}

type Options struct {
	Port          int
	LogMode       string // gin.DebugMode, gin.ReleaseMode(default), gin.TestMode
	OriginAllowed string
	SecretKey     string
}

func New(db *gorm.DB, opts Options) *API {
	opts = setDefaults(opts)
	gin.SetMode(opts.LogMode)

	rt := gin.Default()

	public := rt.Group("/")
	{
		uh := handlers.NewUserHandler(services.NewUserService(opts.SecretKey, repos.NewUserRepository(db)))
		public.POST(pathsUserRegister, uh.Register)
		public.POST(pathsUserLogin, uh.Login)
	}
	protected := rt.Group("/", middleware.AuthMiddleware())

	_ = protected

	a := &API{
		router: rt,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", opts.Port),
			Handler: handleOrigin(rt, opts.OriginAllowed),
		},
	}

	if opts.LogMode != gin.ReleaseMode {
		public.GET("/", a.IndexPage)
	}

	return a
}

func (a *API) Start() error {
	fmt.Printf("Starting API server in %s mode on %s\n", gin.Mode(), a.server.Addr)
	return a.server.ListenAndServe()
}

func (a *API) Stop() error {
	fmt.Println("Shutting down API server")
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
