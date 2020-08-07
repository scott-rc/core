package testsuite

import (
	"template/app"
	"template/loader"
	"template/models"
	"template/resolver"
	"template/test/testsuite/factory"
	"testing"

	"github.com/DATA-DOG/go-txdb"
	_ "github.com/lib/pq"
	"github.com/scott-rc/core"
	"github.com/scott-rc/core/coretest"
	"github.com/stretchr/testify/require"
)

func init() {
	txdb.Register("test", Config.Database.Test.Driver, Config.Database.Test.DataSourceName())
}

var Config = core.Config{
	Database: &core.DatabaseConfig{
		Test: &core.DatabaseConnectionConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Dbname:   "test",
			User:     "test",
			Password: "test",
			Schema:   "public",
			Sslmode:  "disable",
		},
	},
}

type Suite struct {
	*require.Assertions
	t        *testing.T
	Resolver *resolver.Resolver
	Core     *app.Core
	Factory  *factory.Factory
	User     *models.User
}

func (s *Suite) SignIn(overrides map[string]interface{}) {
	s.User = s.Factory.CreateUser(overrides)
	s.Core.Session.Login(s.User.UserID)
}

func (s *Suite) SetupSuite() {
	// before all
}

func (s *Suite) SetupTest() {
	// before each
	c := coretest.NewCore(s.t, coretest.Options{
		Config:                   &Config,
		ResolverContextDecorator: app.ContextDecorator(),
	})

	s.Core = &app.Core{Core: c, Loader: loader.NewLoader()}
	s.Resolver = &resolver.Resolver{}
	s.Factory = factory.NewFactory(s.Core, s.Assertions)
}

func (s *Suite) TearDownTest() {
	// after each
	s.NoError(s.Core.Db.Close())
}

func (s *Suite) TearDownSuite() {
	// after all
}

func (s *Suite) T() *testing.T {
	return s.t
}

func (s *Suite) SetT(t *testing.T) {
	s.t = t
	s.Assertions = require.New(t)
}
