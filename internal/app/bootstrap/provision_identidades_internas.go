package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
)

// La huella identifica los bytes acreditados por Dirección a las 05:00 CEST.
// La acreditación de procedencia no es una aprobación de perfiles ni un permiso.
const HuellaFuenteIdentidadesInternasH6 = "14ee4c7b94e08d28ba299a7278e65261bec1d3344cb485b7ebef3ebd00cec056"

// El plan aprobado para cotejo conserva exactamente cuatro denegaciones.
const HuellaPlanIdentidadesInternasH6 = "38e5a7cd8a0b43df0e99d5bb1f2e8ee93f5eb5652688f7ba6ddf9d3cc0260e18"

var (
	ErrFuenteIdentidadesInternasH6    = errors.New("fuente_invalida")
	ErrPlanIdentidadesInternasH6      = errors.New("plan_no_coincide")
	ErrProvisionIdentidadesInternasH6 = errors.New("autoridad_no_disponible")
)

const maxFuenteIdentidadesInternasH6 = 256 << 10

type fuenteIdentidadesInternasH6 struct {
	Esquema            string `json:"esquema"`
	Version            int    `json:"version"`
	Estado             string `json:"estado"`
	DatosSinteticos    bool   `json:"datos_sinteticos"`
	AutoridadMaestra   string `json:"autoridad_maestra"`
	ProvisionEjecutada bool   `json:"provision_ejecutada"`
	Acreditacion       any    `json:"acreditacion"`
	Personas           []struct {
		Funcion           string  `json:"funcion"`
		Sujeto            *string `json:"sujeto"`
		PersonaRef        string  `json:"persona_ref"`
		PrincipalH6       string  `json:"principal_h6"`
		CentroRef         string  `json:"centro_ref"`
		PuestoRef         string  `json:"puesto_ref"`
		PerfilPublicadoH6 *string `json:"perfil_publicado_h6"`
		Certificado       struct {
			Estado               string `json:"estado"`
			DER                  string `json:"der_sha256"`
			PEM                  string `json:"pem_sha256"`
			ClavePrivadaCotejada *bool  `json:"clave_privada_cotejada"`
		} `json:"certificado"`
		PreimagenAusenciaH6 *struct {
			Cuenta     bool `json:"derived_account_exists"`
			Proyeccion bool `json:"derived_account_projection_exists"`
			Asignacion bool `json:"derived_profile_assignment_exists"`
			Contexto   bool `json:"derived_profile_context_exists"`
		} `json:"preimagen_ausencia_h6"`
		PreimagenH6 *struct {
			Actor             string `json:"actor"`
			AssignmentRef     string `json:"assignment_ref"`
			AssignmentSHA256  string `json:"assignment_sha256"`
			AssignmentVersion int    `json:"assignment_version"`
			Center            string `json:"center"`
			ControlState      string `json:"control_state"`
			Position          string `json:"position"`
			Profile           string `json:"profile"`
		} `json:"preimagen_h6"`
	} `json:"personas"`
	Centros struct {
		CertificadosCotejados bool `json:"certificados_cotejados"`
		Solicitante           struct {
			PersonaRef string `json:"persona_ref"`
			CentroRef  string `json:"centro_ref"`
			PuestoRef  string `json:"puesto_ref"`
			PerfilH6   string `json:"perfil_h6"`
		} `json:"solicitante"`
		Ratificador struct {
			PersonaRef string `json:"persona_ref"`
			CentroRef  string `json:"centro_ref"`
			PuestoRef  string `json:"puesto_ref"`
			PerfilH6   string `json:"perfil_h6"`
		} `json:"ratificador"`
	} `json:"centros_propuestos"`
}

// ActorPlanIdentidadesInternasH6 sólo publica función y causas codificadas.
// No contiene nombres, sujetos, referencias personales ni huellas de certificado.
type ActorPlanIdentidadesInternasH6 struct {
	Funcion  string   `json:"funcion"`
	Estado   string   `json:"estado"`
	Bloqueos []string `json:"bloqueos"`
}

type PlanIdentidadesInternasH6 struct {
	Esquema      string                           `json:"esquema"`
	FuenteSHA256 string                           `json:"fuente_sha256"`
	PlanSHA256   string                           `json:"plan_sha256"`
	Estado       string                           `json:"estado"`
	Actores      []ActorPlanIdentidadesInternasH6 `json:"actores"`
}

// PlanificarIdentidadesInternasH6 lee exclusivamente la fuente sintética
// acreditada. Un archivo con idéntica forma y otra huella no adquiere autoridad.
func PlanificarIdentidadesInternasH6(r io.Reader) (PlanIdentidadesInternasH6, error) {
	var vacio PlanIdentidadesInternasH6
	if r == nil {
		return vacio, ErrFuenteIdentidadesInternasH6
	}
	b, err := io.ReadAll(io.LimitReader(r, maxFuenteIdentidadesInternasH6+1))
	if err != nil || len(b) == 0 || len(b) > maxFuenteIdentidadesInternasH6 {
		return vacio, ErrFuenteIdentidadesInternasH6
	}
	suma := sha256.Sum256(b)
	if hex.EncodeToString(suma[:]) != HuellaFuenteIdentidadesInternasH6 {
		return vacio, ErrFuenteIdentidadesInternasH6
	}
	var f fuenteIdentidadesInternasH6
	if validarClavesJSONUnicas(b) != nil || json.Unmarshal(b, &f) != nil || f.Esquema != "vec.fuente.sintetica.interna.propuesta.v1" || f.Version != 1 ||
		f.Estado != "propuesta" || !f.DatosSinteticos || f.AutoridadMaestra != "direccion-vec:sintetico:20261001" ||
		f.ProvisionEjecutada || f.Acreditacion != nil || len(f.Personas) != 4 ||
		f.Centros.CertificadosCotejados || f.Centros.Solicitante.PersonaRef == "" ||
		f.Centros.Solicitante.CentroRef == "" || f.Centros.Solicitante.PuestoRef == "" || f.Centros.Solicitante.PerfilH6 == "" ||
		f.Centros.Ratificador.PersonaRef == "" || f.Centros.Ratificador.CentroRef == "" ||
		f.Centros.Ratificador.PuestoRef == "" || f.Centros.Ratificador.PerfilH6 == "" ||
		f.Centros.Solicitante.PersonaRef == f.Centros.Ratificador.PersonaRef {
		return vacio, ErrFuenteIdentidadesInternasH6
	}
	funciones := []string{"rrhh", "intervencion", "solicitante_centro", "ratificador_centro"}
	actores := make([]ActorPlanIdentidadesInternasH6, 0, 4)
	for _, funcion := range funciones {
		var encontrado bool
		for _, p := range f.Personas {
			if p.Funcion != funcion {
				continue
			}
			if encontrado || p.PersonaRef == "" {
				return vacio, ErrFuenteIdentidadesInternasH6
			}
			encontrado = true
			bloqueos := []string{"aprobacion_nominal_ausente", "autoridad_cas_ausente"}
			if funcion == "rrhh" || funcion == "intervencion" {
				if p.PreimagenAusenciaH6 == nil || p.PreimagenH6 != nil ||
					p.PreimagenAusenciaH6.Cuenta || p.PreimagenAusenciaH6.Proyeccion ||
					p.PreimagenAusenciaH6.Asignacion || p.PreimagenAusenciaH6.Contexto {
					return vacio, ErrFuenteIdentidadesInternasH6
				}
				if p.Sujeto == nil || strings.TrimSpace(*p.Sujeto) == "" || p.Certificado.Estado != "publico_h1_cotejado" ||
					len(p.Certificado.DER) != 64 || len(p.Certificado.PEM) != 64 || p.PerfilPublicadoH6 != nil {
					return vacio, ErrFuenteIdentidadesInternasH6
				}
				bloqueos = append(bloqueos, "cuenta_h6_ausente", "contexto_h6_ausente", "asignacion_h6_ausente", "cuenta_hmac_sin_provisionar", "posesion_certificado_no_acreditada")
			} else {
				centroPersona, centroPerfil := f.Centros.Solicitante.PersonaRef, f.Centros.Solicitante.PerfilH6
				centroRef, puestoRef := f.Centros.Solicitante.CentroRef, f.Centros.Solicitante.PuestoRef
				if funcion == "ratificador_centro" {
					centroPersona, centroPerfil = f.Centros.Ratificador.PersonaRef, f.Centros.Ratificador.PerfilH6
					centroRef, puestoRef = f.Centros.Ratificador.CentroRef, f.Centros.Ratificador.PuestoRef
				}
				if p.PreimagenAusenciaH6 != nil || p.PreimagenH6 == nil ||
					p.PreimagenH6.AssignmentRef == "" || len(p.PreimagenH6.AssignmentSHA256) != 64 ||
					p.PreimagenH6.AssignmentVersion != 2 || p.PreimagenH6.ControlState != "habilitada" ||
					p.PreimagenH6.Actor != p.PrincipalH6 || p.PreimagenH6.Center != centroRef ||
					p.PreimagenH6.Position != puestoRef || p.PreimagenH6.Profile != centroPerfil ||
					p.CentroRef != centroRef || p.PuestoRef != puestoRef ||
					p.Sujeto != nil || p.Certificado.Estado != "pendiente" || p.Certificado.DER != "" ||
					p.Certificado.PEM != "" || p.PerfilPublicadoH6 == nil || *p.PerfilPublicadoH6 != centroPerfil ||
					p.PersonaRef != centroPersona {
					return vacio, ErrFuenteIdentidadesInternasH6
				}
				bloqueos = append(bloqueos, "asignacion_h6_preimagen_sin_revalidar", "cuenta_nominal_no_acreditada", "contexto_nominal_no_acreditado", "sujeto_pendiente", "certificado_pendiente", "material_hmac_no_acreditado")
			}
			slices.Sort(bloqueos)
			actores = append(actores, ActorPlanIdentidadesInternasH6{Funcion: funcion, Estado: "denegado", Bloqueos: bloqueos})
		}
		if !encontrado {
			return vacio, ErrFuenteIdentidadesInternasH6
		}
	}
	huellaPlan, err := huellaActoresPlanIdentidadesInternasH6(actores)
	if err != nil || huellaPlan != HuellaPlanIdentidadesInternasH6 {
		return vacio, ErrFuenteIdentidadesInternasH6
	}
	return PlanIdentidadesInternasH6{
		Esquema: "vec.h6.provision.plan.v1", FuenteSHA256: HuellaFuenteIdentidadesInternasH6,
		PlanSHA256: huellaPlan, Estado: "bloqueado", Actores: actores,
	}, nil
}

func huellaActoresPlanIdentidadesInternasH6(actores []ActorPlanIdentidadesInternasH6) (string, error) {
	canon, err := json.Marshal(actores)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(append([]byte("vec.h6.provision.plan.v1\x00"+HuellaFuenteIdentidadesInternasH6+"\x00"), canon...))
	return hex.EncodeToString(suma[:]), nil
}

// LeerFuenteIdentidadesInternasH6 exige fichero regular privado. La huella
// acreditada sigue siendo la guarda decisiva ante sustituciones del path.
func LeerFuenteIdentidadesInternasH6(path string) (PlanIdentidadesInternasH6, error) {
	b, err := LeerMaterialProvisionExterna(path, maxFuenteIdentidadesInternasH6)
	if err != nil {
		return PlanIdentidadesInternasH6{}, ErrFuenteIdentidadesInternasH6
	}
	defer clear(b)
	return PlanificarIdentidadesInternasH6(bytes.NewReader(b))
}

// Reconciliar coteja la huella de un plan previo con la fuente exacta. Nunca
// convierte un acuse de origen en una asignación de perfil.
func ReconciliarIdentidadesInternasH6(plan PlanIdentidadesInternasH6, huellaEsperada string) (PlanIdentidadesInternasH6, error) {
	huellaActores, err := huellaActoresPlanIdentidadesInternasH6(plan.Actores)
	if err != nil || plan.Esquema != "vec.h6.provision.plan.v1" ||
		plan.FuenteSHA256 != HuellaFuenteIdentidadesInternasH6 || plan.Estado != "bloqueado" ||
		plan.PlanSHA256 != HuellaPlanIdentidadesInternasH6 || huellaEsperada != HuellaPlanIdentidadesInternasH6 ||
		huellaActores != HuellaPlanIdentidadesInternasH6 {
		return PlanIdentidadesInternasH6{}, ErrPlanIdentidadesInternasH6
	}
	return plan, nil
}

// Aplicar requiere un caso de uso durable gobernado que este corte no posee:
// preimagen CAS, aprobación separada, clave HMAC bajo custodio y cotejo nominal.
// Cualquier llamada falla antes de escribir o crear una concesión.
func AplicarIdentidadesInternasH6(_ PlanIdentidadesInternasH6) error {
	return ErrProvisionIdentidadesInternasH6
}
