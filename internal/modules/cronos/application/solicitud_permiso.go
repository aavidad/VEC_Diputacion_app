package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrPermisoContextoInvalido = errors.New("cronos contexto de permiso invalido")

type GeneradorReferenciasPermiso interface {
	NuevaReferenciaPermiso() (solicitudRef, reciboRef string, err error)
}
type ServicioPermisos struct {
	repo        ports.RepositorioPermisos
	calculador  ports.CalculadorCantidadPermiso
	reloj       ports.Reloj
	referencias GeneradorReferenciasPermiso
}

func NuevoServicioPermisos(repo ports.RepositorioPermisos, calculador ports.CalculadorCantidadPermiso, reloj ports.Reloj, referencias GeneradorReferenciasPermiso) (*ServicioPermisos, error) {
	if repo == nil || calculador == nil || reloj == nil || referencias == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioPermisos{repo, calculador, reloj, referencias}, nil
}

func (s *ServicioPermisos) contexto(ctx context.Context, orden ports.OrdenPermiso, exigirEmpleado bool) (vecdomain.ContextoActor, string, time.Time, error) {
	if s == nil || s.repo == nil || s.calculador == nil || s.reloj == nil || s.referencias == nil || ctx == nil {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrPermisoContextoInvalido
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrPermisoContextoInvalido
	}
	ahora := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrPermisoContextoInvalido
	}
	if !exigirEmpleado {
		return actor, "", ahora, nil
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return vecdomain.ContextoActor{}, "", time.Time{}, ErrPermisoContextoInvalido
	}
	return actor, empleados[0], ahora, nil
}

func huellaPermiso(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func (s *ServicioPermisos) ConsultarResumenAnual(ctx context.Context, orden ports.OrdenPermiso, permisoRef string, anio int) (domain.ResumenAnualPermiso, error) {
	_, empleado, _, err := s.contexto(ctx, orden, true)
	if err != nil {
		return domain.ResumenAnualPermiso{}, err
	}
	if anio < 1 || anio > 9999 || permisoRef == "" {
		return domain.ResumenAnualPermiso{}, domain.ErrCatalogoPermisoInvalido
	}
	c, err := s.repo.CatalogoVigente(ctx, permisoRef)
	if err != nil || c.Validar() != nil || c.PermisoRef != permisoRef {
		return domain.ResumenAnualPermiso{}, ports.ErrPermisoNoDisponible
	}
	hechos, err := s.repo.HechosAnuales(ctx, orden, empleado, c.PermisoRef, anio)
	if err != nil {
		return domain.ResumenAnualPermiso{}, err
	}
	return domain.ProyectarPermisoAnual(c, anio, hechos)
}

func (s *ServicioPermisos) SolicitarPermiso(ctx context.Context, orden ports.OrdenPermiso, p ports.PeticionPermiso) (ports.ReciboSolicitudPermiso, error) {
	actor, empleado, ahora, err := s.contexto(ctx, orden, true)
	if err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	if p.PermisoRef == "" || p.ClaveOperacion == "" || p.Desde == "" || p.Hasta == "" {
		return ports.ReciboSolicitudPermiso{}, domain.ErrSolicitudPermisoInvalida
	}
	huella, err := huellaPermiso(struct{ Accion, Empleado, Permiso, Desde, Hasta, Tramo string }{"solicitar", empleado, p.PermisoRef, p.Desde, p.Hasta, p.TramoHorarioRef})
	if err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	if recibo, ok, err := s.repo.RecuperarOperacion(ctx, orden, p.ClaveOperacion, huella); err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	} else if ok {
		return recibo, nil
	}
	c, err := s.repo.CatalogoVigente(ctx, p.PermisoRef)
	if err != nil || !c.VigenteEn(ahora) || c.PermisoRef != p.PermisoRef {
		return ports.ReciboSolicitudPermiso{}, ports.ErrPermisoNoDisponible
	}
	desde, e1 := time.Parse("2006-01-02", p.Desde)
	hasta, e2 := time.Parse("2006-01-02", p.Hasta)
	if e1 != nil || e2 != nil || desde.Format("2006-01-02") != p.Desde || hasta.Format("2006-01-02") != p.Hasta || hasta.Before(desde) || desde.Year() != hasta.Year() || (c.MaximoMensual != nil && (desde.Year() != hasta.Year() || desde.Month() != hasta.Month())) {
		return ports.ReciboSolicitudPermiso{}, domain.ErrSolicitudPermisoInvalida
	}
	cantidad, err := s.calculador.CalcularCantidad(ctx, c, p.Desde, p.Hasta, p.TramoHorarioRef)
	if err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	ref, reciboRef, err := s.referencias.NuevaReferenciaPermiso()
	if err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	sol := domain.SolicitudPermiso{Referencia: ref, EmpleadoRef: empleado, CatalogoVersionRef: c.VersionRef, PermisoRef: c.PermisoRef, Desde: p.Desde, Hasta: p.Hasta, Cantidad: cantidad, Unidad: c.Unidad, Estado: domain.EstadoPermisoSolicitado, Version: 1, ClaveOperacion: p.ClaveOperacion, HuellaMaterial: huella, ReciboRef: reciboRef, CreadaUTC: ahora}
	if err := sol.Validar(c); err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	material := ports.MaterialPermiso{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, SolicitudRef: ref, CatalogoVersionRef: c.VersionRef, ClaveOperacion: p.ClaveOperacion, HuellaMaterial: huella, Accion: "cronos.permiso.solicitar"}
	v3, err := orden.Proveedor().ProveerMaterialPermiso(ctx, material)
	if err != nil || v3.ValidarEstructura() != nil {
		return ports.ReciboSolicitudPermiso{}, ErrPermisoContextoInvalido
	}
	return s.repo.RegistrarSolicitud(ctx, sol, material, v3)
}

func (s *ServicioPermisos) DecidirPermiso(ctx context.Context, orden ports.OrdenPermiso, d ports.DecisionSolicitudPermiso) (ports.ReciboSolicitudPermiso, error) {
	actor, _, _, err := s.contexto(ctx, orden, false)
	if err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	if d.SolicitudRef == "" || d.ClaveOperacion == "" || d.VersionEsperada < 1 || d.MotivoRef == "" {
		return ports.ReciboSolicitudPermiso{}, domain.ErrSolicitudPermisoInvalida
	}
	huella, err := huellaPermiso(struct {
		Accion, Actor, Solicitud, Paso, Decision, Motivo string
		Version                                          int64
	}{"decidir", actor.PersonaRef, d.SolicitudRef, string(d.Paso), string(d.Decision), d.MotivoRef, d.VersionEsperada})
	if err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	if recibo, ok, err := s.repo.RecuperarOperacion(ctx, orden, d.ClaveOperacion, huella); err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	} else if ok {
		return recibo, nil
	}
	sol, err := s.repo.ConsultarSolicitud(ctx, orden, d.SolicitudRef)
	if err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	c, err := s.repo.CatalogoPorVersion(ctx, sol.CatalogoVersionRef)
	if err != nil || sol.Validar(c) != nil || sol.Version != d.VersionEsperada {
		return ports.ReciboSolicitudPermiso{}, ports.ErrPermisoEnConflicto
	}
	if _, err := domain.SiguienteEstadoPermiso(c.Circuito, sol.Estado, d.Paso, d.Decision); err != nil {
		return ports.ReciboSolicitudPermiso{}, err
	}
	material := ports.MaterialPermiso{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: sol.EmpleadoRef, SolicitudRef: sol.Referencia, CatalogoVersionRef: c.VersionRef, ClaveOperacion: d.ClaveOperacion, HuellaMaterial: huella, Accion: "cronos.permiso.decidir." + string(d.Paso), VersionEsperada: d.VersionEsperada}
	v3, err := orden.Proveedor().ProveerMaterialPermiso(ctx, material)
	if err != nil || v3.ValidarEstructura() != nil {
		return ports.ReciboSolicitudPermiso{}, ErrPermisoContextoInvalido
	}
	return s.repo.DecidirSolicitud(ctx, d, material, v3)
}
