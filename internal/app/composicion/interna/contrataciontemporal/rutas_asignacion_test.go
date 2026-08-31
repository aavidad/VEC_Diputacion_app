package contrataciontemporal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

type autoridadAsignacionComposicionPrueba struct {
	resoluciones atomic.Int64
}

func (a *autoridadAsignacionComposicionPrueba) ResolverContextoCanalAsignacion(
	context.Context,
) (httpinterno.ContextoCanalAsignacion, error) {
	a.resoluciones.Add(1)
	return httpinterno.ContextoCanalAsignacion{}, nil
}

type ejecutorAsignacionComposicionPrueba struct {
	asignaciones   atomic.Int64
	reasignaciones atomic.Int64
}

func (e *ejecutorAsignacionComposicionPrueba) Asignar(
	context.Context,
	application.SolicitudAsignarUnidad,
) (ports.ReciboAsignacion, error) {
	e.asignaciones.Add(1)
	return ports.ReciboAsignacion{}, nil
}

func (e *ejecutorAsignacionComposicionPrueba) Reasignar(
	context.Context,
	application.SolicitudReasignarUnidad,
) (ports.ReciboAsignacion, error) {
	e.reasignaciones.Add(1)
	return ports.ReciboAsignacion{}, nil
}

func (e *ejecutorAsignacionComposicionPrueba) total() int64 {
	return e.asignaciones.Load() + e.reasignaciones.Load()
}

func TestNuevasRutasRegistranAsignacionYReasignacionUnaVezAlFinal(
	t *testing.T,
) {
	t.Parallel()
	rutas, err := NuevasRutas(dependenciasRutasPrueba())
	if err != nil {
		t.Fatalf("construir rutas: %v", err)
	}
	if len(rutas) < 2 {
		t.Fatalf("numero de rutas = %d", len(rutas))
	}
	penultima, ultima := rutas[len(rutas)-2], rutas[len(rutas)-1]
	if penultima.Ruta != httpinterno.RutaAsignaciones ||
		ultima.Ruta != httpinterno.RutaReasignaciones {
		t.Fatalf("rutas finales inesperadas: %#v, %#v", penultima, ultima)
	}
	if penultima.Manejador != ultima.Manejador {
		t.Fatal("asignacion y reasignacion no comparten manejador")
	}
	registradas := map[string]int{}
	for _, ruta := range rutas {
		registradas[ruta.Ruta]++
	}
	for _, ruta := range []string{
		httpinterno.RutaAsignaciones,
		httpinterno.RutaReasignaciones,
	} {
		if registradas[ruta] != 1 {
			t.Fatalf("registros de %q = %d", ruta, registradas[ruta])
		}
	}
}

func TestNuevasRutasAsignacionRechazanDependenciasNulasAtomicamente(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre   string
		preparar func(*DependenciasRutas)
	}{
		{"autoridad nil", func(d *DependenciasRutas) {
			d.AutoridadAsignacion = nil
		}},
		{"ejecutor nil", func(d *DependenciasRutas) {
			d.EjecutorAsignacion = nil
		}},
		{"autoridad nil tipada", func(d *DependenciasRutas) {
			var nula *autoridadAsignacionComposicionPrueba
			d.AutoridadAsignacion = nula
		}},
		{"ejecutor nil tipado", func(d *DependenciasRutas) {
			var nulo *ejecutorAsignacionComposicionPrueba
			d.EjecutorAsignacion = nulo
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasPrueba()
			caso.preparar(&dependencias)
			rutas, err := NuevasRutas(dependencias)
			if rutas != nil ||
				!errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
				t.Fatalf("resultado = (%#v, %v)", rutas, err)
			}
		})
	}
}

func TestRutasAsignacionYReasignacionDenegadasNoInvocanManejador(
	t *testing.T,
) {
	t.Parallel()
	for _, ruta := range []string{
		httpinterno.RutaAsignaciones,
		httpinterno.RutaReasignaciones,
	} {
		ruta := ruta
		t.Run(ruta, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasPrueba()
			autoridadInterior := dependencias.AutoridadAsignacion.(*autoridadAsignacionComposicionPrueba)
			ejecutor := dependencias.EjecutorAsignacion.(*ejecutorAsignacionComposicionPrueba)
			autoridadExterior := &autoridadDespachoContratacionEspiaPrueba{
				err: httpapi.ErrAccesoRutaExactaDenegado,
			}
			handler := nuevoHandlerContratacionConDependenciasPrueba(
				t, dependencias, autoridadExterior,
			)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(
				respuesta,
				nuevaPeticionAsignacionComposicionPrueba(ruta),
			)
			llamadas, rutaAutorizada := autoridadExterior.estado()
			if respuesta.Code != http.StatusForbidden || llamadas != 1 ||
				rutaAutorizada != ruta ||
				autoridadInterior.resoluciones.Load() != 0 ||
				ejecutor.total() != 0 {
				t.Fatalf(
					"estado=%d exterior=%d/%q autoridad=%d ejecutor=%d cuerpo=%s",
					respuesta.Code, llamadas, rutaAutorizada,
					autoridadInterior.resoluciones.Load(), ejecutor.total(),
					respuesta.Body.String(),
				)
			}
		})
	}
}

func nuevaPeticionAsignacionComposicionPrueba(ruta string) *http.Request {
	cuerpo := `{"expediente_ref":"expediente:asignacion:001",` +
		`"version_esperada":1,` +
		`"clave_idempotencia":"12345678-1234-4567-8abc-123456789abc",` +
		`"unidad_ref":"unidad:gestora:001",` +
		`"responsable_ref":"responsable:opaco:001"}`
	if ruta == httpinterno.RutaReasignaciones {
		cuerpo = strings.TrimSuffix(cuerpo, "}") +
			`,"motivo_reasignacion_clave":"cambio_unidad",` +
			`"observaciones":"Reasignacion motivada."}`
	}
	peticion := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}
