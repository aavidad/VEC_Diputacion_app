package administracionperfiles

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestFronteraConservaFaseSesionResueltaIncompatible(t *testing.T) {
	for _, sinV2 := range []bool{false, true} {
		s := sesionADMINPrueba(t)
		if sinV2 {
			s = SesionConfiable{}
		} else {
			s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID = "per_ajena"
		}
		sesiones := &sesionPrueba{resultado: s}
		aud := &auditorPrueba{err: errors.New("sin_acuse")}
		lecturas := &lecturasPrueba{}
		h, err := NuevoHandlerLecturas("https://admin.example.test", sesiones, lecturas, aud)
		if err != nil {
			t.Fatal(err)
		}
		ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas", "").WithContext(ctx))
		if w.Code != 503 || lecturas.llamadas != 0 || aud.llamadas != 1 || !aud.ultima.SesionResuelta || strings.Contains(w.Body.String(), "per_ajena") {
			t.Fatal("fase_descartada_o_datos_expuestos")
		}
		if !sinV2 && aud.ultima.Evidencia.ValidarPara(aud.ultima.Actor) != nil {
			t.Fatal("V2_original_descartado")
		}
	}
}
func TestFronteraAnteriorAlResolverNoInventaSesion(t *testing.T) {
	aud := &auditorPrueba{}
	s := &sesionPrueba{}
	h, err := NuevoHandlerLecturas("https://admin.example.test", s, &lecturasPrueba{}, aud)
	if err != nil {
		t.Fatal(err)
	}
	r := peticionADMIN(http.MethodGet, PrefijoV1+"/personas", "")
	r.TLS = nil
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if s.llamadas != 0 || aud.ultima.SesionResuelta || aud.ultima.Actor.PersonaRef != "" {
		t.Fatal("fase_nominal_inventada")
	}
}
