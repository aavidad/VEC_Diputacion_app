package interna

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	nombrePoliticaFirmaVecDesarrollo  = "politica_firma_vec_desarrollo.json"
	limitePoliticaFirmaVecDesarrollo  = 16 << 10
	esquemaPoliticaFirmaVecDesarrollo = "vec.firma-vec.admision-desarrollo.v1"
)

var ErrPoliticaFirmaVecDesarrolloNoDisponible = errors.New("composicion interna: politica firma vec desarrollo no disponible")

type codigoCargaPoliticaFirmaVec string

const (
	codigoPoliticaConfiguracion codigoCargaPoliticaFirmaVec = "configuracion"
	codigoPoliticaPin           codigoCargaPoliticaFirmaVec = "pin"
	codigoPoliticaDirectorio    codigoCargaPoliticaFirmaVec = "directorio"
	codigoPoliticaArchivo       codigoCargaPoliticaFirmaVec = "archivo"
	codigoPoliticaLectura       codigoCargaPoliticaFirmaVec = "lectura"
	codigoPoliticaIntegridad    codigoCargaPoliticaFirmaVec = "integridad"
	codigoPoliticaJSON          codigoCargaPoliticaFirmaVec = "json"
	codigoPoliticaContrato      codigoCargaPoliticaFirmaVec = "contrato"
	codigoPoliticaConstructor   codigoCargaPoliticaFirmaVec = "constructor"
	codigoPoliticaVentana       codigoCargaPoliticaFirmaVec = "ventana"
)

var (
	errPoliticaConfiguracion = errors.New("configuracion de politica no valida")
	errPoliticaMaterial      = errors.New("material privado de politica no valido")
	errPoliticaPin           = errors.New("pin de politica no valido")
	errPoliticaIntegridad    = errors.New("integridad de politica no valida")
	errPoliticaContrato      = errors.New("contrato de politica no valido")
	errPoliticaVentana       = errors.New("ventana de politica no valida")
)

// ErrorCargaPoliticaFirmaVecDesarrollo conserva la causa para diagnóstico
// interno mediante errors.Is/As. Error() nunca imprime rutas ni material.
type ErrorCargaPoliticaFirmaVecDesarrollo struct {
	codigo codigoCargaPoliticaFirmaVec
	causa  error
}

func (e *ErrorCargaPoliticaFirmaVecDesarrollo) Error() string {
	return ErrPoliticaFirmaVecDesarrolloNoDisponible.Error()
}

func (e *ErrorCargaPoliticaFirmaVecDesarrollo) Unwrap() []error {
	if e == nil || e.causa == nil {
		return []error{ErrPoliticaFirmaVecDesarrolloNoDisponible}
	}
	return []error{ErrPoliticaFirmaVecDesarrolloNoDisponible, e.causa}
}

func (e *ErrorCargaPoliticaFirmaVecDesarrollo) Codigo() string {
	if e == nil {
		return ""
	}
	return string(e.codigo)
}

func falloCargaPoliticaFirmaVec(codigo codigoCargaPoliticaFirmaVec, causa error) error {
	return &ErrorCargaPoliticaFirmaVecDesarrollo{codigo: codigo, causa: causa}
}

type documentoPoliticaFirmaVecDesarrollo struct {
	Esquema                     string `json:"esquema"`
	Entorno                     string `json:"entorno"`
	VigenteDesde                string `json:"vigente_desde"`
	Referencia                  string `json:"referencia"`
	Version                     uint64 `json:"version"`
	HuellaSHA256                string `json:"huella_sha256"`
	RetiradaEn                  string `json:"retirada_en"`
	PoliticaAutenticacionRef    string `json:"politica_autenticacion_ref"`
	PoliticaAutenticacionSHA256 string `json:"politica_autenticacion_sha256"`
	RolVersionRef               string `json:"rol_version_ref"`
	RolSHA256                   string `json:"rol_sha256"`
	ControlRevision             uint64 `json:"control_revision"`
	ControlSHA256               string `json:"control_sha256"`
}

// cargarPoliticaFirmaVecDesarrolloPrivada consume una aprobación ya
// aprovisionada. El pin SHA256 del archivo llega por configuración protegida
// externa; el contenido del archivo nunca se toma como su propia aprobación.
// Borrar este archivo después de cargarlo no revoca la política durable V3.
func cargarPoliticaFirmaVecDesarrolloPrivada(
	directorio, entornoConfiable, huellaArchivoAprobadaSHA256 string,
	reloj core.RelojVinculoAutenticacionActorV2,
) (vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo, error) {
	var vacia vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo
	if entornoConfiable != vecapp.EntornoAdmisionFirmaDesarrollo ||
		!filepath.IsAbs(directorio) || filepath.Clean(directorio) != directorio ||
		strings.TrimSpace(directorio) != directorio {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaConfiguracion, errPoliticaConfiguracion)
	}
	huellaAprobada, err := decodificarPinPoliticaFirmaVec(huellaArchivoAprobadaSHA256)
	if err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaPin, err)
	}
	infoDir, err := os.Lstat(directorio)
	if err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaDirectorio, err)
	}
	if !infoDir.IsDir() || infoDir.Mode().Perm() != 0o700 {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaDirectorio, errPoliticaMaterial)
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaDirectorio, err)
	}
	defer raiz.Close()
	actualDir, err := raiz.Stat(".")
	if err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaDirectorio, err)
	}
	if !os.SameFile(infoDir, actualDir) || actualDir.Mode().Perm() != 0o700 {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaDirectorio, errPoliticaMaterial)
	}
	info, err := raiz.Lstat(nombrePoliticaFirmaVecDesarrollo)
	if err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaArchivo, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 ||
		info.Size() <= 0 || info.Size() > limitePoliticaFirmaVecDesarrollo {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaArchivo, errPoliticaMaterial)
	}
	archivo, err := raiz.Open(nombrePoliticaFirmaVecDesarrollo)
	if err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaArchivo, err)
	}
	defer archivo.Close()
	actual, err := archivo.Stat()
	if err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaArchivo, err)
	}
	if !os.SameFile(info, actual) || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0o600 {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaArchivo, errPoliticaMaterial)
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, limitePoliticaFirmaVecDesarrollo+1))
	if err != nil {
		clear(contenido)
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaLectura, err)
	}
	if int64(len(contenido)) != actual.Size() || len(contenido) > limitePoliticaFirmaVecDesarrollo {
		clear(contenido)
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaLectura, errPoliticaMaterial)
	}
	defer clear(contenido)
	huellaActual := sha256.Sum256(contenido)
	if subtle.ConstantTimeCompare(huellaActual[:], huellaAprobada) != 1 {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaIntegridad, errPoliticaIntegridad)
	}
	if err := rechazarClavesDuplicadasPools(contenido); err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaJSON, err)
	}
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	var d documentoPoliticaFirmaVecDesarrollo
	if err := lector.Decode(&d); err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaJSON, err)
	}
	if err := lector.Decode(new(any)); err != io.EOF {
		if err == nil {
			err = errPoliticaContrato
		}
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaJSON, err)
	}
	if d.Esquema != esquemaPoliticaFirmaVecDesarrollo || d.Entorno != entornoConfiable {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaContrato, errPoliticaContrato)
	}
	desde, validoDesde := instantePoliticaFirmaVecDesarrollo(d.VigenteDesde)
	retirada, validaRetirada := instantePoliticaFirmaVecDesarrollo(d.RetiradaEn)
	if !validoDesde || !validaRetirada || !desde.Before(retirada) {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaVentana, errPoliticaVentana)
	}
	politica := vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo{
		Referencia: d.Referencia, Version: d.Version, HuellaSHA256: d.HuellaSHA256,
		RetiradaEn: retirada, PoliticaAutenticacionRef: d.PoliticaAutenticacionRef,
		PoliticaAutenticacionSHA256: d.PoliticaAutenticacionSHA256,
		RolVersionRef:               d.RolVersionRef, RolSHA256: d.RolSHA256,
		ControlRevision: d.ControlRevision, ControlSHA256: d.ControlSHA256,
	}
	if _, err := vecapp.NuevaAdmisionGarantiaFirmaVecDesarrollo(vecapp.ConfiguracionAdmisionGarantiaFirmaVecDesarrollo{
		EntornoConfiable: entornoConfiable, Politica: politica, Reloj: reloj,
	}); err != nil {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaConstructor, err)
	}
	ahora := reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.Before(desde) || !ahora.Before(retirada) {
		return vacia, falloCargaPoliticaFirmaVec(codigoPoliticaVentana, errPoliticaVentana)
	}
	return politica, nil
}

func decodificarPinPoliticaFirmaVec(valor string) ([]byte, error) {
	if !strings.HasPrefix(valor, "sha256:") || len(valor) != len("sha256:")+sha256.Size*2 {
		return nil, errPoliticaPin
	}
	texto := strings.TrimPrefix(valor, "sha256:")
	binario, err := hex.DecodeString(texto)
	if err != nil || len(binario) != sha256.Size || hex.EncodeToString(binario) != texto {
		return nil, errPoliticaPin
	}
	return binario, nil
}

func instantePoliticaFirmaVecDesarrollo(valor string) (time.Time, bool) {
	instante, err := time.Parse(time.RFC3339, valor)
	return instante, err == nil && instante.Location() == time.UTC && instante.Nanosecond() == 0 &&
		instante.Format(time.RFC3339) == valor
}
