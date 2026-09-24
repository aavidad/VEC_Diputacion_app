package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionRegistrarMensajeResolucion    = "cronos.mensaje.resolucion.registrar"
	FinalidadRegistrarMensajeResolucion = "avisar_resolucion_constatada"
	AccionArchivarMensaje               = "cronos.mensaje.propio.archivar"
	FinalidadArchivarMensaje            = "archivar_aviso_propio"
)

func RecursoMensajeResolucion(m ports.MaterialMensajeResolucion) (vecdomain.RecursoAutorizable, error) {
	if !domain.ReferenciaMensajeValida(m.ResolucionRef) || !domain.ClaveMensajeValida(m.ClaveOperacion) || m.ResolucionVersion < 1 {
		return vecdomain.RecursoAutorizable{}, ports.ErrComunicacionNoAcreditada
	}
	b, err := json.Marshal(map[string]any{"version": "cronos.mensaje_resolucion.v1", "actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "resolucion_ref": m.ResolucionRef, "resolucion_version": m.ResolucionVersion, "clave_operacion": m.ClaveOperacion})
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrComunicacionNoAcreditada
	}
	h := sha256.Sum256(b)
	r := vecdomain.RecursoAutorizable{Referencia: "mensaje:cronos:" + m.ClaveOperacion, ModuloID: "cronos", Tipo: "mensaje_resolucion", Ambitos: map[string]string{"resolucion_ref": m.ResolucionRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrComunicacionNoAcreditada
	}
	return r, nil
}

func RecursoArchivoMensaje(m ports.MaterialArchivoMensaje) (vecdomain.RecursoAutorizable, error) {
	if m.Archivo.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrComunicacionNoAcreditada
	}
	b, err := json.Marshal(map[string]any{"version": "cronos.archivo_mensaje.v1", "actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.Archivo.EmpleadoRef, "mensaje_ref": m.Archivo.MensajeRef, "version_esperada": m.Archivo.VersionEsperada, "clave_operacion": m.Archivo.ClaveOperacion})
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrComunicacionNoAcreditada
	}
	h := sha256.Sum256(b)
	r := vecdomain.RecursoAutorizable{Referencia: "mensaje:cronos:" + m.Archivo.ClaveOperacion, ModuloID: "cronos", Tipo: "mensaje_archivo", Ambitos: map[string]string{"empleado_ref": m.Archivo.EmpleadoRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrComunicacionNoAcreditada
	}
	return r, nil
}

type ServicioMensajes struct {
	repositorio ports.RepositorioMensajes
	reloj       ports.Reloj
}

func NuevoServicioMensajes(r ports.RepositorioMensajes, reloj ports.Reloj) (*ServicioMensajes, error) {
	if r == nil || reloj == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ServicioMensajes{repositorio: r, reloj: reloj}, nil
}

func (s *ServicioMensajes) RegistrarMensajeResolucion(ctx context.Context, orden ports.OrdenComunicaciones, sol ports.SolicitudMensajeResolucion) (ports.ReciboMensaje, error) {
	if s == nil || s.repositorio == nil {
		return ports.ReciboMensaje{}, ports.ErrDependenciaNoDisponible
	}
	actor, ahora, err := contextoActorComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return ports.ReciboMensaje{}, err
	}
	if !domain.ReferenciaMensajeValida(sol.ResolucionRef) || !domain.ClaveMensajeValida(sol.ClaveOperacion) || sol.ResolucionVersion < 1 {
		return ports.ReciboMensaje{}, domain.ErrMensajeInvalido
	}
	m := ports.MaterialMensajeResolucion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, ResolucionRef: sol.ResolucionRef, ResolucionVersion: sol.ResolucionVersion, ClaveOperacion: sol.ClaveOperacion, InstanteUTC: ahora}
	if _, err := RecursoMensajeResolucion(m); err != nil {
		return ports.ReciboMensaje{}, ports.ErrComunicacionNoAcreditada
	}
	v3, err := orden.Proveedor().ProveerMensajeResolucion(ctx, m)
	if err != nil || v3.ValidarEstructura() != nil {
		return ports.ReciboMensaje{}, ports.ErrComunicacionNoAcreditada
	}
	recibo, err := s.repositorio.RegistrarDesdeResolucionAutorizada(ctx, m, v3)
	if err != nil {
		return ports.ReciboMensaje{}, err
	}
	if recibo.Estado != domain.MensajePendiente || !reciboMensajeValido(recibo) {
		return ports.ReciboMensaje{}, ports.ErrDependenciaNoDisponible
	}
	return recibo, nil
}

func (s *ServicioMensajes) ArchivarMensaje(ctx context.Context, orden ports.OrdenComunicaciones, archivo domain.ArchivoMensaje) (ports.ReciboMensaje, error) {
	if s == nil || s.repositorio == nil {
		return ports.ReciboMensaje{}, ports.ErrDependenciaNoDisponible
	}
	actor, empleado, ahora, err := contextoComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return ports.ReciboMensaje{}, err
	}
	archivo.EmpleadoRef = empleado
	archivo.InstanteUTC = ahora
	if archivo.Validar() != nil {
		return ports.ReciboMensaje{}, domain.ErrMensajeInvalido
	}
	m := ports.MaterialArchivoMensaje{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Archivo: archivo}
	if _, err := RecursoArchivoMensaje(m); err != nil {
		return ports.ReciboMensaje{}, ports.ErrComunicacionNoAcreditada
	}
	v3, err := orden.Proveedor().ProveerArchivoMensaje(ctx, m)
	if err != nil || v3.ValidarEstructura() != nil {
		return ports.ReciboMensaje{}, ports.ErrComunicacionNoAcreditada
	}
	recibo, err := s.repositorio.ArchivarAutorizado(ctx, m, v3)
	if err != nil {
		return ports.ReciboMensaje{}, err
	}
	if recibo.Estado != domain.MensajeArchivado || !reciboMensajeValido(recibo) {
		return ports.ReciboMensaje{}, ports.ErrDependenciaNoDisponible
	}
	return recibo, nil
}

func reciboMensajeValido(r ports.ReciboMensaje) bool {
	return domain.ReferenciaMensajeValida(r.Referencia) && domain.ReferenciaMensajeValida(r.MensajeRef) && r.Version >= 1 &&
		!r.InstanteUTC.IsZero() && r.InstanteUTC.Location() == time.UTC && r.InstanteUTC.Nanosecond()%1000 == 0
}

func (s *ServicioMensajes) ListarMensajes(ctx context.Context, orden ports.OrdenComunicaciones, pendientes bool) ([]domain.MensajeResolucion, error) {
	if s == nil || s.repositorio == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	actor, empleado, _, err := contextoComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return nil, err
	}
	return s.repositorio.ListarPropios(ctx, actor, empleado, pendientes)
}

func (s *ServicioMensajes) ConsultarMensaje(ctx context.Context, orden ports.OrdenComunicaciones, referencia string) (domain.MensajeResolucion, error) {
	if s == nil || s.repositorio == nil {
		return domain.MensajeResolucion{}, ports.ErrDependenciaNoDisponible
	}
	actor, empleado, _, err := contextoComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return domain.MensajeResolucion{}, err
	}
	if !domain.ReferenciaMensajeValida(referencia) {
		return domain.MensajeResolucion{}, domain.ErrMensajeInvalido
	}
	return s.repositorio.ConsultarPropio(ctx, actor, empleado, referencia)
}

func (s *ServicioMensajes) ConsultarHistoriaMensaje(ctx context.Context, orden ports.OrdenComunicaciones, referencia string) ([]ports.EventoMensaje, error) {
	if s == nil || s.repositorio == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	actor, empleado, _, err := contextoComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return nil, err
	}
	if !domain.ReferenciaMensajeValida(referencia) {
		return nil, domain.ErrMensajeInvalido
	}
	return s.repositorio.HistoriaPropia(ctx, actor, empleado, referencia)
}
