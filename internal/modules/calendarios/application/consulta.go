// Package application coordina el dominio de Calendarios con su repositorio
// de lectura. Es común a web, CLI u otros módulos consumidores.
package application

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"time"

	"vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/calendarios/ports"
)

var (
	ErrNoDisponible      = errors.New("calendarios: servicio no disponible")
	ErrSolicitudInvalida = errors.New("calendarios: solicitud invalida")
)

// Años consultables: acotan el trabajo y evitan fechas absurdas.
const (
	AnioMinimoConsulta = 2000
	AnioMaximoConsulta = 2100
)

type Servicio struct {
	repositorio ports.RepositorioCalendarios
	reloj       ports.Reloj
}

var _ ports.ConsultaCalendarios = (*Servicio)(nil)

func NuevoServicio(r ports.RepositorioCalendarios, reloj ports.Reloj) (*Servicio, error) {
	if nulo(r) || nulo(reloj) {
		return nil, ErrNoDisponible
	}
	return &Servicio{repositorio: r, reloj: reloj}, nil
}

func (s *Servicio) Centros(ctx context.Context, anio int, conocidoEn time.Time) ([]ports.CentroConCalendario, error) {
	if err := s.preparado(ctx); err != nil {
		return nil, err
	}
	conocido, err := s.conocidoEn(conocidoEn)
	if err != nil || !anioConsultable(anio) {
		return nil, ErrSolicitudInvalida
	}
	versiones, err := s.repositorio.CentrosConCalendario(ctx, anio, conocido)
	if err != nil {
		return nil, opaco(ctx, err)
	}
	centros := make([]ports.CentroConCalendario, 0, len(versiones))
	vistos := map[string]struct{}{}
	for _, v := range versiones {
		if v.Validar() != nil || v.Ambito.Tipo != domain.AmbitoCentro || v.Anio != anio || v.ConocidoDesde.After(conocido) {
			return nil, ErrNoDisponible
		}
		if _, repetido := vistos[v.Ambito.Ref]; repetido {
			return nil, ErrNoDisponible
		}
		vistos[v.Ambito.Ref] = struct{}{}
		centros = append(centros, ports.CentroConCalendario{
			CentroRef: v.Ambito.Ref, Denominacion: v.Denominacion, MunicipioRef: v.MunicipioRef, VersionID: v.ID, Numero: v.Numero,
		})
	}
	sort.Slice(centros, func(i, j int) bool { return centros[i].Denominacion < centros[j].Denominacion })
	return centros, nil
}

// CalendarioCentro compone nacional, autonómico, local del municipio del
// centro y calendario propio del centro. Si falta cualquiera, falla cerrado.
func (s *Servicio) CalendarioCentro(ctx context.Context, sol ports.SolicitudCalendarioCentro) (ports.CalendarioCentro, error) {
	var vacio ports.CalendarioCentro
	if err := s.preparado(ctx); err != nil {
		return vacio, err
	}
	conocido, err := s.conocidoEn(sol.ConocidoEn)
	centro := domain.Ambito{Tipo: domain.AmbitoCentro, Ref: sol.CentroRef}
	if err != nil || !anioConsultable(sol.Anio) || centro.Validar() != nil {
		return vacio, ErrSolicitudInvalida
	}
	propias, err := s.versiones(ctx, sol.Anio, conocido, []domain.Ambito{centro})
	if err != nil {
		return vacio, err
	}
	vc := propias[0].Version
	oficiales, err := s.versiones(ctx, sol.Anio, conocido, []domain.Ambito{
		{Tipo: domain.AmbitoNacional, Ref: domain.ReferenciaNacional},
		{Tipo: domain.AmbitoAutonomico, Ref: vc.ComunidadRef},
		{Tipo: domain.AmbitoLocal, Ref: vc.MunicipioRef},
	})
	if err != nil {
		return vacio, err
	}
	if oficiales[2].Version.ComunidadRef != vc.ComunidadRef {
		return vacio, ErrNoDisponible
	}
	todas := append(oficiales, propias...)
	cal, err := domain.NuevoCalendario([]int{sol.Anio}, todas)
	if err != nil {
		return vacio, ErrNoDisponible
	}
	r := ports.CalendarioCentro{
		CentroRef: centro.Ref, Anio: sol.Anio, ConocidoEn: conocido, Zona: domain.ZonaOficial,
		ComunidadRef: vc.ComunidadRef, MunicipioRef: vc.MunicipioRef,
	}
	for _, v := range todas {
		r.Versiones = append(r.Versiones, v.Version)
	}
	primero, _ := domain.NuevaFechaCivil(sol.Anio, 1, 1)
	for f := primero; f.EsValida() && f.Anio() == sol.Anio; f, _ = f.SumarDias(1) {
		c, err := cal.Clasificar(f)
		if err != nil {
			return vacio, ErrNoDisponible
		}
		r.Dias = append(r.Dias, c)
		r.Resumen.DiasNaturales++
		if c.Laborable {
			r.Resumen.Laborables++
		}
		if !c.InhabilAdministrativo {
			r.Resumen.HabilesAdministrativos++
		}
		if c.FestivoOficial {
			r.Resumen.FestivosOficiales++
		}
		for _, m := range c.Motivos {
			if m.Efecto == domain.EfectoNoLaborable {
				r.Resumen.NoLaborablesCentro++
				break
			}
		}
	}
	return r, nil
}

// CalcularPlazo aplica el artículo 30 de la Ley 39/2015 con los calendarios de
// la sede del órgano y, si se indica, de la residencia del interesado: una
// fecha inhábil en cualquiera de los dos es inhábil (art. 30.6).
func (s *Servicio) CalcularPlazo(ctx context.Context, sol ports.SolicitudCalculoPlazo) (ports.ResultadoCalculoPlazo, error) {
	var vacio ports.ResultadoCalculoPlazo
	if err := s.preparado(ctx); err != nil {
		return vacio, err
	}
	conocido, err := s.conocidoEn(sol.ConocidoEn)
	if err != nil || sol.Inicio.EsValida() == !sol.NotificadoEn.IsZero() {
		return vacio, ErrSolicitudInvalida
	}
	inicio := sol.Inicio
	if !sol.NotificadoEn.IsZero() {
		if inicio, err = domain.FechaCivilDe(sol.NotificadoEn); err != nil {
			return vacio, ErrSolicitudInvalida
		}
	}
	sede := domain.Ambito{Tipo: domain.AmbitoLocal, Ref: sol.MunicipioSede}
	municipios := []domain.Ambito{sede}
	if sol.MunicipioResidencia != "" && sol.MunicipioResidencia != sol.MunicipioSede {
		municipios = append(municipios, domain.Ambito{Tipo: domain.AmbitoLocal, Ref: sol.MunicipioResidencia})
	}
	for _, m := range municipios {
		if m.Validar() != nil {
			return vacio, ErrSolicitudInvalida
		}
	}
	if !anioConsultable(inicio.Anio()) {
		return vacio, ErrSolicitudInvalida
	}
	perezoso := &calendarioPorAnios{ctx: ctx, servicio: s, conocido: conocido, municipios: municipios, anios: map[int]*domain.Calendario{}}
	resultado, err := domain.CalcularPlazo(domain.SolicitudPlazo{Inicio: inicio, Unidad: sol.Unidad, Cantidad: sol.Cantidad}, perezoso)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPlazoInvalido):
			return vacio, ErrSolicitudInvalida
		case errors.Is(err, domain.ErrCalendarioNoCubre), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return vacio, err
		default:
			return vacio, opaco(ctx, err)
		}
	}
	return ports.ResultadoCalculoPlazo{
		ResultadoPlazo: resultado, Unidad: sol.Unidad, Cantidad: sol.Cantidad,
		MunicipioSede: sol.MunicipioSede, MunicipioResidencia: municipioResidencia(municipios),
		ConocidoEn: conocido, Zona: domain.ZonaOficial, VersionesUtilizadas: perezoso.usadas,
	}, nil
}

// calendarioPorAnios carga cada año cuando el cómputo lo necesita: primero
// los calendarios locales, que declaran su comunidad, y después los
// autonómicos correspondientes y el nacional.
type calendarioPorAnios struct {
	ctx        context.Context
	servicio   *Servicio
	conocido   time.Time
	municipios []domain.Ambito
	anios      map[int]*domain.Calendario
	usadas     []domain.VersionCalendario
}

const maximoAniosPorCalculo = 8

func (c *calendarioPorAnios) EsInhabil(f domain.FechaCivil) (bool, []domain.Motivo, error) {
	cal, ok := c.anios[f.Anio()]
	if !ok {
		if len(c.anios) >= maximoAniosPorCalculo || !anioConsultable(f.Anio()) {
			return false, nil, domain.ErrCalculoNoDeterminado
		}
		var err error
		if cal, err = c.cargar(f.Anio()); err != nil {
			return false, nil, err
		}
		c.anios[f.Anio()] = cal
	}
	return cal.EsInhabil(f)
}

func (c *calendarioPorAnios) cargar(anio int) (*domain.Calendario, error) {
	locales, err := c.servicio.versiones(c.ctx, anio, c.conocido, c.municipios)
	if err != nil {
		return nil, err
	}
	ambitos := []domain.Ambito{{Tipo: domain.AmbitoNacional, Ref: domain.ReferenciaNacional}}
	comunidades := map[string]struct{}{}
	for _, l := range locales {
		if _, vista := comunidades[l.Version.ComunidadRef]; !vista {
			comunidades[l.Version.ComunidadRef] = struct{}{}
			ambitos = append(ambitos, domain.Ambito{Tipo: domain.AmbitoAutonomico, Ref: l.Version.ComunidadRef})
		}
	}
	oficiales, err := c.servicio.versiones(c.ctx, anio, c.conocido, ambitos)
	if err != nil {
		return nil, err
	}
	todas := append(oficiales, locales...)
	cal, err := domain.NuevoCalendario([]int{anio}, todas)
	if err != nil {
		return nil, ErrNoDisponible
	}
	for _, v := range todas {
		c.usadas = append(c.usadas, v.Version)
	}
	return cal, nil
}

// versiones devuelve exactamente una versión por ámbito pedido, en el mismo
// orden, o un ErrorCobertura con los que faltan.
func (s *Servicio) versiones(ctx context.Context, anio int, conocido time.Time, ambitos []domain.Ambito) ([]domain.VersionConDias, error) {
	encontradas, err := s.repositorio.VersionesVigentes(ctx, ports.ConsultaVersiones{Anio: anio, Ambitos: ambitos, ConocidoEn: conocido})
	if err != nil {
		return nil, opaco(ctx, err)
	}
	porAmbito := map[string]domain.VersionConDias{}
	for _, v := range encontradas {
		if v.Validar() != nil || v.Version.Anio != anio || v.Version.ConocidoDesde.After(conocido) {
			return nil, ErrNoDisponible
		}
		clave := v.Version.Ambito.Clave()
		if _, repetida := porAmbito[clave]; repetida {
			return nil, ErrNoDisponible
		}
		porAmbito[clave] = v
	}
	resultado := make([]domain.VersionConDias, 0, len(ambitos))
	var faltan []domain.Ambito
	for _, a := range ambitos {
		v, ok := porAmbito[a.Clave()]
		if !ok {
			faltan = append(faltan, a)
			continue
		}
		resultado = append(resultado, v)
		delete(porAmbito, a.Clave())
	}
	if len(porAmbito) != 0 {
		return nil, ErrNoDisponible
	}
	if len(faltan) != 0 {
		return nil, &domain.ErrorCobertura{Anio: anio, Faltan: faltan}
	}
	return resultado, nil
}

func (s *Servicio) preparado(ctx context.Context) error {
	if s == nil || nulo(s.repositorio) || nulo(s.reloj) || ctx == nil {
		return ErrNoDisponible
	}
	return ctx.Err()
}

func (s *Servicio) conocidoEn(pedido time.Time) (time.Time, error) {
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if pedido.IsZero() {
		return ahora, nil
	}
	pedido = pedido.UTC()
	if pedido.After(ahora) || pedido.Year() < AnioMinimoConsulta {
		return time.Time{}, ErrSolicitudInvalida
	}
	return pedido.Truncate(time.Microsecond), nil
}

func municipioResidencia(m []domain.Ambito) string {
	if len(m) > 1 {
		return m[1].Ref
	}
	return ""
}

func anioConsultable(a int) bool { return a >= AnioMinimoConsulta && a <= AnioMaximoConsulta }

func opaco(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, domain.ErrCalendarioNoCubre) {
		return err
	}
	return ErrNoDisponible
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice:
		return r.IsNil()
	}
	return false
}
