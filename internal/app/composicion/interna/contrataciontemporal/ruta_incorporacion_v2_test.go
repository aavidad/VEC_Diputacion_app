package contrataciontemporal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorIncorporacionComposicionPrueba struct {
	consultas, confirmaciones int
	exp                       string
	version                   uint64
}

func (e *ejecutorIncorporacionComposicionPrueba) Consultar(_ context.Context, exp string) (ct.ProyeccionIncorporacionAplicacionV2, error) {
	e.consultas++
	e.exp = exp
	return ct.ProyeccionIncorporacionAplicacionV2{}, ct.ErrConflictoIncorporacionAplicacion
}
func (e *ejecutorIncorporacionComposicionPrueba) Confirmar(_ context.Context, i ct.IntencionIncorporacionAplicacionV2) (ct.ReciboIncorporacionAplicacionV2, error) {
	e.confirmaciones++
	e.exp = i.ExpedienteRef
	e.version = i.VersionActualExpedienteObservada
	return ct.ReciboIncorporacionAplicacionV2{}, ct.ErrConflictoIncorporacionAplicacion
}
func TestIncorporacionV2EnsamblajeHTTPUnico(t *testing.T) {
	var ejecutores []*ejecutorIncorporacionComposicionPrueba
	h := manejadorIncorporacionV2{nueva: func(context.Context) (ct.ServicioIncorporacionAplicacionV2, error) {
		e := new(ejecutorIncorporacionComposicionPrueba)
		ejecutores = append(ejecutores, e)
		return e, nil
	}}
	body := `{"expediente_ref":"expediente:real:1","solicitud_personal_ref":"solicitud:estable:1","version_actual_expediente_observada":8,"motivo_clave":"confirmar_incorporacion","documentos_refs":[],"confirma_revision_personal":true,"confirma_ejercicio_sintetico":true}`
	for _, method := range []string{http.MethodGet, http.MethodGet, http.MethodPost} {
		ruta := httpinterno.RutaIncorporacionEjercicioV2
		if method == http.MethodGet {
			ruta += "?expediente_ref=expediente:real:1"
		}
		r := httptest.NewRequest(method, ruta, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if method == http.MethodGet {
			r.Body = http.NoBody
			r.ContentLength = 0
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusConflict {
			t.Fatalf("contrato o ejecución divergente: %d", w.Code)
		}
	}
	if len(ejecutores) != 3 || ejecutores[0] == ejecutores[1] || ejecutores[0].consultas != 1 || ejecutores[0].confirmaciones != 0 || ejecutores[2].confirmaciones != 1 || ejecutores[2].version != 8 {
		t.Fatal("estado cruzado o GET escribió")
	}
	antes := len(ejecutores)
	r := httptest.NewRequest(http.MethodGet, httpinterno.RutaIncorporacionEjercicioV2+"?expediente_ref=expediente:real:1&solicitud_personal_ref=inventada", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || len(ejecutores) != antes {
		t.Fatal("aceptó selector extra o creó autoridad antes de validar")
	}
	h.nueva = func(context.Context) (ct.ServicioIncorporacionAplicacionV2, error) {
		return nil, ct.ErrDenegadaIncorporacionAplicacion
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, httpinterno.RutaIncorporacionEjercicioV2+"?expediente_ref=expediente:real:1", nil))
	if w.Code != http.StatusForbidden {
		t.Fatal("denegación no conservada")
	}
}

func TestIncorporacionV2EnsamblajeMontajeOpcional(t *testing.T) {
	x := dependenciasRutasPrueba()
	rutas, err := NuevasRutas(x)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rutas {
		if r.Ruta == httpinterno.RutaIncorporacionEjercicioV2 {
			t.Fatal("montaje sin configuración")
		}
	}
	x.IncorporacionV2 = new(inc.ServidorV2PostgreSQL)
	rutas, err = NuevasRutas(x)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, r := range rutas {
		if r.Ruta == httpinterno.RutaIncorporacionEjercicioV2 {
			n++
		}
	}
	if n != 1 {
		t.Fatal("ruta duplicada o no montada")
	}
	if _, err := NuevaRutaIncorporacionV2(nil); err == nil {
		t.Fatal("servidor nulo aceptado")
	}
}
