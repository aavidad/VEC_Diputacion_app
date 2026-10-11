package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorVinculoRPTPrueba struct {
	consultas, registros int
	entrada              EntradaVinculoCategoriaRPTB2
	lectura              ports.LecturaVinculoCategoriaRPT
	recibo               ports.ReciboVinculoCategoriaRPT
	err                  error
}

func (e *ejecutorVinculoRPTPrueba) ConsultarVinculo(context.Context, string) (ports.LecturaVinculoCategoriaRPT, error) {
	e.consultas++
	return e.lectura, e.err
}
func (e *ejecutorVinculoRPTPrueba) RegistrarVinculo(_ context.Context, x EntradaVinculoCategoriaRPTB2) (ports.ReciboVinculoCategoriaRPT, error) {
	e.registros++
	e.entrada = x
	return e.recibo, e.err
}

func entradaVinculoRPTPrueba() EntradaVinculoCategoriaRPTB2 {
	return EntradaVinculoCategoriaRPTB2{ExpedienteRef: "expediente:b2", VersionExpedienteEsperada: 3, AnalisisVersion: 2,
		AnalisisReciboRef: "recibo:analisis", AnalisisHuellaSHA256: strings.Repeat("a", 64), CategoriaRef: "auxiliar_administrativo",
		CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), CategoriaID: "auxiliar_administrativo",
		FuenteRef: "fuente:rpt", MotivoRef: "motivo:vinculo", AprobacionRef: "aprobacion:rrhh",
		ClaveIdempotencia: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}
}

func estadoVinculoRPTPrueba() ports.EstadoVinculoCategoriaRPT {
	return ports.EstadoVinculoCategoriaRPT{Revision: 1, ReciboRef: "recibo:vinculo", CatalogoID: "categorias_rpt", ModuloID: "personal",
		CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), CategoriaID: "auxiliar_administrativo",
		FuenteRef: "fuente:rpt", MotivoRef: "motivo:vinculo", AprobacionRef: "aprobacion:rrhh", Prospectivo: true}
}

func reciboVinculoRPTPrueba() ports.ReciboVinculoCategoriaRPT {
	return ports.ReciboVinculoCategoriaRPT{ReciboRef: "recibo:vinculo", RegistradoEn: time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
		Revision: 1, Prospectivo: true, DecisionRef: "decision:interna", AuditoriaRef: "auditoria:interna",
		RPTPerfilRef: "prf_interno", Vinculo: estadoVinculoRPTPrueba()}
}

func servirVinculoRPTPrueba(t *testing.T, e *ejecutorVinculoRPTPrueba, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	h, err := NuevoManejadorVinculoCategoriaRPTB2(e)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// El registro devuelve el recibo sin referencias internas de decisión,
// auditoría ni perfil, y pasa al ejecutor exactamente la intención recibida.
func TestVinculoRPTB2RegistroDevuelveReciboMinimizado(t *testing.T) {
	e := &ejecutorVinculoRPTPrueba{recibo: reciboVinculoRPTPrueba()}
	entrada := entradaVinculoRPTPrueba()
	w := servirVinculoRPTPrueba(t, e, peticionHTTPB2Prueba(http.MethodPost, RutaVinculoCategoriaRPTB2, jsonHTTPB2Prueba(t, entrada)))
	if w.Code != 200 || e.registros != 1 || e.entrada != entrada {
		t.Fatalf("HTTP %d registros=%d: %s", w.Code, e.registros, w.Body.String())
	}
	for _, interno := range []string{"decision:interna", "auditoria:interna", "prf_interno", "decision_ref", "auditoria_ref"} {
		if strings.Contains(w.Body.String(), interno) {
			t.Fatalf("respuesta con dato interno %q", interno)
		}
	}
	var out struct {
		Data ReciboVinculoCategoriaRPTB2HTTP `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Data.ReciboRef != "recibo:vinculo" || out.Data.Revision != 1 || out.Data.ExpedienteRef != entrada.ExpedienteRef || !out.Data.Prospectivo {
		t.Fatalf("recibo: %s", w.Body.String())
	}
}

// Campos de más o de menos son 400; contenido incoherente es 422; en ningún
// caso se llama al servicio.
func TestVinculoRPTB2RechazaEntradaSinLlamarAlServicio(t *testing.T) {
	sobra := map[string]any{}
	_ = json.Unmarshal([]byte(jsonHTTPB2Prueba(t, entradaVinculoRPTPrueba())), &sobra)
	sobra["organizacion_ref"] = "organizacion:ajena"
	falta := map[string]any{}
	_ = json.Unmarshal([]byte(jsonHTTPB2Prueba(t, entradaVinculoRPTPrueba())), &falta)
	delete(falta, "clave_idempotencia")
	anterior := "recibo:previo"
	incoherentes := []func(*EntradaVinculoCategoriaRPTB2){
		func(x *EntradaVinculoCategoriaRPTB2) { x.AnteriorReciboRef = anterior },
		func(x *EntradaVinculoCategoriaRPTB2) { x.RevisionEsperada = 1 },
		func(x *EntradaVinculoCategoriaRPTB2) { x.CategoriaRef = "otra_categoria" },
		func(x *EntradaVinculoCategoriaRPTB2) { x.ClaveIdempotencia = "no-uuid" },
		func(x *EntradaVinculoCategoriaRPTB2) { x.AnalisisHuellaSHA256 = "corta" },
	}
	for _, c := range []struct {
		cuerpo string
		estado int
	}{{jsonHTTPB2Prueba(t, sobra), 400}, {jsonHTTPB2Prueba(t, falta), 400}} {
		e := &ejecutorVinculoRPTPrueba{}
		if w := servirVinculoRPTPrueba(t, e, peticionHTTPB2Prueba(http.MethodPost, RutaVinculoCategoriaRPTB2, c.cuerpo)); w.Code != c.estado || e.registros != 0 {
			t.Fatalf("HTTP %d, se esperaba %d", w.Code, c.estado)
		}
	}
	for i, cambiar := range incoherentes {
		x := entradaVinculoRPTPrueba()
		cambiar(&x)
		e := &ejecutorVinculoRPTPrueba{}
		if w := servirVinculoRPTPrueba(t, e, peticionHTTPB2Prueba(http.MethodPost, RutaVinculoCategoriaRPTB2, jsonHTTPB2Prueba(t, x))); w.Code != 422 || e.registros != 0 {
			t.Fatalf("caso %d: HTTP %d", i, w.Code)
		}
	}
	e := &ejecutorVinculoRPTPrueba{}
	if w := servirVinculoRPTPrueba(t, e, peticionHTTPB2Prueba(http.MethodPut, RutaVinculoCategoriaRPTB2, "")); w.Code != 405 || w.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("PUT: HTTP %d", w.Code)
	}
}

// Los errores nominales del servicio llegan como 403, 409 o 503 sin detalle.
func TestVinculoRPTB2TraduceErroresDelServicio(t *testing.T) {
	for _, c := range []struct {
		err    error
		estado int
		codigo string
	}{
		{ErrDenegadaIncorporacionPersonalB2, 403, "acceso_denegado"},
		{ErrConflictoIncorporacionPersonalB2, 409, "conflicto"},
		{ErrPreparacionPendienteIncorporacionPersonalB2, 409, "preparacion_pendiente"},
		{ErrPeticionIncorporacionPersonalB2, 422, "contenido_no_valido"},
		{ErrManejadorIncorporacionPersonalB2, 503, "servicio_no_disponible"},
	} {
		e := &ejecutorVinculoRPTPrueba{err: c.err}
		w := servirVinculoRPTPrueba(t, e, peticionHTTPB2Prueba(http.MethodPost, RutaVinculoCategoriaRPTB2, jsonHTTPB2Prueba(t, entradaVinculoRPTPrueba())))
		if w.Code != c.estado || !strings.Contains(w.Body.String(), `"codigo":"`+c.codigo+`"`) {
			t.Fatalf("%v: HTTP %d %s", c.err, w.Code, w.Body.String())
		}
	}
	// Un recibo que no corresponde a la intención no se presenta como éxito.
	r := reciboVinculoRPTPrueba()
	r.Revision = 2
	e := &ejecutorVinculoRPTPrueba{recibo: r}
	if w := servirVinculoRPTPrueba(t, e, peticionHTTPB2Prueba(http.MethodPost, RutaVinculoCategoriaRPTB2, jsonHTTPB2Prueba(t, entradaVinculoRPTPrueba()))); w.Code != 503 {
		t.Fatalf("recibo incoherente: HTTP %d", w.Code)
	}
}

func TestVinculoRPTB2ConsultaDevuelveAnclajeYVinculo(t *testing.T) {
	estado := estadoVinculoRPTPrueba()
	e := &ejecutorVinculoRPTPrueba{lectura: ports.LecturaVinculoCategoriaRPT{OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:b2",
		Analisis: ports.AnclajeAnalisisCategoriaRPT{VersionExpediente: 3, AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis",
			AnalisisHuellaSHA256: strings.Repeat("a", 64), CategoriaRef: "auxiliar_administrativo"}, Vinculo: &estado}}
	w := servirVinculoRPTPrueba(t, e, peticionHTTPB2Prueba(http.MethodGet, RutaVinculoCategoriaRPTB2+"?expediente_ref=expediente:b2", ""))
	if w.Code != 200 || e.consultas != 1 || !strings.Contains(w.Body.String(), `"analisis_huella_sha256"`) {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	for _, consulta := range []string{"", "?expediente_ref=expediente:b2&otra=1", "?expediente_ref=expediente:otro"} {
		e := &ejecutorVinculoRPTPrueba{lectura: e.lectura}
		w := servirVinculoRPTPrueba(t, e, peticionHTTPB2Prueba(http.MethodGet, RutaVinculoCategoriaRPTB2+consulta, ""))
		if w.Code == 200 {
			t.Fatalf("consulta %q aceptada", consulta)
		}
	}
}
