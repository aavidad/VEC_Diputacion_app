package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const (
	comisionPrueba = "dco_0123456789abcdefghijklmn"
	relacionPrueba = "rel_0123456789abcdefghijklmn"
)

type resolutorPrueba struct {
	llamadas  int
	solicitud dietasports.SolicitudOperacionBorrador
	err       error
}

func (r *resolutorPrueba) ResolverIdentidadEfectivaBorrador(_ context.Context, solicitud dietasports.SolicitudOperacionBorrador) (dietasports.IdentidadEfectivaBorrador, error) {
	r.llamadas++
	r.solicitud = solicitud
	return dietasports.IdentidadEfectivaBorrador{}, r.err
}

type casoUsoPrueba struct {
	creaciones, listas, detalles int
	resultado                    dietasports.ResultadoBorradorComision
	pagina                       dietasports.PaginaBorradoresPropios
	err                          error
}

func (c *casoUsoPrueba) CrearPropio(context.Context, dietasports.IdentidadEfectivaBorrador, dietasports.SolicitudCrearBorradorPropio) (dietasports.ResultadoBorradorComision, error) {
	c.creaciones++
	return c.resultado, c.err
}
func (c *casoUsoPrueba) ObtenerPropio(context.Context, dietasports.IdentidadEfectivaBorrador, string) (dietasports.ResultadoBorradorComision, error) {
	c.detalles++
	return c.resultado, c.err
}
func (c *casoUsoPrueba) ListarPropios(context.Context, dietasports.IdentidadEfectivaBorrador, dietasports.ConsultaBorradoresPropios) (dietasports.PaginaBorradoresPropios, error) {
	c.listas++
	return c.pagina, c.err
}

func nuevoManejadorPrueba(t *testing.T) (*ManejadorBorradores, *resolutorPrueba, *casoUsoPrueba) {
	t.Helper()
	r := &resolutorPrueba{}
	c := &casoUsoPrueba{}
	m, err := NuevoManejadorBorradores(r, c)
	if err != nil {
		t.Fatal(err)
	}
	return m, r, c
}

func TestManejadorBorradoresListaPropiaSinIdentidadDelCliente(t *testing.T) {
	m, r, c := nuevoManejadorPrueba(t)
	peticion := httptest.NewRequest(http.MethodGet, RutaBorradores+"?limit=20&relacion_ref="+relacionPrueba, nil)
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || c.listas != 1 || r.llamadas != 1 {
		t.Fatalf("status=%d resolver=%d listar=%d: %s", respuesta.Code, r.llamadas, c.listas, respuesta.Body.String())
	}
	if r.solicitud.RelacionRef != relacionPrueba || r.solicitud.Consulta.Limite != 20 || r.solicitud.Operacion != dietasports.OperacionConsultarBorrador {
		t.Fatalf("solicitud inesperada: %#v", r.solicitud)
	}
	if respuesta.Body.String() != "{\"items\":[]}\n" {
		t.Fatalf("respuesta inesperada: %s", respuesta.Body.String())
	}
	if respuesta.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("falta no-store")
	}
}

func TestManejadorBorradoresRechazaIdentidadEnCabecerasAntesDeResolver(t *testing.T) {
	m, r, c := nuevoManejadorPrueba(t)
	peticion := httptest.NewRequest(http.MethodGet, RutaBorradores+"?limit=20", nil)
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("X-Vec-Actor", "act_inventado")
	respuesta := httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest || r.llamadas != 0 || c.listas != 0 {
		t.Fatalf("status=%d resolver=%d listar=%d", respuesta.Code, r.llamadas, c.listas)
	}
}

func TestManejadorBorradoresRechazaCookieAntesDeResolverSinFiltrarDetalle(t *testing.T) {
	m, r, c := nuevoManejadorPrueba(t)
	peticion := httptest.NewRequest(http.MethodGet, RutaBorradores+"?limit=20", nil)
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Cookie", "sesion=prueba")
	respuesta := httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest || r.llamadas != 0 || c.listas != 0 || respuesta.Body.String() != "{\"error\":\"dietas.error.peticion_invalida\"}\n" || respuesta.Header().Get("Set-Cookie") != "" {
		t.Fatalf("status=%d resolver=%d listar=%d cuerpo=%s set-cookie=%q", respuesta.Code, r.llamadas, c.listas, respuesta.Body.String(), respuesta.Header().Get("Set-Cookie"))
	}
}

func TestManejadorBorradoresCreaYDistingueReplay(t *testing.T) {
	m, r, c := nuevoManejadorPrueba(t)
	c.resultado.Recibo = dietasports.ReciboBorradorComision{Referencia: "rcd_01234567-89ab-cdef-0123-456789abcdef", Version: 1, RegistradoEn: time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)}
	c.resultado.Comision.Referencia = comisionPrueba
	c.resultado.Comision.Estado = "borrador"
	c.resultado.Comision.FechaInicio = "2026-09-22"
	c.resultado.Comision.FechaFin = "2026-09-22"
	c.resultado.Comision.Motivo = "Reunión de coordinación"
	c.resultado.Comision.RelacionRef = relacionPrueba
	cuerpo := `{"clave_idempotencia":"clave_idempotente_0001","fecha_inicio":"2026-09-22","fecha_fin":"2026-09-22","motivo":"Reunión de coordinación","codigos_ruta":[],"relacion_ref":"` + relacionPrueba + `"}`
	peticion := httptest.NewRequest(http.MethodPost, RutaBorradores, strings.NewReader(cuerpo))
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	respuesta := httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusCreated || c.creaciones != 1 || r.solicitud.Crear.RelacionRef != relacionPrueba {
		t.Fatalf("status=%d crear=%d: %s", respuesta.Code, c.creaciones, respuesta.Body.String())
	}
	if !strings.Contains(respuesta.Body.String(), `"registrado_en":"2026-09-21T08:00:00.000000Z"`) || !strings.Contains(respuesta.Body.String(), `"codigos_ruta":[]`) {
		t.Fatalf("contrato HTTP incompatible: %s", respuesta.Body.String())
	}

	c.resultado.Recibo.Repeticion = true
	peticion = httptest.NewRequest(http.MethodPost, RutaBorradores, strings.NewReader(cuerpo))
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	respuesta = httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || c.creaciones != 2 {
		t.Fatalf("replay status=%d crear=%d", respuesta.Code, c.creaciones)
	}
}

func TestManejadorBorradoresTraduceConflictoIdempotenciaSinDetalle(t *testing.T) {
	m, _, c := nuevoManejadorPrueba(t)
	c.err = dietasports.ErrConflictoIdempotencia
	cuerpo := `{"clave_idempotencia":"clave_idempotente_0001","fecha_inicio":"2026-09-22","fecha_fin":"2026-09-22","motivo":"Reunión de coordinación","codigos_ruta":[],"relacion_ref":"` + relacionPrueba + `"}`
	peticion := httptest.NewRequest(http.MethodPost, RutaBorradores, strings.NewReader(cuerpo))
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	respuesta := httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusConflict || respuesta.Body.String() != "{\"error\":\"dietas.error.conflicto_idempotencia\"}\n" {
		t.Fatalf("status=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}

func TestManejadorBorradoresRechazaEntradaAntesDelCasoDeUso(t *testing.T) {
	casos := []struct{ nombre, ruta, cuerpo, tipo string }{
		{"campo desconocido", RutaBorradores, `{"desconocido":true}`, "application/json; charset=utf-8"},
		{"tipo laxo", RutaBorradores, `{}`, "application/json"},
		{"query desconocida", RutaBorradores + "?limit=20&actor_ref=act_x", "", ""},
		{"ruta codificada", RutaBorradores + "/%64co_0123456789abcdefghijklmn", "", ""},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			m, r, c := nuevoManejadorPrueba(t)
			metodo := http.MethodGet
			var lector *strings.Reader
			if caso.cuerpo != "" {
				metodo = http.MethodPost
				lector = strings.NewReader(caso.cuerpo)
			} else {
				lector = strings.NewReader("")
			}
			peticion := httptest.NewRequest(metodo, caso.ruta, lector)
			peticion.Header.Set("Accept", "application/json")
			if caso.tipo != "" {
				peticion.Header.Set("Content-Type", caso.tipo)
			}
			respuesta := httptest.NewRecorder()
			m.ServeHTTP(respuesta, peticion)
			if respuesta.Code < 400 || r.llamadas != 0 || c.creaciones+c.listas+c.detalles != 0 {
				t.Fatalf("status=%d resolver=%d casos=%d", respuesta.Code, r.llamadas, c.creaciones+c.listas+c.detalles)
			}
		})
	}
}

func TestManejadorBorradoresDetalleOcultaAusenciaYClasificaErrores(t *testing.T) {
	m, r, c := nuevoManejadorPrueba(t)
	c.err = dietasports.ErrComisionNoEncontrada
	peticion := httptest.NewRequest(http.MethodGet, RutaBorradores+"/"+comisionPrueba+"?relacion_ref="+relacionPrueba, nil)
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusNotFound || c.detalles != 1 || r.solicitud.Referencia != comisionPrueba {
		t.Fatalf("status=%d detalle=%d: %s", respuesta.Code, c.detalles, respuesta.Body.String())
	}

	r.err = dietasports.ErrAccesoBorradorDenegado
	c.err = nil
	respuesta = httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusForbidden || c.detalles != 1 {
		t.Fatalf("status=%d detalle=%d", respuesta.Code, c.detalles)
	}

	r.err = errors.New("fuente caída")
	respuesta = httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", respuesta.Code)
	}

	r.err = dietasports.ErrRelacionNoValida
	respuesta = httptest.NewRecorder()
	m.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusUnprocessableEntity || !strings.Contains(respuesta.Body.String(), "dietas.error.relacion_no_valida") {
		t.Fatalf("status=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}

func TestNuevoManejadorBorradoresExigeDependencias(t *testing.T) {
	if _, err := NuevoManejadorBorradores(nil, &casoUsoPrueba{}); !errors.Is(err, ErrManejadorNoDisponible) {
		t.Fatalf("error=%v", err)
	}
	if _, err := NuevoManejadorBorradores(&resolutorPrueba{}, nil); !errors.Is(err, ErrManejadorNoDisponible) {
		t.Fatalf("error=%v", err)
	}
}
