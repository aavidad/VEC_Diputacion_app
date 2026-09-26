package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Relevo de no incorporaciones (duda 12 de RRHH): lee la publicación de CT
// 000124 y la entrega a la bandeja de Bolsa 000042, que aplica la baja que
// fija el catálogo de Bolsa y habilita el siguiente llamamiento. Como el de
// contratos, solo compone puertos: ningún módulo lee tablas del otro.

var (
	ErrIncorporacionAcreditadaFaltaBolsa42 = fmt.Errorf("%w: falta Bolsa 000042 (bandeja de no incorporaciones)", ErrIncorporacionAcreditadaMigracionesNoDisponibles)
	errNoIncorporacionesSinCatalogoBolsa   = errors.New("bootstrap: la no incorporación exige el paquete de reglas de Bolsa (consecuencias b24)")
)

// consultaMigracionBolsa42 se ejecuta con el LOGIN ejecutor de Bolsa.
const consultaMigracionBolsa42 = `SELECT coalesce(pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(
  'vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(jsonb,text,timestamptz,bigint,jsonb)'),'EXECUTE'),false)
 AND coalesce(pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(
  'vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1()'),'EXECUTE'),false)`

type entregaNoIncorporacionesCTBolsa struct {
	lector   puertosct.LectorPublicacionNoIncorporacionesBolsa
	receptor *aplicacionbolsa.ServicioRecepcionNoIncorporaciones
	lote     int
}

// entregar hace una pasada completa desde el cursor de la bandeja. Un evento
// inválido o divergente se registra y no bloquea a los demás; una
// indisponibilidad (también del catálogo) detiene la pasada.
func (e *entregaNoIncorporacionesCTBolsa) entregar(ctx context.Context) (resultadoEntregaContratosCT, error) {
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
		eventos, err := e.lector.LeerNoIncorporacionesBolsa(ctx, desde, e.lote)
		if err != nil {
			return r, err
		}
		for _, evento := range eventos {
			res, err := e.receptor.Recibir(ctx, evento.Contenido, evento.HuellaSHA256, evento.OrigenCreadaEn, evento.OrigenPosicion)
			switch {
			case errors.Is(err, dominiobolsa.ErrEventoContratoParticipacionInvalido), errors.Is(err, puertosbolsa.ErrEventoContratoDivergente):
				r.rechazados++
				slog.Warn("no incorporación de CT rechazada por la bandeja de Bolsa", "evento_ref", evento.EventoRef, "causa", err)
			case err != nil:
				return r, err
			case res.Reutilizado:
				r.reentregas++
			default:
				r.nuevos++
				if res.Estado != "aplicada" {
					slog.Warn("no incorporación de CT registrada en Bolsa sin baja; revisar", "evento_ref", evento.EventoRef, "estado", res.Estado)
				}
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

// iniciarEntregaNoIncorporacionesCTBolsaDesarrollo arranca el relevo con la
// misma configuración que el de contratos. Con la incorporación acreditada
// encendida, la falta de Bolsa 000042 o del catálogo de Bolsa impide arrancar.
func iniciarEntregaNoIncorporacionesCTBolsaDesarrollo(ctx context.Context, cfg config.ConfiguracionEntregaContratosCTBolsa, ejecucionCT, bolsa *pgxpool.Pool,
	catalogo puertosbolsa.ResolvedorSancionNoIncorporacion) (func(), error) {
	nada := func() {}
	if dependenciaEsNulaContratacionTemporalDesarrollo(catalogo) {
		return nada, errNoIncorporacionesSinCatalogoBolsa
	}
	if bolsa == nil || ejecucionCT == nil {
		return nada, ErrIncorporacionAcreditadaFaltaBolsa42
	}
	var instalada bool
	if err := bolsa.QueryRow(ctx, consultaMigracionBolsa42).Scan(&instalada); err != nil || !instalada {
		return nada, ErrIncorporacionAcreditadaFaltaBolsa42
	}
	opciones, err := cfg.Resolver()
	if err != nil {
		return nada, err
	}
	if !opciones.Activa {
		slog.Warn("entrega de no incorporaciones CT a Bolsa desactivada por configuración: la baja y el siguiente esperan al relevo")
		return nada, nil
	}
	lector, err := postgresct.NuevoLectorPublicacionContratosBolsaPostgreSQL(ejecucionCT)
	if err != nil {
		return nada, err
	}
	buzon, err := postgresbolsa.NuevoBuzonNoIncorporacionesPostgreSQL(bolsa)
	if err != nil {
		return nada, err
	}
	receptor, err := aplicacionbolsa.NuevoServicioRecepcionNoIncorporaciones(buzon, catalogo)
	if err != nil {
		return nada, err
	}
	relevo := &entregaNoIncorporacionesCTBolsa{lector: lector, receptor: receptor, lote: opciones.Lote}
	return mantenerEntregaCTBolsa("no incorporaciones", relevo.entregar, opciones.Intervalo, esperarTemporizadorCTDesarrollo), nil
}
