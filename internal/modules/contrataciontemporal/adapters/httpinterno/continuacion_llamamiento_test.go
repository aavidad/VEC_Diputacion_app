package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const cuerpoContinuacionHTTPPrueba = `{"clave_idempotencia":"818f47a6-5d2b-4c10-8a11-1234567890ab","organizacion_ref":"organizacion:http-comunicacion","expediente_ref":"expediente:http-comunicacion","resolucion_ref":"resolucion:http-local","intencion_ref":"outbox:http-siguiente"}`

// Reutiliza los receptores anteriores; este doble solo añade la quinta operación.
type ejecutorContinuacionHTTPPrueba struct {
	*ejecutorComunicacionLlamamientoHTTPPrueba
	resultado   ports.ResultadoContinuacionLlamamiento
	err         error
	solicitudes []ports.SolicitudContinuarLlamamiento
	durante     func(context.Context)
}

func (e *ejecutorContinuacionHTTPPrueba) Continuar(ctx context.Context, s ports.SolicitudContinuarLlamamiento) (ports.ResultadoContinuacionLlamamiento, error) {
	e.solicitudes = append(e.solicitudes, s)
	if e.durante != nil {
		e.durante(ctx)
	}
	return e.resultado, e.err
}

func nuevaContinuacionHTTPPrueba() *ejecutorContinuacionHTTPPrueba {
	anterior := ejecutorComunicacionHTTPValidoPrueba()
	s := ports.SolicitudContinuarLlamamiento{
		ClaveIdempotencia: "818f47a6-5d2b-4c10-8a11-1234567890ab",
		OrganizacionRef:   anterior.resolucion.Solicitud.OrganizacionRef, ExpedienteRef: anterior.resolucion.Solicitud.ExpedienteRef,
		ResolucionRef: anterior.resolucion.ResolucionRef, IntencionRef: "outbox:http-siguiente",
	}
	fecha := time.Date(2026, 9, 6, 10, 0, 0, 123456000, time.UTC)
	return &ejecutorContinuacionHTTPPrueba{
		ejecutorComunicacionLlamamientoHTTPPrueba: anterior,
		resultado: ports.ResultadoContinuacionLlamamiento{
			Solicitud: s, LlamamientoAnteriorRef: anterior.resolucion.Solicitud.LlamamientoRef,
			ReciboBolsa: ports.ReciboBolsaContinuacion{
				IntencionRef: s.IntencionRef, TerminalOperacionRef: "operacion:terminal-interna", OperacionRef: "operacion:siguiente-interna",
				LlamamientoRef: "llamamiento:http-siguiente", PropuestaRef: "propuesta:interna", ReciboRef: "recibo:bolsa-siguiente",
				AuditoriaRef: "auditoria:bolsa-interna", EventoRef: "evento:interno", RegistroSHA256: strings.Repeat("a", 64), ConfirmadaEn: fecha.Add(-time.Microsecond),
			},
			ReciboRef: "recibo:ct-siguiente", AuditoriaRef: "auditoria:ct-siguiente", ConfirmadaEn: fecha, Estado: "confirmado",
		},
	}
}

func comprobarErrorContinuacionHTTP(t *testing.T, r *httptest.ResponseRecorder, estado int, codigo string) {
	t.Helper()
	var salida struct {
		Data  json.RawMessage `json:"data"`
		Error struct {
			Codigo string `json:"codigo"`
			Clave  string `json:"clave_i18n"`
		} `json:"error"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if r.Code != estado || len(salida.Data) != 0 || salida.Error.Codigo != codigo ||
		salida.Error.Clave != "api.contratacion_temporal.comunicacion_llamamiento.error."+codigo {
		t.Fatalf("estado=%d cuerpo=%s", r.Code, r.Body)
	}
	if strings.Contains(r.Body.String(), "privado") || r.Header().Get("Set-Cookie") != "" ||
		!strings.Contains(r.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("respuesta con filtración o almacenamiento: %s", r.Body)
	}
}

func TestContinuacionLlamamientoHTTPContratoYReplay(t *testing.T) {
	e := nuevaContinuacionHTTPPrueba()
	h := nuevoManejadorComunicacionHTTPPrueba(t, e)
	const esperado = `{"data":{"esquema":"vec.contratacion-temporal.continuacion-llamamiento.v1","organizacion_ref":"organizacion:http-comunicacion","expediente_ref":"expediente:http-comunicacion","resolucion_ref":"resolucion:http-local","intencion_ref":"outbox:http-siguiente","llamamiento_anterior_ref":"llamamiento:http-comunicacion","llamamiento_ref":"llamamiento:http-siguiente","version_llamamiento":1,"recibo_bolsa_ref":"recibo:bolsa-siguiente","recibo_ref":"recibo:ct-siguiente","auditoria_ref":"auditoria:ct-siguiente","confirmada_en":"2026-09-06T10:00:00.123456Z","estado_intencion":"despachada","estado_local":"confirmado"}}`
	for _, replay := range []bool{false, true} {
		estado, cuerpo := http.StatusCreated, esperado
		if replay {
			e.resultado.Estado = "replay_confirmado"
			estado = http.StatusOK
			cuerpo = strings.Replace(esperado, `"estado_local":"confirmado"`, `"estado_local":"replay_confirmado"`, 1)
		}
		r := httptest.NewRecorder()
		h.ServeHTTP(r, peticionComunicacionHTTPPrueba(RutaContinuacionLlamamiento, cuerpoContinuacionHTTPPrueba))
		if r.Code != estado || r.Body.String() != cuerpo {
			t.Fatalf("estado=%d cuerpo=%s", r.Code, r.Body)
		}
		var salida map[string]map[string]json.RawMessage
		if err := json.Unmarshal(r.Body.Bytes(), &salida); err != nil || len(salida) != 1 || len(salida["data"]) != 14 {
			t.Fatalf("salida no consta exclusivamente de data con 14 campos: %s", r.Body)
		}
	}
	registros, resoluciones := e.totales()
	if len(e.solicitudes) != 2 || e.solicitudes[0] != e.resultado.Solicitud || e.solicitudes[1] != e.solicitudes[0] || registros != 0 || resoluciones != 0 {
		t.Fatal("material alterado o receptor equivocado")
	}
}

func TestContinuacionLlamamientoHTTPEntradaEstricta(t *testing.T) {
	valido := cuerpoContinuacionHTTPPrueba
	casos := []struct {
		nombre, cuerpo string
		estado         int
		codigo         string
	}{
		{"desconocido", strings.TrimSuffix(valido, "}") + `,"actor":"privado"}`, 400, "peticion_no_valida"},
		{"duplicado", strings.Replace(valido, `"intencion_ref":`, `"intencion_ref":"outbox:otra","intencion_ref":`, 1), 400, "peticion_no_valida"},
		{"segundo JSON", valido + `{}`, 400, "peticion_no_valida"},
		{"null", `null`, 400, "peticion_no_valida"},
		{"orden", strings.Replace(valido, `"resolucion_ref":"resolucion:http-local","intencion_ref":"outbox:http-siguiente"`, `"intencion_ref":"outbox:http-siguiente","resolucion_ref":"resolucion:http-local"`, 1), 422, "contenido_no_valido"},
		{"espacios", " " + valido, 422, "contenido_no_valido"},
		{"alias", strings.Replace(valido, "organizacion_ref", "Organizacion_ref", 1), 400, "peticion_no_valida"},
		{"UUID no v4", strings.Replace(valido, "5d2b-4c10", "5d2b-1c10", 1), 422, "contenido_no_valido"},
		{"referencia directa", strings.Replace(valido, "expediente:http-comunicacion", "persona@example.invalid", 1), 422, "contenido_no_valido"},
		{"limite 4KiB", valido + strings.Repeat(" ", MaximoCuerpoComunicacionLlamamientoBytes), 413, "peticion_demasiado_grande"},
	}
	campos := strings.Split(strings.TrimSuffix(strings.TrimPrefix(valido, "{"), "}"), ",")
	for i, campo := range campos {
		sinCampo := append([]string{}, campos[:i]...)
		sinCampo = append(sinCampo, campos[i+1:]...)
		clave := strings.SplitN(campo, ":", 2)[0]
		for _, variante := range []struct {
			nombre, cuerpo string
			estado         int
			codigo         string
		}{
			{"omitido", "{" + strings.Join(sinCampo, ",") + "}", 422, "contenido_no_valido"},
			{"nulo", strings.Replace(valido, campo, clave+":null", 1), 400, "peticion_no_valida"},
			{"tipo", strings.Replace(valido, campo, clave+":true", 1), 400, "peticion_no_valida"},
		} {
			variante.nombre = clave + " " + variante.nombre
			casos = append(casos, variante)
		}
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevaContinuacionHTTPPrueba()
			r := httptest.NewRecorder()
			nuevoManejadorComunicacionHTTPPrueba(t, e).ServeHTTP(r, peticionComunicacionHTTPPrueba(RutaContinuacionLlamamiento, caso.cuerpo))
			comprobarErrorContinuacionHTTP(t, r, caso.estado, caso.codigo)
			registros, resoluciones := e.totales()
			if len(e.solicitudes) != 0 || registros != 0 || resoluciones != 0 {
				t.Fatal("entrada inválida llegó al ejecutor")
			}
		})
	}
}

func TestContinuacionLlamamientoHTTPRechazaReciboAjenoOInvalido(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*ports.ResultadoContinuacionLlamamiento)
	}{
		{"clave", func(r *ports.ResultadoContinuacionLlamamiento) {
			r.Solicitud.ClaveIdempotencia = claveRegistroComunicacionHTTPPrueba
		}},
		{"organizacion", func(r *ports.ResultadoContinuacionLlamamiento) { r.Solicitud.OrganizacionRef = "organizacion:otra" }},
		{"expediente", func(r *ports.ResultadoContinuacionLlamamiento) { r.Solicitud.ExpedienteRef = "expediente:otro" }},
		{"resolucion", func(r *ports.ResultadoContinuacionLlamamiento) { r.Solicitud.ResolucionRef = "resolucion:otra" }},
		{"intencion", func(r *ports.ResultadoContinuacionLlamamiento) { r.Solicitud.IntencionRef = "outbox:otra" }},
		{"intencion Bolsa", func(r *ports.ResultadoContinuacionLlamamiento) { r.ReciboBolsa.IntencionRef = "outbox:otra" }},
		{"mismo llamamiento", func(r *ports.ResultadoContinuacionLlamamiento) {
			r.ReciboBolsa.LlamamientoRef = r.LlamamientoAnteriorRef
		}},
		{"antecesor ausente", func(r *ports.ResultadoContinuacionLlamamiento) { r.LlamamientoAnteriorRef = "" }},
		{"recibo CT ausente", func(r *ports.ResultadoContinuacionLlamamiento) { r.ReciboRef = "" }},
		{"auditoria CT ausente", func(r *ports.ResultadoContinuacionLlamamiento) { r.AuditoriaRef = "" }},
		{"recibo Bolsa ausente", func(r *ports.ResultadoContinuacionLlamamiento) { r.ReciboBolsa.ReciboRef = "" }},
		{"fecha cero", func(r *ports.ResultadoContinuacionLlamamiento) { r.ConfirmadaEn = time.Time{} }},
		{"fecha no UTC", func(r *ports.ResultadoContinuacionLlamamiento) {
			r.ConfirmadaEn = r.ConfirmadaEn.In(time.FixedZone("otra", 3600))
		}},
		{"nanosegundos", func(r *ports.ResultadoContinuacionLlamamiento) { r.ConfirmadaEn = r.ConfirmadaEn.Add(time.Nanosecond) }},
		{"antes de Bolsa", func(r *ports.ResultadoContinuacionLlamamiento) {
			r.ConfirmadaEn = r.ReciboBolsa.ConfirmadaEn.Add(-time.Microsecond)
		}},
		{"estado inventado", func(r *ports.ResultadoContinuacionLlamamiento) { r.Estado = "enviado" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevaContinuacionHTTPPrueba()
			caso.mutar(&e.resultado)
			r := httptest.NewRecorder()
			nuevoManejadorComunicacionHTTPPrueba(t, e).ServeHTTP(r, peticionComunicacionHTTPPrueba(RutaContinuacionLlamamiento, cuerpoContinuacionHTTPPrueba))
			comprobarErrorContinuacionHTTP(t, r, 502, "resultado_no_confiable")
			if len(e.solicitudes) != 1 {
				t.Fatal("no se comprobó el resultado del ejecutor")
			}
		})
	}
}

func TestContinuacionLlamamientoHTTPErroresSinExito(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
		codigo string
	}{
		{ports.ErrOperacionContinuacionInvalida, 422, "contenido_no_valido"},
		{ports.ErrOperacionContinuacionDenegada, 403, "acceso_denegado"},
		{ports.ErrOperacionContinuacionConflicto, 409, "clave_idempotencia_reutilizada"},
		{ports.ErrOperacionContinuacionNoDisponible, 503, "servicio_no_disponible"},
		{errors.New("postgres://privado"), 503, "servicio_no_disponible"},
		{context.Canceled, 408, "peticion_cancelada"},
		{context.DeadlineExceeded, 504, "plazo_agotado"},
	} {
		t.Run(caso.err.Error(), func(t *testing.T) {
			e := nuevaContinuacionHTTPPrueba()
			// Incluso si llega un recibo válido junto al error, no se publica.
			e.err = fmt.Errorf("detalle privado: %w", caso.err)
			r := httptest.NewRecorder()
			nuevoManejadorComunicacionHTTPPrueba(t, e).ServeHTTP(r, peticionComunicacionHTTPPrueba(RutaContinuacionLlamamiento, cuerpoContinuacionHTTPPrueba))
			comprobarErrorContinuacionHTTP(t, r, caso.estado, caso.codigo)
			if len(e.solicitudes) != 1 {
				t.Fatal("reintento automático o ejecutor omitido")
			}
		})
	}
}

func TestContinuacionLlamamientoHTTPNoPublicaExitoTrasCancelacion(t *testing.T) {
	e := nuevaContinuacionHTTPPrueba()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	e.durante = func(recibido context.Context) {
		if recibido != ctx {
			t.Fatal("se sustituyó el contexto HTTP")
		}
		cancelar()
	}
	r := httptest.NewRecorder()
	peticion := peticionComunicacionHTTPPrueba(RutaContinuacionLlamamiento, cuerpoContinuacionHTTPPrueba).WithContext(ctx)
	nuevoManejadorComunicacionHTTPPrueba(t, e).ServeHTTP(r, peticion)
	comprobarErrorContinuacionHTTP(t, r, 408, "peticion_cancelada")
}

func TestContinuacionLlamamientoHTTPReceptoresYRutaExacta(t *testing.T) {
	for _, caso := range []struct {
		nombre, ruta, cuerpo                    string
		estado                                  int
		registros, resoluciones, continuaciones int
	}{
		{"continuacion", RutaContinuacionLlamamiento, cuerpoContinuacionHTTPPrueba, 201, 0, 0, 1},
		{"registro anterior", RutaRegistroComunicacionLlamamiento, cuerpoRegistroComunicacionHTTPPrueba(), 201, 1, 0, 0},
		{"resolucion anterior", RutaResolucionComunicacionLlamamiento, cuerpoResolucionComunicacionHTTPPrueba(ports.RespuestaLlamamientoAceptada), 201, 0, 1, 0},
		{"barra", RutaContinuacionLlamamiento + "/", cuerpoContinuacionHTTPPrueba, 404, 0, 0, 0},
		{"query", RutaContinuacionLlamamiento + "?actor=privado", cuerpoContinuacionHTTPPrueba, 404, 0, 0, 0},
		{"continuacion no es registro", RutaRegistroComunicacionLlamamiento, cuerpoContinuacionHTTPPrueba, 400, 0, 0, 0},
		{"continuacion no es resolucion", RutaResolucionComunicacionLlamamiento, cuerpoContinuacionHTTPPrueba, 400, 0, 0, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevaContinuacionHTTPPrueba()
			r := httptest.NewRecorder()
			nuevoManejadorComunicacionHTTPPrueba(t, e).ServeHTTP(r, peticionComunicacionHTTPPrueba(caso.ruta, caso.cuerpo))
			registros, resoluciones := e.totales()
			if r.Code != caso.estado || registros != caso.registros || resoluciones != caso.resoluciones || len(e.solicitudes) != caso.continuaciones {
				t.Fatalf("estado=%d cuerpo=%s llamadas=%d/%d/%d", r.Code, r.Body, registros, resoluciones, len(e.solicitudes))
			}
		})
	}
}

func TestContinuacionLlamamientoHTTPSinInterfazDevuelve503(t *testing.T) {
	e := ejecutorComunicacionHTTPValidoPrueba()
	r := httptest.NewRecorder()
	nuevoManejadorComunicacionHTTPPrueba(t, e).ServeHTTP(r, peticionComunicacionHTTPPrueba(RutaContinuacionLlamamiento, cuerpoContinuacionHTTPPrueba))
	comprobarErrorContinuacionHTTP(t, r, 503, "servicio_no_disponible")
	registros, resoluciones := e.totales()
	if registros != 0 || resoluciones != 0 {
		t.Fatal("se desvió la continuación hacia un receptor anterior")
	}
}
