package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioOriginalFirmableCT prepara una sola version del PDF desde la fuente
// CT autorizada. La comprobacion posterior de una firma solo usa Leer: nunca
// vuelve a pedir a CT que renderice los bytes originales.
type ServicioOriginalFirmableCT struct {
	fuente   ports.FuentePDFOriginalCT
	custodia ports.CustodiaOriginalFirmableCT
}

func NuevoServicioOriginalFirmableCT(fuente ports.FuentePDFOriginalCT, custodia ports.CustodiaOriginalFirmableCT) (*ServicioOriginalFirmableCT, error) {
	if dependenciaOriginalCTNula(fuente) || dependenciaOriginalCTNula(custodia) {
		return nil, ports.ErrOriginalFirmableCTNoDisponible
	}
	return &ServicioOriginalFirmableCT{fuente: fuente, custodia: custodia}, nil
}

func dependenciaOriginalCTNula(v any) bool {
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

func (s *ServicioOriginalFirmableCT) Preparar(ctx context.Context, solicitud ports.SolicitudOriginalFirmableCT) (ports.OriginalFirmableCT, error) {
	identidad := identidadOriginalFirmableCT(solicitud)
	if s == nil || ctx == nil || ctx.Err() != nil || !identidad.Valida() ||
		(solicitud.OriginalRef != "" && solicitud.OriginalRef != identidad.Referencia()) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	if dependenciaOriginalCTNula(s.fuente) || dependenciaOriginalCTNula(s.custodia) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	ref := identidad.Referencia()
	previo, err := s.custodia.LeerOriginal(ctx, solicitud, ref)
	if err == nil {
		if cotejarOriginalFirmableCT(solicitud, previo) != nil {
			return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
		}
		previo.Contenido = bytes.Clone(previo.Contenido)
		return previo, nil
	}
	if !errors.Is(err, ports.ErrOriginalFirmableCTNoEncontrado) {
		return ports.OriginalFirmableCT{}, err
	}
	pdf, err := s.fuente.ObtenerPDFOriginalCT(ctx, solicitud)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	if !almacencanonico.ReferenciaDocumentoValida(pdf.TipoRef) || !pdfOriginalCTValido(pdf.Contenido) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	// La fuente puede reutilizar su buffer. La custodia recibe una copia
	// inmutable durante toda la llamada.
	pdf.Contenido = bytes.Clone(pdf.Contenido)
	suma := sha256.Sum256(pdf.Contenido)
	huella := hex.EncodeToString(suma[:])
	original, err := s.custodia.GuardarUnaVez(ctx, solicitud, pdf, ref, huella)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	if err := cotejarOriginalFirmableCT(solicitud, original); err != nil ||
		original.TipoRef != pdf.TipoRef ||
		original.HuellaSHA256 != huella || !bytes.Equal(original.Contenido, pdf.Contenido) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTConflicto
	}
	original.Contenido = bytes.Clone(original.Contenido)
	return original, nil
}

func (s *ServicioOriginalFirmableCT) Leer(ctx context.Context, solicitud ports.SolicitudOriginalFirmableCT) (ports.OriginalFirmableCT, error) {
	identidad := identidadOriginalFirmableCT(solicitud)
	if s == nil || ctx == nil || ctx.Err() != nil || !identidad.Valida() ||
		solicitud.OriginalRef == "" || solicitud.OriginalRef != identidad.Referencia() {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	if dependenciaOriginalCTNula(s.custodia) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	original, err := s.custodia.LeerOriginal(ctx, solicitud, solicitud.OriginalRef)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	if err := cotejarOriginalFirmableCT(solicitud, original); err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	original.Contenido = bytes.Clone(original.Contenido)
	return original, nil
}

func pdfOriginalCTValido(contenido []byte) bool {
	return len(contenido) >= 8 && len(contenido) <= ports.LimiteOriginalFirmableCT &&
		bytes.HasPrefix(contenido, []byte("%PDF-"))
}

func cotejarOriginalFirmableCT(s ports.SolicitudOriginalFirmableCT, original ports.OriginalFirmableCT) error {
	if original.Referencia != identidadOriginalFirmableCT(s).Referencia() ||
		original.Version != s.OriginalVersion || !pdfOriginalCTValido(original.Contenido) ||
		!almacencanonico.HuellaSHA256Valida(original.HuellaSHA256) {
		return ports.ErrOriginalFirmableCTNoDisponible
	}
	suma := sha256.Sum256(original.Contenido)
	if hex.EncodeToString(suma[:]) != original.HuellaSHA256 {
		return ports.ErrOriginalFirmableCTNoDisponible
	}
	return nil
}

func identidadOriginalFirmableCT(s ports.SolicitudOriginalFirmableCT) almacencanonico.IdentidadOriginalCT {
	return almacencanonico.IdentidadOriginalCT{
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		Documento: s.Documento, Version: s.OriginalVersion,
	}
}
