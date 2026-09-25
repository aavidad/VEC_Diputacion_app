package application

import (
	"context"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
)

type CasoUsoMutacionComision interface {
	EditarPropio(context.Context, ports.IdentidadEfectivaBorrador, ports.SolicitudEditarComisionPropia) (ports.ResultadoBorradorComision, error)
	BorrarPropio(context.Context, ports.IdentidadEfectivaBorrador, ports.SolicitudMutacionComisionPropia) (ports.ResultadoBorradorComision, error)
	EnviarPropio(context.Context, ports.IdentidadEfectivaBorrador, ports.SolicitudMutacionComisionPropia) (ports.ResultadoBorradorComision, error)
}

type CasoUsoConsultaDocumento interface {
	ObtenerDocumentoPropio(context.Context, ports.IdentidadEfectivaBorrador, string) (ports.ResultadoBorradorComision, error)
	ListarDocumentosPropios(context.Context, ports.IdentidadEfectivaBorrador, ports.ConsultaBorradoresPropios) (ports.PaginaBorradoresPropios, error)
}

func (s *ServicioBorradorComision) ObtenerDocumentoPropio(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, ref string) (ports.ResultadoBorradorComision, error) {
	var cero ports.ResultadoBorradorComision
	if ctx == nil || ctx.Err() != nil || !servicioValido(s) || !referenciaComision.MatchString(ref) {
		return cero, ports.ErrBorradorNoDisponible
	}
	if err := identidadValida(identidad, AccionConsultarDocumentoPropio, ref, FinalidadConsultarDocumentoPropio); err != nil {
		return cero, err
	}
	r, ok := s.repositorio.(ports.RepositorioConsultaDocumento)
	if !ok || interfazNula(r) {
		return cero, ports.ErrBorradorNoDisponible
	}
	return r.ObtenerDocumentoPropio(ctx, identidad, ref)
}

func (s *ServicioBorradorComision) ListarDocumentosPropios(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, consulta ports.ConsultaBorradoresPropios) (ports.PaginaBorradoresPropios, error) {
	var cero ports.PaginaBorradoresPropios
	if ctx == nil || ctx.Err() != nil || !servicioValido(s) {
		return cero, ports.ErrBorradorNoDisponible
	}
	if err := identidadValida(identidad, AccionConsultarDocumentoPropio, RecursoMisBorradores, FinalidadConsultarDocumentoPropio); err != nil {
		return cero, err
	}
	if consulta.Limite == 0 {
		consulta.Limite = 20
	}
	if consulta.Limite < 1 || consulta.Limite > 50 || (consulta.Cursor != "" && !referenciaComision.MatchString(consulta.Cursor)) {
		return cero, domain.ErrDocumentoComisionInvalido
	}
	r, ok := s.repositorio.(ports.RepositorioConsultaDocumento)
	if !ok || interfazNula(r) {
		return cero, ports.ErrBorradorNoDisponible
	}
	return r.ListarDocumentosPropios(ctx, identidad, consulta)
}

func ValidarSolicitudEditar(s ports.SolicitudEditarComisionPropia) error {
	if err := ValidarSolicitudEditarBase(s); err != nil {
		return err
	}
	if s.Asignacion == nil || s.Calculo == nil || s.Documento == nil || s.Documento.VehiculoPropio != s.VehiculoPropio || s.Documento.GrupoDieta != s.Asignacion.GrupoDieta || s.Documento.VersionTarifaAceptada != s.VersionTarifaAceptada || s.Documento.Validar(*s.Calculo, s.CodigosRuta) != nil {
		return domain.ErrDocumentoComisionInvalido
	}
	esperado, err := domain.ConstruirDocumentoComision(*s.Calculo, s.CodigosRuta, s.Rutas, s.VehiculoPropio, s.Asignacion.GrupoDieta, s.TramosAceptados, s.VersionTarifaAceptada, s.Otros)
	if err != nil || !reflect.DeepEqual(*s.Documento, esperado) {
		return domain.ErrDocumentoComisionInvalido
	}
	return nil
}

// Base autoriza únicamente el trabajo previo del servidor. El efecto durable
// se emite después con cálculo y documento completos.
func ValidarSolicitudEditarBase(s ports.SolicitudEditarComisionPropia) error {
	if !referenciaDietas(s.Referencia, "dco_") || !claveIdempotenciaDietas.MatchString(s.ClaveIdempotencia) || s.VersionEsperada == 0 || (s.RelacionRef != "" && !referenciaDietas(s.RelacionRef, "rel_")) {
		return domain.ErrDocumentoComisionInvalido
	}
	crear := ports.SolicitudCrearBorradorPropio{ClaveIdempotencia: s.ClaveIdempotencia, FechaInicio: s.FechaInicio, FechaFin: s.FechaFin, HoraInicio: s.HoraInicio, HoraFin: s.HoraFin, Motivo: s.Motivo, CodigosRuta: s.CodigosRuta, RelacionRef: s.RelacionRef}
	if validarSolicitudCrear(crear) != nil || len(s.CodigosRuta) < 2 || (s.Calculo == nil) != (s.Documento == nil) || (s.VehiculoPropio && (len(s.Rutas) < 1 || len(s.Rutas) > 8)) || (!s.VehiculoPropio && len(s.Rutas) != 0) || !domain.VersionTarifaProvisionalValida(s.VersionTarifaAceptada) || len(s.TramosAceptados) > 62 {
		return domain.ErrDocumentoComisionInvalido
	}
	ant := -1
	for _, indice := range s.TramosAceptados {
		if indice <= ant || indice < 0 {
			return domain.ErrDocumentoComisionInvalido
		}
		ant = indice
	}
	if s.Asignacion != nil && !asignacionEnvioValida(*s.Asignacion, s.RelacionRef) {
		return domain.ErrDocumentoComisionInvalido
	}
	for _, ruta := range s.Rutas {
		if ruta.Validar() != nil {
			return domain.ErrDocumentoComisionInvalido
		}
	}
	// D5: toda línea que se guarda lleva tipo catalogado, fecha dentro de la
	// comisión y justificante; las antiguas solo se leen.
	if len(s.Otros) > 32 {
		return domain.ErrDocumentoComisionInvalido
	}
	for _, otro := range s.Otros {
		if otro.ValidarAlta(s.FechaInicio, s.FechaFin) != nil {
			return domain.ErrDocumentoComisionInvalido
		}
	}
	if s.Calculo != nil && (s.Calculo.ValidarDocumento(s.CodigosRuta, s.Rutas, s.VehiculoPropio) != nil || s.Documento.Validar(*s.Calculo, s.CodigosRuta) != nil) {
		return domain.ErrDocumentoComisionInvalido
	}
	return nil
}

func ValidarSolicitudMutacion(s ports.SolicitudMutacionComisionPropia) error {
	if !referenciaDietas(s.Referencia, "dco_") || !claveIdempotenciaDietas.MatchString(s.ClaveIdempotencia) || s.VersionEsperada == 0 || (s.RelacionRef != "" && !referenciaDietas(s.RelacionRef, "rel_")) {
		return domain.ErrDocumentoComisionInvalido
	}
	if s.Asignacion != nil && !asignacionEnvioValida(*s.Asignacion, s.RelacionRef) {
		return domain.ErrDocumentoComisionInvalido
	}
	return nil
}

var huellaAsignacion = regexp.MustCompile(`^[0-9a-f]{64}$`)

func asignacionEnvioValida(a ports.AsignacionDietasAcreditada, relacion string) bool {
	if a.AsignacionRef == "" || a.RelacionRef == "" || a.RelacionRef != relacion || !referenciaDietas(a.PersonaRef, "per_") || a.UnidadRef == "" || a.CentroRef == "" || !referenciaDietas(a.AdministrativoPersonaRef, "per_") || !referenciaDietas(a.ResponsablePersonaRef, "per_") || a.PersonaRef == a.AdministrativoPersonaRef || a.PersonaRef == a.ResponsablePersonaRef || a.AdministrativoPersonaRef == a.ResponsablePersonaRef || a.GrupoDieta < 1 || a.GrupoDieta > 3 || a.Version < 1 || !fechaCivilCanonica(a.VigenteDesde) || a.ReciboRef == "" || a.DecisionRef == "" || a.EfectoRef == "" || !huellaAsignacion.MatchString(a.ConsumoHuellaSHA256) || a.AuditoriaRef == "" || a.RegistradaEn.IsZero() || a.RegistradaEn.Location() != time.UTC {
		return false
	}
	return true
}

func (s *ServicioBorradorComision) PrepararEnvio(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, solicitud ports.SolicitudMutacionComisionPropia) (ports.SolicitudMutacionComisionPropia, error) {
	if ctx == nil || ctx.Err() != nil || !servicioValido(s) || interfazNula(s.asignaciones) || solicitud.Asignacion != nil {
		return solicitud, ports.ErrRelacionNoDisponible
	}
	if err := ValidarSolicitudMutacion(solicitud); err != nil {
		return solicitud, err
	}
	if err := identidadValida(identidad, AccionEnviarBorradorPropio, solicitud.Referencia, FinalidadEnviarBorradorPropio); err != nil {
		return solicitud, err
	}
	if solicitud.RelacionRef == "" {
		solicitud.RelacionRef = identidad.Relacion.RelacionRef
	}
	a, err := s.asignaciones.ConsultarAsignacionParaEnvio(ctx, identidad)
	if err != nil || !asignacionEnvioValida(a, identidad.Relacion.RelacionRef) || a.PersonaRef != identidad.Relacion.PersonaRef || a.UnidadRef != identidad.Relacion.UnidadRef || a.RelacionRef != solicitud.RelacionRef {
		return solicitud, ports.ErrRelacionNoDisponible
	}
	solicitud.Asignacion = &a
	return solicitud, nil
}

func (s *ServicioBorradorComision) PrepararAsignacionEdicion(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, solicitud ports.SolicitudEditarComisionPropia) (ports.SolicitudEditarComisionPropia, error) {
	if ctx == nil || ctx.Err() != nil || !servicioValido(s) || interfazNula(s.asignaciones) || solicitud.Asignacion != nil || solicitud.Calculo != nil || solicitud.Documento != nil {
		return solicitud, ports.ErrRelacionNoDisponible
	}
	if err := ValidarSolicitudEditarBase(solicitud); err != nil {
		return solicitud, err
	}
	if err := identidadValida(identidad, AccionEditarBorradorPropio, solicitud.Referencia, FinalidadEditarBorradorPropio); err != nil {
		return solicitud, err
	}
	if solicitud.RelacionRef == "" {
		solicitud.RelacionRef = identidad.Relacion.RelacionRef
	}
	a, err := s.asignaciones.ConsultarAsignacionParaEnvio(ctx, identidad)
	if err != nil || !asignacionEnvioValida(a, identidad.Relacion.RelacionRef) || a.PersonaRef != identidad.Relacion.PersonaRef || a.UnidadRef != identidad.Relacion.UnidadRef || a.RelacionRef != solicitud.RelacionRef {
		return solicitud, ports.ErrRelacionNoDisponible
	}
	solicitud.Asignacion = &a
	return solicitud, nil
}

func NuevaSolicitudOperacionEditarBorrador(s ports.SolicitudEditarComisionPropia) (ports.SolicitudOperacionBorrador, error) {
	if err := ValidarSolicitudEditar(s); err != nil {
		return ports.SolicitudOperacionBorrador{}, err
	}
	return ports.SolicitudOperacionBorrador{Operacion: ports.OperacionEditarBorrador, Referencia: s.Referencia, RelacionRef: s.RelacionRef, Editar: s}, nil
}

func NuevaSolicitudOperacionPrepararEdicion(s ports.SolicitudEditarComisionPropia) (ports.SolicitudOperacionBorrador, error) {
	if s.Calculo != nil || s.Documento != nil {
		return ports.SolicitudOperacionBorrador{}, domain.ErrDocumentoComisionInvalido
	}
	if err := ValidarSolicitudEditarBase(s); err != nil {
		return ports.SolicitudOperacionBorrador{}, err
	}
	return ports.SolicitudOperacionBorrador{Operacion: ports.OperacionEditarBorrador, Referencia: s.Referencia, RelacionRef: s.RelacionRef, Editar: s}, nil
}

func (s *ServicioBorradorComision) RecuperarEdicionPorClave(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, solicitud ports.SolicitudEditarComisionPropia) (ports.ResultadoBorradorComision, bool, error) {
	var cero ports.ResultadoBorradorComision
	if ctx == nil || ctx.Err() != nil || !servicioValido(s) || solicitud.Calculo != nil || solicitud.Documento != nil {
		return cero, false, ports.ErrBorradorNoDisponible
	}
	if err := ValidarSolicitudEditarBase(solicitud); err != nil {
		return cero, false, err
	}
	if err := identidadValida(identidad, AccionEditarBorradorPropio, solicitud.Referencia, FinalidadEditarBorradorPropio); err != nil {
		return cero, false, err
	}
	r, ok := s.repositorio.(ports.RecuperadorEdicionComision)
	if !ok || interfazNula(r) {
		return cero, false, ports.ErrBorradorNoDisponible
	}
	return r.RecuperarEdicionPorClave(ctx, identidad, solicitud)
}

func NuevaSolicitudOperacionMutarBorrador(op ports.OperacionBorrador, s ports.SolicitudMutacionComisionPropia) (ports.SolicitudOperacionBorrador, error) {
	if op != ports.OperacionBorrarBorrador && op != ports.OperacionEnviarBorrador {
		return ports.SolicitudOperacionBorrador{}, domain.ErrDocumentoComisionInvalido
	}
	if err := ValidarSolicitudMutacion(s); err != nil {
		return ports.SolicitudOperacionBorrador{}, err
	}
	return ports.SolicitudOperacionBorrador{Operacion: op, Referencia: s.Referencia, RelacionRef: s.RelacionRef, Mutacion: s}, nil
}

func (s *ServicioBorradorComision) EditarPropio(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, solicitud ports.SolicitudEditarComisionPropia) (ports.ResultadoBorradorComision, error) {
	var cero ports.ResultadoBorradorComision
	if ctx == nil || ctx.Err() != nil || !servicioValido(s) {
		return cero, ports.ErrBorradorNoDisponible
	}
	if err := ValidarSolicitudEditar(solicitud); err != nil {
		return cero, err
	}
	if err := identidadValida(identidad, AccionEditarBorradorPropio, solicitud.Referencia, FinalidadEditarBorradorPropio); err != nil {
		return cero, err
	}
	if solicitud.RelacionRef != "" && solicitud.RelacionRef != identidad.Relacion.RelacionRef {
		return cero, ports.ErrRelacionNoValida
	}
	a := solicitud.Asignacion
	if interfazNula(s.asignaciones) || a.RelacionRef != identidad.Relacion.RelacionRef || a.PersonaRef != identidad.Relacion.PersonaRef || a.UnidadRef != identidad.Relacion.UnidadRef {
		return cero, ports.ErrRelacionNoDisponible
	}
	r, ok := s.repositorio.(ports.RepositorioMutacionComision)
	if !ok || interfazNula(r) {
		return cero, ports.ErrBorradorNoDisponible
	}
	return r.EditarPropio(ctx, identidad, solicitud)
}

func (s *ServicioBorradorComision) BorrarPropio(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, solicitud ports.SolicitudMutacionComisionPropia) (ports.ResultadoBorradorComision, error) {
	return s.mutarPropio(ctx, identidad, solicitud, AccionBorrarBorradorPropio, FinalidadBorrarBorradorPropio, false)
}

func (s *ServicioBorradorComision) EnviarPropio(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, solicitud ports.SolicitudMutacionComisionPropia) (ports.ResultadoBorradorComision, error) {
	return s.mutarPropio(ctx, identidad, solicitud, AccionEnviarBorradorPropio, FinalidadEnviarBorradorPropio, true)
}

func (s *ServicioBorradorComision) mutarPropio(ctx context.Context, identidad ports.IdentidadEfectivaBorrador, solicitud ports.SolicitudMutacionComisionPropia, accion, finalidad string, enviar bool) (ports.ResultadoBorradorComision, error) {
	var cero ports.ResultadoBorradorComision
	if ctx == nil || ctx.Err() != nil || !servicioValido(s) {
		return cero, ports.ErrBorradorNoDisponible
	}
	if err := ValidarSolicitudMutacion(solicitud); err != nil {
		return cero, err
	}
	if enviar && (interfazNula(s.asignaciones) || solicitud.Asignacion == nil) {
		return cero, ports.ErrRelacionNoDisponible
	}
	if !enviar && solicitud.Asignacion != nil {
		return cero, domain.ErrDocumentoComisionInvalido
	}
	if err := identidadValida(identidad, accion, solicitud.Referencia, finalidad); err != nil {
		return cero, err
	}
	if solicitud.RelacionRef != "" && solicitud.RelacionRef != identidad.Relacion.RelacionRef {
		return cero, ports.ErrRelacionNoValida
	}
	if enviar {
		a := solicitud.Asignacion
		if a.RelacionRef != identidad.Relacion.RelacionRef || a.PersonaRef != identidad.Relacion.PersonaRef || a.UnidadRef != identidad.Relacion.UnidadRef || a.VigenteDesde > identidad.Autorizacion.Revalidacion.FechaReferencia {
			return cero, ports.ErrRelacionNoDisponible
		}
	}
	r, ok := s.repositorio.(ports.RepositorioMutacionComision)
	if !ok || interfazNula(r) {
		return cero, ports.ErrBorradorNoDisponible
	}
	if enviar {
		return r.EnviarPropio(ctx, identidad, solicitud)
	}
	return r.BorrarPropio(ctx, identidad, solicitud)
}

// PrepararEdicion reutiliza el cálculo autoritativo de alta y anexa gastos
// declarados; el navegador nunca puede fijar líneas ni totales calculados.
func (p *PreparadorComision) PrepararEdicion(ctx context.Context, s ports.SolicitudEditarComisionPropia) (ports.SolicitudEditarComisionPropia, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || s.Calculo != nil || s.Documento != nil || s.Asignacion == nil {
		return s, domain.ErrDocumentoComisionInvalido
	}
	if s.HoraInicio == "" {
		s.HoraInicio = "08:00"
	}
	if s.HoraFin == "" {
		s.HoraFin = "18:00"
	}
	if err := ValidarSolicitudEditarBase(s); err != nil {
		return s, err
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
		return s, domain.ErrDocumentoComisionInvalido
	}
	for _, codigo := range s.CodigosRuta {
		if _, ok := p.puntos[codigo]; !ok {
			return s, domain.ErrDocumentoComisionInvalido
		}
	}
	fechaLocal := inicio.In(zona)
	fechaTarifa := time.Date(fechaLocal.Year(), fechaLocal.Month(), fechaLocal.Day(), 0, 0, 0, 0, time.UTC)
	regla, eRegla := p.tarifas.ConsultarRegla(ctx, s.VersionTarifaAceptada, fechaTarifa)
	if eRegla != nil || regla.Validar() != nil || regla.VersionTarifaRef != s.VersionTarifaAceptada {
		return s, domain.ErrTramosProvisionalesNoDisponibles
	}
	calculo := domain.CalculoComision{Procedencia: "sin_vehiculo_propio", VersionGrafo: "no_aplica", Motor: "no_aplica", VersionTarifa: regla.VersionTarifaRef, Rotulo: domain.RotuloTarifaProvisional, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256, HoraInicio: s.HoraInicio, HoraFin: s.HoraFin, Kilometros: "0.0000", TramosRuta: []domain.TramoRutaComision{}, Rutas: []domain.RutaCalculadaComision{}, VehiculoPropio: s.VehiculoPropio, OpcionesDieta: make([]domain.OpcionDietaComision, 0, 3)}
	if s.VehiculoPropio {
		calculo.Procedencia = "osrm_interno"
		calculo.VersionGrafo = ""
		calculo.Motor = "OSRM"
	}
	for grupo := 1; grupo <= 3; grupo++ {
		tarifa, e := p.tarifas.Consultar(ctx, regla.VersionTarifaRef, grupo, "automovil", fechaTarifa)
		if e != nil || tarifa.Dieta.VersionRef != calculo.VersionTarifa {
			return s, domain.ErrTramosProvisionalesNoDisponibles
		}
		tramos, e := domain.CalcularTramosNacionalesProvisionales(inicio, fin, zona, tarifa.Dieta, regla)
		if e != nil {
			return s, e
		}
		if grupo == 1 {
			calculo.EURPorKM = tarifa.EURPorKM
		} else if calculo.EURPorKM != tarifa.EURPorKM {
			return s, domain.ErrTramosProvisionalesNoDisponibles
		}
		calculo.OpcionesDieta = append(calculo.OpcionesDieta, domain.OpcionDietaComision{Grupo: grupo, Calculo: tramos})
	}
	if len(calculo.EURPorKM) != 6 || calculo.EURPorKM[0] != '0' || calculo.EURPorKM[1] != '.' {
		return s, domain.ErrTramosProvisionalesNoDisponibles
	}
	tarifa4, err := strconv.ParseInt(calculo.EURPorKM[:1]+calculo.EURPorKM[2:], 10, 64)
	if err != nil {
		return s, domain.ErrTramosProvisionalesNoDisponibles
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
		respuesta, e := p.rutas.Calcular(ctx, ports.SolicitudCalculoRuta{Coordenadas: coords, Alternativas: 1})
		if e != nil {
			return s, e
		}
		if respuesta.Motor != "osrm_on_premise" || respuesta.VersionGrafo == "" || len(respuesta.Alternativas) != 1 || len(respuesta.Alternativas[0].Tramos) != len(coords)-1 {
			return s, ports.ErrRespuestaMotorRutasInvalida
		}
		if calculo.VersionGrafo == "" {
			calculo.VersionGrafo = respuesta.VersionGrafo
		} else if calculo.VersionGrafo != respuesta.VersionGrafo {
			return s, ports.ErrRespuestaMotorRutasInvalida
		}
		ruta := domain.RutaCalculadaComision{CodigosRuta: append([]string{}, declarada.CodigosRuta...), VersionGrafo: respuesta.VersionGrafo, TramosRuta: make([]domain.TramoRutaComision, 0, len(coords)-1), AjusteKilometros: declarada.AjusteKilometros, MotivoAjuste: declarada.MotivoAjuste}
		var baseKM int64
		for i, tramo := range respuesta.Alternativas[0].Tramos {
			if !finitePositive(tramo.DistanciaMetros) {
				return s, ports.ErrRespuestaMotorRutasInvalida
			}
			km := int64(math.Round(tramo.DistanciaMetros * 10))
			if km < 1 {
				return s, ports.ErrRespuestaMotorRutasInvalida
			}
			baseKM += km
			ruta.TramosRuta = append(ruta.TramosRuta, domain.TramoRutaComision{OrigenCodigo: declarada.CodigosRuta[i], DestinoCodigo: declarada.CodigosRuta[i+1], Kilometros: decimal4Comision(km)})
		}
		ajuste, e := declarada.AjusteEscalado()
		if e != nil {
			return s, e
		}
		finalKM := baseKM + ajuste
		if finalKM < 1 || finalKM > 10000*10000 {
			return s, domain.ErrDocumentoComisionInvalido
		}
		ruta.KilometrosBase = decimal4Comision(baseKM)
		ruta.KilometrosFinales = decimal4Comision(finalKM)
		ruta.ImporteCentimos = (finalKM*tarifa4 + 500000) / 1000000
		calculo.Rutas = append(calculo.Rutas, ruta)
		totalKM += finalKM
		calculo.ImporteKilometrajeCentimos += ruta.ImporteCentimos
	}
	if totalKM > 10000*10000 {
		return s, domain.ErrDocumentoComisionInvalido
	}
	calculo.Kilometros = decimal4Comision(totalKM)
	documento, err := domain.ConstruirDocumentoComision(calculo, s.CodigosRuta, s.Rutas, s.VehiculoPropio, s.Asignacion.GrupoDieta, s.TramosAceptados, s.VersionTarifaAceptada, s.Otros)
	if err != nil {
		return s, err
	}
	s.Calculo, s.Documento = &calculo, &documento
	if err := ValidarSolicitudEditar(s); err != nil {
		return s, err
	}
	return s, nil
}

func solicitudEditarVacia(s ports.SolicitudEditarComisionPropia) bool {
	return reflect.DeepEqual(s, ports.SolicitudEditarComisionPropia{})
}
func solicitudMutacionVacia(s ports.SolicitudMutacionComisionPropia) bool {
	return s == (ports.SolicitudMutacionComisionPropia{})
}

var _ CasoUsoMutacionComision = (*ServicioBorradorComision)(nil)
var _ CasoUsoConsultaDocumento = (*ServicioBorradorComision)(nil)
