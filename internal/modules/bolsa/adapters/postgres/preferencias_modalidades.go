package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"regexp"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
)

const funcionGuardarPreferenciasModalidades = "vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1"

var modalidadCanonicaPG = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)
var reciboModalidadesPG = regexp.MustCompile(`^recibo:preferencias-modalidades:[0-9a-f]{64}$`)

type EscrituraPreferenciasModalidadesPostgreSQL struct{ pool iniciadorTransacciones }
type LecturaPreferenciasModalidadesPostgreSQL struct {
	pool    iniciadorTransacciones
	funcion string
}

var _ bolsa.EscrituraPreferenciasModalidades = (*EscrituraPreferenciasModalidadesPostgreSQL)(nil)
var _ bolsa.LecturaPreferenciasModalidades = (*LecturaPreferenciasModalidadesPostgreSQL)(nil)

func NuevaEscrituraPreferenciasModalidadesPostgreSQL(pool *pgxpool.Pool) (*EscrituraPreferenciasModalidadesPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &EscrituraPreferenciasModalidadesPostgreSQL{pool: pool}, nil
}

func NuevaEscrituraPreferenciasModalidadesInternaPostgreSQL(pool *pgxpool.Pool) (*EscrituraPreferenciasModalidadesPostgreSQL, error) {
	return NuevaEscrituraPreferenciasModalidadesPostgreSQL(pool)
}

func NuevaLecturaPreferenciasModalidadesPostgreSQL(pool *pgxpool.Pool) (*LecturaPreferenciasModalidadesPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &LecturaPreferenciasModalidadesPostgreSQL{pool: pool, funcion: funcionConsultarPreferenciasBolsas}, nil
}

func NuevaLecturaPreferenciasModalidadesInternaPostgreSQL(pool *pgxpool.Pool) (*LecturaPreferenciasModalidadesPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &LecturaPreferenciasModalidadesPostgreSQL{pool: pool, funcion: funcionConsultarPreferenciasBolsasInterna}, nil
}

type modalidadSnapshotSQL struct {
	Clave                string `json:"clave"`
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      int    `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
	EntradaClave         string `json:"entrada_clave"`
}

func snapshotModalidadesSQL(disponibles []bolsa.ModalidadPreferencia) ([]byte, error) {
	opciones := make([]modalidadSnapshotSQL, 0, len(disponibles))
	for _, d := range disponibles {
		opciones = append(opciones, modalidadSnapshotSQL{d.Clave, d.Fuente.CatalogoID,
			d.Fuente.CatalogoVersion, d.Fuente.CatalogoHuellaSHA256, d.Fuente.EntradaClave})
	}
	return json.Marshal(opciones)
}

func (a *EscrituraPreferenciasModalidadesPostgreSQL) GuardarPreferenciasModalidades(ctx context.Context,
	s bolsa.GuardarPreferenciasModalidades,
) (bolsa.ReciboPreferenciasModalidades, error) {
	vacia := bolsa.ReciboPreferenciasModalidades{}
	if ctx == nil || a == nil || valorNulo(a.pool) || s.CandidatoRef == "" ||
		s.ConsultadaEn.IsZero() || s.Material.ValidarEstructura() != nil ||
		s.VersionEsperada >= 2147483647 || s.ClaveIdempotencia == "" {
		return vacia, bolsa.ErrOrdenPreferenciasInvalido
	}
	if _, err := bolsaapp.HuellaSolicitudPreferenciasModalidades(s.CandidatoRef, s.VersionEsperada,
		s.Modalidades, s.Disponibles); err != nil {
		return vacia, err
	}
	snapshot, err := snapshotModalidadesSQL(s.Disponibles)
	if err != nil {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	var recibo bolsa.ReciboPreferenciasModalidades
	err = postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var fallo error
		recibo, fallo = a.guardarIntento(ctx, s, snapshot)
		return fallo
	})
	if err != nil {
		return vacia, errorPreferenciasBolsas(ctx, err)
	}
	return recibo, nil
}

func configurarTransaccionModalidades(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
 set_config('row_security','on',true),set_config('timezone','UTC',true),
 set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
 set_config('idle_in_transaction_session_timeout','20s',true)`)
	return err
}

func (a *EscrituraPreferenciasModalidadesPostgreSQL) guardarIntento(ctx context.Context,
	s bolsa.GuardarPreferenciasModalidades, snapshot []byte,
) (bolsa.ReciboPreferenciasModalidades, error) {
	vacia := bolsa.ReciboPreferenciasModalidades{}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, err
	}
	defer revertir(tx)
	if err = configurarTransaccionModalidades(ctx, tx); err != nil {
		return vacia, err
	}
	m := s.Material
	var version int64
	var recibo bolsa.ReciboPreferenciasModalidades
	err = tx.QueryRow(ctx, `SELECT * FROM `+funcionGuardarPreferenciasModalidades+`(
 $1::text,$2::text[],$3::jsonb,$4::bigint,$5::text,$6::timestamptz,
 $7::bytea,$8::bytea,$9::bytea,$10::bytea,$11::numeric,$12::numeric,
 $13::bytea,$14::bytea,$15::bytea,$16::bytea)`,
		s.CandidatoRef, s.Modalidades, snapshot, int64(s.VersionEsperada),
		s.ClaveIdempotencia, s.ConsultadaEn.UTC(), m.CapacidadCanonica(), m.DecisionCanonica(),
		m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()),
		int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(
		&recibo.Reutilizado, &recibo.ReciboRef, &version, &recibo.Modalidades, &recibo.RegistradoEn)
	if err != nil {
		return vacia, err
	}
	canonicas := slices.Clone(s.Modalidades)
	if canonicas != nil {
		slices.Sort(canonicas)
	}
	if version != int64(s.VersionEsperada)+1 || !reciboModalidadesPG.MatchString(recibo.ReciboRef) ||
		recibo.RegistradoEn.IsZero() || (recibo.Modalidades == nil) != (canonicas == nil) ||
		!slices.Equal(recibo.Modalidades, canonicas) {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	recibo.Version = uint64(version)
	if err = ctx.Err(); err != nil {
		return vacia, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacia, err
	}
	return recibo, nil
}

func (a *LecturaPreferenciasModalidadesPostgreSQL) LeerPreferenciasModalidades(ctx context.Context,
	c bolsa.CapturaLecturaPreferencias,
) (bolsa.EstadoPreferenciasModalidades, error) {
	vacia := bolsa.EstadoPreferenciasModalidades{}
	if ctx == nil || a == nil || valorNulo(a.pool) ||
		(a.funcion != funcionConsultarPreferenciasBolsas && a.funcion != funcionConsultarPreferenciasBolsasInterna) ||
		c.CandidatoRef() == "" || c.CorrelacionRef() == "" || len(c.ContextoCanonico()) == 0 ||
		len(c.VinculoCanonico()) == 0 || len(c.OrdenJSON()) == 0 || !c.ValidaHasta().After(time.Now().UTC()) {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	var estado bolsa.EstadoPreferenciasModalidades
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var fallo error
		estado, fallo = a.leerIntento(ctx, c)
		return fallo
	})
	if err != nil {
		return vacia, errorLecturaPreferencias(ctx, err)
	}
	return estado, nil
}

func (a *LecturaPreferenciasModalidadesPostgreSQL) leerIntento(ctx context.Context,
	c bolsa.CapturaLecturaPreferencias,
) (bolsa.EstadoPreferenciasModalidades, error) {
	vacia := bolsa.EstadoPreferenciasModalidades{}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, err
	}
	defer revertir(tx)
	if err = configurarTransaccionModalidades(ctx, tx); err != nil {
		return vacia, err
	}
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT `+a.funcion+`($1::text,$2::bytea,$3::bytea,$4::jsonb)`,
		c.CandidatoRef(), c.ContextoCanonico(), c.VinculoCanonico(), []byte(c.OrdenJSON())).Scan(&contenido)
	if err != nil {
		return vacia, err
	}
	defer borrarBytesPostgreSQL(contenido)
	estado, err := decodificarLecturaModalidades(contenido, c.CorrelacionRef())
	if err != nil {
		return vacia, err
	}
	if err = ctx.Err(); err != nil {
		return vacia, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacia, err
	}
	return estado, nil
}

func decodificarLecturaModalidades(contenido []byte, correlacion string) (bolsa.EstadoPreferenciasModalidades, error) {
	vacia := bolsa.EstadoPreferenciasModalidades{}
	if len(contenido) == 0 || len(contenido) > 131072 ||
		validarClavesJSONPreferencias(contenido, []string{"estado", "auditoria_ref", "auditoria_huella_sha256", "correlacion_ref", "auditada_en"}) != nil {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	var envoltura struct {
		Estado json.RawMessage `json:"estado"`
	}
	if json.Unmarshal(contenido, &envoltura) != nil ||
		validarClavesJSONPreferencias(envoltura.Estado, bolsa.CamposLecturaPreferenciasModalidades()) != nil {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	var p struct {
		Estado struct {
			Version      uint64          `json:"version"`
			Guardadas    []string        `json:"modalidades_guardadas"`
			Disponibles  json.RawMessage `json:"modalidades_disponibles"`
			ReciboRef    *string         `json:"recibo_ref"`
			RegistradoEn *time.Time      `json:"registrado_en"`
		} `json:"estado"`
		AuditoriaRef          string    `json:"auditoria_ref"`
		AuditoriaHuellaSHA256 string    `json:"auditoria_huella_sha256"`
		CorrelacionRef        string    `json:"correlacion_ref"`
		AuditadaEn            time.Time `json:"auditada_en"`
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&p) != nil || dec.Decode(&struct{}{}) != io.EOF ||
		!auditoriaLecturaPreferenciasRef.MatchString(p.AuditoriaRef) ||
		!auditoriaLecturaPreferenciasHuella.MatchString(p.AuditoriaHuellaSHA256) ||
		p.CorrelacionRef != correlacion || p.AuditadaEn.IsZero() ||
		p.AuditadaEn.After(time.Now().UTC().Add(time.Second)) ||
		!bytes.Equal(bytes.TrimSpace(p.Estado.Disponibles), []byte("[]")) {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	e := p.Estado
	if e.Version > 2147483647 ||
		e.Version == 0 && (e.Guardadas != nil || e.ReciboRef != nil || e.RegistradoEn != nil) ||
		e.Version > 0 && (e.ReciboRef == nil || !reciboModalidadesPG.MatchString(*e.ReciboRef) || e.RegistradoEn == nil ||
			e.RegistradoEn.IsZero() || e.RegistradoEn.After(p.AuditadaEn)) {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	for i, clave := range e.Guardadas {
		if !modalidadCanonicaPG.MatchString(clave) || i > 0 && e.Guardadas[i-1] >= clave {
			return vacia, bolsa.ErrPreferenciasNoDisponibles
		}
	}
	vacia.Version, vacia.ModalidadesGuardadas, vacia.RegistradoEn = e.Version, e.Guardadas, e.RegistradoEn
	if e.ReciboRef != nil {
		vacia.ReciboRef = *e.ReciboRef
	}
	vacia.ModalidadesDisponibles = []bolsa.ModalidadPreferencia{}
	return vacia, nil
}
