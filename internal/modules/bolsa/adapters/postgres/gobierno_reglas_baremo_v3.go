package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	ports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const operarGobiernoReglasV3SQL = `SELECT resultado,version_canonica,recibo,acceso,replay FROM vec_bolsa_reglas_baremo.operar_borrador_v3($1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

// La fachada central se inspecciona por OID de catálogo: resolver su nombre
// con regprocedure exige USAGE de un esquema que el LOGIN no debe alcanzar.
// ::name aplica exactamente el límite de identificadores de PostgreSQL, sin
// coincidencias parciales ni variantes del nombre nominal.
// #nosec G101 -- Consulta estática de metadatos del catálogo, sin credenciales ni secretos.
const acreditarGobiernoReglasV3SQL = `WITH tipos AS (
 SELECT 'pg_catalog.bytea'::pg_catalog.regtype::oid AS b,
        'pg_catalog.numeric'::pg_catalog.regtype::oid AS n,
        'pg_catalog.text'::pg_catalog.regtype::oid AS t,
        'pg_catalog.timestamptz'::pg_catalog.regtype::oid AS z,
        'pg_catalog.bool'::pg_catalog.regtype::oid AS v,
        'pg_catalog.record'::pg_catalog.regtype::oid AS r
)
 SELECT session_user=current_user AND l.rolcanlogin AND l.rolinherit
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND NOT g.rolcanlogin AND NOT(g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members WHERE member=l.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid AND m.inherit_option AND NOT m.admin_option AND NOT m.set_option)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid OR roleid=l.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace ns ON ns.oid=p.pronamespace
  JOIN pg_catalog.pg_roles o ON o.oid=p.proowner
  WHERE ns.nspname='vec_bolsa_reglas_baremo' AND p.proname='operar_borrador_v3'::pg_catalog.name
   AND p.proargtypes=ARRAY[t.b,t.b,t.b,t.b,t.b,t.n,t.n,t.b,t.b,t.b,t.b]::pg_catalog.oidvector
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND p.prosecdef AND o.rolname='vec_bolsa_reglas_baremo_propietario'
   AND ARRAY(SELECT lower(split_part(cfg,'=',1))||'='||split_part(cfg,'=',2) FROM unnest(p.proconfig) cfg) @> ARRAY['search_path=pg_catalog','row_security=on','timezone=UTC'])
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace ns ON ns.oid=p.pronamespace
  JOIN pg_catalog.pg_roles o ON o.oid=p.proowner
  WHERE ns.nspname='vec_autorizacion_atestada_v3'
   AND p.proname='registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada'::pg_catalog.name
   AND p.pronargs=10 AND p.pronargdefaults=0
   AND p.proargtypes=ARRAY[t.b,t.b,t.b,t.b,t.n,t.n,t.b,t.b,t.b,t.b]::pg_catalog.oidvector
   AND p.proretset AND p.prorettype=t.r
   AND p.proallargtypes=ARRAY[t.b,t.b,t.b,t.b,t.n,t.n,t.b,t.b,t.b,t.b,t.t,t.t,t.t,t.t,t.t,t.z,t.v]::oid[]
   AND p.proargmodes=ARRAY['i','i','i','i','i','i','i','i','i','i','t','t','t','t','t','t','t']::"char"[]
   AND p.proargnames[11:17]=ARRAY['decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo']
   AND p.prosecdef AND o.rolname='vec_autorizacion_atestada_v3_propietario'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND NOT pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND (SELECT count(*)=2 FROM pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))))
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
    WHERE a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner
     OR a.grantee NOT IN (p.proowner,(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_reglas_baremo_propietario'))))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace ns ON ns.oid=c.relnamespace
  WHERE ns.nspname='vec_bolsa_reglas_baremo' AND c.relkind IN ('r','p') AND pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname=$1 CROSS JOIN tipos t WHERE l.rolname=session_user`

// Una respuesta de COMMIT fallida puede haber confirmado el alta. Resolverla
// reintentando la MISMA intención; nunca inventar otra clave ni hacer un POST
// para una consulta. El adaptador no reintenta escrituras automáticamente.
var ErrConfirmacionGobiernoReglasV3Incierta = errors.New("gobierno_reglas_v3_confirmacion_incierta")

type transaccionGobiernoReglasV3 interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Commit(context.Context) error
	Rollback(context.Context) error
}

type RepositorioGobiernoReglasBaremoV3PostgreSQL struct {
	iniciar func(context.Context) (transaccionGobiernoReglasV3, error)
	rol     string
}

var _ ports.RepositorioGobiernoReglasBaremoV3 = (*RepositorioGobiernoReglasBaremoV3PostgreSQL)(nil)
var _ ports.ConsultaGobiernoReglasBaremoV3 = (*RepositorioGobiernoReglasBaremoV3PostgreSQL)(nil)

// Bootstrap conserva y cierra el pool. El rol técnico se fija en composición,
// jamás por HTTP. Sin función, consumidor nominal o EXECUTE falla cerrado.
// La autorización positiva por actor/perfil pertenece al broker y AD3.
func NuevoRepositorioGobiernoReglasBaremoV3PostgreSQL(ctx context.Context, pool *pgxpool.Pool, rol string) (*RepositorioGobiernoReglasBaremoV3PostgreSQL, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil || !rolGobiernoReglasV3Valido(rol) {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	r := &RepositorioGobiernoReglasBaremoV3PostgreSQL{rol: rol, iniciar: func(ctx context.Context) (transaccionGobiernoReglasV3, error) {
		return pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	}}
	tx, err := r.abrirGobiernoReglasV3(ctx)
	if err != nil {
		return nil, err
	}
	defer revertirGobiernoReglasV3(tx)
	if tx.Commit(ctx) != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	return r, nil
}
func rolGobiernoReglasV3Valido(r string) bool {
	return r == "vec_bolsa_reglas_baremo_ejecutor_gobierno" || r == "vec_bolsa_reglas_baremo_ejecutor_consulta"
}
func (r *RepositorioGobiernoReglasBaremoV3PostgreSQL) abrirGobiernoReglasV3(ctx context.Context) (transaccionGobiernoReglasV3, error) {
	if r == nil || r.iniciar == nil || ctx == nil || ctx.Err() != nil || !rolGobiernoReglasV3Valido(r.rol) {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return nil, errorGobiernoReglasV3(err)
	}
	if _, err = tx.Exec(ctx, `SET LOCAL search_path='pg_catalog'; SET LOCAL timezone='UTC'; SET LOCAL row_security='on'; SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'`); err != nil {
		revertirGobiernoReglasV3(tx)
		return nil, errorGobiernoReglasV3(err)
	}
	var acreditado bool
	if err = tx.QueryRow(ctx, acreditarGobiernoReglasV3SQL, r.rol).Scan(&acreditado); err != nil || !acreditado {
		revertirGobiernoReglasV3(tx)
		return nil, app.ErrGobiernoV3NoDisponible
	}
	return tx, nil
}
func revertirGobiernoReglasV3(tx transaccionGobiernoReglasV3) {
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
	defer cancel()
	_ = tx.Rollback(ctx)
}

func (r *RepositorioGobiernoReglasBaremoV3PostgreSQL) ConfirmarAltaBorrador(ctx context.Context, o ports.OrdenAltaBorradorReglasV3) (ports.ResultadoAltaBorradorReglasV3, error) {
	var vacio ports.ResultadoAltaBorradorReglasV3
	m, err := validarMaterialPGGobiernoV3(o.MaterialCanonico, o.HuellaMaterialSHA256, o.Autorizacion, "alta_borrador")
	if err != nil || r == nil || r.rol != "vec_bolsa_reglas_baremo_ejecutor_gobierno" {
		return vacio, app.ErrGobiernoV3Prohibido
	}
	estadoMaterial, err := estadoPGGobiernoV3(m.Estado)
	if err != nil {
		return vacio, err
	}
	if !bytes.Equal(m.VersionCanonica, o.VersionCanonica) || shaPGGobiernoV3(o.VersionCanonica) != o.HuellaVersionSHA256 || m.ClaveOperacion != o.ClaveOperacion || m.HuellaSolicitudSHA256 != o.HuellaSolicitudSHA256 || !estadoMaterial.CoincideExactamenteCon(o.EstadoPropuesto) {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	propuesta, err := restaurarVersionPGGobiernoV3(o.VersionCanonica, m.Estado, m)
	if err != nil {
		return vacio, err
	}
	tx, err := r.abrirGobiernoReglasV3(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertirGobiernoReglasV3(tx)
	estado, canon, reciboJSON, accesoJSON, replay, err := operarPGGobiernoV3(ctx, tx, o.MaterialCanonico, o.Autorizacion)
	if err != nil {
		return vacio, err
	}
	if (estado != "creado" || replay) && (estado != "replay" || !replay) {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	acceso, err := decodificarAccesoPGGobiernoV3(accesoJSON, o.Autorizacion)
	if err != nil {
		return vacio, err
	}
	recibo, err := decodificarReciboPGGobiernoV3(reciboJSON, canon, m)
	if err != nil {
		return vacio, err
	}
	original, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(canon, recibo.Estado.HuellaEstadoSHA256())
	if err != nil || !mismoNegocioPGGobiernoV3(propuesta, original) {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if !replay {
		if !bytes.Equal(canon, o.VersionCanonica) || recibo.ConsumoOriginal != acceso {
			return vacio, ports.ErrConfirmacionReglasBaremoInvalida
		}
	} else if recibo.ConsumoOriginal.DecisionRef == acceso.DecisionRef || recibo.AuditoriaRef == acceso.AuditoriaRef {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if tx.Commit(ctx) != nil {
		return vacio, ErrConfirmacionGobiernoReglasV3Incierta
	}
	return ports.ResultadoAltaBorradorReglasV3{Recibo: recibo, Acceso: acceso, Replay: replay}, nil
}

func (r *RepositorioGobiernoReglasBaremoV3PostgreSQL) ObtenerExacta(ctx context.Context, o ports.OrdenConsultaGobiernoReglasV3) (ports.ResultadoConsultaGobiernoReglasV3, error) {
	var vacio ports.ResultadoConsultaGobiernoReglasV3
	m, err := validarConsultaPGGobiernoV3(o, "consultar_exacta")
	if err != nil {
		return vacio, err
	}
	tx, err := r.abrirGobiernoReglasV3(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertirGobiernoReglasV3(tx)
	estado, canon, recibo, accesoJSON, replay, err := operarPGGobiernoV3(ctx, tx, o.MaterialCanonico, o.Autorizacion)
	if err != nil {
		return vacio, err
	}
	acceso, err := decodificarAccesoPGGobiernoV3(accesoJSON, o.Autorizacion)
	if err != nil || replay || len(recibo) != 0 {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if estado == "no_encontrada" {
		if len(canon) != 0 {
			return vacio, ports.ErrConfirmacionReglasBaremoInvalida
		}
		if tx.Commit(ctx) != nil {
			return vacio, ErrConfirmacionGobiernoReglasV3Incierta
		}
		return vacio, ports.ErrReglasBaremoNoEncontradas
	}
	if estado != "obtenida" {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if _, err = restaurarVersionPGGobiernoV3(canon, m.Estado, m); err != nil {
		return vacio, err
	}
	if tx.Commit(ctx) != nil {
		return vacio, ErrConfirmacionGobiernoReglasV3Incierta
	}
	return ports.ResultadoConsultaGobiernoReglasV3{VersionCanonica: bytes.Clone(canon), Acceso: acceso}, nil
}

func (r *RepositorioGobiernoReglasBaremoV3PostgreSQL) RecuperarRecibo(ctx context.Context, o ports.OrdenConsultaGobiernoReglasV3) (ports.ResultadoRecuperacionGobiernoReglasV3, error) {
	var vacio ports.ResultadoRecuperacionGobiernoReglasV3
	m, err := validarConsultaPGGobiernoV3(o, "recuperar_recibo")
	if err != nil {
		return vacio, err
	}
	tx, err := r.abrirGobiernoReglasV3(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertirGobiernoReglasV3(tx)
	estado, canon, reciboJSON, accesoJSON, replay, err := operarPGGobiernoV3(ctx, tx, o.MaterialCanonico, o.Autorizacion)
	if err != nil {
		return vacio, err
	}
	acceso, err := decodificarAccesoPGGobiernoV3(accesoJSON, o.Autorizacion)
	if err != nil || replay {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if estado == "no_encontrada" {
		if len(canon) != 0 || len(reciboJSON) != 0 {
			return vacio, ports.ErrConfirmacionReglasBaremoInvalida
		}
		if tx.Commit(ctx) != nil {
			return vacio, ErrConfirmacionGobiernoReglasV3Incierta
		}
		return ports.ResultadoRecuperacionGobiernoReglasV3{Acceso: acceso}, nil
	}
	if estado != "recuperado" {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	recibo, err := decodificarReciboPGGobiernoV3(reciboJSON, canon, m)
	if err != nil {
		return vacio, err
	}
	if _, err = restaurarVersionPGGobiernoV3(canon, m.Estado, m); err != nil || recibo.ConsumoOriginal.DecisionRef == acceso.DecisionRef || recibo.AuditoriaRef == acceso.AuditoriaRef {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if tx.Commit(ctx) != nil {
		return vacio, ErrConfirmacionGobiernoReglasV3Incierta
	}
	return ports.ResultadoRecuperacionGobiernoReglasV3{Recibo: recibo, Acceso: acceso, Existe: true}, nil
}

func operarPGGobiernoV3(ctx context.Context, tx transaccionGobiernoReglasV3, m []byte, v vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (string, []byte, []byte, []byte, bool, error) {
	var estado string
	var canon, recibo, acceso []byte
	var replay bool
	personaVersion, perfilVersion := v.PersonaVersion(), v.PerfilVersion()
	if personaVersion == 0 || perfilVersion == 0 || personaVersion > math.MaxInt64 || perfilVersion > math.MaxInt64 {
		return "", nil, nil, nil, false, app.ErrGobiernoV3PeticionInvalida
	}
	err := tx.QueryRow(ctx, operarGobiernoReglasV3SQL, bytes.Clone(m), v.CapacidadCanonica(), v.DecisionCanonica(), v.MotivoCanonico(), v.ContextoActorCanonico(), int64(personaVersion), int64(perfilVersion), v.PayloadVECAD3(), v.SobreCOSESign1(), v.EvidenciaVerificacion(), v.RaizPublicaSPKI()).Scan(&estado, &canon, &recibo, &acceso, &replay)
	if err != nil {
		return "", nil, nil, nil, false, errorGobiernoReglasV3(err)
	}
	// SQL NULL puede llegar como nil; JSON null no es un recibo.
	if bytes.Equal(bytes.TrimSpace(recibo), []byte("null")) {
		recibo = nil
	}
	return estado, canon, recibo, acceso, replay, nil
}

func validarConsultaPGGobiernoV3(o ports.OrdenConsultaGobiernoReglasV3, op string) (app.MaterialGobiernoV3, error) {
	m, err := validarMaterialPGGobiernoV3(o.MaterialCanonico, o.HuellaMaterialSHA256, o.Autorizacion, op)
	if err != nil {
		return app.MaterialGobiernoV3{}, err
	}
	id := o.Selector.Identidad
	estadoMaterial, err := estadoPGGobiernoV3(m.Estado)
	if err != nil {
		return app.MaterialGobiernoV3{}, err
	}
	if !estadoMaterial.CoincideExactamenteCon(o.Selector.Estado) || m.Estado.Referencia != id.Referencia() || m.Estado.Version != id.Version() || m.ConvocatoriaRef != id.ConvocatoriaRef() || m.ExpedienteRef != id.ExpedienteRef() || m.ClaveOperacion != o.ClaveOperacion || m.HuellaSolicitudSHA256 != o.HuellaSolicitudSHA256 {
		return app.MaterialGobiernoV3{}, ports.ErrConfirmacionReglasBaremoInvalida
	}
	return m, nil
}
func validarMaterialPGGobiernoV3(datos []byte, sha string, v vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, op string) (app.MaterialGobiernoV3, error) {
	var m app.MaterialGobiernoV3
	if len(datos) > 8<<20 || shaPGGobiernoV3(datos) != sha || jsonPGGobiernoV3(datos, &m, 8<<20) != nil || v.ValidarEstructura() != nil {
		return m, app.ErrGobiernoV3PeticionInvalida
	}
	canon, err := json.Marshal(m)
	if err != nil || !bytes.Equal(canon, datos) || m.Esquema != "vec.bolsa.gobierno-borrador.material.v3" || m.Operacion != op || m.ModuloID != "bolsa" || m.EstadoEsperado != nil {
		return m, app.ErrGobiernoV3PeticionInvalida
	}
	actor, err := vd.RehidratarContextoActorVinculadoV2(v.ContextoActorCanonico())
	if err != nil || actor.PersonaRef != m.PersonaRef || actor.PerfilActivoRef != m.PerfilRef || actor.Instantanea.PersonaVersion != v.PersonaVersion() || actor.Instantanea.PerfilVersion != v.PerfilVersion() {
		return m, app.ErrGobiernoV3Prohibido
	}
	accion, finalidad, tipo, prefijo, ref := "bolsa.reglas_baremo.version.consultar", "consulta_gobierno_reglas_baremo", "version_reglas_baremo_gobernada", "reglas-baremo:", m.Estado.HuellaEstadoSHA256
	if op == "alta_borrador" {
		accion, finalidad, tipo, prefijo, ref = "bolsa.reglas_baremo.borrador.crear", "gobierno_reglas_baremo", "intencion_gobierno_reglas_baremo", "intencion-reglas-baremo:", m.HuellaSolicitudSHA256
	} else if op == "recuperar_recibo" {
		accion = "bolsa.reglas_baremo.recibo.consultar"
	}
	recurso := vd.RecursoAutorizable{Referencia: prefijo + ref, ModuloID: "bolsa", Tipo: tipo, Ambitos: map[string]string{"convocatoria_ref": m.ConvocatoriaRef, "expediente_ref": m.ExpedienteRef}, Atributos: map[string]string{"material_sha256": sha}}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := v.ResumenCapacidad()
	if err != nil || m.Accion != accion || m.Finalidad != finalidad || m.TipoRecurso != tipo || resumen.Operacion() != accion || resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella || resumen.AudienciaConsumo() != app.AudienciaGobiernoBorradorReglasV3 || shaPGGobiernoV3(v.DecisionCanonica()) != resumen.DecisionHuellaSHA256() || shaPGGobiernoV3(v.ContextoActorCanonico()) != resumen.ContextoHuellaSHA256() || !bytes.Equal(m.MotivoCanonico, v.MotivoCanonico()) || shaPGGobiernoV3(v.MotivoCanonico()) != resumen.MotivoHuellaSHA256() {
		return m, app.ErrGobiernoV3Prohibido
	}
	if _, err := estadoPGGobiernoV3(m.Estado); err != nil || m.Estado.Revision != 1 {
		return m, app.ErrGobiernoV3PeticionInvalida
	}
	if op != "alta_borrador" && len(m.VersionCanonica) != 0 {
		return m, app.ErrGobiernoV3PeticionInvalida
	}
	return m, nil
}
func estadoPGGobiernoV3(s app.EstadoMaterialGobiernoV3) (reglas.VinculoEstadoReglasBaremo, error) {
	r, err := reglas.NuevaReferenciaVersionada(s.Referencia, s.Version, s.HuellaContenidoSHA256)
	if err != nil {
		return reglas.VinculoEstadoReglasBaremo{}, ports.ErrConfirmacionReglasBaremoInvalida
	}
	v, err := reglas.NuevoVinculoEstadoReglasBaremo(r, s.Revision, s.HuellaEstadoSHA256)
	if err != nil {
		return reglas.VinculoEstadoReglasBaremo{}, ports.ErrConfirmacionReglasBaremoInvalida
	}
	return v, nil
}
func restaurarVersionPGGobiernoV3(canon []byte, s app.EstadoMaterialGobiernoV3, m app.MaterialGobiernoV3) (reglas.VersionGobernadaReglasBaremo, error) {
	var vacio reglas.VersionGobernadaReglasBaremo
	v, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(canon, s.HuellaEstadoSHA256)
	if err != nil || v.Estado() != reglas.EstadoReglasBaremoBorrador || v.Revision() != 1 {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	estado, err := v.VinculoEstado()
	conjunto, ec := v.Conjunto()
	id := conjunto.Identidad()
	estadoMaterial, em := estadoPGGobiernoV3(s)
	if err != nil || ec != nil || em != nil || !estado.CoincideExactamenteCon(estadoMaterial) || id.Referencia() != s.Referencia || id.Version() != s.Version || id.ConvocatoriaRef() != m.ConvocatoriaRef || id.ExpedienteRef() != m.ExpedienteRef {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	return v, nil
}
func mismoNegocioPGGobiernoV3(a, b reglas.VersionGobernadaReglasBaremo) bool {
	ca, ea := a.Conjunto()
	cb, eb := b.Conjunto()
	if ea != nil || eb != nil {
		return false
	}
	ba, ea := ca.RepresentacionCanonica()
	bb, eb := cb.RepresentacionCanonica()
	ma, mb := a.MotivoCreacion(), b.MotivoCreacion()
	return ea == nil && eb == nil && bytes.Equal(ba, bb) && a.CreadaPor() == b.CreadaPor() && ma.Clave() == mb.Clave() && ma.Catalogo().CoincideExactamenteCon(mb.Catalogo())
}

type accesoPGGobiernoV3 struct {
	DecisionRef  string    `json:"decision_ref"`
	DecisionSHA  string    `json:"decision_huella_sha256"`
	EfectoRef    string    `json:"efecto_ref"`
	EfectoSHA    string    `json:"efecto_huella_sha256"`
	ConsumoSHA   string    `json:"consumo_huella_sha256"`
	AuditoriaRef string    `json:"auditoria_ref"`
	ConsumidaEn  time.Time `json:"consumida_en"`
}

func (a accesoPGGobiernoV3) dto() ports.EvidenciaAccesoGobiernoReglasV3 {
	return ports.EvidenciaAccesoGobiernoReglasV3{DecisionRef: a.DecisionRef, DecisionHuellaSHA256: a.DecisionSHA, EfectoRef: a.EfectoRef, EfectoHuellaSHA256: a.EfectoSHA, ConsumoHuellaSHA256: a.ConsumoSHA, AuditoriaRef: a.AuditoriaRef, ConsumidaEn: a.ConsumidaEn.UTC()}
}
func decodificarAccesoPGGobiernoV3(datos []byte, v vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EvidenciaAccesoGobiernoReglasV3, error) {
	var a accesoPGGobiernoV3
	r := v.ResumenCapacidad()
	if jsonPGGobiernoV3(datos, &a, 64<<10) != nil || !accesoPGGobiernoV3Valido(a) || a.DecisionRef != r.DecisionRef() || a.DecisionSHA != r.DecisionHuellaSHA256() || a.EfectoRef != r.EfectoRef() || a.EfectoSHA != r.EfectoHuellaSHA256() || a.ConsumidaEn.Before(r.EmitidaEn()) || !a.ConsumidaEn.Before(r.ExpiraEn()) {
		return ports.EvidenciaAccesoGobiernoReglasV3{}, ports.ErrConfirmacionReglasBaremoInvalida
	}
	return a.dto(), nil
}
func accesoPGGobiernoV3Valido(a accesoPGGobiernoV3) bool {
	return refPGGobiernoV3(a.DecisionRef) && shaPGGobiernoV3Valido(a.DecisionSHA) && refPGGobiernoV3(a.EfectoRef) && shaPGGobiernoV3Valido(a.EfectoSHA) && shaPGGobiernoV3Valido(a.ConsumoSHA) && refPGGobiernoV3(a.AuditoriaRef) && instantePGGobiernoV3(a.ConsumidaEn)
}

type reciboPGGobiernoV3 struct {
	ReciboRef       string                       `json:"recibo_ref"`
	Clave           string                       `json:"clave_operacion"`
	HuellaSolicitud string                       `json:"huella_solicitud_sha256"`
	Estado          app.EstadoMaterialGobiernoV3 `json:"estado"`
	TransaccionRef  string                       `json:"transaccion_ref"`
	AuditoriaRef    string                       `json:"auditoria_ref"`
	OutboxRef       string                       `json:"outbox_ref"`
	ConsumoOriginal accesoPGGobiernoV3           `json:"consumo_original"`
	ConfirmadaEn    time.Time                    `json:"confirmada_en"`
}

func decodificarReciboPGGobiernoV3(datos, canon []byte, m app.MaterialGobiernoV3) (ports.ReciboAltaBorradorReglasV3, error) {
	var p reciboPGGobiernoV3
	var vacio ports.ReciboAltaBorradorReglasV3
	if jsonPGGobiernoV3(datos, &p, 64<<10) != nil || !refPGGobiernoV3(p.ReciboRef) || !refPGGobiernoV3(p.TransaccionRef) || !refPGGobiernoV3(p.OutboxRef) || p.Clave != m.ClaveOperacion || p.HuellaSolicitud != m.HuellaSolicitudSHA256 || !accesoPGGobiernoV3Valido(p.ConsumoOriginal) || p.AuditoriaRef != p.ConsumoOriginal.AuditoriaRef || p.ConsumoOriginal.EfectoRef != "intencion-reglas-baremo:"+p.HuellaSolicitud || !instantePGGobiernoV3(p.ConfirmadaEn) || p.ConfirmadaEn.Before(p.ConsumoOriginal.ConsumidaEn) {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	version, err := restaurarVersionPGGobiernoV3(canon, p.Estado, m)
	if err != nil || version.CreadaPor() != m.PersonaRef || version.CreadaEn().After(p.ConfirmadaEn) {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	conjunto, err := version.Conjunto()
	if err != nil {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	estable, err := app.HuellaSolicitudAltaBorradorV3(version.CreadaPor(), app.PeticionAltaBorradorV3{Conjunto: conjunto, Motivo: version.MotivoCreacion(), ClaveOperacion: p.Clave})
	if err != nil || estable != p.HuellaSolicitud {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	estadoMaterial, err := estadoPGGobiernoV3(p.Estado)
	if err != nil {
		return vacio, err
	}
	return ports.ReciboAltaBorradorReglasV3{ReciboRef: p.ReciboRef, ClaveOperacion: p.Clave, HuellaSolicitudSHA256: p.HuellaSolicitud, Estado: estadoMaterial, VersionCanonica: bytes.Clone(canon), TransaccionRef: p.TransaccionRef, AuditoriaRef: p.AuditoriaRef, OutboxRef: p.OutboxRef, ConsumoOriginal: p.ConsumoOriginal.dto(), ConfirmadaEn: p.ConfirmadaEn.UTC()}, nil
}
func jsonPGGobiernoV3(datos []byte, destino any, maximo int) error {
	if len(datos) == 0 || len(datos) > maximo || bytes.Equal(bytes.TrimSpace(datos), []byte("null")) {
		return ports.ErrConfirmacionReglasBaremoInvalida
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return ports.ErrConfirmacionReglasBaremoInvalida
	}
	if d.Decode(new(any)) != io.EOF {
		return ports.ErrConfirmacionReglasBaremoInvalida
	}
	return nil
}
func shaPGGobiernoV3(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func shaPGGobiernoV3Valido(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && s == hex.EncodeToString(b)
}
func refPGGobiernoV3(s string) bool {
	return len(s) >= 3 && len(s) <= 256 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\r\n\t*")
}
func instantePGGobiernoV3(t time.Time) bool {
	_, z := t.Zone()
	return !t.IsZero() && z == 0 && t.Nanosecond()%1000 == 0 && t.Year() >= 1 && t.Year() <= 9999
}
func errorGobiernoReglasV3(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return ports.ErrClaveIdempotenciaReglasReutilizada
		case "40001", "P1409":
			return ports.ErrConflictoOCCReglasBaremo
		case "42501":
			return app.ErrGobiernoV3Prohibido
		case "22023":
			return app.ErrGobiernoV3PeticionInvalida
		}
	}
	return app.ErrGobiernoV3NoDisponible
}
