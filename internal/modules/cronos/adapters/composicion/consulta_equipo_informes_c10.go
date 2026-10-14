package composicion

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El resolutor de perfiles comprueba la asignación fija de la sesión en el
// núcleo común. No puede recibir el alcance de una cabecera o de la URL.
type ResolutorAlcanceEquipoC10 interface {
	ResolverAlcanceEquipoC10(context.Context, IdentidadRegistradaCronos) (ports.AlcanceConsultaEquipo, error)
}
type MotivosEquipoC10 struct {
	PresenciaJefatura, PresenciaRRHH, IncidenciasJefatura, IncidenciasRRHH vecdomain.ReferenciaEntradaCatalogo
}

func (m MotivosEquipoC10) motivo(v domain.TipoConsultaEquipo, a ports.AlcanceConsultaEquipo) (vecdomain.ReferenciaEntradaCatalogo, bool) {
	switch {
	case v == domain.ConsultaPresenciaEquipo && a == ports.AlcanceEquipoJefatura:
		return m.PresenciaJefatura, true
	case v == domain.ConsultaPresenciaEquipo && a == ports.AlcanceEquipoRRHH:
		return m.PresenciaRRHH, true
	case v == domain.ConsultaIncidenciasEquipo && a == ports.AlcanceEquipoJefatura:
		return m.IncidenciasJefatura, true
	case v == domain.ConsultaIncidenciasEquipo && a == ports.AlcanceEquipoRRHH:
		return m.IncidenciasRRHH, true
	default:
		return vecdomain.ReferenciaEntradaCatalogo{}, false
	}
}

type ResolutorEquipoC10 struct {
	identidad   ResolutorIdentidadRegistradaCronos
	perfiles    ResolutorAlcanceEquipoC10
	autorizador *AutorizadorCronos
	motivos     MotivosEquipoC10
}

func NuevoResolutorEquipoC10(id ResolutorIdentidadRegistradaCronos, perfiles ResolutorAlcanceEquipoC10, a *AutorizadorCronos, m MotivosEquipoC10) (*ResolutorEquipoC10, error) {
	if nulo(id) || nulo(perfiles) || a == nil {
		return nil, ErrComposicionCronosNoDisponible
	}
	for _, ref := range []vecdomain.ReferenciaEntradaCatalogo{m.PresenciaJefatura, m.PresenciaRRHH, m.IncidenciasJefatura, m.IncidenciasRRHH} {
		if !vecdomain.ReferenciaMotivoAutorizacionV2Valida(ref) {
			return nil, ErrComposicionCronosNoDisponible
		}
	}
	return &ResolutorEquipoC10{id, perfiles, a, m}, nil
}
func (r *ResolutorEquipoC10) ResolverConsultaEquipo(req *http.Request) (ports.OrdenConsultaEquipo, error) {
	if r == nil || req == nil || nulo(r.identidad) || nulo(r.perfiles) || r.autorizador == nil {
		return ports.OrdenConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	id, err := r.identidad.ResolverIdentidadRegistradaCronos(req.Context())
	if err != nil || id.Contexto.Validar() != nil || id.Vinculo.ValidarPara(id.Contexto) != nil {
		return ports.OrdenConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	if _, err := empleadoUnico(id); err != nil {
		return ports.OrdenConsultaEquipo{}, err
	}
	alcance, err := r.perfiles.ResolverAlcanceEquipoC10(req.Context(), id)
	if err != nil {
		return ports.OrdenConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	if alcance != ports.AlcanceEquipoJefatura && alcance != ports.AlcanceEquipoRRHH {
		return ports.OrdenConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	return ports.NuevaOrdenConsultaEquipo(id.Contexto.Contexto, alcance, proveedorEquipoC10{id: id, autorizador: r.autorizador, motivos: r.motivos, alcance: alcance})
}

type proveedorEquipoC10 struct {
	id          IdentidadRegistradaCronos
	autorizador *AutorizadorCronos
	motivos     MotivosEquipoC10
	alcance     ports.AlcanceConsultaEquipo
}

func (p proveedorEquipoC10) ProveerMaterialConsultaEquipo(ctx context.Context, m domain.MaterialConsultaEquipo) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.autorizador == nil || !mismaPersona(p.id, m.ActorRef, m.PerfilRef, m.EmpleadoRef) || p.alcance != ports.AlcanceConsultaEquipo(m.Alcance) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	accion, finalidad, audiencia, ok := application.ContratoConsultaEquipo(m.Tipo, p.alcance)
	if !ok {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	motivo, ok := p.motivos.motivo(m.Tipo, p.alcance)
	if !ok {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := m.Recurso()
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.id, contratoV3{accion, finalidad, audiencia, motivo}, recurso)
}

var _ httpinterno.ResolverConsultaEquipo = (*ResolutorEquipoC10)(nil)
