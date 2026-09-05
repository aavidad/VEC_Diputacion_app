package bootstrap

import (
	"context"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type claveSolicitudRespuestaRecibidaDesarrollo struct{}

// Una declaración de RRHH no sustituye la autoridad de Bolsa ni convierte el
// registro de aviso local en evidencia de entrega. Solo registra su respuesta.
type ejecutorRespuestaRecibidaDesarrollo struct {
	soporte  *soporteAltaContratacionTemporalDesarrollo
	lector   ports.LectorExpedienteSeleccionLlamamiento
	servicio httpinterno.EjecutorRespuestaRecibida
}

type proveedorRespuestaRecibidaDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizacionComunicacionLlamamientoDesarrollo
	reloj       ports.Reloj
}

func nuevoManejadorRespuestaRecibidaDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo, reloj ports.Reloj) (http.Handler, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil || alta.postgresql.proveedorMaterial == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, application.ErrServicioRespuestasRecibidasInvalido
	}
	lector, err := postgresct.NuevoLectorExpedienteSeleccionLlamamientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	proveedor := &proveedorRespuestaRecibidaDesarrollo{soporte: alta.soporte, reloj: reloj,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: alta.postgresql.proveedorMaterial, respuestaRecibida: true}}
	registro, err := postgresct.NuevoRegistroRespuestasRecibidasPostgreSQL(alta.postgresql.ejecucion, proveedor)
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioRespuestasRecibidas(registro)
	if err != nil {
		return nil, err
	}
	return httpinterno.NuevoManejadorRespuestaRecibida(&ejecutorRespuestaRecibidaDesarrollo{soporte: alta.soporte, lector: lector, servicio: servicio})
}

func (e *ejecutorRespuestaRecibidaDesarrollo) Registrar(ctx context.Context, s ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
	vacio := ports.RespuestaRecibidaRegistrada{}
	if e == nil || contextoInterfazNulo(ctx) || e.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.lector) || dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) {
		return vacio, application.ErrServicioRespuestasRecibidasInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if s.Validar() != nil {
		return vacio, application.ErrSolicitudRespuestaRecibidaInvalida
	}
	c, valida := e.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaRegistroRespuestaRecibida || s.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacio, application.ErrRespuestaRecibidaDenegada
	}
	expediente, err := e.lector.LeerExpedienteParaSeleccion(ctx, s.OrganizacionRef, s.ExpedienteRef, 6)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || !expedienteRespuestaRecibidaDesarrolloValido(expediente, s) {
		return vacio, application.ErrRespuestaRecibidaNoDisponible
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: expediente})
	ctx = context.WithValue(ctx, claveSolicitudRespuestaRecibidaDesarrollo{}, s)
	return e.servicio.Registrar(ctx, s)
}

func expedienteRespuestaRecibidaDesarrolloValido(e ports.ExpedienteParaSeleccion, s ports.SolicitudRegistrarRespuestaRecibida) bool {
	// Reutiliza la precondición de expediente fiscalizado. La relación exacta
	// con la comunicación y su versión se comprueba dentro de CT56 tras AD3.
	return s.Validar() == nil && e.Fiscalizado.Validar() == nil &&
		expedienteComunicacionLlamamientoDesarrolloValido(e, ports.SolicitudRegistrarComunicacionLlamamiento{
			OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		})
}

func (p *proveedorRespuestaRecibidaDesarrollo) AutorizarRegistroRespuestaRecibida(ctx context.Context, s ports.SolicitudRegistrarRespuestaRecibida) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || contextoInterfazNulo(ctx) || p.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizador) || dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	c, valida := p.soporte.capacidadValida(ctx)
	ligada, existe := ctx.Value(claveSolicitudRespuestaRecibidaDesarrollo{}).(ports.SolicitudRegistrarRespuestaRecibida)
	preparacion, preparada := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !valida || c.ruta != httpinterno.RutaRegistroRespuestaRecibida || !existe || ligada != s ||
		!preparada || !expedienteRespuestaRecibidaDesarrolloValido(preparacion.expediente, s) {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	recurso, err := postgresct.RecursoRegistroRespuestaRecibida(s)
	if err != nil {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	a, err := p.autorizador.AutorizarOperacion(ctx, postgresct.AccionRegistroRespuestaRecibida, recurso)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || a.ValidarEstructura() != nil {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	r, ahora := a.ResumenCapacidad(), p.reloj.Ahora()
	if ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) || r.ExpiraEn().Sub(r.EmitidaEn()) > 5*time.Minute {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	return a, nil
}
