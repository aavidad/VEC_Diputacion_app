package ports

import (
	"context"
	"errors"
	"time"
)

var ErrLecturaConcesionHistoricaV3 = errors.New("vec: concesion historica no disponible")

// LectorConcesionHistoricaAutorizacionLigadaV3 es una autoridad de composición
// propietaria de lectura. Relee la concesión ORIGINAL exacta de esta candidata
// (decisión/canon/contexto/motivo/ventana), nunca latest ni una concesión nueva.
// Devuelve su fecha original de registro sólo si todas las ligaduras coinciden.
// No INSERT, CAS, renovación, emisión, consumo ni efecto de negocio. Un doble
// deshonesto puede mentir sobre I/O: el tipo resultante no prueba persistencia.
type LectorConcesionHistoricaAutorizacionLigadaV3 interface {
	LeerConcesionHistoricaAutorizacionLigadaV3(context.Context, OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error)
}

// RecuperarConcesionHistoricaAutorizacionLigadaV3 reconstruye por el propietario
// una confirmación histórica después de una única lectura nominal. No hace
// vigente la autorización: DentroDeVentanaEn(ahora) y el consumo fresco siguen
// siendo obligatorios para todo efecto actual. No admite DTO→confirmación.
func RecuperarConcesionHistoricaAutorizacionLigadaV3(ctx context.Context, lector LectorConcesionHistoricaAutorizacionLigadaV3, orden OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	cero := ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	if ctx == nil {
		return cero, ErrLecturaConcesionHistoricaV3
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if dependenciaRegistroAutorizacionLigadaV3Nula(lector) || !ordenConcesionAutorizacionLigadaV3Valida(orden) {
		return cero, ErrLecturaConcesionHistoricaV3
	}
	fecha, err := lector.LeerConcesionHistoricaAutorizacionLigadaV3(ctx, orden)
	if ce := ctx.Err(); ce != nil {
		return cero, ce
	}
	if err != nil || !ordenConcesionAutorizacionLigadaV3Valida(orden) {
		return cero, ErrLecturaConcesionHistoricaV3
	}
	c, err := nuevaConfirmacionRegistroConcesionAutorizacionLigadaV3(orden, fecha)
	if err != nil || c.ValidarPara(orden) != nil {
		return cero, ErrLecturaConcesionHistoricaV3
	}
	if ce := ctx.Err(); ce != nil {
		return cero, ce
	}
	return c, nil
}
