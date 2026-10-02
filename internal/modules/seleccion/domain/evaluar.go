package domain

import (
	"sort"
)

func Evaluar(c Configuracion, entrada Entrada, meritos map[string]MeritosValorados) (Resultado, error) {
	if err := c.Validar(); err != nil {
		return Resultado{}, err
	}
	if err := entrada.Validar(); err != nil {
		return Resultado{}, err
	}
	r := Resultado{Version: c.Version, ConvocatoriaRef: entrada.ConvocatoriaRef, BasesVersion: entrada.BasesVersion, Modalidad: c.Modalidad, TurnoAcceso: c.TurnoAcceso, Destino: c.Destino, Plazas: c.Plazas, Alcance: "ensayo_sintetico", Estado: "provisional", Causas: []string{}, Solicitudes: []SolicitudResultado{}}
	for _, s := range entrada.Solicitudes {
		v := SolicitudResultado{Referencia: s.Referencia, Nombre: s.Nombre, Acceso: Acceso(s.Requisitos), AccesoDetalle: append([]Requisito(nil), s.Requisitos...), Estado: "apta", Causas: []string{}, Fases: []FaseResultado{}, Propuesta: "sin_propuesta"}
		if v.Acceso == "no_cumple" {
			v.Estado = "no_apta"
			v.Causas = append(v.Causas, "requisito_no_cumplido")
		}
		if v.Acceso == "pendiente" {
			v.Estado = "pendiente"
			v.Causas = append(v.Causas, "requisito_pendiente")
		}
		var ponderado int64
		indeterminada, eliminada := false, v.Acceso == "no_cumple"
		for _, f := range c.Fases {
			p := FaseResultado{Referencia: f.Referencia, Tipo: f.Tipo, MinimoMicropuntos: copiarPuntos(f.MinimoMicropuntos), MaximoMicropuntos: f.MaximoMicropuntos, Peso: f.Peso, Estado: "pendiente", Origen: "prueba_embebida", Reglas: []ReglaValorada{}}
			if f.Tipo == "meritos" {
				p.Origen = "motor_bolsa"
				m, ok := meritos[s.Referencia]
				if !ok {
					return Resultado{}, ErrConfiguracion
				}
				p.PuntosMicropuntos = copiarPuntos(m.PuntosMicropuntos)
				p.Reglas = append(p.Reglas, m.Reglas...)
				p.HuellaMeritosSHA256 = m.HuellaSHA256
			} else if nota := s.Notas[f.Referencia]; nota != nil {
				n := *nota
				p.PuntosMicropuntos = &n
			}
			causa := ""
			switch {
			case f.MinimoMicropuntos == nil:
				causa = "minimo_pendiente"
			case p.PuntosMicropuntos == nil:
				causa = "nota_pendiente"
				if f.Tipo == "meritos" {
					causa = "meritos_pendientes"
				}
			case *p.PuntosMicropuntos < 0 || *p.PuntosMicropuntos > f.MaximoMicropuntos:
				causa = "nota_fuera_rango"
			case *p.PuntosMicropuntos < *f.MinimoMicropuntos:
				causa = "minimo_no_superado"
				p.Estado = "no_superada"
				eliminada = true
			default:
				p.Estado = "superada"
			}
			if causa != "" {
				v.Causas = append(v.Causas, causa)
				if p.Estado == "pendiente" {
					indeterminada = true
				}
			}
			if p.PuntosMicropuntos != nil && *p.PuntosMicropuntos >= 0 && *p.PuntosMicropuntos <= f.MaximoMicropuntos {
				ponderado += *p.PuntosMicropuntos * f.Peso
			}
			v.Fases = append(v.Fases, p)
		}
		if ponderado%100 != 0 {
			indeterminada = true
			v.Causas = append(v.Causas, "precision_pendiente")
		}
		if !indeterminada {
			total := ponderado / 100
			v.TotalMicropuntos = &total
		}
		if eliminada {
			v.Estado = "no_apta"
		} else if indeterminada || v.Acceso == "pendiente" {
			v.Estado = "pendiente"
		}
		r.Solicitudes = append(r.Solicitudes, v)
	}
	clasificar(&r, c)
	return r, nil
}

func clasificar(r *Resultado, c Configuracion) {
	indices := []int{}
	pendiente := false
	for i, s := range r.Solicitudes {
		if s.Estado == "apta" {
			indices = append(indices, i)
		}
		if s.Estado == "pendiente" {
			pendiente = true
		}
	}
	if pendiente {
		r.Estado = "indeterminado"
		r.Causas = append(r.Causas, "clasificacion_pendiente")
		for _, i := range indices {
			r.Solicitudes[i].Propuesta = "pendiente"
			r.Solicitudes[i].Causas = append(r.Solicitudes[i].Causas, "clasificacion_pendiente")
		}
		return
	}
	comparar := func(a, b SolicitudResultado) int {
		if *a.TotalMicropuntos > *b.TotalMicropuntos {
			return -1
		}
		if *a.TotalMicropuntos < *b.TotalMicropuntos {
			return 1
		}
		for _, ref := range c.Desempates {
			var pa, pb int64
			for _, f := range a.Fases {
				if f.Referencia == ref {
					pa = *f.PuntosMicropuntos
				}
			}
			for _, f := range b.Fases {
				if f.Referencia == ref {
					pb = *f.PuntosMicropuntos
				}
			}
			if pa > pb {
				return -1
			}
			if pa < pb {
				return 1
			}
		}
		return 0
	}
	// El orden de presentación de un empate no es posición ni criterio de desempate.
	sort.SliceStable(indices, func(i, j int) bool { return comparar(r.Solicitudes[indices[i]], r.Solicitudes[indices[j]]) < 0 })
	for start := 0; start < len(indices); {
		end := start + 1
		for end < len(indices) && comparar(r.Solicitudes[indices[start]], r.Solicitudes[indices[end]]) == 0 {
			end++
		}
		if end-start > 1 {
			r.Estado = "indeterminado"
			if len(r.Causas) == 0 {
				r.Causas = append(r.Causas, "desempate_pendiente")
			}
			for _, idx := range indices[start:end] {
				s := &r.Solicitudes[idx]
				s.Estado = "empate_pendiente"
				s.Causas = append(s.Causas, "desempate_pendiente")
				if start < c.Plazas && end > c.Plazas {
					s.Propuesta = "pendiente"
				} else if end <= c.Plazas {
					s.Propuesta = "propuesta_provisional"
				}
			}
		} else {
			s := &r.Solicitudes[indices[start]]
			orden := start + 1
			s.Orden = &orden
			if orden <= c.Plazas {
				s.Propuesta = "propuesta_provisional"
			}
		}
		start = end
	}
}
