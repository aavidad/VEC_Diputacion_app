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

type MotivosInformeC12 struct {
	Propio, Jefatura, RRHH vecdomain.ReferenciaEntradaCatalogo
}
type ResolutorInformeC12 struct {
	identidad   ResolutorIdentidadRegistradaCronos
	perfiles    ResolutorAlcanceEquipoC10
	autorizador *AutorizadorCronos
	motivos     MotivosInformeC12
}

func NuevoResolutorInformeC12(id ResolutorIdentidadRegistradaCronos, perfiles ResolutorAlcanceEquipoC10, a *AutorizadorCronos, m MotivosInformeC12) (*ResolutorInformeC12, error) {
	if nulo(id) || nulo(perfiles) || a == nil {
		return nil, ErrComposicionCronosNoDisponible
	}
	for _, v := range []vecdomain.ReferenciaEntradaCatalogo{m.Propio, m.Jefatura, m.RRHH} {
		if !vecdomain.ReferenciaMotivoAutorizacionV2Valida(v) {
			return nil, ErrComposicionCronosNoDisponible
		}
	}
	return &ResolutorInformeC12{id, perfiles, a, m}, nil
}
func (r *ResolutorInformeC12) ResolverInformeC12(req *http.Request, equipo string) (ports.OrdenInformeC12, error) {
	if r == nil || req == nil || nulo(r.identidad) || r.autorizador == nil {
		return ports.OrdenInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	id, err := r.identidad.ResolverIdentidadRegistradaCronos(req.Context())
	if err != nil || id.Contexto.Validar() != nil || id.Vinculo.ValidarPara(id.Contexto) != nil {
		return ports.OrdenInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	if _, err := empleadoUnico(id); err != nil {
		return ports.OrdenInformeC12{}, err
	}
	alcance := ports.InformeC12Empleado
	if equipo != "" {
		a, err := r.perfiles.ResolverAlcanceEquipoC10(req.Context(), id)
		if err != nil {
			return ports.OrdenInformeC12{}, ports.ErrDependenciaNoDisponible
		}
		switch a {
		case ports.AlcanceEquipoJefatura:
			alcance = ports.InformeC12Jefatura
		case ports.AlcanceEquipoRRHH:
			alcance = ports.InformeC12RRHH
		default:
			return ports.OrdenInformeC12{}, ports.ErrDependenciaNoDisponible
		}
	}
	return ports.NuevaOrdenInformeC12(id.Contexto.Contexto, alcance, proveedorInformeC12{id, r.autorizador, r.motivos, alcance})
}

type proveedorInformeC12 struct {
	id          IdentidadRegistradaCronos
	autorizador *AutorizadorCronos
	motivos     MotivosInformeC12
	alcance     ports.AlcanceInformeC12
}

func (p proveedorInformeC12) ProveerMaterialInformeC12(ctx context.Context, m domain.MaterialExportacionCronos, paso ports.PasoInformeC12) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.autorizador == nil || !mismaPersona(p.id, m.ActorRef, m.PerfilRef, m.EmpleadoRef) || string(p.alcance) != m.Alcance || (paso != ports.PasoPrepararInformeC12 && paso != ports.PasoConfirmarInformeC12) || string(paso) != m.Fase {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	accion, finalidad, audiencia, ok := application.ContratoInformeC12(p.alcance)
	if !ok {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	motivo := p.motivos.Propio
	switch p.alcance {
	case ports.InformeC12Jefatura:
		motivo = p.motivos.Jefatura
	case ports.InformeC12RRHH:
		motivo = p.motivos.RRHH
	}
	recurso, err := m.Recurso()
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.id, contratoV3{accion, finalidad, audiencia, motivo}, recurso)
}

var _ httpinterno.ResolverInformeC12 = (*ResolutorInformeC12)(nil)
