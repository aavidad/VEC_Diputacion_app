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
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

type autoridadRegistroFirmaExternaPrueba struct {
	organizacion string
	err          error
}

func (a autoridadRegistroFirmaExternaPrueba) ResolverOrganizacionFirmaExterna(context.Context) (string, error) {
	return a.organizacion, a.err
}

type servicioRegistroFirmaExternaPrueba struct {
	llamadas  int
	solicitud application.SolicitudFirmaExterna
	respuesta application.ResultadoFirmaExterna
	err       error
}

func (s *servicioRegistroFirmaExternaPrueba) Registrar(_ context.Context, solicitud application.SolicitudFirmaExterna) (application.ResultadoFirmaExterna, error) {
	s.llamadas++
	s.solicitud = solicitud
	s.solicitud.PDFFirmado = append([]byte(nil), solicitud.PDFFirmado...)
	r := s.respuesta
	if s.llamadas > 1 {
		r.Recibo.YaRegistrada = true
	}
	return r, s.err
}

const cuerpoRegistroFirmaExternaPrueba = `{"expediente_ref":"expediente:ct:001","version_expediente":7,"documento":"resolucion","paso_orden":1,"original_ref":"documento:original:001","original_version":2,"firmado_base64":"JVBERi0xLjcKJSVFT0Y=","referencia_portafirmas_declarada":"PF-2026-0001","fecha_portafirmas_declarada":"2026-10-02T10:30:00Z","clave_idempotencia":"clave-firma-externa-00001"}`

func peticionRegistroFirmaExterna(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaRegistroFirmaExterna, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func resultadoRegistroFirmaExternaPrueba() application.ResultadoFirmaExterna {
	huellaPDF := sha256.Sum256([]byte("%PDF-1.7\n%%EOF"))
	m := ports.MaterialFirmaExterna{
		Via: ports.ViaFirmaExternaPortafirmas, OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001",
		VersionExpediente: 7, Documento: "resolucion", CatalogoRef: "catalogo:firma:001", CatalogoHuella: strings.Repeat("a", 64),
		PasoRef: "paso:firma:001", PasoOrden: 1, Secuencia: 1,
		HistoriaRevision: 0, HistoriaHuella: strings.Repeat("f", 64),
		OriginalRef: "documento:original:001", OriginalVersion: 2,
		OriginalHuella: strings.Repeat("b", 64), FirmadoHuella: hex.EncodeToString(huellaPDF[:]), CertificadoHuella: strings.Repeat("d", 64),
		FirmanteRef: "ref:" + strings.Repeat("d", 64), FirmantePrincipalRef: "per_firmante_principal_001", PerfilFirmanteRef: "perfil:firmante:001", CargoFirmante: "Órgano competente",
		UnidadFirmanteRef: "unidad:firmante:001", PerfilActivoFirmanteRef: "perfil:activo:001",
		AsignacionFirmanteRef: "asignacion:001", AsignacionFirmanteVersion: 1,
		AsignacionFirmanteHuella: strings.Repeat("e", 64), AsignacionVigenteDesde: "2026-01-01T00:00:00Z", AsignacionVigenteHasta: "2027-01-01T00:00:00Z",
		VersionRolFirmanteRef: "rol:version:001", VersionRolFirmanteHuella: strings.Repeat("1", 64),
		ControlVigenciaFirmanteRef: "rol:version:001", ControlVigenciaFirmanteRevision: 1, ControlVigenciaFirmanteHuella: strings.Repeat("2", 64),
		PoliticaVerificacion: ports.PoliticaVerificacionFirma,
		RevocacionEstado:     "vigente", SelloTiempoEstado: "valido", ReferenciaPortafirmasDeclarada: "PF-2026-0001",
		FechaPortafirmasDeclarada: "2026-10-02T10:30:00Z", ClaveIdempotencia: "clave-firma-externa-00001",
		DocumentoCustodiaRef: "documento:custodiado:001", DocumentoCustodiaVersion: 1,
	}
	return application.ResultadoFirmaExterna{
		Material: m, MotivoVerificacion: docports.MotivoFirmaVerificada,
		Custodiado: ports.DocumentoCustodiado{Ref: m.DocumentoCustodiaRef, Version: 1, HuellaSHA256: m.FirmadoHuella},
		Recibo: ports.ReciboFirmaDocumento{FirmaRef: "firma:externa:001", ReciboRef: "recibo:firma:externa:001", Secuencia: 1,
			Resultado: domain.ResultadoFirmaFirmado, ExpedienteVersion: 7, RegistradaEn: time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC),
			DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion},
	}
}

func TestRegistroFirmaExternaDistingueVerificacionDeclaracionYReplay(t *testing.T) {
	s := &servicioRegistroFirmaExternaPrueba{respuesta: resultadoRegistroFirmaExternaPrueba()}
	if err := s.respuesta.Material.Validar(); err != nil {
		t.Fatalf("fixture inválido: %v", err)
	}
	h, err := NuevoManejadorRegistroFirmaExterna(autoridadRegistroFirmaExternaPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	if err != nil {
		t.Fatal(err)
	}
	for n, estado := range []int{http.StatusCreated, http.StatusOK} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba))
		var salida struct {
			Data map[string]any `json:"data"`
		}
		if w.Code != estado || json.Unmarshal(w.Body.Bytes(), &salida) != nil {
			t.Fatalf("POST %d: %d %s", n, w.Code, w.Body)
		}
		if salida.Data["firma_eficaz"] != false || salida.Data["recibo_ref"] != "recibo:firma:externa:001" ||
			salida.Data["ya_registrada"] != (n == 1) || salida.Data["verificacion_tecnica"].(map[string]any)["estado"] != "valida" ||
			salida.Data["procedencia_portafirmas"].(map[string]any)["estado"] != "declarada_por_rrhh" ||
			strings.Contains(w.Body.String(), "conectado") || strings.Contains(w.Body.String(), "per_firmante_principal_001") ||
			strings.Contains(w.Body.String(), "asignacion:001") || strings.Contains(w.Body.String(), "rol:version:001") {
			t.Fatalf("respuesta engañosa: %s", w.Body)
		}
	}
	if s.llamadas != 2 || s.solicitud.OrganizacionRef != "organizacion:desarrollo:dipgra" ||
		s.solicitud.OriginalRef != "documento:original:001" || string(s.solicitud.PDFFirmado) != "%PDF-1.7\n%%EOF" {
		t.Fatalf("solicitud al servicio: %+v", s.solicitud)
	}
}

func TestRegistroFirmaExternaRechazaEntradasAjenas(t *testing.T) {
	s := &servicioRegistroFirmaExternaPrueba{}
	h, _ := NuevoManejadorRegistroFirmaExterna(autoridadRegistroFirmaExternaPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	casos := map[string]string{
		"actor":                strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"documento":`, `"actor_ref":"actor:forjado","documento":`, 1),
		"organización":         strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"documento":`, `"organizacion_ref":"organizacion:forjada","documento":`, 1),
		"original bytes":       strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"documento":`, `"original_base64":"JVBERi0x","documento":`, 1),
		"certificado":          strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"documento":`, `"certificado_sha256":"x","documento":`, 1),
		"campo duplicado":      strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"documento":`, `"documento":"informe_definitivo","documento":`, 1),
		"PDF inválido":         strings.Replace(cuerpoRegistroFirmaExternaPrueba, "JVBERi0xLjcKJSVFT0Y=", base64.StdEncoding.EncodeToString([]byte("texto")), 1),
		"base64 inválido":      strings.Replace(cuerpoRegistroFirmaExternaPrueba, "JVBERi0xLjcKJSVFT0Y=", "@@", 1),
		"fecha con zona":       strings.Replace(cuerpoRegistroFirmaExternaPrueba, "2026-10-02T10:30:00Z", "2026-10-02T12:30:00+02:00", 1),
		"fecha inexistente":    strings.Replace(cuerpoRegistroFirmaExternaPrueba, "2026-10-02T10:30:00Z", "2026-02-30T10:30:00Z", 1),
		"sin expediente":       strings.Replace(cuerpoRegistroFirmaExternaPrueba, "expediente:ct:001", "", 1),
		"sin versión":          strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"version_expediente":7`, `"version_expediente":0`, 1),
		"sin documento":        strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"documento":"resolucion"`, `"documento":""`, 1),
		"sin paso":             strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"paso_orden":1`, `"paso_orden":0`, 1),
		"sin original":         strings.Replace(cuerpoRegistroFirmaExternaPrueba, "documento:original:001", "", 1),
		"sin versión original": strings.Replace(cuerpoRegistroFirmaExternaPrueba, `"original_version":2`, `"original_version":0`, 1),
		"sin clave":            strings.Replace(cuerpoRegistroFirmaExternaPrueba, "clave-firma-externa-00001", "", 1),
	}
	for nombre, cuerpo := range casos {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpo))
		if w.Code < 400 {
			t.Errorf("%s aceptado: %d %s", nombre, w.Code, w.Body)
		}
	}
	for nombre, alterar := range map[string]func(*http.Request){
		"query":              func(r *http.Request) { r.URL.RawQuery = "x=1" },
		"cookie":             func(r *http.Request) { r.Header.Set("Cookie", "x=y") },
		"identidad cabecera": func(r *http.Request) { r.Header.Set("X-Organizacion", "organizacion:forjada") },
		"otro canal":         func(r *http.Request) { r.URL.Path = RutaFirmaDocumento },
	} {
		r := peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba)
		alterar(r)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Errorf("%s aceptado: %d", nombre, w.Code)
		}
	}
	grande := strings.Replace(cuerpoRegistroFirmaExternaPrueba, "JVBERi0xLjcKJSVFT0Y=", base64.StdEncoding.EncodeToString(append([]byte("%PDF-1.7\n"), make([]byte, ports.MaximoDocumentoFirmaBytes)...)), 1)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaExterna(grande))
	if w.Code < 400 || s.llamadas != 0 {
		t.Fatalf("tamaño o entrada ajena ejecutó servicio: %d, %d", w.Code, s.llamadas)
	}
}

func TestRegistroFirmaExternaRechazaCanalYRespuestaNoConfiable(t *testing.T) {
	s := &servicioRegistroFirmaExternaPrueba{respuesta: resultadoRegistroFirmaExternaPrueba()}
	h, _ := NuevoManejadorRegistroFirmaExterna(autoridadRegistroFirmaExternaPrueba{err: ErrContextoCanalAusente}, s)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba))
	if w.Code != http.StatusUnauthorized || s.llamadas != 0 {
		t.Fatalf("canal ausente: %d %s", w.Code, w.Body)
	}
	h, _ = NuevoManejadorRegistroFirmaExterna(autoridadRegistroFirmaExternaPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	s.respuesta.Material.OrganizacionRef = "organizacion:otra"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("respuesta ajena: %d %s", w.Code, w.Body)
	}
	s.err = ports.ErrClaveFirmaDocumentoUsada
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba))
	if w.Code != http.StatusConflict {
		t.Fatalf("clave en conflicto: %d %s", w.Code, w.Body)
	}
	for _, conflicto := range []error{ports.ErrAntecedenteFirmaR5NoAcreditado, ports.ErrOriginalTrasReparoNoNuevo, ports.ErrMismaPersonaEnOtroPasoR5} {
		s.err = conflicto
		w = httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba))
		if w.Code != http.StatusConflict {
			t.Fatalf("conflicto R5 %v: %d %s", conflicto, w.Code, w.Body)
		}
	}
	s.err = errors.New("sin dependencia")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("dependencia: %d %s", w.Code, w.Body)
	}
}

func TestRegistroFirmaExternaV2PreservaOriginalYRevisionAnterior(t *testing.T) {
	for _, paso := range []int{1, 2} {
		r := resultadoRegistroFirmaExternaPrueba()
		m := materialRegistroFirmaV2Prueba(ports.ViaFirmaExternaPortafirmas, []byte("%PDF-1.7\n%%EOF"))
		cuerpo := cuerpoRegistroFirmaExternaPrueba
		if paso == 2 {
			m.PasoOrden, m.OrdenFirmaPDF, m.Secuencia = 2, 2, 2
			m.FirmaAnteriorRef, m.ReciboAnteriorRef = "firma:previa:001", "recibo:previo:001"
			m.EntradaDocumentoRef, m.EntradaDocumentoVersion = "documento:firmado:previo", 1
			m.EntradaDocumentoHuella = strings.Repeat("e", 64)
			m.EvidenciaFirmasCanonica = json.RawMessage(`[{},{}]`)
			huella := sha256.Sum256(m.EvidenciaFirmasCanonica)
			m.EvidenciaFirmasHuellaSHA256 = hex.EncodeToString(huella[:])
			r.Recibo.Secuencia = 2
			cuerpo = strings.Replace(cuerpo, `"paso_orden":1`, `"paso_orden":2`, 1)
		}
		r.Material = ports.MaterialFirmaExterna{}
		r.MaterialMultiple = &m
		var err error
		r.Recibo.SolicitudHuella, err = m.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		s := &servicioRegistroFirmaExternaPrueba{respuesta: r}
		h, _ := NuevoManejadorRegistroFirmaExternaV2(autoridadRegistroFirmaExternaPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
		for _, estado := range []int{http.StatusCreated, http.StatusOK} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpo))
			var salida struct {
				Data map[string]any `json:"data"`
			}
			if w.Code != estado || json.Unmarshal(w.Body.Bytes(), &salida) != nil {
				t.Fatalf("paso %d: %d %s", paso, w.Code, w.Body)
			}
			d := salida.Data
			v := d["verificacion_tecnica"].(map[string]any)
			revision := d["revision_pdf"].(map[string]any)
			if d["esquema"] != EsquemaRegistroFirmaExternaV2 || d["firma_eficaz"] != false || d["material_root_sha256"] != r.Recibo.SolicitudHuella ||
				v["original_sha256"] != m.OriginalHuella || revision["entrada_sha256"] != m.EntradaDocumentoHuella || revision["orden_firma"] != float64(paso) {
				t.Fatalf("original o revisión alterados: %s", w.Body)
			}
		}
	}
}

func TestRegistroFirmaExternaV2RechazaMaterialAmbiguoYReciboAjeno(t *testing.T) {
	for _, ambiguo := range []bool{false, true} {
		r := resultadoRegistroFirmaExternaPrueba()
		m := materialRegistroFirmaV2Prueba(ports.ViaFirmaExternaPortafirmas, []byte("%PDF-1.7\n%%EOF"))
		r.MaterialMultiple = &m
		if !ambiguo {
			r.Material = ports.MaterialFirmaExterna{}
		}
		r.Recibo.SolicitudHuella = strings.Repeat("a", 64)
		s := &servicioRegistroFirmaExternaPrueba{respuesta: r}
		h, _ := NuevoManejadorRegistroFirmaExternaV2(autoridadRegistroFirmaExternaPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba))
		if w.Code != http.StatusBadGateway {
			t.Fatalf("material ambiguo %v: %d %s", ambiguo, w.Code, w.Body)
		}
	}
}

func TestRegistroFirmaExternaV2NoDegradaContratoEnReplay(t *testing.T) {
	s := &servicioRegistroFirmaExternaPrueba{respuesta: resultadoRegistroFirmaExternaPrueba()}
	s.respuesta.Recibo.YaRegistrada = true
	h, err := NuevoManejadorRegistroFirmaExternaV2(autoridadRegistroFirmaExternaPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpoRegistroFirmaExternaPrueba))
	if w.Code != http.StatusBadGateway || strings.Contains(w.Body.String(), EsquemaRegistroFirmaExterna) {
		t.Fatalf("degradación V1: %d %s", w.Code, w.Body)
	}
}

type autoridadRegistroFirmaExternaContada struct {
	llamadas int
}

func (a *autoridadRegistroFirmaExternaContada) ResolverOrganizacionFirmaExterna(context.Context) (string, error) {
	a.llamadas++
	return "organizacion:desarrollo:dipgra", nil
}

func TestRegistroFirmaExternaRechazaUTF8InvalidoAntesDeAutoridad(t *testing.T) {
	a := &autoridadRegistroFirmaExternaContada{}
	s := &servicioRegistroFirmaExternaPrueba{respuesta: resultadoRegistroFirmaExternaPrueba()}
	h, err := NuevoManejadorRegistroFirmaExternaV2(a, s)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := strings.Replace(cuerpoRegistroFirmaExternaPrueba, "PF-2026-0001", "PF-2026-\xff", 1)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionRegistroFirmaExterna(cuerpo))
	if w.Code != http.StatusBadRequest || a.llamadas != 0 || s.llamadas != 0 ||
		!strings.Contains(w.Body.String(), `"codigo":"peticion_no_valida"`) {
		t.Fatalf("UTF-8 inválido alcanzó la autoridad o aplicación: estado=%d autoridad=%d servicio=%d respuesta=%s",
			w.Code, a.llamadas, s.llamadas, w.Body)
	}
}
