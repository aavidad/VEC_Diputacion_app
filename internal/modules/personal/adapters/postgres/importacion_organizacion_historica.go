package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const ejecutarImportacionOrganizacion = `SELECT vec_personal.ejecutar_importacion_organizacion_v1($1,$2::jsonb,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`
const ajustesImportacionOrganizacion = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`
const maxReciboImportacionOrganizacion = 16 << 10

type iniciadorImportacionOrganizacion interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type RepositorioImportacionOrganizacionPostgreSQL struct {
	pool iniciadorImportacionOrganizacion
}

var _ ports.RepositorioImportacionOrganizacion = (*RepositorioImportacionOrganizacionPostgreSQL)(nil)

func NuevoRepositorioImportacionOrganizacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioImportacionOrganizacionPostgreSQL, error) {
	return nuevoRepositorioImportacionOrganizacionPostgreSQL(pool)
}
func nuevoRepositorioImportacionOrganizacionPostgreSQL(pool iniciadorImportacionOrganizacion) (*RepositorioImportacionOrganizacionPostgreSQL, error) {
	if nuloImportacionOrganizacion(pool) {
		return nil, domain.ErrImportacionOrganizacionNoDisponible
	}
	return &RepositorioImportacionOrganizacionPostgreSQL{pool: pool}, nil
}

// SQL conserva el único efecto: consumo V3, CAS, historia, auditoría, outbox y
// recibo. También autoriza el replay antes de recuperar el recibo original.
func (r *RepositorioImportacionOrganizacionPostgreSQL) Ejecutar(ctx context.Context, o ports.OrdenImportacionOrganizacion) (ports.ReciboImportacionOrganizacion, error) {
	var vacio ports.ReciboImportacionOrganizacion
	if r == nil || ctx == nil || nuloImportacionOrganizacion(r.pool) {
		return vacio, domain.ErrImportacionOrganizacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	s := o.Material.Solicitud()
	reconstruido, err := domain.NuevoMaterialImportacionOrganizacion(s)
	if err != nil || !bytes.Equal(reconstruido.Canonico(), o.Material.Canonico()) || o.Material.HuellaSHA256() != reconstruido.HuellaSHA256() || s.RevisionEsperada == math.MaxInt64 || !autorizacionImportacionOrganizacionLigada(o.Material, o.Autorizacion) || !acreditacionImportacionOrganizacionLigada(s.Manifiesto, s.Fase, o.Acreditacion) {
		return vacio, domain.ErrImportacionOrganizacionInvalida
	}
	var acreditacion any
	if s.Fase == domain.FasePublicarOrganizacion {
		b, err := json.Marshal(o.Acreditacion)
		if err != nil {
			return vacio, domain.ErrImportacionOrganizacionInvalida
		}
		acreditacion = string(b)
	}
	a := o.Autorizacion
	parametros := []any{string(o.Material.Canonico()), acreditacion, a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), int64(a.PersonaVersion()), int64(a.PerfilVersion()), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, p := range parametros {
			if b, ok := p.([]byte); ok {
				for i := range b {
					b[i] = 0
				}
			}
		}
	}()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errorImportacionOrganizacion(ctx, err)
	}
	if tx == nil {
		return vacio, domain.ErrImportacionOrganizacionNoDisponible
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err = tx.Exec(ctx, ajustesImportacionOrganizacion); err != nil {
		return vacio, errorImportacionOrganizacion(ctx, err)
	}
	var bruto []byte
	if err = tx.QueryRow(ctx, ejecutarImportacionOrganizacion, parametros...).Scan(&bruto); err != nil {
		return vacio, errorImportacionOrganizacion(ctx, err)
	}
	recibo, err := decodificarReciboImportacionOrganizacion(bruto, o)
	if err != nil {
		return vacio, domain.ErrImportacionOrganizacionNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, errorImportacionOrganizacion(ctx, err)
	}
	confirmada = true
	return recibo, nil
}

func autorizacionImportacionOrganizacionLigada(m domain.MaterialImportacionOrganizacion, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if a.ValidarEstructura() != nil {
		return false
	}
	s, r, x := m.Solicitud(), m.Recurso(), a.ResumenCapacidad()
	h, err := r.HuellaContextoAutorizacionSHA256()
	return err == nil && a.PersonaVersion() == s.Actor.Instantanea.PersonaVersion && a.PerfilVersion() == s.Actor.Instantanea.PerfilVersion &&
		x.Operacion() == s.Fase.Accion() && x.AudienciaConsumo() == domain.AudienciaImportacionOrganizacion && x.EfectoRef() == r.Referencia && x.EfectoHuellaSHA256() == h
}

func acreditacionImportacionOrganizacionLigada(m domain.ManifiestoImportacionOrganizacion, fase domain.FaseImportacionOrganizacion, a ports.AcreditacionFuenteOrganizacionHistorica) bool {
	if fase != domain.FasePublicarOrganizacion {
		return reflect.ValueOf(a).IsZero()
	}
	huella, err := m.HuellaSHA256()
	return err == nil && a.ManifiestoHuellaSHA256 == huella && a.OrganismoRef == m.OrganismoRef && a.Tipo == m.Tipo && a.FuenteRef == m.FuenteRef &&
		a.FuenteVersion == m.FuenteVersion && a.FuenteHuellaSHA256 == m.FuenteHuellaSHA256 &&
		a.DiccionarioRef == m.DiccionarioRef && a.ActoRef == m.ActoRef && a.CustodiaRef == m.CustodiaRef &&
		referenciaImportacionOrganizacionValida(a.AcreditacionRef) && huellaOH.MatchString(a.AcreditacionHuellaSHA256) && instanteImportacionOrganizacionValido(a.AcreditadaEn)
}

func decodificarReciboImportacionOrganizacion(bruto []byte, o ports.OrdenImportacionOrganizacion) (ports.ReciboImportacionOrganizacion, error) {
	var vacio ports.ReciboImportacionOrganizacion
	if len(bruto) == 0 || len(bruto) > maxReciboImportacionOrganizacion || bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) || verificarJSONOrganizacionHistorica(bruto) != nil {
		return vacio, errors.New("recibo inválido")
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(bruto, &campos) != nil || !clavesOH(campos, []string{"recibo_ref", "lote_ref", "fase", "estado", "revision_anterior", "revision_nueva", "clave_idempotencia", "material_huella_sha256", "fuente_huella_sha256", "actor_ref", "decision_ref", "auditoria_ref", "registrado_en", "replay"}, nil) {
		return vacio, errors.New("recibo inválido")
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	var r ports.ReciboImportacionOrganizacion
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF {
		return vacio, errors.New("recibo inválido")
	}
	s, x := o.Material.Solicitud(), o.Autorizacion.ResumenCapacidad()
	if !referenciaImportacionOrganizacionValida(r.ReciboRef) || !referenciaImportacionOrganizacionValida(r.LoteRef) ||
		r.Fase != s.Fase || r.RevisionAnterior != s.RevisionEsperada || r.RevisionNueva != s.RevisionEsperada+1 ||
		r.ClaveIdempotencia != s.ClaveIdempotencia || r.MaterialHuellaSHA256 != o.Material.HuellaSHA256() ||
		r.FuenteHuellaSHA256 != s.Manifiesto.FuenteHuellaSHA256 || r.ActorRef != s.Actor.Principal.ID ||
		!referenciaImportacionOrganizacionValida(r.DecisionRef) || !referenciaImportacionOrganizacionValida(r.AuditoriaRef) ||
		!instanteImportacionOrganizacionValido(r.RegistradoEn) ||
		(!r.Replay && (r.DecisionRef != x.DecisionRef() || r.RegistradoEn.Before(x.EmitidaEn()) || !r.RegistradoEn.Before(x.ExpiraEn()))) {
		return vacio, errors.New("recibo incompatible")
	}
	switch s.Fase {
	case domain.FasePrepararOrganizacion:
		if r.Estado != "preparacion_no_autoritativa" {
			return vacio, errors.New("estado incompatible")
		}
	case domain.FaseConciliarOrganizacion:
		if r.LoteRef != s.LoteRef || r.Estado != "conciliacion_pendiente" && r.Estado != "conciliada" {
			return vacio, errors.New("estado incompatible")
		}
	case domain.FasePublicarOrganizacion:
		if r.LoteRef != s.LoteRef || r.Estado != "publicada" {
			return vacio, errors.New("estado incompatible")
		}
	default:
		return vacio, errors.New("fase incompatible")
	}
	return r, nil
}

func referenciaImportacionOrganizacionValida(v string) bool {
	if len(v) < 3 || len(v) > 256 {
		return false
	}
	for _, c := range v {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}
func instanteImportacionOrganizacionValido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}
func nuloImportacionOrganizacion(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}
func errorImportacionOrganizacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return domain.ErrImportacionOrganizacionDenegada
		case "22023":
			return domain.ErrImportacionOrganizacionInvalida
		case "23505", "P0112":
			return domain.ErrImportacionOrganizacionConflicto
		}
	}
	return domain.ErrImportacionOrganizacionNoDisponible
}
