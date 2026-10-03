package ports

import (
	"context"
	"time"
)

// CambioFirmaPDF conserva las categorías del dictamen, sin texto del proveedor.
type CambioFirmaPDF struct {
	Estado  string
	Detalle []string
}

type CambioPosteriorFirmaPDF struct {
	Estado          string
	Detalle         []string
	BytesNoFirmados *uint64
}

// FirmaPDFVerificada describe una firma y los bytes de su revisión. Orden es
// uno basado y no identifica el cargo ni concede competencia administrativa.
type FirmaPDFVerificada struct {
	Orden                           int
	ByteRange                       [4]uint64
	RevisionHuellaSHA256            string
	ContenidoFirmadoHuellaSHA256    string
	RevisionLongitud                uint64
	CubreDocumentoCompletoHastaAqui bool
	FirmanteRef                     string
	CertificadoHuellaSHA256         string
	IntegridadEstado                string
	CadenaEstado                    string
	CertificadoEstado               string
	RevocacionEstado                string
	SelloTiempoEstado               string
	TipoFirma                       string
	NivelDocMDP                     *int
	CambiosDesdeAnterior            CambioFirmaPDF
}

// VerificacionFirmasDocumento conserva la cadena completa; no contiene un
// firmante global que pueda sustituir la selección del paso del circuito.
type VerificacionFirmasDocumento struct {
	Estado               EstadoVerificacionFirma
	Motivo               MotivoVerificacionFirma
	Formato              string
	VinculoOriginal      bool
	HuellaOriginalSHA256 string
	HuellaFirmadoSHA256  string
	ComprobadoEn         time.Time
	Firmas               []FirmaPDFVerificada
	CambiosPosteriores   CambioPosteriorFirmaPDF
}

type VerificadorFirmasDocumento interface {
	VerificarFirmas(context.Context, SolicitudVerificacionFirma) (VerificacionFirmasDocumento, error)
}
