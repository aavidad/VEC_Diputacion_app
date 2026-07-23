package ports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestRespuestaExigeVentanaFirmadaDeCincoSegundos(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	base := MetadatosAtestacionRespuestaFuenteAnalisis{
		AutoridadRef: "fuente_presupuesto_0123456789",
		Generacion:   1,
		ReciboRef:    "recibo_presupuesto_0123456789",
		EmitidaEn:    inicio,
		ValidaHasta:  inicio.Add(VigenciaMaximaRespuestaFuenteAnalisis),
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("ventana exacta rechazada: %v", err)
	}
	base.ValidaHasta = base.ValidaHasta.Add(time.Microsecond)
	if err := base.Validar(); err == nil {
		t.Fatal("ventana superior a cinco segundos aceptada")
	}
	if TiempoMaximoFuenteAnalisis != 5*time.Second {
		t.Fatalf("timeout no endurecido: %s", TiempoMaximoFuenteAnalisis)
	}
}

func TestVerificadorTCBIndependienteRechazaSalidaRemacNoAutorizada(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	metadatos := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	resultado := resultadoCosteFirmadoPrueba(t, solicitud, metadatos)
	resultado.datos.Importe.Centimos++
	canon, err := canonRespuestaCalculoCoste(
		solicitud,
		resultado.datos.FuenteRef,
		resultado.datos.ReciboRef,
		resultado.datos.Importe,
		resultado.datos.CalculadoEn,
		resultado.datos.Atestacion.Metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado.preimagen = canon
	huella, _ := (PreimagenRespuestaFuenteAnalisis{contenido: canon}).huellaSHA256()
	resultado.datos.HuellaRespuestaSHA256 = huella
	verificadorInvocado, consumidorInvocado := false, false
	_, err = CalcularCosteConFuente(
		context.Background(),
		calculadorCosteDoble(func(
			context.Context,
			SolicitudCalcularCoste,
		) (ResultadoCalculoCoste, error) {
			return resultado, nil
		}),
		verificadorRespuestaDoble(func(
			ctx context.Context,
			s SolicitudVerificarRespuestaFuenteAnalisis,
		) (ConfirmacionRespuestaFuenteAnalisis, error) {
			verificadorInvocado = true
			return verificadorRespuestaHMACPrueba(
				metadatos.EmitidaEn.Add(500*time.Millisecond),
			)(ctx, s)
		}),
		consumidorRespuestaDoble(func(
			context.Context,
			OrdenConsumoRespuestaFuenteAnalisis,
		) (ReciboConsumoRespuestaFuenteAnalisis, error) {
			consumidorInvocado = true
			return ReciboConsumoRespuestaFuenteAnalisis{}, nil
		}),
		relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(time.Second)),
		solicitud,
	)
	if !verificadorInvocado || consumidorInvocado ||
		!errors.Is(err, ErrVerificacionFuenteAnalisisNoDisponible) {
		t.Fatalf("se eludió el verificador TCB: %v", err)
	}
}

func TestFuenteYVerificadorTCBDebenSerInstanciasSeparadas(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	dependencia := &fuenteYVerificadorMismaInstanciaPrueba{}
	_, err := ValidarRCConFuente(
		context.Background(),
		dependencia,
		dependencia,
		verificadorPublicacionNoInvocablePrueba(t),
		consumidorRespuestaPrueba(inicio),
		relojFijoFuenteAnalisis(inicio),
		solicitudValidarRCPrueba(t, inicio),
	)
	if !errors.Is(err, ErrPeticionFuenteAnalisisInvalida) ||
		dependencia.invocada {
		t.Fatalf("fuente y TCB no quedaron separados: %v", err)
	}
}

func TestReplayExactoSoloSeAceptaDentroDeLaVentana(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	metadatos := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	resultado := resultadoCosteFirmadoPrueba(t, solicitud, metadatos)
	consumidor := nuevoConsumidorDurablePrueba()
	invocar := func(reloj time.Time) error {
		_, err := CalcularCosteConFuente(
			context.Background(),
			calculadorCosteDoble(func(
				context.Context,
				SolicitudCalcularCoste,
			) (ResultadoCalculoCoste, error) {
				return resultado, nil
			}),
			verificadorRespuestaHMACPrueba(metadatos.EmitidaEn.Add(500*time.Millisecond)),
			consumidor,
			relojFijoFuenteAnalisis(reloj),
			solicitud,
		)
		return err
	}
	if err := invocar(metadatos.EmitidaEn.Add(time.Second)); err != nil {
		t.Fatalf("primer consumo: %v", err)
	}
	if err := invocar(metadatos.EmitidaEn.Add(1500 * time.Millisecond)); err != nil {
		t.Fatalf("replay exacto vigente: %v", err)
	}
	llamadasAntes := consumidor.llamadas
	if err := invocar(metadatos.ValidaHasta); !errors.Is(
		err,
		ErrResultadoFuenteAnalisisNoConfiable,
	) {
		t.Fatalf("replay caducado aceptado: %v", err)
	}
	if consumidor.llamadas != llamadasAntes {
		t.Fatal("el replay caducado alcanzó el consumo durable")
	}
}

func TestMismoReciboConOtraRespuestaFallaCerrado(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	metadatos := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	primero := resultadoCosteFirmadoPrueba(t, solicitud, metadatos)
	consumidor := nuevoConsumidorDurablePrueba()
	invocar := func(resultado ResultadoCalculoCoste) error {
		_, err := CalcularCosteConFuente(
			context.Background(),
			calculadorCosteDoble(func(
				context.Context,
				SolicitudCalcularCoste,
			) (ResultadoCalculoCoste, error) {
				return resultado, nil
			}),
			verificadorRespuestaHMACPrueba(metadatos.EmitidaEn.Add(500*time.Millisecond)),
			consumidor,
			relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(time.Second)),
			solicitud,
		)
		return err
	}
	if err := invocar(primero); err != nil {
		t.Fatal(err)
	}
	importe := domain.Importe{Centimos: 3_148_026, Moneda: "EUR"}
	preimagen, err := NuevaPreimagenRespuestaCalculoCoste(
		solicitud,
		metadatos.AutoridadRef,
		metadatos.ReciboRef,
		importe,
		inicio.Add(time.Second),
		metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := NuevoResultadoCalculoCoste(
		solicitud,
		metadatos.AutoridadRef,
		metadatos.ReciboRef,
		importe,
		inicio.Add(time.Second),
		atestacionRespuestaPrueba(t, preimagen, metadatos),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := invocar(segundo); !errors.Is(
		err,
		ErrRespuestaFuenteAnalisisYaConsumida,
	) {
		t.Fatalf("recibo reutilizado aceptado: %v", err)
	}
}

func TestCatalogoDinamicoNoCompilaValoresYRedactaPII(t *testing.T) {
	motivo, err := NuevoMotivoFuenteAnalisis(
		"catalogo_motivos_publicado_2028",
		19,
		strings.Repeat("e", 64),
		"motivo_nuevo_sobrevenido",
		"contratacion_temporal.rc.motivo_nuevo_sobrevenido",
		[]ParametroMotivoFuenteAnalisis{
			{Clave: "parametro_futuro", Valor: "valor_publicado_2028"},
		},
	)
	if err != nil {
		t.Fatalf("entrada gobernada futura rechazada: %v", err)
	}
	const marca = "[MOTIVO-FUENTE-ANALISIS-CATALOGADO-REDACTADO]"
	for _, texto := range []string{
		fmt.Sprint(motivo),
		fmt.Sprintf("%+v", motivo),
		fmt.Sprintf("%#v", motivo),
	} {
		if texto != marca || strings.Contains(texto, "publicado_2028") {
			t.Fatalf("motivo no redactado: %q", texto)
		}
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "motivo", motivo)
	if !strings.Contains(registro.String(), marca) ||
		strings.Contains(registro.String(), "publicado_2028") {
		t.Fatalf("slog filtró el catálogo: %s", registro.String())
	}
}

func TestResultadosAtestadosNoFiltranMaterialAlFormatear(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	metadatos := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	resultado := resultadoCosteFirmadoPrueba(t, solicitud, metadatos)
	const marca = "[RESULTADO-CALCULO-COSTE-ATESTADO-REDACTADO]"
	for _, texto := range []string{
		fmt.Sprint(resultado),
		fmt.Sprintf("%+v", resultado),
		fmt.Sprintf("%#v", resultado),
	} {
		if texto != marca || strings.Contains(texto, "tabla_retributiva") {
			t.Fatalf("resultado no redactado: %q", texto)
		}
	}
}

func TestReciboFuncionalYAtestacionSonElMismo(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCPrueba(t, solicitud, inicio.Add(time.Second))
	metadatos := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		"recibo_atestacion_distinto_012345",
		inicio,
	)
	if _, err := NuevaPreimagenRespuestaValidacionRC(
		solicitud,
		validacion,
		MotivoFuenteAnalisis{},
		metadatos,
	); err == nil {
		t.Fatal("se aceptaron dos recibos probatorios ambiguos")
	}
}

type consumidorDurablePrueba struct {
	huella   string
	recibo   ReciboConsumoRespuestaFuenteAnalisis
	llamadas int
}

type fuenteYVerificadorMismaInstanciaPrueba struct {
	invocada bool
}

func (f *fuenteYVerificadorMismaInstanciaPrueba) ValidarRC(
	context.Context,
	SolicitudValidarRC,
) (ResultadoValidacionRC, error) {
	f.invocada = true
	return ResultadoValidacionRC{}, nil
}

func (f *fuenteYVerificadorMismaInstanciaPrueba) VerificarRespuestaFuenteAnalisis(
	context.Context,
	SolicitudVerificarRespuestaFuenteAnalisis,
) (ConfirmacionRespuestaFuenteAnalisis, error) {
	f.invocada = true
	return ConfirmacionRespuestaFuenteAnalisis{}, nil
}

func nuevoConsumidorDurablePrueba() *consumidorDurablePrueba {
	return &consumidorDurablePrueba{}
}

func (c *consumidorDurablePrueba) ConsumirRespuestaFuenteAnalisis(
	_ context.Context,
	orden OrdenConsumoRespuestaFuenteAnalisis,
) (ReciboConsumoRespuestaFuenteAnalisis, error) {
	c.llamadas++
	datos, err := orden.Datos()
	if err != nil {
		return ReciboConsumoRespuestaFuenteAnalisis{}, err
	}
	if c.huella != "" {
		if c.huella != datos.HuellaRespuestaSHA256 {
			return ReciboConsumoRespuestaFuenteAnalisis{},
				ErrRespuestaFuenteAnalisisYaConsumida
		}
		return c.recibo, nil
	}
	c.huella = datos.HuellaRespuestaSHA256
	c.recibo, err = NuevoReciboConsumoRespuestaFuenteAnalisis(
		orden,
		"consumo_durable_respuesta_0123456789",
		datos.Atestacion.Metadatos.EmitidaEn.Add(time.Second),
	)
	return c.recibo, err
}
