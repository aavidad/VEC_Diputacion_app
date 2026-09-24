package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrContextoMarcajeNoAcreditado = errors.New("cronos contexto de marcaje no acreditado")

type ServicioMarcajes struct {
	repositorio ports.RepositorioMarcajes
	reloj       ports.Reloj
}

func NuevoServicioMarcajes(r ports.RepositorioMarcajes, reloj ports.Reloj) (*ServicioMarcajes, error) {
	if r == nil || reloj == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioMarcajes{r, reloj}, nil
}
func (s *ServicioMarcajes) RegistrarMarcajePropio(ctx context.Context, c ports.ContextoMarcajePropio, sol ports.SolicitudMarcajePropio) (ports.ReciboMarcajePropio, error) {
	if s == nil || s.repositorio == nil || s.reloj == nil || ctx == nil || c.CanalAcreditado.Validar() != nil {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	actor, e := c.OrdenConsumo.ContextoActor()
	if e != nil {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	empleados, e := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if e != nil || len(empleados) != 1 {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	i := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if i.IsZero() || !actor.Instantanea.VigenteEn(i) {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	m := domain.MarcajeOriginal{EmpleadoRef: empleados[0], ClaveOperacion: sol.ClaveOperacion, Movimiento: sol.Movimiento, InstanteUTC: i, CanalAcreditado: c.CanalAcreditado}
	if e := m.Validate(); e != nil {
		return ports.ReciboMarcajePropio{}, e
	}
	material := domain.MaterialAutorizacionMarcajePropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleados[0], ClaveOperacion: sol.ClaveOperacion, Movimiento: sol.Movimiento, InstanteUTC: i, Canal: c.CanalAcreditado}
	proveedor := c.OrdenConsumo.ProveedorMaterial()
	if proveedor == nil || material.Validar() != nil {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	v3, e := proveedor.ProveerMaterialMarcajePropio(ctx, material)
	if e != nil || v3.ValidarEstructura() != nil {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	return s.repositorio.RegistrarOriginalAutorizado(ctx, m, material, v3)
}

const (
	AudienciaMarcajePropio          = "vec_cronos_v1.marcaje_propio.v1"
	AccionRegistrarMarcajePropio    = "cronos.marcaje.propio.registrar"
	FinalidadRegistrarMarcajePropio = "registrar_marcaje_propio"
)

// RecursoMarcajePropio is server-derived, VEC-neutral resource material for the real provider.
func RecursoMarcajePropio(m domain.MaterialAutorizacionMarcajePropio) (vecdomain.RecursoAutorizable, error) {
	if m.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	b, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	r := vecdomain.RecursoAutorizable{Referencia: "marcaje:cronos:" + m.ClaveOperacion, ModuloID: "cronos", Tipo: "marcaje_propio", Ambitos: map[string]string{"empleado_ref": m.EmpleadoRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return r, nil
}
