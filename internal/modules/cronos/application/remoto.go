package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ErrTeletrabajoNoAutorizado permite al transporte dar un motivo genérico.
// No revela quién autorizó, fuente, unidad ni datos de otra persona.
var ErrTeletrabajoNoAutorizado = ports.ErrTeletrabajoNoAutorizado

var ErrContinuidadMarcajeNoConfirmada = ports.ErrContinuidadMarcajeNoConfirmada

var ErrMovimientoRemotoNoPermitido = ports.ErrMovimientoRemotoNoPermitido

const (
	AudienciaRecuperacionMarcajeRemoto = "vec_cronos_v1.marcaje_remoto_recibo.v1"
	AccionRecuperarMarcajeRemoto       = "cronos.marcaje.remoto.recibo.consultar"
)

type ServicioMarcajesRemotos struct {
	repositorio ports.RepositorioMarcajesRemotos
	consulta    ports.ConsultaAutorizacionTeletrabajo
	reloj       ports.Reloj
}

func NuevoServicioMarcajesRemotos(r ports.RepositorioMarcajesRemotos, c ports.ConsultaAutorizacionTeletrabajo, reloj ports.Reloj) (*ServicioMarcajesRemotos, error) {
	if dependenciaRemotaNula(r) || dependenciaRemotaNula(c) || dependenciaRemotaNula(reloj) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ServicioMarcajesRemotos{repositorio: r, consulta: c, reloj: reloj}, nil
}

func (s *ServicioMarcajesRemotos) ConsultarDisponibilidadMarcajeRemoto(ctx context.Context, c ports.ContextoMarcajePropio) (ports.DisponibilidadMarcajeRemoto, error) {
	actor, empleado, instante, err := s.contextoPropioRemoto(ctx, c)
	if err != nil {
		return ports.DisponibilidadMarcajeRemoto{}, err
	}
	periodo, autorizado, err := s.consulta.ConsultarTeletrabajoPropio(ctx, actor, empleado, instante)
	if err != nil {
		return ports.DisponibilidadMarcajeRemoto{}, ports.ErrDependenciaNoDisponible
	}
	if !autorizado {
		return ports.DisponibilidadMarcajeRemoto{MovimientosPermitidos: []domain.PunchKind{}, Motivo: "teletrabajo_no_autorizado"}, nil
	}
	if periodo.Validar() != nil || !periodo.Contiene(instante) {
		return ports.DisponibilidadMarcajeRemoto{}, ports.ErrDependenciaNoDisponible
	}
	estado, err := s.repositorio.ConfirmarContinuidadMarcajeRemoto(ctx, actor, empleado, periodo, "")
	if err != nil {
		return ports.DisponibilidadMarcajeRemoto{}, ports.ErrDependenciaNoDisponible
	}
	if domain.ValidarMovimientosRemotosPermitidos(estado.MovimientosPermitidos) != nil ||
		(!estado.ContinuidadConfirmada && len(estado.MovimientosPermitidos) != 0) {
		return ports.DisponibilidadMarcajeRemoto{}, ports.ErrDependenciaNoDisponible
	}
	disponibilidad := ports.DisponibilidadMarcajeRemoto{
		Autorizado: true, ContinuidadConfirmada: estado.ContinuidadConfirmada,
		MovimientosPermitidos: append([]domain.PunchKind{}, estado.MovimientosPermitidos...),
		Periodo:               &ports.PeriodoTeletrabajo{DesdeUTC: periodo.DesdeUTC, HastaUTC: periodo.HastaUTC},
		Motivo:                "continuidad_no_confirmada",
	}
	if estado.ContinuidadConfirmada && len(estado.MovimientosPermitidos) == 0 {
		disponibilidad.Motivo = "secuencia_no_permitida"
	} else if estado.ContinuidadConfirmada {
		disponibilidad.Motivo = "autorizado"
	}
	return disponibilidad, nil
}

func (s *ServicioMarcajesRemotos) RegistrarMarcajeRemoto(ctx context.Context, c ports.ContextoMarcajePropio, sol ports.SolicitudMarcajePropio) (ports.ReciboMarcajePropio, error) {
	actor, empleado, instante, err := s.contextoPropioRemoto(ctx, c)
	if err != nil {
		return ports.ReciboMarcajePropio{}, err
	}
	periodo, autorizado, err := s.consulta.ConsultarTeletrabajoPropio(ctx, actor, empleado, instante)
	if err != nil {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	if !autorizado {
		return ports.ReciboMarcajePropio{}, ErrTeletrabajoNoAutorizado
	}
	if periodo.Validar() != nil || !periodo.Contiene(instante) {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	if err := (domain.MarcajeOriginal{EmpleadoRef: empleado, ClaveOperacion: sol.ClaveOperacion, Movimiento: sol.Movimiento, InstanteUTC: instante, CanalAcreditado: c.CanalAcreditado}).Validate(); err != nil {
		return ports.ReciboMarcajePropio{}, err
	}
	estado, err := s.repositorio.ConfirmarContinuidadMarcajeRemoto(ctx, actor, empleado, periodo, sol.ClaveOperacion)
	if err != nil {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	if domain.ValidarMovimientosRemotosPermitidos(estado.MovimientosPermitidos) != nil ||
		(!estado.ContinuidadConfirmada && len(estado.MovimientosPermitidos) != 0) {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	if !estado.ContinuidadConfirmada {
		return ports.ReciboMarcajePropio{}, ErrContinuidadMarcajeNoConfirmada
	}
	permitido := false
	for _, movimiento := range estado.MovimientosPermitidos {
		if movimiento == sol.Movimiento {
			permitido = true
			break
		}
	}
	if !permitido {
		return ports.ReciboMarcajePropio{}, ErrMovimientoRemotoNoPermitido
	}
	base, err := NuevoServicioMarcajes(repositorioMarcajeRemoto{remoto: s.repositorio}, relojMarcajeRemoto{instante: instante})
	if err != nil {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	return base.RegistrarMarcajePropio(ctx, c, sol)
}

// RecuperarReciboMarcajeRemoto es lectura propia auditada: no comprueba
// teletrabajo vigente, continuidad ni secuencia actual y jamás crea otro hecho.
func (s *ServicioMarcajesRemotos) RecuperarReciboMarcajeRemoto(ctx context.Context, c ports.ContextoRecuperacionMarcajeRemoto, sol ports.SolicitudMarcajePropio) (ports.ReciboMarcajePropio, error) {
	if s == nil || dependenciaRemotaNula(s.repositorio) || dependenciaRemotaNula(s.reloj) || ctx == nil ||
		c.CanalAcreditado.Validar() != nil || c.CanalAcreditado.OrigenRef() != domain.OrigenMarcajeRemoto {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	actor, err := c.OrdenLectura.ContextoActor()
	if err != nil || dependenciaRemotaNula(c.OrdenLectura.ProveedorMaterial()) {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	ahora := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	material := domain.MaterialRecuperacionMarcajeRemoto{
		ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleados[0],
		ClaveOperacion: sol.ClaveOperacion, Movimiento: sol.Movimiento, Canal: c.CanalAcreditado,
	}
	if err := material.Validar(); err != nil {
		return ports.ReciboMarcajePropio{}, err
	}
	proveedor := c.OrdenLectura.ProveedorMaterial()
	if dependenciaRemotaNula(proveedor) {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	v3, err := proveedor.ProveerMaterialRecuperacionMarcajeRemoto(ctx, material)
	if err != nil || v3.ValidarEstructura() != nil || !recuperacionV3Ligada(v3, material) {
		return ports.ReciboMarcajePropio{}, ErrContextoMarcajeNoAcreditado
	}
	recibo, err := s.repositorio.RecuperarOriginalRemotoAutorizado(ctx, material, v3)
	if err != nil {
		return ports.ReciboMarcajePropio{}, err
	}
	if !recibo.Replay || recibo.MarcajeOriginalRef != "marcaje:cronos:"+sol.ClaveOperacion ||
		!strings.HasPrefix(recibo.Referencia, "recibo:cronos:") || recibo.InstanteUTC.IsZero() ||
		recibo.InstanteUTC.Location() != time.UTC || recibo.InstanteUTC.Nanosecond()%1000 != 0 {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	return recibo, nil
}

func RecursoRecuperacionMarcajeRemoto(m domain.MaterialRecuperacionMarcajeRemoto) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	h := sha256.Sum256(canonico)
	r := vecdomain.RecursoAutorizable{
		Referencia: "marcaje:cronos:" + m.ClaveOperacion, ModuloID: "cronos", Tipo: "marcaje_remoto_recibo",
		Ambitos:   map[string]string{"empleado_ref": m.EmpleadoRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return r, nil
}

func recuperacionV3Ligada(v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, m domain.MaterialRecuperacionMarcajeRemoto) bool {
	recurso, err := RecursoRecuperacionMarcajeRemoto(m)
	if err != nil {
		return false
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := v3.ResumenCapacidad()
	return err == nil && resumen.AudienciaConsumo() == AudienciaRecuperacionMarcajeRemoto &&
		resumen.Operacion() == AccionRecuperarMarcajeRemoto && resumen.EfectoRef() == recurso.Referencia &&
		resumen.EfectoHuellaSHA256() == huella
}

func (s *ServicioMarcajesRemotos) contextoPropioRemoto(ctx context.Context, c ports.ContextoMarcajePropio) (vecdomain.ContextoActor, string, time.Time, error) {
	if s == nil || dependenciaRemotaNula(s.repositorio) || dependenciaRemotaNula(s.consulta) || dependenciaRemotaNula(s.reloj) || ctx == nil ||
		c.CanalAcreditado.Validar() != nil || c.CanalAcreditado.OrigenRef() != domain.OrigenMarcajeRemoto {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrContextoMarcajeNoAcreditado
	}
	actor, err := c.OrdenConsumo.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrContextoMarcajeNoAcreditado
	}
	if dependenciaRemotaNula(c.OrdenConsumo.ProveedorMaterial()) {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrContextoMarcajeNoAcreditado
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrContextoMarcajeNoAcreditado
	}
	instante := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if instante.IsZero() || !actor.Instantanea.VigenteEn(instante) {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrContextoMarcajeNoAcreditado
	}
	return actor, empleados[0], instante, nil
}

type relojMarcajeRemoto struct{ instante time.Time }

func (r relojMarcajeRemoto) AhoraUTC() time.Time { return r.instante }

type repositorioMarcajeRemoto struct {
	remoto ports.RepositorioMarcajesRemotos
}

func (r repositorioMarcajeRemoto) RegistrarOriginalAutorizado(ctx context.Context, m domain.MarcajeOriginal, material domain.MaterialAutorizacionMarcajePropio, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMarcajePropio, error) {
	if dependenciaRemotaNula(r.remoto) || m.CanalAcreditado.OrigenRef() != domain.OrigenMarcajeRemoto {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	return r.remoto.RegistrarOriginalRemotoAutorizado(ctx, m, material, v3)
}

func dependenciaRemotaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}
