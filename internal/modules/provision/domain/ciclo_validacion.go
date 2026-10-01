package domain

import (
	"encoding/hex"
	"strings"
)

const MaximoReclamacionesCiclo = 32

func ValidarCatalogoCausas(c CatalogoCausas) error {
	if !referencia(c.Version) || len(c.Causas) == 0 || len(c.Causas) > 100 {
		return fallo("catalogo_causas_invalido", "catalogo_causas")
	}
	vistas := map[string]bool{}
	for _, causa := range c.Causas {
		if !referencia(causa.Codigo) || !referencia(causa.ReferenciaBase) || vistas[causa.Codigo] {
			return fallo("catalogo_causas_ambiguo", "catalogo_causas")
		}
		vistas[causa.Codigo] = true
	}
	return nil
}

func ValidarReclamacion(c CatalogoCausas, r Reclamacion, v ValoracionCiclo) error {
	if !referencia(r.Referencia) || !referencia(r.EvidenciaRef) {
		return fallo("reclamacion_invalida", "reclamacion")
	}
	if r.VersionValoracion == 0 || r.VersionValoracion != v.Version || !huellaCicloValida(r.HuellaValoracion) || r.HuellaValoracion != v.HuellaRevision {
		return fallo("valoracion_reclamada_no_coincide", "version_valoracion")
	}
	if r.VersionCatalogo != c.Version {
		return fallo("version_catalogo_no_coincide", "version_catalogo")
	}
	for _, causa := range c.Causas {
		if causa.Codigo == r.CausaCodigo {
			return nil
		}
	}
	return fallo("causa_no_catalogada", "causa_codigo")
}

func ValidarDecision(d DecisionRevision, r Reclamacion, actual ValoracionCiclo) error {
	if !referencia(d.Referencia) || !referencia(d.MotivacionRef) || !referencia(d.EvidenciaRef) || d.ReclamacionRef != r.Referencia {
		return fallo("decision_invalida", "decision")
	}
	if d.VersionEsperada == 0 || d.VersionEsperada != actual.Version {
		return fallo("version_esperada_no_coincide", "version_esperada")
	}
	switch d.Tipo {
	case MantenerValoracion:
		if d.EntradaCorregida != nil {
			return fallo("decision_incompatible", "entrada_corregida")
		}
	case RectificarValoracion:
		if d.EntradaCorregida == nil {
			return fallo("rectificacion_sin_instantanea", "entrada_corregida")
		}
		e := *d.EntradaCorregida
		if e.InstantaneaRef == actual.Entrada.InstantaneaRef || e.PuestoRef != actual.Entrada.PuestoRef || e.NivelPuesto != actual.Entrada.NivelPuesto {
			return fallo("rectificacion_incompatible", "entrada_corregida")
		}
		return ValidarEntrada(e)
	default:
		return fallo("tipo_decision_invalido", "tipo")
	}
	return nil
}

// PrimeraValoracion conserva la valoración provisional de ensayo calculada.
func PrimeraValoracion(e Entrada, r Resultado, c CatalogoCausas, revisionInicialRef string) ValoracionCiclo {
	v := ValoracionCiclo{Version: 1, Entrada: CopiarEntradaCiclo(e), Resultado: copiarResultadoCiclo(r), HuellaCatalogoCausas: huella(c), RevisionInicialRef: revisionInicialRef}
	v.HuellaRevision = huella(v)
	return v
}

// RevisarValoracion añade una versión incluso cuando se mantiene el cálculo.
// El resultado procede del motor, nunca de la puntuación remitida en una decisión.
func RevisarValoracion(actual ValoracionCiclo, reclamacion Reclamacion, d DecisionRevision, e Entrada, r Resultado) (ValoracionCiclo, error) {
	if err := ValidarDecision(d, reclamacion, actual); err != nil {
		return ValoracionCiclo{}, err
	}
	if actual.Version >= MaximoReclamacionesCiclo+1 || r.ConvocatoriaRef != actual.Resultado.ConvocatoriaRef || r.PuestoRef != actual.Resultado.PuestoRef || r.HuellaReglas != actual.Resultado.HuellaReglas || r.InstantaneaRef != e.InstantaneaRef {
		return ValoracionCiclo{}, fallo("revision_incompatible", "valoracion")
	}
	if (d.Tipo == MantenerValoracion && (huella(e) != huella(actual.Entrada) || r.HuellaResultado != actual.Resultado.HuellaResultado)) || (d.Tipo == RectificarValoracion && huella(e) != huella(*d.EntradaCorregida)) {
		return ValoracionCiclo{}, fallo("revision_sin_causa_coherente", "valoracion")
	}
	v := ValoracionCiclo{Version: actual.Version + 1, VersionAnterior: actual.Version, HuellaAnterior: actual.HuellaRevision, HuellaCatalogoCausas: actual.HuellaCatalogoCausas, RevisionInicialRef: actual.RevisionInicialRef, HuellaReclamacion: huella(reclamacion), HuellaDecision: huella(d), ReclamacionRef: reclamacion.Referencia, DecisionRef: d.Referencia, Entrada: CopiarEntradaCiclo(e), Resultado: copiarResultadoCiclo(r)}
	v.HuellaRevision = huella(v)
	return v, nil
}

func PrepararResolucion(actual ValoracionCiclo, revisionInicialRef string, reclamaciones []Reclamacion, decisiones []DecisionRevision) ResolucionBorrador {
	decididas := map[string]bool{}
	for _, d := range decisiones {
		decididas[d.ReclamacionRef] = true
	}
	r := ResolucionBorrador{Estado: "borrador", VersionValoracion: actual.Version, HuellaValoracion: actual.HuellaRevision, ReclamacionesPendientes: []string{}, Pendientes: []string{"bases_y_catalogo_sin_contraste", "persistencia_institucional_pendiente", "firma_pendiente", "publicacion_pendiente", "efectos_personal_pendientes"}}
	if revisionInicialRef == "" {
		r.Pendientes = append(r.Pendientes, "revision_inicial_pendiente")
	}
	if !actual.Resultado.Completo {
		r.Pendientes = append(r.Pendientes, "fuentes_pendientes")
	}
	for _, reclamacion := range reclamaciones {
		if !decididas[reclamacion.Referencia] {
			r.ReclamacionesPendientes = append(r.ReclamacionesPendientes, reclamacion.Referencia)
		}
	}
	if len(r.ReclamacionesPendientes) > 0 {
		r.Pendientes = append(r.Pendientes, "reclamaciones_pendientes")
	}
	return r
}

func huellaCicloValida(h string) bool {
	if len(h) != 64 || strings.ToLower(h) != h {
		return false
	}
	_, err := hex.DecodeString(h)
	return err == nil
}
