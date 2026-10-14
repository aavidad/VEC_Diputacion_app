package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type casoUsoJustificantesC8Prueba struct{ llamadas int }

func (c *casoUsoJustificantesC8Prueba) Aportar(context.Context, ports.OrdenJustificacion, ports.PeticionAporteOriginal) (ports.ResultadoAnexoJustificacion, error) {
	c.llamadas++
	return ports.ResultadoAnexoJustificacion{}, ports.ErrJustificacionNoDisponible
}
func (c *casoUsoJustificantesC8Prueba) Revisar(context.Context, ports.OrdenJustificacion, ports.PeticionRevisionOriginal) (ports.ReciboJustificacion, error) {
	c.llamadas++
	return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
}

type resolverJustificantesC8Prueba struct{ llamadas int }

func (r *resolverJustificantesC8Prueba) ResolverJustificacion(*http.Request) (ports.OrdenJustificacion, error) {
	r.llamadas++
	return ports.OrdenJustificacion{}, ports.ErrJustificacionNoDisponible
}

func TestC8HTTPRechazaCamposExtraDuplicadosYRutasAjenaAntesDeResolver(t *testing.T) {
	c := &casoUsoJustificantesC8Prueba{}
	r := &resolverJustificantesC8Prueba{}
	m, err := NuevoManejadorJustificantesC8(c, r)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		ruta, cuerpo string
		estado       int
	}{
		{RutaAportarJustificanteC8, `{"solicitud_ref":"x","solicitud_ref":"y"}`, 400},
		{RutaAportarJustificanteC8, `{"solicitud_ref":"x","campo_extra":"y"}`, 400},
		{RutaRevisarJustificanteC8, `{"solicitud_ref":"x","clave_operacion":"y","version_esperada":"01","decision":"aceptada","motivo_ref":"m"}`, 400},
		{RutaAportarJustificanteC8 + "/ajeno", `{}`, 404},
	} {
		req := httptest.NewRequest(http.MethodPost, caso.ruta, strings.NewReader(caso.cuerpo))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, req)
		if w.Code != caso.estado || r.llamadas != 0 || c.llamadas != 0 {
			t.Fatalf("ruta %s: estado=%d resolver=%d caso=%d", caso.ruta, w.Code, r.llamadas, c.llamadas)
		}
	}
}
