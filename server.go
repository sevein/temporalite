// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporalite

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/DataDog/temporalite/internal/liteconfig"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/schema/sqlite"
	"go.temporal.io/server/temporal"
)

// Server wraps a temporal.Server.
type Server struct {
	internal         *temporal.Server
	frontendHostPort string
	config           *liteconfig.Config
}

type ServerOption interface {
	apply(*liteconfig.Config)
}

// NewServer returns a new instance of Server.
func NewServer(opts ...ServerOption) (*Server, error) {
	c, err := liteconfig.NewDefaultConfig()
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	cfg := liteconfig.Convert(c)
	sqlConfig := cfg.Persistence.DataStores[liteconfig.PersistenceStoreName].SQL

	// Apply migrations if file does not already exist
	if c.Ephemeral {
		if err := sqlite.SetupSchema(sqlConfig); err != nil {
			return nil, fmt.Errorf("error setting up schema: %w", err)
		}
	} else if _, err := os.Stat(c.DatabaseFilePath); os.IsNotExist(err) {
		if err := sqlite.SetupSchema(sqlConfig); err != nil {
			return nil, fmt.Errorf("error setting up schema: %w", err)
		}
	}

	// Pre-create namespaces
	var namespaces []*sqlite.NamespaceConfig
	for _, ns := range c.Namespaces {
		namespaces = append(namespaces, sqlite.NewNamespaceConfig(cfg.ClusterMetadata.CurrentClusterName, ns, false))
	}
	if err := sqlite.CreateNamespaces(sqlConfig, namespaces...); err != nil {
		return nil, fmt.Errorf("error creating namespaces: %w", err)
	}

	authorizer, err := authorization.GetAuthorizerFromConfig(&cfg.Global.Authorization)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate authorizer: %w", err)
	}

	claimMapper, err := authorization.GetClaimMapperFromConfig(&cfg.Global.Authorization, c.Logger)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate claim mapper: %w", err)
	}

	serverOpts := []temporal.ServerOption{
		temporal.WithConfig(cfg),
		temporal.ForServices(temporal.Services),
		temporal.WithLogger(c.Logger),
		temporal.WithAuthorizer(authorizer),
		temporal.WithClaimMapper(func(cfg *config.Config) authorization.ClaimMapper {
			return claimMapper
		}),
		temporal.WithDynamicConfigClient(dynamicconfig.NewNoopClient()),
	}
	if c.InterruptOn != nil {
		serverOpts = append(serverOpts, *c.InterruptOn)
	}

	s := &Server{
		internal:         temporal.NewServer(serverOpts...),
		frontendHostPort: cfg.PublicClient.HostPort,
		config:           c,
	}

	return s, nil
}

// Start temporal server.
func (s *Server) Start() error {
	return s.internal.Start()
}

// Stop the server.
func (s *Server) Stop() {
	s.internal.Stop()
}

// NewClient initializes a client ready to communicate with the Temporal
// server in the target namespace.
func (s *Server) NewClient(ctx context.Context, namespace string) (client.Client, error) {
	return s.NewClientWithOptions(ctx, client.Options{Namespace: namespace})
}

// NewClientWithOptions is the same as NewClient but allows further customization.
//
// To set the client's namespace, use the corresponding field in client.Options.
//
// Note that the HostPort and ConnectionOptions fields of client.Options will always be overridden.
func (s *Server) NewClientWithOptions(ctx context.Context, options client.Options) (client.Client, error) {
	options.HostPort = s.frontendHostPort
	options.ConnectionOptions = client.ConnectionOptions{
		DisableHealthCheck: false,
		HealthCheckTimeout: timeoutFromContext(ctx, time.Minute),
	}
	return client.NewClient(options)
}

func timeoutFromContext(ctx context.Context, defaultTimeout time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return deadline.Sub(time.Now())
	}
	return defaultTimeout
}
