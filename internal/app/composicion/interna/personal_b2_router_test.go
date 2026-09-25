package interna

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autoridadPersonalB2Prueba struct{ err error }

func (a autoridadPersonalB2Prueba) AutorizarRutaExacta(context.Context, string) error { return a.err }

type auditoriaPersonalB2Prueba struct {
	ordenes []vecports.OrdenAuditoriaFronteraRutaExacta
}

type auditoriaFronteraFallidaPrueba struct{}

func (auditoriaFronteraFallidaPrueba) RegistrarAuditoriaFronteraRutaExacta(context.Context, vecports.OrdenAuditoriaFronteraRutaExacta) error {
	return errors.New("almacén sintético no disponible")
}

func (a *auditoriaPersonalB2Prueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	a.ordenes = append(a.ordenes, orden)
	return orden.Validar()
}

func TestEnrutadorPersonalB2CierraDependenciasYRutas(t *testing.T) {
	llamadas := 0
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { llamadas++; w.WriteHeader(http.StatusNoContent) })
	auditoria := &auditoriaPersonalB2Prueba{}
	if _, err := nuevoEnrutadorPersonalB2(h, h, nil, h, h, h, h, autoridadPersonalB2Prueba{}, auditoria); !errors.Is(err, ErrAPIInternaNoDisponible) {
		t.Fatalf("dependencia ausente: %v", err)
	}
	if _, err := nuevoEnrutadorPersonalB2(h, h, h, h, h, h, nil, autoridadPersonalB2Prueba{}, auditoria); !errors.Is(err, ErrAPIInternaNoDisponible) {
		t.Fatalf("catálogo ausente: %v", err)
	}
	if _, err := nuevoEnrutadorPersonalB2(h, h, h, nil, h, h, h, autoridadPersonalB2Prueba{}, auditoria); !errors.Is(err, ErrAPIInternaNoDisponible) {
		t.Fatalf("lista de empleados ausente: %v", err)
	}
	router, err := nuevoEnrutadorPersonalB2(h, h, h, h, h, h, h, autoridadPersonalB2Prueba{}, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodPost, "/api/vec/personal/empleados"},
		{http.MethodPost, "/api/vec/personal/hechos"},
		{http.MethodGet, "/api/vec/personal/vacantes"},
		{http.MethodGet, httpapi.RutaEmpleadosOrganismoB2},
		{http.MethodGet, "/api/vec/personal/empleados/emp_0123456789abcdefghijkl"},
		{http.MethodGet, httpapi.RutaCatalogosRegistroEmpleadoB2},
		{http.MethodPost, httpapi.RutaCatalogosRegistroEmpleadoB2},
	} {
		respuesta := httptest.NewRecorder()
		router.ServeHTTP(respuesta, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if respuesta.Code != http.StatusNoContent {
			t.Fatalf("%s %q: %d", caso.metodo, caso.ruta, respuesta.Code)
		}
	}
	for _, ruta := range []string{
		"/api/vec/admin", "/api/vec/personal/empleados/emp_corta",
		"/api/vec/personal/empleados/emp_0123456789abcdefghijkl/otra",
		"/api/vec/personal/hechos/otra", "/api/vec/personal/empleados%2Femp_0123456789abcdefghijkl",
		"/api/vec/personal/catalogos-registro-empleado/otra",
		"/api/vec/personal/catalogos-registro-empleado%2Fotra",
		"/api/vec/personal/empleados-organismo/otra",
	} {
		respuesta := httptest.NewRecorder()
		router.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, nil))
		if respuesta.Code != http.StatusNotFound {
			t.Fatalf("ruta no canónica %q: %d", ruta, respuesta.Code)
		}
	}
	if llamadas != 7 || len(auditoria.ordenes) != 0 {
		t.Fatalf("llamadas=%d auditorías=%d", llamadas, len(auditoria.ordenes))
	}
}

func TestEnrutadorSinDependenciasB2ConservaGETCTYCierraCatalogos(t *testing.T) {
	llamadasCT := 0
	rutaCT := "/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento"
	manejadorCT := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadasCT++
		w.WriteHeader(http.StatusNoContent)
	})
	auditoria := &auditoriaPersonalB2Prueba{}
	ct, err := httpapi.NewHandlerSoloRutasExactas([]httpapi.RutaExacta{{Ruta: rutaCT, Manejador: manejadorCT}},
		autoridadPersonalB2Prueba{}, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	router, err := nuevoEnrutadorPersonalB2(ct, nil, nil, nil, nil, nil, nil,
		autoridadPersonalB2Prueba{}, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	ctRespuesta := httptest.NewRecorder()
	router.ServeHTTP(ctRespuesta, httptest.NewRequest(http.MethodGet, rutaCT, nil))
	if ctRespuesta.Code != http.StatusNoContent || llamadasCT != 1 {
		t.Fatalf("CT sin B2: estado=%d llamadas=%d", ctRespuesta.Code, llamadasCT)
	}
	for _, metodo := range []string{http.MethodGet, http.MethodPost} {
		respuesta := httptest.NewRecorder()
		router.ServeHTTP(respuesta, httptest.NewRequest(metodo, httpapi.RutaCatalogosRegistroEmpleadoB2, nil))
		if respuesta.Code != http.StatusNotFound || llamadasCT != 1 {
			t.Fatalf("catálogo ausente %s: estado=%d llamadasCT=%d", metodo, respuesta.Code, llamadasCT)
		}
	}
}

func TestEnrutadorPersonalB2Audita401Y403AntesDeDelegar(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
		motivo vecports.MotivoAuditoriaFronteraRutaExacta
	}{
		{httpapi.ErrAutenticacionRutaExactaRequerida, http.StatusUnauthorized, vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida},
		{httpapi.ErrAccesoRutaExactaDenegado, http.StatusForbidden, vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
	} {
		llamadas := 0
		h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadas++ })
		auditoria := &auditoriaPersonalB2Prueba{}
		router, err := nuevoEnrutadorPersonalB2(h, h, h, h, h, h, h, autoridadPersonalB2Prueba{caso.err}, auditoria)
		if err != nil {
			t.Fatal(err)
		}
		respuesta := httptest.NewRecorder()
		router.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/api/vec/personal/empleados/emp_0123456789abcdefghijkl", nil))
		if respuesta.Code != caso.estado || llamadas != 0 || len(auditoria.ordenes) != 1 ||
			auditoria.ordenes[0].Motivo != caso.motivo || auditoria.ordenes[0].Ruta != "/api/vec/personal/empleados/{emp_ref}" {
			t.Fatalf("denegación: estado=%d llamadas=%d auditoría=%#v", respuesta.Code, llamadas, auditoria.ordenes)
		}
	}
}

func TestEnrutadorCatalogosB2AuditaGETyPOSTDenegados(t *testing.T) {
	for _, metodo := range []string{http.MethodGet, http.MethodPost} {
		for _, caso := range []struct {
			err    error
			estado int
		}{
			{httpapi.ErrAutenticacionRutaExactaRequerida, http.StatusUnauthorized},
			{httpapi.ErrAccesoRutaExactaDenegado, http.StatusForbidden},
		} {
			llamadas := 0
			h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadas++ })
			auditoria := &auditoriaPersonalB2Prueba{}
			router, err := nuevoEnrutadorPersonalB2(h, h, h, h, h, h, h, autoridadPersonalB2Prueba{caso.err}, auditoria)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			router.ServeHTTP(respuesta, httptest.NewRequest(metodo, httpapi.RutaCatalogosRegistroEmpleadoB2, nil))
			if respuesta.Code != caso.estado || llamadas != 0 || len(auditoria.ordenes) != 1 ||
				auditoria.ordenes[0].Ruta != httpapi.RutaCatalogosRegistroEmpleadoB2 || auditoria.ordenes[0].Validar() != nil {
				t.Fatalf("%s catálogo: estado=%d llamadas=%d auditoría=%#v", metodo, respuesta.Code, llamadas, auditoria.ordenes)
			}
		}
	}
}

func TestPuentePersonalB2ExigeCertificadoYAuditaSinEmpleadoEnRuta(t *testing.T) {
	extractor := &extractorAsercionSeguimientoPrueba{err: errors.New("certificado no disponible")}
	auditoria := &auditoriaDenegacionSeguimientoPrueba{}
	llamadas := 0
	puente := &puenteConsultaSeguimiento{
		extractor: extractor, auditoria: auditoria, personalB2: true, limiteCuerpo: 1024,
		api: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadas++ }),
	}
	puente.fachada.Store(&FachadaIdentidadOffline{})
	respuesta := httptest.NewRecorder()
	puente.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet,
		"/api/vec/personal/empleados/emp_0123456789abcdefghijkl?vigente_en=2026-09-25", nil))
	if respuesta.Code != http.StatusUnauthorized || llamadas != 0 || len(auditoria.ordenes) != 1 ||
		auditoria.ordenes[0].Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaPersonal ||
		auditoria.ordenes[0].Ruta != "/api/vec/personal/empleados/{emp_ref}" ||
		auditoria.ordenes[0].Validar() != nil {
		t.Fatalf("frontera B2: estado=%d llamadas=%d auditoría=%#v", respuesta.Code, llamadas, auditoria.ordenes)
	}
	respuesta = httptest.NewRecorder()
	puente.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet,
		"/api/vec/personal/empleados%2Femp_0123456789abcdefghijkl", nil))
	if respuesta.Code != http.StatusNotFound || extractor.llamadas != 1 || len(auditoria.ordenes) != 1 {
		t.Fatalf("ruta no canónica: estado=%d extracciones=%d auditorías=%d", respuesta.Code, extractor.llamadas, len(auditoria.ordenes))
	}
}

func TestPuenteNoResponde401SinAuditoriaDurableEnCTYCatalogosB2(t *testing.T) {
	for _, caso := range []struct {
		ruta     string
		personal bool
	}{
		{"/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento", false},
		{httpapi.RutaCatalogosRegistroEmpleadoB2, true},
	} {
		llamadas := 0
		puente := &puenteConsultaSeguimiento{
			extractor: &extractorAsercionSeguimientoPrueba{err: errors.New("certificado no disponible")},
			auditoria: auditoriaFronteraFallidaPrueba{}, personalB2: caso.personal, limiteCuerpo: 1024,
			api: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadas++ }),
		}
		puente.fachada.Store(&FachadaIdentidadOffline{})
		respuesta := httptest.NewRecorder()
		puente.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, caso.ruta, nil))
		if respuesta.Code != http.StatusServiceUnavailable || llamadas != 0 {
			t.Fatalf("ruta %q: estado=%d llamadas=%d", caso.ruta, respuesta.Code, llamadas)
		}
	}
}

func TestEnrutadorListaEmpleadosB2AuditaRutaFija(t *testing.T) {
	llamadas := 0
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadas++ })
	auditoria := &auditoriaPersonalB2Prueba{}
	router, err := nuevoEnrutadorPersonalB2(h, h, h, h, h, h, h, autoridadPersonalB2Prueba{httpapi.ErrAccesoRutaExactaDenegado}, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	router.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, httpapi.RutaEmpleadosOrganismoB2, nil))
	if respuesta.Code != http.StatusForbidden || llamadas != 0 || len(auditoria.ordenes) != 1 ||
		auditoria.ordenes[0].Ruta != httpapi.RutaEmpleadosOrganismoB2 || auditoria.ordenes[0].Validar() != nil {
		t.Fatalf("lista denegada: estado=%d llamadas=%d auditoría=%#v", respuesta.Code, llamadas, auditoria.ordenes)
	}
}
