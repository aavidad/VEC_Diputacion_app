package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type auditorRectificacionMontajePrueba struct {
	orden    personalports.OrdenAuditoriaFronteraRectificacionDietas
	llamadas int
	err      error
}

func (a *auditorRectificacionMontajePrueba) RegistrarAuditoriaFronteraRectificacionDietas(_ context.Context, orden personalports.OrdenAuditoriaFronteraRectificacionDietas) error {
	a.orden = orden
	a.llamadas++
	if err := orden.Validar(); err != nil {
		return err
	}
	return a.err
}

func TestRectificacionDietasSoloAbreSolicitudYLecturasNominales(t *testing.T) {
	ruta := personalhttp.RutaSolicitudesRectificacionDietas
	for _, caso := range []struct {
		ruta, metodo string
		admitido     bool
	}{
		{ruta, http.MethodGet, true}, {ruta, http.MethodPost, true},
		{ruta + "/competentes", http.MethodGet, true}, {ruta + "/competentes", http.MethodPost, false},
		{ruta + "/srd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", http.MethodPut, false},
		{ruta + "/competentes/ajeno", http.MethodGet, false}, {ruta + "/", http.MethodGet, false},
		{ruta, http.MethodDelete, false},
	} {
		if obtenido := metodoComisionesDietasValido(caso.ruta, caso.metodo); obtenido != caso.admitido {
			t.Fatalf("%s %s=%t", caso.metodo, caso.ruta, obtenido)
		}
		if !esRutaComisionesDietas(caso.ruta) {
			t.Fatalf("fuera de frontera Dietas: %s", caso.ruta)
		}
	}
}

func TestRectificacionDietasFronteraDeniegaYAuditaSinSesion(t *testing.T) {
	auditor := &auditorRectificacionMontajePrueba{}
	a := &autoridadComisionesDietasDesarrollo{registradorRectificacion: auditor}
	servidas := 0
	protegida := a.proteger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { servidas++ }))
	for _, ruta := range []string{personalhttp.RutaSolicitudesRectificacionDietas, personalhttp.RutaSolicitudesRectificacionDietas + "/competentes"} {
		w := httptest.NewRecorder()
		protegida.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusUnauthorized || servidas != 0 || auditor.orden.Validar() != nil || auditor.orden.ActorRef != "" || auditor.orden.Ruta != personalports.RutaSolicitudesRectificacionDietas || w.Header().Get("Set-Cookie") != "" {
			t.Fatalf("frontera: status=%d orden=%+v", w.Code, auditor.orden)
		}
	}
	if auditor.llamadas != 2 || auditor.orden.Accion != "consultar_competentes" {
		t.Fatalf("auditoría=%+v", auditor)
	}
	auditor.err = errors.New("auditoría no disponible")
	w := httptest.NewRecorder()
	protegida.ServeHTTP(w, httptest.NewRequest(http.MethodGet, personalhttp.RutaSolicitudesRectificacionDietas, nil))
	if w.Code != http.StatusServiceUnavailable || servidas != 0 {
		t.Fatal("fallo de auditoría cruzó frontera")
	}
}

func TestRectificacionDietasFronteraConservaDenegacionEstructural(t *testing.T) {
	auditor := &auditorRectificacionMontajePrueba{}
	a := &autoridadComisionesDietasDesarrollo{registradorRectificacion: auditor}
	if err := a.registrarDenegacionRectificacion(context.Background(), personalhttp.RutaSolicitudesRectificacionDietas, http.MethodPost, http.StatusBadRequest, ""); err != nil || auditor.orden.EstadoHTTP != http.StatusBadRequest || auditor.orden.Motivo != personalports.MotivoFronteraPersonalDenegado {
		t.Fatalf("orden=%+v error=%v", auditor.orden, err)
	}
}
