package simulacion

import (
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

// EjemploCiclo construye otro caso desde el ejemplo público sintético. Cada
// llamada obtiene datos independientes; no importa el prototipo personal.
func EjemploCiclo() (ports.PeticionCiclo, error) {
	ejemplos, err := Ejemplos()
	if err != nil {
		return ports.PeticionCiclo{}, err
	}
	if len(ejemplos) == 0 {
		return ports.PeticionCiclo{}, &domain.Error{Codigo: "ejemplo_inexistente", Campo: "ciclo"}
	}
	x := ejemplos[0]
	p := ports.PeticionCiclo{SchemaVersion: domain.VersionCiclo, Configuracion: x.Configuracion, Entrada: x.Entrada, CatalogoCausas: domain.CatalogoCausas{Version: "catalogo:ensayo:v1", Causas: []domain.CausaReclamacion{{Codigo: "rectificacion_dato_ensayo", ReferenciaBase: "base:sintetica:revision"}}}, RevisionInicialRef: "revision:sintetica:inicial", Reclamaciones: []domain.Reclamacion{}, Decisiones: []domain.DecisionRevision{}}
	inicial, err := application.EnsayarCiclo(p)
	if err != nil {
		return ports.PeticionCiclo{}, err
	}
	primera := inicial.Valoraciones[0]
	p.Reclamaciones = append(p.Reclamaciones, domain.Reclamacion{Referencia: "reclamacion:sintetica:1", VersionValoracion: primera.Version, HuellaValoracion: primera.HuellaRevision, VersionCatalogo: p.CatalogoCausas.Version, CausaCodigo: p.CatalogoCausas.Causas[0].Codigo, EvidenciaRef: "evidencia:sintetica:reclamacion"})
	corregida := domain.CopiarEntradaCiclo(p.Entrada)
	corregida.InstantaneaRef = "instantanea:sintetica:rectificada"
	// El curso del fixture cambia su condición de acreditación en el ejercicio;
	// ninguna fuente administrativa resulta modificada o acreditada por esto.
	if len(corregida.Cursos) == 0 {
		return ports.PeticionCiclo{}, &domain.Error{Codigo: "ejemplo_incompatible", Campo: "cursos"}
	}
	corregida.Cursos[0].Acreditado = !corregida.Cursos[0].Acreditado
	corregida.Cursos[0].EvidenciaRef = "evidencia:sintetica:curso:rectificado"
	p.Decisiones = append(p.Decisiones, domain.DecisionRevision{Referencia: "decision:sintetica:1", ReclamacionRef: p.Reclamaciones[0].Referencia, VersionEsperada: 1, Tipo: domain.RectificarValoracion, MotivacionRef: "motivacion:sintetica:revision", EvidenciaRef: "evidencia:sintetica:decision", EntradaCorregida: &corregida})
	return p, nil
}
