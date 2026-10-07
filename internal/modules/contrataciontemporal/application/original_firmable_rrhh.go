package application

import (
	"bytes"
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// FuenteOriginalFirmableRRHH prepara únicamente un original todavía ausente
// de Documentos. El servicio documental consulta primero su custodia y usa
// siempre los bytes conservados en los reintentos y descargas posteriores.
type FuenteOriginalFirmableRRHH struct {
	actual    ports.ConsultorOriginalFirmableRRHH
	historico ports.ConsultorOriginalFirmableRRHH
	PDF       ports.RenderizadorBorradorRRHH
	Tipos     ports.TiposOriginalFirmableRRHH
}

var _ vecports.FuentePDFOriginalCT = (*FuenteOriginalFirmableRRHH)(nil)

func NuevaFuenteOriginalFirmableRRHH(
	actual, historico ports.ConsultorOriginalFirmableRRHH,
	pdf ports.RenderizadorBorradorRRHH,
	tipos ports.TiposOriginalFirmableRRHH,
) (*FuenteOriginalFirmableRRHH, error) {
	if originalRRHHDependenciaNula(actual) || originalRRHHDependenciaNula(historico) ||
		originalRRHHDependenciaNula(pdf) || originalRRHHDependenciaNula(tipos) {
		return nil, vecports.ErrOriginalFirmableCTNoDisponible
	}
	return &FuenteOriginalFirmableRRHH{actual: actual, historico: historico, PDF: pdf, Tipos: tipos}, nil
}

func (f *FuenteOriginalFirmableRRHH) ObtenerPDFOriginalCT(
	ctx context.Context, in vecports.SolicitudOriginalFirmableCT,
) (vecports.PDFOriginalCT, error) {
	var vacio vecports.PDFOriginalCT
	identidad := identidadOriginalFirmableRRHH(in)
	if ctx == nil || !identidad.Valida() ||
		(in.OriginalRef != "" && in.OriginalRef != identidad.Referencia()) ||
		!domain.ReferenciaOpacaValida(in.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(in.ExpedienteRef) ||
		f == nil || originalRRHHDependenciaNula(f.actual) ||
		originalRRHHDependenciaNula(f.historico) ||
		originalRRHHDependenciaNula(f.PDF) || originalRRHHDependenciaNula(f.Tipos) {
		return vacio, vecports.ErrOriginalFirmableCTInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	tipo := ports.TipoBorradorRRHH(in.Documento)
	if !tipoOriginalFirmableRRHHAdmitido(tipo) {
		return vacio, vecports.ErrOriginalFirmableCTInvalido
	}

	// La versión observada procede de la lectura actual V3, que registra el
	// acceso. Una versión histórica solo se obtiene mediante la fachada V3
	// específica de propuesta v7, nunca por una cifra enviada aisladamente.
	solicitudActual, err := ports.NuevaSolicitudDetalleRRHH(in.ExpedienteRef, 0)
	if err != nil {
		return vacio, vecports.ErrOriginalFirmableCTInvalido
	}
	actual, err := f.actual.Consultar(ctx, solicitudActual)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, err
	}
	if actual.ValidarContenidoPublicablePara(solicitudActual) != nil ||
		actual.Resumen.OrganizacionRef != in.OrganizacionRef {
		return vacio, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	if actual.Resumen.Version < in.OriginalVersion {
		return vacio, vecports.ErrOriginalFirmableCTConflicto
	}
	detalle := actual
	if actual.Resumen.Version != in.OriginalVersion {
		if in.OriginalVersion != 7 ||
			(actual.Resumen.Version != 8 && actual.Resumen.Version != 9) {
			return vacio, vecports.ErrOriginalFirmableCTConflicto
		}
		historica, e := ports.NuevaSolicitudDetalleRRHH(in.ExpedienteRef, 7)
		if e != nil {
			return vacio, ports.ErrResultadoConsultaRRHHNoConfiable
		}
		detalle, err = f.historico.Consultar(ctx, historica)
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		if err != nil {
			return vacio, err
		}
		if detalle.ValidarContenidoPublicablePara(historica) != nil ||
			detalle.Resumen.OrganizacionRef != in.OrganizacionRef {
			return vacio, ports.ErrResultadoConsultaRRHHNoConfiable
		}
		if len(actual.Hitos) < len(detalle.Hitos) ||
			!reflect.DeepEqual(actual.Hitos[:len(detalle.Hitos)], detalle.Hitos) {
			return vacio, vecports.ErrOriginalFirmableCTConflicto
		}
	}
	if detalle.Resumen.Version != in.OriginalVersion ||
		detalle.Resumen.ExpedienteRef != in.ExpedienteRef {
		return vacio, vecports.ErrOriginalFirmableCTConflicto
	}
	if detalle.Resumen.Version < 7 || len(detalle.Hitos) < 7 ||
		detalle.Hitos[6].AccionClave != "registrar_propuesta_formalizacion" ||
		detalle.Hitos[6].FaseDestino != "nombramiento" ||
		detalle.Hitos[6].EstadoDestino != domain.EstadoEnCurso {
		return vacio, vecports.ErrOriginalFirmableCTConflicto
	}

	tipoRef, err := f.Tipos.ResolverTipoOriginalRRHH(ctx, tipo)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, err
	}
	if !almacencanonico.ReferenciaDocumentoValida(tipoRef) {
		return vacio, vecports.ErrOriginalFirmableCTNoDisponible
	}
	contenido, err := f.PDF.RenderizarBorrador(ctx, tipo, detalle.Clonar())
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
		return vacio, vecports.ErrOriginalFirmableCTConflicto
	}
	if err != nil {
		return vacio, err
	}
	if len(contenido) == 0 || len(contenido) > vecports.LimiteOriginalFirmableCT ||
		!bytes.HasPrefix(contenido, []byte("%PDF-")) {
		return vacio, vecports.ErrOriginalFirmableCTNoDisponible
	}
	return vecports.PDFOriginalCT{TipoRef: tipoRef, Contenido: bytes.Clone(contenido)}, nil
}

func identidadOriginalFirmableRRHH(in vecports.SolicitudOriginalFirmableCT) almacencanonico.IdentidadOriginalCT {
	return almacencanonico.IdentidadOriginalCT{
		OrganizacionRef: in.OrganizacionRef, ExpedienteRef: in.ExpedienteRef,
		Documento: in.Documento, Version: in.OriginalVersion,
	}
}

func tipoOriginalFirmableRRHHAdmitido(t ports.TipoBorradorRRHH) bool {
	switch t {
	case ports.BorradorInformeDefinitivo, ports.BorradorResolucion,
		ports.BorradorDiligencia, ports.BorradorTomaPosesion,
		ports.BorradorNotificacion, ports.BorradorComunicacionCentro:
		return true
	default:
		return false
	}
}

func originalRRHHDependenciaNula(v any) bool {
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
