package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"time"

	cal "vec-diputacion-granada/internal/modules/calendarios/domain"
	calports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

var ErrEnsayoCalendarioInvalido = errors.New("cronos.ensayo_calendario.entrada_invalida")
var ErrEnsayoCalendarioNoDisponible = errors.New("cronos.ensayo_calendario.no_disponible")

type EnsayoCalendarioHistorico struct{ calendarios calports.ConsultaCalendarios }

func NuevoEnsayoCalendarioHistorico(c calports.ConsultaCalendarios) (*EnsayoCalendarioHistorico, error) {
	if c == nil || (reflect.ValueOf(c).Kind() == reflect.Pointer && reflect.ValueOf(c).IsNil()) {
		return nil, ErrEnsayoCalendarioNoDisponible
	}
	return &EnsayoCalendarioHistorico{calendarios: c}, nil
}

// Consultar selecciona una adscripción por fecha y delega el calendario al
// servicio transversal. No resuelve vigencia/conocimiento de datos de Personal.
func (s *EnsayoCalendarioHistorico) Consultar(ctx context.Context, sol ports.SolicitudEnsayoCalendario) (ports.ResultadoEnsayoCalendario, error) {
	var vacio ports.ResultadoEnsayoCalendario
	if s == nil || s.calendarios == nil || ctx == nil {
		return vacio, ErrEnsayoCalendarioNoDisponible
	}
	if err := validarEnsayoCalendario(sol); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	snapshot := sol.Snapshot
	snapshot.ConocidoEn = snapshot.ConocidoEn.UTC()
	snapshot.Adscripciones = append([]ports.AdscripcionEnsayoCalendario(nil), snapshot.Adscripciones...)
	sort.Slice(snapshot.Adscripciones, func(i, j int) bool { return snapshot.Adscripciones[i].Ref < snapshot.Adscripciones[j].Ref })
	r := ports.ResultadoEnsayoCalendario{Demostracion: true, PersonaRef: snapshot.PersonaRef, SnapshotRef: snapshot.Ref, FuenteSnapshot: snapshot.Fuente, HuellaSnapshot: huellaEnsayo(snapshot), Desde: sol.Desde, Hasta: sol.Hasta, ConocidoEn: sol.ConocidoEn.UTC(), Estado: "determinado", Tramos: []ports.TramoEnsayoCalendario{}, Carencias: []string{"programacion_persona_no_disponible", "presencia_y_saldo_fuera_de_alcance"}}
	// Las fronteras de adscripción y año producen tramos semiabiertos adyacentes.
	limites := []cal.FechaCivil{sol.Desde, sol.Hasta}
	for _, a := range snapshot.Adscripciones {
		for _, f := range []cal.FechaCivil{a.Desde, a.Hasta} {
			if sol.Desde.Antes(f) && f.Antes(sol.Hasta) {
				limites = append(limites, f)
			}
		}
	}
	for anio := sol.Desde.Anio() + 1; anio <= sol.Hasta.Anio(); anio++ {
		f, _ := cal.NuevaFechaCivil(anio, 1, 1)
		if f.Antes(sol.Hasta) {
			limites = append(limites, f)
		}
	}
	sort.Slice(limites, func(i, j int) bool { return limites[i].Antes(limites[j]) })
	determinados := 0
	for i := 0; i < len(limites)-1; i++ {
		if limites[i].Igual(limites[i+1]) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return vacio, err
		}
		t := ports.TramoEnsayoCalendario{Desde: limites[i], Hasta: limites[i+1], Estado: "indeterminado", Adscripciones: []ports.AdscripcionEnsayoCalendario{}, Carencias: []string{}}
		for _, a := range snapshot.Adscripciones {
			if !t.Desde.Antes(a.Desde) && t.Desde.Antes(a.Hasta) {
				t.Adscripciones = append(t.Adscripciones, a)
			}
		}
		switch len(t.Adscripciones) {
		case 0:
			t.Carencias = append(t.Carencias, "adscripcion_sin_cobertura")
		case 1:
			c, err := s.calendarios.CalendarioCentro(ctx, calports.SolicitudCalendarioCentro{CentroRef: t.Adscripciones[0].CentroRef, Anio: t.Desde.Anio(), ConocidoEn: r.ConocidoEn})
			if err != nil {
				if ctx.Err() != nil {
					return vacio, ctx.Err()
				}
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return vacio, err
				}
				var cobertura *cal.ErrorCobertura
				if errors.As(err, &cobertura) {
					t.Carencias = append(t.Carencias, "calendario_sin_cobertura")
					t.AmbitosSinCalendario = append([]cal.Ambito(nil), cobertura.Faltan...)
				} else {
					t.Carencias = append(t.Carencias, "calendario_no_disponible")
				}
			} else {
				if !calendarioEnsayoValido(c, t, r.ConocidoEn) {
					return vacio, ErrEnsayoCalendarioNoDisponible
				}
				dias := []cal.Clasificacion{}
				for _, d := range c.Dias {
					if !d.Fecha.Antes(t.Desde) && d.Fecha.Antes(t.Hasta) {
						dias = append(dias, d)
					}
				}
				t.Calendario = &ports.CalendarioTramoEnsayo{CentroRef: c.CentroRef, Anio: c.Anio, MunicipioRef: c.MunicipioRef, Zona: c.Zona, Versiones: c.Versiones, HuellaResultadoAnual: huellaEnsayo(c), Dias: dias}
				t.Estado = "determinado"
				determinados++
			}
		default:
			t.Carencias = append(t.Carencias, "adscripcion_multiple")
		}
		r.Tramos = append(r.Tramos, t)
	}
	if determinados == 0 {
		r.Estado = "indeterminado"
	} else if determinados < len(r.Tramos) {
		r.Estado = "parcial"
	}
	return r, nil
}

func validarEnsayoCalendario(s ports.SolicitudEnsayoCalendario) error {
	if !s.Desde.EsValida() || !s.Hasta.EsValida() || !s.Desde.Antes(s.Hasta) || s.Desde.Anio() < 2000 || s.Hasta.Anio() > 2100 || s.Hasta.Anio()-s.Desde.Anio() > 7 || s.ConocidoEn.IsZero() || s.ConocidoEn.Year() < 2000 || s.ConocidoEn.Year() > 2100 || s.ConocidoEn.Nanosecond()%1000 != 0 {
		return ErrEnsayoCalendarioInvalido
	}
	x := s.Snapshot
	if !x.Sintetico || !cal.ReferenciaValida(x.Ref) || !cal.ReferenciaValida(x.PersonaRef) || !x.ConocidoEn.Equal(s.ConocidoEn) || !fuenteEnsayoValida(x.Fuente) || len(x.Adscripciones) > 256 {
		return ErrEnsayoCalendarioInvalido
	}
	refs := map[string]bool{}
	for _, a := range x.Adscripciones {
		if !cal.ReferenciaValida(a.Ref) || refs[a.Ref] || a.Version < 1 || !cal.ReferenciaValida(a.CentroRef) || !a.Desde.EsValida() || !a.Hasta.EsValida() || !a.Desde.Antes(a.Hasta) || !fuenteEnsayoValida(a.Fuente) {
			return ErrEnsayoCalendarioInvalido
		}
		refs[a.Ref] = true
	}
	return nil
}

func fuenteEnsayoValida(f ports.FuenteEnsayoCalendario) bool {
	b, err := hex.DecodeString(f.SHA256)
	return cal.ReferenciaValida(f.Referencia) && err == nil && len(b) == sha256.Size && hex.EncodeToString(b) == f.SHA256
}

func huellaEnsayo(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func calendarioEnsayoValido(c calports.CalendarioCentro, t ports.TramoEnsayoCalendario, conocido time.Time) bool {
	if c.CentroRef != t.Adscripciones[0].CentroRef || c.Anio != t.Desde.Anio() || !c.ConocidoEn.Equal(conocido) || c.Zona != cal.ZonaOficial || len(c.Versiones) != 4 {
		return false
	}
	ambitos := map[cal.Ambito]bool{
		{Tipo: cal.AmbitoNacional, Ref: cal.ReferenciaNacional}: true,
		{Tipo: cal.AmbitoAutonomico, Ref: c.ComunidadRef}:       true,
		{Tipo: cal.AmbitoLocal, Ref: c.MunicipioRef}:            true,
		{Tipo: cal.AmbitoCentro, Ref: c.CentroRef}:              true,
	}
	for _, v := range c.Versiones {
		if v.Validar() != nil || !v.Procedencia.Sintetica || v.ConocidoDesde.After(conocido) || v.Anio != c.Anio || !ambitos[v.Ambito] {
			return false
		}
		delete(ambitos, v.Ambito)
		if v.Ambito.Tipo == cal.AmbitoCentro && (v.ComunidadRef != c.ComunidadRef || v.MunicipioRef != c.MunicipioRef) {
			return false
		}
		if v.Ambito.Tipo == cal.AmbitoLocal && v.ComunidadRef != c.ComunidadRef {
			return false
		}
	}
	primero, _ := cal.NuevaFechaCivil(c.Anio, 1, 1)
	esperada := primero
	for _, d := range c.Dias {
		if !d.Fecha.Igual(esperada) {
			return false
		}
		esperada, _ = esperada.SumarDias(1)
	}
	return esperada.Anio() == c.Anio+1 && esperada.Mes() == 1 && esperada.Dia() == 1
}
