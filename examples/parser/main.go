// Change the parsers used by the router.
package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/extractor"
	"github.com/zhamlin/routey/param"
)

type Object struct {
	Name string
}

type GetRequest struct {
	Object routey.Query[Object]
}

type GetResponse struct {
	Message string
}

func Get(p GetRequest) (GetResponse, error) {
	return GetResponse{
		Message: p.Object.Value.Name,
	}, nil
}

func parseObject(value any, params []string) error {
	err := param.ErrInvalidParamType
	if obj, ok := value.(*Object); ok {
		obj.Name, err = params[0], nil
	}
	return err
}

func newRouter() *routey.Router {
	r := routey.New()

	r.Response = func(w http.ResponseWriter, _ *http.Request, resp extractor.Response) {
		b, _ := json.Marshal(resp.Response)
		w.Write(b)
	}

	r.Params.Parser = param.Parsers{
		parseObject,
		r.Params.Parser,
	}.Parse

	return r
}

func main() {
	r := newRouter()
	routey.Get(r, "/", Get)

	server := http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: r,
	}

	slog.Info("listening for requests", "addr", server.Addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}

	// curl '127.0.0.1:8080/?object=test'
}
