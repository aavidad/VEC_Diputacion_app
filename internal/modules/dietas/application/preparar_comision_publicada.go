package application

import (
	"context"
	"math"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
)

// NuevoPreparadorComisionPublicado monta únicamente la fuente publicada. El
// constructor antiguo queda disponible para leer y validar historia previa.
func NuevoPreparadorComisionPublicado(puntos map[string]ports.CoordenadaRuta, rutas ports.CalculadorRutas,
	tarifas LectorTarifasPublicadas) (*PreparadorComision, error) {
	if len(puntos) == 0 || interfazNula(rutas) || interfazNula(tarifas) {
		return nil, ErrComposicionBorradorInvalida
	}
	copia := make(map[string]ports.CoordenadaRuta, len(puntos))
	for k, v := range puntos {
		copia[k] = v
	}
	return &PreparadorComision{puntos: copia, rutas: rutas, publicadas: tarifas}, nil
}

func (p *PreparadorComision) catalogosPublicados(ctx context.Context, version, vehiculo string,
	fecha time.Time, inicio, fin time.Time) ([]domain.CatalogoLiquidacionPublicado, []domain.CalculoDietasProvisional, int64, error) {
	catalogos := make([]domain.CatalogoLiquidacionPublicado, 0, 3)
	tramos := make([]domain.CalculoDietasProvisional, 0, 3)
	var kmCentimos int64
	for grupo := 1; grupo <= 3; grupo++ {
		c, err := p.publicadas.Consultar(ctx, version, grupo, vehiculo, fecha)
		if err != nil || c.ValidarForma() != nil || !domain.VersionTarifaPublicadaValida(c.VersionTarifaRef) ||
			c.Grupo != grupo || c.PaisISO2 != "ES" || c.Vehiculo != vehiculo {
			return nil, nil, 0, domain.ErrLiquidacionPublicada
		}
		if grupo == 1 {
			version = c.VersionTarifaRef
		} else if c.VersionTarifaRef != version || c.RotuloClave != catalogos[0].RotuloClave ||
			!reflect.DeepEqual(c.ReglaDevengo, catalogos[0].ReglaDevengo) ||
			c.VigenteDesde != catalogos[0].VigenteDesde || c.VigenteHasta != catalogos[0].VigenteHasta {
			return nil, nil, 0, domain.ErrLiquidacionPublicada
		}
		if vehiculo == "propio" {
			var encontrado bool
			for _, r := range c.Reglas {
				if r.Tipo == "kilometraje" && r.Concepto == "vehiculo_propio" {
					if encontrado || r.CentimosPorKM < 1 || r.CentimosPorKM > 99 ||
						(grupo > 1 && r.CentimosPorKM != kmCentimos) {
						return nil, nil, 0, domain.ErrLiquidacionPublicada
					}
					kmCentimos, encontrado = r.CentimosPorKM, true
				}
			}
			if !encontrado {
				return nil, nil, 0, domain.ErrLiquidacionPublicada
			}
		}
		calculado, err := domain.CalcularTramosNacionalesPublicados(inicio, fin, c)
		if err != nil || calculado.AlojamientoTopeCentimos != 0 {
			return nil, nil, 0, domain.ErrLiquidacionPublicada
		}
		catalogos = append(catalogos, c)
		tramos = append(tramos, calculado)
	}
	return catalogos, tramos, kmCentimos, nil
}

func calculoBasePublicado(c domain.CatalogoLiquidacionPublicado, tramos []domain.CalculoDietasProvisional,
	hInicio, hFin string, vehiculo bool) domain.CalculoComision {
	calculo := domain.CalculoComision{Procedencia: "sin_vehiculo_propio", VersionGrafo: "no_aplica", Motor: "no_aplica",
		VersionTarifa: c.VersionTarifaRef, Rotulo: c.RotuloClave, ReglaRef: c.ReglaDevengo.Referencia,
		ReglaHuellaSHA256: c.ReglaDevengo.HuellaSHA256, HoraInicio: hInicio, HoraFin: hFin,
		Kilometros: "0.0000", EURPorKM: "0.0000", TramosRuta: []domain.TramoRutaComision{},
		Rutas: []domain.RutaCalculadaComision{}, VehiculoPropio: vehiculo,
		OpcionesDieta: make([]domain.OpcionDietaComision, 0, 3)}
	for i, tramo := range tramos {
		calculo.OpcionesDieta = append(calculo.OpcionesDieta, domain.OpcionDietaComision{Grupo: i + 1, Calculo: tramo})
	}
	return calculo
}

func (p *PreparadorComision) prepararConTarifaPublicada(ctx context.Context, s ports.SolicitudCrearBorradorPropio,
	inicio, fin, fecha time.Time, ruta ports.ResultadoCalculoRuta) (ports.SolicitudCrearBorradorPropio, error) {
	catalogos, tramos, kmCentimos, err := p.catalogosPublicados(ctx, "", "propio", fecha, inicio, fin)
	if err != nil {
		return s, err
	}
	calculo := calculoBasePublicado(catalogos[0], tramos, s.HoraInicio, s.HoraFin, false)
	calculo.Procedencia, calculo.Motor, calculo.VersionGrafo = "osrm_interno", "OSRM", ruta.VersionGrafo
	calculo.EURPorKM = decimal4Comision(kmCentimos * 100)
	var total int64
	for i, tramo := range ruta.Alternativas[0].Tramos {
		if !finitePositive(tramo.DistanciaMetros) {
			return s, ports.ErrRespuestaMotorRutasInvalida
		}
		decimas := int64(math.Round(tramo.DistanciaMetros * 10))
		if decimas < 1 || decimas > 10000*10000-total {
			return s, ports.ErrRespuestaMotorRutasInvalida
		}
		total += decimas
		calculo.TramosRuta = append(calculo.TramosRuta, domain.TramoRutaComision{OrigenCodigo: s.CodigosRuta[i],
			DestinoCodigo: s.CodigosRuta[i+1], Kilometros: decimal4Comision(decimas)})
	}
	calculo.Kilometros = decimal4Comision(total)
	calculo.ImporteKilometrajeCentimos = (total*kmCentimos + 5000) / 10000
	if calculo.Validar(s.CodigosRuta) != nil {
		return s, domain.ErrCalculoComisionInvalido
	}
	s.Calculo = &calculo
	return s, nil
}

func (p *PreparadorComision) prepararEdicionConTarifaPublicada(ctx context.Context, s ports.SolicitudEditarComisionPropia,
	inicio, fin, fecha time.Time) (ports.SolicitudEditarComisionPropia, error) {
	vehiculo := "sin_vehiculo_propio"
	if s.VehiculoPropio {
		vehiculo = "propio"
	}
	catalogos, tramos, kmCentimos, err := p.catalogosPublicados(ctx, s.VersionTarifaAceptada, vehiculo, fecha, inicio, fin)
	if err != nil {
		return s, err
	}
	calculo := calculoBasePublicado(catalogos[0], tramos, s.HoraInicio, s.HoraFin, s.VehiculoPropio)
	if s.VehiculoPropio {
		calculo.Procedencia, calculo.Motor, calculo.VersionGrafo = "osrm_interno", "OSRM", ""
		calculo.EURPorKM = decimal4Comision(kmCentimos * 100)
	}
	var totalKM int64
	for _, declarada := range s.Rutas {
		coords := make([]ports.CoordenadaRuta, 0, len(declarada.CodigosRuta))
		for _, codigo := range declarada.CodigosRuta {
			punto, ok := p.puntos[codigo]
			if !ok {
				return s, domain.ErrDocumentoComisionInvalido
			}
			coords = append(coords, punto)
		}
		respuesta, err := p.rutas.Calcular(ctx, ports.SolicitudCalculoRuta{Coordenadas: coords, Alternativas: 1})
		if err != nil {
			return s, err
		}
		if respuesta.Motor != "osrm_on_premise" || respuesta.VersionGrafo == "" ||
			len(respuesta.Alternativas) != 1 || len(respuesta.Alternativas[0].Tramos) != len(coords)-1 {
			return s, ports.ErrRespuestaMotorRutasInvalida
		}
		if calculo.VersionGrafo == "" {
			calculo.VersionGrafo = respuesta.VersionGrafo
		} else if calculo.VersionGrafo != respuesta.VersionGrafo {
			return s, ports.ErrRespuestaMotorRutasInvalida
		}
		ruta := domain.RutaCalculadaComision{CodigosRuta: append([]string{}, declarada.CodigosRuta...),
			VersionGrafo: respuesta.VersionGrafo, TramosRuta: make([]domain.TramoRutaComision, 0, len(coords)-1),
			AjusteKilometros: declarada.AjusteKilometros, MotivoAjuste: declarada.MotivoAjuste}
		var baseKM int64
		for i, tramo := range respuesta.Alternativas[0].Tramos {
			if !finitePositive(tramo.DistanciaMetros) {
				return s, ports.ErrRespuestaMotorRutasInvalida
			}
			km := int64(math.Round(tramo.DistanciaMetros * 10))
			if km < 1 || km > 10000*10000-baseKM {
				return s, ports.ErrRespuestaMotorRutasInvalida
			}
			baseKM += km
			ruta.TramosRuta = append(ruta.TramosRuta, domain.TramoRutaComision{OrigenCodigo: declarada.CodigosRuta[i],
				DestinoCodigo: declarada.CodigosRuta[i+1], Kilometros: decimal4Comision(km)})
		}
		ajuste, err := declarada.AjusteEscalado()
		if err != nil {
			return s, err
		}
		finalKM := baseKM + ajuste
		if finalKM < 1 || finalKM > 10000*10000-totalKM {
			return s, domain.ErrDocumentoComisionInvalido
		}
		ruta.KilometrosBase = decimal4Comision(baseKM)
		ruta.KilometrosFinales = decimal4Comision(finalKM)
		ruta.ImporteCentimos = (finalKM*kmCentimos + 5000) / 10000
		calculo.Rutas = append(calculo.Rutas, ruta)
		totalKM += finalKM
		calculo.ImporteKilometrajeCentimos += ruta.ImporteCentimos
	}
	calculo.Kilometros = decimal4Comision(totalKM)
	documento, err := domain.ConstruirDocumentoComision(calculo, s.CodigosRuta, s.Rutas,
		s.VehiculoPropio, s.Asignacion.GrupoDieta, s.TramosAceptados, s.VersionTarifaAceptada, s.Otros)
	if err != nil {
		return s, err
	}
	s.Calculo, s.Documento = &calculo, &documento
	if err := ValidarSolicitudEditar(s); err != nil {
		return s, err
	}
	return s, nil
}
