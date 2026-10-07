package documentosjustificacion

import (
	"context"
	"reflect"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La composición interna proporciona ambos contratos reales de la MISMA
// petición, ligados al actor de Orden. No acepta permisos ni políticas del CLI.
type ContextosRegistro interface {
	PrepararContextoRegistro(context.Context, ports.OrdenJustificacion, domain.SolicitudJustificable, domain.PoliticaJustificacion) (vecports.SolicitudPoliticaConservacionDocumental, docports.AutorizadorRegistroExterno, error)
}
type Adapter struct {
	servicio  *docapp.Servicio
	contextos ContextosRegistro
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}
func Nuevo(s *docapp.Servicio, c ContextosRegistro) (*Adapter, error) {
	if s == nil || nulo(c) || nulo(s.Repositorio) || nulo(s.Politicas) || nulo(s.Reloj) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &Adapter{s, c}, nil
}
func (a *Adapter) preparar(ctx context.Context, o ports.OrdenJustificacion, s domain.SolicitudJustificable, p domain.PoliticaJustificacion) (vecports.SolicitudPoliticaConservacionDocumental, docports.AutorizadorRegistroExterno, error) {
	if a == nil || a.servicio == nil || nulo(a.contextos) || nulo(a.servicio.Repositorio) || nulo(a.servicio.Politicas) || nulo(a.servicio.Reloj) || ctx == nil || ctx.Err() != nil || s.Validar(p) != nil {
		return vecports.SolicitudPoliticaConservacionDocumental{}, nil, ports.ErrJustificacionNoDisponible
	}
	if _, e := o.ContextoActor(); e != nil {
		return vecports.SolicitudPoliticaConservacionDocumental{}, nil, ports.ErrJustificacionNoDisponible
	}
	politica, aut, e := a.contextos.PrepararContextoRegistro(ctx, o, s, p)
	if e != nil {
		return vecports.SolicitudPoliticaConservacionDocumental{}, nil, e
	}
	if politica.Validar() != nil || politica.ExpedienteRef() != s.ExpedienteDocumentalRef || politica.TipoDocumentalRef() != p.TipoDocumentalRef || nulo(aut) {
		return vecports.SolicitudPoliticaConservacionDocumental{}, nil, ports.ErrJustificacionNoDisponible
	}
	return politica, aut, nil
}
func (a *Adapter) PrepararRegistro(ctx context.Context, o ports.OrdenJustificacion, s domain.SolicitudJustificable, p domain.PoliticaJustificacion) error {
	_, _, e := a.preparar(ctx, o, s, p)
	return e
}
func (a *Adapter) RegistrarJustificante(ctx context.Context, o ports.OrdenJustificacion, s domain.SolicitudJustificable, p domain.PoliticaJustificacion, d domain.DocumentoJustificacion, key string) (ports.RegistroDocumentalConfirmado, error) {
	if d.Validar() != nil || d.CustodioID != p.CustodioID || !domain.RefDocumentoJustificacionValida(key) {
		return ports.RegistroDocumentalConfirmado{}, domain.ErrJustificacionInvalida
	}
	politica, aut, e := a.preparar(ctx, o, s, p)
	if e != nil {
		return ports.RegistroDocumentalConfirmado{}, e
	}
	entrada := docports.AltaExterna{ID: d.ID, ClaveIdempotencia: key, ModuloID: "cronos", ExpedienteRef: s.ExpedienteDocumentalRef, TipoRef: p.TipoDocumentalRef, Version: d.Version, Custodia: docdomain.ReferenciaCustodiaExterna{CustodioID: d.CustodioID, Referencia: d.CustodiaRef, HuellaSHA256: d.SHA256}, SolicitudPolitica: politica}
	confirmado, e := a.servicio.RegistrarExternoAutorizado(ctx, entrada, aut)
	if e != nil {
		return ports.RegistroDocumentalConfirmado{}, e
	}
	return registroConfirmado(confirmado, entrada, d)
}

func registroConfirmado(confirmado docdomain.Documento, entrada docports.AltaExterna, d domain.DocumentoJustificacion) (ports.RegistroDocumentalConfirmado, error) {
	if confirmado.Validar() != nil || confirmado.ID != d.ID || confirmado.Version != d.Version || confirmado.ExpedienteRef != entrada.ExpedienteRef || confirmado.TipoRef != entrada.TipoRef || confirmado.ModuloID != entrada.ModuloID || confirmado.ModuloID != "cronos" || confirmado.Custodia != docdomain.CustodiaExterna || confirmado.HuellaSHA256 != d.SHA256 || confirmado.CustodiaExternaRef != entrada.Custodia {
		return ports.RegistroDocumentalConfirmado{}, ports.ErrJustificacionNoDisponible
	}
	return ports.RegistroDocumentalConfirmado{
		Documento: domain.DocumentoJustificacion{ID: confirmado.ID, Version: confirmado.Version, SHA256: confirmado.HuellaSHA256, CustodioID: confirmado.CustodiaExternaRef.CustodioID, CustodiaRef: confirmado.CustodiaExternaRef.Referencia},
		ModuloID:  confirmado.ModuloID, ExpedienteRef: confirmado.ExpedienteRef, TipoRef: confirmado.TipoRef,
		NumeroVEC: confirmado.NumeroVEC, CreadoEnUTC: confirmado.CreadoEn.UTC(),
		PoliticaRef: confirmado.PoliticaRef, PoliticaVersion: confirmado.VersionPolitica, PoliticaSHA256: confirmado.HuellaPoliticaSHA256,
		ConservacionHastaUTC: confirmado.ConservacionHasta.UTC(), Proteccion: confirmado.Proteccion, EstadoPolitica: confirmado.EstadoPolitica,
	}, nil
}

var _ ports.DocumentosJustificacion = (*Adapter)(nil)
