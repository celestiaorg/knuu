package instance

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/pkg/system"
)

type SidecarManager interface {
	// namePrefix is the name of the instance that is the parent of the sidecar
	// it is used to set the name of the sidecar to avoid name collisions
	Initialize(ctx context.Context, namePrefix string, sysDeps *system.SystemDependencies) error
	Instance() *Instance
	PreStart(ctx context.Context) error
	Clone(namePrefix string) (SidecarManager, error)
}

type sidecars struct {
	instance  *Instance
	isSidecar bool             // indicating that if the current instance is a sidecar
	sidecars  []SidecarManager // the sidecars for the current instance
}

func (i *Instance) Sidecars() *sidecars {
	return i.sidecars
}

func (s *sidecars) SetIsSidecar(isSidecar bool) {
	s.isSidecar = isSidecar
}

func (s *sidecars) IsSidecar() bool {
	return s.isSidecar
}

// Add adds a sidecar to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (s *sidecars) Add(ctx context.Context, sc SidecarManager) error {
	if sc == nil {
		return ErrSidecarIsNil
	}
	if !s.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingSidecarNotAllowed.WithParams(s.instance.state.String())
	}

	if err := sc.Initialize(ctx, s.instance.Name(), s.instance.SystemDependencies); err != nil {
		return ErrInitializingSidecar.WithParams(s.instance.name).Wrap(err)
	}

	if sc.Instance() == nil {
		return ErrSidecarInstanceIsNil.WithParams(s.instance.name)
	}

	if !sc.Instance().IsInState(StateCommitted) {
		return ErrSidecarNotCommitted.WithParams(sc.Instance().Name())
	}
	if s.isSidecar {
		return ErrSidecarCannotHaveSidecar.WithParams(s.instance.name)
	}

	s.sidecars = append(s.sidecars, sc)
	sc.Instance().parentInstance = s.instance
	s.instance.Logger.WithFields(logrus.Fields{
		"sidecar":  sc.Instance().Name(),
		"instance": s.instance.name,
	}).Debug("added sidecar to instance")
	return nil
}

// verifySidecarsStates verifies that all sidecars are in the state 'Committed' or 'Stopped'
func (s *sidecars) verifySidecarsStates() error {
	for _, sc := range s.sidecars {
		if !sc.Instance().IsInState(StateCommitted, StateStopped) {
			return ErrStartingNotAllowedForSidecar.
				WithParams(sc.Instance().Name(), sc.Instance().state.String())
		}
	}
	return nil
}

// applyFunctionToSidecars applies a function to all sidecars
func (s *sidecars) applyFunctionToSidecars(fn func(sc SidecarManager) error) error {
	for _, i := range s.sidecars {
		if err := fn(i); err != nil {
			return ErrApplyingFunctionToSidecar.WithParams(i.Instance().name).Wrap(err)
		}
	}
	return nil
}

func (s *sidecars) setStateForSidecars(state InstanceState) {
	// We don't handle errors here, as the function can't return an error
	_ = s.applyFunctionToSidecars(
		func(sc SidecarManager) error {
			sc.Instance().state = state
			return nil
		})
}

func (s *sidecars) clone(namePrefix string) (*sidecars, error) {
	clonedSidecars := make([]SidecarManager, len(s.sidecars))
	for i, sc := range s.sidecars {
		cloned, err := sc.Clone(namePrefix)
		if err != nil {
			return nil, err
		}
		clonedSidecars[i] = cloned
	}
	return &sidecars{
		sidecars: clonedSidecars,
	}, nil
}
