package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestRegistroSeleccionNativaSoloExponeCausasFijas(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	for _, caso := range []struct {
		nombre   string
		causa    error
		esperado string
	}{
		{"origen", puertosbolsa.ErrOrigenLlamamientoCTNoDisponible, puertosbolsa.ErrOrigenLlamamientoCTNoDisponible.Error()},
		{"emision", puertosbolsa.ErrEmisionLlamamientoNoDisponible, puertosbolsa.ErrEmisionLlamamientoNoDisponible.Error()},
		{"contexto", appbolsa.ErrCambioSituacionParticipacionNoDisponible, appbolsa.ErrCambioSituacionParticipacionNoDisponible.Error()},
		{"desconocida", errors.New("PRIVADO: postgres://credencial@conexion"), "error_*errors.errorString"},
		{"sql", &pgconn.PgError{Code: "42501", Message: "PRIVADO", Detail: "PRIVADO"}, "sqlstate_42501"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			for _, envolver := range []bool{false, true} {
				var registro bytes.Buffer
				slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
				causa := caso.causa
				if envolver {
					causa = fmt.Errorf("PRIVADO: %w", causa)
				}
				registrarFalloSeleccionNativaCT(context.Background(), causa)
				var entrada map[string]any
				if err := json.Unmarshal(registro.Bytes(), &entrada); err != nil {
					t.Fatal(err)
				}
				esperado := caso.esperado
				if envolver && caso.nombre == "desconocida" {
					esperado = "error_*fmt.wrapError"
				}
				if entrada["causa_interna"] != esperado || entrada["etapa"] != "solicitar_llamamiento" || strings.Contains(registro.String(), "PRIVADO") {
					t.Fatalf("clasificación o privacidad incorrectas: %s", registro.String())
				}
			}
		})
	}
}
