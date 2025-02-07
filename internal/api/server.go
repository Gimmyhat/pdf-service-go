package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Router   *gin.Engine
	Handlers *Handlers
}

func (s *Server) SetupRoutes() {
	s.Router.POST("/api/v1/docx", s.Handlers.PDF.GenerateDocx)
}

func (s *Server) Start(addr string) error {
	return s.Router.Run(addr)
}

func (s *Server) Stop() {
	// Implementation of Stop method
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}
