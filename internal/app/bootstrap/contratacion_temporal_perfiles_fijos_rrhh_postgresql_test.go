package bootstrap

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
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

func TestEntregaGETyPOSTConsumenAsignacionesSinPublicarPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s, _, _ := soportePerfilesFijosPostgreSQLPrueba(t, ctx, gobierno)
	principal := principalDeSoportePrueba(s)
	for _, caso := range []struct {
		metodo string
		modo   string
	}{
		{"GET", "bandeja"},
		{"POST", "preparar"},
	} {
		fijo := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, caso.metodo)
		if fijo == nil {
			t.Fatal("perfil fijo ausente", caso.metodo)
		}
		v, err := fijo.contexto.Vinculo.Datos()
		if err != nil {
			t.Fatal(err)
		}
		m := ports.MaterialEntregaPeticionCentro{Modo: caso.modo, ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef}
		if caso.modo == "preparar" {
			m.PeticionRef, m.CentroRef, m.CategoriaRef = "peticion:centro:prueba", centroAltaContratacionTemporalDesarrollo, categoriaAltaContratacionTemporalDesarrollo
			m.VersionEsperada, m.ClaveAltaCandidata = 2, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
			m.AmbitoAltaHMAC = "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" + strings.Repeat("a", 64)
		}
		recurso, err := postgresct.RecursoEntregaPeticionCentro(m)
		if err != nil {
			t.Fatal(err)
		}
		datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: fijo.contexto.Vinculo, ReferenciaMotivo: motivoEntregaPeticionDesarrollo(),
			Accion: postgresct.AccionEntregaPeticionCentro(m), Recurso: recurso, Finalidad: ports.FinalidadEntregaPeticionCentro,
		}
		peticion := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidadConsultaContratacionTemporalDesarrollo{sello: s.sello, ruta: rutaEntregaPeticionCentro,
				metodo: caso.metodo, principal: principal})
		peticion = context.WithValue(peticion, claveMaterialEntregaPeticionDesarrollo{}, m)
		peticion = context.WithValue(peticion, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
		antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, fijo.perfilRef())
		instantanea, ok := s.instantaneaParaContexto(peticion, rutaEntregaPeticionCentro)
		if !ok || instantanea.AsignacionPerfil.PerfilActivoRef != fijo.perfilRef() ||
			!instantanea.AsignacionPerfil.Cubre(recurso) {
			t.Fatalf("%s no consumió su permiso: %v", caso.metodo, ok)
		}
		if historiaPerfilPostgreSQLPrueba(t, ctx, admin, fijo.perfilRef()) != antes {
			t.Fatalf("%s publicó por petición", caso.metodo)
		}
	}
}

// soportePerfilesFijosPostgreSQLPrueba compone el soporte de RRHH con sus
// perfiles fijos de alta y cobertura y asegura ambos como al arrancar.
func soportePerfilesFijosPostgreSQLPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (*soporteAltaContratacionTemporalDesarrollo, *perfilFijoCTDesarrollo, *perfilFijoCTDesarrollo) {
	t.Helper()
	s := soporteRRHHPostgreSQLPrueba(t, ctx, pool)
	if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, pool, s); err != nil {
		t.Fatal(err)
	}
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(s, principalDeSoportePrueba(s), time.Now().UTC().Truncate(time.Microsecond), origenEntregaPerfilFijoPrueba(t)); err != nil {
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
	if s.perfilFijoParaRuta(rutaEntregaPeticionCentro) == nil || s.perfilFijoParaRuta(httpinterno.RutaSubsanacionReparos) != nil {
		t.Fatal("perfil de entrega ausente o subsanación registrada antes de su configuración")
	}
	if err := asegurarPerfilesFijosCTDesarrollo(ctx, pool, s, aprobacionProvisionPerfilesRRHHDesarrollo{}, s.perfilesFijosRegistrados()...); err != nil {
		t.Fatalf("arranque de los perfiles fijos: %v", err)
	}
	return s, alta, cobertura
}

func TestPerfilesFijosRRHHSeConsumenSinPublicarPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s, alta, cobertura := soportePerfilesFijosPostgreSQLPrueba(t, ctx, gobierno)
	entrega := s.perfilFijoParaRuta(rutaEntregaPeticionCentro)
	lectorEntrega := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "GET")
	if entrega == nil || entrega == alta || entrega == cobertura || lectorEntrega == nil || lectorEntrega == entrega {
		t.Fatal("GET y POST de entrega no tienen perfiles propios")
	}
	analisis := s.perfilFijoParaRuta(httpinterno.RutaRectificacionAnalisisRRHH)
	if analisis == nil || analisis != s.perfilFijoParaRuta(httpinterno.RutaRegistroAnalisisRRHH) ||
		analisis == alta || analisis == cobertura {
		t.Fatal("el análisis no tiene su propio perfil fijo")
	}
	asignacion, informe := s.perfilFijoParaRuta(httpinterno.RutaAsignaciones), s.perfilFijoParaRuta(httpinterno.RutaPreparacionesInformeJuridico)
	if asignacion == nil || informe == nil || asignacion == informe {
		t.Fatal("la asignación y el informe no tienen su perfil fijo")
	}
	for _, p := range []*perfilFijoCTDesarrollo{alta, entrega, lectorEntrega, cobertura, analisis, asignacion, informe} {
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
	antesEntrega := historiaPerfilPostgreSQLPrueba(t, ctx, admin, entrega.perfilRef())
	antesLectorEntrega := historiaPerfilPostgreSQLPrueba(t, ctx, admin, lectorEntrega.perfilRef())
	antesAnalisis := historiaPerfilPostgreSQLPrueba(t, ctx, admin, analisis.perfilRef())
	antesAsignacion := historiaPerfilPostgreSQLPrueba(t, ctx, admin, asignacion.perfilRef())
	antesInforme := historiaPerfilPostgreSQLPrueba(t, ctx, admin, informe.perfilRef())
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
		}([]*perfilFijoCTDesarrollo{alta, entrega, lectorEntrega, cobertura, analisis, asignacion, informe}[i%7])
	}
	espera.Wait()
	if fallos != 0 {
		t.Fatalf("%d de 30 consumos simultáneos fallaron", fallos)
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, alta.perfilRef()) != antesAlta ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, entrega.perfilRef()) != antesEntrega ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, lectorEntrega.perfilRef()) != antesLectorEntrega ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, cobertura.perfilRef()) != antesCobertura ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, analisis.perfilRef()) != antesAnalisis ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, asignacion.perfilRef()) != antesAsignacion ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, informe.perfilRef()) != antesInforme {
		t.Fatal("los consumos escribieron en los perfiles fijos")
	}
	// Las rutas del perfil dinámico no tocan los perfiles fijos.
	if err := peticionRRHHPrueba(ctx, s, rolDeExpedientePrueba(s, "expediente:fijos:uno")); err != nil {
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
		analisis := s.perfilFijoParaRuta(httpinterno.RutaRegistroAnalisisRRHH)
		for _, cerrado := range []*perfilFijoCTDesarrollo{
			alta, analisis, s.perfilFijoParaRuta(httpinterno.RutaPreparacionesInformeJuridico),
			s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "GET"),
			s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "POST"),
		} {
			revocarPerfilFijoPrueba(t, ctx, gobierno, cerrado, estrechar)
			historia := historiaPerfilPostgreSQLPrueba(t, ctx, admin, cerrado.perfilRef())
			huella := huellaVigentePrueba(t, ctx, gobierno, cerrado)
			// Ni con la aprobación y la huella exacta del estado revocado.
			aprobacion := aprobacionProvisionPerfilesRRHHDesarrollo{referencia: "aprobacion:prueba:rrhh", preimagenes: map[string]bool{huella: true}}
			for i := 0; i < 2; i++ {
				if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, cerrado); ok {
					t.Fatalf("%s restringido=%v: consumió un perfil cerrado", cerrado.clave, estrechar)
				}
				if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, cerrado, aprobacion,
					preimagenPropiaPerfilFijoCTDesarrollo(cerrado, actoAsignacionPerfilFijoCTDesarrollo)); err != nil || estado != perfilFijoPendienteProvision {
					t.Fatalf("%s restringido=%v: rearranque %s %v", cerrado.clave, estrechar, estado, err)
				}
				if historiaPerfilPostgreSQLPrueba(t, ctx, admin, cerrado.perfilRef()) != historia {
					t.Fatalf("%s restringido=%v: se escribió sobre el perfil cerrado", cerrado.clave, estrechar)
				}
			}
		}
		// Cerrar el alta y el análisis no cierra la cobertura (perfiles independientes).
		if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, cobertura); !ok {
			t.Fatal("la cobertura quedó cerrada por revocar otros perfiles")
		}
	}
}

func TestPerfilesEntregaGETyPOSTRevocacionAisladaPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s, _, _ := soportePerfilesFijosPostgreSQLPrueba(t, ctx, gobierno)
	get := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "GET")
	post := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "POST")
	if get == nil || post == nil || get == post {
		t.Fatal("GET y POST sin perfiles separados")
	}
	revocarPerfilFijoPrueba(t, ctx, gobierno, get, false)
	historiaGET := historiaPerfilPostgreSQLPrueba(t, ctx, admin, get.perfilRef())
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, get); ok {
		t.Fatal("GET revocado siguió consumible")
	}
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, post); !ok {
		t.Fatal("revocar GET cerró también POST")
	}
	if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, get,
		aprobacionProvisionPerfilesRRHHDesarrollo{}, preimagenPropiaPerfilFijoCTDesarrollo(get, actoAsignacionPerfilFijoCTDesarrollo)); err != nil ||
		estado != perfilFijoPendienteProvision || historiaPerfilPostgreSQLPrueba(t, ctx, admin, get.perfilRef()) != historiaGET {
		t.Fatalf("GET revocado se reparó al arrancar: %s %v", estado, err)
	}
	revocarPerfilFijoPrueba(t, ctx, gobierno, post, false)
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, post); ok {
		t.Fatal("POST revocado siguió consumible")
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

// Intervención pasa a perfil fijo: con el permiso por expediente del binario
// anterior vigente, el arranque no escribe y la ruta se deniega; con la
// aprobación de esa huella exacta se sustituye por CAS; y revocado, ni con
// aprobación vuelve.
func TestIntervencionPasaAPerfilFijoConProvisionPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	v, err := s.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	fase := fasesOperacionPredeterminadasCT()[operacionFaseFiscalizacionCT]
	plantilla, err := nuevaInstantaneaAutorizacionFiscalizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef,
		time.Now().UTC().Truncate(time.Microsecond), fase)
	if err != nil {
		t.Fatal(err)
	}
	// Estado del binario anterior: el rol de Intervención con el expediente.
	anterior := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(plantilla)
	anterior.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
		{Clave: "expediente_ref", Valores: []string{"expediente:intervencion:previo"}},
		{Clave: "fase_previa", Valores: []string{"informe_juridico"}},
		{Clave: "estado_previo", Valores: []string{"en_curso"}},
	}
	s.instantanea = anterior
	if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, v.PerfilActivoRef)
	if antes.versiones != 1 || antes.acto != actoAsignacionCTDesarrollo {
		t.Fatalf("estado anterior inesperado: %+v", antes)
	}
	fijo, err := componerPerfilFijoIntervencionCTDesarrollo(ctx, gobierno, s, plantilla, aprobacionProvisionPerfilesRRHHDesarrollo{})
	if err != nil {
		t.Fatalf("sin aprobación el arranque se detuvo: %v", err)
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, v.PerfilActivoRef) != antes {
		t.Fatal("sin aprobación se escribió")
	}
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, fijo); ok {
		t.Fatal("se consumió el permiso por expediente")
	}
	aprobacion := aprobacionProvisionPerfilesRRHHDesarrollo{referencia: "aprobacion:prueba:intervencion",
		preimagenes: map[string]bool{huellaVigentePrueba(t, ctx, gobierno, fijo): true}}
	if fijo, err = componerPerfilFijoIntervencionCTDesarrollo(ctx, gobierno, s, plantilla, aprobacion); err != nil {
		t.Fatal(err)
	}
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, v.PerfilActivoRef); h.versiones != antes.versiones+1 || h.acto != actoAsignacionPerfilFijoCTDesarrollo {
		t.Fatalf("la provisión de Intervención no se aplicó: %+v", h)
	}
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, fijo); !ok {
		t.Fatal("Intervención no consume su perfil fijo")
	}
	// Rearrancar con la misma aprobación no escribe.
	provisionada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, v.PerfilActivoRef)
	if _, err := componerPerfilFijoIntervencionCTDesarrollo(ctx, gobierno, s, plantilla, aprobacion); err != nil ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, v.PerfilActivoRef) != provisionada {
		t.Fatalf("el rearranque escribió: %v", err)
	}
	revocarPerfilFijoPrueba(t, ctx, gobierno, fijo, false)
	cerrada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, v.PerfilActivoRef)
	revocada := aprobacionProvisionPerfilesRRHHDesarrollo{referencia: "aprobacion:prueba:intervencion",
		preimagenes: map[string]bool{huellaVigentePrueba(t, ctx, gobierno, fijo): true}}
	if _, err := componerPerfilFijoIntervencionCTDesarrollo(ctx, gobierno, s, plantilla, revocada); err != nil {
		t.Fatal(err)
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, v.PerfilActivoRef) != cerrada {
		t.Fatal("la provisión reactivó una Intervención revocada")
	}
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, fijo); ok {
		t.Fatal("se consumió una Intervención revocada")
	}
}
