// Package postgres adapta las funciones nominales de Usuarios. El pool debe
// conectarse con el login ejecutor exclusivo; el constructor comprueba sus ACL.
package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	consultarCorreosSQL     = `SELECT vec_usuarios.consultar_correos_propios_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	recuperarCorreosSQL     = `SELECT vec_usuarios.recuperar_correos_operacion_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	aplicarCorreosSQL       = `SELECT vec_usuarios.aplicar_correos_propios_v1($1::text,$2::jsonb,$3::jsonb,$4::bytea,$5::bytea,$6::bytea,$7::bytea,$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`
	prepararVerificacionSQL = `SELECT vec_usuarios.preparar_verificacion_correo_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	cerrarVerificacionSQL   = `SELECT vec_usuarios.cerrar_verificacion_correo_v1($1::text,$2::text,$3::boolean)`
	consultarActivoSQL      = `SELECT vec_usuarios.consultar_correo_activo_verificado_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	maxRespuestaCorreos     = 256 << 10
	accionActivoCT          = "vec.correos.activo.ct.consultar"
	accionActivoBolsa       = "vec.correos.activo.bolsa.consultar"
	finalidadActivoCT       = "finalidad:usuarios:correo-activo:activo_ct:v1"
	finalidadActivoBolsa    = "finalidad:usuarios:correo-activo:activo_bolsa:v1"
	audienciaActivoCT       = "vec_usuarios.correos.activo_ct.v1"
	audienciaActivoBolsa    = "vec_usuarios.correos.activo_bolsa.v1"
)

// No basta con un pool al mismo servidor: este login solo puede heredar el
// ejecutor de su superficie y ejecutar las funciones personales, sin SET ROLE.
const acreditarEjecutorCorreosSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND g.rolname=$1::text AND NOT g.rolcanlogin
 AND g.rolinherit AND NOT g.rolbypassrls
 AND pg_has_role(session_user,g.oid,'MEMBER') AND pg_has_role(session_user,g.oid,'USAGE')
 AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)=1
 AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=g.oid)
 AND has_function_privilege(session_user,'vec_usuarios.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND has_function_privilege(session_user,'vec_usuarios.recuperar_correos_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND has_function_privilege(session_user,'vec_usuarios.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND has_function_privilege(session_user,'vec_usuarios.preparar_verificacion_correo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND has_function_privilege(session_user,'vec_usuarios.cerrar_verificacion_correo_v1(text,text,boolean)','EXECUTE')
 AND NOT has_function_privilege(session_user,'vec_usuarios.consultar_correo_activo_verificado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 FROM pg_roles l JOIN pg_roles g ON g.rolname=$1::text
 WHERE l.rolname=session_user`

type filaCorreos interface{ Scan(...any) error }
type transaccionCorreos interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) filaCorreos
	Commit(context.Context) error
	Rollback(context.Context) error
}
type transaccionCorreosPGX struct{ pgx.Tx }

func (t transaccionCorreosPGX) QueryRow(ctx context.Context, sql string, args ...any) filaCorreos {
	return t.Tx.QueryRow(ctx, sql, args...)
}

// DescifradorDireccionCorreo recibe un sobre únicamente después de autorizar
// la lectura en SQL. El protector valida AAD persona+ref+versión y borra los
// bytes claros al salir del callback.
type DescifradorDireccionCorreo interface {
	ConDireccionCorreoDescifrada(context.Context, string, ports.SobreDireccionCorreo, func([]byte) error) error
}

type RegistroCorreosPostgreSQL struct {
	iniciar     func(context.Context) (transaccionCorreos, error)
	descifrador DescifradorDireccionCorreo
	superficie  vecdomain.SuperficieAutenticacionActorV1
}

var _ ports.RegistroCorreos = (*RegistroCorreosPostgreSQL)(nil)

// LectorCorreoActivoPostgreSQL exige un login técnico server-to-server distinto
// de los dos logins personales. AD3-107 no concede EXECUTE al lector: el
// constructor permanece cerrado hasta el subcorte nominal de CT/Bolsa.
type LectorCorreoActivoPostgreSQL struct {
	registro   *RegistroCorreosPostgreSQL
	habilitado bool
}

var _ ports.LectorCorreoActivoVerificado = (*LectorCorreoActivoPostgreSQL)(nil)

func NuevoLectorCorreoActivoPostgreSQL(context.Context, *pgxpool.Pool, DescifradorDireccionCorreo) (*LectorCorreoActivoPostgreSQL, error) {
	return nil, ports.ErrCorreosNoDisponible
}

// NuevoRegistroCorreosPostgreSQL no posee ni cierra el pool. Su sonda usa el
// mismo camino SERIALIZABLE que las operaciones; bootstrap aporta el pool
// exclusivo del LOGIN de la superficie fija y el descifrador KMS vinculado.
func NuevoRegistroCorreosPostgreSQL(ctx context.Context, pool *pgxpool.Pool, descifrador DescifradorDireccionCorreo, superficie vecdomain.SuperficieAutenticacionActorV1) (*RegistroCorreosPostgreSQL, error) {
	if ctx == nil || pool == nil || descifrador == nil || ctx.Err() != nil || rolEjecutorCorreos(superficie) == "" {
		return nil, ports.ErrCorreosNoDisponible
	}
	r := &RegistroCorreosPostgreSQL{descifrador: descifrador, superficie: superficie, iniciar: func(ctx context.Context) (transaccionCorreos, error) {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return nil, err
		}
		return transaccionCorreosPGX{tx}, nil
	}}
	tx, err := r.abrir(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if err := tx.Commit(ctx); err != nil {
		return nil, errorCorreosSeguro(ctx, err)
	}
	return r, nil
}

func rolEjecutorCorreos(superficie vecdomain.SuperficieAutenticacionActorV1) string {
	switch superficie {
	case vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		return "vec_usuarios_ejecutor_interno"
	case vecdomain.SuperficieAutenticacionExternaPersonalV1:
		return "vec_usuarios_ejecutor_externo"
	default:
		return ""
	}
}

func (r *RegistroCorreosPostgreSQL) abrir(ctx context.Context) (transaccionCorreos, error) {
	if ctx == nil || r == nil || r.iniciar == nil || r.descifrador == nil || rolEjecutorCorreos(r.superficie) == "" {
		return nil, ports.ErrCorreosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return nil, errorCorreosSeguro(ctx, err)
	}
	if tx == nil {
		return nil, ports.ErrCorreosNoDisponible
	}
	fallar := func(err error) (transaccionCorreos, error) {
		_ = tx.Rollback(context.Background())
		return nil, errorCorreosSeguro(ctx, err)
	}
	for _, ajuste := range [...]string{
		"SET LOCAL search_path = pg_catalog",
		"SET LOCAL row_security = on",
		"SET LOCAL TIME ZONE 'UTC'",
		"SET LOCAL lock_timeout = '3s'",
		"SET LOCAL statement_timeout = '15s'",
		"SET LOCAL idle_in_transaction_session_timeout = '20s'",
	} {
		if _, err := tx.Exec(ctx, ajuste); err != nil {
			return fallar(err)
		}
	}
	var valido bool
	if err := tx.QueryRow(ctx, acreditarEjecutorCorreosSQL, rolEjecutorCorreos(r.superficie)).Scan(&valido); err != nil {
		return fallar(err)
	}
	if !valido {
		_ = tx.Rollback(context.Background())
		return nil, ports.ErrCorreosNoDisponible
	}
	return tx, nil
}

func argumentosCorreosV3(material []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) []any {
	return []any{string(material), v3.CapacidadCanonica(), v3.DecisionCanonica(), v3.MotivoCanonico(), v3.ContextoActorCanonico(), int64(v3.PersonaVersion()), int64(v3.PerfilVersion()), v3.PayloadVECAD3(), v3.SobreCOSESign1(), v3.EvidenciaVerificacion(), v3.RaizPublicaSPKI()}
}

func (r *RegistroCorreosPostgreSQL) validarOrdenCorreos(orden ports.OrdenCorreos, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion string) error {
	if r == nil || rolEjecutorCorreos(r.superficie) == "" {
		return ports.ErrCorreosNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return ports.ErrCorreosNoAutenticado
	}
	superficie, err := orden.Superficie()
	if err != nil {
		return ports.ErrCorreosNoAutenticado
	}
	audiencia, err := ports.AudienciaCorreos(accion, superficie)
	if err != nil {
		return ports.ErrCorreosProhibido
	}
	recurso, err := ports.RecursoCorreos(m)
	if err != nil {
		return ports.ErrCorreosInvalidos
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.ErrCorreosInvalidos
	}
	if m.PersonaRef != actor.PersonaRef || m.PerfilRef != actor.PerfilActivoRef || m.Accion != accion ||
		m.Superficie != superficie || superficie != r.superficie || m.VersionEsperada >= math.MaxInt64 ||
		m.FinalidadRef != ports.FinalidadCorreosPropios || v3.ValidarEstructura() != nil ||
		v3.PersonaVersion() != actor.Instantanea.PersonaVersion || v3.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		v3.ResumenCapacidad().Operacion() != accion || v3.ResumenCapacidad().AudienciaConsumo() != audiencia ||
		v3.ResumenCapacidad().EfectoRef() != m.PersonaRef || v3.ResumenCapacidad().EfectoHuellaSHA256() != huella {
		return ports.ErrCorreosProhibido
	}
	return nil
}

func decodificarCorreosEstricto(bruto []byte, destino any) error {
	if len(bruto) == 0 || len(bruto) > maxRespuestaCorreos {
		return ports.ErrCorreosNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ports.ErrCorreosNoDisponible
	}
	return nil
}

func reciboCorreosDesdeSQL(bruto []byte, m ports.MaterialCorreos) (ports.ReciboCorreos, error) {
	var r ports.ReciboCorreos
	if decodificarCorreosEstricto(bruto, &r) != nil || r.ReciboRef == "" || r.PersonaRef != m.PersonaRef ||
		r.Accion != m.Accion || r.CorreoRef == "" || r.Version != m.VersionEsperada+1 || r.FechaUTC.IsZero() {
		return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
	}
	_, offset := r.FechaUTC.Zone()
	if offset != 0 || r.FechaUTC.Nanosecond()%1000 != 0 {
		return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
	}
	if m.Accion != ports.AccionAnadirCorreo && r.CorreoRef != m.CorreoRef {
		return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
	}
	return r, nil
}

func errorCorreosSeguro(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ports.ErrCorreosProhibido
		case "P1409":
			return ports.ErrCorreosConflicto
		case "P1429":
			return ports.ErrCorreosLimite
		case "22023":
			return ports.ErrCorreosInvalidos
		}
	}
	// Incluye 40001: no convertir una transacción abortada en replay exitoso.
	return ports.ErrCorreosNoDisponible
}

func serializacionCorreos(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "40001"
}

// Una carrera SERIALIZABLE sólo se resuelve si una transacción nueva con V3
// fresco recupera el recibo ya confirmado. Una ausencia o fallo sigue siendo
// error; 40001 por sí solo jamás acredita el efecto ni se reintenta la mutación.
func (r *RegistroCorreosPostgreSQL) recuperarTrasSerializacion(ctx context.Context, orden ports.OrdenCorreos, m ports.MaterialCorreos) (ports.ReciboCorreos, error) {
	var vacio ports.ReciboCorreos
	if ctx == nil || ctx.Err() != nil || orden.Proveedor() == nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	vinculo, err := orden.Vinculo()
	if err != nil {
		return vacio, ports.ErrCorreosNoAutenticado
	}
	v3, err := orden.Proveedor().ProveerMaterialCorreos(ctx, vinculo, m)
	if err != nil || v3.ValidarEstructura() != nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	recibo, existe, err := r.RecuperarOperacion(ctx, orden, m, v3)
	if err != nil || !existe || !recibo.Replay {
		return vacio, ports.ErrCorreosNoDisponible
	}
	return recibo, nil
}

type sobreCorreoSQL struct {
	Version    uint64 `json:"version"`
	ClaveRef   string `json:"clave_ref"`
	NonceHex   string `json:"nonce_hex"`
	CifradoHex string `json:"cifrado_hex"`
}

func (s sobreCorreoSQL) decodificar(persona, ref string) (ports.SobreDireccionCorreo, error) {
	if persona == "" || ref == "" || s.Version == 0 || s.Version > math.MaxInt64 || s.ClaveRef == "" || len(s.NonceHex) > 64 || len(s.CifradoHex) > 8192 {
		return ports.SobreDireccionCorreo{}, ports.ErrCorreosNoDisponible
	}
	nonce, ne := hex.DecodeString(s.NonceHex)
	cifrado, ce := hex.DecodeString(s.CifradoHex)
	if ne != nil || ce != nil || len(nonce) < 12 || len(nonce) > 32 || len(cifrado) < 16 || len(cifrado) > 4096 {
		return ports.SobreDireccionCorreo{}, ports.ErrCorreosNoDisponible
	}
	return ports.SobreDireccionCorreo{CorreoRef: ref, Version: s.Version, ClaveRef: s.ClaveRef, Nonce: nonce, Cifrado: cifrado}, nil
}

type correoPropioSQL struct {
	CorreoRef     string              `json:"correo_ref"`
	Estado        domain.EstadoCorreo `json:"estado"`
	Activo        bool                `json:"activo"`
	CreadoUTC     time.Time           `json:"creado_utc"`
	VerificadoUTC *time.Time          `json:"verificado_utc"`
	Sobre         sobreCorreoSQL      `json:"sobre"`
}

func (r *RegistroCorreosPostgreSQL) ConsultarPropios(ctx context.Context, orden ports.OrdenCorreos, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.VistaCorreos, error) {
	var vacio ports.VistaCorreos
	if err := r.validarOrdenCorreos(orden, m, v3, ports.AccionConsultarCorreos); err != nil {
		return vacio, err
	}
	material, err := ports.SerializarMaterialCorreos(m)
	if err != nil {
		return vacio, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, consultarCorreosSQL, argumentosCorreosV3(material, v3)...).Scan(&bruto); err != nil {
		return vacio, errorCorreosSeguro(ctx, err)
	}
	var sql struct {
		PersonaRef string            `json:"persona_ref"`
		Version    uint64            `json:"version"`
		Correos    []correoPropioSQL `json:"correos"`
	}
	if decodificarCorreosEstricto(bruto, &sql) != nil || sql.PersonaRef != m.PersonaRef || sql.Version > math.MaxInt64 || sql.Correos == nil || len(sql.Correos) > 32 {
		return vacio, ports.ErrCorreosNoDisponible
	}
	vista := ports.VistaCorreos{PersonaRef: sql.PersonaRef, Version: sql.Version, Correos: make([]domain.CorreoPropio, 0, len(sql.Correos))}
	for _, c := range sql.Correos {
		sobre, err := c.Sobre.decodificar(m.PersonaRef, c.CorreoRef)
		if err != nil || sobre.Version > vista.Version {
			return vacio, ports.ErrCorreosNoDisponible
		}
		var direccion string
		err = r.descifrador.ConDireccionCorreoDescifrada(ctx, m.PersonaRef, sobre, func(claro []byte) error {
			direccion = string(claro)
			if !domain.DireccionCorreoValida(direccion) {
				return ports.ErrCorreosNoDisponible
			}
			return nil
		})
		if err != nil || !domain.DireccionCorreoValida(direccion) {
			return vacio, ports.ErrCorreosNoDisponible
		}
		vista.Correos = append(vista.Correos, domain.CorreoPropio{CorreoRef: c.CorreoRef, Direccion: direccion, Estado: c.Estado, Activo: c.Activo, CreadoUTC: c.CreadoUTC, VerificadoUTC: c.VerificadoUTC})
	}
	if domain.ValidarConjuntoCorreos(vista.Correos) != nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorCorreosSeguro(ctx, err)
	}
	return vista, nil
}

func (r *RegistroCorreosPostgreSQL) RecuperarOperacion(ctx context.Context, orden ports.OrdenCorreos, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCorreos, bool, error) {
	var vacio ports.ReciboCorreos
	if err := r.validarOrdenCorreos(orden, m, v3, m.Accion); err != nil {
		return vacio, false, err
	}
	if m.Accion == ports.AccionConsultarCorreos {
		return vacio, false, ports.ErrCorreosInvalidos
	}
	material, err := ports.SerializarMaterialCorreos(m)
	if err != nil {
		return vacio, false, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, false, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, recuperarCorreosSQL, argumentosCorreosV3(material, v3)...).Scan(&bruto); err != nil {
		return vacio, false, errorCorreosSeguro(ctx, err)
	}
	if bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) {
		if err := tx.Commit(ctx); err != nil {
			return vacio, false, errorCorreosSeguro(ctx, err)
		}
		return vacio, false, nil
	}
	recibo, err := reciboCorreosDesdeSQL(bruto, m)
	if err != nil || !recibo.Replay {
		return vacio, false, ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, false, errorCorreosSeguro(ctx, err)
	}
	return recibo, true, nil
}

type sobreAplicarJSON struct {
	CorreoRef         string `json:"correo_ref"`
	Version           uint64 `json:"version"`
	ClaveRef          string `json:"clave_ref"`
	ClaveIgualdadRef  string `json:"clave_igualdad_ref"`
	NonceHex          string `json:"nonce_hex"`
	CifradoHex        string `json:"cifrado_hex"`
	HuellaIgualdadHex string `json:"huella_igualdad_hex"`
}
type reservaAplicarJSON struct {
	DesafioRef      string    `json:"desafio_ref"`
	DesafioHex      string    `json:"desafio_hex"`
	HuellaCodigoHex string    `json:"huella_codigo_hex"`
	ClaveRef        string    `json:"clave_ref"`
	VenceUTC        time.Time `json:"vence_utc"`
}

func parametrosMutacionCorreos(p ports.PeticionCorreo, m ports.MaterialCorreos, sobre ports.SobreDireccionCorreo, reserva ports.ReservaDesafio) ([]byte, []byte, error) {
	if p.VersionEsperada != m.VersionEsperada || p.ClaveOperacion != m.ClaveOperacion || p.Codigo != "" || p.Direccion != "" ||
		p.CorreoRef != m.CorreoRef && m.Accion != ports.AccionAnadirCorreo || p.SustitutoRef != m.SustitutoRef {
		return nil, nil, ports.ErrCorreosInvalidos
	}
	var sj any
	var rj any
	switch m.Accion {
	case ports.AccionAnadirCorreo:
		if m.CorreoRef != "" || p.CorreoRef == "" || sobre.CorreoRef != p.CorreoRef || sobre.Version != m.VersionEsperada+1 ||
			sobre.ClaveRef == "" || sobre.ClaveIgualdadRef == "" || sobre.ClaveIgualdadRef == sobre.ClaveRef || len(sobre.Nonce) < 12 || len(sobre.Nonce) > 32 || len(sobre.Cifrado) < 16 || len(sobre.Cifrado) > 4096 || len(sobre.HuellaIgualdad) != 32 {
			return nil, nil, ports.ErrCorreosInvalidos
		}
		sj = sobreAplicarJSON{p.CorreoRef, sobre.Version, sobre.ClaveRef, sobre.ClaveIgualdadRef, hex.EncodeToString(sobre.Nonce), hex.EncodeToString(sobre.Cifrado), hex.EncodeToString(sobre.HuellaIgualdad)}
		fallthrough
	case ports.AccionReenviarCorreo:
		_, offset := reserva.VenceUTC.Zone()
		if reserva.DesafioRef == "" || len(reserva.Desafio) < 16 || len(reserva.Desafio) > 256 || len(reserva.HuellaCodigo) != 32 || reserva.ClaveRef == "" || reserva.VenceUTC.IsZero() || offset != 0 || reserva.VenceUTC.Nanosecond()%1000 != 0 {
			return nil, nil, ports.ErrCorreosInvalidos
		}
		rj = reservaAplicarJSON{reserva.DesafioRef, hex.EncodeToString(reserva.Desafio), hex.EncodeToString(reserva.HuellaCodigo), reserva.ClaveRef, reserva.VenceUTC}
	case ports.AccionActivarCorreo, ports.AccionRetirarCorreo:
		// Ningún sobre ni desafío nuevo forma parte de estas operaciones.
	default:
		return nil, nil, ports.ErrCorreosInvalidos
	}
	sb, err := json.Marshal(sj)
	if err != nil {
		return nil, nil, ports.ErrCorreosInvalidos
	}
	rb, err := json.Marshal(rj)
	if err != nil {
		return nil, nil, ports.ErrCorreosInvalidos
	}
	return sb, rb, nil
}

func (r *RegistroCorreosPostgreSQL) Aplicar(ctx context.Context, orden ports.OrdenCorreos, p ports.PeticionCorreo, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, sobre ports.SobreDireccionCorreo, reserva ports.ReservaDesafio, comprobador ports.ComprobadorCodigoCorreo) (ports.ReciboCorreos, error) {
	var vacio ports.ReciboCorreos
	if err := r.validarOrdenCorreos(orden, m, v3, m.Accion); err != nil {
		return vacio, err
	}
	if m.Accion == ports.AccionVerificarCorreo {
		return r.verificar(ctx, orden, p, m, v3, comprobador)
	}
	sb, rb, err := parametrosMutacionCorreos(p, m, sobre, reserva)
	if err != nil {
		return vacio, err
	}
	material, err := ports.SerializarMaterialCorreos(m)
	if err != nil {
		return vacio, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	base := argumentosCorreosV3(material, v3)
	args := make([]any, 0, 13)
	args = append(args, base[0], sb, rb)
	args = append(args, base[1:]...)
	var bruto []byte
	if err := tx.QueryRow(ctx, aplicarCorreosSQL, args...).Scan(&bruto); err != nil {
		_ = tx.Rollback(context.Background())
		if serializacionCorreos(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorCorreosSeguro(ctx, err)
	}
	recibo, err := reciboCorreosDesdeSQL(bruto, m)
	if err != nil || (!recibo.Replay && m.Accion == ports.AccionAnadirCorreo && recibo.CorreoRef != p.CorreoRef) {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		if serializacionCorreos(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorCorreosSeguro(ctx, err)
	}
	return recibo, nil
}

func (r *RegistroCorreosPostgreSQL) verificar(ctx context.Context, orden ports.OrdenCorreos, p ports.PeticionCorreo, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, comprobador ports.ComprobadorCodigoCorreo) (ports.ReciboCorreos, error) {
	var vacio ports.ReciboCorreos
	if comprobador == nil || p.Codigo != "" || p.Direccion != "" || p.CorreoRef != m.CorreoRef || p.VersionEsperada != m.VersionEsperada || p.ClaveOperacion != m.ClaveOperacion || p.SustitutoRef != "" {
		return vacio, ports.ErrCorreosInvalidos
	}
	material, err := ports.SerializarMaterialCorreos(m)
	if err != nil {
		return vacio, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, prepararVerificacionSQL, argumentosCorreosV3(material, v3)...).Scan(&bruto); err != nil {
		_ = tx.Rollback(context.Background())
		if serializacionCorreos(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorCorreosSeguro(ctx, err)
	}
	var meta struct {
		PersonaRef      string    `json:"persona_ref"`
		CorreoRef       string    `json:"correo_ref"`
		DesafioRef      string    `json:"desafio_ref"`
		HuellaCodigoHex string    `json:"huella_codigo_hex"`
		ClaveRef        string    `json:"clave_ref"`
		VenceUTC        time.Time `json:"vence_utc"`
	}
	if decodificarCorreosEstricto(bruto, &meta) != nil || meta.PersonaRef != m.PersonaRef || meta.CorreoRef != m.CorreoRef || meta.DesafioRef == "" || meta.ClaveRef == "" || meta.VenceUTC.IsZero() {
		return vacio, ports.ErrCorreosNoDisponible
	}
	_, offset := meta.VenceUTC.Zone()
	if offset != 0 || meta.VenceUTC.Nanosecond()%1000 != 0 {
		return vacio, ports.ErrCorreosNoDisponible
	}
	huella, err := hex.DecodeString(meta.HuellaCodigoHex)
	if err != nil || len(huella) != 32 {
		return vacio, ports.ErrCorreosNoDisponible
	}
	valido, err := comprobador.Comprobar(ctx, ports.MetadatosDesafioCorreo{PersonaRef: meta.PersonaRef, CorreoRef: meta.CorreoRef, DesafioRef: meta.DesafioRef, HuellaCodigo: huella, ClaveRef: meta.ClaveRef, VenceUTC: meta.VenceUTC})
	if err != nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if err := tx.QueryRow(ctx, cerrarVerificacionSQL, m.PersonaRef, m.ClaveOperacion, valido).Scan(&bruto); err != nil {
		_ = tx.Rollback(context.Background())
		if serializacionCorreos(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorCorreosSeguro(ctx, err)
	}
	if !valido {
		var resultado struct {
			Valido bool `json:"valido"`
		}
		if decodificarCorreosEstricto(bruto, &resultado) != nil || resultado.Valido {
			return vacio, ports.ErrCorreosNoDisponible
		}
		if err := tx.Commit(ctx); err != nil {
			return vacio, errorCorreosSeguro(ctx, err)
		}
		return vacio, ports.ErrCorreosInvalidos
	}
	recibo, err := reciboCorreosDesdeSQL(bruto, m)
	if err != nil {
		return vacio, err
	}
	if err := tx.Commit(ctx); err != nil {
		if serializacionCorreos(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorCorreosSeguro(ctx, err)
	}
	return recibo, nil
}

// ConsultarActivoVerificado permite a CT y Bolsa consultar la autoridad de
// Usuarios sin acceso a sus tablas. La finalidad y persona se cotejan con la
// capacidad V3 en la función nominal, que audita la lectura en la transacción.
func (l *LectorCorreoActivoPostgreSQL) ConsultarActivoVerificado(ctx context.Context, s ports.SolicitudCorreoActivo) (ports.CorreoActivoVerificado, bool, error) {
	var vacio ports.CorreoActivoVerificado
	if l == nil || !l.habilitado || l.registro == nil {
		return vacio, false, ports.ErrCorreosNoDisponible
	}
	r := l.registro
	if s.PersonaRef == "" || s.Material.ValidarEstructura() != nil ||
		s.Material.ResumenCapacidad().EfectoRef() != s.PersonaRef {
		return vacio, false, ports.ErrCorreosProhibido
	}
	resumen := s.Material.ResumenCapacidad()
	correcto := (s.FinalidadRef == finalidadActivoCT && resumen.Operacion() == accionActivoCT && resumen.AudienciaConsumo() == audienciaActivoCT) ||
		(s.FinalidadRef == finalidadActivoBolsa && resumen.Operacion() == accionActivoBolsa && resumen.AudienciaConsumo() == audienciaActivoBolsa)
	if !correcto {
		return vacio, false, ports.ErrCorreosProhibido
	}
	actor, err := vecdomain.RehidratarContextoActorVinculadoV2(s.Material.ContextoActorCanonico())
	if err != nil || actor.PerfilActivoRef == "" || actor.Instantanea.PerfilVersion != s.Material.PerfilVersion() || actor.Instantanea.PersonaVersion != s.Material.PersonaVersion() {
		return vacio, false, ports.ErrCorreosProhibido
	}
	// El contrato técnico lector queda pendiente de AD3 nominal. Este código
	// conserva la traducción de sobres; su constructor no lo habilita.
	material, err := json.Marshal(struct {
		PersonaRef   string `json:"persona_ref"`
		PerfilRef    string `json:"perfil_ref"`
		Accion       string `json:"accion"`
		FinalidadRef string `json:"finalidad_ref"`
	}{s.PersonaRef, actor.PerfilActivoRef, resumen.Operacion(), s.FinalidadRef})
	if err != nil {
		return vacio, false, ports.ErrCorreosInvalidos
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, false, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, consultarActivoSQL, argumentosCorreosV3(material, s.Material)...).Scan(&bruto); err != nil {
		return vacio, false, errorCorreosSeguro(ctx, err)
	}
	if bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) {
		if err := tx.Commit(ctx); err != nil {
			return vacio, false, errorCorreosSeguro(ctx, err)
		}
		return vacio, false, nil
	}
	var salida struct {
		PersonaRef      string `json:"persona_ref"`
		CorreoRef       string `json:"correo_ref"`
		Version         uint64 `json:"version"`
		ConjuntoVersion uint64 `json:"conjunto_version"`
		ClaveRef        string `json:"clave_ref"`
		NonceHex        string `json:"nonce_hex"`
		CifradoHex      string `json:"cifrado_hex"`
	}
	if decodificarCorreosEstricto(bruto, &salida) != nil || salida.PersonaRef != s.PersonaRef || salida.CorreoRef == "" || salida.Version == 0 || salida.ConjuntoVersion < salida.Version || salida.ConjuntoVersion > math.MaxInt64 {
		return vacio, false, ports.ErrCorreosNoDisponible
	}
	sobre, err := (sobreCorreoSQL{Version: salida.Version, ClaveRef: salida.ClaveRef, NonceHex: salida.NonceHex, CifradoHex: salida.CifradoHex}).decodificar(s.PersonaRef, salida.CorreoRef)
	if err != nil {
		return vacio, false, err
	}
	var direccion string
	err = r.descifrador.ConDireccionCorreoDescifrada(ctx, s.PersonaRef, sobre, func(claro []byte) error {
		direccion = string(claro)
		if !domain.DireccionCorreoValida(direccion) {
			return ports.ErrCorreosNoDisponible
		}
		return nil
	})
	if err != nil || !domain.DireccionCorreoValida(direccion) {
		return vacio, false, ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, false, errorCorreosSeguro(ctx, err)
	}
	return ports.CorreoActivoVerificado{PersonaRef: s.PersonaRef, CorreoRef: salida.CorreoRef, Direccion: direccion, Version: salida.ConjuntoVersion}, true, nil
}
