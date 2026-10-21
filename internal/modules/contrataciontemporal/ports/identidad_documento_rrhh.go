package ports

import "context"

// FuenteIdentidadDocumentoRRHH aporta solo la identidad acreditada por el
// módulo propietario para la aceptación conservada en este expediente.
// No admite un selector ni una identidad proporcionados por el navegador.
type FuenteIdentidadDocumentoRRHH func(context.Context, string) (*IdentidadDocumentoRRHH, error)

type IdentidadDocumentoRRHH struct {
	Nombre               string
	Apellidos            string
	DocumentoEnmascarado string
	VersionFuente        uint64
}
