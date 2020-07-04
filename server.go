package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/graph-gophers/graphql-go"
	graphqlErrors "github.com/graph-gophers/graphql-go/errors"
	nanoid "github.com/matoous/go-nanoid"
)

var (
	projectId = os.Getenv("GOOGLE_CLOUD_PROJECT")
)

// response
type response struct {
	w      http.ResponseWriter
	core   *Core
	result *graphql.Response
	status int
}

// newResponse
func newResponse(core *Core, w http.ResponseWriter) response {
	return response{
		w:      w,
		core:   core,
		result: nil,
		status: 200,
	}
}

// write
func (r *response) write() {
	if r.result == nil {
		r.core.Logger.DPanic("response.Result is nil", "response", r)
		r.writeError(KindUnknown)
	}

	r.result.Extensions = r.core.Extensions()
	r.w.Header().Add("Content-Type", "application/json")
	r.w.WriteHeader(r.status)

	err := json.NewEncoder(r.w).Encode(r.result)
	if err != nil {
		r.core.Logger.DPanic("failed to encode result", "error", err)
	}
}

// writeError
func (r *response) writeError(err error, args ...interface{}) {
	e := NewError(r.core, err, args...)
	r.status = e.HttpStatus()

	r.result = &graphql.Response{
		Errors: []*graphqlErrors.QueryError{
			{
				Message:       e.Error(),
				Extensions:    e.Extensions(),
				ResolverError: err,
			},
		},
	}

	r.write()
}

// server
type server struct {
	config   Configuration
	router   chi.Router
	logger   Logger
	schema   *graphql.Schema
	resolver interface{}
	decorate ResolverContextDecorator
	db       *sql.DB
}

// newCore
func (s *server) newCore(r *http.Request, operation string) (*Core, error) {
	// this should never error
	id, _ := nanoid.Nanoid()

	core := &Core{
		Config:     s.config,
		Db:         s.db,
		Id:         id,
		Operations: []string{operation},
		Validate:   validate,

		// set later
		Context: nil,
		Request: nil,
		Logger:  nil,
		Session: nil,
	}

	// attach logger to core
	core.Logger = s.logger.WithCore(core)

	// look for google cloud tracing header
	// https://cloud.google.com/run/docs/logging?hl=en#writing_structured_logs
	traceParts := strings.Split(r.Header.Get("X-Cloud-Trace-Context"), "/")
	if len(traceParts) > 0 && len(traceParts[0]) > 0 {
		// attach google cloud tracing id to logger
		core.Logger = core.Logger.With(
			"logging.googleapis.com/trace",
			fmt.Sprintf("projects/%s/traces/%s", projectId, traceParts[0]))
	}

	// attach core to context, decorate context with decorator, and attach context to core
	core.Context = s.decorate(context.WithValue(r.Context(), ContextKey, core))

	// attach request to core with new decorated context
	core.Request = r.WithContext(core.Context)

	return core, core.StartSession()
}

// routes
func (s *server) setupRoutes() {
	s.router.Use(middleware.RealIP)
	s.router.Use(cors.New(s.config.CoreConfig().Server.corsOptions()).Handler)
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
					s.logger.Error("recovering from panic", "error", rvr)

					// error would be session related, which doesn't matter here
					core, _ := s.newCore(r, "server.Recover")
					response := newResponse(core, w)
					response.writeError(KindUnknown)
				}
			}()

			next.ServeHTTP(w, r)
		})
	})

	s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		core, err := s.newCore(r, "server.handleGraphql")
		response := newResponse(core, w)
		if err != nil {
			response.writeError(err)
			return
		}

		var request struct {
			Query         string                 `json:"query"`
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}

		if core.Request.Method == http.MethodGet {
			request.Query = core.Request.URL.Query().Get("query")
			request.OperationName = core.Request.URL.Query().Get("operationName")
			vars := core.Request.URL.Query().Get("variables")
			if vars != "" {
				err = json.NewDecoder(strings.NewReader(vars)).Decode(&request.Variables)
				if err != nil {
					response.writeError(err, KindInvalidJson, "Your variables query parameter contains invalid JSON")
					return
				}
			}
		} else {
			err = json.NewDecoder(core.Request.Body).Decode(&request)
			if err != nil {
				response.writeError(err, KindInvalidJson)
				return
			}
		}

		response.result = s.schema.Exec(core.Context, request.Query, request.OperationName, request.Variables)
		if response.result.Errors != nil {
			if len(response.result.Errors) == 0 {
				core.Logger.DPanic("response contains an empty list of errors", "response", response)
				response.status = http.StatusInternalServerError
			} else {
				// convert any errors to Error
				for _, err := range response.result.Errors {
					if err.ResolverError != nil {
						e := NewError(core, err.ResolverError)
						err.Extensions = e.Extensions()
						err.Message = e.Error()
						err.ResolverError = e
						response.status = e.HttpStatus()
					} else {
						// an error occurred before the resolver was called
						// most likely a query validation error
						response.status = http.StatusBadRequest
					}
				}
			}
		}

		response.write()
	})

	s.router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		core, err := s.newCore(r, "server.MethodNotAllowed")
		response := newResponse(core, w)
		if err != nil {
			response.writeError(err)
			return
		}
		response.writeError(KindMethodNotAllowed)
	})

	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		core, err := s.newCore(r, "server.NotFound")
		response := newResponse(core, w)
		if err != nil {
			response.writeError(err)
			return
		}
		response.writeError(KindRouteNotFound)
	})
}

// ServeHTTP
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Run
func Run(opts Options) {
	logger := newLogger(opts.Config.CoreConfig())

	detail = opts.ErrorDetailer
	if detail == nil {
		logger.Fatal("ErrorDetailer must not be nil", "config", opts.Config)
	}

	s := &server{
		logger:   logger,
		config:   opts.Config,
		resolver: opts.Resolver,
		decorate: opts.ResolverContextDecorator,
		router:   chi.NewRouter(),

		// set later
		db:     nil,
		schema: nil,
	}

	if s.resolver == nil {
		logger.Fatal("Resolver must not be nil", "config", opts.Config)
	}
	if s.decorate == nil {
		logger.Fatal("ResolverContextDecorator must not be nil", "config", opts.Config)
	}

	if s.config.CoreConfig().Database.Main.Driver != "" {
		db, err := sql.Open(s.config.CoreConfig().Database.Main.Driver, s.config.CoreConfig().Database.Main.DataSourceName())
		if err != nil {
			logger.Fatal("failed to open database connection", "error", err, "config", s.config)
		}
		s.db = db

		if s.config.CoreConfig().Database.Migrations.RunOnStart {
			logger.Info("running database migrations (database.migrations.run_on_start = true)", "config", s.config)
			m, err := migrate.New(s.config.CoreConfig().Database.Migrations.Location, s.config.CoreConfig().Database.Main.ConnectionString())
			if err != nil {
				s.logger.Fatal("failed to create an instance of Migrate", "error", err, "config", s.config)
			}
			m.Log = logger

			err = m.Up()
			if err != nil && err.Error() != "no change" {
				s.logger.Fatal("failed to migrate main database", "error", err, "config", s.config)
			}
		}
	}

	bytes, err := ioutil.ReadFile(s.config.CoreConfig().Server.Graphql.Schema)
	if err != nil {
		s.logger.Fatal("failed to read graphql schema", "error", err, "config", s.config)
	}

	s.schema, err = graphql.ParseSchema(string(bytes), s.resolver)
	if err != nil {
		s.logger.Fatal("failed to parse graphql schema", "error", err, "config", s.config)
	}

	s.setupRoutes()

	// cleanup server resources
	defer func() {
		if s.db != nil {
			if err := s.db.Close(); err != nil {
				s.logger.Error("failed to close database", "error", err)
			}
		}
		if s.logger != nil {
			if err := s.logger.Close(); err != nil {
				log.Printf("failed to close logger: %v", err)
			}
		}
	}()

	s.logger.Info(fmt.Sprintf("listening on port %d", s.config.CoreConfig().Server.Port), "config", s.config)
	if err = http.ListenAndServe(fmt.Sprintf(":%d", s.config.CoreConfig().Server.Port), s); err != nil {
		logger.Fatal("failed to run server", "error", err)
	}
}

// Options
type Options struct {
	Config                   Configuration
	ErrorDetailer            ErrorDetailer
	ResolverContextDecorator ResolverContextDecorator
	Resolver                 interface{}
}

// ResolverContextDecorator
type ResolverContextDecorator func(ctx context.Context) context.Context
