package httpinterno

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorRecuperacionHTTPPrueba struct {
	t        *testing.T
	err      error
	llamadas int
}

func (a *autorizadorRecuperacionHTTPPrueba) AutorizarRecuperacionFirmasV2(_ context.Context, m ct.MaterialConsultaFirmasR5V2) (ct.CapacidadRecuperacionFirmasV2, error) {
	a.llamadas++
	if a.err != nil {
		return ct.CapacidadRecuperacionFirmasV2{}, a.err
	}
	r, err := firmaautorizacionv2.RecursoConsultaFirmasR5V2(m)
	if err != nil {
		a.t.Fatal(err)
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		a.t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	res, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:recuperacion-v2", strings.Repeat("a", 64), strings.Repeat("b", 64),
		"contexto:recuperacion-v2", strings.Repeat("c", 64), ct.AccionRecuperarFirmasR5V2, r.Referencia, h, ct.AudienciaRecuperacionFirmasR5V2, ahora, ahora.Add(5*time.Second))
	if err != nil {
		a.t.Fatal(err)
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{5}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		a.t.Fatal(err)
	}
	exportado, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{4}, vp.TamanoMinimoCapacidadCanonicaV3), res,
		[]byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		a.t.Fatal(err)
	}
	return ct.TransportarMaterialRecuperacionFirmasV2(exportado, ct.CamposRecuperacionFirmasV2(), nil), nil
}

type lectorRecuperacionHTTPPrueba struct {
	err      error
	llamadas int
	cerrado  *registroConsultaV2Prueba
	cancelar context.CancelFunc
}

func (l *lectorRecuperacionHTTPPrueba) RecuperarFirmasAutorizadasV2(_ context.Context, _ ct.MaterialConsultaFirmasR5V2, _ ct.CapacidadRecuperacionFirmasV2) (ct.LecturaRecuperacionFirmasV2, error) {
	l.llamadas++
	l.cerrado.cerrado = true
	if l.cancelar != nil {
		l.cancelar()
	}
	return ct.LecturaRecuperacionFirmasV2{LecturaFirmasR5V2: ct.LecturaFirmasR5V2{
		LecturaFirmasR5: ct.LecturaFirmasR5{HistoriaHuella: strings.Repeat("a", 64)}},
		Recuperaciones: []ct.RecuperacionFirmaV2{}}, l.err
}

func manejadorRecuperacionHTTPPrueba(t *testing.T) (http.Handler, *fuenteConsultaV2Prueba, *autorizadorRecuperacionHTTPPrueba,
	*lectorRecuperacionHTTPPrueba, *intentosConsultaV2Prueba, *fabricaConsultaV2Prueba) {
	t.Helper()
	fuente := &fuenteConsultaV2Prueba{}
	autorizador := &autorizadorRecuperacionHTTPPrueba{t: t}
	marca := &registroConsultaV2Prueba{}
	lector := &lectorRecuperacionHTTPPrueba{cerrado: marca}
	intentos := &intentosConsultaV2Prueba{t: t}
	fabrica := &fabricaConsultaV2Prueba{t: t, registro: marca}
	h, err := NuevoManejadorRecuperacionFirmasR5V2(fuente, autorizador, lector, intentos, fabrica)
	if err != nil {
		t.Fatal(err)
	}
	return h, fuente, autorizador, lector, intentos, fabrica
}

func peticionRecuperacionHTTPPrueba(t *testing.T, cuerpo string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, RutaRecuperacionFirmasR5V2, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	ctx, err := vp.ConCorrelacionIncidenciasPeticion(r.Context())
	if err != nil {
		t.Fatal(err)
	}
	return r.WithContext(ctx)
}

func TestRecuperacionHTTPSalidaYCanonExacto(t *testing.T) {
	h, _, a, l, intentos, _ := manejadorRecuperacionHTTPPrueba(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionRecuperacionHTTPPrueba(t, cuerpoConsultaV2()))
	if w.Code != http.StatusOK || a.llamadas != 1 || l.llamadas != 1 || intentos.llamadas != 0 {
		t.Fatalf("recuperación vacía no válida: HTTP%d", w.Code)
	}
	var cuerpo map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	data := cuerpo["data"].(map[string]any)
	if data["esquema"] != EsquemaRecuperacionFirmasR5V2 || data["recuperacion"] != "recuperada" ||
		len(data["campos_no_disponibles"].([]any)) != 0 || len(data["recuperaciones"].([]any)) != 0 || data["firma_eficaz"] != false {
		t.Fatal("la respuesta convierte el estado o inventa firma eficaz")
	}
	canon := "{\n  \"esquema\": \"sintético\", \"ñ\": \"<&\"\n}"
	vista := proyectarRecuperacionFirmasR5V2(consultafirmasv2.ResultadoRecuperacion{
		Recuperaciones: []ct.RecuperacionFirmaV2{{FirmaRef: "firma:uno", MaterialRootSHA256: strings.Repeat("1", 64),
			CanonNominal: canon, CanonNominalSHA256: strings.Repeat("2", 64), CanonNominalRef: "evidencia:prueba"}}})
	contenido, err := json.Marshal(vista)
	if err != nil {
		t.Fatal(err)
	}
	var decodificada map[string]any
	if err := json.Unmarshal(contenido, &decodificada); err != nil {
		t.Fatal(err)
	}
	recuperada := decodificada["recuperaciones"].([]any)[0].(map[string]any)
	if recuperada["canon_nominal"] != canon || recuperada["canon_nominal_sha256"] != strings.Repeat("2", 64) ||
		recuperada["material_root_sha256"] != strings.Repeat("1", 64) || recuperada["canon_nominal_ref"] != "evidencia:prueba" {
		t.Fatal("los bytes o las referencias del canon se alteraron")
	}
}

func TestRecuperacionHTTPRechazaCamposAjenosYAuditaAntesFuente(t *testing.T) {
	for _, extra := range []string{`,"actor":"cliente"`, `,"documento":"resolucion"`, `,"Recuperaciones":[]`} {
		h, fuente, _, lector, intentos, _ := manejadorRecuperacionHTTPPrueba(t)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRecuperacionHTTPPrueba(t, strings.TrimSuffix(cuerpoConsultaV2(), "}")+extra+"}"))
		if w.Code != http.StatusBadRequest || fuente.llamadas != 0 || lector.llamadas != 0 || intentos.llamadas != 1 {
			t.Fatalf("entrada ajena admitida/no auditada: HTTP%d", w.Code)
		}
	}
}

func TestRecuperacionHTTPDenegacionErrorYAcuseCruzado(t *testing.T) {
	for _, caso := range []string{"autorizador", "lector", "acuse"} {
		h, _, a, l, intentos, fabrica := manejadorRecuperacionHTTPPrueba(t)
		switch caso {
		case "autorizador":
			a.err = ct.ErrFirmaDocumentoDenegada
		case "lector", "acuse":
			l.err = ct.ErrFirmaDocumentoDenegada
			fabrica.requiereCerrado = true
			intentos.acuseAjeno = caso == "acuse"
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRecuperacionHTTPPrueba(t, cuerpoConsultaV2()))
		esperado := http.StatusNotFound
		if caso == "acuse" {
			esperado = http.StatusServiceUnavailable
		}
		if w.Code != esperado || intentos.llamadas != 1 || fabrica.llamadas != 1 ||
			(caso == "autorizador" && l.llamadas != 0) || (caso != "autorizador" && (!l.cerrado.cerrado || l.llamadas != 1)) {
			t.Fatalf("%s perdió auditoría/duplicó intento: HTTP%d", caso, w.Code)
		}
		if strings.Contains(w.Body.String(), "recuperaciones") || strings.Contains(w.Body.String(), "canon_nominal") {
			t.Fatal("fallo expuso recuperación")
		}
	}
}

func TestRecuperacionHTTPCancelacionTrasLectorNoOcultaFalloAuditoria(t *testing.T) {
	h, _, _, lector, intentos, fabrica := manejadorRecuperacionHTTPPrueba(t)
	fabrica.requiereCerrado = true
	intentos.fallo = true
	peticion := peticionRecuperacionHTTPPrueba(t, cuerpoConsultaV2())
	ctx, cancelar := context.WithCancel(peticion.Context())
	defer cancelar()
	lector.cancelar = cancelar
	lector.err = context.Canceled
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion.WithContext(ctx))
	if w.Code != http.StatusServiceUnavailable || intentos.llamadas != 1 || fabrica.llamadas != 1 || !lector.cerrado.cerrado {
		t.Fatalf("cancelación ocultó fallo de auditoría: HTTP%d", w.Code)
	}
}

func TestRecuperacionHTTPConstructorRechazaAutoridadNula(t *testing.T) {
	var lector *lectorRecuperacionHTTPPrueba
	if _, err := NuevoManejadorRecuperacionFirmasR5V2(&fuenteConsultaV2Prueba{},
		&autorizadorRecuperacionHTTPPrueba{t: t}, lector, &intentosConsultaV2Prueba{t: t},
		&fabricaConsultaV2Prueba{t: t}); !errors.Is(err, ct.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatal("lector nulo admitido")
	}
}

func TestRecuperacionHTTPSinCorrelacionNoLlegaALector(t *testing.T) {
	h, fuente, autorizador, lector, _, _ := manejadorRecuperacionHTTPPrueba(t)
	r := httptest.NewRequest(http.MethodPost, RutaRecuperacionFirmasR5V2, strings.NewReader(cuerpoConsultaV2()))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || fuente.llamadas != 0 || autorizador.llamadas != 0 || lector.llamadas != 0 {
		t.Fatal("sin correlación se inició lectura nominal")
	}
}
