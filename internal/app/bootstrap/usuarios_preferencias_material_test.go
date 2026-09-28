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
	m := ports.MaterialPreferencias{
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
	if hex.EncodeToString(huella[:]) != "aa8e47b6de7519c4c2ca35179a88959263d553235f46e1b5a96dbd5b5cb67780" {
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
	if err != nil || huellaRecurso != "7cec22dba98ed3341ab10ef5bd6a259f8423c4ead83c15d42b6426f6160417ed" {
		t.Fatalf("contexto V3 divergente: %s %v", huellaRecurso, err)
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
	if !principalParaSuperficieUsuariosPreferenciasValido(identidad, exterior, "externa_personal") ||
		!principalParaSuperficieUsuariosPreferenciasValido(identidad, interna, "interna_corporativa") ||
		principalParaSuperficieUsuariosPreferenciasValido(identidad, exterior, "interna_corporativa") ||
		principalParaSuperficieUsuariosPreferenciasValido(identidad, interna, "externa_personal") {
		t.Fatal("sustitución de audiencia exterior/interior aceptada")
	}
}

func TestConfiguracionUsuariosResuelveCuentaSinPersonaDelCliente(t *testing.T) {
	var c configuracionUsuariosPreferenciasDesarrollo
	b := []byte(`{"version":1,"autoridad":"desarrollo-no-autoritativo","cuentas":[{"certificado_sha256":"` + strings.Repeat("a", 64) + `","sujeto":"sujeto","cuenta_ref":"cta_0123456789abcdefghijkl","perfil_ref":"prf_0123456789abcdefghijkl","superficie":"externa_personal"}]}`)
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Cuentas) != 1 || c.Cuentas[0].CertificadoSHA256 != strings.Repeat("a", 64) || c.Cuentas[0].CuentaRef != "cta_0123456789abcdefghijkl" || c.Cuentas[0].Superficie != "externa_personal" {
		t.Fatalf("cuenta de configuración no resuelta: %+v", c.Cuentas)
	}
}

func TestRutaPreferenciasNoExisteEnAPIPublicaAnonima(t *testing.T) {
	api, err := NewAPIPublicaBolsaWithConfig(configuracionAPIPrueba(config.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, metodo := range []string{http.MethodGet, http.MethodPut} {
		rec := httptest.NewRecorder()
		api.ServeHTTP(rec, httptest.NewRequest(metodo, usuarioshttp.RutaMisPreferencias, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("ruta privada en API pública: %s %d", metodo, rec.Code)
		}
	}
}

func TestPreflightUsuariosRechazaSQLAusenteAntesDePublicarMaterial(t *testing.T) {
	cfg := config.Config{DevelopmentMaterialDir: t.TempDir(), ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	if err := preflightSQLPreferenciasUsuariosDesarrollo(cfg); err == nil {
		t.Fatal("sin material/SQL Usuarios publicó audiencia V3")
	}
}

func TestDescriptorUsuariosSoloConSelectorYAudienciasDistintas(t *testing.T) {
	apagado := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{})
	encendido := append(append([]descriptorMaterialConsumidorV3Desarrollo(nil), apagado...), descriptoresMaterialPreferenciasUsuariosDesarrollo()...)
	if len(encendido) != len(apagado)+2 {
		t.Fatal("selector no publica dos audiencias exactas")
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(encendido)
	if err != nil {
		t.Fatal(err)
	}
	for _, audiencia := range []string{audienciaConsultaPreferenciasUsuarios, audienciaActualizacionPreferenciasUsuarios} {
		if _, ok := catalogo.descriptorPara(audiencia); !ok {
			t.Fatalf("audiencia ausente: %s", audiencia)
		}
	}
	previo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(apagado)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := previo.descriptorPara(audienciaConsultaPreferenciasUsuarios); ok {
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
