package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	almacenvec "vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type puertaOriginalCTPrueba struct{ err error }

func (p puertaOriginalCTPrueba) Verificar(context.Context) error { return p.err }

type consultorOriginalCTMontajePrueba struct {
	ctports.ConsultorOriginalFirmableRRHH
}
type renderOriginalCTMontajePrueba struct {
	ctports.RenderizadorBorradorRRHH
}
type autorizacionesOriginalCTMontajePrueba struct {
	almacenvec.AutorizacionesDocumentosOriginalCT
	docports.FabricaContextoLectura
}
type almacenOriginalCTMontajePrueba struct{ vecports.AlmacenObjetos }
type lecturaOriginalCTMontajePrueba struct {
	docports.FabricaContextoLectura
}

func preparacionOriginalCTPrueba(t *testing.T, v2 bool, puerta puertaOriginalFirmableCTDesarrollo) *autoridadDocumentosDesarrollo {
	t.Helper()
	reloj := relojRutasDietas{}
	var (
		catalogo *conservacion.Catalogo
		err      error
	)
	if v2 {
		catalogo, err = conservacion.NuevoCatalogoProvisionalV2(reloj)
	} else {
		catalogo, err = conservacion.NuevoCatalogoProvisional(reloj)
	}
	if err != nil {
		t.Fatal(err)
	}
	return &autoridadDocumentosDesarrollo{originalCT: &preparacionOriginalFirmableCTDesarrollo{
		servicio: &docapp.Servicio{Repositorio: &docpg.Repositorio{}, Almacen: almacenOriginalCTMontajePrueba{},
			Politicas: catalogo, Reloj: reloj, ContextosLectura: lecturaOriginalCTMontajePrueba{}},
		catalogo: catalogo, puerta: puerta,
	}}
}

func TestOriginalFirmableCTMontajeDeniegaSinAutoridadOContrato(t *testing.T) {
	ctx := context.Background()
	lector := consultorOriginalCTMontajePrueba{}
	render := renderOriginalCTMontajePrueba{}
	autorizaciones := autorizacionesOriginalCTMontajePrueba{}
	for _, tc := range []struct {
		nombre     string
		documentos *autoridadDocumentosDesarrollo
		permisos   almacenvec.AutorizacionesDocumentosOriginalCT
	}{
		{"sin_documentos", nil, autorizaciones},
		{"sin_autorizaciones_nominales", preparacionOriginalCTPrueba(t, true, puertaOriginalCTPrueba{}), nil},
		{"sin_doc13_o_ad158", preparacionOriginalCTPrueba(t, true, puertaOriginalCTPrueba{errors.New("fachada ausente")}), autorizaciones},
		{"catalogo_v1_sin_reserva", preparacionOriginalCTPrueba(t, false, puertaOriginalCTPrueba{}), autorizaciones},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			servicio, err := nuevoServicioOriginalFirmableCTDesarrollo(ctx, tc.documentos, lector, lector, render, tc.permisos)
			if servicio != nil || !errors.Is(err, ErrOriginalFirmableCTMontajeNoDisponible) {
				t.Fatalf("montaje debe denegarse: servicio=%v error=%v", servicio, err)
			}
		})
	}
}

func TestOriginalFirmableCTMontajeCatalogoV2YDependencias(t *testing.T) {
	documentos := preparacionOriginalCTPrueba(t, true, puertaOriginalCTPrueba{})
	lector := consultorOriginalCTMontajePrueba{}
	servicio, err := nuevoServicioOriginalFirmableCTDesarrollo(context.Background(), documentos,
		lector, lector, renderOriginalCTMontajePrueba{}, autorizacionesOriginalCTMontajePrueba{})
	if err != nil || servicio == nil {
		t.Fatalf("montaje con tipos reservados: servicio=%v error=%v", servicio, err)
	}
}

type lecturaPDFAnteriorPrueba struct {
	pdf      docports.Original
	doc      docdomain.Documento
	err      error
	llamadas int
}

func (p *lecturaPDFAnteriorPrueba) DescargarOriginalConDocumento(context.Context, docports.ConsultaDocumento) (docports.Original, docdomain.Documento, error) {
	p.llamadas++
	return p.pdf, p.doc, p.err
}

type autorizadorPDFAnteriorPrueba struct {
	err      error
	llamadas int
}

func (a *autorizadorPDFAnteriorPrueba) AutorizarLecturaPDFFirmaAnteriorCT(_ context.Context, q ctports.SolicitudPDFFirmaAnterior) (docports.ConsultaDocumento, error) {
	a.llamadas++
	exp, _ := ctapp.ReferenciaExpedienteDocumentalFormalizacion(q.ExpedienteRef)
	return docports.ConsultaDocumento{DocumentoID: q.DocumentoRef, Version: q.DocumentoVersion,
		Autorizacion: docports.AutorizacionV3{RecursoRef: q.DocumentoRef, AmbitoRef: exp}}, a.err
}

func TestPDFAnteriorCTLeeRevisionExactaYDeniegaMutaciones(t *testing.T) {
	contenido := []byte("%PDF-1.7\nrevision anterior custodiada")
	h := sha256.Sum256(contenido)
	q := ctports.SolicitudPDFFirmaAnterior{OrganizacionRef: docports.OrganizacionRefV3, ExpedienteRef: "expediente:ct:anterior",
		Documento: "informe_definitivo", FirmaRef: "firma:ct:anterior", ReciboRef: "recibo:ct:anterior", DocumentoRef: "documento:ct:anterior", DocumentoVersion: 1, DocumentoHuella: hex.EncodeToString(h[:])}
	exp, _ := ctapp.ReferenciaExpedienteDocumentalFormalizacion(q.ExpedienteRef)
	d := docdomain.Documento{ID: q.DocumentoRef, NumeroVEC: "VEC-2026-000000001", ModuloID: moduloProductorCustodiaCT,
		ExpedienteRef: exp, TipoRef: "tipo:informe:firmado", Version: q.DocumentoVersion, MIME: "application/pdf", HuellaSHA256: q.DocumentoHuella,
		Tamano: int64(len(contenido)), ObjetoRef: "objeto:anterior", ObjetoVersion: "v1", PoliticaRef: "politica:anterior", VersionPolitica: 1,
		HuellaPoliticaSHA256: strings.Repeat("b", 64), ConservacionHasta: time.Now().UTC().Add(time.Hour), CreadoEn: time.Now().UTC(),
		Proteccion: "conservacion", EstadoPolitica: docdomain.EstadoPoliticaProvisional, EstadoFirma: docdomain.EstadoFirmaPendienteProveedor, Custodia: docdomain.CustodiaVEC}
	if err := d.Validar(); err != nil {
		t.Fatal(err)
	}
	lector := &lecturaPDFAnteriorPrueba{doc: d, pdf: docports.Original{Contenido: contenido, MIME: d.MIME, HuellaSHA256: d.HuellaSHA256}}
	a := &autorizadorPDFAnteriorPrueba{}
	f := &fuentePDFFirmaAnteriorCT{servicio: lector, autorizador: a, tipos: map[string]string{q.Documento: d.TipoRef}}
	r, err := f.ObtenerPDFFirmaAnterior(context.Background(), q)
	if err != nil || r.Solicitud != q || string(r.Contenido) != string(contenido) || a.llamadas != 1 || lector.llamadas != 1 {
		t.Fatalf("revisión: %v", err)
	}
	r.Contenido[0] = 'X'
	if contenido[0] != '%' {
		t.Fatal("bytes compartidos")
	}
	for nombre, mutar := range map[string]func(){
		"expediente": func() { lector.doc.ExpedienteRef = "expediente:ajeno" },
		"modulo":     func() { lector.doc.ModuloID = "personal" },
		"tipo":       func() { lector.doc.TipoRef = "tipo:ajeno" },
		"version":    func() { lector.doc.Version++ },
		"contenido":  func() { lector.pdf.Contenido = []byte("%PDF-1.7\ncontenido sustituido") },
	} {
		t.Run(nombre, func(t *testing.T) {
			lector.doc, lector.pdf = d, docports.Original{Contenido: contenido, MIME: d.MIME, HuellaSHA256: d.HuellaSHA256}
			mutar()
			if r, err := f.ObtenerPDFFirmaAnterior(context.Background(), q); !errors.Is(err, ctports.ErrOriginalFirmaNoAutorizado) || len(r.Contenido) != 0 {
				t.Fatalf("sustitución: %v", err)
			}
		})
	}
	a.err = docports.ErrAccesoDenegado
	antes := lector.llamadas
	if _, err := f.ObtenerPDFFirmaAnterior(context.Background(), q); !errors.Is(err, docports.ErrAccesoDenegado) || lector.llamadas != antes {
		t.Fatalf("denegación alcanzó lectura: %v", err)
	}
}
