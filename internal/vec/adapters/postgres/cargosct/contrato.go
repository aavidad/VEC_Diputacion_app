// Package cargosct adapts the private central cargo publisher. It neither
// creates identities nor installs roles while processing a CT request.
package cargosct

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrRechazada = errors.New("cargo_ct_rechazado")
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var key = regexp.MustCompile(`^[0-9a-f]{32}$`)

// Plan freezes the target and the complete preimage before approval. The
// assignment is the canonical document of the existing central authority.
type Plan struct {
	Version                    int       `json:"version"`
	Clave                      string    `json:"clave"`
	RolID                      string    `json:"rol_id"`
	VersionRolRef              string    `json:"version_rol_ref"`
	VersionRolSHA256           string    `json:"version_rol_sha256"`
	ControlRolSHA256           string    `json:"control_rol_sha256"`
	PreimagenSHA256            string    `json:"preimagen_sha256"`
	RegistroDestinoRef         string    `json:"registro_destino_ref"`
	DestinoSHA256              string    `json:"destino_sha256"`
	OrganizacionRef            string    `json:"organizacion_ref"`
	UnidadRef                  string    `json:"unidad_ref"`
	AsignacionCanonica         []byte    `json:"asignacion_canonica"`
	CaducaEn                   time.Time `json:"caduca_en"`
	VinculoCertificadoCanonico string    `json:"vinculo_certificado_canonico,omitempty"`
}

// MaterialOperador contains a genuine registered V3 decision, not claims of
// identity. SQL revalidates its session, context, assignment, role and policy.
type MaterialOperador struct {
	Decision       []byte `json:"decision"`
	Motivo         []byte `json:"motivo"`
	PersonaVersion uint64 `json:"persona_version"`
	PerfilVersion  uint64 `json:"perfil_version"`
}

type Solicitud struct {
	Plan     Plan             `json:"plan"`
	Operador MaterialOperador `json:"operador"`
}

type Resultado struct {
	Estado       string    `json:"estado"`
	Clave        string    `json:"clave"`
	PlanSHA256   string    `json:"plan_sha256"`
	ReciboRef    string    `json:"recibo_ref,omitempty"`
	AuditoriaRef string    `json:"auditoria_ref"`
	Fecha        time.Time `json:"fecha"`
	Version      int       `json:"version,omitempty"`
}

func referencia(s string) bool {
	if len(s) < 3 || len(s) > 256 {
		return false
	}
	for _, c := range s {
		if c < 33 || c > 126 || c == '*' || c == '\\' || c == '"' {
			return false
		}
	}
	return true
}

func (p Plan) Canonica() ([]byte, error) {
	if len(p.VinculoCertificadoCanonico) > 16384 {
		return nil, ErrRechazada
	}
	if p.VinculoCertificadoCanonico != "" {
		var x map[string]any
		if decodificar([]byte(p.VinculoCertificadoCanonico), &x) != nil || x == nil {
			return nil, ErrRechazada
		}
	}

	if p.Version != 1 || !key.MatchString(p.Clave) || !referencia(p.RolID) || !referencia(p.VersionRolRef) || !referencia(p.RegistroDestinoRef) || !referencia(p.OrganizacionRef) || !referencia(p.UnidadRef) || p.CaducaEn.IsZero() || p.CaducaEn.Location() != time.UTC || p.CaducaEn.Nanosecond()%1000 != 0 {
		return nil, ErrRechazada
	}
	for _, h := range []string{p.VersionRolSHA256, p.ControlRolSHA256, p.PreimagenSHA256, p.DestinoSHA256} {
		if !digest.MatchString(h) || h == string(bytes.Repeat([]byte("0"), 64)) {
			return nil, ErrRechazada
		}
	}
	var a core.AsignacionPerfil
	if len(p.AsignacionCanonica) > 16<<10 || decodificar(p.AsignacionCanonica, &a) != nil || a.Validar() != nil || a.Estado != core.EstadoAsignacionPerfilActiva || a.VersionRolRef != p.VersionRolRef || len(a.Ambitos) != 2 {
		return nil, ErrRechazada
	}
	got, _ := json.Marshal(a)
	if !bytes.Equal(got, p.AsignacionCanonica) {
		return nil, ErrRechazada
	}
	scope := map[string]string{}
	for _, x := range a.Ambitos {
		if len(x.Valores) != 1 {
			return nil, ErrRechazada
		}
		scope[x.Clave] = x.Valores[0]
	}
	if scope["organizacion_ref"] != p.OrganizacionRef || scope["unidad_ref"] != p.UnidadRef {
		return nil, ErrRechazada
	}
	return json.Marshal(p)
}

func (p Plan) Huella() (string, error) {
	b, err := p.Canonica()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Recurso is the exact central PDP input for a reviewed plan. The caller must
// obtain a fresh registered decision through the trusted central channel.
func (p Plan) Recurso(operacion string) (core.RecursoAutorizable, error) {
	h, err := p.Huella()
	if err != nil {
		return core.RecursoAutorizable{}, err
	}
	tipo := "perfil"
	switch operacion {
	case "preparar", "aplicar":
	case "aprobar":
		tipo = "propuesta_perfil"
	case "recuperar":
		tipo = "recibo_perfil"
	default:
		return core.RecursoAutorizable{}, ErrRechazada
	}
	return core.RecursoAutorizable{Referencia: "cargo_ct:" + p.Clave, ModuloID: "administracion", Tipo: tipo, Ambitos: map[string]string{"organizacion_ref": p.OrganizacionRef, "unidad_ref": p.UnidadRef}, Atributos: map[string]string{"plan_sha256": h}}, nil
}

func (m MaterialOperador) validar() bool {
	return len(m.Decision) > 0 && len(m.Decision) <= 524288 && len(m.Motivo) > 0 && len(m.Motivo) <= 65536 && m.PersonaVersion > 0 && m.PersonaVersion <= 9007199254740991 && m.PerfilVersion > 0 && m.PerfilVersion <= 9007199254740991
}

func decodificar(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return ErrRechazada
	}
	var extra any
	if !errors.Is(dec.Decode(&extra), io.EOF) {
		return ErrRechazada
	}
	// Reject repeated JSON keys, including within nested material.
	if err := clavesUnicas(json.NewDecoder(bytes.NewReader(b))); err != nil {
		return ErrRechazada
	}
	return nil
}
func clavesUnicas(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	x, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch x {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			s, ok := k.(string)
			if !ok || seen[s] {
				return ErrRechazada
			}
			seen[s] = true
			if err := clavesUnicas(d); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := clavesUnicas(d); err != nil {
				return err
			}
		}
	default:
		return ErrRechazada
	}
	_, err = d.Token()
	return err
}

// LeerPlan only reads the offline plan; it does not require operator material
// because constructing a PDP resource neither authorizes nor publishes.
func LeerPlan(b []byte) (Plan, error) {
	var s Solicitud
	if len(b) > 1<<20 || decodificar(b, &s) != nil {
		return Plan{}, ErrRechazada
	}
	if _, err := s.Plan.Canonica(); err != nil {
		return Plan{}, err
	}
	return s.Plan, nil
}

func LeerSolicitud(b []byte) (Solicitud, error) {
	var s Solicitud
	if len(b) > 1<<20 || decodificar(b, &s) != nil || !s.Operador.validar() {
		return Solicitud{}, ErrRechazada
	}
	if _, err := s.Plan.Canonica(); err != nil {
		return Solicitud{}, err
	}
	return s, nil
}

// AsignacionBase64 helps the offline operator construct a plan using the
// canonical central domain, without inventing an authentication binding.
func AsignacionBase64(a core.AsignacionPerfil) (string, error) {
	if a.Validar() != nil {
		return "", ErrRechazada
	}
	b, err := json.Marshal(a)
	return base64.StdEncoding.EncodeToString(b), err
}
