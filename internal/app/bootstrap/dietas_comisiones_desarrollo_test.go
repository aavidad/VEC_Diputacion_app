package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

type registradorFronteraComisionPrueba struct {
	orden    dietasports.OrdenAuditoriaFronteraComision
	llamadas int
	err      error
	ctxErr   error
}

func (r *registradorFronteraComisionPrueba) RegistrarAuditoriaFronteraComision(ctx context.Context, orden dietasports.OrdenAuditoriaFronteraComision) error {
	r.orden, r.llamadas = orden, r.llamadas+1
	r.ctxErr = ctx.Err()
	return r.err
}

type autoridadExactaDelegadaPrueba struct{ llamadas int }

func (a *autoridadExactaDelegadaPrueba) AutorizarRutaExacta(context.Context, string) error {
	a.llamadas++
	return nil
}

func TestComisionesDietasSoloSeMontanConSelectorYMaterialNominal(t *testing.T) {
	if a, err := nuevasComisionesDietasDesarrollo(config.Config{DietasBorradoresEnabled: "false"}, nil, nil, materialDietasDesdeCTDesarrollo{}); err != nil || a != nil {
		t.Fatalf("Dietas inerte no debe abrir rutas: %v %v", a, err)
	}
	if a, err := nuevasComisionesDietasDesarrollo(config.Config{DietasBorradoresEnabled: "true"}, nil, nil, materialDietasDesdeCTDesarrollo{}); err == nil || a != nil {
		t.Fatalf("selector sin gobierno abrió ruta: %v %v", a, err)
	}
}

func TestFronteraComisionesDietasNoDelegaNiSirveSinSesion(t *testing.T) {
	for _, caso := range []struct {
		ruta, metodo string
		admitido     bool
	}{
		{dietashttp.RutaBorradores, http.MethodGet, true},
		{dietashttp.RutaBorradores, http.MethodPost, true},
		{dietashttp.RutaBorradores, http.MethodPut, false},
		{dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa", http.MethodGet, true},
		{dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa", http.MethodPost, false},
		{dietashttp.RutaBorradores + "/a/b", http.MethodGet, false},
		{dietashttp.RutaBorradores, http.MethodDelete, false},
	} {
		if obtenido := metodoComisionesDietasValido(caso.ruta, caso.metodo); obtenido != caso.admitido {
			t.Fatalf("método %s %s admitido=%t", caso.metodo, caso.ruta, obtenido)
		}
	}
	delegada := &autoridadExactaDelegadaPrueba{}
	a := autoridadExactasConDietas{delegada: delegada, dietas: &autoridadComisionesDietasDesarrollo{}}
	for _, ruta := range []string{dietashttp.RutaBorradores, dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa"} {
		if err := a.AutorizarRutaExacta(context.Background(), ruta); err != vechttp.ErrAutenticacionRutaExactaRequerida {
			t.Fatalf("ruta %q: se esperaba autenticación, recibida %v", ruta, err)
		}
	}
	if delegada.llamadas != 0 {
		t.Fatal("Dietas pasó por la autoridad ajena")
	}
	if err := a.AutorizarRutaExacta(context.Background(), "/api/vec/contratacion/alta"); err != nil || delegada.llamadas != 1 {
		t.Fatalf("ruta ajena no delegada: %v %d", err, delegada.llamadas)
	}
	llamadas := 0
	registrador := &registradorFronteraComisionPrueba{}
	protegida := (&autoridadComisionesDietasDesarrollo{registrador: registrador}).proteger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { llamadas++; w.WriteHeader(http.StatusNoContent) }))
	respuesta := httptest.NewRecorder()
	protegida.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, dietashttp.RutaBorradores, nil))
	if respuesta.Code != http.StatusUnauthorized || llamadas != 0 || respuesta.Header().Get("Cache-Control") != "no-store" || registrador.llamadas != 1 || registrador.orden.Motivo != dietasports.MotivoFronteraAutenticacion || registrador.orden.Ruta != dietasports.RutaAuditoriaFronteraComision || registrador.orden.Accion != dietasports.AccionFronteraListar {
		t.Fatalf("sin certificado: estado=%d llamadas=%d", respuesta.Code, llamadas)
	}
	registrador.err = errors.New("almacén caído")
	respuesta = httptest.NewRecorder()
	protegida.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, dietashttp.RutaBorradores+"/dco_sintetico", nil))
	if respuesta.Code != http.StatusServiceUnavailable || registrador.llamadas != 2 || registrador.orden.Ruta != dietasports.RutaAuditoriaFronteraDetalle || registrador.orden.Accion != dietasports.AccionFronteraMetodoNoAdmitido || registrador.orden.ActorRef != "" {
		t.Fatalf("auditoría caída o detalle filtrado: estado=%d llamadas=%d orden=%+v", respuesta.Code, registrador.llamadas, registrador.orden)
	}
	registrador.err = nil
	respuesta = httptest.NewRecorder()
	(&autoridadComisionesDietasDesarrollo{registrador: registrador}).denegar(respuesta, httptest.NewRequest(http.MethodPost, dietashttp.RutaBorradores, nil), http.StatusForbidden)
	if respuesta.Code != http.StatusForbidden || registrador.llamadas != 3 || registrador.orden.Motivo != dietasports.MotivoFronteraAccesoDenegado || registrador.orden.Accion != dietasports.AccionFronteraCrear || registrador.orden.ActorRef != "" {
		t.Fatalf("403 filtró sujeto o perdió acción: estado=%d orden=%+v", respuesta.Code, registrador.orden)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	respuesta = httptest.NewRecorder()
	(&autoridadComisionesDietasDesarrollo{registrador: registrador}).denegar(respuesta, httptest.NewRequest(http.MethodDelete, dietashttp.RutaBorradores, nil).WithContext(ctx), http.StatusForbidden)
	if respuesta.Code != http.StatusForbidden || registrador.ctxErr != nil || registrador.orden.Accion != dietasports.AccionFronteraMetodoNoAdmitido {
		t.Fatalf("cancelación o método denegado: estado=%d orden=%+v ctx=%v", respuesta.Code, registrador.orden, registrador.ctxErr)
	}
	for _, solicitud := range []*http.Request{
		httptest.NewRequest(http.MethodGet, dietashttp.RutaBorradores+"/a/b", nil),
		httptest.NewRequest(http.MethodDelete, dietashttp.RutaBorradores, nil),
	} {
		respuesta = httptest.NewRecorder()
		protegida.ServeHTTP(respuesta, solicitud)
		if respuesta.Code != http.StatusUnauthorized || llamadas != 0 || registrador.orden.Accion != dietasports.AccionFronteraMetodoNoAdmitido {
			t.Fatalf("familia o método eludió protección: estado=%d orden=%+v llamadas=%d", respuesta.Code, registrador.orden, llamadas)
		}
	}
}

func TestAutoridadComisionesDietasAuditaSesionInvalidaSinDelegar(t *testing.T) {
	registrador := &registradorFronteraComisionPrueba{}
	a := &autoridadComisionesDietasDesarrollo{registrador: registrador}
	autoridad := autoridadExactasConDietas{delegada: &autoridadExactaDelegadaPrueba{}, dietas: a}
	ctx := context.WithValue(context.Background(), claveContextoComisionesDietas{}, contextoComisionesDietas{autoridad: a, ruta: dietashttp.RutaBorradores, metodo: http.MethodGet})
	if err := autoridad.AutorizarRutaExacta(ctx, dietashttp.RutaBorradores); err != vechttp.ErrAccesoRutaExactaDenegado || registrador.llamadas != 1 || registrador.orden.Accion != dietasports.AccionFronteraListar {
		t.Fatalf("sesión inválida sin registro Dietas: err=%v orden=%+v", err, registrador.orden)
	}
	registrador.err = errors.New("auditoría caída")
	if err := autoridad.AutorizarRutaExacta(ctx, dietashttp.RutaBorradores); !errors.Is(err, ErrComposicionBorradoresDietasNoDisponible) || registrador.llamadas != 2 {
		t.Fatalf("fallo auditoría no cerró ruta: %v", err)
	}
}
