package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverRemotoPrueba struct {
	contexto ports.ContextoMarcajePropio
	llamadas int
	err      error
}

func (r *resolverRemotoPrueba) ResolverMarcajeRemoto(*http.Request) (ports.ContextoMarcajePropio, error) {
	r.llamadas++
	return r.contexto, r.err
}

type casoRemotoPrueba struct {
	post           int
	get            int
	err            error
	disponibilidad ports.DisponibilidadMarcajeRemoto
}

func (c *casoRemotoPrueba) RegistrarMarcajeRemoto(_ context.Context, _ ports.ContextoMarcajePropio, s ports.SolicitudMarcajePropio) (ports.ReciboMarcajePropio, error) {
	c.post++
	return ports.ReciboMarcajePropio{Referencia: "recibo:cronos:00000000-0000-4000-8000-000000000001", MarcajeOriginalRef: "marcaje:cronos:" + s.ClaveOperacion, InstanteUTC: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)}, c.err
}

func (c *casoRemotoPrueba) ConsultarDisponibilidadMarcajeRemoto(context.Context, ports.ContextoMarcajePropio) (ports.DisponibilidadMarcajeRemoto, error) {
	c.get++
	if c.disponibilidad.Motivo != "" {
		return c.disponibilidad, c.err
	}
	return ports.DisponibilidadMarcajeRemoto{Motivo: "teletrabajo_no_autorizado"}, c.err
}

func (c *casoRemotoPrueba) RecuperarReciboMarcajeRemoto(_ context.Context, _ ports.ContextoRecuperacionMarcajeRemoto, s ports.SolicitudMarcajePropio) (ports.ReciboMarcajePropio, error) {
	return ports.ReciboMarcajePropio{Referencia: "recibo:cronos:00000000-0000-4000-8000-000000000001", MarcajeOriginalRef: "marcaje:cronos:" + s.ClaveOperacion, InstanteUTC: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC), Replay: true}, c.err
}

func contextoRemotoPrueba(t *testing.T, origen string) ports.ContextoMarcajePropio {
	t.Helper()
	canal, err := domain.NuevaAcreditacionCanalMarcaje(domain.DatosAcreditacionCanalMarcaje{
		PoliticaVersionRef: "politica:1", CanalRef: "canal:interno", OrigenRef: origen, CalidadRef: "calidad:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoMarcajePropio{CanalAcreditado: canal}
}

func TestRemotoNoAceptaOrigenNiEmpleadoDelCliente(t *testing.T) {
	resolver := &resolverRemotoPrueba{contexto: contextoRemotoPrueba(t, domain.OrigenMarcajeRemoto)}
	caso := &casoRemotoPrueba{}
	h, err := NuevoManejadorMarcajeRemoto(caso, resolver)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaRegistrarMarcajeRemoto, strings.NewReader(`{"movimiento":"entrada","clave_operacion":"op-cronos-0001","origen":"remoto"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || resolver.llamadas != 0 || caso.post != 0 {
		t.Fatalf("origen cliente aceptado: %d", w.Code)
	}
	r = httptest.NewRequest(http.MethodPost, RutaRegistrarMarcajeRemoto, strings.NewReader(`{"movimiento":"entrada","clave_operacion":"op-cronos-0001"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Empleado", "emp_ajeno")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || caso.post != 1 || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("marcaje remoto no delegado: %d", w.Code)
	}
}

func TestRemotoRechazaCanalNoRemotoYDisponibilidadIndisponible(t *testing.T) {
	resolver := &resolverRemotoPrueba{contexto: contextoRemotoPrueba(t, "oficina")}
	caso := &casoRemotoPrueba{}
	h, _ := NuevoManejadorMarcajeRemoto(caso, resolver)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaDisponibilidadMarcajeRemoto, nil))
	if w.Code != http.StatusServiceUnavailable || caso.get != 0 {
		t.Fatalf("origen ajeno consulto disponibilidad: %d", w.Code)
	}
	resolver.contexto = contextoRemotoPrueba(t, domain.OrigenMarcajeRemoto)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaDisponibilidadMarcajeRemoto, nil))
	var respuesta ports.DisponibilidadMarcajeRemoto
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &respuesta) != nil || respuesta.Autorizado || respuesta.ContinuidadConfirmada || respuesta.Periodo != nil || respuesta.Motivo != "teletrabajo_no_autorizado" || !strings.Contains(w.Body.String(), `"continuidad_confirmada":false`) || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("disponibilidad incorrecta: %d", w.Code)
	}
}

func TestRemotoDisponibilidadExponeContinuidadSinDatosAjenos(t *testing.T) {
	periodo := &ports.PeriodoTeletrabajo{DesdeUTC: time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC), HastaUTC: time.Date(2026, 9, 24, 16, 0, 0, 0, time.UTC)}
	for _, tc := range []struct {
		confirmada  bool
		motivo      string
		movimientos []domain.PunchKind
	}{
		{false, "continuidad_no_confirmada", nil},
		{true, "secuencia_no_permitida", nil},
		{true, "autorizado", []domain.PunchKind{domain.PunchEntry}},
	} {
		resolver := &resolverRemotoPrueba{contexto: contextoRemotoPrueba(t, domain.OrigenMarcajeRemoto)}
		caso := &casoRemotoPrueba{disponibilidad: ports.DisponibilidadMarcajeRemoto{Autorizado: true, ContinuidadConfirmada: tc.confirmada, MovimientosPermitidos: tc.movimientos, Periodo: periodo, Motivo: tc.motivo}}
		h, _ := NuevoManejadorMarcajeRemoto(caso, resolver)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaDisponibilidadMarcajeRemoto, nil))
		var datos map[string]any
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &datos) != nil || datos["continuidad_confirmada"] != tc.confirmada || datos["motivo"] != tc.motivo || len(datos["movimientos_permitidos"].([]any)) != len(tc.movimientos) || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
			t.Fatalf("continuidad HTTP: %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "empleado_ref") || strings.Contains(w.Body.String(), "ultimo_marcaje") {
			t.Fatalf("dato ajeno filtrado: %s", w.Body.String())
		}
	}
}

func TestRemotoNoPublicaDisponibilidadContradictoria(t *testing.T) {
	periodo := &ports.PeriodoTeletrabajo{DesdeUTC: time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC), HastaUTC: time.Date(2026, 9, 24, 16, 0, 0, 0, time.UTC)}
	for _, d := range []ports.DisponibilidadMarcajeRemoto{
		{Autorizado: true, ContinuidadConfirmada: false, Periodo: periodo, Motivo: "autorizado"},
		{Autorizado: false, ContinuidadConfirmada: true, Motivo: "teletrabajo_no_autorizado"},
		{Autorizado: true, ContinuidadConfirmada: true, Periodo: periodo, Motivo: "autorizado", MovimientosPermitidos: []domain.PunchKind{domain.PunchEntry, domain.PunchEntry}},
	} {
		resolver := &resolverRemotoPrueba{contexto: contextoRemotoPrueba(t, domain.OrigenMarcajeRemoto)}
		caso := &casoRemotoPrueba{disponibilidad: d}
		h, _ := NuevoManejadorMarcajeRemoto(caso, resolver)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaDisponibilidadMarcajeRemoto, nil))
		if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "empleado") || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("disponibilidad contradictoria: %d %s", w.Code, w.Body.String())
		}
	}
}

func TestRemotoDistingueErrorResolverEnGETyPOST(t *testing.T) {
	for _, tc := range []struct {
		nombre string
		err    error
		estado int
	}{
		{"identidad", ErrAutenticacionCronosRequerida, http.StatusUnauthorized},
		{"denegacion", ErrAccesoCronosDenegado, http.StatusForbidden},
		{"dependencia", ports.ErrDependenciaNoDisponible, http.StatusServiceUnavailable},
		{"fallo privado", errors.New("detalle_pdp_privado"), http.StatusServiceUnavailable},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			resolver := &resolverRemotoPrueba{err: tc.err}
			caso := &casoRemotoPrueba{}
			h, _ := NuevoManejadorMarcajeRemoto(caso, resolver)
			for _, ruta := range []string{RutaDisponibilidadMarcajeRemoto, RutaRegistrarMarcajeRemoto} {
				metodo := http.MethodGet
				var cuerpo *strings.Reader
				if ruta == RutaRegistrarMarcajeRemoto {
					metodo = http.MethodPost
					cuerpo = strings.NewReader(`{"movimiento":"entrada","clave_operacion":"op-cronos-0001"}`)
				} else {
					cuerpo = strings.NewReader("")
				}
				r := httptest.NewRequest(metodo, ruta, cuerpo)
				if metodo == http.MethodPost {
					r.Header.Set("Content-Type", "application/json")
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != tc.estado || caso.get != 0 || caso.post != 0 || strings.Contains(w.Body.String(), "privado") {
					t.Fatalf("resolver %s: %d %s", ruta, w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestRemotoDistingueErrorCasoDeUsoEnGETyPOST(t *testing.T) {
	for _, tc := range []struct {
		err    error
		estado int
	}{
		{ErrAccesoCronosDenegado, http.StatusForbidden},
		{ports.ErrDependenciaNoDisponible, http.StatusServiceUnavailable},
	} {
		resolver := &resolverRemotoPrueba{contexto: contextoRemotoPrueba(t, domain.OrigenMarcajeRemoto)}
		caso := &casoRemotoPrueba{err: tc.err}
		h, _ := NuevoManejadorMarcajeRemoto(caso, resolver)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaDisponibilidadMarcajeRemoto, nil))
		if w.Code != tc.estado || caso.get != 1 {
			t.Fatalf("disponibilidad: %d", w.Code)
		}
		r := httptest.NewRequest(http.MethodPost, RutaRegistrarMarcajeRemoto, strings.NewReader(`{"movimiento":"entrada","clave_operacion":"op-cronos-0001"}`))
		r.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.estado || caso.post != 1 {
			t.Fatalf("marcaje: %d", w.Code)
		}
	}
}

func TestRemotoPOSTTeletrabajoDenegadoYClaveEnConflicto(t *testing.T) {
	for _, tc := range []struct {
		err    error
		estado int
		codigo string
	}{
		{ports.ErrTeletrabajoNoAutorizado, http.StatusForbidden, "teletrabajo_no_autorizado"},
		{ports.ErrContinuidadMarcajeNoConfirmada, http.StatusServiceUnavailable, "continuidad_no_confirmada"},
		{ports.ErrMovimientoRemotoNoPermitido, http.StatusConflict, "secuencia_no_permitida"},
		{ports.ErrClaveOperacionEnConflicto, http.StatusConflict, "conflicto"},
	} {
		resolver := &resolverRemotoPrueba{contexto: contextoRemotoPrueba(t, domain.OrigenMarcajeRemoto)}
		caso := &casoRemotoPrueba{err: tc.err}
		h, _ := NuevoManejadorMarcajeRemoto(caso, resolver)
		r := httptest.NewRequest(http.MethodPost, RutaRegistrarMarcajeRemoto, strings.NewReader(`{"movimiento":"entrada","clave_operacion":"op-cronos-0001"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.estado || caso.post != 1 || strings.Contains(w.Body.String(), tc.err.Error()) || !strings.Contains(w.Body.String(), `"error":"`+tc.codigo+`"`) {
			t.Fatalf("efecto remoto: %d %s", w.Code, w.Body.String())
		}
	}
}

func TestRemotoNoAtribuyeDenegacionCentralATeletrabajo(t *testing.T) {
	resolver := &resolverRemotoPrueba{contexto: contextoRemotoPrueba(t, domain.OrigenMarcajeRemoto)}
	caso := &casoRemotoPrueba{err: ErrAccesoCronosDenegado}
	h, _ := NuevoManejadorMarcajeRemoto(caso, resolver)
	r := httptest.NewRequest(http.MethodPost, RutaRegistrarMarcajeRemoto, strings.NewReader(`{"movimiento":"entrada","clave_operacion":"op-cronos-0001"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), `"error":"acceso_denegado"`) || strings.Contains(w.Body.String(), "teletrabajo") {
		t.Fatalf("denegacion central mal atribuida: %d %s", w.Code, w.Body.String())
	}
}
