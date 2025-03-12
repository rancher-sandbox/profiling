package web

import (
	"fmt"
	"net/http"
	"path"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector/web/internal/pprof"
	"github.com/google/pprof/driver"
)

type PprofWebWrapper struct {
	filepath    string
	profileType string
}

func (p *PprofWebWrapper) Driver() (*http.ServeMux, error) {

	mux := http.NewServeMux()

	options := &driver.Options{
		Flagset: &pprof.Flags{
			Args: []string{"-http=localhost:0", "-no_browser", p.filepath},
		},
		HTTPServer: p.server(mux),
	}

	if err := driver.PProf(options); err != nil {
		return mux, err
	}

	return mux, nil
}

func debugWrapper(baseHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Embedded pprof Request: ", r.URL.Path)
		baseHandler.ServeHTTP(w, r)
	})
}

func (p *PprofWebWrapper) server(mux *http.ServeMux) func(*driver.HTTPServerArgs) error {
	return func(args *driver.HTTPServerArgs) error {
		for pattern, handler := range args.Handlers {
			var joinedPattern string
			if pattern == "/" {
				joinedPattern = path.Join(pprofPrefix, p.profileType)
			} else {
				joinedPattern = path.Join(pprofPrefix, p.profileType, pattern)
			}
			mux.Handle(joinedPattern, debugWrapper(handler))
		}
		return nil
	}
}
