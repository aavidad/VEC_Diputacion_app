package administracionperfiles

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

type gobiernoPlanHTTPPrueba struct {
	solicitudes []SolicitudGobiernoPlanFirmaADMIN
	err         error
}

func (g *gobiernoPlanHTTPPrueba) GobernarPlanFirma(_ context.Context, s SolicitudGobiernoPlanFirmaADMIN) (ReciboGobiernoPlanFirmaADMIN, error) {
	s.Material = bytes.Clone(s.Material)
	g.solicitudes = append(g.solicitudes, s)
	if g.err != nil {
		return ReciboGobiernoPlanFirmaADMIN{}, g.err
	}
	return ReciboGobiernoPlanFirmaADMIN{Accion: "vec.catalogos.publicar", CatalogoRef: "ct.plan.firma.sintetico:1",
		ReciboRef: "recibo:prueba", Estado: "publicado", Revision: 2, AuditoriaRef: "aud", ConsumoAuditoriaRef: "aud2"}, nil
}

func postPlanPrueba(h *Handler, cuerpo string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaGobiernoPlanFirma, cuerpo))
	return w
}

// La ruta entrega a la autoridad la sesión confiable y los bytes exactos del
// material; sin servicio compuesto no existe; los errores se traducen sin
// segundo registro (la autoridad ya dejó el intento común).
func TestHTTPGobiernoPlanFirma(t *testing.T) {
	h, _, _, sesion, auditor, _ := loteHTTPPrueba(t)
	material := []byte(`{"esquema":"vec.catalogos.plan-firma.gobierno.v1"}`)
	cuerpo := `{"material_base64":"` + base64.StdEncoding.EncodeToString(material) + `"}`
	if w := postPlanPrueba(h, cuerpo); w.Code != http.StatusNotFound || auditor.llamadas != 1 {
		t.Fatalf("ruta abierta sin servicio: %d", w.Code)
	}
	servicio := &gobiernoPlanHTTPPrueba{}
	// El handler de actos no es el de usuarios: no admite el gobierno del plan.
	if h.ConGobiernoPlanFirma(servicio) == nil {
		t.Fatal("gobierno del plan compuesto fuera del handler de usuarios")
	}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
	if err != nil {
		t.Fatal(err)
	}
	if w := postPlanPrueba(h, cuerpo); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("handler de sólo lectura con escritura: %d", w.Code)
	}
	if h.ConGobiernoPlanFirma(servicio) != nil || h.ConGobiernoPlanFirma(servicio) == nil {
		t.Fatal("el servicio se compone una sola vez")
	}
	// Con sólo el gobierno del plan, cualquier otra escritura no existe.
	if w := httptest.NewRecorder(); true {
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/lotes-ordinarios", `{}`))
		if w.Code != http.StatusNotFound {
			t.Fatalf("otra escritura abierta: %d", w.Code)
		}
	}
	w := postPlanPrueba(h, cuerpo)
	var r struct {
		Recibo ReciboGobiernoPlanFirmaADMIN `json:"recibo"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &r) != nil || r.Recibo.ReciboRef != "recibo:prueba" || len(servicio.solicitudes) != 1 {
		t.Fatalf("gobierno no aplicado: %d %s", w.Code, w.Body.String())
	}
	s := servicio.solicitudes[0]
	if !bytes.Equal(s.Material, material) || s.Actor.PersonaRef != sesion.resultado.Actor.PersonaRef ||
		s.CorrelacionRef != sesion.resultado.CorrelacionRef || s.Instantanea.Validar() != nil {
		t.Fatal("la orden perdió la identidad confiable o los bytes exactos")
	}
	for nombre, caso := range map[string]struct {
		cuerpo string
		err    error
		estado int
	}{
		"base64 inválido":   {`{"material_base64":"%%%"}`, nil, http.StatusBadRequest},
		"campo desconocido": {`{"material_base64":"e30=","organizacion_ref":"org_x"}`, nil, http.StatusBadRequest},
		"denegado":          {cuerpo, domain.ErrAutorizacionDenegada, http.StatusForbidden},
		"inválido":          {cuerpo, domain.ErrActoAdministracionPerfilesInvalido, http.StatusBadRequest},
		"conflicto":         {cuerpo, ErrConflictoEstado, http.StatusConflict},
		"caída":             {cuerpo, errors.New("x"), http.StatusServiceUnavailable},
	} {
		servicio.err = caso.err
		if w := postPlanPrueba(h, caso.cuerpo); w.Code != caso.estado {
			t.Fatalf("%s: %d", nombre, w.Code)
		}
	}
}
