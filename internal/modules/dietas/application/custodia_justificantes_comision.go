package application

import (
	"context"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	vecports "vec-diputacion-granada/internal/vec/documentos/ports"
	baseports "vec-diputacion-granada/internal/vec/ports"
)

// CustodioJustificantesDietas identifica en Documentos quién aporta la
// referencia del justificante: Dietas la recibe declarada con su huella y no
// guarda ni sube el fichero.
const CustodioJustificantesDietas = "dietas"

// VersionJustificanteDietas es la versión documental de un justificante
// declarado: la línea no se reescribe, se sustituye en otra versión del borrador.
const VersionJustificanteDietas = 1

type RegistroDocumentoExterno interface {
	RegistrarExterno(context.Context, vecports.AltaExterna) (vecdomain.Documento, error)
}

// JustificanteComision es la referencia común de un justificante declarado en
// una línea de «otros gastos» del borrador durable.
type JustificanteComision struct {
	IndiceLinea int
	Custodia    vecdomain.ReferenciaCustodiaExterna
}

// JustificantesComision enumera, en el orden del documento, los justificantes
// con referencia y huella de una versión concreta del borrador. Una línea sin
// ambas no identifica custodia y no se anota.
func JustificantesComision(resultado dietasports.ResultadoBorradorComision) []JustificanteComision {
	documento := resultado.Comision.Documento
	if documento == nil {
		return nil
	}
	var salida []JustificanteComision
	for i, linea := range documento.Lineas {
		if linea.JustificanteRef == nil || linea.JustificanteSHA256 == nil {
			continue
		}
		salida = append(salida, JustificanteComision{IndiceLinea: i, Custodia: vecdomain.ReferenciaCustodiaExterna{
			CustodioID: CustodioJustificantesDietas, Referencia: *linea.JustificanteRef, HuellaSHA256: *linea.JustificanteSHA256,
		}})
	}
	return salida
}

// OrdenJustificanteComision anota un justificante en el expediente documental
// de la misma versión del borrador que su documento de comisión.
type OrdenJustificanteComision struct {
	ReferenciaComision string
	VersionEsperada    uint64
	IndiceLinea        int
	DocumentoID        string
	ClaveIdempotencia  string
	TipoDocumentalRef  string
	IdentidadDietas    dietasports.IdentidadEfectivaBorrador
	SolicitudPolitica  baseports.SolicitudPoliticaConservacionDocumental
	Autorizacion       vecports.AutorizacionV3
}

type ServicioJustificantesComision struct {
	lector   LectorBorradorParaDocumento
	registro RegistroDocumentoExterno
}

func NuevoServicioJustificantesComision(lector LectorBorradorParaDocumento, registro RegistroDocumentoExterno) (*ServicioJustificantesComision, error) {
	if dependenciaDocumentoComisionNula(lector) || dependenciaDocumentoComisionNula(registro) {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	return &ServicioJustificantesComision{lector: lector, registro: registro}, nil
}

// Registrar relee el borrador en su versión exacta y anota en Documentos la
// referencia y huella de la línea indicada. Nunca envía bytes: el contrato
// AltaExterna no tiene contenido. La autorización V3 la concede su autoridad;
// este caso de uso no la fabrica.
func (s *ServicioJustificantesComision) Registrar(ctx context.Context, orden OrdenJustificanteComision) (vecdomain.Documento, error) {
	agrupacionRef, err := ReferenciaAgrupacionDocumentoComision(orden.ReferenciaComision, orden.VersionEsperada)
	if ctx == nil || s == nil || dependenciaDocumentoComisionNula(s.lector) || dependenciaDocumentoComisionNula(s.registro) ||
		err != nil || orden.IndiceLinea < 0 ||
		!vecdomain.ReferenciaOpacaValida(orden.DocumentoID) || !vecdomain.ReferenciaOpacaValida(orden.ClaveIdempotencia) ||
		!vecdomain.ReferenciaOpacaValida(orden.TipoDocumentalRef) ||
		orden.SolicitudPolitica.Validar() != nil ||
		orden.SolicitudPolitica.ExpedienteRef() != agrupacionRef ||
		orden.SolicitudPolitica.TipoDocumentalRef() != orden.TipoDocumentalRef ||
		orden.Autorizacion.Accion != vecports.AccionRegistrarExterno {
		return vecdomain.Documento{}, dietasports.ErrDocumentoComisionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vecdomain.Documento{}, err
	}
	resultado, err := s.lector.ObtenerPropio(ctx, orden.IdentidadDietas, orden.ReferenciaComision)
	if err != nil {
		return vecdomain.Documento{}, err
	}
	if resultado.Comision.Referencia != orden.ReferenciaComision || resultado.Recibo.Version != orden.VersionEsperada {
		return vecdomain.Documento{}, dietasports.ErrInstantaneaDocumentoComisionInvalida
	}
	var elegido *JustificanteComision
	for _, j := range JustificantesComision(resultado) {
		if j.IndiceLinea == orden.IndiceLinea {
			elegido = &j
			break
		}
	}
	if elegido == nil || elegido.Custodia.Validar() != nil {
		return vecdomain.Documento{}, dietasports.ErrInstantaneaDocumentoComisionInvalida
	}
	registrado, err := s.registro.RegistrarExterno(ctx, vecports.AltaExterna{
		ID: orden.DocumentoID, ClaveIdempotencia: orden.ClaveIdempotencia,
		ModuloID: "dietas", ExpedienteRef: agrupacionRef, TipoRef: orden.TipoDocumentalRef,
		Version: VersionJustificanteDietas, Custodia: elegido.Custodia,
		SolicitudPolitica: orden.SolicitudPolitica, Autorizacion: orden.Autorizacion,
	})
	if err != nil {
		return vecdomain.Documento{}, err
	}
	if registrado.Validar() != nil || registrado.Custodia != vecdomain.CustodiaExterna ||
		registrado.ID != orden.DocumentoID || registrado.ModuloID != "dietas" ||
		registrado.ExpedienteRef != agrupacionRef || registrado.TipoRef != orden.TipoDocumentalRef ||
		registrado.CustodiaExternaRef != elegido.Custodia {
		return vecdomain.Documento{}, dietasports.ErrDocumentoComisionNoDisponible
	}
	return registrado, nil
}
