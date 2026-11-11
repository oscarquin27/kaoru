//go:build integration

package bootstrap

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// Dos escrituras simultáneas sobre la misma entidad van una tras otra: la segunda ve lo que
// dejó la primera. Sin el bloqueo de fila, varias transiciones iguales «ganarían» a la vez.
func TestConcurrentTransitionsOnTheSameEntityDoNotOverlap(t *testing.T) {
	a := newAPI(t)
	var chg changeBody
	a.call("manager", "POST", "/api/changes", map[string]any{"title": "Concurrencia", "system": "SYPAGO"}, 201, &chg)

	const parallel = 16
	var ok, rejected, other atomic.Int32
	var wg sync.WaitGroup
	for range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(map[string]any{"to": "approved"})
			req := httptest.NewRequest("POST", "/api/changes/"+chg.Code+"/transitions", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer manager")
			req.Header.Set("Origin", testOrigin)
			rec := httptest.NewRecorder()
			a.handler.ServeHTTP(rec, req)
			switch rec.Code {
			case 200:
				ok.Add(1)
			case 409:
				rejected.Add(1)
			default:
				other.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 || rejected.Load() != parallel-1 || other.Load() != 0 {
		t.Fatalf("de %d aprobaciones simultáneas debe valer una sola: %d ok, %d rechazadas, %d con otro código",
			parallel, ok.Load(), rejected.Load(), other.Load())
	}

	var audit auditPage
	a.call("auditor", "GET", "/api/audit?entityCode="+chg.Code, nil, 200, &audit)
	transitions := 0
	for _, item := range audit.Items {
		if item.Action == "transition" {
			transitions++
		}
	}
	if transitions != 1 {
		t.Errorf("la auditoría debe tener una sola transición, tiene %d", transitions)
	}
}
