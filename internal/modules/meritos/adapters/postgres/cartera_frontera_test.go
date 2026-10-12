package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaBoolFronteraCartera bool

func (f filaBoolFronteraCartera) Scan(dest ...any) error {
	*dest[0].(*bool) = bool(f)
	return nil
}

type consultaFronteraCarteraPrueba struct {
	consultas  []string
	argumentos [][]any
	permitido  bool
}

func (c *consultaFronteraCarteraPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	c.consultas = append(c.consultas, sql)
	c.argumentos = append(c.argumentos, args)
	return filaBoolFronteraCartera(c.permitido)
}

func TestFronteraCarteraSoloAceptaSuLoginYRutasNominales(t *testing.T) {
	c := &consultaFronteraCarteraPrueba{permitido: true}
	r, err := nuevoRegistradorFronteraCartera(context.Background(), c)
	if err != nil || len(c.consultas) != 1 || c.consultas[0] != preflightFronteraCarteraSQL {
		t.Fatalf("preflight M4: %v / %v", err, c.consultas)
	}
	o := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Motivo:     vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
		Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaMeritosCartera,
		Ruta:       "/api/vec/meritos/cartera"}
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), o); err != nil ||
		len(c.consultas) != 2 || c.consultas[1] != registrarFronteraCarteraSQL ||
		c.argumentos[1][4] != "" {
		t.Fatalf("401 minimizado M4: %v / %v", err, c.argumentos)
	}
	o.Ruta += "?persona_ref=otra"
	if r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), o) == nil || len(c.consultas) != 2 {
		t.Fatal("frontera aceptó ruta con selector")
	}
	o.Ruta = "/api/vec/meritos/cartera/retiradas"
	o.Motivo = vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	o.ActorRef = "per_aaaaaaaaaaaaaaaaaaaaaa"
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), o); err != nil || len(c.consultas) != 3 ||
		c.argumentos[2][4] != o.ActorRef {
		t.Fatalf("403 nominal M4: %v / %v", err, c.argumentos)
	}
	c.permitido = false
	if _, err := nuevoRegistradorFronteraCartera(context.Background(), c); err == nil {
		t.Fatal("preflight denegado montaría ruta")
	}
}
