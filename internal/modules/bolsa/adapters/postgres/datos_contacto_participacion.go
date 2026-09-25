package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// RepositorioDatosContactoParticipacionPostgreSQL (B4) guarda el sobre cifrado
// de los datos de contacto; nunca recibe ni devuelve el claro.
type RepositorioDatosContactoParticipacionPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioDatosContactoParticipacion = (*RepositorioDatosContactoParticipacionPostgreSQL)(nil)

func NuevoRepositorioDatosContactoParticipacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioDatosContactoParticipacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	return &RepositorioDatosContactoParticipacionPostgreSQL{pool}, nil
}

func (r *RepositorioDatosContactoParticipacionPostgreSQL) DatosContactoVigentes(ctx context.Context, ref string) (ports.RegistroDatosContactoParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	return r.leer(ctx, ref, `SELECT version,clave_ref,nonce,cifrado,motivo,registrada_en,recibo_ref FROM vec_bolsa_llamamientos.leer_datos_contacto_participacion_v1($1)`, ref)
}

func (r *RepositorioDatosContactoParticipacionPostgreSQL) BuscarRegistroDatosContacto(ctx context.Context, ref, clave string) (ports.RegistroDatosContactoParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" || clave == "" {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	registro, err := r.leer(ctx, ref, `SELECT version,clave_ref,nonce,cifrado,motivo,registrada_en,recibo_ref FROM vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v1($1,$2)`, ref, clave)
	if err != nil {
		return ports.RegistroDatosContactoParticipacion{}, err
	}
	registro.Reutilizada = true
	return registro, nil
}

func (r *RepositorioDatosContactoParticipacionPostgreSQL) leer(ctx context.Context, ref, consulta string, args ...any) (ports.RegistroDatosContactoParticipacion, error) {
	var registro ports.RegistroDatosContactoParticipacion
	registro.ParticipacionRef = ref
	var version int64
	err := r.pool.QueryRow(ctx, consulta, args...).Scan(&version, &registro.Sobre.ClaveRef, &registro.Sobre.Nonce, &registro.Sobre.Cifrado, &registro.Motivo, &registro.RegistradaEn, &registro.ReciboRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoEncontrados
	}
	if err != nil {
		return ports.RegistroDatosContactoParticipacion{}, errorDatosContactoParticipacion(err)
	}
	if version <= 0 {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	registro.Version = uint64(version)
	registro.Sobre.Version = registro.Version
	registro.RegistradaEn = registro.RegistradaEn.UTC()
	if registro.Sobre.Validar() != nil {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	return registro, nil
}

func (r *RepositorioDatosContactoParticipacionPostgreSQL) RegistrarDatosContacto(ctx context.Context, comando ports.ComandoRegistrarDatosContactoParticipacion) (ports.RegistroDatosContactoParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || comando.ParticipacionRef == "" || comando.BolsaRef == "" || comando.Sobre.Validar() != nil ||
		comando.Motivo == "" || comando.Actor == "" || comando.RegistradaEn.IsZero() || comando.ClaveIdempotencia == "" || comando.ReciboRef == "" ||
		comando.Material.ValidarEstructura() != nil {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	defer tx.Rollback(context.Background())
	m := comando.Material
	// La traza de valores (000034) recibe en esta misma transacción qué
	// campos cambiaron; la base nunca ve sus valores.
	if len(comando.CamposCambiados) > 0 {
		if _, err = tx.Exec(ctx, `SELECT set_config('vec_bolsa.campos_contacto_cambiados',$1,true)`, strings.Join(comando.CamposCambiados, ",")); err != nil {
			return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
		}
	}
	var registro ports.RegistroDatosContactoParticipacion
	var version int64
	err = tx.QueryRow(ctx, `SELECT reutilizada,recibo_ref,version,registrada_en FROM vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21)`,
		comando.BolsaRef, comando.ParticipacionRef, int64(comando.Sobre.Version), comando.Sobre.ClaveRef, comando.Sobre.Nonce, comando.Sobre.Cifrado,
		comando.Motivo, comando.Actor, comando.RegistradaEn.UTC(), comando.ClaveIdempotencia, comando.ReciboRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&registro.Reutilizada, &registro.ReciboRef, &version, &registro.RegistradaEn)
	if err != nil {
		return ports.RegistroDatosContactoParticipacion{}, errorDatosContactoParticipacion(err)
	}
	if registro.ReciboRef != comando.ReciboRef || version <= 0 || (!registro.Reutilizada && uint64(version) != comando.Sobre.Version) {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.RegistroDatosContactoParticipacion{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	if registro.Reutilizada {
		return r.BuscarRegistroDatosContacto(ctx, comando.ParticipacionRef, comando.ClaveIdempotencia)
	}
	registro.ParticipacionRef, registro.Version, registro.Motivo = comando.ParticipacionRef, uint64(version), comando.Motivo
	registro.RegistradaEn = registro.RegistradaEn.UTC()
	registro.Sobre = comando.Sobre
	return registro, nil
}

func errorDatosContactoParticipacion(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return ports.ErrDatosContactoParticipacionNoEncontrados
		case "VBS02", "22023":
			return dominiobolsa.ErrDatosContactoParticipacionInvalidos
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		}
	}
	return ports.ErrDatosContactoParticipacionNoDisponibles
}
