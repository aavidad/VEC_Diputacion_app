package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type claveConsultaComunicacionesExpedienteDesarrollo struct{}

type solicitudLigadaComunicacionesExpedienteDesarrollo struct {
	consulta        ports.ConsultaComunicacionesExpediente
	organizacionRef string
}

type proveedorConsultaComunicacionesExpedienteDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizacionComunicacionLlamamientoDesarrollo
	reloj       ports.Reloj
}

func motivoConsultaComunicacionesExpedienteDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_consulta_comunicaciones_expediente_rrhh", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("consulta-comunicaciones-expediente-rrhh-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "consulta-comunicaciones-expediente-rrhh"),
	}
}

func concesionConsultaComunicacionesExpedienteDesarrollo() dominiovec.ConcesionRol {
	return dominiovec.ConcesionRol{
		Accion: postgresct.AccionConsultaComunicacionesExpediente, ModuloID: "contratacion_temporal",
		TipoRecurso: postgresct.TipoRecursoConsultaComunicacionesExpediente,
		Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		CamposPermitidos: []string{
			"antecedente_tipo", "comunicacion_ref", "estado", "estado_respuesta", "expediente_ref", "llamamiento_ref",
			"organizacion_ref", "recibo_antecedente_ref", "recibo_comunicacion_ref", "registrada_en", "version",
		},
	}
}

func descriptorMaterialConsultaComunicacionesExpedienteDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        postgresct.AudienciaConsultaComunicacionesExpediente,
		Dominio:          "vec.ct.desarrollo.consulta-comunicaciones-expediente.capacidad-v3",
		Prefijo:          "clave:capacidad:ct:comunicaciones-expediente:",
		ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

func nuevoManejadorConsultaComunicacionesExpedienteDesarrollo(
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (http.Handler, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.proveedorMaterial == nil || derivador == nil || !derivador.valido() {
		return nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	proveedorMaterial, err := nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(
		ctx, alta.postgresql.gobierno, material, alta.soporte, reloj,
		descriptorMaterialConsultaComunicacionesExpedienteDesarrollo())
	if err != nil {
		return nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	proveedor := &proveedorConsultaComunicacionesExpedienteDesarrollo{
		soporte: alta.soporte, reloj: reloj,
		autorizador: &autorizadorLlamamientoDesarrollo{
			alta: alta, material: proveedorMaterial, consultaComunicacionesExpediente: true,
		},
	}
	lector, err := postgresct.NuevoLectorComunicacionesExpedientePostgreSQL(alta.postgresql.ejecucion, proveedor)
	if err != nil {
		return nil, err
	}
	lectorAuditado, err := nuevoLectorComunicacionesAuditadoCT(lector, alta.soporte, alta.auditoriaLecturasCT,
		configuracionAuditoriaLecturasCT{Proceso: alta.procesoAuditoriaLecturasCT, Canal: string(dominiovec.SuperficieAutenticacionInternaCorporativaV1)})
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioConsultaComunicacionesExpediente(lectorAuditado)
	if err != nil {
		return nil, err
	}
	return httpinterno.NuevoManejadorConsultaComunicacionesExpediente(proveedor, servicio)
}

func (p *proveedorConsultaComunicacionesExpedienteDesarrollo) ResolverContextoConsultaComunicacionesExpediente(ctx context.Context) error {
	if p == nil || contextoInterfazNulo(ctx) || p.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	capacidad, valida := p.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaConsultaComunicacionesExpediente {
		return ports.ErrConsultaComunicacionesExpedienteDenegada
	}
	if !certificadoConsultaReciboRespuestaVigente(capacidad, p.reloj.Ahora()) {
		return httpinterno.ErrContextoCanalCaducado
	}
	if _, err := p.soporte.contextoOperativoDesarrollo(ctx); errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		return ports.ErrConsultaComunicacionesExpedienteNoDisponible
	} else if err != nil {
		return ports.ErrConsultaComunicacionesExpedienteDenegada
	}
	return nil
}

func (p *proveedorConsultaComunicacionesExpedienteDesarrollo) AutorizarConsultaComunicacionesExpediente(
	ctx context.Context, c ports.ConsultaComunicacionesExpediente,
) (postgresct.AmbitoConsultaComunicacionesExpediente, error) {
	vacio := postgresct.AmbitoConsultaComunicacionesExpediente{}
	if p == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizador) || c.Validar() != nil {
		return vacio, ports.ErrConsultaComunicacionesExpedienteDenegada
	}
	if err := p.ResolverContextoConsultaComunicacionesExpediente(ctx); err != nil {
		return vacio, err
	}
	// El ámbito de este perfil de desarrollo se fija en la composición y se
	// usa solo después de revalidar la capacidad mTLS y la sesión operativa.
	organizacion := organizacionAltaContratacionTemporalDesarrollo
	recurso, err := postgresct.RecursoConsultaComunicacionesExpediente(c, organizacion)
	if err != nil {
		return vacio, ports.ErrConsultaComunicacionesExpedienteDenegada
	}
	ctx = context.WithValue(ctx, claveConsultaComunicacionesExpedienteDesarrollo{}, solicitudLigadaComunicacionesExpedienteDesarrollo{c, organizacion})
	a, err := p.autorizador.AutorizarOperacion(ctx, postgresct.AccionConsultaComunicacionesExpediente, recurso)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, errorAutorizacionConsultaComunicacionesExpediente(err)
	}
	if a.ValidarEstructura() != nil {
		return vacio, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	r, ahora := a.ResumenCapacidad(), p.reloj.Ahora()
	if ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) || r.ExpiraEn().Sub(r.EmitidaEn()) > 5*time.Minute ||
		r.AudienciaConsumo() != postgresct.AudienciaConsultaComunicacionesExpediente {
		return vacio, ports.ErrConsultaComunicacionesExpedienteDenegada
	}
	return postgresct.AmbitoConsultaComunicacionesExpediente{OrganizacionRef: organizacion, Autorizacion: a}, nil
}

func errorAutorizacionConsultaComunicacionesExpediente(err error) error {
	for _, dependencia := range []error{ports.ErrPersistenciaNoDisponible, puertosvec.ErrFuenteAutorizacionNoDisponible,
		puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible,
		dominiovec.ErrConfiguracionAccesoInvalida} {
		if errors.Is(err, dependencia) {
			return ports.ErrConsultaComunicacionesExpedienteNoDisponible
		}
	}
	if errors.Is(err, ports.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		return ports.ErrConsultaComunicacionesExpedienteDenegada
	}
	return ports.ErrConsultaComunicacionesExpedienteNoDisponible
}

var _ postgresct.ProveedorConsultaComunicacionesExpediente = (*proveedorConsultaComunicacionesExpedienteDesarrollo)(nil)
var _ httpinterno.AutoridadCanalConsultaComunicacionesExpediente = (*proveedorConsultaComunicacionesExpedienteDesarrollo)(nil)
