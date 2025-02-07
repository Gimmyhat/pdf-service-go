package handlers

import (
	"fmt"
	"net/http"
	"pdf-service-go/internal/domain/pdf"

	"github.com/gin-gonic/gin"
)

type PDFHandler struct {
	service pdf.Service
}

func NewPDFHandler(service pdf.Service) *PDFHandler {
	return &PDFHandler{
		service: service,
	}
}

// ... existing code ...

func (h *PDFHandler) GenerateDocx(c *gin.Context) {
	var req pdf.DocxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pdfBytes, err := h.service.GenerateDocx(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("%s.pdf", req.ID)

	// Устанавливаем правильные заголовки для скачивания PDF файла
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Length", fmt.Sprint(len(pdfBytes)))
	c.Header("Cache-Control", "private")
	c.Header("Pragma", "public")

	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
