package httppersonal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type camposPrueba struct {
	lista []string
	err   error
}

func (c camposPrueba) CamposVisiblesMiBolsa(context.Context) ([]string, error) { return c.lista, c.err }

type consultorCompleto struct{ llamadas int }

func (c *consultorCompleto) Consultar(context.Context, mibolsa.Orden) (puertosbolsa.InstantaneaMiBolsa, error) {
	c.llamadas++
	desde := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	fecha := desde.Add(48 * time.Hour)
	return puertosbolsa.InstantaneaMiBolsa{ConsultadaEn: desde.Add(time.Hour), Participaciones: []puertosbolsa.ParticipacionMiBolsa{{
		Bolsa: "bolsa:01", Categoria: "Auxiliar", Version: 3, OrdenInicial: 7, TotalInstantanea: 40,
		EstadoBolsa: "vigente", VigenteDesde: desde.Add(-time.Hour),
		SituacionActual:   &puertosbolsa.SituacionActualMiBolsa{Estado: "disponible_desde", Desde: desde, FechaDisponible: &fecha},
		UltimoLlamamiento: &puertosbolsa.UltimoLlamamientoMiBolsa{EmitidoEn: desde, Canal: "correo", Resultado: "enviado"},
	}}}, nil
}

func consultarConCampos(t *testing.T, campos puertosbolsa.CamposPortalMiBolsa) (int, string, int) {
	t.Helper()
	p, c := new(preparadorPrueba), new(consultorCompleto)
	h, err := NuevoConCampos(p, c, campos)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsa, nil))
	return w.Code, w.Body.String(), c.llamadas
}

func TestMiBolsaCamposOcultosNoSalenDelServidor(t *testing.T) {
	codigo, cuerpo, _ := consultarConCampos(t, camposPrueba{lista: []string{"bolsa", "fecha_disponible"}})
	if codigo != 200 || !strings.Contains(cuerpo, `"campos_visibles":["bolsa","fecha_disponible"]`) ||
		!strings.Contains(cuerpo, `"situacion_actual":{"fecha_disponible":"2026-09-22T10:00:00.000000Z"}`) {
		t.Fatalf("status=%d %s", codigo, cuerpo)
	}
	for _, oculto := range []string{"orden_inicial", "total_instantanea", "estado_bolsa", `"estado"`, "emitido_en", "disponible_desde"} {
		if strings.Contains(cuerpo, oculto) {
			t.Fatalf("sale %q oculto por catálogo: %s", oculto, cuerpo)
		}
	}
}

func TestMiBolsaCamposCompletosConservanLaRespuesta(t *testing.T) {
	codigo, cuerpo, _ := consultarConCampos(t, camposPrueba{lista: puertosbolsa.CamposPortalMiBolsaTodos()})
	for _, requerido := range []string{`"orden_inicial":7`, `"total_instantanea":40`, `"estado_bolsa":"vigente"`, `"estado":"disponible_desde"`, `"fecha_disponible":"2026-09-22`, `"ultimo_llamamiento":{`} {
		if codigo != 200 || !strings.Contains(cuerpo, requerido) {
			t.Fatalf("falta %q: %d %s", requerido, codigo, cuerpo)
		}
	}
	codigo, cuerpo, _ = consultarConCampos(t, camposPrueba{lista: []string{"bolsa", "estado"}})
	if codigo != 200 || !strings.Contains(cuerpo, `"estado":"disponible_desde"`) || !strings.Contains(cuerpo, `"fecha_disponible":null`) {
		t.Fatalf("estado sin fecha: %d %s", codigo, cuerpo)
	}
}

func TestMiBolsaCatalogoInvalidoONoDisponibleNoConsulta(t *testing.T) {
	for nombre, campos := range map[string]camposPrueba{
		"sin bolsa":     {lista: []string{"posicion"}},
		"desconocido":   {lista: []string{"bolsa", "puntuacion"}},
		"repetido":      {lista: []string{"bolsa", "bolsa"}},
		"no disponible": {err: errors.New("catálogo caído")},
		"lista vacía":   {lista: nil},
	} {
		codigo, cuerpo, llamadas := consultarConCampos(t, campos)
		if codigo != 503 || llamadas != 0 || !strings.Contains(cuerpo, "servicio_no_disponible") {
			t.Fatalf("%s: status=%d consultas=%d %s", nombre, codigo, llamadas, cuerpo)
		}
	}
	if _, err := NuevoConCampos(new(preparadorPrueba), new(consultorCompleto), nil); !errors.Is(err, ErrDependenciaNoDisponible) {
		t.Fatalf("campos nulos admitidos: %v", err)
	}
}
