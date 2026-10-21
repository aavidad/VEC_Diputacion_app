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
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRegistroPropuestaSoloExponeEtapasYCausasFijas(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	for _, caso := range []struct {
		etapa, etapaEsperada string
		causa                error
		esperada             string
	}{
		{"preimagen", "preimagen", nil, "comprobacion_rechazada"},
		{"antecedente", "antecedente", ct.ErrResultadoPropuestaFormalizacionNoConfiable, ct.ErrResultadoPropuestaFormalizacionNoConfiable.Error()},
		{"resultado_emision", "resultado_emision", bp.ErrResultadoEmisionCTDenegado, bp.ErrResultadoEmisionCTDenegado.Error()},
		{"actor_anclaje", "actor_anclaje", ct.ErrAutorizacionDenegada, ct.ErrAutorizacionDenegada.Error()},
		{"consulta_anclaje", "consulta_anclaje", bp.ErrAceptacionNativaCTNoDisponible, bp.ErrAceptacionNativaCTNoDisponible.Error()},
		{"confirmacion", "confirmacion", errors.New("PRIVADO: postgres://credencial@conexion"), "no_clasificado"},
		{"PRIVADO", "desconocida", &pgconn.PgError{Code: "42501", Message: "PRIVADO", Detail: "PRIVADO"}, "no_clasificado"},
	} {
		t.Run(caso.etapaEsperada, func(t *testing.T) {
			for _, envolver := range []bool{false, true} {
				var registro bytes.Buffer
				slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
				causa := caso.causa
				if envolver && causa != nil {
					causa = fmt.Errorf("PRIVADO: %w", causa)
				}
				registrarFalloPropuestaCT(context.Background(), caso.etapa, causa)
				var entrada map[string]any
				if err := json.Unmarshal(registro.Bytes(), &entrada); err != nil {
					t.Fatal(err)
				}
				if entrada["etapa"] != caso.etapaEsperada || entrada["causa_interna"] != caso.esperada ||
					entrada["operacion"] != "propuesta_formalizacion" || strings.Contains(registro.String(), "PRIVADO") {
					t.Fatalf("clasificación o privacidad incorrectas: %s", registro.String())
				}
			}
		})
	}
}
