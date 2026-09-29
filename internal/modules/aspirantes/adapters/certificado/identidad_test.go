package certificado

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"strings"
	"testing"
)

// Certificados sintéticos: solo el sujeto importa. Personas y números
// inventados.
func cert(atributos ...pkix.AttributeTypeAndValue) *x509.Certificate {
	return &x509.Certificate{Subject: pkix.Name{Names: atributos}}
}

func at(oid asn1.ObjectIdentifier, v any) pkix.AttributeTypeAndValue {
	return pkix.AttributeTypeAndValue{Type: oid, Value: v}
}

func TestPerfilFNMT(t *testing.T) {
	d, err := Leer(cert(at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES ÁLVAREZ"),
		at(oidNombreComun, "REYES ÁLVAREZ ANTONIO - 48123456G")))
	if err != nil || d.Nombre != "ANTONIO" || d.Apellidos != "REYES ÁLVAREZ" || d.Numero != "48123456G" || d.EsNIE {
		t.Fatalf("%+v %v", d, err)
	}
	d, err = Leer(cert(at(oidNumeroSerie, "IDCES-X1234567L"), at(oidNombre, "KARIM"), at(oidApellidos, "BENALI")))
	if err != nil || !d.EsNIE || d.Numero != "X1234567L" {
		t.Fatalf("NIE %+v %v", d, err)
	}
}

func TestPerfilDNIeConApellidosEnNombreComun(t *testing.T) {
	d, err := Leer(cert(at(oidNumeroSerie, "12345678Z"), at(oidNombre, "LUCÍA"), at(oidApellidos, "FERNÁNDEZ"),
		at(oidNombreComun, "FERNÁNDEZ MORENO, LUCÍA (AUTENTICACIÓN)")))
	if err != nil || d.Apellidos != "FERNÁNDEZ MORENO" {
		t.Fatalf("%+v %v", d, err)
	}
	// Un nombre común que no casa con el nombre no cambia los apellidos.
	d, _ = Leer(cert(at(oidNumeroSerie, "12345678Z"), at(oidNombre, "LUCÍA"), at(oidApellidos, "FERNÁNDEZ"),
		at(oidNombreComun, "OTRA PERSONA, MARÍA (AUTENTICACIÓN)")))
	if d.Apellidos != "FERNÁNDEZ" {
		t.Fatalf("apellidos ajenos %+v", d)
	}
}

func TestRechazos(t *testing.T) {
	casos := []*x509.Certificate{
		nil,
		cert(at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")),
		cert(at(oidNumeroSerie, "IDCES-48123456G"), at(oidApellidos, "REYES")),
		cert(at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES"), at(oidIdentificadorEntidad, "VATES-B00000000")),
		cert(at(oidNumeroSerie, "IDCES-48123456G"), at(oidNumeroSerie, "IDCES-12345678Z"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")),
		cert(at(oidNumeroSerie, "PASES-AB123"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")),
		cert(at(oidNumeroSerie, 12345678), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")),
		cert(at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "   "), at(oidApellidos, "REYES")),
	}
	for i, c := range casos {
		if _, err := Leer(c); err == nil {
			t.Fatalf("caso %d aceptado", i)
		}
	}
}

func TestDatosRedactados(t *testing.T) {
	d, _ := Leer(cert(at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")))
	if s := fmt.Sprintf("%v %+v %#v", d, d, d); strings.Contains(s, "48123456") || strings.Contains(s, "REYES") {
		t.Fatalf("fuga %s", s)
	}
}

func TestIdentidadDelDominio(t *testing.T) {
	id, err := Identidad(cert(at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES ÁLVAREZ")))
	if err != nil || id.Documento().Enmascarado() != "***2345**" || id.Valores()["apellidos"] != "REYES ÁLVAREZ" {
		t.Fatalf("%v %v", id.Valores(), err)
	}
	if _, err := Identidad(cert(at(oidNumeroSerie, "IDCES-48123456A"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES"))); err == nil {
		t.Fatal("letra de control incorrecta")
	}
	if _, err := Identidad(cert(at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "R2D2"), at(oidApellidos, "REYES"))); err == nil {
		t.Fatal("nombre inválido")
	}
}
