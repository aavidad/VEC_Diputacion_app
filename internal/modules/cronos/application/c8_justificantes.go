package application

import (
	"context"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// ServicioJustificantesC8 exige un original admitido por Documentos antes de
// utilizar el enlace, la autorización y el recibo duradero de Justificacion.
type ServicioJustificantesC8 struct {
	justificacion *ServicioJustificacion
	originales    ports.FuenteOriginalJustificante
}

func NuevoServicioJustificantesC8(j *ServicioJustificacion, d ports.FuenteOriginalJustificante) (*ServicioJustificantesC8, error) {
	if j == nil || dependenciaRemotaNula(d) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &ServicioJustificantesC8{j, d}, nil
}

func (s *ServicioJustificantesC8) acreditar(ctx context.Context, o ports.OrdenJustificacion, p ports.PreparacionJustificacion, id string) (ports.OriginalAcreditado, error) {
	if s == nil || s.justificacion == nil || dependenciaRemotaNula(s.originales) || !domain.RefDocumentoJustificacionValida(id) {
		return ports.OriginalAcreditado{}, ports.ErrJustificacionNoDisponible
	}
	a, err := s.originales.AcreditarOriginal(ctx, o, p.Solicitud, p.Politica, id)
	if err != nil {
		return ports.OriginalAcreditado{}, err
	}
	actor, err := o.ContextoActor()
	if err != nil || a.Documento.Validar() != nil || a.Documento.ID != id || a.ModuloID != "cronos" ||
		a.ExpedienteRef != p.Solicitud.ExpedienteDocumentalRef || a.TipoRef != p.Politica.TipoDocumentalRef ||
		a.EmpleadoPropietario != p.Solicitud.EmpleadoRef || a.PersonaAportante == "" ||
		a.Documento.CustodioID != p.Politica.CustodioID || a.EstadoAdmision != "admitido" || actor.PersonaRef == "" {
		return ports.OriginalAcreditado{}, ports.ErrJustificacionNoDisponible
	}
	return a, nil
}

func (s *ServicioJustificantesC8) Aportar(ctx context.Context, o ports.OrdenJustificacion, in ports.PeticionAporteOriginal) (ports.ResultadoAnexoJustificacion, error) {
	if s == nil || s.justificacion == nil {
		return ports.ResultadoAnexoJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	actor, p, err := s.justificacion.preparar(ctx, o, in.SolicitudRef)
	if err != nil {
		return ports.ResultadoAnexoJustificacion{}, err
	}
	if actor.PersonaRef == "" || in.DocumentoVersion == 0 || !domain.HuellaEfectosValida(in.SHA256) {
		return ports.ResultadoAnexoJustificacion{}, domain.ErrJustificacionInvalida
	}
	a, err := s.acreditar(ctx, o, p, in.DocumentoID)
	if err != nil {
		return ports.ResultadoAnexoJustificacion{}, err
	}
	if a.PersonaAportante != actor.PersonaRef {
		return ports.ResultadoAnexoJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	d := a.Documento
	if d.Version != in.DocumentoVersion || d.SHA256 != in.SHA256 {
		return ports.ResultadoAnexoJustificacion{}, domain.ErrJustificacionConflicto
	}
	return s.justificacion.Anexar(ctx, o, ports.PeticionAnexoJustificacion{
		SolicitudRef: in.SolicitudRef, ClaveOperacion: in.ClaveOperacion,
		VersionEsperada: in.VersionEsperada, Documento: d,
	})
}

func (s *ServicioJustificantesC8) Revisar(ctx context.Context, o ports.OrdenJustificacion, in ports.PeticionRevisionOriginal) (ports.ReciboJustificacion, error) {
	if s == nil || s.justificacion == nil {
		return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	_, p, err := s.justificacion.preparar(ctx, o, in.SolicitudRef)
	if err != nil {
		return ports.ReciboJustificacion{}, err
	}
	if p.Actual == nil {
		return ports.ReciboJustificacion{}, domain.ErrJustificacionConflicto
	}
	a, err := s.acreditar(ctx, o, p, p.Actual.Vinculo.Documento.ID)
	if err != nil {
		return ports.ReciboJustificacion{}, err
	}
	if a.Documento != p.Actual.Vinculo.Documento {
		return ports.ReciboJustificacion{}, domain.ErrJustificacionConflicto
	}
	return s.justificacion.Revisar(ctx, o, ports.PeticionRevisionJustificacion{
		SolicitudRef: in.SolicitudRef, ClaveOperacion: in.ClaveOperacion,
		VersionEsperada: in.VersionEsperada, Vinculo: p.Actual.Vinculo,
		Decision: in.Decision, MotivoRef: in.MotivoRef,
	})
}
