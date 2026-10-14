package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

// OriginalAcreditado procede de Documentos tras comprobar lectura nominal del
// actor de esta petición. Referencias o metadatos enviados por el navegador no
// acreditan una subida ni el estado del original.
type OriginalAcreditado struct {
	Documento           domain.DocumentoJustificacion
	ModuloID            string
	ExpedienteRef       string
	TipoRef             string
	EmpleadoPropietario string
	PersonaAportante    string
	EstadoAdmision      string
}

// FuenteOriginalJustificante la implementará el adaptador del puerto de subida
// común D20. Debe leer el original con permiso nominal y devolver la versión
// y huella de los bytes admitidos, nunca una declaración del consumidor.
type FuenteOriginalJustificante interface {
	AcreditarOriginal(context.Context, OrdenJustificacion, domain.SolicitudJustificable, domain.PoliticaJustificacion, string) (OriginalAcreditado, error)
}

type PeticionAporteOriginal struct {
	SolicitudRef, ClaveOperacion, DocumentoID, SHA256 string
	VersionEsperada                                   int64
	DocumentoVersion                                  uint64
}

type PeticionRevisionOriginal struct {
	SolicitudRef, ClaveOperacion string
	VersionEsperada              int64
	Decision                     domain.EstadoJustificacion
	MotivoRef                    string
}
