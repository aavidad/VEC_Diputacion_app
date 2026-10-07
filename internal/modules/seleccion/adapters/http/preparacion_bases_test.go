package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Doble exclusivamente de transporte. No resuelve permisos ni persiste datos.
type preparadorHTTPPrueba struct {
	llamadas int
	fallo    error
	guardar  ports.SolicitudGuardarPreparacionBasesV3
}

func (p *preparadorHTTPPrueba) Guardar(_ context.Context, q ports.SolicitudGuardarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	p.llamadas++
	p.guardar = q
	return respuestaHTTPPrueba(q), p.fallo
}
func (p *preparadorHTTPPrueba) Consultar(context.Context, ports.SolicitudConsultarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	p.llamadas++
	return ports.ResultadoPreparacionBasesV3{}, p.fallo
}

func respuestaHTTPPrueba(q ports.SolicitudGuardarPreparacionBasesV3) ports.ResultadoPreparacionBasesV3 {
	h, _ := q.Material.HuellaSHA256()
	i, _ := prep.HuellaIntencion(q.Esperada, q.Material, q.Ambito)
	return ports.ResultadoPreparacionBasesV3{Estado: "guardada", Version: prep.Version{Ambito: q.Ambito, Estado: prep.Esperada{PreparacionRef: q.Esperada.PreparacionRef, Revision: 1, HuellaMaterialSHA256: h}, Material: q.Material}, Recibo: ports.ReciboPreparacionBases{ReciboRef: "recibo:prueba", HistoriaRef: "historia:prueba", AuditoriaRef: "auditoria:prueba", EventoRef: "evento:prueba", HuellaIntencionSHA256: i, ConfirmadaEn: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}}
}

const cuerpoGuardarPrueba = `{"esperada":{"preparacion_ref":"prep:sintetica","revision":0,"huella_material_sha256":""},"material":{"contenido":{},"referencias":[]},"clave_operacion":"operacion:sintetica"}`

func handlerPreparacionPrueba(t *testing.T, p *preparadorHTTPPrueba, frontera func(*http.Request) error) http.Handler {
	t.Helper()
	a, _ := bolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "uni_seleccionexterna")
	h, err := NuevaPreparacionBasesHandler(ConfigPreparacionBases{Preparador: p, ValidarFrontera: frontera, ResolverContexto: func(*http.Request) (ContextoPreparacionBases, error) {
		return ContextoPreparacionBases{Ambito: a, RegistrarErrorEntrada: func(_ context.Context, err error) error { return err }}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestPreparacionHTTPEntregaVersionPendientesYReciboSinCookies(t *testing.T) {
	p := &preparadorHTTPPrueba{}
	h := handlerPreparacionPrueba(t, p, func(*http.Request) error { return nil })
	r := httptest.NewRequest(http.MethodPost, RutaGuardarPreparacionBases, strings.NewReader(cuerpoGuardarPrueba))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var salida struct {
		Estado     string
		Pendientes []prep.Pendiente
		Recibo     map[string]any
	}
	if w.Code != http.StatusCreated || json.Unmarshal(w.Body.Bytes(), &salida) != nil || salida.Estado != "guardada" || len(salida.Pendientes) != 23 || len(salida.Recibo) != 6 || p.llamadas != 1 {
		t.Fatalf("HTTP=%d cuerpo=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("cabeceras fuera del contrato")
	}
	if p.guardar.Ambito.OrganizacionRef() != "org_diputaciongranada" {
		t.Fatal("ambito no procede del servidor")
	}
}

func TestPreparacionHTTPNoPermiteIdentidadAliasNiDuplicadosEnJSON(t *testing.T) {
	for _, body := range []string{
		strings.Replace(cuerpoGuardarPrueba, `"clave_operacion":"operacion:sintetica"`, `"clave_operacion":"operacion:sintetica","actor_ref":"per_cliente"`, 1),
		strings.Replace(cuerpoGuardarPrueba, `"revision":0`, `"revision":0,"revision":1`, 1),
		strings.Replace(cuerpoGuardarPrueba, `"revision":0`, `"Revision":0`, 1),
		strings.Replace(cuerpoGuardarPrueba, `"contenido":{}`, `"contenido":{"titulo":"x","titulo":"y"}`, 1),
		cuerpoGuardarPrueba + `{}`,
	} {
		p := &preparadorHTTPPrueba{}
		h := handlerPreparacionPrueba(t, p, func(*http.Request) error { return nil })
		r := httptest.NewRequest(http.MethodPost, RutaGuardarPreparacionBases, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || p.llamadas != 0 {
			t.Fatalf("entrada alcanzó servicio: %d / %d", w.Code, p.llamadas)
		}
	}
}

func TestPreparacionHTTPFronteraAntesDeConsumirYErroresNominales(t *testing.T) {
	p := &preparadorHTTPPrueba{}
	h := handlerPreparacionPrueba(t, p, func(*http.Request) error { return ports.ErrPreparacionBasesDenegada })
	r := httptest.NewRequest(http.MethodPost, RutaGuardarPreparacionBases, strings.NewReader(cuerpoGuardarPrueba))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || p.llamadas != 0 {
		t.Fatal("frontera no cerrada")
	}
	for _, caso := range []struct {
		err    error
		estado int
	}{{ports.ErrPreparacionBasesDenegada, 403}, {ports.ErrPreparacionBasesConflicto, 409}, {ports.ErrPreparacionBasesClaveReutilizada, 409}, {ports.ErrPreparacionBasesNoDisponible, 503}} {
		p := &preparadorHTTPPrueba{fallo: caso.err}
		h := handlerPreparacionPrueba(t, p, func(*http.Request) error { return nil })
		r := httptest.NewRequest(http.MethodPost, RutaGuardarPreparacionBases, strings.NewReader(cuerpoGuardarPrueba))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.estado || strings.Contains(w.Body.String(), `"preparacion"`) {
			t.Fatal("error emite material o cambia estado")
		}
	}
}

func TestPreparacionHTTPConstructorRechazaDependenciasVacias(t *testing.T) {
	var p *preparadorHTTPPrueba
	if _, err := NuevaPreparacionBasesHandler(ConfigPreparacionBases{Preparador: p}); !errors.Is(err, ports.ErrPreparacionBasesNoDisponible) {
		t.Fatal(err)
	}
}

func TestPreparacionHTTPDistingueDenegacionDeTimeoutOFalloContexto(t *testing.T) {
	for _, frontera := range []bool{true, false} {
		for _, caso := range []struct {
			err    error
			estado int
		}{
			{ports.ErrPreparacionBasesDenegada, 403}, {ports.ErrPreparacionBasesNoDisponible, 503},
			{context.DeadlineExceeded, 503}, {context.Canceled, 503}, {errors.New("dependencia_prueba"), 503},
			{errors.Join(ports.ErrPreparacionBasesDenegada, ports.ErrPreparacionBasesNoDisponible), 503},
		} {
			p := &preparadorHTTPPrueba{}
			c := ConfigPreparacionBases{Preparador: p, ValidarFrontera: func(*http.Request) error { return nil }, ResolverContexto: func(*http.Request) (ContextoPreparacionBases, error) { return ContextoPreparacionBases{}, nil }}
			if frontera {
				c.ValidarFrontera = func(*http.Request) error { return caso.err }
			} else {
				c.ResolverContexto = func(*http.Request) (ContextoPreparacionBases, error) { return ContextoPreparacionBases{}, caso.err }
			}
			h, err := NuevaPreparacionBasesHandler(c)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, RutaGuardarPreparacionBases, strings.NewReader(cuerpoGuardarPrueba))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || p.llamadas != 0 || strings.Contains(w.Body.String(), `"preparacion"`) || strings.Contains(w.Body.String(), `"acceso"`) {
				t.Fatalf("frontera=%t HTTP=%d llamadas=%d cuerpo=%s", frontera, w.Code, p.llamadas, w.Body.String())
			}
		}
	}
}
