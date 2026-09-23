package postgres

import (
	"context"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestConsultaMiBolsaPostgreSQLRechazaSolicitudSinMaterialAntesDeAbrirTransaccion(t *testing.T) {
	consulta := &ConsultaMiBolsaPostgreSQL{}
	_, err := consulta.ConsultarMiBolsa(context.Background(), puertosbolsa.SolicitudConsultaMiBolsa{})
	if err != puertosbolsa.ErrConsultaMiBolsaInvalida {
		t.Fatalf("error = %v", err)
	}
}

func TestNuevaConsultaMiBolsaPostgreSQLRechazaPoolNulo(t *testing.T) {
	if _, err := NuevaConsultaMiBolsaPostgreSQL(nil); err != puertosbolsa.ErrMaterialMiBolsaNoDisponible {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodificarInstantaneaMiBolsaMapeaSnakeCase(t *testing.T) {
	ahora := time.Date(2026, time.September, 19, 12, 0, 0, 123000000, time.UTC)
	contenido := []byte(`{"consultada_en":"2026-09-19T12:00:00.123Z","participaciones":[{"bolsa":"bol_0123456789012345678901","categoria":"cat_0123456789012345678901","version":3,"orden_inicial":2,"total_instantanea":4,"estado_bolsa":"vigente","vigente_desde":"2026-09-18T12:00:00Z","vigente_hasta":null}]}`)
	resultado, err := decodificarInstantaneaMiBolsa(contenido, ahora)
	if err != nil || len(resultado.Participaciones) != 1 {
		t.Fatalf("resultado = %#v, error = %v", resultado, err)
	}
	p := resultado.Participaciones[0]
	if p.OrdenInicial != 2 || p.TotalInstantanea != 4 || p.EstadoBolsa != "vigente" || p.Bolsa == "" || p.Categoria == "" {
		t.Fatalf("proyección no mapeada: %#v", p)
	}
}

func TestDecodificarMiBolsaConSituacionActualMinimizada(t *testing.T) {
	ahora := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	contenido := []byte(`{"consultada_en":"2026-09-23T10:00:00Z","participaciones":[{"bolsa":"bolsa:01","categoria":"Auxiliar","version":3,"orden_inicial":2,"total_instantanea":4,"estado_bolsa":"vigente","vigente_desde":"2026-09-01T00:00:00Z","vigente_hasta":null,"situacion_actual":{"estado":"no_disponible","desde":"2026-09-20T10:00:00Z","hasta":null,"fecha_disponible":null}}]}`)
	resultado, err := decodificarInstantaneaMiBolsa(contenido, ahora)
	if err != nil {
		t.Fatal(err)
	}
	s := resultado.Participaciones[0].SituacionActual
	if s == nil || s.Estado != "no_disponible" || s.Desde != time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC) || s.Hasta != nil || s.FechaDisponible != nil {
		t.Fatalf("situación actual incorrecta: %#v", s)
	}
}
