package application

import (
	"context"
	"errors"
	"testing"
	"time"

	caldomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calports "vec-diputacion-granada/internal/modules/calendarios/ports"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type centroC4Prueba struct {
	resultado personalports.ResultadoUnidadTrabajoPropiaCronos
	llamadas  int
}

type relojSecuenciaC4 struct {
	ahora    time.Time
	llamadas int
}

func (r *relojSecuenciaC4) AhoraUTC() time.Time {
	r.llamadas++
	if r.llamadas == 1 {
		return r.ahora
	}
	return r.ahora.Add(time.Second)
}

func (p *centroC4Prueba) ConsultarUnidadTrabajoPropiaCronos(_ context.Context, q personalports.ConsultaUnidadTrabajoPropiaCronos) (personalports.ResultadoUnidadTrabajoPropiaCronos, error) {
	p.llamadas++
	if q.Fecha != "2026-06-15" || q.Actor.Validar() != nil || q.ConocidoEn.IsZero() {
		return personalports.ResultadoUnidadTrabajoPropiaCronos{}, errors.New("consulta sin identidad o fecha")
	}
	return p.resultado, nil
}

type calendarioC4Prueba struct {
	resultado calports.CalendarioCentro
	err       error
	llamadas  int
}

func (*calendarioC4Prueba) Centros(context.Context, int, time.Time) ([]calports.CentroConCalendario, error) {
	return nil, nil
}
func (*calendarioC4Prueba) CalcularPlazo(context.Context, calports.SolicitudCalculoPlazo) (calports.ResultadoCalculoPlazo, error) {
	return calports.ResultadoCalculoPlazo{}, nil
}
func (c *calendarioC4Prueba) CalendarioCentro(_ context.Context, q calports.SolicitudCalendarioCentro) (calports.CalendarioCentro, error) {
	c.llamadas++
	if q.CentroRef != "centro:demo" || q.Anio != 2026 || q.ConocidoEn.IsZero() {
		return calports.CalendarioCentro{}, errors.New("centro o corte equivocado")
	}
	r := c.resultado
	r.ConocidoEn = q.ConocidoEn
	return r, c.err
}

func resultadoCentroC4Prueba(ahora time.Time) personalports.ResultadoUnidadTrabajoPropiaCronos {
	return personalports.ResultadoUnidadTrabajoPropiaCronos{
		Cobertura: "completa",
		Unidad: &personalports.UnidadTrabajoCronos{
			EmpleadoRef: "emp_0123456789abcdefghijkl", RelacionRef: "rel_0123456789abcdefghijkl",
			OrganismoRef: "organismo:demo", UnidadClave: "unidad:demo", RegimenRef: "regimen:demo", Desde: "2026-01-01",
			FuenteRef: "fuente:personal", FuenteVersionRef: "version:personal:1", Version: 1,
		},
		Evidencia: personalports.EvidenciaRegistroEmpleadoB2{
			ReciboRef: "recibo:personal:1", DecisionRef: "decision:1", AuditoriaRef: "auditoria:1", ConsultadaEn: ahora.Add(-time.Second),
		},
	}
}

type organizacionC4Prueba struct {
	resultado personalports.ResultadoConsultaOrganizacionHistorica
	llamadas  int
}

type programacionC4Prueba struct {
	resultado *ProgramacionLaboralC4
	llamadas  int
}

func (p *programacionC4Prueba) ConsultarProgramacionC4(_ context.Context, _ cronosports.OrdenConsultaSaldo, fecha string) (*ProgramacionLaboralC4, error) {
	p.llamadas++
	if fecha != "2026-06-15" {
		return nil, errors.New("fecha ajena")
	}
	return p.resultado, nil
}

func (o *organizacionC4Prueba) ConsultarC4(_ context.Context, q personaldomain.SolicitudConsultaOrganizacionHistorica, unidad personalports.ResultadoUnidadTrabajoPropiaCronos) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	o.llamadas++
	if q.Selector.OrganismoRef != "organismo:demo" || q.Selector.VigenteEn != "2026-06-15" || unidad.Unidad == nil || unidad.Unidad.UnidadClave != "unidad:demo" {
		return personalports.ResultadoConsultaOrganizacionHistorica{}, errors.New("organización no ligada")
	}
	return o.resultado, nil
}

func organizacionPublicadaC4Prueba() personalports.ResultadoConsultaOrganizacionHistorica {
	return personalports.ResultadoConsultaOrganizacionHistorica{
		Pagina: personalports.PaginaOrganizacionHistorica{
			Cobertura: personalports.CoberturaFuentesOrganizacionHistorica{Unidades: "parcial"},
			Unidades: []personaldomain.UnidadOrganizacionHistorica{
				{Traza: personaldomain.TrazaOrganizacionHistorica{ID: "centro:id", FuenteRef: "fuente:organizacion"}, ClaveCatalogo: "centro:demo", Tipo: "centro", CatalogoVersion: 1},
				{Traza: personaldomain.TrazaOrganizacionHistorica{ID: "unidad:id", FuenteRef: "fuente:organizacion"}, ClaveCatalogo: "unidad:demo", PadreID: "centro:id", Tipo: "delegacion", CatalogoVersion: 1},
			},
		},
		Evidencia: personalports.EvidenciaConsultaOrganizacionHistorica{ReciboRef: "recibo:organizacion:1", AuditoriaRef: "auditoria:organizacion:1"},
	}
}

func calendarioPublicadoC4Prueba(ahora time.Time) calports.CalendarioCentro {
	f, _ := caldomain.ParsearFechaCivil("2026-06-15")
	publicada, _ := caldomain.ParsearFechaCivil("2025-12-01")
	procedencia := caldomain.Procedencia{Norma: "Fuente sintética", Referencia: "prueba", PublicadaEn: publicada, Sintetica: true}
	centro, comunidad, municipio := "centro:demo", "comunidad:demo", "municipio:demo"
	versiones := []caldomain.VersionCalendario{
		{ID: "version:nacional", Ambito: caldomain.Ambito{Tipo: caldomain.AmbitoNacional, Ref: caldomain.ReferenciaNacional}, Anio: 2026, Numero: 1, Denominacion: "Nacional", Procedencia: procedencia, ConocidoDesde: ahora.Add(-time.Hour)},
		{ID: "version:autonomica", Ambito: caldomain.Ambito{Tipo: caldomain.AmbitoAutonomico, Ref: comunidad}, Anio: 2026, Numero: 1, Denominacion: "Autonómica", Procedencia: procedencia, ConocidoDesde: ahora.Add(-time.Hour)},
		{ID: "version:local", Ambito: caldomain.Ambito{Tipo: caldomain.AmbitoLocal, Ref: municipio}, Anio: 2026, Numero: 1, Denominacion: "Local", Procedencia: procedencia, ConocidoDesde: ahora.Add(-time.Hour), ComunidadRef: comunidad},
		{ID: "version:centro", Ambito: caldomain.Ambito{Tipo: caldomain.AmbitoCentro, Ref: centro}, Anio: 2026, Numero: 1, Denominacion: "Centro", Procedencia: procedencia, ConocidoDesde: ahora.Add(-time.Hour), ComunidadRef: comunidad, MunicipioRef: municipio},
	}
	return calports.CalendarioCentro{CentroRef: centro, Anio: 2026, Zona: caldomain.ZonaOficial,
		ComunidadRef: comunidad, MunicipioRef: municipio,
		Versiones: versiones, Dias: []caldomain.Clasificacion{{Fecha: f, Laborable: true, Motivos: []caldomain.Motivo{}}},
	}
}

func TestCalendarioC4SoloConsumeCentroAcreditadoYNoDeduceJornada(t *testing.T) {
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	p := &centroC4Prueba{resultado: resultadoCentroC4Prueba(ahora)}
	o := &organizacionC4Prueba{resultado: organizacionPublicadaC4Prueba()}
	c := &calendarioC4Prueba{resultado: calendarioPublicadoC4Prueba(ahora)}
	s, err := NuevoServicioCalendarioLaboralC4(p, o, c, relojMarcajePrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.ConsultarFechaPropia(context.Background(), ConsultaCalendarioLaboralC4{Actor: actor, Fecha: "2026-06-15"})
	if err != nil || r.Estado != EstadoCalendarioC4Publicado || r.CentroRef != "centro:demo" ||
		r.Dia == nil || !r.Dia.Laborable || r.MinutosProgramados != nil || r.TurnoRef != "" ||
		len(r.VersionesCalendario) != 4 || p.llamadas != 1 || o.llamadas != 1 || c.llamadas != 1 {
		t.Fatalf("resultado=%+v error=%v llamadas=%d/%d/%d", r, err, p.llamadas, o.llamadas, c.llamadas)
	}
	p.resultado = personalports.ResultadoUnidadTrabajoPropiaCronos{
		Cobertura: "sin_datos", Evidencia: resultadoCentroC4Prueba(ahora).Evidencia,
	}
	r, err = s.ConsultarFechaPropia(context.Background(), ConsultaCalendarioLaboralC4{Actor: actor, Fecha: "2026-06-15"})
	if err != nil || r.Estado != EstadoCalendarioC4NoDeterminable || c.llamadas != 1 {
		t.Fatalf("sin centro=%+v error=%v consultas calendario=%d", r, err, c.llamadas)
	}
	p.resultado = resultadoCentroC4Prueba(ahora)
	p.resultado.Unidad.EmpleadoRef = "emp_ajenoajenoajenoajenoajeno"
	if _, err = s.ConsultarFechaPropia(context.Background(), ConsultaCalendarioLaboralC4{Actor: actor, Fecha: "2026-06-15"}); !errors.Is(err, ErrCalendarioLaboralC4NoDisponible) || c.llamadas != 1 {
		t.Fatalf("centro ajeno aceptado: %v", err)
	}
}

func TestCalendarioC4AceptaReciboConfirmadoDespuesDeIniciarConsulta(t *testing.T) {
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	p := &centroC4Prueba{resultado: resultadoCentroC4Prueba(ahora)}
	p.resultado.Evidencia.ConsultadaEn = ahora.Add(500 * time.Millisecond)
	o := &organizacionC4Prueba{resultado: organizacionPublicadaC4Prueba()}
	c := &calendarioC4Prueba{resultado: calendarioPublicadoC4Prueba(ahora)}
	reloj := &relojSecuenciaC4{ahora: ahora}
	s, err := NuevoServicioCalendarioLaboralC4(p, o, c, reloj)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.ConsultarFechaPropia(context.Background(), ConsultaCalendarioLaboralC4{Actor: actor, Fecha: "2026-06-15"})
	if err != nil || r.Estado != EstadoCalendarioC4Publicado || reloj.llamadas < 2 {
		t.Fatalf("recibo posterior al inicio: %+v, %v", r, err)
	}
}

func TestCalendarioC4TurnoNocturnoAcreditadoYCentroDivergente(t *testing.T) {
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	programacion := &programacionC4Prueba{resultado: &ProgramacionLaboralC4{
		Fecha: "2026-06-15", EmpleadoRef: "emp_0123456789abcdefghijkl", RelacionRef: "rel_0123456789abcdefghijkl",
		CentroRef: "centro:demo", RegimenRef: "regimen:demo", ProgramacionRef: "programacion:cronos:noche1",
		TurnoRef: "turno_noche", PoliticaVersionRef: "politica:noche:v1", FuenteRef: "fuente:cronos:demo", Version: 1, FuenteVersion: 1,
		InicioUTC: time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC), FinUTC: time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC),
		PublicadaEn: ahora.Add(-time.Hour), MinutosProgramados: 480,
	}}
	s, err := NuevoServicioCalendarioLaboralC4ConProgramacion(&centroC4Prueba{resultado: resultadoCentroC4Prueba(ahora)},
		&organizacionC4Prueba{resultado: organizacionPublicadaC4Prueba()}, &calendarioC4Prueba{resultado: calendarioPublicadoC4Prueba(ahora)},
		relojMarcajePrueba{ahora}, programacion)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.ConsultarFechaPropia(context.Background(), ConsultaCalendarioLaboralC4{Actor: actor, Fecha: "2026-06-15"})
	if err != nil || r.Estado != EstadoCalendarioC4Programado || r.MinutosProgramados == nil || *r.MinutosProgramados != 480 ||
		r.InicioUTC == nil || r.FinUTC == nil || r.FinUTC.In(time.FixedZone("Madrid", 2*3600)).Day() != 16 || programacion.llamadas != 1 {
		t.Fatalf("turno nocturno=%+v, error=%v", r, err)
	}
	programacion.resultado.PublicadaEn = ahora
	r, err = s.ConsultarFechaPropia(context.Background(), ConsultaCalendarioLaboralC4{Actor: actor, Fecha: "2026-06-15"})
	if err != nil || r.Estado != EstadoCalendarioC4Publicado || r.MinutosProgramados != nil {
		t.Fatalf("programación posterior al corte: %+v, %v", r, err)
	}
	programacion.resultado.PublicadaEn = ahora.Add(-time.Hour)
	programacion.resultado.CentroRef = "centro:otro"
	r, err = s.ConsultarFechaPropia(context.Background(), ConsultaCalendarioLaboralC4{Actor: actor, Fecha: "2026-06-15"})
	if !errors.Is(err, ErrCalendarioLaboralC4NoDisponible) || r.MinutosProgramados != nil {
		t.Fatalf("centro divergente no rechazado: %+v, %v", r, err)
	}
}
