package config

import (
	"path/filepath"
	"strings"
)

const (
	// EnvExecutionProfile separa de forma explicita los datos y proveedores de
	// produccion de los no autoritativos. La variable heredada de Bolsa no puede
	// seleccionar este perfil.
	EnvExecutionProfile       = "VEC_EXECUTION_PROFILE"
	EnvDevelopmentGuard       = "VEC_DEVELOPMENT_GUARD"
	EnvDevelopmentMaterialDir = "VEC_DEVELOPMENT_MATERIAL_DIR"

	ExecutionProfileProduction  = "produccion"
	ExecutionProfileDevelopment = "desarrollo"
	AuthModeDevelopment         = "desarrollo"

	DefaultExecutionProfile = ExecutionProfileProduction

	// DevelopmentGuardAcknowledgement no es un secreto. Es la segunda llave
	// deliberada que evita seleccionar el perfil por una unica variable mal
	// copiada. El bootstrap comprueba las tres condiciones: perfil, modo y
	// reconocimiento literal.
	DevelopmentGuardAcknowledgement = "ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO"

	DevelopmentCACertificateRelativePath           = "ca/ca.crt"
	DevelopmentCAPrivateKeyRelativePath            = "ca/ca.key"
	DevelopmentServerCertificateRelativePath       = "tls/servidor.crt"
	DevelopmentServerPrivateKeyRelativePath        = "tls/servidor.key"
	DevelopmentClientCertificateRelativePath       = "mtls/cliente.crt"
	DevelopmentClientPrivateKeyRelativePath        = "mtls/cliente.key"
	DevelopmentIntervencionCertificateRelativePath = "mtls/intervencion.crt"
	DevelopmentIntervencionPrivateKeyRelativePath  = "mtls/intervencion.key"
	DevelopmentKMSSecretRelativePath               = "kms/clave-maestra.bin"
	DevelopmentKMSAttestationKeyRelativePath       = "kms/atestacion-ed25519.key"
	DevelopmentKMSAttestationPublicRelativePath    = "kms/atestacion-ed25519.pub"
	DevelopmentKMSRevalidationKeyRelativePath      = "kms/revalidacion-ed25519.key"
	DevelopmentKMSRevalidationPublicRelativePath   = "kms/revalidacion-ed25519.pub"
	DevelopmentTSASecretRelativePath               = "tsa/clave-hmac.bin"
	DevelopmentIdentityRelativePath                = "identidad/identidad.json"
	DevelopmentIntervencionIdentityRelativePath    = "identidad/intervencion.json"
	DevelopmentIdempotencyHMACConfigRelativePath   = "idempotencia/configuracion.json"
)

type DevelopmentMaterialPaths struct {
	CACertificate           string
	CAPrivateKey            string
	ServerCertificate       string
	ServerPrivateKey        string
	ClientCertificate       string
	ClientPrivateKey        string
	IntervencionCertificate string
	IntervencionPrivateKey  string
	KMSSecret               string
	KMSAttestationKey       string
	KMSAttestationPublic    string
	KMSRevalidationKey      string
	KMSRevalidationPublic   string
	TSASecret               string
	Identity                string
	IntervencionIdentity    string
	IdempotencyHMACConfig   string
}

func normalizeExecutionProfile(profile string) string {
	normalizado := strings.ToLower(strings.TrimSpace(profile))
	switch normalizado {
	case "":
		return DefaultExecutionProfile
	case ExecutionProfileDevelopment:
		return ExecutionProfileDevelopment
	case ExecutionProfileRRHHPresentation:
		return ExecutionProfileRRHHPresentation
	case ExecutionProfileProduction:
		return ExecutionProfileProduction
	default:
		// No degradar valores desconocidos a produccion. El bootstrap conserva
		// el valor canonico y falla con un error de configuracion explicito.
		return normalizado
	}
}

// DevelopmentEnabledByDoubleKey solo describe la seleccion solicitada. La
// raiz de composicion debe validar ademas que todos los proveedores requeridos
// son de desarrollo y rechazar mezclas con el perfil productivo.
func (c Config) DevelopmentEnabledByDoubleKey() bool {
	c = c.Normalize()
	return c.ExecutionProfile == ExecutionProfileDevelopment &&
		c.AuthMode == AuthModeDevelopment &&
		c.DevelopmentGuard == DevelopmentGuardAcknowledgement
}

// DevelopmentPaths centraliza el formato local generado por el script T21.
// Son ubicaciones de ejecucion, nunca valores que deban versionarse.
func (c Config) DevelopmentPaths() DevelopmentMaterialPaths {
	raiz := strings.TrimSpace(c.DevelopmentMaterialDir)
	unir := func(relativa string) string {
		if raiz == "" {
			return ""
		}
		return filepath.Join(raiz, filepath.FromSlash(relativa))
	}
	return DevelopmentMaterialPaths{
		CACertificate:           unir(DevelopmentCACertificateRelativePath),
		CAPrivateKey:            unir(DevelopmentCAPrivateKeyRelativePath),
		ServerCertificate:       unir(DevelopmentServerCertificateRelativePath),
		ServerPrivateKey:        unir(DevelopmentServerPrivateKeyRelativePath),
		ClientCertificate:       unir(DevelopmentClientCertificateRelativePath),
		ClientPrivateKey:        unir(DevelopmentClientPrivateKeyRelativePath),
		IntervencionCertificate: unir(DevelopmentIntervencionCertificateRelativePath),
		IntervencionPrivateKey:  unir(DevelopmentIntervencionPrivateKeyRelativePath),
		KMSSecret:               unir(DevelopmentKMSSecretRelativePath),
		KMSAttestationKey:       unir(DevelopmentKMSAttestationKeyRelativePath),
		KMSAttestationPublic:    unir(DevelopmentKMSAttestationPublicRelativePath),
		KMSRevalidationKey:      unir(DevelopmentKMSRevalidationKeyRelativePath),
		KMSRevalidationPublic:   unir(DevelopmentKMSRevalidationPublicRelativePath),
		TSASecret:               unir(DevelopmentTSASecretRelativePath),
		Identity:                unir(DevelopmentIdentityRelativePath),
		IntervencionIdentity:    unir(DevelopmentIntervencionIdentityRelativePath),
		IdempotencyHMACConfig:   unir(DevelopmentIdempotencyHMACConfigRelativePath),
	}
}
