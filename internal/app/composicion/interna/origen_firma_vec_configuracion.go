package interna

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	nombreArchivoOrigenFirmaVec = "origen_firma_vec"
	limiteArchivoOrigenFirmaVec = 512
	prefijoOrigenFirmaVec       = "origen_firma_vec="
)

var ErrOrigenFirmaVecNoDisponible = errors.New("composicion interna: origen firma vec no disponible")

// cargarOrigenFirmaVecPrivado lee el origen aprobado en el directorio de
// material interno. El nombre TLS procede de la configuración del servidor,
// cuya hoja ya se comprueba contra ese nombre antes de abrir el listener.
// Ninguna cabecera HTTP, dirección remota ni valor ambiental fija este origen.
func cargarOrigenFirmaVecPrivado(directorio, nombreServidorTLS string) (string, error) {
	fallo := ErrOrigenFirmaVecNoDisponible
	if !filepath.IsAbs(directorio) || filepath.Clean(directorio) != directorio ||
		strings.TrimSpace(directorio) != directorio || !nombreHostFirmaVecValido(nombreServidorTLS) {
		return "", fallo
	}
	infoDir, err := os.Lstat(directorio)
	if err != nil || !infoDir.IsDir() || infoDir.Mode().Perm() != 0o700 {
		return "", fallo
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return "", fallo
	}
	defer raiz.Close()
	actualDir, err := raiz.Stat(".")
	if err != nil || !os.SameFile(infoDir, actualDir) || actualDir.Mode().Perm() != 0o700 {
		return "", fallo
	}
	info, err := raiz.Lstat(nombreArchivoOrigenFirmaVec)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 ||
		info.Size() <= int64(len(prefijoOrigenFirmaVec)) || info.Size() > limiteArchivoOrigenFirmaVec {
		return "", fallo
	}
	archivo, err := raiz.Open(nombreArchivoOrigenFirmaVec)
	if err != nil {
		return "", fallo
	}
	defer archivo.Close()
	actual, err := archivo.Stat()
	if err != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0o600 {
		return "", fallo
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, limiteArchivoOrigenFirmaVec+1))
	if err != nil || int64(len(contenido)) != actual.Size() || len(contenido) > limiteArchivoOrigenFirmaVec {
		clear(contenido)
		return "", fallo
	}
	defer clear(contenido)
	linea := bytes.TrimSuffix(contenido, []byte{'\n'})
	if !bytes.HasPrefix(linea, []byte(prefijoOrigenFirmaVec)) || bytes.IndexByte(linea, '\n') >= 0 ||
		bytes.IndexByte(linea, '\r') >= 0 {
		return "", fallo
	}
	origen := string(linea[len(prefijoOrigenFirmaVec):])
	if !origenFirmaVecCanonico(origen, nombreServidorTLS) {
		return "", fallo
	}
	return origen, nil
}

func origenFirmaVecCanonico(origen, nombreServidorTLS string) bool {
	u, err := url.Parse(origen)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery ||
		u.Fragment != "" || u.RawFragment != "" || u.Opaque != "" || u.String() != origen {
		return false
	}
	host := u.Hostname()
	if !nombreHostFirmaVecValido(host) || host != nombreServidorTLS {
		return false
	}
	autoridad := host
	if strings.Contains(host, ":") {
		autoridad = "[" + host + "]"
	}
	puerto := u.Port()
	if puerto == "" {
		return u.Host == autoridad
	}
	numero, err := strconv.Atoi(puerto)
	return err == nil && numero > 0 && numero <= 65535 && numero != 443 &&
		strconv.Itoa(numero) == puerto && u.Host == autoridad+":"+puerto
}

func nombreHostFirmaVecValido(nombre string) bool {
	if ip := net.ParseIP(nombre); ip != nil {
		return ip.String() == nombre
	}
	return nombreDNSFirmaVecValido(nombre)
}

func nombreDNSFirmaVecValido(nombre string) bool {
	if nombre == "" || len(nombre) > 253 || nombre != strings.ToLower(nombre) || net.ParseIP(nombre) != nil {
		return false
	}
	tieneLetra := false
	for _, etiqueta := range strings.Split(nombre, ".") {
		if len(etiqueta) == 0 || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for _, c := range etiqueta {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
			if c >= 'a' && c <= 'z' {
				tieneLetra = true
			}
		}
	}
	return tieneLetra
}
