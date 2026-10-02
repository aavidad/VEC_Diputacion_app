package ports

import (
	"context"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Contrato propuesto de consulta, sin provisión, emisor ni montaje. La raíz
// resuelve actor/objetivo/organismo y revalida la concesión propia de Carrera.
// B conserva otra autorización y su transacción de lectura/auditoría.
const AccionConsultaPreparacionCarrera = "carrera.preparacion.consultar"
const FinalidadConsultaPreparacionCarrera = "preparar_expediente_carrera"

// La confirmación común liga el registro de concesión a la orden nominal;
// no es un booleano ni sustituye la revalidación transaccional del lector B.
// Ninguna referencia o perfil se toma de parámetros libres del navegador.
type FuenteConsultaAutorizadaPreparacionCarrera interface {
	ResolverConsultaPreparacionCarrera(context.Context) (personalports.ConsultaAntecedentesCarreraV1, vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error)
}
