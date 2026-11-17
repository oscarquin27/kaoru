package domain

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"kaoru/internal/core/pkg/axis"
)

// LinkKind es el tipo de un enlace externo.
type LinkKind string

// Tipos de enlace.
const (
	LinkGoogleSheets LinkKind = "google_sheets"
	LinkYouTrack     LinkKind = "youtrack"
	LinkSharedFolder LinkKind = "shared_folder"
	LinkWiki         LinkKind = "wiki"
	LinkOther        LinkKind = "other"
)

const (
	maxURLLength   = 2000
	maxLabelLength = 200
)

// SheetRef es lo que identifica una hoja de Google dentro de su URL.
type SheetRef struct {
	Id    string
	Gid   string
	Range string
}

var (
	sheetIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{10,}$`)
	gidPattern     = regexp.MustCompile(`^\d+$`)
	rangePattern   = regexp.MustCompile(`^[A-Za-z0-9_!':$ ]{1,100}$`)
)

// ParseSheetLink extrae el ID de la hoja, la pestaña (gid) y el rango de una URL de
// Google Sheets. Solo valida la forma: el servidor no comprueba que la hoja exista.
func ParseSheetLink(raw string) (SheetRef, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host != "docs.google.com" {
		return SheetRef{}, false
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 3 || segments[0] != "spreadsheets" {
		return SheetRef{}, false
	}
	var ref SheetRef
	for i := 1; i+1 < len(segments); i++ {
		if segments[i] == "d" {
			ref.Id = segments[i+1]
			break
		}
	}
	if !sheetIDPattern.MatchString(ref.Id) {
		return SheetRef{}, false
	}
	// La pestaña y el rango pueden venir en la consulta o en el fragmento; manda el fragmento.
	fragment, _ := url.ParseQuery(u.Fragment)
	for _, params := range []url.Values{u.Query(), fragment} {
		if gid := params.Get("gid"); gidPattern.MatchString(gid) {
			ref.Gid = gid
		}
		if rng := params.Get("range"); rangePattern.MatchString(rng) {
			ref.Range = rng
		}
	}
	return ref, true
}

// LinkSnapshot es el contenido de un enlace.
type LinkSnapshot struct {
	Id            string
	RequirementId string
	Kind          LinkKind
	Label         string
	URL           string
	// Sheet solo existe en enlaces de Google Sheets.
	Sheet      *SheetRef
	IsOfficial bool
	// SnapshotDocumentId y SnapshotCode son el DOC con la instantánea; vacíos si no hay.
	SnapshotDocumentId string
	SnapshotCode       string
	AddedBy            UserRef
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Link es un enlace externo de un REQ.
type Link struct {
	s LinkSnapshot
}

// NewLinkParams son los datos para agregar un enlace. Kind y Label vacíos se deducen de la URL.
type NewLinkParams struct {
	Id            string
	RequirementId string
	URL           string
	Kind          string
	Label         string
	IsOfficial    bool
	AddedBy       UserRef
	Now           time.Time
}

// NewLink valida y crea un enlace. Una URL de Google Sheets siempre es de tipo google_sheets.
func NewLink(p NewLinkParams) (Link, error) {
	var errs axis.FieldErrors
	raw := strings.TrimSpace(p.URL)
	parsed, err := url.Parse(raw)
	switch {
	case raw == "":
		errs.Add(axis.Required("url", "Pega la URL del enlace."))
	case len(raw) > maxURLLength:
		errs.Add(axis.InvalidField("url", "La URL es demasiado larga."))
	case err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "":
		errs.Add(axis.InvalidField("url", "La URL debe empezar con https:// o http://."))
	}
	if err := errs.Err(); err != nil {
		return Link{}, err
	}

	s := LinkSnapshot{
		Id: p.Id, RequirementId: p.RequirementId, URL: raw, IsOfficial: p.IsOfficial,
		AddedBy: p.AddedBy, CreatedAt: p.Now, UpdatedAt: p.Now,
	}
	sheet, isSheet := ParseSheetLink(raw)
	switch {
	case isSheet:
		s.Kind, s.Sheet = LinkGoogleSheets, &sheet
	case p.Kind == string(LinkGoogleSheets):
		errs.Add(axis.InvalidField("url", "No es una URL de Google Sheets."))
	case p.Kind == "":
		s.Kind = LinkOther
		if strings.Contains(parsed.Host, "youtrack") {
			s.Kind = LinkYouTrack
		}
	default:
		s.Kind = validKind(p.Kind, &errs)
	}
	s.Label = validLabel(p.Label, &errs)
	if s.Label == "" {
		s.Label = defaultLabel(s.Sheet, parsed)
	}
	if err := errs.Err(); err != nil {
		return Link{}, err
	}
	return Link{s: s}, nil
}

// RehydrateLink arma un enlace ya guardado. Solo lo usa el adaptador de persistencia.
func RehydrateLink(s LinkSnapshot) Link { return Link{s: s} }

// Snapshot devuelve el contenido del enlace.
func (l Link) Snapshot() LinkSnapshot { return l.s }

// NeedsSnapshot dice si es un enlace vivo de Google Sheets sin instantánea adjunta.
func (l Link) NeedsSnapshot() bool {
	return l.s.Kind == LinkGoogleSheets && l.s.SnapshotDocumentId == ""
}

// SnapshotEdit cambia la instantánea: Set dice si se toca; DocumentId vacío la quita.
type SnapshotEdit struct {
	Set        bool
	DocumentId string
	Code       string
}

// LinkEdit son los campos a cambiar de un enlace. La URL no se edita: se agrega otro enlace.
type LinkEdit struct {
	Label      *string
	IsOfficial *bool
	Snapshot   SnapshotEdit
}

// Apply valida y aplica una edición. Devuelve solo los campos que cambiaron de verdad.
func (l *Link) Apply(e LinkEdit, now time.Time) ([]FieldChange, error) {
	var errs axis.FieldErrors
	next := l.s
	if e.Label != nil {
		if next.Label = validLabel(*e.Label, &errs); next.Label == "" {
			errs.Add(axis.Required("label", "Escribe una etiqueta."))
		}
	}
	if e.IsOfficial != nil {
		next.IsOfficial = *e.IsOfficial
	}
	if e.Snapshot.Set {
		next.SnapshotDocumentId, next.SnapshotCode = e.Snapshot.DocumentId, e.Snapshot.Code
	}
	if err := errs.Err(); err != nil {
		return nil, err
	}
	var changes []FieldChange
	add := func(field, old, new string) {
		if old != new {
			changes = append(changes, FieldChange{Field: field, Old: old, New: new})
		}
	}
	add("link.label", l.s.Label, next.Label)
	add("link.isOfficial", boolText(l.s.IsOfficial), boolText(next.IsOfficial))
	add("link.snapshotDocument", l.s.SnapshotCode, next.SnapshotCode)
	if len(changes) > 0 {
		next.UpdatedAt = now
		l.s = next
	}
	return changes, nil
}

func validKind(raw string, errs *axis.FieldErrors) LinkKind {
	switch k := LinkKind(raw); k {
	case LinkYouTrack, LinkSharedFolder, LinkWiki, LinkOther:
		return k
	default:
		errs.Add(axis.InvalidField("kind", "El tipo de enlace no existe."))
		return ""
	}
}

func validLabel(raw string, errs *axis.FieldErrors) string {
	label := strings.TrimSpace(raw)
	if len([]rune(label)) > maxLabelLength {
		errs.Add(axis.InvalidField("label", "La etiqueta no puede pasar de 200 caracteres."))
	}
	return label
}

// googleLabels nombra los enlaces de Google que no son hojas por la primera parte de su ruta.
var googleLabels = map[string]string{"document": "Google Docs", "presentation": "Google Slides", "forms": "Google Forms", "drawings": "Google Drawings"}

// defaultLabel arma una etiqueta legible en lugar de la URL cruda.
func defaultLabel(sheet *SheetRef, u *url.URL) string {
	if sheet == nil {
		first, _, _ := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
		switch {
		case u.Host == "docs.google.com" && googleLabels[first] != "":
			return googleLabels[first]
		case u.Host == "drive.google.com":
			return "Google Drive"
		}
		return u.Host
	}
	label := "Google Sheets"
	if sheet.Gid != "" {
		label += " · pestaña " + sheet.Gid
	}
	if sheet.Range != "" {
		label += " · " + sheet.Range
	}
	return label
}

func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
