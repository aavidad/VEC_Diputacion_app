package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
)

const VersionTarifaComisionProvisional = "provisional:rd462:20260923"

type motorComisionPrueba struct{ llamadas int }

func (m *motorComisionPrueba) Calcular(_ context.Context, s ports.SolicitudCalculoRuta) (ports.ResultadoCalculoRuta, error) {
	m.llamadas++
	if len(s.Coordenadas) != 2 || s.Coordenadas[0].Nombre != "Granada" || s.Coordenadas[1].Nombre != "Albolote" {
		return ports.ResultadoCalculoRuta{}, errors.New("orden invertido")
	}
	return ports.ResultadoCalculoRuta{Motor: "osrm_on_premise", VersionGrafo: "grafo-sintetico-v1", Alternativas: []ports.AlternativaRuta{{Tramos: []ports.TramoRuta{{DistanciaMetros: 12000}}}}}, nil
}

type tarifasComisionPrueba struct{}

func (tarifasComisionPrueba) ConsultarRegla(_ context.Context, version string, fecha time.Time) (domain.ReglaDevengoProvisional, error) {
	if (version != "" && version != VersionTarifaComisionProvisional) || fecha.Format("2006-01-02") != "2026-09-23" {
		return domain.ReglaDevengoProvisional{}, errors.New("regla no disponible")
	}
	return domain.ReglaDevengoProvisional{ReglaRef: "provisional:regla:nacional-ordinaria:20260923", VersionTarifaRef: VersionTarifaComisionProvisional, PaisISO2: "ES", Variante: "nacional_ordinaria", HuellaSHA256: strings.Repeat("a", 64), Configuracion: domain.ConfiguracionDevengoProvisional{Regla: "nacional_ordinaria_provisional_v1", Zona: "Europe/Madrid", DuracionMinimaMismoDiaHoras: 5, HoraSalida100AntesDe: 14, HoraSalida50AntesDe: 22, HoraRegreso50DespuesDe: 14, HoraRegresoMismoDiaDespuesDe: 16, DiasMaximos: 31, Alojamiento: "tope_pendiente_justificante", PorcentajeMismoDia: 50, PorcentajeSalidaTemprana: 100, PorcentajeSalidaMedia: 50, PorcentajeRegreso: 50, PorcentajeIntermedio: 100, PorcentajeAlojamientoTope: 100}}, nil
}

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
	medianoche := s
	medianoche.HoraInicio = "00:30"
	medianoche.HoraFin = "06:00"
	if _, err = p.Preparar(context.Background(), medianoche); err != nil {
		t.Fatalf("fecha civil local cambió al consultar catálogo: %v", err)
	}
	s.CodigosRuta[1] = "fuera"
	if _, err = p.Preparar(context.Background(), s); err == nil || motor.llamadas != 2 {
		t.Fatal("localidad ajena llegó a OSRM")
	}
}

func TestPrepararEdicionSinVehiculoNoConsultaOSRMYConDosRutasSumaAjuste(t *testing.T) {
	motor := &motorComisionPrueba{}
	p, err := NuevoPreparadorComision(map[string]ports.CoordenadaRuta{"18087": {Nombre: "Granada", Latitud: 37.17, Longitud: -3.59}, "18003": {Nombre: "Albolote", Latitud: 37.23, Longitud: -3.65}}, motor, tarifasComisionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	relacion := "rel_0123456789abcdefghijklmn"
	asignacion := &ports.AsignacionDietasAcreditada{AsignacionRef: "ads_0123456789abcdefghijklmn", RelacionRef: relacion, PersonaRef: "per_0123456789abcdefghijklmn", UnidadRef: "unidad:prueba", CentroRef: "centro:prueba", AdministrativoPersonaRef: "per_aaaaaaaaaaaaaaaaaaaaaa", ResponsablePersonaRef: "per_bbbbbbbbbbbbbbbbbbbbbb", GrupoDieta: 2, VigenteDesde: "2026-09-01", Version: 1, ReciboRef: "rad_0123456789abcdef0123456789abcdef", DecisionRef: "decision:prueba", EfectoRef: relacion, ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:prueba", RegistradaEn: time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)}
	base := ports.SolicitudEditarComisionPropia{Referencia: "dco_0123456789abcdefghijklmn", ClaveIdempotencia: "clave_0123456789abcdef", VersionEsperada: 1, RelacionRef: relacion, FechaInicio: "2026-09-23", FechaFin: "2026-09-23", HoraInicio: "08:00", HoraFin: "18:00", Motivo: "Visita técnica", CodigosRuta: []string{"18087", "18003"}, Asignacion: asignacion, TramosAceptados: []int{0}, VersionTarifaAceptada: VersionTarifaComisionProvisional}
	s, err := p.PrepararEdicion(context.Background(), base)
	if err != nil || motor.llamadas != 0 || s.Calculo == nil || s.Calculo.Kilometros != "0.0000" || s.Calculo.Procedencia != "sin_vehiculo_propio" || s.Documento == nil || s.Documento.GrupoDieta != 2 || s.Documento.TotalOrientativoCentimos <= 0 {
		t.Fatalf("sin vehículo: %v llamadas=%d cálculo=%+v", err, motor.llamadas, s.Calculo)
	}
	base.VehiculoPropio = true
	base.Rutas = []domain.RutaDeclaradaComision{{CodigosRuta: []string{"18087", "18003"}, AjusteKilometros: "1.0000", MotivoAjuste: "Desvío justificado"}, {CodigosRuta: []string{"18087", "18003"}, AjusteKilometros: "0.0000"}}
	s, err = p.PrepararEdicion(context.Background(), base)
	if err != nil || motor.llamadas != 2 || s.Calculo == nil || len(s.Calculo.Rutas) != 2 || s.Calculo.Kilometros != "25.0000" || s.Calculo.ImporteKilometrajeCentimos != 650 || s.Documento == nil || s.Documento.KilometrajeCentimos != 650 {
		t.Fatalf("dos rutas: %v llamadas=%d cálculo=%+v", err, motor.llamadas, s.Calculo)
	}
}
