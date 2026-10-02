package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarFirmasR5     = "contratacion_temporal.documento.firmas_r5.consultar"
	AudienciaConsultaFirmasR5V3 = "vec_contratacion_temporal.firmas_r5.consultar.v1"
	TipoRecursoConsultaFirmasR5 = "expediente_contratacion_temporal"
)

// La versión y el documento forman parte de la decisión de lectura. CT170
// devuelve solo la proyección necesaria para calcular el siguiente paso.
type MaterialConsultaFirmasR5 struct {
	OrganizacionRef               string
	ExpedienteRef                 string
	VersionExpediente             uint64
	Documento                     string
	FirmantePrincipalCandidatoRef string
	ClaveIdempotencia             string
	PasoOrden                     int
	CatalogoHuella                string
}

func (m MaterialConsultaFirmasR5) Canonico() ([]byte, error) {
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) || !domain.ReferenciaOpacaValida(m.ExpedienteRef) ||
		m.VersionExpediente == 0 || m.VersionExpediente > 9007199254740991 ||
		!domain.ClaveDocumentoFirmaValida(m.Documento) ||
		!domain.ReferenciaOpacaValida(m.FirmantePrincipalCandidatoRef) ||
		!strings.HasPrefix(m.FirmantePrincipalCandidatoRef, "per_") ||
		!ClaveIdempotenciaFirmaValida(m.ClaveIdempotencia) ||
		m.PasoOrden < 1 || m.PasoOrden > domain.MaximoPasosCircuitoFirma ||
		!domain.HuellaSHA256FirmaValida(m.CatalogoHuella) {
		return nil, ErrSolicitudFirmaDocumentoInvalida
	}
	return json.Marshal(m)
}

func (m MaterialConsultaFirmasR5) HuellaSHA256() (string, error) {
	c, err := m.Canonico()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(c)
	return hex.EncodeToString(h[:]), nil
}

type CapacidadConsultaFirmasR5 struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialConsultaFirmasR5(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadConsultaFirmasR5 {
	return CapacidadConsultaFirmasR5{material: m}
}

func (c CapacidadConsultaFirmasR5) ExportarMaterialParaConsumidor() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

type AutorizadorConsultaFirmasR5 interface {
	AutorizarConsultaFirmasR5(context.Context, MaterialConsultaFirmasR5) (CapacidadConsultaFirmasR5, error)
}

// La cabeza abarca toda la historia de organización/expediente hasta la
// versión autorizada, aun cuando Firmas proyecte solo un documento. Para una
// operación nueva CT170 compara la cabeza actual bajo bloqueo; un replay usa
// la cabeza original guardada en su fila.
type LecturaFirmasR5 struct {
	Firmas                       []FirmaRegistrada
	HistoriaRevision             uint64
	HistoriaHuella               string
	CoincideFirmanteEnOtroPaso   bool
	HistoriaSeparacionAcreditada bool
}
