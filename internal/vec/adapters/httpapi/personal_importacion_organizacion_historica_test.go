package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const claveImportacionHTTPPrueba = "018f47a2-6b31-4c80-8a95-4d2e707c5a11"

type autoridadImportacionHTTPPrueba struct {
	actor     vecdomain.ContextoActor
	organismo string
	err       error
	llamadas  int
}

func (a *autoridadImportacionHTTPPrueba) ResolverContextoImportacionOrganizacion(context.Context) (vecdomain.ContextoActor, string, error) {
	a.llamadas++
	return a.actor, a.organismo, a.err
}

type operadorImportacionHTTPPrueba struct {
	solicitudes []personaldomain.SolicitudImportacionOrganizacion
	err         error
	replay      bool
	alterar     func(*personalports.ReciboImportacionOrganizacion)
}

func (o *operadorImportacionHTTPPrueba) Preparar(_ context.Context, s personaldomain.SolicitudImportacionOrganizacion) (personalports.ReciboImportacionOrganizacion, error) {
	return o.ejecutar(s)
}
func (o *operadorImportacionHTTPPrueba) Conciliar(_ context.Context, s personaldomain.SolicitudImportacionOrganizacion) (personalports.ReciboImportacionOrganizacion, error) {
	return o.ejecutar(s)
}
func (o *operadorImportacionHTTPPrueba) Publicar(_ context.Context, s personaldomain.SolicitudImportacionOrganizacion) (personalports.ReciboImportacionOrganizacion, error) {
	return o.ejecutar(s)
}
func (o *operadorImportacionHTTPPrueba) ejecutar(s personaldomain.SolicitudImportacionOrganizacion) (personalports.ReciboImportacionOrganizacion, error) {
	o.solicitudes = append(o.solicitudes, s)
	if o.err != nil {
		return personalports.ReciboImportacionOrganizacion{}, o.err
	}
	estado := "preparacion_no_autoritativa"
	if s.Fase == personaldomain.FaseConciliarOrganizacion {
		estado = "conciliada"
	}
	if s.Fase == personaldomain.FasePublicarOrganizacion {
		estado = "publicada"
	}
	lote := s.LoteRef
	if lote == "" {
		lote = "lote:prueba"
	}
	material, _ := personaldomain.NuevoMaterialImportacionOrganizacion(s)
	r := personalports.ReciboImportacionOrganizacion{ReciboRef: "recibo:prueba", LoteRef: lote, Fase: s.Fase, Estado: estado,
		RevisionAnterior: s.RevisionEsperada, RevisionNueva: s.RevisionEsperada + 1, ClaveIdempotencia: s.ClaveIdempotencia,
		MaterialHuellaSHA256: material.HuellaSHA256(), FuenteHuellaSHA256: s.Manifiesto.FuenteHuellaSHA256,
		ActorRef: s.Actor.Principal.ID, DecisionRef: "decision:prueba", AuditoriaRef: "auditoria:prueba",
		RegistradoEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC), Replay: o.replay}
	if o.alterar != nil {
		o.alterar(&r)
	}
	return r, nil
}

type auditorImportacionHTTPPrueba struct {
	ordenes []DenegacionImportacionOrganizacion
	err     error
}

func (a *auditorImportacionHTTPPrueba) RegistrarDenegacionImportacionOrganizacion(_ context.Context, o DenegacionImportacionOrganizacion) error {
	a.ordenes = append(a.ordenes, o)
	return a.err
}

func cuerpoImportacionHTTPPrueba(fase personaldomain.FaseImportacionOrganizacion) []byte {
	manifest := map[string]any{
		"tipo": "rpt", "version_ref": claveImportacionHTTPPrueba, "version_revision": 1,
		"fuente_ref": "fuente:pdf", "fuente_version": "v1", "fuente_huella_sha256": strings.Repeat("a", 64),
		"catalogo_unidades":        map[string]any{"id": "catalogo:unidades", "version": 1, "revision": 1, "huella_sha256": strings.Repeat("b", 64)},
		"catalogo_clasificaciones": map[string]any{"id": "catalogo:clasificaciones", "version": 1, "revision": 1, "huella_sha256": strings.Repeat("c", 64)},
	}
	body := map[string]any{"manifiesto": manifest, "revision_esperada": 0}
	switch fase {
	case personaldomain.FasePrepararOrganizacion:
		body["hechos"] = []any{map[string]any{"clase": "puesto_tipo", "hecho_ref": "018f47a2-6b31-4c80-8a95-4d2e707c5a12", "revision": 1,
			"fila_fuente_ref": "fila:uno", "unidad_ref": "centro:uno", "vigente_desde": "2026-01-03", "codigo_fuente": "001",
			"denominacion": "Puesto tipo", "clasificacion_ref": "categoria:uno"}}
	case personaldomain.FaseConciliarOrganizacion:
		body["lote_ref"] = "lote:prueba"
		body["revision_esperada"] = 1
		body["decisiones"] = []any{map[string]any{"fila_fuente_ref": "fila:uno", "clase": "puesto_tipo", "resultado": "pendiente", "motivo": "Pendiente de contraste", "evidencia_ref": "evidencia:uno"}}
	case personaldomain.FasePublicarOrganizacion:
		body["lote_ref"] = "lote:prueba"
		body["revision_esperada"] = 2
		manifest["documento_ref"] = "documento:uno"
		manifest["custodia_ref"] = "custodia:uno"
		manifest["diccionario_ref"] = "diccionario:uno"
		manifest["acto_ref"] = "acto:uno"
		manifest["aprobada_en"] = "2026-01-01"
		manifest["publicada_en"] = "2026-01-02"
		manifest["efectos_desde"] = "2026-01-03"
	}
	b, _ := json.Marshal(body)
	return b
}

func peticionImportacionHTTPPrueba(ruta string, body []byte) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", claveImportacionHTTPPrueba)
	return r
}
func entornoImportacionHTTPPrueba(t *testing.T) (http.Handler, *autoridadImportacionHTTPPrueba, *operadorImportacionHTTPPrueba, *auditorImportacionHTTPPrueba) {
	t.Helper()
	a := &autoridadImportacionHTTPPrueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "organismo:dipgra"}
	o, audit := &operadorImportacionHTTPPrueba{}, &auditorImportacionHTTPPrueba{}
	h, err := NewHandlerImportacionOrganizacionPersonal(a, o, audit)
	if err != nil {
		t.Fatal(err)
	}
	return h, a, o, audit
}

func TestImportacionOrganizacionHTTPFallaSinDependencias(t *testing.T) {
	a, o, audit := &autoridadImportacionHTTPPrueba{}, &operadorImportacionHTTPPrueba{}, &auditorImportacionHTTPPrueba{}
	for _, caso := range []struct {
		a     AutoridadContextoImportacionOrganizacion
		o     OperadorImportacionOrganizacion
		audit AuditorDenegacionImportacionOrganizacion
	}{
		{nil, o, audit}, {a, nil, audit}, {a, o, nil},
	} {
		if h, err := NewHandlerImportacionOrganizacionPersonal(caso.a, caso.o, caso.audit); !errors.Is(err, ErrHandlerImportacionOrganizacionInvalido) || h != nil {
			t.Fatalf("montaje inesperado: %v", err)
		}
	}
}

func TestImportacionOrganizacionHTTPFasesYReplay(t *testing.T) {
	for _, caso := range []struct {
		fase personaldomain.FaseImportacionOrganizacion
		ruta string
	}{
		{personaldomain.FasePrepararOrganizacion, RutaPrepararImportacionOrganizacion},
		{personaldomain.FaseConciliarOrganizacion, RutaConciliarImportacionOrganizacion},
		{personaldomain.FasePublicarOrganizacion, RutaPublicarImportacionOrganizacion},
	} {
		t.Run(string(caso.fase), func(t *testing.T) {
			h, a, o, audit := entornoImportacionHTTPPrueba(t)
			body := cuerpoImportacionHTTPPrueba(caso.fase)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionImportacionHTTPPrueba(caso.ruta, body))
			if w.Code != 201 || len(o.solicitudes) != 1 || len(audit.ordenes) != 0 || o.solicitudes[0].Fase != caso.fase ||
				o.solicitudes[0].Manifiesto.OrganismoRef != a.organismo || o.solicitudes[0].Actor.Principal.ID != a.actor.Principal.ID || !strings.Contains(w.Body.String(), `"data"`) {
				t.Fatalf("fase=%s status=%d cuerpo=%s solicitudes=%+v", caso.fase, w.Code, w.Body.String(), o.solicitudes)
			}
			if caso.fase == personaldomain.FasePrepararOrganizacion && o.solicitudes[0].Hechos[0].OrganismoRef != a.organismo {
				t.Fatal("hecho no ligado al organismo servidor")
			}
			if caso.fase == personaldomain.FasePublicarOrganizacion && o.solicitudes[0].RevisorActorRef != a.actor.Principal.ID {
				t.Fatal("revisor no derivado del servidor")
			}
			if w.Header().Get("Cache-Control") == "" || w.Header().Get("Set-Cookie") != "" {
				t.Fatal("cabeceras inseguras")
			}
			o.replay = true
			w = httptest.NewRecorder()
			h.ServeHTTP(w, peticionImportacionHTTPPrueba(caso.ruta, body))
			if w.Code != 200 || len(o.solicitudes) != 2 || o.solicitudes[0].CorrelacionRef != o.solicitudes[1].CorrelacionRef {
				t.Fatalf("replay=%d solicitudes=%+v", w.Code, o.solicitudes)
			}
		})
	}
}

func TestImportacionOrganizacionHTTPJSONYClaveEstrictos(t *testing.T) {
	base := cuerpoImportacionHTTPPrueba(personaldomain.FasePrepararOrganizacion)
	for _, caso := range []struct {
		nombre  string
		body    []byte
		alterar func(*http.Request)
	}{
		{"clave ausente", base, func(r *http.Request) { r.Header.Del("Idempotency-Key") }},
		{"clave duplicada", base, func(r *http.Request) {
			r.Header["Idempotency-Key"] = []string{claveImportacionHTTPPrueba, claveImportacionHTTPPrueba}
		}},
		{"clave no v4", base, func(r *http.Request) {
			r.Header.Set("Idempotency-Key", strings.Replace(claveImportacionHTTPPrueba, "-4c80-", "-1c80-", 1))
		}},
		{"organismo cliente", append([]byte(`{"organismo_ref":"org:otra",`), base[1:]...), nil},
		{"organismo cliente con mayusculas", append([]byte(`{"Organismo_Ref":"org:otra",`), base[1:]...), nil},
		{"organismo en manifiesto", bytes.Replace(base, []byte(`"tipo":"rpt"`), []byte(`"organismo_ref":"org:otra","tipo":"rpt"`), 1), nil},
		{"organismo en hecho", bytes.Replace(base, []byte(`"clase":"puesto_tipo"`), []byte(`"organismo_ref":"org:otra","clase":"puesto_tipo"`), 1), nil},
		{"actor cliente", append([]byte(`{"actor":"per_ajeno",`), base[1:]...), nil},
		{"duplicado anidado", []byte(`{"manifiesto":{"tipo":"rpt","tipo":"plantilla"}}`), nil},
		{"desconocido", append([]byte(`{"desconocido":true,`), base[1:]...), nil},
		{"dos documentos", append(append([]byte{}, base...), base...), nil},
		{"tipo incorrecto", base, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }},
		{"cookie", base, func(r *http.Request) { r.Header.Set("Cookie", "") }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			h, a, o, _ := entornoImportacionHTTPPrueba(t)
			r := peticionImportacionHTTPPrueba(RutaPrepararImportacionOrganizacion, caso.body)
			if caso.alterar != nil {
				caso.alterar(r)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 400 || a.llamadas != 0 || len(o.solicitudes) != 0 {
				t.Fatalf("status=%d cuerpo=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestImportacionOrganizacionHTTPLimiteRawOchoMiB(t *testing.T) {
	h, a, o, _ := entornoImportacionHTTPPrueba(t)
	cuerpo := bytes.Repeat([]byte{' '}, maximoCuerpoImportacionOrganizacion+1)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionImportacionHTTPPrueba(RutaPrepararImportacionOrganizacion, cuerpo))
	if w.Code != 400 || a.llamadas != 0 || len(o.solicitudes) != 0 {
		t.Fatalf("límite raw no aplicado: %d", w.Code)
	}
}

func TestImportacionOrganizacionHTTPDeniegaYAudita(t *testing.T) {
	for _, caso := range []struct {
		nombre                        string
		errorAutoridad, errorOperador error
		status                        int
		auditorias                    int
	}{
		{"identidad", ErrAutenticacionRutaExactaRequerida, nil, 401, 1},
		{"permiso", ErrAccesoRutaExactaDenegado, nil, 403, 1},
		{"denegacion v3", nil, personaldomain.ErrImportacionOrganizacionDenegada, 403, 1},
		{"conflicto", nil, personaldomain.ErrImportacionOrganizacionConflicto, 409, 0},
		{"fuente no acreditada", nil, personaldomain.ErrImportacionOrganizacionNoDisponible, 503, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			h, a, o, audit := entornoImportacionHTTPPrueba(t)
			a.err, o.err = caso.errorAutoridad, caso.errorOperador
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionImportacionHTTPPrueba(RutaPublicarImportacionOrganizacion, cuerpoImportacionHTTPPrueba(personaldomain.FasePublicarOrganizacion)))
			if w.Code != caso.status || len(audit.ordenes) != caso.auditorias || strings.Contains(w.Body.String(), "personal: ") {
				t.Fatalf("status=%d cuerpo=%s auditoria=%+v", w.Code, w.Body.String(), audit.ordenes)
			}
		})
	}
}

func TestImportacionOrganizacionHTTPPublicacionIncompletaNoSeEjecuta(t *testing.T) {
	h, _, o, _ := entornoImportacionHTTPPrueba(t)
	body := cuerpoImportacionHTTPPrueba(personaldomain.FaseConciliarOrganizacion)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionImportacionHTTPPrueba(RutaPublicarImportacionOrganizacion, body))
	if w.Code != 400 || len(o.solicitudes) != 0 {
		t.Fatalf("publicacion incompleta: %d %s", w.Code, w.Body.String())
	}
}

func TestImportacionOrganizacionHTTPNoPublicaReciboIncoherente(t *testing.T) {
	h, _, o, _ := entornoImportacionHTTPPrueba(t)
	o.alterar = func(r *personalports.ReciboImportacionOrganizacion) { r.Estado = "preparacion_no_autoritativa" }
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionImportacionHTTPPrueba(RutaPublicarImportacionOrganizacion, cuerpoImportacionHTTPPrueba(personaldomain.FasePublicarOrganizacion)))
	if w.Code != 503 || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("recibo falso publicado: %d %s", w.Code, w.Body.String())
	}
}
