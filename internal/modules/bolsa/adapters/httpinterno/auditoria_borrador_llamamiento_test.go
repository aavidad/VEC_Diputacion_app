package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type registradorIntentoBorradorDoble struct {
	intentos      []puertosbolsa.IntentoBorradorLlamamiento
	err           error
	ctx           context.Context
	errEnRegistro error
}

func (r *registradorIntentoBorradorDoble) RegistrarIntentoBorradorLlamamiento(ctx context.Context, intento puertosbolsa.IntentoBorradorLlamamiento) error {
	r.ctx = ctx
	r.errEnRegistro = ctx.Err()
	r.intentos = append(r.intentos, intento)
	return r.err
}

type actorVerificadoIntentoBorradorDoble struct{ actor string }

func (a actorVerificadoIntentoBorradorDoble) ActorVerificadoParaAuditoriaBorradorLlamamiento(context.Context) (string, bool) {
	return a.actor, true
}

func nuevaAuditoriaBorradorPrueba(t *testing.T, siguiente http.Handler, registrador *registradorIntentoBorradorDoble, actor ResolutorActorVerificadoBorradorLlamamiento) http.Handler {
	t.Helper()
	h, err := NuevaAuditoriaBorradorLlamamiento(siguiente, registrador, generadorCorrelacionBorradorLlamamientoPrueba{}, actor)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestAuditoriaBorradorLlamamientoRegistraDenegacionTempranaFueraDeProteccion(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		aplicarCabeceras(w)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"codigo":"acceso_denegado"}}`))
	}), registrador, actorVerificadoIntentoBorradorDoble{actor: "per_0123456789abcdefghijkl"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCrearBorradorLlamamiento())
	if w.Code != http.StatusForbidden || len(registrador.intentos) != 1 {
		t.Fatalf("denegacion perdida: estado=%d intentos=%#v", w.Code, registrador.intentos)
	}
	i := registrador.intentos[0]
	if i.Accion != puertosbolsa.AccionIntentoCrearBorradorLlamamiento || i.ClaseRuta != puertosbolsa.ClaseRutaColeccionBorradorLlamamiento || i.Resultado != puertosbolsa.ResultadoIntentoAccesoDenegadoBorradorLlamamiento || i.ActorVerificado != "per_0123456789abcdefghijkl" || i.Correlacion.Validar() != nil {
		t.Fatalf("intento temprano no nominal: %#v", i)
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("cabecera segura no conservada: %#v", w.Header())
	}
}

func TestAuditoriaBorradorLlamamientoRegistraFalloTardioSinCambiarSuRespuesta(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"codigo":"servicio_no_disponible"}}`))
	}), registrador, nil)
	ref := "borrador-llamamiento:alta:" + strings.Repeat("a", 64)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaBorradoresLlamamiento+"/"+ref, nil))
	if w.Code != http.StatusServiceUnavailable || len(registrador.intentos) != 1 || registrador.intentos[0].Accion != puertosbolsa.AccionIntentoConsultarBorradorLlamamiento || registrador.intentos[0].Resultado != puertosbolsa.ResultadoIntentoInfraestructuraNoDisponibleBorradorLlamamiento {
		t.Fatalf("fallo tardio no auditado: estado=%d intentos=%#v", w.Code, registrador.intentos)
	}
	if !strings.Contains(w.Body.String(), "servicio_no_disponible") || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("respuesta tardia alterada: %#v %q", w.Header(), w.Body.String())
	}
}

func TestAuditoriaBorradorLlamamientoRegistraDenegacionDeSituacionB2(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}), registrador, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/situacion", strings.NewReader(`{}`))
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || len(registrador.intentos) != 1 || registrador.intentos[0].Accion != puertosbolsa.AccionIntentoCambiarSituacionParticipacion || registrador.intentos[0].ClaseRuta != puertosbolsa.ClaseRutaSituacionParticipacion || registrador.intentos[0].Resultado != puertosbolsa.ResultadoIntentoAccesoDenegadoBorradorLlamamiento {
		t.Fatalf("denegación B2 no auditada: estado=%d intentos=%#v", w.Code, registrador.intentos)
	}
}

func TestAuditoriaOperacionesB8RegistraGETyPOST(t *testing.T) {
	ruta := RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/operaciones"
	for _, caso := range []struct {
		metodo string
		accion puertosbolsa.AccionIntentoBorradorLlamamiento
	}{{http.MethodGet, puertosbolsa.AccionIntentoConsultarBorradorLlamamiento}, {http.MethodPost, puertosbolsa.AccionIntentoCambiarSituacionParticipacion}} {
		registrador := &registradorIntentoBorradorDoble{}
		h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }), registrador, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(caso.metodo, ruta, nil))
		if w.Code != http.StatusForbidden || len(registrador.intentos) != 1 || registrador.intentos[0].Accion != caso.accion || registrador.intentos[0].ClaseRuta != puertosbolsa.ClaseRutaSituacionParticipacion || registrador.intentos[0].Resultado != puertosbolsa.ResultadoIntentoAccesoDenegadoBorradorLlamamiento {
			t.Fatalf("B8 %s no auditado: estado=%d intentos=%#v", caso.metodo, w.Code, registrador.intentos)
		}
	}
}

func TestAuditoriaBorradorLlamamientoIncluyeFronterasB4YB7(t *testing.T) {
	casos := []struct {
		metodo, ruta string
		accion       puertosbolsa.AccionIntentoBorradorLlamamiento
		clase        puertosbolsa.ClaseRutaIntentoBorradorLlamamiento
	}{
		{http.MethodGet, RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/datos-contacto", puertosbolsa.AccionIntentoConsultarDatosContactoParticipacion, puertosbolsa.ClaseRutaDatosContactoParticipacion},
		{http.MethodPost, RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/datos-contacto", puertosbolsa.AccionIntentoRegistrarDatosContactoParticipacion, puertosbolsa.ClaseRutaDatosContactoParticipacion},
		{http.MethodGet, RutaEmisionesLlamamiento + "?bolsa_ref=bolsa:01&clave_idempotencia=clave-b7", puertosbolsa.AccionIntentoRecuperarLlamamiento, puertosbolsa.ClaseRutaEmisionesLlamamiento},
		{http.MethodPost, RutaEmisionesLlamamiento, puertosbolsa.AccionIntentoEmitirLlamamiento, puertosbolsa.ClaseRutaEmisionesLlamamiento},
	}
	for _, caso := range casos {
		registrador := &registradorIntentoBorradorDoble{}
		h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }), registrador, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if len(registrador.intentos) != 1 || registrador.intentos[0].Accion != caso.accion || registrador.intentos[0].ClaseRuta != caso.clase {
			t.Fatalf("frontera %s %s no auditada: %#v", caso.metodo, caso.ruta, registrador.intentos)
		}
	}
}

func TestAuditoriaBorradorLlamamientoFallaCerradoSiNoPuedePersistir(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{err: errors.New("base no disponible")}
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`detalle interno que no debe salir`))
	}), registrador, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCrearBorradorLlamamiento())
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "servicio_no_disponible") || strings.Contains(w.Body.String(), "detalle interno") {
		t.Fatalf("fallo de auditoria no cerro respuesta: %d %q", w.Code, w.Body.String())
	}
}

func TestAuditoriaBorradorLlamamientoAcotaCuerpoYAuditaSinFiltrarlo(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", maximoRespuestaAuditableBorradorLlamamientoBytes+1)))
	}), registrador, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCrearBorradorLlamamiento())
	if w.Code != http.StatusServiceUnavailable || len(registrador.intentos) != 1 || registrador.intentos[0].Resultado != puertosbolsa.ResultadoIntentoInfraestructuraNoDisponibleBorradorLlamamiento || strings.Contains(w.Body.String(), "xxxxx") {
		t.Fatalf("cuerpo excesivo no contenido: estado=%d intento=%#v cuerpo=%q", w.Code, registrador.intentos, w.Body.String())
	}
}

func TestAuditoriaBorradorLlamamientoHEADDelega405SinAuditoria(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	llamadas := 0
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}), registrador, nil)
	ref := "borrador-llamamiento:alta:" + strings.Repeat("a", 64)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/"+ref, nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != http.MethodGet || llamadas != 1 || len(registrador.intentos) != 0 {
		t.Fatalf("HEAD no se delego intacto: codigo=%d allow=%q llamadas=%d intentos=%#v", w.Code, w.Header().Get("Allow"), llamadas, registrador.intentos)
	}
}

func TestRechazoHEADExteriorEvitaAuditoriaYCadenaFuncional(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	llamadas := 0
	auditoria := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusTeapot)
	}), registrador, nil)
	h, err := EnvolverRechazoHEADDetalleBorradorLlamamiento(auditoria)
	if err != nil {
		t.Fatal(err)
	}
	ref := "borrador-llamamiento:alta:" + strings.Repeat("a", 64)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodHead, RutaBorradoresLlamamiento+"/"+ref, nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != http.MethodGet || w.Body.Len() != 0 || llamadas != 0 || len(registrador.intentos) != 0 {
		t.Fatalf("HEAD alcanzó auditoría o cadena: codigo=%d allow=%q cuerpo=%q llamadas=%d intentos=%#v", w.Code, w.Header().Get("Allow"), w.Body.String(), llamadas, registrador.intentos)
	}
}

func TestAuditoriaBorradorLlamamientoGETNoEmiteCookieNiTrailer(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Set-Cookie", "secreto=prohibido")
		w.Header().Set("Trailer", "X-Interno")
		w.Header().Set("X-Interno", "prohibido")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}), registrador, nil)
	ref := "borrador-llamamiento:alta:" + strings.Repeat("a", 64)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaBorradoresLlamamiento+"/"+ref, nil))
	if w.Code != http.StatusOK || w.Header().Get("Set-Cookie") != "" || w.Header().Get("Trailer") != "" || w.Header().Get("X-Interno") != "" || len(registrador.intentos) != 1 || registrador.intentos[0].Resultado != puertosbolsa.ResultadoIntentoCorrectoBorradorLlamamiento || registrador.intentos[0].ActorVerificado != "" {
		t.Fatalf("GET filtra estado: codigo=%d cabeceras=%#v intentos=%#v", w.Code, w.Header(), registrador.intentos)
	}
}

func TestAuditoriaBorradorLlamamientoNoInterfiereConRutaAjena(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	cuerpo := strings.Repeat("x", maximoRespuestaAuditableBorradorLlamamientoBytes+1)
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Set-Cookie", "ajena=conservada")
		w.Header().Set("Trailer", "X-Ajena")
		w.Header().Set("X-Ajena", "conservada")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(cuerpo))
	}), registrador, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/vec/ajena", nil))
	if w.Code != http.StatusTeapot || w.Body.String() != cuerpo || w.Header().Get("Set-Cookie") != "ajena=conservada" || w.Header().Get("Trailer") != "X-Ajena" || w.Header().Get("X-Ajena") != "conservada" || len(registrador.intentos) != 0 {
		t.Fatalf("ruta ajena modificada: codigo=%d cuerpo=%d cabeceras=%#v intentos=%#v", w.Code, w.Body.Len(), w.Header(), registrador.intentos)
	}
}

func TestAuditoriaBorradorLlamamientoRegistraFalloTrasCancelacionDuranteHandler(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancelar()
		if r.Context().Err() == nil {
			t.Fatal("el handler no observo la cancelacion")
		}
		w.WriteHeader(http.StatusForbidden)
	}), registrador, nil)
	r := peticionCrearBorradorLlamamiento().WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || len(registrador.intentos) != 1 || registrador.ctx == nil || registrador.errEnRegistro != nil {
		t.Fatalf("cancelacion durante handler no cerro ni audito: codigo=%d intentos=%#v err_en_registro=%v", w.Code, registrador.intentos, registrador.errEnRegistro)
	}
	limite, acotado := registrador.ctx.Deadline()
	if !acotado || time.Until(limite) <= 0 || time.Until(limite) > tiempoMaximoAuditoriaBorradorLlamamiento {
		t.Fatalf("contexto de auditoria sin limite corto: acotado=%t limite=%s", acotado, limite)
	}
}

func TestNuevaAuditoriaBorradorLlamamientoRechazaDependenciasNulas(t *testing.T) {
	registrador := &registradorIntentoBorradorDoble{}
	if h, err := NuevaAuditoriaBorradorLlamamiento(nil, registrador, generadorCorrelacionBorradorLlamamientoPrueba{}, nil); h != nil || !errors.Is(err, ErrAuditoriaBorradorLlamamientoInvalida) {
		t.Fatalf("handler nulo aceptado: %#v %v", h, err)
	}
}
