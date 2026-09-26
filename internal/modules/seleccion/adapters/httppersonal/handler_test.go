package httppersonal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/httpcomun"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/baremacion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type preparadorPrueba struct{ err error }

func (p preparadorPrueba) PrepararOrdenPersona(*http.Request) (application.Orden, error) {
	return application.Orden{Ambitos: map[string]string{"ambito_ref": "x"}}, p.err
}

type casosPrueba struct {
	err      error
	entrada  application.EntradaBorrador
	presenta application.EntradaPresentacion
	repetida bool
	llamadas int
}

func (c *casosPrueba) Convocatorias(context.Context) ([]domain.ConvocatoriaPublicada, error) {
	c.llamadas++
	return []domain.ConvocatoriaPublicada{{Convocatoria: domain.Convocatoria{Ref: "bolsa-operario-diputacion-2026", Titulo: "Operario",
		AbreEn: time.Date(2026, 9, 20, 22, 0, 0, 0, time.UTC), CierraEn: time.Date(2026, 10, 30, 22, 59, 59, 0, time.UTC)}, Version: 1, Abierta: true}}, c.err
}

func (c *casosPrueba) Convocatoria(context.Context, string) (domain.ConvocatoriaPublicada, error) {
	c.llamadas++
	return domain.ConvocatoriaPublicada{}, c.err
}

func (c *casosPrueba) Listar(context.Context, application.Orden) ([]ports.ResumenSolicitudPropia, error) {
	c.llamadas++
	return []ports.ResumenSolicitudPropia{{SolicitudRef: "sol_x", Estado: domain.EstadoBorrador, Version: 2}}, c.err
}

func (c *casosPrueba) LeerBorrador(context.Context, application.Orden, string) (application.BorradorLeido, error) {
	c.llamadas++
	return application.BorradorLeido{}, c.err
}

func (c *casosPrueba) GuardarBorrador(_ context.Context, _ application.Orden, e application.EntradaBorrador) (ports.ResultadoGuardado, error) {
	c.llamadas++
	c.entrada = e
	p, _ := baremacion.PuntosDesdeMicropuntos(1_400_000)
	return ports.ResultadoGuardado{SolicitudRef: "sol_x", Version: e.VersionEsperada + 1, Puntuacion: p, Reutilizada: c.repetida}, c.err
}

func (c *casosPrueba) Presentar(_ context.Context, _ application.Orden, e application.EntradaPresentacion) (application.ReciboPresentacion, error) {
	c.llamadas++
	c.presenta = e
	return application.ReciboPresentacion{ResultadoPresentacion: ports.ResultadoPresentacion{SolicitudRef: e.SolicitudRef, NumeroJustificante: "2026/SOL-000001",
		ReciboRef: "recibo:seleccion-presentacion:x", PresentadaEn: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)},
		Servicios: map[string]ports.EstadoServicioExterno{"firma": ports.ServicioNoDisponible}}, c.err
}

func peticion(metodo, ruta, cuerpo string, cabeceras map[string]string) *http.Request {
	var r *http.Request
	if cuerpo == "" {
		r = httptest.NewRequest(metodo, ruta, nil)
	} else {
		r = httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
	}
	for k, v := range cabeceras {
		r.Header.Set(k, v)
	}
	return r
}

func codigo(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &e)
	return e.Error.Codigo
}

func TestRutasMetodosYCabeceras(t *testing.T) {
	casos := &casosPrueba{}
	h, err := Nuevo(preparadorPrueba{}, casos)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		r      *http.Request
		estado int
		codigo string
	}{
		{peticion(http.MethodGet, RutaMisSolicitudes+"/otra", "", nil), 404, "recurso_no_encontrado"},
		{peticion(http.MethodPost, RutaMisSolicitudes, "", nil), 405, "metodo_no_permitido"},
		{peticion(http.MethodDelete, RutaBorrador, "", nil), 405, "metodo_no_permitido"},
		{peticion(http.MethodGet, RutaMisSolicitudes, "", map[string]string{"Cookie": "a=b"}), 400, "peticion_no_permitida"},
		{peticion(http.MethodGet, RutaMisSolicitudes+"?x=1", "", nil), 400, "peticion_no_permitida"},
		{peticion(http.MethodGet, RutaBorrador, "", nil), 400, "peticion_no_permitida"},
		{peticion(http.MethodPut, RutaBorrador, `{"convocatoria_ref":"c","version_esperada":0}`, nil), 400, "peticion_no_permitida"},
		{peticion(http.MethodPut, RutaBorrador, `{"convocatoria_ref":"c","version_esperada":0,"otro":1}`, map[string]string{"Idempotency-Key": "clave-000001"}), 400, "datos_no_validos"},
		{peticion(http.MethodPut, RutaBorrador, `{"convocatoria_ref":"c"}`, map[string]string{"Idempotency-Key": "clave-000001"}), 400, "datos_no_validos"},
		{peticion(http.MethodPost, RutaPresentacion, `{"solicitud_ref":"s","version_esperada":1,"declaracion_responsable":true}`, map[string]string{"Idempotency-Key": "clave-000001", "Sec-Fetch-Site": "cross-site"}), 400, "peticion_no_permitida"},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, c.r)
		if w.Code != c.estado || codigo(t, w) != c.codigo || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s: %d %s", c.r.Method, c.r.URL, w.Code, w.Body.String())
		}
	}
	if casos.llamadas != 0 {
		t.Fatal("una petición rechazada llegó a los casos de uso")
	}
}

func TestSinIdentidadResponde401SinLlamarCasos(t *testing.T) {
	casos := &casosPrueba{}
	h, _ := Nuevo(preparadorPrueba{err: httpcomun.ErrAutenticacionAusente}, casos)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodGet, RutaMisSolicitudes, "", nil))
	if w.Code != 401 || codigo(t, w) != "autenticacion_requerida" || casos.llamadas != 0 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestGuardarYPresentar(t *testing.T) {
	casos := &casosPrueba{}
	h, _ := Nuevo(preparadorPrueba{}, casos)
	cuerpo := `{"convocatoria_ref":"bolsa-operario-diputacion-2026","version_esperada":0,"turno":"libre","datos":{"nombre":"Antonio"},` +
		`"requisitos":[{"clave":"nacionalidad","estado":"cumple"}],"meritos":[{"clave_grupo":"experiencia","clave_merito":"meses","descripcion":"","cantidad":"14"}]}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodPut, RutaBorrador, cuerpo, map[string]string{"Idempotency-Key": "clave-borrador-1"}))
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"puntuacion_autobaremo":"1.4"`) || casos.entrada.Clave != "clave-borrador-1" ||
		casos.entrada.Borrador.Datos.Nombre != "Antonio" || casos.entrada.Borrador.Meritos[0].Cantidad != "14" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	casos.repetida = true
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodPut, RutaBorrador, cuerpo, map[string]string{"Idempotency-Key": "clave-borrador-1"}))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"repetida":true`) {
		t.Fatalf("repetición: %d %s", w.Code, w.Body.String())
	}
	casos.repetida = false
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodPost, RutaPresentacion, `{"solicitud_ref":"sol_x","version_esperada":2,"declaracion_responsable":true}`, map[string]string{"Idempotency-Key": "clave-presenta-1"}))
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"numero_justificante":"2026/SOL-000001"`) || !strings.Contains(w.Body.String(), `"firma":"no_disponible"`) ||
		!casos.presenta.DeclaracionResponsable {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestErroresDelCasoDeUsoComoCodigosEstables(t *testing.T) {
	for err, esperado := range map[error]struct {
		estado int
		codigo string
	}{
		ports.ErrFueraDePlazo:                                             {409, "fuera_de_plazo"},
		ports.ErrClaveReutilizada:                                         {409, "clave_reutilizada"},
		ports.ErrVersionObsoleta:                                          {409, "version_obsoleta"},
		ports.ErrYaPresentada:                                             {409, "ya_presentada"},
		ports.ErrConvocatoriaActualizada:                                  {409, "convocatoria_actualizada"},
		ports.ErrRequisitoNoCumplido:                                      {422, "requisito_no_cumplido"},
		ports.ErrDatosIncompletos:                                         {422, "datos_incompletos"},
		ports.ErrDeclaracionRequerida:                                     {400, "declaracion_requerida"},
		ports.ErrConvocatoriaNoDisponible:                                 {404, "convocatoria_no_disponible"},
		ports.ErrSolicitudNoEncontrada:                                    {404, "recurso_no_encontrado"},
		errors.Join(ports.ErrDatosNoValidos, domain.ErrSolicitudInvalida): {400, "datos_no_validos"},
		vecdomain.ErrAutorizacionDenegada:                                 {403, "acceso_denegado"},
		ports.ErrNoDisponible:                                             {503, "servicio_no_disponible"},
		errors.New("cualquier otro"):                                      {500, "error_interno"},
	} {
		h, _ := Nuevo(preparadorPrueba{}, &casosPrueba{err: err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion(http.MethodPost, RutaPresentacion, `{"solicitud_ref":"sol_x","version_esperada":2,"declaracion_responsable":true}`, map[string]string{"Idempotency-Key": "clave-presenta-1"}))
		if w.Code != esperado.estado || codigo(t, w) != esperado.codigo || strings.Contains(w.Body.String(), "seleccion:") {
			t.Fatalf("%v → %d %s", err, w.Code, w.Body.String())
		}
	}
}

func TestSinBorradorEs404(t *testing.T) {
	h, _ := Nuevo(preparadorPrueba{}, &casosPrueba{err: ports.ErrSinBorrador})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodGet, RutaBorrador+"?convocatoria_ref=bolsa-operario-diputacion-2026", "", nil))
	if w.Code != 404 || codigo(t, w) != "sin_borrador" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestConvocatoriasListaPublica(t *testing.T) {
	h, _ := Nuevo(preparadorPrueba{}, &casosPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodGet, RutaConvocatorias, "", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"convocatoria_ref":"bolsa-operario-diputacion-2026"`) || !strings.Contains(w.Body.String(), `"abierta":true`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
