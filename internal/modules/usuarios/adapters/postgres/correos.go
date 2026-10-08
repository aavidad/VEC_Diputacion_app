// Adaptador PostgreSQL de «Mis correos». El pool se conecta con el LOGIN
// ejecutor exclusivo de la superficie; cada transacción vuelve a comprobarlo.
package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const maxRespuestaCorreos = 64 << 10

// Los seis ajustes mantienen los valores y el alcance de SET LOCAL.
const ajustesTransaccionCorreosSQL = `SELECT pg_catalog.set_config('search_path','pg_catalog',true),
 pg_catalog.set_config('row_security','on',true),
 pg_catalog.set_config('timezone','UTC',true),
 pg_catalog.set_config('lock_timeout','3s',true),
 pg_catalog.set_config('statement_timeout','15s',true),
 pg_catalog.set_config('idle_in_transaction_session_timeout','20s',true)`

// Las fachadas de «Mis correos» viven en un esquema por población (Usuarios
// 000010): el personal en vec_usuarios_correos_interno y el Área personal en
// vec_usuarios_correos_externo. @ESQ@ se sustituye por el esquema de la
// superficie del registro, tomado de una tabla cerrada y nunca de la petición.
const (
	plantillaConsultarCorreosSQL     = `SELECT @ESQ@.consultar_correos_propios_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	plantillaRecuperarCorreosSQL     = `SELECT @ESQ@.recuperar_correos_operacion_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	plantillaAplicarCorreosSQL       = `SELECT @ESQ@.aplicar_correos_propios_v1($1::text,$2::jsonb,$3::jsonb,$4::bytea,$5::bytea,$6::bytea,$7::bytea,$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`
	plantillaPrepararVerificacionSQL = `SELECT @ESQ@.preparar_verificacion_correo_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	plantillaCerrarVerificacionSQL   = `SELECT @ESQ@.cerrar_verificacion_correo_v1($1::text,$2::text,$3::boolean)`
	plantillaConfirmarEnvioSQL       = `SELECT @ESQ@.confirmar_envio_correo_v1($1::text,$2::text,$3::text,$4::boolean)`
)

// No basta con un pool al mismo servidor: este login sólo puede heredar el
// ejecutor de su superficie y ejecutar las funciones personales de su
// esquema, sin SET ROLE, sin tablas y sin acceso al esquema de la otra
// población (@OTRO@).
const plantillaAcreditarEjecutorCorreosSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND g.rolname=$1::text AND NOT g.rolcanlogin
 AND g.rolinherit AND NOT g.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER') AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)=1
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=g.oid)
 AND pg_catalog.has_function_privilege(session_user,'@ESQ@.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'@ESQ@.recuperar_correos_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'@ESQ@.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'@ESQ@.preparar_verificacion_correo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'@ESQ@.cerrar_verificacion_correo_v1(text,text,boolean)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'@ESQ@.confirmar_envio_correo_v1(text,text,text,boolean)','EXECUTE')
 AND NOT pg_catalog.has_table_privilege(session_user,'@ESQ@.correos_direccion','SELECT')
 AND NOT pg_catalog.has_schema_privilege(session_user,'@OTRO@','USAGE')
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname=$1::text
 WHERE l.rolname=session_user`

// sentenciasCorreos son las llamadas SQL ya resueltas para una población.
type sentenciasCorreos struct {
	consultar, recuperar, aplicar, preparar, cerrar, confirmar, acreditar string
}

// sentenciasCorreosSuperficie devuelve las sentencias del esquema de la
// superficie y false para cualquier otra.
func sentenciasCorreosSuperficie(superficie vecdomain.SuperficieAutenticacionActorV1) (sentenciasCorreos, bool) {
	var esquema, otro string
	switch superficie {
	case vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		esquema, otro = "vec_usuarios_correos_interno", "vec_usuarios_correos_externo"
	case vecdomain.SuperficieAutenticacionExternaPersonalV1:
		esquema, otro = "vec_usuarios_correos_externo", "vec_usuarios_correos_interno"
	default:
		return sentenciasCorreos{}, false
	}
	r := strings.NewReplacer("@ESQ@", esquema, "@OTRO@", otro)
	return sentenciasCorreos{
		consultar: r.Replace(plantillaConsultarCorreosSQL), recuperar: r.Replace(plantillaRecuperarCorreosSQL),
		aplicar: r.Replace(plantillaAplicarCorreosSQL), preparar: r.Replace(plantillaPrepararVerificacionSQL),
		cerrar: r.Replace(plantillaCerrarVerificacionSQL), confirmar: r.Replace(plantillaConfirmarEnvioSQL),
		acreditar: r.Replace(plantillaAcreditarEjecutorCorreosSQL),
	}, true
}

var (
	patronEnvioRef   = regexp.MustCompile(`^correo_envio:[0-9a-f]{32}$`)
	patronReservaRef = regexp.MustCompile(`^reserva:[0-9a-f]{32}$`)
	patronDesafioRef = regexp.MustCompile(`^desafio:[0-9a-f]{32}$`)
	patronCorreoRef  = regexp.MustCompile(`^correo:[0-9a-f]{32}$`)
	patronReciboRef  = regexp.MustCompile(`^correo_recibo:[0-9a-f]{32}$`)
)

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
	sql         sentenciasCorreos
}

var _ ports.RegistroCorreos = (*RegistroCorreosPostgreSQL)(nil)

// NuevoRegistroCorreosPostgreSQL no posee ni cierra el pool. Su sonda usa el
// mismo camino SERIALIZABLE que las operaciones.
func NuevoRegistroCorreosPostgreSQL(ctx context.Context, pool *pgxpool.Pool, descifrador DescifradorDireccionCorreo, superficie vecdomain.SuperficieAutenticacionActorV1) (*RegistroCorreosPostgreSQL, error) {
	sentencias, ok := sentenciasCorreosSuperficie(superficie)
	if ctx == nil || pool == nil || descifrador == nil || ctx.Err() != nil || rolEjecutorPreferencias(superficie) == "" || !ok {
		return nil, ports.ErrCorreosNoDisponible
	}
	r := &RegistroCorreosPostgreSQL{descifrador: descifrador, superficie: superficie, sql: sentencias, iniciar: func(ctx context.Context) (transaccionCorreos, error) {
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

func (r *RegistroCorreosPostgreSQL) abrir(ctx context.Context) (transaccionCorreos, error) {
	if ctx == nil || r == nil || r.iniciar == nil || r.descifrador == nil || rolEjecutorPreferencias(r.superficie) == "" || r.sql.acreditar == "" {
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
	if _, err := tx.Exec(ctx, ajustesTransaccionCorreosSQL); err != nil {
		return fallar(err)
	}
	var valido bool
	if err := tx.QueryRow(ctx, r.sql.acreditar, rolEjecutorPreferencias(r.superficie)).Scan(&valido); err != nil {
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

// validarOrdenCorreos repite en el adaptador el cotejo de identidad,
// superficie, recurso y capacidad antes de abrir una transacción.
func (r *RegistroCorreosPostgreSQL) validarOrdenCorreos(orden ports.OrdenCorreos, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion string) ([]byte, error) {
	if r == nil || rolEjecutorPreferencias(r.superficie) == "" {
		return nil, ports.ErrCorreosNoDisponible
	}
	actor, _, superficie, err := orden.Identidad.Datos()
	if err != nil {
		return nil, ports.ErrCorreosNoAutenticado
	}
	audiencia, err := canonico.AudienciaCorreos(accion, superficie)
	if err != nil {
		return nil, ports.ErrCorreosProhibido
	}
	recurso, err := canonico.RecursoCorreos(m)
	if err != nil {
		return nil, ports.ErrCorreosInvalidos
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return nil, ports.ErrCorreosInvalidos
	}
	resumen := v3.ResumenCapacidad()
	if m.PersonaRef != actor.PersonaRef || m.PerfilRef != actor.PerfilActivoRef || m.Accion != accion ||
		m.Superficie != superficie || superficie != r.superficie || m.VersionEsperada >= math.MaxInt64 ||
		m.FinalidadRef != ports.FinalidadCorreosPropios || v3.ValidarEstructura() != nil ||
		v3.PersonaVersion() != actor.Instantanea.PersonaVersion || v3.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia ||
		resumen.EfectoRef() != m.PersonaRef || resumen.EfectoHuellaSHA256() != huella {
		return nil, ports.ErrCorreosProhibido
	}
	return canonico.SerializarMaterialCorreos(m)
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

func fechaUTCMicro(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}

type reciboCorreosSQL struct {
	ReciboRef  string            `json:"recibo_ref"`
	PersonaRef string            `json:"persona_ref"`
	Accion     string            `json:"accion"`
	CorreoRef  string            `json:"correo_ref"`
	Version    uint64            `json:"version"`
	FechaUTC   time.Time         `json:"fecha_utc"`
	Replay     bool              `json:"replay"`
	Envios     []envioCorreosSQL `json:"envios"`
}

type envioCorreosSQL struct {
	EnvioRef   string         `json:"envio_ref"`
	ReservaRef string         `json:"reserva_ref"`
	Tipo       string         `json:"tipo"`
	CorreoRef  string         `json:"correo_ref"`
	DesafioRef *string        `json:"desafio_ref"`
	Sobre      sobreCorreoSQL `json:"sobre"`
}

func resultadoCorreosDesdeSQL(bruto []byte, m ports.MaterialCorreos) (ports.ResultadoCorreos, error) {
	var vacio ports.ResultadoCorreos
	var r reciboCorreosSQL
	if decodificarCorreosEstricto(bruto, &r) != nil || !patronReciboRef.MatchString(r.ReciboRef) || r.PersonaRef != m.PersonaRef ||
		r.Accion != m.Accion || !patronCorreoRef.MatchString(r.CorreoRef) || r.Version != m.VersionEsperada+1 || !fechaUTCMicro(r.FechaUTC) ||
		r.Envios == nil || len(r.Envios) > 1 || (r.Replay && len(r.Envios) != 0) {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if m.Accion != ports.AccionAnadirCorreo && r.CorreoRef != m.CorreoRef {
		return vacio, ports.ErrCorreosNoDisponible
	}
	resultado := ports.ResultadoCorreos{Recibo: ports.ReciboCorreos{ReciboRef: r.ReciboRef, PersonaRef: r.PersonaRef, Accion: r.Accion, CorreoRef: r.CorreoRef, Version: r.Version, FechaUTC: r.FechaUTC, Replay: r.Replay}}
	for _, e := range r.Envios {
		desafio := ""
		if e.DesafioRef != nil {
			desafio = *e.DesafioRef
		}
		if !patronEnvioRef.MatchString(e.EnvioRef) || !patronReservaRef.MatchString(e.ReservaRef) || !patronCorreoRef.MatchString(e.CorreoRef) ||
			(e.Tipo == ports.TipoEnvioCodigo && (!patronDesafioRef.MatchString(desafio) || e.CorreoRef != r.CorreoRef)) ||
			(e.Tipo == ports.TipoEnvioAviso && (desafio != "" || e.CorreoRef == r.CorreoRef)) ||
			(e.Tipo != ports.TipoEnvioCodigo && e.Tipo != ports.TipoEnvioAviso) {
			return vacio, ports.ErrCorreosNoDisponible
		}
		sobre, err := e.Sobre.decodificar(m.PersonaRef, e.CorreoRef)
		if err != nil {
			return vacio, err
		}
		resultado.Envios = append(resultado.Envios, ports.EnvioPendiente{EnvioRef: e.EnvioRef, ReservaRef: e.ReservaRef, Tipo: e.Tipo, CorreoRef: e.CorreoRef, DesafioRef: desafio, Sobre: sobre})
	}
	return resultado, nil
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
		case "P1410":
			return ports.ErrCorreosCodigoCaducado
		case "P1411":
			return ports.ErrCorreosYaRegistrado
		case "P1412":
			return ports.ErrCorreosMaximo
		case "P1413":
			return ports.ErrCorreosEnUso
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
// fresco recupera el recibo ya confirmado. Sin recibo propio, la carrera la
// ganó otra operación y se responde conflicto; 40001 jamás acredita el efecto
// ni se reintenta la mutación.
func (r *RegistroCorreosPostgreSQL) recuperarTrasSerializacion(ctx context.Context, orden ports.OrdenCorreos, m ports.MaterialCorreos) (ports.ResultadoCorreos, error) {
	var vacio ports.ResultadoCorreos
	if ctx == nil || ctx.Err() != nil || orden.Proveedor == nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	_, vinculo, _, err := orden.Identidad.Datos()
	if err != nil {
		return vacio, ports.ErrCorreosNoAutenticado
	}
	v3, err := orden.Proveedor.ProveerMaterialCorreos(ctx, vinculo, m)
	if err != nil || v3.ValidarEstructura() != nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	recibo, existe, err := r.RecuperarOperacion(ctx, orden, m, v3)
	if errors.Is(err, ports.ErrCorreosCodigoIncorrecto) {
		return vacio, err
	}
	if err != nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if !existe || !recibo.Replay {
		// Otra operación de la misma persona terminó antes: la lista cambió.
		return vacio, ports.ErrCorreosConflicto
	}
	return ports.ResultadoCorreos{Recibo: recibo}, nil
}

type sobreCorreoSQL struct {
	Version    uint64 `json:"version"`
	ClaveRef   string `json:"clave_ref"`
	NonceHex   string `json:"nonce_hex"`
	CifradoHex string `json:"cifrado_hex"`
}

func (s sobreCorreoSQL) decodificar(persona, ref string) (ports.SobreDireccionCorreo, error) {
	if persona == "" || ref == "" || s.Version == 0 || s.Version > math.MaxInt64 || s.ClaveRef == "" || len(s.ClaveRef) > 128 || len(s.NonceHex) != 24 || len(s.CifradoHex) > 540 {
		return ports.SobreDireccionCorreo{}, ports.ErrCorreosNoDisponible
	}
	nonce, ne := hex.DecodeString(s.NonceHex)
	cifrado, ce := hex.DecodeString(s.CifradoHex)
	if ne != nil || ce != nil || len(nonce) != 12 || len(cifrado) < 19 || len(cifrado) > 270 {
		return ports.SobreDireccionCorreo{}, ports.ErrCorreosNoDisponible
	}
	return ports.SobreDireccionCorreo{CorreoRef: ref, Version: s.Version, ClaveRef: s.ClaveRef, Nonce: nonce, Cifrado: cifrado}, nil
}

type codigoCorreoSQL struct {
	VenceUTC          time.Time `json:"vence_utc"`
	IntentosRestantes int       `json:"intentos_restantes"`
}

type correoPropioSQL struct {
	CorreoRef     string              `json:"correo_ref"`
	Estado        domain.EstadoCorreo `json:"estado"`
	Activo        bool                `json:"activo"`
	CreadoUTC     time.Time           `json:"creado_utc"`
	VerificadoUTC *time.Time          `json:"verificado_utc"`
	Sobre         sobreCorreoSQL      `json:"sobre"`
	Codigo        *codigoCorreoSQL    `json:"codigo"`
}

func (r *RegistroCorreosPostgreSQL) ConsultarPropios(ctx context.Context, orden ports.OrdenCorreos, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.VistaCorreos, error) {
	var vacio ports.VistaCorreos
	material, err := r.validarOrdenCorreos(orden, m, v3, ports.AccionConsultarCorreos)
	if err != nil {
		return vacio, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, r.sql.consultar, argumentosCorreosV3(material, v3)...).Scan(&bruto); err != nil {
		return vacio, errorCorreosSeguro(ctx, err)
	}
	var sql struct {
		PersonaRef string            `json:"persona_ref"`
		Version    uint64            `json:"version"`
		Correos    []correoPropioSQL `json:"correos"`
	}
	if decodificarCorreosEstricto(bruto, &sql) != nil || sql.PersonaRef != m.PersonaRef || sql.Version > math.MaxInt64 || sql.Correos == nil || len(sql.Correos) > domain.MaxCorreosVivos {
		return vacio, ports.ErrCorreosNoDisponible
	}
	vista := ports.VistaCorreos{PersonaRef: sql.PersonaRef, Version: sql.Version, Correos: make([]domain.CorreoPropio, 0, len(sql.Correos))}
	for _, c := range sql.Correos {
		sobre, err := c.Sobre.decodificar(m.PersonaRef, c.CorreoRef)
		if err != nil || !patronCorreoRef.MatchString(c.CorreoRef) || sobre.Version > vista.Version || !fechaUTCMicro(c.CreadoUTC) {
			return vacio, ports.ErrCorreosNoDisponible
		}
		var direccion string
		err = r.descifrador.ConDireccionCorreoDescifrada(ctx, m.PersonaRef, sobre, func(claro []byte) error {
			direccion = string(claro)
			return nil
		})
		if err != nil || !domain.DireccionCorreoValida(direccion) {
			return vacio, ports.ErrCorreosNoDisponible
		}
		correo := domain.CorreoPropio{CorreoRef: c.CorreoRef, Direccion: direccion, Estado: c.Estado, Activo: c.Activo, CreadoUTC: c.CreadoUTC, VerificadoUTC: c.VerificadoUTC}
		if c.Codigo != nil {
			if !fechaUTCMicro(c.Codigo.VenceUTC) {
				return vacio, ports.ErrCorreosNoDisponible
			}
			correo.Codigo = &domain.CodigoPendiente{VenceUTC: c.Codigo.VenceUTC, IntentosRestantes: c.Codigo.IntentosRestantes}
		}
		vista.Correos = append(vista.Correos, correo)
	}
	if domain.ValidarConjuntoCorreos(vista.Correos) != nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorCorreosSeguro(ctx, err)
	}
	return vista, nil
}

// replayCorreos interpreta la respuesta de recuperación. Un intento fallido
// repetido con la misma clave vuelve a responder «código incorrecto».
func replayCorreos(bruto []byte, m ports.MaterialCorreos) (ports.ReciboCorreos, bool, error) {
	var vacio ports.ReciboCorreos
	// SQL devuelve NULL cuando la clave no tiene operación registrada.
	if bruto == nil || bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) {
		return vacio, false, nil
	}
	var campos map[string]json.RawMessage
	if len(bruto) > maxRespuestaCorreos || json.Unmarshal(bruto, &campos) != nil || campos == nil {
		return vacio, false, ports.ErrCorreosNoDisponible
	}
	if _, negativo := campos["resultado"]; negativo {
		var centinela struct {
			Resultado  string `json:"resultado"`
			PersonaRef string `json:"persona_ref"`
			CorreoRef  string `json:"correo_ref"`
			Replay     bool   `json:"replay"`
		}
		if m.Accion != ports.AccionVerificarCorreo || decodificarCorreosEstricto(bruto, &centinela) != nil ||
			centinela.Resultado != "codigo_invalido" || centinela.PersonaRef != m.PersonaRef ||
			centinela.CorreoRef != m.CorreoRef || !centinela.Replay {
			return vacio, false, ports.ErrCorreosNoDisponible
		}
		return vacio, false, ports.CodigoIncorrecto{IntentosRestantes: -1}
	}
	resultado, err := resultadoCorreosDesdeSQL(bruto, m)
	if err != nil || !resultado.Recibo.Replay {
		return vacio, false, ports.ErrCorreosNoDisponible
	}
	return resultado.Recibo, true, nil
}

func (r *RegistroCorreosPostgreSQL) RecuperarOperacion(ctx context.Context, orden ports.OrdenCorreos, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCorreos, bool, error) {
	var vacio ports.ReciboCorreos
	if m.Accion == ports.AccionConsultarCorreos {
		return vacio, false, ports.ErrCorreosInvalidos
	}
	material, err := r.validarOrdenCorreos(orden, m, v3, m.Accion)
	if err != nil {
		return vacio, false, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, false, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, r.sql.recuperar, argumentosCorreosV3(material, v3)...).Scan(&bruto); err != nil {
		return vacio, false, errorCorreosSeguro(ctx, err)
	}
	recibo, existe, errReplay := replayCorreos(bruto, m)
	if errReplay != nil && !errors.Is(errReplay, ports.ErrCorreosCodigoIncorrecto) {
		return vacio, false, errReplay
	}
	// El consumo V3 y su auditoría se confirman también si no hay recibo.
	if err := tx.Commit(ctx); err != nil {
		return vacio, false, errorCorreosSeguro(ctx, err)
	}
	return recibo, existe, errReplay
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
	DesafioRef      string `json:"desafio_ref"`
	HuellaCodigoHex string `json:"huella_codigo_hex"`
	ClaveRef        string `json:"clave_ref"`
	VenceUTC        string `json:"vence_utc"`
}

func parametrosMutacionCorreos(p ports.PeticionCorreo, m ports.MaterialCorreos, sobre ports.SobreDireccionCorreo, reserva ports.ReservaDesafio) ([]byte, []byte, error) {
	if p.VersionEsperada != m.VersionEsperada || p.ClaveOperacion != m.ClaveOperacion || p.Codigo != "" || p.Direccion != "" ||
		(p.CorreoRef != m.CorreoRef && m.Accion != ports.AccionAnadirCorreo) || reserva.Codigo != "" {
		return nil, nil, ports.ErrCorreosInvalidos
	}
	var sj, rj any
	switch m.Accion {
	case ports.AccionAnadirCorreo:
		if m.CorreoRef != "" || !patronCorreoRef.MatchString(p.CorreoRef) || sobre.CorreoRef != p.CorreoRef || sobre.Version != m.VersionEsperada+1 ||
			sobre.ClaveRef == "" || sobre.ClaveIgualdadRef == "" || sobre.ClaveIgualdadRef == sobre.ClaveRef || len(sobre.Nonce) != 12 || len(sobre.Cifrado) < 19 || len(sobre.Cifrado) > 270 || len(sobre.HuellaIgualdad) != 32 {
			return nil, nil, ports.ErrCorreosInvalidos
		}
		sj = sobreAplicarJSON{p.CorreoRef, sobre.Version, sobre.ClaveRef, sobre.ClaveIgualdadRef, hex.EncodeToString(sobre.Nonce), hex.EncodeToString(sobre.Cifrado), hex.EncodeToString(sobre.HuellaIgualdad)}
		fallthrough
	case ports.AccionReenviarCorreo:
		if !patronDesafioRef.MatchString(reserva.DesafioRef) || len(reserva.HuellaCodigo) != 32 || reserva.ClaveRef == "" || !fechaUTCMicro(reserva.VenceUTC.UTC()) {
			return nil, nil, ports.ErrCorreosInvalidos
		}
		rj = reservaAplicarJSON{reserva.DesafioRef, hex.EncodeToString(reserva.HuellaCodigo), reserva.ClaveRef, reserva.VenceUTC.UTC().Format("2006-01-02T15:04:05.000000Z07:00")}
	case ports.AccionActivarCorreo, ports.AccionRetirarCorreo:
		if sobre.CorreoRef != "" || reserva.DesafioRef != "" {
			return nil, nil, ports.ErrCorreosInvalidos
		}
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

func (r *RegistroCorreosPostgreSQL) Aplicar(ctx context.Context, orden ports.OrdenCorreos, p ports.PeticionCorreo, m ports.MaterialCorreos, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, sobre ports.SobreDireccionCorreo, reserva ports.ReservaDesafio, comprobador ports.ComprobadorCodigoCorreo) (ports.ResultadoCorreos, error) {
	var vacio ports.ResultadoCorreos
	material, err := r.validarOrdenCorreos(orden, m, v3, m.Accion)
	if err != nil {
		return vacio, err
	}
	if m.Accion == ports.AccionVerificarCorreo {
		return r.verificar(ctx, orden, p, m, material, v3, comprobador)
	}
	sb, rb, err := parametrosMutacionCorreos(p, m, sobre, reserva)
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
	if err := tx.QueryRow(ctx, r.sql.aplicar, args...).Scan(&bruto); err != nil {
		_ = tx.Rollback(context.Background())
		if serializacionCorreos(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorCorreosSeguro(ctx, err)
	}
	resultado, err := resultadoCorreosDesdeSQL(bruto, m)
	if err != nil || (!resultado.Recibo.Replay && m.Accion == ports.AccionAnadirCorreo && resultado.Recibo.CorreoRef != p.CorreoRef) {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		if serializacionCorreos(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorCorreosSeguro(ctx, err)
	}
	return resultado, nil
}

func (r *RegistroCorreosPostgreSQL) verificar(ctx context.Context, orden ports.OrdenCorreos, p ports.PeticionCorreo, m ports.MaterialCorreos, material []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, comprobador ports.ComprobadorCodigoCorreo) (ports.ResultadoCorreos, error) {
	var vacio ports.ResultadoCorreos
	if comprobador == nil || p.Codigo != "" || p.Direccion != "" || p.CorreoRef != m.CorreoRef || p.VersionEsperada != m.VersionEsperada || p.ClaveOperacion != m.ClaveOperacion {
		return vacio, ports.ErrCorreosInvalidos
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	fallo := func(err error) (ports.ResultadoCorreos, error) {
		_ = tx.Rollback(context.Background())
		if serializacionCorreos(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorCorreosSeguro(ctx, err)
	}
	var bruto []byte
	if err := tx.QueryRow(ctx, r.sql.preparar, argumentosCorreosV3(material, v3)...).Scan(&bruto); err != nil {
		return fallo(err)
	}
	var campos map[string]json.RawMessage
	if len(bruto) > maxRespuestaCorreos || json.Unmarshal(bruto, &campos) != nil || campos == nil {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if previo, ok := campos["replay"]; ok {
		if len(campos) != 1 {
			return vacio, ports.ErrCorreosNoDisponible
		}
		recibo, existe, errReplay := replayCorreos(previo, m)
		if errReplay != nil && !errors.Is(errReplay, ports.ErrCorreosCodigoIncorrecto) {
			return vacio, errReplay
		}
		if err := tx.Commit(ctx); err != nil {
			return vacio, errorCorreosSeguro(ctx, err)
		}
		if errReplay != nil {
			return vacio, errReplay
		}
		if !existe {
			return vacio, ports.ErrCorreosNoDisponible
		}
		return ports.ResultadoCorreos{Recibo: recibo}, nil
	}
	var meta struct {
		PersonaRef      string    `json:"persona_ref"`
		CorreoRef       string    `json:"correo_ref"`
		DesafioRef      string    `json:"desafio_ref"`
		HuellaCodigoHex string    `json:"huella_codigo_hex"`
		ClaveRef        string    `json:"clave_ref"`
		VenceUTC        time.Time `json:"vence_utc"`
		Intentos        int       `json:"intentos"`
	}
	if decodificarCorreosEstricto(bruto, &meta) != nil || meta.PersonaRef != m.PersonaRef || meta.CorreoRef != m.CorreoRef || !patronDesafioRef.MatchString(meta.DesafioRef) || meta.ClaveRef == "" || !fechaUTCMicro(meta.VenceUTC) || meta.Intentos < 0 || meta.Intentos >= domain.MaxIntentosCodigoCorreo {
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
	if err := tx.QueryRow(ctx, r.sql.cerrar, m.PersonaRef, m.ClaveOperacion, valido).Scan(&bruto); err != nil {
		return fallo(err)
	}
	if !valido {
		var resultado struct {
			Valido            bool `json:"valido"`
			IntentosRestantes int  `json:"intentos_restantes"`
		}
		if decodificarCorreosEstricto(bruto, &resultado) != nil || resultado.Valido || resultado.IntentosRestantes < 0 || resultado.IntentosRestantes >= domain.MaxIntentosCodigoCorreo {
			return vacio, ports.ErrCorreosNoDisponible
		}
		if err := tx.Commit(ctx); err != nil {
			return fallo(err)
		}
		return vacio, ports.CodigoIncorrecto{IntentosRestantes: resultado.IntentosRestantes}
	}
	resultado, err := resultadoCorreosDesdeSQL(bruto, m)
	if err != nil || resultado.Recibo.Replay || len(resultado.Envios) != 0 {
		return vacio, ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return fallo(err)
	}
	return resultado, nil
}

// ConfirmarEnvio anota la respuesta del relay. La reserva, que sólo conoce
// este proceso, es la llave; SQL comprueba persona, superficie y estado.
func (r *RegistroCorreosPostgreSQL) ConfirmarEnvio(ctx context.Context, orden ports.OrdenCorreos, envio ports.EnvioPendiente, aceptado bool) error {
	actor, _, superficie, err := orden.Identidad.Datos()
	if err != nil {
		return ports.ErrCorreosNoAutenticado
	}
	if r == nil || superficie != r.superficie || !patronEnvioRef.MatchString(envio.EnvioRef) || !patronReservaRef.MatchString(envio.ReservaRef) {
		return ports.ErrCorreosInvalidos
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var anotado bool
	if err := tx.QueryRow(ctx, r.sql.confirmar, actor.PersonaRef, envio.EnvioRef, envio.ReservaRef, aceptado).Scan(&anotado); err != nil {
		return errorCorreosSeguro(ctx, err)
	}
	if !anotado {
		return ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return errorCorreosSeguro(ctx, err)
	}
	return nil
}
