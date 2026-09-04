package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

func TestCapacidadNoCompuestaContratacionTemporalDeniegaYAuditaUnaVez(t *testing.T) {
	t.Parallel()

	var registro bytes.Buffer
	capacidad, err := nuevaCapacidadNoCompuestaContratacionTemporalDesarrollo(&registro)
	if err != nil {
		t.Fatalf("construir capacidad: %v", err)
	}
	if err := capacidad.denegarRuta(
		context.Background(),
		httpinterno.RutaSeleccionLlamamiento,
	); !errors.Is(err, vechttp.ErrAutoridadRutaExactaNoDisponible) {
		t.Fatalf("denegacion inesperada: %v", err)
	}
	var entrada map[string]string
	if err := json.NewDecoder(&registro).Decode(&entrada); err != nil {
		t.Fatalf("decodificar auditoria: %v", err)
	}
	esperada := map[string]string{
		"modulo":     "contratacion_temporal",
		"ruta":       httpinterno.RutaSeleccionLlamamiento,
		"resultado":  "denegado",
		"motivo":     "servicio_no_disponible",
		"clave_i18n": claveI18nCapacidadNoCompuestaContratacionTemporal,
	}
	if len(entrada) != len(esperada) {
		t.Fatalf("auditoria contiene campos no autorizados: %#v", entrada)
	}
	for clave, valor := range esperada {
		if entrada[clave] != valor {
			t.Fatalf("campo %q: obtenido %q, esperado %q", clave, entrada[clave], valor)
		}
	}
}

func TestCapacidadNoCompuestaContratacionTemporalAcotaOchoRutas(t *testing.T) {
	t.Parallel()

	var registro bytes.Buffer
	capacidad, err := nuevaCapacidadNoCompuestaContratacionTemporalDesarrollo(&registro)
	if err != nil {
		t.Fatalf("construir capacidad: %v", err)
	}
	if len(rutasCapacidadNoCompuestaContratacionTemporal) != 8 {
		t.Fatalf("rutas no compuestas: %d", len(rutasCapacidadNoCompuestaContratacionTemporal))
	}
	for _, ruta := range []string{
		httpinterno.RutaAltaSolicitudes,
		httpinterno.RutaRegistroAnalisisRRHH,
		httpinterno.RutaPropuestaCobertura,
		httpinterno.RutaDecisionCobertura,
		httpinterno.RutaRectificacionCobertura,
		httpinterno.RutaResultadoCobertura,
		httpinterno.RutaAsignaciones,
	} {
		if capacidad.esRuta(ruta) {
			t.Fatalf("ruta compuesta marcada no compuesta: %s", ruta)
		}
	}
	if !capacidad.esRuta(httpinterno.RutaRectificacionAnalisisRRHH) {
		t.Fatal("rectificacion de analisis salio de la barrera no compuesta")
	}
	if !capacidad.esRuta(httpinterno.RutaReasignaciones) {
		t.Fatal("reasignacion fuera del corte salio de la barrera no compuesta")
	}
}

func TestCapacidadNoCompuestaContratacionTemporalExigeRegistro(t *testing.T) {
	t.Parallel()

	if capacidad, err := nuevaCapacidadNoCompuestaContratacionTemporalDesarrollo(nil); capacidad != nil || err == nil {
		t.Fatalf("constructor no fallo cerrado: capacidad=%#v err=%v", capacidad, err)
	}
}
