package ports

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type fuentePresupuestariaDoble func(
	context.Context,
	SolicitudValidarRC,
) (ResultadoValidacionRC, error)

func (f fuentePresupuestariaDoble) ValidarRC(
	ctx context.Context,
	solicitud SolicitudValidarRC,
) (ResultadoValidacionRC, error) {
	return f(ctx, solicitud)
}

type calculadorCosteDoble func(
	context.Context,
	SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error)

func (c calculadorCosteDoble) CalcularCoste(
	ctx context.Context,
	solicitud SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error) {
	return c(ctx, solicitud)
}

func TestValidarRCConFuenteDevuelveCopiaLigada(t *testing.T) {
	solicitud := solicitudValidarRCPrueba()
	validacion := validacionRCPrueba(solicitud)
	fuente := fuentePresupuestariaDoble(func(
		_ context.Context,
		recibida SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		if recibida != solicitud {
			t.Fatalf("solicitud alterada: %#v", recibida)
		}
		return ResultadoValidacionRC{
			PeticionRef: recibida.PeticionRef,
			Validacion:  validacion,
		}, nil
	})

	resultado, err := ValidarRCConFuente(context.Background(), fuente, solicitud)
	if err != nil {
		t.Fatalf("validar RC: %v", err)
	}
	if resultado.Validar() != nil || resultado.Importe == validacion.Importe ||
		resultado.FechaRC == validacion.FechaRC {
		t.Fatal("el resultado no es válido o comparte punteros con el conector")
	}
	*resultado.Importe = domain.Importe{Centimos: 1, Moneda: "EUR"}
	if validacion.Importe.Centimos == resultado.Importe.Centimos {
		t.Fatal("la mutación del consumidor alcanzó al resultado del conector")
	}
}

func TestValidarRCConFuenteFallaCerradoAnteErrorEIndisponibilidad(t *testing.T) {
	solicitud := solicitudValidarRCPrueba()
	resultadoAparentementePositivo := ResultadoValidacionRC{
		PeticionRef: solicitud.PeticionRef,
		Validacion: domain.ValidacionRC{
			Resultado:           RCNoRequeridaPrueba(),
			EntradaRef:          solicitud.Entrada.Referencia,
			HuellaEntradaSHA256: solicitud.Entrada.HuellaSHA256,
			FuenteRef:           "fuente:presupuesto-prueba",
			ReciboRef:           "recibo:presupuesto-prueba",
			ValidadaEn:          solicitud.SolicitadaEn,
			Motivo:              "La fuente informa de que no procede retención de crédito.",
		},
	}
	fallo := errors.New("fuente temporalmente inaccesible")
	fuente := fuentePresupuestariaDoble(func(
		context.Context,
		SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		return resultadoAparentementePositivo, fallo
	})

	resultado, err := ValidarRCConFuente(context.Background(), fuente, solicitud)
	if !errors.Is(err, ErrFuentePresupuestariaNoDisponible) ||
		!errors.Is(err, fallo) || resultado.HabilitaAvance() {
		t.Fatalf("la indisponibilidad se interpretó como validación: %#v, %v", resultado, err)
	}
}

func TestValidarRCConFuenteRechazaResultadoCruzadoYNuloTipado(t *testing.T) {
	solicitud := solicitudValidarRCPrueba()
	cruzado := ResultadoValidacionRC{
		PeticionRef: "peticion:rc-distinta",
		Validacion:  validacionRCPrueba(solicitud),
	}
	fuente := fuentePresupuestariaDoble(func(
		context.Context,
		SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		return cruzado, nil
	})
	if _, err := ValidarRCConFuente(
		context.Background(),
		fuente,
		solicitud,
	); !errors.Is(err, ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("aceptó un resultado de otra petición: %v", err)
	}

	var nula *fuentePresupuestariaNula
	if _, err := ValidarRCConFuente(
		context.Background(),
		nula,
		solicitud,
	); !errors.Is(err, ErrPeticionFuenteAnalisisInvalida) {
		t.Fatalf("aceptó una fuente con puntero nulo: %v", err)
	}
}

type fuentePresupuestariaNula struct{}

func (*fuentePresupuestariaNula) ValidarRC(
	context.Context,
	SolicitudValidarRC,
) (ResultadoValidacionRC, error) {
	panic("no debe invocarse")
}

func TestCalcularCosteConFuenteLigaPeticionYRespetaCancelacion(t *testing.T) {
	solicitud := solicitudCalcularCostePrueba()
	calculador := calculadorCosteDoble(func(
		_ context.Context,
		recibida SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error) {
		return ResultadoCalculoCoste{
			PeticionRef: recibida.PeticionRef, ExpedienteRef: recibida.ExpedienteRef,
			FuenteRef: "tabla:retributiva-2026-v3", ReciboRef: "recibo:coste-001",
			Importe:     domain.Importe{Centimos: 3_148_025, Moneda: "EUR"},
			CalculadoEn: recibida.SolicitadaEn.Add(time.Second),
		}, nil
	})
	resultado, err := CalcularCosteConFuente(context.Background(), calculador, solicitud)
	if err != nil || resultado.ValidarPara(solicitud) != nil {
		t.Fatalf("calcular coste: %#v, %v", resultado, err)
	}

	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := CalcularCosteConFuente(
		ctx,
		calculador,
		solicitud,
	); !errors.Is(err, ErrCalculadorCosteNoDisponible) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("no propagó la cancelación de forma cerrada: %v", err)
	}
}

func TestCalcularCosteConFuenteRechazaResultadoDeOtroExpediente(t *testing.T) {
	solicitud := solicitudCalcularCostePrueba()
	calculador := calculadorCosteDoble(func(
		_ context.Context,
		recibida SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error) {
		return ResultadoCalculoCoste{
			PeticionRef: recibida.PeticionRef, ExpedienteRef: "expediente:ajeno",
			FuenteRef: "tabla:retributiva-2026-v3", ReciboRef: "recibo:coste-001",
			Importe:     domain.Importe{Centimos: 3_148_025, Moneda: "EUR"},
			CalculadoEn: recibida.SolicitadaEn,
		}, nil
	})
	if _, err := CalcularCosteConFuente(
		context.Background(),
		calculador,
		solicitud,
	); !errors.Is(err, ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("aceptó el coste de otro expediente: %v", err)
	}
}

func TestCalcularCosteConFuenteDescartaValorSiElConectorFalla(t *testing.T) {
	solicitud := solicitudCalcularCostePrueba()
	fallo := errors.New("detalle privado del proveedor")
	calculador := calculadorCosteDoble(func(
		_ context.Context,
		recibida SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error) {
		return ResultadoCalculoCoste{
			PeticionRef: recibida.PeticionRef, ExpedienteRef: recibida.ExpedienteRef,
			FuenteRef: "tabla:retributiva-2026-v3", ReciboRef: "recibo:coste-001",
			Importe:     domain.Importe{Centimos: 3_148_025, Moneda: "EUR"},
			CalculadoEn: recibida.SolicitadaEn,
		}, fallo
	})
	resultado, err := CalcularCosteConFuente(context.Background(), calculador, solicitud)
	if !errors.Is(err, ErrCalculadorCosteNoDisponible) || !errors.Is(err, fallo) ||
		err.Error() != ErrCalculadorCosteNoDisponible.Error() ||
		resultado != (ResultadoCalculoCoste{}) {
		t.Fatalf("el fallo filtró o aceptó un resultado: %#v, %v", resultado, err)
	}
}

func solicitudValidarRCPrueba() SolicitudValidarRC {
	fechaRC := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return SolicitudValidarRC{
		PeticionRef:     "peticion:validacion-rc-001",
		OrganizacionRef: "organizacion:diputacion-granada",
		ExpedienteRef:   "expediente:temporal-001",
		Entrada: domain.VinculoEntradaRC{
			Referencia:   "entrada:rc-001",
			HuellaSHA256: cadena64Puertos("a"),
		},
		Declaracion: domain.DeclaracionRC{
			Existe: true, Numero: "rc:2026-001", Fecha: fechaRC,
			Importe:      domain.Importe{Centimos: 3_245_000, Moneda: "EUR"},
			DocumentoRef: "documento:rc-001",
		},
		SolicitadaEn: time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
	}
}

func validacionRCPrueba(solicitud SolicitudValidarRC) domain.ValidacionRC {
	fechaRC := solicitud.Declaracion.Fecha
	importe := solicitud.Declaracion.Importe
	return domain.ValidacionRC{
		Resultado:           domain.RCValidada,
		EntradaRef:          solicitud.Entrada.Referencia,
		HuellaEntradaSHA256: solicitud.Entrada.HuellaSHA256,
		FuenteRef:           "fuente:presupuesto-prueba",
		ReciboRef:           "recibo:presupuesto-prueba",
		ValidadaEn:          solicitud.SolicitadaEn.Add(time.Second),
		FechaRC:             &fechaRC, Numero: solicitud.Declaracion.Numero,
		Importe: &importe, DocumentoRef: solicitud.Declaracion.DocumentoRef,
	}
}

func solicitudCalcularCostePrueba() SolicitudCalcularCoste {
	return SolicitudCalcularCoste{
		PeticionRef:     "peticion:calculo-coste-001",
		OrganizacionRef: "organizacion:diputacion-granada",
		ExpedienteRef:   "expediente:temporal-001",
		CategoriaRef:    "categoria:trabajo-social", GrupoSubgrupo: "A2",
		ModalidadClave: "sustitucion", CausaClave: "incapacidad_temporal",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		Jornada:      domain.JornadaCompletaDiezmilesimas,
		SolicitadaEn: time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
	}
}

func cadena64Puertos(caracter string) string {
	var resultado string
	for range 64 {
		resultado += caracter
	}
	return resultado
}

func RCNoRequeridaPrueba() domain.ResultadoValidacionRC {
	return domain.RCNoRequerida
}
