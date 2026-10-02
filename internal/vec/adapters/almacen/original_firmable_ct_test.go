package almacen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

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
}

func (f *servicioDocumentosOriginalCTPrueba) AltaGenerado(_ context.Context, alta docports.AltaGenerado) (docdomain.Documento, error) {
	f.altas++
	f.contenido = bytes.Clone(alta.Contenido)
	f.referencia, f.version, f.tipoRef, f.expediente = alta.ID, alta.Version, alta.TipoRef, alta.ExpedienteRef
	suma := sha256.Sum256(f.contenido)
	return docdomain.Documento{
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
	}, nil
}

func (f *servicioDocumentosOriginalCTPrueba) DescargarOriginal(_ context.Context, q docports.ConsultaDocumento) (docports.Original, error) {
	f.lecturas++
	if f.referencia == "" {
		return docports.Original{}, docports.ErrNoEncontrado
	}
	if q.DocumentoID != f.referencia || q.Version != f.version || q.Autorizacion.AmbitoRef != f.expediente {
		return docports.Original{}, docports.ErrAccesoDenegado
	}
	suma := sha256.Sum256(f.contenido)
	return docports.Original{Contenido: bytes.Clone(f.contenido), MIME: "application/pdf", HuellaSHA256: hex.EncodeToString(suma[:])}, nil
}

type autorizacionesOriginalCTPrueba struct{ permitir bool }

func (a *autorizacionesOriginalCTPrueba) AutorizarLecturaOriginalCT(_ context.Context, s ports.SolicitudOriginalFirmableCT, ref string) (docports.ConsultaDocumento, error) {
	if !a.permitir {
		return docports.ConsultaDocumento{}, docports.ErrAccesoDenegado
	}
	return docports.ConsultaDocumento{DocumentoID: ref, Version: s.OriginalVersion,
		Autorizacion: docports.AutorizacionV3{RecursoRef: ref, AmbitoRef: s.ExpedienteRef}}, nil
}

func (a *autorizacionesOriginalCTPrueba) AutorizarAltaOriginalCT(_ context.Context, s ports.SolicitudOriginalFirmableCT, pdf ports.PDFOriginalCT, ref string) (docports.AltaGenerado, error) {
	if !a.permitir {
		return docports.AltaGenerado{}, docports.ErrAccesoDenegado
	}
	return docports.AltaGenerado{ID: ref, ClaveIdempotencia: ports.ClaveAltaOriginalFirmableCT(s),
		ModuloID: moduloOriginalFirmableCT, ExpedienteRef: s.ExpedienteRef,
		TipoRef: pdf.TipoRef, Version: s.OriginalVersion, MIME: "application/pdf", Contenido: bytes.Clone(pdf.Contenido),
		Autorizacion: docports.AutorizacionV3{RecursoRef: ref, AmbitoRef: s.ExpedienteRef}}, nil
}

func TestCustodiaDocumentosOriginalCTAltaUnicaConflictoYPermiso(t *testing.T) {
	s := ports.SolicitudOriginalFirmableCT{
		OrganizacionRef: "ref:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ExpedienteRef:   "ref:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Documento:       "informe_definitivo", OriginalVersion: 7,
	}
	s.OriginalRef = ports.ReferenciaOriginalFirmableCT(s)
	contenidos := []byte("%PDF-1.7\noriginal custodiado\n%%EOF")
	pdf := ports.PDFOriginalCT{TipoRef: "ref:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Contenido: contenidos}
	suma := sha256.Sum256(contenidos)
	huella := hex.EncodeToString(suma[:])
	f := &servicioDocumentosOriginalCTPrueba{}
	auth := &autorizacionesOriginalCTPrueba{permitir: true}
	a, err := NuevaCustodiaDocumentosOriginalCT(f, auth)
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
	alterado := ports.PDFOriginalCT{TipoRef: pdf.TipoRef, Contenido: []byte("%PDF-1.7\notros bytes\n%%EOF")}
	suma = sha256.Sum256(alterado.Contenido)
	if _, err := a.GuardarUnaVez(context.Background(), s, alterado, s.OriginalRef, hex.EncodeToString(suma[:])); !errors.Is(err, ports.ErrOriginalFirmableCTConflicto) || f.altas != 1 {
		t.Fatalf("bytes cambiados: err=%v altas=%d", err, f.altas)
	}
	auth.permitir = false
	antes := f.lecturas
	if _, err := a.LeerOriginal(context.Background(), s, s.OriginalRef); !errors.Is(err, docports.ErrAccesoDenegado) || f.lecturas != antes {
		t.Fatalf("permiso revocado: err=%v lecturas=%d", err, f.lecturas)
	}
	ajena := s
	ajena.ExpedienteRef = "ref:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	if _, err := a.LeerOriginal(context.Background(), ajena, s.OriginalRef); !errors.Is(err, ports.ErrOriginalFirmableCTInvalido) {
		t.Fatalf("ref de otro expediente: %v", err)
	}
}
