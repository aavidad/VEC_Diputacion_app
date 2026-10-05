package httpcopias

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

func TestErroresDeNegocioConservanIntentoNominal(t *testing.T) {
	for _, caso := range []struct {
		metodo, ruta, cuerpo string
		accion               p.Operacion
		err                  error
		estado               int
	}{
		{"GET", PrefijoV1, "", p.Consultar, p.ErrNoDisponible, 503},
		{"GET", PrefijoV1 + "/copia_aaaaaaaa", "", p.Consultar, p.ErrDenegado, 403},
		{"GET", PrefijoV1 + "/calendario", "", p.Consultar, p.ErrConflicto, 409},
		{"POST", PrefijoV1 + "/lanzamientos", lanzamiento, p.Lanzar, p.ErrNoDisponible, 503},
	} {
		t.Run(caso.ruta+caso.metodo, func(t *testing.T) {
			h, _, b := preparar(t)
			b.err = caso.err
			var intentos []Denegacion
			h.auditor = func(ctx context.Context, d Denegacion) error {
				intentos = append(intentos, d)
				return nil
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion(caso.metodo, caso.ruta, caso.cuerpo))
			if w.Code != caso.estado || len(intentos) != 1 || intentos[0].Accion != string(caso.accion) {
				t.Fatalf("estado=%d, intentos=%d", w.Code, len(intentos))
			}
			if intentos[0].ActorPersonaRef == "" || intentos[0].PerfilActivoRef == "" || intentos[0].CorrelacionRef == "" {
				t.Fatal("el intento perdió la identidad nominal")
			}
			if caso.metodo == "POST" && b.llamadas != 1 {
				t.Fatal("el error provocó una repetición del efecto")
			}
		})
	}
}

func TestCancelacionYFalloDelAuditorNoOcultanIntento(t *testing.T) {
	h, _, b := preparar(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	r := peticion(http.MethodPost, PrefijoV1+"/lanzamientos", lanzamiento).WithContext(ctx)
	llamadas := 0
	h.auditor = func(ctx context.Context, d Denegacion) error {
		llamadas++
		if ctx.Err() != nil || d.Accion != string(p.Lanzar) || d.ActorPersonaRef == "" {
			t.Fatal("el contexto cancelado impidió el registro nominal")
		}
		return errors.New("causa_privada_no_exponible")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if llamadas != 1 || b.llamadas != 0 || w.Code != 503 || strings.Contains(w.Body.String(), "causa_privada") {
		t.Fatal("no se conservó el fallo cerrado del intento")
	}
}
