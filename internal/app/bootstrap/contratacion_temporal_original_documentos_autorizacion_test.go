package bootstrap

import (
	"context"
	"errors"
	"testing"

	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestOriginalDocumentosCTSigueCerradoSinEmisorNominal(t *testing.T) {
	proveedor, err := nuevasAutorizacionesOriginalDocumentosCTDesarrollo()
	if proveedor != nil || !errors.Is(err, errAutoridadOriginalDocumentosCTNoDisponible) {
		t.Fatalf("la fábrica debe cerrar sin autoridad CT nominal: proveedor=%T err=%v", proveedor, err)
	}

	// El contexto de la ruta Documentos tampoco es una capacidad CT.
	ctx := context.WithValue(context.Background(), claveContextoDocumentos{}, contextoDocumentos{})
	s := vecports.SolicitudOriginalFirmableCT{
		OrganizacionRef: "organizacion:demo", ExpedienteRef: "expediente:demo",
		Documento: "informe", OriginalRef: "ref:original", OriginalVersion: 1,
	}
	p := autorizacionesOriginalDocumentosCTCerradas{}
	if q, err := p.AutorizarLecturaOriginalCT(ctx, s, s.OriginalRef); q.DocumentoID != "" || q.Version != 0 ||
		!errors.Is(err, errAutoridadOriginalDocumentosCTNoDisponible) {
		t.Fatalf("la lectura no debe obtener capacidad: consulta=%+v err=%v", q, err)
	}
	orden, autoridad, err := p.PrepararCustodiaOriginalCT(ctx, s,
		vecports.PDFOriginalCT{TipoRef: "ref:tipo", Contenido: []byte("%PDF-1.7")}, s.OriginalRef)
	if orden.ID != "" || autoridad != nil || !errors.Is(err, errAutoridadOriginalDocumentosCTNoDisponible) {
		t.Fatalf("la custodia no debe obtener autoridad: orden=%+v autoridad=%T err=%v", orden, autoridad, err)
	}
	if a, err := p.AutorizarOperacionOriginalFirmable(ctx, docports.AccionReservarOriginalFirmable,
		[]byte(`{"accion":"reserva"}`), s.OriginalRef, s.ExpedienteRef); a.Accion != "" ||
		!errors.Is(err, errAutoridadOriginalDocumentosCTNoDisponible) {
		t.Fatalf("AD158 no debe obtener capacidad de reserva: accion=%q err=%v", a.Accion, err)
	}
	if a, err := p.AutorizarOperacionOriginalFirmable(ctx, docports.AccionConfirmarOriginalFirmable,
		[]byte(`{"accion":"confirmacion"}`), s.OriginalRef, s.ExpedienteRef); a.Accion != "" ||
		!errors.Is(err, errAutoridadOriginalDocumentosCTNoDisponible) {
		t.Fatalf("AD158 no debe obtener capacidad de confirmación: accion=%q err=%v", a.Accion, err)
	}
	if seudonimos, err := p.SeudonimosLecturaOriginal(ctx, docports.AutorizacionV3{}); seudonimos.SujetoHMAC != "" ||
		!errors.Is(err, errAutoridadOriginalDocumentosCTNoDisponible) {
		t.Fatalf("el replay no debe obtener seudónimos: %+v err=%v", seudonimos, err)
	}
	concesion, err := p.EmitirConcesionAlmacenV3(ctx, docautorizacion.SolicitudConcesionAlmacenV3{
		Accion: vecports.AccionNegocioEscribirOriginalFirmable,
	})
	_, errDatos := concesion.Solicitud.Datos()
	if !errors.Is(err, errAutoridadOriginalDocumentosCTNoDisponible) || errDatos == nil {
		t.Fatalf("el almacén no debe obtener concesión: err=%v", err)
	}
}
