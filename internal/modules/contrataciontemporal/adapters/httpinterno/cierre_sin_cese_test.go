package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorCierreSinCeseHTTPPrueba struct {
	*ejecutorCierreAdministrativoHTTPPrueba
	llamadasSinCese int
	ultimaSinCese   application.SolicitudCerrarAdministrativamente
}

func (e *ejecutorCierreSinCeseHTTPPrueba) CerrarSinCese(
	ctx context.Context,
	solicitud application.SolicitudCerrarAdministrativamente,
) (ports.ResultadoCierreAdministrativo, error) {
	e.llamadasSinCese++
	e.ultimaSinCese = solicitud
	return e.responder(ctx, ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:       ports.OperacionCerrarAdministrativamenteSinCese,
		OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef,
		SeguimientoRef: solicitud.SeguimientoRef, VersionEsperada: solicitud.VersionEsperada,
		ClaveIdempotencia: solicitud.ClaveIdempotencia, TransicionClave: solicitud.TransicionClave,
		MotivoClave: solicitud.MotivoClave,
	})
}

func TestManejadorCierreSinCeseEjecutaOperacionNominalYMinimizaRecibo(t *testing.T) {
	autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
	ejecutor := &ejecutorCierreSinCeseHTTPPrueba{
		ejecutorCierreAdministrativoHTTPPrueba: &ejecutorCierreAdministrativoHTTPPrueba{},
	}
	respuesta := httptest.NewRecorder()
	nuevoManejadorCierreAdministrativoHTTPPrueba(t, autoridad, ejecutor).ServeHTTP(
		respuesta, peticionCierreSinCeseHTTPPrueba(t),
	)
	if respuesta.Code != http.StatusCreated || autoridad.llamadas != 1 ||
		ejecutor.llamadasSinCese != 1 || ejecutor.llamadasCerrar != 0 ||
		ejecutor.llamadasReabrir != 0 {
		t.Fatalf("estado=%d autoridad=%d sin_cese=%d legado=%d/%d cuerpo=%s",
			respuesta.Code, autoridad.llamadas, ejecutor.llamadasSinCese,
			ejecutor.llamadasCerrar, ejecutor.llamadasReabrir, respuesta.Body)
	}
	comprobarSalidaCierreAdministrativoHTTPPrueba(t, respuesta)
	entrada := entradaCierreSinCeseHTTPPrueba()
	solicitud := ejecutor.ultimaSinCese
	if solicitud.OrganizacionRef != autoridad.organizacionRef ||
		solicitud.ExpedienteRef != entrada.ExpedienteRef ||
		solicitud.SeguimientoRef != entrada.SeguimientoRef ||
		solicitud.VersionEsperada != *entrada.VersionEsperada ||
		solicitud.ClaveIdempotencia != entrada.ClaveIdempotencia ||
		solicitud.TransicionClave != domain.TransicionCerrarAdministrativamenteSinCese ||
		solicitud.MotivoClave != domain.ClaveCatalogo(entrada.MotivoClave) {
		t.Fatalf("solicitud sin cese alterada: %#v", solicitud)
	}
}

func TestManejadorCierreSinCeseDeniegaYNoPublicaReciboInvalido(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		ejecutor *ejecutorCierreSinCeseHTTPPrueba
		estado   int
		codigo   string
	}{
		{
			nombre: "denegacion",
			ejecutor: &ejecutorCierreSinCeseHTTPPrueba{
				ejecutorCierreAdministrativoHTTPPrueba: &ejecutorCierreAdministrativoHTTPPrueba{
					err: application.ErrCierreAdministrativoNoPermitido,
				},
			},
			estado: http.StatusForbidden, codigo: "acceso_denegado",
		},
		{
			nombre: "recibo invalido",
			ejecutor: &ejecutorCierreSinCeseHTTPPrueba{
				ejecutorCierreAdministrativoHTTPPrueba: &ejecutorCierreAdministrativoHTTPPrueba{
					resultadoInvalido: true,
				},
			},
			estado: http.StatusBadGateway, codigo: "resultado_no_confiable",
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			nuevoManejadorCierreAdministrativoHTTPPrueba(
				t, autoridadCierreAdministrativoHTTPValidaPrueba(), caso.ejecutor,
			).ServeHTTP(respuesta, peticionCierreSinCeseHTTPPrueba(t))
			if respuesta.Code != caso.estado || caso.ejecutor.llamadasSinCese != 1 ||
				!strings.Contains(respuesta.Body.String(), `"codigo":"`+caso.codigo+`"`) ||
				strings.Contains(respuesta.Body.String(), `"data"`) {
				t.Fatalf("estado=%d llamadas=%d cuerpo=%s", respuesta.Code,
					caso.ejecutor.llamadasSinCese, respuesta.Body)
			}
		})
	}
}

func entradaCierreSinCeseHTTPPrueba() cierreAdministrativoEntradaJSON {
	entrada := entradaCierreAdministrativoHTTPPrueba()
	entrada.TransicionClave = string(domain.TransicionCerrarAdministrativamenteSinCese)
	return entrada
}

func peticionCierreSinCeseHTTPPrueba(t *testing.T) *http.Request {
	t.Helper()
	cuerpo, err := json.Marshal(entradaCierreSinCeseHTTPPrueba())
	if err != nil {
		t.Fatalf("codificar cierre sin cese: %v", err)
	}
	peticion := httptest.NewRequest(http.MethodPost,
		RutaCerrarAdministrativamenteSinCese, strings.NewReader(string(cuerpo)))
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}
