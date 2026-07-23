package application

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestServicioConsultaCoberturaCompletaEfectoDurable(t *testing.T) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)

	comprobacion, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if err != nil {
		t.Fatal(err)
	}
	if comprobacion.Resultado != domain.ComprobacionAfirmativa {
		t.Fatalf("resultado inesperado: %q", comprobacion.Resultado)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 ||
		len(entorno.consumidor.ordenes) != 1 {
		t.Fatalf(
			"efecto durable inesperado: registros=%d ordenes=%d",
			len(entorno.consumidor.registros),
			len(entorno.consumidor.ordenes),
		)
	}
}

func TestServicioConsultaCoberturaReintentoExactoConRelojAvanzado(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}
	entorno.reloj.fijar(entorno.inicio.Add(3 * time.Second))

	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatalf("el reintento exacto debe aceptarse: %v", err)
	}

	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 ||
		len(entorno.consumidor.ordenes) != 2 {
		t.Fatalf(
			"el replay no debe duplicar efecto: registros=%d ordenes=%d",
			len(entorno.consumidor.registros),
			len(entorno.consumidor.ordenes),
		)
	}
	segundaOrden := entorno.consumidor.ordenes[1]
	datosSegunda, err := segundaOrden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosConfirmacion, err := datosSegunda.ConfirmacionRespuesta.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if !datosConfirmacion.VerificadaEn.Equal(
		entorno.inicio.Add(3 * time.Second),
	) {
		t.Fatalf(
			"el reintento no obtuvo confirmacion nueva: %s",
			datosConfirmacion.VerificadaEn,
		)
	}
	for _, registro := range entorno.consumidor.registros {
		if !registro.recibo.ConsumidaEn.Equal(
			entorno.inicio.Add(2 * time.Second),
		) {
			t.Fatalf(
				"se sustituyo el recibo original: %s",
				registro.recibo.ConsumidaEn,
			)
		}
		if err := registro.recibo.ValidarPara(segundaOrden); err != nil {
			t.Fatalf(
				"el recibo original debe ligar con la segunda orden: %v",
				err,
			)
		}
	}
}

func TestServicioConsultaCoberturaRechazaReciboInicialRetrodatado(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	entorno.consumidor.consumir = func(
		_ context.Context,
		orden ports.OrdenConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		recibo, err := ports.NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_0123456789",
			entorno.inicio.Add(2*time.Second),
		)
		recibo.ConsumidaEn = entorno.inicio.Add(1500 * time.Millisecond)
		return recibo, err
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se esperaba rechazo probatorio, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaRechazaConflictoDeReplay(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	var consultas atomic.Int32
	entorno.fuente.consultar = func(
		_ context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		numero := consultas.Add(1)
		return resultadoCoberturaAplicacionPrueba(
			t,
			solicitud,
			func(datos *ports.DatosResultadoConsultaCobertura) {
				if numero > 1 {
					datos.Comprobacion.Resultado =
						domain.ComprobacionNegativa
				}
			},
		), nil
	}
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrRespuestaCoberturaYaConsumida) {
		t.Fatalf("se esperaba conflicto durable, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaRechazaReciboDelFuturo(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	entorno.consumidor.consumir = func(
		_ context.Context,
		orden ports.OrdenConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		return ports.NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_0123456789",
			entorno.inicio.Add(3*time.Second),
		)
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se esperaba rechazo temporal, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaRechazaRetrocesoDeRelojTrasConsumo(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	entorno.consumidor.consumir = func(
		_ context.Context,
		orden ports.OrdenConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		recibo, err := ports.NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_0123456789",
			entorno.inicio.Add(2*time.Second),
		)
		entorno.reloj.fijar(entorno.inicio.Add(1500 * time.Millisecond))
		return recibo, err
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se esperaba rechazo temporal, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaRevalidaCaducidadTrasConsumo(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	entorno.consumidor.consumir = func(
		_ context.Context,
		orden ports.OrdenConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		recibo, err := ports.NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_0123456789",
			entorno.inicio.Add(2*time.Second),
		)
		entorno.reloj.fijar(entorno.inicio.Add(5 * time.Second))
		return recibo, err
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se esperaba rechazo por caducidad, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaImponePlazoAlConsumidor(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	entorno.reconstruirServicio(t, 5*time.Millisecond)
	entorno.consumidor.consumir = func(
		_ context.Context,
		orden ports.OrdenConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		time.Sleep(15 * time.Millisecond)
		return ports.NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_0123456789",
			entorno.inicio.Add(2*time.Second),
		)
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ErrConsumoCoberturaNoDisponible) ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("se esperaba plazo de consumo agotado, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaRespetaCancelacionDuranteConsumo(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	entorno.consumidor.consumir = func(
		_ context.Context,
		orden ports.OrdenConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		cancelar()
		return ports.NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_0123456789",
			entorno.inicio.Add(2*time.Second),
		)
	}

	_, err := entorno.servicio.Consultar(ctx, entorno.solicitud)

	if !errors.Is(err, ErrConsumoCoberturaNoDisponible) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("se esperaba cancelacion de consumo, recibido: %v", err)
	}
}
