package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

func TestOrganizacionHistoricaCierreEsperaGETYRechazaEntradasNuevas(t *testing.T) {
	barrera := &barreraPeticionesOrganizacionHistorica{}
	entro, liberar, termino := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var consultas, cierres atomic.Int32
	h := barrera.proteger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		consultas.Add(1)
		close(entro)
		<-liberar
		w.WriteHeader(http.StatusNoContent)
	}))
	primera := httptest.NewRecorder()
	go func() {
		h.ServeHTTP(primera, httptest.NewRequest(http.MethodGet, httpapi.RutaOrganizacionHistoricaPersonal, nil))
		close(termino)
	}()
	<-entro
	cerrado := make(chan struct{})
	ctCerrado := make(chan struct{})
	cerrar := cierreOrganizacionHistoricaCombinada(barrera, func() {
		cierres.Add(1)
		close(cerrado)
	})
	go cerrar()
	go func() { barrera.detener(); close(ctCerrado) }()
	vence := time.After(time.Second)
	for {
		barrera.mu.Lock()
		cerrando := barrera.cerrando
		barrera.mu.Unlock()
		if cerrando {
			break
		}
		select {
		case <-vence:
			t.Fatal("el cierre no inició")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	select {
	case <-cerrado:
		t.Fatal("se cerraron recursos con GET activo")
	case <-ctCerrado:
		t.Fatal("CT cerró la sesión compartida con GET activo")
	default:
	}
	segunda := httptest.NewRecorder()
	h.ServeHTTP(segunda, httptest.NewRequest(http.MethodGet, httpapi.RutaOrganizacionHistoricaPersonal, nil))
	if segunda.Code != http.StatusServiceUnavailable || consultas.Load() != 1 {
		t.Fatal("una petición nueva usó recursos en cierre")
	}
	close(liberar)
	select {
	case <-termino:
	case <-time.After(time.Second):
		t.Fatal("el GET activo no terminó")
	}
	select {
	case <-cerrado:
	case <-time.After(time.Second):
		t.Fatal("el cierre no drenó el GET")
	}
	select {
	case <-ctCerrado:
	case <-time.After(time.Second):
		t.Fatal("el cierre CT no esperó la consulta")
	}
	cerrar()
	if primera.Code != http.StatusNoContent || cierres.Load() != 1 {
		t.Fatal("respuesta activa interrumpida o doble cierre de recursos")
	}
}
