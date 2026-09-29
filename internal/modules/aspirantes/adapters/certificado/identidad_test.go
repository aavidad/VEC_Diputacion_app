package certificado

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Certificados sintéticos: solo importan el sujeto y la política. Personas y
// números inventados.
var (
	politicaFNMT     = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 5734, 3, 10, 1}
	politicaDNIe     = asn1.ObjectIdentifier{2, 16, 724, 1, 2, 2, 2, 3}
	politicaEmpleado = asn1.ObjectIdentifier{2, 16, 724, 1, 3, 5, 7, 1}
)

func cert(politica asn1.ObjectIdentifier, atributos ...pkix.AttributeTypeAndValue) *x509.Certificate {
	c := &x509.Certificate{Subject: pkix.Name{Names: atributos}}
	if politica != nil {
		c.PolicyIdentifiers = []asn1.ObjectIdentifier{politica}
	}
	return c
}

func at(oid asn1.ObjectIdentifier, v any) pkix.AttributeTypeAndValue {
	return pkix.AttributeTypeAndValue{Type: oid, Value: v}
}

func lector(t *testing.T) *Lector {
	t.Helper()
	l, err := NuevoLector(PerfilesOficiales())
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestPerfilFNMT(t *testing.T) {
	d, err := lector(t).Leer(cert(politicaFNMT, at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES ÁLVAREZ"),
		at(oidNombreComun, "REYES ÁLVAREZ ANTONIO - 48123456G")))
	if err != nil || d.Nombre() != "ANTONIO" || d.Apellidos() != "REYES ÁLVAREZ" || d.Numero() != "48123456G" || d.EsNIE() {
		t.Fatalf("%v", err)
	}
	d, err = lector(t).Leer(cert(politicaFNMT, at(oidNumeroSerie, "IDCES-X1234567L"), at(oidNombre, "KARIM"), at(oidApellidos, "BENALI")))
	if err != nil || !d.EsNIE() || d.Numero() != "X1234567L" {
		t.Fatalf("NIE %v", err)
	}
}

func TestPerfilDNIeExigeApellidosCompletos(t *testing.T) {
	d, err := lector(t).Leer(cert(politicaDNIe, at(oidNumeroSerie, "12345678Z"), at(oidNombre, "LUCÍA"), at(oidApellidos, "FERNÁNDEZ"),
		at(oidNombreComun, "FERNÁNDEZ MORENO, LUCÍA (AUTENTICACIÓN)")))
	if err != nil || d.Apellidos() != "FERNÁNDEZ MORENO" {
		t.Fatalf("%v", err)
	}
	for _, cn := range []string{"OTRA PERSONA, MARÍA (AUTENTICACIÓN)", "FERNÁNDEZ MORENO LUCÍA", "GARCÍA FERNÁNDEZ, LUCÍA (AUTENTICACIÓN)"} {
		if _, err := lector(t).Leer(cert(politicaDNIe, at(oidNumeroSerie, "12345678Z"), at(oidNombre, "LUCÍA"), at(oidApellidos, "FERNÁNDEZ"), at(oidNombreComun, cn))); err == nil {
			t.Fatalf("CN %q aceptado", cn)
		}
	}
	// En la FNMT el CN no cambia los apellidos.
	d, _ = lector(t).Leer(cert(politicaFNMT, at(oidNumeroSerie, "IDCES-12345678Z"), at(oidNombre, "LUCÍA"), at(oidApellidos, "FERNÁNDEZ"),
		at(oidNombreComun, "FERNÁNDEZ MORENO, LUCÍA (AUTENTICACIÓN)")))
	if d.Apellidos() != "FERNÁNDEZ" {
		t.Fatal("el CN solo cuenta en el DNIe")
	}
}

func TestRechazos(t *testing.T) {
	bien := []pkix.AttributeTypeAndValue{at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")}
	casos := []*x509.Certificate{
		nil,
		cert(nil, bien...),              // sin política
		cert(politicaEmpleado, bien...), // empleado público u otra política
		cert(politicaFNMT, at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")),
		cert(politicaFNMT, at(oidNumeroSerie, "IDCES-48123456G"), at(oidApellidos, "REYES")),
		cert(politicaFNMT, append(append([]pkix.AttributeTypeAndValue{}, bien...), at(oidIdentificadorEntidad, "VATES-B00000000"))...),
		cert(politicaFNMT, at(oidNumeroSerie, "IDCES-48123456G"), at(oidNumeroSerie, "IDCES-12345678Z"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")),
		cert(politicaFNMT, at(oidNumeroSerie, "PASES-AB123"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")),
		cert(politicaFNMT, at(oidNumeroSerie, 12345678), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")),
		cert(politicaFNMT, at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "   "), at(oidApellidos, "REYES")),
	}
	dos := cert(politicaFNMT, bien...)
	dos.PolicyIdentifiers = append(dos.PolicyIdentifiers, politicaDNIe)
	casos = append(casos, dos) // dos perfiles a la vez
	for i, c := range casos {
		if _, err := lector(t).Leer(c); err == nil {
			t.Fatalf("caso %d aceptado", i)
		}
	}
	for _, p := range []map[string]Perfil{nil, {"1.2": PerfilFNMT}, {"1.2.x": PerfilFNMT}, {"1.3.6.1.4.1.9": "otro"}} {
		if _, err := NuevoLector(p); err == nil {
			t.Fatalf("configuración %v aceptada", p)
		}
	}
}

func TestDatosRedactados(t *testing.T) {
	d, _ := lector(t).Leer(cert(politicaFNMT, at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES")))
	j, _ := json.Marshal(struct{ D Datos }{d})
	if s := fmt.Sprintf("%v %+v %#v %d %s", d, d, d, d, j); strings.Contains(s, "48123456") || strings.Contains(s, "REYES") {
		t.Fatalf("fuga %s", s)
	}
}

func TestIdentidadDelDominio(t *testing.T) {
	id, err := lector(t).Identidad(cert(politicaFNMT, at(oidNumeroSerie, "IDCES-48123456G"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES ÁLVAREZ")))
	if err != nil || id.Documento().Enmascarado() != "***2345**" || id.Valores()["apellidos"] != "REYES ÁLVAREZ" {
		t.Fatalf("%v", err)
	}
	if _, err := lector(t).Identidad(cert(politicaFNMT, at(oidNumeroSerie, "IDCES-48123456A"), at(oidNombre, "ANTONIO"), at(oidApellidos, "REYES"))); err == nil {
		t.Fatal("letra de control incorrecta")
	}
}
