package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestServicioConsultaCoberturaReplayConRotacionK1AK2(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	identidadK1 := entorno.verificador.identidad
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}

	claveK2 := claveEd25519CoberturaPrueba("verificador-k2")
	identidadK2 := identidadCoberturaAplicacionPrueba(
		t,
		identidadK1.AutoridadRef(),
		identidadK1.BackendRef(),
		claveK2,
		ports.RolVerificadorCobertura,
	)
	entorno.verificador.identidad = identidadK2
	entorno.autenticador.identidades[ports.RolVerificadorCobertura] =
		identidadK2
	entorno.verificador.verificar = func(
		_ context.Context,
		solicitud ports.SolicitudVerificarRespuestaCobertura,
	) (ports.ConfirmacionRespuestaCobertura, error) {
		return verificarRespuestaCoberturaAplicacionPrueba(
			solicitud,
			identidadK2.AutoridadRef(),
			claveK2,
			entorno.reloj.Ahora(),
		)
	}
	instanteK1 := entorno.inicio.Add(2 * time.Second)
	instanteK2 := entorno.inicio.Add(3 * time.Second)
	entorno.autenticador.resolver = func(
		evidencia ports.EvidenciaPublicaAutoridadFuenteAnalisis,
	) (ports.IdentidadAutoridadFuenteAnalisis, error) {
		_, _, rol, comprobadaEn, err := evidencia.Datos()
		if err != nil {
			return ports.IdentidadAutoridadFuenteAnalisis{}, err
		}
		if rol == ports.RolVerificadorCobertura {
			if comprobadaEn.Equal(instanteK1) {
				return identidadK1, nil
			}
			if comprobadaEn.Equal(instanteK2) {
				return identidadK2, nil
			}
		}
		identidad, existe := entorno.autenticador.identidades[rol]
		if !existe {
			return ports.IdentidadAutoridadFuenteAnalisis{},
				ports.ErrResultadoFuenteAnalisisNoConfiable
		}
		return identidad, nil
	}
	entorno.reloj.fijar(instanteK2)

	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatalf("el replay K1 con operación actual K2 falló: %v", err)
	}

	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 ||
		len(entorno.consumidor.ordenes) != 2 {
		t.Fatalf(
			"efecto rotado incorrecto: registros=%d ordenes=%d",
			len(entorno.consumidor.registros),
			len(entorno.consumidor.ordenes),
		)
	}
	for _, registro := range entorno.consumidor.registros {
		if registro.recibo.ValidarIdentidadVerificadorOriginal(
			identidadK1,
		) != nil ||
			registro.recibo.ValidarPara(
				entorno.consumidor.ordenes[1],
			) != nil {
			t.Fatal("el recibo histórico K1 no quedó ligado al efecto")
		}
	}
}

func TestServicioConsultaCoberturaRechazaEvidenciaHistoricaRevocada(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}
	instanteOriginal := entorno.inicio.Add(2 * time.Second)
	entorno.autenticador.resolver = func(
		evidencia ports.EvidenciaPublicaAutoridadFuenteAnalisis,
	) (ports.IdentidadAutoridadFuenteAnalisis, error) {
		_, _, rol, comprobadaEn, err := evidencia.Datos()
		if err != nil {
			return ports.IdentidadAutoridadFuenteAnalisis{}, err
		}
		if rol == ports.RolVerificadorCobertura &&
			comprobadaEn.Equal(instanteOriginal) {
			return ports.IdentidadAutoridadFuenteAnalisis{},
				ports.ErrResultadoFuenteAnalisisNoConfiable
		}
		return entorno.autenticador.identidades[rol], nil
	}
	entorno.reloj.fijar(entorno.inicio.Add(3 * time.Second))

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("evidencia histórica revocada aceptada: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 {
		t.Fatalf("la revocación duplicó %d efectos", len(entorno.consumidor.registros))
	}
}

func TestServicioConsultaCoberturaReplayNoEnmascaraAutoridadActualInvalida(
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
	entorno.autenticador.antes = func(
		rol ports.RolAutoridadFuenteAnalisis,
		comprobadaEn time.Time,
	) error {
		if rol == ports.RolVerificadorCobertura &&
			comprobadaEn.Equal(entorno.inicio.Add(3*time.Second)) {
			return ports.ErrResultadoFuenteAnalisisNoConfiable
		}
		return nil
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("el recibo histórico enmascaró la autoridad actual: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.ordenes) != 1 ||
		len(entorno.consumidor.registros) != 1 {
		t.Fatal("la autoridad actual inválida alcanzó el consumo")
	}
}

func TestServicioConsultaCoberturaRechazaClaveHistoricaAdulterada(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}
	identidadAjena := identidadCoberturaAplicacionPrueba(
		t,
		entorno.verificador.identidad.AutoridadRef(),
		entorno.verificador.identidad.BackendRef(),
		claveEd25519CoberturaPrueba("verificador-historico-ajeno"),
		ports.RolVerificadorCobertura,
	)
	entorno.consumidor.responder = func(
		recibo ports.ReciboConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error) {
		recibo.IdentidadVerificadorOriginal = identidadAjena
		return recibo, nil
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("clave histórica adulterada aceptada: %v", err)
	}
}

func TestServicioConsultaCoberturaReplayCaducadoNoAlcanzaConsumo(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}
	entorno.reloj.fijar(entorno.inicio.Add(5 * time.Second))

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("replay en fin exclusivo aceptado: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.ordenes) != 1 ||
		len(entorno.consumidor.registros) != 1 {
		t.Fatal("el replay caducado alcanzó el consumo durable")
	}
}
