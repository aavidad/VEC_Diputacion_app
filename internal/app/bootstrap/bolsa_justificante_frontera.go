package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func auditarRechazosJustificantePropio(h http.Handler, a vecports.RegistradorAuditoriaFronteraRutaExacta) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		salida := &respuestaAuditadaJustificanteRRHH{ResponseWriter: w, peticion: r,
			ruta: bolsahttp.RutaJustificantesMiBolsa, alRechazar: func(estado int) error {
				if estado != http.StatusUnauthorized && estado != http.StatusForbidden {
					return nil
				}
				if a == nil {
					return vecports.ErrIntentoAuditoriaNoDisponible
				}
				ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
				defer cancelar()
				corr, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
				if err != nil {
					return err
				}
				ref, err := corr.ValorCanonico()
				if err != nil {
					return err
				}
				actor := ""
				motivo := vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
				c, ok := r.Context().Value(claveOrdenMiBolsaPortalExterno{}).(contextoMiBolsaPortalExterno)
				if ok && c.err == nil && c.orden.ResultadoContexto.Validar() == nil && c.orden.Vinculo.ValidarPara(c.orden.ResultadoContexto) == nil {
					actor = c.orden.ResultadoContexto.Contexto.PersonaRef
					motivo = vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
				}
				o := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_" + strings.TrimPrefix(ref, "correlacion_"),
					Motivo: motivo, Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaBolsaCandidato,
					Ruta: bolsahttp.RutaJustificantesMiBolsa, ActorRef: actor}
				if err := o.Validar(); err != nil {
					return err
				}
				return a.RegistrarAuditoriaFronteraRutaExacta(ctx, o)
			}}
		h.ServeHTTP(salida, r)
	})
}
