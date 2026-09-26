package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var (
	_ ports.BuzonNoIncorporaciones            = (*BuzonNoIncorporacionesPostgreSQL)(nil)
	_ ports.PublicadorPoliticaNoIncorporacion = (*PoliticaNoIncorporacionPostgreSQL)(nil)
)

// BuzonNoIncorporacionesPostgreSQL es la bandeja de Bolsa 000042 con el rol
// propio del relevo (vec_bolsa_llamamientos_relevo_no_incorporacion). No lee
// ni escribe tablas de Contratación temporal: la base comprueba el origen con
// la función que CT le concede.
type BuzonNoIncorporacionesPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoBuzonNoIncorporacionesPostgreSQL(pool *pgxpool.Pool) (*BuzonNoIncorporacionesPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	return &BuzonNoIncorporacionesPostgreSQL{pool: pool}, nil
}

func (b *BuzonNoIncorporacionesPostgreSQL) CursorNoIncorporaciones(ctx context.Context) (ports.CursorContratosParticipacion, bool, error) {
	var c ports.CursorContratosParticipacion
	if b == nil || b.pool == nil || ctx == nil {
		return c, false, ports.ErrContratosParticipacionNoDisponible
	}
	err := b.pool.QueryRow(ctx, `SELECT origen_posicion, origen_ref FROM vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1()`).Scan(&c.Posicion, &c.OrigenRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.CursorContratosParticipacion{}, false, nil
	}
	if err != nil {
		return ports.CursorContratosParticipacion{}, false, errorContratosParticipacion(err)
	}
	return c, true, nil
}

func plazosNoIncorporacionJSON(p *ports.PlazosNoIncorporacion) (*string, error) {
	if p == nil {
		return nil, nil
	}
	j, err := json.Marshal(p)
	if err != nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	texto := string(j)
	return &texto, nil
}

func (b *BuzonNoIncorporacionesPostgreSQL) RegistrarNoIncorporacion(ctx context.Context, e ports.EventoNoIncorporacionRecibido) (ports.ResultadoRegistroNoIncorporacion, error) {
	var r ports.ResultadoRegistroNoIncorporacion
	if b == nil || b.pool == nil || ctx == nil || e.Evento.Validar() != nil || len(e.Contenido) == 0 || e.HuellaSHA256 == "" ||
		e.OrigenCreadaEn.IsZero() || e.OrigenPosicion < 0 {
		return r, ports.ErrContratosParticipacionNoDisponible
	}
	plazos, err := plazosNoIncorporacionJSON(e.Plazos)
	if err != nil {
		return r, err
	}
	var estado, participacion *string
	var enCuarentena bool
	err = pgx.BeginTxFunc(ctx, b.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT reutilizado, estado, participacion_ref, en_cuarentena FROM vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1($1::text::jsonb,$2,$3,$4,$5::text::jsonb)`,
			string(e.Contenido), e.HuellaSHA256, e.OrigenCreadaEn.UTC().Truncate(time.Microsecond), e.OrigenPosicion, plazos).
			Scan(&r.Reutilizado, &estado, &participacion, &enCuarentena)
	})
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, errorContratosParticipacion(err)
	}
	// Lo que CT no publicó, o un segundo evento para el mismo llamamiento, ya
	// quedó en la cuarentena de Bolsa: se informa como divergente para que el
	// relevo lo registre y continúe.
	if enCuarentena {
		return ports.ResultadoRegistroNoIncorporacion{}, ports.ErrEventoContratoDivergente
	}
	return resultadoNoIncorporacion(r, estado, participacion)
}

func resultadoNoIncorporacion(r ports.ResultadoRegistroNoIncorporacion, estado, participacion *string) (ports.ResultadoRegistroNoIncorporacion, error) {
	if estado == nil || *estado == "" {
		return ports.ResultadoRegistroNoIncorporacion{}, fmt.Errorf("%w: estado ausente", ports.ErrContratosParticipacionNoDisponible)
	}
	r.Estado = *estado
	if participacion != nil {
		r.ParticipacionRef = *participacion
	}
	return r, nil
}

func (b *BuzonNoIncorporacionesPostgreSQL) PendientesNoIncorporacion(ctx context.Context, limite int) ([]ports.PendienteNoIncorporacion, error) {
	if b == nil || b.pool == nil || ctx == nil || limite < 1 || limite > ports.LimitePendientesNoIncorporacion {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	filas, err := b.pool.Query(ctx, `SELECT evento_ref, consecuencia_clave, fecha_notificacion FROM vec_bolsa_llamamientos.pendientes_no_incorporacion_bolsa_v1($1)`, limite)
	if err != nil {
		return nil, errorContratosParticipacion(err)
	}
	defer filas.Close()
	salida := make([]ports.PendienteNoIncorporacion, 0, limite)
	for filas.Next() {
		var p ports.PendienteNoIncorporacion
		if err := filas.Scan(&p.EventoRef, &p.ConsecuenciaClave, &p.FechaNotificacion); err != nil {
			return nil, errorContratosParticipacion(err)
		}
		salida = append(salida, p)
	}
	if err := filas.Err(); err != nil {
		return nil, errorContratosParticipacion(err)
	}
	return salida, nil
}

func (b *BuzonNoIncorporacionesPostgreSQL) ReevaluarNoIncorporacion(ctx context.Context, eventoRef string, plazos *ports.PlazosNoIncorporacion) (ports.ResultadoRegistroNoIncorporacion, error) {
	if b == nil || b.pool == nil || ctx == nil || eventoRef == "" {
		return ports.ResultadoRegistroNoIncorporacion{}, ports.ErrContratosParticipacionNoDisponible
	}
	texto, err := plazosNoIncorporacionJSON(plazos)
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, err
	}
	var estado, participacion *string
	err = pgx.BeginTxFunc(ctx, b.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT estado, participacion_ref FROM vec_bolsa_llamamientos.reevaluar_no_incorporacion_bolsa_v1($1,$2::text::jsonb)`,
			eventoRef, texto).Scan(&estado, &participacion)
	})
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, errorContratosParticipacion(err)
	}
	return resultadoNoIncorporacion(ports.ResultadoRegistroNoIncorporacion{Reutilizado: true}, estado, participacion)
}

// PoliticaNoIncorporacionPostgreSQL publica la política de no incorporación
// con la cuenta de ejecución general de Bolsa, como la de avisos (000041).
type PoliticaNoIncorporacionPostgreSQL struct{ pool *pgxpool.Pool }

func NuevaPoliticaNoIncorporacionPostgreSQL(pool *pgxpool.Pool) (*PoliticaNoIncorporacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	return &PoliticaNoIncorporacionPostgreSQL{pool: pool}, nil
}

func (p *PoliticaNoIncorporacionPostgreSQL) PublicarPoliticaNoIncorporacion(ctx context.Context, politica ports.PoliticaNoIncorporacion) (int64, error) {
	if p == nil || p.pool == nil || ctx == nil || politica.CatalogoRef == "" || politica.CatalogoSHA256 == "" ||
		politica.RecursoReglaRef == "" || politica.RecursoReglaHuellaSHA256 == "" || politica.Consecuencias == nil {
		return 0, ports.ErrContratosParticipacionNoDisponible
	}
	consecuencias, err := json.Marshal(politica.Consecuencias)
	if err != nil {
		return 0, ports.ErrContratosParticipacionNoDisponible
	}
	var version int64
	var reutilizada bool
	err = p.pool.QueryRow(ctx, `SELECT version, reutilizada FROM vec_bolsa_llamamientos.publicar_politica_no_incorporacion_bolsa_v1($1,$2,$3::text::jsonb,$4,$5)`,
		politica.CatalogoRef, politica.CatalogoSHA256, string(consecuencias), politica.RecursoReglaRef, politica.RecursoReglaHuellaSHA256).Scan(&version, &reutilizada)
	if err != nil {
		return 0, errorContratosParticipacion(err)
	}
	return version, nil
}
