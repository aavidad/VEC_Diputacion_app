package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type autoridadHTTPB2Prueba struct {
	err      error
	llamadas int
	cancelar context.CancelFunc
}

func (a *autoridadHTTPB2Prueba) ResolverContextoIncorporacionEjercicioV2(context.Context) error {
	a.llamadas++
	if a.cancelar != nil {
		a.cancelar()
	}
	return a.err
}

type ejecutorHTTPB2Prueba struct {
	consultas, planes, confirmaciones int
	proyeccion                        ProyeccionIncorporacionPersonalB2HTTP
	recibo                            ReciboIncorporacionPersonalB2HTTP
	err                               error
	plan                              EntradaPlanB2
	confirmacion                      EntradaConfirmacionB2
	cancelar                          context.CancelFunc
}

func (e *ejecutorHTTPB2Prueba) Consultar(context.Context, string) (ProyeccionIncorporacionPersonalB2HTTP, error) {
	e.consultas++
	if e.cancelar != nil {
		e.cancelar()
	}
	return e.proyeccion, e.err
}
func (e *ejecutorHTTPB2Prueba) Preparar(_ context.Context, p EntradaPlanB2) (ProyeccionIncorporacionPersonalB2HTTP, error) {
	e.planes++
	e.plan = p
	return e.proyeccion, e.err
}
func (e *ejecutorHTTPB2Prueba) Confirmar(_ context.Context, p EntradaConfirmacionB2) (ReciboIncorporacionPersonalB2HTTP, error) {
	e.confirmaciones++
	e.confirmacion = p
	return e.recibo, e.err
}
func planHTTPB2Prueba() EntradaPlanB2 {
	return EntradaPlanB2{ExpedienteRef: "expediente:b2", VersionExpediente: 7, PuestoRef: "puesto:b2", PlazaRef: "plaza:b2", VersionPlantillaRef: "plantilla:v1", VersionRPTRef: "rpt:v1", Regimen: EntradaCatalogoB2{"regimen:funcionario", 1}, Modalidad: EntradaCatalogoB2{"modalidad:interino", 1}, ClaseOcupacion: "temporal", Desde: "2026-10-01", Hasta: "2027-10-01", MotivoClave: "incorporacion_confirmada", DocumentoRef: "documento:b2", DocumentoSHA256: strings.Repeat("a", 64), ClaveIdempotencia: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}
}
func proyeccionHTTPB2Prueba() ProyeccionIncorporacionPersonalB2HTTP {
	return ProyeccionIncorporacionPersonalB2HTTP{Esquema: EsquemaConsultaIncorporacionPersonalB2, ExpedienteRef: "expediente:b2", VersionExpedienteActual: 7, Estado: "sin_plan"}
}
func TestIncorporacionPersonalB2CatalogoProyectadoConProcedencia(t *testing.T) {
	v := proyeccionHTTPB2Prueba()
	v.Opciones.ClasesOcupacion = []OpcionClaseOcupacionB2{{Valor: "temporal", TextoClave: "personal.clases.temporal"}}
	if proyeccionHTTPB2Valida(v, v.ExpedienteRef) {
		t.Fatal("opciones sin procedencia admitidas")
	}
	v.Opciones.CatalogoClasesOcupacion = CatalogoClasesOcupacionB2{Ref: "personal:clases", Version: 3, HuellaSHA256: strings.Repeat("a", 64)}
	if !proyeccionHTTPB2Valida(v, v.ExpedienteRef) {
		t.Fatal("catálogo publicado rechazado")
	}
	v.Opciones.CatalogoClasesOcupacion.Version = 0
	if proyeccionHTTPB2Valida(v, v.ExpedienteRef) {
		t.Fatal("catálogo sin versión admitido")
	}
}
func reciboHTTPB2Prueba() ReciboIncorporacionPersonalB2HTTP {
	return ReciboIncorporacionPersonalB2HTTP{Esquema: EsquemaReciboIncorporacionPersonalB2, ExpedienteRef: "expediente:b2", PlanRef: "plan:b2", PlanVersion: 1, ReciboRef: "recibo:b2", RegistradaEn: "2026-10-01T12:00:00Z", EmpleadoRef: "empleado:b2", RelacionRef: "relacion:b2", OcupacionRef: "ocupacion:b2"}
}
func peticionHTTPB2Prueba(metodo, ruta, cuerpo string) *http.Request {
	var r *http.Request
	if cuerpo == "" {
		r = httptest.NewRequest(metodo, ruta, nil)
	} else {
		r = httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
	}
	r.Header.Set("Accept", "application/json")
	return r
}
func ejecutarHTTPB2Prueba(t *testing.T, a *autoridadHTTPB2Prueba, e *ejecutorHTTPB2Prueba, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	h, err := NuevoManejadorIncorporacionPersonalB2(a, e)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func jsonHTTPB2Prueba(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func TestIncorporacionPersonalB2ConsultaSoloLecturaYPlanConfirmacion(t *testing.T) {
	a := &autoridadHTTPB2Prueba{}
	e := &ejecutorHTTPB2Prueba{proyeccion: proyeccionHTTPB2Prueba()}
	w := ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba("GET", RutaPlanB2+"?expediente_ref=expediente:b2", ""))
	if w.Code != 200 || a.llamadas != 1 || e.consultas != 1 || e.planes != 0 || e.confirmaciones != 0 {
		t.Fatalf("consulta status=%d llamadas=%+v", w.Code, e)
	}
	if strings.Contains(w.Body.String(), "null") && strings.Contains(w.Body.String(), `"vacantes":null`) {
		t.Fatal("listas vacías no normalizadas")
	}
	p := planHTTPB2Prueba()
	e.proyeccion.Estado = "plan_preparado"
	e.proyeccion.Plan = &PlanIncorporacionPersonalB2HTTP{"plan:b2", 1, strings.Repeat("a", 64), p}
	w = ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba("POST", RutaPlanB2, jsonHTTPB2Prueba(t, p)))
	if w.Code != 200 || e.planes != 1 || e.plan != p || e.confirmaciones != 0 {
		t.Fatalf("plan status=%d", w.Code)
	}
	e.recibo = reciboHTTPB2Prueba()
	c := EntradaConfirmacionB2{p.ExpedienteRef, "plan:b2", 1, p.ClaveIdempotencia}
	w = ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba("POST", RutaConfirmacionB2, jsonHTTPB2Prueba(t, c)))
	if w.Code != 200 || e.confirmaciones != 1 || e.confirmacion != c || e.planes != 1 {
		t.Fatalf("confirmacion status=%d", w.Code)
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store, no-transform" {
		t.Fatal("cabeceras de privacidad incorrectas")
	}
}
func TestIncorporacionPersonalB2RechazaAutoridadYPollutionCliente(t *testing.T) {
	base := jsonHTTPB2Prueba(t, planHTTPB2Prueba())
	for _, campo := range []string{"persona_ref", "empleado_ref", "actor", "sesion_ref", "capacidad_ref", "recibo_ref", "source", "organizacion_ref", "firma_oficial"} {
		t.Run(campo, func(t *testing.T) {
			a := &autoridadHTTPB2Prueba{}
			e := &ejecutorHTTPB2Prueba{}
			b := strings.TrimSuffix(base, "}") + `,"` + campo + `":"inventado"}`
			w := ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba("POST", RutaPlanB2, b))
			if w.Code != 400 || a.llamadas != 0 || e.planes != 0 {
				t.Fatalf("status=%d", w.Code)
			}
		})
	}
	casos := []string{
		strings.Replace(base, `"expediente_ref":"expediente:b2"`, `"expediente_ref":"expediente:b2","expediente_ref":"expediente:ajeno"`, 1),
		strings.Replace(base, `"ref":"regimen:funcionario"`, `"ref":"regimen:funcionario","ref":"regimen:otro"`, 1),
		strings.Replace(base, `"ref":"regimen:funcionario"`, `"ref":"regimen:funcionario","VERSION":1`, 1),
		strings.Replace(base, `"version_expediente":7`, `"version_expediente":7e0`, 1),
		strings.Replace(base, `"hasta":"2027-10-01"`, `"hasta":null`, 1),
		base + base, strings.Repeat(" ", 8193) + base,
	}
	for i, b := range casos {
		a := &autoridadHTTPB2Prueba{}
		e := &ejecutorHTTPB2Prueba{}
		w := ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba("POST", RutaPlanB2, b))
		if w.Code != 400 || a.llamadas != 0 || e.planes != 0 {
			t.Fatalf("caso=%d status=%d", i, w.Code)
		}
	}
}
func TestIncorporacionPersonalB2MetodoRutaQueryCabeceras(t *testing.T) {
	for _, caso := range []struct {
		metodo, ruta, cuerpo string
		status               int
	}{
		{"GET", RutaConfirmacionB2, "", 405}, {"PUT", RutaPlanB2, "", 405},
		{"GET", RutaPlanB2 + "/", "", 400}, {"GET", RutaPlanB2 + "?expediente_ref=expediente:b2&expediente_ref=expediente:otro", "", 400},
		{"GET", RutaPlanB2 + "?expediente_ref=expediente:b2&actor=actor:otro", "", 400},
		{"POST", RutaPlanB2 + "?expediente_ref=expediente:b2", jsonHTTPB2Prueba(t, planHTTPB2Prueba()), 400},
		{"GET", RutaPlanB2 + "?expediente_ref=expediente:b2", "{}", 400},
	} {
		a := &autoridadHTTPB2Prueba{}
		e := &ejecutorHTTPB2Prueba{}
		w := ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba(caso.metodo, caso.ruta, caso.cuerpo))
		if w.Code != caso.status || a.llamadas != 0 {
			t.Fatalf("ruta=%s status=%d", caso.ruta, w.Code)
		}
	}
	for _, nombre := range []string{"Authorization", "Cookie", "X-Actor", "X-Correlation-ID"} {
		a := &autoridadHTTPB2Prueba{}
		e := &ejecutorHTTPB2Prueba{}
		r := peticionHTTPB2Prueba("GET", RutaPlanB2+"?expediente_ref=expediente:b2", "")
		r.Header.Set(nombre, "libre")
		w := ejecutarHTTPB2Prueba(t, a, e, r)
		if w.Code != 400 || a.llamadas != 0 {
			t.Fatalf("cabecera=%s status=%d", nombre, w.Code)
		}
	}
}
func TestIncorporacionPersonalB2DenegacionConflictoDependenciaYMinimizacion(t *testing.T) {
	for _, caso := range []struct {
		err    error
		status int
		codigo string
	}{
		{ErrDenegadaIncorporacionPersonalB2, 403, "acceso_denegado"}, {ErrConflictoIncorporacionPersonalB2, 409, "conflicto"}, {ErrPreparacionPendienteIncorporacionPersonalB2, 409, "preparacion_pendiente"}, {errors.New("secreto:contenido-privado"), 503, "servicio_no_disponible"},
	} {
		a := &autoridadHTTPB2Prueba{}
		e := &ejecutorHTTPB2Prueba{err: caso.err}
		w := ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba("GET", RutaPlanB2+"?expediente_ref=expediente:b2", ""))
		if w.Code != caso.status || !strings.Contains(w.Body.String(), caso.codigo) || strings.Contains(w.Body.String(), "secreto") || strings.Contains(w.Body.String(), "expediente:b2") {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	a := &autoridadHTTPB2Prueba{err: ErrDenegadaIncorporacionPersonalB2}
	e := &ejecutorHTTPB2Prueba{proyeccion: proyeccionHTTPB2Prueba()}
	w := ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba("GET", RutaPlanB2+"?expediente_ref=expediente:b2", ""))
	if w.Code != 403 || e.consultas != 0 {
		t.Fatal("se invocó servicio tras denegación")
	}
}
func TestIncorporacionPersonalB2NoExponeSalidaAjenaParcialNiFirma(t *testing.T) {
	for _, caso := range []string{"ajena", "parcial", "firma"} {
		a := &autoridadHTTPB2Prueba{}
		e := &ejecutorHTTPB2Prueba{proyeccion: proyeccionHTTPB2Prueba()}
		switch caso {
		case "ajena":
			e.proyeccion.ExpedienteRef = "expediente:ajeno"
		case "parcial":
			e.err = ErrDenegadaIncorporacionPersonalB2
		case "firma":
			e.proyeccion.Estado = "incorporacion_confirmada"
			e.proyeccion.Plan = &PlanIncorporacionPersonalB2HTTP{"plan:b2", 1, strings.Repeat("a", 64), planHTTPB2Prueba()}
			r := reciboHTTPB2Prueba()
			r.FirmaOficial = true
			e.proyeccion.Recibo = &r
		}
		w := ejecutarHTTPB2Prueba(t, a, e, peticionHTTPB2Prueba("GET", RutaPlanB2+"?expediente_ref=expediente:b2", ""))
		if w.Code != 503 || strings.Contains(w.Body.String(), "data") || strings.Contains(w.Body.String(), "expediente:ajeno") {
			t.Fatalf("caso=%s status=%d", caso, w.Code)
		}
	}
}
func TestIncorporacionPersonalB2CancelacionAntesYDespuesAutoridadServicio(t *testing.T) {
	for _, fase := range []string{"antes", "autoridad", "servicio"} {
		ctx, cancelar := context.WithCancel(context.Background())
		a := &autoridadHTTPB2Prueba{}
		e := &ejecutorHTTPB2Prueba{proyeccion: proyeccionHTTPB2Prueba()}
		switch fase {
		case "antes":
			cancelar()
		case "autoridad":
			a.cancelar = cancelar
		case "servicio":
			e.cancelar = cancelar
		}
		r := peticionHTTPB2Prueba("GET", RutaPlanB2+"?expediente_ref=expediente:b2", "").WithContext(ctx)
		w := ejecutarHTTPB2Prueba(t, a, e, r)
		cancelar()
		if w.Code != 503 || (fase != "servicio" && e.consultas != 0) || strings.Contains(w.Body.String(), "data") {
			t.Fatalf("fase=%s status=%d", fase, w.Code)
		}
	}
}
func TestIncorporacionPersonalB2DependenciasNulas(t *testing.T) {
	var a *autoridadHTTPB2Prueba
	var e *ejecutorHTTPB2Prueba
	if _, err := NuevoManejadorIncorporacionPersonalB2(a, &ejecutorHTTPB2Prueba{}); err == nil {
		t.Fatal("autoridad nil admitida")
	}
	if _, err := NuevoManejadorIncorporacionPersonalB2(&autoridadHTTPB2Prueba{}, e); err == nil {
		t.Fatal("servicio nil admitido")
	}
}
