package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Dobles exclusivos de transporte: no prueban mTLS, PDP ni persistencia.
type autoridadGobiernoHTTPPrueba struct {
	err      error
	llamadas int
}

func (a *autoridadGobiernoHTTPPrueba) Credenciales(context.Context) (app.CredencialesGobiernoV3, error) {
	a.llamadas++
	return app.CredencialesGobiernoV3{}, a.err
}

type operadorGobiernoHTTPPrueba struct {
	err            error
	recibo         ports.ReciboAltaBorradorReglasV3
	llamadas       int
	replay, existe bool
	altas          []app.PeticionAltaBorradorV3
}

func (o *operadorGobiernoHTTPPrueba) GuardarAltaBorrador(_ context.Context, _ app.CredencialesGobiernoV3, p app.PeticionAltaBorradorV3) (ports.ResultadoAltaBorradorReglasV3, error) {
	o.llamadas++
	o.altas = append(o.altas, p)
	return ports.ResultadoAltaBorradorReglasV3{Recibo: o.recibo, Replay: o.replay}, o.err
}
func (o *operadorGobiernoHTTPPrueba) ConsultarExacta(context.Context, app.CredencialesGobiernoV3, app.PeticionConsultaExactaV3) (ports.ResultadoConsultaGobiernoReglasV3, error) {
	o.llamadas++
	return ports.ResultadoConsultaGobiernoReglasV3{VersionCanonica: o.recibo.VersionCanonica}, o.err
}
func (o *operadorGobiernoHTTPPrueba) RecuperarRecibo(context.Context, app.CredencialesGobiernoV3, app.PeticionRecuperarReciboV3) (ports.ResultadoRecuperacionGobiernoReglasV3, error) {
	o.llamadas++
	return ports.ResultadoRecuperacionGobiernoReglasV3{Recibo: o.recibo, Existe: o.existe}, o.err
}

func escenarioGobiernoHTTPPrueba(t *testing.T) (*HandlerGobiernoReglasBaremoV3, *autoridadGobiernoHTTPPrueba, *operadorGobiernoHTTPPrueba, map[string]any) {
	t.Helper()
	canon, err := os.ReadFile("../../application/simulacionbaremo/testdata/reglas_a.json")
	if err != nil {
		t.Fatal(err)
	}
	motivo := motivoGobiernoHTTPV3{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("d", 32)}
	body := map[string]any{"reglas": json.RawMessage(canon), "motivo": motivo, "clave_operacion": strings.Repeat("a", 32)}
	e := entradaGobiernoHTTPV3{Reglas: canon, Motivo: motivo, ClaveOperacion: strings.Repeat("a", 32)}
	p, err := e.alta()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 123456000, time.UTC)
	v, err := reglas.NuevaVersionGobernadaReglasBaremo(p.Conjunto, "per_"+strings.Repeat("a", 24), p.Motivo, ahora)
	if err != nil {
		t.Fatal(err)
	}
	version, err := v.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	estado, err := v.VinculoEstado()
	if err != nil {
		t.Fatal(err)
	}
	a := &autoridadGobiernoHTTPPrueba{}
	o := &operadorGobiernoHTTPPrueba{existe: true, recibo: ports.ReciboAltaBorradorReglasV3{ReciboRef: "recibo:original", ConfirmadaEn: ahora, Estado: estado, VersionCanonica: version, ClaveOperacion: p.ClaveOperacion, HuellaSolicitudSHA256: strings.Repeat("b", 64)}}
	h, err := NuevoHandlerGobiernoReglasBaremoV3(a, o, func(context.Context, string, error) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	return h, a, o, body
}
func peticionGobiernoHTTPPrueba(t *testing.T, h http.Handler, ruta string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, ruta, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestGobiernoReglasHTTPV3AltaReplayConsultaYRecuperacion(t *testing.T) {
	h, _, o, body := escenarioGobiernoHTTPPrueba(t)
	alta := peticionGobiernoHTTPPrueba(t, h, RutaAltaGobiernoReglasBaremoV3, body)
	if alta.Code != http.StatusCreated || alta.Header().Get("Cache-Control") != "no-store" || alta.Header().Get("Set-Cookie") != "" {
		t.Fatalf("alta: %d %s", alta.Code, alta.Body.String())
	}
	var primera struct {
		Recibo reciboGobiernoHTTPV3 `json:"recibo"`
		Replay bool                 `json:"replay"`
	}
	if err := json.Unmarshal(alta.Body.Bytes(), &primera); err != nil {
		t.Fatal(err)
	}
	if primera.Recibo.Disponibilidad != "disponible_para_preparacion" || primera.Recibo.Estado != reglas.EstadoReglasBaremoBorrador || primera.Replay {
		t.Fatal("alta anticipó otro estado")
	}
	o.replay = true
	replay := peticionGobiernoHTTPPrueba(t, h, RutaAltaGobiernoReglasBaremoV3, body)
	var repetida struct {
		Recibo reciboGobiernoHTTPV3 `json:"recibo"`
		Replay bool                 `json:"replay"`
	}
	if err := json.Unmarshal(replay.Body.Bytes(), &repetida); err != nil {
		t.Fatal(err)
	}
	if replay.Code != http.StatusOK || !repetida.Replay || primera.Recibo != repetida.Recibo || o.altas[0].ClaveOperacion != o.altas[1].ClaveOperacion {
		t.Fatal("transporte perdió clave/recibo/fecha originales")
	}
	consultaBody := map[string]any{"selector": primera.Recibo.Selector, "motivo": body["motivo"]}
	consulta := peticionGobiernoHTTPPrueba(t, h, RutaConsultaGobiernoReglasBaremoV3, consultaBody)
	if consulta.Code != http.StatusOK || !bytes.Contains(consulta.Body.Bytes(), []byte(`"reglas"`)) {
		t.Fatalf("consulta: %d %s", consulta.Code, consulta.Body.String())
	}
	consultaBody["clave_operacion"] = body["clave_operacion"]
	consultaBody["huella_solicitud_sha256"] = o.recibo.HuellaSolicitudSHA256
	recuperacion := peticionGobiernoHTTPPrueba(t, h, RutaRecuperarGobiernoReglasBaremoV3, consultaBody)
	if recuperacion.Code != http.StatusOK || !bytes.Contains(recuperacion.Body.Bytes(), []byte(`"recibo_ref":"recibo:original"`)) {
		t.Fatal("recuperación no devolvió original")
	}
	o.existe = false
	ausente := peticionGobiernoHTTPPrueba(t, h, RutaRecuperarGobiernoReglasBaremoV3, consultaBody)
	if ausente.Code != http.StatusOK || ausente.Body.String() != `{"existe":false}` {
		t.Fatalf("ausencia: %d %s", ausente.Code, ausente.Body.String())
	}
	for _, respuesta := range []*httptest.ResponseRecorder{alta, replay, consulta, recuperacion, ausente} {
		for _, prohibido := range []string{"capacidad_canonica", "payload_vec", "sobre_cose", "contexto_actor_canonico", "persona_ref", "perfil_ref"} {
			if strings.Contains(respuesta.Body.String(), prohibido) {
				t.Fatalf("exposición: %s", prohibido)
			}
		}
	}
}

func TestGobiernoReglasHTTPV3RechazaAutoridadYMaterialFueraContrato(t *testing.T) {
	for _, caso := range []string{"actor", "perfil", "token", "schema_meritos", "json_trailing", "campo_motivo_ajeno", "grande"} {
		t.Run(caso, func(t *testing.T) {
			h, _, o, body := escenarioGobiernoHTTPPrueba(t)
			if caso == "actor" || caso == "perfil" || caso == "token" {
				body[caso] = "dato:cliente"
			}
			if caso == "schema_meritos" {
				body["reglas"] = map[string]any{"esquema": "vec.bolsa.reglas_meritos.v1"}
			}
			if caso == "campo_motivo_ajeno" {
				body["motivo"] = map[string]any{"actor_ref": "actor:cliente"}
			}
			b, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			if caso == "json_trailing" {
				b = append(b, []byte(` {}`)...)
			}
			esperado := http.StatusBadRequest
			if caso == "grande" {
				b = bytes.Repeat([]byte(" "), maximoEntradaGobiernoReglasV3+1)
				esperado = http.StatusRequestEntityTooLarge
			}
			r := httptest.NewRequest(http.MethodPost, RutaAltaGobiernoReglasBaremoV3, bytes.NewReader(b))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != esperado || o.llamadas != 0 {
				t.Fatalf("entrada llegó a aplicación: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestGobiernoReglasHTTPV3DenegacionIndisponibilidadYConflictoSinRecibo(t *testing.T) {
	for _, caso := range []struct {
		err    error
		status int
	}{
		{app.ErrGobiernoV3NoAutenticado, 401}, {app.ErrGobiernoV3Prohibido, 403}, {app.ErrGobiernoV3NoDisponible, 503},
		{ports.ErrConflictoOCCReglasBaremo, 409}, {ports.ErrClaveIdempotenciaReglasReutilizada, 409},
		{ports.ErrConfirmacionReglasBaremoInvalida, 503}, {errors.New("detalle privado de COMMIT incierto"), 503},
	} {
		h, _, o, body := escenarioGobiernoHTTPPrueba(t)
		o.err = caso.err
		w := peticionGobiernoHTTPPrueba(t, h, RutaAltaGobiernoReglasBaremoV3, body)
		if w.Code != caso.status || strings.Contains(w.Body.String(), "recibo_ref") || strings.Contains(w.Body.String(), "privado") {
			t.Fatalf("categoría o fuga: %d %s", w.Code, w.Body.String())
		}
	}
	h, a, o, body := escenarioGobiernoHTTPPrueba(t)
	a.err = app.ErrGobiernoV3NoDisponible
	w := peticionGobiernoHTTPPrueba(t, h, RutaAltaGobiernoReglasBaremoV3, body)
	if w.Code != 503 || o.llamadas != 0 {
		t.Fatal("fallo de credenciales alcanzó operador")
	}
}

func TestGobiernoReglasHTTPV3FamiliaAusenteYRutasExactas(t *testing.T) {
	for _, ruta := range []string{RutaAltaGobiernoReglasBaremoV3, RutaConsultaGobiernoReglasBaremoV3, RutaRecuperarGobiernoReglasBaremoV3} {
		w := peticionGobiernoHTTPPrueba(t, &HandlerGobiernoReglasBaremoV3{}, ruta, map[string]any{})
		if w.Code != 503 || strings.Contains(w.Body.String(), "recibo") {
			t.Fatal("familia ausente confirmó algo")
		}
	}
	h, a, o, _ := escenarioGobiernoHTTPPrueba(t)
	for _, caso := range []struct {
		metodo, ruta string
		status       int
	}{
		{http.MethodGet, RutaAltaGobiernoReglasBaremoV3, 405}, {http.MethodHead, RutaConsultaGobiernoReglasBaremoV3, 405},
		{http.MethodPost, RutaAltaGobiernoReglasBaremoV3 + "/", 404}, {http.MethodPost, RutaAltaGobiernoReglasBaremoV3 + "?actor=x", 404},
	} {
		r := httptest.NewRequest(caso.metodo, caso.ruta, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.status || a.llamadas != 0 || o.llamadas != 0 {
			t.Fatal("ruta/método ajeno alcanzó identidad")
		}
	}
}

func TestGobiernoReglasHTTPV3CallbackSoloParaSesionAnteriorPDP(t *testing.T) {
	h, a, o, body := escenarioGobiernoHTTPPrueba(t)
	llamadas := 0
	h.auditar = func(context.Context, string, error) error { llamadas++; return nil }
	// El operador representa aquí la denegación posterior del PDP. Su
	// autoridad V3 conserva ese registro; no se inventa otro de sesión.
	o.err = app.ErrGobiernoV3Prohibido
	w := peticionGobiernoHTTPPrueba(t, h, RutaAltaGobiernoReglasBaremoV3, body)
	if w.Code != 403 || llamadas != 0 || o.llamadas != 1 {
		t.Fatal("denegación PDP adquirió callback de sesión")
	}
	a.err = app.ErrGobiernoV3NoAutenticado
	w = peticionGobiernoHTTPPrueba(t, h, RutaAltaGobiernoReglasBaremoV3, body)
	if w.Code != 401 || llamadas != 1 || o.llamadas != 1 {
		t.Fatal("rechazo sesión llegó al operador o perdió auditoría")
	}
	a.err = app.ErrGobiernoV3NoDisponible
	w = peticionGobiernoHTTPPrueba(t, h, RutaAltaGobiernoReglasBaremoV3, body)
	if w.Code != 503 || llamadas != 1 || o.llamadas != 1 {
		t.Fatal("caída de sesión inventó un motivo de denegación")
	}
	if _, err := NuevoHandlerGobiernoReglasBaremoV3(a, o, nil); err == nil {
		t.Fatal("constructor habilitó familia sin callback")
	}
}
