package netshaper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/bittwister/sdk"
	"github.com/celestiaorg/knuu/pkg/system"
)

type TestSuite struct {
	suite.Suite
	bt         *NetShaper
	ctx        context.Context
	sysDeps    *system.SystemDependencies
	mockServer *httptest.Server
}

func (s *TestSuite) SetupTest() {
	s.bt = New()
	s.ctx = context.Background()
	s.sysDeps = &system.SystemDependencies{
		Logger: logrus.New(),
	}

	s.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Since the only test that checks the response is the WaitForStart test,
		// we can hardcode the response and return it for all requests.
		expectedOutput := []sdk.ServiceStatus{{
			Name:                 "test-service",
			Ready:                true,
			NetworkInterfaceName: "eth0",
			Params:               map[string]interface{}{"key": "value"},
		}}

		jsonBytes, err := json.Marshal(expectedOutput)
		s.Require().NoError(err)
		_, err = w.Write(jsonBytes)
		s.Require().NoError(err)
	}))
}

func (s *TestSuite) TearDownTest() {
	s.mockServer.Close()
}

func TestNetShaper(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestNew() {
	bt := New()
	s.Assert().NotNil(bt)
	s.Assert().Equal(DefaultImage, bt.image)
	s.Assert().Equal(DefaultPort, bt.port)
	s.Assert().Equal(DefaultNetworkInterface, bt.networkInterface)
}

func (s *TestSuite) TestInitialize() {
	err := s.bt.Initialize(s.ctx, "test-init", s.sysDeps)
	s.Require().NoError(err)
	s.Assert().NotNil(s.bt.Instance())
	s.Assert().Equal(DefaultImage, s.bt.Instance().Build().ImageName())
	s.Assert().True(s.bt.Instance().Sidecars().IsSidecar())
}

func (s *TestSuite) TestPreStart() {
	s.T().Skip("skipping as it is tested in e2e tests")
}

func (s *TestSuite) TestClone() {
	err := s.bt.Initialize(s.ctx, "test-clone", s.sysDeps)
	s.Require().NoError(err)
	s.Require().NotNil(s.bt.instance, "Instance should be initialized before cloning")

	clonePrefixName := "test-clone-prefix"
	clone, err := s.bt.Clone(clonePrefixName)
	s.Require().NoError(err)
	s.Assert().NotNil(clone)

	clonedBt, ok := clone.(*NetShaper)
	s.Assert().True(ok)

	s.Assert().Equal(s.bt.port, clonedBt.port)
	s.Assert().Equal(s.bt.image, clonedBt.image)
	s.Assert().Equal(s.bt.networkInterface, clonedBt.networkInterface)
	s.Assert().Nil(clonedBt.client)
	s.Assert().NotNil(clonedBt.instance, "Cloned instance should not be nil")
	s.Assert().NotEqual(s.bt.instance, clonedBt.instance, "Cloned instance should be a new object")
	s.Assert().Equal(s.bt.instance.Build().ImageName(), clonedBt.instance.Build().ImageName())
	s.Assert().Equal(clonePrefixName+"-"+instanceName, clonedBt.instance.Name())
	clonedBt.SetPort(9090)
	s.Assert().NotEqual(s.bt.port, clonedBt.port)
}

func (s *TestSuite) TestCloneWithCustomValues() {
	err := s.bt.Initialize(s.ctx, "test-clone-custom", s.sysDeps)
	s.Require().NoError(err)
	s.Require().NotNil(s.bt.instance, "Instance should be initialized before cloning")

	s.bt.SetPort(8080)
	s.bt.SetImage("nginx")
	s.bt.SetNetworkInterface("eth0")

	clonePrefixName := "test-clone-custom-prefix"
	clone, err := s.bt.Clone(clonePrefixName)
	s.Require().NoError(err)
	s.Assert().NotNil(clone)

	clonedBt, ok := clone.(*NetShaper)
	s.Assert().True(ok)

	s.Assert().Equal(s.bt.port, clonedBt.port)
	s.Assert().Equal(s.bt.image, clonedBt.image)
	s.Assert().Equal(s.bt.networkInterface, clonedBt.networkInterface)
	s.Assert().Nil(clonedBt.client)
	s.Assert().NotNil(clonedBt.instance, "Cloned instance should not be nil")
	s.Assert().NotEqual(s.bt.instance, clonedBt.instance, "Cloned instance should be a new object")
	s.Assert().Equal(s.bt.instance.Build().ImageName(), clonedBt.instance.Build().ImageName())
	s.Assert().Equal(clonePrefixName+"-"+instanceName, clonedBt.instance.Name())

	clonedBt.SetPort(9090)
	s.Assert().NotEqual(s.bt.port, clonedBt.port)
}

func (s *TestSuite) TestSetters() {
	s.bt.SetPort(8080)
	s.Assert().Equal(8080, s.bt.port)

	s.bt.SetImage("test-image")
	s.Assert().Equal("test-image", s.bt.image)

	s.bt.SetNetworkInterface("test-if")
	s.Assert().Equal("test-if", s.bt.networkInterface)
}

func (s *TestSuite) TestSetBandwidthLimit() {
	tests := []struct {
		name  string
		limit int64
		err   error
	}{
		{"Valid limit", 1000, nil},
		{"Invalid client", 1000, ErrBitTwisterNotInitialized},
	}

	for _, tt := range tests {
		tt := tt
		s.Run(tt.name, func() {
			s.bt.client = nil
			if tt.err == nil {
				s.bt.client = sdk.NewClient(s.mockServer.URL)
			}
			err := s.bt.SetBandwidthLimit(tt.limit)
			if tt.err != nil {
				s.Assert().Error(err)
				return
			}
			s.Assert().NoError(err)
		})
	}
}

func (s *TestSuite) TestSetLatencyAndJitter() {
	tests := []struct {
		name    string
		latency int64
		jitter  int64
		err     error
	}{
		{"Valid latency and jitter", 1000, 1000, nil},
		{"Invalid client", 1000, 1000, ErrBitTwisterNotInitialized},
	}

	for _, tt := range tests {
		tt := tt
		s.Run(tt.name, func() {
			s.bt.client = nil
			if tt.err == nil {
				s.bt.client = sdk.NewClient(s.mockServer.URL)
			}
			err := s.bt.SetLatencyAndJitter(tt.latency, tt.jitter)
			if tt.err != nil {
				s.Assert().Error(err)
				return
			}
			s.Assert().NoError(err)
		})
	}
}

func (s *TestSuite) TestSetPacketLoss() {
	tests := []struct {
		name       string
		packetLoss int32
		err        error
	}{
		{"Valid packet loss", 10, nil},
		{"Invalid client", 10, ErrBitTwisterNotInitialized},
	}

	for _, tt := range tests {
		tt := tt
		s.Run(tt.name, func() {
			s.bt.client = nil
			if tt.err == nil {
				s.bt.client = sdk.NewClient(s.mockServer.URL)
			}
			err := s.bt.SetPacketLoss(tt.packetLoss)
			if tt.err != nil {
				s.Assert().Error(err)
				return
			}
			s.Assert().NoError(err)
		})
	}
}

func (s *TestSuite) TestWaitForStart() {
	tests := []struct {
		name     string
		client   *sdk.Client
		expected error
	}{
		{"Valid start", sdk.NewClient(s.mockServer.URL), nil},
		{"Invalid client", nil, ErrBitTwisterNotInitialized},
	}

	for _, tt := range tests {
		tt := tt
		s.Run(tt.name, func() {
			s.bt.client = tt.client
			err := s.bt.WaitForStart(s.ctx)
			if tt.expected != nil {
				s.Assert().Error(err)
				return
			}
			s.Assert().NoError(err)
		})
	}
}
