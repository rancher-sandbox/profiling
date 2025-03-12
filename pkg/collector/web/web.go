package web

import (
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage/hack"
	"github.com/gin-gonic/gin"
)

type WebServer struct {
	port   int
	logger *slog.Logger
	store  storage.Store

	reloadF func() error
}

func NewWebServer(
	logger *slog.Logger,
	port int,
	store storage.Store,
	reloadF func() error,
) *WebServer {
	return &WebServer{
		port:    port,
		logger:  logger.With("component", "web-server"),
		store:   store,
		reloadF: reloadF,
	}
}

const pprofPrefix = "/pprof/web/"

func (w *WebServer) Start() error {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	router.POST("/reload", func(c *gin.Context) {
		if err := w.reloadF(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "reloaded"})
	})
	router.GET(path.Join(pprofPrefix, ":profileType", "*key"), func(c *gin.Context) {
		logger := w.logger.With("path", c.Request.URL.Path)
		logger.Debug("received request")
		profileType := c.Param("profileType")
		if profileType == "" {
			// FIXME validate that it is a valid profile type
			c.JSON(400, gin.H{"error": "invalid profile type"})
			return
		}
		paramKey := c.Param("key")
		paramKey = strings.TrimPrefix(strings.TrimSpace(paramKey), "/")

		parts := strings.Split(paramKey, "/")
		if len(parts) < 3 {
			c.JSON(400, gin.H{"error": "invalid key " + paramKey})
			return
		}
		if len(parts) > 4 {
			c.JSON(400, gin.H{"error": "invalid key " + paramKey})
			return
		}

		logger.With("key", paramKey, "profileType", profileType).Debug("getting profile")

		actualKey := path.Join(parts[:3]...)

		filepaths, err := w.store.Get(profileType, actualKey)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		pprofServer := &PprofWebWrapper{
			filepath:    filepaths[len(filepaths)-1],
			profileType: profileType,
		}
		mux, err := pprofServer.Driver()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		newPath := path.Join(pprofPrefix, profileType) + "/"
		if len(parts) == 4 {
			newPath = path.Join(pprofPrefix, profileType, parts[3])
		}
		c.Request.URL.Path = newPath
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
		toW := ""

		if len(keys) == 0 || keys[0] == "" {
			toW = "<h2> No profiles collected </h2>"
		} else {
			for _, key := range keys {
				toW += fmt.Sprintf("<a href=\"%s/\"> %s </a> <br/> ", path.Join(pprofPrefix, key), key)
			}
		}

		// Write raw HTML directly to the response
		c.Writer.Write([]byte(fmt.Sprintf(htmlContent, toW)))

	})

	addr := fmt.Sprintf(":%d", w.port)
	w.logger.With("addr", addr).Info("starting web server")
	return router.Run(fmt.Sprintf(":%d", w.port))
}

// FIXME: copilot generated slop
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
					sb.WriteString(fmt.Sprintf("<li><a href=\"%s%s\">%s</a></li>\n", pprofPrefix, path.Join(namespace, name, target, profile), profile))
				}
				sb.WriteString("</ul>\n")
			}
		}
	}

	return sb.String()
}
