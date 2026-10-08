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
	resultado       aplicacionbolsa.ResultadoCargaConvoca
	err             error
	excluir         bool
}

func (o *operadorCargaPrueba) Previsualizar(ctx context.Context, nombre string, contenido []byte) (aplicacionbolsa.VistaPreviaCargaConvoca, error) {
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
	p, o, a := &preparadorCargaPrueba{}, &operadorCargaPrueba{previsualizador: previsualizador}, &auditorCargaPrueba{}
	h, err := NuevoHandlerCargaConvoca(p, o, a)
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

func TestCargaConvocaVistaPreviaDevuelveFilasYNoAudita(t *testing.T) {
	h, _, _, a := handlerCargaPrueba(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCarga(RutaVistaPreviaCargaConvoca, map[string]any{"nombre_fichero": "carga_convoca_ejemplo.xlsx", "contenido_base64": ejemploCargaHTTP(t)}))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("estado %d cabeceras %v: %s", w.Code, w.Header(), w.Body.String())
	}
	var cuerpo struct {
		Data struct {
			Esquema    string `json:"esquema"`
			Aceptadas  int    `json:"aceptadas"`
			Rechazadas int    `json:"rechazadas"`
			ConAvisos  int    `json:"con_avisos"`
			Bloqueo    string `json:"bloqueo"`
			Filas      []struct {
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
	if d.Esquema != EsquemaVistaPreviaCargaConvoca || d.Aceptadas != 11 || d.Rechazadas != 1 || d.ConAvisos != 2 || d.Bloqueo != "" || len(d.Filas) != 12 ||
		d.Filas[0].Posicion != 1 || d.Filas[0].Nombre != "Antonio" || d.Filas[11].Estado != "rechazada" || d.Filas[11].Errores[0].Codigo != "total_incoherente" {
		t.Fatalf("vista previa inesperada: %+v", d)
	}
	if len(a.operaciones) != 0 {
		t.Fatal("auditó como fallido un intento correcto")
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
		p.entrada.CategoriaClave != "auxiliar_administrativo" || len(p.entrada.Contenido) == 0 || len(a.operaciones) != 0 {
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
