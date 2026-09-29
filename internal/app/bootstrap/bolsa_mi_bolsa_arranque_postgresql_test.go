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

	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Requiere un PostgreSQL 18 desechable y recién restaurado (por ejemplo, un
// clon de la principal en /dev/shm); nunca una base conservada. El fichero se
// ordena antes que bolsa_mi_bolsa_postgresql_test.go porque aquel retira el
// control de la versión 1 del rol de consulta, que comparten los perfiles.
// VEC_MI_BOLSA_PG18_DSN_FUENTE (opcional) es un rol de lectura de la fuente
// de autorización V3; con él se coteja lo que ve la fuente oficial.
type escenarioArranqueMiBolsaPrueba struct {
	ctx       context.Context
	pool      *pgxpool.Pool
	fuente    *postgresvec.AlmacenAutorizacion
	autoridad autoridadPostgreSQLDesarrollo
	identidad *identidadCandidatoBolsaDesarrollo
	ahora     time.Time
}

func nuevoEscenarioArranqueMiBolsaPrueba(t *testing.T, ctx context.Context, pool, fuente *pgxpool.Pool) escenarioArranqueMiBolsaPrueba {
	t.Helper()
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
	var err error
	if soporte.contexto, err = nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora); err != nil {
		t.Fatal(err)
	}
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
		t.Fatalf("contexto sintético: %v", err)
	}
	e := escenarioArranqueMiBolsaPrueba{ctx: ctx, pool: pool, ahora: ahora,
		identidad: &identidadCandidatoBolsaDesarrollo{personaRef: vinculo.PrincipalID, perfilRef: vinculo.PerfilActivoRef,
			candidatoRef: "can_arranque_pg18_" + hex.EncodeToString(aleatorio[:])},
		autoridad: autoridadPostgreSQLDesarrollo{pool: pool, vinculo: soporte.contexto.Vinculo,
			prefijoBloqueo: "vec:bolsa:mi-bolsa:autorizacion:",
			actoControlRol: "acto:bolsa:mi-bolsa:control-rol:v1",
			actoAsignacion: "acto:bolsa:mi-bolsa:asignacion:v1",
			actoSesion:     "acto:bolsa:mi-bolsa:sesion:v1"}}
	if fuente != nil {
		if e.fuente, err = postgresvec.NuevoAlmacenAutorizacion(fuente); err != nil {
			t.Fatal(err)
		}
	}
	return e
}

// historia devuelve el puntero vigente y el número de versiones del perfil.
func (e escenarioArranqueMiBolsaPrueba) historia(t *testing.T) (string, int) {
	t.Helper()
	tx, err := e.pool.BeginTx(e.ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(e.ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
		t.Fatal(err)
	}
	var ref string
	var total int
	if err := tx.QueryRow(e.ctx, `SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE perfil_activo_ref=$1`, e.identidad.perfilRef).Scan(&total); err != nil {
		t.Fatal(err)
	}
	_ = tx.QueryRow(e.ctx, `SELECT asignacion_ref FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref=$1`, e.identidad.perfilRef).Scan(&ref)
	return ref, total
}

func (e escenarioArranqueMiBolsaPrueba) arrancar(portal bool, aprobacion aprobacionProvisionMiBolsaDesarrollo) (dominiovec.InstantaneaAutorizacion, estadoPerfilMiBolsaDesarrollo, error) {
	a := e.autoridad
	return asegurarPerfilMiBolsaDesarrollo(e.ctx, &a, e.identidad, e.ahora, portal, aprobacion)
}

// publicarAjena publica, con los mismos actos del circuito (el peor caso),
// una asignación distinta contra la preimagen vigente exacta.
func (e escenarioArranqueMiBolsaPrueba) publicarAjena(t *testing.T, preimagen, mutada dominiovec.InstantaneaAutorizacion) {
	t.Helper()
	preparada, err := e.autoridad.prepararInstantanea(e.ctx, mutada, false)
	if err != nil || preparada.AsignacionPerfil.Version != preimagen.AsignacionPerfil.Version+1 {
		t.Fatalf("preparación ajena: %v", err)
	}
	if err := e.autoridad.publicarInstantaneaDesdePreimagen(e.ctx, preparada, preimagen); err != nil {
		t.Fatalf("publicación ajena: %v", err)
	}
}

func (e escenarioArranqueMiBolsaPrueba) huellaVigente(t *testing.T) string {
	t.Helper()
	vigente, encontrada, err := leerInstantaneaVigenteMiBolsaDesarrollo(e.ctx, e.pool, e.identidad.perfilRef)
	if err != nil || !encontrada {
		t.Fatalf("sin asignación vigente: %v", err)
	}
	return vigente.huella
}

// retirarRolVigente retira, como lo haría seguridad con su propio acto, el
// control de la versión de rol de la asignación vigente.
func (e escenarioArranqueMiBolsaPrueba) retirarRolVigente(t *testing.T) {
	t.Helper()
	vigente, encontrada, err := leerInstantaneaVigenteMiBolsaDesarrollo(e.ctx, e.pool, e.identidad.perfilRef)
	if err != nil || !encontrada {
		t.Fatalf("sin asignación vigente: %v", err)
	}
	control := vigente.instantanea.ControlVigenciaVersionRol
	previa := control.Revision
	control.Revision++
	control.Estado = dominiovec.EstadoControlVigenciaVersionRolRetirada
	control.ActualizadoEn = e.ahora.Add(time.Second)
	control.ActualizadoPor = "seguridad:prueba"
	control.ActoRef = "acto:mi-bolsa:retirada-arranque"
	control.MotivoCodigo = "baja"
	huella, err := control.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	documento, err := json.Marshal(control)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := e.pool.BeginTx(e.ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(e.ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(e.ctx, `INSERT INTO vec_autorizacion.control_vigencia_version_rol
		(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, control.VersionRolRef, control.Revision,
		string(control.Estado), huella, control.ActualizadoEn, documento); err != nil {
		t.Fatal(err)
	}
	actualizada, err := tx.Exec(e.ctx, `UPDATE vec_autorizacion.control_vigencia_version_rol_actual
		SET revision=$2, actualizada_en=$3, actualizada_por=$4, acto_ref=$5
		WHERE version_rol_ref=$1 AND revision=$6`, control.VersionRolRef, control.Revision,
		control.ActualizadoEn, control.ActualizadoPor, control.ActoRef, previa)
	if err != nil || actualizada.RowsAffected() != 1 || tx.Commit(e.ctx) != nil {
		t.Fatalf("control no retirado: %v", err)
	}
}

func TestArranqueMiBolsaNoReactivaPostgreSQL18(t *testing.T) {
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
	var fuente *pgxpool.Pool
	if dsn := os.Getenv("VEC_MI_BOLSA_PG18_DSN_FUENTE"); dsn != "" {
		if fuente, err = pgxpool.New(ctx, dsn); err != nil {
			t.Fatal(err)
		}
		defer fuente.Close()
	}
	sin := aprobacionProvisionMiBolsaDesarrollo{}

	t.Run("reinicio_sin_escribir", func(t *testing.T) {
		e := nuevoEscenarioArranqueMiBolsaPrueba(t, ctx, pool, fuente)
		primera, estado, err := e.arrancar(true, sin)
		if err != nil || estado != perfilMiBolsaPublicado || primera.AsignacionPerfil.Version != 1 {
			t.Fatalf("primer arranque: %v %s", err, estado)
		}
		ref, filas := e.historia(t)
		for range 3 {
			tras, estado, err := e.arrancar(true, sin)
			if err != nil || estado != perfilMiBolsaVigente || !mismasHuellasInstantaneaMiBolsa(tras, primera) {
				t.Fatalf("reinicio: %v %s", err, estado)
			}
		}
		if r, f := e.historia(t); r != ref || f != filas {
			t.Fatal("el reinicio escribió en el permiso")
		}
		if e.fuente != nil {
			oficial, err := e.fuente.ObtenerInstantaneaAutorizacion(ctx, e.identidad.personaRef, e.identidad.perfilRef)
			if err != nil || !mismasHuellasInstantaneaMiBolsa(oficial, primera) {
				t.Fatalf("la fuente V3 no ve el permiso que usa Mi Bolsa: %v", err)
			}
		}
	})

	t.Run("consulta_a_portal_y_reinicio", func(t *testing.T) {
		e := nuevoEscenarioArranqueMiBolsaPrueba(t, ctx, pool, fuente)
		if _, estado, err := e.arrancar(false, sin); err != nil || estado != perfilMiBolsaPublicado {
			t.Fatalf("consulta: %v %s", err, estado)
		}
		portal, estado, err := e.arrancar(true, sin)
		if err != nil || estado != perfilMiBolsaPublicado || portal.AsignacionPerfil.Version != 2 {
			t.Fatalf("consulta→portal: %v %s", err, estado)
		}
		ref, filas := e.historia(t)
		if _, estado, err := e.arrancar(true, sin); err != nil || estado != perfilMiBolsaVigente {
			t.Fatalf("reinicio portal: %v %s", err, estado)
		}
		if r, f := e.historia(t); r != ref || f != filas {
			t.Fatal("el reinicio del portal escribió")
		}
	})

	for _, caso := range []string{"revocada", "restringida", "vigencia_acortada"} {
		t.Run(caso+"_no_revive_ni_con_aprobacion", func(t *testing.T) {
			e := nuevoEscenarioArranqueMiBolsaPrueba(t, ctx, pool, fuente)
			primera, _, err := e.arrancar(true, sin)
			if err != nil {
				t.Fatal(err)
			}
			mutada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(primera)
			switch caso {
			case "revocada":
				mutada.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
				mutada.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
				mutada.AsignacionPerfil.RevocadaEn = e.ahora.Add(time.Second)
				mutada.AsignacionPerfil.RevocacionRef = "revocacion:mi-bolsa-arranque"
			case "restringida":
				mutada.AsignacionPerfil.Ambitos[0].Valores[0] = e.identidad.candidatoRef + "_otro"
			default:
				mutada.AsignacionPerfil.VigenteHasta = e.ahora.Add(time.Hour)
			}
			e.publicarAjena(t, primera, mutada)
			ref, filas := e.historia(t)
			huella := e.huellaVigente(t)
			for _, aprobacion := range []aprobacionProvisionMiBolsaDesarrollo{sin, {referencia: "aprobacion:bolsa:prueba", preimagen: huella}} {
				obtenida, estado, err := e.arrancar(true, aprobacion)
				if err != nil || estado != perfilMiBolsaPendienteProvision || obtenida.Validar() == nil {
					t.Fatalf("arranque sobre %s: %v %s", caso, err, estado)
				}
			}
			if r, f := e.historia(t); r != ref || f != filas {
				t.Fatalf("el arranque tocó una asignación %s", caso)
			}
		})
	}

	// Con la aprobación exacta puesta, un rol retirado o un puntero movido
	// por otro acto siguen sin restaurarse aunque por lo demás lo fueran.
	for _, caso := range []string{"rol_retirado", "otro_acto"} {
		t.Run(caso+"_no_se_restaura_con_aprobacion", func(t *testing.T) {
			e := nuevoEscenarioArranqueMiBolsaPrueba(t, ctx, pool, fuente)
			primera, _, err := e.arrancar(false, sin)
			if err != nil {
				t.Fatal(err)
			}
			anterior, err := nuevaInstantaneaMiBolsaDesarrollo(e.identidad, e.ahora, true)
			if err != nil {
				t.Fatal(err)
			}
			// Versión propia del rol del portal (otro nombre y una acción
			// menos): retirarla no afecta a otros perfiles.
			anterior.VersionRol.Concesiones = anterior.VersionRol.Concesiones[:len(anterior.VersionRol.Concesiones)-1]
			anterior.VersionRol.Nombre += " " + caso
			anterior.AsignacionPerfil.AsignacionID = primera.AsignacionPerfil.AsignacionID
			anterior.AsignacionPerfil.VersionRolRef = anterior.VersionRol.Referencia()
			anterior.ControlVigenciaVersionRol.VersionRolRef = anterior.VersionRol.Referencia()
			if caso == "otro_acto" {
				otra := e
				otra.autoridad.actoAsignacion = "acto:seguridad:prueba:otro"
				otra.publicarAjena(t, primera, anterior)
			} else {
				e.publicarAjena(t, primera, anterior)
				e.retirarRolVigente(t)
			}
			ref, filas := e.historia(t)
			aprobacion := aprobacionProvisionMiBolsaDesarrollo{referencia: "aprobacion:bolsa:prueba", preimagen: e.huellaVigente(t)}
			if _, estado, err := e.arrancar(true, aprobacion); err != nil || estado != perfilMiBolsaPendienteProvision {
				t.Fatalf("%s con aprobación: %v %s", caso, err, estado)
			}
			if r, f := e.historia(t); r != ref || f != filas {
				t.Fatalf("%s restaurado con aprobación", caso)
			}
		})
	}

	t.Run("otra_forma_exige_aprobacion_exacta", func(t *testing.T) {
		e := nuevoEscenarioArranqueMiBolsaPrueba(t, ctx, pool, fuente)
		primera, _, err := e.arrancar(false, sin)
		if err != nil {
			t.Fatal(err)
		}
		// Un binario anterior publicó el portal con otras concesiones.
		anterior, err := nuevaInstantaneaMiBolsaDesarrollo(e.identidad, e.ahora, true)
		if err != nil {
			t.Fatal(err)
		}
		anterior.VersionRol.Concesiones = anterior.VersionRol.Concesiones[:len(anterior.VersionRol.Concesiones)-1]
		anterior.AsignacionPerfil.AsignacionID = primera.AsignacionPerfil.AsignacionID
		e.publicarAjena(t, primera, anterior)
		ref, filas := e.historia(t)
		huella := e.huellaVigente(t)
		for _, aprobacion := range []aprobacionProvisionMiBolsaDesarrollo{
			sin, {referencia: "aprobacion:bolsa:prueba", preimagen: hex.EncodeToString(make([]byte, 32))},
			{preimagen: huella},
		} {
			if _, estado, err := e.arrancar(true, aprobacion); err != nil || estado != perfilMiBolsaPendienteProvision {
				t.Fatalf("sin aprobación exacta: %v %s", err, estado)
			}
		}
		if r, f := e.historia(t); r != ref || f != filas {
			t.Fatal("sin aprobación exacta se escribió")
		}
		aprobacion := aprobacionProvisionMiBolsaDesarrollo{referencia: "aprobacion:bolsa:prueba", preimagen: huella}
		provisionada, estado, err := e.arrancar(true, aprobacion)
		if err != nil || estado != perfilMiBolsaProvisionado || provisionada.AsignacionPerfil.Version != 3 {
			t.Fatalf("provisión aprobada: %v %s", err, estado)
		}
		ref, filas = e.historia(t)
		// Las variables puestas no aprueban ningún estado posterior.
		if _, estado, err := e.arrancar(true, aprobacion); err != nil || estado != perfilMiBolsaVigente {
			t.Fatalf("reinicio tras provisión: %v %s", err, estado)
		}
		revocada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(provisionada)
		revocada.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
		revocada.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
		revocada.AsignacionPerfil.RevocadaEn = e.ahora.Add(time.Second)
		revocada.AsignacionPerfil.RevocacionRef = "revocacion:mi-bolsa-tras-provision"
		e.publicarAjena(t, provisionada, revocada)
		refRevocada, filasRevocadas := e.historia(t)
		if refRevocada == ref || filasRevocadas != filas+1 {
			t.Fatal("revocación no publicada")
		}
		if _, estado, err := e.arrancar(true, aprobacion); err != nil || estado != perfilMiBolsaPendienteProvision {
			t.Fatalf("aprobación antigua sobre revocada: %v %s", err, estado)
		}
		if r, f := e.historia(t); r != refRevocada || f != filasRevocadas {
			t.Fatal("la aprobación antigua reactivó la revocación")
		}
		if e.fuente != nil {
			// La fuente oficial sigue viendo la revocación: el PDP la deniega.
			oficial, err := e.fuente.ObtenerInstantaneaAutorizacion(ctx, e.identidad.personaRef, e.identidad.perfilRef)
			if err == nil && oficial.AsignacionPerfil.Estado != dominiovec.EstadoAsignacionPerfilRevocada {
				t.Fatal("la fuente V3 no ve la revocación")
			}
		}
	})
}
