package application

import (
	"context"
	"math"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
)

type LectorTarifasComision interface {
	Consultar(context.Context, string, int, string, time.Time) (domain.TarifaComisionProvisional, error)
	ConsultarRegla(context.Context, string, time.Time) (domain.ReglaDevengoProvisional, error)
}

// PreparadorComision sólo toma códigos de localidad y tiempos del navegador.
// Coordenadas, distancias, grafo y cuantías proceden de autoridades del servidor.
type PreparadorComision struct {
	puntos  map[string]ports.CoordenadaRuta
	rutas   ports.CalculadorRutas
	tarifas LectorTarifasComision
}

func NuevoPreparadorComision(puntos map[string]ports.CoordenadaRuta, rutas ports.CalculadorRutas, tarifas LectorTarifasComision) (*PreparadorComision, error) {
	if len(puntos) == 0 || rutas == nil || tarifas == nil {
		return nil, ErrComposicionBorradorInvalida
	}
	copia := make(map[string]ports.CoordenadaRuta, len(puntos))
	for k, v := range puntos {
		copia[k] = v
	}
	return &PreparadorComision{puntos: copia, rutas: rutas, tarifas: tarifas}, nil
}

func (p *PreparadorComision) Preparar(ctx context.Context, s ports.SolicitudCrearBorradorPropio) (ports.SolicitudCrearBorradorPropio, error) {
	if p == nil || ctx == nil || len(s.CodigosRuta) < 2 || len(s.CodigosRuta) > 12 || s.Calculo != nil || validarSolicitudCrear(s) != nil {
		return s, domain.ErrComisionBorradorInvalida
	}
	if s.HoraInicio == "" {
		s.HoraInicio = "08:00"
	}
	if s.HoraFin == "" {
		s.HoraFin = "18:00"
	}
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return s, domain.ErrTramosProvisionalesNoDisponibles
	}
	inicio, err := domain.ResolverInstanteCivil(s.FechaInicio, s.HoraInicio, zona)
	if err != nil {
		return s, err
	}
	fin, err := domain.ResolverInstanteCivil(s.FechaFin, s.HoraFin, zona)
	if err != nil || !fin.After(inicio) {
		return s, domain.ErrComisionBorradorInvalida
	}
	coords := make([]ports.CoordenadaRuta, 0, len(s.CodigosRuta))
	vistos := map[string]bool{}
	for _, codigo := range s.CodigosRuta {
		punto, ok := p.puntos[codigo]
		if !ok || vistos[codigo] {
			return s, domain.ErrComisionBorradorInvalida
		}
		vistos[codigo] = true
		coords = append(coords, punto)
	}
	ruta, err := p.rutas.Calcular(ctx, ports.SolicitudCalculoRuta{Coordenadas: coords, Alternativas: 1})
	if err != nil {
		return s, err
	}
	if len(ruta.Alternativas) != 1 || len(ruta.Alternativas[0].Tramos) != len(coords)-1 || ruta.Motor != "osrm_on_premise" || ruta.VersionGrafo == "" {
		return s, ports.ErrRespuestaMotorRutasInvalida
	}
	fechaLocal := inicio.In(zona)
	fechaTarifa := time.Date(fechaLocal.Year(), fechaLocal.Month(), fechaLocal.Day(), 0, 0, 0, 0, time.UTC)
	regla, err := p.tarifas.ConsultarRegla(ctx, "", fechaTarifa)
	if err != nil || regla.Validar() != nil {
		return s, domain.ErrTramosProvisionalesNoDisponibles
	}
	calculo := domain.CalculoComision{Procedencia: "osrm_interno", VersionGrafo: ruta.VersionGrafo, Motor: "OSRM", VersionTarifa: regla.VersionTarifaRef, Rotulo: domain.RotuloTarifaProvisional, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256, HoraInicio: s.HoraInicio, HoraFin: s.HoraFin, TramosRuta: make([]domain.TramoRutaComision, 0, len(coords)-1), OpcionesDieta: make([]domain.OpcionDietaComision, 0, 3)}
	var total int64
	for i, tramo := range ruta.Alternativas[0].Tramos {
		if !finitePositive(tramo.DistanciaMetros) {
			return s, ports.ErrRespuestaMotorRutasInvalida
		}
		decimas := int64(math.Round(tramo.DistanciaMetros * 10))
		if decimas < 1 {
			return s, ports.ErrRespuestaMotorRutasInvalida
		}
		total += decimas
		calculo.TramosRuta = append(calculo.TramosRuta, domain.TramoRutaComision{OrigenCodigo: s.CodigosRuta[i], DestinoCodigo: s.CodigosRuta[i+1], Kilometros: decimal4Comision(decimas)})
	}
	calculo.Kilometros = decimal4Comision(total)
	for grupo := 1; grupo <= 3; grupo++ {
		tarifa, e := p.tarifas.Consultar(ctx, regla.VersionTarifaRef, grupo, "automovil", fechaTarifa)
		if e != nil || tarifa.Dieta.VersionRef != calculo.VersionTarifa {
			return s, domain.ErrTramosProvisionalesNoDisponibles
		}
		t, e := domain.CalcularTramosNacionalesProvisionales(inicio, fin, zona, tarifa.Dieta, regla)
		if e != nil {
			return s, e
		}
		if grupo == 1 {
			calculo.EURPorKM = tarifa.EURPorKM
		} else if calculo.EURPorKM != tarifa.EURPorKM {
			return s, domain.ErrTramosProvisionalesNoDisponibles
		}
		calculo.OpcionesDieta = append(calculo.OpcionesDieta, domain.OpcionDietaComision{Grupo: grupo, Calculo: t})
	}
	tarifa4, err := strconv.ParseInt(calculo.EURPorKM[:1]+calculo.EURPorKM[2:], 10, 64)
	if err != nil {
		return s, domain.ErrTramosProvisionalesNoDisponibles
	}
	calculo.ImporteKilometrajeCentimos = (total*tarifa4 + 500000) / 1000000
	if calculo.Validar(s.CodigosRuta) != nil {
		return s, domain.ErrCalculoComisionInvalido
	}
	s.Calculo = &calculo
	return s, nil
}

func decimal4Comision(v int64) string {
	return strconv.FormatInt(v/10000, 10) + "." + func() string {
		n := strconv.FormatInt(v%10000, 10)
		for len(n) < 4 {
			n = "0" + n
		}
		return n
	}()
}
func finitePositive(v float64) bool {
	return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) && v < 10000000
}
