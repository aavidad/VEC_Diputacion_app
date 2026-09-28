package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestRecursoPreferenciasUsuariosVectorV3(t *testing.T) {
	for _, caso := range []struct {
		superficie         core.SuperficieAutenticacionActorV1
		material, contexto string
	}{
		{core.SuperficieAutenticacionInternaCorporativaV1, "93e51bfaf653b06d10b6033cd348af16ae15796e4c36d99fe7ae19fa6ccda517", "c00f649660c182645235e15e2e58a3d94c985dac9dca5e7b423b8a19bb03e96d"},
		{core.SuperficieAutenticacionExternaPersonalV1, "e22d42d7a997fdbdd9a71ff10e7b805767bc5b7d28bb61870b4097f78cd4e723", "4776384ae63a032cf2c69e245aa8323c1683cf00515a2a79bdbfa9a76b4ade41"},
	} {
		m := ports.MaterialPreferencias{Superficie: caso.superficie,
			PersonaRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl",
			Accion: ports.AccionActualizarPreferencias, FinalidadRef: ports.FinalidadPreferenciasPropias,
			CatalogoVersionRef: "usuarios-preferencias-v1", VersionEsperada: 0,
			ClaveOperacion: "operacion-1234567890",
			HuellaPeticion: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Valores:        domain.CatalogoBasePreferencias().Predeterminados,
		}
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		huella := sha256.Sum256(b)
		if hex.EncodeToString(huella[:]) != caso.material {
			t.Fatalf("material divergente: %s", b)
		}
		recurso, err := recursoPreferenciasUsuarios(m)
		if err != nil {
			t.Fatal(err)
		}
		if recurso.Referencia != m.PersonaRef || recurso.ModuloID != "usuarios" || recurso.Tipo != "preferencias_persona" || recurso.Ambitos["persona_ref"] != m.PersonaRef || len(recurso.Ambitos) != 1 || recurso.Atributos["material_sha256"] != hex.EncodeToString(huella[:]) {
			t.Fatalf("recurso V3 divergente: %+v", recurso)
		}
		huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
		if err != nil || huellaRecurso != caso.contexto {
			t.Fatalf("contexto V3 divergente: %s %v", huellaRecurso, err)
		}
	}
}

func TestAudienciasUsuariosNoIntercambianCertificados(t *testing.T) {
	hExterior := sha256.Sum256([]byte("certificado-exterior"))
	hInterna := sha256.Sum256([]byte("certificado-interno"))
	principal := func(rol, id string, h [32]byte) core.Principal {
		return core.Principal{ID: id, Roles: []string{rol}, AuthMethod: core.AuthMethodCertificate, AuthAssurance: core.AuthAssuranceHigh,
			Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": hex.EncodeToString(h[:])}}
	}
	exterior := principal("candidato_bolsa", "per_candidato_sintetico_1234567890123456", hExterior)
	interna := principal("tecnico_rrhh", "per_rrhh_sintetico_1234567890123456789", hInterna)
	identidad, err := nuevoResolvedorIdentidadDesarrollo(identidadCertificadoDesarrollo{huella: hExterior, principal: exterior}, identidadCertificadoDesarrollo{huella: hInterna, principal: interna})
	if err != nil {
		t.Fatal(err)
	}
	if err = identidad.registrarCandidatoBolsa(identidadCandidatoBolsaDesarrollo{identidad: identidadCertificadoDesarrollo{huella: hExterior, principal: exterior}, cuentaRef: "cta_candidato_sintetico_1234567890123456", personaRef: exterior.ID, perfilRef: "prf_candidato_sintetico_1234567890123456", candidatoRef: "can_candidato_sintetico_1234567890123456"}); err != nil {
		t.Fatal(err)
	}
	aspirante := principal("aspirante", "per_aspirante_sintetico_1234567890123", sha256.Sum256([]byte("certificado-aspirante")))
	if !principalParaSuperficieUsuariosPreferenciasValido(identidad, exterior, "externa_personal") ||
		!principalParaSuperficieUsuariosPreferenciasValido(identidad, aspirante, "externa_personal") ||
		!principalParaSuperficieUsuariosPreferenciasValido(identidad, interna, "interna_corporativa") ||
		principalParaSuperficieUsuariosPreferenciasValido(identidad, aspirante, "publica_anonima") {
		t.Fatal("la superficie usa rol candidato como permiso o admite público")
	}
}

func TestConfiguracionUsuariosResuelveCuentaSinPersonaDelCliente(t *testing.T) {
	var c configuracionUsuariosPreferenciasDesarrollo
	b := []byte(`{"version":1,"autoridad":"desarrollo-no-autoritativo","superficie":"externa_personal","cuentas":[{"certificado_sha256":"` + strings.Repeat("a", 64) + `","sujeto":"sujeto","cuenta_ref":"cta_0123456789abcdefghijkl","perfil_ref":"prf_0123456789abcdefghijkl"}]}`)
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Cuentas) != 1 || c.Cuentas[0].CertificadoSHA256 != strings.Repeat("a", 64) || c.Cuentas[0].CuentaRef != "cta_0123456789abcdefghijkl" || c.Superficie != core.SuperficieAutenticacionExternaPersonalV1 {
		t.Fatalf("cuenta de configuración no resuelta: %+v", c.Cuentas)
	}
}

func TestRutaPreferenciasNoExisteEnAPIPublicaAnonima(t *testing.T) {
	api, err := NewAPIPublicaBolsaWithConfig(configuracionAPIPrueba(config.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, metodo := range []string{http.MethodGet, http.MethodPut} {
		for _, ruta := range []string{usuarioshttp.RutaMisPreferencias, usuarioshttp.RutaMisPreferenciasAreaPersonal} {
			rec := httptest.NewRecorder()
			api.ServeHTTP(rec, httptest.NewRequest(metodo, ruta, nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("ruta privada en API pública: %s %d", metodo, rec.Code)
			}
		}
	}
}

func TestPreflightUsuariosRechazaSQLAusenteAntesDePublicarMaterial(t *testing.T) {
	cfg := config.Config{DevelopmentMaterialDir: t.TempDir(), ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	if err := preflightSQLPreferenciasUsuariosDesarrollo(cfg); err == nil {
		t.Fatal("sin material/SQL Usuarios publicó audiencia V3")
	}
}

func TestMontajeExigeCuentasPerfilesYPoolsSeparados(t *testing.T) {
	interna := &autoridadPreferenciasUsuariosDesarrollo{superficie: core.SuperficieAutenticacionInternaCorporativaV1, ruta: usuarioshttp.RutaMisPreferencias, logins: map[string]bool{"login-interno": true}, cuentas: map[string]cuentaUsuariosPreferenciasDesarrollo{"cert-i": {cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{CuentaRef: "cta_interna", PerfilRef: "prf_interno"}}}}
	externa := &autoridadPreferenciasUsuariosDesarrollo{superficie: core.SuperficieAutenticacionExternaPersonalV1, ruta: usuarioshttp.RutaMisPreferenciasAreaPersonal, logins: map[string]bool{"login-externo": true}, cuentas: map[string]cuentaUsuariosPreferenciasDesarrollo{"cert-e": {cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{CuentaRef: "cta_externa", PerfilRef: "prf_externo"}}}}
	if !superficiesPreferenciasSeparadas(interna, externa) {
		t.Fatal("montajes disjuntos rechazados")
	}
	externa.logins["login-interno"] = true
	if superficiesPreferenciasSeparadas(interna, externa) {
		t.Fatal("LOGIN compartido entre superficies")
	}
	delete(externa.logins, "login-interno")
	externa.cuentas["cert-i"] = externa.cuentas["cert-e"]
	if superficiesPreferenciasSeparadas(interna, externa) {
		t.Fatal("certificado compartido entre superficies")
	}
	delete(externa.cuentas, "cert-i")
	externa.cuentas["cert-e"] = cuentaUsuariosPreferenciasDesarrollo{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{CuentaRef: "cta_interna", PerfilRef: "prf_externo"}}
	if superficiesPreferenciasSeparadas(interna, externa) {
		t.Fatal("cuenta compartida entre superficies")
	}
	externa.cuentas["cert-e"] = cuentaUsuariosPreferenciasDesarrollo{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{CuentaRef: "cta_externa", PerfilRef: "prf_interno"}}
	if superficiesPreferenciasSeparadas(interna, externa) {
		t.Fatal("perfil compartido entre superficies")
	}
}

func TestPreflightRechazaCuentasMezcladasAntesDeAbrirSQL(t *testing.T) {
	interna := configuracionUsuariosPreferenciasDesarrollo{Superficie: core.SuperficieAutenticacionInternaCorporativaV1, Cuentas: []cuentaUsuariosPreferenciasDesarrollo{{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{CertificadoSHA256: strings.Repeat("a", 64), Sujeto: "sujeto-i", CuentaRef: "cta_i", PerfilRef: "prf_i"}}}}
	externa := configuracionUsuariosPreferenciasDesarrollo{Superficie: core.SuperficieAutenticacionExternaPersonalV1, Cuentas: []cuentaUsuariosPreferenciasDesarrollo{{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{CertificadoSHA256: strings.Repeat("b", 64), Sujeto: "sujeto-e", CuentaRef: "cta_e", PerfilRef: "prf_e"}}}}
	if !configuracionesPreferenciasSeparadas(interna, externa) {
		t.Fatal("configuración válida rechazada")
	}
	externa.Cuentas[0].CertificadoSHA256 = interna.Cuentas[0].CertificadoSHA256
	if configuracionesPreferenciasSeparadas(interna, externa) {
		t.Fatal("certificado mezclado pasó preflight")
	}
	externa.Cuentas[0].CertificadoSHA256 = strings.Repeat("b", 64)
	externa.Cuentas[0].PerfilRef = interna.Cuentas[0].PerfilRef
	if configuracionesPreferenciasSeparadas(interna, externa) {
		t.Fatal("perfil mezclado pasó preflight")
	}
}

func TestDescriptorUsuariosSoloConSelectorYAudienciasDistintas(t *testing.T) {
	apagado := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{})
	encendido := append(append([]descriptorMaterialConsumidorV3Desarrollo(nil), apagado...), descriptoresMaterialPreferenciasUsuariosDesarrollo()...)
	if len(encendido) != len(apagado)+4 {
		t.Fatal("selector no publica cuatro audiencias exactas")
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(encendido)
	if err != nil {
		t.Fatal(err)
	}
	for _, audiencia := range []string{audienciaConsultaPreferenciasUsuariosInterna, audienciaActualizacionPreferenciasUsuariosInterna, audienciaConsultaPreferenciasUsuariosExterna, audienciaActualizacionPreferenciasUsuariosExterna} {
		if _, ok := catalogo.descriptorPara(audiencia); !ok {
			t.Fatalf("audiencia ausente: %s", audiencia)
		}
	}
	previo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(apagado)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := previo.descriptorPara(audienciaConsultaPreferenciasUsuariosInterna); ok {
		t.Fatal("audiencia publicada sin selector")
	}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	t.Setenv(envUsuariosPreferenciasDesarrollo, "false")
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil || activo {
		t.Fatal("selector apagado publica Usuarios")
	}
	t.Setenv(envUsuariosPreferenciasDesarrollo, "true")
	activo, err = selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil || !activo {
		t.Fatal("selector autenticado no activa Usuarios")
	}
	if _, err = selectorCapacidadRRHHDesarrollo(config.Config{}, envUsuariosPreferenciasDesarrollo); err == nil {
		t.Fatal("selector público aceptó Usuarios")
	}
}
