package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"
)

// ErrProvisionCargoCT no incluye datos de la persona ni de la conexión.
var ErrProvisionCargoCT = errors.New("provision_cargo_ct_rechazada")

var huellaProvisionCargoCT = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ConfiguracionProvisionCargoCT procede exclusivamente de un fichero privado
// fuera de Git. La referencia del acto de delegación documenta la competencia;
// no sustituye la aprobación técnica de esta instalación.
type ConfiguracionProvisionCargoCT struct {
	Version                  int       `json:"version"`
	Cargo                    string    `json:"cargo"`
	PrincipalID              string    `json:"principal_id"`
	PerfilRef                string    `json:"perfil_ref"`
	OrganizacionRef          string    `json:"organizacion_ref"`
	UnidadRef                string    `json:"unidad_ref"`
	VigenteDesde             time.Time `json:"vigente_desde"`
	VigenteHasta             time.Time `json:"vigente_hasta"`
	AprobacionInstalacionRef string    `json:"aprobacion_instalacion_ref"`
	AprobadorPrincipalID     string    `json:"aprobador_principal_id"`
	PreimagenSHA256          string    `json:"preimagen_sha256"`
	VersionRolSHA256         string    `json:"version_rol_sha256"`
	ActoDelegacionRef        string    `json:"acto_delegacion_ref,omitempty"`
}

// Los identificadores del catálogo son datos; ninguna etiqueta concede una
// acción. La versión exacta del rol debe existir ya en la fuente común.
var rolesCargoCT = map[string]string{
	"tecnico_solicitante":      "ct_cargo_tecnico_solicitante",
	"delegacion_solicitante":   "ct_cargo_delegacion_solicitante",
	"direccion_rrhh":           "ct_cargo_direccion_rrhh",
	"jefatura_servicio_rrhh":   "ct_cargo_jefatura_servicio_rrhh",
	"diputacion_delegada_rrhh": "ct_cargo_diputacion_delegada_rrhh",
}

func (c ConfiguracionProvisionCargoCT) validar(ahora time.Time) bool {
	rol, conocida := rolesCargoCT[c.Cargo]
	return c.Version == 1 && conocida && rol != "" &&
		referenciaCargoCT(c.PrincipalID) && referenciaCargoCT(c.PerfilRef) &&
		referenciaCargoCT(c.OrganizacionRef) && referenciaCargoCT(c.UnidadRef) &&
		referenciaCargoCT(c.AprobacionInstalacionRef) && referenciaCargoCT(c.AprobadorPrincipalID) &&
		c.PrincipalID != c.AprobadorPrincipalID &&
		(c.ActoDelegacionRef == "" || referenciaCargoCT(c.ActoDelegacionRef)) &&
		huellaProvisionCargoCT.MatchString(c.PreimagenSHA256) &&
		huellaProvisionCargoCT.MatchString(c.VersionRolSHA256) &&
		instanteCargoCTCanonico(c.VigenteDesde) && instanteCargoCTCanonico(c.VigenteHasta) &&
		c.VigenteHasta.After(c.VigenteDesde) && !c.VigenteDesde.Before(ahora) &&
		instanteCargoCTCanonico(ahora)
}

func referenciaCargoCT(s string) bool {
	if len(s) < 3 || len(s) > 256 {
		return false
	}
	for _, r := range s {
		if r < 33 || r > 126 || r == '*' || r == '\\' || r == '"' {
			return false
		}
	}
	return true
}

func instanteCargoCTCanonico(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}

// CargarConfiguracionProvisionCargoCT reutiliza el lector de material privado:
// ruta absoluta fuera de Git, sin enlaces y con permisos restringidos.
func CargarConfiguracionProvisionCargoCT(ruta string) (ConfiguracionProvisionCargoCT, error) {
	var c ConfiguracionProvisionCargoCT
	b, err := LeerMaterialProvisionExterna(ruta, 16<<10)
	if err != nil {
		return c, ErrProvisionCargoCT
	}
	defer clear(b)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var sobra any
	if validarClavesJSONUnicas(b) != nil || dec.Decode(&c) != nil || !errors.Is(dec.Decode(&sobra), io.EOF) ||
		!c.validar(time.Now().UTC().Truncate(time.Microsecond)) {
		return ConfiguracionProvisionCargoCT{}, ErrProvisionCargoCT
	}
	return c, nil
}
