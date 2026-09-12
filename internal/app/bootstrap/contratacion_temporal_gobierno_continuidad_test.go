package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	altapersonal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	lecturapersonal "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

// El doble comprueba el contrato de selección del gobierno; no suplanta
// PostgreSQL ni acredita la transacción o la instalación de AD3-30/31.
type txGobiernoContinuidadPrueba struct {
	pgx.Tx
	audienciaActual string
}

type filaGobiernoContinuidadPrueba func(...any) error

func (f filaGobiernoContinuidadPrueba) Scan(destinos ...any) error { return f(destinos...) }

func (tx *txGobiernoContinuidadPrueba) QueryRow(
	_ context.Context, sql string, args ...any,
) pgx.Row {
	return filaGobiernoContinuidadPrueba(func(destinos ...any) error {
		switch {
		case strings.Contains(sql, "count(*) FROM vec_autorizacion_atestada_v3.puntero_clave_emision"):
			*destinos[0].(*int64) = 1
			*destinos[1].(*int64) = 1
			return nil
		case strings.Contains(sql, "c.audiencia_consumo IN"):
			if len(args) != 10 {
				return errors.New("numero de audiencias de gobierno inesperado")
			}
			admitida := false
			for _, indice := range []int{0, 2, 3, 4, 5, 6, 7, 8, 9} {
				if args[indice] == tx.audienciaActual {
					admitida = true
				}
			}
			*destinos[0].(*bool) = admitida
			return nil
		default:
			return errors.New("consulta de gobierno fuera del contrato")
		}
	})
}

func TestGobiernoPostgreSQLContinuidadNominalAD330YAD331(t *testing.T) {
	audienciasPropias := []string{
		audienciaConsumoAltaContratacionTemporal,
		ports.AudienciaIntegracionLlamamientoDesarrollo,
		puertosct.AudienciaConsumoConsultaCuadroRRHHV3,
		puertosct.AudienciaConsumoConsultaDetalleRRHHV3,
		altapersonal.AudienciaAltaEjercicio,
		lecturapersonal.AudienciaV2,
		puertosct.AudienciaConfirmacionIncorporacionV2,
		postgrescontratacion.AudienciaAnotacionAdministrativaV1,
		postgrescontratacion.AudienciaCierreAdministrativoSinCese,
	}
	for _, audiencia := range audienciasPropias {
		t.Run(audiencia, func(t *testing.T) {
			propio, err := gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(
				context.Background(), &txGobiernoContinuidadPrueba{audienciaActual: audiencia},
			)
			if err != nil || !propio {
				t.Fatalf("audiencia propia rechazada: propio=%t err=%v", propio, err)
			}
			if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(audiencia) {
				t.Fatal("publicador no reconoce una audiencia propia")
			}
		})
	}

	const audienciaAjena = "vec_contratacion_temporal.ajena.v1"
	propio, err := gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(
		context.Background(), &txGobiernoContinuidadPrueba{audienciaActual: audienciaAjena},
	)
	if err != nil || propio {
		t.Fatalf("audiencia ajena aceptada: propio=%t err=%v", propio, err)
	}
	if audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(audienciaAjena) {
		t.Fatal("publicador acepta una audiencia ajena")
	}
}
