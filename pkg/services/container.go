package services

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"golang.org/x/time/rate"

	"github.com/tiny-blob/tinyblob/config"
)

// Container for all the application services
type Container struct {
	// Cache stores the cache client
	Cache *CacheClient
	// Config stores the application configuration
	Config *config.Config
	// Web stores the web framework
	Web *echo.Echo
	// SolRPC stores the solana RPC client
	SolRPC *rpc.Client
	// TemplateRenderer stores the template renderer
	TemplateRenderer *TemplateRenderer
	// Validator stores the validator
	Validator *Validator
}

// NewContainer creates and initialize a new container
func NewContainer() *Container {
	c := new(Container)
	c.initConfig()
	c.initValidator()
	c.initWeb()
	c.initCache()
	c.initTemplateRenderer()
	c.initSolRPC()

	return c
}

// Shutdown the container and disconnect from all connections
func (c *Container) Shutdown() error {
	if err := c.Cache.Close(); err != nil {
		return err
	}

	return nil
}

// Application configuration
func (c *Container) initConfig() {
	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	c.Config = &cfg
}

// initValidator initializes the validator
func (c *Container) initValidator() {
	c.Validator = NewValidator()
}

// initWeb initializes the web server
func (c *Container) initWeb() {
	c.Web = echo.New()
	c.Web.HideBanner = true

	switch c.Config.App.Environment {
	case config.EnvProd:
		c.Web.Logger.SetLevel(log.WARN)
	default:
		c.Web.Logger.SetLevel(log.DEBUG)
	}

	c.Web.Validator = c.Validator
}

// initCache initializes the cache client
func (c *Container) initCache() {
	var err error
	if c.Cache, err = NewCacheClient(c.Config); err != nil {
		panic(err)
	}
}

// initTemplateRenderer initializes the template renderer
func (c *Container) initTemplateRenderer() {
	c.TemplateRenderer = NewTemplateRenderer(c.Config)
}

// initSolRPC initializes the solana RPC client
func (c *Container) initSolRPC() {
	var cluster rpc.Cluster

	switch c.Config.App.Environment {
	case config.EnvProd:
		cluster = rpc.MainNetBeta
	case config.EnvDev:
		cluster = rpc.DevNet
	case config.EnvLocal:
		cluster = rpc.LocalNet
	default:
		cluster = rpc.TestNet
	}

	rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
		cluster.RPC,
		rate.Every(time.Second),
		5,
	))

	c.SolRPC = rpcClient
}
