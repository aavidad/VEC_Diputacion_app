package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorIncorporacionEjercicioV2Prueba struct {
	err       error
	consultas int
}

func (e *ejecutorIncorporacionEjercicioV2Prueba) Consultar(context.Context, string) (ports.ProyeccionIncorporacionAplicacionV2, error) {
	e.consultas++
	return ports.ProyeccionIncorporacionAplicacionV2{}, e.err
}

func (e *ejecutorIncorporacionEjercicioV2Prueba) Confirmar(context.Context, ports.IntencionIncorporacionAplicacionV2) (ports.ReciboIncorporacionAplicacionV2, error) {
	return ports.ReciboIncorporacionAplicacionV2{}, errors.New("no debe confirmar")
}

func TestManejadorIncorporacionEjercicioV2PreparacionPendienteYFalloNoTipado(t *testing.T) {
	for _, caso := range []struct {
		nombre, codigo string
		err            error
		estado         int
	}{
		{"pendiente", "preparacion_pendiente", ports.ErrPreparacionIncorporacionPendiente, http.StatusConflict},
		{"composicion", "servicio_no_disponible", ports.ErrComposicionIncorporacionAplicacion, http.StatusServiceUnavailable},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ejecutor := &ejecutorIncorporacionEjercicioV2Prueba{err: caso.err}
			h, err := NuevoManejadorIncorporacionEjercicioV2(autoridadConsultaSeguimientoV2Prueba{}, ejecutor)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, RutaIncorporacionEjercicioV2+"?expediente_ref=expediente:ejercicio:sin-plan", nil)
			h.ServeHTTP(w, r)
			var cuerpo struct {
				Error struct {
					Codigo string `json:"codigo"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
				t.Fatal(err)
			}
			if w.Code != caso.estado || cuerpo.Error.Codigo != caso.codigo || ejecutor.consultas != 1 {
				t.Fatalf("estado=%d codigo=%q consultas=%d", w.Code, cuerpo.Error.Codigo, ejecutor.consultas)
			}
		})
	}
}
