package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const consultarPublicacionesOficialesSQL = `SELECT vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1($1::text,$2::integer)`
const normalizarPublicacionOficialSQL = `SELECT p::text, encode(sha256(convert_to(p::text,'UTF8')),'hex') FROM (SELECT $1::jsonb AS p) s`
const declararPublicacionOficialSQL = `SELECT vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

var referenciaPublicacionOficialBases = regexp.MustCompile(`^publicacion_bases:[0-9a-f]{64}$`)
var errCommitPublicacionOficialIndeterminado = errors.New("commit de la publicacion oficial indeterminado")

// RepositorioPublicacionOficialPostgreSQL usa el pool de Bolsa RRHH; BC15
// comprueba esa sesión y solo le concede sus dos funciones.
type RepositorioPublicacionOficialPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioPublicacionOficial = (*RepositorioPublicacionOficialPostgreSQL)(nil)

func NuevoRepositorioPublicacionOficialPostgreSQL(pool *pgxpool.Pool) (*RepositorioPublicacionOficialPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrPublicacionOficialNoDisponible
	}
	return &RepositorioPublicacionOficialPostgreSQL{pool}, nil
}

func (r *RepositorioPublicacionOficialPostgreSQL) ConsultarVersiones(ctx context.Context, ref string, limite int) (ports.PaginaVersionesPublicacionOficial, error) {
	var cero ports.PaginaVersionesPublicacionOficial
	if r == nil || r.pool == nil || ctx == nil || limite < 1 || limite > ports.LimiteConsultaPublicacionesOficialesBases {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	var selector *string
	if ref != "" {
		selector = &ref
	}
	plazo, cancelar := context.WithTimeout(ctx, 15*time.Second)
	defer cancelar()
	var b []byte
	if err := r.pool.QueryRow(plazo, consultarPublicacionesOficialesSQL, selector, limite).Scan(&b); err != nil {
		return cero, errorPublicacionOficialSQL(err)
	}
	if len(b) == 0 || len(b) > 1<<20 {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	var pagina ports.PaginaVersionesPublicacionOficial
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&pagina) != nil || !errors.Is(d.Decode(new(any)), io.EOF) || pagina.Items == nil ||
		len(pagina.Items) > limite || (ref != "" && len(pagina.Items) != 1) {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	for _, v := range pagina.Items {
		if v.ConvocatoriaRef == "" || v.Secuencia < 1 || len(v.HuellaVersionSHA256) != 64 ||
			(v.Publicacion != nil && !referenciaPublicacionOficialBases.MatchString(v.Publicacion.PublicacionRef)) {
			return cero, ports.ErrPublicacionOficialNoDisponible
		}
	}
	return pagina, nil
}

func (r *RepositorioPublicacionOficialPostgreSQL) NormalizarDeclaracion(ctx context.Context, d ports.DeclaracionPublicacionOficial) (string, string, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return "", "", ports.ErrPublicacionOficialNoDisponible
	}
	b, err := json.Marshal(d)
	if err != nil || len(b) > ports.MaximoSolicitudPublicacionOficialBasesJSON {
		return "", "", ports.ErrPublicacionOficialInvalida
	}
	plazo, cancelar := context.WithTimeout(ctx, 15*time.Second)
	defer cancelar()
	var canon, huella string
	if err := r.pool.QueryRow(plazo, normalizarPublicacionOficialSQL, string(b)).Scan(&canon, &huella); err != nil {
		return "", "", errorPublicacionOficialSQL(err)
	}
	sum := sha256.Sum256([]byte(canon))
	if hex.EncodeToString(sum[:]) != huella || len(canon) > ports.MaximoSolicitudPublicacionOficialBasesJSON {
		return "", "", ports.ErrPublicacionOficialNoDisponible
	}
	return canon, huella, nil
}

func (r *RepositorioPublicacionOficialPostgreSQL) Declarar(ctx context.Context, i ports.IntencionPublicacionOficial) (ports.ReciboPublicacionOficial, error) {
	var cero ports.ReciboPublicacionOficial
	if r == nil || r.pool == nil || ctx == nil || i.CanonPostgreSQL == "" || i.Material.ValidarEstructura() != nil {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	sum := sha256.Sum256([]byte(i.CanonPostgreSQL))
	if hex.EncodeToString(sum[:]) != i.HuellaSHA256 || !mismaDeclaracionPublicacionOficial(i) {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	return declararPublicacionOficialConRecuperacion(ctx, i, r.declararUnaVez)
}

func declararPublicacionOficialConRecuperacion(ctx context.Context, i ports.IntencionPublicacionOficial,
	ejecutar func(context.Context, ports.IntencionPublicacionOficial) (ports.ReciboPublicacionOficial, error),
) (ports.ReciboPublicacionOficial, error) {
	var cero ports.ReciboPublicacionOficial
	var err error
	for intento := 0; intento < 3; intento++ {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		var recibo ports.ReciboPublicacionOficial
		if recibo, err = ejecutar(ctx, i); err == nil {
			return recibo, nil
		}
		if errors.Is(err, errCommitPublicacionOficialIndeterminado) {
			// Un COMMIT sin respuesta puede haber escrito. Con la misma clave y
			// el mismo material BC14 devuelve el recibo original.
			limite, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
			recuperado, fallo := ejecutar(limite, i)
			cancelar()
			if fallo != nil || !reciboPublicacionOficialValido(recuperado, i.Solicitud.Declaracion) {
				if fallo == nil {
					fallo = ports.ErrPublicacionOficialNoDisponible
				}
				return cero, fmt.Errorf("%w: %w", ports.ErrPublicacionOficialNoConfirmada, errors.Join(err, fallo))
			}
			return recuperado, nil
		}
		if errors.Is(err, errRollbackComprobacion) || !reintentarComprobacion(err) {
			return cero, errorPublicacionOficialSQL(err)
		}
	}
	return cero, errorPublicacionOficialSQL(err)
}

func (r *RepositorioPublicacionOficialPostgreSQL) declararUnaVez(ctx context.Context, i ports.IntencionPublicacionOficial) (out ports.ReciboPublicacionOficial, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return out, err
	}
	confirmado := false
	defer func() {
		if confirmado {
			return
		}
		c, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancelar()
		if fallo := tx.Rollback(c); fallo != nil && !errors.Is(fallo, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("%w: %w", errRollbackComprobacion, fallo))
		}
	}()
	if _, err = tx.Exec(ctx, ajustesComprobacionSQL); err != nil {
		return out, err
	}
	m := i.Material
	var b []byte
	err = tx.QueryRow(ctx, declararPublicacionOficialSQL, i.CanonPostgreSQL, m.CapacidadCanonica(), m.DecisionCanonica(),
		m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(),
		m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&b)
	if err != nil {
		return out, err
	}
	if len(b) == 0 || len(b) > 4096 {
		return out, ports.ErrPublicacionOficialNoDisponible
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 14 {
		return out, ports.ErrPublicacionOficialNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&out) != nil || !errors.Is(d.Decode(new(any)), io.EOF) || !reciboPublicacionOficialValido(out, i.Solicitud.Declaracion) {
		return ports.ReciboPublicacionOficial{}, ports.ErrPublicacionOficialNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		if commitComprobacionIndeterminado(err) {
			return ports.ReciboPublicacionOficial{}, fmt.Errorf("%w: %w", errCommitPublicacionOficialIndeterminado, err)
		}
		return ports.ReciboPublicacionOficial{}, err
	}
	confirmado = true
	return out, nil
}

// El canon de PostgreSQL debe decir lo mismo que la declaración validada.
func mismaDeclaracionPublicacionOficial(i ports.IntencionPublicacionOficial) bool {
	b, err := json.Marshal(i.Solicitud.Declaracion)
	var enviada, normalizada any
	return err == nil && json.Unmarshal(b, &enviada) == nil && json.Unmarshal([]byte(i.CanonPostgreSQL), &normalizada) == nil &&
		reflect.DeepEqual(enviada, normalizada)
}

func reciboPublicacionOficialValido(r ports.ReciboPublicacionOficial, d ports.DeclaracionPublicacionOficial) bool {
	return referenciaPublicacionOficialBases.MatchString(r.PublicacionRef) &&
		r.ReciboRef == "recibo:"+r.PublicacionRef && r.EventoRef == "evento:"+r.PublicacionRef &&
		r.ConvocatoriaRef == d.ConvocatoriaRef && r.Secuencia >= 1 && r.Medio == d.Medio &&
		reflect.DeepEqual(r.BoletinNumero, d.BoletinNumero) && r.AnuncioRef == d.AnuncioRef &&
		r.FechaPublicacion == d.FechaPublicacion && reflect.DeepEqual(r.Direccion, d.Direccion) &&
		reflect.DeepEqual(r.SustituyeRef, d.SustituyeRef) &&
		r.AuditoriaRef != "" && len(r.AuditoriaRef) <= 256 && !r.RegistradaEn.IsZero()
}

func errorPublicacionOficialSQL(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errRollbackComprobacion) {
		return fmt.Errorf("%w: %w", ports.ErrPublicacionOficialNoDisponible, err)
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return fmt.Errorf("%w: %w", vd.ErrAutorizacionDenegada, err)
		case "22023", "22007", "22008":
			return fmt.Errorf("%w: %w", ports.ErrPublicacionOficialInvalida, err)
		case "B9601":
			return fmt.Errorf("%w: %w", ports.ErrPublicacionOficialNoVigente, err)
		case "23505":
			return fmt.Errorf("%w: %w", ports.ErrPublicacionOficialConflicto, err)
		}
	}
	if errors.Is(err, ports.ErrPublicacionOficialInvalida) || errors.Is(err, ports.ErrPublicacionOficialNoConfirmada) {
		return err
	}
	return fmt.Errorf("%w: %w", ports.ErrPublicacionOficialNoDisponible, err)
}
