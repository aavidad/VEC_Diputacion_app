package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestAltaRecuperacionFallidaConserva503SinCambiarCancelacionDirecta(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		causa  error
		estado int
		codigo string
	}{
		{"recuperacion cancelada", errors.Join(ports.ErrPersistenciaNoDisponible,
			ports.ErrFuenteNecesidadesAltaNoDisponible, context.Canceled),
			http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"recuperacion vencida", errors.Join(ports.ErrPersistenciaNoDisponible,
			ports.ErrFuenteNecesidadesAltaNoDisponible, context.DeadlineExceeded),
			http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"cancelacion directa", context.Canceled, http.StatusRequestTimeout, "peticion_cancelada"},
		{"plazo directo", context.DeadlineExceeded, http.StatusGatewayTimeout, "plazo_agotado"},
		{"persistencia sin fuente cancelada", errors.Join(ports.ErrPersistenciaNoDisponible,
			context.Canceled), http.StatusRequestTimeout, "peticion_cancelada"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			publico := clasificarErrorAlta(caso.causa)
			if publico.estado != caso.estado || publico.codigo != caso.codigo {
				t.Fatalf("estado/código público inesperado: %d %s", publico.estado, publico.codigo)
			}
		})
	}
}

type falloSQLInstantaneaPrueba struct{ codigo string }

func (e *falloSQLInstantaneaPrueba) Error() string {
	return "postgres://privado:secreto@host HMAC SQL actor"
}
func (e *falloSQLInstantaneaPrueba) SQLState() string { return e.codigo }

func TestAltaDiagnosticaLecturaOriginalSinPublicarCausa(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	for _, caso := range []struct{ codigo, etapa, sqlstate string }{
		{"42501", "sql", "42501"},
		{"42501\nsecreto", "desconocida", ""},
	} {
		t.Run(caso.codigo, func(t *testing.T) {
			var registro bytes.Buffer
			slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
			causa := errors.Join(ports.ErrFuenteNecesidadesAltaNoDisponible,
				&falloSQLInstantaneaPrueba{codigo: caso.codigo})
			w := httptest.NewRecorder()
			responderErrorAlta(w, httptest.NewRequest(http.MethodPost, RutaAltaSolicitudes, nil),
				errorServicioNoDisponible, causa)
			var log map[string]any
			var cuerpo envoltorioErrorAlta
			if err := json.Unmarshal(registro.Bytes(), &log); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusServiceUnavailable || cuerpo.Error.Codigo != "servicio_no_disponible" ||
				log["etapa"] != caso.etapa || log["sqlstate"] != caso.sqlstate ||
				log["centinela"] != "fuente_necesidades_no_disponible" ||
				log["correlacion_ref"] != cuerpo.Error.CorrelacionRef ||
				strings.Contains(registro.String()+w.Body.String(), "privado") ||
				strings.Contains(registro.String()+w.Body.String(), "secreto") {
				t.Fatal("diagnóstico de lectura original incompleto o expuesto")
			}
		})
	}
}
