package interna

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"slices"
	"time"
)

// clonarConfiguracionTLSMutuo separa todas las referencias mutables entre la
// carga, el servidor y el sello. La clave se recodifica como PKCS#8.
func clonarConfiguracionTLSMutuo(origen *tls.Config) *tls.Config {
	clon := origen.Clone()
	clon.ClientCAs = origen.ClientCAs.Clone()
	clon.NextProtos = append([]string(nil), origen.NextProtos...)
	clon.Certificates = make([]tls.Certificate, len(origen.Certificates))
	for indice := range origen.Certificates {
		clon.Certificates[indice] = clonarCertificadoTLS(origen.Certificates[indice])
		clave, err := clonarClavePrivada(origen.Certificates[indice].PrivateKey)
		if err == nil {
			clon.Certificates[indice].PrivateKey = clave
		}
	}
	return clon
}

func clonarCertificadoTLS(origen tls.Certificate) tls.Certificate {
	clon := origen
	clon.Certificate = clonarBytesBidimensionales(origen.Certificate)
	clon.SupportedSignatureAlgorithms = append(
		[]tls.SignatureScheme(nil), origen.SupportedSignatureAlgorithms...,
	)
	clon.OCSPStaple = append([]byte(nil), origen.OCSPStaple...)
	clon.SignedCertificateTimestamps = clonarBytesBidimensionales(
		origen.SignedCertificateTimestamps,
	)
	if len(clon.Certificate) != 0 {
		// validarTLSMutuo ya ha comprobado la cadena; esta nueva instancia evita
		// compartir el puntero mutable Leaf con el proveedor.
		clon.Leaf, _ = x509.ParseCertificate(clon.Certificate[0])
	}
	return clon
}

func clonarBytesBidimensionales(origen [][]byte) [][]byte {
	if origen == nil {
		return nil
	}
	clon := make([][]byte, len(origen))
	for indice := range origen {
		clon[indice] = append([]byte(nil), origen[indice]...)
	}
	return clon
}

func aprobarMaterialTLS(
	configuracion *tls.Config,
	cargado materialTLSCargado,
) (materialTLSAprobado, error) {
	huellaCadena, huellaClave, huellaPrivada, err := resumirCertificadoServidor(configuracion.Certificates[0])
	if err != nil {
		return materialTLSAprobado{}, err
	}
	certificado := clonarCertificadoTLS(configuracion.Certificates[0])
	// La aprobacion conserva material publico y metadatos, no duplica la clave.
	certificado.PrivateKey = nil
	return materialTLSAprobado{
		autoridadesClientes:        configuracion.ClientCAs.Clone(),
		certificadosAutoridades:    clonarCertificadosX509(cargado.autoridadesClientes),
		nombreServidor:             cargado.nombreServidor,
		certificadoServidor:        certificado,
		huellaCadenaServidor:       huellaCadena,
		huellaClavePublicaServidor: huellaClave,
		huellaClavePrivadaServidor: huellaPrivada,
		huellaCertPEM:              cargado.huellaCertPEM,
		huellaClavePEM:             cargado.huellaClavePEM,
		huellaCAPEM:                cargado.huellaCAPEM,
	}, nil
}

func (aprobado materialTLSAprobado) coincide(configuracion *tls.Config) bool {
	if configuracion == nil || configuracion.ClientCAs == nil ||
		aprobado.autoridadesClientes == nil ||
		!configuracion.ClientCAs.Equal(aprobado.autoridadesClientes) ||
		len(configuracion.Certificates) != 1 {
		return false
	}
	actual := configuracion.Certificates[0]
	ahora := time.Now()
	for _, autoridad := range aprobado.certificadosAutoridades {
		if validarAutoridad(autoridad, ahora) != nil {
			return false
		}
	}
	cadena := make([]*x509.Certificate, len(actual.Certificate))
	for indice, der := range actual.Certificate {
		parseado, err := x509.ParseCertificate(der)
		if err != nil {
			return false
		}
		cadena[indice] = parseado
	}
	if validarCadenaServidor(
		Configuracion{NombreServidorTLS: aprobado.nombreServidor}, cadena, actual,
	) != nil {
		return false
	}
	huellaCadena, huellaClave, huellaPrivada, err := resumirCertificadoServidor(actual)
	if err != nil || huellaCadena != aprobado.huellaCadenaServidor ||
		huellaClave != aprobado.huellaClavePublicaServidor ||
		huellaPrivada != aprobado.huellaClavePrivadaServidor ||
		aprobado.huellaCertPEM == ([sha256.Size]byte{}) ||
		aprobado.huellaClavePEM == ([sha256.Size]byte{}) ||
		aprobado.huellaCAPEM == ([sha256.Size]byte{}) {
		return false
	}
	return certificadoTLSEquivalente(actual, aprobado.certificadoServidor)
}

func clonarCertificadosX509(origen []*x509.Certificate) []*x509.Certificate {
	clon := make([]*x509.Certificate, 0, len(origen))
	for _, certificado := range origen {
		if certificado == nil {
			clon = append(clon, nil)
			continue
		}
		parseado, err := x509.ParseCertificate(append([]byte(nil), certificado.Raw...))
		if err != nil {
			clon = append(clon, nil)
			continue
		}
		clon = append(clon, parseado)
	}
	return clon
}

func resumirCertificadoServidor(
	certificado tls.Certificate,
) ([sha256.Size]byte, [sha256.Size]byte, [sha256.Size]byte, error) {
	var vacia [sha256.Size]byte
	if len(certificado.Certificate) == 0 {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	certificadosParseados := make([]*x509.Certificate, len(certificado.Certificate))
	for indice, der := range certificado.Certificate {
		parseado, err := x509.ParseCertificate(der)
		if err != nil {
			return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
		}
		certificadosParseados[indice] = parseado
	}
	if certificado.Leaf != nil && !certificado.Leaf.Equal(certificadosParseados[0]) {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	firmante, valido := certificado.PrivateKey.(crypto.Signer)
	if !valido || firmante == nil {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	clavePublica, err := x509.MarshalPKIXPublicKey(firmante.Public())
	if err != nil || !bytes.Equal(clavePublica, certificadosParseados[0].RawSubjectPublicKeyInfo) {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	clavePrivada, err := x509.MarshalPKCS8PrivateKey(certificado.PrivateKey)
	if err != nil {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	defer limpiarBytesPropios(clavePrivada)

	hashCadena := sha256.New()
	var longitud [8]byte
	for _, der := range certificado.Certificate {
		binary.BigEndian.PutUint64(longitud[:], uint64(len(der)))
		_, _ = hashCadena.Write(longitud[:])
		_, _ = hashCadena.Write(der)
	}
	var huellaCadena [sha256.Size]byte
	copy(huellaCadena[:], hashCadena.Sum(nil))
	huellaClavePrivada := sha256.Sum256(clavePrivada)
	return huellaCadena, sha256.Sum256(clavePublica), huellaClavePrivada, nil
}

func certificadoTLSEquivalente(actual, aprobado tls.Certificate) bool {
	if !bytesBidimensionalesIguales(actual.Certificate, aprobado.Certificate) ||
		!slices.Equal(actual.SupportedSignatureAlgorithms, aprobado.SupportedSignatureAlgorithms) ||
		!bytes.Equal(actual.OCSPStaple, aprobado.OCSPStaple) ||
		!bytesBidimensionalesIguales(
			actual.SignedCertificateTimestamps, aprobado.SignedCertificateTimestamps,
		) || (actual.Leaf == nil) != (aprobado.Leaf == nil) {
		return false
	}
	return actual.Leaf == nil || actual.Leaf.Equal(aprobado.Leaf)
}

func bytesBidimensionalesIguales(izquierda, derecha [][]byte) bool {
	if len(izquierda) != len(derecha) {
		return false
	}
	for indice := range izquierda {
		if !bytes.Equal(izquierda[indice], derecha[indice]) {
			return false
		}
	}
	return true
}
