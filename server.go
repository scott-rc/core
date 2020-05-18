package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	graphqlErrors "github.com/graph-gophers/graphql-go/errors"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/graph-gophers/graphql-go"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/opentracing/opentracing-go/log"
)

var (
	resolver interface{}
	decorate ContextDecorator
)

type response struct {
	w      http.ResponseWriter
	core   *Core
	result *graphql.Response
	status int
}

func newResponse(core *Core, w http.ResponseWriter) response {
	return response{
		w:      w,
		core:   core,
		result: nil,
		status: 200,
	}
}

func (r *response) write() {
	if r.result == nil {
		r.core.Logger.DPanic("response.Result is nil", "response", r)
		r.writeError(NewError(r.core, KindUnknown))
	}
	r.result.Extensions = r.core.Extensions()

	r.w.Header().Add("Content-Type", "application/json")
	r.w.WriteHeader(r.status)

	r.core.Logger.Debug("writing response", "response", r)
	err := json.NewEncoder(r.w).Encode(r.result)
	if err != nil {
		r.core.Logger.DPanic("failed to encode result", "error", err)
	}
}

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

// ContextDecorator
type ContextDecorator func(ctx context.Context) context.Context

// SetContextDecorator
func SetContextDecorator(decorator ContextDecorator) {
	decorate = decorator
}

// SetGraphqlResolver
func SetGraphqlResolver(r interface{}) {
	resolver = r
}

type server struct {
	cfg    Configuration
	router chi.Router
	logger Logger
	db     *sql.DB
}

func (s *server) newCore(r *http.Request, operation string) (*Core, error) {
	// this should never error
	id, _ := gonanoid.Nanoid()

	ctx := &Core{
		Config:     s.cfg,
		Context:    r.Context(),
		Db:         s.db,
		Id:         id,
		Logger:     nil,
		Operations: []string{operation},
		Request:    r,
		Session:    nil,
	}

	// create circular reference to core and logger
	ctx.Logger = s.logger.WithCore(ctx)

	// set google cloud tracing
	traceParts := strings.Split(r.Header.Get("X-Cloud-Trace-Context"), "/")
	if len(traceParts) > 0 && len(traceParts[0]) > 0 {
		ctx.Logger = ctx.Logger.With(
			"logging.googleapis.com/trace",
			fmt.Sprintf("projects/%s/traces/%s", s.cfg.CoreConfig().GoogleCloud.ProjectId, traceParts[0]))
	}

	// create circular reference to core and r.Context()
	c := context.WithValue(ctx.Context, ContextKey, ctx)

	// decorate context with users decorator
	c = decorate(c)

	// re-assign request with new context
	*r = *r.WithContext(c)

	return ctx, ctx.StartSession()
}

func (s *server) cleanup() {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("failed to close database", "error", err)
		}
	}

	if s.logger != nil {
		if err := s.logger.Close(); err != nil {
			log.Error(err)
		}
	}
}

func (s *server) routes() {
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.StripSlashes)
	s.router.Use(cors.New(s.cfg.CoreConfig().Server.corsOptions()).Handler)

	bytes, err := ioutil.ReadFile(s.cfg.CoreConfig().Graphql.Schema)
	if err != nil {
		s.logger.Fatal("failed to read graphql schema", "error", err, "config", s.cfg)
	}

	schema, err := graphql.ParseSchema(string(bytes), resolver)
	if err != nil {
		s.logger.Fatal("failed to parse graphql schema", "error", err, "config", s.cfg)
	}

	s.router.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
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

		err = json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			response.writeError(err, KindInvalidJson)
			return
		}

		response.result = schema.Exec(r.Context(), request.Query, request.OperationName, request.Variables)
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
						response.status = 400
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

func checkGlobals() error {
	if resolver == nil {
		return errors.New("you must set the GraphQL resolver (help: call core.SetGraphqlResolver before calling core.Run)")
	}
	if detail == nil {
		return errors.New("you must set the core.ErrorDetailer (help: call core.SetErrorDetailer before calling core.Run)")
	}
	if determine == nil {
		return errors.New("you must set the core.ErrorDeterminer (help: call core.SetErrorDeterminer before calling core.Run)")
	}
	if decorate == nil {
		return errors.New("you must set the core.ContextDetailer (help: call core.SetContextDecorator before calling core.Run)")
	}
	return nil
}

// Run
func Run(cfg Configuration) error {
	logger, err := newLogger(cfg.CoreConfig())
	if err != nil {
		return err
	}

	err = checkGlobals()
	if err != nil {
		return err
	}

	s := &server{cfg: cfg, router: chi.NewRouter(), logger: logger, db: nil}
	defer s.cleanup()

	if cfg.CoreConfig().Database.Driver != "" {
		db, err := sql.Open(cfg.CoreConfig().Database.Driver, cfg.CoreConfig().Database.DataSourceName())
		if err != nil {
			return err
		}
		s.db = db
	}

	s.routes()
	s.logger.Info(fmt.Sprintf("listening on port %d", s.cfg.CoreConfig().Server.Port), "config", cfg)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.CoreConfig().Server.Port), s)
}
