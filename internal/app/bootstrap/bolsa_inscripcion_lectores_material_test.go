package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestLectoraInscripcionSeparadaNoEntraEnConsultasCT(t *testing.T) {
	raiz, ca, entrada, _, _, certificadoIncorrecto := fixtureConsultaRRHHDesarrollo(t)
	if err := os.Mkdir(filepath.Join(raiz, "ca"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "ca", "ca.crt"),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	var huellaCarmen [sha256.Size]byte
	huellaCarmen[0] = 1
	baseIdentidad := identidadCertificadoDesarrollo{huella: huellaCarmen, principal: vecdomain.Principal{
		ID: "desarrollo:operador-rrhh", DisplayName: "Carmen Molina Ortega",
		Roles:      []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion":   config.ExecutionProfileDevelopment,
			"certificate_sha256": hex.EncodeToString(huellaCarmen[:])},
	}}
	base, err := nuevoResolvedorIdentidadDesarrollo(baseIdentidad)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.registrarLectoresRRHH([]identidadConsultaRRHHDesarrollo{{
		identidad: baseIdentidad, perfilRef: "prf_carmen_ct",
		organizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		clase:           ports.AmbitoOrganizacionRRHH, ambitoRef: organizacionAltaContratacionTemporalDesarrollo,
	}}); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{DevelopmentMaterialDir: raiz}
	if sinExtra, err := resolverLectoresNominalesInscripcionRRHH(cfg, base, nil); err != nil || sinExtra != base {
		t.Fatalf("sin lectora extra: %v", err)
	}
	local, err := resolverLectoresNominalesInscripcionRRHH(cfg, base,
		[]archivoEntradaConsultaRRHHDesarrollo{entrada})
	if err != nil || local == nil || local == base {
		t.Fatalf("lector nominal: %v", err)
	}
	if len(base.lectoresConsultaRRHH()) != 1 || len(base.porHuella) != 1 ||
		len(local.lectoresConsultaRRHH()) != 2 || len(local.porHuella) != 2 {
		t.Fatal("la lectora de Bolsa alteró el resolvedor de CT")
	}
	certificado, err := os.ReadFile(filepath.Join(raiz, entrada.Certificate))
	if err != nil {
		t.Fatal(err)
	}
	bloque, _ := pem.Decode(certificado)
	huellaElena := sha256.Sum256(bloque.Bytes)
	cuentas := []cuentaRutasDietasDesarrollo{
		{CertificadoSHA256: hex.EncodeToString(huellaCarmen[:]), Sujeto: baseIdentidad.principal.ID,
			CuentaRef: "cta_carmen", PerfilRef: "prf_carmen_ct"},
		{CertificadoSHA256: hex.EncodeToString(huellaElena[:]), Sujeto: entrada.Subject,
			CuentaRef: "cta_elena", PerfilRef: entrada.PerfilRef},
	}
	if rrhh, err := cuentasRRHHInscripcionDesarrollo(local, cuentas); err != nil || len(rrhh) != 2 {
		t.Fatalf("cuentas Bolsa: %v", err)
	}
	global := *base
	global.porHuella = local.porHuella
	if len(global.lectoresConsultaRRHH()) != 1 ||
		principalContratacionTemporalDesarrolloValidoParaRuta(local.porHuella[huellaElena], httpinterno.RutaConsultaCuadroRRHH) {
		t.Fatal("Elena entró en lecturas o rutas de CT")
	}
	if relectura, err := resolverLectoresNominalesInscripcionRRHH(cfg, &global,
		[]archivoEntradaConsultaRRHHDesarrollo{entrada}); err != nil || len(relectura.lectoresConsultaRRHH()) != 2 {
		t.Fatalf("actor global no pudo montar su bandeja nominal: %v", err)
	}
	for _, incorrecta := range []archivoEntradaConsultaRRHHDesarrollo{
		func() archivoEntradaConsultaRRHHDesarrollo { v := entrada; v.PerfilRef = "prf_ajeno"; return v }(),
		func() archivoEntradaConsultaRRHHDesarrollo {
			v := entrada
			v.Subject = baseIdentidad.principal.ID
			return v
		}(),
	} {
		if _, err := resolverLectoresNominalesInscripcionRRHH(cfg, base,
			[]archivoEntradaConsultaRRHHDesarrollo{incorrecta}); err == nil {
			t.Fatal("lector divergente aceptado")
		}
	}
	if len(base.lectoresConsultaRRHH()) != 1 || len(base.porHuella) != 1 {
		t.Fatal("un fallo opcional cambió el resolvedor de CT")
	}
	certificadoIncorrecto(t)
	if _, err := resolverLectoresNominalesInscripcionRRHH(cfg, base,
		[]archivoEntradaConsultaRRHHDesarrollo{entrada}); err == nil {
		t.Fatal("certificado fuera de la CA de desarrollo aceptado")
	}
}

func TestActorGlobalInscripcionExigeCuentaYMaterialExactos(t *testing.T) {
	raiz, ca, entrada, _, _, _ := fixtureConsultaRRHHDesarrollo(t)
	if err := os.Mkdir(filepath.Join(raiz, "ca"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "ca", "ca.crt"),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	base := &resolvedorIdentidadDesarrollo{
		porHuella:    map[[sha256.Size]byte]vecdomain.Principal{},
		lectoresRRHH: map[[sha256.Size]byte]identidadConsultaRRHHDesarrollo{},
		porSujeto:    map[string]adscripcionCentroDesarrollo{},
	}
	certPEM, err := os.ReadFile(filepath.Join(raiz, entrada.Certificate))
	if err != nil {
		t.Fatal(err)
	}
	bloque, _ := pem.Decode(certPEM)
	huella := sha256.Sum256(bloque.Bytes)
	cuenta := cuentaRutasDietasDesarrollo{CertificadoSHA256: hex.EncodeToString(huella[:]),
		Sujeto: entrada.Subject, CuentaRef: "cta_elena", PerfilRef: entrada.PerfilRef}
	motivo := motivoPresentarInscripcionBolsaDesarrollo()
	incorporar := motivo
	incorporar.EntradaClave = referenciaAltaContratacionTemporalDesarrollo("motivo_", "prueba-incorporar")
	manifest := map[string]any{
		"version": 1, "autoridad": AutoridadNoAutoritativa,
		"cuentas": []cuentaRutasDietasDesarrollo{cuenta}, "lectores": []archivoEntradaConsultaRRHHDesarrollo{entrada},
		"dsn_registro_identidad":     "postgres://registro@localhost/vec",
		"dsn_revalidacion_identidad": "postgres://revalidar@localhost/vec",
		"dsn_contexto":               "postgres://contexto@localhost/vec",
		"dsn_fuente_autorizacion":    "postgres://fuente@localhost/vec",
		"dsn_registro_autorizacion":  "postgres://auditoria@localhost/vec",
		"dsn_motivos":                "postgres://motivos@localhost/vec",
		"motivos":                    map[string]any{"decidir": motivo, "incorporar": incorporar},
	}
	ruta := filepath.Join(raiz, "identidad", "inscripcion-bolsa-rrhh.json")
	escribir := func() {
		t.Helper()
		contenido, e := json.Marshal(manifest)
		if e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(ruta, contenido, 0600); e != nil {
			t.Fatal(e)
		}
	}
	escribir()
	cfg := configInscripcionEncendidaPrueba(t)
	cfg.DevelopmentMaterialDir = raiz
	global, err := registrarActoresGlobalesInscripcionRRHH(cfg, base)
	if err != nil || global == nil || global == base || len(global.porHuella) != 1 ||
		len(global.lectoresConsultaRRHH()) != 0 || len(base.porHuella) != 0 {
		t.Fatalf("actor global y CT no están separados: %v", err)
	}
	manifest["cuentas"] = []cuentaRutasDietasDesarrollo{{CertificadoSHA256: cuenta.CertificadoSHA256,
		Sujeto: cuenta.Sujeto, CuentaRef: cuenta.CuentaRef, PerfilRef: "prf_ajeno"}}
	escribir()
	if incorrecto, err := registrarActoresGlobalesInscripcionRRHH(cfg, base); err == nil || incorrecto != base {
		t.Fatal("cuenta divergente habilitó al actor global")
	}
	var huellaCarmen [sha256.Size]byte
	huellaCarmen[0] = 1
	carmen := identidadCertificadoDesarrollo{huella: huellaCarmen, principal: vecdomain.Principal{
		ID: "desarrollo:operador-rrhh", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"certificate_sha256": hex.EncodeToString(huellaCarmen[:])},
	}}
	baseConCarmen := &resolvedorIdentidadDesarrollo{
		porHuella: map[[sha256.Size]byte]vecdomain.Principal{huellaCarmen: carmen.principal},
		lectoresRRHH: map[[sha256.Size]byte]identidadConsultaRRHHDesarrollo{huellaCarmen: {
			identidad: carmen, perfilRef: "prf_carmen_ct",
		}},
		porSujeto: map[string]adscripcionCentroDesarrollo{},
	}
	manifest["cuentas"] = []cuentaRutasDietasDesarrollo{{
		CertificadoSHA256: hex.EncodeToString(huellaCarmen[:]), Sujeto: carmen.principal.ID,
		CuentaRef: "cta_carmen_12345678901234567890123456789012", PerfilRef: "prf_carmen_ct",
	}}
	escribir()
	if sinElena, err := registrarActoresGlobalesInscripcionRRHH(cfg, baseConCarmen); err == nil ||
		sinElena != baseConCarmen || len(baseConCarmen.porHuella) != 1 {
		t.Fatal("una lectora sin cuenta propia recibió identidad global")
	}
}
