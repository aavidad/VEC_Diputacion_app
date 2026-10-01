package domain

import "sort"

// SimularAdjudicacion ensaya exclusivamente la familia seleccionada por datos.
// La aceptación diferida usa rondas simultáneas. Una igualdad en los máximos
// detiene todo el ensayo, sin elegir por referencia ni producir asignaciones.
func SimularAdjudicacion(c ConfiguracionAdjudicacion, e EntradaAdjudicacion) (ResultadoAdjudicacion, error) {
	if err := ValidarAdjudicacion(c, e); err != nil {
		return ResultadoAdjudicacion{}, err
	}
	e = canonizarEntradaAdjudicacion(e)
	r := ResultadoAdjudicacion{
		SchemaVersion: VersionAdjudicacion, Alcance: "simulacion_sintetica_sin_efectos", Estado: "pendiente",
		ProcesoRef: c.ProcesoRef, Version: c.Version, PoliticaRef: c.PoliticaRef, PoliticaVersion: c.PoliticaVersion, Metodo: c.Metodo,
		UniversoRef: e.UniversoRef, UniversoVersion: e.UniversoVersion,
		HuellaConfiguracion: huella(c), HuellaEntrada: huella(e),
		Asignaciones: []AsignacionSimulada{}, PersonasSinAsignacion: []string{}, VacantesSinAsignacion: []string{}, Incidencias: []IncidenciaAdjudicacion{},
	}
	if c.Metodo != MetodoAdjudicacionEnsayo {
		r.Estado = "no_soportado"
		r.Incidencias = append(r.Incidencias, IncidenciaAdjudicacion{Codigo: "metodo_no_soportado"})
		return sellarAdjudicacion(r), nil
	}
	if !e.Cerrado {
		r.Incidencias = append(r.Incidencias, IncidenciaAdjudicacion{Codigo: "universo_no_cerrado"})
	}
	for _, s := range e.Solicitudes {
		for _, p := range s.Preferencias {
			codigo := ""
			if p.Admision == "pendiente" {
				codigo = "admision_pendiente"
			}
			if p.Admision == "admitida" && !p.Valoracion.Completo {
				codigo = "valoracion_pendiente"
			}
			if codigo != "" {
				r.Incidencias = append(r.Incidencias, IncidenciaAdjudicacion{Codigo: codigo, SolicitudRef: s.SolicitudRef, VacanteRef: p.VacanteRef})
			}
		}
	}
	if len(r.Incidencias) > 0 {
		return sellarAdjudicacion(r), nil
	}
	asignadas, incidencias := aceptarDiferidamente(c, e)
	if len(incidencias) > 0 {
		r.Incidencias = incidencias
		return sellarAdjudicacion(r), nil
	}
	r.Estado = "propuesta_simulada"
	personas := map[string]bool{}
	for i, a := range asignadas {
		v := e.Vacantes[i]
		if a.persona < 0 {
			r.VacantesSinAsignacion = append(r.VacantesSinAsignacion, v.VacanteRef)
			continue
		}
		s := e.Solicitudes[a.persona]
		p := s.Preferencias[a.preferencia]
		personas[s.PersonaRef] = true
		r.Asignaciones = append(r.Asignaciones, AsignacionSimulada{s.PersonaRef, s.SolicitudRef, s.Version, v.VacanteRef, v.PuestoRef, a.preferencia + 1, p.Valoracion.HuellaResultado})
	}
	for _, s := range e.Solicitudes {
		if !personas[s.PersonaRef] {
			r.PersonasSinAsignacion = append(r.PersonasSinAsignacion, s.PersonaRef)
		}
	}
	sort.Strings(r.PersonasSinAsignacion)
	return sellarAdjudicacion(r), nil
}

func sellarAdjudicacion(r ResultadoAdjudicacion) ResultadoAdjudicacion {
	r.HuellaResultado = huella(r)
	return r
}

// El orden por referencia solo canoniza transporte y presentación. La
// elección depende exclusivamente de puntos, cadena y preferencias.
func canonizarEntradaAdjudicacion(e EntradaAdjudicacion) EntradaAdjudicacion {
	e.Vacantes = append([]VacanteAdjudicacion(nil), e.Vacantes...)
	e.Solicitudes = append([]SolicitudAdjudicacion{}, e.Solicitudes...)
	sort.Slice(e.Vacantes, func(i, j int) bool { return e.Vacantes[i].VacanteRef < e.Vacantes[j].VacanteRef })
	sort.Slice(e.Solicitudes, func(i, j int) bool { return e.Solicitudes[i].SolicitudRef < e.Solicitudes[j].SolicitudRef })
	return e
}

type candidaturaAdjudicacion struct{ persona, preferencia int }

func aceptarDiferidamente(c ConfiguracionAdjudicacion, e EntradaAdjudicacion) ([]candidaturaAdjudicacion, []IncidenciaAdjudicacion) {
	actuales := make([]candidaturaAdjudicacion, len(e.Vacantes))
	indices := map[string]int{}
	for i, v := range e.Vacantes {
		actuales[i].persona = -1
		indices[v.VacanteRef] = i
	}
	siguientes := make([]int, len(e.Solicitudes))
	ocupadas := make([]bool, len(e.Solicitudes))
	for {
		grupos := make([][]candidaturaAdjudicacion, len(e.Vacantes))
		for i, a := range actuales {
			if a.persona >= 0 {
				grupos[i] = append(grupos[i], a)
			}
		}
		propuestas := 0
		for i, s := range e.Solicitudes {
			if ocupadas[i] {
				continue
			}
			for siguientes[i] < len(s.Preferencias) {
				n := siguientes[i]
				siguientes[i]++
				if s.Preferencias[n].Admision != "admitida" {
					continue
				}
				puesto := indices[s.Preferencias[n].VacanteRef]
				grupos[puesto] = append(grupos[puesto], candidaturaAdjudicacion{i, n})
				propuestas++
				break
			}
		}
		if propuestas == 0 {
			return actuales, nil
		}
		nuevas := make([]candidaturaAdjudicacion, len(e.Vacantes))
		incidencias := []IncidenciaAdjudicacion{}
		for i, grupo := range grupos {
			if len(grupo) == 0 {
				nuevas[i].persona = -1
				continue
			}
			mejor := grupo[0]
			empate := false
			for _, a := range grupo[1:] {
				orden := compararCandidaturasAdjudicacion(c, e, a, mejor)
				if orden > 0 {
					mejor = a
					empate = false
				} else if orden == 0 {
					empate = true
				}
			}
			if empate {
				incidencias = append(incidencias, IncidenciaAdjudicacion{Codigo: "empate_residual", VacanteRef: e.Vacantes[i].VacanteRef})
			}
			nuevas[i] = mejor
		}
		if len(incidencias) > 0 {
			return nil, incidencias
		}
		clear(ocupadas)
		for _, a := range nuevas {
			if a.persona >= 0 {
				ocupadas[a.persona] = true
			}
		}
		actuales = nuevas
	}
}

func compararCandidaturasAdjudicacion(c ConfiguracionAdjudicacion, e EntradaAdjudicacion, a, b candidaturaAdjudicacion) int {
	ra := e.Solicitudes[a.persona].Preferencias[a.preferencia].Valoracion
	rb := e.Solicitudes[b.persona].Preferencias[b.preferencia].Valoracion
	orden, _ := ra.Total.Comparar(*rb.Total) // validado antes de entrar al algoritmo
	if orden != 0 {
		return orden
	}
	for _, d := range c.Desempates {
		pa, pb := puntosReglaAdjudicacion(ra, d.ReglaID), puntosReglaAdjudicacion(rb, d.ReglaID)
		if pa == pb {
			continue
		}
		orden = -1
		if pa > pb {
			orden = 1
		}
		if d.Sentido == "menor" {
			orden = -orden
		}
		return orden
	}
	return 0
}
func puntosReglaAdjudicacion(r *Resultado, id string) int64 {
	for _, d := range r.Desglose {
		if d.ReglaID == id {
			return d.Resultado.Micropuntos()
		}
	}
	return 0 // presencia cerrada por ValidarAdjudicacion
}
