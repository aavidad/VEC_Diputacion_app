package telemetria

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// destinoMemoria es un io.Writer concurrente para leer las líneas escritas.
type destinoMemoria struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (d *destinoMemoria) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.buf.Write(p)
}

func (d *destinoMemoria) lineas(t *testing.T) []map[string]any {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	var salida []map[string]any
	sc := bufio.NewScanner(bytes.NewReader(d.buf.Bytes()))
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("linea no JSON: %q", sc.Text())
		}
		salida = append(salida, m)
	}
	return salida
}

func (d *destinoMemoria) texto() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.buf.String()
}

func nuevoRegistroPrueba(t *testing.T, d io.Writer, u Umbrales) *Registro {
	t.Helper()
	reg, err := NuevoRegistro(Opciones{Destino: d, Servicio: "vec-server", Superficie: "interno",
		Entorno: "desarrollo", Version: "abcdef1234", Umbrales: u})
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func cerrar(t *testing.T, reg *Registro) {
	t.Helper()
	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	if err := reg.Cerrar(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAccesoEscribeLineaConPlantillaSinValores(t *testing.T) {
	d := &destinoMemoria{}
	reg := nuevoRegistroPrueba(t, d, Umbrales{})
	interior := http.NewServeMux()
	interior.HandleFunc("GET /api/vec/bolsa/{bolsa}/participaciones", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "hola")
	})
	exterior := http.NewServeMux()
	// El enrutador exterior solo conoce el subárbol; manda el patrón interior.
	exterior.Handle("/api/vec/", http.StripPrefix("", CapturarPatron(interior)))
	h := reg.Envolver(CapturarPatron(exterior))

	pet := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/9f1c2a7e-0000-4000-8000-123456789abc/participaciones?dni=12345678Z", strings.NewReader("cuerpo-secreto"))
	pet.Header.Set("Authorization", "Bearer secreto-de-prueba")
	h.ServeHTTP(httptest.NewRecorder(), pet)
	cerrar(t, reg)

	lineas := d.lineas(t)
	if len(lineas) != 1 {
		t.Fatalf("lineas = %d", len(lineas))
	}
	l := lineas[0]
	esperado := map[string]any{
		"esquema": EsquemaAcceso, "nivel": "info", "servicio": "vec-server", "superficie": "interno",
		"entorno": "desarrollo", "version": "abcdef1234", "metodo": "GET",
		"ruta": "/api/vec/bolsa/{bolsa}/participaciones", "estado": float64(200), "bytes": float64(4),
	}
	for k, v := range esperado {
		if l[k] != v {
			t.Errorf("%s = %v, se esperaba %v", k, l[k], v)
		}
	}
	if c, _ := l["correlacion"].(string); !domain.EsCorrelacionTecnicaValida(c) {
		t.Errorf("correlacion no valida: %q", c)
	}
	for _, prohibido := range []string{"9f1c2a7e", "12345678Z", "dni", "secreto", "Bearer"} {
		if strings.Contains(d.texto(), prohibido) {
			t.Errorf("el registro contiene %q", prohibido)
		}
	}
}

func TestAccesoNormalizaCaminoSinPatronYOcultaNoEncontradas(t *testing.T) {
	d := &destinoMemoria{}
	reg := nuevoRegistroPrueba(t, d, Umbrales{})
	h := reg.Envolver(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		http.NotFound(w, r)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v2/expedientes/recibo:408fda57/Juan", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/wp-admin/juan.perez", nil))
	denegar := reg.Envolver(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	denegar.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/portal-empleado/juanperez", nil))
	cerrar(t, reg)
	lineas := d.lineas(t)
	if lineas[0]["ruta"] != "/api/v2/expedientes/{valor}/{valor}" || lineas[0]["estado"] != float64(202) {
		t.Errorf("primera = %v %v", lineas[0]["ruta"], lineas[0]["estado"])
	}
	if lineas[1]["ruta"] != rutaNoEncontrada {
		t.Errorf("404 sin patron = %v", lineas[1]["ruta"])
	}
	if lineas[2]["ruta"] != rutaSinPlantilla {
		t.Errorf("401 sin patron = %v", lineas[2]["ruta"])
	}
	if strings.Contains(d.texto(), "juan") || strings.Contains(d.texto(), "Juan") {
		t.Error("el camino escrito por la persona llego al registro")
	}
}

func TestAccesoMarcaLentaYAnotaIncidenciaYCausa(t *testing.T) {
	d := &destinoMemoria{}
	reg := nuevoRegistroPrueba(t, d, Umbrales{Lenta: time.Nanosecond})
	reloj := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	reg.reloj = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		reloj = reloj.Add(200 * time.Millisecond)
		return reloj
	}
	emisor := NuevoEmisorAnotado(nil)
	h := reg.Envolver(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ports.EmitirIncidenciaTecnicaEnPeticion(r.Context(), emisor, domain.SolicitudIncidenciaTecnica{
			Codigo: domain.IncidenciaPostgresNoDisponible, Componente: domain.ComponenteIncidenciaPostgreSQL, Etapa: domain.EtapaIncidenciaConsulta,
		})
		AnotarFallo(r.Context(), "consulta_participaciones", &pgconn.PgError{Code: "57014", Message: "datos de Juan"})
		AnotarFallo(r.Context(), "segunda", errors.New("otra"))
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/vec/salud", nil))
	cerrar(t, reg)
	l := d.lineas(t)[0]
	esperado := map[string]any{
		"nivel": "error", "estado": float64(503), "lenta": true, "duracion_ms": float64(200),
		"incidencia": "POSTGRES_NO_DISPONIBLE", "componente": "postgresql", "etapa_incidencia": "consulta",
		"causa": "bd_57014", "etapa_fallo": "consulta_participaciones",
	}
	for k, v := range esperado {
		if l[k] != v {
			t.Errorf("%s = %v, se esperaba %v", k, l[k], v)
		}
	}
	if strings.Contains(d.texto(), "Juan") {
		t.Error("el texto del error llego al registro")
	}
}

func TestAccesoConservaCorrelacionDeLaSupervision(t *testing.T) {
	d := &destinoMemoria{}
	reg := nuevoRegistroPrueba(t, d, Umbrales{})
	var vista string
	h := reg.Envolver(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vista, _ = ports.CorrelacionIncidenciasPeticion(r.Context())
	}))
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	esperada, _ := ports.CorrelacionIncidenciasPeticion(ctx)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx))
	cerrar(t, reg)
	if vista != esperada || d.lineas(t)[0]["correlacion"] != esperada {
		t.Errorf("correlacion distinta: handler %q, linea %v, esperada %q", vista, d.lineas(t)[0]["correlacion"], esperada)
	}
}

func TestAccesoDejaConstanciaDelPanicoYLoPropaga(t *testing.T) {
	d := &destinoMemoria{}
	reg := nuevoRegistroPrueba(t, d, Umbrales{})
	h := reg.Envolver(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("fallo con datos de Juan") }))
	func() {
		defer func() {
			if recover() == nil {
				t.Error("el panico no se propago a la supervision")
			}
		}()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	}()
	cerrar(t, reg)
	l := d.lineas(t)[0]
	if l["interrumpida"] != true || l["estado"] != float64(500) || l["nivel"] != "error" {
		t.Errorf("linea = %v", l)
	}
	if strings.Contains(d.texto(), "Juan") {
		t.Error("el mensaje del panico llego al registro")
	}
}

func TestAccesoAvisaDePeticionesEnCursoConEscalado(t *testing.T) {
	d := &destinoMemoria{}
	reg := nuevoRegistroPrueba(t, d, Umbrales{EnCurso: 10 * time.Second})
	inicio := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	ahora := inicio
	var mu sync.Mutex
	reg.reloj = func() time.Time { mu.Lock(); defer mu.Unlock(); return ahora }
	avanzar := func(d time.Duration) { mu.Lock(); ahora = ahora.Add(d); mu.Unlock() }
	entrar, salir := make(chan struct{}), make(chan struct{})
	h := reg.Envolver(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(entrar)
		<-salir
	}))
	go h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/vec/lenta/123", nil))
	<-entrar
	if reg.EnCurso() != 1 {
		t.Errorf("en curso = %d", reg.EnCurso())
	}
	avanzar(9 * time.Second)
	reg.vigilarEnCurso() // 9 s: nada
	avanzar(2 * time.Second)
	reg.vigilarEnCurso() // 11 s: primer aviso
	reg.vigilarEnCurso() // sin repetir
	avanzar(20 * time.Second)
	reg.vigilarEnCurso() // 31 s: segundo aviso (umbral 30 s)
	close(salir)
	cerrar(t, reg)
	var enCurso []map[string]any
	for _, l := range d.lineas(t) {
		if l["esquema"] == EsquemaEnCurso {
			enCurso = append(enCurso, l)
		}
	}
	if len(enCurso) != 2 || enCurso[0]["transcurrido_ms"] != float64(11000) || enCurso[1]["transcurrido_ms"] != float64(31000) {
		t.Fatalf("avisos en curso = %v", enCurso)
	}
	if enCurso[0]["ruta"] != "/api/vec/lenta/{valor}" {
		t.Errorf("ruta en curso = %v", enCurso[0]["ruta"])
	}
}

// destinoBloqueado retiene la primera escritura hasta que se libera.
type destinoBloqueado struct {
	destinoMemoria
	liberar chan struct{}
}

func (d *destinoBloqueado) Write(p []byte) (int, error) {
	<-d.liberar
	return d.destinoMemoria.Write(p)
}

func TestAccesoDescartaSinBloquearYLoInforma(t *testing.T) {
	d := &destinoBloqueado{liberar: make(chan struct{})}
	reg, err := NuevoRegistro(Opciones{Destino: d, Servicio: "vec-server", Capacidad: 1})
	if err != nil {
		t.Fatal(err)
	}
	h := reg.Envolver(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	hecho := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		}
		close(hecho)
	}()
	select {
	case <-hecho:
	case <-time.After(5 * time.Second):
		t.Fatal("un destino bloqueado freno las peticiones")
	}
	if reg.Descartadas() == 0 {
		t.Fatal("no se contaron descartes")
	}
	close(d.liberar)
	cerrar(t, reg)
	if !strings.Contains(d.texto(), EsquemaDescartes) {
		t.Error("no se informo de los descartes")
	}
}

func TestNormalizarCamino(t *testing.T) {
	casos := map[string]string{
		"":                              "/",
		"/":                             "/",
		"/portal-empleado/":             "/portal-empleado/",
		"/static/app.js":                "/static/app.js",
		"/static/app.v123.js":           "/static/{valor}",
		"/api/v1/x":                     "/api/v1/x",
		"/api/vec/12345678Z":            "/api/vec/{valor}",
		"/api/vec/Juan":                 "/api/vec/{valor}",
		"/api/vec/peña":                 "/api/vec/{valor}",
		"/a/" + strings.Repeat("b", 41): "/a/{valor}",
		strings.Repeat("/a", 20):        strings.Repeat("/a", 16) + "/{valor}",
	}
	for entrada, esperada := range casos {
		if got := normalizarCamino(entrada); got != esperada {
			t.Errorf("normalizarCamino(%q) = %q, se esperaba %q", entrada, got, esperada)
		}
	}
}

func TestPlantillaDePatron(t *testing.T) {
	casos := map[string]string{
		"GET /api/x/{ref}":        "/api/x/{ref}",
		"POST host.example/api/y": "/api/y",
		"/api/vec/":               "",
		"/":                       "",
		"GET /x/{$}":              "/x/",
		"/static/{ruta...}":       "/static/{ruta...}",
		"":                        "",
	}
	for entrada, esperada := range casos {
		if got := plantillaDePatron(entrada); got != esperada {
			t.Errorf("plantillaDePatron(%q) = %q, se esperaba %q", entrada, got, esperada)
		}
	}
}

func TestClasificarError(t *testing.T) {
	casos := []struct {
		err  error
		want string
	}{
		{nil, ""},
		{context.Canceled, "cancelada"},
		{errors.Join(errors.New("x"), context.DeadlineExceeded), "plazo_vencido"},
		{&pgconn.PgError{Code: "53300"}, "bd_53300"},
		{&pgconn.PgError{Code: "x'; --"}, "bd"},
		{errors.New("texto libre con datos"), "desconocida"},
	}
	for _, c := range casos {
		if got := ClasificarError(c.err); got != c.want {
			t.Errorf("ClasificarError(%v) = %q, se esperaba %q", c.err, got, c.want)
		}
	}
}

func TestEntornoYRevision(t *testing.T) {
	casos := []struct{ declarado, perfil, want string }{
		{"presentacion", "desarrollo", "presentacion"},
		{"10.1.2.3", "desarrollo", "desconocido"},
		{"", "desarrollo", "desarrollo"},
		{"", "cidonia", "desarrollo"},
		{"", "presentacion_rrhh", "presentacion"},
		{"", "produccion", "produccion"},
		{"", "", "desconocido"},
	}
	for _, c := range casos {
		if got := EntornoDe(c.declarado, c.perfil); got != c.want {
			t.Errorf("EntornoDe(%q, %q) = %q, se esperaba %q", c.declarado, c.perfil, got, c.want)
		}
	}
	anterior := Revision
	t.Cleanup(func() { Revision = anterior })
	Revision = "aae093f6f0aa"
	if got := RevisionBinario(); got != "aae093f6f0aa" {
		t.Errorf("RevisionBinario() = %q", got)
	}
	Revision = "rama-con-cambios"
	if got := RevisionBinario(); got == "rama-con-cambios" {
		t.Error("una revision no conforme no se normalizo")
	}
}

func TestUmbralesDeEntorno(t *testing.T) {
	entorno := map[string]string{EnvUmbralLentaMS: "500", EnvUmbralEnCursoS: "0"}
	u, valido := UmbralesDeEntorno(func(k string) string { return entorno[k] })
	if valido || u.Lenta != 500*time.Millisecond || u.EnCurso != 0 {
		t.Errorf("umbrales = %+v valido=%t", u, valido)
	}
}
