package bootstrap

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestIdentidadAdministracionDesarrolloAusenteNoCreaMaterial(t *testing.T) {
	raiz := t.TempDir()
	identidad, err := cargarIdentidadAdministracionDesarrollo(raiz, nil)
	if err != nil || identidad != nil {
		t.Fatal("material opcional ausente no debe activar administración")
	}
	entradas, err := os.ReadDir(raiz)
	if err != nil || len(entradas) != 0 {
		t.Fatal("el loader creó material")
	}
}

func TestMaterialAdministracionMantieneCatalogoRRHHSeparado(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	antes, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil || antes.identidadAdministracion != nil {
		t.Fatalf("material sin ADMIN: %v", err)
	}
	ca, err := tls.LoadX509KeyPair(rutas.CACertificate, rutas.CAPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	certCA, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC()
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(999), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, NotBefore: ahora.Add(-time.Minute), NotAfter: ahora.Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, certCA, publica, ca.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(der)
	escribir := func(ruta string, contenido []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(cfg.DevelopmentMaterialDir, ruta), contenido, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribirJSON := func(ruta string, valor any) {
		t.Helper()
		contenido, err := json.Marshal(valor)
		if err != nil {
			t.Fatal(err)
		}
		escribir(ruta, contenido)
	}
	escribir("identidad/admin.crt", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	escribirJSON("identidad/admin-persona.json", archivoIdentidadDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, CertificateSHA256: hex.EncodeToString(huella[:]), Subject: "administracion-sintetica", Roles: []string{"administrador"}})
	escribirJSON("identidad/administracion.json", archivoAdministracionDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, Certificado: "identidad/admin.crt", Identidad: "identidad/admin-persona.json", CuentaRef: "cta_aaaaaaaaaaaaaaaaaaaaaa", CuentaOrdinariaRef: "cta_bbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccc", PersonaRef: "per_dddddddddddddddddddddd"})
	despues, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(antes.identidad, despues.identidad) || despues.identidad.administracion != nil {
		t.Fatal("ADMIN alteró el catálogo RRHH")
	}
	admin := despues.identidadAdministracion
	if admin == nil || admin.administracion == nil || len(admin.porHuella) != 1 || len(admin.porSujeto) != 0 {
		t.Fatal("catálogo ADMIN ausente o mezclado")
	}
	if _, existe := admin.porHuella[huella]; !existe {
		t.Fatal("certificado ADMIN ausente de su catálogo")
	}
	for huellaRRHH := range despues.identidad.porHuella {
		if _, existe := admin.porHuella[huellaRRHH]; existe {
			t.Fatal("certificado RRHH presente en ADMIN")
		}
	}
}

func TestIdentidadAdministracionDesarrolloNominalSeparada(t *testing.T) {
	raiz := t.TempDir()
	if err := os.Mkdir(filepath.Join(raiz, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC()
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(1), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	derCA, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, publica, privada)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	publicaCliente, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cliente := &x509.Certificate{SerialNumber: big.NewInt(2), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, cliente, ca, publicaCliente, privada)
	if err != nil {
		t.Fatal(err)
	}
	escribir := func(ruta string, valor any) {
		t.Helper()
		b, err := json.Marshal(valor)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(raiz, ruta), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(raiz, "identidad/admin.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(der)
	identidadArchivo := archivoIdentidadDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, CertificateSHA256: hex.EncodeToString(huella[:]), Subject: "administracion-sintetica", Roles: []string{"administrador"}}
	escribir("identidad/admin-persona.json", identidadArchivo)
	archivo := archivoAdministracionDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, Certificado: "identidad/admin.crt", Identidad: "identidad/admin-persona.json", CuentaRef: "cta_aaaaaaaaaaaaaaaaaaaaaa", CuentaOrdinariaRef: "cta_bbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccc", PersonaRef: "per_dddddddddddddddddddddd"}
	escribir("identidad/administracion.json", archivo)
	identidad, err := cargarIdentidadAdministracionDesarrollo(raiz, ca)
	if err != nil || identidad == nil || identidad.identidad.huella != huella || identidad.personaRef != archivo.PersonaRef || len(identidad.identidad.principal.Permissions) != 0 {
		t.Fatalf("identidad nominal no válida o permisos fabricados: %v", err)
	}
	if _, err := cargarIdentidadAdministracionDesarrollo(raiz, ca, identidad.identidad); err == nil {
		t.Fatal("certificado o sujeto reutilizado")
	}
	for nombre, cambiar := range map[string]func(*archivoAdministracionDesarrollo){
		"misma cuenta":     func(a *archivoAdministracionDesarrollo) { a.CuentaOrdinariaRef = a.CuentaRef },
		"cuenta inválida":  func(a *archivoAdministracionDesarrollo) { a.CuentaRef = "administrador" },
		"perfil inválido":  func(a *archivoAdministracionDesarrollo) { a.PerfilRef = "administrador" },
		"persona inválida": func(a *archivoAdministracionDesarrollo) { a.PersonaRef = "persona" },
		"ruta ascendente":  func(a *archivoAdministracionDesarrollo) { a.Certificado = "../admin.crt" },
		"autoridad real":   func(a *archivoAdministracionDesarrollo) { a.Autoridad = "corporativa" },
	} {
		t.Run(nombre, func(t *testing.T) {
			copia := archivo
			cambiar(&copia)
			escribir("identidad/administracion.json", copia)
			if _, err := cargarIdentidadAdministracionDesarrollo(raiz, ca); err == nil {
				t.Fatal("entrada inválida aceptada")
			}
		})
	}
	escribir("identidad/administracion.json", archivo)
	identidadArchivo.Roles = []string{"tecnico_rrhh"}
	escribir("identidad/admin-persona.json", identidadArchivo)
	if _, err := cargarIdentidadAdministracionDesarrollo(raiz, ca); err == nil {
		t.Fatal("identidad RRHH reutilizada como admin")
	}
}
