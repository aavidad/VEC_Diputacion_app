package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strconv"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	vecports "vec-diputacion-granada/internal/vec/documentos/ports"
	baseports "vec-diputacion-granada/internal/vec/ports"
)

const TipoDocumentoComision = "dietas.comision.borrador.v1"
const mimeDocumentoComision = "application/pdf"
const limitePDFComision = 16 << 20

type LectorBorradorParaDocumento interface {
	ObtenerPropio(context.Context, dietasports.IdentidadEfectivaBorrador, string) (dietasports.ResultadoBorradorComision, error)
}

type AltaDocumentoComun interface {
	AltaGenerado(context.Context, vecports.AltaGenerado) (vecdomain.Documento, error)
}

// OrdenDocumentoComision recibe exclusivamente capacidades ya concedidas por
// sus autoridades. El caso de uso no crea una decision V3 ni una plantilla.
type OrdenDocumentoComision struct {
	ReferenciaComision string
	VersionEsperada    uint64
	DocumentoID        string
	ClaveIdempotencia  string
	TipoDocumentalRef  string
	IdentidadDietas    dietasports.IdentidadEfectivaBorrador
	ContextoAlmacen    baseports.ContextoOperacionAlmacen
	SolicitudPolitica  baseports.SolicitudPoliticaConservacionDocumental
	Autorizacion       vecports.AutorizacionV3
}

// ReferenciaAgrupacionDocumentoComision separa cada borrador y version de
// procedencia. Es una referencia opaca de agrupacion, no un expediente legal.
func ReferenciaAgrupacionDocumentoComision(referencia string, version uint64) (string, error) {
	if !referenciaComision.MatchString(referencia) || version == 0 {
		return "", dietasports.ErrInstantaneaDocumentoComisionInvalida
	}
	suma := sha256.Sum256([]byte("vec.dietas.comision.documento.v1\x00" + referencia + "\x00" + strconv.FormatUint(version, 10)))
	return "ref:" + hex.EncodeToString(suma[:]), nil
}

type ServicioDocumentoComision struct {
	lector       LectorBorradorParaDocumento
	renderizador dietasports.RenderizadorDocumentoComision
	alta         AltaDocumentoComun
}

func NuevoServicioDocumentoComision(lector LectorBorradorParaDocumento, renderizador dietasports.RenderizadorDocumentoComision, alta AltaDocumentoComun) (*ServicioDocumentoComision, error) {
	if dependenciaDocumentoComisionNula(lector) || dependenciaDocumentoComisionNula(renderizador) || dependenciaDocumentoComisionNula(alta) {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	return &ServicioDocumentoComision{lector: lector, renderizador: renderizador, alta: alta}, nil
}

// Registrar crea una representacion del borrador durable en su version exacta.
// La custodia, politica, V3, numeracion y firma quedan en Documentos. Si
// cualquiera de esas capacidades no esta disponible, no anuncia documento.
func (s *ServicioDocumentoComision) Registrar(ctx context.Context, orden OrdenDocumentoComision) (vecdomain.Documento, error) {
	agrupacionRef, err := ReferenciaAgrupacionDocumentoComision(orden.ReferenciaComision, orden.VersionEsperada)
	if ctx == nil || s == nil || dependenciaDocumentoComisionNula(s.lector) ||
		dependenciaDocumentoComisionNula(s.renderizador) || dependenciaDocumentoComisionNula(s.alta) ||
		err != nil ||
		!vecdomain.ReferenciaOpacaValida(orden.DocumentoID) || !vecdomain.ReferenciaOpacaValida(orden.ClaveIdempotencia) ||
		!vecdomain.ReferenciaOpacaValida(orden.TipoDocumentalRef) ||
		orden.SolicitudPolitica.Validar() != nil ||
		orden.SolicitudPolitica.ExpedienteRef() != agrupacionRef ||
		orden.SolicitudPolitica.TipoDocumentalRef() != orden.TipoDocumentalRef ||
		orden.Autorizacion.Accion != vecports.AccionAlta {
		return vecdomain.Documento{}, dietasports.ErrDocumentoComisionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vecdomain.Documento{}, err
	}
	resultado, err := s.lector.ObtenerPropio(ctx, orden.IdentidadDietas, orden.ReferenciaComision)
	if err != nil {
		return vecdomain.Documento{}, err
	}
	instantanea := dietasports.InstantaneaDocumentoComision{
		Referencia:   resultado.Comision.Referencia,
		Version:      resultado.Recibo.Version,
		ReciboRef:    resultado.Recibo.Referencia,
		RegistradoEn: resultado.Recibo.RegistradoEn,
		Borrador:     resultado.Comision,
	}
	if instantanea.Validar() != nil || instantanea.Version != orden.VersionEsperada ||
		instantanea.Referencia != orden.ReferenciaComision {
		return vecdomain.Documento{}, dietasports.ErrInstantaneaDocumentoComisionInvalida
	}
	pdf, err := s.renderizador.Renderizar(ctx, instantanea.Clonar())
	if err != nil {
		return vecdomain.Documento{}, err
	}
	if len(pdf) < 5 || len(pdf) > limitePDFComision || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return vecdomain.Documento{}, dietasports.ErrDocumentoComisionNoDisponible
	}
	registrado, err := s.alta.AltaGenerado(ctx, vecports.AltaGenerado{
		ID: orden.DocumentoID, ClaveIdempotencia: orden.ClaveIdempotencia,
		ModuloID: "dietas", ExpedienteRef: agrupacionRef,
		TipoRef: orden.TipoDocumentalRef, Version: orden.VersionEsperada,
		MIME: mimeDocumentoComision, Contenido: append([]byte(nil), pdf...),
		ContextoAlmacen:   orden.ContextoAlmacen,
		SolicitudPolitica: orden.SolicitudPolitica,
		Autorizacion:      orden.Autorizacion,
	})
	if err != nil {
		return vecdomain.Documento{}, err
	}
	if registrado.Validar() != nil || registrado.ID != orden.DocumentoID ||
		registrado.ModuloID != "dietas" || registrado.ExpedienteRef != agrupacionRef ||
		registrado.TipoRef != orden.TipoDocumentalRef || registrado.Version != orden.VersionEsperada ||
		registrado.MIME != mimeDocumentoComision {
		return vecdomain.Documento{}, dietasports.ErrDocumentoComisionNoDisponible
	}
	return registrado, nil
}

func dependenciaDocumentoComisionNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
