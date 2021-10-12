// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporalite

import (
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/temporal"

	"github.com/DataDog/temporalite/internal/liteconfig"
)

// WithLogger overrides the default logger.
func WithLogger(logger log.Logger) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.Logger = logger
	})
}

// WithDatabaseFilePath persists state to the file at the specified path.
func WithDatabaseFilePath(filepath string) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.Ephemeral = false
		cfg.DatabaseFilePath = filepath
	})
}

// WithPersistenceDisabled disables file persistence and uses the in-memory storage driver. State will be reset on each process restart.
func WithPersistenceDisabled() ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.Ephemeral = true
	})
}

// WithFrontendPort sets the listening port for the temporal-frontend GRPC service.
// When unspecified, the default port number of 7233 is used.
func WithFrontendPort(port int) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.FrontendPort = port
	})
}

// WithDynamicPorts starts Temporal on system-chosen ports.
func WithDynamicPorts() ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.DynamicPorts = true
	})
}

// WithNamespaces registers each namespace on Temporal start.
func WithNamespaces(namespaces ...string) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		cfg.Namespaces = append(cfg.Namespaces, namespaces...)
	})
}

// WithInterruptOn registers a channel that interrupts the server on the signal
// from that channel.
func WithInterruptOn(interruptCh <-chan interface{}) ServerOption {
	return newApplyFuncContainer(func(cfg *liteconfig.Config) {
		option := temporal.InterruptOn(interruptCh)
		cfg.InterruptOn = &option
	})
}

type applyFuncContainer struct {
	applyInternal func(*liteconfig.Config)
}

func (fso *applyFuncContainer) apply(cfg *liteconfig.Config) {
	fso.applyInternal(cfg)
}

func newApplyFuncContainer(apply func(*liteconfig.Config)) *applyFuncContainer {
	return &applyFuncContainer{
		applyInternal: apply,
	}
}
