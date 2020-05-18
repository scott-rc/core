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

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/go-chi/cors"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

type Config struct {
	// Path is the argument given to the -c flag.
	Path string `mapstructure:"-" validate:"-" json:"path"`
	// Env indicates the current environment of the application (development, staging, production).
	Env string `mapstructure:"env" validate:"required,oneof=development staging production" json:"env"`
	// Server contains the configuration about the server.
	Server ServerConfig `mapstructure:"server" validate:"required" json:"server"`
	// Log contains the configuration about logging.
	Log LogConfig `mapstructure:"log" validate:"required" json:"log"`
	// Graphql contains the configuration about GraphQL.
	Graphql GraphqlConfig `mapstructure:"graphql" validate:"required" json:"graphql"`
	// Database contains the configuration about database connections.
	// This is optional, if no database configuration is found, then no database connection is created.
	Database DatabaseConfig `mapstructure:"database" validate:"" json:"database"`
	// GoogleCloud contains the configuration when running on google cloud.
	GoogleCloud GoogleCloudConfig `mapstructure:"google_cloud" validate:"" json:"googleCloud"`
}

// ServerConfig contains the configuration about the server.
type ServerConfig struct {
	// Port determines the port the server runs on.
	Port int `mapstructure:"port" validate:"required,min=1" json:"port"`
	// Cors contains the configuration about CORS.
	Cors CorsConfig `mapstructure:"cors" validate:"required" json:"cors"`
	// Jwt contains the configuration about JSON web tokens.
	Jwt JwtConfig `mapstructure:"jwt" validate:"required" json:"-"`
}

// CorsConfig contains the configuration about CORS.
type CorsConfig struct {
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge int `mapstructure:"max_age" validate:"min=0" json:"maxAge"`
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool `mapstructure:"allow_credentials" validate:"required" json:"allowCredentials"`
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// An origin may contain a wildcard (*) to replace 0 or more characters
	// (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penalty.
	// Only one wildcard can be used per origin.
	// Default value is ["*"]
	AllowedOrigins []string `mapstructure:"allowed_origins" validate:"required,min=1" json:"allowedOrigins"`
	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (HEAD, GET and POST).
	AllowedMethods []string `mapstructure:"allowed_methods" validate:"required,min=1" json:"allowedMethods"`
	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [] but "Origin" is always appended to the list.
	AllowedHeaders []string `mapstructure:"allowed_headers" validate:"required,min=1" json:"allowedHeaders"`
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
	Audience []string `mapstructure:"audience" validate:"required,min=1" json:"audience"`
	// The "iss" (issuer) claim identifies the principal that issued the
	// JWT.  The processing of this claim is generally application specific.
	// The "iss" value is a case-sensitive string containing a StringOrURI
	// value.  Use of this claim is OPTIONAL.
	Issuer string `mapstructure:"issuer" validate:"required" json:"issuer"`
	// The "exp" (expiration time) claim identifies the expiration time on
	// or after which the JWT MUST NOT be accepted for processing.  The
	// processing of the "exp" claim requires that the current date/time
	// MUST be before the expiration date/time listed in the "exp" claim.
	ExpiresAt time.Duration `mapstructure:"expires_at" validate:"required" json:"expiresAt"`
	// The "nbf" (not before) claim identifies the time before which the JWT
	// MUST NOT be accepted for processing.  The processing of the "nbf"
	// claim requires that the current date/time MUST be after or equal to
	// the not-before date/time listed in the "nbf" claim.  Implementers MAY
	// provide for some small leeway, usually no more than a few minutes, to
	// account for clock skew.  Its value MUST be a number containing a
	// NumericDate value.  Use of this claim is OPTIONAL.
	NotBefore time.Duration `mapstructure:"not_before" validate:"min=0" json:"not_before"`
	// Secret is the key that is used to sign the JWT.
	Secret string `mapstructure:"secret" validate:"required,min=20" json:"-"`
}

// Log contains the configuration about logging.
type LogConfig struct {
	// Level indicates the level the application should log at. Any levels greater than or equal to
	// this will be logged.
	//
	// Log levels from least severe to highest: debug, info, warn, error
	Level string `mapstructure:"level" validate:"required,oneof=debug info warn error" json:"level"`
}

// GraphqlConfig contains the configuration about GraphQL.
type GraphqlConfig struct {
	// Schema indicates where the schema.graphql file is located.
	Schema string `mapstructure:"schema" validate:"required,file" json:"schema"`
}

// DatabaseConfig contains the configuration about database connections.
type DatabaseConfig struct {
	// Driver indicates the database driver to be used.
	Driver string `mapstructure:"driver" validate:"required" json:"driver"`
	// Host indicates the host of the database server.
	Host string `mapstructure:"host" validate:"required" json:"host"`
	// Port indicates the port of the database server.
	Port int `mapstructure:"port" validate:"required" json:"port"`
	// Dbname indicates the name of the database.
	Dbname string `mapstructure:"dbname" validate:"required" json:"dbname"`
	// User indicates the username of the user you want to connect the database with.
	User string `mapstructure:"user" validate:"required" json:"user"`
	// Password indicates the password of the user you want to connect the database with.
	Password string `mapstructure:"password" validate:"required" json:"-"`
}

// GoogleCloudConfig contains the configuration when running on google cloud.
type GoogleCloudConfig struct {
	ProjectId string `mapstructure:"project_id" validate:"required" json:"projectId"`
}

func (dCfg *DatabaseConfig) DataSourceName() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable", dCfg.User, dCfg.Password, dCfg.Dbname, dCfg.Host, dCfg.Port)
}

func (sCfg *ServerConfig) corsOptions() cors.Options {
	return cors.Options{
		MaxAge:             sCfg.Cors.MaxAge,
		AllowCredentials:   sCfg.Cors.AllowCredentials,
		AllowedOrigins:     sCfg.Cors.AllowedOrigins,
		AllowedMethods:     sCfg.Cors.AllowedMethods,
		AllowedHeaders:     sCfg.Cors.AllowedHeaders,
		ExposedHeaders:     nil,
		OptionsPassthrough: false,
		Debug:              false,
	}
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
		return nil
	}))
	_ = enc.AddObject("log", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("level", cfg.Log.Level)
		return nil
	}))
	_ = enc.AddObject("graphql", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("schema", cfg.Graphql.Schema)
		return nil
	}))
	_ = enc.AddObject("database", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("driver", cfg.Database.Driver)
		enc.AddString("dbname", cfg.Database.Dbname)
		enc.AddString("host", cfg.Database.Host)
		enc.AddInt("port", cfg.Database.Port)
		enc.AddString("user", cfg.Database.User)
		return nil
	}))
	return nil
}

// Configuration indicates a struct contains a core.Config
type Configuration interface {
	zapcore.ObjectMarshaler
	CoreConfig() *Config
}

// CoreConfig is used to implement the core.Configuration interface.
func (cfg *Config) CoreConfig() *Config {
	return cfg
}

// LoadConfig takes a reference to a struct that contains a core.Config.
// It uses the config flag to locate the configuration file and maps it to the referenced struct using viper.
//
// If any of the following occurs, it terminates the application with a fatal error:
// - config flag not provided
// - cannot locate the configuration file
// - cannot map the configuration file to the referenced struct
// - the referenced struct is not valid
func LoadConfig(cfg Configuration) {
	var path string
	flag.StringVar(&path, "c", "", "relative path to the config file")
	flag.Parse()

	if path == "" {
		log.Fatal("you must pass the relative path to the config file using the -c argument")
	}

	if strings.HasPrefix(path, "gcp:") {
		ctx := context.Background()
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			log.Fatal("failed to setup google cloud secret manager client", err)
		}

		result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
			Name: path[4:],
		})
		if err != nil {
			log.Fatal("failed to access secret", err)
		}

		viper.SetConfigType("toml")
		err = viper.ReadConfig(bytes.NewBuffer(result.Payload.Data))
		if err != nil {
			log.Fatal("failed to read config from secret", err)
		}
	} else {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			log.Fatal("failed to get absolute path to config file", err)
		}

		viper.SetConfigFile(path)
		err = viper.ReadInConfig()
		if err != nil {
			log.Fatal("failed to read in the config", err)
		}
	}

	err := viper.Unmarshal(cfg)
	if err != nil {
		log.Fatal("failed to unmarshal the config to your struct", err)
	}

	// make absolute paths
	coreCfg := cfg.CoreConfig()
	coreCfg.Path = path
	coreCfg.Graphql.Schema, err = filepath.Abs(coreCfg.Graphql.Schema)
	if err != nil {
		log.Fatal("failed to get absolute path to graphql schema", err)
	}

	// set port
	envPort := os.Getenv("PORT")
	if envPort != "" {
		coreCfg.Server.Port, err = strconv.Atoi(envPort)
		if err != nil {
			log.Fatal("failed to parse PORT from environment variable", err)
		}
	}

	err = validator.New().Struct(cfg)
	if err != nil {
		log.Fatal("config failed validation", err)
	}
}
