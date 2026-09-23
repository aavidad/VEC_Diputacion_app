package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestSoporteSesionBorradorBolsaMantienePrincipalYSeparaPerfilCT(t *testing.T) {
	directorio, soporte, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	ct, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	bolsa, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporte, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if bolsa.soporteCanal == nil {
		t.Fatal("no se construyó el soporte exclusivo para el proveedor de sesión")
	}
	datos, err := bolsa.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if ct.CuentaRef != datos.CuentaRef || ct.PrincipalID != datos.PrincipalID ||
		ct.PerfilActivoRef == datos.PerfilActivoRef || ct.ContextoActorRef == datos.ContextoActorRef ||
		ct.RegistroContextoRef == datos.RegistroContextoRef {
		t.Fatalf("contextos CT/Bolsa no quedaron separados: CT=%+v Bolsa=%+v", ct, datos)
	}
	// El soporte no implementa ResolverContexto: el único resolutor de la
	// cadena debe usar el proveedor con catálogo de fronteras inyectado,
	// que requiere registro, revalidador, ruta y certificado por petición.
	if bolsa.soporteCanal == soporte {
		t.Fatal("B-BACK reutiliza el soporte CT en vez de su canal dedicado")
	}
	ctEsperado, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	ctRepetido, err := ctEsperado.Vinculo.Datos()
	if err != nil || ctRepetido.PerfilActivoRef != ct.PerfilActivoRef ||
		ctRepetido.ContextoActorRef != ct.ContextoActorRef || ctRepetido.RegistroContextoRef != ct.RegistroContextoRef {
		t.Fatal("el wrapper CT dejó de conservar las referencias históricas")
	}
	manifiestoCT, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(ctEsperado.Resultado.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	manifiestoBolsa, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(bolsa.soporteCanal.contexto.Resultado.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	if manifiestoCT.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1 != manifiestoBolsa.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1 ||
		manifiestoCT.Persona.AcreditacionProcedenciaComponenteContextoActorV1 != manifiestoBolsa.Persona.AcreditacionProcedenciaComponenteContextoActorV1 ||
		manifiestoCT.Perfil.PerfilRef == manifiestoBolsa.Perfil.PerfilRef || manifiestoCT.Contexto.VinculoRef == manifiestoBolsa.Contexto.VinculoRef {
		t.Fatal("cuenta/persona no preservan acreditación CT o perfil/vínculo no quedaron separados")
	}
}

func TestContextoSituacionB2ExigeBolsaDeclaradaEnIdentidad(t *testing.T) {
	directorio, soporteCT, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	soporte, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporteCT, ahora)
	if err != nil {
		t.Fatal(err)
	}
	preparador := &preparadorBorradorLlamamientoDesarrollo{soporte: soporte}
	actor := soporte.soporteCanal.contexto.Resultado.Contexto
	resuelto, err := preparador.ResolverContextoSituacionParticipacion(context.Background(), actor, "bolsa:b2:desarrollo", "participacion:b2")
	if err != nil || resuelto.UnidadRef != soporte.unidadRef || resuelto.AmbitoRef != soporte.ambitoRef {
		t.Fatalf("bolsa nominal rechazada: %+v err=%v", resuelto, err)
	}
	if _, err := preparador.ResolverContextoSituacionParticipacion(context.Background(), actor, "bolsa:ajena", "participacion:b2"); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("bolsa ajena no fue denegada: %v", err)
	}
}

func TestSoporteSesionBorradorBolsaFallaCerradoAnteManifiestoInvalido(t *testing.T) {
	directorio, soporte, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	if _, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporte, ahora); !errors.Is(err, ErrMaterialDesarrolloInvalido) {
		t.Fatalf("ausente: %v", err)
	}
	casos := map[string]func(*archivoManifiestoIdentidadBorradorBolsaDesarrollo){
		"sujeto ajeno": func(m *archivoManifiestoIdentidadBorradorBolsaDesarrollo) { m.Sujeto = "desarrollo:ajeno" },
		"certificado ajeno": func(m *archivoManifiestoIdentidadBorradorBolsaDesarrollo) {
			m.CertificadoSHA256 = strings.Repeat("b", 64)
		},
		"perfil CT": func(m *archivoManifiestoIdentidadBorradorBolsaDesarrollo) {
			m.PerfilRef = referenciaAltaContratacionTemporalDesarrollo("prf_", principal.ID+"\x00"+principal.Attributes["certificate_sha256"]+"\x00perfil")
		},
		"perfil ausente": func(m *archivoManifiestoIdentidadBorradorBolsaDesarrollo) { m.PerfilRef = "" },
		"ámbito vacío":   func(m *archivoManifiestoIdentidadBorradorBolsaDesarrollo) { m.AmbitoRef = "" },
		"bolsas vacías":  func(m *archivoManifiestoIdentidadBorradorBolsaDesarrollo) { m.BolsasRef = nil },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, mutar)
			if _, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporte, ahora); !errors.Is(err, ErrMaterialDesarrolloInvalido) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestProveedorSesionBorradorBolsaResuelvePerfilNominalSinCruzarCT(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	directorio := t.TempDir()
	if err := os.Mkdir(filepath.Join(directorio, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, e.principal, e.reloj.Ahora(), nil)
	bolsa, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, e.soporte, e.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	ctPerfil := e.soporte.contexto.Resultado.Contexto.PerfilActivoRef
	bolsaPerfil := bolsa.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef
	if ctPerfil == bolsaPerfil {
		t.Fatal("el soporte Bolsa conserva el perfil CT")
	}
	// El doble de resolución representa el contexto durable que compone el
	// proveedor Bolsa; cuenta y persona siguen siendo los del mismo sujeto.
	e.resolutor.base = bolsa.soporteCanal.contexto.Resultado
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(bolsaPerfil)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	proveedor, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(bolsa.soporteCanal, e.registro, e.revalidador, e.reloj, e.resolutor, catalogo)
	if err != nil {
		t.Fatal(err)
	}
	contextoRuta := func(metodo, ruta string) context.Context {
		ctx := contextoRutaCoberturaDesarrolloPrueba(bolsa.soporteCanal, e.principal, ruta)
		canal := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		canal.certificadoVerificadoEn = e.reloj.Ahora().Add(-time.Second)
		canal.certificadoValidoHasta = e.reloj.Ahora().Add(time.Minute)
		ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
		descriptor, ok := catalogo.resolver(metodo, ruta)
		if !ok {
			t.Fatalf("frontera no declarada: %s %s", metodo, ruta)
		}
		return context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: metodo, ruta: ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: descriptor})
	}
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodPost, "/api/vec/bolsa/llamamientos/borradores"},
		{http.MethodGet, "/api/vec/bolsa/llamamientos/borradores/borrador-llamamiento:alta:" + strings.Repeat("a", 64)},
	} {
		resultado, err := proveedor.ResolverContexto(contextoRuta(caso.metodo, caso.ruta))
		if err != nil || resultado.Resultado.Contexto.PerfilActivoRef != bolsaPerfil ||
			resultado.Resultado.Contexto.PersonaRef != e.soporte.contexto.Resultado.Contexto.PersonaRef ||
			resultado.Resultado.Contexto.Instantanea.CuentaRef != e.soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef {
			t.Fatalf("%s %s: perfil/contexto Bolsa no resuelto: %+v err=%v", caso.metodo, caso.ruta, resultado.Resultado.Contexto, err)
		}
	}
	ctxAjeno := contextoRutaCoberturaDesarrolloPrueba(bolsa.soporteCanal, e.principal, "/api/vec/contratacion-temporal/consultas")
	ctxAjeno = context.WithValue(ctxAjeno, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: http.MethodPost, ruta: "/api/vec/contratacion-temporal/consultas", superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo})
	if _, err := proveedor.ResolverContexto(ctxAjeno); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatalf("el proveedor Bolsa admitió una ruta/perfil CT: %v", err)
	}
	// Aunque cuenta y persona coincidan, una respuesta del resolutor con el
	// perfil CT no puede satisfacer la ruta nominal Bolsa.
	e.resolutor.base = e.soporte.contexto.Resultado
	if _, err := proveedor.ResolverContexto(contextoRuta(http.MethodPost, "/api/vec/bolsa/llamamientos/borradores")); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatalf("la ruta Bolsa aceptó un contexto CT: %v", err)
	}
}

func fixtureSoporteSesionBorradorBolsa(t *testing.T) (string, *soporteAltaContratacionTemporalDesarrollo, dominiovec.Principal, time.Time) {
	t.Helper()
	directorio := t.TempDir()
	if err := os.Mkdir(filepath.Join(directorio, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	principal := dominiovec.Principal{
		ID: "desarrollo:rrhh-bback", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)},
	}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return directorio, &soporteAltaContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{}, principalID: principal.ID, certificadoSHA256: principal.Attributes["certificate_sha256"], contexto: contexto}, principal, ahora
}

func escribirManifiestoIdentidadBorradorBolsa(t *testing.T, directorio string, principal dominiovec.Principal, ahora time.Time, mutar func(*archivoManifiestoIdentidadBorradorBolsaDesarrollo)) {
	t.Helper()
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principal, ahora, discriminadorContextoSinteticoBorradorBolsaDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	datos, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto := archivoManifiestoIdentidadBorradorBolsaDesarrollo{Version: 2, Autoridad: AutoridadNoAutoritativa, Sujeto: principal.ID, CertificadoSHA256: principal.Attributes["certificate_sha256"], PerfilRef: datos.PerfilActivoRef, UnidadRef: "unidad:desarrollo:rrhh", AmbitoRef: "ambito:desarrollo:bolsa", BolsasRef: []string{"bolsa:b2:desarrollo"}}
	if mutar != nil {
		mutar(&manifiesto)
	}
	contenido, err := json.Marshal(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directorio, "identidad", nombreManifiestoIdentidadBorradorBolsaDesarrollo), contenido, 0600); err != nil {
		t.Fatal(err)
	}
}
