package bootstrap

import (
	"context"
	"errors"

	almacenvec "vec-diputacion-granada/internal/vec/adapters/almacen"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El perfil CT de firma actualmente concede la firma del borrador y la
// custodia del PDF firmado. No concede la lectura, reserva, confirmación ni
// escritura de almacén del original firmable. Tampoco hay un emisor nominal
// CT de material AD158 para esas acciones. Este puerto permanece cerrado
// hasta que ambas autoridades se compongan desde la petición CT sellada.
var errAutoridadOriginalDocumentosCTNoDisponible = errors.New("bootstrap: autoridad CT para el original de Documentos no disponible")

type autorizacionesOriginalDocumentosCTCerradas struct{}

// La fábrica es deliberadamente total y cerrada: una petición, un DTO CT o
// la autoridad de la ruta propia de Documentos no pueden crear este proveedor.
func nuevasAutorizacionesOriginalDocumentosCTDesarrollo() (almacenvec.AutorizacionesDocumentosOriginalCT, error) {
	return nil, errAutoridadOriginalDocumentosCTNoDisponible
}

func (autorizacionesOriginalDocumentosCTCerradas) AutorizarLecturaOriginalCT(
	context.Context, vecports.SolicitudOriginalFirmableCT, string,
) (docports.ConsultaDocumento, error) {
	return docports.ConsultaDocumento{}, errAutoridadOriginalDocumentosCTNoDisponible
}

func (autorizacionesOriginalDocumentosCTCerradas) PrepararCustodiaOriginalCT(
	context.Context, vecports.SolicitudOriginalFirmableCT, vecports.PDFOriginalCT, string,
) (docports.OrdenCustodiarOriginalFirmable, docports.AutorizarOriginalFirmable, error) {
	return docports.OrdenCustodiarOriginalFirmable{}, nil, errAutoridadOriginalDocumentosCTNoDisponible
}

func (autorizacionesOriginalDocumentosCTCerradas) AutorizarOperacionOriginalFirmable(
	context.Context, string, []byte, string, string,
) (docports.AutorizacionV3, error) {
	return docports.AutorizacionV3{}, errAutoridadOriginalDocumentosCTNoDisponible
}

func (autorizacionesOriginalDocumentosCTCerradas) SeudonimosLecturaOriginal(
	context.Context, docports.AutorizacionV3,
) (docautorizacion.DatosSeudonimosLectura, error) {
	return docautorizacion.DatosSeudonimosLectura{}, errAutoridadOriginalDocumentosCTNoDisponible
}

func (autorizacionesOriginalDocumentosCTCerradas) EmitirConcesionAlmacenV3(
	context.Context, docautorizacion.SolicitudConcesionAlmacenV3,
) (docautorizacion.ConcesionAlmacenV3, error) {
	return docautorizacion.ConcesionAlmacenV3{}, errAutoridadOriginalDocumentosCTNoDisponible
}

var (
	_ almacenvec.AutorizacionesDocumentosOriginalCT   = autorizacionesOriginalDocumentosCTCerradas{}
	_ docautorizacion.EmisorOperacionOriginalFirmable = autorizacionesOriginalDocumentosCTCerradas{}
	_ docautorizacion.EmisorConcesionAlmacenV3        = autorizacionesOriginalDocumentosCTCerradas{}
)
