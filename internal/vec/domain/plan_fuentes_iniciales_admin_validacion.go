package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrPlanFuentesInicialesAdminInvalido = errors.New("plan_fuentes_iniciales_admin_invalido")

// Validar coteja el contrato finito de fuentes nuevas. No acredita procedencia,
// LOGIN, preimagen actual, aprobación externa ni material HMAC del proveedor.
func (p PlanFuentesInicialesAdminV1) Validar() error {
	if p.Version != 1 || !referenciaFuentesAdmin(p.OperacionRef, "pfi_") || len(p.OperacionRef) > 128 ||
		p.Entorno != "desarrollo" || p.AlcanceFuente != "sintetico_declarado" ||
		!instanteFuentesAdmin(p.PreparadoEn) || !instanteFuentesAdmin(p.CaducaEn) || !p.CaducaEn.After(p.PreparadoEn) ||
		!evidenciaFuentesAdmin(p.Procedencia) || !evidenciaFuentesAdmin(p.FuenteHMAC) ||
		!organizacionFuentesAdmin(p.Organizacion.OrganizacionRef) || p.Organizacion.VersionEsperada != 0 ||
		!vigenciaFuentesAdmin(p.Organizacion.VigenteHasta, p.CaducaEn) ||
		p.Personas[0].PersonaRef >= p.Personas[1].PersonaRef {
		return ErrPlanFuentesInicialesAdminInvalido
	}
	operaciones := map[string]bool{p.OperacionRef: true}
	for _, persona := range p.Personas {
		if !referenciaFuentesAdmin(persona.PersonaRef, "per_") || persona.VersionEsperada != 0 ||
			!vigenciaFuentesAdmin(persona.VigenteHasta, p.CaducaEn) || persona.VigenteHasta.After(p.Organizacion.VigenteHasta) ||
			!evidenciaFuentesAdmin(persona.FuenteTitularidad) {
			return ErrPlanFuentesInicialesAdminInvalido
		}
		for _, ref := range []string{persona.OperacionCuentaOrdinariaRef, persona.OperacionCuentaPrivilegiadaRef} {
			if !referenciaFuentesAdmin(ref, "opr_") || operaciones[ref] {
				return ErrPlanFuentesInicialesAdminInvalido
			}
			operaciones[ref] = true
		}
	}
	politica := p.PoliticaADMIN
	if !referenciaFuentesAdmin(politica.PoliticaRef, "pga_") || !hostFuentesAdmin(politica.HostADMIN) ||
		!HuellaAdministracionPerfilesValida(politica.CAHuellaSHA256) || politica.CAHuellaSHA256 == strings.Repeat("0", 64) || !HuellaAdministracionPerfilesValida(politica.HuellaAprobacionSHA256) ||
		politica.MaximaEdadRevocacionSegundos == 0 || politica.MaximaEdadRevocacionSegundos > 2147483647 ||
		!vigenciaFuentesAdmin(politica.VigenteHasta, p.CaducaEn) {
		return ErrPlanFuentesInicialesAdminInvalido
	}
	return nil
}

// ValidarEn usa un reloj confiable del invocador; el instante no forma parte de
// la entrada ni cambia el canon. La caducidad es exclusiva.
func (p PlanFuentesInicialesAdminV1) ValidarEn(ahora time.Time) error {
	if p.Validar() != nil || ahora.IsZero() || ahora.Before(p.PreparadoEn) || !ahora.Before(p.CaducaEn) {
		return ErrPlanFuentesInicialesAdminInvalido
	}
	return nil
}

// CanonicoYHuella exige personas ya ordenadas. El orden de campos del DTO es
// parte de V1 y SHA256 cubre todos sus datos, sin salto de línea final.
func (p PlanFuentesInicialesAdminV1) CanonicoYHuella() ([]byte, string, error) {
	if p.Validar() != nil {
		return nil, "", ErrPlanFuentesInicialesAdminInvalido
	}
	b, err := json.Marshal(p)
	if err != nil || len(b) > 64<<10 {
		return nil, "", ErrPlanFuentesInicialesAdminInvalido
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}

func evidenciaFuentesAdmin(e EvidenciaFuentesInicialesAdmin) bool {
	return referenciaFuentesAdmin(e.Referencia, "prc_") && e.Version == 1 && HuellaAdministracionPerfilesValida(e.HuellaSHA256) && e.HuellaSHA256 != strings.Repeat("0", 64)
}
func instanteFuentesAdmin(t time.Time) bool {
	return instanteBootstrap(t) && t.Year() >= 1 && t.Year() <= 9999
}
func vigenciaFuentesAdmin(hasta, caduca time.Time) bool {
	return instanteFuentesAdmin(hasta) && !hasta.Before(caduca)
}
func referenciaFuentesAdmin(ref, prefijo string) bool {
	if !strings.HasPrefix(ref, prefijo) || len(ref) < len(prefijo)+22 || len(ref) > len(prefijo)+128 {
		return false
	}
	for _, c := range ref[len(prefijo):] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func hostFuentesAdmin(host string) bool {
	if len(host) < 4 || len(host) > 253 || !strings.Contains(host, ".") {
		return false
	}
	for _, etiqueta := range strings.Split(host, ".") {
		if etiqueta == "" || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for _, c := range etiqueta {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

// Organización conserva el formato publicado por CA, distinto de operación.
func organizacionFuentesAdmin(ref string) bool {
	if !strings.HasPrefix(ref, "org_") || len(ref) < 20 || len(ref) > 84 {
		return false
	}
	for _, c := range ref[4:] {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
