package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// v3SustituidoEnTransaccion reemplaza, SÓLO dentro de la transacción de la
// prueba que después se revierte, el consumo V3 y las dos revalidaciones vivas:
// la emisión atestada V3 real necesita claves del conjunto que una base
// desechable no tiene. Todo lo demás (fachadas AUT63, AUT62, CAS, historia,
// recibo y el adaptador Go con su decodificador) es lo real.
const v3SustituidoEnTransaccion = `
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb;d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('a',64),
  'aud_v3_'||md5(d->>'decision_ref'),clock_timestamp()::timestamptz,true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion.acreditar_version_rol_bolsa_v1(d jsonb)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$ SELECT true $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision_canonica bytea,
 p_motivo_canonico bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS timestamptz LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$ SELECT clock_timestamp() $f$;
`

// TestVersionarRolBolsaFachadaYAdaptadorRealesPostgreSQL18 recorre con el
// adaptador Go real y las fachadas SQL reales la propuesta del ADMIN 1, su
// replay, el cierre del ADMIN 2 y su replay. El decodificador del adaptador
// recibe exactamente lo que devuelve PostgreSQL (por ejemplo, instantes
// «+00:00» de jsonb), que los dobles en memoria no reproducen: así se cazó el
// 503 «sql_respuesta_invalida» del clon.
//
// Sólo corre contra una base desechable tras los actos ADMIN (dos ADMIN v9 y el
// catálogo B1 cargado). El DSN debe ser de un superusuario de esa copia. Todo
// ocurre en una transacción que se revierte; cada operación del adaptador va en
// un punto de guardado dentro de ella.
func TestVersionarRolBolsaFachadaYAdaptadorRealesPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_AUT62_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18 desechable tras los actos ADMIN con AUT62/AUT63")
	}
	dsn := os.Getenv("VEC_AUT62_PG_DSN")
	if dsn == "" {
		t.Fatal("falta VEC_AUT62_PG_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conexión: %v", err)
	}
	defer pool.Close()
	plan, intencion := planB1RealConAsignaciones(ctx, t, pool)
	admins := adminsV9Reales(ctx, t, pool)
	fuente, err := nuevaFuenteCatalogoAcciones(ctx, pool)
	if err != nil {
		t.Fatal("fuente:", err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, v3SustituidoEnTransaccion); err != nil {
		t.Fatal("sustitución V3:", err)
	}
	emisor := &emisorV3SustituidoPrueba{t: t}
	a := &AutoridadVersionarRolBolsa{pool: poolSobreTxPrueba{tx: tx}, catalogo: fuente, emisor: emisor,
		reloj: relojSistemaPrueba{}}

	hp, err := plan.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	proponente := admins[0]
	solicitud := domain.SolicitudPropuestaVersionarRolBolsa{OperacionRef: "propuesta_admin:" + hexAleatorio(t),
		Actor: proponente.actor, Evidencia: proponente.evidencia, InstantaneaAutorizacion: proponente.instantanea,
		Intencion: intencion, HuellaPlanEsperada: hp, CorrelacionRef: "correlacion_" + hexAleatorio(t)}
	orden := domain.OrdenPropuestaVersionarRolBolsa{Solicitud: solicitud,
		Material: domain.MaterialPropuestaVersionarRolBolsa{OperacionRef: solicitud.OperacionRef,
			ProponentePersonaRef: proponente.actor.PersonaRef, PerfilActivoRef: proponente.actor.PerfilActivoRef,
			AsignacionPerfilRef: proponente.instantanea.AsignacionPerfil.Referencia(), Plan: plan}}
	if err := orden.Validar(); err != nil {
		t.Fatal("orden de prueba:", err)
	}
	emisor.motivo = plan.Motivo
	propuesta, err := a.ProponerVersionarRolBolsa(ctx, orden)
	if err != nil {
		t.Fatalf("propuesta: %v", err)
	}
	if propuesta.Replay || propuesta.Propuesta.CaducaEn.Location() != time.UTC {
		t.Fatalf("propuesta: replay=%v caduca_en=%v", propuesta.Replay, propuesta.Propuesta.CaducaEn)
	}
	repetida, err := a.ProponerVersionarRolBolsa(ctx, orden)
	if err != nil || !repetida.Replay || repetida.Propuesta.HuellaSHA256 != propuesta.Propuesta.HuellaSHA256 ||
		!repetida.Propuesta.CaducaEn.Equal(propuesta.Propuesta.CaducaEn) {
		t.Fatalf("replay de la propuesta: %+v %v", repetida.Replay, err)
	}

	aprobador := admins[1]
	cierre := domain.SolicitudCierreVersionarRolBolsa{OperacionRef: "cierre_admin:" + hexAleatorio(t),
		PropuestaRef: orden.Material.OperacionRef, PropuestaHuellaSHA256: propuesta.Propuesta.HuellaSHA256,
		Aprobador: aprobador.actor, Evidencia: aprobador.evidencia, InstantaneaAutorizacion: aprobador.instantanea,
		Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: plan.Motivo,
		CorrelacionRef: "correlacion_" + hexAleatorio(t)}
	if err := cierre.Validar(); err != nil {
		t.Fatal("cierre de prueba:", err)
	}
	cerrado, err := a.CerrarVersionarRolBolsa(ctx, cierre)
	if err != nil || cerrado.Recibo == nil {
		t.Fatalf("cierre: %v", err)
	}
	if cerrado.Recibo.VersionRol.Referencia() != plan.VersionRolObjetivoRef ||
		len(cerrado.Recibo.Asignaciones) != len(plan.Asignaciones) {
		t.Fatalf("recibo: versión %q, %d asignaciones", cerrado.Recibo.VersionRol.Referencia(),
			len(cerrado.Recibo.Asignaciones))
	}
	otra, err := a.CerrarVersionarRolBolsa(ctx, cierre)
	if err != nil || otra.Recibo == nil || otra.Recibo.ReciboRef != cerrado.Recibo.ReciboRef ||
		!otra.ConfirmadoEn.Equal(cerrado.ConfirmadoEn) {
		t.Fatalf("replay del cierre: %v", err)
	}
}

// adminV9Real reúne, para un ADMIN v9 actual de la base, lo que la composición
// real entrega al adaptador: actor y evidencia de sesión (sintéticos, con su
// persona y perfil reales) e instantánea de autorización con sus documentos.
type adminV9Real struct {
	actor       domain.ContextoActor
	evidencia   domain.EvidenciaSesionAdministracionPerfiles
	instantanea domain.InstantaneaAutorizacion
}

func adminsV9Reales(ctx context.Context, t *testing.T, pool *pgxpool.Pool) []adminV9Real {
	t.Helper()
	var rolDoc, controlDoc string
	if err := pool.QueryRow(ctx, `SELECT r.documento::text,c.documento::text FROM vec_autorizacion.version_rol r
	 JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref)
	 WHERE r.version_rol_ref='rol:administracion_perfiles:v9' ORDER BY c.revision DESC LIMIT 1`).Scan(&rolDoc, &controlDoc); err != nil {
		t.Fatal("ADMIN v9:", err)
	}
	var rol domain.VersionRol
	var control domain.ControlVigenciaVersionRol
	if json.Unmarshal([]byte(rolDoc), &rol) != nil || json.Unmarshal([]byte(controlDoc), &control) != nil {
		t.Fatal("documentos ADMIN v9 ilegibles")
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	filas, err := pool.Query(ctx, `SELECT a.documento::text FROM vec_autorizacion.asignacion_perfil_actual q
	 JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
	 WHERE a.version_rol_ref='rol:administracion_perfiles:v9' ORDER BY a.principal_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer filas.Close()
	var salida []adminV9Real
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	for filas.Next() {
		var doc string
		if err := filas.Scan(&doc); err != nil {
			t.Fatal(err)
		}
		var asignacion domain.AsignacionPerfil
		if err := json.Unmarshal([]byte(doc), &asignacion); err != nil {
			t.Fatal(err)
		}
		resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, asignacion.PrincipalID,
			asignacion.PerfilActivoRef, domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
		if err != nil {
			t.Fatal(err)
		}
		salida = append(salida, adminV9Real{actor: resultado.Contexto,
			evidencia: domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo},
			instantanea: domain.InstantaneaAutorizacion{VersionRol: rol, AsignacionPerfil: asignacion,
				ControlVigenciaVersionRol: control, RevisionCatalogoPoliticas: 1,
				CatalogoPoliticasHuellaSHA256: huellaCatalogo}})
	}
	if err := filas.Err(); err != nil {
		t.Fatal(err)
	}
	if len(salida) < 2 || salida[0].actor.PersonaRef == salida[1].actor.PersonaRef {
		t.Fatal("la base necesita dos ADMIN v9 actuales de personas distintas")
	}
	return salida
}

// poolSobreTxPrueba abre cada transacción del adaptador como punto de
// guardado de la transacción de la prueba: su COMMIT libera el punto y la
// prueba lo revierte todo al final.
type poolSobreTxPrueba struct{ tx pgx.Tx }

func (p poolSobreTxPrueba) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	return p.tx.Begin(ctx)
}

func (p poolSobreTxPrueba) QueryRow(ctx context.Context, consulta string, args ...any) pgx.Row {
	return p.tx.QueryRow(ctx, consulta, args...)
}

type relojSistemaPrueba struct{}

func (relojSistemaPrueba) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// emisorV3SustituidoPrueba entrega el material V3 con la forma que exige el
// adaptador y con la decisión y la capacidad en JSON que la sustitución SQL
// interpreta. No firma nada: el consumo V3 real queda fuera de esta prueba.
type emisorV3SustituidoPrueba struct {
	t      *testing.T
	motivo domain.ReferenciaEntradaCatalogo
}

func (e *emisorV3SustituidoPrueba) EmitirVersionarRolBolsa(_ context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion,
	efecto Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Helper()
	recurso, err := RecursoVersionarRolBolsa(efecto, instantanea.AsignacionPerfil)
	if err != nil {
		e.t.Fatal(err)
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		e.t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	decisionRef := "decision_" + hexAleatorio(e.t)
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, strings.Repeat("a", 64),
		strings.Repeat("b", 64), evidencia.ResultadoContexto.RegistroContextoRef,
		evidencia.ResultadoContexto.HuellaSHA256, efecto.Accion, recurso.Referencia, h, efecto.Audiencia,
		ahora.Add(-time.Second), ahora.Add(4*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	decision, _ := json.Marshal(map[string]any{"decision_ref": decisionRef, "principal_id": actor.PersonaRef,
		"perfil_activo_ref": actor.PerfilActivoRef, "asignacion_ref": instantanea.AsignacionPerfil.Referencia(),
		"accion": efecto.Accion, "correlacion_ref": efecto.CorrelacionAccesoRef, "recurso_ref": recurso.Referencia,
		"contexto_recurso_huella_sha256": h})
	capacidad, _ := json.Marshal(map[string]any{"audiencia_consumo": efecto.Audiencia,
		"efecto_ref": recurso.Referencia, "huella_efecto_sha256": h})
	if relleno := ports.TamanoMinimoCapacidadCanonicaV3 - len(capacidad); relleno > 0 {
		capacidad = append(capacidad, []byte(strings.Repeat(" ", relleno))...)
	}
	motivo, _ := json.Marshal(map[string]any{"referencia": e.motivo})
	raiz, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		e.t.Fatal(err)
	}
	return ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidad, resumen, decision, motivo,
		[]byte("{}"), actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion,
		[]byte("{}"), []byte("{}"), []byte("{}"), raiz)
}

func hexAleatorio(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}
