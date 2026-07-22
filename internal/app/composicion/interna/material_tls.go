package interna

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const maximoBytesMaterialTLS = 1 << 20

type materialTLSCargado struct {
	configuracion       *tls.Config
	autoridadesClientes []*x509.Certificate
	nombreServidor      string
	huellaCertPEM       [sha256.Size]byte
	huellaClavePEM      [sha256.Size]byte
	huellaCAPEM         [sha256.Size]byte
}

// cargarMaterialTLS abre cada referencia mediante una cadena de descriptores
// openat(2). O_NOFOLLOW en todos los componentes evita enlaces simbolicos y la
// cadena de descriptores reduce la ventana TOCTOU ante renombres concurrentes.
func cargarMaterialTLS(cfg Configuracion) (materialTLSCargado, error) {
	if os.Geteuid() == 0 {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	certPEM, err := leerFicheroTLSSeguro(cfg.CertificadoServidorTLS, false)
	if err != nil {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	defer limpiarBytesPropios(certPEM)
	clavePEM, err := leerFicheroTLSSeguro(cfg.ClaveServidorTLS, true)
	if err != nil {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	defer limpiarBytesPropios(clavePEM)
	caPEM, err := leerFicheroTLSSeguro(cfg.AutoridadClientesTLS, false)
	if err != nil {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	defer limpiarBytesPropios(caPEM)
	return materializarTLS(cfg, certPEM, clavePEM, caPEM)
}

// limpiarBytesPropios reduce best-effort la permanencia de serializaciones en
// memoria administrada por Go. No promete borrar copias internas ni la clave
// parseada que crypto/tls necesita mantener viva durante la escucha.
func limpiarBytesPropios(contenido []byte) {
	clear(contenido)
	runtime.KeepAlive(contenido)
}

func materializarTLS(cfg Configuracion, certPEM, clavePEM, caPEM []byte) (materialTLSCargado, error) {
	// El fichero de servidor debe ser fullchain: hoja seguida por uno o mas
	// emisores. El ultimo emisor aportado actua como ancla explicita y no tiene
	// que ser una raiz autofirmada; asi se admite el fullchain operativo usual.
	cadena, err := certificadosPEMEstrictos(certPEM)
	if err != nil || len(cadena) < 2 {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	if err := validarClavePEMEstricta(clavePEM); err != nil {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	certificado, err := tls.X509KeyPair(certPEM, clavePEM)
	if err != nil {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	certificado.PrivateKey, err = clonarClavePrivada(certificado.PrivateKey)
	if err != nil {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	certificado.Leaf = cadena[0]
	if err := validarCadenaServidor(cfg, cadena, certificado); err != nil {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}

	autoridades, err := certificadosPEMEstrictos(caPEM)
	if err != nil || len(autoridades) == 0 {
		return materialTLSCargado{}, ErrTLSMutuoNoVerificado
	}
	pool := x509.NewCertPool()
	ahora := time.Now()
	for _, autoridad := range autoridades {
		if err := validarAutoridad(autoridad, ahora); err != nil {
			return materialTLSCargado{}, ErrTLSMutuoNoVerificado
		}
		pool.AddCert(autoridad)
	}

	configuracion := &tls.Config{
		Certificates:           []tls.Certificate{clonarCertificadoTLS(certificado)},
		NextProtos:             []string{protocoloALPNHTTPUno},
		ClientAuth:             tls.RequireAndVerifyClientCert,
		ClientCAs:              pool,
		SessionTicketsDisabled: true,
		MinVersion:             tls.VersionTLS13,
		MaxVersion:             tls.VersionTLS13,
	}
	if err := validarTLSMutuo(configuracion); err != nil {
		return materialTLSCargado{}, err
	}
	return materialTLSCargado{
		configuracion:       configuracion,
		autoridadesClientes: autoridades,
		nombreServidor:      cfg.NombreServidorTLS,
		huellaCertPEM:       sha256.Sum256(certPEM),
		huellaClavePEM:      sha256.Sum256(clavePEM),
		huellaCAPEM:         sha256.Sum256(caPEM),
	}, nil
}

func leerFicheroTLSSeguro(ruta string, privado bool) ([]byte, error) {
	if os.Geteuid() == 0 {
		return nil, ErrTLSMutuoNoVerificado
	}
	componentes := strings.Split(strings.TrimPrefix(filepath.Clean(ruta), string(filepath.Separator)), string(filepath.Separator))
	directorio, err := syscall.Open(string(filepath.Separator), syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	if err := validarDirectorioTLS(directorio); err != nil {
		_ = syscall.Close(directorio)
		return nil, err
	}
	for _, componente := range componentes[:len(componentes)-1] {
		siguiente, err := syscall.Openat(directorio, componente, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if err != nil {
			_ = syscall.Close(directorio)
			return nil, err
		}
		if err := validarDirectorioTLS(siguiente); err != nil {
			_ = syscall.Close(siguiente)
			_ = syscall.Close(directorio)
			return nil, err
		}
		if err := syscall.Close(directorio); err != nil {
			syscall.Close(siguiente)
			return nil, err
		}
		directorio = siguiente
	}
	contenido, err := abrirFicheroTLSFinal(
		directorio, componentes[len(componentes)-1], privado,
	)
	_ = syscall.Close(directorio)
	return contenido, err
}

func abrirFicheroTLSFinal(directorio int, nombre string, privado bool) ([]byte, error) {
	fd, err := syscall.Openat(
		directorio,
		nombre,
		syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOCTTY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC,
		0,
	)
	if err != nil {
		return nil, err
	}
	fichero := os.NewFile(uintptr(fd), "material-tls")
	if fichero == nil {
		syscall.Close(fd)
		return nil, ErrTLSMutuoNoVerificado
	}
	defer fichero.Close()
	informacion, err := fichero.Stat()
	if err != nil || !informacion.Mode().IsRegular() || informacion.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return nil, ErrTLSMutuoNoVerificado
	}
	estadistica, valida := informacion.Sys().(*syscall.Stat_t)
	if !valida || estadistica.Uid != 0 {
		return nil, ErrTLSMutuoNoVerificado
	}
	permisos := informacion.Mode().Perm()
	if permisos&0o022 != 0 || permisos&0o111 != 0 || (privado && permisos&0o007 != 0) ||
		(privado && permisos&0o110 != 0) ||
		(privado && permisos&0o040 != 0 && !grupoProceso(estadistica.Gid)) {
		return nil, ErrTLSMutuoNoVerificado
	}
	contenido, err := io.ReadAll(io.LimitReader(fichero, maximoBytesMaterialTLS+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesMaterialTLS {
		return nil, ErrTLSMutuoNoVerificado
	}
	return contenido, nil
}

func validarDirectorioTLS(fd int) error {
	var informacion syscall.Stat_t
	if err := syscall.Fstat(fd, &informacion); err != nil ||
		informacion.Mode&syscall.S_IFMT != syscall.S_IFDIR || informacion.Uid != 0 ||
		informacion.Mode&0o022 != 0 {
		return ErrTLSMutuoNoVerificado
	}
	return nil
}

func grupoProceso(gid uint32) bool {
	if int(gid) == os.Getegid() {
		return true
	}
	grupos, err := os.Getgroups()
	if err != nil {
		return false
	}
	for _, grupo := range grupos {
		if grupo == int(gid) {
			return true
		}
	}
	return false
}

func certificadosPEMEstrictos(contenido []byte) ([]*x509.Certificate, error) {
	var certificados []*x509.Certificate
	vistos := make(map[[sha256.Size]byte]struct{})
	resto := contenido
	for len(bytes.TrimSpace(resto)) != 0 {
		resto = bytes.TrimSpace(resto)
		if !bytes.HasPrefix(resto, []byte("-----BEGIN CERTIFICATE-----")) {
			return nil, ErrTLSMutuoNoVerificado
		}
		bloque, siguiente := pem.Decode(resto)
		if bloque == nil || bloque.Type != "CERTIFICATE" || len(bloque.Headers) != 0 {
			return nil, ErrTLSMutuoNoVerificado
		}
		certificado, err := x509.ParseCertificate(bloque.Bytes)
		if err != nil {
			return nil, ErrTLSMutuoNoVerificado
		}
		huella := sha256.Sum256(certificado.Raw)
		if _, duplicado := vistos[huella]; duplicado {
			return nil, ErrTLSMutuoNoVerificado
		}
		vistos[huella] = struct{}{}
		certificados = append(certificados, certificado)
		resto = siguiente
	}
	if len(certificados) == 0 {
		return nil, ErrTLSMutuoNoVerificado
	}
	return certificados, nil
}

func validarClavePEMEstricta(contenido []byte) error {
	contenido = bytes.TrimSpace(contenido)
	if !bytes.HasPrefix(contenido, []byte("-----BEGIN ")) {
		return ErrTLSMutuoNoVerificado
	}
	bloque, resto := pem.Decode(contenido)
	if bloque == nil {
		return ErrTLSMutuoNoVerificado
	}
	defer limpiarBytesPropios(bloque.Bytes)
	if len(bloque.Headers) != 0 || !strings.HasSuffix(bloque.Type, "PRIVATE KEY") ||
		len(bytes.TrimSpace(resto)) != 0 {
		return ErrTLSMutuoNoVerificado
	}
	return nil
}

func validarCadenaServidor(cfg Configuracion, cadena []*x509.Certificate, certificado tls.Certificate) error {
	ahora := time.Now()
	hoja := cadena[0]
	if hoja.IsCA || ahora.Before(hoja.NotBefore) || ahora.After(hoja.NotAfter) ||
		(len(hoja.DNSNames) == 0 && len(hoja.IPAddresses) == 0) ||
		!contieneUsoExtendido(hoja.ExtKeyUsage, x509.ExtKeyUsageServerAuth) {
		return ErrTLSMutuoNoVerificado
	}
	if hoja.VerifyHostname(cfg.NombreServidorTLS) != nil {
		return ErrTLSMutuoNoVerificado
	}
	for indice := 1; indice < len(cadena); indice++ {
		if err := validarAutoridad(cadena[indice], ahora); err != nil ||
			cadena[indice-1].CheckSignatureFrom(cadena[indice]) != nil {
			return ErrTLSMutuoNoVerificado
		}
	}
	raiz := cadena[len(cadena)-1]
	raices := x509.NewCertPool()
	raices.AddCert(raiz)
	intermedias := x509.NewCertPool()
	for _, intermedia := range cadena[1 : len(cadena)-1] {
		intermedias.AddCert(intermedia)
	}
	cadenas, err := hoja.Verify(x509.VerifyOptions{
		DNSName:       cfg.NombreServidorTLS,
		Roots:         raices,
		Intermediates: intermedias,
		CurrentTime:   ahora,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil || len(cadenas) == 0 || len(cadenas[0]) != len(cadena) {
		return ErrTLSMutuoNoVerificado
	}
	_, _, _, err = resumirCertificadoServidor(certificado)
	return err
}

func validarAutoridad(certificado *x509.Certificate, ahora time.Time) error {
	if certificado == nil || !certificado.IsCA || !certificado.BasicConstraintsValid ||
		certificado.KeyUsage&x509.KeyUsageCertSign == 0 || ahora.Before(certificado.NotBefore) ||
		ahora.After(certificado.NotAfter) {
		return ErrTLSMutuoNoVerificado
	}
	return nil
}

func contieneUsoExtendido(usos []x509.ExtKeyUsage, esperado x509.ExtKeyUsage) bool {
	for _, uso := range usos {
		if uso == esperado {
			return true
		}
	}
	return false
}

func clonarClavePrivada(clave crypto.PrivateKey) (crypto.PrivateKey, error) {
	der, err := x509.MarshalPKCS8PrivateKey(clave)
	if err != nil {
		return nil, err
	}
	defer limpiarBytesPropios(der)
	return x509.ParsePKCS8PrivateKey(der)
}
