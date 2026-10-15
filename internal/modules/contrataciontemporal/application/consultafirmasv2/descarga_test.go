package consultafirmasv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type selectorDescargaPrueba struct {
	salida   SeleccionDescarga
	err      error
	llamadas int
}

func (s *selectorDescargaPrueba) SeleccionarConsultaDescarga(_ context.Context, _ SolicitudDescarga) (SeleccionDescarga, error) {
	s.llamadas++
	return s.salida, s.err
}

type consultaDescargaPrueba struct {
	salida   Resultado
	err      error
	llamadas int
}

func (s *consultaDescargaPrueba) Consultar(_ context.Context, _ Solicitud) (Resultado, error) {
	s.llamadas++
	return s.salida, s.err
}

type pdfDescargaPrueba struct {
	contenido []byte
	err       error
	llamadas  int
	ultima    ports.SolicitudPDFFirmaAnterior
}

func (p *pdfDescargaPrueba) ObtenerPDFFirmaAnterior(_ context.Context, q ports.SolicitudPDFFirmaAnterior) (ports.PDFFirmaAnterior, error) {
	p.llamadas++
	p.ultima = q
	return ports.PDFFirmaAnterior{Solicitud: q, Contenido: bytes.Clone(p.contenido)}, p.err
}

func escenarioDescargaPrueba(t *testing.T) (SolicitudDescarga, *selectorDescargaPrueba, *consultaDescargaPrueba, *pdfDescargaPrueba, *ServicioDescarga, []byte) {
	t.Helper()
	q := SolicitudDescarga{ExpedienteRef: "expediente:ct:prueba", VersionObservada: 7,
		Documento: "resolucion", FirmaRef: "firma:ultima:prueba"}
	seleccion := &selectorDescargaPrueba{salida: SeleccionDescarga{OrganizacionRef: "organizacion:central",
		Consulta: Solicitud{ExpedienteRef: q.ExpedienteRef, VersionExpediente: q.VersionObservada,
			Documento: q.Documento, PasoOrden: 1, ClaveIdempotencia: "consulta-descarga-000001",
			CatalogoHuella: strings.Repeat("a", 64), Via: ports.ViaFirmaCertificadoVEC}}}
	pdf := []byte("%PDF-1.4\nrevision-firmada")
	h := sha256.Sum256(pdf)
	huella := hex.EncodeToString(h[:])
	consulta := &consultaDescargaPrueba{salida: Resultado{ExpedienteRef: q.ExpedienteRef,
		VersionExpediente: q.VersionObservada, Documento: q.Documento,
		ExpedienteDocumentalRef: ports.ExpedienteDocumentalRef(seleccion.salida.OrganizacionRef, q.ExpedienteRef),
		Firmas: []Firma{{FirmaRef: q.FirmaRef, ReciboRef: "recibo:firma:ultima", Documento: q.Documento,
			VersionExpediente: 6, PasoOrden: 3, Resultado: domain.ResultadoFirmaFirmado,
			Custodiado:  &Documento{Ref: "documento:firmado:ultimo", Version: 3, SHA256: huella},
			RevisionPDF: &RevisionPDF{RevisionSHA256: huella, RevisionLongitud: uint64(len(pdf))}}}}}
	almacen := &pdfDescargaPrueba{contenido: pdf}
	servicio, err := NuevaDescarga(seleccion, consulta, almacen)
	if err != nil {
		t.Fatal(err)
	}
	return q, seleccion, consulta, almacen, servicio, pdf
}

func TestDescargaNominalUsaHistoriaAutorizadaYBytesCustodiados(t *testing.T) {
	q, selector, consulta, almacen, servicio, pdf := escenarioDescargaPrueba(t)
	r, err := servicio.Descargar(t.Context(), q)
	if err != nil || !bytes.Equal(r.PDF, pdf) || r.FirmaRef != q.FirmaRef ||
		r.ReciboRef != "recibo:firma:ultima" || r.HuellaSHA256 != almacen.ultima.DocumentoHuella ||
		almacen.ultima.DocumentoRef != "documento:firmado:ultimo" || almacen.ultima.DocumentoVersion != 3 ||
		selector.llamadas != 1 || consulta.llamadas != 1 || almacen.llamadas != 1 {
		t.Fatalf("la descarga no quedó ligada a la firma consultada: err=%v", err)
	}
	// Quien consulta el paso 1 puede descargar la última revisión del paso 3
	// si la consulta R5 de su propio perfil devolvió esa firma.
	if selector.salida.Consulta.PasoOrden != 1 || consulta.salida.Firmas[0].PasoOrden != 3 {
		t.Fatal("el escenario perdió la distinción entre paso propio y firma pedida")
	}
}

func TestDescargaRechazaFirmaAjenaVersionYHuellaSinLeerOtroObjeto(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*SolicitudDescarga, *selectorDescargaPrueba, *consultaDescargaPrueba, *pdfDescargaPrueba)
		causa  error
		leePDF bool
	}{
		{"firma_ajena", func(q *SolicitudDescarga, _ *selectorDescargaPrueba, _ *consultaDescargaPrueba, _ *pdfDescargaPrueba) {
			q.FirmaRef = "firma:ajena:prueba"
		}, ports.ErrFirmaDocumentoDenegada, false},
		{"version_observada_ajena", func(_ *SolicitudDescarga, _ *selectorDescargaPrueba, c *consultaDescargaPrueba, _ *pdfDescargaPrueba) {
			c.salida.VersionExpediente = 8
		}, ports.ErrResultadoFirmaDocumentoInvalido, false},
		{"custodia_sin_huella_coherente", func(_ *SolicitudDescarga, _ *selectorDescargaPrueba, c *consultaDescargaPrueba, _ *pdfDescargaPrueba) {
			c.salida.Firmas[0].RevisionPDF.RevisionSHA256 = strings.Repeat("b", 64)
		}, ports.ErrResultadoFirmaDocumentoInvalido, false},
		{"bytes_distintos", func(_ *SolicitudDescarga, _ *selectorDescargaPrueba, _ *consultaDescargaPrueba, p *pdfDescargaPrueba) {
			p.contenido = []byte("%PDF-1.4\nrevision-marcada")
		}, ports.ErrResultadoFirmaDocumentoInvalido, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			q, _, _, almacen, servicio, _ := escenarioDescargaPrueba(t)
			selector := servicio.selector.(*selectorDescargaPrueba)
			consulta := servicio.consulta.(*consultaDescargaPrueba)
			caso.mutar(&q, selector, consulta, almacen)
			r, err := servicio.Descargar(t.Context(), q)
			if !errors.Is(err, caso.causa) || r.PDF != nil || (almacen.llamadas == 1) != caso.leePDF {
				t.Fatalf("lectura insegura: err=%v llamadas_pdf=%d", err, almacen.llamadas)
			}
		})
	}
}

func TestDescargaDeniegaPerfilRetiradoAntesDeConsultaYDocumento(t *testing.T) {
	q, selector, consulta, almacen, servicio, _ := escenarioDescargaPrueba(t)
	selector.err = ports.ErrFirmaDocumentoDenegada
	if _, err := servicio.Descargar(t.Context(), q); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) ||
		selector.llamadas != 1 || consulta.llamadas != 0 || almacen.llamadas != 0 {
		t.Fatalf("el perfil retirado llegó a lectores: err=%v", err)
	}
}
