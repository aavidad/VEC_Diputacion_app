package adapters

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecapplication "vec-diputacion-granada/internal/vec/application"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// FuenteOriginalFirmaDocumentos obtiene la revisión custodiada mediante la
// lectura autorizada de Documentos. La composición aporta el servicio con su
// custodia y PDP nominal; este adaptador no emite permisos ni renderiza PDF.
type FuenteOriginalFirmaDocumentos struct {
	servicio *vecapplication.ServicioOriginalFirmableCT
	tipos    ports.TiposOriginalFirmableRRHH
}

var _ ports.FuenteOriginalFirmaAutorizado = (*FuenteOriginalFirmaDocumentos)(nil)

func originalRRHHTiposNulos(tipos ports.TiposOriginalFirmableRRHH) bool {
	if tipos == nil {
		return true
	}
	valor := reflect.ValueOf(tipos)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func NuevaFuenteOriginalFirmaDocumentos(
	servicio *vecapplication.ServicioOriginalFirmableCT,
	tipos ports.TiposOriginalFirmableRRHH,
) (*FuenteOriginalFirmaDocumentos, error) {
	if servicio == nil || originalRRHHTiposNulos(tipos) {
		return nil, ports.ErrFuenteOriginalFirmaNoDisponible
	}
	return &FuenteOriginalFirmaDocumentos{servicio: servicio, tipos: tipos}, nil
}

func (f *FuenteOriginalFirmaDocumentos) ObtenerOriginalFirma(
	ctx context.Context, q ports.SolicitudOriginalFirma,
) (ports.OriginalFirmaAutorizado, error) {
	var vacio ports.OriginalFirmaAutorizado
	if f == nil || f.servicio == nil || originalRRHHTiposNulos(f.tipos) || ctx == nil {
		return vacio, ports.ErrFuenteOriginalFirmaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	identidad := almacencanonico.IdentidadOriginalCT{
		OrganizacionRef: q.OrganizacionRef, ExpedienteRef: q.ExpedienteRef,
		Documento: q.Documento, Version: q.OriginalVersion,
	}
	if !identidad.Valida() || !domain.ReferenciaOpacaValida(q.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(q.ExpedienteRef) ||
		!domain.ClaveDocumentoFirmaValida(q.Documento) ||
		q.OriginalRef != identidad.Referencia() {
		return vacio, ports.ErrOriginalFirmaNoAutorizado
	}
	tipoRef, err := f.tipos.ResolverTipoOriginalRRHH(ctx, ports.TipoBorradorRRHH(q.Documento))
	if errContexto := ctx.Err(); errContexto != nil {
		return vacio, errContexto
	}
	if err != nil || !almacencanonico.ReferenciaDocumentoValida(tipoRef) {
		return vacio, ports.ErrFuenteOriginalFirmaNoDisponible
	}
	solicitud := vecports.SolicitudOriginalFirmableCT{
		OrganizacionRef: q.OrganizacionRef, ExpedienteRef: q.ExpedienteRef,
		Documento: q.Documento, OriginalRef: q.OriginalRef,
		OriginalVersion: q.OriginalVersion,
	}
	original, err := f.servicio.Leer(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return vacio, errContexto
	}
	if err != nil {
		if errors.Is(err, docports.ErrAccesoDenegado) ||
			errors.Is(err, vecports.ErrOriginalFirmableCTInvalido) {
			return vacio, ports.ErrOriginalFirmaNoAutorizado
		}
		return vacio, ports.ErrFuenteOriginalFirmaNoDisponible
	}
	if original.Referencia != q.OriginalRef || original.Version != q.OriginalVersion ||
		original.TipoRef != tipoRef || len(original.Contenido) < 8 ||
		len(original.Contenido) > ports.MaximoDocumentoFirmaBytes ||
		len(original.Contenido) > vecports.LimiteOriginalFirmableCT ||
		!bytes.HasPrefix(original.Contenido, []byte("%PDF-")) ||
		!almacencanonico.HuellaSHA256Valida(original.HuellaSHA256) {
		return vacio, ports.ErrOriginalFirmaNoAutorizado
	}
	suma := sha256.Sum256(original.Contenido)
	if hex.EncodeToString(suma[:]) != original.HuellaSHA256 {
		return vacio, ports.ErrOriginalFirmaNoAutorizado
	}
	return ports.OriginalFirmaAutorizado{
		Solicitud: q, Contenido: bytes.Clone(original.Contenido), HuellaSHA256: original.HuellaSHA256,
	}, nil
}
