package bootstrap

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vec-diputacion-granada/config"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

var errMontajePreparacionBasesV3 = errors.New("seleccion.preparacion_bases.montaje_no_disponible")

const nombreConfiguracionPreparacionBasesV3 = "seleccion-preparacion-bases.json"

// Las referencias privadas delimitan la composición. No sustituyen la
// concesión central vigente ni contienen reglas de aprobación de las bases.
type configuracionPreparacionBasesV3 struct {
	Esquema               string                                `json:"esquema"`
	Ambito                ambitoConfiguracionPreparacionBasesV3 `json:"ambito"`
	MotivoGuardar         core.ReferenciaEntradaCatalogo        `json:"motivo_guardar"`
	MotivoConsultar       core.ReferenciaEntradaCatalogo        `json:"motivo_consultar"`
	MotivoIntentoDenegado core.ReferenciaEntradaCatalogo        `json:"motivo_intento_denegado"`
	MotivoIntentoError    core.ReferenciaEntradaCatalogo        `json:"motivo_intento_error"`
	EscrituraFile         string                                `json:"escritura_dsn_file"`
	LecturaFile           string                                `json:"lectura_dsn_file"`
	Guardar               provisionPreparacionBasesV3           `json:"provision_guardar"`
	Consultar             provisionPreparacionBasesV3           `json:"provision_consultar"`
}

type ambitoConfiguracionPreparacionBasesV3 struct {
	OrganizacionRef  string `json:"organizacion_ref"`
	UnidadGestionRef string `json:"unidad_gestion_ref,omitempty"`
}

func (a ambitoConfiguracionPreparacionBasesV3) dominio() (bolsa.AmbitoOrganizativoConvocatoria, error) {
	return bolsa.NuevoAmbitoOrganizativoConvocatoria(a.OrganizacionRef, a.UnidadGestionRef)
}

type provisionPreparacionBasesV3 struct {
	AprobacionRef   string `json:"aprobacion_ref,omitempty"`
	PreimagenSHA256 string `json:"preimagen_sha256,omitempty"`
}

func (p provisionPreparacionBasesV3) valida() bool {
	if p.AprobacionRef == "" && p.PreimagenSHA256 == "" {
		return true
	}
	h, err := hex.DecodeString(p.PreimagenSHA256)
	return p.AprobacionRef != "" && len(p.AprobacionRef) <= 256 &&
		!strings.ContainsAny(p.AprobacionRef, " \t\r\n") && err == nil && len(h) == 32 &&
		hex.EncodeToString(h) == p.PreimagenSHA256
}

func (c configuracionPreparacionBasesV3) valida() bool {
	_, err := c.Ambito.dominio()
	return c.Esquema == "vec.seleccion.preparacion-bases.servidor.v1" && err == nil &&
		core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoGuardar) &&
		core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoConsultar) &&
		core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoIntentoDenegado) &&
		core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoIntentoError) &&
		c.MotivoIntentoDenegado.CatalogoID == c.MotivoGuardar.CatalogoID &&
		c.MotivoIntentoError.CatalogoID == c.MotivoGuardar.CatalogoID &&
		c.MotivoGuardar.CatalogoID == c.MotivoConsultar.CatalogoID &&
		filepath.IsLocal(c.EscrituraFile) && filepath.IsLocal(c.LecturaFile) &&
		c.EscrituraFile != c.LecturaFile && c.Guardar.valida() && c.Consultar.valida()
}

// La ausencia es opcional. Un archivo presente pero inválido nunca se trata
// como ausencia ni permite activar el consumidor con valores predeterminados.
func leerConfiguracionPreparacionBasesV3(cfg config.Config) (configuracionPreparacionBasesV3, bool, error) {
	var c configuracionPreparacionBasesV3
	if !cfg.DevelopmentEnabledByDoubleKey() || !filepath.IsAbs(cfg.DevelopmentMaterialDir) ||
		dentroDeRepositorioGit(cfg.DevelopmentMaterialDir) || validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir) != nil {
		return c, false, errMontajePreparacionBasesV3
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, nombreConfiguracionPreparacionBasesV3)
	if _, err := os.Lstat(ruta); errors.Is(err, os.ErrNotExist) {
		return c, false, nil
	} else if err != nil {
		return c, false, errMontajePreparacionBasesV3
	}
	b, err := leerFicheroMaterialSeguro(ruta, 64<<10)
	if err != nil {
		return c, true, errMontajePreparacionBasesV3
	}
	defer borrarBytes(b)
	if validarClavesJSONUnicas(b) != nil {
		return c, true, errMontajePreparacionBasesV3
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || !c.valida() {
		return configuracionPreparacionBasesV3{}, true, errMontajePreparacionBasesV3
	}
	return c, true, nil
}
