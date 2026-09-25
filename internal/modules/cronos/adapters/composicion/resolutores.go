package composicion

import (
	"net/http"

	"vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// ResolutorPeticionCronos construye las órdenes de los manejadores internos
// a partir de la identidad registrada de la petición. El canal remoto es el
// que acredita la política configurada; ni el cuerpo ni las cabeceras libres
// aportan identidad, empleado, canal u hora.
type ResolutorPeticionCronos struct {
	identidad   ResolutorIdentidadRegistradaCronos
	autorizador *AutorizadorCronos
	canal       domain.AcreditacionCanalMarcaje
}

func NuevoResolutorPeticionCronos(identidad ResolutorIdentidadRegistradaCronos, autorizador *AutorizadorCronos, canal domain.AcreditacionCanalMarcaje) (*ResolutorPeticionCronos, error) {
	if nulo(identidad) || autorizador == nil || canal.Validar() != nil || canal.OrigenRef() != domain.OrigenMarcajeRemoto {
		return nil, ErrComposicionCronosNoDisponible
	}
	return &ResolutorPeticionCronos{identidad: identidad, autorizador: autorizador, canal: canal}, nil
}

func (r *ResolutorPeticionCronos) peticion(req *http.Request) (proveedorPeticion, error) {
	if r == nil || req == nil || nulo(r.identidad) {
		return proveedorPeticion{}, ports.ErrDependenciaNoDisponible
	}
	id, err := r.identidad.ResolverIdentidadRegistradaCronos(req.Context())
	if err != nil {
		return proveedorPeticion{}, ports.ErrDependenciaNoDisponible
	}
	if _, err := empleadoUnico(id); err != nil {
		return proveedorPeticion{}, err
	}
	return proveedorPeticion{autorizador: r.autorizador, identidad: id}, nil
}

func (r *ResolutorPeticionCronos) ResolverConsultaSaldoPropio(req *http.Request, _ ports.PeriodoSaldo, _, _ string) (ports.OrdenConsultaSaldo, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenConsultaSaldo{}, err
	}
	return ports.NuevaOrdenConsultaSaldoAutorizada(p.identidad.Contexto.Contexto, p)
}

func (r *ResolutorPeticionCronos) ResolverMarcajeRemoto(req *http.Request) (ports.ContextoMarcajePropio, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.ContextoMarcajePropio{}, err
	}
	orden, err := ports.NuevaOrdenConsumoAutorizacion(p.identidad.Contexto.Contexto, p)
	if err != nil {
		return ports.ContextoMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	return ports.ContextoMarcajePropio{CanalAcreditado: r.canal, OrdenConsumo: orden}, nil
}

func (r *ResolutorPeticionCronos) ResolverRecuperacionMarcajeRemoto(req *http.Request) (ports.ContextoRecuperacionMarcajeRemoto, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.ContextoRecuperacionMarcajeRemoto{}, err
	}
	orden, err := ports.NuevaOrdenLecturaMarcajeRemoto(p.identidad.Contexto.Contexto, p)
	if err != nil {
		return ports.ContextoRecuperacionMarcajeRemoto{}, ports.ErrDependenciaNoDisponible
	}
	return ports.ContextoRecuperacionMarcajeRemoto{CanalAcreditado: r.canal, OrdenLectura: orden}, nil
}

var (
	_ httpinterno.ResolverConsultaSaldoPropio       = (*ResolutorPeticionCronos)(nil)
	_ httpinterno.ResolverMarcajeRemoto             = (*ResolutorPeticionCronos)(nil)
	_ httpinterno.ResolverRecuperacionMarcajeRemoto = (*ResolutorPeticionCronos)(nil)
)
