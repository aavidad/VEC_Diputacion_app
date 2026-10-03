package ports

import (
	"time"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const TipoSolicitudDocumentalRRHH = "documental_rrhh"

// SolicitudDocumentalPortal es una aportación declarada. Referencia y huella
// no acreditan custodia del archivo; RRHH verifica el documento expresamente.
type SolicitudDocumentalPortal struct {
	SolicitudRef, ReciboRef, ContenidoSHA256 string
	CandidatoRef, Bolsa                      string
	DocumentoRef, DocumentoSHA256            string
	FechaFinCausa, Clave                     string
	RegistradaEn                             time.Time
	Material                                 puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboSolicitudDocumentalPortal struct {
	Reutilizada                              bool
	SolicitudRef, ReciboRef, ContenidoSHA256 string
	RegistradaEn                             time.Time
	Version                                  int64
	Estado                                   string
}
