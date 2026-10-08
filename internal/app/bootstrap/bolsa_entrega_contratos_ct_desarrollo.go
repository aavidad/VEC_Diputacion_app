package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// maximoPaginasEntregaContratosCT acota una pasada: con lote 100 son 10.000
// eventos; el resto queda para la siguiente, nunca se pierde.
const maximoPaginasEntregaContratosCT = 100

const (
	rolRelevoCeseBolsaDesarrollo = "vec_bolsa_llamamientos_relevo_cese"
)

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
	// sinCandidato cuenta ceses B81 sin candidato de bolsa constituida.
	sinCandidato int
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
			pasada, cancelarPasada := context.WithTimeout(ctx, plazoarranque.Ampliar(max(intervalo, 30*time.Second)))
			r, err := entregar(pasada)
			cancelarPasada()
			if err != nil && ctx.Err() == nil {
				slog.Error("entrega CT a Bolsa no disponible; se reintentará", "relevo", nombre, "causa", err)
			} else if r.nuevos > 0 || r.rechazados > 0 || r.sinCandidato > 0 {
				slog.Info("entrega CT a Bolsa", "relevo", nombre, "nuevos", r.nuevos, "reentregas", r.reentregas,
					"rechazados", r.rechazados, "sin_candidato", r.sinCandidato)
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

// entregaCesesCTBolsa usa un cursor B45 distinto del inbox B13. Lee solo
// ceses CT confirmados, ya filtrados antes de paginar por el propietario CT.
// Un fallo detiene la pasada y conserva el cursor durable anterior.
type consultorCesesCTBolsa interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type entregaCesesCTBolsa struct {
	lector puertosct.LectorPublicacionCesesBolsa
	pool   consultorCesesCTBolsa
	lote   int
}

// El worker compartido escribe el error recibido en slog. Nunca se le
// entrega el texto de pgconn: puede contener host, usuario o DSN privado.
func falloRelevoCeseBolsaDesarrollo(err error) error {
	slog.Error("relevo de ceses CT a Bolsa no disponible", "causa", causaFalloPostgreSQLCTDesarrollo(err))
	return puertosbolsa.ErrContratosParticipacionNoDisponible
}

// B90 instala en una migración la vista de pendientes y la lectura de estado
// que impide elegirlos. El relevo no arranca si falta alguna de las dos.
func verificarGuardasCesePendienteB90(ctx context.Context, pool consultorCesesCTBolsa) error {
	if ctx == nil || pool == nil {
		return fmt.Errorf("%w: clave=consulta_guardas_cese_pendiente esperado=contexto_y_pool actual=ausente",
			puertosbolsa.ErrContratosParticipacionNoDisponible)
	}
	var vista, lote bool
	err := pool.QueryRow(ctx, `SELECT to_regclass('vec_bolsa_llamamientos.candidatos_cese_pendiente_b90') IS NOT NULL,
		to_regprocedure('vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(text[],timestamptz)') IS NOT NULL`).Scan(&vista, &lote)
	if err != nil {
		return fmt.Errorf("%w: clave=consulta_guardas_cese_pendiente esperado=ejecutada actual=fallida",
			puertosbolsa.ErrContratosParticipacionNoDisponible)
	}
	if !vista || !lote {
		return fmt.Errorf("%w: clave=B90_vista esperado=true actual=%t; clave=B90_lote esperado=true actual=%t",
			puertosbolsa.ErrContratosParticipacionNoDisponible, vista, lote)
	}
	return nil
}

func (e *entregaCesesCTBolsa) entregar(ctx context.Context) (resultadoEntregaContratosCT, error) {
	var resultado resultadoEntregaContratosCT
	if e == nil || e.lector == nil || e.pool == nil || e.lote < 1 || e.lote > puertosct.LimiteLecturaContratosBolsa || ctx == nil {
		return resultado, puertosct.ErrPublicacionContratosBolsaNoDisponible
	}
	var desde puertosct.CursorPublicacionContratosBolsa
	err := e.pool.QueryRow(ctx, `SELECT origen_posicion,origen_ref FROM vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()`).Scan(&desde.Posicion, &desde.OrigenRef)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return resultado, falloRelevoCeseBolsaDesarrollo(err)
	}
	for pagina := 0; pagina < maximoPaginasEntregaContratosCT; pagina++ {
		eventos, err := e.lector.LeerCesesBolsa(ctx, desde, e.lote)
		if err != nil {
			return resultado, falloRelevoCeseBolsaDesarrollo(err)
		}
		for _, evento := range eventos {
			var contenido struct {
				Esquema   string `json:"esquema"`
				Tipo      string `json:"tipo"`
				OrigenRef string `json:"origen_ref"`
			}
			if json.Unmarshal(evento.Contenido, &contenido) != nil || contenido.Esquema != "vec.contratacion-temporal.contrato-bolsa.v1" ||
				contenido.Tipo != "cese" || contenido.OrigenRef != evento.OrigenRef || evento.HuellaSHA256 == "" || evento.OrigenPosicion < 0 {
				return resultado, puertosct.ErrPublicacionContratosBolsaNoDisponible
			}
			var reutilizada, sinCandidato bool
			var recibo, candidato string
			var disponible time.Time
			var politica int64
			err := e.pool.QueryRow(ctx, `SELECT reutilizada,recibo_ref,candidato_ref,disponible_desde,politica_version
				FROM vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1($1,$2,$3)`, evento.OrigenRef, evento.HuellaSHA256, evento.OrigenPosicion).
				Scan(&reutilizada, &recibo, &candidato, &disponible, &politica)
			if err != nil {
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "23503" {
					return resultado, falloRelevoCeseBolsaDesarrollo(err)
				}
				// Un cese sin llamamiento Bolsa también queda acreditado. Si el
				// llamamiento existe pero falta su vínculo, SQL deniega y reintenta.
				err = e.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.confirmar_cese_ajeno_bolsa_v1($1,$2,$3)`,
					evento.OrigenRef, evento.HuellaSHA256, evento.OrigenPosicion).Scan(&reutilizada)
				if err != nil {
					if !errors.As(err, &pg) || pg.Code != "23503" {
						return resultado, falloRelevoCeseBolsaDesarrollo(err)
					}
					// B81: la participación elegida no pertenece a ninguna bolsa
					// constituida y nunca tendrá candidato. Queda auditado y el
					// cursor avanza; con bolsa constituida SQL vuelve a negar con
					// 23503 y el cese sigue pendiente.
					if err := e.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1($1,$2,$3)`,
						evento.OrigenRef, evento.HuellaSHA256, evento.OrigenPosicion).Scan(&reutilizada); err != nil {
						return resultado, falloRelevoCeseBolsaDesarrollo(err)
					}
					sinCandidato = true
				}
			} else if recibo == "" || candidato == "" || disponible.IsZero() || politica < 1 {
				return resultado, puertosbolsa.ErrContratosParticipacionNoDisponible
			}
			switch {
			case reutilizada:
				resultado.reentregas++
			case sinCandidato:
				resultado.sinCandidato++
			default:
				resultado.nuevos++
			}
			desde = puertosct.CursorPublicacionContratosBolsa{Posicion: evento.OrigenPosicion, OrigenRef: evento.OrigenRef}
		}
		if len(eventos) < e.lote {
			return resultado, nil
		}
	}
	return resultado, nil
}

// iniciarEntregaCesesCTBolsaDesarrollo exige un LOGIN exclusivo del relevo y
// las funciones CT129/B45 antes de iniciar el trabajador. No reutiliza el
// pool del ejecutor Bolsa ni su cursor de contratos B13.
func iniciarEntregaCesesCTBolsaDesarrollo(ctx context.Context, cfg config.Config, ejecucionCT *pgxpool.Pool) (func(), error) {
	nada := func() {}
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaCeseCTEnabled)
	if err != nil || !activo {
		return nada, err
	}
	opciones, err := cfg.BolsaContratosCT.Resolver()
	if err != nil || !opciones.Activa || ejecucionCT == nil || ctx == nil {
		return nada, puertosct.ErrPublicacionContratosBolsaNoDisponible
	}
	dsn, err := cfg.DSNBolsaRelevoCeseSeparado()
	if err != nil {
		return nada, err
	}
	pool, err := abrirPoolRelevoBolsaDesarrollo(ctx, dsn, rolRelevoCeseBolsaDesarrollo, "vec-bolsa-relevo-cese-ct")
	if err != nil {
		return nada, falloRelevoCeseBolsaDesarrollo(err)
	}
	var instalada bool
	err = pool.QueryRow(ctx, `SELECT bool_and(coalesce(has_function_privilege(to_regprocedure(f),'EXECUTE'),false))
		FROM unnest(ARRAY[
			'vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(text,text,bigint)',
			'vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()',
			'vec_bolsa_llamamientos.confirmar_cese_ajeno_bolsa_v1(text,text,bigint)',
			'vec_bolsa_llamamientos.confirmar_cese_sin_candidato_bolsa_v1(text,text,bigint)',
			'vec_bolsa_llamamientos.listar_ceses_sin_candidato_pendientes_v1(integer)']) f`).Scan(&instalada)
	if err != nil || !instalada {
		pool.Close()
		return nada, puertosbolsa.ErrContratosParticipacionNoDisponible
	}
	if err = verificarGuardasCesePendienteB90(ctx, pool); err != nil {
		pool.Close()
		return nada, err
	}
	if err = ejecucionCT.QueryRow(ctx, `SELECT to_regprocedure('vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)') IS NOT NULL
		AND to_regprocedure('vec_contratacion_temporal.verificar_auditoria_cese_publicado_bolsa_v1(text,text,bigint)') IS NOT NULL`).Scan(&instalada); err != nil || !instalada {
		pool.Close()
		return nada, puertosct.ErrPublicacionContratosBolsaNoDisponible
	}
	lector, err := postgresct.NuevoLectorPublicacionContratosBolsaPostgreSQL(ejecucionCT)
	if err != nil {
		pool.Close()
		return nada, err
	}
	relevo := &entregaCesesCTBolsa{lector: lector, pool: pool, lote: opciones.Lote}
	pendientes := &reconciliacionCesesB81{pool: pool, lote: opciones.Lote}
	detener := mantenerEntregaCTBolsa("ceses", func(ctx context.Context) (resultadoEntregaContratosCT, error) {
		return entregarCesesConReconciliacionB81(ctx, relevo, pendientes)
	}, opciones.Intervalo, esperarTemporizadorCTDesarrollo)
	return func() { detener(); pool.Close() }, nil
}
