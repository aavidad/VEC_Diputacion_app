package bootstrap

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

//go:embed bolsa_documentos_motivos.json
var motivosLecturasDocumentosBolsa []byte

func nuevoAuditorDocumentosBolsa(resolver func(context.Context) (*soporteAltaContratacionTemporalDesarrollo, error),
	registrador vecports.RegistradorIntentosAuditoria, proceso string, frontera vecports.RegistradorAuditoriaFronteraRutaExacta,
) func(context.Context, string, string, error) error {
	var catalogo struct {
		ID       string            `json:"catalogo_id"`
		Version  int               `json:"version"`
		Entradas map[string]string `json:"entradas"`
	}
	if resolver == nil || registrador == nil || frontera == nil || json.Unmarshal(motivosLecturasDocumentosBolsa, &catalogo) != nil {
		return nil
	}
	huella := sha256.Sum256(motivosLecturasDocumentosBolsa)
	return func(ctx context.Context, bolsaRef, accion string, fallo error) error {
		entrada, ok := catalogo.Entradas[accion]
		if !ok || ctx == nil {
			return vecports.ErrIntentoAuditoriaNoDisponible
		}
		ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelar()
		operativo, err := contextoNominalAuditadoDocumentosBolsa(ctx, bolsaRef, accion)
		if err != nil {
			if fallo == nil {
				return err
			}
			return registrarFronteraDocumentosBolsa(ctxAuditoria, ctx, resolver, frontera, fallo)
		}
		correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctxAuditoria, seguridadvec.GeneradorReferenciasCriptograficas{})
		if err != nil {
			return err
		}
		valor, err := correlacion.ValorCanonico()
		if err != nil {
			return err
		}
		ref, err := vecports.NuevaReferenciaIntentoAuditoria()
		if err != nil {
			return err
		}
		resultado := vecdomain.ResultadoIntentoAuditoriaExito
		if fallo != nil {
			resultado = vecdomain.ResultadoIntentoAuditoriaError
			if errors.Is(fallo, vecdomain.ErrAutorizacionDenegada) || errors.Is(fallo, vecdomain.ErrPermissionDenied) {
				resultado = vecdomain.ResultadoIntentoAuditoriaDenegado
			}
		}
		orden, err := vecports.NuevaOrdenIntentoAuditoria(ref, operativo.Resultado, operativo.Vinculo, vecdomain.DatosIntentoAuditoria{
			Accion: accion, ModuloID: "bolsa", RecursoRef: bolsaRef, FinalidadRef: "gestion_bolsa_rrhh",
			Resultado: resultado, Motivo: vecdomain.ReferenciaEntradaCatalogo{CatalogoID: catalogo.ID,
				CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: hex.EncodeToString(huella[:]), EntradaClave: entrada},
			Proceso: proceso, Canal: string(vecdomain.SuperficieAutenticacionInternaCorporativaV1), CorrelacionRef: valor,
		})
		if err != nil {
			return err
		}
		acuse, err := registrador.AppendIntentoAuditoria(ctxAuditoria, orden)
		if err != nil {
			return err
		}
		return acuse.ValidarPara(orden)
	}
}

// Sin contexto operativo válido se registra el rechazo de frontera. El actor
// solo procede del sello mTLS y del soporte nominal configurado; no se crea
// una sesión ni una identidad para escribir el apunte.
func registrarFronteraDocumentosBolsa(ctxAuditoria, ctxPeticion context.Context,
	resolver func(context.Context) (*soporteAltaContratacionTemporalDesarrollo, error),
	frontera vecports.RegistradorAuditoriaFronteraRutaExacta, fallo error,
) error {
	capacidad, _ := ctxPeticion.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctxAuditoria, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return err
	}
	valor, err := correlacion.ValorCanonico()
	if err != nil {
		return err
	}
	motivo := vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
	actor := ""
	if soporte, err := resolver(ctxAuditoria); err == nil && soporteDocumentoCoincideConCapacidad(soporte, capacidad) {
		actor = capacidad.principal.ID
		motivo = vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	}
	if errors.Is(fallo, context.Canceled) || errors.Is(fallo, context.DeadlineExceeded) || errors.Is(fallo, ErrComposicionDesarrolloIncompleta) || errors.Is(fallo, bolsaapp.ErrConsultaDocumentosBolsaNoDisponible) {
		motivo = vecports.MotivoAuditoriaFronteraRutaExactaServicioNoDisponible
	}
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_" + strings.TrimPrefix(valor, "correlacion_"), Motivo: motivo,
		Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaDocumentosBolsaRRHH, Ruta: capacidad.ruta, ActorRef: actor}
	if orden.Validar() != nil {
		return vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida
	}
	return frontera.RegistrarAuditoriaFronteraRutaExacta(ctxAuditoria, orden)
}
