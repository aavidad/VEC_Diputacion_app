package httpinscripcion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

type diagnosticoPrueba struct{}

func (diagnosticoPrueba) Error() string                            { return "dato privado: 12345678Z" }
func (diagnosticoPrueba) DiagnosticoInscripcion() (string, string) { return "ejecutar", "42501" }

func TestFalloTecnicoNoRegistraDatosPersonales(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	r := httptest.NewRequest(http.MethodGet, RutaRRHH+"/solicitud_inscripcion_"+strings.Repeat("a", 64), nil)
	registrarFalloInscripcion(r, errors.Join(inscripcion.ErrNoDisponible, diagnosticoPrueba{}))
	texto := registro.String()
	if !strings.Contains(texto, "sqlstate=42501") || !strings.Contains(texto, "etapa=ejecutar") ||
		strings.Contains(texto, "12345678Z") || strings.Contains(texto, "solicitud_inscripcion_") {
		t.Fatalf("diagnostico no minimizado: %s", texto)
	}
}

func TestDetalleConvocatoriaConEtiquetasLargasSeEntregaCompletoComoJSON(t *testing.T) {
	for _, etiqueta := range []string{strings.Repeat("<", 2048), strings.Repeat("\"", 2048), strings.Repeat("á", 1024)} {
		categorias := make([]map[string]string, 128)
		for i := range categorias {
			categorias[i] = map[string]string{"categoria_ref": "categoria:rpt:auxiliar", "categoria": etiqueta}
		}
		requisitos := make([]map[string]string, 32)
		for i := range requisitos {
			requisitos[i] = map[string]string{"codigo": "requisito", "descripcion": strings.Repeat("A", 2000)}
		}
		w := httptest.NewRecorder()
		responder(w, http.StatusOK, "vec.bolsa.inscripcion.convocatoria.v1", map[string]any{
			"convocatoria": map[string]any{"categorias": categorias, "requisitos": requisitos},
		})
		if w.Code != http.StatusOK || w.Body.Len() > 1<<20 || w.Header().Get("X-Content-Type-Options") != "nosniff" ||
			strings.Contains(w.Body.String(), `\u003c`) {
			t.Fatalf("detalle legítimo truncado o HTML escapado: estado=%d bytes=%d", w.Code, w.Body.Len())
		}
		var salida struct {
			Data struct {
				Convocatoria struct {
					Categorias []struct {
						Categoria string `json:"categoria"`
					} `json:"categorias"`
				} `json:"convocatoria"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil || len(salida.Data.Convocatoria.Categorias) != 128 ||
			salida.Data.Convocatoria.Categorias[127].Categoria != etiqueta {
			t.Fatalf("categoría final ausente: %v", err)
		}
	}
}

type preparadorPrueba struct{ propias, rrhh int }

func (p *preparadorPrueba) PrepararLecturaAspirante(_ *http.Request, _, _ string, _ inscripcion.Filtro, _ string) (inscripcion.Actor, error) {
	p.propias++
	return inscripcion.Actor{PersonaRef: "per_prueba_0001", PerfilRef: "prf_prueba_0001", SesionRef: "ses_prueba_0001"}, nil
}
func (p *preparadorPrueba) PrepararLecturaRRHH(_ *http.Request, _, _ string, _ inscripcion.Filtro, _ string) (inscripcion.Actor, error) {
	p.rrhh++
	return inscripcion.Actor{PersonaRef: "per_rrhh_0001", PerfilRef: "prf_rrhh_0001", SesionRef: "ses_rrhh_0001"}, nil
}
func (p *preparadorPrueba) PrepararPresentacion(r *http.Request, _ inscripcion.Presentacion) (inscripcion.Actor, error) {
	return p.PrepararLecturaAspirante(r, "", "", inscripcion.Filtro{}, "es")
}
func (p *preparadorPrueba) PrepararDecision(r *http.Request, _ inscripcion.Decision) (inscripcion.Actor, error) {
	return p.PrepararLecturaRRHH(r, "", "", inscripcion.Filtro{}, "es")
}
func (p *preparadorPrueba) PrepararIncorporacion(r *http.Request, _ inscripcion.Incorporacion) (inscripcion.Actor, error) {
	return p.PrepararLecturaRRHH(r, "", "", inscripcion.Filtro{}, "es")
}

type servicioPrueba struct {
	presentaciones int
	ultimoFiltro   inscripcion.Filtro
}

func TestInscripcionHandlerExternoNoDespachaRRHH(t *testing.T) {
	preparador := &preparadorPrueba{}
	h, err := NuevoExterno(preparador, &servicioPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{RutaRRHH, RutaRRHH + "/motivos", RutaRRHH + "/solicitud_inscripcion_" + strings.Repeat("a", 64)} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusNotFound || preparador.rrhh != 0 {
			t.Fatalf("RRHH despachado desde portal externo: %s / %d", ruta, w.Code)
		}
	}
}

func TestInscripcionRRHHSeleccionaConvocatoriaAntesDeBandeja(t *testing.T) {
	p := &preparadorPrueba{}
	h, err := NuevoInterno(p, &servicioPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaRRHH, nil))
	if w.Code != http.StatusBadRequest || p.rrhh != 0 {
		t.Fatalf("bandeja sin convocatoria: %d, preparaciones %d", w.Code, p.rrhh)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaRRHH+"/convocatorias?limite=20&idioma=es", nil))
	if w.Code != http.StatusOK || p.rrhh != 1 || !strings.Contains(w.Body.String(), "vec.bolsa.inscripciones.rrhh.convocatorias.v1") {
		t.Fatalf("selector histórico inaccesible: %d, preparaciones %d", w.Code, p.rrhh)
	}
}

func (*servicioPrueba) Abiertas(context.Context, inscripcion.Actor, int, string) (inscripcion.PaginaAbiertas, error) {
	return inscripcion.PaginaAbiertas{}, nil
}
func (*servicioPrueba) DetalleAbierta(context.Context, inscripcion.Actor, string) (inscripcion.BolsaAbierta, error) {
	return inscripcion.BolsaAbierta{}, nil
}
func (s *servicioPrueba) Presentar(_ context.Context, _ inscripcion.Actor, p inscripcion.Presentacion) (inscripcion.Recibo, error) {
	s.presentaciones++
	return inscripcion.Recibo{Solicitud: inscripcion.Solicitud{
		SolicitudRef: "sol_0000000000000001", ReciboRef: "rec_0000000000000001", ConvocatoriaRef: p.ConvocatoriaRef,
		Categoria: "Auxiliar administrativo", Estado: inscripcion.EstadoPendiente, Version: 1, RegistradaEn: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
	}}, nil
}
func (*servicioPrueba) Propias(context.Context, inscripcion.Actor, inscripcion.Filtro) (inscripcion.Pagina, error) {
	return inscripcion.Pagina{}, nil
}
func (*servicioPrueba) Propia(context.Context, inscripcion.Actor, string) (inscripcion.Solicitud, error) {
	return inscripcion.Solicitud{}, nil
}
func (s *servicioPrueba) PendientesRRHH(_ context.Context, _ inscripcion.Actor, f inscripcion.Filtro) (inscripcion.Pagina, error) {
	s.ultimoFiltro = f
	return inscripcion.Pagina{Solicitudes: []inscripcion.Solicitud{}, Total: 0}, nil
}
func (*servicioPrueba) ConvocatoriasRRHH(context.Context, inscripcion.Actor, int, string) (inscripcion.PaginaConvocatoriasGestion, error) {
	return inscripcion.PaginaConvocatoriasGestion{}, nil
}
func (*servicioPrueba) DetalleRRHH(context.Context, inscripcion.Actor, string) (inscripcion.Solicitud, error) {
	return inscripcion.Solicitud{}, nil
}
func (*servicioPrueba) MotivosRRHH(context.Context, inscripcion.Actor, string) (inscripcion.CatalogoMotivos, error) {
	return inscripcion.CatalogoMotivos{}, nil
}
func (*servicioPrueba) Decidir(context.Context, inscripcion.Actor, inscripcion.Decision) (inscripcion.Recibo, error) {
	return inscripcion.Recibo{}, nil
}
func (*servicioPrueba) Incorporar(context.Context, inscripcion.Actor, inscripcion.Incorporacion) (inscripcion.Recibo, error) {
	return inscripcion.Recibo{}, nil
}

func TestPresentacionNoAceptaIdentidadDelNavegador(t *testing.T) {
	p := &preparadorPrueba{}
	s := &servicioPrueba{}
	h, err := Nuevo(p, s)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, cuerpo string
		cabecera       string
	}{
		{"persona en cuerpo", `{"convocatoria_ref":"cv1_cHJ1ZWJh_v1","catalogo_version":1,"clave_idempotencia":"clave-aspirante-0001","persona_ref":"per_ajena"}`, ""},
		{"candidato en cuerpo", `{"convocatoria_ref":"cv1_cHJ1ZWJh_v1","catalogo_version":1,"clave_idempotencia":"clave-aspirante-0001","candidato_ref":"can_ajeno"}`, ""},
		{"cabecera de identidad", `{"convocatoria_ref":"cv1_cHJ1ZWJh_v1","catalogo_version":1,"clave_idempotencia":"clave-aspirante-0001"}`, "X-Vec-Persona"},
		{"cookie", `{"convocatoria_ref":"cv1_cHJ1ZWJh_v1","catalogo_version":1,"clave_idempotencia":"clave-aspirante-0001"}`, "Cookie"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, RutaPropias, strings.NewReader(caso.cuerpo))
			r.Header.Set("Content-Type", "application/json")
			if caso.cabecera != "" {
				r.Header.Set(caso.cabecera, "valor")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 400 || s.presentaciones != 0 {
				t.Fatalf("respuesta=%d presentaciones=%d", w.Code, s.presentaciones)
			}
		})
	}
}

func TestBandejaEstadoOmitidoEsPendienteYPaginaMismoFiltro(t *testing.T) {
	p := &preparadorPrueba{}
	s := &servicioPrueba{}
	h, _ := Nuevo(p, s)
	r := httptest.NewRequest(http.MethodGet, RutaRRHH+"?convocatoria_ref=cv1_cHJ1ZWJh_v1&limite=20", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || s.ultimoFiltro.Estado != inscripcion.EstadoPendiente ||
		s.ultimoFiltro.ConvocatoriaRef != "cv1_cHJ1ZWJh_v1" || s.ultimoFiltro.Limite != 20 || p.rrhh != 1 {
		t.Fatalf("estado=%d filtro=%+v rrhh=%d", w.Code, s.ultimoFiltro, p.rrhh)
	}
}

func TestInscripcionFiltraConsultaYCuerpoAntesDeAbrirSesion(t *testing.T) {
	preparador := &preparadorPrueba{}
	externo, err := NuevoExterno(preparador, &servicioPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	rechazadas := []*http.Request{
		httptest.NewRequest(http.MethodGet, RutaAbiertas+"?persona_ref=per_ajena", nil),
		httptest.NewRequest(http.MethodGet, RutaAbiertas+"?limite=1&limite=2", nil),
		httptest.NewRequest(http.MethodGet, RutaAbiertas+"?limite=%zz", nil),
		httptest.NewRequest(http.MethodGet, RutaPropias+"/solicitud_inscripcion_"+strings.Repeat("a", 64)+"?limite=1", nil),
		httptest.NewRequest(http.MethodPost, RutaPropias+"?idioma=es", strings.NewReader("{}")),
	}
	conCuerpo := httptest.NewRequest(http.MethodGet, RutaAbiertas, nil)
	conCuerpo.TransferEncoding = []string{"chunked"}
	rechazadas = append(rechazadas, conCuerpo)
	for _, r := range rechazadas {
		w := httptest.NewRecorder()
		externo.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || preparador.propias != 0 {
			t.Fatalf("entrada no permitida llegó al preparador: %s %s → %d", r.Method, r.URL, w.Code)
		}
	}
	w := httptest.NewRecorder()
	externo.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaAbiertas+"?limite=5&idioma=en", nil))
	if w.Code != http.StatusOK || preparador.propias != 1 {
		t.Fatalf("consulta legítima rechazada: %d", w.Code)
	}
}

func TestInscripcionHandlerInternoNoDespachaAspirante(t *testing.T) {
	preparador := &preparadorPrueba{}
	h, err := NuevoInterno(preparador, &servicioPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{RutaAbiertas, RutaPropias, RutaPropias + "/solicitud_inscripcion_" + strings.Repeat("a", 64)} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusNotFound || preparador.propias != 0 {
			t.Fatalf("ruta de aspirante despachada en vec-server: %s / %d", ruta, w.Code)
		}
	}
	if strings.HasPrefix(RutaPropias, "/api/vec/bolsa/mi-bolsa/") {
		t.Fatal("la inscripción comparte prefijo con Mi Bolsa")
	}
}

type preparadorDenegado struct {
	preparadorPrueba
	err error
}

func (p *preparadorDenegado) PrepararLecturaAspirante(*http.Request, string, string, inscripcion.Filtro, string) (inscripcion.Actor, error) {
	return inscripcion.Actor{}, p.err
}

func TestInscripcionDistingueSesionAusenteDeServicioCaidoYRegistraDenegacion(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	for _, caso := range []struct {
		err    error
		estado int
	}{{inscripcion.ErrSesionAusente, 401}, {inscripcion.ErrNoDisponible, 503}, {inscripcion.ErrAccesoDenegado, 403}} {
		h, err := NuevoExterno(&preparadorDenegado{err: caso.err}, &servicioPrueba{})
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaAbiertas, nil))
		if w.Code != caso.estado {
			t.Fatalf("%v → %d, se esperaba %d", caso.err, w.Code, caso.estado)
		}
	}
	texto := registro.String()
	if !strings.Contains(texto, "motivo=autenticacion_requerida") || !strings.Contains(texto, "motivo=acceso_denegado") ||
		!strings.Contains(texto, "ruta_clase=convocatorias_abiertas") || strings.Contains(texto, "per_") {
		t.Fatalf("denegación sin registrar o con datos personales: %s", texto)
	}
}
