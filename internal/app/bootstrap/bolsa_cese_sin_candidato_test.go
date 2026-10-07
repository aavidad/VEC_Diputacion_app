package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type filaCeseB81Prueba struct {
	err         error
	reutilizada bool
}

func (f filaCeseB81Prueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*dest[0].(*bool) = f.reutilizada
	return nil
}

func TestCeseSinCandidatoSoloTrasDosAusenciasDeVinculo(t *testing.T) {
	evento := puertosct.EventoContratoBolsaPublicado{OrigenRef: "origen:cese:b81", HuellaSHA256: strings.Repeat("a", 64), OrigenPosicion: 81}
	claveAjena := &pgconn.PgError{Code: "23503"}
	casos := []struct {
		nombre       string
		filas        []filaCeseB81Prueba
		consultas    int
		sinCandidato bool
		reutilizada  bool
		fallo        bool
	}{
		{"sin candidato nuevo", []filaCeseB81Prueba{{err: claveAjena}, {err: claveAjena}, {}}, 3, true, false, false},
		{"replay conservado", []filaCeseB81Prueba{{err: claveAjena}, {err: claveAjena}, {reutilizada: true}}, 3, true, true, false},
		{"cese ajeno", []filaCeseB81Prueba{{err: claveAjena}, {}}, 2, false, false, false},
		{"error inicial detiene", []filaCeseB81Prueba{{err: errors.New("indisponible")}}, 1, false, false, true},
		{"bolsa constituida sigue pendiente", []filaCeseB81Prueba{{err: claveAjena}, {err: claveAjena}, {err: claveAjena}}, 3, false, false, true},
	}
	funciones := []string{"registrar_restriccion_cese_bolsa_v1", "confirmar_cese_ajeno_bolsa_v1", "confirmar_cese_sin_candidato_bolsa_v1"}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			llamadas := 0
			consulta := func(_ context.Context, sql string, args ...any) pgx.Row {
				if llamadas >= len(caso.filas) || !strings.Contains(sql, funciones[llamadas]) {
					t.Fatalf("consulta inesperada %d: %s", llamadas, sql)
				}
				if len(args) != 3 || args[0] != evento.OrigenRef || args[1] != evento.HuellaSHA256 || args[2] != evento.OrigenPosicion {
					t.Fatalf("triple de origen alterado: %v", args)
				}
				fila := caso.filas[llamadas]
				llamadas++
				return fila
			}
			reutilizada, sinCandidato, err := registrarCeseCTBolsa(context.Background(), consulta, evento)
			if llamadas != caso.consultas || (err != nil) != caso.fallo || reutilizada != caso.reutilizada || sinCandidato != caso.sinCandidato {
				t.Fatalf("consultas=%d, reutilizada=%t, sin candidato=%t, error=%v", llamadas, reutilizada, sinCandidato, err)
			}
		})
	}
}
