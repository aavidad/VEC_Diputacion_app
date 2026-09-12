package contrataciontemporal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadRutaAnotacionNuevaPrueba struct{ llamadas int }

func (a *autoridadRutaAnotacionNuevaPrueba) ResolverContextoCanalAnotacionAdministrativa(context.Context) (httpinterno.ContextoCanalAnotacionAdministrativa, error) {
	a.llamadas++
	return httpinterno.ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: "ref_dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}, nil
}

type ejecutorRutaAnotacionNuevaPrueba struct{ registros int }

func (e *ejecutorRutaAnotacionNuevaPrueba) RegistrarAnotacionAdministrativa(context.Context, appct.SolicitudRegistrarAnotacionAdministrativa) (ct.ReciboAnotacionAdministrativa, error) {
	e.registros++
	return ct.ReciboAnotacionAdministrativa{}, errors.New("no esperado")
}
func (*ejecutorRutaAnotacionNuevaPrueba) RecuperarAnotacionAdministrativa(context.Context, string, string, httpinterno.ContextoCanalAnotacionAdministrativa) (ct.ReciboAnotacionAdministrativa, error) {
	return ct.ReciboAnotacionAdministrativa{}, errors.New("no esperado")
}

func TestNuevasRutasAnotacionAdministrativaDeclaranContratoYNoAceptanIdentidadNavegador(t *testing.T) {
	a, e := new(autoridadRutaAnotacionNuevaPrueba), new(ejecutorRutaAnotacionNuevaPrueba)
	rutas, err := NuevasRutasAnotacionAdministrativa(a, e)
	if err != nil || len(rutas) != 2 || rutas[0].Ruta != httpinterno.RutaAnotacionesAdministrativas || rutas[1].Ruta != httpinterno.RutaRecuperacionAnotacionesAdministrativas || rutas[0].Manejador == nil || rutas[0].Manejador != rutas[1].Manejador {
		t.Fatalf("declaracion inesperada: rutas=%#v err=%v", rutas, err)
	}
	body := `{"expediente_ref":"ref_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","version_esperada":1,"clave_idempotencia":"clave-anotacion-12345678","observaciones":"anotacion sintetica","autenticacion_ref":"forjada"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, rutas[0].Ruta, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rutas[0].Manejador.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity || e.registros != 0 || a.llamadas != 1 {
		t.Fatalf("el navegador altero identidad o ejecuto: status=%d autoridad=%d ejecuciones=%d", w.Code, a.llamadas, e.registros)
	}
	if _, err := NuevasRutasAnotacionAdministrativa(nil, e); err == nil {
		t.Fatal("autoridad nula aceptada")
	}
}
