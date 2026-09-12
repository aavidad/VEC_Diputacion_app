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

type autoridadRutaCierreSinCesePrueba struct{ llamadas int }

func (a *autoridadRutaCierreSinCesePrueba) ResolverOrganizacionCierreAdministrativo(context.Context) (string, error) {
	a.llamadas++
	return "ref_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil
}

type ejecutorRutaCierreSinCesePrueba struct{ llamadas int }

func (*ejecutorRutaCierreSinCesePrueba) Cerrar(context.Context, appct.SolicitudCerrarAdministrativamente) (ct.ResultadoCierreAdministrativo, error) {
	return ct.ResultadoCierreAdministrativo{}, errors.New("no esperado")
}
func (*ejecutorRutaCierreSinCesePrueba) ReabrirExcepcionalmente(context.Context, appct.SolicitudReabrirExcepcionalmente) (ct.ResultadoCierreAdministrativo, error) {
	return ct.ResultadoCierreAdministrativo{}, errors.New("no esperado")
}
func (e *ejecutorRutaCierreSinCesePrueba) CerrarSinCese(context.Context, appct.SolicitudCerrarAdministrativamente) (ct.ResultadoCierreAdministrativo, error) {
	e.llamadas++
	return ct.ResultadoCierreAdministrativo{}, errors.New("no esperado")
}

func TestNuevaRutaCierreSinCeseEsNominalYRechazaIdentidadNavegador(t *testing.T) {
	a, e := new(autoridadRutaCierreSinCesePrueba), new(ejecutorRutaCierreSinCesePrueba)
	ruta, err := NuevaRutaCierreAdministrativoSinCese(a, e)
	if err != nil || ruta.Ruta != httpinterno.RutaCerrarAdministrativamenteSinCese || ruta.Manejador == nil {
		t.Fatalf("declaracion inesperada: %#v %v", ruta, err)
	}
	body := `{"expediente_ref":"ref_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","seguimiento_ref":"ref_cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","version_esperada":1,"clave_idempotencia":"clave-cierre-12345678","transicion_clave":"cerrar_administrativamente_sin_cese","motivo_clave":"motivo_cierre","organizacion_ref":"forjada"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, ruta.Ruta, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	ruta.Manejador.ServeHTTP(w, r)
	if w.Code < http.StatusBadRequest || w.Code >= http.StatusInternalServerError || a.llamadas != 0 || e.llamadas != 0 {
		t.Fatalf("el navegador altero autoridad o ejecuto: status=%d autoridad=%d ejecuciones=%d", w.Code, a.llamadas, e.llamadas)
	}
	if _, err := NuevaRutaCierreAdministrativoSinCese(a, nil); err == nil {
		t.Fatal("ejecutor nulo aceptado")
	}
}
