// Package certificado lee la identidad que acredita un certificado de
// persona física ya verificado por la frontera TLS del portal externo: nombre,
// apellidos y DNI o NIE. No verifica cadenas ni revocación; eso lo hace la
// frontera antes. Nunca lee cabeceras, JSON ni configuración.
package certificado

import (
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/aspirantes/domain"
)

var ErrCertificadoNoAdmitido = errors.New("aspirantes: certificado sin identidad de persona física")

var (
	oidNumeroSerie          = asn1.ObjectIdentifier{2, 5, 4, 5}
	oidApellidos            = asn1.ObjectIdentifier{2, 5, 4, 4}
	oidNombre               = asn1.ObjectIdentifier{2, 5, 4, 42}
	oidNombreComun          = asn1.ObjectIdentifier{2, 5, 4, 3}
	oidIdentificadorEntidad = asn1.ObjectIdentifier{2, 5, 4, 97}
)

// Datos es lo que acredita el certificado, sin interpretar.
type Datos struct {
	Nombre    string
	Apellidos string
	// Numero es el DNI o NIE tal como figura, sin el prefijo IDCES-.
	Numero string
	// EsNIE distingue X/Y/Z inicial; el país es siempre España.
	EsNIE bool
}

func (Datos) String() string   { return "certificado.Datos{redactados}" }
func (Datos) GoString() string { return "certificado.Datos{redactados}" }

// Leer extrae la identidad del sujeto según los perfiles de persona física
// de la FNMT (serialNumber «IDCES-12345678Z», givenName y surname con los
// dos apellidos juntos) y del DNIe (serialNumber sin prefijo y, si el
// surname solo trae el primer apellido, el nombre común «APELLIDOS, NOMBRE
// (AUTENTICACIÓN)»). Un certificado de representante de una entidad
// (organizationIdentifier) no acredita a una persona aspirante.
func Leer(c *x509.Certificate) (Datos, error) {
	if c == nil {
		return Datos{}, ErrCertificadoNoAdmitido
	}
	valores := map[string][]string{}
	for _, atributo := range c.Subject.Names {
		texto, ok := atributo.Value.(string)
		if !ok {
			return Datos{}, ErrCertificadoNoAdmitido
		}
		clave := atributo.Type.String()
		valores[clave] = append(valores[clave], texto)
	}
	unico := func(oid asn1.ObjectIdentifier) (string, bool) {
		v := valores[oid.String()]
		if len(v) != 1 || !utf8.ValidString(v[0]) || len(v[0]) > 200 {
			return "", false
		}
		return strings.Join(strings.Fields(v[0]), " "), true
	}
	if len(valores[oidIdentificadorEntidad.String()]) != 0 {
		return Datos{}, ErrCertificadoNoAdmitido
	}
	serie, ok1 := unico(oidNumeroSerie)
	nombre, ok2 := unico(oidNombre)
	apellidos, ok3 := unico(oidApellidos)
	if !ok1 || !ok2 || !ok3 || nombre == "" || apellidos == "" {
		return Datos{}, ErrCertificadoNoAdmitido
	}
	numero := strings.ToUpper(strings.TrimPrefix(strings.ToUpper(serie), "IDCES-"))
	if len(numero) != 9 {
		return Datos{}, ErrCertificadoNoAdmitido
	}
	// DNIe: si el nombre común es «APELLIDOS, NOMBRE (…)» y empieza por el
	// surname, sus apellidos son los completos.
	if comun, ok := unico(oidNombreComun); ok {
		if antes, despues, hayComa := strings.Cut(comun, ", "); hayComa {
			nombreComun, _, _ := strings.Cut(despues, " (")
			if strings.EqualFold(nombreComun, nombre) && len(antes) > len(apellidos) &&
				strings.HasPrefix(strings.ToUpper(antes), strings.ToUpper(apellidos)+" ") {
				apellidos = antes
			}
		}
	}
	return Datos{Nombre: nombre, Apellidos: apellidos, Numero: numero, EsNIE: strings.ContainsRune("XYZ", rune(numero[0]))}, nil
}

// Identidad convierte lo que acredita el certificado en la identidad del
// dominio: nombre, apellidos y DNI o NIE español validados (letra incluida).
func Identidad(c *x509.Certificate) (domain.IdentidadAcreditada, error) {
	d, err := Leer(c)
	if err != nil {
		return domain.IdentidadAcreditada{}, err
	}
	tipo := domain.DocumentoDNI
	if d.EsNIE {
		tipo = domain.DocumentoNIE
	}
	doc, err := domain.NuevoDocumentoIdentidad(tipo, "ES", d.Numero)
	if err != nil {
		return domain.IdentidadAcreditada{}, ErrCertificadoNoAdmitido
	}
	id, err := domain.NuevaIdentidadAcreditada(d.Nombre, d.Apellidos, doc)
	if err != nil {
		return domain.IdentidadAcreditada{}, ErrCertificadoNoAdmitido
	}
	return id, nil
}
