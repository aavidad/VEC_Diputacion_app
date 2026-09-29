package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Pruebas del perfil dinámico de RRHH (y de los lectores RRHH) contra un
// PostgreSQL 18 desechable con la estructura de la principal (un clon
// restaurado en /dev/shm). Nunca apuntan a una base conservada.
func poolesPerfilDinamicoRRHHPostgreSQLPrueba(t *testing.T) (context.Context, *pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("VEC_PERFIL_DINAMICO_RRHH_PG_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable con la estructura de la principal")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancelar)
	abrir := func(variable string) *pgxpool.Pool {
		pool, err := pgxpool.New(ctx, os.Getenv(variable))
		if err != nil {
			t.Fatalf("pool %s no disponible", variable)
		}
		t.Cleanup(pool.Close)
		return pool
	}
	return ctx, abrir("VEC_PERFIL_DINAMICO_RRHH_PG_DSN_GOBIERNO"), abrir("VEC_PERFIL_DINAMICO_RRHH_PG_DSN_ADMIN")
}

// soporteRRHHPostgreSQLPrueba compone el soporte del técnico RRHH con una
// persona sintética nueva (sin colisiones entre ejecuciones), su contexto ya
// registrado y la autoridad PostgreSQL real.
func soporteRRHHPostgreSQLPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *soporteAltaContratacionTemporalDesarrollo {
	t.Helper()
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	var aleatorio [8]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		t.Fatal(err)
	}
	principal.ID += "_" + hex.EncodeToString(aleatorio[:])
	huella := sha256.Sum256(aleatorio[:])
	principal.Attributes["certificate_sha256"] = hex.EncodeToString(huella[:])
	soporte.principalID, soporte.certificadoSHA256 = principal.ID, principal.Attributes["certificate_sha256"]
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	var err error
	if soporte.contexto, err = nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora); err != nil {
		t.Fatal(err)
	}
	v, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if soporte.instantanea, err = nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora); err != nil {
		t.Fatal(err)
	}
	if soporte.instantaneaAnalisis, err = nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora); err != nil {
		t.Fatal(err)
	}
	if soporte.instantaneaCobertura, err = nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora); err != nil {
		t.Fatal(err)
	}
	soporte.autoridadAsignaciones = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, pool, soporte); err != nil {
		t.Fatalf("contexto: %v", err)
	}
	return soporte
}

// analisisDeExpedientePrueba es el rol de análisis con los ámbitos que la
// composición le da por petición (organización, expediente, fase y estado).
func analisisDeExpedientePrueba(s *soporteAltaContratacionTemporalDesarrollo, expediente string) vecdomain.InstantaneaAutorizacion {
	i := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaAnalisis)
	i.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
		{Clave: "expediente_ref", Valores: []string{expediente}},
		{Clave: "fase_previa", Valores: []string{"solicitud"}},
		{Clave: "estado_previo", Valores: []string{"en_curso"}},
	}
	return i
}

// peticionRRHHPrueba hace lo que hace una ruta dinámica de RRHH: prepara el
// rol de la ruta con sus ámbitos y lo publica.
func peticionRRHHPrueba(ctx context.Context, s *soporteAltaContratacionTemporalDesarrollo, i vecdomain.InstantaneaAutorizacion) error {
	preparada, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, i)
	if err != nil {
		return err
	}
	return s.autoridadAsignaciones.PublicarInstantanea(ctx, preparada)
}

// retirarRolPorOtroActoPrueba retira, como haría un acto gobernado, la
// versión de rol de la asignación vigente (nueva revisión de su control).
func retirarRolPorOtroActoPrueba(t *testing.T, ctx context.Context, admin *pgxpool.Pool, s *soporteAltaContratacionTemporalDesarrollo) {
	t.Helper()
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, admin, s.instantanea.AsignacionPerfil.PerfilActivoRef)
	if err != nil || !encontrada {
		t.Fatalf("sin asignación cuyo rol retirar: %v", err)
	}
	control := publicada.instantanea.ControlVigenciaVersionRol
	control.Revision++
	control.Estado = vecdomain.EstadoControlVigenciaVersionRolRetirada
	control.ActualizadoEn = time.Now().UTC().Truncate(time.Microsecond)
	control.ActualizadoPor = "seguridad:prueba"
	control.ActoRef, control.MotivoCodigo = "acto:seguridad:prueba:retirada-rol", "retirada_prueba"
	huella, err := control.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	documento, err := json.Marshal(control)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `INSERT INTO vec_autorizacion.control_vigencia_version_rol
		(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento) VALUES ($1,$2,$3,$4,$5,$6::jsonb)`,
		control.VersionRolRef, control.Revision, string(control.Estado), huella, control.ActualizadoEn, documento); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE vec_autorizacion.control_vigencia_version_rol_actual
		SET revision=$2, actualizada_en=$3, actualizada_por=$4, acto_ref=$5 WHERE version_rol_ref=$1`,
		control.VersionRolRef, control.Revision, control.ActualizadoEn, control.ActualizadoPor, "acto:seguridad:prueba:retirada-rol"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

// rolPropioPrueba es el rol de alta con un identificador exclusivo de la
// ejecución, para poder retirar su versión sin tocar a otros perfiles.
func rolPropioPrueba(t *testing.T, s *soporteAltaContratacionTemporalDesarrollo) vecdomain.InstantaneaAutorizacion {
	t.Helper()
	var aleatorio [6]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		t.Fatal(err)
	}
	i := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantanea)
	i.VersionRol.RolID = "tecnico_rrhh_prueba_" + hex.EncodeToString(aleatorio[:])
	i.VersionRol.Version = 1
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	return i
}

func TestPerfilDinamicoRRHHArranqueSoloLeePostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef

	if estado, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, s); err != nil || estado != perfilDinamicoPublicadoInicial {
		t.Fatalf("primer arranque: %s %v", estado, err)
	}
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h.versiones != 1 || h.acto != actoAsignacionCTDesarrollo {
		t.Fatalf("la publicación inicial no quedó única y con el acto del circuito: %+v", h)
	}
	// Una ruta de análisis ajusta el perfil a su expediente (sigue siendo dinámica).
	if err := peticionRRHHPrueba(ctx, s, analisisDeExpedientePrueba(s, "expediente:rrhh:uno")); err != nil {
		t.Fatalf("la ruta de análisis no se sirve: %v", err)
	}
	tras := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if tras.versiones != 2 {
		t.Fatalf("versiones tras la ruta: %d", tras.versiones)
	}
	// Reiniciar no devuelve el perfil al rol de alta ni escribe nada.
	for i := 0; i < 2; i++ {
		if estado, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, s); err != nil || estado != perfilDinamicoOperativo {
			t.Fatalf("rearranque %d: %s %v", i, estado, err)
		}
		if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, s); err != nil {
			t.Fatalf("rearranque %d: %v", i, err)
		}
		if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != tras {
			t.Fatalf("el rearranque %d escribió: %+v frente a %+v", i, h, tras)
		}
	}
	// Las demás rutas siguen alternando encima de lo operativo.
	for _, i := range []vecdomain.InstantaneaAutorizacion{s.instantanea, s.instantaneaCobertura, analisisDeExpedientePrueba(s, "expediente:rrhh:dos")} {
		if err := peticionRRHHPrueba(ctx, s, i); err != nil {
			t.Fatalf("ruta %s: %v", i.VersionRol.RolID, err)
		}
	}
}

func TestPerfilDinamicoRRHHNoReactivaRevocadoRestringidoNiRetiradoPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	for _, caso := range []string{"revocada", "restringida", "rol_retirado"} {
		t.Run(caso, func(t *testing.T) {
			s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
			perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
			if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, s); err != nil {
				t.Fatal(err)
			}
			if err := peticionRRHHPrueba(ctx, s, analisisDeExpedientePrueba(s, "expediente:rrhh:previo")); err != nil {
				t.Fatal(err)
			}
			// Un candidato preparado antes del cierre no puede usarse después.
			antiguo, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, s.instantaneaCobertura)
			if err != nil {
				t.Fatal(err)
			}
			switch caso {
			case "revocada":
				revocarPorOtroActoPrueba(t, ctx, s, gobierno, false)
			case "restringida":
				revocarPorOtroActoPrueba(t, ctx, s, gobierno, true)
			default:
				// El control de una versión de rol es común a todos sus perfiles:
				// se retira un rol propio de la prueba, nunca uno compartido.
				propio := rolPropioPrueba(t, s)
				if err := peticionRRHHPrueba(ctx, s, propio); err != nil {
					t.Fatal(err)
				}
				retirarRolPorOtroActoPrueba(t, ctx, admin, s)
			}
			cerrada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
			comprobar := func(momento string) {
				t.Helper()
				for _, i := range []vecdomain.InstantaneaAutorizacion{s.instantanea, s.instantaneaCobertura,
					analisisDeExpedientePrueba(s, "expediente:rrhh:previo"), analisisDeExpedientePrueba(s, "expediente:rrhh:otro")} {
					if err := peticionRRHHPrueba(ctx, s, i); err == nil {
						t.Fatalf("%s: la ruta %s reactivó el perfil %s", momento, i.VersionRol.RolID, caso)
					}
				}
				if err := s.autoridadAsignaciones.PublicarInstantanea(ctx, antiguo); err == nil {
					t.Fatalf("%s: un candidato preparado antes reactivó el perfil %s", momento, caso)
				}
				if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != cerrada {
					t.Fatalf("%s: se escribió sobre el perfil %s: %+v frente a %+v", momento, caso, h, cerrada)
				}
			}
			comprobar("en caliente")
			for i := 0; i < 2; i++ {
				if estado, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, s); err != nil || estado != perfilDinamicoPendienteProvision {
					t.Fatalf("rearranque %d: %s %v", i, estado, err)
				}
				if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, s); err != nil {
					t.Fatalf("el rearranque %d se detuvo: %v", i, err)
				}
				comprobar("tras reiniciar")
			}
		})
	}
}

func TestPerfilDinamicoRRHHConcurrenciaSinFallosNiReactivacionPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	lanzar := func(n int, i vecdomain.InstantaneaAutorizacion) (fallos int) {
		var espera sync.WaitGroup
		var mu sync.Mutex
		inicio := make(chan struct{})
		for k := 0; k < n; k++ {
			espera.Add(1)
			go func() {
				defer espera.Done()
				<-inicio
				if err := peticionRRHHPrueba(ctx, s, i); err != nil {
					mu.Lock()
					fallos++
					mu.Unlock()
				}
			}()
		}
		close(inicio)
		espera.Wait()
		return fallos
	}
	// 30 operaciones simultáneas de la misma ruta y expediente: ninguna falla
	// y, como mucho, una versión nueva.
	antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if fallos := lanzar(30, analisisDeExpedientePrueba(s, "expediente:rrhh:concurrente")); fallos != 0 {
		t.Fatalf("%d de 30 operaciones simultáneas fallaron", fallos)
	}
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h.versiones != antes.versiones+1 {
		t.Fatalf("30 operaciones iguales crearon %d versiones", h.versiones-antes.versiones)
	}
	// Tras revocar, 30 simultáneas se deniegan todas y no escriben.
	revocarPorOtroActoPrueba(t, ctx, s, gobierno, false)
	cerrada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if fallos := lanzar(30, analisisDeExpedientePrueba(s, "expediente:rrhh:tras-revocar")); fallos != 30 {
		t.Fatalf("%d de 30 operaciones pasaron sobre el perfil revocado", 30-fallos)
	}
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != cerrada {
		t.Fatalf("operaciones concurrentes escribieron sobre el perfil revocado: %+v", h)
	}
}

// soporteLectorRRHHPostgreSQLPrueba compone un lector RRHH que no es el
// técnico (su propio perfil), con las plantillas de cuadro y detalle.
func soporteLectorRRHHPostgreSQLPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *soporteAltaContratacionTemporalDesarrollo {
	t.Helper()
	s := soporteRRHHPostgreSQLPrueba(t, ctx, pool)
	v, err := s.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	nueva := func(rol, nombre, accion, finalidad, tipo string) vecdomain.InstantaneaAutorizacion {
		i, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora, rol, nombre, rol,
			[]vecdomain.ConcesionRol{{Accion: accion, ModuloID: ports.ModuloContratacion, TipoRecurso: tipo,
				Finalidades: []string{finalidad}, GarantiaMinima: vecdomain.AuthAssuranceHigh}},
			[]vecdomain.AmbitoPerfil{
				{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
				{Clave: "clase_ambito", Valores: []string{"servicio"}},
				{Clave: "ambito_ref", Valores: []string{"servicio:prueba"}},
			})
		if err != nil {
			t.Fatal(err)
		}
		return i
	}
	s.lectorConsultasRRHH, s.tecnicoConsultaRRHH = true, false
	s.instantaneaCuadroRRHH = nueva("consulta_cuadro_rrhh_desarrollo", "Consulta de bandeja de desarrollo", ports.AccionConsultarCuadroRRHH, ports.FinalidadConsultarCuadroRRHH, ports.TipoRecursoCuadroRRHH)
	s.instantaneaDetalleRRHH = nueva("consulta_detalle_rrhh_desarrollo", "Consulta de expediente de desarrollo", ports.AccionConsultarDetalleRRHH, ports.FinalidadConsultarDetalleRRHH, ports.TipoRecursoExpediente)
	return s
}

func TestLectorRRHHArranqueNoReactivaNiSeDetienePostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteLectorRRHHPostgreSQLPrueba(t, ctx, gobierno)
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	if err := prepararInstantaneasInicialesLectorRRHHDesarrollo(ctx, s); err != nil {
		t.Fatalf("primer arranque del lector: %v", err)
	}
	inicial := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if inicial.versiones != 1 || inicial.acto != actoAsignacionCTDesarrollo {
		t.Fatalf("la inicial del lector no es única: %+v", inicial)
	}
	// El lector alterna cuadro y detalle por petición, encima de lo operativo.
	if err := peticionRRHHPrueba(ctx, s, s.instantaneaDetalleRRHH); err != nil {
		t.Fatalf("el detalle no se sirve: %v", err)
	}
	operativa := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if err := prepararInstantaneasInicialesLectorRRHHDesarrollo(ctx, s); err != nil ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil) != operativa {
		t.Fatalf("el rearranque del lector escribió o falló: %v", err)
	}
	revocarPorOtroActoPrueba(t, ctx, s, gobierno, false)
	cerrada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	for i := 0; i < 2; i++ {
		if err := prepararInstantaneasInicialesLectorRRHHDesarrollo(ctx, s); err != nil {
			t.Fatalf("un lector revocado detuvo el arranque: %v", err)
		}
		for _, plantilla := range []vecdomain.InstantaneaAutorizacion{s.instantaneaCuadroRRHH, s.instantaneaDetalleRRHH} {
			if err := peticionRRHHPrueba(ctx, s, plantilla); err == nil {
				t.Fatal("una consulta reactivó el lector revocado")
			}
		}
		if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != cerrada {
			t.Fatalf("se escribió sobre el lector revocado: %+v", h)
		}
	}
}
