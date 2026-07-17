package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	EsquemaPanelInternoBolsaV1 = "vec.bolsa.panel.interno.v1"

	ModuloPanelInternoBolsa      = "bolsa"
	TipoRecursoPanelInternoBolsa = "panel_interno_agregado"
	AccionConsultarPanelInterno  = "bolsa.panel_interno.consultar"
	FinalidadPanelInternoBolsa   = "gestion_operativa_bolsa"
	CampoPanelInternoAgregado    = "panel_agregado_sin_datos_personales"
)

var (
	ErrSelectorPanelInternoInvalido  = errors.New("bolsa: selector de panel interno invalido")
	ErrConsultaPanelInternoInvalida  = errors.New("bolsa: consulta de panel interno invalida")
	ErrResultadoPanelInternoInvalido = errors.New("bolsa: resultado de panel interno invalido")
)

// ClaseAmbitoPanelInterno obliga a elegir un alcance exacto. El valor cero no
// significa toda la organizacion y nunca se interpreta como valor por defecto.
type ClaseAmbitoPanelInterno string

const (
	AmbitoPanelOrganizacion ClaseAmbitoPanelInterno = "organizacion"
	AmbitoPanelUnidad       ClaseAmbitoPanelInterno = "unidad_gestion"
)

// SelectorPanelInterno no contiene identidad de la persona operadora. Sus
// referencias proceden de configuracion interna y quedan ligadas al PDP.
type SelectorPanelInterno struct {
	Clase            ClaseAmbitoPanelInterno `json:"clase"`
	OrganizacionRef  string                  `json:"organizacion_ref"`
	UnidadGestionRef string                  `json:"unidad_gestion_ref,omitempty"`
}

func (s SelectorPanelInterno) Validar() error {
	if !referenciaOpacaPanelValida(s.OrganizacionRef, "org_") {
		return ErrSelectorPanelInternoInvalido
	}
	switch s.Clase {
	case AmbitoPanelOrganizacion:
		if s.UnidadGestionRef != "" {
			return ErrSelectorPanelInternoInvalido
		}
	case AmbitoPanelUnidad:
		if !referenciaOpacaPanelValida(s.UnidadGestionRef, "uni_") {
			return ErrSelectorPanelInternoInvalido
		}
	default:
		return ErrSelectorPanelInternoInvalido
	}
	return nil
}

// RecursoAutorizablePanelInterno liga el alcance exacto y el motivo
// catalogado a la decision V2. No acepta comodines ni ambitos solapados.
func RecursoAutorizablePanelInterno(
	selector SelectorPanelInterno,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) (dominiovec.RecursoAutorizable, error) {
	if selector.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return dominiovec.RecursoAutorizable{}, ErrSelectorPanelInternoInvalido
	}
	referencia := "panel:" + selector.OrganizacionRef
	ambitos := map[string]string{
		"clase":            selectorClaseCanonica(selector.Clase),
		"organizacion_ref": selector.OrganizacionRef,
	}
	if selector.Clase == AmbitoPanelUnidad {
		referencia += ":" + selector.UnidadGestionRef
		ambitos["unidad_gestion_ref"] = selector.UnidadGestionRef
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: referencia,
		ModuloID:   ModuloPanelInternoBolsa,
		Tipo:       TipoRecursoPanelInternoBolsa,
		Ambitos:    ambitos,
		Atributos: map[string]string{
			"motivo_catalogo_id":      motivo.CatalogoID,
			"motivo_catalogo_version": enteroDecimal(motivo.CatalogoVersion),
			"motivo_catalogo_huella":  motivo.CatalogoHuellaSHA256,
			"motivo_entrada_clave":    motivo.EntradaClave,
		},
	}
	if recurso.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrSelectorPanelInternoInvalido
	}
	return recurso, nil
}

// SolicitudConsultaPanelInterno es una capacidad opaca para el adaptador
// durable. Este debe revalidar y consumir la decision V2 en la misma
// transaccion que calcula los agregados y confirma la auditoria de lectura.
type SolicitudConsultaPanelInterno struct {
	datos *datosSolicitudConsultaPanelInterno
}

type datosSolicitudConsultaPanelInterno struct {
	selector     SelectorPanelInterno
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	motivo       dominiovec.ReferenciaEntradaCatalogo
	correlacion  dominiovec.ReferenciaCorrelacionAutorizacionV2
	consultadaEn time.Time
}

func NuevaSolicitudConsultaPanelInterno(
	selector SelectorPanelInterno,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	consultadaEn time.Time,
) (SolicitudConsultaPanelInterno, error) {
	datos := datosSolicitudConsultaPanelInterno{
		selector: selector, autorizacion: autorizacion, motivo: motivo,
		correlacion: correlacion, consultadaEn: consultadaEn,
	}
	if validarSolicitudConsultaPanelInterno(datos) != nil {
		return SolicitudConsultaPanelInterno{}, ErrConsultaPanelInternoInvalida
	}
	return SolicitudConsultaPanelInterno{datos: &datos}, nil
}

func (s SolicitudConsultaPanelInterno) Selector() (SelectorPanelInterno, error) {
	if s.validar() != nil {
		return SelectorPanelInterno{}, ErrConsultaPanelInternoInvalida
	}
	return s.datos.selector, nil
}

func (s SolicitudConsultaPanelInterno) Autorizacion() (
	puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	error,
) {
	if s.validar() != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, ErrConsultaPanelInternoInvalida
	}
	return s.datos.autorizacion, nil
}

func (s SolicitudConsultaPanelInterno) Motivo() (dominiovec.ReferenciaEntradaCatalogo, error) {
	if s.validar() != nil {
		return dominiovec.ReferenciaEntradaCatalogo{}, ErrConsultaPanelInternoInvalida
	}
	return s.datos.motivo, nil
}

func (s SolicitudConsultaPanelInterno) Correlacion() (
	dominiovec.ReferenciaCorrelacionAutorizacionV2,
	error,
) {
	if s.validar() != nil {
		return dominiovec.ReferenciaCorrelacionAutorizacionV2{}, ErrConsultaPanelInternoInvalida
	}
	return s.datos.correlacion, nil
}

func (s SolicitudConsultaPanelInterno) ConsultadaEn() (time.Time, error) {
	if s.validar() != nil {
		return time.Time{}, ErrConsultaPanelInternoInvalida
	}
	return s.datos.consultadaEn, nil
}

func (s SolicitudConsultaPanelInterno) validar() error {
	if s.datos == nil {
		return ErrConsultaPanelInternoInvalida
	}
	return validarSolicitudConsultaPanelInterno(*s.datos)
}

func (SolicitudConsultaPanelInterno) String() string {
	return "[SOLICITUD-CONSULTA-PANEL-INTERNO-BOLSA-OPACA]"
}

// ConsultaPanelInterno no es un DAO libre. La implementacion productiva debe
// consumir la evidencia una sola vez, auditar el acceso y devolver el panel
// solo despues de confirmar la transaccion.
type ConsultaPanelInterno interface {
	ConsultarPanel(
		context.Context,
		SolicitudConsultaPanelInterno,
	) (InstantaneaPanelInterno, error)
}
