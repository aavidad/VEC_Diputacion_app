// Package identidadcertificado conserva la única lectura revocable del
// certificado personal para las composiciones interna y nominal.
package identidadcertificado

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrNoDisponible = errors.New("composicion: certificado personal no disponible")

const ProteccionClavePKCS11 = "pkcs11-pin-no-exportable"
const limiteRegistro = 64 << 10

type Registro struct {
	ruta, rutaCRL string
}

type CertificadoRegistrado struct {
	HuellaSHA256       string `json:"huella_sha256"`
	SujetoID           string `json:"sujeto_id"`
	CuentaID           string `json:"cuenta_id"`
	ProteccionClaveRef string `json:"proteccion_clave_ref"`
	Activo             bool   `json:"activo"`
}

type DocumentoRegistro struct {
	Version      int                     `json:"version"`
	Certificados []CertificadoRegistrado `json:"certificados"`
}

func NuevoRegistro(ruta string) (*Registro, error) {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return nil, ErrNoDisponible
	}
	if err := directoriosReales(ruta); err != nil {
		return nil, err
	}
	directorio, err := os.Lstat(filepath.Dir(ruta))
	if err != nil || !directorio.IsDir() || directorio.Mode().Perm() != 0700 {
		return nil, ErrNoDisponible
	}
	r := &Registro{ruta: ruta, rutaCRL: filepath.Join(filepath.Dir(ruta), "clientes.crl")}
	if _, err := r.leer(); err != nil {
		return nil, err
	}
	if _, err := r.LeerCRLActual(); err != nil {
		return nil, err
	}
	return r, nil
}

// LeerArchivoPrivado exige un fichero regular 0600 sin enlace simbólico y
// comprueba que la preimagen de Lstat sigue siendo el mismo fichero abierto.
func LeerArchivoPrivado(ruta string, limite int64) ([]byte, error) {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return nil, ErrNoDisponible
	}
	if err := directoriosReales(ruta); err != nil {
		return nil, err
	}
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 ||
		info.Size() <= 0 || info.Size() > limite {
		return nil, ErrNoDisponible
	}
	archivo, err := os.Open(ruta)
	if err != nil {
		return nil, ErrNoDisponible
	}
	defer archivo.Close()
	actual, err := archivo.Stat()
	if err != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() ||
		actual.Mode().Perm() != 0600 {
		return nil, ErrNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, limite+1))
	if err != nil || len(contenido) == 0 || int64(len(contenido)) > limite {
		clear(contenido)
		return nil, ErrNoDisponible
	}
	return contenido, nil
}

func directoriosReales(ruta string) error {
	for directorio := filepath.Dir(ruta); directorio != "/"; directorio = filepath.Dir(directorio) {
		info, err := os.Lstat(directorio)
		if err != nil {
			// Se conserva la causa del sistema sin incluir la ruta privada.
			var fallo *os.PathError
			if errors.As(err, &fallo) {
				return errors.Join(ErrNoDisponible, fallo.Err)
			}
			return errors.Join(ErrNoDisponible, err)
		}
		if !info.IsDir() {
			return ErrNoDisponible
		}
	}
	return nil
}

// RechazarClavesDuplicadas comprueba todos los objetos JSON, incluidos los de
// cada entrada del registro; el decodificador estricto comprueba después forma.
func RechazarClavesDuplicadas(contenido []byte) error {
	lector := json.NewDecoder(bytes.NewReader(contenido))
	var recorrer func() error
	recorrer = func() error {
		token, err := lector.Token()
		if err != nil {
			return err
		}
		apertura, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch apertura {
		case '{':
			vistas := map[string]bool{}
			for lector.More() {
				clave, err := lector.Token()
				if err != nil {
					return err
				}
				nombre, ok := clave.(string)
				if !ok || vistas[nombre] {
					return ErrNoDisponible
				}
				vistas[nombre] = true
				if err := recorrer(); err != nil {
					return err
				}
			}
		case '[':
			for lector.More() {
				if err := recorrer(); err != nil {
					return err
				}
			}
		default:
			return ErrNoDisponible
		}
		_, err = lector.Token()
		return err
	}
	if err := recorrer(); err != nil {
		return ErrNoDisponible
	}
	if _, err := lector.Token(); err != io.EOF {
		return ErrNoDisponible
	}
	return nil
}

func (r *Registro) LeerCRLActual() (*x509.RevocationList, error) {
	if r == nil || r.rutaCRL == "" {
		return nil, ErrNoDisponible
	}
	contenido, err := LeerArchivoPrivado(r.rutaCRL, limiteRegistro)
	if err != nil {
		return nil, err
	}
	defer clear(contenido)
	bloque, resto := pem.Decode(contenido)
	if bloque == nil || bloque.Type != "X509 CRL" || len(bloque.Headers) != 0 ||
		len(bytes.TrimSpace(resto)) != 0 {
		return nil, ErrNoDisponible
	}
	lista, err := x509.ParseRevocationList(bloque.Bytes)
	if err != nil || lista == nil || lista.ThisUpdate.IsZero() || lista.NextUpdate.IsZero() {
		return nil, ErrNoDisponible
	}
	return lista, nil
}

func (r *Registro) leer() (map[string]CertificadoRegistrado, error) {
	if r == nil || !filepath.IsAbs(r.ruta) {
		return nil, ErrNoDisponible
	}
	contenido, err := LeerArchivoPrivado(r.ruta, limiteRegistro)
	if err != nil {
		return nil, err
	}
	defer clear(contenido)
	if RechazarClavesDuplicadas(contenido) != nil {
		return nil, ErrNoDisponible
	}
	var documento DocumentoRegistro
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if decodificador.Decode(&documento) != nil || decodificador.Decode(new(any)) != io.EOF ||
		documento.Version != 1 || len(documento.Certificados) == 0 || len(documento.Certificados) > 64 {
		return nil, ErrNoDisponible
	}
	porHuella := make(map[string]CertificadoRegistrado, len(documento.Certificados))
	cuentas := make(map[string]bool, len(documento.Certificados))
	for _, c := range documento.Certificados {
		if !HuellaValida(c.HuellaSHA256) || !IdentificadorValido(c.SujetoID, "per_") ||
			!IdentificadorValido(c.CuentaID, "cta_") ||
			c.ProteccionClaveRef != ProteccionClavePKCS11 ||
			porHuella[c.HuellaSHA256].HuellaSHA256 != "" || cuentas[c.CuentaID] {
			return nil, ErrNoDisponible
		}
		porHuella[c.HuellaSHA256] = c
		cuentas[c.CuentaID] = true
	}
	return porHuella, nil
}

func (r *Registro) Resolver(ctx context.Context, huella string) (CertificadoRegistrado, error) {
	if ctx == nil || ctx.Err() != nil || !HuellaValida(huella) {
		return CertificadoRegistrado{}, ErrNoDisponible
	}
	todos, err := r.leer()
	if err != nil {
		return CertificadoRegistrado{}, err
	}
	c, existe := todos[huella]
	if !existe || !c.Activo || ctx.Err() != nil {
		return CertificadoRegistrado{}, ErrNoDisponible
	}
	return c, nil
}

func HuellaValida(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != len("sha256:")+sha256.Size*2 {
		return false
	}
	for _, c := range v[len("sha256:"):] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func IdentificadorValido(v, prefijo string) bool {
	if !strings.HasPrefix(v, prefijo) || len(v) < len(prefijo)+20 || len(v) > 128 {
		return false
	}
	for _, c := range v[len(prefijo):] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || c == '_') {
			return false
		}
	}
	return true
}

type AcreditacionActual struct {
	SujetoID, CuentaID, CertificadoSHA256, CASHA256 string
	CertificadoValidoHasta, CAValidaHasta           time.Time
	CRLSiguienteEn, RevocacionVerificadaEn          time.Time
}
