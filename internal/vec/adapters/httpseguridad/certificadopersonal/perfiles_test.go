package certificadopersonal

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const dniSintetico = "00000000T"

func oidPrueba(valor string) asn1.ObjectIdentifier {
	var resultado asn1.ObjectIdentifier
	for _, parte := range strings.Split(valor, ".") {
		var numero int
		if _, err := fmt.Sscanf(parte, "%d", &numero); err != nil {
			panic(err)
		}
		resultado = append(resultado, numero)
	}
	return resultado
}

func derPrueba(t *testing.T, valor any) []byte {
	t.Helper()
	der, err := asn1.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

func nombrePrueba(documento string) pkix.RDNSequence {
	return pkix.RDNSequence{
		{{Type: oidPrueba("2.5.4.6"), Value: "ES"}},
		{{Type: oidPrueba("2.5.4.5"), Value: documento}},
		{{Type: oidPrueba("2.5.4.3"), Value: "Persona sintética"}},
	}
}

func sanPrueba(t *testing.T, nombres ...pkix.RDNSequence) []byte {
	t.Helper()
	var contenido []byte
	for _, nombre := range nombres {
		contenido = append(contenido, derPrueba(t, asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 4, IsCompound: true, Bytes: derPrueba(t, nombre)})...)
	}
	return derPrueba(t, asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSequence, IsCompound: true, Bytes: contenido})
}

func nifSANPrueba(documento string) pkix.RDNSequence {
	return pkix.RDNSequence{{{Type: oidPrueba("1.3.6.1.4.1.5734.1.4"), Value: documento}}}
}

func politicasPrueba(t *testing.T, oids ...string) []byte {
	t.Helper()
	var politicas []struct{ ID asn1.ObjectIdentifier }
	for _, oid := range oids {
		politicas = append(politicas, struct{ ID asn1.ObjectIdentifier }{oidPrueba(oid)})
	}
	return derPrueba(t, politicas)
}

func sustituirExtension(cert *x509.Certificate, oid string, valor []byte) {
	for i, extension := range cert.ExtraExtensions {
		if extension.Id.String() == oid {
			cert.ExtraExtensions[i].Value = valor
			return
		}
	}
	cert.ExtraExtensions = append(cert.ExtraExtensions, pkix.Extension{Id: oidPrueba(oid), Value: valor})
}

func certificadoPrueba(t *testing.T, tipo string, mutar func(*x509.Certificate)) *x509.Certificate {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC()
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature}
	politica := "2.16.724.1.2.2.2.4"
	documento := dniSintetico
	if tipo != "dnie" {
		politica = "1.3.6.1.4.1.5734.3.20.1.0"
		if tipo == "fnmt_rsa" {
			politica = "1.3.6.1.4.1.5734.3.10.1"
			cert.KeyUsage |= x509.KeyUsageKeyEncipherment
		}
		cert.KeyUsage |= x509.KeyUsageContentCommitment
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		cert.UnknownExtKeyUsage = []asn1.ObjectIdentifier{oidPrueba("1.3.6.1.4.1.311.10.3.12"), oidPrueba("1.2.840.113583.1.1.5")}
		documento = "IDCES-" + documento
		sustituirExtension(cert, "2.5.29.17", sanPrueba(t, nifSANPrueba(dniSintetico)))
	}
	cert.RawSubject = derPrueba(t, nombrePrueba(documento))
	sustituirExtension(cert, "2.5.29.32", politicasPrueba(t, politica))
	if mutar != nil {
		mutar(cert)
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	// Sólo transporte de DER sintético, sin simular validación de cadena TLS.
	return &x509.Certificate{Raw: der}
}

func TestExtraerPerfilesPersonalesYCotejoPrivado(t *testing.T) {
	for _, tipo := range []string{"fnmt_rsa", "fnmt_g2", "dnie"} {
		t.Run(tipo, func(t *testing.T) {
			cert := certificadoPrueba(t, tipo, nil)
			perfil, err := ExtraerPerfilPersonal(cert)
			metodo := domain.AuthMethodCertificate
			if tipo == "dnie" {
				metodo = domain.AuthMethodDNIe
			}
			if err != nil || perfil.MetodoDelPerfil() != metodo || !perfil.CoincideDocumentoAutorizado([]byte(dniSintetico)) {
				t.Fatalf("perfil o cotejo válido denegados: %v", err)
			}
			for _, invalido := range []string{"", "00000001R", "00000000R", "00000000t", "0000000T", " 00000000T", "IDCES-00000000T", "X0000000T"} {
				if perfil.CoincideDocumentoAutorizado([]byte(invalido)) {
					t.Fatal("cotejo admitió documento distinto o no canónico")
				}
			}
			// La proyección mutable de x509.Certificate no cambia el DER analizado.
			cert.Subject.SerialNumber = "IDCES-00000001R"
			cert.KeyUsage = x509.KeyUsageCertSign
			segundo, err := ExtraerPerfilPersonal(cert)
			if err != nil || !segundo.CoincideDocumentoAutorizado([]byte(dniSintetico)) {
				t.Fatal("se confiaron campos mutables ajenos al DER")
			}
		})
	}
}

func TestPerfilPersonalNoAfirmaConfianzaNiVigencia(t *testing.T) {
	cert := certificadoPrueba(t, "dnie", func(c *x509.Certificate) { c.NotBefore = time.Unix(1, 0); c.NotAfter = time.Unix(2, 0) })
	perfil, err := ExtraerPerfilPersonal(cert)
	if err != nil || perfil.MetodoDelPerfil() != domain.AuthMethodDNIe {
		t.Fatal("el parser asumió la responsabilidad de cadena o vigencia")
	}
	// Certificado autofirmado y caducado: el llamador TLS debe rechazarlo antes
	// de llegar al parser. El resultado nunca basta para autenticar a la persona.
}

func TestPerfilPersonalDeniegaASN1AmbiguoYPerfilesContrarios(t *testing.T) {
	casos := []struct {
		nombre, tipo string
		mutar        func(*x509.Certificate)
	}{
		{"política desconocida", "dnie", func(c *x509.Certificate) { sustituirExtension(c, "2.5.29.32", politicasPrueba(t, "1.2.3.4")) }},
		{"firma DNIe", "dnie", func(c *x509.Certificate) {
			sustituirExtension(c, "2.5.29.32", politicasPrueba(t, "2.16.724.1.2.2.2.3"))
		}},
		{"firma centralizada", "dnie", func(c *x509.Certificate) {
			sustituirExtension(c, "2.5.29.32", politicasPrueba(t, "2.16.724.1.2.2.2.11"))
		}},
		{"prefijo de política", "dnie", func(c *x509.Certificate) {
			sustituirExtension(c, "2.5.29.32", politicasPrueba(t, "2.16.724.1.2.2.2.4.3.4"))
		}},
		{"políticas cruzadas", "fnmt_rsa", func(c *x509.Certificate) {
			sustituirExtension(c, "2.5.29.32", politicasPrueba(t, "1.3.6.1.4.1.5734.3.10.1", "2.16.724.1.2.2.2.4"))
		}},
		{"política duplicada", "dnie", func(c *x509.Certificate) {
			sustituirExtension(c, "2.5.29.32", politicasPrueba(t, "2.16.724.1.2.2.2.4", "2.16.724.1.2.2.2.4"))
		}},
		{"DNIe EKU cliente", "dnie", func(c *x509.Certificate) { c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth} }},
		{"DNIe KU firma", "dnie", func(c *x509.Certificate) { c.KeyUsage |= x509.KeyUsageContentCommitment }},
		{"FNMT EKU servidor", "fnmt_g2", func(c *x509.Certificate) { c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth} }},
		{"FNMT EKU cualquiera", "fnmt_g2", func(c *x509.Certificate) { c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageAny} }},
		{"FNMT EKU ausente", "fnmt_g2", func(c *x509.Certificate) { c.ExtKeyUsage = nil; c.UnknownExtKeyUsage = nil }},
		{"FNMT EKU duplicado", "fnmt_g2", func(c *x509.Certificate) {
			c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageClientAuth}
		}},
		{"FNMT KU contrario", "fnmt_g2", func(c *x509.Certificate) { c.KeyUsage |= x509.KeyUsageKeyEncipherment }},
		{"autoridad certificadora", "dnie", func(c *x509.Certificate) { c.IsCA = true }},
		{"sin restricciones básicas", "dnie", func(c *x509.Certificate) { c.BasicConstraintsValid = false }},
		{"sin documento", "dnie", func(c *x509.Certificate) {
			c.RawSubject = derPrueba(t, pkix.RDNSequence{{{Type: oidPrueba("2.5.4.6"), Value: "ES"}}})
		}},
		{"letra incorrecta", "dnie", func(c *x509.Certificate) { c.RawSubject = derPrueba(t, nombrePrueba("00000000R")) }},
		{"DNIe prefijo FNMT", "dnie", func(c *x509.Certificate) { c.RawSubject = derPrueba(t, nombrePrueba("IDCES-"+dniSintetico)) }},
		{"FNMT sin prefijo", "fnmt_rsa", func(c *x509.Certificate) { c.RawSubject = derPrueba(t, nombrePrueba(dniSintetico)) }},
		{"Subject duplicado igual", "dnie", func(c *x509.Certificate) {
			n := nombrePrueba(dniSintetico)
			n = append(n, n[1])
			c.RawSubject = derPrueba(t, n)
		}},
		{"Subject duplicado distinto mismo RDN", "dnie", func(c *x509.Certificate) {
			n := nombrePrueba(dniSintetico)
			n[1] = append(n[1], pkix.AttributeTypeAndValue{Type: oidPrueba("2.5.4.5"), Value: "00000001R"})
			c.RawSubject = derPrueba(t, n)
		}},
		{"SAN contradice Subject", "fnmt_rsa", func(c *x509.Certificate) { sustituirExtension(c, "2.5.29.17", sanPrueba(t, nifSANPrueba("00000001R"))) }},
		{"SAN dos directoryName iguales", "fnmt_rsa", func(c *x509.Certificate) {
			sustituirExtension(c, "2.5.29.17", sanPrueba(t, nifSANPrueba(dniSintetico), nifSANPrueba(dniSintetico)))
		}},
		{"SAN dos directoryName distintos", "fnmt_rsa", func(c *x509.Certificate) {
			sustituirExtension(c, "2.5.29.17", sanPrueba(t, nifSANPrueba(dniSintetico), nifSANPrueba("00000001R")))
		}},
		{"SAN NIF duplicado", "fnmt_g2", func(c *x509.Certificate) {
			n := nifSANPrueba(dniSintetico)
			n = append(n, n[0])
			sustituirExtension(c, "2.5.29.17", sanPrueba(t, n))
		}},
		{"SAN extensión duplicada", "fnmt_rsa", func(c *x509.Certificate) { c.ExtraExtensions = append(c.ExtraExtensions, c.ExtraExtensions[0]) }},
		{"SAN ausente", "fnmt_rsa", func(c *x509.Certificate) { c.ExtraExtensions = c.ExtraExtensions[1:] }},
		{"DNIe SAN directoryName", "dnie", func(c *x509.Certificate) {
			sustituirExtension(c, "2.5.29.17", sanPrueba(t, nifSANPrueba(dniSintetico)))
		}},
		{"SAN ASN1 inválido", "fnmt_rsa", func(c *x509.Certificate) { sustituirExtension(c, "2.5.29.17", []byte{0x30, 0x03, 0xa4}) }},
		{"país distinto", "dnie", func(c *x509.Certificate) {
			n := nombrePrueba(dniSintetico)
			n[0][0].Value = "FR"
			c.RawSubject = derPrueba(t, n)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			perfil, err := ExtraerPerfilPersonal(certificadoPrueba(t, caso.tipo, caso.mutar))
			if !errors.Is(err, ErrPerfilPersonalInvalido) || perfil != (PerfilPersonal{}) || perfil.MetodoDelPerfil() != "" || perfil.CoincideDocumentoAutorizado([]byte(dniSintetico)) {
				t.Fatal("entrada ambigua o contraria aceptada")
			}
			if strings.Contains(err.Error(), dniSintetico) {
				t.Fatal("documento filtrado en error")
			}
		})
	}
	for _, cert := range []*x509.Certificate{nil, {}, {Raw: []byte{0x30}}, {Raw: make([]byte, (64<<10)+1)}} {
		perfil, err := ExtraerPerfilPersonal(cert)
		if !errors.Is(err, ErrPerfilPersonalInvalido) || perfil != (PerfilPersonal{}) {
			t.Fatal("DER ausente o inválido aceptado")
		}
	}
}

func TestPerfilPersonalNoFiltraDocumentoEnFormatos(t *testing.T) {
	perfil, err := ExtraerPerfilPersonal(certificadoPrueba(t, "fnmt_g2", nil))
	if err != nil {
		t.Fatal(err)
	}
	for _, valor := range []any{perfil, &perfil, struct{ Perfil PerfilPersonal }{perfil}} {
		for _, formato := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x"} {
			if strings.Contains(fmt.Sprintf(formato, valor), dniSintetico) {
				t.Fatal("documento expuesto por fmt")
			}
		}
		if salida, err := json.Marshal(valor); err == nil || bytes.Contains(salida, []byte(dniSintetico)) {
			t.Fatal("perfil serializado a JSON")
		}
		if salida, err := xml.Marshal(valor); err == nil || bytes.Contains(salida, []byte(dniSintetico)) {
			t.Fatal("perfil serializado a XML")
		}
	}
	var salida bytes.Buffer
	if err := gob.NewEncoder(&salida).Encode(perfil); err == nil || bytes.Contains(salida.Bytes(), []byte(dniSintetico)) {
		t.Fatal("perfil serializado a Gob")
	}
	salida.Reset()
	slog.New(slog.NewJSONHandler(&salida, nil)).Info("perfil", "valor", perfil)
	if bytes.Contains(salida.Bytes(), []byte(dniSintetico)) {
		t.Fatal("documento expuesto por slog")
	}
	var cero PerfilPersonal
	if cero.MetodoDelPerfil() != "" || cero.CoincideDocumentoAutorizado(nil) || cero.CoincideDocumentoAutorizado([]byte(dniSintetico)) {
		t.Fatal("valor cero aceptado")
	}
	if json.Unmarshal([]byte(`{"documento":"00000000T"}`), &cero) == nil || cero != (PerfilPersonal{}) {
		t.Fatal("perfil fabricado desde JSON")
	}
}
