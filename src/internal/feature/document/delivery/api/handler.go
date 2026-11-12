// Package api expone los documentos por HTTP.
package api

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/oidc"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/document/application/usecase"
	"kaoru/internal/feature/document/domain"
	shared "kaoru/internal/shared/domain"
)

// Handler atiende las rutas de document.
type Handler struct {
	uc *usecase.UseCases
}

// NewHandler crea el handler.
func NewHandler(uc *usecase.UseCases) *Handler { return &Handler{uc: uc} }

// Register monta las rutas.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/api/documents", h.list)
	r.POST("/api/documents", h.create)
	r.GET("/api/documents/:code", h.get)
	r.POST("/api/documents/:code/versions", h.addVersion)
	r.GET("/api/documents/:code/versions/:version/content", h.content)
	r.GET("/api/documents/:code/versions/:version/preview", h.preview)
	r.POST("/api/documents/:code/links", h.link)
}

type userRefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type versionDTO struct {
	Version    int        `json:"version"`
	FileName   string     `json:"fileName"`
	MediaType  string     `json:"mediaType"`
	SizeBytes  int64      `json:"sizeBytes"`
	Checksum   string     `json:"checksum"`
	UploadedBy userRefDTO `json:"uploadedBy"`
	CreatedAt  time.Time  `json:"createdAt"`
	// Previewable dice si hay vista previa en PDF de un archivo que el navegador no muestra.
	Previewable bool `json:"previewable"`
}

type entityRefDTO struct {
	Type string `json:"type"`
	Code string `json:"code"`
}

type documentDTO struct {
	Code      string         `json:"code"`
	Title     string         `json:"title"`
	Kind      string         `json:"kind"`
	Versions  []versionDTO   `json:"versions"`
	LinkedTo  []entityRefDTO `json:"linkedTo"`
	CreatedAt time.Time      `json:"createdAt"`
}

type linkDTO struct {
	EntityType string `json:"entityType"`
	EntityCode string `json:"entityCode"`
}

func actorOf(c *gin.Context) shared.Actor {
	p := oidc.PrincipalFrom(c)
	return shared.Actor{Id: shared.UserId(p.Sub), Roles: p.Roles}
}

func (h *Handler) toDTO(s domain.Snapshot) documentDTO {
	dto := documentDTO{
		Code: s.Code.String(), Title: s.Title, Kind: string(s.Kind), CreatedAt: s.CreatedAt,
		Versions: make([]versionDTO, len(s.Versions)), LinkedTo: make([]entityRefDTO, len(s.LinkedTo)),
	}
	for i, v := range s.Versions {
		dto.Versions[i] = versionDTO{
			Version: v.Number, FileName: v.FileName, MediaType: v.MediaType, SizeBytes: v.SizeBytes, Checksum: v.Checksum,
			UploadedBy: userRefDTO{ID: string(v.UploadedBy.Id), DisplayName: v.UploadedBy.Name}, CreatedAt: v.CreatedAt,
			Previewable: h.uc.OpenPreview.CanPreview(v.MediaType),
		}
	}
	for i, l := range s.LinkedTo {
		dto.LinkedTo[i] = entityRefDTO{Type: l.Type, Code: l.Code}
	}
	return dto
}

// fileOf abre el archivo del formulario. Un cuerpo que pasa el límite es RESOURCE_EXHAUSTED.
func fileOf(c *gin.Context) (usecase.FileInput, func(), error) {
	header, err := c.FormFile("file")
	if err != nil {
		if tooLarge(err) {
			return usecase.FileInput{}, nil, axis.Wrap(axis.ResourceExhausted, "El archivo pasa el tamaño máximo permitido.", err)
		}
		return usecase.FileInput{}, nil, axis.Invalid(axis.Required("file", "Adjunta un archivo."))
	}
	file, err := header.Open()
	if err != nil {
		return usecase.FileInput{}, nil, axis.Wrap(axis.InvalidFormat, "No se pudo leer el archivo.", err)
	}
	return usecase.FileInput{Name: header.Filename, Content: file}, func() { _ = file.Close() }, nil
}

func tooLarge(err error) bool {
	var limit *http.MaxBytesError
	return errors.As(err, &limit) || strings.Contains(err.Error(), "request body too large")
}

func (h *Handler) list(c *gin.Context) {
	out, err := h.uc.ListDocuments.Execute(c.Request.Context(), usecase.ListDocumentsInput{
		EntityType: c.Query("entityType"), EntityCode: c.Query("entityCode"),
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	items := make([]documentDTO, len(out))
	for i, s := range out {
		items[i] = h.toDTO(s)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) create(c *gin.Context) {
	file, closeFile, err := fileOf(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	defer closeFile()
	out, err := h.uc.CreateDocument.Execute(c.Request.Context(), usecase.CreateDocumentInput{
		Actor: actorOf(c), Title: c.PostForm("title"), Kind: c.PostForm("kind"),
		EntityType: c.PostForm("entityType"), EntityCode: c.PostForm("entityCode"), File: file,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toDTO(out))
}

func (h *Handler) addVersion(c *gin.Context) {
	file, closeFile, err := fileOf(c)
	if err != nil {
		problem.Write(c, err)
		return
	}
	defer closeFile()
	out, err := h.uc.AddVersion.Execute(c.Request.Context(), usecase.AddVersionInput{Actor: actorOf(c), Code: c.Param("code"), File: file})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toDTO(out))
}

func (h *Handler) get(c *gin.Context) {
	out, err := h.uc.GetDocument.Execute(c.Request.Context(), usecase.GetDocumentInput{Code: c.Param("code")})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toDTO(out))
}

func (h *Handler) link(c *gin.Context) {
	var body linkDTO
	if err := httpserver.DecodeJSON(c, &body); err != nil {
		problem.Write(c, err)
		return
	}
	out, err := h.uc.LinkDocument.Execute(c.Request.Context(), usecase.LinkDocumentInput{
		Actor: actorOf(c), Code: c.Param("code"), EntityType: body.EntityType, EntityCode: body.EntityCode,
	})
	if err != nil {
		problem.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toDTO(out))
}

// inlineTypes son los tipos que el visor muestra dentro del sistema; el resto se descarga.
var inlineTypes = map[string]string{
	"application/pdf": "application/pdf",
	"image/png":       "image/png",
	"image/jpeg":      "image/jpeg",
	// El texto se sirve siempre como texto plano: el front lo pinta, el navegador no lo interpreta.
	"text/plain":    "text/plain; charset=utf-8",
	"text/markdown": "text/plain; charset=utf-8",
	"text/csv":      "text/plain; charset=utf-8",
}

func (h *Handler) content(c *gin.Context) {
	number, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		problem.Write(c, usecase.ErrDocumentNotFound)
		return
	}
	out, err := h.uc.OpenContent.Execute(c.Request.Context(), usecase.OpenContentInput{Code: c.Param("code"), Version: number})
	if err != nil {
		problem.Write(c, err)
		return
	}
	defer func() { _ = out.Content.Close() }()

	contentType, inline := inlineTypes[out.Version.MediaType]
	disposition := "inline"
	if !inline || c.Query("download") == "true" {
		contentType, disposition = out.Version.MediaType, "attachment"
	}
	c.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": out.Version.FileName}))
	// El contenido es de un usuario: el navegador no debe adivinar su tipo ni ejecutar nada de él.
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "sandbox; default-src 'none'")
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.DataFromReader(http.StatusOK, out.Version.SizeBytes, contentType, out.Content, nil)
}

// preview sirve la vista previa en PDF de una versión de Office.
func (h *Handler) preview(c *gin.Context) {
	number, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		problem.Write(c, usecase.ErrDocumentNotFound)
		return
	}
	out, err := h.uc.OpenPreview.Execute(c.Request.Context(), usecase.OpenContentInput{Code: c.Param("code"), Version: number})
	if err != nil {
		problem.Write(c, err)
		return
	}
	defer func() { _ = out.Content.Close() }()

	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": out.Version.FileName + ".pdf"}))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "sandbox; default-src 'none'")
	// La versión no cambia nunca: el navegador puede guardar la conversión.
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.DataFromReader(http.StatusOK, -1, "application/pdf", out.Content, nil)
}
