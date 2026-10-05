package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioExportacionServiciosPropios struct {
	autorizador ports.ProveedorAutorizacionExportacionServiciosPropios
	repositorio ports.RepositorioExportacionServiciosPropios
	formatos    ports.ProveedorFormatoExportacionServiciosPropios
	intentos    ports.RegistroIntentosExportacionServiciosPropios
}

func NuevoServicioExportacionServiciosPropios(a ports.ProveedorAutorizacionExportacionServiciosPropios, r ports.RepositorioExportacionServiciosPropios, f ports.ProveedorFormatoExportacionServiciosPropios, i ports.RegistroIntentosExportacionServiciosPropios) (*ServicioExportacionServiciosPropios, error) {
	if nulo(a) || nulo(r) || nulo(f) || nulo(i) {
		return nil, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	return &ServicioExportacionServiciosPropios{a, r, f, i}, nil
}
func (s *ServicioExportacionServiciosPropios) Exportar(ctx context.Context, in domain.SolicitudExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error) {
	var cero ports.ResultadoExportacionServiciosPropios
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) || nulo(s.formatos) || nulo(s.intentos) {
		return cero, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	fallar := func(e error) (ports.ResultadoExportacionServiciosPropios, error) {
		return cero, s.registrarFallo(ctx, e)
	}
	if err := ctx.Err(); err != nil {
		return fallar(err)
	}
	if s.intentos.VerificarRegistroExportacionServiciosPropios(ctx) != nil {
		return fallar(domain.ErrExportacionServiciosPropiosNoDisponible)
	}
	formato, err := s.formatos.FormatoParaIdioma(ctx, in.Idioma)
	if err != nil {
		return fallar(err)
	}
	material, err := domain.NuevoMaterialExportacionServiciosPropios(in, formato)
	if err != nil {
		return fallar(err)
	}
	a, err := s.autorizador.AutorizarExportacionServiciosPropios(ctx, material)
	if err != nil {
		return fallar(err)
	}
	if !AutorizacionExportacionServiciosPropiosValida(material, a) {
		return fallar(domain.ErrExportacionServiciosPropiosNoDisponible)
	}
	r, err := s.repositorio.ExportarServiciosPropios(ctx, ports.OrdenExportacionServiciosPropios{Material: material, Autorizacion: a})
	if err != nil {
		return fallar(err)
	}
	if err := ctx.Err(); err != nil {
		return fallar(err)
	}
	if !ResultadoExportacionServiciosPropiosValido(material, a, r) {
		return fallar(domain.ErrExportacionServiciosPropiosNoDisponible)
	}
	return r, nil
}
func AutorizacionExportacionServiciosPropiosValida(m domain.MaterialExportacionServiciosPropios, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	h, err := m.HuellaSHA256()
	actor := m.Actor()
	canon, e := actor.RepresentacionCanonicaVinculadaV2()
	x := a.ResumenCapacidad()
	return err == nil && e == nil && a.ValidarEstructura() == nil && bytes.Equal(canon, a.ContextoActorCanonico()) && a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion && x.Operacion() == domain.AccionExportacionServiciosPropios && x.AudienciaConsumo() == domain.AudienciaExportacionServiciosPropios && x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == h
}
func ResultadoExportacionServiciosPropiosValido(m domain.MaterialExportacionServiciosPropios, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ResultadoExportacionServiciosPropios) bool {
	h := sha256.Sum256(r.ContenidoCSV)
	return len(r.ContenidoCSV) > 0 && len(r.ContenidoCSV) <= domain.LimiteBytesExportacionServiciosPropios && hex.EncodeToString(h[:]) == r.ContenidoSHA256 && r.NombreArchivo == m.Formato().Datos().NombreArchivo && r.Corte.VigenteEn == m.Corte().VigenteEn && r.Corte.ConocidoEn.Equal(m.Corte().ConocidoEn) && r.Evidencia.ReciboRef == m.Solicitud().ReciboRef && evidenciaFichaPropiaValida(a, r.Evidencia)
}
func errorExportacionServiciosPropios(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, domain.ErrExportacionServiciosPropiosInvalida):
		return domain.ErrExportacionServiciosPropiosInvalida
	case errors.Is(err, domain.ErrExportacionServiciosPropiosDenegada):
		return domain.ErrExportacionServiciosPropiosDenegada
	default:
		return domain.ErrExportacionServiciosPropiosNoDisponible
	}
}
func (s *ServicioExportacionServiciosPropios) registrarFallo(ctx context.Context, e error) error {
	nominal := errorExportacionServiciosPropios(ctx, e)
	motivo := "no_disponible"
	if errors.Is(nominal, domain.ErrExportacionServiciosPropiosInvalida) {
		motivo = "entrada_invalida"
	}
	if errors.Is(nominal, domain.ErrExportacionServiciosPropiosDenegada) {
		motivo = "denegado"
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if s.intentos.RegistrarIntentoExportacionServiciosPropios(auditCtx, ports.IntentoFichaPropia{Motivo: motivo}) != nil {
		return domain.ErrExportacionServiciosPropiosNoDisponible
	}
	return nominal
}
