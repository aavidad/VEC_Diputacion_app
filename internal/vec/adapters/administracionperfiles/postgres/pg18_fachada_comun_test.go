package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// Piezas comunes de las pruebas «*_fachada_pg18_test.go»: adaptador Go y
// fachadas SQL reales contra una base desechable tras los actos ADMIN, con
// sólo el consumo V3 sustituido dentro de una transacción que se revierte.

// revalidarV3SustituidaEnTransaccion acompaña a la sustitución del consumo V3
// de cada fachada: sin decisión V3 real no hay nada vivo que revalidar.
const revalidarV3SustituidaEnTransaccion = `
CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision_canonica bytea,
 p_motivo_canonico bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS timestamptz LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$ SELECT clock_timestamp() $f$;
`

// consumoV3SustituidoSQL devuelve el CREATE OR REPLACE de un consumidor V3
// atestado con la forma común (diez argumentos y la misma tabla de salida).
func consumoV3SustituidoSQL(funcion string) string {
	return `CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.` + funcion + `(
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
`
}

// acreditacionSustituidaSQL hace que una acreditación de decisión V3 acepte la
// decisión sintética de la prueba; la decisión real la firma el emisor V3.
func acreditacionSustituidaSQL(funcion string) string {
	return `CREATE OR REPLACE FUNCTION vec_autorizacion.` + funcion + `(d jsonb)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$ SELECT true $f$;
`
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

// exportacionV3SustituidaPrueba entrega el material V3 con la forma que exige
// el adaptador y con la decisión y la capacidad en JSON que la sustitución SQL
// interpreta. No firma nada: el consumo V3 real queda fuera de estas pruebas.
func exportacionV3SustituidaPrueba(t *testing.T, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, asignacion domain.AsignacionPerfil, efecto Efecto,
	recursoRef, huella string, motivo domain.ReferenciaEntradaCatalogo) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	decisionRef := "decision_" + hexAleatorio(t)
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, strings.Repeat("a", 64),
		strings.Repeat("b", 64), evidencia.ResultadoContexto.RegistroContextoRef,
		evidencia.ResultadoContexto.HuellaSHA256, efecto.Accion, recursoRef, huella, efecto.Audiencia,
		ahora.Add(-time.Second), ahora.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decision, _ := json.Marshal(map[string]any{"decision_ref": decisionRef, "principal_id": actor.PersonaRef,
		"perfil_activo_ref": actor.PerfilActivoRef, "asignacion_ref": asignacion.Referencia(),
		"accion": efecto.Accion, "correlacion_ref": efecto.CorrelacionAccesoRef, "recurso_ref": recursoRef,
		"contexto_recurso_huella_sha256": huella})
	capacidad, _ := json.Marshal(map[string]any{"audiencia_consumo": efecto.Audiencia,
		"efecto_ref": recursoRef, "huella_efecto_sha256": huella})
	if relleno := ports.TamanoMinimoCapacidadCanonicaV3 - len(capacidad); relleno > 0 {
		capacidad = append(capacidad, []byte(strings.Repeat(" ", relleno))...)
	}
	motivoJSON, _ := json.Marshal(map[string]any{"referencia": motivo})
	raiz, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		t.Fatal(err)
	}
	return ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidad, resumen, decision, motivoJSON,
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
