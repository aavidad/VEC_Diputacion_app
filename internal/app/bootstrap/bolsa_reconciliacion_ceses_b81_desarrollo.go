package bootstrap

import (
	"context"
	"encoding/json"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// reconciliacionCesesB81 aplica B45 a proyecciones cuyo vínculo apareció
// después del cese. Su consulta no usa el cursor de publicación de CT.
type reconciliacionCesesB81 struct {
	pool consultorCesesCTBolsa
	lote int
}

type cesePendienteB81 struct {
	OrigenRef      string `json:"origen_ref"`
	HuellaSHA256   string `json:"huella_sha256"`
	OrigenPosicion int64  `json:"origen_posicion"`
}

func (r *reconciliacionCesesB81) entregar(ctx context.Context) (resultadoEntregaContratosCT, error) {
	var resultado resultadoEntregaContratosCT
	if r == nil || r.pool == nil || r.lote < 1 || r.lote > 100 || ctx == nil {
		return resultado, puertosct.ErrPublicacionContratosBolsaNoDisponible
	}
	vistos := make(map[string]struct{})
	for pagina := 0; pagina < maximoPaginasEntregaContratosCT; pagina++ {
		var bruto []byte
		if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1($1)`, r.lote).Scan(&bruto); err != nil {
			return resultado, falloRelevoCeseBolsaDesarrollo(err)
		}
		var pendientes []cesePendienteB81
		if len(bruto) == 0 || json.Unmarshal(bruto, &pendientes) != nil || pendientes == nil || len(pendientes) > r.lote {
			return resultado, puertosbolsa.ErrContratosParticipacionNoDisponible
		}
		for _, pendiente := range pendientes {
			if pendiente.OrigenRef == "" || pendiente.HuellaSHA256 == "" || pendiente.OrigenPosicion < 0 {
				return resultado, puertosbolsa.ErrContratosParticipacionNoDisponible
			}
			if _, repetido := vistos[pendiente.OrigenRef]; repetido {
				return resultado, puertosbolsa.ErrContratosParticipacionNoDisponible
			}
			vistos[pendiente.OrigenRef] = struct{}{}
			var reutilizada bool
			var recibo, candidato string
			var disponible time.Time
			var politica int64
			if err := r.pool.QueryRow(ctx, `SELECT reutilizada,recibo_ref,candidato_ref,disponible_desde,politica_version
				FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1($1,$2,$3)`,
				pendiente.OrigenRef, pendiente.HuellaSHA256, pendiente.OrigenPosicion).
				Scan(&reutilizada, &recibo, &candidato, &disponible, &politica); err != nil {
				return resultado, falloRelevoCeseBolsaDesarrollo(err)
			}
			if recibo == "" || candidato == "" || disponible.IsZero() || politica < 1 {
				return resultado, puertosbolsa.ErrContratosParticipacionNoDisponible
			}
			if reutilizada {
				resultado.reentregas++
			} else {
				resultado.nuevos++
			}
		}
		if len(pendientes) < r.lote {
			return resultado, nil
		}
	}
	return resultado, nil
}

// Ambas pasadas usan estado propio. Un error de publicación CT no oculta los
// vínculos nuevos, y un fallo B45 deja la proyección para el próximo intento.
func entregarCesesConReconciliacionB81(ctx context.Context, relevo *entregaCesesCTBolsa, pendientes *reconciliacionCesesB81) (resultadoEntregaContratosCT, error) {
	principal, errPrincipal := relevo.entregar(ctx)
	recuperados, errRecuperacion := pendientes.entregar(ctx)
	principal.nuevos += recuperados.nuevos
	principal.reentregas += recuperados.reentregas
	if errPrincipal != nil {
		return principal, errPrincipal
	}
	return principal, errRecuperacion
}
