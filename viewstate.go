package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

type ViewState struct {
	RequestCount int
}

const viewStateKey = "__view_state"

func getViewState(c echo.Context) *ViewState {
	if vs, ok := c.Get(viewStateKey).(*ViewState); ok {
		return vs
	}

	vs := &ViewState{}
	c.Set(viewStateKey, vs)

	jsonStr := c.FormValue(viewStateKey)
	if jsonStr == "" {
		return vs
	}

	err := json.Unmarshal([]byte(jsonStr), vs)
	if err != nil {
		slog.Warn("invalid view state JSON", "error", err)
		return vs
	}

	return vs
}

func renderViewState(viewState *ViewState) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		jsonBytes, err := json.Marshal(viewState)
		if err != nil {
			return err
		}

		return renderViewStateAsString(string(jsonBytes)).Render(ctx, w)
	})
}

func viewStateMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		vs := getViewState(c)
		vs.RequestCount++

		err := respondTempl(c, renderCounter(vs.RequestCount))
		if err != nil {
			return err
		}

		handlerErr := next(c)

		err = respondTempl(c, renderViewState(vs))
		if err != nil {
			return err
		}

		return handlerErr
	}
}
