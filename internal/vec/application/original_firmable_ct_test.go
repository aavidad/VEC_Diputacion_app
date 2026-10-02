package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteOriginalCTPrueba struct {
	pdf      ports.PDFOriginalCT
	llamadas int
}

func (f *fuenteOriginalCTPrueba) ObtenerPDFOriginalCT(context.Context, ports.SolicitudOriginalFirmableCT) (ports.PDFOriginalCT, error) {
	f.llamadas++
	return f.pdf, nil
}

type custodiaOriginalCTPrueba struct {
	guardado  ports.OriginalFirmableCT
	altas     int
	lecturas  int
	permitir  bool
	pertenece string
}

func (c *custodiaOriginalCTPrueba) GuardarUnaVez(_ context.Context, s ports.SolicitudOriginalFirmableCT, pdf ports.PDFOriginalCT, ref, huella string) (ports.OriginalFirmableCT, error) {
	if !c.permitir || s.ExpedienteRef != c.pertenece {
		return ports.OriginalFirmableCT{}, errors.New("pdp: acceso denegado")
	}
	if c.guardado.Referencia != "" {
		if c.guardado.Referencia != ref || c.guardado.Version != s.OriginalVersion ||
			c.guardado.HuellaSHA256 != huella || !bytes.Equal(c.guardado.Contenido, pdf.Contenido) {
			return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTConflicto
		}
		return c.guardado, nil
	}
	c.altas++
	c.guardado = ports.OriginalFirmableCT{Referencia: ref, TipoRef: pdf.TipoRef, Version: s.OriginalVersion, HuellaSHA256: huella, Contenido: bytes.Clone(pdf.Contenido)}
	return c.guardado, nil
}

func (c *custodiaOriginalCTPrueba) LeerOriginal(_ context.Context, s ports.SolicitudOriginalFirmableCT, ref string) (ports.OriginalFirmableCT, error) {
	c.lecturas++
	if !c.permitir || s.ExpedienteRef != c.pertenece {
		return ports.OriginalFirmableCT{}, errors.New("pdp: acceso denegado")
	}
	if c.guardado.Referencia != ref {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	return c.guardado, nil
}

func solicitudOriginalCTPrueba() ports.SolicitudOriginalFirmableCT {
	s := ports.SolicitudOriginalFirmableCT{
		OrganizacionRef: "ref:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ExpedienteRef:   "ref:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Documento:       "informe_definitivo",
		OriginalVersion: 7,
	}
	s.OriginalRef = ports.ReferenciaOriginalFirmableCT(s)
	return s
}

func TestOriginalFirmableCTAltaUnicaReplayConflictoYLecturaSinRegenerar(t *testing.T) {
	s := solicitudOriginalCTPrueba()
	contenido := []byte("%PDF-1.7\noriginal version 7\n%%EOF")
	fuente := &fuenteOriginalCTPrueba{pdf: ports.PDFOriginalCT{TipoRef: "ref:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Contenido: contenido}}
	custodia := &custodiaOriginalCTPrueba{permitir: true, pertenece: s.ExpedienteRef}
	servicio, err := NuevoServicioOriginalFirmableCT(fuente, custodia)
	if err != nil {
		t.Fatal(err)
	}
	primero, err := servicio.Preparar(context.Background(), s)
	if err != nil || custodia.altas != 1 {
		t.Fatalf("primera alta: err=%v altas=%d", err, custodia.altas)
	}
	replay, err := servicio.Preparar(context.Background(), s)
	if err != nil || custodia.altas != 1 || !bytes.Equal(primero.Contenido, replay.Contenido) || primero.HuellaSHA256 != replay.HuellaSHA256 {
		t.Fatalf("replay cambió el original: err=%v altas=%d", err, custodia.altas)
	}
	fuente.pdf.Contenido = []byte("%PDF-1.7\noriginal alterado\n%%EOF")
	if _, err := servicio.Preparar(context.Background(), s); !errors.Is(err, ports.ErrOriginalFirmableCTConflicto) || custodia.altas != 1 {
		t.Fatalf("cambio de bytes no rechazado: err=%v altas=%d", err, custodia.altas)
	}
	antes := fuente.llamadas
	leido, err := servicio.Leer(context.Background(), s)
	if err != nil || fuente.llamadas != antes || !bytes.Equal(leido.Contenido, contenido) || leido.HuellaSHA256 != primero.HuellaSHA256 {
		t.Fatalf("lectura regeneró o cambió el PDF: err=%v fuente=%d antes=%d", err, fuente.llamadas, antes)
	}
	suma := sha256.Sum256(contenido)
	if leido.HuellaSHA256 != hex.EncodeToString(suma[:]) {
		t.Fatal("huella no corresponde a bytes custodiados")
	}
}

func TestOriginalFirmableCTDeniegaLecturaAjenaPermisoCortadoYRefAlterada(t *testing.T) {
	s := solicitudOriginalCTPrueba()
	fuente := &fuenteOriginalCTPrueba{pdf: ports.PDFOriginalCT{TipoRef: "ref:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Contenido: []byte("%PDF-1.7\noriginal\n%%EOF")}}
	custodia := &custodiaOriginalCTPrueba{permitir: true, pertenece: s.ExpedienteRef}
	servicio, err := NuevoServicioOriginalFirmableCT(fuente, custodia)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Preparar(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	ajena := s
	ajena.ExpedienteRef = "ref:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	ajena.OriginalRef = ports.ReferenciaOriginalFirmableCT(ajena)
	if _, err := servicio.Leer(context.Background(), ajena); err == nil {
		t.Fatal("lectura ajena autorizada")
	}
	alterada := s
	alterada.OriginalRef = ajena.OriginalRef
	antes := custodia.lecturas
	if _, err := servicio.Leer(context.Background(), alterada); !errors.Is(err, ports.ErrOriginalFirmableCTInvalido) || custodia.lecturas != antes {
		t.Fatalf("ref ajena llegó a custodia: err=%v lecturas=%d", err, custodia.lecturas)
	}
	custodia.permitir = false
	if _, err := servicio.Leer(context.Background(), s); err == nil {
		t.Fatal("permiso revocado no cortó la lectura")
	}
}
