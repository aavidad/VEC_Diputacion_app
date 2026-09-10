package ports

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrDenegadaIncorporacionAplicacion    = errors.New("incorporacion: operacion denegada")
	ErrIntencionIncorporacionAplicacion   = errors.New("incorporacion: intencion invalida")
	ErrComposicionIncorporacionAplicacion = errors.New("incorporacion: servicio no disponible")
	ErrConflictoIncorporacionAplicacion   = errors.New("incorporacion: conflicto")
)

// Intencion es exclusivamente selección/confirmación del usuario. No transporta autoridad.
type IntencionIncorporacionAplicacionV2 struct {
	ExpedienteRef                    string               `json:"expediente_ref"`
	SolicitudPersonalRef             string               `json:"solicitud_personal_ref"`
	VersionActualExpedienteObservada uint64               `json:"version_actual_expediente_observada"`
	MotivoClave                      domain.ClaveCatalogo `json:"motivo_clave"`
	DocumentosRefs                   []string             `json:"documentos_refs"`
	ConfirmaRevisionPersonal         bool                 `json:"confirma_revision_personal"`
	ConfirmaEjercicioSintetico       bool                 `json:"confirma_ejercicio_sintetico"`
}

func (i IntencionIncorporacionAplicacionV2) Copia() IntencionIncorporacionAplicacionV2 {
	i.DocumentosRefs = append([]string(nil), i.DocumentosRefs...)
	return i
}
func (i IntencionIncorporacionAplicacionV2) Validar() error {
	if !domain.ReferenciaOpacaValida(i.ExpedienteRef) || !domain.ReferenciaOpacaValida(i.SolicitudPersonalRef) ||
		i.VersionActualExpedienteObservada == 0 || i.VersionActualExpedienteObservada > MaximoEnteroSeguroOperacionAnalisis ||
		!i.MotivoClave.Valida() || len(i.DocumentosRefs) > 32 ||
		!i.ConfirmaRevisionPersonal || !i.ConfirmaEjercicioSintetico {
		return ErrIntencionIncorporacionAplicacion
	}
	seen := map[string]bool{}
	for _, r := range i.DocumentosRefs {
		if !domain.ReferenciaOpacaValida(r) || seen[r] {
			return ErrIntencionIncorporacionAplicacion
		}
		seen[r] = true
	}
	return nil
}

// Preparacion procede de resolución autorizada del servidor. El implementador
// relee expediente/solicitud estable/documentos/período; nunca genera idempotencia
// al replay. No contiene resultado Personal esperado ni material del navegador.
type PreparacionIncorporacionAplicacionV2 struct {
	SolicitudPersonal          SolicitudAltaPersonalRPT
	VersionActualExpediente    uint64
	VersionSeguimientoEsperada uint64
	Periodo                    domain.IntervaloSeguimiento
	MotivoClave                domain.ClaveCatalogo
	Documentos                 []domain.DocumentoSeguimiento
	Preparacion                PreparacionSeguimientoConfirmacionIncorporacion
	SolicitudContexto          SolicitudResolverContextoAutorizacionAltaV3
	Contexto                   ContextoAutorizacionAltaV3
	MotivoV3                   core.ReferenciaEntradaCatalogo
	CorrelacionV3              core.ReferenciaCorrelacionAutorizacionV2
}

// Proyecciones públicas: no contienen capacidades, canon, principal, sesiones o
// datos que puedan reconstruir una Orden. La procedencia es obligación del lector.
type PreparacionVisibleIncorporacionV2 struct {
	SolicitudPersonalRef       string                      `json:"solicitud_personal_ref"`
	VersionSolicitudPersonal   uint64                      `json:"version_solicitud_personal"`
	VersionSeguimientoEsperada uint64                      `json:"version_seguimiento_esperada"`
	Periodo                    domain.IntervaloSeguimiento `json:"periodo_incorporacion"`
	Motivos                    []domain.ClaveCatalogo      `json:"motivos"`
	DocumentosRefs             []string                    `json:"documentos_refs"`
	Disponible                 bool                        `json:"disponible"`
}
type ReciboIncorporacionAplicacionV2 struct {
	Esquema                      string                      `json:"esquema"`
	ExpedienteRef                string                      `json:"expediente_ref"`
	SolicitudPersonalRef         string                      `json:"solicitud_personal_ref"`
	RelacionRef                  string                      `json:"relacion_ref"`
	ReciboRef                    string                      `json:"recibo_ref"`
	ActuacionRef                 string                      `json:"actuacion_ref"`
	RegistradaEn                 time.Time                   `json:"registrada_en"`
	Periodo                      domain.IntervaloSeguimiento `json:"periodo_incorporacion"`
	VersionSolicitudPersonal     uint64                      `json:"version_solicitud_personal"`
	VersionActualExpediente      uint64                      `json:"version_actual_expediente"`
	SeguimientoRef               string                      `json:"seguimiento_ref"`
	VersionSeguimientoAnterior   uint64                      `json:"version_seguimiento_anterior"`
	VersionSeguimientoResultante uint64                      `json:"version_seguimiento_resultante"`
	AuditoriaRef                 string                      `json:"auditoria_ref"`
	OutboxRef                    string                      `json:"outbox_ref"`
	EjercicioSintetico           bool                        `json:"ejercicio_sintetico"`
	FirmaOficial                 bool                        `json:"firma_oficial"`
	EficaciaAdministrativa       bool                        `json:"eficacia_administrativa"`
}
type ProyeccionIncorporacionAplicacionV2 struct {
	Esquema                 string                             `json:"esquema"`
	ExpedienteRef           string                             `json:"expediente_ref"`
	VersionActualExpediente uint64                             `json:"version_actual_expediente"`
	Preparacion             *PreparacionVisibleIncorporacionV2 `json:"preparacion"`
	Recibo                  *ReciboIncorporacionAplicacionV2   `json:"recibo"`
}

func (p ProyeccionIncorporacionAplicacionV2) Copia() ProyeccionIncorporacionAplicacionV2 {
	if p.Preparacion != nil {
		x := *p.Preparacion
		x.Motivos = append([]domain.ClaveCatalogo(nil), x.Motivos...)
		x.DocumentosRefs = append([]string(nil), x.DocumentosRefs...)
		p.Preparacion = &x
	}
	if p.Recibo != nil {
		x := *p.Recibo
		p.Recibo = &x
	}
	return p
}

// Consultar no da altas, no concede CT ni ejecuta RegistrarORecuperar. Si no hay
// resolución única/autorizada devuelve error, no un recibo ficticio/ausencia.
// Preparar valida autorización para resolver selectores de intención; eso no
// sustituye los permisos frescos independientes de alta, lectura Personal y CT.
type ProveedorPreparacionIncorporacionAplicacionV2 interface {
	Consultar(context.Context, string) (ProyeccionIncorporacionAplicacionV2, error)
	Preparar(context.Context, IntencionIncorporacionAplicacionV2) (PreparacionIncorporacionAplicacionV2, error)
}
type ServicioIncorporacionAplicacionV2 interface {
	Consultar(context.Context, string) (ProyeccionIncorporacionAplicacionV2, error)
	Confirmar(context.Context, IntencionIncorporacionAplicacionV2) (ReciboIncorporacionAplicacionV2, error)
}
