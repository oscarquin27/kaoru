package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/problem"
)

// spyLogger guarda los mensajes y sus argumentos.
type spyLogger struct {
	lines []string
}

func (s *spyLogger) log(level, msg string, args []any) {
	raw, _ := json.Marshal(args)
	s.lines = append(s.lines, level+" "+msg+" "+string(raw))
}
func (s *spyLogger) InfoContext(_ context.Context, msg string, args ...any) { s.log("INFO", msg, args) }
func (s *spyLogger) WarnContext(_ context.Context, msg string, args ...any) { s.log("WARN", msg, args) }
func (s *spyLogger) ErrorContext(_ context.Context, msg string, args ...any) {
	s.log("ERROR", msg, args)
}

const origin = "https://kaoru.lab.invalid"

func newEngine(log *spyLogger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(httpserver.RequestID(), httpserver.AccessLog(log), httpserver.Recovery(log),
		httpserver.MaxBody(64, 1024), httpserver.Origin(origin))
	engine.GET("/ok", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.POST("/echo", func(c *gin.Context) {
		var body map[string]any
		if err := httpserver.DecodeJSON(c, &body); err != nil {
			problem.Write(c, err)
			return
		}
		c.JSON(http.StatusOK, body)
	})
	engine.GET("/panic", func(*gin.Context) { panic("secreto interno") })
	engine.GET("/boom", func(c *gin.Context) { problem.Write(c, errors.New("pq: clave del usuario kaoru_app")) })
	return engine
}

func do(engine *gin.Engine, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func problemOf(t *testing.T, rec *httptest.ResponseRecorder) problem.Body {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, quiero application/problem+json", ct)
	}
	var body problem.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("cuerpo ilegible: %v: %s", err, rec.Body.String())
	}
	return body
}

func TestRequestID(t *testing.T) {
	engine := newEngine(&spyLogger{})
	if rec := do(engine, "GET", "/ok", "", map[string]string{"X-Request-Id": "desde-nginx"}); rec.Header().Get("X-Request-Id") != "desde-nginx" {
		t.Error("debe conservar el id que manda Nginx")
	}
	if rec := do(engine, "GET", "/ok", "", nil); len(rec.Header().Get("X-Request-Id")) != 32 {
		t.Errorf("debe generar un id si no viene: %q", rec.Header().Get("X-Request-Id"))
	}
}

func TestOrigin(t *testing.T) {
	engine := newEngine(&spyLogger{})
	cases := []struct {
		name       string
		method     string
		origin     string
		wantStatus int
	}{
		{"leer no exige Origin", "GET", "", 204},
		{"escribir con el origen público", "POST", origin, 200},
		{"escribir sin Origin", "POST", "", 403},
		{"escribir desde otro origen", "POST", "https://malo.invalid", 403},
		{"origen parecido", "POST", origin + ".malo.invalid", 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, headers := "/ok", map[string]string{}
			if tc.method == "POST" {
				path = "/echo"
			}
			if tc.origin != "" {
				headers["Origin"] = tc.origin
			}
			rec := do(engine, tc.method, path, `{"a":1}`, headers)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, quiero %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus == 403 && problemOf(t, rec).Code != "PERMISSION_DENIED" {
				t.Error("un Origin que no vale debe ser PERMISSION_DENIED")
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	engine := newEngine(&spyLogger{})
	headers := map[string]string{"Origin": origin}
	if body := problemOf(t, do(engine, "POST", "/echo", `{mal`, headers)); body.Status != 400 || body.Code != "INVALID_FORMAT" {
		t.Errorf("JSON ilegible: %+v", body)
	}
	big := `{"a":"` + strings.Repeat("x", 100) + `"}`
	if body := problemOf(t, do(engine, "POST", "/echo", big, headers)); body.Status != 413 || body.Code != "RESOURCE_EXHAUSTED" {
		t.Errorf("cuerpo que pasa el límite: %+v", body)
	}
}

func TestRecoveryAndLogs(t *testing.T) {
	log := &spyLogger{}
	engine := newEngine(log)

	for _, path := range []string{"/panic", "/boom"} {
		rec := do(engine, "GET", path, "", map[string]string{"X-Request-Id": "traza-1"})
		body := problemOf(t, rec)
		if rec.Code != 500 || body.Code != "INTERNAL_SERVER_ERROR" || body.TraceID != "traza-1" {
			t.Errorf("%s: respuesta inesperada: %d %+v", path, rec.Code, body)
		}
		if strings.Contains(rec.Body.String(), "secreto") || strings.Contains(rec.Body.String(), "kaoru_app") {
			t.Errorf("%s: la respuesta filtra el detalle interno: %s", path, rec.Body.String())
		}
	}
	joined := strings.Join(log.lines, "\n")
	if !strings.Contains(joined, "secreto interno") || !strings.Contains(joined, "kaoru_app") {
		t.Errorf("el detalle técnico de un 5xx debe quedar en el log:\n%s", joined)
	}

	// Un 4xx se registra con su código, sin el detalle.
	log.lines = nil
	do(engine, "POST", "/echo", `{}`, nil)
	if len(log.lines) != 1 || !strings.HasPrefix(log.lines[0], "WARN") || !strings.Contains(log.lines[0], "PERMISSION_DENIED") ||
		strings.Contains(log.lines[0], `"error"`) {
		t.Errorf("log de un 4xx inesperado: %v", log.lines)
	}
}

func TestHealth(t *testing.T) {
	var pingErr error
	health := httpserver.NewHealth(func(context.Context) error { return pingErr })
	status := func(path string) int {
		rec := httptest.NewRecorder()
		health.Handler().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		return rec.Code
	}
	if status("/livez") != 200 || status("/readyz") != 200 {
		t.Fatal("con la base arriba, /livez y /readyz responden 200")
	}
	pingErr = errors.New("sin base")
	if status("/livez") != 200 || status("/readyz") != 503 {
		t.Error("sin base, /readyz es 503 y /livez sigue en 200")
	}
	pingErr = nil
	health.Drain()
	if status("/readyz") != 503 {
		t.Error("al apagar, /readyz pasa a 503")
	}
}
