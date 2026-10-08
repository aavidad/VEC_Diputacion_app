// Adaptador PostgreSQL de «Mi imagen». El pool se conecta con el LOGIN
// ejecutor exclusivo de la superficie; cada transacción vuelve a comprobarlo.
// Los bytes de la foto solo cruzan este adaptador hacia/desde las fachadas
// SQL; nunca se registran ni se incluyen en errores.
package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
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

const (
	catalogoImagenSQL  = `SELECT vec_usuarios.catalogo_vigente_imagen_v1($1::text)`
	consultarImagenSQL = `SELECT datos,foto FROM vec_usuarios.consultar_imagen_propia_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	recuperarImagenSQL = `SELECT vec_usuarios.recuperar_imagen_operacion_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	guardarImagenSQL   = `SELECT vec_usuarios.guardar_imagen_propia_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
	maxRespuestaImagen = 16 << 10
	// Los seis ajustes mantienen los valores y el alcance de SET LOCAL.
	ajustesTransaccionImagenSQL = `SELECT pg_catalog.set_config('search_path','pg_catalog',true),
 pg_catalog.set_config('row_security','on',true),
 pg_catalog.set_config('timezone','UTC',true),
 pg_catalog.set_config('lock_timeout','3s',true),
 pg_catalog.set_config('statement_timeout','15s',true),
 pg_catalog.set_config('idle_in_transaction_session_timeout','20s',true)`
)

// El LOGIN sólo puede heredar el ejecutor de su superficie y ejecutar las
// fachadas de imagen; ni tablas propias ni funciones de Documentos.
const acreditarEjecutorImagenSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND g.rolname=$1::text AND NOT g.rolcanlogin AND g.rolinherit AND NOT g.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER') AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)=1
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=g.oid)
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios.catalogo_vigente_imagen_v1(text)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios.consultar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios.recuperar_imagen_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios.guardar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT pg_catalog.has_table_privilege(session_user,'vec_usuarios.imagen_actual','SELECT')
 AND NOT pg_catalog.has_schema_privilege(session_user,'vec_documentos','USAGE')
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname=$1::text
 WHERE l.rolname=session_user`

var patronReciboImagen = regexp.MustCompile(`^img_[0-9a-f]{32}$`)

type RegistroImagenPostgreSQL struct {
	iniciar    func(context.Context) (transaccionCorreos, error)
	superficie vecdomain.SuperficieAutenticacionActorV1
}

var _ ports.RegistroImagen = (*RegistroImagenPostgreSQL)(nil)

// NuevoRegistroImagenPostgreSQL no posee ni cierra el pool. Su sonda usa el
// mismo camino SERIALIZABLE que las operaciones.
func NuevoRegistroImagenPostgreSQL(ctx context.Context, pool *pgxpool.Pool, superficie vecdomain.SuperficieAutenticacionActorV1) (*RegistroImagenPostgreSQL, error) {
	if ctx == nil || pool == nil || ctx.Err() != nil || rolEjecutorPreferencias(superficie) == "" {
		return nil, ports.ErrImagenNoDisponible
	}
	r := &RegistroImagenPostgreSQL{superficie: superficie, iniciar: func(ctx context.Context) (transaccionCorreos, error) {
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
		return nil, errorImagenSeguro(ctx, err)
	}
	return r, nil
}

func (r *RegistroImagenPostgreSQL) abrir(ctx context.Context) (transaccionCorreos, error) {
	if ctx == nil || r == nil || r.iniciar == nil || rolEjecutorPreferencias(r.superficie) == "" {
		return nil, ports.ErrImagenNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return nil, errorImagenSeguro(ctx, err)
	}
	if tx == nil {
		return nil, ports.ErrImagenNoDisponible
	}
	fallar := func(err error) (transaccionCorreos, error) {
		_ = tx.Rollback(context.Background())
		return nil, errorImagenSeguro(ctx, err)
	}
	if _, err := tx.Exec(ctx, ajustesTransaccionImagenSQL); err != nil {
		return fallar(err)
	}
	var valido bool
	if err := tx.QueryRow(ctx, acreditarEjecutorImagenSQL, rolEjecutorPreferencias(r.superficie)).Scan(&valido); err != nil {
		return fallar(err)
	}
	if !valido {
		_ = tx.Rollback(context.Background())
		return nil, ports.ErrImagenNoDisponible
	}
	return tx, nil
}

func (r *RegistroImagenPostgreSQL) CatalogoVigente(ctx context.Context, orden ports.OrdenImagen) (domain.CatalogoImagen, error) {
	var vacio domain.CatalogoImagen
	_, _, superficie, err := orden.Identidad.Datos()
	if err != nil {
		return vacio, ports.ErrImagenNoAutenticado
	}
	if r == nil || superficie != r.superficie {
		return vacio, ports.ErrImagenProhibido
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, catalogoImagenSQL, string(superficie)).Scan(&bruto); err != nil {
		return vacio, errorImagenSeguro(ctx, err)
	}
	var c domain.CatalogoImagen
	if decodificarImagenEstricto(bruto, &c) != nil || c.Validar() != nil {
		return vacio, ports.ErrImagenNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorImagenSeguro(ctx, err)
	}
	return c, nil
}

// validarOrdenImagen repite en el adaptador el cotejo de identidad,
// superficie, recurso y capacidad antes de abrir una transacción.
func (r *RegistroImagenPostgreSQL) validarOrdenImagen(orden ports.OrdenImagen, m ports.MaterialImagen, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion string) ([]byte, error) {
	if r == nil || rolEjecutorPreferencias(r.superficie) == "" {
		return nil, ports.ErrImagenNoDisponible
	}
	actor, _, superficie, err := orden.Identidad.Datos()
	if err != nil {
		return nil, ports.ErrImagenNoAutenticado
	}
	audiencia, err := canonico.AudienciaImagen(accion, superficie)
	if err != nil {
		return nil, ports.ErrImagenProhibido
	}
	recurso, err := canonico.RecursoImagen(m)
	if err != nil {
		return nil, ports.ErrImagenPeticionInvalida
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return nil, ports.ErrImagenPeticionInvalida
	}
	resumen := v3.ResumenCapacidad()
	if m.PersonaRef != actor.PersonaRef || m.PerfilRef != actor.PerfilActivoRef || m.Accion != accion ||
		m.Superficie != superficie || superficie != r.superficie || m.VersionEsperada >= math.MaxInt64 ||
		m.FinalidadRef != ports.FinalidadImagenPropia || v3.ValidarEstructura() != nil ||
		v3.PersonaVersion() != actor.Instantanea.PersonaVersion || v3.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia ||
		resumen.EfectoRef() != m.PersonaRef || resumen.EfectoHuellaSHA256() != huella {
		return nil, ports.ErrImagenProhibido
	}
	return canonico.SerializarMaterialImagen(m)
}

func argumentosImagenV3(material []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) []any {
	return []any{string(material), v3.CapacidadCanonica(), v3.DecisionCanonica(), v3.MotivoCanonico(), v3.ContextoActorCanonico(), int64(v3.PersonaVersion()), int64(v3.PerfilVersion()), v3.PayloadVECAD3(), v3.SobreCOSESign1(), v3.EvidenciaVerificacion(), v3.RaizPublicaSPKI()}
}

type estadoImagenSQL struct {
	Existe             bool                  `json:"existe"`
	PersonaRef         string                `json:"persona_ref"`
	Version            uint64                `json:"version"`
	CatalogoVersionRef string                `json:"catalogo_version_ref"`
	Eleccion           domain.EleccionImagen `json:"eleccion"`
}

func (r *RegistroImagenPostgreSQL) Consultar(ctx context.Context, orden ports.OrdenImagen, m ports.MaterialImagen, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EstadoImagen, bool, *ports.FotoImagen, error) {
	var vacio ports.EstadoImagen
	material, err := r.validarOrdenImagen(orden, m, v3, ports.AccionConsultarImagen)
	if err != nil {
		return vacio, false, nil, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, false, nil, err
	}
	defer tx.Rollback(context.Background())
	var bruto, foto []byte
	if err := tx.QueryRow(ctx, consultarImagenSQL, argumentosImagenV3(material, v3)...).Scan(&bruto, &foto); err != nil {
		return vacio, false, nil, errorImagenSeguro(ctx, err)
	}
	var e estadoImagenSQL
	if decodificarImagenEstricto(bruto, &e) != nil || e.PersonaRef != m.PersonaRef || e.Eleccion.ValidarCodigos() != nil ||
		e.Version >= math.MaxInt64 || (e.Existe != (e.Version > 0)) || e.CatalogoVersionRef == "" ||
		len(foto) > ports.TamanoMaximoFotoCustodia || (len(foto) > 0 && e.Eleccion.Modo != domain.ModoImagenFoto) {
		return vacio, false, nil, ports.ErrImagenNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, false, nil, errorImagenSeguro(ctx, err)
	}
	var f *ports.FotoImagen
	if len(foto) > 0 {
		f = &ports.FotoImagen{Tipo: ports.TipoFotoImagen, Datos: foto}
	}
	return ports.EstadoImagen{PersonaRef: e.PersonaRef, Version: e.Version, CatalogoVersionRef: e.CatalogoVersionRef, Eleccion: e.Eleccion}, e.Existe, f, nil
}

func (r *RegistroImagenPostgreSQL) RecuperarOperacion(ctx context.Context, orden ports.OrdenImagen, m ports.MaterialImagen, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboImagen, bool, error) {
	var vacio ports.ReciboImagen
	material, err := r.validarOrdenImagen(orden, m, v3, ports.AccionActualizarImagen)
	if err != nil {
		return vacio, false, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, false, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, recuperarImagenSQL, argumentosImagenV3(material, v3)...).Scan(&bruto); err != nil {
		return vacio, false, errorImagenSeguro(ctx, err)
	}
	if len(bruto) == 0 || bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) {
		if err := tx.Commit(ctx); err != nil {
			return vacio, false, errorImagenSeguro(ctx, err)
		}
		return vacio, false, nil
	}
	recibo, err := reciboImagenDesdeSQL(bruto, m)
	if err != nil || !recibo.Replay {
		return vacio, false, ports.ErrImagenNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, false, errorImagenSeguro(ctx, err)
	}
	return recibo, true, nil
}

func (r *RegistroImagenPostgreSQL) Guardar(ctx context.Context, orden ports.OrdenImagen, m ports.MaterialImagen, foto []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboImagen, error) {
	var vacio ports.ReciboImagen
	material, err := r.validarOrdenImagen(orden, m, v3, ports.AccionActualizarImagen)
	if err != nil {
		return vacio, err
	}
	if (m.FotoSHA256 == "") != (len(foto) == 0) || len(foto) > ports.TamanoMaximoFotoCustodia {
		return vacio, ports.ErrImagenPeticionInvalida
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	var fotoSQL any
	if len(foto) > 0 {
		fotoSQL = foto
	}
	argumentos := argumentosImagenV3(material, v3)
	argumentos = append(argumentos[:1], append([]any{fotoSQL}, argumentos[1:]...)...)
	var bruto []byte
	if err := tx.QueryRow(ctx, guardarImagenSQL, argumentos...).Scan(&bruto); err != nil {
		_ = tx.Rollback(context.Background())
		if serializacionImagen(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorImagenSeguro(ctx, err)
	}
	recibo, err := reciboImagenDesdeSQL(bruto, m)
	if err != nil || recibo.Replay || recibo.FotoNueva != (len(foto) > 0) {
		return vacio, ports.ErrImagenNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		if serializacionImagen(err) {
			return r.recuperarTrasSerializacion(ctx, orden, m)
		}
		return vacio, errorImagenSeguro(ctx, err)
	}
	return recibo, nil
}

// Una carrera SERIALIZABLE solo se resuelve si una transacción nueva con V3
// fresco recupera el recibo ya confirmado. Sin recibo propio, la carrera la
// ganó otra operación y se responde conflicto; 40001 jamás acredita el efecto.
func (r *RegistroImagenPostgreSQL) recuperarTrasSerializacion(ctx context.Context, orden ports.OrdenImagen, m ports.MaterialImagen) (ports.ReciboImagen, error) {
	var vacio ports.ReciboImagen
	if ctx == nil || ctx.Err() != nil || orden.Proveedor == nil {
		return vacio, ports.ErrImagenNoDisponible
	}
	_, vinculo, _, err := orden.Identidad.Datos()
	if err != nil {
		return vacio, ports.ErrImagenNoAutenticado
	}
	v3, err := orden.Proveedor.ProveerMaterialImagen(ctx, vinculo, m)
	if err != nil || v3.ValidarEstructura() != nil {
		return vacio, ports.ErrImagenNoDisponible
	}
	recibo, existe, err := r.RecuperarOperacion(ctx, orden, m, v3)
	if err != nil {
		return vacio, ports.ErrImagenNoDisponible
	}
	if !existe || !recibo.Replay {
		return vacio, ports.ErrImagenConflicto
	}
	return recibo, nil
}

type reciboImagenSQL struct {
	ReciboRef          string                `json:"recibo_ref"`
	PersonaRef         string                `json:"persona_ref"`
	Version            uint64                `json:"version"`
	CatalogoVersionRef string                `json:"catalogo_version_ref"`
	Eleccion           domain.EleccionImagen `json:"eleccion"`
	FotoNueva          bool                  `json:"foto_nueva"`
	FotoRetirada       bool                  `json:"foto_retirada"`
	FechaUTC           time.Time             `json:"fecha_utc"`
	Replay             bool                  `json:"replay"`
}

func reciboImagenDesdeSQL(bruto []byte, m ports.MaterialImagen) (ports.ReciboImagen, error) {
	var r reciboImagenSQL
	if decodificarImagenEstricto(bruto, &r) != nil || !patronReciboImagen.MatchString(r.ReciboRef) || r.PersonaRef != m.PersonaRef ||
		r.Version != m.VersionEsperada+1 || r.CatalogoVersionRef != m.CatalogoVersionRef || r.Eleccion != m.Eleccion ||
		!fechaUTCMicro(r.FechaUTC) {
		return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
	}
	return ports.ReciboImagen{ReciboRef: r.ReciboRef, PersonaRef: r.PersonaRef, Version: r.Version, CatalogoVersionRef: r.CatalogoVersionRef,
		Eleccion: r.Eleccion, FotoNueva: r.FotoNueva, FotoRetirada: r.FotoRetirada, FechaUTC: r.FechaUTC.UTC(), Replay: r.Replay}, nil
}

func decodificarImagenEstricto(bruto []byte, destino any) error {
	if len(bruto) == 0 || len(bruto) > maxRespuestaImagen {
		return ports.ErrImagenNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ports.ErrImagenNoDisponible
	}
	return nil
}

func errorImagenSeguro(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ports.ErrImagenProhibido
		case "P1409":
			return ports.ErrImagenConflicto
		case "22023":
			return ports.ErrImagenPeticionInvalida
		}
	}
	return ports.ErrImagenNoDisponible
}

func serializacionImagen(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "40001"
}
