package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	apierrs "k8s.io/apimachinery/pkg/api/errors"

	"github.com/celestiaorg/knuu/internal/database"
	"github.com/celestiaorg/knuu/internal/database/models"
	"github.com/celestiaorg/knuu/internal/database/repos"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/celestiaorg/knuu/pkg/minio"
)

const (
	DefaultTestTimeout      = time.Hour * 1
	DefaultNamespace        = "default"
	DefaultTestLogsPath     = "/tmp/knuu-logs" // directory to store logs of each test
	LogsDirPermission       = 0755
	LogsFilePermission      = 0644
	PeriodicCleanupInterval = time.Minute * 10
)

type testServiceCleanup struct {
	logFiles []*os.File
}

type TestService struct {
	repo             *repos.TestRepository
	knuuList         map[uint]map[string]*knuu.Knuu // key is the user ID, second key is the scope
	knuuListMu       sync.RWMutex
	defaultK8sClient *k8s.Client
	testsLogsPath    string
	cleanup          *testServiceCleanup
	logger           *logrus.Logger
	stopCleanupChan  chan struct{}
}

type TestServiceOptions struct {
	TestsLogsPath string // optional directory where the logs of all tests will be stored each test has one log file
	Logger        *logrus.Logger
}

func NewTestService(ctx context.Context, repo *repos.TestRepository, opts TestServiceOptions) (*TestService, error) {
	opts = setServiceOptsDefaults(opts)

	s := &TestService{
		repo:            repo,
		knuuList:        make(map[uint]map[string]*knuu.Knuu),
		testsLogsPath:   opts.TestsLogsPath,
		logger:          opts.Logger,
		stopCleanupChan: make(chan struct{}),
	}

	if _, err := os.Stat(s.testsLogsPath); os.IsNotExist(err) {
		if err := os.MkdirAll(s.testsLogsPath, LogsDirPermission); err != nil {
			return nil, err
		}
	}

	k8sClient, err := k8s.NewClient(ctx, DefaultNamespace, logrus.New())
	if err != nil {
		return nil, err
	}
	s.defaultK8sClient = k8sClient

	if err := s.loadRunningTestsFromDB(ctx); err != nil {
		return nil, err
	}

	go s.startPeriodicCleanup()
	return s, nil
}

func setServiceOptsDefaults(opts TestServiceOptions) TestServiceOptions {
	if opts.Logger == nil {
		opts.Logger = logrus.New()
	}

	if opts.TestsLogsPath == "" {
		opts.TestsLogsPath = DefaultTestLogsPath
	}

	return opts
}

func (s *TestService) Create(ctx context.Context, test *models.Test) error {
	if test.UserID == 0 {
		return ErrUserIDRequired
	}

	err := s.repo.Create(ctx, test)
	if database.IsDuplicateKeyError(err) {
		return ErrTestAlreadyExists
	} else if err != nil {
		return err
	}

	// TODO: currently this process is blocking the request until the knuu is ready
	// we need to make it non-blocking
	err = s.prepareKnuu(ctx, test)
	if err == nil {
		return nil
	}

	return errors.Join(err, s.repo.Delete(ctx, test.Scope))
}

func (s *TestService) Knuu(userID uint, scope string) (*knuu.Knuu, error) {
	s.knuuListMu.RLock()
	defer s.knuuListMu.RUnlock()

	kn, ok := s.knuuList[userID][scope]
	if !ok {
		return nil, ErrTestNotFound
	}

	return kn, nil
}

func (s *TestService) Delete(ctx context.Context, userID uint, scope string) error {
	if err := s.forceCleanupTest(ctx, userID, scope); err != nil {
		return err
	}

	return s.repo.Delete(ctx, scope)
}

func (s *TestService) Details(ctx context.Context, userID uint, scope string) (*models.Test, error) {
	return s.repo.Get(ctx, userID, scope)
}

func (s *TestService) List(ctx context.Context, userID uint, limit int, offset int) ([]models.Test, error) {
	return s.repo.List(ctx, userID, limit, offset)
}

func (s *TestService) Count(ctx context.Context, userID uint) (int64, error) {
	return s.repo.Count(ctx, userID)
}

func (s *TestService) Update(ctx context.Context, userID uint, scope string, test *models.Test) error {
	// for security reasons, these have to be explicitly set
	test.UserID = userID
	test.Scope = scope
	return s.repo.Update(ctx, test)
}

func (s *TestService) SetFinished(ctx context.Context, userID uint, scope string) error {
	test, err := s.repo.Get(ctx, userID, scope)
	if err != nil {
		return err
	}

	test.Finished = true
	return s.repo.Update(ctx, test)
}

func (s *TestService) Shutdown(ctx context.Context) error {
	close(s.stopCleanupChan)
	for _, logFile := range s.cleanup.logFiles {
		if logFile == nil {
			continue
		}

		if err := logFile.Close(); err != nil {
			return err
		}
	}
	s.cleanup.logFiles = nil

	for userID, users := range s.knuuList {
		for scope := range users {
			if err := s.cleanupIfFinishedTest(ctx, userID, scope); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *TestService) TestLogsPath(ctx context.Context, userID uint, scope string) (string, error) {
	// TODO: we need to apply roles here so Admins can access all tests logs
	// Check if the test exists and beongs to the user
	_, err := s.repo.Get(ctx, userID, scope)
	if err != nil {
		return "", err
	}

	logFilePath := filepath.Join(s.testsLogsPath, fmt.Sprintf("%s.log", scope))

	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		return "", ErrLogFileNotFound
	} else if err != nil {
		return "", err
	}

	return logFilePath, nil
}

func (s *TestService) cleanupIfFinishedTest(ctx context.Context, userID uint, scope string) error {
	running, err := s.isTestRunning(ctx, scope)
	if err != nil {
		return err
	}
	if !running {
		return nil
	}

	return s.forceCleanupTest(ctx, userID, scope)
}

func (s *TestService) forceCleanupTest(ctx context.Context, userID uint, scope string) error {
	if err := s.SetFinished(ctx, userID, scope); err != nil {
		return err
	}

	kn, ok := s.knuuList[userID][scope]
	if !ok {
		return nil
	}

	if err := kn.CleanUp(ctx); err != nil {
		return err
	}

	s.knuuListMu.Lock()
	defer s.knuuListMu.Unlock()

	delete(s.knuuList[userID], scope)
	if len(s.knuuList[userID]) == 0 {
		delete(s.knuuList, userID)
	}
	return nil
}

func (s *TestService) isTestRunning(ctx context.Context, scope string) (bool, error) {
	_, err := s.defaultK8sClient.GetNamespace(ctx, scope)
	if apierrs.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

func (s *TestService) loadRunningTestsFromDB(ctx context.Context) error {
	tests, err := s.repo.ListAllAlive(ctx)
	if err != nil {
		return err
	}

	for _, test := range tests {
		isRunning, err := s.isTestRunning(ctx, test.Scope)
		if err != nil {
			return err
		}
		if !isRunning {
			continue
		}

		err = s.prepareKnuu(ctx, &test)
		if err != nil && err != ErrTestAlreadyExists {
			return err
		}
	}

	return nil
}

func (s *TestService) prepareKnuu(ctx context.Context, test *models.Test) error {
	if err := k8s.ValidateNamespace(test.Scope); err != nil {
		return err
	}

	s.knuuListMu.Lock()
	if _, ok := s.knuuList[test.UserID]; !ok {
		s.knuuList[test.UserID] = make(map[string]*knuu.Knuu)
	}
	s.knuuListMu.Unlock()

	if test.Scope == "" {
		return ErrScopeRequired
	}

	_, ok := s.knuuList[test.UserID][test.Scope]
	if ok {
		return ErrTestAlreadyExists
	}

	logFile, err := os.OpenFile(
		filepath.Join(s.testsLogsPath, fmt.Sprintf("%s.log", test.Scope)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		LogsFilePermission,
	)
	if err != nil {
		s.logger.Errorf("opening log file for test %s: %v", test.Scope, err)
		return err
	}

	testLogger := logrus.New()
	testLogger.SetOutput(logFile)

	if test.LogLevel != "" {
		level, err := logrus.ParseLevel(test.LogLevel)
		if err != nil {
			return err
		}
		testLogger.SetLevel(level)
	}

	k8sClient, err := k8s.NewClient(ctx, test.Scope, testLogger)
	if err != nil {
		return err
	}

	var minioClient *minio.Minio
	if test.MinioEnabled {
		minioClient, err = minio.New(ctx, k8sClient, testLogger)
		if err != nil {
			return err
		}
	}

	if test.Deadline.IsZero() {
		test.Deadline = time.Now().Add(DefaultTestTimeout)
	}

	kn, err := knuu.New(ctx, knuu.Options{
		ProxyEnabled: test.ProxyEnabled,
		K8sClient:    k8sClient,
		MinioClient:  minioClient,
		Timeout:      time.Until(test.Deadline), // TODO: replace it with deadline when the deadline PR is merged
	})
	if err != nil {
		return err
	}

	s.knuuListMu.Lock()
	defer s.knuuListMu.Unlock()
	s.knuuList[test.UserID][test.Scope] = kn

	return nil
}

func (s *TestService) startPeriodicCleanup() {
	ticker := time.NewTicker(PeriodicCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performCleanup()
		case <-s.stopCleanupChan:
			s.logger.Info("TestService: Stopping periodic cleanup")
			return
		}
	}
}

func (s *TestService) performCleanup() {
	s.knuuListMu.RLock()
	userScopes := make(map[uint][]string)
	for userID, users := range s.knuuList {
		for scope := range users {
			userScopes[userID] = append(userScopes[userID], scope)
		}
	}
	s.knuuListMu.RUnlock()

	for userID, scopes := range userScopes {
		for _, scope := range scopes {
			s.logger.Debugf("TestService: Running periodic cleanup for userID: %d, scope: %s", userID, scope)
			if err := s.cleanupIfFinishedTest(context.Background(), userID, scope); err != nil {
				s.logger.Errorf("TestService: Error cleaning up test %s for user %d: %v", scope, userID, err)
			}
		}
	}
}
