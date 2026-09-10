package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultorSeguimientoV2Prueba struct {
	vista    ports.VistaSeguimientoIncorporacionV2
	err      error
	llamadas int
}

func (c *consultorSeguimientoV2Prueba) ConsultarSeguimientoIncorporacionV2(context.Context, string) (ports.VistaSeguimientoIncorporacionV2, error) {
	c.llamadas++
	return c.vista, c.err
}

type autoridadConsultaSeguimientoV2Prueba struct{ err error }

func (a autoridadConsultaSeguimientoV2Prueba) ResolverContextoIncorporacionEjercicioV2(context.Context) error {
	return a.err
}

func vistaSeguimientoV2Prueba() ports.VistaSeguimientoIncorporacionV2 {
	t := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	p := domain.IntervaloSeguimiento{Desde: t.AddDate(0, 0, 1), Hasta: t.AddDate(0, 0, 2)}
	return ports.VistaSeguimientoIncorporacionV2{Esquema: "vec.contratacion-temporal.seguimiento-incorporacion.v2", Alcance: "original_incorporacion",
		ExpedienteRef: "expediente:ejercicio:1", VersionExpediente: 8, ReciboIncorporacionRef: "recibo:ejercicio:1", SeguimientoRef: "seguimiento:ejercicio:1",
		VersionSeguimiento: 1, EstadoClave: "vigente", Periodo: p, RegistradoEn: t, EjercicioSintetico: true,
		Actuaciones: []ports.ActuacionVisibleSeguimientoIncorporacionV2{{ActuacionRef: "actuacion:ejercicio:1", TransicionClave: "confirmar_incorporacion",
			EstadoOrigen: "pendiente_incorporacion", EstadoDestino: "vigente", EfectivoEn: p.Desde, RegistradaEn: t,
			Documentos: []domain.DocumentoSeguimiento{{TipoClave: "resolucion_ejercicio", Referencia: "documento:ejercicio:1"}}}}}
}

func TestManejadorConsultaSeguimientoV2DenegacionConflictoYOtraReferencia(t *testing.T) {
	casos := []struct {
		nombre, expediente string
		autoridad, err     error
		want               int
		llamadas           int
	}{
		{"denegacion_antes_de_exportar", "expediente:ejercicio:1", ports.ErrDenegadaIncorporacionAplicacion, nil, http.StatusForbidden, 0},
		{"sin_recibo", "expediente:ejercicio:1", nil, ports.ErrConflictoIncorporacionAplicacion, http.StatusConflict, 1},
		{"otra_referencia", "expediente:ejercicio:otra", nil, nil, http.StatusServiceUnavailable, 1},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			a := autoridadConsultaSeguimientoV2Prueba{err: tc.autoridad}
			vista := vistaSeguimientoV2Prueba()
			if tc.err != nil {
				vista = ports.VistaSeguimientoIncorporacionV2{}
			}
			c := &consultorSeguimientoV2Prueba{vista: vista, err: tc.err}
			h, err := NuevoManejadorConsultaSeguimientoV2(a, c)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodGet, RutaConsultaSeguimientoV2+"?expediente_ref="+tc.expediente, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want || c.llamadas != tc.llamadas {
				t.Fatalf("status=%d llamadas=%d", w.Code, c.llamadas)
			}
		})
	}
}
