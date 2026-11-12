// Package bootstrap es la composición: lee la configuración, arma y conecta todo.
package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ConfigPathEnv es la variable con la ruta de config.json.
const ConfigPathEnv = "KAORU_CONFIG_PATH"

// Config es la configuración completa. Se lee una vez al arrancar.
type Config struct {
	HTTP      HTTPConfig      `json:"http"`
	Postgres  PostgresConfig  `json:"postgres"`
	OIDC      OIDCConfig      `json:"oidc"`
	Documents DocumentsConfig `json:"documents"`
	Log       LogConfig       `json:"log"`
}

// HTTPConfig configura los servidores.
type HTTPConfig struct {
	// Addr es el puerto público, detrás de Nginx.
	Addr string `json:"addr"`
	// InternalAddr sirve /livez y /readyz; Nginx no lo publica.
	InternalAddr string `json:"internalAddr"`
	// PublicOrigin es el origen público de la app; se compara con la cabecera Origin.
	PublicOrigin string `json:"publicOrigin"`
	// TrustedProxies es la red de Nginx.
	TrustedProxies []string `json:"trustedProxies"`
	MaxBodyBytes   int64    `json:"maxBodyBytes"`
}

// PostgresConfig trae las dos conexiones: la que migra y la que ejecuta.
type PostgresConfig struct {
	OwnerURL string `json:"ownerUrl"`
	AppURL   string `json:"appUrl"`
}

// OIDCConfig son los datos del realm de Keycloak.
type OIDCConfig struct {
	Issuer     string `json:"issuer"`
	JWKSURL    string `json:"jwksUrl"`
	ClientID   string `json:"clientId"`
	LogoutURL  string `json:"logoutUrl"`
	AccountURL string `json:"accountUrl"`
}

// DocumentsConfig configura los adjuntos.
type DocumentsConfig struct {
	// Dir es el directorio (un volumen) donde vive el contenido, nombrado por su SHA-256.
	Dir string `json:"dir"`
	// MaxBytes es el tamaño máximo de una subida.
	MaxBytes int64 `json:"maxBytes"`
	// AllowedExtensions es la lista de tipos permitidos, como extensiones con punto.
	AllowedExtensions []string `json:"allowedExtensions"`
	// ConverterURL es la dirección interna del conversor de Office a PDF (el servicio opcional
	// `libreoffice`). Vacía: los archivos de Office solo se descargan.
	ConverterURL string `json:"converterUrl"`
}

// LogConfig elige el formato de los logs: "json" o "text".
type LogConfig struct {
	Format string `json:"format"`
}

// LoadConfig lee config.json de la ruta de KAORU_CONFIG_PATH, reemplaza las
// referencias ${VARIABLE} con el entorno (${VARIABLE:-} si es opcional) y valida que no falte nada.
func LoadConfig() (Config, error) {
	path := os.Getenv(ConfigPathEnv)
	if path == "" {
		return Config{}, fmt.Errorf("falta la variable %s con la ruta de config.json", ConfigPathEnv)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // la ruta la pone quien despliega
	if err != nil {
		return Config{}, fmt.Errorf("leer configuración: %w", err)
	}
	return ParseConfig(raw, os.LookupEnv)
}

// ParseConfig arma la configuración a partir del JSON y de una función que lee el entorno.
func ParseConfig(raw []byte, lookup func(string) (string, bool)) (Config, error) {
	var missing []string
	expanded := os.Expand(string(raw), func(name string) string {
		// ${VARIABLE:-} marca una variable opcional: si falta, queda vacía.
		name, optional := strings.CutSuffix(name, ":-")
		value, ok := lookup(name)
		if (!ok || value == "") && !optional {
			missing = append(missing, name)
		}
		// El valor va dentro de una cadena JSON: se escapa.
		escaped, _ := json.Marshal(value)
		return string(escaped[1 : len(escaped)-1])
	})
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("faltan variables de entorno: %s", strings.Join(missing, ", "))
	}
	var cfg Config
	if err := json.Unmarshal([]byte(expanded), &cfg); err != nil {
		return Config{}, fmt.Errorf("configuración ilegible: %w", err)
	}
	return cfg, cfg.validate()
}

func (c Config) validate() error {
	required := map[string]string{
		"http.addr":         c.HTTP.Addr,
		"http.internalAddr": c.HTTP.InternalAddr,
		"http.publicOrigin": c.HTTP.PublicOrigin,
		"postgres.ownerUrl": c.Postgres.OwnerURL,
		"postgres.appUrl":   c.Postgres.AppURL,
		"oidc.issuer":       c.OIDC.Issuer,
		"oidc.jwksUrl":      c.OIDC.JWKSURL,
		"oidc.clientId":     c.OIDC.ClientID,
		"oidc.logoutUrl":    c.OIDC.LogoutURL,
		"documents.dir":     c.Documents.Dir,
	}
	var errs []error
	for name, value := range required {
		if value == "" {
			errs = append(errs, fmt.Errorf("falta %s en la configuración", name))
		}
	}
	if c.HTTP.MaxBodyBytes <= 0 {
		errs = append(errs, errors.New("http.maxBodyBytes debe ser mayor que cero"))
	}
	if c.Documents.MaxBytes <= 0 {
		errs = append(errs, errors.New("documents.maxBytes debe ser mayor que cero"))
	}
	if len(c.Documents.AllowedExtensions) == 0 {
		errs = append(errs, errors.New("documents.allowedExtensions no puede estar vacío"))
	}
	if c.Log.Format != "json" && c.Log.Format != "text" {
		errs = append(errs, errors.New(`log.format debe ser "json" o "text"`))
	}
	return errors.Join(errs...)
}
