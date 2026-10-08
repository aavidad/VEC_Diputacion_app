package administracionperfiles

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

type efectoHTTPPrueba struct {
	solicitudes []SolicitudEfectoNominalADMIN
	recibo      ReciboEfectoNominalADMIN
	err         error
}

func (g *efectoHTTPPrueba) AplicarEfectoNominal(_ context.Context, s SolicitudEfectoNominalADMIN) (ReciboEfectoNominalADMIN, error) {
	s.Material = bytes.Clone(s.Material)
	g.solicitudes = append(g.solicitudes, s)
	if g.err != nil {
		return ReciboEfectoNominalADMIN{}, g.err
	}
	return g.recibo, nil
}

func postCargoPrueba(h *Handler, cuerpo string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaPublicacionCargoCompetencial, cuerpo))
	return w
}

// La ruta de cargos entrega a la autoridad la sesión confiable y los bytes
// exactos; sólo existe en el handler de usuarios y con su límite; los errores
// se traducen sin segundo registro.
func TestHTTPEfectoNominalCargos(t *testing.T) {
	h, _, _, sesion, auditor, _ := loteHTTPPrueba(t)
	material := []byte(`{"esquema":"vec.personal.cargo-competencial.publicacion.v1"}`)
	cuerpo := `{"material_base64":"` + base64.StdEncoding.EncodeToString(material) + `"}`
	if w := postCargoPrueba(h, cuerpo); w.Code != http.StatusNotFound || auditor.llamadas != 1 {
		t.Fatalf("ruta abierta sin servicio: %d", w.Code)
	}
	servicio := &efectoHTTPPrueba{recibo: ReciboEfectoNominalADMIN{Cuerpo: json.RawMessage(`{"recibo_ref":"percar_x"}`), ConsumoAuditoriaRef: "aud_v3_x"}}
	if h.ConEfectoNominal(RutaPublicacionCargoCompetencial, 64, servicio) == nil {
		t.Fatal("efecto compuesto fuera del handler de usuarios")
	}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	if h.ConEfectoNominal(PrefijoV1+"/otra", 64, servicio) == nil || h.ConEfectoNominal(RutaPublicacionCargoCompetencial, 1, servicio) == nil ||
		h.ConEfectoNominal(RutaPublicacionCargoCompetencial, 5<<20, servicio) == nil {
		t.Fatal("ruta libre o límite fuera de rango admitidos")
	}
	if w := postCargoPrueba(h, cuerpo); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("handler de sólo lectura con escritura: %d", w.Code)
	}
	if h.ConEfectoNominal(RutaPublicacionCargoCompetencial, 64, servicio) != nil || h.ConEfectoNominal(RutaPublicacionCargoCompetencial, 64, servicio) == nil {
		t.Fatal("el efecto se compone una sola vez")
	}
	for _, otra := range []string{PrefijoV1 + "/lotes-ordinarios", RutaGobiernoPlanFirma} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, otra, `{}`))
		if w.Code != http.StatusNotFound {
			t.Fatalf("otra escritura abierta (%s): %d", otra, w.Code)
		}
	}
	w := postCargoPrueba(h, cuerpo)
	var r struct {
		Recibo              map[string]any `json:"recibo"`
		ConsumoAuditoriaRef string         `json:"consumo_auditoria_ref"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &r) != nil || r.Recibo["recibo_ref"] != "percar_x" ||
		r.ConsumoAuditoriaRef != "aud_v3_x" || len(servicio.solicitudes) != 1 {
		t.Fatalf("efecto no aplicado: %d %s", w.Code, w.Body.String())
	}
	s := servicio.solicitudes[0]
	if !bytes.Equal(s.Material, material) || s.Actor.PersonaRef != sesion.resultado.Actor.PersonaRef ||
		s.CorrelacionRef != sesion.resultado.CorrelacionRef || s.Instantanea.Validar() != nil {
		t.Fatal("la orden perdió la identidad confiable o los bytes exactos")
	}
	grande := `{"material_base64":"` + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 65))) + `"}`
	for nombre, caso := range map[string]struct {
		cuerpo string
		err    error
		recibo ReciboEfectoNominalADMIN
		estado int
	}{
		"material de más":   {grande, nil, servicio.recibo, http.StatusBadRequest},
		"base64 inválido":   {`{"material_base64":"%%%"}`, nil, servicio.recibo, http.StatusBadRequest},
		"campo desconocido": {`{"material_base64":"e30=","unidad_ref":"x"}`, nil, servicio.recibo, http.StatusBadRequest},
		"denegado":          {cuerpo, domain.ErrAutorizacionDenegada, servicio.recibo, http.StatusForbidden},
		"inválido":          {cuerpo, domain.ErrActoAdministracionPerfilesInvalido, servicio.recibo, http.StatusBadRequest},
		"conflicto":         {cuerpo, ErrConflictoEstado, servicio.recibo, http.StatusConflict},
		"caída":             {cuerpo, errors.New("x"), servicio.recibo, http.StatusServiceUnavailable},
		"recibo vacío":      {cuerpo, nil, ReciboEfectoNominalADMIN{}, http.StatusServiceUnavailable},
		"recibo no objeto":  {cuerpo, nil, ReciboEfectoNominalADMIN{Cuerpo: json.RawMessage(`[1]`), ConsumoAuditoriaRef: "a"}, http.StatusServiceUnavailable},
	} {
		servicio.err, servicio.recibo = caso.err, caso.recibo
		if w := postCargoPrueba(h, caso.cuerpo); w.Code != caso.estado {
			t.Fatalf("%s: %d", nombre, w.Code)
		}
	}
}

// Las dos rutas fijas se montan a la vez, cada una una sola vez, y cada POST
// llega a su servicio.
func TestHTTPEfectosNominalesDosRutas(t *testing.T) {
	_, _, _, sesion, auditor, _ := loteHTTPPrueba(t)
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	cargos := &efectoHTTPPrueba{recibo: ReciboEfectoNominalADMIN{Cuerpo: json.RawMessage(`{"recibo_ref":"percar_x"}`), ConsumoAuditoriaRef: "aud_v3_c"}}
	certs := &efectoHTTPPrueba{recibo: ReciboEfectoNominalADMIN{Cuerpo: json.RawMessage(`{"recibo_ref":"recibo_certificado_nominal:x"}`), ConsumoAuditoriaRef: "aud_v3_k"}}
	if h.ConEfectoNominal(RutaPublicacionCargoCompetencial, 64, cargos) != nil || h.ConEfectoNominal(RutaPublicacionCertificadoNominal, 64, certs) != nil ||
		h.ConEfectoNominal(RutaPublicacionCertificadoNominal, 64, cargos) == nil || h.ConEfectoNominal(RutaPublicacionCertificadoNominal, 64, nil) == nil {
		t.Fatal("montaje de dos efectos distinto del esperado")
	}
	cuerpo := `{"material_base64":"` + base64.StdEncoding.EncodeToString([]byte(`{"x":1}`)) + `"}`
	for ruta, s := range map[string]*efectoHTTPPrueba{RutaPublicacionCargoCompetencial: cargos, RutaPublicacionCertificadoNominal: certs} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, ruta, cuerpo))
		if w.Code != http.StatusOK || len(s.solicitudes) != 1 || !strings.Contains(w.Body.String(), s.recibo.ConsumoAuditoriaRef) {
			t.Fatalf("%s no llegó a su servicio: %d %s", ruta, w.Code, w.Body.String())
		}
	}
}
