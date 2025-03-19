package web

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rancher-sandbox/profiling/pkg/collector/storage"
)

type WebServer struct {
	port    int
	logger  *slog.Logger
	store   storage.Store
	reloadF func() error

	fsDataDir string

	// set in Start
	static    http.Handler
	templates *template.Template
}

func NewWebServer(
	logger *slog.Logger,
	port int,
	store storage.Store,
	reloadF func() error,
	fsDataDir string,
) *WebServer {
	return &WebServer{
		port:      port,
		logger:    logger.With("component", "web-server"),
		store:     store,
		reloadF:   reloadF,
		fsDataDir: fsDataDir,
	}
}

const pprofPrefix = "/pprof/web/"

func (w *WebServer) Start() error {
	static, err := static()
	if err != nil {
		return fmt.Errorf("failed to start static file server %w", err)
	}
	templates, err := htmlTemplates()
	if err != nil {
		return fmt.Errorf("failed to parse templates %w", err)
	}
	w.templates = templates
	w.static = static
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/static/*path", func(c *gin.Context) {
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/static")
		w.static.ServeHTTP(c.Writer, c.Request)
	})

	router.GET("/ui/dashboard", func(c *gin.Context) {
		nsList, err := w.store.GroupKeys()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := templates.ExecuteTemplate(c.Writer, "dashboard.html.tmpl", nsList); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
	})

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/ui/dashboard")
	})

	//FIXME: move out to generic http api
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

	// temporary function to expose raw profiles for debugging
	router.GET("/raw/*path", func(c *gin.Context) {
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/raw")
		dir := http.Dir(w.fsDataDir)
		http.FileServer(dir).ServeHTTP(c.Writer, c.Request)
	})

	addr := fmt.Sprintf(":%d", w.port)
	w.logger.With("addr", addr).Info("starting web server")
	return router.Run(fmt.Sprintf(":%d", w.port))
}
