package bootstrap

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestCargarIdentidadesConsultasRRHHDesarrolloAusente(t *testing.T) {
	v, activo, err := cargarIdentidadesConsultasRRHHDesarrollo(t.TempDir(), nil)
	if err != nil || activo || v != nil {
		t.Fatalf("ausente: %v %v %v", v, activo, err)
	}
}

func TestCargarIdentidadesConsultasRRHHDesarrolloRechazaMaterialNoNominal(t *testing.T) {
	raiz, ca, entrada, escribir, escribirIdentidad, escribirCertificado := fixtureConsultaRRHHDesarrollo(t)
	manifiesto := func(e []archivoEntradaConsultaRRHHDesarrollo) archivoManifiestoConsultasRRHHDesarrollo {
		return archivoManifiestoConsultasRRHHDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, Entradas: e}
	}
	guardar := func(t *testing.T, e []archivoEntradaConsultaRRHHDesarrollo) { t.Helper(); escribir(t, manifiesto(e)) }
	guardar(t, []archivoEntradaConsultaRRHHDesarrollo{entrada})
	lectores, activo, err := cargarIdentidadesConsultasRRHHDesarrollo(raiz, ca)
	if err != nil || !activo || len(lectores) != 1 || lectores[0].clase != ports.AmbitoUnidadGestionRRHH || lectores[0].ambitoRef != entrada.AmbitoRef {
		t.Fatalf("unidad válida: %+v %v %v", lectores, activo, err)
	}
	rechaza := func(t *testing.T) {
		t.Helper()
		if _, _, err := cargarIdentidadesConsultasRRHHDesarrollo(raiz, ca); err != ErrMaterialDesarrolloInvalido {
			t.Fatalf("err=%v", err)
		}
	}
	for nombre, cambio := range map[string]func(*archivoEntradaConsultaRRHHDesarrollo){
		"ruta ascendente": func(e *archivoEntradaConsultaRRHHDesarrollo) { e.Certificate = "../lector.crt" },
		"ruta absoluta": func(e *archivoEntradaConsultaRRHHDesarrollo) {
			e.Identity = filepath.Join(raiz, "identidad", "lector.json")
		},
		"subject distinto": func(e *archivoEntradaConsultaRRHHDesarrollo) { e.Subject = "lector:ajeno" },
		"ámbito inválido":  func(e *archivoEntradaConsultaRRHHDesarrollo) { e.ClaseAmbito = "ajeno" },
		"organización divergente": func(e *archivoEntradaConsultaRRHHDesarrollo) {
			e.ClaseAmbito = string(ports.AmbitoOrganizacionRRHH)
			e.AmbitoRef = "organizacion:otra"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			e := entrada
			cambio(&e)
			guardar(t, []archivoEntradaConsultaRRHHDesarrollo{e})
			rechaza(t)
		})
	}
	t.Run("JSON desconocido y duplicado", func(t *testing.T) {
		for _, contenido := range [][]byte{[]byte(`{"version":1,"autoridad":"no_autoritativa","entradas":[],"ajeno":true}`), []byte(`{"version":1,"version":1,"autoridad":"no_autoritativa","entradas":[]}`)} {
			if err := os.WriteFile(filepath.Join(raiz, "identidad", nombreManifiestoConsultasRRHHDesarrollo), contenido, 0600); err != nil {
				t.Fatal(err)
			}
			rechaza(t)
		}
	})
	t.Run("rol incorrecto", func(t *testing.T) {
		escribirIdentidad(t, "lector_rrhh_ajeno")
		guardar(t, []archivoEntradaConsultaRRHHDesarrollo{entrada})
		rechaza(t)
		escribirIdentidad(t, "lector_rrhh")
	})
	t.Run("duplicado huella y sujeto", func(t *testing.T) { guardar(t, []archivoEntradaConsultaRRHHDesarrollo{entrada, entrada}); rechaza(t) })
	t.Run("certificado de otra CA", func(t *testing.T) {
		escribirCertificado(t)
		guardar(t, []archivoEntradaConsultaRRHHDesarrollo{entrada})
		rechaza(t)
	})
}

func TestCargarIdentidadesConsultasRRHHDesarrolloRegistraTecnicoExistente(t *testing.T) {
	raiz, ca, entrada, escribir, _, _ := fixtureConsultaRRHHDesarrollo(t)
	certificadoPEM, err := os.ReadFile(filepath.Join(raiz, "identidad", "lector.crt"))
	if err != nil {
		t.Fatal(err)
	}
	certificado, err := decodificarCertificadoUnico(certificadoPEM)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(certificado.Raw)
	const sujeto = "tecnico:rrhh"
	identidadJSON, err := json.Marshal(archivoIdentidadDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, CertificateSHA256: hex.EncodeToString(huella[:]), Subject: sujeto, DisplayName: "Técnico sintético", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(raiz, "identidad", "lector.json"), identidadJSON, 0600); err != nil {
		t.Fatal(err)
	}
	tecnico, err := cargarIdentidadDesarrollo(filepath.Join(raiz, "identidad", "lector.json"), certificado, rolTecnicoRRHHContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal(err)
	}
	entrada.Subject = sujeto
	escribir(t, archivoManifiestoConsultasRRHHDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, Entradas: []archivoEntradaConsultaRRHHDesarrollo{entrada}})
	lectores, activo, err := cargarIdentidadesConsultasRRHHDesarrollo(raiz, ca, tecnico)
	if err != nil || !activo || len(lectores) != 1 || lectores[0].identidad.huella != tecnico.huella || lectores[0].identidad.principal.ID != tecnico.principal.ID {
		t.Fatalf("técnico válido: %+v activo=%v err=%v", lectores, activo, err)
	}
	for nombre, cambiar := range map[string]func(*archivoEntradaConsultaRRHHDesarrollo){
		"subject técnico distinto": func(e *archivoEntradaConsultaRRHHDesarrollo) { e.Subject = "tecnico:ajeno" },
		"organización ajena":       func(e *archivoEntradaConsultaRRHHDesarrollo) { e.OrganizacionRef = "organizacion:ajena" },
	} {
		t.Run(nombre, func(t *testing.T) {
			e := entrada
			cambiar(&e)
			escribir(t, archivoManifiestoConsultasRRHHDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, Entradas: []archivoEntradaConsultaRRHHDesarrollo{e}})
			if _, _, err := cargarIdentidadesConsultasRRHHDesarrollo(raiz, ca, tecnico); err != ErrMaterialDesarrolloInvalido {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func fixtureConsultaRRHHDesarrollo(t *testing.T) (string, *x509.Certificate, archivoEntradaConsultaRRHHDesarrollo, func(*testing.T, archivoManifiestoConsultasRRHHDesarrollo), func(*testing.T, string), func(*testing.T)) {
	t.Helper()
	raiz := t.TempDir()
	if err := os.Mkdir(filepath.Join(raiz, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC()
	pubCA, privCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantillaCA := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	derCA, err := x509.CreateCertificate(rand.Reader, plantillaCA, plantillaCA, pubCA, privCA)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	crear := func(firmante *x509.Certificate, clave ed25519.PrivateKey) []byte {
		pub, _, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			t.Fatal(e)
		}
		der, e := x509.CreateCertificate(rand.Reader, &x509.Certificate{SerialNumber: big.NewInt(2), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}, firmante, pub, clave)
		if e != nil {
			t.Fatal(e)
		}
		return der
	}
	der := crear(ca, privCA)
	cert := func(b []byte) {
		if err := os.WriteFile(filepath.Join(raiz, "identidad", "lector.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: b}), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cert(der)
	escribirIdentidad := func(t *testing.T, rol string) {
		t.Helper()
		h := sha256.Sum256(der)
		b, e := json.Marshal(archivoIdentidadDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, CertificateSHA256: hex.EncodeToString(h[:]), Subject: "lector:unidad", DisplayName: "Lector sintético", Roles: []string{rol}})
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(raiz, "identidad", "lector.json"), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	escribirIdentidad(t, "lector_rrhh")
	escribir := func(t *testing.T, m archivoManifiestoConsultasRRHHDesarrollo) {
		t.Helper()
		b, e := json.Marshal(m)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(raiz, "identidad", nombreManifiestoConsultasRRHHDesarrollo), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	incorrecto := func(t *testing.T) {
		t.Helper()
		_, otra, _ := ed25519.GenerateKey(rand.Reader)
		_ = otra
		pub, priv, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			t.Fatal(e)
		}
		otraCA := &x509.Certificate{SerialNumber: big.NewInt(9), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
		derOtra, e := x509.CreateCertificate(rand.Reader, otraCA, otraCA, pub, priv)
		if e != nil {
			t.Fatal(e)
		}
		caOtra, e := x509.ParseCertificate(derOtra)
		if e != nil {
			t.Fatal(e)
		}
		cert(crear(caOtra, priv))
	}
	return raiz, ca, archivoEntradaConsultaRRHHDesarrollo{Certificate: "identidad/lector.crt", Identity: "identidad/lector.json", Subject: "lector:unidad", PerfilRef: "perfil:rrhh", OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ClaseAmbito: string(ports.AmbitoUnidadGestionRRHH), AmbitoRef: "unidad:rrhh:prueba"}, escribir, escribirIdentidad, incorrecto
}
