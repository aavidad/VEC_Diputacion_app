package postgres

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var ErrPreparacionIncorporacionNoConfiable = errors.New("contratacion temporal: preparacion de incorporacion no confiable")

// ReferenciasPreparacionIncorporacion son resueltas por el futuro llamador
// durable. No proceden del material ni conceden permiso.
type ReferenciasPreparacionIncorporacion struct {
	ActuacionRef string
	ReciboRef    string
}

// EntradaPreparacionIncorporacion reúne exclusivamente material ya preparado,
// definición de una fuente confiable y el snapshot leído por el propietario.
// La terna de definición, la raíz y los vínculos se cotejan antes de aplicar.
type EntradaPreparacionIncorporacion struct {
	Definicion   domain.DefinicionSeguimiento
	Expectativa  ExpectativaSeguimientoPersistido
	Snapshot     SnapshotSeguimientoPersistido
	Referencias  ReferenciasPreparacionIncorporacion
	RegistradaEn time.Time
}

// ResultadoPreparacionIncorporacion no acredita commit, permiso, consumo V3 ni
// procedencia Personal. El futuro almacén debe revalidar todo en su transacción.
type ResultadoPreparacionIncorporacion struct {
	SnapshotPosterior SnapshotSeguimientoPersistido
	Evento            domain.ActuacionSeguimiento
}

// PrepararIncorporacionDesdeSnapshot restaura un seguimiento exacto y aplica
// confirmar_incorporacion usando solo campos ya comprometidos por el material.
// No convierte la orden V2 a la API histórica v1.
func PrepararIncorporacionDesdeSnapshot(
	ctx context.Context,
	material ports.MaterialConfirmacionIncorporacionV2,
	entrada EntradaPreparacionIncorporacion,
) (ResultadoPreparacionIncorporacion, error) {
	var cero ResultadoPreparacionIncorporacion
	if ctx == nil {
		return cero, ErrPreparacionIncorporacionNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	datos, err := material.Datos()
	if err != nil || !domain.InstanteUTCCanonico(entrada.RegistradaEn) ||
		!domain.ReferenciaOpacaValida(entrada.Referencias.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(entrada.Referencias.ReciboRef) ||
		entrada.Expectativa.OrganizacionRef != datos.Preparacion.OrganizacionRef ||
		entrada.Expectativa.ExpedienteRef != datos.Confirmacion.SolicitudPersonal.ExpedienteRef ||
		entrada.Expectativa.RelacionRef != datos.Confirmacion.ResultadoPersonal.RelacionRef ||
		entrada.Expectativa.Version != datos.Confirmacion.VersionSeguimientoEsperada ||
		!entrada.Definicion.Referencia().Coincide(entrada.Expectativa.Definicion) {
		return cero, ErrPreparacionIncorporacionNoConfiable
	}
	seguimiento, err := RestaurarSeguimientoPersistido(entrada.Definicion, entrada.Expectativa, entrada.Snapshot)
	if err != nil {
		return cero, ErrPreparacionIncorporacionNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	periodo := datos.Confirmacion.PeriodoIncorporacion
	siguiente, err := seguimiento.Aplicar(entrada.Definicion, datos.Confirmacion.VersionSeguimientoEsperada, domain.DatosTransicionSeguimiento{
		ActuacionRef:    entrada.Referencias.ActuacionRef,
		TransicionClave: ports.TransicionConfirmarIncorporacion,
		MotivoClave:     datos.Confirmacion.MotivoClave,
		ActorRef:        datos.Preparacion.ActorRef,
		UnidadRef:       datos.Preparacion.UnidadRef,
		EfectivoEn:      periodo.Desde,
		RegistradaEn:    entrada.RegistradaEn,
		Documentos:      append([]domain.DocumentoSeguimiento(nil), datos.Confirmacion.Documentos...),
		Periodo:         &periodo,
		ReciboRef:       entrada.Referencias.ReciboRef,
		CorrelacionRef:  datos.Preparacion.CorrelacionRef,
	})
	if err != nil {
		return cero, ErrPreparacionIncorporacionNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	snapshot, err := PrepararSnapshotSeguimientoPersistido(entrada.Definicion, siguiente)
	if err != nil {
		return cero, ErrPreparacionIncorporacionNoConfiable
	}
	actuaciones := siguiente.Actuaciones()
	if len(actuaciones) == 0 {
		return cero, ErrPreparacionIncorporacionNoConfiable
	}
	return ResultadoPreparacionIncorporacion{SnapshotPosterior: snapshot.clonar(), Evento: actuaciones[len(actuaciones)-1]}, nil
}
