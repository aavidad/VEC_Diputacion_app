package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestAsignacionActualOperativaPostgreSQLDesarrollo(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	instantanea, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"prf_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ahora.Add(-time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	base := instantanea.AsignacionPerfil
	actual := func(asignacion dominiovec.AsignacionPerfil) asignacionActualPostgreSQLDesarrollo {
		t.Helper()
		documento, err := json.Marshal(asignacion)
		if err != nil {
			t.Fatal(err)
		}
		huella, err := asignacion.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		return asignacionActualPostgreSQLDesarrollo{
			referencia: asignacion.Referencia(), identificador: asignacion.AsignacionID,
			version: int64(asignacion.Version), perfilRef: asignacion.PerfilActivoRef,
			principalID: asignacion.PrincipalID, versionRolRef: asignacion.VersionRolRef,
			huella: huella, documento: documento, actualizadaPor: asignacion.EmitidaPor,
		}
	}
	habilitada := string(dominiovec.EstadoControlVigenciaVersionRolHabilitada)
	if !asignacionActualOperativaPostgreSQLDesarrollo(actual(base), habilitada, ahora) {
		t.Fatal("una asignación activa y vigente con control habilitado debe poder evolucionar")
	}
	revocada := base
	revocada.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	revocada.RevocadaEn = ahora
	revocada.RevocadaPor = "seguridad:prueba"
	revocada.RevocacionRef = "revocacion:prueba"
	if asignacionActualOperativaPostgreSQLDesarrollo(actual(revocada), habilitada, ahora) {
		t.Fatal("la asignación revocada no puede servir de preimagen")
	}
	if asignacionActualOperativaPostgreSQLDesarrollo(actual(base),
		string(dominiovec.EstadoControlVigenciaVersionRolRetirada), ahora) {
		t.Fatal("un control de rol retirado no puede servir de preimagen ni de replay")
	}
	if asignacionActualOperativaPostgreSQLDesarrollo(actual(base), habilitada, base.VigenteHasta) {
		t.Fatal("una asignación caducada no puede servir de preimagen")
	}
	alterada := actual(base)
	alterada.huella = "huella:distinta"
	if asignacionActualOperativaPostgreSQLDesarrollo(alterada, habilitada, ahora) {
		t.Fatal("el documento debe corresponder exactamente a la huella bloqueada")
	}
	alterada = actual(base)
	alterada.actualizadaPor = "identidad:ajena"
	if asignacionActualOperativaPostgreSQLDesarrollo(alterada, habilitada, ahora) {
		t.Fatal("la procedencia del puntero debe concordar con el documento")
	}
}

// Se ejecuta solo sobre un PostgreSQL 18 efímero con las migraciones VEC
// instaladas. Cada caso utiliza identidad y perfil sintéticos distintos.
func TestGuardPublicadorPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_CT_GUARD_PG_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, os.Getenv("VEC_CT_GUARD_PG_DSN_GOBIERNO"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, caso := range []string{"activa", "restringida", "revocada", "control_retirado", "carrera_control"} {
		t.Run(caso, func(t *testing.T) {
			soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			var aleatorio [8]byte
			if _, err := rand.Read(aleatorio[:]); err != nil {
				t.Fatal(err)
			}
			principal.ID += "_" + hex.EncodeToString(aleatorio[:])
			huellaCertificado := sha256.Sum256(aleatorio[:])
			principal.Attributes["certificate_sha256"] = hex.EncodeToString(huellaCertificado[:])
			soporte.principalID = principal.ID
			soporte.certificadoSHA256 = principal.Attributes["certificate_sha256"]
			ahora := time.Now().UTC().Truncate(time.Microsecond)
			soporte.contexto, err = nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
			if err != nil {
				t.Fatal(err)
			}
			vinculo, err := soporte.contexto.Vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			soporte.instantanea, err = nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(vinculo.PrincipalID, vinculo.PerfilActivoRef, ahora)
			if err != nil {
				t.Fatal(err)
			}
			soporte.instantanea.VersionRol.RolID += "_" + hex.EncodeToString(aleatorio[:])
			soporte.instantanea.ControlVigenciaVersionRol.VersionRolRef = soporte.instantanea.VersionRol.Referencia()
			soporte.instantanea.AsignacionPerfil.VersionRolRef = soporte.instantanea.VersionRol.Referencia()
			if err := soporte.instantanea.Validar(); err != nil {
				t.Fatal(err)
			}
			if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
				t.Fatal(err)
			}
			autoridad := (&autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}).autoridadComun()
			inicial := soporte.instantanea
			alta := autoridad
			alta.soloInicial = true
			if err := alta.publicarInstantanea(ctx, inicial); err != nil {
				t.Fatalf("alta inicial: %v", err)
			}
			contar := func() (string, int) {
				t.Helper()
				tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback(context.Background())
				if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
					t.Fatal(err)
				}
				var referencia string
				var total int
				err = tx.QueryRow(ctx, `SELECT vigente.asignacion_ref,
					(SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE perfil_activo_ref=$1)
					FROM vec_autorizacion.asignacion_perfil_actual AS vigente
					WHERE vigente.perfil_activo_ref=$1`, vinculo.PerfilActivoRef).Scan(&referencia, &total)
				if err != nil {
					t.Fatal(err)
				}
				return referencia, total
			}
			objetivo := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(inicial)
			objetivo.AsignacionPerfil.Ambitos = append(objetivo.AsignacionPerfil.Ambitos,
				dominiovec.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{"expediente:prueba:guard"}})
			preparada, err := autoridad.prepararInstantanea(ctx, objetivo, false)
			if err != nil || preparada.AsignacionPerfil.Version != 2 {
				t.Fatalf("preparación activa: %v", err)
			}
			switch caso {
			case "activa":
				if err := autoridad.publicarInstantaneaDesdePreimagen(ctx, preparada, inicial); err != nil {
					t.Fatalf("publicación activa: %v", err)
				}
				if err := autoridad.publicarInstantanea(ctx, preparada); err != nil {
					t.Fatalf("replay exacto activo: %v", err)
				}
				if ref, n := contar(); ref != preparada.AsignacionPerfil.Referencia() || n != 2 {
					t.Fatal("la evolución o replay activo alteró la historia")
				}
				return
			case "restringida", "revocada":
				administrativa := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(inicial)
				if caso == "restringida" {
					administrativa.AsignacionPerfil.Ambitos[0].Valores[0] = "organizacion:prueba:restringida"
				} else {
					administrativa.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
					administrativa.AsignacionPerfil.RevocadaEn = ahora.Add(time.Second)
					administrativa.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
					administrativa.AsignacionPerfil.RevocacionRef = "revocacion:prueba:guard"
				}
				administrativa, err = autoridad.prepararInstantanea(ctx, administrativa, false)
				autoridadAdministrativa := autoridad
				autoridadAdministrativa.actoAsignacion = "acto:prueba:restriccion-expresa"
				if err != nil || autoridadAdministrativa.publicarInstantaneaDesdePreimagen(ctx, administrativa, inicial) != nil {
					t.Fatalf("acto administrativo previo: %v", err)
				}
				if err := autoridad.publicarInstantanea(ctx, preparada); err == nil {
					t.Fatal("publicación obsoleta rehabilitó autoridad cambiada")
				}
				if caso == "revocada" {
					if _, err := autoridad.prepararInstantanea(ctx, objetivo, false); err == nil {
						t.Fatal("preparación desde asignación revocada")
					}
					if err := autoridad.publicarInstantanea(ctx, administrativa); err == nil {
						t.Fatal("replay encubrió asignación revocada")
					}
				} else {
					posterior, err := autoridad.prepararInstantanea(ctx, objetivo, false)
					if err != nil || posterior.AsignacionPerfil.Version != 3 {
						t.Fatalf("la prueba no preparó candidato tras restricción: %v", err)
					}
					if err := autoridad.publicarInstantanea(ctx, posterior); err == nil {
						t.Fatal("el publicador sin preimagen revivió ámbito restringido")
					}
				}
				if ref, n := contar(); ref != administrativa.AsignacionPerfil.Referencia() || n != 2 {
					t.Fatal("el rechazo movió el puntero o añadió historia")
				}
				return
			}
			control := inicial.ControlVigenciaVersionRol
			control.Revision++
			control.Estado = dominiovec.EstadoControlVigenciaVersionRolRetirada
			control.ActualizadoEn = ahora.Add(time.Second)
			control.ActualizadoPor = "seguridad:prueba"
			control.ActoRef = "acto:prueba:retirada-guard"
			control.MotivoCodigo = "baja"
			huellaControl, err := control.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			documento, err := json.Marshal(control)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(context.Background())
			if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO vec_autorizacion.control_vigencia_version_rol
				(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
				VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, control.VersionRolRef, control.Revision,
				string(control.Estado), huellaControl, control.ActualizadoEn, documento); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `UPDATE vec_autorizacion.control_vigencia_version_rol_actual
				SET revision=$2, actualizada_en=$3, actualizada_por=$4, acto_ref=$5
				WHERE version_rol_ref=$1`, control.VersionRolRef, control.Revision,
				control.ActualizadoEn, control.ActualizadoPor, control.ActoRef); err != nil {
				t.Fatal(err)
			}
			if caso == "carrera_control" {
				resultado := make(chan error, 1)
				go func() { resultado <- autoridad.publicarInstantanea(ctx, preparada) }()
				time.Sleep(100 * time.Millisecond)
				if err := tx.Commit(ctx); err != nil {
					t.Fatal(err)
				}
				if err := <-resultado; err == nil {
					t.Fatal("la publicación ganó después de retirarse el control")
				}
			} else if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := autoridad.prepararInstantanea(ctx, objetivo, false); err == nil {
				t.Fatal("preparación desde control retirado")
			}
			if err := autoridad.publicarInstantanea(ctx, preparada); err == nil {
				t.Fatal("publicación desde control retirado")
			}
			if err := autoridad.publicarInstantanea(ctx, inicial); err == nil {
				t.Fatal("replay ocultó control retirado")
			}
			if ref, n := contar(); ref != inicial.AsignacionPerfil.Referencia() || n != 1 {
				t.Fatal("la retirada movió el puntero o añadió historia")
			}
		})
	}
}
