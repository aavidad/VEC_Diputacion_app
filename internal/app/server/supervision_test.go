package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type emisorIncidenciasPrueba struct {
	mu          sync.Mutex
	solicitudes []domain.SolicitudIncidenciaTecnica
}

func (e *emisorIncidenciasPrueba) Emitir(s domain.SolicitudIncidenciaTecnica) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.solicitudes = append(e.solicitudes, s)
}

func (e *emisorIncidenciasPrueba) codigos() []domain.CodigoIncidenciaTecnica {
	e.mu.Lock()
	defer e.mu.Unlock()
	codigos := make([]domain.CodigoIncidenciaTecnica, 0, len(e.solicitudes))
	for _, s := range e.solicitudes {
		codigos = append(codigos, s.Codigo)
		if _, saneada := domain.ClasificarIncidenciaTecnica(s); saneada {
			codigos = append(codigos, "NO_CATALOGADA")
		}
	}
	return codigos
}

func TestSupervisarRespuestasRegistra5xxSinCambiarLaRespuesta(t *testing.T) {
	for _, estado := range []int{http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusBadGateway} {
		emisor := &emisorIncidenciasPrueba{}
		original := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Prueba", "conservada")
			w.WriteHeader(estado)
			_, _ = w.Write([]byte(`{"error":"servicio_no_disponible"}`))
		})
		directa := httptest.NewRecorder()
		original.ServeHTTP(directa, httptest.NewRequest(http.MethodGet, "/api/vec/x?dni=12345678Z", nil))
		supervisada := httptest.NewRecorder()
		SupervisarRespuestas(original, emisor).ServeHTTP(supervisada, httptest.NewRequest(http.MethodGet, "/api/vec/x?dni=12345678Z", nil))

		if supervisada.Code != directa.Code || supervisada.Body.String() != directa.Body.String() {
			t.Fatalf("estado %d: el middleware cambió la respuesta: %d %q frente a %d %q", estado, supervisada.Code, supervisada.Body.String(), directa.Code, directa.Body.String())
		}
		for nombre, valores := range directa.Header() {
			if strings.Join(supervisada.Header()[nombre], ",") != strings.Join(valores, ",") {
				t.Fatalf("estado %d: cabecera %s alterada", estado, nombre)
			}
		}
		if got := emisor.codigos(); len(got) != 1 || got[0] != domain.IncidenciaHTTPInternoFallido {
			t.Fatalf("estado %d: incidencias = %v, se esperaba una HTTP_INTERNO_FALLIDO catalogada", estado, got)
		}
	}
}

func TestSupervisarRespuestasNoRegistraRespuestasCorrectasNiDeCliente(t *testing.T) {
	emisor := &emisorIncidenciasPrueba{}
	for _, estado := range []int{0, http.StatusOK, http.StatusNotFound, http.StatusForbidden, http.StatusTooManyRequests} {
		manejador := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if estado != 0 {
				w.WriteHeader(http.StatusContinue)
				w.WriteHeader(estado)
			}
			_, _ = w.Write([]byte("ok"))
		})
		SupervisarRespuestas(manejador, emisor).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}
	if got := emisor.codigos(); len(got) != 0 {
		t.Fatalf("incidencias inesperadas: %v", got)
	}
}

func TestSupervisarRespuestasSoloCuentaElPrimerEstadoFinal(t *testing.T) {
	emisor := &emisorIncidenciasPrueba{}
	manejador := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.WriteHeader(http.StatusInternalServerError)
	})
	SupervisarRespuestas(manejador, emisor).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if got := emisor.codigos(); len(got) != 0 {
		t.Fatalf("un WriteHeader superfluo no cambia el estado enviado: %v", got)
	}
}

func TestSupervisarRespuestasContienePanicoConRespuestaFija(t *testing.T) {
	emisor := &emisorIncidenciasPrueba{}
	manejador := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Set-Cookie", "no-debe-salir")
		panic("secreto 12345678Z postgres://usuario:clave@host/base")
	})
	respuesta := httptest.NewRecorder()
	SupervisarRespuestas(manejador, emisor).ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/", nil))
	if respuesta.Code != http.StatusInternalServerError || respuesta.Body.String() != cuerpoPanicoControlado {
		t.Fatalf("respuesta tras pánico = %d %q", respuesta.Code, respuesta.Body.String())
	}
	if respuesta.Header().Get("Set-Cookie") != "" {
		t.Fatal("la respuesta tras pánico conserva la cookie del handler")
	}
	if got := emisor.codigos(); len(got) != 1 || got[0] != domain.IncidenciaPanicoControlado {
		t.Fatalf("incidencias = %v, se esperaba PANICO_CONTROLADO sin HTTP_INTERNO_FALLIDO duplicada", got)
	}
}

func TestSupervisarRespuestasAbortaSiElPanicoLlegaTrasLasCabeceras(t *testing.T) {
	emisor := &emisorIncidenciasPrueba{}
	manejador := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		panic("tarde")
	})
	defer func() {
		if valor := recover(); valor != http.ErrAbortHandler {
			t.Fatalf("pánico propagado = %v, se esperaba http.ErrAbortHandler", valor)
		}
		if got := emisor.codigos(); len(got) != 1 || got[0] != domain.IncidenciaPanicoControlado {
			t.Fatalf("incidencias = %v", got)
		}
	}()
	SupervisarRespuestas(manejador, emisor).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestSupervisarRespuestasRespetaErrAbortHandler(t *testing.T) {
	emisor := &emisorIncidenciasPrueba{}
	manejador := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })
	defer func() {
		if valor := recover(); valor != http.ErrAbortHandler {
			t.Fatalf("pánico propagado = %v", valor)
		}
		if got := emisor.codigos(); len(got) != 0 {
			t.Fatalf("ErrAbortHandler es un aborto deliberado, no una incidencia: %v", got)
		}
	}()
	SupervisarRespuestas(manejador, emisor).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

// P2-3: si el adaptador ya declaró un código específico en la petición, el
// middleware no añade HTTP_INTERNO_FALLIDO por el mismo fallo.
func TestSupervisarRespuestasNoDuplicaUnCodigoEspecifico(t *testing.T) {
	emisor := &emisorIncidenciasPrueba{}
	especifico := &emisorIncidenciasPrueba{}
	manejador := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ports.EmitirIncidenciaTecnicaEnPeticion(r.Context(), especifico, domain.SolicitudIncidenciaTecnica{
			Codigo: domain.IncidenciaCatalogoModulosInvalido, Componente: domain.ComponenteIncidenciaCatalogoModulos, Etapa: domain.EtapaIncidenciaValidacion,
		})
		w.WriteHeader(http.StatusInternalServerError)
	})
	SupervisarRespuestas(manejador, emisor).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if got := especifico.codigos(); len(got) != 1 || got[0] != domain.IncidenciaCatalogoModulosInvalido {
		t.Fatalf("incidencia específica = %v", got)
	}
	if got := emisor.codigos(); len(got) != 0 {
		t.Fatalf("el middleware duplicó el fallo: %v", got)
	}
	// La marca es por petición: la siguiente sin código específico sí cuenta.
	SupervisarRespuestas(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}), emisor).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if got := emisor.codigos(); len(got) != 1 || got[0] != domain.IncidenciaHTTPInternoFallido {
		t.Fatalf("incidencias = %v", got)
	}
}

// P2-2: tras un pánico se conservan las cabeceras de seguridad y se retiran
// cookies y cabeceras de contenido.
func TestPanicoConservaCabecerasDeSeguridad(t *testing.T) {
	seguridad := map[string]string{
		"Content-Security-Policy":   "default-src 'none'",
		"Strict-Transport-Security": "max-age=63072000",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "no-referrer",
		"Permissions-Policy":        "camera=()",
	}
	manejador := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for nombre, valor := range seguridad {
			w.Header().Set(nombre, valor)
		}
		w.Header().Set("Set-Cookie", "sesion=secreta")
		w.Header().Set("Content-Disposition", "attachment; filename=12345678Z.pdf")
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Location", "/interna")
		w.Header().Set(http.TrailerPrefix+"X-Huella", "x")
		panic("x")
	})
	respuesta := httptest.NewRecorder()
	SupervisarRespuestas(manejador, nil).ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/", nil))
	for nombre, valor := range seguridad {
		if respuesta.Header().Get(nombre) != valor {
			t.Fatalf("cabecera de seguridad %s perdida: %q", nombre, respuesta.Header().Get(nombre))
		}
	}
	for _, nombre := range []string{"Set-Cookie", "Content-Disposition", "Location", http.TrailerPrefix + "X-Huella"} {
		if _, presente := respuesta.Header()[http.CanonicalHeaderKey(nombre)]; presente {
			t.Fatalf("cabecera %s conservada tras el pánico", nombre)
		}
	}
	if respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" || respuesta.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cabeceras de la respuesta fija: %v", respuesta.Header())
	}
}

type escritorConReadFrom struct {
	*httptest.ResponseRecorder
	usado bool
}

func (e *escritorConReadFrom) ReadFrom(origen io.Reader) (int64, error) {
	e.usado = true
	return io.Copy(e.ResponseRecorder, origen)
}

// P2-1: el ResponseWriter supervisado conserva ReadFrom (sendfile).
func TestEscritorSupervisadoConservaReadFrom(t *testing.T) {
	destino := &escritorConReadFrom{ResponseRecorder: httptest.NewRecorder()}
	emisor := &emisorIncidenciasPrueba{}
	manejador := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		lector, ok := w.(io.ReaderFrom)
		if !ok {
			t.Error("el escritor supervisado no expone io.ReaderFrom")
			return
		}
		if _, err := lector.ReadFrom(strings.NewReader("contenido")); err != nil {
			t.Error(err)
		}
	})
	SupervisarRespuestas(manejador, emisor).ServeHTTP(destino, httptest.NewRequest(http.MethodGet, "/", nil))
	if !destino.usado || destino.Body.String() != "contenido" || destino.Code != http.StatusOK {
		t.Fatalf("ReadFrom no delegado: usado=%v cuerpo=%q estado=%d", destino.usado, destino.Body.String(), destino.Code)
	}
	if got := emisor.codigos(); len(got) != 0 {
		t.Fatalf("incidencias = %v", got)
	}
}

func TestSupervisarRespuestasSinEmisorSigueConteniendoPanicos(t *testing.T) {
	manejador := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("x") })
	respuesta := httptest.NewRecorder()
	SupervisarRespuestas(manejador, nil).ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/", nil))
	if respuesta.Code != http.StatusInternalServerError {
		t.Fatalf("estado = %d", respuesta.Code)
	}
}

func TestSupervisarServidorRegistraPanicoRealYSaneaErrorLog(t *testing.T) {
	emisor := &emisorIncidenciasPrueba{}
	var registro bytes.Buffer
	var mu sync.Mutex
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fallo" {
			http.Error(w, "servicio no disponible", http.StatusServiceUnavailable)
			return
		}
		panic("dato 12345678Z")
	})}
	SupervisarServidor(srv, emisor, escritorSincronizado{mu: &mu, w: &registro})
	prueba := httptest.NewUnstartedServer(srv.Handler)
	prueba.Config = srv
	prueba.Start()
	defer prueba.Close()

	for _, ruta := range []string{"/fallo", "/panico"} {
		resp, err := prueba.Client().Get(prueba.URL + ruta)
		if err != nil {
			t.Fatalf("%s: %v", ruta, err)
		}
		_ = resp.Body.Close()
	}
	got := emisor.codigos()
	if len(got) != 2 || got[0] != domain.IncidenciaHTTPInternoFallido || got[1] != domain.IncidenciaPanicoControlado {
		t.Fatalf("incidencias = %v", got)
	}
	srv.ErrorLog.Printf("http: TLS handshake error from 192.0.2.10:5555: remote error")
	mu.Lock()
	texto := registro.String()
	mu.Unlock()
	if strings.Contains(texto, "192.0.2.10") || strings.Contains(texto, "12345678Z") || !strings.Contains(texto, mensajeErrorLogEvento) {
		t.Fatalf("ErrorLog no saneado: %q", texto)
	}
}

type escritorSincronizado struct {
	mu *sync.Mutex
	w  *bytes.Buffer
}

func (e escritorSincronizado) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.w.Write(p)
}

func TestErrorLogSaneadoAgrupaYContabiliza(t *testing.T) {
	var destino bytes.Buffer
	emisor := &emisorIncidenciasPrueba{}
	escritor := nuevoEscritorErrorLogSaneado(&destino, emisor)
	instante := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	escritor.ahora = func() time.Time { return instante }
	registro := log.New(escritor, "", 0)
	registro.Print("http: TLS handshake error from 198.51.100.7:4242: EOF")
	registro.Print("http: TLS handshake error from 198.51.100.8:4242: EOF")
	registro.Print("http: TLS handshake error from 198.51.100.9:4242: EOF")
	instante = instante.Add(intervaloEventoServidorSaneado)
	registro.Print("http: Accept error: accept tcp: too many open files; retrying in 5ms")
	registro.Print("http: panic serving 198.51.100.7:4242: dato\ngoroutine 1 [running]:\n")
	lineas := strings.Split(strings.TrimSpace(destino.String()), "\n")
	esperadas := []string{
		mensajeErrorLogEvento + " agrupados_previos=0",
		mensajeErrorLogEvento + " agrupados_previos=2",
		mensajeErrorLogPanico + " agrupados_previos=0",
	}
	if strings.Join(lineas, "|") != strings.Join(esperadas, "|") {
		t.Fatalf("líneas = %q", lineas)
	}
	if got := emisor.codigos(); len(got) != 1 || got[0] != domain.IncidenciaPanicoControlado {
		t.Fatalf("incidencias = %v", got)
	}
}

// P2-4: el último grupo no se pierde: se vuelca al vencer el intervalo o al
// cerrar, sin escribir con el mutex tomado.
func TestErrorLogSaneadoVuelcaElUltimoGrupo(t *testing.T) {
	var destino bytes.Buffer
	escritor := nuevoEscritorErrorLogSaneado(&destino, nil)
	instante := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	escritor.ahora = func() time.Time { return instante }
	var programados []func()
	escritor.programar = func(espera time.Duration, f func()) *time.Timer {
		if espera <= 0 || espera > intervaloEventoServidorSaneado {
			t.Errorf("espera del volcado = %v", espera)
		}
		programados = append(programados, f)
		return time.NewTimer(time.Hour)
	}
	registro := log.New(escritor, "", 0)
	registro.Print("http: TLS handshake error from 198.51.100.7:4242: EOF")
	registro.Print("http: TLS handshake error from 198.51.100.8:4242: EOF")
	registro.Print("http: TLS handshake error from 198.51.100.9:4242: EOF")
	if len(programados) != 1 {
		t.Fatalf("volcados programados = %d", len(programados))
	}
	instante = instante.Add(intervaloEventoServidorSaneado)
	programados[0]()
	registro.Print("http: TLS handshake error from 198.51.100.7:4242: EOF")
	registro.Print("http: TLS handshake error from 198.51.100.7:4242: EOF")
	escritor.Cerrar()
	escritor.Cerrar()
	lineas := strings.Split(strings.TrimSpace(destino.String()), "\n")
	esperadas := []string{
		mensajeErrorLogEvento + " agrupados_previos=0",
		mensajeErrorLogGrupo + " agrupados=2",
		mensajeErrorLogGrupo + " agrupados=2",
	}
	if strings.Join(lineas, "|") != strings.Join(esperadas, "|") {
		t.Fatalf("líneas = %q", lineas)
	}
	if strings.Contains(destino.String(), "198.51.100") {
		t.Fatal("ErrorLog no saneado")
	}
}

// escritorQueComprueba falla si se le escribe mientras el escritor saneado
// tiene su mutex tomado.
type escritorQueComprueba struct {
	t        *testing.T
	saneado  *escritorErrorLogSaneado
	escritas int
}

func (e *escritorQueComprueba) Write(p []byte) (int, error) {
	if e.saneado.mu.TryLock() {
		e.saneado.mu.Unlock()
	} else {
		e.t.Error("escritura al destino con el mutex tomado")
	}
	e.escritas++
	return len(p), nil
}

func TestErrorLogSaneadoNoEscribeConElMutexTomado(t *testing.T) {
	destino := &escritorQueComprueba{t: t}
	escritor := nuevoEscritorErrorLogSaneado(destino, nil)
	destino.saneado = escritor
	instante := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	escritor.ahora = func() time.Time { return instante }
	var volcar func()
	escritor.programar = func(_ time.Duration, f func()) *time.Timer {
		volcar = f
		return time.NewTimer(time.Hour)
	}
	registro := log.New(escritor, "", 0)
	registro.Print("http: TLS handshake error")
	registro.Print("http: TLS handshake error")
	volcar()
	registro.Print("http: TLS handshake error")
	escritor.Cerrar()
	registro.Print("http: panic serving 198.51.100.7:4242: x")
	if destino.escritas != 4 {
		t.Fatalf("escritas = %d", destino.escritas)
	}
}

func BenchmarkSupervisarRespuestas(b *testing.B) {
	emisor := &emisorDescarte{}
	manejador := SupervisarRespuestas(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), emisor)
	peticion := httptest.NewRequest(http.MethodGet, "/", nil)
	b.ReportAllocs()
	for b.Loop() {
		manejador.ServeHTTP(httptest.NewRecorder(), peticion)
	}
}

type emisorDescarte struct{}

func (emisorDescarte) Emitir(domain.SolicitudIncidenciaTecnica) {}
