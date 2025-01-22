package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/celestiaorg/knuu/internal/database/models"
	"github.com/celestiaorg/knuu/internal/database/repos"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/celestiaorg/knuu/pkg/minio"
	"github.com/sirupsen/logrus"
)

type TestService struct {
	repo       *repos.TestRepository
	knuuList   map[uint]map[string]*knuu.Knuu // key is the user ID, second key is the scope
	knuuListMu sync.RWMutex
}

func NewTestService(ctx context.Context, repo *repos.TestRepository) (*TestService, error) {
	s := &TestService{
		repo:     repo,
		knuuList: make(map[uint]map[string]*knuu.Knuu),
	}

	if err := s.loadKnuuFromDB(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *TestService) Create(ctx context.Context, test *models.Test) error {
	if test.UserID == 0 {
		return ErrUserIDRequired
	}

	if err := s.prepareKnuu(ctx, test); err != nil {
		return err
	}

	return s.repo.Create(ctx, test)
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
	s.knuuListMu.Lock()
	defer s.knuuListMu.Unlock()

	kn, ok := s.knuuList[userID][scope]
	if !ok {
		return nil
	}

	if err := kn.CleanUp(ctx); err != nil {
		return err
	}

	delete(s.knuuList[userID], scope)
	if len(s.knuuList[userID]) == 0 {
		delete(s.knuuList, userID)
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

func (s *TestService) Update(test *models.Test) error {
	// Update the knuu object if needed e.g. deadline(timeout),...
	// return s.repo.Update(test)
	return fmt.Errorf("not implemented")
}

func (s *TestService) loadKnuuFromDB(ctx context.Context) error {
	tests, err := s.repo.ListAllAlive(ctx)
	if err != nil {
		return err
	}

	for _, test := range tests {
		err := s.prepareKnuu(ctx, &test)
		if err != nil && err != ErrTestAlreadyExists {
			return err
		}
	}

	return nil
}

func (s *TestService) prepareKnuu(ctx context.Context, test *models.Test) error {
	s.knuuListMu.Lock()
	defer s.knuuListMu.Unlock()

	if _, ok := s.knuuList[test.UserID]; !ok {
		s.knuuList[test.UserID] = make(map[string]*knuu.Knuu)
	}

	if test.Scope == "" {
		return ErrScopeRequired
	}

	_, ok := s.knuuList[test.UserID][test.Scope]
	if ok {
		return ErrTestAlreadyExists
	}

	var (
		logger      = logrus.New()
		minioClient *minio.Minio
	)

	k8sClient, err := k8s.NewClient(ctx, test.Scope, logger)
	if err != nil {
		return err
	}

	if test.MinioEnabled {
		minioClient, err = minio.New(ctx, k8sClient, logger)
		if err != nil {
			return err
		}
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
	s.knuuList[test.UserID][test.Scope] = kn

	return nil
}
