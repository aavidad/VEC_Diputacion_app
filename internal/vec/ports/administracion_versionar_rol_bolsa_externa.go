package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// ResultadoPropuestaVersionarRolBolsaExterna distingue recuperación auditada de
// primera propuesta vigente. Replay sólo puede declararlo la autoridad SQL.
type ResultadoPropuestaVersionarRolBolsaExterna struct {
	Propuesta          domain.PropuestaVersionarRolBolsaExterna
	Replay             bool
	AuditoriaAccesoRef string
}

func (r ResultadoPropuestaVersionarRolBolsaExterna) ValidarPara(
	o domain.OrdenPropuestaVersionarRolBolsaExterna, ahora time.Time) error {
	if r.Propuesta.ValidarPara(o) != nil || !referenciaAuditoriaGobiernoRol(r.AuditoriaAccesoRef) ||
		(!r.Replay && !r.Propuesta.CaducaEn.After(ahora)) {
		return domain.ErrVersionarRolBolsaExternaInvalido
	}
	return nil
}

// FuentesVersionarRolBolsaExterna conserva separados el catálogo gobernado y
// la preimagen v1 resuelta por AUT77. La huella del catálogo no incluye la base.
type FuentesVersionarRolBolsaExterna struct {
	Catalogo domain.CatalogoAccionesAdministracionV1
	Base     domain.PerfilPublicadoAdministracionV1
}

// AutoridadVersionarRolBolsaExterna es nominal y durable. Las preimágenes de asignación son expectativas
// sin autoridad que SQL debe cotejar bajo bloqueo. Proponer no publica. Cerrar
// recupera la propuesta original por referencia y consume V3 en la misma
// transacción que publicación, CAS de las asignaciones seleccionadas, historia,
// auditoría y recibo. Un fallo de cualquier parte revierte el efecto completo.
type AutoridadVersionarRolBolsaExterna interface {
	ResolverFuentesVersionarRolBolsaExterna(context.Context,
		domain.SolicitudPropuestaVersionarRolBolsaExterna) (FuentesVersionarRolBolsaExterna, error)
	ProponerVersionarRolBolsaExterna(context.Context,
		domain.OrdenPropuestaVersionarRolBolsaExterna) (ResultadoPropuestaVersionarRolBolsaExterna, error)
	CerrarVersionarRolBolsaExterna(context.Context,
		domain.SolicitudCierreVersionarRolBolsaExterna) (domain.CierreVersionarRolBolsaExterna, error)
}

// AutoridadCompletaVersionarRolBolsaExterna permite montar una sola autoridad
// SQL para el catálogo del ADMIN, los actos nominales y los métodos denegados.
type AutoridadCompletaVersionarRolBolsaExterna interface {
	AutoridadVersionarRolBolsaExterna
	CatalogoRolesAdministrables
	AutoridadActosAdministracionPerfiles
}

// ServicioVersionarRolBolsaExterna es la interfaz del handler ADMIN.
// El actor y la instantánea proceden de la sesión acreditada por la frontera.
type ServicioVersionarRolBolsaExterna interface {
	ProponerVersionarRolBolsaExterna(context.Context, domain.SolicitudPropuestaVersionarRolBolsaExterna) (ResultadoPropuestaVersionarRolBolsaExterna, error)
	CerrarVersionarRolBolsaExternaPorReferencia(context.Context, domain.SolicitudCierreVersionarRolBolsaExterna) (domain.CierreVersionarRolBolsaExterna, error)
}
