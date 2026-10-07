package application

import (
	"strings"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

// EnsayarCiclo calcula un provisional y sus revisiones de un solo puesto. No
// registra reclamaciones institucionales ni consulta fuentes administrativas.
func EnsayarCiclo(p ports.PeticionCiclo) (domain.CicloEnsayado, error) {
	fallo := func(codigo, campo string) (domain.CicloEnsayado, error) {
		return domain.CicloEnsayado{}, &domain.Error{Codigo: codigo, Campo: campo}
	}
	if p.SchemaVersion != domain.VersionCiclo || len(p.Reclamaciones) > domain.MaximoReclamacionesCiclo || len(p.Decisiones) > len(p.Reclamaciones) {
		return fallo("ciclo_invalido", "ciclo")
	}
	if p.RevisionInicialRef != "" && (len(p.RevisionInicialRef) > 160 || strings.TrimSpace(p.RevisionInicialRef) != p.RevisionInicialRef) {
		return fallo("revision_inicial_invalida", "revision_inicial_ref")
	}
	if err := domain.ValidarCatalogoCausas(p.CatalogoCausas); err != nil {
		return domain.CicloEnsayado{}, err
	}
	// El límite acumulado evita multiplicar el tamaño de las instantáneas por
	// cada revisión, incluso cuando el caso de uso se invoca sin transporte JSON.
	meritos := len(p.Entrada.Periodos) + len(p.Entrada.Cursos) + len(p.Entrada.Titulaciones)
	for _, d := range p.Decisiones {
		if d.EntradaCorregida != nil {
			meritos += len(d.EntradaCorregida.Periodos) + len(d.EntradaCorregida.Cursos) + len(d.EntradaCorregida.Titulaciones)
		}
		if meritos > 10000 {
			return fallo("ciclo_excesivo", "instantaneas")
		}
	}
	if meritos > 10000 {
		return fallo("ciclo_excesivo", "instantaneas")
	}
	r, err := Simular(p.Configuracion, p.Entrada)
	if err != nil {
		return domain.CicloEnsayado{}, err
	}
	primera := domain.PrimeraValoracion(p.Entrada, r, p.CatalogoCausas, p.RevisionInicialRef)
	salida := domain.CicloEnsayado{SchemaVersion: domain.VersionCiclo, Alcance: "ensayo_sintetico_sin_efectos", Configuracion: domain.CopiarConfiguracionCiclo(p.Configuracion), CatalogoCausas: domain.CatalogoCausas{Version: p.CatalogoCausas.Version, Causas: append([]domain.CausaReclamacion{}, p.CatalogoCausas.Causas...)}, RevisionInicialRef: p.RevisionInicialRef, Valoraciones: []domain.ValoracionCiclo{primera}, Reclamaciones: append([]domain.Reclamacion{}, p.Reclamaciones...), Decisiones: []domain.DecisionRevision{}}
	reclamaciones := map[string]domain.Reclamacion{}
	for _, reclamacion := range p.Reclamaciones {
		if _, repetida := reclamaciones[reclamacion.Referencia]; repetida {
			return fallo("reclamacion_duplicada", "reclamaciones")
		}
		if err := domain.ValidarReclamacion(p.CatalogoCausas, reclamacion, primera); err != nil {
			return domain.CicloEnsayado{}, err
		}
		reclamaciones[reclamacion.Referencia] = reclamacion
	}
	decididas, referencias := map[string]bool{}, map[string]bool{}
	instantaneas := map[string]bool{p.Entrada.InstantaneaRef: true}
	for _, d := range p.Decisiones {
		reclamacion, existe := reclamaciones[d.ReclamacionRef]
		if !existe || decididas[d.ReclamacionRef] || referencias[d.Referencia] {
			return fallo("decision_sin_reclamacion_unica", "decisiones")
		}
		actual := salida.Valoraciones[len(salida.Valoraciones)-1]
		if err := domain.ValidarDecision(d, reclamacion, actual); err != nil {
			return domain.CicloEnsayado{}, err
		}
		e, resultado := actual.Entrada, actual.Resultado
		if d.Tipo == domain.RectificarValoracion {
			e = domain.CopiarEntradaCiclo(*d.EntradaCorregida)
			if instantaneas[e.InstantaneaRef] {
				return fallo("instantanea_reutilizada", "entrada_corregida")
			}
			resultado, err = Simular(p.Configuracion, e)
			if err != nil {
				return domain.CicloEnsayado{}, err
			}
			instantaneas[e.InstantaneaRef] = true
			copia := domain.CopiarEntradaCiclo(e)
			d.EntradaCorregida = &copia
		}
		siguiente, err := domain.RevisarValoracion(actual, reclamacion, d, e, resultado)
		if err != nil {
			return domain.CicloEnsayado{}, err
		}
		salida.Valoraciones = append(salida.Valoraciones, siguiente)
		salida.Decisiones = append(salida.Decisiones, d)
		decididas[d.ReclamacionRef], referencias[d.Referencia] = true, true
	}
	salida.Resolucion = domain.PrepararResolucion(salida.Valoraciones[len(salida.Valoraciones)-1], p.RevisionInicialRef, salida.Reclamaciones, salida.Decisiones)
	return salida, nil
}
