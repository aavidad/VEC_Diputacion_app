// Package certificado lee la identidad que acredita un certificado de
// persona física ya verificado por la frontera TLS del portal externo: nombre,
// apellidos y DNI o NIE. No verifica cadenas ni revocación; eso lo hace la
// frontera antes. Nunca lee cabeceras, JSON ni configuración.
package certificado

import (
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/aspirantes/domain"
)

var (
	ErrCertificadoNoAdmitido = errors.New("aspirantes: certificado sin identidad de persona física")
	ErrConfiguracion         = errors.New("aspirantes: perfiles de certificado mal configurados")
)

var (
	oidNumeroSerie          = asn1.ObjectIdentifier{2, 5, 4, 5}
	oidApellidos            = asn1.ObjectIdentifier{2, 5, 4, 4}
	oidNombre               = asn1.ObjectIdentifier{2, 5, 4, 42}
	oidNombreComun          = asn1.ObjectIdentifier{2, 5, 4, 3}
	oidIdentificadorEntidad = asn1.ObjectIdentifier{2, 5, 4, 97}
)

// Perfil dice cómo se leen los apellidos de un certificado.
type Perfil string

const (
	// PerfilFNMT: surname trae los dos apellidos juntos.
	PerfilFNMT Perfil = "fnmt_persona_fisica"
	// PerfilDNIe: surname trae solo el primero; los dos van en el nombre
	// común «APELLIDOS, NOMBRE (AUTENTICACIÓN)».
	PerfilDNIe Perfil = "dnie_autenticacion"
)

// PerfilesOficiales son las políticas admitidas en producción: FNMT persona
// física y DNIe de autenticación. Cualquier otra (empleado público,
// representante, otros prestadores) se rechaza.
func PerfilesOficiales() map[string]Perfil {
	return map[string]Perfil{
		"1.3.6.1.4.1.5734.3.10.1": PerfilFNMT,
		"2.16.724.1.2.2.2.3":      PerfilDNIe,
	}
}

// Lector elige el perfil por la política del certificado, de una lista
// cerrada. El arranque de desarrollo puede añadir la política de sus
// certificados sintéticos; nunca se acepta un certificado sin política.
type Lector struct{ perfiles map[string]Perfil }

func NuevoLector(perfiles map[string]Perfil) (*Lector, error) {
	if len(perfiles) == 0 || len(perfiles) > 8 {
		return nil, ErrConfiguracion
	}
	copia := make(map[string]Perfil, len(perfiles))
	for oid, p := range perfiles {
		if (p != PerfilFNMT && p != PerfilDNIe) || !oidValido(oid) {
			return nil, ErrConfiguracion
		}
		copia[oid] = p
	}
	return &Lector{perfiles: copia}, nil
}

func oidValido(s string) bool {
	partes := strings.Split(s, ".")
	if len(partes) < 3 || len(s) > 64 {
		return false
	}
	for _, p := range partes {
		if p == "" || strings.Trim(p, "0123456789") != "" {
			return false
		}
	}
	return true
}

// Datos es lo que acredita el certificado, sin interpretar. Los campos no se
// exportan: ni %v, ni JSON, ni un registro estructurado los muestran.
type Datos struct {
	nombre, apellidos, numero string
	esNIE                     bool
}

func (d Datos) Nombre() string    { return d.nombre }
func (d Datos) Apellidos() string { return d.apellidos }
func (d Datos) Numero() string    { return d.numero }
func (d Datos) EsNIE() bool       { return d.esNIE }

func (Datos) String() string               { return "certificado.Datos{redactados}" }
func (Datos) GoString() string             { return "certificado.Datos{redactados}" }
func (Datos) LogValue() slog.Value         { return slog.StringValue("certificado.Datos{redactados}") }
func (Datos) MarshalJSON() ([]byte, error) { return []byte(`"redactados"`), nil }

// Format redacta con cualquier verbo (%d o %x recorrerían los campos).
func (Datos) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte("certificado.Datos{redactados}")) }

func (l *Lector) perfil(c *x509.Certificate) (Perfil, bool) {
	var elegido Perfil
	for _, oid := range c.PolicyIdentifiers {
		if p, ok := l.perfiles[oid.String()]; ok {
			if elegido != "" && elegido != p {
				return "", false
			}
			elegido = p
		}
	}
	return elegido, elegido != ""
}

// Leer extrae la identidad del sujeto según el perfil de su política. Un
// certificado de representante de una entidad (organizationIdentifier) no
// acredita a una persona aspirante.
func (l *Lector) Leer(c *x509.Certificate) (Datos, error) {
	if l == nil || c == nil {
		return Datos{}, ErrCertificadoNoAdmitido
	}
	perfil, ok := l.perfil(c)
	if !ok {
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
	numero := strings.TrimPrefix(strings.ToUpper(serie), "IDCES-")
	if len(numero) != 9 {
		return Datos{}, ErrCertificadoNoAdmitido
	}
	if perfil == PerfilDNIe {
		// El DNIe exige el nombre común «APELLIDOS, NOMBRE (…)» con los
		// apellidos completos empezando por el surname; si no, se rechaza.
		comun, ok := unico(oidNombreComun)
		antes, despues, hayComa := strings.Cut(comun, ", ")
		nombreComun, _, _ := strings.Cut(despues, " (")
		if !ok || !hayComa || !strings.EqualFold(nombreComun, nombre) ||
			(!strings.EqualFold(antes, apellidos) && !strings.HasPrefix(strings.ToUpper(antes), strings.ToUpper(apellidos)+" ")) {
			return Datos{}, ErrCertificadoNoAdmitido
		}
		apellidos = antes
	}
	// Un NIF que empieza por K, L o M (menores sin DNI, extranjeros sin NIE)
	// no se admite todavía: la ficha solo conoce DNI y NIE.
	return Datos{nombre: nombre, apellidos: apellidos, numero: numero, esNIE: strings.ContainsRune("XYZ", rune(numero[0]))}, nil
}

// Identidad convierte lo que acredita el certificado en la identidad del
// dominio: nombre, apellidos y DNI o NIE español validados (letra incluida).
func (l *Lector) Identidad(c *x509.Certificate) (domain.IdentidadAcreditada, error) {
	d, err := l.Leer(c)
	if err != nil {
		return domain.IdentidadAcreditada{}, err
	}
	tipo := domain.DocumentoDNI
	if d.esNIE {
		tipo = domain.DocumentoNIE
	}
	doc, err := domain.NuevoDocumentoIdentidad(tipo, "ES", d.numero)
	if err != nil {
		return domain.IdentidadAcreditada{}, ErrCertificadoNoAdmitido
	}
	id, err := domain.NuevaIdentidadAcreditada(d.nombre, d.apellidos, doc)
	if err != nil {
		return domain.IdentidadAcreditada{}, ErrCertificadoNoAdmitido
	}
	return id, nil
}
