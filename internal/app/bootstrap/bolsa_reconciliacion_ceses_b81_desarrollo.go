package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	var desdePosicion any
	var desdeRef any
	var ultimo cesePendienteB81
	hayUltimo := false
	var primerError error
	fallos := 0
	errorAplicaciones := func() error {
		if fallos == 0 {
			return nil
		}
		return fmt.Errorf("%w: clave=aplicaciones_B45_fallidas esperado=0 actual=%d", primerError, fallos)
	}
	for pagina := 0; pagina < maximoPaginasEntregaContratosCT; pagina++ {
		var bruto []byte
		if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1($1,$2,$3)`,
			r.lote, desdePosicion, desdeRef).Scan(&bruto); err != nil {
			return resultado, errors.Join(falloRelevoCeseBolsaDesarrollo(err), errorAplicaciones())
		}
		var pendientes []cesePendienteB81
		if len(bruto) == 0 || json.Unmarshal(bruto, &pendientes) != nil || pendientes == nil || len(pendientes) > r.lote {
			return resultado, errors.Join(fmt.Errorf("%w: clave=pagina_ceses_B81 esperado=json_array_hasta_%d actual=invalida",
				puertosbolsa.ErrContratosParticipacionNoDisponible, r.lote), errorAplicaciones())
		}
		// Validar la página completa antes de invocar B45: el cursor debe
		// avanzar estrictamente también cuando un elemento falle.
		for _, pendiente := range pendientes {
			if pendiente.OrigenRef == "" || pendiente.HuellaSHA256 == "" || pendiente.OrigenPosicion < 0 {
				return resultado, errors.Join(fmt.Errorf("%w: clave=marcador_cese_B81 esperado=ref_huella_posicion_validos actual=invalido",
					puertosbolsa.ErrContratosParticipacionNoDisponible), errorAplicaciones())
			}
			if _, repetido := vistos[pendiente.OrigenRef]; repetido {
				return resultado, errors.Join(fmt.Errorf("%w: clave=origen_ref_B81 esperado=unico actual=repetido",
					puertosbolsa.ErrContratosParticipacionNoDisponible), errorAplicaciones())
			}
			if hayUltimo && (pendiente.OrigenPosicion < ultimo.OrigenPosicion ||
				pendiente.OrigenPosicion == ultimo.OrigenPosicion && pendiente.OrigenRef <= ultimo.OrigenRef) {
				return resultado, errors.Join(fmt.Errorf("%w: clave=orden_ceses_B81 esperado=posterior_a_%d/%s actual=%d/%s",
					puertosbolsa.ErrContratosParticipacionNoDisponible,
					ultimo.OrigenPosicion, ultimo.OrigenRef, pendiente.OrigenPosicion, pendiente.OrigenRef), errorAplicaciones())
			}
			vistos[pendiente.OrigenRef] = struct{}{}
			ultimo = pendiente
			hayUltimo = true
		}
		for _, pendiente := range pendientes {
			var reutilizada bool
			var recibo, candidato string
			var disponible time.Time
			var politica int64
			if err := r.pool.QueryRow(ctx, `SELECT reutilizada,recibo_ref,candidato_ref,disponible_desde,politica_version
				FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1($1,$2,$3)`,
				pendiente.OrigenRef, pendiente.HuellaSHA256, pendiente.OrigenPosicion).
				Scan(&reutilizada, &recibo, &candidato, &disponible, &politica); err != nil {
				fallo := falloRelevoCeseBolsaDesarrollo(err)
				if primerError == nil {
					primerError = fallo
				}
				fallos++
				continue
			}
			if recibo == "" || candidato == "" || disponible.IsZero() || politica < 1 {
				if primerError == nil {
					primerError = puertosbolsa.ErrContratosParticipacionNoDisponible
				}
				fallos++
				continue
			}
			if reutilizada {
				resultado.reentregas++
			} else {
				resultado.nuevos++
			}
		}
		if len(pendientes) < r.lote {
			return resultado, errorAplicaciones()
		}
		desdePosicion, desdeRef = ultimo.OrigenPosicion, ultimo.OrigenRef
	}
	return resultado, errors.Join(
		fmt.Errorf("%w: clave=paginas_reconciliacion_B81 esperado<%d actual=%d",
			puertosbolsa.ErrContratosParticipacionNoDisponible,
			maximoPaginasEntregaContratosCT, maximoPaginasEntregaContratosCT),
		errorAplicaciones(),
	)
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
