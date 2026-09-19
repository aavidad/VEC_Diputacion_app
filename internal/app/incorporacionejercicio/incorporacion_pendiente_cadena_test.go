package incorporacionejercicio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

type autoridadHTTPIncorporacionPendientePrueba struct{}

func (autoridadHTTPIncorporacionPendientePrueba) ResolverContextoIncorporacionEjercicioV2(context.Context) error {
	return nil
}

func TestCadenaRealPreparacionPendienteHastaHTTPNoCreaEfectos(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	otro := c.plan.Copia()
	otro.SolicitudPersonal.ExpedienteRef = "expediente:ejercicio:otro"
	bytes := documentoPlanesV2Prueba(t, otro)
	fuente, err := NuevaFuentePlanesPreparacionV2(bytes, ternaPlanesV2Prueba(bytes))
	if err != nil {
		t.Fatal(err)
	}
	c.p.c.Planes = fuente
	servicio, err := Nuevo(Configuracion{
		Preparador: c.p, FuentePersonal: c.app.s.c.FuentePersonal, TernaPersonal: c.app.s.c.TernaPersonal,
		ProveedorAlta: c.app.s.c.ProveedorAlta, TransaccionAlta: c.app.s.c.TransaccionAlta,
		LectorPersonal: c.app.s.c.LectorPersonal, Confirmador: c.app.s.c.Confirmador, Reloj: c.app.s.c.Reloj,
	})
	if err != nil {
		t.Fatal(err)
	}
	h, err := httpct.NuevoManejadorIncorporacionEjercicioV2(autoridadHTTPIncorporacionPendientePrueba{}, servicio)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, httpct.RutaIncorporacionEjercicioV2+"?expediente_ref="+c.plan.SolicitudPersonal.ExpedienteRef, nil)
	h.ServeHTTP(w, r)
	var cuerpo struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusConflict || cuerpo.Error.Codigo != "preparacion_pendiente" {
		t.Fatalf("estado=%d codigo=%q", w.Code, cuerpo.Error.Codigo)
	}
	if c.app.alta.n != 0 || c.app.ctTX.llamadas != 0 || c.a.store.registros != 0 {
		t.Fatal("la cadena de preparación pendiente creó un efecto")
	}
}
