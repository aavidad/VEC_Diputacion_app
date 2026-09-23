package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
)

type motorComisionPrueba struct{ llamadas int }

func (m *motorComisionPrueba) Calcular(_ context.Context, s ports.SolicitudCalculoRuta) (ports.ResultadoCalculoRuta, error) {
	m.llamadas++
	if len(s.Coordenadas) != 2 || s.Coordenadas[0].Nombre != "Granada" || s.Coordenadas[1].Nombre != "Albolote" {
		return ports.ResultadoCalculoRuta{}, errors.New("orden invertido")
	}
	return ports.ResultadoCalculoRuta{Motor: "osrm_on_premise", VersionGrafo: "grafo-sintetico-v1", Alternativas: []ports.AlternativaRuta{{Tramos: []ports.TramoRuta{{DistanciaMetros: 12000}}}}}, nil
}

type tarifasComisionPrueba struct{}

func (tarifasComisionPrueba) Consultar(_ context.Context, version string, grupo int, vehiculo string, fecha time.Time) (domain.TarifaComisionProvisional, error) {
	if version != VersionTarifaComisionProvisional || vehiculo != "automovil" || fecha.Format("2006-01-02") != "2026-09-23" {
		return domain.TarifaComisionProvisional{}, errors.New("tarifa incorrecta")
	}
	return domain.TarifaComisionProvisional{Dieta: domain.TarifaNacionalProvisional{VersionRef: version, Rotulo: domain.RotuloTarifaProvisional, PaisISO2: "ES", Grupo: grupo, VigenteDesde: "2026-09-23", ManutencionCentimos: 5000, AlojamientoTopeCentimos: 6000}, EURPorKM: "0.2600"}, nil
}
func TestPrepararComisionConservaOrdenYCalculaImporteServidor(t *testing.T) {
	motor := &motorComisionPrueba{}
	p, err := NuevoPreparadorComision(map[string]ports.CoordenadaRuta{"18087": {Nombre: "Granada", Latitud: 37.17, Longitud: -3.59}, "18003": {Nombre: "Albolote", Latitud: 37.23, Longitud: -3.65}}, motor, tarifasComisionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	s := ports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_0123456789abcdef", FechaInicio: "2026-09-23", FechaFin: "2026-09-23", HoraInicio: "08:00", HoraFin: "18:00", Motivo: "Visita técnica", CodigosRuta: []string{"18087", "18003"}}
	result, err := p.Preparar(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if motor.llamadas != 1 || result.Calculo == nil || result.Calculo.Kilometros != "12.0000" || result.Calculo.ImporteKilometrajeCentimos != 312 || len(result.Calculo.OpcionesDieta) != 3 || result.Calculo.TramosRuta[0].OrigenCodigo != "18087" {
		t.Fatalf("cálculo no acreditado: %+v", result.Calculo)
	}
	s.CodigosRuta[1] = "fuera"
	if _, err = p.Preparar(context.Background(), s); err == nil || motor.llamadas != 1 {
		t.Fatal("localidad ajena llegó a OSRM")
	}
}
