package auditoria_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/auditoria"
)

type identidadIntentoConsulta struct{ escenario escenarioConsulta }

func (i identidadIntentoConsulta) ResolverIdentidadConsulta(context.Context, *http.Request, auditoria.FuenteConsulta) (auditoria.IdentidadResuelta, error) {
	c := i.escenario.peticion.Contexto
	return auditoria.IdentidadResuelta{Vinculo: c.Vinculo, Resultado: c.Resultado, Correlacion: c.Correlacion}, nil
}

type opcionesIntentoConsulta struct{ escenario escenarioConsulta }

func (o opcionesIntentoConsulta) Actuales(context.Context) (auditoria.Opciones, error) {
	c := o.escenario.cfg
	return auditoria.Opciones{FinalidadRef: c.FinalidadRef, MotivoRef: c.Motivo.Referencia(), Motivo: c.Motivo, PermisoRequerido: auditoria.AccionConsultar, Fuentes: []string{"ct", "bolsa"}}, nil
}

func TestConsultaFallidaConservaAcuseConfirmadoEnHTTP(t *testing.T) {
	for _, invalido := range []bool{false, true} {
		t.Run(map[bool]string{false: "confirmado", true: "acuse_invalido"}[invalido], func(t *testing.T) {
			e := nuevoEscenarioConsulta(t, "ct")
			e.emisor.err = auditoria.ErrDenegada
			e.registro.invalido = invalido
			h, err := auditoria.NuevoManejador(e.servicio, opcionesIntentoConsulta{e}, identidadIntentoConsulta{e})
			if err != nil {
				t.Fatal(err)
			}
			f := e.peticion.Filtro
			body, err := json.Marshal(map[string]any{"fuente": f.Fuente, "expediente_ref": f.ExpedienteRef, "desde": f.Desde.Format(time.RFC3339Nano), "hasta": f.Hasta.Format(time.RFC3339Nano), "limite": f.Limite, "finalidad_ref": f.FinalidadRef, "motivo_ref": f.MotivoRef})
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, auditoria.RutaConsulta, strings.NewReader(string(body)))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			esperado, ref := http.StatusForbidden, "aud_v3_i_prueba"
			if invalido {
				esperado, ref = http.StatusServiceUnavailable, ""
			}
			if w.Code != esperado || w.Header().Get("X-Audit-Ref") != ref || len(e.registro.ordenes) != 1 || strings.Contains(w.Body.String(), ref) && ref != "" {
				t.Fatalf("respuesta o acuse inesperados: HTTP %d", w.Code)
			}
		})
	}
}

func TestAcuseConsultaNoSeInventaCuandoFallaElRegistro(t *testing.T) {
	e := nuevoEscenarioConsulta(t, "ct")
	e.emisor.err = auditoria.ErrDenegada
	_, err := e.servicio.Consultar(t.Context(), e.peticion)
	acuse, confirmado := auditoria.AcuseIntentoConsulta(err)
	if !errors.Is(err, auditoria.ErrDenegada) || !confirmado || acuse.AuditoriaRef != "aud_v3_i_prueba" {
		t.Fatal("acuse confirmado no conservado")
	}
	e.registro.invalido = true
	_, err = e.servicio.Consultar(t.Context(), e.peticion)
	if _, confirmado := auditoria.AcuseIntentoConsulta(err); confirmado || !errors.Is(err, auditoria.ErrNoDisponible) {
		t.Fatal("acuse inválido expuesto como confirmado")
	}
}
