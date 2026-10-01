package domain

import "encoding/json"

func validarMeritos(e Entrada, vistas map[Familia]bool) error {
	cursos := map[string]Curso{}
	evidenciasCursos := map[string]string{}
	for _, c := range e.Cursos {
		if !vistas[Cursos] || !referencia(c.ID) || !referencia(c.EvidenciaRef) || !referencia(c.Tipo) || !c.Horas.EsValido() || c.Horas.Numerador() <= 0 || !c.Fecha.EsValida() {
			return fallo("curso_invalido", "cursos")
		}
		if c.VigenteHasta != nil {
			cmp, err := c.Fecha.Comparar(*c.VigenteHasta)
			if err != nil || cmp >= 0 {
				return fallo("curso_invalido", "vigencia")
			}
		}
		if anterior, ok := cursos[c.ID]; ok && !iguales(anterior, c) {
			return fallo("merito_duplicado_conflictivo", "cursos")
		}
		if id, ok := evidenciasCursos[c.EvidenciaRef]; ok && id != c.ID {
			return fallo("evidencia_duplicada", "cursos")
		}
		evidenciasCursos[c.EvidenciaRef] = c.ID
		cursos[c.ID] = c
	}
	titulos := map[string]Titulo{}
	evidenciasTitulos := map[string]string{}
	for _, t := range e.Titulaciones {
		if !vistas[Titulaciones] || !referencia(t.ID) || !referencia(t.EvidenciaRef) || !referencia(t.Tipo) || !t.Fecha.EsValida() {
			return fallo("titulo_invalido", "titulaciones")
		}
		if anterior, ok := titulos[t.ID]; ok && !iguales(anterior, t) {
			return fallo("merito_duplicado_conflictivo", "titulaciones")
		}
		if id, ok := evidenciasTitulos[t.EvidenciaRef]; ok && id != t.ID {
			return fallo("evidencia_duplicada", "titulaciones")
		}
		evidenciasTitulos[t.EvidenciaRef] = t.ID
		titulos[t.ID] = t
	}
	return nil
}
func iguales(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
