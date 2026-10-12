package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRankingBolsaPaginaActaYPuntuacionOriginal(t *testing.T) {
	datos := datosBolsasRRHHPrueba()
	datos.Candidaturas[0].Orden = punteroEnteroBolsaRRHHPrueba(2)
	datos.Candidaturas[1].Orden = punteroEnteroBolsaRRHHPrueba(1)
	datos.Puntuaciones = map[string]string{"participacion:001": "12.3456"}
	h := nuevoManejadorBolsasRRHHDesarrollo(func(context.Context) (datasetBolsasRRHHDesarrollo, error) { return datos, nil })
	ruta := prefijoCandidatosRRHHDesarrollo + "/bolsa:constituida:administrativo/candidatos"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta+"?orden=acta&limite=1", nil))
	var r struct {
		Data struct {
			Candidatos []struct {
				Ref        string  `json:"participacion_ref"`
				Puntuacion *string `json:"puntuacion"`
			} `json:"candidatos"`
			Cursor string `json:"cursor_siguiente"`
		} `json:"data"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &r) != nil || len(r.Data.Candidatos) != 1 || r.Data.Candidatos[0].Ref != "participacion:001" || r.Data.Candidatos[0].Puntuacion == nil || *r.Data.Candidatos[0].Puntuacion != "12.3456" {
		t.Fatalf("ranking status=%d respuesta=%s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta+"?orden=acta&limite=1&cursor="+r.Data.Cursor, nil))
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &r) != nil || len(r.Data.Candidatos) != 1 || r.Data.Candidatos[0].Ref != "participacion:002" || r.Data.Candidatos[0].Puntuacion != nil {
		t.Fatalf("pagina2 status=%d respuesta=%s", w.Code, w.Body.String())
	}
	if _, ok := consultaCandidatos("orden=otro"); ok {
		t.Fatal("orden ajeno admitido")
	}
	if _, ok := consultaCandidatos("orden=acta&llamamiento=llamamiento:uno"); ok {
		t.Fatal("orden mezclado con llamamiento")
	}
}
