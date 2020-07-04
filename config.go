package core

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/go-chi/cors"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// Configuration indicates a struct contains a core.Config and can be logged.
type Configuration interface {
	zapcore.ObjectMarshaler
	CoreConfig() *Config
}

// Config contains the configuration about core.
type Config struct {
	// Path is the argument given to the -c flag.
	Path string `mapstructure:"-" validate:"-"`
	// Env indicates the current environment of the application (development, staging, production).
	Env string `mapstructure:"env" validate:"required,oneof=development staging production"`
	// Deploy
	Deploy Deploy `mapstructure:"deploy" validate:"required"`
	// Server contains the configuration about the server.
	Server ServerConfig `mapstructure:"server" validate:"required"`
	// Database contains the configuration about database connections.
	// This is optional, if no database configuration is found, then no database connection is created.
	Database DatabaseConfig `mapstructure:"database" validate:""`
}

type Deploy struct {
	Docker DockerConfig `mapstructure:"docker"`
	GCloud GCloudConfig `mapstructure:"gcloud"`
}

type DockerConfig struct {
	Name string `mapstructure:"name"`
}

type GCloudConfig struct {
	Service  string `mapstructure:"service"`
	Platform string `mapstructure:"platform"`
	Region   string `mapstructure:"region"`
}

// ServerConfig contains the configuration about the server.
type ServerConfig struct {
	// Port determines the port the server runs on.
	Port int `mapstructure:"port" validate:"required,min=1"`
	// Cors contains the configuration about CORS.
	Cors CorsConfig `mapstructure:"cors" validate:"required"`
	// Jwt contains the configuration about JSON web tokens.
	Jwt JwtConfig `mapstructure:"jwt" validate:"required"`
	// Log contains the configuration about logging.
	Log LogConfig `mapstructure:"log" validate:"required"`
	// Graphql contains the configuration about GraphQL.
	Graphql GraphqlConfig `mapstructure:"graphql" validate:"required"`
}

// corsOptions
func (svr *ServerConfig) corsOptions() cors.Options {
	return cors.Options{
		MaxAge:             svr.Cors.MaxAge,
		AllowCredentials:   svr.Cors.AllowCredentials,
		AllowedOrigins:     svr.Cors.AllowedOrigins,
		AllowedMethods:     svr.Cors.AllowedMethods,
		AllowedHeaders:     svr.Cors.AllowedHeaders,
		ExposedHeaders:     nil,
		OptionsPassthrough: false,
		Debug:              false,
	}
}

// CorsConfig contains the configuration about CORS.
type CorsConfig struct {
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge int `mapstructure:"max_age" validate:"min=0"`
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool `mapstructure:"allow_credentials" validate:"required"`
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// An origin may contain a wildcard (*) to replace 0 or more characters
	// (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penalty.
	// Only one wildcard can be used per origin.
	// Default value is ["*"]
	AllowedOrigins []string `mapstructure:"allowed_origins" validate:"required,min=1"`
	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (HEAD, GET and POST).
	AllowedMethods []string `mapstructure:"allowed_methods" validate:"required,min=1"`
	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [] but "Origin" is always appended to the list.
	AllowedHeaders []string `mapstructure:"allowed_headers" validate:"required,min=1"`
}

// JwtConfig contains the configuration about JSON web tokens.
type JwtConfig struct {
	// The "aud" (audience) claim identifies the recipients that the JWT is
	// intended for.  Each principal intended to process the JWT MUST
	// identify itself with a value in the audience claim.  If the principal
	// processing the claim does not identify itself with a value in the
	// "aud" claim when this claim is present, then the JWT MUST be
	// rejected.  In the general case, the "aud" value is an array of case-
	// sensitive strings, each containing a StringOrURI value.  In the
	// special case when the JWT has one audience, the "aud" value MAY be a
	// single case-sensitive string containing a StringOrURI value.  The
	// interpretation of audience values is generally application specific.
	// Use of this claim is OPTIONAL.
	Audience []string `mapstructure:"audience" validate:"required,min=1"`
	// The "iss" (issuer) claim identifies the principal that issued the
	// JWT.  The processing of this claim is generally application specific.
	// The "iss" value is a case-sensitive string containing a StringOrURI
	// value.  Use of this claim is OPTIONAL.
	Issuer string `mapstructure:"issuer" validate:"required"`
	// The "exp" (expiration time) claim identifies the expiration time on
	// or after which the JWT MUST NOT be accepted for processing.  The
	// processing of the "exp" claim requires that the current date/time
	// MUST be before the expiration date/time listed in the "exp" claim.
	ExpiresAt time.Duration `mapstructure:"expires_at" validate:"required"`
	// The "nbf" (not before) claim identifies the time before which the JWT
	// MUST NOT be accepted for processing.  The processing of the "nbf"
	// claim requires that the current date/time MUST be after or equal to
	// the not-before date/time listed in the "nbf" claim.  Implementers MAY
	// provide for some small leeway, usually no more than a few minutes, to
	// account for clock skew.  Its value MUST be a number containing a
	// NumericDate value.  Use of this claim is OPTIONAL.
	NotBefore time.Duration `mapstructure:"not_before" validate:"min=0"`
	// Secret is the key that is used to sign the JWT.
	Secret string `mapstructure:"secret" validate:"required,min=20"`
}

// Log contains the configuration about logging.
type LogConfig struct {
	// Level indicates the level the application should log at. Any levels greater than or equal to
	// this will be logged. Log levels from least severe to highest: debug, info, warn, error
	Level string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
}

// GraphqlConfig contains the configuration about GraphQL.
type GraphqlConfig struct {
	// Schema indicates where the schema.graphql file is located.
	Schema string `mapstructure:"schema" validate:"required,file"`
}

// DatabaseConfig contains the configuration about the Database.
type DatabaseConfig struct {
	// Migrations
	Migrations struct {
		Location   string `mapstructure:"location" validate:"url"`
		RunOnStart bool   `mapstructure:"run_on_start" validate:"required"`
	} `mapstructure:"migrations" validate:""`
	// Main
	Main DatabaseConnectionConfig `mapstructure:"main" validate:""`
	// Test
	Test DatabaseConnectionConfig `mapstructure:"test" validate:"-"`
	// Models
	Models struct {
		Wipe            bool   `mapstructure:"wipe" validate:"required" toml:"wipe"`
		Output          string `mapstructure:"output" validate:"required" toml:"output"`
		StructTagCasing string `mapstructure:"struct-tag-casing" validate:"required" toml:"struct-tag-casing"`
	} `mapstructure:"models" validate:"required"`
}

// DatabaseConnectionConfig contains the configuration about database connections.
type DatabaseConnectionConfig struct {
	// Driver indicates the database driver to be used.
	Driver string `mapstructure:"driver" validate:"required" toml:"driver"`
	// Host indicates the host of the database server.
	Host string `mapstructure:"host" validate:"required" toml:"host"`
	// Port indicates the port of the database server.
	Port int `mapstructure:"port" validate:"required" toml:"port"`
	// Dbname indicates the name of the database.
	Dbname string `mapstructure:"dbname" validate:"required" toml:"dbname"`
	// User indicates the username of the user you want to connect the database with.
	User string `mapstructure:"user" validate:"required" toml:"user"`
	// Password indicates the password of the user you want to connect the database with.
	Password string `mapstructure:"password" validate:"required" toml:"pass"`
	// Schema indicates the name of the schema you want to connect to.
	Schema string `mapstructure:"schema" validate:"required" toml:"schema" toml:"schema"`
	// Sslmode indicates the ssl mode you want to connect to the database with.
	Sslmode string `mapstructure:"sslmode" validate:"required" toml:"sslmode" toml:"sslmode"`
}

// DataSourceName returns a connection string in the format of "user={user} password={password} dbname={dbname} host={host} port={port} sslmode={sslmode}"
func (dcc *DatabaseConnectionConfig) DataSourceName() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=%s", dcc.User, dcc.Password, dcc.Dbname, dcc.Host, dcc.Port, dcc.Sslmode)
}

// ConnectionString returns a connection string in the format of "{driver}://{user}:{password}@{host}:{port}/{dbname}?sslmode={sslmode}"
func (dcc DatabaseConnectionConfig) ConnectionString() string {
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=%s", dcc.Driver, dcc.User, dcc.Password, dcc.Host, dcc.Port, dcc.Dbname, dcc.Sslmode)
}

// CoreConfig is used to implement the core.Configuration interface.
func (cfg *Config) CoreConfig() *Config {
	return cfg
}

// MarshalLogObject is used to implement the zapcore.ObjectMarshaler interface.
func (cfg *Config) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("env", cfg.Env)
	enc.AddString("path", cfg.Path)
	_ = enc.AddObject("server", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddInt("port", cfg.Server.Port)
		_ = enc.AddObject("cors", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddInt("maxAge", cfg.Server.Cors.MaxAge)
			enc.AddBool("allowCredentials", cfg.Server.Cors.AllowCredentials)
			_ = enc.AddArray("allowedOrigins", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
				for _, origin := range cfg.Server.Cors.AllowedOrigins {
					enc.AppendString(origin)
				}
				return nil
			}))
			_ = enc.AddArray("allowedMethods", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
				for _, method := range cfg.Server.Cors.AllowedMethods {
					enc.AppendString(method)
				}
				return nil
			}))
			_ = enc.AddArray("allowedHeaders", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
				for _, header := range cfg.Server.Cors.AllowedHeaders {
					enc.AppendString(header)
				}
				return nil
			}))
			return nil
		}))
		_ = enc.AddObject("jwt", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			_ = enc.AddArray("audience", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
				for _, audience := range cfg.Server.Jwt.Audience {
					enc.AppendString(audience)
				}
				return nil
			}))
			enc.AddString("issuer", cfg.Server.Jwt.Issuer)
			enc.AddString("expiresAt", cfg.Server.Jwt.ExpiresAt.String())
			enc.AddString("notBefore", cfg.Server.Jwt.NotBefore.String())
			return nil
		}))
		_ = enc.AddObject("log", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("level", cfg.Server.Log.Level)
			return nil
		}))
		_ = enc.AddObject("graphql", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("schema", cfg.Server.Graphql.Schema)
			return nil
		}))
		return nil
	}))
	_ = enc.AddObject("database", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		_ = enc.AddObject("migrations", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("location", cfg.Database.Migrations.Location)
			enc.AddBool("run_on_start", cfg.Database.Migrations.RunOnStart)
			return nil
		}))
		_ = enc.AddObject("main", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("driver", cfg.Database.Main.Driver)
			enc.AddString("dbname", cfg.Database.Main.Dbname)
			enc.AddString("host", cfg.Database.Main.Host)
			enc.AddInt("port", cfg.Database.Main.Port)
			enc.AddString("user", cfg.Database.Main.User)
			enc.AddString("schema", cfg.Database.Main.Schema)
			enc.AddString("sslmode", cfg.Database.Main.Sslmode)
			return nil
		}))
		_ = enc.AddObject("test", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("driver", cfg.Database.Test.Driver)
			enc.AddString("dbname", cfg.Database.Test.Dbname)
			enc.AddString("host", cfg.Database.Test.Host)
			enc.AddInt("port", cfg.Database.Test.Port)
			enc.AddString("user", cfg.Database.Test.User)
			enc.AddString("schema", cfg.Database.Test.Schema)
			enc.AddString("sslmode", cfg.Database.Test.Sslmode)
			return nil
		}))
		_ = enc.AddObject("models", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("output", cfg.Database.Models.Output)
			enc.AddString("struct-tag-casing", cfg.Database.Models.StructTagCasing)
			enc.AddBool("wipe", cfg.Database.Models.Wipe)
			return nil
		}))
		return nil
	}))
	return nil
}

// LoadConfig takes a reference to a struct that contains a core.Config.
// It uses the --config flag to locate the configuration file and maps it to the referenced struct using viper.
//
// If any of the following occurs, it terminates the application with a fatal error:
// - config flag not provided
// - cannot locate the configuration file
// - cannot map the configuration file to the referenced struct
// - the referenced struct is not valid
func LoadConfig(cfg Configuration) {
	var path string
	flag.StringVar(&path, "config", "", "path to the config file")
	flag.Parse()

	if path == "" {
		log.Fatal("you must pass the path to the config file using the --config argument")
	}

	if strings.HasPrefix(path, "gcloud://") {
		// the path is a secret in google cloud's secret manager
		ctx := context.Background()
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			log.Fatalf("failed to setup google cloud secret manager client: %v", err)
		}

		// remove the "gcloud://" prefix
		req := &secretmanagerpb.AccessSecretVersionRequest{Name: path[9:]}
		result, err := client.AccessSecretVersion(ctx, req)
		if err != nil {
			log.Fatalf("failed to access secret: %v", err)
		}

		viper.SetConfigType("toml")
		err = viper.ReadConfig(bytes.NewBuffer(result.Payload.Data))
		if err != nil {
			log.Fatalf("failed to read config from secret: %v", err)
		}
	} else {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			log.Fatalf("failed to get absolute path to config file: %v", err)
		}

		viper.SetConfigFile(path)
		err = viper.ReadInConfig()
		if err != nil {
			log.Fatalf("failed to read in the config: %v", err)
		}
	}

	err := viper.Unmarshal(cfg)
	if err != nil {
		log.Fatalf("failed to unmarshal the config to your struct: %v", err)
	}

	// make absolute paths
	coreCfg := cfg.CoreConfig()
	coreCfg.Path = path
	coreCfg.Server.Graphql.Schema, err = filepath.Abs(coreCfg.Server.Graphql.Schema)
	if err != nil {
		log.Fatalf("failed to get absolute path to graphql schema: %v", err)
	}

	envPort := os.Getenv("PORT")
	if envPort != "" {
		// PORT environment variable always takes precedence as many hosting platforms require its use
		coreCfg.Server.Port, err = strconv.Atoi(envPort)
		if err != nil {
			log.Fatalf("failed to parse PORT from environment variable: %v", err)
		}
	}

	err = validate.Struct(cfg)
	if err != nil {
		fmt.Println("config failed validation")
		for _, e := range err.(validator.ValidationErrors) {
			fmt.Printf("  - %s: %s\n", e.StructNamespace(), e.Translate(uni.GetFallback()))
		}
		os.Exit(1)
	}
}
