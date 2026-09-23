package ports

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

var patronHuellaAmbitoRRHHV1 = regexp.MustCompile(`^[0-9a-f]{64}$`)
var patronOpacoAmbitoRRHHV1 = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var patronOrganizacionAmbitoRRHHV1 = regexp.MustCompile(`^org_[a-z0-9]{16,80}$`)

var ErrComprobanteAmbitoCorporativoRRHHV1Invalido = errors.New("vec: comprobante de ambito corporativo RRHH invalido")

// DatosComprobanteAmbitoCorporativoRRHHV1 conserva la seleccion exacta de
// ContextoActor. Es una expectativa para SQL, no una concesion por si misma.
type DatosComprobanteAmbitoCorporativoRRHHV1 struct {
	CuentaRef, PersonaRef, PerfilRef, ContextoRef                         string
	CuentaVersion, PersonaVersion, PerfilVersion, ContextoVersion         uint64
	VinculoCorporativoRef                                                 string
	VinculoCorporativoVersion                                             uint64
	OrganizacionRef                                                       string
	OrganizacionVersion                                                   uint64
	OrganizacionProcedenciaRef                                            string
	OrganizacionProcedenciaVersion                                        uint64
	OrganizacionProcedenciaHuellaSHA256, OrganizacionProcedenciaAutoridad string
	VinculoProcedenciaRef                                                 string
	VinculoProcedenciaVersion                                             uint64
	VinculoProcedenciaHuellaSHA256, VinculoProcedenciaAutoridad           string
	Superficie, Uso                                                       string
	VigenteDesde, VigenteHasta                                            time.Time
}

type ComprobanteAmbitoCorporativoRRHHV1 struct {
	datos *DatosComprobanteAmbitoCorporativoRRHHV1
}

func NuevoComprobanteAmbitoCorporativoRRHHV1(d DatosComprobanteAmbitoCorporativoRRHHV1) (ComprobanteAmbitoCorporativoRRHHV1, error) {
	if !datosComprobanteAmbitoRRHHValidos(d) {
		return ComprobanteAmbitoCorporativoRRHHV1{}, ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	return ComprobanteAmbitoCorporativoRRHHV1{datos: &d}, nil
}

func (c ComprobanteAmbitoCorporativoRRHHV1) Datos() (DatosComprobanteAmbitoCorporativoRRHHV1, error) {
	if c.datos == nil || !datosComprobanteAmbitoRRHHValidos(*c.datos) {
		return DatosComprobanteAmbitoCorporativoRRHHV1{}, ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	return *c.datos, nil
}

func (ComprobanteAmbitoCorporativoRRHHV1) MarshalJSON() ([]byte, error) {
	return nil, ErrComprobanteAmbitoCorporativoRRHHV1Invalido
}

// JSONParaSQL exporta una copia para comparacion exacta con el estado actual
// dentro de la transaccion del consumidor. No debe registrarse en logs.
func (c ComprobanteAmbitoCorporativoRRHHV1) JSONParaSQL() ([]byte, error) {
	d, err := c.Datos()
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"cuenta_ref": d.CuentaRef, "cuenta_version": d.CuentaVersion,
		"persona_ref": d.PersonaRef, "persona_version": d.PersonaVersion,
		"perfil_ref": d.PerfilRef, "perfil_version": d.PerfilVersion,
		"contexto_ref": d.ContextoRef, "contexto_version": d.ContextoVersion,
		"vinculo_corporativo_ref":     d.VinculoCorporativoRef,
		"vinculo_corporativo_version": d.VinculoCorporativoVersion,
		"organizacion_ref":            d.OrganizacionRef, "organizacion_version": d.OrganizacionVersion,
		"organizacion_procedencia_ref":           d.OrganizacionProcedenciaRef,
		"organizacion_procedencia_version":       d.OrganizacionProcedenciaVersion,
		"organizacion_procedencia_huella_sha256": d.OrganizacionProcedenciaHuellaSHA256,
		"organizacion_procedencia_autoridad":     d.OrganizacionProcedenciaAutoridad,
		"vinculo_procedencia_ref":                d.VinculoProcedenciaRef,
		"vinculo_procedencia_version":            d.VinculoProcedenciaVersion,
		"vinculo_procedencia_huella_sha256":      d.VinculoProcedenciaHuellaSHA256,
		"vinculo_procedencia_autoridad":          d.VinculoProcedenciaAutoridad,
		"superficie":                             d.Superficie, "uso": d.Uso,
		"vigente_desde": d.VigenteDesde.UTC().Format("2006-01-02T15:04:05.000000Z"),
		"vigente_hasta": d.VigenteHasta.UTC().Format("2006-01-02T15:04:05.000000Z"),
	})
}

func datosComprobanteAmbitoRRHHValidos(d DatosComprobanteAmbitoCorporativoRRHHV1) bool {
	return referenciaAmbitoRRHHV1(d.CuentaRef, "cta_") && referenciaAmbitoRRHHV1(d.PersonaRef, "per_") &&
		referenciaAmbitoRRHHV1(d.PerfilRef, "prf_") && referenciaAmbitoRRHHV1(d.ContextoRef, "vca_") &&
		d.CuentaVersion > 0 && d.PersonaVersion > 0 && d.PerfilVersion > 0 && d.ContextoVersion > 0 &&
		referenciaAmbitoRRHHV1(d.VinculoCorporativoRef, "vcr_") && d.VinculoCorporativoVersion > 0 &&
		patronOrganizacionAmbitoRRHHV1.MatchString(d.OrganizacionRef) && d.OrganizacionVersion > 0 &&
		referenciaAmbitoRRHHV1(d.OrganizacionProcedenciaRef, "prc_") && d.OrganizacionProcedenciaVersion > 0 &&
		patronHuellaAmbitoRRHHV1.MatchString(d.OrganizacionProcedenciaHuellaSHA256) && d.OrganizacionProcedenciaAutoridad == "autoridad_maestra_acreditada" &&
		referenciaAmbitoRRHHV1(d.VinculoProcedenciaRef, "prc_") && d.VinculoProcedenciaVersion > 0 &&
		patronHuellaAmbitoRRHHV1.MatchString(d.VinculoProcedenciaHuellaSHA256) && d.VinculoProcedenciaAutoridad == "autoridad_maestra_acreditada" &&
		d.Superficie == "interna_corporativa" && d.Uso == "consulta_rrhh" &&
		instanteAmbitoRRHHV1Valido(d.VigenteDesde) && instanteAmbitoRRHHV1Valido(d.VigenteHasta) &&
		d.VigenteHasta.After(d.VigenteDesde)
}

func referenciaAmbitoRRHHV1(valor, prefijo string) bool {
	return len(valor) >= len(prefijo)+22 && len(valor) <= len(prefijo)+128 &&
		len(valor) >= len(prefijo) && valor[:len(prefijo)] == prefijo &&
		patronOpacoAmbitoRRHHV1.MatchString(valor[len(prefijo):])
}

func instanteAmbitoRRHHV1Valido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Year() >= 1 && t.Year() <= 9999 && t.Nanosecond()%1_000 == 0
}
