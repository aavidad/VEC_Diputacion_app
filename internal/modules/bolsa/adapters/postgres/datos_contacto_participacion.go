package postgres

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
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
	if registro.Origen, err = r.leerOrigen(ctx, ref, version); err != nil {
		return ports.RegistroDatosContactoParticipacion{}, err
	}
	return registro, nil
}

// leerOrigen devuelve la marca de origen CONVOCA de esa versión, o nil si el
// contacto es propio.
func (r *RepositorioDatosContactoParticipacionPostgreSQL) leerOrigen(ctx context.Context, ref string, version int64) (*dominiobolsa.MarcaOrigenDatosContacto, error) {
	var marca dominiobolsa.MarcaOrigenDatosContacto
	var ultimoDia time.Time
	err := r.pool.QueryRow(ctx, `SELECT origen,vigente_hasta,ultimo_dia,regla_ref,regla_huella_sha256 FROM vec_bolsa_llamamientos.leer_origen_datos_contacto_participacion_v1($1,$2)`, ref, version).
		Scan(&marca.Origen, &marca.VigenteHasta, &ultimoDia, &marca.ReglaRef, &marca.ReglaHuella)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	marca.VigenteHasta, marca.UltimoDia = marca.VigenteHasta.UTC(), ultimoDia.Format(time.DateOnly)
	if marca.Validar() != nil {
		return nil, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	// Una versión confirmada por la persona (Bolsa 000040) cuenta como propia.
	var confirmada *time.Time
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.leer_confirmacion_contacto_participacion_v1($1,$2)`, ref, version).Scan(&confirmada); err != nil {
		return nil, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	if confirmada != nil {
		return nil, nil
	}
	return &marca, nil
}

func (r *RepositorioDatosContactoParticipacionPostgreSQL) RegistrarDatosContacto(ctx context.Context, comando ports.ComandoRegistrarDatosContactoParticipacion) (ports.RegistroDatosContactoParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || comando.ParticipacionRef == "" || comando.BolsaRef == "" || comando.Sobre.Validar() != nil ||
		comando.Motivo == "" || comando.Actor == "" || comando.RegistradaEn.IsZero() || comando.ClaveIdempotencia == "" || comando.ReciboRef == "" ||
		comando.Material.ValidarEstructura() != nil || (comando.Origen != nil && comando.Origen.Validar() != nil) {
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
	argumentos := []any{
		comando.BolsaRef, comando.ParticipacionRef, int64(comando.Sobre.Version), comando.Sobre.ClaveRef, comando.Sobre.Nonce, comando.Sobre.Cifrado,
		comando.Motivo, comando.Actor, comando.RegistradaEn.UTC(), comando.ClaveIdempotencia, comando.ReciboRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	}
	consulta := `SELECT reutilizada,recibo_ref,version,registrada_en FROM vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21)`
	if o := comando.Origen; o != nil {
		// Duda 45: la versión y su marca de origen CONVOCA en la misma llamada.
		consulta = `SELECT o_reutilizada,o_recibo_ref,o_version,o_registrada_en FROM vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21,$22,$23,$24::date,$25,$26)`
		argumentos = append(argumentos, o.Origen, o.VigenteHasta.UTC(), o.UltimoDia, o.ReglaRef, o.ReglaHuella)
	}
	err = tx.QueryRow(ctx, consulta, argumentos...).Scan(&registro.Reutilizada, &registro.ReciboRef, &version, &registro.RegistradaEn)
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
	if comando.Origen != nil {
		marca := *comando.Origen
		registro.Origen = &marca
	}
	return registro, nil
}

// ConsultarDatosContactoAutorizados (B78) consume la decisión de consulta
// completa (AD197) y lee la versión vigente, su origen y su confirmación en una
// sola transacción SERIALIZABLE. Devuelve el sobre cifrado: el claro solo
// existe en el caso de uso, que lo descifra en entregar antes del COMMIT.
func (r *RepositorioDatosContactoParticipacionPostgreSQL) ConsultarDatosContactoAutorizados(ctx context.Context, bolsa, participacion, actor string, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, entregar func(ports.LecturaDatosContactoAutorizada) error) (ports.LecturaDatosContactoAutorizada, error) {
	if r == nil || r.pool == nil || ctx == nil || bolsa == "" || participacion == "" || actor == "" || m.ValidarEstructura() != nil || entregar == nil {
		return ports.LecturaDatosContactoAutorizada{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.LecturaDatosContactoAutorizada{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, configuracionTransaccionPortal); err != nil {
		return ports.LecturaDatosContactoAutorizada{}, errorConsultaDatosContacto(err)
	}
	var (
		lectura                  ports.LecturaDatosContactoAutorizada
		version                  int64
		origen, reglaRef, huella *string
		vigenteHasta, confirmada *time.Time
		ultimoDia                *time.Time
	)
	registro := &lectura.Registro
	err = tx.QueryRow(ctx, `SELECT version,clave_ref,nonce,cifrado,registrada_en,recibo_ref,origen,vigente_hasta,ultimo_dia,regla_ref,regla_huella_sha256,confirmada_en,decision_ref,auditoria_ref,consumida_en FROM vec_bolsa_llamamientos.consultar_datos_contacto_participacion_rrhh_v1($1::text,$2::text,$3::text,$4::bytea,$5::bytea,$6::bytea,$7::bytea,$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`,
		bolsa, participacion, actor, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).
		Scan(&version, &registro.Sobre.ClaveRef, &registro.Sobre.Nonce, &registro.Sobre.Cifrado, &registro.RegistradaEn, &registro.ReciboRef,
			&origen, &vigenteHasta, &ultimoDia, &reglaRef, &huella, &confirmada, &lectura.DecisionRef, &lectura.AuditoriaRef, &lectura.ConsumidaEn)
	if err != nil {
		return ports.LecturaDatosContactoAutorizada{}, errorConsultaDatosContacto(err)
	}
	if version <= 0 || lectura.DecisionRef == "" || !auditoriaRefConsumoV3.MatchString(lectura.AuditoriaRef) || lectura.ConsumidaEn.IsZero() {
		return ports.LecturaDatosContactoAutorizada{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	registro.ParticipacionRef, registro.Version = participacion, uint64(version)
	registro.Sobre.Version = registro.Version
	registro.RegistradaEn, lectura.ConsumidaEn = registro.RegistradaEn.UTC(), lectura.ConsumidaEn.UTC()
	if registro.Sobre.Validar() != nil {
		return ports.LecturaDatosContactoAutorizada{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	// Una versión confirmada por la persona (Bolsa 000040) cuenta como propia.
	if origen != nil && confirmada == nil {
		if vigenteHasta == nil || ultimoDia == nil || reglaRef == nil || huella == nil {
			return ports.LecturaDatosContactoAutorizada{}, ports.ErrDatosContactoParticipacionNoDisponibles
		}
		marca := dominiobolsa.MarcaOrigenDatosContacto{Origen: *origen, VigenteHasta: vigenteHasta.UTC(), UltimoDia: ultimoDia.Format(time.DateOnly), ReglaRef: *reglaRef, ReglaHuella: *huella}
		if marca.Validar() != nil {
			return ports.LecturaDatosContactoAutorizada{}, ports.ErrDatosContactoParticipacionNoDisponibles
		}
		registro.Origen = &marca
	}
	// El consumo solo se confirma si el caso de uso puede entregar la lectura.
	if err = entregar(lectura); err != nil {
		return ports.LecturaDatosContactoAutorizada{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.LecturaDatosContactoAutorizada{}, ports.ErrDatosContactoParticipacionNoDisponibles
	}
	return lectura, nil
}

// auditoriaRefConsumoV3 es la forma del asiento de consumo del núcleo AD3.
var auditoriaRefConsumoV3 = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)

// errorConsultaDatosContacto distingue la ausencia de datos (B78 revierte con
// P0002) de una denegación y de cualquier otro fallo.
func errorConsultaDatosContacto(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "P0002" {
		return ports.ErrDatosContactoParticipacionNoEncontrados
	}
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return dominiovec.ErrAutorizacionDenegada
	}
	return ports.ErrDatosContactoParticipacionNoDisponibles
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
