package ports

import (
	"context"
	"errors"

	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConfiguracionCorreoNoDisponible = errors.New("administracion: configuracion de correo no disponible")
	ErrConfiguracionCorreoConflicto    = errors.New("administracion: conflicto de configuracion de correo")
)

// VerificadorAccesoConfiguracionCorreo concentra la frontera compuesta:
// intranet verificada por el listener, DNIe/certificado y permiso explícito.
// Ninguna cabecera del cliente constituye prueba suficiente.
type VerificadorAccesoConfiguracionCorreo interface {
	VerificarAccesoConfiguracionCorreo(context.Context, vecdomain.Principal) error
}

// SobreSecretoConfiguracionCorreo sólo contiene el resultado cifrado de la
// preparación. El secreto claro no cruza esta frontera.
type SobreSecretoConfiguracionCorreo struct {
	Version  uint64
	ClaveRef string
	Nonce    []byte
	Cifrado  []byte
}

// PreparacionConfiguracionCorreo fija la preimagen canónica de una mutación:
// CAS, operación de secreto, sobre cifrado, AAD y payload de negocio. La
// autorización V3 se emite sobre estos mismos bytes.
type PreparacionConfiguracionCorreo struct {
	Entrada         admindomain.ActualizacionConfiguracionCorreo
	SobreNuevo      SobreSecretoConfiguracionCorreo
	Sustituir       bool
	HuellaAADSHA256 string
	PayloadNegocio  []byte
}

// OrdenConfiguracionCorreoAutorizada es lo único que puede recibir el
// registro durable después de Preparar→Autorizar. No admite secreto claro ni
// una autorización desligada de la preparación.
type OrdenConfiguracionCorreoAutorizada struct {
	Preparacion PreparacionConfiguracionCorreo
	Material    vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type PreparadorConfiguracionCorreo interface {
	PrepararConfiguracionCorreo(context.Context, admindomain.ActualizacionConfiguracionCorreo, vecdomain.AuditEntry) (PreparacionConfiguracionCorreo, error)
}

type AutorizadorConfiguracionCorreo interface {
	AutorizarConfiguracionCorreo(context.Context, PreparacionConfiguracionCorreo, vecdomain.AuditEntry) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// RegistroConfiguracionCorreo debe guardar configuración, secreto protegido,
// material V3 y auditoría en una misma transacción. No permite un guardado
// parcial ni recibe texto claro.
type RegistroConfiguracionCorreo interface {
	LeerConfiguracionCorreo(context.Context) (admindomain.VistaConfiguracionCorreo, error)
	GuardarConfiguracionCorreo(context.Context, OrdenConfiguracionCorreoAutorizada) (admindomain.VistaConfiguracionCorreo, error)
}
