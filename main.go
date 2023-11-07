package main

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/danfarinoeyecue/go-templ/memstore"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:generate templ generate

//go:embed static
var static embed.FS

func main() {
	err := run()
	if err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

type Item struct {
	ID      string `form:"id" validate:"required,alphanum"`
	Message string `form:"message" validate:"required"`
}

func (i Item) GetID() string {
	return i.ID
}

func run() error {
	store := memstore.New[Item]()
	_ = store.Create(Item{
		ID:      "1",
		Message: "foo",
	})
	validate := validator.New(validator.WithRequiredStructEnabled())

	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.RequestID())

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper:        nil,
		BeforeNextFunc: nil,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			slog.InfoContext(c.Request().Context(),
				"echo request complete",
				slog.String("request_id", v.RequestID),
				slog.Int("status", v.Status),
				slog.String("method", v.Method),
				slog.String("url", v.URI),
				slog.Duration("latency", v.Latency),
				slog.Any("error", v.Error),
			)
			return nil
		},
		LogLatency:   true,
		LogMethod:    true,
		LogURI:       true,
		LogRequestID: true,
		LogStatus:    true,
		LogError:     true,
	}))

	e.Pre(middleware.Gzip())

	e.Pre(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: http.FS(static),
	}))

	e.GET("/", func(c echo.Context) error {
		items, err := store.All()
		if err != nil {
			return err
		}
		return respondTempl(c, index("", items))
	}, htmlContentTypeMiddleware)

	apiGroup := e.Group("/api", htmlContentTypeMiddleware, templErrorsMiddleware, viewStateMiddleware)

	apiGroup.POST("/increment", func(c echo.Context) error {
		return nil
	})

	apiGroup.POST("/error", func(c echo.Context) error {
		vs := getViewState(c)
		return fmt.Errorf("oops from request %d", vs.RequestCount)
	})

	apiGroup.POST("/create", func(c echo.Context) error {
		var item Item
		err := c.Bind(&item)
		if err != nil {
			return err
		}

		err = validate.Struct(item)
		if err != nil {
			return err
		}

		err = store.Create(item)
		if err != nil {
			return err
		}

		items, err := store.All()
		if err != nil {
			return err
		}

		return respondTempl(c, renderItems(items), renderCreationForm(item.Message))
	})

	apiGroup.POST("/delete", func(c echo.Context) error {
		id := c.FormValue("id")
		if id == "" {
			return errors.New("missing ID")
		}

		err := store.Delete(id)
		if err != nil {
			return err
		}

		items, err := store.All()
		if err != nil {
			return err
		}

		return respondTempl(c, renderItems(items))
	})

	return e.Start("127.0.0.1:8989")
}

func htmlContentTypeMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/html")
		return next(c)
	}
}

func templErrorsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			return respondTempl(c, renderError(err.Error()))
		}
		return respondTempl(c, renderError(""))
	}
}

func respondTempl(c echo.Context, cos ...templ.Component) error {
	for _, co := range cos {
		err := co.Render(c.Request().Context(), c.Response())
		if err != nil {
			// note: the only reason Render can fail is when writing to the HTTP response writer, which means all
			// bets are off for this request. No need to run further templs, or worry about HTTP status code.
			return err
		}
	}

	return nil
}
