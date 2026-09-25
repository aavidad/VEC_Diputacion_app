package interna

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
)

type materialIdentidadCertificado struct {
	claveID, politicaRef, huellaPolitica string
	firmante                             crypto.Signer
	rutaCertificados                     string
}

type documentoEmisorCertificado struct {
	Version        int    `json:"version"`
	ClaveID        string `json:"clave_id"`
	PoliticaRef    string `json:"politica_ref"`
	PoliticaHuella string `json:"politica_huella"`
}

func cargarMaterialIdentidadCertificado(directorio string) (materialIdentidadCertificado, error) {
	var vacio materialIdentidadCertificado
	if !filepath.IsAbs(directorio) || filepath.Clean(directorio) != directorio {
		return vacio, ErrCertificadoPersonalNoDisponible
	}
	base, err := os.Lstat(directorio)
	if err != nil || !base.IsDir() || base.Mode().Perm() != 0700 {
		return vacio, ErrCertificadoPersonalNoDisponible
	}
	directorioIdentidad := filepath.Join(directorio, "identidad")
	info, err := os.Lstat(directorioIdentidad)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return vacio, ErrCertificadoPersonalNoDisponible
	}
	meta, err := leerArchivoPrivadoIdentidad(filepath.Join(directorioIdentidad, "emisor.json"), 4096)
	if err != nil {
		return vacio, err
	}
	defer clear(meta)
	if rechazarClavesDuplicadasPools(meta) != nil {
		return vacio, ErrCertificadoPersonalNoDisponible
	}
	var documento documentoEmisorCertificado
	lector := json.NewDecoder(bytes.NewReader(meta))
	lector.DisallowUnknownFields()
	if lector.Decode(&documento) != nil || lector.Decode(new(any)) != io.EOF ||
		documento.Version != 1 || !claveEmisorCertificadoValida(documento.ClaveID) ||
		!identificadorCertificadoPersonalValido(documento.PoliticaRef, "pga_") ||
		!huellaCertificadoPersonalValida(documento.PoliticaHuella) {
		return vacio, ErrCertificadoPersonalNoDisponible
	}
	claveBytes, err := leerArchivoPrivadoIdentidad(filepath.Join(directorioIdentidad, "emisor.key"), 8192)
	if err != nil {
		return vacio, err
	}
	defer clear(claveBytes)
	bloque, resto := pem.Decode(claveBytes)
	if bloque == nil || bloque.Type != "PRIVATE KEY" || len(bytes.TrimSpace(resto)) != 0 {
		return vacio, ErrCertificadoPersonalNoDisponible
	}
	privada, err := x509.ParsePKCS8PrivateKey(bloque.Bytes)
	if err != nil {
		return vacio, ErrCertificadoPersonalNoDisponible
	}
	firmante, valido := privada.(ed25519.PrivateKey)
	if !valido || len(firmante) != ed25519.PrivateKeySize {
		return vacio, ErrCertificadoPersonalNoDisponible
	}
	rutaCertificados := filepath.Join(directorioIdentidad, "certificados.json")
	if _, err := nuevoRegistroCertificadosPersonales(rutaCertificados); err != nil {
		return vacio, err
	}
	return materialIdentidadCertificado{
		claveID: documento.ClaveID, politicaRef: documento.PoliticaRef,
		huellaPolitica: documento.PoliticaHuella, firmante: firmante,
		rutaCertificados: rutaCertificados,
	}, nil
}

func claveEmisorCertificadoValida(v string) bool {
	if len(v) < 8 || len(v) > 128 {
		return false
	}
	for _, c := range v {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func leerArchivoPrivadoIdentidad(ruta string, limite int64) ([]byte, error) {
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 ||
		info.Size() <= 0 || info.Size() > limite {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	archivo, err := os.Open(ruta)
	if err != nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	defer archivo.Close()
	actual, err := archivo.Stat()
	if err != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() ||
		actual.Mode().Perm() != 0600 {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, limite+1))
	if err != nil || len(contenido) == 0 || int64(len(contenido)) > limite {
		clear(contenido)
		return nil, ErrCertificadoPersonalNoDisponible
	}
	return contenido, nil
}
