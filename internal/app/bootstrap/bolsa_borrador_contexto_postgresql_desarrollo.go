package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// La operación es propia de esta composición y constante: ningún valor del
// transporte interviene en la clave que registra el contexto de Bolsa.
func operacionContextoBorradorBolsaDesarrollo() string {
	return referenciaAltaContratacionTemporalDesarrollo(
		"oca_", "bolsa-bback:registro-contexto:v1",
	)
}

// publicarContextoPostgreSQLBorradorBolsaDesarrollo registra el perfil y el
// vínculo nominales ya construidos para Bolsa. No publica una autorización V3.
func publicarContextoPostgreSQLBorradorBolsaDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteSesionBorradorBolsaDesarrollo,
) error {
	if soporte == nil || soporte.soporteCanal == nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	contexto := soporte.soporteCanal.contexto
	if contexto.Resultado.Validar() != nil ||
		contexto.Vinculo.ValidarPara(contexto.Resultado) != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return publicarResultadoContextoPostgreSQLDesarrollo(
		ctx, pool, contexto.Resultado, operacionContextoBorradorBolsaDesarrollo(),
	)
}
