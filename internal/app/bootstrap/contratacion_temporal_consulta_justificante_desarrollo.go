package bootstrap

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type claveConsultaJustificanteRespuestaDesarrollo struct{}

// La consulta acredita el registro original, nunca la validez de su respuesta
// ni el plazo. La selección recuperada se conserva dentro de la composición.
type proveedorConsultaJustificanteRespuestaDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizacionComunicacionLlamamientoDesarrollo
	reloj       ports.Reloj
}

func motivoConsultaJustificanteRespuestaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_consulta_justificante_rrhh", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("consulta-justificante-rrhh-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "consulta-justificante-rrhh"),
	}
}

func consultaJustificanteLigadaAlExpedienteDesarrollo(e ports.ExpedienteParaSeleccion, s ports.SolicitudResolverLlamamiento) bool {
	return s.Validar() == nil && s.VersionEsperada == 2 &&
		(s.Respuesta == ports.RespuestaLlamamientoAceptada || s.Respuesta == ports.RespuestaLlamamientoRenunciada) &&
		e.Fiscalizado.Validar() == nil &&
		expedienteComunicacionLlamamientoDesarrolloValido(e, ports.SolicitudRegistrarComunicacionLlamamiento{
			OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		})
}

func (p *proveedorConsultaJustificanteRespuestaDesarrollo) AutorizarConsultaJustificanteRespuestaRecibida(ctx context.Context, s ports.SolicitudResolverLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || contextoInterfazNulo(ctx) || p.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizador) || dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	c, valida := p.soporte.capacidadValida(ctx)
	ligada, existe := ctx.Value(claveConsultaJustificanteRespuestaDesarrollo{}).(ports.SolicitudResolverLlamamiento)
	preparacion, preparada := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !valida || c.ruta != httpinterno.RutaResolucionComunicacionLlamamiento || !existe || ligada != s ||
		!preparada || !consultaJustificanteLigadaAlExpedienteDesarrollo(preparacion.expediente, s) {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	recurso, err := postgresct.RecursoConsultaJustificanteRespuestaRecibida(s)
	if err != nil {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	a, err := p.autorizador.AutorizarOperacion(ctx, postgresct.AccionConsultaJustificanteRespuestaRecibida, recurso)
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
