package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

// Se invoca sólo al arrancar, antes de componer la sesión lectora. Comparte
// derivador y autoridad de cuentas con RRHH, pero prepara la cuenta de
// Intervención. No registra autenticaciones ni aserciones o sesiones.
func prepararCuentaNominalFirmasIntervencionDesarrollo(ctx context.Context, gobierno *pgxpool.Pool,
	canal *soporteFiscalizacionContratacionTemporalDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo) error {
	if gobierno == nil || derivador == nil || !derivador.valido() {
		return errorCuentaNominalDesarrollo(ctx)
	}
	return prepararCuentaNominalFirmasIntervencionConTransaccion(ctx, gobierno, canal, &seudonimizadorSesionDesarrollo{derivador: derivador})
}

func prepararCuentaNominalFirmasIntervencionConTransaccion(ctx context.Context, gobierno transaccionesCuentaNominalDesarrollo,
	canal *soporteFiscalizacionContratacionTemporalDesarrollo, seudonimizador postgresidentidad.SeudonimizadorAlta) error {
	if contextoInterfazNulo(ctx) || ctx.Err() != nil || canal == nil || dependenciaEsNulaContratacionTemporalDesarrollo(gobierno) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(seudonimizador) {
		return errorCuentaNominalDesarrollo(ctx)
	}
	principal := clonarPrincipalDesarrollo(canal.principalOriginal)
	if !principalIntervencionContratacionTemporalDesarrolloValido(principal) || principal.ID != canal.principalID ||
		principal.Attributes["certificate_sha256"] != canal.certificadoSHA256 || canal.contexto.Vinculo.ValidarPara(canal.contexto.Resultado) != nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	vinculo, err := canal.contexto.Vinculo.Datos()
	if err != nil {
		return errorCuentaNominalDesarrollo(ctx)
	}
	// El validador de cuentas exige el contexto original, con discriminador
	// default. El perfil lector propio no se usa para provisionar la cuenta.
	soporte := &soporteAltaContratacionTemporalDesarrollo{sello: canal.sello, principalID: principal.ID,
		certificadoSHA256: canal.certificadoSHA256, contexto: canal.contexto}
	seudonimos, err := seudonimizador.SeudonimizarAlta(ctx, postgresidentidad.IdentificadoresAlta{
		EspacioIdentidad: espacioIdentidadSesionDesarrollo, AsercionID: "preparacion-cuenta", SesionID: "preparacion-alias",
		CuentaID: "desarrollo:" + vinculo.CuentaRef, SujetoID: principal.ID,
	})
	if err != nil {
		return errorCuentaNominalDesarrollo(ctx)
	}
	return prepararCuentaNominalConsultasDesarrolloConTransaccion(ctx, gobierno, soporte, seudonimos)
}
