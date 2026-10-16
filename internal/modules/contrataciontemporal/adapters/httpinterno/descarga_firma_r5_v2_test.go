package httpinterno

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type selectorHTTPDescarga struct {
	salida   consultafirmasv2.SeleccionDescarga
	llamadas int
}

func (s *selectorHTTPDescarga) SeleccionarConsultaDescarga(_ context.Context, _ consultafirmasv2.SolicitudDescarga) (consultafirmasv2.SeleccionDescarga, error) {
	s.llamadas++
	return s.salida, nil
}

type consultaHTTPDescarga struct {
	salida   consultafirmasv2.Resultado
	llamadas int
}

func (c *consultaHTTPDescarga) Consultar(_ context.Context, _ consultafirmasv2.Solicitud) (consultafirmasv2.Resultado, error) {
	c.llamadas++
	return c.salida, nil
}

type pdfHTTPDescarga struct {
	cuerpo   []byte
	llamadas int
}

func (p *pdfHTTPDescarga) ObtenerPDFFirmaAnterior(_ context.Context, q ports.SolicitudPDFFirmaAnterior) (ports.PDFFirmaAnterior, error) {
	p.llamadas++
	return ports.PDFFirmaAnterior{Solicitud: q, Contenido: bytes.Clone(p.cuerpo)}, nil
}

type garantiaHTTPDescarga struct{ valor string }

func (g *garantiaHTTPDescarga) GarantiaAutenticacionFirmaV2(context.Context) (string, error) {
	return g.valor, nil
}

func manejadorHTTPDescargaPrueba(t *testing.T) (http.Handler, *selectorHTTPDescarga, *consultaHTTPDescarga, *pdfHTTPDescarga, *garantiaHTTPDescarga, *intentosConsultaV2Prueba) {
	t.Helper()
	pdf := []byte("%PDF-1.4\nrevision-verificada")
	h := sha256.Sum256(pdf)
	huella := hex.EncodeToString(h[:])
	selector := &selectorHTTPDescarga{salida: consultafirmasv2.SeleccionDescarga{OrganizacionRef: "organizacion:central",
		Consulta: consultafirmasv2.Solicitud{ExpedienteRef: "expediente:prueba", VersionExpediente: 7,
			Documento: "resolucion", PasoOrden: 1, ClaveIdempotencia: "consulta-descarga-000001",
			CatalogoHuella: strings.Repeat("a", 64), Via: ports.ViaFirmaCertificadoVEC}}}
	consulta := &consultaHTTPDescarga{salida: consultafirmasv2.Resultado{ExpedienteRef: "expediente:prueba",
		Documento: "resolucion", VersionExpediente: 7,
		ExpedienteDocumentalRef: ports.ExpedienteDocumentalRef("organizacion:central", "expediente:prueba"),
		Firmas: []consultafirmasv2.Firma{{FirmaRef: "firma:ultima", ReciboRef: "recibo:firma:ultima",
			Documento: "resolucion", VersionExpediente: 6, Resultado: domain.ResultadoFirmaFirmado,
			Custodiado:  &consultafirmasv2.Documento{Ref: "documento:firmado:ultimo", Version: 3, SHA256: huella},
			RevisionPDF: &consultafirmasv2.RevisionPDF{RevisionSHA256: huella, RevisionLongitud: uint64(len(pdf))}}}}}
	almacen := &pdfHTTPDescarga{cuerpo: pdf}
	servicio, err := consultafirmasv2.NuevaDescarga(selector, consulta, almacen)
	if err != nil {
		t.Fatal(err)
	}
	garantia := &garantiaHTTPDescarga{valor: "sustancial"}
	intentos := &intentosConsultaV2Prueba{t: t}
	fabrica := &fabricaConsultaV2Prueba{t: t, registro: &registroConsultaV2Prueba{}}
	httpHandler, err := NuevoManejadorDescargaFirmaR5V2(servicio, intentos, fabrica, garantia)
	if err != nil {
		t.Fatal(err)
	}
	return httpHandler, selector, consulta, almacen, garantia, intentos
}

func peticionHTTPDescargaPrueba(t *testing.T, cuerpo string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, RutaDescargaFirmaR5V2, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	ctx, err := vp.ConCorrelacionIncidenciasPeticion(r.Context())
	if err != nil {
		t.Fatal(err)
	}
	return r.WithContext(ctx)
}

const cuerpoDescargaFirmaPrueba = `{"expediente_ref":"expediente:prueba","version_observada":7,"documento":"resolucion","firma_ref":"firma:ultima"}`

func TestHTTPDescargaDevuelveBytesExactosYReciboDeFirmaExistente(t *testing.T) {
	h, selector, consulta, pdf, _, intentos := manejadorHTTPDescargaPrueba(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionHTTPDescargaPrueba(t, cuerpoDescargaFirmaPrueba))
	if w.Code != http.StatusOK || selector.llamadas != 1 || consulta.llamadas != 1 || pdf.llamadas != 1 || intentos.llamadas != 0 {
		t.Fatalf("la lectura nominal no terminó: estado=%d", w.Code)
	}
	var respuesta struct {
		Data descargaFirmaR5V2JSON `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &respuesta) != nil || respuesta.Data.Esquema != EsquemaDescargaFirmaR5V2 ||
		respuesta.Data.FirmaRef != "firma:ultima" || respuesta.Data.ReciboRef != "recibo:firma:ultima" ||
		respuesta.Data.DocumentoRef != "documento:firmado:ultimo" || respuesta.Data.DocumentoVersion != 3 {
		t.Fatal("respuesta de descarga o recibo de firma incorrectos")
	}
	contenido, err := base64.StdEncoding.DecodeString(respuesta.Data.PDFBase64)
	if err != nil || !bytes.Equal(contenido, pdf.cuerpo) {
		t.Fatal("los bytes entregados no son los custodiados")
	}
}

func TestHTTPDescargaConservaDosMiBYBloqueaRevisionExcesiva(t *testing.T) {
	h, _, consulta, pdf, _, _ := manejadorHTTPDescargaPrueba(t)
	pdf.cuerpo = append([]byte("%PDF-"), bytes.Repeat([]byte("x"), ports.MaximoDocumentoFirmaBytes-10)...)
	pdf.cuerpo = append(pdf.cuerpo, []byte("%%EOF")...)
	suma := sha256.Sum256(pdf.cuerpo)
	huella := hex.EncodeToString(suma[:])
	consulta.salida.Firmas[0].Custodiado.SHA256 = huella
	consulta.salida.Firmas[0].RevisionPDF.RevisionSHA256 = huella
	consulta.salida.Firmas[0].RevisionPDF.RevisionLongitud = uint64(len(pdf.cuerpo))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionHTTPDescargaPrueba(t, cuerpoDescargaFirmaPrueba))
	var salida struct {
		Data descargaFirmaR5V2JSON `json:"data"`
	}
	if w.Code != http.StatusOK || len(w.Body.Bytes()) <= 2<<20 || json.Unmarshal(w.Body.Bytes(), &salida) != nil {
		t.Fatalf("PDF de 2 MiB no llegó en su envoltorio: estado=%d", w.Code)
	}
	decodificado, err := base64.StdEncoding.DecodeString(salida.Data.PDFBase64)
	if err != nil || !bytes.Equal(decodificado, pdf.cuerpo) {
		t.Fatal("descarga cambió los bytes custodiados")
	}
	pdf.cuerpo = append(pdf.cuerpo, 'x')
	consulta.salida.Firmas[0].RevisionPDF.RevisionLongitud = uint64(len(pdf.cuerpo))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionHTTPDescargaPrueba(t, cuerpoDescargaFirmaPrueba))
	if w.Code == http.StatusOK || pdf.llamadas != 1 {
		t.Fatalf("revisión excesiva alcanzó el objeto: estado=%d lecturas=%d", w.Code, pdf.llamadas)
	}
}

func TestHTTPDescargaRechazaCamposCustodiaDuplicadosYGarantiaInsuficiente(t *testing.T) {
	casos := []struct {
		nombre   string
		cuerpo   string
		garantia string
	}{
		{"custodia_libre", strings.TrimSuffix(cuerpoDescargaFirmaPrueba, "}") + `,"documento_ref":"ajeno"}`, "sustancial"},
		{"firma_duplicada", strings.TrimSuffix(cuerpoDescargaFirmaPrueba, "}") + `,"firma_ref":"firma:otra"}`, "sustancial"},
		{"garantia_baja", cuerpoDescargaFirmaPrueba, "basica"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			h, selector, consulta, pdf, garantia, intentos := manejadorHTTPDescargaPrueba(t)
			garantia.valor = caso.garantia
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionHTTPDescargaPrueba(t, caso.cuerpo))
			if w.Code == http.StatusOK || selector.llamadas != 0 || consulta.llamadas != 0 || pdf.llamadas != 0 || intentos.llamadas != 1 {
				t.Fatalf("la entrada insegura llegó al PDF: estado=%d intentos=%d", w.Code, intentos.llamadas)
			}
		})
	}
}
