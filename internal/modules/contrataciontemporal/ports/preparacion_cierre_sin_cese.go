package ports

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrConsultaPreparacionCierreAdministrativoInvalida     = errors.New("contratacion temporal: consulta de preparacion de cierre invalida")
	ErrConsultaPreparacionCierreAdministrativoNoDisponible = errors.New("contratacion temporal: consulta de preparacion de cierre no disponible")
)

// OrganizacionRef procede de la frontera autenticada. Este lector privado se
// llama sólo después de ConsultaDetalleRRHHV3 para el mismo expediente/organización.
// La vista no concede cierre: el POST vuelve a validar autoridad y CAS en su TX.
type SolicitudPreparacionCierreAdministrativo struct{ OrganizacionRef, ExpedienteRef, SeguimientoRef string }

func (s SolicitudPreparacionCierreAdministrativo) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) || !domain.ReferenciaOpacaValida(s.ExpedienteRef) || !domain.ReferenciaOpacaValida(s.SeguimientoRef) {
		return ErrConsultaPreparacionCierreAdministrativoInvalida
	}
	return nil
}

type MotivoPreparacionCierreAdministrativo struct {
	MotivoClave domain.ClaveCatalogo `json:"motivo_clave"`
}
type AccionPreparacionCierreAdministrativo struct {
	TransicionClave domain.ClaveCatalogo                    `json:"transicion_clave"`
	Motivos         []MotivoPreparacionCierreAdministrativo `json:"motivos"`
}
type PreparacionCierreAdministrativo struct {
	ExpedienteRef  string                                  `json:"expediente_ref"`
	SeguimientoRef string                                  `json:"seguimiento_ref"`
	VersionActual  uint64                                  `json:"version_actual"`
	EstadoActual   domain.ClaveCatalogo                    `json:"estado_actual"`
	Acciones       []AccionPreparacionCierreAdministrativo `json:"acciones"`
	PreparadaEn    time.Time                               `json:"preparada_en"`
}

func (p PreparacionCierreAdministrativo) ValidarPara(s SolicitudPreparacionCierreAdministrativo) error {
	if s.Validar() != nil || p.ExpedienteRef != s.ExpedienteRef || p.SeguimientoRef != s.SeguimientoRef || p.VersionActual == 0 || p.VersionActual > 1<<53-1 || !p.EstadoActual.Valida() || !domain.InstanteUTCCanonico(p.PreparadaEn) || p.Acciones == nil || len(p.Acciones) > 1 {
		return ErrConsultaPreparacionCierreAdministrativoInvalida
	}
	for _, a := range p.Acciones {
		if p.VersionActual != 1 || p.EstadoActual != "vigente" || a.TransicionClave != domain.TransicionCerrarAdministrativamenteSinCese || len(a.Motivos) == 0 || len(a.Motivos) > 256 {
			return ErrConsultaPreparacionCierreAdministrativoInvalida
		}
		vistos := map[domain.ClaveCatalogo]bool{}
		for _, m := range a.Motivos {
			if !m.MotivoClave.Valida() || vistos[m.MotivoClave] {
				return ErrConsultaPreparacionCierreAdministrativoInvalida
			}
			vistos[m.MotivoClave] = true
		}
	}
	return nil
}

type LectorPreparacionCierreAdministrativo interface {
	ConsultarPreparacionCierreAdministrativo(context.Context, SolicitudPreparacionCierreAdministrativo) (PreparacionCierreAdministrativo, error)
}
