package postgres

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var (
	_ puertosbolsa.PublicadorPoliticaAvisos      = (*RepositorioPoliticaAvisosPostgreSQL)(nil)
	_ puertosbolsa.ConsultaMarcasParticipaciones = (*RepositorioPoliticaAvisosPostgreSQL)(nil)

	patronSHA256PoliticaAvisos = regexp.MustCompile(`^[a-f0-9]{64}$`)
	// ErrPoliticaAvisosSinMigracion distingue la base sin Bolsa 000041 para
	// que el arranque lo diga claro en lugar de un fallo genérico.
	ErrPoliticaAvisosSinMigracion = errors.New("bolsa: falta la migracion 000041 (parametros de avisos)")
)

const maximoMarcasBolsa = 20000

// RepositorioPoliticaAvisosPostgreSQL publica los parámetros de avisos del
// catálogo y lee las marcas por participación (Bolsa 000041).
type RepositorioPoliticaAvisosPostgreSQL struct {
	pool *pgxpool.Pool
}

func NuevoRepositorioPoliticaAvisosPostgreSQL(pool *pgxpool.Pool) (*RepositorioPoliticaAvisosPostgreSQL, error) {
	if pool == nil {
		return nil, puertosbolsa.ErrPoliticaAvisosNoDisponible
	}
	return &RepositorioPoliticaAvisosPostgreSQL{pool: pool}, nil
}

func nuloEntero(v int) any {
	if v == 0 {
		return nil
	}
	return int32(v)
}

// PublicarPoliticaAvisos registra los parámetros como versión nueva si
// difieren de la vigente. Los ceros se envían nulos: la base conserva el
// plazo anterior y desactiva lo demás.
func (r *RepositorioPoliticaAvisosPostgreSQL) PublicarPoliticaAvisos(ctx context.Context, p puertosbolsa.PublicacionPoliticaAvisos) (int64, error) {
	if r == nil || r.pool == nil || ctx == nil || p.CatalogoRef == "" || len(p.CatalogoRef) > 512 ||
		!patronSHA256PoliticaAvisos.MatchString(p.CatalogoSHA256) || p.Politica.Validar() != nil {
		return 0, puertosbolsa.ErrPoliticaAvisosNoDisponible
	}
	var meses, antelacion any
	if p.Politica.ContinuadoConfigurado {
		meses, antelacion = int32(p.Politica.ContinuadoMeses), int32(p.Politica.ContinuadoAntelacionDias)
	}
	var modo, situaciones any
	if p.Politica.PrestaServiciosModo != "" {
		modo, situaciones = p.Politica.PrestaServiciosModo, p.Politica.PrestaServiciosSituaciones
	}
	var version int64
	var reutilizada bool
	err := r.pool.QueryRow(ctx, `SELECT version,reutilizada FROM vec_bolsa_llamamientos.publicar_politica_avisos_bolsa_v1($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.CatalogoRef, p.CatalogoSHA256, meses, antelacion, nuloEntero(p.Politica.EncadenamientoUmbralMeses),
		nuloEntero(p.Politica.EncadenamientoVentanaMeses), modo, situaciones).Scan(&version, &reutilizada)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42883" {
		return 0, fmt.Errorf("%w: %w", puertosbolsa.ErrPoliticaAvisosNoDisponible, ErrPoliticaAvisosSinMigracion)
	}
	if err != nil || version < 1 {
		return 0, puertosbolsa.ErrPoliticaAvisosNoDisponible
	}
	return version, nil
}

// MarcasParticipaciones lee las marcas de las participaciones de una bolsa.
func (r *RepositorioPoliticaAvisosPostgreSQL) MarcasParticipaciones(ctx context.Context, bolsaRef string, corte time.Time) (map[string]dominiobolsa.MarcasParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || bolsaRef == "" || len(bolsaRef) > 512 || corte.IsZero() {
		return nil, puertosbolsa.ErrPoliticaAvisosNoDisponible
	}
	filas, err := r.pool.Query(ctx, `SELECT participacion_ref,coalesce(presta_servicios,''),coalesce(en_revision,''),coalesce(encadenamiento_dias,0),coalesce(encadenamiento_umbral_meses,0),coalesce(encadenamiento_ventana_meses,0)
		FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1($1,$2)`, bolsaRef, corte.UTC())
	if err != nil {
		return nil, puertosbolsa.ErrPoliticaAvisosNoDisponible
	}
	defer filas.Close()
	marcas := make(map[string]dominiobolsa.MarcasParticipacion)
	for filas.Next() {
		var m dominiobolsa.MarcasParticipacion
		var dias int64
		var umbral, ventana int32
		if err := filas.Scan(&m.ParticipacionRef, &m.PrestaServicios, &m.EnRevision, &dias, &umbral, &ventana); err != nil || !marcaLeidaValida(m, dias) || len(marcas) >= maximoMarcasBolsa {
			return nil, puertosbolsa.ErrPoliticaAvisosNoDisponible
		}
		m.EncadenamientoDias, m.EncadenamientoUmbralMeses, m.EncadenamientoVentanaMeses = int(dias), int(umbral), int(ventana)
		marcas[m.ParticipacionRef] = m
	}
	if filas.Err() != nil {
		return nil, puertosbolsa.ErrPoliticaAvisosNoDisponible
	}
	return marcas, nil
}

func marcaLeidaValida(m dominiobolsa.MarcasParticipacion, dias int64) bool {
	switch m.PrestaServicios {
	case "", dominiobolsa.ModoPrestaServiciosAviso, dominiobolsa.ModoPrestaServiciosExcluir:
	default:
		return false
	}
	switch m.EnRevision {
	case "", dominiobolsa.RevisionRenunciaPendiente, dominiobolsa.RevisionSolicitudPendiente:
	default:
		return false
	}
	return m.ParticipacionRef != "" && dias >= 0 && dias < 1_000_000
}
