package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/dietas/ports"
)

type filaAuditoriaV2 struct{ permitido bool }

func (f filaAuditoriaV2) Scan(dest ...any) error { *dest[0].(*bool) = f.permitido; return nil }

type consultorAuditoriaV2 struct {
	consultas  []string
	argumentos [][]any
}

func (c *consultorAuditoriaV2) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	c.consultas = append(c.consultas, query)
	c.argumentos = append(c.argumentos, args)
	return filaAuditoriaV2{permitido: true}
}

func TestAuditoriaFronteraUsaSoloFuncionV2YPreflightNominal(t *testing.T) {
	c := &consultorAuditoriaV2{}
	r := &RegistradorAuditoriaFronteraComisionPostgreSQL{consultor: c}
	if err := r.Preflight(context.Background()); err != nil {
		t.Fatal(err)
	}
	orden := ports.OrdenAuditoriaFronteraComision{CorrelacionRef: "corr_no_disponible", Motivo: ports.MotivoFronteraAccesoDenegado, Ruta: ports.RutaAuditoriaFronteraCircuito, Accion: ports.AccionFronteraDecidir}
	if err := r.RegistrarAuditoriaFronteraComision(context.Background(), orden); err != nil {
		t.Fatal(err)
	}
	if len(c.consultas) != 2 || c.consultas[0] != consultaPreflightAuditoriaFronteraComision || c.consultas[1] != consultaRegistrarAuditoriaFronteraComision || strings.Contains(strings.Join(c.consultas, " "), "comision_v1") {
		t.Fatalf("auditoría no usa v2 nominal: %v", c.consultas)
	}
	if len(c.argumentos[1]) != 6 || c.argumentos[1][2] != ports.SuperficieAuditoriaFronteraComision || c.argumentos[1][3] != ports.RutaAuditoriaFronteraCircuito || c.argumentos[1][4] != ports.AccionFronteraDecidir {
		t.Fatalf("ruta/acción de rechazo alteradas: %v", c.argumentos[1])
	}
}
