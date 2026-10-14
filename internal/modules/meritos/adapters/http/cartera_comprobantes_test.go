package meritoshttp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/meritos/ports"
)

type proveedorComprobantePrueba struct{ err error }

func (p proveedorComprobantePrueba) PrepararComprobanteCartera(*http.Request) (IdentidadComprobanteCartera, error) {
	return IdentidadComprobanteCartera{}, p.err
}

type cargadorComprobantePrueba struct{ llamadas int }

func (c *cargadorComprobantePrueba) SubirComprobante(context.Context, IdentidadComprobanteCartera, string, []byte) (ResultadoCargaComprobanteCartera, error) {
	c.llamadas++
	return ResultadoCargaComprobanteCartera{}, nil
}

func (c *cargadorComprobantePrueba) ConsultarEstadoComprobante(context.Context, IdentidadComprobanteCartera, string) (EstadoComprobanteCartera, error) {
	c.llamadas++
	return EstadoComprobanteCartera{}, nil
}

func (c *cargadorComprobantePrueba) AbrirOriginalComprobante(context.Context, IdentidadComprobanteCartera, string) (io.ReadCloser, error) {
	c.llamadas++
	return nil, nil
}

type lectorEspiaComprobante struct{ lecturas int }

func (l *lectorEspiaComprobante) Read([]byte) (int, error) {
	l.lecturas++
	return 0, io.EOF
}
func (l *lectorEspiaComprobante) Close() error { return nil }

func TestComprobanteDeniegaAntesDeLeerArchivo(t *testing.T) {
	cargador := &cargadorComprobantePrueba{}
	h, err := NuevoHandlerComprobantesCartera(proveedorComprobantePrueba{err: ports.ErrCarteraSinSesion}, cargador)
	if err != nil {
		t.Fatal(err)
	}
	espia := &lectorEspiaComprobante{}
	r := httptest.NewRequest(http.MethodPost, RutaComprobantesCartera, nil)
	r.Body = espia
	r.Header.Set("Content-Type", "multipart/form-data; boundary=irrelevante")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || espia.lecturas != 0 || cargador.llamadas != 0 {
		t.Fatalf("sin autoridad: estado=%d lecturas=%d llamadas=%d", w.Code, espia.lecturas, cargador.llamadas)
	}
}

func TestComprobanteDesconexionNoLeeNiCarga(t *testing.T) {
	cargador := &cargadorComprobantePrueba{}
	h, err := NuevoHandlerComprobantesCartera(proveedorComprobantePrueba{}, cargador)
	if err != nil {
		t.Fatal(err)
	}
	espia := &lectorEspiaComprobante{}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	r := httptest.NewRequest(http.MethodPost, RutaComprobantesCartera, nil).WithContext(ctx)
	r.Body = espia
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || espia.lecturas != 0 || cargador.llamadas != 0 {
		t.Fatalf("desconexión: estado=%d lecturas=%d llamadas=%d", w.Code, espia.lecturas, cargador.llamadas)
	}
}

func TestMultipartComprobanteLimitesYCamposCerrados(t *testing.T) {
	casos := []struct {
		nombre string
		partes []struct{ clave, fichero, valor string }
		valido bool
	}{
		{"valido", []struct{ clave, fichero, valor string }{{"clave_idempotencia", "", "ref:declaracion"}, {"documento", "titulo.pdf", "%PDF-1.4\ncontenido"}}, true},
		{"campo_extra", []struct{ clave, fichero, valor string }{{"clave_idempotencia", "", "ref:declaracion"}, {"documento", "titulo.pdf", "%PDF-1.4\ncontenido"}, {"persona_ref", "", "ref:otra"}}, false},
		{"clave_duplicada", []struct{ clave, fichero, valor string }{{"clave_idempotencia", "", "ref:declaracion"}, {"clave_idempotencia", "", "ref:otra"}, {"documento", "titulo.pdf", "%PDF-1.4\ncontenido"}}, false},
		{"archivo_vacio", []struct{ clave, fichero, valor string }{{"clave_idempotencia", "", "ref:declaracion"}, {"documento", "titulo.pdf", ""}}, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var cuerpo bytes.Buffer
			multipartWriter := multipart.NewWriter(&cuerpo)
			for _, parte := range caso.partes {
				var destino io.Writer
				var err error
				if parte.fichero == "" {
					destino, err = multipartWriter.CreateFormField(parte.clave)
				} else {
					destino, err = multipartWriter.CreateFormFile(parte.clave, parte.fichero)
				}
				if err != nil {
					t.Fatal(err)
				}
				_, _ = io.WriteString(destino, parte.valor)
			}
			if err := multipartWriter.Close(); err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, RutaComprobantesCartera, &cuerpo)
			r.Header.Set("Content-Type", multipartWriter.FormDataContentType())
			clave, archivo, err := leerMultipartComprobante(httptest.NewRecorder(), r)
			if (err == nil) != caso.valido || caso.valido && (clave != "ref:declaracion" || !bytes.HasPrefix(archivo, []byte("%PDF"))) {
				t.Fatalf("validación multipart: clave=%q bytes=%d error=%v", clave, len(archivo), err)
			}
		})
	}
}

func TestMultipartComprobanteRechazaMasDeDieciseisMiB(t *testing.T) {
	var cuerpo bytes.Buffer
	escritor := multipart.NewWriter(&cuerpo)
	clave, err := escritor.CreateFormField("clave_idempotencia")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(clave, "ref:declaracion")
	archivo, err := escritor.CreateFormFile("documento", "titulo.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.CopyN(archivo, strings.NewReader(strings.Repeat("x", maximoComprobanteCartera+1)), maximoComprobanteCartera+1)
	if err != nil {
		t.Fatal(err)
	}
	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaComprobantesCartera, &cuerpo)
	r.Header.Set("Content-Type", escritor.FormDataContentType())
	if _, _, err := leerMultipartComprobante(httptest.NewRecorder(), r); err == nil {
		t.Fatal("aceptó un PDF de más de 16 MiB")
	}
}

func TestComprobanteRutaOriginalSoloReferenciaOpaca(t *testing.T) {
	validas := []string{
		RutaComprobantesCartera + "/ref%3Adocumento_1/original",
		RutaComprobantesCartera + "/ref:documento_1/original",
	}
	for _, ruta := range validas {
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
		if ref, ok := referenciaOriginalComprobante(r.URL); !ok || ref != "ref:documento_1" {
			t.Fatalf("ruta válida %q: %q, %t", ruta, ref, ok)
		}
	}
	for _, ruta := range []string{
		RutaComprobantesCartera + "/ref%252Fajeno/original",
		RutaComprobantesCartera + "/ref%3Auno%2Fdos/original",
		RutaComprobantesCartera + "/https:%2F%2Fajeno/original",
	} {
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
		if ref, ok := referenciaOriginalComprobante(r.URL); ok {
			t.Fatalf("ruta aceptada %q: %q", ruta, ref)
		}
	}
}

func TestConsultaEstadoSoloAceptaClaveDeDeclaracion(t *testing.T) {
	clave, err := claveEstadoComprobante("clave_idempotencia=declaracion%3A123")
	if err != nil || clave != "declaracion:123" {
		t.Fatalf("clave propia: %q %v", clave, err)
	}
	for _, query := range []string{
		"", "clave_idempotencia=", "clave_idempotencia=uno&clave_idempotencia=dos",
		"clave_idempotencia=uno&persona_ref=otra", "carga_id=ref%3Aajena",
	} {
		if _, err := claveEstadoComprobante(query); err == nil {
			t.Fatalf("aceptó selector ajeno %q", query)
		}
	}
	pendiente := EstadoComprobanteCartera{CargaID: "ref:carga", Estado: "recibido"}
	if !estadoComprobanteCarteraValido(pendiente) {
		t.Fatal("rechazó estado pendiente sin documento")
	}
	pendiente.ComprobanteRef = "ref:documento"
	if estadoComprobanteCarteraValido(pendiente) {
		t.Fatal("atribuyó un documento a una carga pendiente")
	}
	rechazado := EstadoComprobanteCartera{CargaID: "ref:carga", Estado: "rechazado",
		CodigoRechazo: "contenido_no_permitido"}
	if !estadoComprobanteCarteraValido(rechazado) {
		t.Fatal("rechazó un código genérico")
	}
	rechazado.CodigoRechazo = "firma_clam_secreta"
	if estadoComprobanteCarteraValido(rechazado) {
		t.Fatal("expuso detalle del analizador")
	}
}

func TestComprobanteNoConfirmaSinServicio(t *testing.T) {
	if _, err := NuevoHandlerComprobantesCartera(proveedorComprobantePrueba{}, nil); !errors.Is(err, ports.ErrComprobanteMeritoNoDisponible) {
		t.Fatal(err)
	}
	cargador := &cargadorComprobantePrueba{}
	h, _ := NuevoHandlerComprobantesCartera(proveedorComprobantePrueba{}, cargador)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaComprobantesCartera+"/ref%3Adoc/original", nil))
	if w.Code != http.StatusServiceUnavailable || cargador.llamadas != 0 || strings.Contains(w.Body.String(), "admitido") {
		t.Fatalf("servicio sin identidad válida: %d %s", w.Code, w.Body.String())
	}
}
