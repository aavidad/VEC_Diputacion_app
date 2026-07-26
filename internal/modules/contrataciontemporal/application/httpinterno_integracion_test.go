package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	httpinterno "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type propuestaHTTPAplicacionPrueba struct {
	Data struct {
		Esquema            string                                              `json:"esquema"`
		Estado             string                                              `json:"estado"`
		ViaRecomendada     string                                              `json:"via_recomendada"`
		Evaluaciones       []evaluacionHTTPAplicacionPrueba                    `json:"evaluaciones"`
		IdentidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura `json:"identidad_semantica"`
	} `json:"data"`
}

type evaluacionHTTPAplicacionPrueba struct {
	ViaClave             string   `json:"via_clave"`
	Prioridad            uint16   `json:"prioridad"`
	Estado               string   `json:"estado"`
	ResultadosOmitidos   []string `json:"resultados_omitidos"`
	AusenciasBloqueantes []string `json:"ausencias_bloqueantes"`
	AusenciasAdmitidas   []string `json:"ausencias_admitidas"`
	NoHabilitantes       []string `json:"no_habilitantes"`
	Conflictos           []string `json:"conflictos"`
}

type reciboHTTPAplicacionPrueba struct {
	Data struct {
		Esquema              string `json:"esquema"`
		ReciboRef            string `json:"recibo_ref"`
		Estado               string `json:"estado"`
		DecisionCoberturaRef string `json:"decision_cobertura_ref"`
		VersionResultante    uint64 `json:"version_resultante"`
		ConfirmadaEn         string `json:"confirmada_en"`
	} `json:"data"`
}

type autoridadHTTPAplicacionPrueba struct {
	contexto httpinterno.ContextoCanalCobertura
}

func (a autoridadHTTPAplicacionPrueba) ResolverContextoCanalCobertura(context.Context) (httpinterno.ContextoCanalCobertura, error) {
	return a.contexto, nil
}

func TestManejadorHTTPRecorreServiciosRealesYProyectaResultados(t *testing.T) {
	t.Run("propuesta 200", func(t *testing.T) {
		escenario := application.NuevoEscenarioHTTPDecisionPrueba(t)
		s := escenario.Propuesta
		manejador := nuevoManejadorHTTPAplicacionPrueba(t, s.AutenticacionRef, s.SesionRef, s.PerfilRef, s.OrganizacionRef, escenario.Presentador, escenario.Decisor)
		respuesta := ejecutarHTTPAplicacionPrueba(t, manejador, httpinterno.RutaPropuestaCobertura, map[string]any{"expediente_ref": s.ExpedienteRef, "version_esperada": s.VersionEsperada})
		exigirRespuestaHTTPAplicacionPrueba(t, respuesta, http.StatusOK)
		contrato := decodificarHTTPAplicacionPrueba[propuestaHTTPAplicacionPrueba](t, respuesta)
		if contrato.Data.Esquema != "vec.contratacion-temporal.propuesta-cobertura.v1" || contrato.Data.Estado != "viable" || contrato.Data.ViaRecomendada != string(escenario.Decision.ViaElegida) || len(contrato.Data.Evaluaciones) == 0 || contrato.Data.Evaluaciones[0].ViaClave != string(escenario.Decision.ViaElegida) || !contrato.Data.IdentidadSemantica.CoincideExactamente(escenario.Decision.IdentidadSemantica) {
			t.Fatalf("propuesta no proyectó la rama viable exacta: %+v", contrato.Data)
		}
	})
	t.Run("decision 201", func(t *testing.T) {
		escenario := application.NuevoEscenarioHTTPDecisionPrueba(t)
		s := escenario.Decision
		manejador := nuevoManejadorHTTPAplicacionPrueba(t, s.AutenticacionRef, s.SesionRef, s.PerfilRef, s.OrganizacionRef, escenario.Presentador, escenario.Decisor)
		respuesta := ejecutarHTTPAplicacionPrueba(t, manejador, httpinterno.RutaDecisionCobertura, map[string]any{"expediente_ref": s.ExpedienteRef, "version_esperada": s.VersionEsperada, "clave_idempotencia": s.ClaveIdempotencia, "identidad_semantica": s.IdentidadSemantica, "via_elegida": s.ViaElegida, "motivo_clave": s.MotivoClave})
		exigirRespuestaHTTPAplicacionPrueba(t, respuesta, http.StatusCreated)
		contrato := decodificarHTTPAplicacionPrueba[reciboHTTPAplicacionPrueba](t, respuesta)
		exigirReciboHTTPAplicacionPrueba(t, contrato, "aplicada", s.VersionEsperada+1)
	})
	t.Run("rectificacion 201", func(t *testing.T) {
		escenario := application.NuevoEscenarioHTTPRectificacionPrueba(t)
		s := escenario.Rectificacion
		manejador := nuevoManejadorHTTPAplicacionPrueba(t, s.AutenticacionRef, s.SesionRef, s.PerfilRef, s.OrganizacionRef, escenario.Presentador, escenario.Decisor)
		respuesta := ejecutarHTTPAplicacionPrueba(t, manejador, httpinterno.RutaRectificacionCobertura, map[string]any{"expediente_ref": s.ExpedienteRef, "version_esperada": s.VersionEsperada, "clave_idempotencia": s.ClaveIdempotencia, "identidad_semantica": s.IdentidadSemantica, "via_elegida": s.ViaElegida, "motivo_clave": s.MotivoClave, "predecesora_ref": s.PredecesoraRef, "predecesora_huella": s.PredecesoraHuella})
		exigirRespuestaHTTPAplicacionPrueba(t, respuesta, http.StatusCreated)
		contrato := decodificarHTTPAplicacionPrueba[reciboHTTPAplicacionPrueba](t, respuesta)
		exigirReciboHTTPAplicacionPrueba(t, contrato, "denegada", 0)
	})
}

func nuevoManejadorHTTPAplicacionPrueba(t *testing.T, autenticacion, sesion, perfil, organizacion string, presentador httpinterno.PresentadorPropuestaCobertura, decisor httpinterno.EjecutorDecisionCobertura) http.Handler {
	t.Helper()
	autoridad := autoridadHTTPAplicacionPrueba{contexto: httpinterno.ContextoCanalCobertura{AutenticacionRef: autenticacion, SesionRef: sesion, PerfilRef: perfil, OrganizacionRef: organizacion}}
	manejador, err := httpinterno.NuevoManejadorCobertura(autoridad, presentador, decisor)
	if err != nil {
		t.Fatal(err)
	}
	return manejador
}

func ejecutarHTTPAplicacionPrueba(t *testing.T, manejador http.Handler, ruta string, cuerpo map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	contenido, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodPost, ruta, bytes.NewReader(contenido))
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	return respuesta
}

func exigirRespuestaHTTPAplicacionPrueba(t *testing.T, respuesta *httptest.ResponseRecorder, estado int) {
	t.Helper()
	if respuesta.Code != estado {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if respuesta.Header().Get("Cache-Control") != "no-store, no-transform" || respuesta.Header().Get("Pragma") != "no-cache" || respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" || respuesta.Header().Get("Set-Cookie") != "" || len(respuesta.Result().Cookies()) != 0 || respuesta.Header().Get("Content-Length") != strconv.Itoa(respuesta.Body.Len()) {
		t.Fatalf("cabeceras de respuesta inseguras: %v", respuesta.Header())
	}
	for _, secreto := range []string{"auditoria", "evento_", "actuacion_", "hmac-sha256"} {
		if strings.Contains(strings.ToLower(respuesta.Body.String()), secreto) {
			t.Fatalf("respuesta reveló %q: %s", secreto, respuesta.Body.String())
		}
	}
}

func decodificarHTTPAplicacionPrueba[T any](t *testing.T, respuesta *httptest.ResponseRecorder) T {
	t.Helper()
	var contrato T
	decodificador := json.NewDecoder(bytes.NewReader(respuesta.Body.Bytes()))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&contrato); err != nil {
		t.Fatalf("contrato JSON inválido: %v; cuerpo=%s", err, respuesta.Body.String())
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("JSON con contenido adicional: %v", err)
	}
	return contrato
}

func exigirReciboHTTPAplicacionPrueba(t *testing.T, contrato reciboHTTPAplicacionPrueba, estado string, version uint64) {
	t.Helper()
	instante, err := time.Parse(time.RFC3339Nano, contrato.Data.ConfirmadaEn)
	if err != nil || !domain.InstanteUTCCanonico(instante) || contrato.Data.Esquema != "vec.contratacion-temporal.recibo-cobertura.v1" || !domain.ReferenciaOpacaValida(contrato.Data.ReciboRef) || contrato.Data.Estado != estado {
		t.Fatalf("recibo base incoherente: %+v (%v)", contrato.Data, err)
	}
	if estado == "aplicada" {
		if !domain.ReferenciaOpacaValida(contrato.Data.DecisionCoberturaRef) || contrato.Data.VersionResultante != version {
			t.Fatalf("rama aplicada incompleta: %+v", contrato.Data)
		}
		return
	}
	if contrato.Data.DecisionCoberturaRef != "" || contrato.Data.VersionResultante != 0 {
		t.Fatalf("rama denegada filtró efecto: %+v", contrato.Data)
	}
}
