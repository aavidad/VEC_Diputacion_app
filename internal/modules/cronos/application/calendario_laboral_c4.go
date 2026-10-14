package application

import (
	"context"
	"errors"
	"time"

	caldomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calports "vec-diputacion-granada/internal/modules/calendarios/ports"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrCalendarioLaboralC4NoDisponible = errors.New("cronos: calendario laboral no disponible")
var ErrCalendarioLaboralC4Invalido = errors.New("cronos: consulta de calendario inválida")

const (
	EstadoCalendarioC4NoDeterminable = "no_determinable"
	EstadoCalendarioC4Publicado      = "calendario_publicado_jornada_no_determinable"
)

type ConsultaCalendarioLaboralC4 struct {
	Actor vecdomain.ContextoActor
	Fecha string
}

// El día de Calendarios sí tiene fuente y versiones. Minutos y turno quedan
// indeterminados hasta recibir una programación laboral publicada.
type ResultadoCalendarioLaboralC4 struct {
	Fecha                 string                        `json:"fecha"`
	Estado                string                        `json:"estado"`
	CentroRef             string                        `json:"centro_ref,omitempty"`
	FuenteUnidadRef       string                        `json:"fuente_unidad_ref,omitempty"`
	FuenteUnidadVersion   string                        `json:"fuente_unidad_version,omitempty"`
	FuenteOrganizacionRef string                        `json:"fuente_organizacion_ref,omitempty"`
	VersionOrganizacion   int                           `json:"version_organizacion,omitempty"`
	VersionesCalendario   []caldomain.VersionCalendario `json:"versiones_calendario"`
	Dia                   *caldomain.Clasificacion      `json:"dia,omitempty"`
	MinutosProgramados    *int64                        `json:"minutos_programados"`
	TurnoRef              string                        `json:"turno_ref,omitempty"`
}

type ServicioCalendarioLaboralC4 struct {
	personal     personalports.LectorUnidadTrabajoPropiaCronos
	organizacion consultaOrganizacionHistoricaC4
	calendarios  calports.ConsultaCalendarios
	reloj        cronosports.Reloj
}

type consultaOrganizacionHistoricaC4 interface {
	Consultar(context.Context, personaldomain.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error)
}

func NuevoServicioCalendarioLaboralC4(p personalports.LectorUnidadTrabajoPropiaCronos, o consultaOrganizacionHistoricaC4, c calports.ConsultaCalendarios, reloj cronosports.Reloj) (*ServicioCalendarioLaboralC4, error) {
	if dependenciaRemotaNula(p) || dependenciaRemotaNula(o) || dependenciaRemotaNula(c) || dependenciaRemotaNula(reloj) {
		return nil, ErrCalendarioLaboralC4NoDisponible
	}
	return &ServicioCalendarioLaboralC4{p, o, c, reloj}, nil
}

func (s *ServicioCalendarioLaboralC4) ConsultarFechaPropia(ctx context.Context, q ConsultaCalendarioLaboralC4) (ResultadoCalendarioLaboralC4, error) {
	if s == nil || ctx == nil || dependenciaRemotaNula(s.personal) || dependenciaRemotaNula(s.organizacion) || dependenciaRemotaNula(s.calendarios) || dependenciaRemotaNula(s.reloj) {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ResultadoCalendarioLaboralC4{}, err
	}
	fecha, err := personaldomain.NuevaFechaCivil(q.Fecha)
	if err != nil || q.Actor.Validar() != nil {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4Invalido
	}
	actor, err := q.Actor.Clonar()
	if err != nil {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4Invalido
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4Invalido
	}
	ahora := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4Invalido
	}
	conocido := ahora.Add(-5 * time.Second)
	proyeccion, err := s.personal.ConsultarUnidadTrabajoPropiaCronos(ctx, personalports.ConsultaUnidadTrabajoPropiaCronos{
		Actor: actor, Fecha: fecha, ConocidoEn: conocido,
	})
	if err != nil {
		return ResultadoCalendarioLaboralC4{}, err
	}
	r := ResultadoCalendarioLaboralC4{Fecha: q.Fecha, Estado: EstadoCalendarioC4NoDeterminable, VersionesCalendario: []caldomain.VersionCalendario{}}
	despuesLectura := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if proyeccion.Evidencia.ReciboRef == "" || proyeccion.Evidencia.DecisionRef == "" ||
		proyeccion.Evidencia.AuditoriaRef == "" || proyeccion.Evidencia.ConsultadaEn.IsZero() ||
		proyeccion.Evidencia.ConsultadaEn.Location() != time.UTC ||
		proyeccion.Evidencia.ConsultadaEn.Before(conocido) ||
		proyeccion.Evidencia.ConsultadaEn.After(despuesLectura.Add(5*time.Second)) ||
		!actor.Instantanea.VigenteEn(despuesLectura) {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
	}
	if proyeccion.Cobertura != "completa" || proyeccion.Unidad == nil {
		if proyeccion.Cobertura != "sin_datos" && proyeccion.Cobertura != "no_acreditada" {
			return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
		}
		return r, nil
	}
	unidad := proyeccion.Unidad
	if unidad.EmpleadoRef != empleados[0] || unidad.RelacionRef == "" || unidad.OrganismoRef == "" ||
		unidad.UnidadClave == "" || unidad.FuenteRef == "" || unidad.FuenteVersionRef == "" || unidad.Version < 1 ||
		unidad.Desde.Validar() != nil || fecha.AntesDe(unidad.Desde) ||
		(unidad.HastaExclusivo != "" && (unidad.HastaExclusivo.Validar() != nil || !fecha.AntesDe(unidad.HastaExclusivo))) {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
	}
	organizacion, err := s.organizacion.Consultar(ctx, personaldomain.SolicitudConsultaOrganizacionHistorica{
		Actor: actor, Selector: personaldomain.SelectorOrganizacionHistorica{
			OrganismoRef: unidad.OrganismoRef, VigenteEn: fecha, ConocidoEn: conocido, Limite: 100,
		},
	})
	if err != nil {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
	}
	if organizacion.Pagina.Cobertura.Unidades != "completa" || organizacion.Pagina.CursorSiguiente != "" ||
		len(organizacion.Pagina.Unidades) == 0 || len(organizacion.Pagina.Unidades) > 100 ||
		organizacion.Evidencia.ReciboRef == "" || organizacion.Evidencia.AuditoriaRef == "" {
		return r, nil
	}
	centro, ok := centroDeUnidadHistoricaC4(unidad.UnidadClave, organizacion.Pagina.Unidades)
	if !ok {
		return r, nil
	}
	centroRef := centro.ClaveCatalogo
	anio := int(fecha[0]-'0')*1000 + int(fecha[1]-'0')*100 + int(fecha[2]-'0')*10 + int(fecha[3]-'0')
	calendario, err := s.calendarios.CalendarioCentro(ctx, calports.SolicitudCalendarioCentro{
		CentroRef: centroRef, Anio: anio, ConocidoEn: conocido,
	})
	if err != nil {
		var cobertura *caldomain.ErrorCobertura
		if errors.As(err, &cobertura) {
			return r, nil
		}
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
	}
	if calendario.CentroRef != centroRef || calendario.Anio != anio || len(calendario.Versiones) != 4 ||
		!calendario.ConocidoEn.Equal(conocido) || calendario.Zona != caldomain.ZonaOficial {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
	}
	esperados := map[caldomain.Ambito]bool{
		{Tipo: caldomain.AmbitoNacional, Ref: caldomain.ReferenciaNacional}: true,
		{Tipo: caldomain.AmbitoAutonomico, Ref: calendario.ComunidadRef}:    true,
		{Tipo: caldomain.AmbitoLocal, Ref: calendario.MunicipioRef}:         true,
		{Tipo: caldomain.AmbitoCentro, Ref: centroRef}:                      true,
	}
	for _, v := range calendario.Versiones {
		if v.Validar() != nil || v.ConocidoDesde.After(conocido) || !esperados[v.Ambito] {
			return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
		}
		delete(esperados, v.Ambito)
	}
	dia, err := caldomain.NuevaFechaCivil(anio, int(fecha[5]-'0')*10+int(fecha[6]-'0'), int(fecha[8]-'0')*10+int(fecha[9]-'0'))
	if err != nil {
		return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4Invalido
	}
	for i := range calendario.Dias {
		if calendario.Dias[i].Fecha.Igual(dia) {
			d := calendario.Dias[i]
			r.Estado, r.CentroRef = EstadoCalendarioC4Publicado, centroRef
			r.FuenteUnidadRef, r.FuenteUnidadVersion = unidad.FuenteRef, unidad.FuenteVersionRef
			r.FuenteOrganizacionRef, r.VersionOrganizacion = centro.Traza.FuenteRef, centro.CatalogoVersion
			r.VersionesCalendario, r.Dia = calendario.Versiones, &d
			return r, nil
		}
	}
	return ResultadoCalendarioLaboralC4{}, ErrCalendarioLaboralC4NoDisponible
}

// La estructura histórica ya seleccionó la fecha de efectos y el corte de
// conocimiento. Se exige un camino único unidad → centro, sin elegir un
// antecesor arbitrario ante duplicados, ciclos o enlaces perdidos.
func centroDeUnidadHistoricaC4(clave string, unidades []personaldomain.UnidadOrganizacionHistorica) (personaldomain.UnidadOrganizacionHistorica, bool) {
	porClave := make(map[string]personaldomain.UnidadOrganizacionHistorica, len(unidades))
	porID := make(map[string]personaldomain.UnidadOrganizacionHistorica, len(unidades))
	for _, u := range unidades {
		if u.ClaveCatalogo == "" || u.Traza.ID == "" || porClave[u.ClaveCatalogo].Traza.ID != "" || porID[u.Traza.ID].Traza.ID != "" {
			return personaldomain.UnidadOrganizacionHistorica{}, false
		}
		porClave[u.ClaveCatalogo], porID[u.Traza.ID] = u, u
	}
	u, ok := porClave[clave]
	if !ok {
		return personaldomain.UnidadOrganizacionHistorica{}, false
	}
	vistos := map[string]bool{}
	for i := 0; i < len(unidades) && !vistos[u.Traza.ID]; i++ {
		vistos[u.Traza.ID] = true
		if u.Tipo == "centro" {
			return u, caldomain.ReferenciaValida(u.ClaveCatalogo)
		}
		if u.PadreID == "" {
			return personaldomain.UnidadOrganizacionHistorica{}, false
		}
		u, ok = porID[u.PadreID]
		if !ok {
			return personaldomain.UnidadOrganizacionHistorica{}, false
		}
	}
	return personaldomain.UnidadOrganizacionHistorica{}, false
}
