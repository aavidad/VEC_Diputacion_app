package httpinterno

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadRegistroFirmaVecPrueba struct {
	organizacion string
	err          error
}

func (a autoridadRegistroFirmaVecPrueba) ResolverOrganizacionFirmaVec(context.Context) (string, error) {
	return a.organizacion, a.err
}

type servicioRegistroFirmaVecPrueba struct {
	llamadas    int
	solicitudes []application.SolicitudFirmaVec
	respuesta   application.ResultadoFirmaVec
	err         error
}

func (s *servicioRegistroFirmaVecPrueba) Firmar(_ context.Context, sol application.SolicitudFirmaVec) (application.ResultadoFirmaVec, error) {
	s.llamadas++
	sol.PDFFirmado = append([]byte(nil), sol.PDFFirmado...)
	s.solicitudes = append(s.solicitudes, sol)
	r := s.respuesta
	if s.llamadas > 1 {
		r.Recibo.YaRegistrada = true
	}
	return r, s.err
}

func cuerpoRegistroFirmaVecPrueba() string {
	return strings.Replace(cuerpoRegistroFirmaExternaPrueba, `,"referencia_portafirmas_declarada":"PF-2026-0001","fecha_portafirmas_declarada":"2026-10-02T10:30:00Z"`, "", 1)
}

func peticionRegistroFirmaVec(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaRegistroFirmaVec, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func materialRegistroFirmaV2Prueba(via string, pdf []byte) ports.MaterialFirmaVerificadaV2 {
	base := resultadoRegistroFirmaExternaPrueba().Material
	base.Via = via
	if via == ports.ViaFirmaCertificadoVEC {
		base.ReferenciaPortafirmasDeclarada = ""
		base.FechaPortafirmasDeclarada = ""
	}
	base.PoliticaVerificacion = "politica:vec:firma:verificacion-autonoma:v2"
	huella := sha256.Sum256(pdf)
	base.FirmadoHuella = hex.EncodeToString(huella[:])
	evidencia := json.RawMessage(`[{}]`)
	huellaEvidencia := sha256.Sum256(evidencia)
	return ports.MaterialFirmaVerificadaV2{
		MaterialFirmaExterna: base, CatalogoVersion: 1,
		PerfilActivoOperadorRef: "perfil:operador:001", CuentaFirmanteRef: "cuenta:firmante:001",
		VinculoCredencialFirmanteRef: "vinculo:credencial:001", VinculoCredencialFirmanteRevision: 1,
		VinculoCredencialFirmanteHuella: strings.Repeat("a", 64), RolIDFirmante: base.CargoFirmante,
		EntradaDocumentoRef: base.OriginalRef, EntradaDocumentoVersion: base.OriginalVersion, EntradaDocumentoLongitud: 1,
		EntradaDocumentoHuella: base.OriginalHuella, OrdenFirmaPDF: 1, ByteRange: [4]uint64{0, 1, 2, uint64(len(pdf)) - 2},
		RevisionHuellaSHA256: base.FirmadoHuella, ContenidoFirmadoHuellaSHA256: strings.Repeat("c", 64), RevisionLongitud: uint64(len(pdf)),
		EvidenciaFirmasCanonica: evidencia, EvidenciaFirmasHuellaSHA256: hex.EncodeToString(huellaEvidencia[:]),
	}
}

func resultadoRegistroFirmaVecPrueba() application.ResultadoFirmaVec {
	extra := resultadoRegistroFirmaExternaPrueba()
	m := materialRegistroFirmaV2Prueba(ports.ViaFirmaCertificadoVEC, []byte("%PDF-1.7\n%%EOF"))
	extra.Recibo.SolicitudHuella, _ = m.HuellaSHA256()
	return application.ResultadoFirmaVec{MaterialMultiple: &m, Recibo: extra.Recibo, Custodiado: extra.Custodiado, MotivoVerificacion: extra.MotivoVerificacion}
}

func TestRegistroFirmaVecV2ReciboCustodiaYReplay(t *testing.T) {
	s := &servicioRegistroFirmaVecPrueba{respuesta: resultadoRegistroFirmaVecPrueba()}
	if err := s.respuesta.MaterialMultiple.Validar(); err != nil {
		t.Fatal(err)
	}
	h, err := NuevoManejadorRegistroFirmaVec(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	if err != nil {
		t.Fatal(err)
	}
	var anterior map[string]any
	for n, estado := range []int{http.StatusCreated, http.StatusOK} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba()))
		var salida struct {
			Data map[string]any `json:"data"`
		}
		if w.Code != estado || json.Unmarshal(w.Body.Bytes(), &salida) != nil {
			t.Fatalf("POST %d: %d %s", n, w.Code, w.Body)
		}
		d := salida.Data
		if d["esquema"] != EsquemaRegistroFirmaVec || d["firma_eficaz"] != false || d["material_root_sha256"] != s.respuesta.Recibo.SolicitudHuella ||
			d["ya_registrada"] != (n == 1) || d["verificacion_tecnica"].(map[string]any)["estado"] != "valida" ||
			d["revision_pdf"].(map[string]any)["orden_firma"] != float64(1) || strings.Contains(w.Body.String(), "per_firmante_principal") ||
			strings.Contains(w.Body.String(), "certificado_sha256") || strings.Contains(w.Body.String(), "procedencia_portafirmas") {
			t.Fatalf("respuesta V2: %s", w.Body)
		}
		if n == 1 {
			for _, campo := range []string{"recibo_ref", "firma_ref", "registrada_en", "material_root_sha256"} {
				if d[campo] != anterior[campo] {
					t.Fatalf("replay cambió %s", campo)
				}
			}
		}
		anterior = d
	}
	if s.llamadas != 2 || string(s.solicitudes[0].PDFFirmado) != "%PDF-1.7\n%%EOF" ||
		s.solicitudes[0].ClaveIdempotencia != s.solicitudes[1].ClaveIdempotencia || s.solicitudes[0].OrganizacionRef != "organizacion:desarrollo:dipgra" {
		t.Fatal("el transporte no conservó bytes, clave u organización del canal")
	}
}

func TestRegistroFirmaVecRechazaEntradaNoCanonicaYFronteras(t *testing.T) {
	s := &servicioRegistroFirmaVecPrueba{}
	h, _ := NuevoManejadorRegistroFirmaVec(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	body := cuerpoRegistroFirmaVecPrueba()
	for nombre, cuerpo := range map[string]string{
		"actor":             strings.Replace(body, `"documento":`, `"actor_ref":"actor:forjado","documento":`, 1),
		"organizacion":      strings.Replace(body, `"documento":`, `"organizacion_ref":"organizacion:forjada","documento":`, 1),
		"original bytes":    strings.Replace(body, `"documento":`, `"original_base64":"JVBERg==","documento":`, 1),
		"duplicado":         strings.Replace(body, `"documento":`, `"documento":"resolucion","documento":`, 1),
		"capitalizacion":    strings.Replace(body, `"documento":`, `"DOCUMENTO":`, 1),
		"sin clave":         strings.Replace(body, "clave-firma-externa-00001", "", 1),
		"base64 multilinea": strings.Replace(body, "JVBERi0xLjcKJSVFT0Y=", `JVBERi0x\nLjcKJSVFT0Y=`, 1),
		"PDF invalido":      strings.Replace(body, "JVBERi0xLjcKJSVFT0Y=", base64.StdEncoding.EncodeToString([]byte("texto")), 1),
		"trailing":          body + `{}`,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaVec(cuerpo))
		if w.Code < 400 {
			t.Errorf("%s aceptado", nombre)
		}
	}
	for nombre, alterar := range map[string]func(*http.Request){
		"cookie":    func(r *http.Request) { r.Header.Set("Cookie", "x=y") },
		"identidad": func(r *http.Request) { r.Header.Set("X-Organizacion", "organizacion:forjada") },
		"query":     func(r *http.Request) { r.URL.RawQuery = "x=1" },
		"alias": func(r *http.Request) {
			r.URL.RawPath = "/api/vec/contratacion-temporal/firmas-documento/%72egistro-vec"
		},
		"otro canal": func(r *http.Request) { r.URL.Path = RutaFirmaDocumento },
	} {
		r := peticionRegistroFirmaVec(body)
		alterar(r)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Errorf("%s aceptado", nombre)
		}
	}
	if s.llamadas != 0 {
		t.Fatal("entrada rechazada alcanzó aplicación")
	}
}

func TestRegistroFirmaVecDeniegaAutoridadYResultadoInyectado(t *testing.T) {
	for _, tc := range []struct {
		err    error
		estado int
	}{
		{ErrContextoCanalAusente, http.StatusUnauthorized}, {ErrContextoCanalCaducado, http.StatusUnauthorized},
		{ports.ErrAutorizacionDenegada, http.StatusForbidden}, {errors.New("dependencia"), http.StatusServiceUnavailable},
	} {
		s := &servicioRegistroFirmaVecPrueba{respuesta: resultadoRegistroFirmaVecPrueba()}
		h, _ := NuevoManejadorRegistroFirmaVec(autoridadRegistroFirmaVecPrueba{err: tc.err}, s)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba()))
		if w.Code != tc.estado || s.llamadas != 0 {
			t.Fatalf("autoridad: %d %s", w.Code, w.Body)
		}
	}
	for nombre, alterar := range map[string]func(*application.ResultadoFirmaVec){
		"V1 aislado": func(r *application.ResultadoFirmaVec) {
			r.Material = ports.MaterialFirmaVec(r.MaterialMultiple.MaterialFirmaExterna)
			r.MaterialMultiple = nil
		},
		"ambos contratos": func(r *application.ResultadoFirmaVec) {
			r.Material = ports.MaterialFirmaVec(r.MaterialMultiple.MaterialFirmaExterna)
		},
		"otra via":        func(r *application.ResultadoFirmaVec) { r.MaterialMultiple.Via = ports.ViaFirmaExternaPortafirmas },
		"otro expediente": func(r *application.ResultadoFirmaVec) { r.MaterialMultiple.ExpedienteRef = "expediente:otro" },
		"otro original":   func(r *application.ResultadoFirmaVec) { r.MaterialMultiple.OriginalRef = "documento:otro" },
		"custodia":        func(r *application.ResultadoFirmaVec) { r.Custodiado.HuellaSHA256 = strings.Repeat("a", 64) },
		"recibo":          func(r *application.ResultadoFirmaVec) { r.Recibo.SolicitudHuella = strings.Repeat("a", 64) },
		"longitud":        func(r *application.ResultadoFirmaVec) { r.MaterialMultiple.RevisionLongitud++ },
	} {
		r := resultadoRegistroFirmaVecPrueba()
		alterar(&r)
		s := &servicioRegistroFirmaVecPrueba{respuesta: r}
		h, _ := NuevoManejadorRegistroFirmaVec(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba()))
		if w.Code != http.StatusBadGateway {
			t.Fatalf("%s: %d %s", nombre, w.Code, w.Body)
		}
	}
}

func TestRegistroFirmaVecMapeaErroresDeAplicacion(t *testing.T) {
	for _, tc := range []struct {
		err    error
		estado int
	}{
		{ports.ErrFirmaDocumentoDenegada, http.StatusForbidden}, {ports.ErrOriginalFirmaNoAutorizado, http.StatusForbidden},
		{ports.ErrClaveFirmaDocumentoUsada, http.StatusConflict}, {ports.ErrCustodiaFirmadoEnConflicto, http.StatusConflict},
		{ports.ErrSolicitudFirmaDocumentoInvalida, http.StatusUnprocessableEntity},
		{application.ErrFirmaNoVerificada, http.StatusUnprocessableEntity}, {ports.ErrRegistroFirmaVecNoDisponible, http.StatusServiceUnavailable},
	} {
		s := &servicioRegistroFirmaVecPrueba{err: tc.err}
		h, _ := NuevoManejadorRegistroFirmaVec(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba()))
		if w.Code != tc.estado || !strings.Contains(w.Body.String(), "registro_firma_vec.error") {
			t.Fatalf("error: %d %s", w.Code, w.Body)
		}
	}
}

type cuerpoRegistroFirmaConError struct{}

func (cuerpoRegistroFirmaConError) Read([]byte) (int, error) { return 0, errors.New("lectura") }
func (cuerpoRegistroFirmaConError) Close() error             { return nil }

func TestRegistroFirmaVecLecturaFallidaYLimiteDelPuerto(t *testing.T) {
	s := &servicioRegistroFirmaVecPrueba{respuesta: resultadoRegistroFirmaVecPrueba()}
	h, _ := NuevoManejadorRegistroFirmaVec(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	r := peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba())
	r.Body = cuerpoRegistroFirmaConError{}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || s.llamadas != 0 {
		t.Fatalf("error lectura: %d %s", w.Code, w.Body)
	}
	pdf := []byte("%PDF-1.7\n" + strings.Repeat("x", ports.MaximoDocumentoFirmaBytes-len("%PDF-1.7\n%%EOF")) + "%%EOF")
	cuerpo := strings.Replace(cuerpoRegistroFirmaVecPrueba(), "JVBERi0xLjcKJSVFT0Y=", base64.StdEncoding.EncodeToString(pdf), 1)
	m := materialRegistroFirmaV2Prueba(ports.ViaFirmaCertificadoVEC, pdf)
	s.respuesta.MaterialMultiple = &m
	s.respuesta.Custodiado.HuellaSHA256 = m.FirmadoHuella
	s.respuesta.Recibo.SolicitudHuella, _ = m.HuellaSHA256()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaVec(cuerpo))
	if w.Code != http.StatusCreated || s.llamadas != 1 {
		t.Fatalf("máximo del puerto rechazado: %d %s", w.Code, w.Body)
	}
	cuerpo = strings.Replace(cuerpoRegistroFirmaVecPrueba(), "JVBERi0xLjcKJSVFT0Y=", base64.StdEncoding.EncodeToString(append(pdf, 'x')), 1)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaVec(cuerpo))
	if w.Code != http.StatusUnprocessableEntity || s.llamadas != 1 {
		t.Fatalf("puerto excedido: %d %s", w.Code, w.Body)
	}
}

type autoridadRegistroFirmaVecRevocable struct {
	llamadas int
}

func (a *autoridadRegistroFirmaVecRevocable) ResolverOrganizacionFirmaVec(context.Context) (string, error) {
	a.llamadas++
	if a.llamadas > 1 {
		return "", ports.ErrAutorizacionDenegada
	}
	return "organizacion:desarrollo:dipgra", nil
}

func TestRegistroFirmaVecReplayRevalidaCanal(t *testing.T) {
	a := &autoridadRegistroFirmaVecRevocable{}
	s := &servicioRegistroFirmaVecPrueba{respuesta: resultadoRegistroFirmaVecPrueba()}
	h, _ := NuevoManejadorRegistroFirmaVec(a, s)
	for _, estado := range []int{http.StatusCreated, http.StatusForbidden} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba()))
		if w.Code != estado {
			t.Fatalf("revalidación del canal: %d %s", w.Code, w.Body)
		}
	}
	if a.llamadas != 2 || s.llamadas != 1 {
		t.Fatal("replay reutilizó la autoridad anterior")
	}
}

type cuerpoRegistroFirmaErrorPropagable struct {
	err error
}

func (c cuerpoRegistroFirmaErrorPropagable) Read([]byte) (int, error) { return 0, c.err }
func (cuerpoRegistroFirmaErrorPropagable) Close() error               { return nil }

func TestRegistroFirmaPropagaErroresDeLecturaYBase64(t *testing.T) {
	privado := errors.New("lectura_certificado_privado")
	r := peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba())
	r.Body = cuerpoRegistroFirmaErrorPropagable{err: privado}
	contenido, err := leerCuerpoRegistroFirma(httptest.NewRecorder(), r)
	if contenido != nil || !errors.Is(err, privado) {
		t.Fatalf("se perdió la causa de lectura: %v", err)
	}
	firmado, err := decodificarPDFRegistroFirma("?")
	var corrupto base64.CorruptInputError
	if firmado != nil || !errors.As(err, &corrupto) {
		t.Fatalf("se perdió la causa de Base64: %v", err)
	}
	vec := &servicioRegistroFirmaVecPrueba{}
	externa := &servicioRegistroFirmaExternaPrueba{}
	hVec, err := NuevoManejadorRegistroFirmaVec(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, vec)
	if err != nil {
		t.Fatal(err)
	}
	hExterna, err := NuevoManejadorRegistroFirmaExternaV2(autoridadRegistroFirmaExternaPrueba{organizacion: "organizacion:desarrollo:dipgra"}, externa)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		h        http.Handler
		peticion func(string) *http.Request
		cuerpo   string
	}{
		{hVec, peticionRegistroFirmaVec, cuerpoRegistroFirmaVecPrueba()},
		{hExterna, peticionRegistroFirmaExterna, cuerpoRegistroFirmaExternaPrueba},
	} {
		r = caso.peticion(caso.cuerpo)
		r.Body = cuerpoRegistroFirmaErrorPropagable{err: privado}
		w := httptest.NewRecorder()
		caso.h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || strings.Contains(w.Body.String(), privado.Error()) ||
			!strings.Contains(w.Body.String(), `"codigo":"peticion_no_valida"`) {
			t.Fatalf("lectura sin normalizar: %d %s", w.Code, w.Body)
		}
		r = caso.peticion(strings.Replace(caso.cuerpo, "JVBERi0xLjcKJSVFT0Y=", "?", 1))
		w = httptest.NewRecorder()
		caso.h.ServeHTTP(w, r)
		if w.Code != http.StatusUnprocessableEntity || strings.Contains(w.Body.String(), "illegal base64") ||
			!strings.Contains(w.Body.String(), `"codigo":"contenido_no_valido"`) {
			t.Fatalf("Base64 sin normalizar: %d %s", w.Code, w.Body)
		}
	}
	if vec.llamadas != 0 || externa.llamadas != 0 {
		t.Fatal("una lectura o Base64 fallidos alcanzaron aplicación")
	}
}
