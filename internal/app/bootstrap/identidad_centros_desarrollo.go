package bootstrap

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type adscripcionCentroDesarrollo struct {
	CentroRef          string
	PuestoRef          string
	RatificadorSubject string
}

type archivoCentrosDesarrollo struct {
	Version   int                       `json:"version"`
	Autoridad string                    `json:"autoridad"`
	Entradas  []entradaCentroDesarrollo `json:"entradas"`
}
type entradaCentroDesarrollo struct {
	Certificate        string `json:"certificate"`
	Identity           string `json:"identity"`
	Role               string `json:"role"`
	CentroRef          string `json:"centro_ref"`
	PuestoRef          string `json:"puesto_ref"`
	RatificadorSubject string `json:"ratificador_subject,omitempty"`
}

type identidadesCentrosDesarrollo struct {
	Identidades   []identidadCertificadoDesarrollo
	Adscripciones map[string]adscripcionCentroDesarrollo
}

func cargarAdscripcionesCentrosDesarrollo(root string, ca *x509.Certificate, existentes ...identidadCertificadoDesarrollo) (identidadesCentrosDesarrollo, error) {
	ruta := filepath.Join(root, "identidad", "centros.json")
	if _, err := os.Lstat(ruta); errors.Is(err, os.ErrNotExist) {
		return identidadesCentrosDesarrollo{}, nil
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	if validarClavesJSONUnicas(contenido) != nil {
		return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	var archivo archivoCentrosDesarrollo
	dec := json.NewDecoder(strings.NewReader(string(contenido)))
	dec.DisallowUnknownFields()
	if dec.Decode(&archivo) != nil || !errors.Is(dec.Decode(&struct{}{}), io.EOF) || archivo.Version != 1 || archivo.Autoridad != AutoridadNoAutoritativa || len(archivo.Entradas) == 0 || len(archivo.Entradas) > 32 || ca == nil {
		return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	porSujeto := make(map[string]adscripcionCentroDesarrollo, len(archivo.Entradas))
	identidades := make([]identidadCertificadoDesarrollo, 0, len(archivo.Entradas))
	sujetosEntrada := make([]string, 0, len(archivo.Entradas))
	porRol := make(map[string]string, len(archivo.Entradas))
	sujetos := make(map[string]bool)
	huellas := make(map[[sha256.Size]byte]bool)
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	for _, e := range archivo.Entradas {
		if e.Role != "solicitante_centro" && e.Role != "ratificador_centro" || !domain.ReferenciaOpacaValida(e.CentroRef) || !domain.ReferenciaOpacaValida(e.PuestoRef) || e.Identity == "" || e.Certificate == "" || !rutaMaterialCentroSegura(root, e.Identity) || !rutaMaterialCentroSegura(root, e.Certificate) {
			return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
		}
		certBytes, err := leerFicheroMaterialSeguro(filepath.Join(root, e.Certificate), tamanoMaximoFicheroMaterialDesarrollo)
		if err != nil {
			return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
		}
		cert, err := decodificarCertificadoUnico(certBytes)
		if err != nil {
			return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
		}
		if _, err = cert.Verify(x509.VerifyOptions{Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
			return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
		}
		identidad, err := cargarIdentidadDesarrollo(filepath.Join(root, e.Identity), cert, e.Role)
		if err != nil || sujetos[identidad.principal.ID] || huellas[identidad.huella] {
			return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
		}
		for _, existente := range existentes {
			if existente.principal.ID == identidad.principal.ID || existente.huella == identidad.huella {
				return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
			}
		}
		sujetos[identidad.principal.ID] = true
		huellas[identidad.huella] = true
		porRol[identidad.principal.ID] = e.Role
		if e.Role == "ratificador_centro" && e.RatificadorSubject != "" || e.Role == "solicitante_centro" && e.RatificadorSubject == "" {
			return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
		}
		porSujeto[identidad.principal.ID] = adscripcionCentroDesarrollo{CentroRef: e.CentroRef, PuestoRef: e.PuestoRef, RatificadorSubject: e.RatificadorSubject}
		identidades = append(identidades, identidad)
		sujetosEntrada = append(sujetosEntrada, identidad.principal.ID)
	}
	for indice, e := range archivo.Entradas {
		if e.Role == "solicitante_centro" {
			rat, ok := porSujeto[e.RatificadorSubject]
			if !ok || porRol[e.RatificadorSubject] != "ratificador_centro" || rat.CentroRef != e.CentroRef || sujetosEntrada[indice] == e.RatificadorSubject {
				return identidadesCentrosDesarrollo{}, ErrMaterialDesarrolloInvalido
			}
		}
	}
	return identidadesCentrosDesarrollo{Identidades: identidades, Adscripciones: porSujeto}, nil
}

func rutaMaterialCentroSegura(root, relativa string) bool {
	if filepath.IsAbs(relativa) || filepath.Clean(relativa) != relativa || relativa == "." || relativa == ".." || strings.HasPrefix(relativa, ".."+string(filepath.Separator)) {
		return false
	}
	_, err := os.Lstat(filepath.Join(root, relativa))
	return err == nil
}
