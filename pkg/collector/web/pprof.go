package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage/hack"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/web/internal/pprof"
	"github.com/gin-gonic/gin"
	"github.com/google/pprof/driver"
)

type WebServer struct {
	port   int
	logger *slog.Logger
	store  storage.Store
}

func NewWebServer(logger *slog.Logger, port int, store storage.Store) *WebServer {
	return &WebServer{
		port:   port,
		logger: logger.With("component", "web-server"),
		store:  store,
	}
}

// startPProfServer start the pprof web handler without browser
func startPProfServer(filePath string) (mux *http.ServeMux, err error) {
	mux = http.NewServeMux()

	options := &driver.Options{
		Flagset: &pprof.Flags{
			Args: []string{"-http=localhost:0", "-no_browser", filePath},
		},
		HTTPServer: pprofHttpServer(mux),
	}

	if err = driver.PProf(options); err != nil {
		return
	}

	return
}

// pprofHttpServer wrap http server for pprof profile manager
func pprofHttpServer(mux *http.ServeMux) func(*driver.HTTPServerArgs) error {
	return func(args *driver.HTTPServerArgs) error {
		for pattern, handler := range args.Handlers {
			//if pattern == "/" {
			//	mux.Handle(pprofWebPath, handler)
			//} else {
			//	mux.Handle(path.Join(pprofWebPath, pattern), handler)
			//}
			var joinedPattern string
			if pattern == "/" {
				joinedPattern = profilePrefix
			} else {
				joinedPattern = path.Join(profilePrefix, pattern)
			}
			mux.Handle(joinedPattern, handler)
		}
		return nil
	}
}

const profilePrefix = "/profile/"

func (w *WebServer) Start() error {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.GET(profilePrefix+"*key", func(c *gin.Context) {
		key := c.Param("key")
		filepaths, err := w.store.Get(key)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		mux, err := startPProfServer(filepaths[len(filepaths)-1])
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		mux.ServeHTTP(c.Writer, c.Request)
	})

	// FIXME: this entire function is a mess, done for speed
	router.GET("/", func(c *gin.Context) {
		keys, err := w.store.ListKeys()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		htmlContent := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Raw HTML Example</title>
		</head>
		<body>
			%s
		</body>
		</html>
		`
		md := []hack.Metadata{}
		for _, key := range keys {
			md = append(md, hack.SplitPathToMd(key))
		}
		slices.SortFunc(md, func(a, b hack.Metadata) int {
			if a.Namespace < b.Namespace {
				return -1
			}
			if a.Namespace > b.Namespace {
				return 1
			}
			if a.Name < b.Name {
				return -1
			}
			if a.Name > b.Name {
				return 1
			}
			if a.Target < b.Target {
				return -1
			}
			if a.Target > b.Target {
				return 1
			}
			if a.ProfileType < b.ProfileType {
				return -1
			}
			if a.ProfileType > b.ProfileType {
				return 1
			}
			return 0
		})
		toW := generateHTMLList(md)

		if len(keys) == 0 || keys[0] == "" {
			toW = "<h2> No profiles collected </h2>"
		}

		// Write raw HTML directly to the response
		c.Writer.Write([]byte(fmt.Sprintf(htmlContent, toW)))

	})

	addr := fmt.Sprintf(":%d", w.port)
	w.logger.With("addr", addr).Info("starting web server")
	return router.Run(fmt.Sprintf(":%d", w.port))
}

func generateHTMLList(metadata []hack.Metadata) string {
	var sb strings.Builder

	// Create a map to group Metadata by Namespace -> Name -> Unique Target -> Metadata entries
	namespaces := make(map[string]map[string]map[string][]hack.Metadata)

	// Group metadata by Namespace -> Name -> Target -> Metadata entries
	for _, m := range metadata {
		if _, ok := namespaces[m.Namespace]; !ok {
			namespaces[m.Namespace] = make(map[string]map[string][]hack.Metadata)
		}
		if _, ok := namespaces[m.Namespace][m.Name]; !ok {
			namespaces[m.Namespace][m.Name] = make(map[string][]hack.Metadata)
		}
		namespaces[m.Namespace][m.Name][m.Target] = append(namespaces[m.Namespace][m.Name][m.Target], m)
	}

	// Now loop through each namespace and its names
	for namespace, names := range namespaces {
		// Add the namespace as a header
		sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", namespace))

		// Loop through each name in the namespace
		for name, targets := range names {
			// Add the name as a header
			sb.WriteString(fmt.Sprintf("<h2>%s</h2>\n", name))

			// Loop through all the unique targets for this name
			for target, entries := range targets {
				// Add the target as a header
				sb.WriteString(fmt.Sprintf("<h3>%s</h3>\n", target))

				// Add the ProfileType as a list of <a> tags
				sb.WriteString("<ul>\n")
				profileLinks := make(map[string]struct{}) // Use a map to deduplicate profile types
				for _, entry := range entries {
					profileList := strings.Split(entry.ProfileType, ",") // Assuming ProfileType is a comma-separated list
					for _, profile := range profileList {
						profileLinks[profile] = struct{}{} // Add profile to the map to ensure uniqueness
					}
				}

				// Iterate over the deduplicated profile types and add them as <a> tags
				for profile := range profileLinks {
					sb.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", profile, profile))
				}
				sb.WriteString("</ul>\n")
			}
		}
	}

	return sb.String()
}
