package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const DuracionMaximaCapacidadConsultaRRHH = 5 * time.Second

// CapacidadConsultaRRHH es una concesión AD-3 breve. Conserva el material
// original para que SQL lo verifique y consuma como autoridad final.
type CapacidadConsultaRRHH struct {
	bloqueoSerializacionConsultaRRHH
	material         MaterialAutorizacionConsultaRRHH
	capacidadHuella  string
	materialHuella   string
	decisionRef      string
	decisionHuella   string
	correlacionRef   string
	motivoHuella     string
	autenticacionRef string
	sesionRef        string
	principalRef     string
	perfilRef        string
	organizacionRef  string
	claseAmbito      ClaseAmbitoConsultaRRHH
	ambitoRef        string
	accion           string
	finalidad        string
	expedienteRef    string
	consultaDominio  string
	consultaHuella   string
	validaDesde      time.Time
	validaHasta      time.Time
}

func NuevaCapacidadConsultaCuadroRRHH(
	contexto ContextoConsultaRRHH,
	material MaterialAutorizacionConsultaRRHH,
	solicitud SolicitudCuadroRRHH,
	instante time.Time,
) (CapacidadConsultaRRHH, error) {
	huella, err := huellaSolicitudCuadroRRHH(solicitud)
	if err != nil {
		return CapacidadConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	return nuevaCapacidadConsultaRRHH(
		contexto, material, DominioHuellaConsultaCuadroRRHH, huella,
		AccionConsultarCuadroRRHH, FinalidadConsultarCuadroRRHH, "", instante,
	)
}

func NuevaCapacidadConsultaDetalleRRHH(
	contexto ContextoConsultaRRHH,
	material MaterialAutorizacionConsultaRRHH,
	solicitud SolicitudDetalleRRHH,
	instante time.Time,
) (CapacidadConsultaRRHH, error) {
	huella, err := huellaSolicitudDetalleRRHH(solicitud)
	if err != nil {
		return CapacidadConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	return nuevaCapacidadConsultaRRHH(
		contexto, material, DominioHuellaConsultaDetalleRRHH, huella,
		AccionConsultarDetalleRRHH, FinalidadConsultarDetalleRRHH,
		solicitud.expedienteRef, instante,
	)
}

func nuevaCapacidadConsultaRRHH(
	contexto ContextoConsultaRRHH,
	material MaterialAutorizacionConsultaRRHH,
	consultaDominio, consultaHuella, accion, finalidad, expedienteRef string,
	instante time.Time,
) (CapacidadConsultaRRHH, error) {
	if material.validarPara(contexto, instante) != nil {
		return CapacidadConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	datosSolicitud, err := material.solicitud.Datos()
	datosConfirmacion, errConfirmacion := material.confirmacion.Datos()
	correlacion, errCorrelacion := datosSolicitud.Correlacion.ValorCanonico()
	claseAmbito, ambitoRef, errAmbito := validarRecursoCapacidadConsultaRRHH(
		datosSolicitud.Recurso, contexto, consultaDominio, consultaHuella,
		accion, expedienteRef,
	)
	if err != nil || errConfirmacion != nil || errCorrelacion != nil ||
		errAmbito != nil || datosSolicitud.Accion != accion ||
		datosSolicitud.Finalidad != finalidad {
		return CapacidadConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	capacidad := CapacidadConsultaRRHH{
		material: material, capacidadHuella: material.capacidadHuella,
		materialHuella: material.materialHuella,
		decisionRef:    datosConfirmacion.DecisionRef,
		decisionHuella: material.decisionHuella, correlacionRef: correlacion,
		motivoHuella:     material.motivoHuella,
		autenticacionRef: contexto.autenticacionRef, sesionRef: contexto.sesionRef,
		principalRef: contexto.actorRef, perfilRef: contexto.perfilRef,
		organizacionRef: contexto.organizacionRef,
		claseAmbito:     claseAmbito, ambitoRef: ambitoRef,
		accion: accion, finalidad: finalidad, expedienteRef: expedienteRef,
		consultaDominio: consultaDominio, consultaHuella: consultaHuella,
		validaDesde: material.emitidaEn, validaHasta: material.expiraEn,
	}
	if capacidad.validarEstructura() != nil {
		return CapacidadConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	return capacidad, nil
}

func validarRecursoCapacidadConsultaRRHH(
	recurso dominiovec.RecursoAutorizable,
	contexto ContextoConsultaRRHH,
	dominio, huella, accion, expedienteRef string,
) (ClaseAmbitoConsultaRRHH, string, error) {
	if recurso.Validar() != nil || recurso.ModuloID != ModuloContratacion ||
		len(recurso.Ambitos) != 3 || len(recurso.Atributos) != 2 ||
		recurso.Ambitos[ambitoOrganizacionRecursoRRHH] != contexto.organizacionRef ||
		recurso.Atributos[atributoDominioConsultaRRHH] != dominio ||
		recurso.Atributos[atributoHuellaConsultaRRHH] != huella ||
		!patronHuellaRRHH.MatchString(huella) {
		return "", "", ErrCapacidadConsultaRRHHInvalida
	}
	clase := ClaseAmbitoConsultaRRHH(recurso.Ambitos[ambitoClaseRecursoRRHH])
	ambitoRef := recurso.Ambitos[ambitoReferenciaRecursoRRHH]
	if !clase.valida() || !domain.ReferenciaOpacaValida(ambitoRef) ||
		(clase == AmbitoOrganizacionRRHH && ambitoRef != contexto.organizacionRef) {
		return "", "", ErrCapacidadConsultaRRHHInvalida
	}
	esCuadro := accion == AccionConsultarCuadroRRHH &&
		expedienteRef == "" && recurso.Tipo == TipoRecursoCuadroRRHH &&
		recurso.Referencia == ambitoRef
	esDetalle := accion == AccionConsultarDetalleRRHH &&
		domain.ReferenciaOpacaValida(expedienteRef) &&
		recurso.Tipo == TipoRecursoExpediente &&
		recurso.Referencia == expedienteRef
	if !esCuadro && !esDetalle {
		return "", "", ErrCapacidadConsultaRRHHInvalida
	}
	return clase, ambitoRef, nil
}

func (c CapacidadConsultaRRHH) validarEstructura() error {
	esCuadro := c.accion == AccionConsultarCuadroRRHH &&
		c.finalidad == FinalidadConsultarCuadroRRHH && c.expedienteRef == "" &&
		c.consultaDominio == DominioHuellaConsultaCuadroRRHH
	esDetalle := c.accion == AccionConsultarDetalleRRHH &&
		c.finalidad == FinalidadConsultarDetalleRRHH &&
		domain.ReferenciaOpacaValida(c.expedienteRef) &&
		c.consultaDominio == DominioHuellaConsultaDetalleRRHH
	if !patronHuellaRRHH.MatchString(c.capacidadHuella) ||
		!patronHuellaRRHH.MatchString(c.materialHuella) ||
		!domain.ReferenciaOpacaValida(c.decisionRef) ||
		!patronHuellaRRHH.MatchString(c.decisionHuella) ||
		!domain.ReferenciaOpacaValida(c.correlacionRef) ||
		!patronHuellaRRHH.MatchString(c.motivoHuella) ||
		!domain.ReferenciaOpacaValida(c.autenticacionRef) ||
		!domain.ReferenciaOpacaValida(c.sesionRef) ||
		!domain.ReferenciaOpacaValida(c.principalRef) ||
		!domain.ReferenciaOpacaValida(c.perfilRef) ||
		!domain.ReferenciaOpacaValida(c.organizacionRef) ||
		!c.claseAmbito.valida() || !domain.ReferenciaOpacaValida(c.ambitoRef) ||
		(c.claseAmbito == AmbitoOrganizacionRRHH &&
			c.ambitoRef != c.organizacionRef) ||
		(!esCuadro && !esDetalle) ||
		!patronHuellaRRHH.MatchString(c.consultaHuella) ||
		!domain.InstanteUTCCanonico(c.validaDesde) ||
		!domain.InstanteUTCCanonico(c.validaHasta) ||
		!c.validaHasta.After(c.validaDesde) ||
		c.validaHasta.Sub(c.validaDesde) > DuracionMaximaCapacidadConsultaRRHH {
		return ErrCapacidadConsultaRRHHInvalida
	}
	return nil
}

func (c CapacidadConsultaRRHH) validaPara(
	contexto ContextoConsultaRRHH,
	dominio, huella, accion, finalidad, expedienteRef string,
	instante time.Time,
) error {
	if c.validarEstructura() != nil || contexto.validarEn(instante) != nil ||
		c.material.validarPara(contexto, instante) != nil ||
		c.capacidadHuella != c.material.capacidadHuella ||
		c.materialHuella != c.material.materialHuella ||
		c.autenticacionRef != contexto.autenticacionRef ||
		c.sesionRef != contexto.sesionRef || c.principalRef != contexto.actorRef ||
		c.perfilRef != contexto.perfilRef ||
		c.organizacionRef != contexto.organizacionRef ||
		c.consultaDominio != dominio || c.consultaHuella != huella ||
		c.accion != accion || c.finalidad != finalidad ||
		c.expedienteRef != expedienteRef ||
		instante.Before(c.validaDesde) || !instante.Before(c.validaHasta) {
		return ErrCapacidadConsultaRRHHInvalida
	}
	return nil
}

func (c CapacidadConsultaRRHH) DecisionRef() string { return c.decisionRef }
func (c CapacidadConsultaRRHH) CapacidadHuellaSHA256() string {
	return c.capacidadHuella
}
func (c CapacidadConsultaRRHH) MaterialHuellaSHA256() string {
	return c.materialHuella
}
func (c CapacidadConsultaRRHH) DecisionHuellaSHA256() string {
	return c.decisionHuella
}
func (c CapacidadConsultaRRHH) CorrelacionRef() string { return c.correlacionRef }
func (c CapacidadConsultaRRHH) MotivoHuellaSHA256() string {
	return c.motivoHuella
}
func (c CapacidadConsultaRRHH) OrganizacionRef() string { return c.organizacionRef }
func (c CapacidadConsultaRRHH) ClaseAmbito() ClaseAmbitoConsultaRRHH {
	return c.claseAmbito
}
func (c CapacidadConsultaRRHH) AmbitoRef() string       { return c.ambitoRef }
func (c CapacidadConsultaRRHH) Accion() string          { return c.accion }
func (c CapacidadConsultaRRHH) Finalidad() string       { return c.finalidad }
func (c CapacidadConsultaRRHH) ExpedienteRef() string   { return c.expedienteRef }
func (c CapacidadConsultaRRHH) ConsultaDominio() string { return c.consultaDominio }
func (c CapacidadConsultaRRHH) ConsultaHuellaSHA256() string {
	return c.consultaHuella
}
func (c CapacidadConsultaRRHH) ValidaDesde() time.Time { return c.validaDesde }
func (c CapacidadConsultaRRHH) ValidaHasta() time.Time { return c.validaHasta }

type AutoridadContextoConsultaRRHH interface {
	ResolverContextoConsultaRRHH(context.Context) (ContextoConsultaRRHH, error)
}

// SesionConsultaRRHH mantiene consumo AD-3, lectura y registro durable de
// acceso en la misma transacción. El resumen Go nunca autoriza la consulta.
type SesionConsultaRRHH interface {
	ConsultarCuadroYRegistrar(
		context.Context,
		OrdenConsultaCuadroRRHH,
	) (PaginaCuadroRRHH, error)
	ConsultarDetalleYRegistrar(
		context.Context,
		OrdenConsultaDetalleRRHH,
	) (DetalleExpedienteRRHH, error)
}
