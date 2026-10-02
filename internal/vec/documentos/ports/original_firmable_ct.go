package ports

import (
	"context"

	canonicodoc "vec-diputacion-granada/internal/vec/canonico/documental"
	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionReservarOriginalFirmable  = canonicodoc.AccionReservaOriginalFirmable
	AccionConfirmarOriginalFirmable = canonicodoc.AccionConfirmacionOriginalFirmable
	FinalidadOriginalFirmable       = canonicodoc.FinalidadOriginalFirmable
)

type ReservaTiposOriginalCT interface {
	CustodiaOriginalCTReservada(string) bool
}

// OrdenCustodiarOriginalFirmable procede de una fuente documental autenticada.
// El servicio calcula la huella y obtiene autorizaciones V3 de efecto.
type OrdenCustodiarOriginalFirmable struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME                                                    string
	Contenido                                               []byte
	SolicitudPolitica                                       vecports.SolicitudPoliticaConservacionDocumental
}

type ReservaOriginalFirmable struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME, HuellaSHA256                                      string
	Tamano                                                  int64
	Politica                                                vecports.ResultadoPoliticaConservacionDocumental
	Autorizacion                                            AutorizacionV3
}

type IntentoOriginalFirmable = canonicodoc.IntentoOriginalFirmable
type ObjetoOriginalFirmable = canonicodoc.ObjetoOriginalFirmable

type ConfirmacionOriginalFirmable struct {
	Intento      IntentoOriginalFirmable
	Objeto       ObjetoOriginalFirmable
	Autorizacion AutorizacionV3
}

// AutorizarOriginalFirmable obtiene decisiones nominales para cada preimagen.
// El contexto de escritura requiere otra concesión PDP propia del almacén.
type AutorizarOriginalFirmable interface {
	AutorizarReservaOriginal(context.Context, []byte, string, string) (AutorizacionV3, error)
	AutorizarConfirmacionOriginal(context.Context, []byte, string, string) (AutorizacionV3, error)
	ContextoEscrituraOriginal(context.Context, ReservaOriginalFirmable, IntentoOriginalFirmable) (vecports.ContextoOperacionAlmacen, error)
}

type RepositorioOriginalFirmable interface {
	ReservarOriginalFirmable(context.Context, ReservaOriginalFirmable) (IntentoOriginalFirmable, error)
	ConfirmarOriginalFirmable(context.Context, ConfirmacionOriginalFirmable) (domain.Documento, error)
}
