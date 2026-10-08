package httpinterno

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorCargaPrueba struct {
	errVista, errConfirmar error
	entrada                EntradaConfirmarCargaConvoca
}

func (p *preparadorCargaPrueba) PrepararVistaPreviaCargaConvoca(context.Context) error {
	return p.errVista
}
func (p *preparadorCargaPrueba) PrepararConfirmacionCargaConvoca(_ context.Context, e EntradaConfirmarCargaConvoca) (puertosbolsa.SolicitudConfirmarCargaConvoca, error) {
	p.entrada = e
	return puertosbolsa.SolicitudConfirmarCargaConvoca{CategoriaRef: "categoria:rpt:" + e.CategoriaClave, NombreFichero: e.NombreFichero, Contenido: e.Contenido}, p.errConfirmar
}

type operadorCargaPrueba struct {
	previsualizador *aplicacionbolsa.PrevisualizadorCargaConvoca
	vistaFija       *aplicacionbolsa.VistaPreviaCargaConvoca
	lectura         *registradorVistaPreviaCargaPrueba
	resultado       aplicacionbolsa.ResultadoCargaConvoca
	err             error
	excluir         bool
}

type registradorVistaPreviaCargaPrueba struct {
	registros []RegistroVistaPreviaCargaConvoca
	err       error
	antes     func()
}

func (a *registradorVistaPreviaCargaPrueba) RegistrarVistaPreviaCargaConvoca(_ context.Context, registro RegistroVistaPreviaCargaConvoca) error {
	if a.antes != nil {
		a.antes()
	}
	a.registros = append(a.registros, registro)
	return a.err
}

func (o *operadorCargaPrueba) Previsualizar(ctx context.Context, nombre string, contenido []byte) (aplicacionbolsa.VistaPreviaCargaConvoca, error) {
	if o.vistaFija != nil {
		return *o.vistaFija, nil
	}
	return o.previsualizador.Previsualizar(ctx, nombre, contenido)
}
func (o *operadorCargaPrueba) Confirmar(_ context.Context, _ puertosbolsa.SolicitudConfirmarCargaConvoca, excluir bool) (aplicacionbolsa.ResultadoCargaConvoca, error) {
	o.excluir = excluir
	return o.resultado, o.err
}

type auditorCargaPrueba struct {
	operaciones []string
	fallos      []error
	recursos    []string
	err         error
}

func (a *auditorCargaPrueba) RegistrarIntentoFallidoCargaConvoca(ctx context.Context, operacion string, fallo error) error {
	a.operaciones, a.fallos = append(a.operaciones, operacion), append(a.fallos, fallo)
	recurso, _ := RecursoIntentoCargaConvoca(ctx)
	a.recursos = append(a.recursos, recurso)
	return a.err
}

func handlerCargaPrueba(t *testing.T) (http.Handler, *preparadorCargaPrueba, *operadorCargaPrueba, *auditorCargaPrueba) {
	t.Helper()
	previsualizador, err := aplicacionbolsa.NuevoPrevisualizadorCargaConvoca(xlsconvoca.NuevoLectorConLimiteFilas(aplicacionbolsa.MaximoFilasCargaConvoca + 1))
	if err != nil {
		t.Fatal(err)
	}
	p, o, a := &preparadorCargaPrueba{}, &operadorCargaPrueba{previsualizador: previsualizador, lectura: &registradorVistaPreviaCargaPrueba{}}, &auditorCargaPrueba{}
	h, err := NuevoHandlerCargaConvoca(p, o, a, o.lectura)
	if err != nil {
		t.Fatal(err)
	}
	return h, p, o, a
}

func ejemploCargaHTTP(t *testing.T) string {
	t.Helper()
	contenido, err := os.ReadFile(filepath.Join("..", "..", "application", "testdata", "carga_convoca", "carga_convoca_ejemplo.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(contenido)
}

func peticionCarga(ruta string, cuerpo any) *http.Request {
	b, _ := json.Marshal(cuerpo)
	r := httptest.NewRequest(http.MethodPost, ruta, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	return r
}

func codigoErrorCarga(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var cuerpo struct {
		Error struct {
			Codigo  string `json:"codigo"`
			Mensaje string `json:"mensaje"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("cuerpo de error ilegible: %s", w.Body.String())
	}
	if cuerpo.Error.Mensaje != "" {
		t.Fatalf("el error lleva texto visible: %q", cuerpo.Error.Mensaje)
	}
	return cuerpo.Error.Codigo
}

func TestCargaConvocaVistaPreviaAuditaAntesDeDevolverFilas(t *testing.T) {
	h, _, o, a := handlerCargaPrueba(t)
	w := httptest.NewRecorder()
	o.lectura.antes = func() {
		if w.Body.Len() != 0 || w.Code != http.StatusOK {
			t.Fatal("las filas salieron antes del apunte de lectura")
		}
	}
	h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "carga_convoca_ejemplo.xlsx", "contenido_base64": ejemploCargaHTTP(t)}))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("estado %d cabeceras %v: %s", w.Code, w.Header(), w.Body.String())
	}
	var cuerpo struct {
		Data struct {
			Esquema        string `json:"esquema"`
			Filtro         string `json:"filtro"`
			Limite         int    `json:"limite"`
			Desplazamiento int    `json:"desplazamiento"`
			TotalFiltrado  int    `json:"total_filtrado"`
			Aceptadas      int    `json:"aceptadas"`
			Rechazadas     int    `json:"rechazadas"`
			ConAvisos      int    `json:"con_avisos"`
			Bloqueo        string `json:"bloqueo"`
			Filas          []struct {
				Numero   int    `json:"numero"`
				Estado   string `json:"estado"`
				Posicion int    `json:"posicion"`
				Nombre   string `json:"nombre"`
				Errores  []struct {
					Campo  string `json:"campo"`
					Codigo string `json:"codigo"`
				} `json:"errores"`
				Avisos []string `json:"avisos"`
			} `json:"filas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	d := cuerpo.Data
	if d.Esquema != EsquemaVistaPreviaCargaConvoca || d.Filtro != "todas" || d.Limite != 50 || d.Desplazamiento != 0 || d.TotalFiltrado != 12 || d.Aceptadas != 11 || d.Rechazadas != 1 || d.ConAvisos != 2 || d.Bloqueo != "" || len(d.Filas) != 12 ||
		d.Filas[0].Posicion != 1 || d.Filas[0].Nombre != "Antonio" || d.Filas[11].Estado != "rechazada" || d.Filas[11].Errores[0].Codigo != "total_incoherente" {
		t.Fatalf("vista previa inesperada: %+v", d)
	}
	if len(a.operaciones) != 0 {
		t.Fatal("auditó como fallido un intento correcto")
	}
	if len(o.lectura.registros) != 1 {
		t.Fatalf("apuntes de lectura = %d", len(o.lectura.registros))
	}
	contenido, err := base64.StdEncoding.DecodeString(ejemploCargaHTTP(t))
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(contenido)
	if registro := o.lectura.registros[0]; registro.RecursoRef != "fichero:sha256:"+hex.EncodeToString(huella[:]) ||
		registro.Filtro != "todas" || registro.Limite != 50 || registro.Desplazamiento != 0 {
		t.Fatalf("apunte no nominal: %+v", registro)
	}
}

func TestCargaConvocaVistaPreviaNoDevuelveFilasSinAuditoriaCorrecta(t *testing.T) {
	h, _, o, a := handlerCargaPrueba(t)
	o.lectura.err = errors.New("destino de lectura no disponible")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, map[string]any{
		"nombre_fichero": "carga_convoca_ejemplo.xlsx", "contenido_base64": ejemploCargaHTTP(t),
		"filtro": "con_avisos", "limite": 1, "desplazamiento": 1,
	}))
	if w.Code != http.StatusServiceUnavailable || codigoErrorCarga(t, w) != "servicio_no_disponible" ||
		strings.Contains(w.Body.String(), "Antonio") || strings.Contains(w.Body.String(), `"filas"`) ||
		len(o.lectura.registros) != 1 || len(a.operaciones) != 1 || a.operaciones[0] != OperacionVistaPreviaCargaConvoca {
		t.Fatalf("la vista previa entregó datos sin apunte: %d %s", w.Code, w.Body.String())
	}
	registro := o.lectura.registros[0]
	if registro.Filtro != "con_avisos" || registro.Limite != 1 || registro.Desplazamiento != 1 ||
		!strings.HasPrefix(registro.RecursoRef, "fichero:sha256:") {
		t.Fatalf("intento de lectura sin referencia/página: %+v", registro)
	}
}

func TestCargaConvocaRechazaRegistradorLecturaNuloInclusoTipado(t *testing.T) {
	_, preparador, operador, auditor := handlerCargaPrueba(t)
	for _, lectura := range []RegistradorVistaPreviaCargaConvoca{nil, (*registradorVistaPreviaCargaPrueba)(nil)} {
		h, err := NuevoHandlerCargaConvoca(preparador, operador, auditor, lectura)
		if h != nil || !errors.Is(err, puertosbolsa.ErrCargaConvocaNoDisponible) {
			t.Fatalf("registrador ausente montó la ruta: handler=%v error=%v", h, err)
		}
	}
	// Un handler armado incorrectamente fuera del constructor tampoco emite filas.
	h := &HandlerCargaConvoca{preparador: preparador, operador: operador, auditor: auditor}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, map[string]any{}))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), `"filas"`) {
		t.Fatalf("handler sin auditoría expuso filas: %d %s", w.Code, w.Body.String())
	}
}

type respuestaPaginaCargaPrueba struct {
	Data struct {
		Filtro         string `json:"filtro"`
		Limite         int    `json:"limite"`
		Desplazamiento int    `json:"desplazamiento"`
		TotalFiltrado  int    `json:"total_filtrado"`
		FilasLeidas    int    `json:"filas_leidas"`
		Aceptadas      int    `json:"aceptadas"`
		Rechazadas     int    `json:"rechazadas"`
		ConAvisos      int    `json:"con_avisos"`
		Filas          []struct {
			Numero int `json:"numero"`
		} `json:"filas"`
	} `json:"data"`
}

func TestCargaConvocaVistaPreviaFiltraYPaginaEnServidor(t *testing.T) {
	contenido := ejemploCargaHTTP(t)
	for _, caso := range []struct {
		filtro         string
		limite, offset int
		total          int
		filas          []int
	}{
		{"aceptadas", 1, 0, 11, []int{2}},
		{"aceptadas", 1, 1, 11, []int{3}},
		{"rechazadas", 50, 0, 1, []int{12}},
		{"con_avisos", 50, 0, 2, []int{8, 9}},
		{"con_avisos", 50, 2, 2, []int{}},
	} {
		t.Run(caso.filtro+"/"+strconv.Itoa(caso.offset), func(t *testing.T) {
			h, _, o, _ := handlerCargaPrueba(t)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, map[string]any{
				"nombre_fichero": "carga_convoca_ejemplo.xlsx", "contenido_base64": contenido,
				"filtro": caso.filtro, "limite": caso.limite, "desplazamiento": caso.offset,
			}))
			if w.Code != http.StatusOK {
				t.Fatalf("estado %d: %s", w.Code, w.Body.String())
			}
			var respuesta respuestaPaginaCargaPrueba
			if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
				t.Fatal(err)
			}
			d := respuesta.Data
			if d.Filtro != caso.filtro || d.Limite != caso.limite || d.Desplazamiento != caso.offset || d.TotalFiltrado != caso.total ||
				d.FilasLeidas != 12 || d.Aceptadas != 11 || d.Rechazadas != 1 || d.ConAvisos != 2 || len(d.Filas) != len(caso.filas) {
				t.Fatalf("página o contadores incorrectos: %+v", d)
			}
			if len(o.lectura.registros) != 1 || o.lectura.registros[0].Filtro != caso.filtro ||
				o.lectura.registros[0].Limite != caso.limite || o.lectura.registros[0].Desplazamiento != caso.offset {
				t.Fatalf("apunte de la página: %+v", o.lectura.registros)
			}
			for i, fila := range d.Filas {
				if fila.Numero != caso.filas[i] {
					t.Fatalf("fila %d = %d, se esperaba %d", i, fila.Numero, caso.filas[i])
				}
			}
		})
	}
}

func TestCargaConvocaVistaPreviaAcotaPaginaCincuenta(t *testing.T) {
	h, _, operador, _ := handlerCargaPrueba(t)
	vista := aplicacionbolsa.VistaPreviaCargaConvoca{FilasLeidas: 120, Aceptadas: 120}
	for numero := 1; numero <= 120; numero++ {
		vista.Filas = append(vista.Filas, aplicacionbolsa.FilaVistaPreviaCargaConvoca{Numero: numero, Estado: aplicacionbolsa.EstadoFilaCargaAceptada})
	}
	operador.vistaFija = &vista
	for _, caso := range []struct{ offset, primero, ultimo, cantidad int }{{0, 1, 50, 50}, {50, 51, 100, 50}, {100, 101, 120, 20}, {150, 0, 0, 0}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, map[string]any{
			"nombre_fichero": "carga_convoca_ejemplo.xlsx", "contenido_base64": ejemploCargaHTTP(t),
			"desplazamiento": caso.offset,
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("offset %d: estado %d", caso.offset, w.Code)
		}
		var respuesta respuestaPaginaCargaPrueba
		if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
			t.Fatal(err)
		}
		if respuesta.Data.Limite != 50 || respuesta.Data.TotalFiltrado != 120 || len(respuesta.Data.Filas) != caso.cantidad {
			t.Fatalf("offset %d: %+v", caso.offset, respuesta.Data)
		}
		if caso.cantidad > 0 && (respuesta.Data.Filas[0].Numero != caso.primero || respuesta.Data.Filas[caso.cantidad-1].Numero != caso.ultimo) {
			t.Fatalf("offset %d: filas incorrectas", caso.offset)
		}
	}
}

func TestCargaConvocaPaginacionRechazaValoresYConfirmacionLosProhibe(t *testing.T) {
	contenido := ejemploCargaHTTP(t)
	base := map[string]any{"nombre_fichero": "carga_convoca_ejemplo.xlsx", "contenido_base64": contenido}
	for _, caso := range []struct {
		nombre string
		campo  string
		valor  any
		estado int
		codigo string
	}{
		{"filtro desconocido", "filtro", "otro", 422, "paginacion_no_valida"},
		{"filtro nulo", "filtro", nil, 400, "peticion_no_valida"},
		{"filtro numérico", "filtro", 1, 400, "peticion_no_valida"},
		{"limite cero", "limite", 0, 422, "paginacion_no_valida"},
		{"limite sobre máximo", "limite", 101, 422, "paginacion_no_valida"},
		{"limite cadena", "limite", "50", 400, "peticion_no_valida"},
		{"limite nulo", "limite", nil, 400, "peticion_no_valida"},
		{"desplazamiento negativo", "desplazamiento", -1, 422, "paginacion_no_valida"},
		{"desplazamiento nulo", "desplazamiento", nil, 400, "peticion_no_valida"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			h, _, _, a := handlerCargaPrueba(t)
			cuerpo := map[string]any{}
			for clave, valor := range base {
				cuerpo[clave] = valor
			}
			cuerpo[caso.campo] = caso.valor
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, cuerpo))
			if w.Code != caso.estado || codigoErrorCarga(t, w) != caso.codigo || len(a.operaciones) != 1 {
				t.Fatalf("estado %d cuerpo %s auditados %d", w.Code, w.Body.String(), len(a.operaciones))
			}
		})
	}
	for _, campo := range []string{"filtro", "limite", "desplazamiento"} {
		h, p, _, _ := handlerCargaPrueba(t)
		cuerpo := map[string]any{"nombre_fichero": "carga_convoca_ejemplo.xlsx", "contenido_base64": contenido, "categoria": "auxiliar_administrativo", campo: nil}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionCarga(RutaConfirmarCargaConvoca, cuerpo))
		if w.Code != http.StatusBadRequest || codigoErrorCarga(t, w) != "peticion_no_valida" || p.entrada.Contenido != nil {
			t.Fatalf("confirmación admitió %s: %d %s", campo, w.Code, w.Body.String())
		}
	}
	for _, ruta := range []string{RutaVistaPreviaCargaConvoca, RutaConfirmarCargaConvoca} {
		h, _, _, _ := handlerCargaPrueba(t)
		raw := `{"nombre_fichero":"carga_convoca_ejemplo.xlsx","contenido_base64":"` + contenido + `","limite":1,"limite":2}`
		r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || codigoErrorCarga(t, w) != "peticion_no_valida" {
			t.Fatalf("duplicado en %s: %d %s", ruta, w.Code, w.Body.String())
		}
	}
}

func TestCargaConvocaRechazaEntradaAntesDeLeerYAudita(t *testing.T) {
	contenido := ejemploCargaHTTP(t)
	casos := []struct {
		nombre string
		mutar  func(*http.Request)
		ruta   string
		cuerpo map[string]any
		estado int
		codigo string
	}{
		{"cookie", func(r *http.Request) { r.Header.Set("Cookie", "s=1") }, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": contenido}, 400, "peticion_no_valida"},
		{"identidad heredada", func(r *http.Request) { r.Header.Set("X-Vec-Persona", "per_x") }, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": contenido}, 400, "peticion_no_valida"},
		{"tipo", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": contenido}, 400, "peticion_no_valida"},
		{"campo desconocido", nil, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": contenido, "persona": "per_x"}, 400, "peticion_no_valida"},
		{"base64 roto", nil, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": "no es base64!"}, 400, "peticion_no_valida"},
		{"nombre con ruta", nil, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "../a.xlsx", "contenido_base64": contenido}, 400, "peticion_no_valida"},
		{"vista con categoría", nil, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": contenido, "categoria": "auxiliar"}, 400, "peticion_no_valida"},
		{"categoría mal formada", nil, RutaConfirmarCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": contenido, "categoria": "Aux iliar"}, 422, "categoria_no_valida"},
		{"declarado excesivo", func(r *http.Request) { r.ContentLength = MaximoCuerpoCargaConvoca + 1 }, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": contenido}, 413, "fichero_demasiado_grande"},
		{"fichero excesivo", nil, RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": base64.StdEncoding.EncodeToString(make([]byte, aplicacionbolsa.MaximoBytesCargaConvoca+1))}, 413, "fichero_demasiado_grande"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			h, _, _, a := handlerCargaPrueba(t)
			r := peticionCarga(caso.ruta, caso.cuerpo)
			if caso.mutar != nil {
				caso.mutar(r)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || codigoErrorCarga(t, w) != caso.codigo || len(a.operaciones) != 1 {
				t.Fatalf("estado %d cuerpo %s auditados %d", w.Code, w.Body.String(), len(a.operaciones))
			}
		})
	}
}

func TestCargaConvocaRutaYMetodo(t *testing.T) {
	h, _, _, a := handlerCargaPrueba(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaConfirmarCargaConvoca, nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("GET: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionCarga(RutaConfirmarCargaConvoca+"?x=1", map[string]any{}))
	if w.Code != http.StatusNotFound {
		t.Fatalf("consulta en la ruta: %d", w.Code)
	}
	if len(a.operaciones) != 0 {
		t.Fatal("auditó una ruta inexistente")
	}
}

func TestCargaConvocaDenegacionYAuditorCaido(t *testing.T) {
	h, p, _, a := handlerCargaPrueba(t)
	p.errVista = dominiovec.ErrAutorizacionDenegada
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": ejemploCargaHTTP(t)}))
	if w.Code != http.StatusForbidden || codigoErrorCarga(t, w) != "acceso_denegado" || len(a.operaciones) != 1 || a.operaciones[0] != OperacionVistaPreviaCargaConvoca || !errors.Is(a.fallos[0], dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("denegación: %d %s %v", w.Code, w.Body.String(), a.operaciones)
	}
	a.err = errors.New("auditoría caída")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "a.xlsx", "contenido_base64": ejemploCargaHTTP(t)}))
	if w.Code != http.StatusServiceUnavailable || codigoErrorCarga(t, w) != "servicio_no_disponible" {
		t.Fatalf("auditor caído: %d", w.Code)
	}
}

func TestCargaConvocaConfirmar(t *testing.T) {
	h, p, o, a := handlerCargaPrueba(t)
	o.resultado.Recibo.BolsaRef, o.resultado.Recibo.VersionBolsa, o.resultado.Recibo.AuditoriaRef = "bolsa:auxiliar:2026-10-05", 1, "aud_v3_0123456789abcdef0123456789abcdef"
	o.resultado.Recibo.ConfirmadaEn = time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	o.resultado.Recibo.PendientesRevision = []puertosbolsa.FilaPendienteRevision{{FilaNumero: 8, Motivo: puertosbolsa.MotivoRevisionIdentidadAmbigua}}
	o.resultado.FilasCargadas, o.resultado.FilasExcluidas = 11, 1
	cuerpo := map[string]any{"nombre_fichero": "carga_convoca_ejemplo.xlsx", "contenido_base64": ejemploCargaHTTP(t), "categoria": "auxiliar_administrativo", "excluir_filas_con_errores": true}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCarga(RutaConfirmarCargaConvoca, cuerpo))
	if w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), `"esquema":"vec.bolsa.rrhh.carga_convoca.recibo.v1"`) ||
		!strings.Contains(w.Body.String(), `"pendientes_revision":[{"fila":8,"motivo":"identidad_ambigua"}]`) || !o.excluir ||
		p.entrada.CategoriaClave != "auxiliar_administrativo" || len(p.entrada.Contenido) == 0 || len(a.operaciones) != 0 || len(o.lectura.registros) != 0 {
		t.Fatalf("confirmación: %d %s", w.Code, w.Body.String())
	}
	o.resultado.Recibo.Reutilizada = true
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionCarga(RutaConfirmarCargaConvoca, cuerpo))
	if w.Code != http.StatusOK {
		t.Fatalf("repetición: %d", w.Code)
	}
	for fallo, esperado := range map[error]string{
		aplicacionbolsa.ErrCargaConvocaConErrores:    "filas_con_errores_sin_aceptar",
		aplicacionbolsa.ErrCargaConvocaBloqueada:     "fichero_no_cargable",
		puertosbolsa.ErrConstitucionBolsaEnConflicto: "bolsa_en_conflicto",
		dominiovec.ErrAutorizacionDenegada:           "acceso_denegado",
		errors.New("cualquier otra cosa"):            "servicio_no_disponible",
	} {
		o.err = fallo
		w = httptest.NewRecorder()
		h.ServeHTTP(w, peticionCarga(RutaConfirmarCargaConvoca, cuerpo))
		if codigoErrorCarga(t, w) != esperado {
			t.Fatalf("%v: %d %s", fallo, w.Code, w.Body.String())
		}
	}
	contenido, err := base64.StdEncoding.DecodeString(cuerpo["contenido_base64"].(string))
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(contenido)
	esperado := "fichero:sha256:" + hex.EncodeToString(huella[:])
	for _, recurso := range a.recursos {
		if recurso != esperado {
			t.Fatalf("fallo tras decodificar perdió recurso: %q", recurso)
		}
	}
}
