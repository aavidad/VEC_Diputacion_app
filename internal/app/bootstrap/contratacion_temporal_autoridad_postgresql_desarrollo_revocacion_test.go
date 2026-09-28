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
			contextoCT130, err := nuevoContextoReincorporacionTitularDesarrollo(soporte, ahora)
			if err != nil {
				t.Fatal(err)
			}
			vinculoCT130, err := contextoCT130.Vinculo.Datos()
			if err != nil || vinculoCT130.PerfilActivoRef == vinculo.PerfilActivoRef {
				t.Fatal("CT130 no tiene perfil dedicado")
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
			semilla, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(vinculoCT130.PrincipalID, vinculoCT130.PerfilActivoRef, ahora,
				"reincorporacion_titular_ct_desarrollo", "Reincorporación titular de desarrollo", "reincorporacion-titular-ct-desarrollo", acciones,
				[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
			if err != nil {
				t.Fatal(err)
			}
			soporte.reincorporacionTitular = &soporteSeguimientoCeseDesarrollo{
				instantanea: semilla, contexto: contextoCT130,
				contextoEsperadoRegistrado: contextoCT130.Resultado,
				sesionOperativa:            proveedorSesionOperativaCTPrueba{contexto: contextoCT130},
			}
			if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
				t.Fatal(err)
			}
			if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
				t.Fatal(err)
			}
			if err := publicarContextoPostgreSQLReincorporacionTitularDesarrollo(ctx, pool, soporte); err != nil {
				t.Fatal(err)
			}
			ctxRuta := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaReincorporacionesTitular)
			objetivoUno := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
			objetivoUno.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
				{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
				{Clave: "expediente_ref", Valores: []string{"expediente:ct130:uno"}},
			}
			objetivoDos := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
			objetivoDos.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
				{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
				{Clave: "expediente_ref", Valores: []string{"expediente:ct130:dos"}},
			}
			a := &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
			preparada, err := a.PrepararInstantaneaReincorporacionTitular(ctxRuta, objetivoUno)
			if err != nil {
				t.Fatalf("CT130 ausente con perfil dedicado: %v", err)
			}
			if err := a.PublicarInstantaneaReincorporacionTitular(ctxRuta, preparada); err != nil {
				t.Fatalf("CT130 inicial: %v", err)
			}
			base := soporte.instantanea
			if exacta, encontrada, err := instantaneaCentralCTExacta(ctx, pool, base); err != nil || !encontrada || !exacta {
				t.Fatal("la publicación CT130 alteró el perfil base")
			}
			if caso == "carrera" {
				objetivoTres := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
				objetivoTres.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
					{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
					{Clave: "expediente_ref", Valores: []string{"expediente:ct130:tres"}},
				}
				ctxDos := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaReincorporacionesTitular)
				competidor := &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
				segunda, err := a.PrepararInstantaneaReincorporacionTitular(ctxRuta, objetivoDos)
				if err != nil {
					t.Fatal(err)
				}
				tercera, err := competidor.PrepararInstantaneaReincorporacionTitular(ctxDos, objetivoTres)
				if err != nil || segunda.AsignacionPerfil.Version != tercera.AsignacionPerfil.Version {
					t.Fatal("la prueba no preparó dos candidatos desde la misma preimagen")
				}
				type resultadoPublicacion struct {
					indice int
					err    error
				}
				inicio := make(chan struct{})
				resultados := make(chan resultadoPublicacion, 2)
				go func() {
					<-inicio
					resultados <- resultadoPublicacion{0, a.PublicarInstantaneaReincorporacionTitular(ctxRuta, segunda)}
				}()
				go func() {
					<-inicio
					resultados <- resultadoPublicacion{1, competidor.PublicarInstantaneaReincorporacionTitular(ctxDos, tercera)}
				}()
				close(inicio)
				primera, segundaRespuesta := <-resultados, <-resultados
				exitos := 0
				ganadora := segunda
				for _, respuesta := range []resultadoPublicacion{primera, segundaRespuesta} {
					if respuesta.err == nil {
						exitos++
						if respuesta.indice == 1 {
							ganadora = tercera
						}
					}
				}
				if exitos != 1 {
					t.Fatalf("CAS concurrente confirmó %d candidatos; se esperaba uno", exitos)
				}
				if exacta, encontrada, err := instantaneaCentralCTExacta(ctx, pool, ganadora); err != nil || !encontrada || !exacta {
					t.Fatal("el ganador del CAS no quedó como asignación central")
				}
				administrativa, err := a.autoridadReincorporacionTitular(ctxRuta)
				if err != nil {
					t.Fatal(err)
				}
				administrativa.actoAsignacion = "acto:ct130:prueba:revocacion-concurrente"
				revocada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(ganadora)
				revocada.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
				revocada.AsignacionPerfil.RevocadaEn = ganadora.AsignacionPerfil.EmitidaEn.Add(time.Second)
				revocada.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
				revocada.AsignacionPerfil.RevocacionRef = "revocacion:prueba:carrera"
				revocada, err = administrativa.prepararInstantanea(ctxRuta, revocada, false)
				if err != nil || administrativa.publicarInstantaneaDesdePreimagen(ctxRuta, revocada, ganadora) != nil {
					t.Fatalf("no se pudo revocar tras CAS: %v", err)
				}
				antes, viva := contar(t, vinculoCT130.PerfilActivoRef, vinculoCT130.SesionRef)
				if !viva || a.PublicarInstantaneaReincorporacionTitular(ctxRuta, segunda) == nil ||
					competidor.PublicarInstantaneaReincorporacionTitular(ctxDos, tercera) == nil {
					t.Fatal("un candidato antiguo reactivó CT130 revocada o cayó la sesión")
				}
				despues, viva := contar(t, vinculoCT130.PerfilActivoRef, vinculoCT130.SesionRef)
				if despues != antes || !viva {
					t.Fatal("la carrera tras revocación alteró historia o sesión")
				}
				return
			}
			if caso != "permitida" {
				segunda, err := a.PrepararInstantaneaReincorporacionTitular(ctxRuta, objetivoDos)
				if err != nil {
					t.Fatal(err)
				}
				otra := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(preparada)
				if caso == "revocada" {
					otra.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
					otra.AsignacionPerfil.RevocadaEn = preparada.AsignacionPerfil.EmitidaEn.Add(time.Second)
					otra.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
					otra.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
				} else {
					otra.AsignacionPerfil.Ambitos[1].Valores[0] = "expediente:ct130:restringido"
				}
				administrativa, err := a.autoridadReincorporacionTitular(ctxRuta)
				if err != nil {
					t.Fatal(err)
				}
				administrativa.actoAsignacion = "acto:ct130:prueba:administrativa"
				otra, err = administrativa.prepararInstantanea(ctxRuta, otra, false)
				if err != nil {
					t.Fatal(err)
				}
				if err := administrativa.publicarInstantaneaDesdePreimagen(ctxRuta, otra, preparada); err != nil {
					t.Fatal(err)
				}
				antes, viva := contar(t, vinculoCT130.PerfilActivoRef, vinculoCT130.SesionRef)
				if !viva {
					t.Fatal("la prueba no conserva sesión viva tras cambiar asignación")
				}
				if err := a.PublicarInstantaneaReincorporacionTitular(ctxRuta, segunda); err == nil {
					t.Fatal("CT130 restauró permiso tras cambio central concurrente")
				}
				if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
					t.Fatalf("el reinicio base falló: %v", err)
				}
				if exacta, encontrada, err := instantaneaCentralCTExacta(ctx, pool, base); err != nil || !encontrada || !exacta {
					t.Fatal("el reinicio alteró el perfil base")
				}
				despues, viva := contar(t, vinculoCT130.PerfilActivoRef, vinculoCT130.SesionRef)
				if despues != antes || !viva {
					t.Fatal("el rechazo alteró historia o sesión")
				}
				a = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
				ctxRuta = contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaReincorporacionesTitular)
				if _, err := a.PrepararInstantaneaReincorporacionTitular(ctxRuta, objetivoDos); err == nil {
					t.Fatal("CT130 aceptó preimagen central restringida o revocada tras reinicio")
				}
				return
			}
			segunda, err := a.PrepararInstantaneaReincorporacionTitular(ctxRuta, objetivoDos)
			if err != nil {
				t.Fatalf("segundo expediente CT130: %v", err)
			}
			if err := a.PublicarInstantaneaReincorporacionTitular(ctxRuta, segunda); err != nil {
				t.Fatal(err)
			}
			antes, viva := contar(t, vinculoCT130.PerfilActivoRef, vinculoCT130.SesionRef)
			if !viva {
				t.Fatal("la sesión inicial no sigue activa")
			}
			if exacta, encontrada, err := instantaneaCentralCTExacta(ctx, pool, base); err != nil || !encontrada || !exacta {
				t.Fatal("dos expedientes CT130 alteraron el perfil base")
			}
			if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
				t.Fatal(err)
			}
			a = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
			ctxRuta = contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaReincorporacionesTitular)
			replay, err := a.PrepararInstantaneaReincorporacionTitular(ctxRuta, objetivoDos)
			if err != nil || replay.AsignacionPerfil.Referencia() != segunda.AsignacionPerfil.Referencia() ||
				a.PublicarInstantaneaReincorporacionTitular(ctxRuta, replay) != nil {
				t.Fatalf("CT130 no recuperó publicación exacta: %v", err)
			}
			despues, viva := contar(t, vinculoCT130.PerfilActivoRef, vinculoCT130.SesionRef)
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
