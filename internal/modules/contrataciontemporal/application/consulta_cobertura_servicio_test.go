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

func TestServicioConsultaCoberturaReciboProbatorioNuevoNoDuplicaEfecto(
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
					datos.Comprobacion.ReciboRef =
						"recibo_consulta_bolsa_nuevo_012345"
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

	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatalf("la renovacion probatoria debe ser replay: %v", err)
	}

	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 ||
		len(entorno.consumidor.evidencias) != 2 ||
		len(entorno.consumidor.ordenes) != 2 {
		t.Fatalf(
			"estado durable inesperado: efectos=%d evidencias=%d ordenes=%d",
			len(entorno.consumidor.registros),
			len(entorno.consumidor.evidencias),
			len(entorno.consumidor.ordenes),
		)
	}
	for _, registro := range entorno.consumidor.registros {
		if err := registro.recibo.ValidarPara(
			entorno.consumidor.ordenes[1],
		); err != nil {
			t.Fatalf("el recibo original no valida el replay: %v", err)
		}
		if registro.recibo.ReciboRespuestaRef !=
			"recibo_consulta_bolsa_012345" {
			t.Fatal("el replay sustituyo el recibo durable original")
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
					datos.Comprobacion.ReciboRef =
						"recibo_resultado_incompatible_012345"
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

func TestServicioConsultaCoberturaMismoReciboConOtraEvidenciaEsConflicto(
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
					datos.Comprobacion.EvaluadaEn =
						solicitud.SolicitadaEn.Add(500 * time.Millisecond)
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
		t.Fatalf("se esperaba conflicto probatorio, recibido: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 {
		t.Fatalf(
			"la evidencia incompatible creo %d efectos",
			len(entorno.consumidor.registros),
		)
	}
}

func TestServicioConsultaCoberturaMismaPeticionConOtraSemanticaEsConflicto(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}
	alterada := entorno.solicitud
	alterada.CategoriaRef = "categoria_educacion_social_012345"

	_, err := entorno.servicio.Consultar(context.Background(), alterada)

	if !errors.Is(err, ports.ErrRespuestaCoberturaYaConsumida) {
		t.Fatalf("se esperaba conflicto semantico, recibido: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 {
		t.Fatalf(
			"una semantica conflictiva creo %d efectos",
			len(entorno.consumidor.registros),
		)
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

func TestServicioConsultaCoberturaCommitConfirmadoPrevaleceTrasCaducidad(
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

	if err != nil {
		t.Fatalf("un commit confirmado no debe volverse ambiguo: %v", err)
	}
}

func TestServicioConsultaCoberturaImponePlazoAlConsumidor(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	plazo := entorno.usarPlazoControlado(t)
	entorno.consumidor.consumir = func(
		_ context.Context,
		_ ports.OrdenConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		plazo.finalizar(context.DeadlineExceeded)
		return ports.ReciboConsumoCobertura{}, nil
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

func TestServicioConsultaCoberturaCancelacionPosteriorAlCommitEsExito(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	entorno.consumidor.despues = func(context.Context) {
		cancelar()
	}

	_, err := entorno.servicio.Consultar(ctx, entorno.solicitud)

	if err != nil {
		t.Fatalf("el commit confirmado debe resolver la cancelacion: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 {
		t.Fatalf(
			"la cancelacion posterior creo %d efectos",
			len(entorno.consumidor.registros),
		)
	}
}

func TestServicioConsultaCoberturaTimeoutPosteriorAlCommitEsRecuperable(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	plazo := entorno.usarPlazoControlado(t)
	entorno.consumidor.despues = func(context.Context) {
		plazo.finalizar(context.DeadlineExceeded)
	}

	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatalf("el recibo durable debe confirmar el commit: %v", err)
	}
	entorno.consumidor.despues = nil
	entorno.reconstruirServicio(t, time.Second)
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatalf("el reintento debe recuperar el mismo efecto: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 {
		t.Fatalf(
			"el timeout posterior creo %d efectos",
			len(entorno.consumidor.registros),
		)
	}
}

func TestServicioConsultaCoberturaSinReciboTrasCommitRecuperaPorPeticion(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	plazo := entorno.usarPlazoControlado(t)
	entorno.consumidor.despues = func(context.Context) {
		plazo.finalizar(context.DeadlineExceeded)
	}
	entorno.consumidor.responder = func(
		ports.ReciboConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		return ports.ReciboConsumoCobertura{}, nil
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)
	if !errors.Is(err, ErrConsumoCoberturaNoDisponible) ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("sin recibo verificable debe fallar cerrado: %v", err)
	}

	entorno.consumidor.despues = nil
	entorno.consumidor.responder = nil
	entorno.reconstruirServicio(t, time.Second)
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatalf("el reintento no recupero el commit por peticion: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 {
		t.Fatalf(
			"la recuperacion creo %d efectos",
			len(entorno.consumidor.registros),
		)
	}
}
