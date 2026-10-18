package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// La lectura histórica conserva la preimagen de la clave original durante un
// replay, aunque la cabeza actual del expediente padre haya avanzado.
type preparadorDerivacionNombramientoDesarrollo struct {
	soporte *soporteAltaContratacionTemporalDesarrollo
	lector  ports.LectorExpedienteSeleccionLlamamiento
}

func (p preparadorDerivacionNombramientoDesarrollo) PrepararDerivacionNombramiento(ctx context.Context,
	e ports.EntradaDerivacionNombramiento,
) (ports.SolicitudDerivacionNombramiento, error) {
	vacio := ports.SolicitudDerivacionNombramiento{}
	if ctx == nil || p.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.lector) || e.Validar() != nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	c, valida := p.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaDerivacionNombramiento || c.metodo != http.MethodPost {
		return vacio, ports.ErrDerivacionNombramientoDenegada
	}
	padre, err := p.lector.LeerExpedienteParaSeleccion(ctx,
		organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.VersionPadre)
	if err != nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	if padre.Fiscalizado.Asignacion == nil || padre.Fiscalizado.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		padre.Fiscalizado.Referencia != e.PadreRef || padre.Fiscalizado.Version != e.VersionPadre {
		return vacio, ports.ErrDerivacionNombramientoNoConfiable
	}
	canon, err := json.Marshal(padre.Fiscalizado)
	if err != nil {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	s := ports.SolicitudDerivacionNombramiento{Esquema: ports.EsquemaSolicitudDerivacionNombramiento,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, PadreRef: e.PadreRef,
		VersionPadre: e.VersionPadre, HijoRef: ports.HijoRefDerivacionNombramiento(
			organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.NumeroPlaza),
		OfertaRef: e.OfertaRef, NumeroPlaza: e.NumeroPlaza, SecuenciaPlaza: e.SecuenciaPlaza,
		ParticipacionRef: e.ParticipacionRef, ReciboAdjudicacionRef: e.ReciboAdjudicacionRef,
		ClaveIdempotencia: e.ClaveIdempotencia, UnidadRef: padre.Fiscalizado.Asignacion.UnidadRef,
		PadreCanonico: string(canon)}
	if _, err := s.Canon(); err != nil {
		return vacio, err
	}
	return s, nil
}

// La autoridad se inyecta desde composición. Ausente, solo la derivación
// queda cerrada; nunca se toma el permiso de consulta del padre.
type autoridadDerivacionNoDisponible struct{}

func (autoridadDerivacionNoDisponible) AutorizarDerivacionNombramiento(context.Context,
	ports.SolicitudDerivacionNombramiento) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDerivacionNombramientoNoDisponible
}

func nuevaRutaDerivacionNombramientoDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo,
	autoridad ports.AutorizadorDerivacionNombramiento) (http.Handler, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil {
		return httpinterno.NuevoManejadorDerivacionNombramiento(derivacionNombramientoNoCompuesta{})
	}
	lector, err := postgresct.NuevoLectorExpedienteSeleccionLlamamientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	registro, err := postgresct.NuevoRegistroDerivacionNombramientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	if dependenciaEsNulaContratacionTemporalDesarrollo(autoridad) {
		autoridad = autoridadDerivacionNoDisponible{}
	}
	servicio, err := application.NuevoServicioDerivacionNombramiento(
		preparadorDerivacionNombramientoDesarrollo{soporte: alta.soporte, lector: lector}, autoridad, registro)
	if err != nil {
		return nil, err
	}
	return httpinterno.NuevoManejadorDerivacionNombramiento(servicio)
}

type derivacionNombramientoNoCompuesta struct{}

func (derivacionNombramientoNoCompuesta) Derivar(context.Context,
	ports.EntradaDerivacionNombramiento) (ports.ResultadoDerivacionNombramiento, error) {
	return ports.ResultadoDerivacionNombramiento{}, ports.ErrDerivacionNombramientoNoDisponible
}
