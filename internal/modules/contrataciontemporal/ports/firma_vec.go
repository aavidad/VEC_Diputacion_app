package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrRegistroFirmaVecNoDisponible = errors.New("contratacion temporal: registro de firma con certificado VEC no disponible")

// La vía VEC exige original custodiado y certificado del dictamen igual al
// certificado del canal sellado. AD3-85/CT118 quedan como ejercicio legado.
const (
	ViaFirmaCertificadoVEC  = "certificado_vec"
	AccionRegistrarFirmaVec = "contratacion_temporal.documento.firma_vec.registrar"
	AudienciaFirmaVecV3     = "vec_contratacion_temporal.firma_vec.v1"
	TipoRecursoFirmaVec     = "firma_vec_documento_contratacion_temporal"
	PrefijoRecursoFirmaVec  = "operacion-firma-vec-ct:"
)

// Comparte los hechos técnicos y de competencia de MaterialFirmaExterna.
// ReferenciaPortafirmasDeclarada y FechaPortafirmasDeclarada deben estar
// vacías y no se serializan en el canon de esta vía.
type MaterialFirmaVec MaterialFirmaExterna

func (m MaterialFirmaVec) Validar() error {
	if m.Via != ViaFirmaCertificadoVEC ||
		m.ReferenciaPortafirmasDeclarada != "" || m.FechaPortafirmasDeclarada != "" {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	return MaterialFirmaExterna(m).validarComun()
}

func (m MaterialFirmaVec) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrSolicitudFirmaDocumentoInvalida
	}
	return canonicoFirmaVerificada(MaterialFirmaExterna(m), false)
}

func (m MaterialFirmaVec) HuellaSHA256() (string, error) {
	c, err := m.Canonico()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(c)
	return hex.EncodeToString(h[:]), nil
}

func (m MaterialFirmaVec) RecursoRef() string {
	return PrefijoRecursoFirmaVec + m.ClaveIdempotencia
}

type CapacidadFirmaVec struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialFirmaVec(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadFirmaVec {
	return CapacidadFirmaVec{material: m}
}

func (c CapacidadFirmaVec) ExportarMaterialParaConsumidor() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

// El autorizador debe comparar CertificadoHuella con certificate_sha256 del
// canal sellado y exigir que su principal sea FirmantePrincipalRef antes de
// emitir la decisión V3. SQL vuelve a exigir igualdad de principal al consumo.
type AutorizadorFirmaVec interface {
	AutorizarFirmaVec(context.Context, MaterialFirmaVec) (CapacidadFirmaVec, error)
}

type RegistroFirmasVec interface {
	RegistrarFirmaVec(context.Context, MaterialFirmaVec, CapacidadFirmaVec) (ReciboFirmaDocumento, error)
	ConsultarFirmasAutorizadas(context.Context, MaterialConsultaFirmasR5, CapacidadConsultaFirmasR5) (LecturaFirmasR5, error)
}
