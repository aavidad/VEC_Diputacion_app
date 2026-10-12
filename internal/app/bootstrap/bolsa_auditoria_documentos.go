package bootstrap

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"

	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

//go:embed bolsa_documentos_motivos.json
var motivosLecturasDocumentosBolsa []byte

func nuevoAuditorDocumentosBolsa(resolver func(context.Context) (*soporteAltaContratacionTemporalDesarrollo, error),
	registrador vecports.RegistradorIntentosAuditoria, proceso string,
) func(context.Context, string, string, error) error {
	var catalogo struct {
		ID       string            `json:"catalogo_id"`
		Version  int               `json:"version"`
		Entradas map[string]string `json:"entradas"`
	}
	if resolver == nil || registrador == nil || json.Unmarshal(motivosLecturasDocumentosBolsa, &catalogo) != nil {
		return nil
	}
	huella := sha256.Sum256(motivosLecturasDocumentosBolsa)
	return func(ctx context.Context, bolsaRef, accion string, fallo error) error {
		entrada, ok := catalogo.Entradas[accion]
		if !ok || ctx == nil {
			return vecports.ErrIntentoAuditoriaNoDisponible
		}
		soporte, err := resolver(ctx)
		if err != nil {
			return err
		}
		operativo, err := soporte.contextoOperativoDesarrollo(ctx)
		if err != nil {
			return err
		}
		correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
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
		acuse, err := registrador.AppendIntentoAuditoria(context.WithoutCancel(ctx), orden)
		if err != nil {
			return err
		}
		return acuse.ValidarPara(orden)
	}
}
