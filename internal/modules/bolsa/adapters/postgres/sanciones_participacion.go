package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// RepositorioSancionesParticipacionPostgreSQL usa solo funciones de las
// migraciones 000026 y 000037; no lee tablas directamente.
type RepositorioSancionesParticipacionPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioSancionesParticipacion = (*RepositorioSancionesParticipacionPostgreSQL)(nil)

func NuevoRepositorioSancionesParticipacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioSancionesParticipacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	return &RepositorioSancionesParticipacionPostgreSQL{pool}, nil
}

func (r *RepositorioSancionesParticipacionPostgreSQL) RegistrarSancion(ctx context.Context, cmd ports.ComandoRegistrarSancion) (ports.RegistroSancion, error) {
	s := cmd.Sancion
	if r == nil || r.pool == nil || ctx == nil || s.ParticipacionRef == "" || cmd.BolsaRef == "" || cmd.ClaveIdempotencia == "" || cmd.ReciboRef == "" ||
		!dominiobolsa.EfectoSancionValido(s.Efecto) || (s.Efecto == dominiobolsa.EfectoSancionNinguno) != (cmd.Operacion == nil) || cmd.Material.ValidarEstructura() != nil ||
		(cmd.FinAutomatico && (s.Efecto != dominiobolsa.OperacionPausar || s.SuspensionHasta == "")) || (cmd.OrdenFinal && s.Efecto == dominiobolsa.OperacionExcluir) {
		return ports.RegistroSancion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	fecha, ok := dominiobolsa.FechaCivilSancion(s.Datos.FechaNotificacion)
	vence, okVence := dominiobolsa.FechaCivilSancion(s.RecursoVence)
	if !ok || !okVence {
		return ports.RegistroSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	var hasta *time.Time
	if s.SuspensionHasta != "" {
		fin, ok := dominiobolsa.FechaCivilSancion(s.SuspensionHasta)
		if !ok {
			return ports.RegistroSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
		}
		hasta = &fin
	}
	var desde *time.Time
	if cmd.Operacion != nil {
		instante := cmd.Operacion.Cambio.Desde.UTC()
		desde = &instante
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.RegistroSancion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := cmd.Material
	var resultado ports.RegistroSancion
	var situacion *string
	err = tx.QueryRow(ctx, `SELECT reutilizada,sancion_ref,recibo_ref,situacion,desde,orden_final FROM vec_bolsa_llamamientos.registrar_sancion_participacion_v2($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29::numeric,$30::numeric,$31,$32,$33,$34)`,
		cmd.BolsaRef, s.ParticipacionRef, s.SancionRef, s.Consecuencia, s.ConsecuenciaEtiqueta,
		s.Efecto, s.Datos.Causa, fecha, s.Datos.Resolucion.Referencia, s.Datos.Resolucion.SHA256,
		s.Datos.ResueltaPor, s.ReglaRef, s.ReglaHuella, hasta, vence,
		s.RecursoReglaRef, s.RecursoReglaHuella, desde, s.Actor, cmd.ClaveIdempotencia,
		cmd.ReciboRef, s.RegistradaEn.UTC(), cmd.OrdenFinal, cmd.FinAutomatico,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&resultado.Reutilizada, &resultado.SancionRef, &resultado.ReciboRef, &situacion, &resultado.Desde, &resultado.OrdenFinal)
	if err != nil {
		return ports.RegistroSancion{}, errorSancion(err)
	}
	if situacion != nil {
		resultado.Situacion = *situacion
	}
	if resultado.SancionRef != s.SancionRef || resultado.ReciboRef != cmd.ReciboRef || resultado.OrdenFinal != cmd.OrdenFinal ||
		(cmd.Operacion != nil && resultado.Situacion != cmd.Operacion.Cambio.Destino) {
		return ports.RegistroSancion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.RegistroSancion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return resultado, nil
}

func (r *RepositorioSancionesParticipacionPostgreSQL) RegistrarRecursoSancion(ctx context.Context, cmd ports.ComandoRegistrarRecursoSancion) (ports.RegistroRecursoSancion, error) {
	e := cmd.Evento
	fecha, ok := dominiobolsa.FechaCivilSancion(e.Fecha)
	if r == nil || r.pool == nil || ctx == nil || cmd.ParticipacionRef == "" || cmd.SancionRef == "" || cmd.ClaveIdempotencia == "" || e.Actor == "" || !ok || cmd.Material.ValidarEstructura() != nil {
		return ports.RegistroRecursoSancion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	var documentoRef, documentoSHA *string
	if e.Documento != nil {
		documentoRef, documentoSHA = &e.Documento.Referencia, &e.Documento.SHA256
	}
	var resueltaPor, reglaRef, huella, motivo, recibo *string
	if rv := cmd.Reversion; rv != nil {
		if rv.ResueltaPor == "" || rv.ResueltaPor == e.Actor || rv.ReglaRef == "" || rv.Huella == "" || rv.Motivo == "" || rv.ReciboRef == "" || e.Documento == nil {
			return ports.RegistroRecursoSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
		}
		resueltaPor, reglaRef, huella, motivo, recibo = &rv.ResueltaPor, &rv.ReglaRef, &rv.Huella, &rv.Motivo, &rv.ReciboRef
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.RegistroRecursoSancion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := cmd.Material
	var resultado ports.RegistroRecursoSancion
	var reciboRev, situacion *string
	err = tx.QueryRow(ctx, `SELECT reutilizada,sancion_ref,estado,registrada_en,recibo_ref,situacion,desde FROM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20::numeric,$21::numeric,$22,$23,$24,$25)`,
		cmd.ParticipacionRef, cmd.SancionRef, e.Estado, fecha, documentoRef, documentoSHA, e.Actor, cmd.ClaveIdempotencia, e.RegistradaEn.UTC(),
		cmd.Reversion != nil, resueltaPor, reglaRef, huella, motivo, recibo,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&resultado.Reutilizada, &resultado.SancionRef, &resultado.Estado, &resultado.RegistradaEn, &reciboRev, &situacion, &resultado.Desde)
	if err != nil {
		return ports.RegistroRecursoSancion{}, errorSancion(err)
	}
	if reciboRev != nil {
		resultado.Revertida, resultado.ReciboRef = true, *reciboRev
	}
	if situacion != nil {
		resultado.Situacion = *situacion
	}
	if resultado.SancionRef != cmd.SancionRef || resultado.Estado != e.Estado || resultado.Revertida != (cmd.Reversion != nil) ||
		(cmd.Reversion != nil && resultado.ReciboRef != cmd.Reversion.ReciboRef) {
		return ports.RegistroRecursoSancion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.RegistroRecursoSancion{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return resultado, nil
}

type recursoSancionFila struct {
	Estado          string    `json:"estado"`
	Fecha           string    `json:"fecha"`
	DocumentoRef    *string   `json:"documento_ref"`
	DocumentoSHA256 *string   `json:"documento_sha256"`
	Actor           string    `json:"actor"`
	RegistradaEn    time.Time `json:"registrada_en"`
}

type reversionSancionFila struct {
	EstadoRecurso       string     `json:"estado_recurso"`
	ReglaRef            string     `json:"regla_ref"`
	EfectoRevertido     string     `json:"efecto_revertido"`
	SituacionRestaurada *string    `json:"situacion_restaurada"`
	SituacionDesde      *time.Time `json:"situacion_desde"`
	ResueltaPor         string     `json:"resuelta_por"`
	Actor               string     `json:"actor"`
	ReciboRef           string     `json:"recibo_ref"`
	RegistradaEn        time.Time  `json:"registrada_en"`
}

func (f reversionSancionFila) dominio() *dominiobolsa.ReversionSancion {
	r := &dominiobolsa.ReversionSancion{EstadoRecurso: f.EstadoRecurso, ReglaRef: f.ReglaRef, EfectoRevertido: f.EfectoRevertido,
		ResueltaPor: f.ResueltaPor, Actor: f.Actor, ReciboRef: f.ReciboRef, RegistradaEn: f.RegistradaEn.UTC()}
	if f.SituacionRestaurada != nil {
		r.SituacionRestaurada = *f.SituacionRestaurada
	}
	if f.SituacionDesde != nil {
		desde := f.SituacionDesde.UTC()
		r.SituacionDesde = &desde
	}
	return r
}

func (r *RepositorioSancionesParticipacionPostgreSQL) ListarSanciones(ctx context.Context, participacion, actor string, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]dominiobolsa.SancionParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || participacion == "" || actor == "" || m.ValidarEstructura() != nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	filas, err := tx.Query(ctx, `SELECT sancion_ref,consecuencia,consecuencia_etiqueta,efecto,causa,fecha_notificacion,resolucion_ref,resolucion_sha256,resuelta_por,regla_ref,regla_huella_sha256,suspension_hasta,recurso_vence,recurso_regla_ref,recurso_regla_huella_sha256,situacion_desde,actor,registrada_en,recursos,situacion_aplicada,fecha_disponible,orden_final,reversion FROM vec_bolsa_llamamientos.listar_sanciones_participacion_v2($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`,
		participacion, actor, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return nil, errorSancion(err)
	}
	defer filas.Close()
	sanciones := make([]dominiobolsa.SancionParticipacion, 0)
	for filas.Next() {
		var s dominiobolsa.SancionParticipacion
		var notificada, vence time.Time
		var hasta *time.Time
		var recursos, reversion []byte
		var situacion *string
		if err := filas.Scan(&s.SancionRef, &s.Consecuencia, &s.ConsecuenciaEtiqueta, &s.Efecto, &s.Datos.Causa, &notificada,
			&s.Datos.Resolucion.Referencia, &s.Datos.Resolucion.SHA256, &s.Datos.ResueltaPor, &s.ReglaRef, &s.ReglaHuella,
			&hasta, &vence, &s.RecursoReglaRef, &s.RecursoReglaHuella, &s.SituacionDesde, &s.Actor, &s.RegistradaEn, &recursos,
			&situacion, &s.FechaDisponible, &s.OrdenFinal, &reversion); err != nil {
			return nil, errorSancion(err)
		}
		if situacion != nil {
			s.SituacionAplicada = *situacion
		}
		if s.FechaDisponible != nil {
			fin := s.FechaDisponible.UTC()
			s.FechaDisponible = &fin
		}
		if len(reversion) > 0 {
			var fila reversionSancionFila
			if err := json.Unmarshal(reversion, &fila); err != nil {
				return nil, ports.ErrSituacionParticipacionNoDisponible
			}
			s.Reversion = fila.dominio()
		}
		s.ParticipacionRef, s.RegistradaEn = participacion, s.RegistradaEn.UTC()
		if s.SituacionDesde != nil {
			desde := s.SituacionDesde.UTC()
			s.SituacionDesde = &desde
		}
		s.Datos.Consecuencia = s.Consecuencia
		s.Datos.FechaNotificacion, s.RecursoVence = notificada.Format(time.DateOnly), vence.Format(time.DateOnly)
		if hasta != nil {
			s.SuspensionHasta = hasta.Format(time.DateOnly)
		}
		var eventos []recursoSancionFila
		if err := json.Unmarshal(recursos, &eventos); err != nil {
			return nil, ports.ErrSituacionParticipacionNoDisponible
		}
		for _, e := range eventos {
			evento := dominiobolsa.EventoRecursoSancion{Estado: e.Estado, Fecha: e.Fecha, Actor: e.Actor, RegistradaEn: e.RegistradaEn.UTC()}
			if e.DocumentoRef != nil && e.DocumentoSHA256 != nil {
				evento.Documento = &dominiobolsa.DocumentoSancion{Referencia: *e.DocumentoRef, SHA256: *e.DocumentoSHA256}
			}
			s.Recursos = append(s.Recursos, evento)
		}
		sanciones = append(sanciones, s)
	}
	if filas.Err() != nil {
		return nil, errorSancion(filas.Err())
	}
	filas.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, errorSancion(err)
	}
	return sanciones, nil
}

// errorSancion traduce como la situación y añade la clave reutilizada.
func errorSancion(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "VBS01" {
		return ports.ErrClaveOperacionReutilizada
	}
	return errorSituacionParticipacion(err)
}
