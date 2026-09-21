package bootstrap

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const nombreManifiestoIdentidadBorradorBolsaDesarrollo = "bolsa-bback.json"

type archivoManifiestoIdentidadBorradorBolsaDesarrollo struct {
	Version           int    `json:"version"`
	Autoridad         string `json:"autoridad"`
	Sujeto            string `json:"sujeto"`
	CertificadoSHA256 string `json:"certificado_sha256"`
	PerfilRef         string `json:"perfil_ref"`
	UnidadRef         string `json:"unidad_ref"`
	AmbitoRef         string `json:"ambito_ref"`
}

// soporteSesionBorradorBolsaDesarrollo es una fuente nominal preparada al
// arrancar. No recibe ni acepta perfil, sujeto, ámbito o identidad del cliente.
type soporteSesionBorradorBolsaDesarrollo struct {
	// soporteCanal sólo alimenta al proveedor de sesión que registra y
	// revalida en cada petición; no implementa ResolverContexto por diseño.
	soporteCanal *soporteAltaContratacionTemporalDesarrollo
	unidadRef    string
	ambitoRef    string
}

func discriminadorContextoSinteticoBorradorBolsaDesarrollo() discriminadorContextoSinteticoDesarrollo {
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil-bolsa-bback-v1", vinculo: "vinculo-bolsa-bback-v1",
		// Cuenta y persona ya existen bajo esta acreditación CT. Reutilizarla
		// evita que el publicador intente sustituir su procedencia histórica.
		procedencia: "procedencia", registro: "registro-contexto-bolsa-bback-v1",
		autenticacion: "autenticacion-bolsa-bback-v1", asercion: "asercion-bolsa-bback-v1",
		sesion: "sesion-bolsa-bback-v1", controlSesion: "control-sesion-bolsa-bback-v1",
		politicaGarantia: "politica-garantia-bolsa-bback-v1",
	}
}

func nuevoSoporteSesionBorradorBolsaDesarrollo(
	directorio string,
	soporteCT *soporteAltaContratacionTemporalDesarrollo,
	ahora time.Time,
) (*soporteSesionBorradorBolsaDesarrollo, error) {
	if soporteCT == nil || directorio == "" || !domain.InstanteUTCCanonico(ahora) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	soporteCT.mu.Lock()
	principalID, certificado := soporteCT.principalID, soporteCT.certificadoSHA256
	contextoCT := soporteCT.contexto
	sello := soporteCT.sello
	reloj := soporteCT.reloj
	soporteCT.mu.Unlock()
	if sello == nil || !identificadorSesionDesarrolloValido(principalID) ||
		!contextoSinteticoCTConsistenteParaBorradorBolsa(principalID, certificado, contextoCT) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	manifiesto, err := cargarManifiestoIdentidadBorradorBolsaDesarrollo(directorio)
	if err != nil || manifiesto.Sujeto != principalID || manifiesto.CertificadoSHA256 != certificado ||
		!perfilActivoSeguridadComunValido(manifiesto.PerfilRef) ||
		!domain.ReferenciaOpacaValida(manifiesto.UnidadRef) || !domain.ReferenciaOpacaValida(manifiesto.AmbitoRef) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	principal := dominiovec.Principal{
		ID: principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": certificado,
		},
	}
	contextoBolsa, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorContextoSinteticoBorradorBolsaDesarrollo(),
	)
	if err != nil || !contextoSinteticoBolsaSeparadoDeCT(contextoCT, contextoBolsa) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	datos, err := contextoBolsa.Vinculo.Datos()
	if err != nil || manifiesto.PerfilRef != datos.PerfilActivoRef {
		return nil, ErrMaterialDesarrolloInvalido
	}
	canal := &soporteAltaContratacionTemporalDesarrollo{
		sello: sello, principalID: principalID, certificadoSHA256: certificado,
		contexto: contextoBolsa, reloj: reloj,
	}
	return &soporteSesionBorradorBolsaDesarrollo{soporteCanal: canal, unidadRef: manifiesto.UnidadRef, ambitoRef: manifiesto.AmbitoRef}, nil
}

func cargarManifiestoIdentidadBorradorBolsaDesarrollo(
	directorio string,
) (archivoManifiestoIdentidadBorradorBolsaDesarrollo, error) {
	ruta := filepath.Join(directorio, "identidad", nombreManifiestoIdentidadBorradorBolsaDesarrollo)
	contenido, err := leerFicheroMaterialSeguro(ruta, 16<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return archivoManifiestoIdentidadBorradorBolsaDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	var manifiesto archivoManifiestoIdentidadBorradorBolsaDesarrollo
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&manifiesto); err != nil {
		return archivoManifiestoIdentidadBorradorBolsaDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	var sobra any
	if err := decodificador.Decode(&sobra); !errors.Is(err, io.EOF) || manifiesto.Version != 1 ||
		manifiesto.Autoridad != AutoridadNoAutoritativa || !huellaCertificadoDesarrolloValida(manifiesto.CertificadoSHA256) {
		return archivoManifiestoIdentidadBorradorBolsaDesarrollo{}, ErrMaterialDesarrolloInvalido
	}
	return manifiesto, nil
}

func contextoSinteticoCTConsistenteParaBorradorBolsa(
	principalID, certificado string,
	contexto ports.ContextoAutorizacionAltaV3,
) bool {
	if !huellaCertificadoDesarrolloValida(certificado) || contexto.Vinculo.ValidarPara(contexto.Resultado) != nil {
		return false
	}
	datos, err := contexto.Vinculo.Datos()
	base := principalID + "\x00" + certificado
	return err == nil && datos.CuentaRef == referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta") &&
		datos.PrincipalID == referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona") &&
		datos.PerfilActivoRef == referenciaAltaContratacionTemporalDesarrollo("prf_", base+"\x00perfil") &&
		datos.ContextoActorRef == referenciaAltaContratacionTemporalDesarrollo("vca_", base+"\x00vinculo") &&
		datos.RegistroContextoRef == referenciaAltaContratacionTemporalDesarrollo("rca_", base+"\x00registro-contexto")
}

func contextoSinteticoBolsaSeparadoDeCT(
	ct, bolsa ports.ContextoAutorizacionAltaV3,
) bool {
	ctDatos, errCT := ct.Vinculo.Datos()
	bolsaDatos, errBolsa := bolsa.Vinculo.Datos()
	procedenciaCT, errProcedenciaCT := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(ct.Resultado.ManifiestoProcedenciaCanonico)
	procedenciaBolsa, errProcedenciaBolsa := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(bolsa.Resultado.ManifiestoProcedenciaCanonico)
	return errCT == nil && errBolsa == nil && ct.Vinculo.ValidarPara(ct.Resultado) == nil &&
		bolsa.Vinculo.ValidarPara(bolsa.Resultado) == nil &&
		errProcedenciaCT == nil && errProcedenciaBolsa == nil &&
		ctDatos.CuentaRef == bolsaDatos.CuentaRef && ctDatos.PrincipalID == bolsaDatos.PrincipalID &&
		ct.Resultado.Contexto.PersonaRef == bolsa.Resultado.Contexto.PersonaRef &&
		ct.Resultado.Contexto.Instantanea.CuentaRef == bolsa.Resultado.Contexto.Instantanea.CuentaRef &&
		ct.Resultado.Contexto.PerfilActivoRef != bolsa.Resultado.Contexto.PerfilActivoRef &&
		ctDatos.PerfilActivoRef != bolsaDatos.PerfilActivoRef &&
		ctDatos.ContextoActorRef != bolsaDatos.ContextoActorRef &&
		ctDatos.RegistroContextoRef != bolsaDatos.RegistroContextoRef &&
		procedenciaCT.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1 == procedenciaBolsa.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1 &&
		procedenciaCT.Persona.AcreditacionProcedenciaComponenteContextoActorV1 == procedenciaBolsa.Persona.AcreditacionProcedenciaComponenteContextoActorV1
}

func huellaCertificadoDesarrolloValida(valor string) bool {
	bytes, err := hex.DecodeString(valor)
	return err == nil && len(bytes) == 32 && hex.EncodeToString(bytes) == valor
}
