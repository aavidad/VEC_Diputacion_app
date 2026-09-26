package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// maximoPaginasEntregaContratosCT acota una pasada: con lote 100 son 10.000
// eventos; el resto queda para la siguiente, nunca se pierde.
const maximoPaginasEntregaContratosCT = 100

// entregaContratosCTBolsa es el relevo B13 entre dos módulos: lee la
// publicación de CT (CT113) y entrega cada evento al inbox de Bolsa. Solo
// compone puertos; ninguno de los dos módulos lee tablas del otro.
type entregaContratosCTBolsa struct {
	lector   puertosct.LectorPublicacionContratosBolsa
	receptor *aplicacionbolsa.ServicioRecepcionContratos
	lote     int
}

// resultadoEntregaContratosCT resume una pasada para el registro técnico.
type resultadoEntregaContratosCT struct {
	nuevos, reentregas, rechazados int
}

// entregar hace una pasada completa desde el cursor del inbox. CT solo
// publica eventos de transacciones ya terminadas y en orden de posición, así
// que no hace falta releer hacia atrás. Un evento inválido o divergente se
// registra y no bloquea a los demás; una indisponibilidad detiene la pasada.
func (e *entregaContratosCTBolsa) entregar(ctx context.Context) (resultadoEntregaContratosCT, error) {
	var r resultadoEntregaContratosCT
	if e == nil || e.lector == nil || e.receptor == nil || e.lote < 1 || e.lote > puertosct.LimiteLecturaContratosBolsa || ctx == nil {
		return r, puertosbolsa.ErrContratosParticipacionNoDisponible
	}
	cursor, hay, err := e.receptor.Cursor(ctx)
	if err != nil {
		return r, err
	}
	var desde puertosct.CursorPublicacionContratosBolsa
	if hay {
		desde = puertosct.CursorPublicacionContratosBolsa{Posicion: cursor.Posicion, OrigenRef: cursor.OrigenRef}
	}
	for pagina := 0; pagina < maximoPaginasEntregaContratosCT; pagina++ {
		eventos, err := e.lector.LeerContratosBolsa(ctx, desde, e.lote)
		if err != nil {
			return r, err
		}
		for _, evento := range eventos {
			res, err := e.receptor.Recibir(ctx, evento.Contenido, evento.HuellaSHA256, evento.OrigenCreadaEn, evento.OrigenPosicion)
			switch {
			case errors.Is(err, dominiobolsa.ErrEventoContratoParticipacionInvalido), errors.Is(err, puertosbolsa.ErrEventoContratoDivergente):
				r.rechazados++
				slog.Warn("evento de contrato CT rechazado por el inbox de Bolsa", "evento_ref", evento.EventoRef, "causa", err)
			case err != nil:
				return r, err
			case res.Reutilizado:
				r.reentregas++
			default:
				r.nuevos++
			}
		}
		if len(eventos) < e.lote {
			return r, nil
		}
		ultimo := eventos[len(eventos)-1]
		desde = puertosct.CursorPublicacionContratosBolsa{Posicion: ultimo.OrigenPosicion, OrigenRef: ultimo.OrigenRef}
	}
	return r, nil
}

// iniciarEntregaContratosCTBolsaDesarrollo arranca el relevo si la
// configuración lo permite y devuelve la función que lo detiene y espera.
func iniciarEntregaContratosCTBolsaDesarrollo(cfg config.ConfiguracionEntregaContratosCTBolsa, ejecucionCT, bolsa *pgxpool.Pool) (func(), error) {
	nada := func() {}
	opciones, err := cfg.Resolver()
	if err != nil {
		return nada, err
	}
	if !opciones.Activa {
		slog.Info("entrega de contratos CT a Bolsa desactivada por configuración")
		return nada, nil
	}
	lector, err := postgresct.NuevoLectorPublicacionContratosBolsaPostgreSQL(ejecucionCT)
	if err != nil {
		return nada, err
	}
	buzon, err := postgresbolsa.NuevoBuzonContratosParticipacionPostgreSQL(bolsa)
	if err != nil {
		return nada, err
	}
	receptor, err := aplicacionbolsa.NuevoServicioRecepcionContratos(buzon)
	if err != nil {
		return nada, err
	}
	relevo := &entregaContratosCTBolsa{lector: lector, receptor: receptor, lote: opciones.Lote}
	return mantenerEntregaContratosCTBolsa(relevo, opciones.Intervalo, esperarTemporizadorCTDesarrollo), nil
}

func mantenerEntregaContratosCTBolsa(relevo *entregaContratosCTBolsa, intervalo time.Duration, esperar esperaRenovacionCTDesarrollo) func() {
	return mantenerEntregaCTBolsa("contratos", relevo.entregar, intervalo, esperar)
}

// mantenerEntregaCTBolsa repite una pasada del relevo cada intervalo hasta
// que se detiene; lo comparten los relevos de contratos y de no
// incorporaciones.
func mantenerEntregaCTBolsa(nombre string, entregar func(context.Context) (resultadoEntregaContratosCT, error), intervalo time.Duration, esperar esperaRenovacionCTDesarrollo) func() {
	ctx, cancelar := context.WithCancel(context.Background())
	terminado := make(chan struct{})
	go func() {
		defer close(terminado)
		for ctx.Err() == nil {
			pasada, cancelarPasada := context.WithTimeout(ctx, max(intervalo, 30*time.Second))
			r, err := entregar(pasada)
			cancelarPasada()
			if err != nil && ctx.Err() == nil {
				slog.Error("entrega CT a Bolsa no disponible; se reintentará", "relevo", nombre, "causa", err)
			} else if r.nuevos > 0 || r.rechazados > 0 {
				slog.Info("entrega CT a Bolsa", "relevo", nombre, "nuevos", r.nuevos, "reentregas", r.reentregas, "rechazados", r.rechazados)
			}
			if esperar(ctx, intervalo) != nil {
				return
			}
		}
	}()
	var una sync.Once
	return func() {
		una.Do(func() {
			cancelar()
			<-terminado
		})
	}
}
