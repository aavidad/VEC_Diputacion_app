// Package certificadopersonal analiza perfiles de certificados personales.
// No verifica confianza, posesión de clave, revocación ni permisos.
package certificadopersonal

import (
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrPerfilPersonalInvalido = errors.New("certificado personal: perfil no admitido")
	ErrPerfilNoSerializable   = errors.New("certificado personal: resultado no serializable")
)

const (
	politicaFNMTRSA = "1.3.6.1.4.1.5734.3.10.1"
	politicaFNMTG2  = "1.3.6.1.4.1.5734.3.20.1.0"
	politicaDNIe    = "2.16.724.1.2.2.2.4"
	politicaQCPn    = "0.4.0.194112.1.0"
	oidPoliticas    = "2.5.29.32"
	oidSAN          = "2.5.29.17"
	oidEKU          = "2.5.29.37"
	oidDocumento    = "2.5.4.5"
	oidPais         = "2.5.4.6"
	oidNIFFNMT      = "1.3.6.1.4.1.5734.1.4"
	perfilRedactado = "[PERFIL PERSONAL CONFIDENCIAL]"
)

// PerfilPersonal retiene sólo el DNI necesario para cotejar una autorización
// privada. Su método describe el perfil analizado, no una autenticación válida.
// El valor cero no acredita método ni coincide con ningún documento.
type PerfilPersonal struct {
	documento [9]byte
	metodo    domain.AuthMethod
}

func (p PerfilPersonal) MetodoDelPerfil() domain.AuthMethod { return p.metodo }

// CoincideDocumentoAutorizado exige un DNI canónico de longitud fija y compara
// en tiempo constante. No normaliza, expone ni conserva el argumento privado.
func (p PerfilPersonal) CoincideDocumentoAutorizado(documento []byte) bool {
	if p.metodo == "" || !dniCanonico(documento) {
		return false
	}
	return subtle.ConstantTimeCompare(p.documento[:], documento) == 1
}

func (PerfilPersonal) String() string                 { return perfilRedactado }
func (PerfilPersonal) GoString() string               { return perfilRedactado }
func (PerfilPersonal) Format(s fmt.State, _ rune)     { _, _ = s.Write([]byte(perfilRedactado)) }
func (PerfilPersonal) LogValue() slog.Value           { return slog.StringValue(perfilRedactado) }
func (PerfilPersonal) MarshalJSON() ([]byte, error)   { return nil, ErrPerfilNoSerializable }
func (PerfilPersonal) MarshalText() ([]byte, error)   { return nil, ErrPerfilNoSerializable }
func (PerfilPersonal) MarshalBinary() ([]byte, error) { return nil, ErrPerfilNoSerializable }
func (PerfilPersonal) GobEncode() ([]byte, error)     { return nil, ErrPerfilNoSerializable }
func (PerfilPersonal) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrPerfilNoSerializable
}
func (*PerfilPersonal) UnmarshalJSON([]byte) error   { return ErrPerfilNoSerializable }
func (*PerfilPersonal) UnmarshalText([]byte) error   { return ErrPerfilNoSerializable }
func (*PerfilPersonal) UnmarshalBinary([]byte) error { return ErrPerfilNoSerializable }
func (*PerfilPersonal) GobDecode([]byte) error       { return ErrPerfilNoSerializable }
func (*PerfilPersonal) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrPerfilNoSerializable
}

// ExtraerPerfilPersonal analiza el DER del certificado. El llamador debe haber
// verificado ese mismo certificado en el canal TLS y bajo su confianza propia.
// Esta función NO llama a Verify, no comprueba fechas/revocación y no concede
// permisos. Un certificado sintético con el mismo perfil también puede analizarse.
//
// Fuentes: FNMT perfiles v2.6, pp.6-9 y14-18:
// https://www.sede.fnmt.gob.es/documents/10445900/10575386/perfiles_certificados_ac_usuarios.pdf
// DNIe DPC v3.4, pp.129-132 y135-136:
// https://www.dnielectronico.es/PDFs/Politicas_de_certificacion_v3.4.pdf
func ExtraerPerfilPersonal(cert *x509.Certificate) (PerfilPersonal, error) {
	var cero PerfilPersonal
	if cert == nil || len(cert.Raw) == 0 || len(cert.Raw) > 64<<10 {
		return cero, ErrPerfilPersonalInvalido
	}
	// No se aceptan campos de x509.Certificate alterados tras analizar el DER.
	c, err := x509.ParseCertificate(cert.Raw)
	if err != nil || c.IsCA || !c.BasicConstraintsValid || len(c.UnhandledCriticalExtensions) != 0 {
		return cero, ErrPerfilPersonalInvalido
	}
	extensiones := make(map[string]pkix.Extension, len(c.Extensions))
	for _, extension := range c.Extensions {
		oid := extension.Id.String()
		if _, repetida := extensiones[oid]; repetida {
			return cero, ErrPerfilPersonalInvalido
		}
		extensiones[oid] = extension
	}
	perfil, ok := leerPolitica(extensiones[oidPoliticas].Value)
	if !ok || !usosAdmitidos(c, extensiones, perfil) {
		return cero, ErrPerfilPersonalInvalido
	}
	sujeto, ok := leerNombre(c.RawSubject)
	if !ok || sujeto[oidPais] != "ES" {
		return cero, ErrPerfilPersonalInvalido
	}
	documento := sujeto[oidDocumento]
	metodo := domain.AuthMethodDNIe
	if perfil != politicaDNIe {
		if !strings.HasPrefix(documento, "IDCES-") {
			return cero, ErrPerfilPersonalInvalido
		}
		documento = strings.TrimPrefix(documento, "IDCES-")
		metodo = domain.AuthMethodCertificate
	}
	if !dniCanonico([]byte(documento)) || !sanConcordante(extensiones, perfil, documento) {
		return cero, ErrPerfilPersonalInvalido
	}
	resultado := PerfilPersonal{metodo: metodo}
	copy(resultado.documento[:], documento)
	return resultado, nil
}

func leerPolitica(der []byte) (string, bool) {
	var politicas []struct {
		ID            asn1.ObjectIdentifier
		Calificadores []asn1.RawValue `asn1:"optional"`
	}
	resto, err := asn1.Unmarshal(der, &politicas)
	if err != nil || len(resto) != 0 || len(politicas) == 0 || len(politicas) > 2 {
		return "", false
	}
	perfil := ""
	vistas := make(map[string]bool, len(politicas))
	for _, politica := range politicas {
		oid := politica.ID.String()
		if vistas[oid] {
			return "", false
		}
		vistas[oid] = true
		switch oid {
		case politicaFNMTRSA, politicaFNMTG2, politicaDNIe:
			if perfil != "" {
				return "", false
			}
			perfil = oid
		case politicaQCPn:
		default:
			return "", false
		}
	}
	return perfil, perfil != "" && !(perfil == politicaDNIe && vistas[politicaQCPn])
}

func usosAdmitidos(cert *x509.Certificate, extensiones map[string]pkix.Extension, perfil string) bool {
	uso := x509.KeyUsageDigitalSignature
	if perfil != politicaDNIe {
		uso |= x509.KeyUsageContentCommitment
	}
	if perfil == politicaFNMTRSA {
		uso |= x509.KeyUsageKeyEncipherment
	}
	if cert.KeyUsage != uso {
		return false
	}
	eku, existe := extensiones[oidEKU]
	if perfil == politicaDNIe {
		return !existe
	}
	var propositos []asn1.ObjectIdentifier
	resto, err := asn1.Unmarshal(eku.Value, &propositos)
	if !existe || err != nil || len(resto) != 0 || len(propositos) == 0 || len(propositos) > 3 {
		return false
	}
	vistos := make(map[string]bool, len(propositos))
	for _, proposito := range propositos {
		oid := proposito.String()
		if vistos[oid] {
			return false
		}
		switch oid {
		case "1.3.6.1.5.5.7.3.2", "1.3.6.1.4.1.311.10.3.12", "1.2.840.113583.1.1.5":
		default:
			return false
		}
		vistos[oid] = true
	}
	return vistos["1.3.6.1.5.5.7.3.2"]
}

// Recorre todos los RDN: no usa el valor único que pkix.Name proyecta cuando
// el DER contiene atributos repetidos. Los campos no son normalizados.
func leerNombre(der []byte) (map[string]string, bool) {
	var nombre pkix.RDNSequence
	resto, err := asn1.Unmarshal(der, &nombre)
	if err != nil || len(resto) != 0 || len(nombre) == 0 || len(nombre) > 32 {
		return nil, false
	}
	valores := make(map[string]string)
	for _, conjunto := range nombre {
		if len(conjunto) == 0 || len(conjunto) > 16 {
			return nil, false
		}
		for _, atributo := range conjunto {
			oid := atributo.Type.String()
			valor, cadena := atributo.Value.(string)
			if _, repetido := valores[oid]; repetido || !cadena || valor == "" {
				return nil, false
			}
			valores[oid] = valor
		}
	}
	return valores, true
}

func sanConcordante(extensiones map[string]pkix.Extension, perfil, documento string) bool {
	extension, existe := extensiones[oidSAN]
	if !existe {
		return perfil == politicaDNIe
	}
	var secuencia asn1.RawValue
	resto, err := asn1.Unmarshal(extension.Value, &secuencia)
	if err != nil || len(resto) != 0 || secuencia.Class != asn1.ClassUniversal || secuencia.Tag != asn1.TagSequence || !secuencia.IsCompound || len(secuencia.Bytes) == 0 {
		return false
	}
	pendiente := secuencia.Bytes
	directorios, correos := 0, 0
	for len(pendiente) > 0 {
		var nombre asn1.RawValue
		pendiente, err = asn1.Unmarshal(pendiente, &nombre)
		if err != nil || nombre.Class != asn1.ClassContextSpecific {
			return false
		}
		switch nombre.Tag {
		case 1: // rfc822Name no participa en la identidad ni en el permiso.
			correos++
			if nombre.IsCompound || len(nombre.Bytes) == 0 || correos > 1 {
				return false
			}
		case 4: // directoryName explícito contiene el Name DER completo.
			directorios++
			atributos, ok := leerNombre(nombre.Bytes)
			if !nombre.IsCompound || !ok || perfil == politicaDNIe || directorios > 1 || atributos[oidNIFFNMT] != documento {
				return false
			}
		default:
			return false
		}
	}
	return perfil == politicaDNIe || directorios == 1
}

func dniCanonico(documento []byte) bool {
	if len(documento) != 9 {
		return false
	}
	numero := 0
	for _, digito := range documento[:8] {
		if digito < '0' || digito > '9' {
			return false
		}
		numero = numero*10 + int(digito-'0')
	}
	// Tabla oficial del resto módulo 23; conserva los ceros iniciales.
	// https://www.interior.gob.es/opencms/es/servicios-al-ciudadano/tramites-y-gestiones/dni/calculo-del-digito-de-control-del-nif-nie/
	return documento[8] == "TRWAGMYFPDXBNJZSQVHLCKE"[numero%23]
}
