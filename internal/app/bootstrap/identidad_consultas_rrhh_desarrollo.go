package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const nombreManifiestoConsultasRRHHDesarrollo = "consultas-rrhh.json"

type archivoManifiestoConsultasRRHHDesarrollo struct {
	Version   int                                    `json:"version"`
	Autoridad string                                 `json:"autoridad"`
	Entradas  []archivoEntradaConsultaRRHHDesarrollo `json:"entradas"`
}

type archivoEntradaConsultaRRHHDesarrollo struct {
	Certificate     string `json:"certificate"`
	Identity        string `json:"identity"`
	Subject         string `json:"subject"`
	PerfilRef       string `json:"perfil_ref"`
	OrganizacionRef string `json:"organizacion_ref"`
	ClaseAmbito     string `json:"clase_ambito"`
	AmbitoRef       string `json:"ambito_ref"`
}

type identidadConsultaRRHHDesarrollo struct {
	identidad       identidadCertificadoDesarrollo
	perfilRef       string
	organizacionRef string
	clase           ports.ClaseAmbitoConsultaRRHH
	ambitoRef       string
}

func claseAmbitoConsultaRRHHDesarrollo(valor string) (ports.ClaseAmbitoConsultaRRHH, bool) {
	clase := ports.ClaseAmbitoConsultaRRHH(valor)
	switch clase {
	case ports.AmbitoOrganizacionRRHH, ports.AmbitoCentroRRHH, ports.AmbitoUnidadGestionRRHH:
		return clase, true
	default:
		return "", false
	}
}

// cargarIdentidadesConsultasRRHHDesarrollo deja el manifiesto ausente como
// compatibilidad explícita. Si existe, cada identidad de lectura es nominal.
func cargarIdentidadesConsultasRRHHDesarrollo(
	directorio string, ca *x509.Certificate,
	tecnicos ...identidadCertificadoDesarrollo,
) ([]identidadConsultaRRHHDesarrollo, bool, error) {
	ruta := filepath.Join(directorio, "identidad", nombreManifiestoConsultasRRHHDesarrollo)
	if _, err := os.Lstat(ruta); errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, ErrMaterialDesarrolloInvalido
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, 64<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return nil, false, ErrMaterialDesarrolloInvalido
	}
	var manifiesto archivoManifiestoConsultasRRHHDesarrollo
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&manifiesto); err != nil {
		return nil, false, ErrMaterialDesarrolloInvalido
	}
	var sobra any
	if err := decodificador.Decode(&sobra); !errors.Is(err, io.EOF) || manifiesto.Version != 1 ||
		manifiesto.Autoridad != AutoridadNoAutoritativa || len(manifiesto.Entradas) == 0 || ca == nil {
		return nil, false, ErrMaterialDesarrolloInvalido
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	porHuella := map[[sha256.Size]byte]struct{}{}
	porSujeto := map[string]struct{}{}
	resultado := make([]identidadConsultaRRHHDesarrollo, 0, len(manifiesto.Entradas))
	for _, entrada := range manifiesto.Entradas {
		certRuta, okCert := rutaMaterialConsultaRRHHDesarrollo(directorio, entrada.Certificate)
		identidadRuta, okIdentidad := rutaMaterialConsultaRRHHDesarrollo(directorio, entrada.Identity)
		clase, okClase := claseAmbitoConsultaRRHHDesarrollo(entrada.ClaseAmbito)
		if !okCert || !okIdentidad || !okClase || entrada.Subject == "" ||
			!domain.ReferenciaOpacaValida(entrada.PerfilRef) || !domain.ReferenciaOpacaValida(entrada.OrganizacionRef) ||
			!domain.ReferenciaOpacaValida(entrada.AmbitoRef) ||
			entrada.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
			(clase == ports.AmbitoOrganizacionRRHH && entrada.AmbitoRef != entrada.OrganizacionRef) {
			return nil, false, ErrMaterialDesarrolloInvalido
		}
		pem, err := leerFicheroMaterialSeguro(certRuta, tamanoMaximoFicheroMaterialDesarrollo)
		if err != nil {
			return nil, false, ErrMaterialDesarrolloInvalido
		}
		cert, err := decodificarCertificadoUnico(pem)
		if err != nil || cert == nil {
			return nil, false, ErrMaterialDesarrolloInvalido
		}
		if _, err = cert.Verify(x509.VerifyOptions{Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
			return nil, false, ErrMaterialDesarrolloInvalido
		}
		var identidad identidadCertificadoDesarrollo
		huellaCertificado := sha256.Sum256(cert.Raw)
		var tecnico *identidadCertificadoDesarrollo
		for indice := range tecnicos {
			candidato := &tecnicos[indice]
			if candidato.huella == huellaCertificado {
				if tecnico != nil {
					return nil, false, ErrMaterialDesarrolloInvalido
				}
				tecnico = candidato
			}
		}
		rol := "lector_rrhh"
		if tecnico != nil {
			if len(tecnico.principal.Roles) != 1 ||
				tecnico.principal.Roles[0] != rolTecnicoRRHHContratacionTemporalDesarrollo ||
				tecnico.principal.ID != entrada.Subject {
				return nil, false, ErrMaterialDesarrolloInvalido
			}
			rol = tecnico.principal.Roles[0]
		}
		identidad, err = cargarIdentidadDesarrollo(identidadRuta, cert, rol)
		if err != nil || identidad.principal.ID != entrada.Subject {
			return nil, false, ErrMaterialDesarrolloInvalido
		}
		if tecnico != nil && (identidad.huella != tecnico.huella ||
			identidad.principal.ID != tecnico.principal.ID) {
			return nil, false, ErrMaterialDesarrolloInvalido
		}
		if _, existe := porHuella[identidad.huella]; existe {
			return nil, false, ErrMaterialDesarrolloInvalido
		}
		if _, existe := porSujeto[entrada.Subject]; existe {
			return nil, false, ErrMaterialDesarrolloInvalido
		}
		porHuella[identidad.huella] = struct{}{}
		porSujeto[entrada.Subject] = struct{}{}
		resultado = append(resultado, identidadConsultaRRHHDesarrollo{identidad: identidad, perfilRef: entrada.PerfilRef, organizacionRef: entrada.OrganizacionRef, clase: clase, ambitoRef: entrada.AmbitoRef})
	}
	return resultado, true, nil
}

func rutaMaterialConsultaRRHHDesarrollo(directorio, relativa string) (string, bool) {
	if relativa == "" || filepath.IsAbs(relativa) || strings.Contains(relativa, "\\") {
		return "", false
	}
	limpia := filepath.Clean(relativa)
	if limpia == "." || limpia == ".." || strings.HasPrefix(limpia, ".."+string(filepath.Separator)) {
		return "", false
	}
	ruta := filepath.Join(directorio, limpia)
	if err := validarFicheroMaterialPresente(ruta); err != nil {
		return "", false
	}
	return ruta, true
}
