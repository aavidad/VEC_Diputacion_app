package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Pruebas de los perfiles fijos de RRHH (corte 2) contra el mismo PostgreSQL
// 18 desechable que las del perfil dinámico (VEC_PERFIL_DINAMICO_RRHH_PG_*).

func principalDeSoportePrueba(s *soporteAltaContratacionTemporalDesarrollo) vecdomain.Principal {
	return vecdomain.Principal{ID: s.principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": s.certificadoSHA256}}
}

// soportePerfilesFijosPostgreSQLPrueba compone el soporte de RRHH con sus
// perfiles fijos de alta y cobertura y asegura ambos como al arrancar.
func soportePerfilesFijosPostgreSQLPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (*soporteAltaContratacionTemporalDesarrollo, *perfilFijoCTDesarrollo, *perfilFijoCTDesarrollo) {
	t.Helper()
	s := soporteRRHHPostgreSQLPrueba(t, ctx, pool)
	if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, pool, s); err != nil {
		t.Fatal(err)
	}
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(s, principalDeSoportePrueba(s), time.Now().UTC().Truncate(time.Microsecond), nil); err != nil {
		t.Fatal(err)
	}
	alta, cobertura := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes), s.perfilFijoParaRuta(httpinterno.RutaDecisionCobertura)
	if alta == nil || cobertura == nil || alta.perfilRef() == cobertura.perfilRef() ||
		alta.perfilRef() == s.instantanea.AsignacionPerfil.PerfilActivoRef {
		t.Fatal("perfiles fijos mal compuestos")
	}
	for _, ruta := range []string{httpinterno.RutaPropuestaCobertura, httpinterno.RutaRectificacionCobertura, httpinterno.RutaResultadoCobertura} {
		if s.perfilFijoParaRuta(ruta) != cobertura {
			t.Fatalf("la ruta %s no usa el perfil de cobertura", ruta)
		}
	}
	if s.perfilFijoParaRuta(rutaEntregaPeticionCentro) != nil || s.perfilFijoParaRuta(httpinterno.RutaRegistroAnalisisRRHH) != nil {
		t.Fatal("una ruta dinámica quedó en un perfil fijo")
	}
	if err := asegurarPerfilesFijosCTDesarrollo(ctx, pool, s, aprobacionProvisionPerfilesRRHHDesarrollo{}, alta, cobertura); err != nil {
		t.Fatalf("arranque de los perfiles fijos: %v", err)
	}
	return s, alta, cobertura
}

func TestPerfilesFijosRRHHSeConsumenSinPublicarPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s, alta, cobertura := soportePerfilesFijosPostgreSQLPrueba(t, ctx, gobierno)
	for _, p := range []*perfilFijoCTDesarrollo{alta, cobertura} {
		h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, p.perfilRef())
		if h.versiones != 1 || h.acto != actoAsignacionPerfilFijoCTDesarrollo {
			t.Fatalf("perfil %s: inicial no única o con otro acto: %+v", p.clave, h)
		}
		// Rearrancar no escribe.
		if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, p, aprobacionProvisionPerfilesRRHHDesarrollo{},
			preimagenPropiaPerfilFijoCTDesarrollo(p, actoAsignacionPerfilFijoCTDesarrollo)); err != nil || estado != perfilFijoVigente {
			t.Fatalf("perfil %s: rearranque %s %v", p.clave, estado, err)
		}
		if historiaPerfilPostgreSQLPrueba(t, ctx, admin, p.perfilRef()) != h {
			t.Fatalf("perfil %s: el rearranque escribió", p.clave)
		}
	}
	// 30 consumos simultáneos de cada perfil: todos concedidos, cero escrituras.
	antesAlta, antesCobertura := historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef()), historiaPerfilPostgreSQLPrueba(t, ctx, admin, cobertura.perfilRef())
	var espera sync.WaitGroup
	var mu sync.Mutex
	fallos := 0
	for i := 0; i < 30; i++ {
		espera.Add(1)
		go func(p *perfilFijoCTDesarrollo) {
			defer espera.Done()
			consumida, ok := s.consumirPerfilFijoCTDesarrollo(ctx, p)
			if !ok || consumida.AsignacionPerfil.PerfilActivoRef != p.perfilRef() {
				mu.Lock()
				fallos++
				mu.Unlock()
			}
		}([]*perfilFijoCTDesarrollo{alta, cobertura}[i%2])
	}
	espera.Wait()
	if fallos != 0 {
		t.Fatalf("%d de 30 consumos simultáneos fallaron", fallos)
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef()) != antesAlta ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, cobertura.perfilRef()) != antesCobertura {
		t.Fatal("los consumos escribieron en los perfiles fijos")
	}
	// Las rutas del perfil dinámico no tocan los perfiles fijos.
	if err := peticionRRHHPrueba(ctx, s, analisisDeExpedientePrueba(s, "expediente:fijos:uno")); err != nil {
		t.Fatalf("la ruta dinámica dejó de servirse: %v", err)
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef()) != antesAlta {
		t.Fatal("una ruta dinámica escribió en el perfil fijo")
	}
}

// revocarPerfilFijoPrueba revoca o restringe, como un acto gobernado, la
// asignación vigente del perfil fijo.
func revocarPerfilFijoPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool, p *perfilFijoCTDesarrollo, estrechar bool) {
	t.Helper()
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, p.perfilRef())
	if err != nil || !encontrada {
		t.Fatalf("sin asignación que revocar: %v", err)
	}
	administrativa := autoridadPerfilFijoCTDesarrollo(pool, p)
	administrativa.exigirOrigenOperativo = false
	administrativa.actoAsignacion = "acto:seguridad:prueba:revocacion-perfil-fijo"
	nueva := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(publicada.instantanea)
	if estrechar {
		nueva.AsignacionPerfil.Ambitos = append(nueva.AsignacionPerfil.Ambitos,
			vecdomain.AmbitoPerfil{Clave: "restriccion_ref", Valores: []string{"restriccion:prueba"}})
	} else {
		nueva.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
		nueva.AsignacionPerfil.RevocadaEn = nueva.AsignacionPerfil.EmitidaEn.Add(time.Second)
		nueva.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
		nueva.AsignacionPerfil.RevocacionRef = "revocacion:prueba:perfil-fijo"
	}
	nueva.AsignacionPerfil.Version = publicada.instantanea.AsignacionPerfil.Version + 1
	if err := administrativa.publicarInstantaneaDesdePreimagen(ctx, nueva, publicada.instantanea); err != nil {
		t.Fatalf("la revocación de prueba no se publicó: %v", err)
	}
}

func TestPerfilesFijosRRHHRevocadoNoRevivePostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	for _, estrechar := range []bool{false, true} {
		s, alta, cobertura := soportePerfilesFijosPostgreSQLPrueba(t, ctx, gobierno)
		revocarPerfilFijoPrueba(t, ctx, gobierno, alta, estrechar)
		cerrada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef())
		huella := huellaVigentePrueba(t, ctx, gobierno, alta)
		// Ni con la aprobación y la huella exacta del estado revocado.
		aprobacion := aprobacionProvisionPerfilesRRHHDesarrollo{referencia: "aprobacion:prueba:rrhh", preimagenes: map[string]bool{huella: true}}
		for i := 0; i < 2; i++ {
			if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, alta); ok {
				t.Fatalf("restringido=%v: el alta consumió un perfil cerrado", estrechar)
			}
			if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, alta, aprobacion,
				preimagenPropiaPerfilFijoCTDesarrollo(alta, actoAsignacionPerfilFijoCTDesarrollo)); err != nil || estado != perfilFijoPendienteProvision {
				t.Fatalf("restringido=%v: rearranque %s %v", estrechar, estado, err)
			}
			if historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef()) != cerrada {
				t.Fatalf("restringido=%v: se escribió sobre el perfil cerrado", estrechar)
			}
		}
		// Cerrar el alta no cierra la cobertura (perfiles independientes).
		if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, cobertura); !ok {
			t.Fatal("la cobertura quedó cerrada por revocar el alta")
		}
	}
}

func huellaVigentePrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool, p *perfilFijoCTDesarrollo) string {
	t.Helper()
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, p.perfilRef())
	if err != nil || !encontrada {
		t.Fatalf("sin asignación: %v", err)
	}
	h, err := publicada.instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// Cuando cambia la plantilla (por ejemplo, el catálogo añade un centro), el
// perfil queda pendiente y solo una provisión aprobada lo actualiza (CAS).
func TestPerfilFijoRRHHCambioDePlantillaExigeProvisionPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s, alta, _ := soportePerfilesFijosPostgreSQLPrueba(t, ctx, gobierno)
	s.mu.Lock()
	for i := range alta.plantilla.AsignacionPerfil.Ambitos {
		if alta.plantilla.AsignacionPerfil.Ambitos[i].Clave == "centro_ref" {
			alta.plantilla.AsignacionPerfil.Ambitos[i].Valores = append(alta.plantilla.AsignacionPerfil.Ambitos[i].Valores, "centro:prueba:nuevo")
		}
	}
	s.mu.Unlock()
	antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef())
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, alta); ok {
		t.Fatal("se consumió una asignación que no es la plantilla vigente")
	}
	admitida := preimagenPropiaPerfilFijoCTDesarrollo(alta, actoAsignacionPerfilFijoCTDesarrollo)
	if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, alta, aprobacionProvisionPerfilesRRHHDesarrollo{}, admitida); err != nil || estado != perfilFijoPendienteProvision {
		t.Fatalf("sin aprobación: %s %v", estado, err)
	}
	// Una huella distinta de la vigente no aprueba nada.
	otra := aprobacionProvisionPerfilesRRHHDesarrollo{referencia: "aprobacion:prueba:rrhh", preimagenes: map[string]bool{"0000000000000000000000000000000000000000000000000000000000000000": true}}
	if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, alta, otra, admitida); err != nil || estado != perfilFijoPendienteProvision ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef()) != antes {
		t.Fatalf("con otra huella: %s %v", estado, err)
	}
	aprobacion := aprobacionProvisionPerfilesRRHHDesarrollo{referencia: "aprobacion:prueba:rrhh",
		preimagenes: map[string]bool{huellaVigentePrueba(t, ctx, gobierno, alta): true}}
	if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, alta, aprobacion, admitida); err != nil || estado != perfilFijoProvisionado {
		t.Fatalf("con aprobación: %s %v", estado, err)
	}
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef()); h.versiones != antes.versiones+1 || h.acto != actoAsignacionPerfilFijoCTDesarrollo {
		t.Fatalf("la provisión no dejó una versión nueva propia: %+v", h)
	}
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, alta); !ok {
		t.Fatal("tras la provisión el alta no se consume")
	}
	// Dejar la aprobación puesta no vuelve a escribir.
	despues := historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef())
	if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, alta, aprobacion, admitida); err != nil || estado != perfilFijoVigente ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef()) != despues {
		t.Fatalf("la aprobación repetida escribió: %s %v", estado, err)
	}
}

// El lector de consulta (no el técnico) pasa de alternar la bandeja y el
// expediente a un solo rol consumido; solo con provisión aprobada.
func TestLectorRRHHPasaAPerfilFijoConProvisionPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteLectorRRHHPostgreSQLPrueba(t, ctx, gobierno)
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	// Estado del binario anterior: el lector con el rol del expediente.
	if err := prepararInstantaneasInicialesLectorRRHHDesarrollo(ctx, s); err != nil {
		t.Fatal(err)
	}
	if err := peticionRRHHPrueba(ctx, s, s.instantaneaDetalleRRHH); err != nil {
		t.Fatal(err)
	}
	antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if err := componerPerfilFijoLectorRRHHDesarrollo(ctx, gobierno, s, aprobacionProvisionPerfilesRRHHDesarrollo{}); err != nil {
		t.Fatalf("sin aprobación el arranque se detuvo: %v", err)
	}
	fijo := s.perfilFijoParaRuta(httpinterno.RutaConsultaCuadroRRHH)
	if fijo == nil || fijo != s.perfilFijoParaRuta(httpinterno.RutaConsultaDetalleRRHH) || !fijo.propioDelSoporte || fijo.perfilRef() != perfil {
		t.Fatal("el lector no quedó con su perfil fijo")
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil) != antes {
		t.Fatal("sin aprobación se escribió")
	}
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, fijo); ok {
		t.Fatal("se consumió el rol antiguo")
	}
	aprobacion := aprobacionProvisionPerfilesRRHHDesarrollo{referencia: "aprobacion:prueba:lector",
		preimagenes: map[string]bool{huellaVigentePrueba(t, ctx, gobierno, fijo): true}}
	s.perfilesFijos = nil
	if err := componerPerfilFijoLectorRRHHDesarrollo(ctx, gobierno, s, aprobacion); err != nil {
		t.Fatal(err)
	}
	fijo = s.perfilFijoParaRuta(httpinterno.RutaConsultaCuadroRRHH)
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h.versiones != antes.versiones+1 || h.acto != actoAsignacionPerfilFijoCTDesarrollo {
		t.Fatalf("la provisión del lector no se aplicó: %+v", h)
	}
	consumida, ok := s.consumirPerfilFijoCTDesarrollo(ctx, fijo)
	if !ok || consumida.VersionRol.RolID != rolLectorConsultaRRHHDesarrollo || len(consumida.VersionRol.Concesiones) != 2 {
		t.Fatal("el lector no consume su rol de consulta único")
	}
	// Revocado, ni con aprobación y huella exacta vuelve.
	revocarPerfilFijoPrueba(t, ctx, gobierno, fijo, false)
	cerrada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	revocada := aprobacionProvisionPerfilesRRHHDesarrollo{referencia: "aprobacion:prueba:lector",
		preimagenes: map[string]bool{huellaVigentePrueba(t, ctx, gobierno, fijo): true}}
	s.perfilesFijos = nil
	if err := componerPerfilFijoLectorRRHHDesarrollo(ctx, gobierno, s, revocada); err != nil {
		t.Fatal(err)
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil) != cerrada {
		t.Fatal("la provisión reactivó un lector revocado")
	}
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, s.perfilFijoParaRuta(httpinterno.RutaConsultaCuadroRRHH)); ok {
		t.Fatal("se consumió un lector revocado")
	}
}
