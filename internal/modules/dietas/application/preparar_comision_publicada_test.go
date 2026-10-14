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

const versionPublicadaPrueba = "tarifa:publicada:20261014"

type tarifasPublicadasPrueba struct {
	reglaDistintaGrupo int
	grupoAusente       int
}

func (f tarifasPublicadasPrueba) Consultar(_ context.Context, version string, grupo int, vehiculo string, fecha time.Time) (domain.CatalogoLiquidacionPublicado, error) {
	var vacia domain.CatalogoLiquidacionPublicado
	if (version != "" && version != versionPublicadaPrueba) || fecha.Format("2006-01-02") != "2026-10-14" || grupo == f.grupoAusente {
		return vacia, errors.New("sin publicación")
	}
	c := domain.CatalogoLiquidacionPublicado{Referencia: "catalogo:grupo:" + string(rune('0'+grupo)) + ":" + vehiculo,
		Version: "publicacion:uno", PublicacionRef: "publicacion:acto:uno", FechaPublicacion: "2026-10-01",
		VigenteDesde: "2026-10-01", VigenteHasta: "2027-12-31", Ambito: "granada:ordinario",
		PaisISO2: "ES", Grupo: grupo, Vehiculo: vehiculo, VersionTarifaRef: versionPublicadaPrueba,
		RotuloClave: "dietas_tarifa_publicada", Normas: []domain.NormaLiquidacionPublicada{
			{Referencia: "norma:bases", Articulo: "30", URL: "https://www.dipgra.es/bases"},
			{Referencia: "norma:boe", Articulo: "10", URL: "https://www.boe.es/boe"}},
		Reglas: []domain.ReglaLiquidacionPublicada{{ReglaLiquidacionPropuesta: domain.ReglaLiquidacionPropuesta{
			Referencia: "regla:manutencion:grupo:" + string(rune('0'+grupo)), Tipo: "dieta", Concepto: "manutencion",
			Grupo: grupo, TopeCentimos: int64(grupo * 1000)}, NormaRef: "norma:bases"}}}
	if vehiculo == "propio" {
		c.Reglas = append(c.Reglas, domain.ReglaLiquidacionPublicada{ReglaLiquidacionPropuesta: domain.ReglaLiquidacionPropuesta{
			Referencia: "regla:kilometraje", Tipo: "kilometraje", Concepto: "vehiculo_propio", CentimosPorKM: 26}, NormaRef: "norma:boe"})
	}
	c.ReglaDevengo = domain.ReglaDevengoPublicada{Referencia: "regla:devengo:uno", NormaRef: "norma:bases",
		Configuracion: domain.ConfiguracionDevengoPublicada{Esquema: "nacional_ordinaria_v1", Zona: "Europe/Madrid",
			DuracionMinimaMismoDiaHoras: 5, HoraSalida100AntesDe: 14, HoraSalida50AntesDe: 22,
			HoraRegreso50DespuesDe: 14, HoraRegresoMismoDiaDespuesDe: 16, DiasMaximos: 31,
			PorcentajeMismoDia: 50, PorcentajeSalidaTemprana: 100, PorcentajeSalidaMedia: 50,
			PorcentajeRegreso: 50, PorcentajeIntermedio: 100, Alojamiento: "cerrado_pendiente_rrhh"}}
	if grupo == f.reglaDistintaGrupo {
		c.ReglaDevengo.Configuracion.PorcentajeMismoDia = 75
	}
	c.ReglaDevengo.HuellaSHA256, _ = c.ReglaDevengo.HuellaCalculada()
	c.HuellaSHA256, _ = c.HuellaCalculada()
	return c, nil
}

func TestPreparadorPublicadoCreaYEdiTaConLaMismaRegla(t *testing.T) {
	puntos := map[string]ports.CoordenadaRuta{"18087": {Nombre: "Granada", Latitud: 37.17, Longitud: -3.59},
		"18003": {Nombre: "Albolote", Latitud: 37.23, Longitud: -3.65}}
	motor := &motorComisionPrueba{}
	p, err := NuevoPreparadorComisionPublicado(puntos, motor, tarifasPublicadasPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	alta := ports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_0123456789abcdef", FechaInicio: "2026-10-14",
		FechaFin: "2026-10-14", HoraInicio: "08:00", HoraFin: "18:00", Motivo: "Visita técnica", CodigosRuta: []string{"18087", "18003"}}
	creada, err := p.Preparar(context.Background(), alta)
	if err != nil || creada.Calculo == nil || creada.Calculo.VersionTarifa != versionPublicadaPrueba ||
		creada.Calculo.Rotulo != "dietas_tarifa_publicada" || creada.Calculo.ImporteKilometrajeCentimos != 312 ||
		creada.Calculo.OpcionesDieta[1].Calculo.ManutencionCentimos != 1000 || motor.llamadas != 1 {
		t.Fatalf("alta no calculada con publicación: %+v %v", creada.Calculo, err)
	}
	relacion := "rel_0123456789abcdefghijklmn"
	asignacion := &ports.AsignacionDietasAcreditada{AsignacionRef: "ads_0123456789abcdefghijklmn", RelacionRef: relacion,
		PersonaRef: "per_0123456789abcdefghijklmn", UnidadRef: "unidad:prueba", CentroRef: "centro:prueba",
		AdministrativoPersonaRef: "per_aaaaaaaaaaaaaaaaaaaaaa", ResponsablePersonaRef: "per_bbbbbbbbbbbbbbbbbbbbbb",
		GrupoDieta: 2, VigenteDesde: "2026-10-01", Version: 1, ReciboRef: "rad_0123456789abcdef0123456789abcdef",
		DecisionRef: "decision:prueba", EfectoRef: relacion, ConsumoHuellaSHA256: strings.Repeat("a", 64),
		AuditoriaRef: "auditoria:prueba", RegistradaEn: time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC)}
	edicion := ports.SolicitudEditarComisionPropia{Referencia: "dco_0123456789abcdefghijklmn", ClaveIdempotencia: "clave_0123456789abcdef",
		VersionEsperada: 1, RelacionRef: relacion, FechaInicio: alta.FechaInicio, FechaFin: alta.FechaFin,
		HoraInicio: alta.HoraInicio, HoraFin: alta.HoraFin, Motivo: alta.Motivo, CodigosRuta: alta.CodigosRuta,
		Asignacion: asignacion, TramosAceptados: []int{0}, VersionTarifaAceptada: versionPublicadaPrueba}
	preparada, err := p.PrepararEdicion(context.Background(), edicion)
	if err != nil || preparada.Documento == nil || preparada.Documento.AlojamientoTopeCentimos != 0 ||
		preparada.Documento.ManutencionCentimos != 1000 || preparada.Calculo.EURPorKM != "0.0000" || motor.llamadas != 1 {
		t.Fatalf("edición sin vehículo: %+v %v", preparada.Documento, err)
	}
	edicion.VehiculoPropio = true
	edicion.Rutas = []domain.RutaDeclaradaComision{{CodigosRuta: alta.CodigosRuta, AjusteKilometros: "1.0000", MotivoAjuste: "Desvío justificado"}}
	preparada, err = p.PrepararEdicion(context.Background(), edicion)
	if err != nil || preparada.Documento == nil || preparada.Documento.KilometrajeCentimos != 338 ||
		preparada.Calculo.EURPorKM != "0.2600" || motor.llamadas != 2 {
		t.Fatalf("edición con vehículo: %+v %v", preparada.Documento, err)
	}
}

func TestPreparadorPublicadoRechazaGruposConReglasDistintas(t *testing.T) {
	p, err := NuevoPreparadorComisionPublicado(map[string]ports.CoordenadaRuta{
		"18087": {Nombre: "Granada", Latitud: 37.17, Longitud: -3.59},
		"18003": {Nombre: "Albolote", Latitud: 37.23, Longitud: -3.65}}, &motorComisionPrueba{},
		tarifasPublicadasPrueba{reglaDistintaGrupo: 3})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Preparar(context.Background(), ports.SolicitudCrearBorradorPropio{
		ClaveIdempotencia: "clave_0123456789abcdef", FechaInicio: "2026-10-14", FechaFin: "2026-10-14",
		HoraInicio: "08:00", HoraFin: "18:00", Motivo: "Visita técnica", CodigosRuta: []string{"18087", "18003"}})
	if !errors.Is(err, domain.ErrLiquidacionPublicada) {
		t.Fatalf("reglas incongruentes admitidas: %v", err)
	}
}
