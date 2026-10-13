package bootstrap

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	meritospg "vec-diputacion-granada/internal/modules/meritos/adapters/postgres"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vec "vec-diputacion-granada/internal/vec/domain"
)

// NuevoHandlerRevisionMeritosRRHH reutiliza la sesión, decisión actual y
// servicio de detalle RRHH ya compuestos por Bolsa. El pool es el lector RRHH
// existente; sus únicas nuevas concesiones SQL son B111 y RUM6.
func NuevoHandlerRevisionMeritosRRHH(p httpinscripcion.Preparador,
	s *inscripcion.Servicio, pool *pgxpool.Pool) (http.Handler, error) {
	if nuloInscripcionBolsa(p) || s == nil || pool == nil {
		return nil, ports.ErrRevisionRRHHNoDisponible
	}
	lector, err := meritospg.NuevoRevisionRRHH(pool)
	if err != nil {
		return nil, err
	}
	return meritoshttp.NuevoHandlerRevisionRRHH(
		&preparadorRevisionMeritosRRHH{bolsa: p, servicio: s}, lector)
}

type preparadorRevisionMeritosRRHH struct {
	bolsa    httpinscripcion.Preparador
	servicio *inscripcion.Servicio
}

var _ meritoshttp.PreparadorRevisionRRHH = (*preparadorRevisionMeritosRRHH)(nil)

func (p *preparadorRevisionMeritosRRHH) PrepararRevisionRRHH(
	r *http.Request, solicitudRef string) (ports.SolicitudRevisionRRHH, error) {
	var vacia ports.SolicitudRevisionRRHH
	if p == nil || nuloInscripcionBolsa(p.bolsa) || p.servicio == nil ||
		r == nil || r.URL == nil || r.Method != http.MethodGet ||
		r.URL.Path != meritoshttp.RutaRevisionRRHH || r.URL.EscapedPath() != meritoshttp.RutaRevisionRRHH ||
		r.Context().Err() != nil {
		return vacia, ports.ErrRevisionRRHHNoDisponible
	}
	// Solo se traduce una ruta de consulta exacta ya admitida por el handler.
	// La identidad TLS y el contexto siguen siendo los de la petición original.
	canonica := r.Clone(r.Context())
	urlCopia := *r.URL
	urlCopia.Path = httpinscripcion.RutaRRHH + "/" + solicitudRef
	urlCopia.RawPath = ""
	urlCopia.RawQuery = ""
	urlCopia.ForceQuery = false
	canonica.URL = &urlCopia
	canonica.RequestURI = urlCopia.Path
	actor, err := p.bolsa.PrepararLecturaRRHH(canonica,
		inscripcion.AccionDetalleRRHH, solicitudRef, inscripcion.Filtro{}, "es")
	if err != nil {
		return vacia, ports.ErrRevisionRRHHDenegada
	}
	detalle, err := p.servicio.DetalleRRHH(r.Context(), actor, solicitudRef)
	if err != nil {
		return vacia, ports.ErrRevisionRRHHDenegada
	}
	if detalle.SolicitudRef != solicitudRef || detalle.PersonaRef == "" || actor.Lectura == nil {
		return vacia, ports.ErrRevisionRRHHNoDisponible
	}
	captura, err := postgresbolsa.CapturaDetalleRRHHMeritos(actor, solicitudRef)
	if err != nil {
		return vacia, ports.ErrRevisionRRHHNoDisponible
	}
	correlacion, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(
		r.Context(), seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, ports.ErrRevisionRRHHNoDisponible
	}
	return ports.SolicitudRevisionRRHH{
		Contexto: actor.ResultadoContexto, Vinculo: actor.Vinculo,
		Correlacion: correlacion, SolicitudRef: solicitudRef,
		PersonaRef:              detalle.PersonaRef,
		RevisionPermisos:        actor.Lectura.RevisionPermisos,
		HuellaInstantaneaSHA256: actor.Lectura.HuellaInstantaneaSHA256,
		CapturaDetalleBolsa:     captura,
	}, nil
}
