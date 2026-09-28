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

type autoridadMiBolsaCarreraPG struct {
	base  *autoridadPostgreSQLDesarrollo
	mutar func() error
}

func (a autoridadMiBolsaCarreraPG) prepararInstantanea(ctx context.Context, i dominiovec.InstantaneaAutorizacion, inicial bool) (dominiovec.InstantaneaAutorizacion, error) {
	return a.base.prepararInstantanea(ctx, i, inicial)
}

func (a autoridadMiBolsaCarreraPG) publicarInstantaneaDesdePreimagen(ctx context.Context, i, preimagen dominiovec.InstantaneaAutorizacion) error {
	if err := a.mutar(); err != nil {
		return err
	}
	return a.base.publicarInstantaneaDesdePreimagen(ctx, i, preimagen)
}

// Requiere exclusivamente AD3/ContextoActor reales en PostgreSQL 18 efímero.
// El contexto sintético prepara las claves de actor y sesión; no prueba HTTP.
func TestMiBolsaPreimagenPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_MI_BOLSA_PG18_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, os.Getenv("VEC_MI_BOLSA_PG18_DSN_GOBIERNO"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, caso := range []string{"alta_replay_portal", "revocada", "restringida", "carrera", "carrera_control"} {
		t.Run(caso, func(t *testing.T) {
			soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			var aleatorio [8]byte
			if _, err := rand.Read(aleatorio[:]); err != nil {
				t.Fatal(err)
			}
			principal.ID += "_" + hex.EncodeToString(aleatorio[:])
			huella := sha256.Sum256(aleatorio[:])
			principal.Attributes["certificate_sha256"] = hex.EncodeToString(huella[:])
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
			if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
				t.Fatalf("contexto sintético: %v", err)
			}
			identidad := &identidadCandidatoBolsaDesarrollo{personaRef: vinculo.PrincipalID,
				perfilRef: vinculo.PerfilActivoRef, candidatoRef: "can_mibolsa_pg18_" + hex.EncodeToString(aleatorio[:])}
			a := autoridadPostgreSQLDesarrollo{pool: pool, vinculo: soporte.contexto.Vinculo,
				prefijoBloqueo: "vec:bolsa:mi-bolsa:autorizacion:",
				actoControlRol: "acto:bolsa:mi-bolsa:control-rol:v1",
				actoAsignacion: "acto:bolsa:mi-bolsa:asignacion:v1",
				actoSesion:     "acto:bolsa:mi-bolsa:sesion:v1"}
			actual := func() (string, int) {
				t.Helper()
				tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback(context.Background())
				if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
					t.Fatal(err)
				}
				var ref string
				var total int
				if err := tx.QueryRow(ctx, `SELECT asignacion_ref FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref=$1`, identidad.perfilRef).Scan(&ref); err != nil {
					t.Fatal(err)
				}
				if err := tx.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE perfil_activo_ref=$1`, identidad.perfilRef).Scan(&total); err != nil {
					t.Fatal(err)
				}
				return ref, total
			}
			primera, err := publicarPerfilMiBolsaDesarrollo(ctx, &a, identidad, ahora, false)
			if err != nil || primera.AsignacionPerfil.Version != 1 {
				t.Fatalf("alta v1: %v", err)
			}
			ref1, filas1 := actual()
			if _, err := publicarPerfilMiBolsaDesarrollo(ctx, &a, identidad, ahora, false); err != nil {
				t.Fatalf("replay v1: %v", err)
			}
			if ref, filas := actual(); ref != ref1 || filas != filas1 {
				t.Fatal("replay v1 cambió el puntero o duplicó historia")
			}
			if caso == "carrera_control" {
				control := primera.ControlVigenciaVersionRol
				control.Revision++
				control.Estado = dominiovec.EstadoControlVigenciaVersionRolRetirada
				control.ActualizadoEn = ahora.Add(time.Second)
				control.ActualizadoPor = "seguridad:prueba"
				control.ActoRef = "acto:mi-bolsa:retirada-prueba"
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
				actualizada, err := tx.Exec(ctx, `UPDATE vec_autorizacion.control_vigencia_version_rol_actual
					SET revision=$2, actualizada_en=$3, actualizada_por=$4, acto_ref=$5
					WHERE version_rol_ref=$1 AND revision=$6`, control.VersionRolRef, control.Revision,
					control.ActualizadoEn, control.ActualizadoPor, control.ActoRef, primera.ControlVigenciaVersionRol.Revision)
				if err != nil || actualizada.RowsAffected() != 1 {
					t.Fatalf("puntero control no retirado: %v", err)
				}
				resultado := make(chan error, 1)
				go func() {
					_, err := publicarPerfilMiBolsaDesarrollo(ctx, &a, identidad, ahora, false)
					resultado <- err
				}()
				// La retirada mantiene el bloqueo hasta que el replay intente
				// cotejar el control; después confirma antes que el replay.
				time.Sleep(100 * time.Millisecond)
				if err := tx.Commit(ctx); err != nil {
					t.Fatal(err)
				}
				replay := <-resultado
				if replay == nil {
					t.Fatal("replay admitido tras retirada del control de rol")
				}
				if _, err := publicarPerfilMiBolsaDesarrollo(ctx, &a, identidad, ahora, false); err == nil {
					t.Fatal("rearranque admitido con control de rol retirado")
				}
				if ref, filas := actual(); ref != ref1 || filas != filas1 {
					t.Fatal("retirada de rol movió asignación o añadió historia")
				}
				return
			}
			if caso == "alta_replay_portal" {
				portal, err := publicarPerfilMiBolsaDesarrollo(ctx, &a, identidad, ahora, true)
				if err != nil || portal.AsignacionPerfil.Version != 2 {
					t.Fatalf("legado→portal: %v", err)
				}
				ref2, filas2 := actual()
				if ref2 == ref1 || filas2 != filas1+1 {
					t.Fatal("evolución no conservó historia exacta")
				}
				if _, err := publicarPerfilMiBolsaDesarrollo(ctx, &a, identidad, ahora, true); err != nil {
					t.Fatalf("replay portal: %v", err)
				}
				if ref, filas := actual(); ref != ref2 || filas != filas2 {
					t.Fatal("replay portal duplicó historia")
				}
				return
			}
			mutada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(primera)
			if caso == "restringida" {
				mutada.AsignacionPerfil.Ambitos[0].Valores[0] = "can_restringida_" + hex.EncodeToString(aleatorio[:])
			} else {
				mutada.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
				mutada.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
				mutada.AsignacionPerfil.RevocadaEn = ahora.Add(time.Second)
				mutada.AsignacionPerfil.RevocacionRef = "revocacion:mi-bolsa-pg18"
			}
			publicarAdversaria := func() error {
				preparada, err := a.prepararInstantanea(ctx, mutada, false)
				if err != nil || preparada.AsignacionPerfil.Version != 2 {
					return errMiBolsaNoDisponible
				}
				return a.publicarInstantaneaDesdePreimagen(ctx, preparada, primera)
			}
			if caso == "carrera" {
				if _, err := publicarPerfilMiBolsaDesarrollo(ctx, autoridadMiBolsaCarreraPG{base: &a, mutar: publicarAdversaria}, identidad, ahora, true); err == nil {
					t.Fatal("Mi Bolsa ganó tras revocación entre preparación y publicación")
				}
			} else if err := publicarAdversaria(); err != nil {
				t.Fatalf("preimagen adversaria no publicada: %v", err)
			}
			refAdversaria, filasAdversarias := actual()
			if refAdversaria == ref1 || filasAdversarias != filas1+1 {
				t.Fatal("preimagen adversaria no quedó vigente")
			}
			if _, err := publicarPerfilMiBolsaDesarrollo(ctx, &a, identidad, ahora, true); err == nil {
				t.Fatal("Mi Bolsa reactivó la asignación revocada o restringida")
			}
			if ref, filas := actual(); ref != refAdversaria || filas != filasAdversarias {
				t.Fatal("fallo cerrado movió el puntero o añadió historia")
			}
		})
	}
}
