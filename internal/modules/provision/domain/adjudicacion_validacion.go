package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const MaximoPersonasAdjudicacion = 256
const MaximoVacantesAdjudicacion = 256
const MaximoPreferenciasAdjudicacion = 4096

func huellaAdjudicacionValida(s string) bool {
	if len(s) != 64 || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func referenciaAdjudicacion(s string) bool {
	if !referencia(s) {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ':' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

// ValidarAdjudicacion cierra estructura, procedencia y coherencia de resultado.
// No recalcula méritos ni concede admisión. Todas las entradas son sintéticas.
func ValidarAdjudicacion(c ConfiguracionAdjudicacion, e EntradaAdjudicacion) error {
	if c.SchemaVersion != VersionAdjudicacion || !referenciaAdjudicacion(c.ProcesoRef) || !referenciaAdjudicacion(c.Version) || !referenciaAdjudicacion(c.BasesRef) || !huellaAdjudicacionValida(c.HuellaBases) || !referenciaAdjudicacion(c.PoliticaRef) || !referenciaAdjudicacion(c.PoliticaVersion) || !referenciaAdjudicacion(c.Metodo) {
		return fallo("adjudicacion_configuracion_invalida", "politica_bases")
	}
	if c.Prioridad != "total_descendente" || c.Incompatibilidad != "un_puesto_por_persona" || c.VersionMotorValoracion != VersionMotor || !referenciaAdjudicacion(c.VersionReglasValoracion) || !huellaAdjudicacionValida(c.HuellaReglasValoracion) {
		return fallo("adjudicacion_configuracion_invalida", "prioridad_valoracion")
	}
	if len(c.Desempates) == 0 || len(c.Desempates) > 16 {
		return fallo("adjudicacion_cadena_requerida", "desempates")
	}
	criterios := map[string]bool{}
	for _, d := range c.Desempates {
		if !referenciaAdjudicacion(d.ReglaID) || criterios[d.ReglaID] || (d.Sentido != "mayor" && d.Sentido != "menor") {
			return fallo("adjudicacion_cadena_invalida", "desempates")
		}
		criterios[d.ReglaID] = true
	}
	if !referenciaAdjudicacion(e.UniversoRef) || !referenciaAdjudicacion(e.UniversoVersion) || !e.Sintetico {
		return fallo("adjudicacion_entrada_invalida", "universo_sintetico")
	}
	if len(e.Vacantes) == 0 || len(e.Vacantes) > MaximoVacantesAdjudicacion || e.Solicitudes == nil || len(e.Solicitudes) > MaximoPersonasAdjudicacion {
		return fallo("adjudicacion_universo_invalido", "cardinalidad")
	}
	vacantes := map[string]string{}
	puestos := map[string]bool{}
	for _, v := range e.Vacantes {
		if !referenciaAdjudicacion(v.VacanteRef) || !referenciaAdjudicacion(v.PuestoRef) || vacantes[v.VacanteRef] != "" || puestos[v.PuestoRef] {
			return fallo("adjudicacion_vacante_invalida", "vacantes")
		}
		vacantes[v.VacanteRef] = v.PuestoRef
		puestos[v.PuestoRef] = true // PuestoRef identifica una unidad RPT individual.
	}
	personas, solicitudes := map[string]bool{}, map[string]bool{}
	total := 0
	for i, s := range e.Solicitudes {
		campo := fmt.Sprintf("solicitudes.%d", i)
		if !referenciaAdjudicacion(s.SolicitudRef) || !referenciaAdjudicacion(s.Version) || !referenciaAdjudicacion(s.PersonaRef) || !referenciaAdjudicacion(s.InstantaneaRef) || personas[s.PersonaRef] || solicitudes[s.SolicitudRef] || len(s.Preferencias) == 0 || len(s.Preferencias) > len(e.Vacantes) {
			return fallo("adjudicacion_solicitud_invalida", campo)
		}
		personas[s.PersonaRef], solicitudes[s.SolicitudRef] = true, true
		vistas := map[string]bool{}
		total += len(s.Preferencias)
		if total > MaximoPreferenciasAdjudicacion {
			return fallo("adjudicacion_entrada_excesiva", "preferencias")
		}
		for _, p := range s.Preferencias {
			puesto, ok := vacantes[p.VacanteRef]
			if !ok || vistas[p.VacanteRef] {
				return fallo("adjudicacion_preferencia_invalida", campo)
			}
			vistas[p.VacanteRef] = true
			switch p.Admision {
			case "admitida", "pendiente", "excluida", "renunciada":
			default:
				return fallo("adjudicacion_admision_invalida", campo)
			}
			if p.Valoracion == nil {
				if p.Admision == "admitida" {
					return fallo("adjudicacion_valoracion_requerida", campo)
				}
				continue
			}
			if err := validarValoracionAdjudicacion(c, s, puesto, *p.Valoracion, campo); err != nil {
				return err
			}
		}
	}
	return nil
}

func validarValoracionAdjudicacion(c ConfiguracionAdjudicacion, s SolicitudAdjudicacion, puesto string, r Resultado, campo string) error {
	if r.Estado != "simulacion_local_sin_efectos" || r.VersionMotor != c.VersionMotorValoracion || r.ConvocatoriaRef != c.ProcesoRef || r.VersionReglas != c.VersionReglasValoracion || r.PuestoRef != puesto || r.InstantaneaRef != s.InstantaneaRef || r.HuellaReglas != c.HuellaReglasValoracion || !huellaAdjudicacionValida(r.HuellaEntrada) || !huellaAdjudicacionValida(r.HuellaResultado) {
		return fallo("adjudicacion_valoracion_ajena", campo)
	}
	original := r.HuellaResultado
	r.HuellaResultado = ""
	datos, err := json.Marshal(r)
	if err != nil {
		return fallo("adjudicacion_valoracion_incoherente", campo)
	}
	suma := sha256.Sum256(datos)
	if hex.EncodeToString(suma[:]) != original {
		return fallo("adjudicacion_huella_incoherente", campo)
	}
	if r.Completo != (r.Total != nil) || r.Desglose == nil || len(r.Desglose) > 100 || r.Incidencias == nil || len(r.Incidencias) > 100 || !r.Bruto.EsValido() || !r.MaximoTotal.EsValido() {
		return fallo("adjudicacion_valoracion_incoherente", campo)
	}
	if r.Completo && (len(r.Incidencias) > 0 || !r.Total.EsValido()) {
		return fallo("adjudicacion_valoracion_incoherente", campo)
	}
	reglas := map[string]bool{}
	for _, d := range r.Desglose {
		if !referenciaAdjudicacion(d.ReglaID) || reglas[d.ReglaID] || !d.Resultado.EsValido() || (d.Estado != "calculado" && d.Estado != "pendiente_dato") || r.Completo && d.Estado != "calculado" {
			return fallo("adjudicacion_desglose_incoherente", campo)
		}
		reglas[d.ReglaID] = true
	}
	for _, d := range c.Desempates {
		if !reglas[d.ReglaID] {
			return fallo("adjudicacion_desempate_ausente", campo)
		}
	}
	return nil
}
