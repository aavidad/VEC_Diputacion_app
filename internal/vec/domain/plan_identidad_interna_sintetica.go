package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// PlanIdentidadInternaSinteticaV1 declara una persona nueva y una sola cuenta
// ordinaria en una organización existente. No asigna perfiles ni empleo.
// El orden JSON constituye el canon V1, sin salto de línea final.
type PlanIdentidadInternaSinteticaV1 struct {
	Version       uint64                                `json:"version"`
	OperacionRef  string                                `json:"operacion_ref"`
	PreparadoEn   time.Time                             `json:"preparado_en"`
	CaducaEn      time.Time                             `json:"caduca_en"`
	Entorno       string                                `json:"entorno"`
	AlcanceFuente string                                `json:"alcance_fuente"`
	Procedencia   EvidenciaFuentesInicialesAdmin        `json:"procedencia"`
	Organizacion  OrganizacionIdentidadInternaSintetica `json:"organizacion"`
	Persona       PersonaIdentidadInternaSintetica      `json:"persona"`
	FuenteHMAC    EvidenciaFuentesInicialesAdmin        `json:"fuente_hmac"`
}

// OrganizacionIdentidadInternaSintetica fija la versión y procedencia de la
// organización ya existente. La provisión no modifica esa organización.
type OrganizacionIdentidadInternaSintetica struct {
	OrganizacionRef         string    `json:"organizacion_ref"`
	VersionEsperada         uint64    `json:"version_esperada"`
	ProcedenciaHuellaSHA256 string    `json:"procedencia_huella_sha256"`
	VigenteHasta            time.Time `json:"vigente_hasta"`
}
type PersonaIdentidadInternaSintetica struct {
	PersonaRef                  string                         `json:"persona_ref"`
	VersionEsperada             uint64                         `json:"version_esperada"`
	VigenteHasta                time.Time                      `json:"vigente_hasta"`
	OperacionCuentaOrdinariaRef string                         `json:"operacion_cuenta_ordinaria_ref"`
	FuenteTitularidad           EvidenciaFuentesInicialesAdmin `json:"fuente_titularidad"`
}

var ErrPlanIdentidadInternaSintetica = errors.New("plan_identidad_interna_sintetica_invalido")

func (p PlanIdentidadInternaSinteticaV1) Validar() error {
	for _, ref := range []string{p.Persona.PersonaRef, p.Procedencia.Referencia, p.FuenteHMAC.Referencia, p.Persona.FuenteTitularidad.Referencia, p.Persona.OperacionCuentaOrdinariaRef} {
		if len(ref) > 128 {
			return ErrPlanIdentidadInternaSintetica
		}
	}

	if p.Version != 1 || !referenciaFuentesAdmin(p.OperacionRef, "piis_") || len(p.OperacionRef) > 128 || p.Entorno != "desarrollo" || p.AlcanceFuente != "sintetico_declarado" || !instanteFuentesAdmin(p.PreparadoEn) || !instanteFuentesAdmin(p.CaducaEn) || !p.CaducaEn.After(p.PreparadoEn) || p.CaducaEn.After(p.PreparadoEn.Add(24*time.Hour)) || !evidenciaFuentesAdmin(p.Procedencia) || !evidenciaFuentesAdmin(p.FuenteHMAC) || !organizacionFuentesAdmin(p.Organizacion.OrganizacionRef) || !HuellaAdministracionPerfilesValida(p.Organizacion.ProcedenciaHuellaSHA256) || p.Organizacion.ProcedenciaHuellaSHA256 == strings.Repeat("0", 64) || p.Organizacion.VersionEsperada == 0 || p.Organizacion.VersionEsperada > 1<<53-1 || !vigenciaFuentesAdmin(p.Organizacion.VigenteHasta, p.CaducaEn) || !referenciaFuentesAdmin(p.Persona.PersonaRef, "per_") || p.Persona.VersionEsperada != 0 || !vigenciaFuentesAdmin(p.Persona.VigenteHasta, p.CaducaEn) || p.Persona.VigenteHasta.After(p.Organizacion.VigenteHasta) || !referenciaFuentesAdmin(p.Persona.OperacionCuentaOrdinariaRef, "opr_") || !evidenciaFuentesAdmin(p.Persona.FuenteTitularidad) {
		return ErrPlanIdentidadInternaSintetica
	}
	return nil
}
func (p PlanIdentidadInternaSinteticaV1) ValidarEn(t time.Time) error {
	if p.Validar() != nil || t.IsZero() || t.Before(p.PreparadoEn) || !t.Before(p.CaducaEn) {
		return ErrPlanIdentidadInternaSintetica
	}
	return nil
}
func (p PlanIdentidadInternaSinteticaV1) CanonicoYHuella() ([]byte, string, error) {
	if p.Validar() != nil {
		return nil, "", ErrPlanIdentidadInternaSintetica
	}
	b, e := json.Marshal(p)
	if e != nil || len(b) > 64<<10 {
		return nil, "", ErrPlanIdentidadInternaSintetica
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}
