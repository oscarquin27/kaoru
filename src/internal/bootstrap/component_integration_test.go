//go:build integration

package bootstrap

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

type componentBody struct {
	Code   string `json:"code"`
	System struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"system"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Retired     bool   `json:"retired"`
	CreatedBy   struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type componentPage struct {
	Total int             `json:"total"`
	Items []componentBody `json:"items"`
}

type componentRefs struct {
	Items []struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"items"`
}

func (r componentRefs) names() string {
	names := make([]string, len(r.Items))
	for i, item := range r.Items {
		names[i] = item.Name
	}
	return strings.Join(names, ",")
}

func (p componentPage) names() string {
	names := make([]string, len(p.Items))
	for i, item := range p.Items {
		names[i] = item.Name
	}
	return strings.Join(names, ",")
}

var componentCode = regexp.MustCompile(`^COMP-\d{4}-\d{4}$`)

// El catálogo de componentes: quién lo mantiene, qué valida, cómo se retira y qué queda auditado.
func TestComponentCatalog(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	// Las pruebas comparten la base y el nombre no se repite por sistema: cada corrida usa nombres propios.
	tag := fmt.Sprintf("cat%d", time.Now().UnixNano())
	body := func(system, kind, name string) map[string]any {
		return map[string]any{"system": system, "kind": kind, "name": name}
	}

	// Crear: solo quien tiene comp.manage (dev, qa y qa-admin).
	for _, user := range []string{"manager", "ops", "auditor"} {
		a.call(user, "POST", "/api/components", body("SYPAGO", "microservice", tag+"-x"), 403, &p)
		if p.Code != "PERMISSION_DENIED" {
			t.Errorf("%s no mantiene el catálogo: %+v", user, p)
		}
	}
	a.call("dev", "POST", "/api/components", map[string]any{"kind": "dll"}, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "system" || p.Errors[0].Code != "required" {
		t.Errorf("sin sistema: %+v", p)
	}
	a.call("dev", "POST", "/api/components", body("OTRO", "microservice", tag+"-x"), 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "system" || p.Errors[0].Code != "invalid" {
		t.Errorf("sistema desconocido: %+v", p)
	}
	a.call("dev", "POST", "/api/components", body("SYPAGO", "dll", "  "), 422, &p)
	if len(p.Errors) != 2 || p.Errors[0].Field != "kind" || p.Errors[1].Field != "name" || p.Errors[1].Code != "required" {
		t.Errorf("un formulario junta todos sus errores: %+v", p.Errors)
	}
	a.call("dev", "POST", "/api/components", body("SYPAGO", "script", strings.Repeat("x", 121)), 422, nil)

	var pagos, firmas, viejo, simf componentBody
	a.call("dev", "POST", "/api/components", map[string]any{
		"system": "SYPAGO", "kind": "microservice", "name": "  " + tag + "-pagos  ", "description": " Servicio de pagos ",
	}, 201, &pagos)
	if !componentCode.MatchString(pagos.Code) || pagos.Name != tag+"-pagos" || pagos.Description != "Servicio de pagos" ||
		pagos.Kind != "microservice" || pagos.Retired || pagos.System.Key != "SYPAGO" || pagos.System.Name != "SyPago" ||
		pagos.CreatedBy.DisplayName != "J. Díaz" || pagos.CreatedAt == "" || pagos.CreatedAt != pagos.UpdatedAt {
		t.Fatalf("COMP creado inesperado: %+v", pagos)
	}
	a.call("qa", "POST", "/api/components", body("SYPAGO", "library", tag+"-firmas"), 201, &firmas)
	a.call("admin", "POST", "/api/components", body("SYPAGO", "script", tag+"-viejo"), 201, &viejo)

	// El nombre no se repite dentro de un sistema; en otro sistema sí vale.
	a.call("qa", "POST", "/api/components", body("SYPAGO", "script", " "+tag+"-pagos "), 422, &p)
	if p.Code != "INVALID_REQUEST" || len(p.Errors) != 1 || p.Errors[0].Field != "name" || p.Errors[0].Code != "invalid" {
		t.Errorf("nombre repetido: %+v", p)
	}
	a.call("qa", "POST", "/api/components", body("SIMF", "kafka_topic", tag+"-pagos"), 201, &simf)

	// Leer no exige permiso.
	path := "/api/components/" + pagos.Code
	var got componentBody
	a.call("ops", "GET", path, nil, 200, &got)
	if got != pagos {
		t.Errorf("lo leído difiere de lo creado:\n%+v\n%+v", got, pagos)
	}
	a.call("ops", "GET", "/api/components/COMP-2026-9999", nil, 404, nil)
	a.call("ops", "GET", "/api/components/basura", nil, 404, nil)

	// Editar: el sistema no cambia; el nombre sigue sin repetirse.
	a.call("manager", "PATCH", path, map[string]any{"name": "x"}, 403, nil)
	a.call("qa", "PATCH", "/api/components/COMP-2026-9999", map[string]any{"name": "x"}, 404, nil)
	a.call("qa", "PATCH", path, map[string]any{"name": tag + "-firmas"}, 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "name" || p.Errors[0].Code != "invalid" {
		t.Errorf("renombrar a un nombre que ya existe: %+v", p)
	}
	a.call("qa", "PATCH", path, map[string]any{"kind": "dll", "name": ""}, 422, &p)
	if len(p.Errors) != 2 {
		t.Errorf("edición inválida: %+v", p.Errors)
	}
	a.call("qa", "PATCH", path, map[string]any{"kind": "db_schema", "name": tag + "-cobros", "system": "SIMF"}, 200, &got)
	if got.Kind != "db_schema" || got.Name != tag+"-cobros" || got.System.Key != "SYPAGO" || got.Description != "Servicio de pagos" ||
		got.CreatedBy.DisplayName != "J. Díaz" {
		t.Errorf("edición no aplicada: %+v", got)
	}
	a.call("ops", "GET", path, nil, 200, &got)
	if got.Name != tag+"-cobros" || got.Kind != "db_schema" {
		t.Errorf("la edición no quedó guardada: %+v", got)
	}

	// Retirar: no se borra, deja de ofrecerse.
	a.call("ops", "PATCH", "/api/components/"+viejo.Code, map[string]any{"retired": true}, 403, nil)
	a.call("dev", "PATCH", "/api/components/"+viejo.Code, map[string]any{"retired": true}, 200, &got)
	if !got.Retired {
		t.Errorf("debía quedar retirado: %+v", got)
	}
	a.call("ops", "GET", "/api/components/"+viejo.Code, nil, 200, &got)
	if !got.Retired || got.Name != tag+"-viejo" {
		t.Errorf("un COMP retirado se sigue leyendo: %+v", got)
	}

	// Lista: por nombre, sin retirados salvo que se pidan, con filtros de sistema, clase y nombre.
	var page componentPage
	a.call("ops", "GET", "/api/components?q="+tag, nil, 200, &page)
	if page.Total != 3 || page.names() != tag+"-cobros,"+tag+"-firmas,"+tag+"-pagos" {
		t.Errorf("lista por nombre: %d %s", page.Total, page.names())
	}
	a.call("ops", "GET", "/api/components?includeRetired=true&q="+tag, nil, 200, &page)
	if page.Total != 4 {
		t.Errorf("lista con retirados = %d, quiero 4", page.Total)
	}
	a.call("ops", "GET", "/api/components?system=SYPAGO&q="+tag, nil, 200, &page)
	if page.Total != 2 || page.names() != tag+"-cobros,"+tag+"-firmas" {
		t.Errorf("lista por sistema: %d %s", page.Total, page.names())
	}
	a.call("ops", "GET", "/api/components?kind=library&q="+tag, nil, 200, &page)
	if page.Total != 1 || page.Items[0].Code != firmas.Code {
		t.Errorf("lista por clase: %+v", page)
	}
	a.call("ops", "GET", "/api/components?system=SYPAGO&kind=script&includeRetired=true&q="+tag, nil, 200, &page)
	if page.Total != 1 || page.Items[0].Code != viejo.Code || !page.Items[0].Retired {
		t.Errorf("lista con todos los filtros: %+v", page)
	}
	a.call("ops", "GET", "/api/components?q="+simf.Code, nil, 200, &page)
	if page.Total != 1 || page.Items[0].System.Key != "SIMF" {
		t.Errorf("lista por código: %+v", page)
	}
	a.call("ops", "GET", "/api/components?size=1&page=2&q="+tag, nil, 200, &page)
	if page.Total != 3 || len(page.Items) != 1 || page.Items[0].Name != tag+"-firmas" {
		t.Errorf("segunda página: %+v", page)
	}
	a.call("ops", "GET", "/api/components?kind=script&q="+tag, nil, 200, &page)
	if page.Total != 0 || page.Items == nil {
		t.Errorf("una lista vacía es [] y no null: %+v", page)
	}
	a.call("ops", "GET", "/api/components?kind=dll", nil, 422, nil)
	a.call("ops", "GET", "/api/components?size=500", nil, 422, nil)

	// Reactivar.
	a.call("qa", "PATCH", "/api/components/"+viejo.Code, map[string]any{"retired": false}, 200, &got)
	if got.Retired {
		t.Errorf("debía quedar vigente: %+v", got)
	}
	// Una edición que no cambia nada no deja rastro.
	a.call("qa", "PATCH", "/api/components/"+viejo.Code, map[string]any{"retired": false, "name": tag + "-viejo"}, 200, nil)

	// Todo quedó auditado, del más reciente al más antiguo.
	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+pagos.Code, nil, 200, &audit)
	if audit.Total != 3 || audit.Items[2].Action != "create" || audit.Items[2].NewValue != tag+"-pagos" ||
		audit.Items[2].Actor.DisplayName != "J. Díaz" || audit.Items[1].Field != "kind" || audit.Items[1].OldValue != "microservice" ||
		audit.Items[1].NewValue != "db_schema" || audit.Items[0].Field != "name" || audit.Items[0].OldValue != tag+"-pagos" ||
		audit.Items[0].NewValue != tag+"-cobros" || audit.Items[0].Actor.DisplayName != "A. Rojas" {
		t.Errorf("auditoría del COMP editado: %+v", audit)
	}
	a.call("auditor", "GET", "/api/audit?entityCode="+viejo.Code, nil, 200, &audit)
	if audit.Total != 3 || audit.Items[1].Field != "retired" || audit.Items[1].OldValue != "false" || audit.Items[1].NewValue != "true" ||
		audit.Items[0].Field != "retired" || audit.Items[0].NewValue != "false" {
		t.Errorf("auditoría del COMP retirado y reactivado: %+v", audit)
	}
}

// Impacto: un REQ declara los componentes que toca y un TC los que ejercita, siempre de su sistema y vigentes.
func TestRequirementAndTestCaseDeclareComponents(t *testing.T) {
	a := newAPI(t)
	var p problemBody
	tag := fmt.Sprintf("imp%d", time.Now().UnixNano())
	component := func(system, kind, name string) string {
		t.Helper()
		var out componentBody
		a.call("dev", "POST", "/api/components", map[string]any{"system": system, "kind": kind, "name": tag + "-" + name}, 201, &out)
		return out.Code
	}
	svc := component("SYPAGO", "microservice", "b-api")
	lib := component("SYPAGO", "library", "a-lib")
	old := component("SYPAGO", "script", "c-old")
	other := component("SIMF", "microservice", "z-simf")
	a.call("dev", "PATCH", "/api/components/"+old, map[string]any{"retired": true}, 200, nil)

	var req requirementBody
	a.call("manager", "POST", "/api/requirements", map[string]any{"title": "Impacto " + tag, "system": "SYPAGO"}, 201, &req)
	var uc, tc codeBody
	a.call("qa", "POST", "/api/requirements/"+req.Code+"/functional-cases", map[string]any{"title": "Pagar"}, 201, &uc)
	a.call("qa", "POST", "/api/test-cases", map[string]any{
		"title": "Paga " + tag, "system": "SYPAGO", "steps": "1. Pagar", "expectedResult": "Pagado", "functionalCases": []string{uc.Code},
	}, 201, &tc)

	set := func(codes ...string) map[string]any {
		return map[string]any{"components": append([]string{}, codes...)}
	}
	cases := []struct {
		name       string
		path       string
		missing    string
		writer     string // tiene el permiso
		writerName string
		reader     string // no lo tiene
		entityCode string
	}{
		{"REQ", "/api/requirements/" + req.Code + "/components", "/api/requirements/REQ-2026-9999/components", "dev", "J. Díaz", "ops", req.Code},
		{"TC", "/api/test-cases/" + tc.Code + "/components", "/api/test-cases/TC-2026-9999/components", "qa", "A. Rojas", "dev", tc.Code},
	}
	for _, c := range cases {
		func() {
			var refs componentRefs
			a.call(c.reader, "GET", c.path, nil, 200, &refs)
			if refs.Items == nil || len(refs.Items) != 0 {
				t.Errorf(c.name+": sin componentes la lista es []: %+v", refs)
			}
			a.call("ops", "GET", c.missing, nil, 404, nil)

			// Declarar: exige el permiso de editar la entidad.
			a.call(c.reader, "PUT", c.path, set(svc), 403, &p)
			if p.Code != "PERMISSION_DENIED" {
				t.Errorf(c.name+": %s no declara componentes: %+v", c.reader, p)
			}
			a.call(c.writer, "PUT", c.missing, set(svc), 404, nil)
			a.call(c.writer, "PUT", c.path, map[string]any{}, 422, &p)
			if len(p.Errors) != 1 || p.Errors[0].Field != "components" || p.Errors[0].Code != "required" {
				t.Errorf(c.name+": sin lista: %+v", p)
			}
			for name, codes := range map[string][]string{
				"no existe": {svc, "COMP-2026-9999"}, "mal formado": {"basura"}, "de otro sistema": {svc, other}, "retirado": {old},
			} {
				a.call(c.writer, "PUT", c.path, set(codes...), 422, &p)
				if p.Code != "INVALID_REQUEST" || len(p.Errors) != 1 || p.Errors[0].Field != "components" || p.Errors[0].Code != "invalid" {
					t.Errorf(c.name+": componente %s: %+v", name, p)
				}
			}
			a.call(c.reader, "GET", c.path, nil, 200, &refs)
			if len(refs.Items) != 0 {
				t.Errorf(c.name+": una declaración rechazada no guarda nada: %+v", refs)
			}

			// Camino feliz: devuelve los componentes por nombre; repetir un código no duplica.
			a.call(c.writer, "PUT", c.path, set(svc, lib, svc), 200, &refs)
			if refs.names() != tag+"-a-lib,"+tag+"-b-api" || refs.Items[0].Code != lib || refs.Items[0].Kind != "library" {
				t.Errorf(c.name+": componentes declarados: %+v", refs)
			}
			a.call("ops", "GET", c.path, nil, 200, &refs)
			if refs.names() != tag+"-a-lib,"+tag+"-b-api" {
				t.Errorf(c.name+": componentes leídos: %+v", refs)
			}
			// Declarar lo mismo no audita; reemplazar y vaciar sí.
			a.call(c.writer, "PUT", c.path, set(lib, svc), 200, nil)
			a.call(c.writer, "PUT", c.path, set(svc), 200, &refs)
			if refs.names() != tag+"-b-api" {
				t.Errorf(c.name+": la lista se reemplaza: %+v", refs)
			}
			a.call(c.writer, "PUT", c.path, set(), 200, &refs)
			if refs.Items == nil || len(refs.Items) != 0 {
				t.Errorf(c.name+": una lista vacía quita todos: %+v", refs)
			}
			a.call(c.writer, "PUT", c.path, set(lib), 200, nil)

			var audit auditPage
			a.call("auditor", "GET", "/api/audit?size=4&entityCode="+c.entityCode, nil, 200, &audit)
			both := strings.Join(sorted(svc, lib), ", ")
			if len(audit.Items) != 4 || audit.Items[3].Field != "components" || audit.Items[3].Action != "update" ||
				audit.Items[3].OldValue != "" || audit.Items[3].NewValue != both || audit.Items[3].Actor.DisplayName != c.writerName ||
				audit.Items[2].OldValue != both || audit.Items[2].NewValue != svc ||
				audit.Items[1].OldValue != svc || audit.Items[1].NewValue != "" ||
				audit.Items[0].OldValue != "" || audit.Items[0].NewValue != lib {
				t.Errorf(c.name+": auditoría de los componentes: %+v", audit.Items)
			}
		}()
	}

	// Retirar un componente ya declarado conserva la declaración, pero no deja volver a declararlo.
	a.call("qa", "PATCH", "/api/components/"+lib, map[string]any{"retired": true}, 200, nil)
	var refs componentRefs
	a.call("ops", "GET", cases[0].path, nil, 200, &refs)
	if len(refs.Items) != 1 || refs.Items[0].Code != lib {
		t.Errorf("el REQ conserva el componente retirado: %+v", refs)
	}
	a.call("dev", "PUT", cases[0].path, set(lib, svc), 422, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field != "components" {
		t.Errorf("un componente retirado no se vuelve a declarar: %+v", p)
	}
	a.call("dev", "PUT", cases[0].path, set(svc), 200, &refs)
	if len(refs.Items) != 1 || refs.Items[0].Code != svc {
		t.Errorf("se puede quitar el componente retirado: %+v", refs)
	}
}

func sorted(codes ...string) []string {
	if len(codes) == 2 && codes[0] > codes[1] {
		codes[0], codes[1] = codes[1], codes[0]
	}
	return codes
}
