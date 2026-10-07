package restauracioncopias

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	d "vec-diputacion-granada/internal/modules/administracion/domain/restauracioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/restauracioncopias"
)

var ErrAutoridad = errors.New("copias_restauracion_autoridad_no_vigente")
var ErrDependencia = errors.New("copias_restauracion_dependencia_ausente")
var ErrCompatibilidad = errors.New("copias_restauracion_compatibilidad_bloqueada")
var ErrPreimagen = errors.New("copias_restauracion_preimagen_cambiada")
var ErrCopiaPrevia = errors.New("copias_restauracion_copia_previa_ausente")
var ErrVentana = errors.New("copias_restauracion_ventana_no_vigente")
var ErrCAS = errors.New("copias_restauracion_cas_conflicto")

type Servicio struct {
	Autoridad     p.Autoridad
	Observador    p.Observador
	Registro      p.RegistroPropuestas
	Reloj         p.Reloj
	Configuracion d.Configuracion
}
type Solicitud struct {
	Ref, ConjuntoRef, DestinoRef, MotivoRef, VentanaRef string
	VentanaInicio, VentanaFin, Caduca                   time.Time
}

func (s Servicio) disponible() error {
	if s.Autoridad == nil || s.Observador == nil || s.Registro == nil || s.Reloj == nil {
		return ErrDependencia
	}
	return nil
}
func (s Servicio) autorizar(ctx context.Context, sesion string, accion p.Accion, destino, sello string) (p.Concesion, error) {
	req := p.Acceso{SesionRef: sesion, Accion: accion, DestinoRef: destino, PropuestaSHA256: sello}
	c, err := s.Autoridad.AutorizarActual(ctx, req)
	if err != nil || c.Acceso != req || c.PersonaRef == "" || c.Ref == "" || !d.UTC(c.Caduca) || !s.Reloj.Ahora().Before(c.Caduca) {
		return p.Concesion{}, ErrAutoridad
	}
	return c, nil
}
func (s Servicio) compatible(o p.Observacion) error {
	if !o.ConjuntoAutenticado || !o.PoliticaAutenticada || o.Manifiesto.Verificacion.Estado != "valida" || copias.CompararVersiones(o.Manifiesto, o.Destino, o.Politica, copias.ConjuntoCompleto).Estado != copias.Compatible {
		return ErrCompatibilidad
	}
	return nil
}
func (s Servicio) Proponer(ctx context.Context, sesion string, q Solicitud) (p.Registro, error) {
	if err := s.disponible(); err != nil {
		return p.Registro{}, err
	}
	doble, err := s.Configuracion.ExigirDobleControl()
	if err != nil {
		return p.Registro{}, err
	}
	c, err := s.autorizar(ctx, sesion, p.Proponer, q.DestinoRef, "")
	if err != nil {
		return p.Registro{}, err
	}
	o, err := s.Observador.ObservarActual(ctx, q.ConjuntoRef, q.DestinoRef, q.VentanaRef)
	if err != nil {
		return p.Registro{}, ErrDependencia
	}
	if err = s.compatible(o); err != nil {
		return p.Registro{}, err
	}
	if o.Manifiesto.ConjuntoRef != q.ConjuntoRef || o.Destino.Ref != q.DestinoRef || o.VentanaRef != q.VentanaRef {
		return p.Registro{}, ErrPreimagen
	}
	propuesta := d.Propuesta{FormatoVersion: 1, Ref: q.Ref, ConjuntoRef: q.ConjuntoRef, ConjuntoSHA256: d.Huella("vec-restauracion-conjunto-v1", o.Manifiesto), DestinoRef: q.DestinoRef, PreimagenSHA256: o.PreimagenSHA256, MotivoRef: q.MotivoRef, VentanaRef: q.VentanaRef, VentanaInicio: q.VentanaInicio, VentanaFin: q.VentanaFin, PoliticaRef: o.Politica.Ref, PoliticaSHA256: d.Huella("vec-restauracion-politica-v1", o.Politica), ProponentePersonaRef: c.PersonaRef, Creada: s.Reloj.Ahora(), Caduca: q.Caduca, DobleControl: doble, Entorno: s.Configuracion.Entorno}
	sellada, err := d.Sellar(propuesta)
	if err != nil {
		return p.Registro{}, err
	}
	if err = sellada.Comprobar(s.Reloj.Ahora()); err != nil {
		return p.Registro{}, err
	}
	// Revalidar después de observar; el registro consume esta decisión al escribir.
	c, err = s.autorizar(ctx, sesion, p.Proponer, q.DestinoRef, sellada.SHA256)
	if err != nil || c.PersonaRef != propuesta.ProponentePersonaRef {
		return p.Registro{}, ErrAutoridad
	}
	return s.Registro.Crear(ctx, sellada, c)
}
func (s Servicio) comprobar(ctx context.Context, r p.Registro) error {
	if err := r.Propuesta.Comprobar(s.Reloj.Ahora()); err != nil {
		return err
	}
	q := r.Propuesta.Propuesta
	doble, err := s.Configuracion.ExigirDobleControl()
	if err != nil {
		return err
	}
	if doble != q.DobleControl || s.Configuracion.Entorno != q.Entorno {
		return d.ErrConfiguracion
	}
	if s.Autoridad.RevalidarPersona(ctx, q.ProponentePersonaRef, p.Proponer, q.DestinoRef, r.Propuesta.SHA256) != nil {
		return ErrAutoridad
	}
	o, err := s.Observador.ObservarActual(ctx, q.ConjuntoRef, q.DestinoRef, q.VentanaRef)
	if err != nil {
		return ErrDependencia
	}
	if err = s.compatible(o); err != nil {
		return err
	}
	if o.Manifiesto.ConjuntoRef != q.ConjuntoRef || o.Destino.Ref != q.DestinoRef || o.VentanaRef != q.VentanaRef || o.PreimagenSHA256 != q.PreimagenSHA256 || d.Huella("vec-restauracion-conjunto-v1", o.Manifiesto) != q.ConjuntoSHA256 || o.Politica.Ref != q.PoliticaRef || d.Huella("vec-restauracion-politica-v1", o.Politica) != q.PoliticaSHA256 {
		return ErrPreimagen
	}
	return nil
}
func (s Servicio) Revisar(ctx context.Context, sesion, ref, destino, sello string, version uint64) (p.Registro, error) {
	if err := s.disponible(); err != nil {
		return p.Registro{}, err
	}
	c, err := s.autorizar(ctx, sesion, p.Revisar, destino, sello)
	if err != nil {
		return p.Registro{}, err
	}
	r, err := s.Registro.Leer(ctx, ref, c)
	if err != nil {
		return p.Registro{}, err
	}
	if r.Version != version || r.Propuesta.SHA256 != sello || r.Propuesta.Propuesta.DestinoRef != destino || r.Propuesta.Propuesta.Ref != ref {
		return p.Registro{}, ErrCAS
	}
	if err = s.comprobar(ctx, r); err != nil {
		return p.Registro{}, err
	}
	revision, err := r.Propuesta.Revisar(c.PersonaRef, s.Reloj.Ahora())
	if err != nil {
		return p.Registro{}, err
	}
	c, err = s.autorizar(ctx, sesion, p.Revisar, destino, sello)
	if err != nil || c.PersonaRef != revision.PersonaRef {
		return p.Registro{}, ErrAutoridad
	}
	return s.Registro.RevisarCAS(ctx, ref, version, sello, revision, c)
}

// ComprobarAntesSustitucion cerca el registro; no hay ejecutor ni autorización
// transferible para sustituir. CS11 debe volver a consumir permiso, CAS y exclusión.
func (s Servicio) ComprobarAntesSustitucion(ctx context.Context, sesion, ref, destino, sello string, version uint64) error {
	if err := s.disponible(); err != nil {
		return err
	}
	c, err := s.autorizar(ctx, sesion, p.Sustituir, destino, sello)
	if err != nil {
		return err
	}
	r, err := s.Registro.Leer(ctx, ref, c)
	if err != nil {
		return err
	}
	if r.Version != version || r.Propuesta.SHA256 != sello || r.Propuesta.Propuesta.Ref != ref || r.Propuesta.Propuesta.DestinoRef != destino {
		return ErrCAS
	}
	if err = s.comprobar(ctx, r); err != nil {
		return err
	}
	q := r.Propuesta.Propuesta
	ahora := s.Reloj.Ahora()
	if ahora.Before(q.VentanaInicio) || !ahora.Before(q.VentanaFin) {
		return ErrVentana
	}
	if q.Entorno != "operativo" {
		return d.ErrConfiguracion
	}
	if r.Revision == nil {
		return d.ErrInvalida
	}
	if err = r.Propuesta.ComprobarRevision(*r.Revision, ahora); err != nil {
		return err
	}
	if s.Autoridad.RevalidarPersona(ctx, r.Revision.PersonaRef, p.Revisar, destino, sello) != nil {
		return ErrAutoridad
	}
	o, err := s.Observador.ObservarActual(ctx, q.ConjuntoRef, destino, q.VentanaRef)
	if err != nil {
		return ErrDependencia
	}
	if err = s.compatible(o); err != nil {
		return err
	}
	if o.PreimagenSHA256 != q.PreimagenSHA256 || o.Destino.Ref != destino || o.VentanaRef != q.VentanaRef || d.Huella("vec-restauracion-conjunto-v1", o.Manifiesto) != q.ConjuntoSHA256 || d.Huella("vec-restauracion-politica-v1", o.Politica) != q.PoliticaSHA256 {
		return ErrPreimagen
	}
	previa := o.CopiaPrevia
	if o.ExclusionRef == "" || previa == nil || !previa.Autenticada || previa.Manifiesto.Verificacion.Estado != "valida" || copias.CompararInventarios(o.Destino, previa.Manifiesto.Inventario).Estado != copias.Compatible || previa.DestinoRef != destino || previa.PreimagenSHA256 != q.PreimagenSHA256 || previa.ExclusionRef != o.ExclusionRef || previa.Manifiesto.ConjuntoRef == q.ConjuntoRef || previa.Manifiesto.Inicio.Before(q.VentanaInicio) || previa.Manifiesto.Fin.After(ahora) || previa.Manifiesto.Verificacion.Fecha.After(ahora) || copias.CompararVersiones(previa.Manifiesto, o.Destino, o.Politica, copias.ConjuntoCompleto).Estado != copias.Compatible {
		return ErrCopiaPrevia
	}
	c, err = s.autorizar(ctx, sesion, p.Sustituir, destino, sello)
	if err != nil {
		return err
	}
	if s.Autoridad.RevalidarPersona(ctx, q.ProponentePersonaRef, p.Proponer, destino, sello) != nil || s.Autoridad.RevalidarPersona(ctx, r.Revision.PersonaRef, p.Revisar, destino, sello) != nil {
		return ErrAutoridad
	}
	if err = r.Propuesta.Comprobar(s.Reloj.Ahora()); err != nil {
		return err
	}
	return s.Registro.Cercar(ctx, ref, version, sello, o.ExclusionRef, c)
}
