package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecapplication "vec-diputacion-granada/internal/vec/application"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type fuentePDFOriginalPuentePrueba struct{ llamadas int }

func (f *fuentePDFOriginalPuentePrueba) ObtenerPDFOriginalCT(context.Context, vecports.SolicitudOriginalFirmableCT) (vecports.PDFOriginalCT, error) {
	f.llamadas++
	return vecports.PDFOriginalCT{}, errors.New("no se debe renderizar durante Leer")
}

type custodiaOriginalPuentePrueba struct {
	original  vecports.OriginalFirmableCT
	err       error
	llamadas  int
	solicitud vecports.SolicitudOriginalFirmableCT
	ref       string
}

func (c *custodiaOriginalPuentePrueba) LeerOriginal(_ context.Context, s vecports.SolicitudOriginalFirmableCT, ref string) (vecports.OriginalFirmableCT, error) {
	c.llamadas++
	c.solicitud, c.ref = s, ref
	return c.original, c.err
}

func (*custodiaOriginalPuentePrueba) GuardarUnaVez(context.Context, vecports.SolicitudOriginalFirmableCT, vecports.PDFOriginalCT, string, string) (vecports.OriginalFirmableCT, error) {
	return vecports.OriginalFirmableCT{}, errors.New("no se debe escribir durante Leer")
}

type tiposOriginalPuentePrueba struct {
	ref      string
	err      error
	llamadas int
	tipo     ports.TipoBorradorRRHH
}

func (t *tiposOriginalPuentePrueba) ResolverTipoOriginalRRHH(_ context.Context, tipo ports.TipoBorradorRRHH) (string, error) {
	t.llamadas++
	t.tipo = tipo
	return t.ref, t.err
}

func (t *tiposOriginalPuentePrueba) ResolverTipoOriginalCT(ctx context.Context, documento string) (string, error) {
	return t.ResolverTipoOriginalRRHH(ctx, ports.TipoBorradorRRHH(documento))
}

func puenteOriginalFirmaPrueba(t *testing.T) (*FuenteOriginalFirmaDocumentos, ports.SolicitudOriginalFirma, *fuentePDFOriginalPuentePrueba, *custodiaOriginalPuentePrueba, *tiposOriginalPuentePrueba) {
	t.Helper()
	q := ports.SolicitudOriginalFirma{
		OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:prueba-original",
		Documento: string(ports.BorradorResolucion), OriginalVersion: 7,
	}
	q.OriginalRef = (almacencanonico.IdentidadOriginalCT{
		OrganizacionRef: q.OrganizacionRef, ExpedienteRef: q.ExpedienteRef,
		Documento: q.Documento, Version: q.OriginalVersion,
	}).Referencia()
	contenido := []byte("%PDF-1.7\noriginal autorizado")
	suma := sha256.Sum256(contenido)
	tipos := &tiposOriginalPuentePrueba{ref: "ref:" + strings.Repeat("b", 64)}
	custodia := &custodiaOriginalPuentePrueba{original: vecports.OriginalFirmableCT{
		Referencia: q.OriginalRef, Version: q.OriginalVersion, TipoRef: tipos.ref,
		HuellaSHA256: hex.EncodeToString(suma[:]), Contenido: contenido,
	}}
	fuentePDF := &fuentePDFOriginalPuentePrueba{}
	servicio, err := vecapplication.NuevoServicioOriginalFirmableCT(fuentePDF, custodia, tipos)
	if err != nil {
		t.Fatal(err)
	}
	puente, err := NuevaFuenteOriginalFirmaDocumentos(servicio, tipos)
	if err != nil {
		t.Fatal(err)
	}
	return puente, q, fuentePDF, custodia, tipos
}

func TestPuenteOriginalFirmaLeeCustodiaSinRenderizar(t *testing.T) {
	t.Parallel()
	puente, q, fuentePDF, custodia, tipos := puenteOriginalFirmaPrueba(t)
	original, err := puente.ObtenerOriginalFirma(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if original.Solicitud != q || original.HuellaSHA256 != custodia.original.HuellaSHA256 ||
		string(original.Contenido) != string(custodia.original.Contenido) ||
		custodia.llamadas != 1 || custodia.ref != q.OriginalRef ||
		custodia.solicitud.OrganizacionRef != q.OrganizacionRef ||
		custodia.solicitud.ExpedienteRef != q.ExpedienteRef ||
		custodia.solicitud.Documento != q.Documento ||
		custodia.solicitud.OriginalVersion != q.OriginalVersion ||
		tipos.llamadas != 2 || tipos.tipo != ports.BorradorResolucion || fuentePDF.llamadas != 0 {
		t.Fatal("la lectura no conservó identidad, tipo o contenido original")
	}
	custodia.original.Contenido[0] = 'X'
	if original.Contenido[0] != '%' {
		t.Fatal("se expuso el buffer mutable de la custodia")
	}
}

func TestPuenteOriginalFirmaRechazaIdentidadAjenaAntesDeLeer(t *testing.T) {
	t.Parallel()
	for _, cambiar := range []func(*ports.SolicitudOriginalFirma){
		func(q *ports.SolicitudOriginalFirma) { q.OrganizacionRef = "organizacion:otra" },
		func(q *ports.SolicitudOriginalFirma) { q.ExpedienteRef = "expediente:ct:otro" },
		func(q *ports.SolicitudOriginalFirma) { q.Documento = "diligencia" },
		func(q *ports.SolicitudOriginalFirma) { q.OriginalVersion++ },
		func(q *ports.SolicitudOriginalFirma) { q.OriginalRef = "ref:" + strings.Repeat("a", 64) },
	} {
		puente, q, fuentePDF, custodia, tipos := puenteOriginalFirmaPrueba(t)
		cambiar(&q)
		if _, err := puente.ObtenerOriginalFirma(context.Background(), q); !errors.Is(err, ports.ErrOriginalFirmaNoAutorizado) ||
			custodia.llamadas != 0 || tipos.llamadas != 0 || fuentePDF.llamadas != 0 {
			t.Fatal("identidad ajena llegó a la custodia")
		}
	}
}

func TestPuenteOriginalFirmaCierraDenegacionTipoYHuella(t *testing.T) {
	t.Parallel()
	puente, q, fuentePDF, custodia, tipos := puenteOriginalFirmaPrueba(t)
	custodia.err = docports.ErrAccesoDenegado
	if _, err := puente.ObtenerOriginalFirma(context.Background(), q); !errors.Is(err, ports.ErrOriginalFirmaNoAutorizado) || fuentePDF.llamadas != 0 {
		t.Fatal("la denegación permitió obtener el original")
	}
	custodia.err = nil
	custodia.original.TipoRef = "ref:" + strings.Repeat("c", 64)
	if _, err := puente.ObtenerOriginalFirma(context.Background(), q); !errors.Is(err, ports.ErrFuenteOriginalFirmaNoDisponible) || fuentePDF.llamadas != 0 {
		t.Fatal("se aceptó tipo documental ajeno o se regeneró el original")
	}
	custodia.original.TipoRef = tipos.ref
	custodia.original.HuellaSHA256 = strings.Repeat("d", 64)
	if _, err := puente.ObtenerOriginalFirma(context.Background(), q); err == nil {
		t.Fatal("se aceptaron bytes con otra huella")
	}
	custodia.original.HuellaSHA256 = hex.EncodeToString(sha256Sum(custodia.original.Contenido))
	tipos.err = errors.New("catálogo no disponible")
	llamadas := custodia.llamadas
	if _, err := puente.ObtenerOriginalFirma(context.Background(), q); !errors.Is(err, ports.ErrFuenteOriginalFirmaNoDisponible) || custodia.llamadas != llamadas {
		t.Fatal("se leyó custodia sin tipo gobernado")
	}
}

func sha256Sum(b []byte) []byte {
	suma := sha256.Sum256(b)
	return suma[:]
}
