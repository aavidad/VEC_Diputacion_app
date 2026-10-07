package almacen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	"vec-diputacion-granada/internal/vec/ports"
)

type servicioDocumentosOriginalCTPrueba struct {
	contenido  []byte
	referencia string
	version    uint64
	tipoRef    string
	expediente string
	altas      int
	lecturas   int
	documento  docdomain.Documento
}

func (f *servicioDocumentosOriginalCTPrueba) CustodiarOriginalFirmable(_ context.Context, alta docports.OrdenCustodiarOriginalFirmable, _ docports.AutorizarOriginalFirmable) (docports.IntentoOriginalFirmable, error) {
	f.altas++
	f.contenido = bytes.Clone(alta.Contenido)
	f.referencia, f.version, f.tipoRef, f.expediente = alta.ID, alta.Version, alta.TipoRef, alta.ExpedienteRef
	suma := sha256.Sum256(f.contenido)
	f.documento = docdomain.Documento{
		ID: alta.ID, NumeroVEC: "VEC-2026-1", ModuloID: alta.ModuloID,
		ExpedienteRef: alta.ExpedienteRef, TipoRef: alta.TipoRef, Version: alta.Version,
		MIME: alta.MIME, HuellaSHA256: hex.EncodeToString(suma[:]), Tamano: int64(len(f.contenido)),
		ObjetoRef: "objeto:original", ObjetoVersion: "1",
		PoliticaRef:          "ref:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		VersionPolitica:      1,
		HuellaPoliticaSHA256: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		ConservacionHasta:    time.Now().UTC().AddDate(1, 0, 0), Proteccion: "conservacion",
		EstadoPolitica: docdomain.EstadoPoliticaAprobada,
		EstadoFirma:    docdomain.EstadoFirmaPendienteProveedor, CreadoEn: time.Now().UTC(),
		Custodia: docdomain.CustodiaVEC,
	}
	suma = sha256.Sum256(f.contenido)
	return docports.IntentoOriginalFirmable{Estado: "confirmado", DocumentoID: alta.ID,
		HuellaSHA256: hex.EncodeToString(suma[:]), ReservaRef: "ref:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		ClaveAlmacenRef: "ref:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", Numero: 1}, nil
}

func (f *servicioDocumentosOriginalCTPrueba) DescargarOriginalConDocumento(_ context.Context, q docports.ConsultaDocumento) (docports.Original, docdomain.Documento, error) {
	f.lecturas++
	if f.referencia == "" {
		return docports.Original{}, docdomain.Documento{}, docports.ErrNoEncontrado
	}
	if q.DocumentoID != f.referencia || q.Version != f.version || q.Autorizacion.AmbitoRef != f.expediente {
		return docports.Original{}, docdomain.Documento{}, docports.ErrAccesoDenegado
	}
	suma := sha256.Sum256(f.contenido)
	return docports.Original{Contenido: bytes.Clone(f.contenido), MIME: "application/pdf", HuellaSHA256: hex.EncodeToString(suma[:])}, f.documento, nil
}

type autorizacionesOriginalCTPrueba struct{ permitir bool }

type tiposOriginalCTPrueba struct{ referencia string }

func (t tiposOriginalCTPrueba) ResolverTipoOriginalCT(_ context.Context, documento string) (string, error) {
	if documento != "informe_definitivo" {
		return "", ports.ErrOriginalFirmableCTNoDisponible
	}
	return t.referencia, nil
}

func (a *autorizacionesOriginalCTPrueba) AutorizarLecturaOriginalCT(_ context.Context, s ports.SolicitudOriginalFirmableCT, ref string) (docports.ConsultaDocumento, error) {
	if !a.permitir {
		return docports.ConsultaDocumento{}, docports.ErrAccesoDenegado
	}
	expedienteDocumental, _ := ctapp.ReferenciaExpedienteDocumentalFormalizacion(s.ExpedienteRef)
	return docports.ConsultaDocumento{DocumentoID: ref, Version: s.OriginalVersion,
		Autorizacion: docports.AutorizacionV3{RecursoRef: ref, AmbitoRef: expedienteDocumental}}, nil
}

func (a *autorizacionesOriginalCTPrueba) PrepararCustodiaOriginalCT(_ context.Context, s ports.SolicitudOriginalFirmableCT, pdf ports.PDFOriginalCT, ref string) (docports.OrdenCustodiarOriginalFirmable, docports.AutorizarOriginalFirmable, error) {
	if !a.permitir {
		return docports.OrdenCustodiarOriginalFirmable{}, nil, docports.ErrAccesoDenegado
	}
	identidad := almacencanonico.IdentidadOriginalCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, Documento: s.Documento, Version: s.OriginalVersion}
	expedienteDocumental, _ := ctapp.ReferenciaExpedienteDocumentalFormalizacion(s.ExpedienteRef)
	return docports.OrdenCustodiarOriginalFirmable{ID: ref, ClaveIdempotencia: identidad.ClaveLogica(),
		ModuloID: moduloOriginalFirmableCT, ExpedienteRef: expedienteDocumental,
		TipoRef: pdf.TipoRef, Version: s.OriginalVersion, MIME: "application/pdf", Contenido: bytes.Clone(pdf.Contenido)}, a, nil
}

func (*autorizacionesOriginalCTPrueba) AutorizarReservaOriginal(context.Context, []byte, string, string) (docports.AutorizacionV3, error) {
	return docports.AutorizacionV3{}, nil
}

func (*autorizacionesOriginalCTPrueba) AutorizarConfirmacionOriginal(context.Context, []byte, string, string) (docports.AutorizacionV3, error) {
	return docports.AutorizacionV3{}, nil
}

func (*autorizacionesOriginalCTPrueba) ContextoEscrituraOriginal(context.Context, docports.ReservaOriginalFirmable, docports.IntentoOriginalFirmable) (ports.ContextoOperacionAlmacen, error) {
	return ports.ContextoOperacionAlmacen{}, nil
}

func TestCustodiaDocumentosOriginalCTAltaUnicaConflictoYPermiso(t *testing.T) {
	s := ports.SolicitudOriginalFirmableCT{
		OrganizacionRef: "organizacion:desarrollo:dipgra",
		ExpedienteRef:   "expediente:ct:001",
		Documento:       "informe_definitivo", OriginalVersion: 7,
	}
	s.OriginalRef = almacencanonico.IdentidadOriginalCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, Documento: s.Documento, Version: s.OriginalVersion}.Referencia()
	contenidos := []byte("%PDF-1.7\noriginal custodiado\n%%EOF")
	pdf := ports.PDFOriginalCT{TipoRef: "ref:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Contenido: contenidos}
	suma := sha256.Sum256(contenidos)
	huella := hex.EncodeToString(suma[:])
	f := &servicioDocumentosOriginalCTPrueba{}
	auth := &autorizacionesOriginalCTPrueba{permitir: true}
	a, err := NuevaCustodiaDocumentosOriginalCT(f, auth, FuncionMapeoExpedienteOriginalCT(ctapp.ReferenciaExpedienteDocumentalFormalizacion), tiposOriginalCTPrueba{pdf.TipoRef})
	if err != nil {
		t.Fatal(err)
	}
	primero, err := a.GuardarUnaVez(context.Background(), s, pdf, s.OriginalRef, huella)
	if err != nil || f.altas != 1 || primero.HuellaSHA256 != huella {
		t.Fatalf("alta: err=%v altas=%d", err, f.altas)
	}
	_, err = a.GuardarUnaVez(context.Background(), s, pdf, s.OriginalRef, huella)
	if err != nil || f.altas != 1 {
		t.Fatalf("replay reescribio: err=%v altas=%d", err, f.altas)
	}
	otroTipo := pdf
	otroTipo.TipoRef = "ref:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	if _, err := a.GuardarUnaVez(context.Background(), s, otroTipo, s.OriginalRef, huella); !errors.Is(err, ports.ErrOriginalFirmableCTNoDisponible) || f.altas != 1 {
		t.Fatalf("tipo cambiado no rechazado: err=%v altas=%d", err, f.altas)
	}
	alterado := ports.PDFOriginalCT{TipoRef: pdf.TipoRef, Contenido: []byte("%PDF-1.7\notros bytes\n%%EOF")}
	suma = sha256.Sum256(alterado.Contenido)
	if _, err := a.GuardarUnaVez(context.Background(), s, alterado, s.OriginalRef, hex.EncodeToString(suma[:])); !errors.Is(err, ports.ErrOriginalFirmableCTConflicto) || f.altas != 1 {
		t.Fatalf("bytes cambiados: err=%v altas=%d", err, f.altas)
	}
	// El registro puede conservar los mismos bytes y huella bajo otro tipo.
	// Ni la lectura directa ni el replay deben devolver ese documento.
	f.documento.TipoRef = otroTipo.TipoRef
	if _, err := a.LeerOriginal(context.Background(), s, s.OriginalRef); !errors.Is(err, ports.ErrOriginalFirmableCTNoDisponible) {
		t.Fatalf("lectura de tipo ajeno aceptada: %v", err)
	}
	if _, err := a.GuardarUnaVez(context.Background(), s, pdf, s.OriginalRef, huella); !errors.Is(err, ports.ErrOriginalFirmableCTNoDisponible) || f.altas != 1 {
		t.Fatalf("replay de tipo ajeno aceptado: %v, altas=%d", err, f.altas)
	}
	f.documento.TipoRef = pdf.TipoRef
	auth.permitir = false
	antes := f.lecturas
	if _, err := a.LeerOriginal(context.Background(), s, s.OriginalRef); !errors.Is(err, docports.ErrAccesoDenegado) || f.lecturas != antes {
		t.Fatalf("permiso revocado: err=%v lecturas=%d", err, f.lecturas)
	}
	ajena := s
	ajena.ExpedienteRef = "expediente:ct:otro"
	if _, err := a.LeerOriginal(context.Background(), ajena, s.OriginalRef); !errors.Is(err, ports.ErrOriginalFirmableCTInvalido) {
		t.Fatalf("ref de otro expediente: %v", err)
	}
}
