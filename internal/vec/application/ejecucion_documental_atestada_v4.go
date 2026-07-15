package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrEjecucionDocumentalAtestadaV4NoDisponible = errors.New(
	"vec: ejecucion documental atestada v4 no disponible",
)

// EjecutorDocumentalAtestadoV4 es el caso de uso neutral del nucleo. Solo
// conoce el puerto de ejecucion atestada y no conoce controladores, motores de
// datos, sockets, claves ni repositorios concretos.
type EjecutorDocumentalAtestadoV4 struct {
	conector ports.ConectorEjecucionDocumentalAtestadaV4
}

// NuevoEjecutorDocumentalAtestadoV4 recibe un puerto fijado en la raiz de
// composicion. Sustituir el motor exige otro conector homologado, no modificar
// ni recompilar este caso de uso.
func NuevoEjecutorDocumentalAtestadoV4(
	conector ports.ConectorEjecucionDocumentalAtestadaV4,
) (*EjecutorDocumentalAtestadoV4, error) {
	if dependenciaEjecucionDocumentalAtestadaV4Nula(conector) {
		return nil, ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	ejecutor := &EjecutorDocumentalAtestadoV4{conector: conector}
	if ejecutor.validar() != nil {
		return nil, ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return ejecutor, nil
}

type ResultadoEjecucionDocumentalAtestadaV4 struct {
	OrdenRef        string
	Estado          string
	AuditoriaRef    string
	EventoOutboxRef string
	RegistradaEn    time.Time
}

// Ejecutar es la unica operacion de la fachada. Recibe el vinculo estructural
// y el COSE del PDP, pero nunca un DTO persistible, un verificador, una raiz,
// un repositorio ni un instante elegido por el llamador.
func (e *EjecutorDocumentalAtestadoV4) Ejecutar(
	ctx context.Context,
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (ResultadoEjecucionDocumentalAtestadaV4, error) {
	if e.validar() != nil || ctx == nil {
		return ResultadoEjecucionDocumentalAtestadaV4{},
			ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ResultadoEjecucionDocumentalAtestadaV4{}, errors.Join(
			ErrEjecucionDocumentalAtestadaV4NoDisponible, err,
		)
	}
	// El nucleo solo hace la prevalidacion neutral disponible en sus contratos.
	// El conector debe repetirla y completar reloj, firma, confianza, revocacion
	// y consumo atomico dentro de su frontera autoritativa.
	if _, err := vinculo.HuellaSHA256(); err != nil || cabecera.Validar() != nil ||
		sobre.ValidarSintaxis() != nil {
		return ResultadoEjecucionDocumentalAtestadaV4{},
			ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	resultado, err := e.conector.EjecutarDocumentalAtestadoV4(
		ctx, vinculo, cabecera, sobre,
	)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return ResultadoEjecucionDocumentalAtestadaV4{}, errors.Join(
				ErrEjecucionDocumentalAtestadaV4NoDisponible, contextoErr,
			)
		}
		return ResultadoEjecucionDocumentalAtestadaV4{},
			ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	ordenRef, errOrden := resultado.OrdenRef()
	estado, errEstado := resultado.Estado()
	auditoriaRef, errAuditoria := resultado.AuditoriaRef()
	eventoRef, errEvento := resultado.EventoOutboxRef()
	registradaEn, errRegistro := resultado.RegistradaEn()
	if resultado.Validar() != nil || errOrden != nil || errEstado != nil ||
		errAuditoria != nil || errEvento != nil || errRegistro != nil {
		return ResultadoEjecucionDocumentalAtestadaV4{},
			ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return ResultadoEjecucionDocumentalAtestadaV4{
		OrdenRef: ordenRef, Estado: estado, AuditoriaRef: auditoriaRef,
		EventoOutboxRef: eventoRef, RegistradaEn: registradaEn,
	}, nil
}

func (e *EjecutorDocumentalAtestadoV4) validar() error {
	if e == nil || dependenciaEjecucionDocumentalAtestadaV4Nula(e.conector) {
		return ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return nil
}

func (*EjecutorDocumentalAtestadoV4) String() string {
	return "[EJECUTOR-DOCUMENTAL-ATESTADO-V4-SELLADO]"
}
func (e *EjecutorDocumentalAtestadoV4) GoString() string { return e.String() }
func (e *EjecutorDocumentalAtestadoV4) LogValue() slog.Value {
	return slog.StringValue(e.String())
}
func (e *EjecutorDocumentalAtestadoV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (*EjecutorDocumentalAtestadoV4) MarshalJSON() ([]byte, error) {
	return nil, ErrEjecucionDocumentalAtestadaV4NoDisponible
}

func dependenciaEjecucionDocumentalAtestadaV4Nula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
