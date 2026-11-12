// Package converter implementa el puerto Converter con Gotenberg, un servicio HTTP que
// envuelve LibreOffice. Corre como un contenedor aparte y opcional (el servicio `libreoffice`).
package converter

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/document/application/port"
)

// timeout acota una conversión: un documento grande tarda segundos, no minutos.
const timeout = 60 * time.Second

// Gotenberg convierte con el servicio que atiende en baseURL.
type Gotenberg struct {
	baseURL string
	client  *http.Client
}

var _ port.Converter = (*Gotenberg)(nil)

// New crea el adaptador. baseURL es la dirección interna del servicio, sin barra final.
func New(baseURL string) *Gotenberg {
	return &Gotenberg{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{}}
}

// ToPDF envía el archivo y devuelve el PDF. El nombre que se envía es fijo salvo la extensión:
// el conversor solo la usa para saber el formato.
func (g *Gotenberg) ToPDF(ctx context.Context, fileName string, content io.Reader) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	body, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	go func() {
		part, err := form.CreateFormFile("files", "documento"+strings.ToLower(filepath.Ext(fileName)))
		if err == nil {
			_, err = io.Copy(part, content)
		}
		if err == nil {
			err = form.Close()
		}
		_ = writer.CloseWithError(err)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/forms/libreoffice/convert", body)
	if err != nil {
		cancel()
		return nil, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := g.client.Do(req)
	if err != nil {
		cancel()
		_ = body.CloseWithError(err)
		return nil, axis.Wrap(axis.ServiceUnavailable, "La vista previa no está disponible ahora. Descarga el archivo para abrirlo.", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		cancel()
		return nil, axis.Wrap(axis.ServiceUnavailable, "No se pudo generar la vista previa. Descarga el archivo para abrirlo.",
			fmt.Errorf("el conversor respondió %d", resp.StatusCode))
	}
	return &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}, nil
}

// cancelOnClose libera el contexto de la conversión cuando se termina de leer el PDF.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	defer c.cancel()
	return c.ReadCloser.Close()
}
