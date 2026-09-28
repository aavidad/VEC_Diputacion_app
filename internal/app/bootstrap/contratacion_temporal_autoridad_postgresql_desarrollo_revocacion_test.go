package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func (a *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) PrepararInstantaneaReincorporacionTitular(
	ctx context.Context, i vecdomain.InstantaneaAutorizacion,
) (vecdomain.InstantaneaAutorizacion, error) {
	return a.PrepararInstantanea(ctx, i)
}

func (a *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) PublicarInstantaneaReincorporacionTitular(
	ctx context.Context, i vecdomain.InstantaneaAutorizacion,
) error {
	return a.PublicarInstantanea(ctx, i)
}

type autoridadSinContratoCT130Prueba struct{}

func (autoridadSinContratoCT130Prueba) PrepararInstantanea(_ context.Context, i vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error) {
	return i, nil
}

func (autoridadSinContratoCT130Prueba) PublicarInstantanea(context.Context, vecdomain.InstantaneaAutorizacion) error {
	return nil
}

func TestCT130NoUsaPublicadorGenericoSinContratoEspecifico(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	for _, ruta := range []string{httpinterno.RutaReincorporacionesTitular, httpinterno.RutaCapacidadReincorporacionTitular} {
		if err := publicarInstantaneaAsignacionCTSegunRuta(context.Background(), ruta, autoridadSinContratoCT130Prueba{}, soporte.instantanea); err == nil {
			t.Fatalf("%s aceptó autoridad sin contrato CT130", ruta)
		}
	}
}

// Requiere exclusivamente una base PostgreSQL 18 desechable con las migraciones
// de autorización y contexto instaladas. Nunca apunta a la base conservada.
func TestCT130PreimagenCentralPostgreSQL(t *testing.T) {
	if os.Getenv("VEC_CT130_REVOCACION_PG_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, os.Getenv("VEC_CT130_REVOCACION_PG_DSN_GOBIERNO"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	contar := func(t *testing.T, perfil, sesion string) (int, bool) {
		t.Helper()
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
			t.Fatal(err)
		}
		var n int
		var sesionViva bool
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE perfil_activo_ref=$1`, perfil).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if err := tx.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM vec_autorizacion.control_sesion_actual_v1 AS actual
			JOIN vec_autorizacion.control_sesion_v1 AS control
			  ON control.control_sesion_ref=actual.control_sesion_ref AND control.revision=actual.revision
			WHERE actual.sesion_ref=$1 AND control.estado='activa')`, sesion).Scan(&sesionViva); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		return n, sesionViva
	}
	for _, caso := range []string{"permitida", "restringida", "revocada", "carrera"} {
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
			acciones := []vecdomain.ConcesionRol{
				{Accion: accionConsultarSeguimientoCeseDesarrollo, ModuloID: ctports.ModuloContratacion,
					TipoRecurso: "seguimiento_contratacion_temporal", Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
				{Accion: string(ctdomain.AccionRegistrarReincorporacionTitular), ModuloID: ctports.ModuloContratacion,
					TipoRecurso: ctports.TipoRecursoReincorporacionTitular, Finalidades: []string{ctports.FinalidadRegistrarReincorporacionTitular}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
				{Accion: string(ctports.AccionConsultarAntecedenteReincorporacionTitular), ModuloID: ctports.ModuloContratacion,
					TipoRecurso: ctports.TipoRecursoLecturaReincorporacionTitular, Finalidades: []string{ctports.FinalidadLecturaReincorporacionTitular},
					CamposPermitidos: []string{"cese_evento_ref", "cese_recibo_ref", "documento_ref", "documento_sha256", "existe_cese", "fecha_efectiva", "relacion_ref"},
					GarantiaMinima:   vecdomain.AuthAssuranceHigh},
			}
			semilla, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(vinculo.PrincipalID, vinculo.PerfilActivoRef, ahora,
				"reincorporacion_titular_ct_desarrollo", "Reincorporación titular de desarrollo", "reincorporacion-titular-ct-desarrollo", acciones,
				[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
			if err != nil {
				t.Fatal(err)
			}
			soporte.reincorporacionTitular = &soporteSeguimientoCeseDesarrollo{instantanea: semilla}
			if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
				t.Fatal(err)
			}
			if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
				t.Fatal(err)
			}
			a := &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
			preparada, err := a.PrepararInstantaneaReincorporacionTitular(ctx, semilla)
			if err != nil {
				t.Fatalf("CT130 con preimagen base exacta: %v", err)
			}
			base := soporte.instantanea
			if caso != "permitida" {
				otra := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(base)
				if caso == "revocada" {
					otra.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
					otra.AsignacionPerfil.RevocadaEn = base.AsignacionPerfil.EmitidaEn.Add(time.Second)
					otra.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
					otra.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
				} else {
					otra.AsignacionPerfil.Ambitos[0].Valores[0] = "organizacion:restringida"
				}
				otra, err = a.autoridadComun().prepararInstantanea(ctx, otra, false)
				if err != nil {
					t.Fatal(err)
				}
				if err := a.autoridadComun().publicarInstantaneaDesdePreimagen(ctx, otra, base); err != nil {
					t.Fatal(err)
				}
				antes, viva := contar(t, vinculo.PerfilActivoRef, vinculo.SesionRef)
				if !viva {
					t.Fatal("la prueba no conserva sesión viva tras cambiar asignación")
				}
				if err := a.PublicarInstantaneaReincorporacionTitular(ctx, preparada); err == nil {
					t.Fatal("CT130 restauró permiso tras cambio central concurrente")
				}
				despues, viva := contar(t, vinculo.PerfilActivoRef, vinculo.SesionRef)
				if despues != antes || !viva {
					t.Fatal("el rechazo alteró historia o sesión")
				}
				a = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
				if _, err := a.PrepararInstantaneaReincorporacionTitular(ctx, semilla); err == nil {
					t.Fatal("CT130 aceptó preimagen central restringida o revocada tras reinicio")
				}
				return
			}
			if err := a.PublicarInstantaneaReincorporacionTitular(ctx, preparada); err != nil {
				t.Fatal(err)
			}
			antes, viva := contar(t, vinculo.PerfilActivoRef, vinculo.SesionRef)
			if !viva {
				t.Fatal("la sesión inicial no sigue activa")
			}
			a = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
			replay, err := a.PrepararInstantaneaReincorporacionTitular(ctx, semilla)
			if err != nil || replay.AsignacionPerfil.Referencia() != preparada.AsignacionPerfil.Referencia() ||
				a.PublicarInstantaneaReincorporacionTitular(ctx, replay) != nil {
				t.Fatalf("CT130 no recuperó publicación exacta: %v", err)
			}
			despues, viva := contar(t, vinculo.PerfilActivoRef, vinculo.SesionRef)
			if despues != antes || !viva {
				t.Fatal("el replay duplicó asignación o alteró sesión")
			}
		})
	}
}

func TestCT130PublicacionSinPreimagenPreparadaFallaCerrada(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	autoridad := &autoridadPostgreSQLContratacionTemporalDesarrollo{soporte: soporte}
	if err := autoridad.PublicarInstantaneaReincorporacionTitular(context.Background(), soporte.instantanea); err == nil {
		t.Fatal("CT130 publicó sin marca de preparación central")
	}
}
