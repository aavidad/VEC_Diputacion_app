package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
)

type claveActorInternoCompuestoPrueba struct{}

type autoridadRutaInternaCompuestaPrueba struct{ llamadas int }

func (a *autoridadRutaInternaCompuestaPrueba) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	a.llamadas++
	if ruta != "/api/vec/contratacion-temporal/cuadro/consultas" ||
		ctx.Value(claveActorInternoCompuestoPrueba{}) != "actor autenticado" {
		return ErrAutenticacionRutaExactaRequerida
	}
	return nil
}

func TestHandlerInternoCompuestoSoloPublicaMetodoYRutaDeclarados(t *testing.T) {
	store := memory.NewStore()
	servicio, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadRutaInternaCompuestaPrueba{}
	rutaCT := "/api/vec/contratacion-temporal/cuadro/consultas"
	h, err := NewHandlerInternoConCapacidades(servicio, HandlerOptions{
		RutasExactas: []RutaExacta{{Ruta: rutaCT, Manejador: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})}},
		AutoridadRutasExactas: autoridad,
	}, []CapacidadRutaInterna{{Metodo: http.MethodPost, Ruta: rutaCT}})
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		metodo, ruta string
		estado       int
	}{
		{http.MethodPost, rutaCT, http.StatusOK},
		{http.MethodGet, rutaCT, http.StatusNotFound},
		{http.MethodPost, rutaCT + "/", http.StatusNotFound},
		{http.MethodGet, "/api/vec/cronos/timecards", http.StatusNotFound},
		{http.MethodGet, "/api/vec/personal/rpt/positions", http.StatusNotFound},
		{http.MethodPost, "/api/vec/dietas/road-route", http.StatusNotFound},
		{http.MethodGet, "/api/vec/audit", http.StatusNotFound},
		{http.MethodGet, "/api/vec/admin", http.StatusNotFound},
		{http.MethodGet, "/api/vec/menu", http.StatusNotFound},
	}
	for _, caso := range casos {
		rec := httptest.NewRecorder()
		peticion := httptest.NewRequest(caso.metodo, caso.ruta, nil)
		peticion = peticion.WithContext(context.WithValue(peticion.Context(), claveActorInternoCompuestoPrueba{}, "actor autenticado"))
		h.ServeHTTP(rec, peticion)
		if rec.Code != caso.estado {
			t.Fatalf("%s %s: %d, se esperaba %d", caso.metodo, caso.ruta, rec.Code, caso.estado)
		}
	}
	if autoridad.llamadas != 1 {
		t.Fatalf("autoridad llamada %d veces; rutas ajenas llegaron al despacho", autoridad.llamadas)
	}
}

func TestHandlerInternoCompuestoRechazaCatalogoVacioDuplicadoYAjeno(t *testing.T) {
	store := memory.NewStore()
	servicio, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	ruta := "/api/vec/contratacion-temporal/cuadro/consultas"
	opciones := HandlerOptions{
		RutasExactas:          []RutaExacta{{Ruta: ruta, Manejador: http.NotFoundHandler()}},
		AutoridadRutasExactas: &autoridadRutaInternaCompuestaPrueba{},
	}
	for _, permitidas := range [][]CapacidadRutaInterna{
		nil,
		{{Metodo: http.MethodPost, Ruta: ruta}, {Metodo: http.MethodPost, Ruta: ruta}},
		{{Metodo: http.MethodGet, Ruta: ruta}},
		{{Metodo: http.MethodPost, Ruta: "/api/vec/otra"}},
		{{Metodo: http.MethodPost, Ruta: "/api/vec/otra*"}},
	} {
		h, err := NewHandlerInternoConCapacidades(servicio, opciones, permitidas)
		if h != nil || !errors.Is(err, ErrRutaExactaInvalida) {
			t.Fatalf("catálogo inválido = (%v, %v)", h, err)
		}
	}
}

func TestHandlerInternoCompuestoRechazaAutoridadesHeredadas(t *testing.T) {
	store := memory.NewStore()
	servicio, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	ruta := "/api/vec/contratacion-temporal/cuadro/consultas"
	base := HandlerOptions{
		RutasExactas:          []RutaExacta{{Ruta: ruta, Manejador: http.NotFoundHandler()}},
		AutoridadRutasExactas: &autoridadRutaInternaCompuestaPrueba{},
	}
	for _, alterar := range []func(*HandlerOptions){
		func(o *HandlerOptions) { o.AllowDemoIdentity = true },
		func(o *HandlerOptions) { o.TrustIdentityHeaders = true },
		func(o *HandlerOptions) { o.ManejadorRutaDietas = http.NotFoundHandler() },
		func(o *HandlerOptions) { o.IdentitySubjectHeader = "X-Identidad-Libre" },
	} {
		opciones := base
		alterar(&opciones)
		h, err := NewHandlerInternoConCapacidades(servicio, opciones, []CapacidadRutaInterna{{Metodo: http.MethodPost, Ruta: ruta}})
		if h != nil || !errors.Is(err, ErrRutaExactaInvalida) {
			t.Fatalf("autoridad heredada aceptada: (%v, %v)", h, err)
		}
	}
}
