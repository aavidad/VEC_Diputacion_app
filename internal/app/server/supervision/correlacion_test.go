package supervision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestCorrelacionHTTPHastaEscrituraTecnica(t *testing.T) {
	var destino bytes.Buffer
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: &destino})
	if err != nil {
		t.Fatal(err)
	}
	var esperadas []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlacion, ok := ports.CorrelacionIncidenciasPeticion(r.Context())
		if !ok {
			t.Fatal("petición sin correlación")
		}
		esperadas = append(esperadas, correlacion)
		switch len(esperadas) {
		case 1:
			for i := 0; i < 2; i++ {
				ports.EmitirIncidenciaTecnicaEnPeticion(r.Context(), emisor, domain.SolicitudIncidenciaTecnica{
					Codigo: domain.IncidenciaPostgresNoDisponible, Componente: domain.ComponenteIncidenciaPostgreSQL, Etapa: domain.EtapaIncidenciaConsulta,
				})
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			w.WriteHeader(http.StatusBadGateway)
		case 3:
			panic("dato privado")
		}
	})
	middleware := SupervisarRespuestas(handler, emisor)
	for i := 0; i < 3; i++ {
		// Ninguna cabecera ni dato externo es fuente de correlación.
		r := httptest.NewRequest(http.MethodGet, "/dato-privado?correlacion=00000000000000000000000000000000", nil)
		r.Header.Set("X-Correlation-ID", strings.Repeat("a", 32))
		r.Header.Set("Traceparent", "00-"+strings.Repeat("a", 32)+"-"+strings.Repeat("a", 16)+"-01")
		middleware.ServeHTTP(httptest.NewRecorder(), r)
	}
	if err := emisor.Cerrar(context.Background()); err != nil {
		t.Fatal(err)
	}
	var lineas []struct{ Codigo, Correlacion string }
	decodificador := json.NewDecoder(&destino)
	for decodificador.More() {
		var linea struct{ Codigo, Correlacion string }
		if err := decodificador.Decode(&linea); err != nil {
			t.Fatal(err)
		}
		lineas = append(lineas, linea)
	}
	if len(lineas) != 4 {
		t.Fatalf("incidencias = %v", lineas)
	}
	for i, esperado := range []string{esperadas[0], esperadas[0], esperadas[1], esperadas[2]} {
		if lineas[i].Correlacion != esperado || esperado == strings.Repeat("a", 32) {
			t.Fatal("correlación perdida o inyectada")
		}
	}
	if esperadas[0] == esperadas[1] || esperadas[1] == esperadas[2] || esperadas[0] == esperadas[2] {
		t.Fatal("peticiones comparten correlación")
	}
	if lineas[2].Codigo != string(domain.IncidenciaHTTPInternoFallido) || lineas[3].Codigo != string(domain.IncidenciaPanicoControlado) {
		t.Fatalf("códigos: %v", lineas)
	}
}

func TestFalloEntropiaNoImpidePeticionNiRevelaError(t *testing.T) {
	emisor := &emisorIncidenciasPrueba{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ports.CorrelacionIncidenciasPeticion(r.Context()); ok {
			t.Fatal("fallo tiene coincidencia")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	middleware := supervisarRespuestas(handler, emisor, func(ctx context.Context) (context.Context, error) {
		return ctx, errors.New("secreto fuente aleatoria")
	})
	respuesta := httptest.NewRecorder()
	middleware.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/", nil))
	if respuesta.Code != http.StatusNoContent || respuesta.Body.Len() != 0 {
		t.Fatal("fallo alteró respuesta")
	}
	if codigos := emisor.codigos(); len(codigos) != 1 || codigos[0] != domain.IncidenciaRecoleccionDegradada {
		t.Fatalf("pérdida no declarada: %v", codigos)
	}
}
