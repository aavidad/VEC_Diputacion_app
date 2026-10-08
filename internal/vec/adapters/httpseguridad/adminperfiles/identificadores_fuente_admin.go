package adminperfiles

import (
	"context"
	identidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

// La fuente privada conserva las preimágenes originales y nunca las deriva
// de referencias, nombres, certificados ni digests del registro SQL.
type FuenteIdentificadoresADMIN interface {
	ResolverIdentificadoresADMIN(context.Context, ReferenciaFuenteIdentificadoresADMIN) (IdentificadoresFuenteADMIN, error)
}
type ReferenciaFuenteIdentificadoresADMIN struct {
	PersonaRef, CuentaRef, CuentaOrdinariaRef                  string
	CertificadoSHA256, CASHA256                                string
	EspacioIdentidad, EsquemaHMAC, DominioHMACRef, ClaveHMACID string
	ClaveHMACVersion                                           uint64
	SujetoHMAC, CuentaHMAC, CuentaOrdinariaHMAC                [32]byte
	FuenteRef, FuenteSHA256                                    string
}
type IdentificadoresFuenteADMIN struct {
	SujetoID, CuentaID, CuentaOrdinariaID         string
	EspacioIdentidad, DominioHMACRef, ClaveHMACID string
	ClaveHMACVersion                              uint64
	FuenteRef, FuenteSHA256                       string
}

func (IdentificadoresFuenteADMIN) MarshalJSON() ([]byte, error) {
	return []byte(`{"identificadores":"ocultos"}`), nil
}

// Conector original: el mismo contrato usado por RegistroSesionesPostgreSQL.
type SeudonimizadorFuenteADMIN interface {
	SeudonimizarAlta(context.Context, identidad.IdentificadoresAlta) (identidad.SeudonimosAlta, error)
}
