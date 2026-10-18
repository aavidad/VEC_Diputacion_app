package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSeleccionLlamamiento409PublicaSoloContextoCoherente(t *testing.T) {
	c := &ports.ContextoComprobacionPendiente{
		AmbitoRef: "ambito:uno", BolsaRef: "bolsa:uno", CategoriaRef: "categoria:uno",
		CausaOfrecida: "baja", ExpedienteRef: "expediente:uno", HuellaPoliticaOfertasSHA256: strings.Repeat("a", 64),
		ModalidadOfrecida: "sustitucion", NecesidadRef: "necesidad:uno", OrdenActa: 2,
		OrganizacionRef: "organizacion:uno", ParticipacionRef: "participacion:dos", UnidadRef: "unidad:uno",
		VersionBolsa: 5, VersionExpediente: 13, VersionInstantanea: 9, VersionPoliticaOfertas: 7,
	}
	causa := &ports.ErrorCriteriosOfrecimientoPendientesRRHH{Comprobacion: ports.ComprobacionOfrecimientoPendiente{
		BolsaRef: "bolsa:uno", OrdenActa: 2, VersionPoliticaOfertas: 7,
		MotivoClave: "pendiente_comprobacion_rrhh", Contexto: c,
	}}
	grabar := func() *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		responderErrorSeleccionLlamamiento(r, httptest.NewRequest(http.MethodPost, "/", nil), clasificarErrorSeleccionLlamamiento(causa), causa)
		return r
	}
	r := grabar()
	var body struct {
		Comprobacion struct {
			Contexto map[string]any `json:"contexto"`
		} `json:"comprobacion_ofrecimiento"`
	}
	if r.Code != http.StatusConflict || json.Unmarshal(r.Body.Bytes(), &body) != nil || len(body.Comprobacion.Contexto) != 16 ||
		body.Comprobacion.Contexto["participacion_ref"] != "participacion:dos" ||
		body.Comprobacion.Contexto["version_expediente"] != float64(13) ||
		strings.Contains(r.Body.String(), "candidato_ref") || strings.Contains(r.Body.String(), "correo") {
		t.Fatalf("409 sin contexto mínimo: %d %s", r.Code, r.Body.String())
	}
	c.UnidadRef = "12345678Z"
	r = grabar()
	var invalido map[string]json.RawMessage
	if json.Unmarshal(r.Body.Bytes(), &invalido) != nil || invalido["comprobacion_ofrecimiento"] != nil {
		t.Fatalf("publicó contexto inválido: %s", r.Body.String())
	}
}
