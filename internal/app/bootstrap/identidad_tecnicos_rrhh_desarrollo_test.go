package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/config"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestSegundoTecnicoRRHHDesarrolloAusenteYOriginalFijo(t *testing.T) {
	original := identidadRRHHOriginalPrueba()
	if adicional, err := cargarSegundoTecnicoRRHHDesarrollo(t.TempDir(), nil, original); err != nil || adicional != nil {
		t.Fatalf("sin suplemento: %+v, %v", adicional, err)
	}
	otro := original
	otro.huella = sha256.Sum256([]byte("otro-certificado"))
	otro.principal = clonarPrincipalDesarrollo(original.principal)
	otro.principal.ID = "desarrollo:otro-rrhh"
	otro.principal.Attributes["certificate_sha256"] = hex.EncodeToString(otro.huella[:])
	r, err := nuevoResolvedorIdentidadDesarrollo(original, otro)
	if err != nil || r.registrarTecnicoRRHHOriginal(original) != nil || r.registrarSegundoTecnicoRRHH(otro) != nil {
		t.Fatalf("registro de técnicos: %v", err)
	}
	elegido, ok := r.principalConRolUnico("tecnico_rrhh")
	if !ok || elegido.ID != original.principal.ID || elegido.Attributes["certificate_sha256"] != original.principal.Attributes["certificate_sha256"] {
		t.Fatalf("el original ha cambiado: %+v %v", elegido, ok)
	}
	if _, ok := r.principalConRolUnico("intervencion"); ok {
		t.Fatal("se eligió una familia distinta")
	}
	if err := r.registrarSegundoTecnicoRRHH(otro); err != ErrMaterialDesarrolloInvalido {
		t.Fatalf("duplicado aceptado: %v", err)
	}
}

func identidadRRHHOriginalPrueba() identidadCertificadoDesarrollo {
	huella := sha256.Sum256([]byte("hoja-original"))
	return identidadCertificadoDesarrollo{huella: huella, principal: vecdomain.Principal{
		ID: "desarrollo:original-rrhh", Roles: []string{"tecnico_rrhh"},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": hex.EncodeToString(huella[:])},
	}}
}

func TestSegundoTecnicoRRHHDesarrolloValidaHojaPersonaCuenta(t *testing.T) {
	raiz, ca, _, _, escribirIdentidad, cambiarCA := fixtureConsultaRRHHDesarrollo(t)
	original := identidadRRHHOriginalPrueba()
	escribirIdentidad(t, "tecnico_rrhh")
	certPEM, err := os.ReadFile(filepath.Join(raiz, "identidad/lector.crt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(raiz, "mtls"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, certificadoSegundoTecnicoRRHHDesarrollo), certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	identidadJSON, err := os.ReadFile(filepath.Join(raiz, "identidad/lector.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, identidadSegundoTecnicoRRHHDesarrollo), identidadJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	cert, err := decodificarCertificadoUnico(certPEM)
	if err != nil {
		t.Fatal(err)
	}
	sujeto := "lector:unidad"
	huella := sha256.Sum256(cert.Raw)
	base := sujeto + "\x00" + hex.EncodeToString(huella[:])
	manifiesto := archivoSegundoTecnicoRRHHDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa,
		Certificado: certificadoSegundoTecnicoRRHHDesarrollo, Identidad: identidadSegundoTecnicoRRHHDesarrollo, Sujeto: sujeto,
		CuentaRef:  referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta"),
		PersonaRef: referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona"),
		PerfilRef:  referenciaAltaContratacionTemporalDesarrollo("prf_", base+"\x00perfil")}
	guardar := func(m archivoSegundoTecnicoRRHHDesarrollo) {
		t.Helper()
		b, e := json.Marshal(m)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(raiz, "identidad", nombreSegundoTecnicoRRHHDesarrollo), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	comprobarRechazo := func() {
		t.Helper()
		if _, e := cargarSegundoTecnicoRRHHDesarrollo(raiz, ca, original); e != ErrMaterialDesarrolloInvalido {
			t.Fatalf("suplemento inválido aceptado: %v", e)
		}
	}
	guardar(manifiesto)
	got, err := cargarSegundoTecnicoRRHHDesarrollo(raiz, ca, original)
	if err != nil || got == nil || got.huella != huella || got.principal.ID != sujeto {
		t.Fatalf("suplemento válido: %+v %v", got, err)
	}
	casos := map[string]func(*archivoSegundoTecnicoRRHHDesarrollo){
		"persona original": func(m *archivoSegundoTecnicoRRHHDesarrollo) { m.Sujeto = original.principal.ID },
		"cuenta falsa":     func(m *archivoSegundoTecnicoRRHHDesarrollo) { m.CuentaRef = "cta_ajena" },
		"persona falsa":    func(m *archivoSegundoTecnicoRRHHDesarrollo) { m.PersonaRef = "per_ajena" },
		"perfil falso":     func(m *archivoSegundoTecnicoRRHHDesarrollo) { m.PerfilRef = "prf_ajeno" },
		"ruta ascendente":  func(m *archivoSegundoTecnicoRRHHDesarrollo) { m.Certificado = "../lector.crt" },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) { m := manifiesto; cambiar(&m); guardar(m); comprobarRechazo() })
	}
	guardar(manifiesto)
	cambiarCA(t)
	certPEM, err = os.ReadFile(filepath.Join(raiz, "identidad/lector.crt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, certificadoSegundoTecnicoRRHHDesarrollo), certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	comprobarRechazo()
}

func TestSegundoTecnicoRRHHDesarrolloRechazaDuplicados(t *testing.T) {
	original := identidadRRHHOriginalPrueba()
	otro := original
	if segundoTecnicoRRHHDistintoDeIdentidades(otro, []identidadCertificadoDesarrollo{original}) {
		t.Fatal("huella y persona duplicadas")
	}
	otro.huella = sha256.Sum256([]byte("otra-hoja"))
	if segundoTecnicoRRHHDistintoDeIdentidades(otro, []identidadCertificadoDesarrollo{original}) {
		t.Fatal("persona duplicada")
	}
	otro.principal.ID = "desarrollo:otra-persona"
	if !segundoTecnicoRRHHDistintoDeIdentidades(otro, []identidadCertificadoDesarrollo{original}) {
		t.Fatal("persona y certificado distintos rechazados")
	}
}

func TestMaterialDesarrolloConSuplementoRRHHInvalidoConservaOriginal(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	comprobar := func() {
		t.Helper()
		material, err := cargarMaterialSeguridadDesarrollo(cfg)
		if err != nil {
			t.Fatalf("el primario dejó de arrancar: %v", err)
		}
		principal, valido := material.identidad.principalConRolUnico("tecnico_rrhh")
		if !valido || principal.ID == "" || material.identidad.segundoTecnicoRRHH != nil {
			t.Fatalf("selección original alterada: %+v, %v", principal, valido)
		}
	}
	comprobar()
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", nombreSegundoTecnicoRRHHDesarrollo)
	if err := os.WriteFile(ruta, []byte(`{"version":1,"autoridad":"no_autoritativo","certificado":"../ajeno"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	comprobar()
}
