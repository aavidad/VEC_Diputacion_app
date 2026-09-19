package bootstrap

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type origenConsultasContratacionTemporalDesarrollo struct {
	mu                 sync.RWMutex
	autoridad          string
	pagina             ports.PaginaCuadroRRHH
	detalles           map[string]ports.DetalleExpedienteRRHH
	catalogoDesarrollo *catalogosAltaContratacionTemporalDesarrollo
}

func nuevoOrigenConsultasContratacionTemporalDesarrollo(fuenteOrganizacion ...string) *origenConsultasContratacionTemporalDesarrollo {
	rutaFuente := ""
	if len(fuenteOrganizacion) > 0 {
		rutaFuente = fuenteOrganizacion[0]
	}
	catalogos, err := nuevoCatalogoDesarrollo(rutaFuente, "")
	if err != nil {
		catalogos, _ = nuevoCatalogoDesarrollo("", "")
	}
	return nuevoOrigenConsultasConCatalogoDesarrollo(catalogos)
}

func nuevoOrigenConsultasConCatalogoDesarrollo(catalogos *catalogosAltaContratacionTemporalDesarrollo) *origenConsultasContratacionTemporalDesarrollo {
	creadoEn := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	actualizadoEn := time.Date(2026, 9, 2, 9, 30, 0, 0, time.UTC)
	resumen := ports.ResumenExpedienteRRHH{
		ExpedienteRef:   expedienteContratacionTemporalDesarrolloRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		NumeroVisible:   "2026/CT-0001",
		Version:         3,
		FlujoRef:        "flujo:ct:desarrollo",
		FlujoVersion:    1,
		FlujoHuella:     strings.Repeat("d", 64),
		FaseClave:       domain.ClaveFase("analisis"),
		EstadoClave:     domain.EstadoEnCurso,
		CentroRef:       centroAltaContratacionTemporalDesarrollo,
		CategoriaRef:    categoriaAltaContratacionTemporalDesarrollo,
		ModalidadClave:  domain.ClaveCatalogo("interinidad"),
		CreadoEn:        creadoEn,
		ActualizadoEn:   actualizadoEn,
	}
	resumenDetalle := resumen
	resumenDetalle.ModalidadClave = ""
	detalle := ports.DetalleExpedienteRRHH{
		Resumen: resumenDetalle,
		Solicitud: ports.SolicitudOperativaRRHH{
			GrupoSubgrupo: grupoSubgrupoAltaContratacionTemporalDesarrollo,
			MotivoClave:   motivoAltaContratacionTemporalDesarrollo,
			PeriodoInicio: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
			PeriodoFin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		Hitos: []ports.HitoExpedienteRRHH{
			{
				Secuencia: 1, VersionExpediente: 1,
				AccionClave:   domain.ClaveCatalogo("alta"),
				RealizadaEn:   creadoEn,
				FaseDestino:   domain.ClaveFase("solicitud"),
				EstadoOrigen:  domain.EstadoPendiente,
				EstadoDestino: domain.EstadoEnCurso,
			},
			{
				Secuencia: 2, VersionExpediente: 2,
				AccionClave:   domain.ClaveCatalogo("analizar"),
				RealizadaEn:   creadoEn.Add(time.Hour),
				FaseOrigen:    domain.ClaveFase("solicitud"),
				FaseDestino:   domain.ClaveFase("analisis"),
				EstadoOrigen:  domain.EstadoEnCurso,
				EstadoDestino: domain.EstadoEnCurso,
			},
			{
				Secuencia: 3, VersionExpediente: 3,
				AccionClave:   domain.ClaveCatalogo("actualizar"),
				RealizadaEn:   actualizadoEn,
				FaseOrigen:    domain.ClaveFase("analisis"),
				FaseDestino:   domain.ClaveFase("analisis"),
				EstadoOrigen:  domain.EstadoEnCurso,
				EstadoDestino: domain.EstadoEnCurso,
			},
		},
	}
	return &origenConsultasContratacionTemporalDesarrollo{
		autoridad:          AutoridadNoAutoritativa,
		catalogoDesarrollo: catalogos,
		pagina: ports.PaginaCuadroRRHH{
			GeneradaEn:  actualizadoEn.Add(time.Minute),
			Expedientes: []ports.ResumenExpedienteRRHH{resumen},
		},
		detalles: map[string]ports.DetalleExpedienteRRHH{
			expedienteContratacionTemporalDesarrolloRef: detalle,
		},
	}
}

func (o *origenConsultasContratacionTemporalDesarrollo) registrarExpediente(
	expediente domain.Expediente,
) error {
	if o == nil || o.autoridad != AutoridadNoAutoritativa || expediente.Validar() != nil {
		return application.ErrConsultaRRHHNoDisponible
	}
	resumen := ports.ResumenExpedienteRRHH{
		ExpedienteRef: expediente.Referencia, OrganizacionRef: expediente.OrganizacionRef,
		NumeroVisible: expediente.NumeroVisible, Version: expediente.Version,
		FlujoRef: expediente.Flujo.DefinicionRef, FlujoVersion: expediente.Flujo.Version,
		FlujoHuella: expediente.Flujo.HuellaSHA256, FaseClave: expediente.FaseActual,
		EstadoClave: expediente.EstadoActual, CentroRef: expediente.Solicitud.CentroRef,
		CategoriaRef: expediente.Solicitud.CategoriaRef,
		CreadoEn:     expediente.CreadoEn, ActualizadoEn: expediente.ActualizadoEn,
	}
	hitos := make([]ports.HitoExpedienteRRHH, len(expediente.Actuaciones))
	for indice, actuacion := range expediente.Actuaciones {
		hitos[indice] = ports.HitoExpedienteRRHH{
			Secuencia: actuacion.Secuencia, VersionExpediente: actuacion.VersionExpediente,
			AccionClave: actuacion.AccionClave, RealizadaEn: actuacion.RealizadaEn,
			FaseOrigen: actuacion.FaseOrigen, FaseDestino: actuacion.FaseDestino,
			EstadoOrigen: actuacion.EstadoOrigen, EstadoDestino: actuacion.EstadoDestino,
		}
	}
	detalle := ports.DetalleExpedienteRRHH{
		Resumen: resumen,
		Solicitud: ports.SolicitudOperativaRRHH{
			GrupoSubgrupo: expediente.Solicitud.GrupoSubgrupo,
			MotivoClave:   expediente.Solicitud.MotivoClave,
			PeriodoInicio: expediente.Solicitud.Periodo.Inicio,
			PeriodoFin:    expediente.Solicitud.Periodo.Fin,
		},
		Hitos: hitos,
	}
	if expediente.Analisis != nil {
		var coste *ports.ImporteOperativoRRHH
		if expediente.Analisis.CostePrevisto != nil {
			coste = &ports.ImporteOperativoRRHH{
				Centimos: expediente.Analisis.CostePrevisto.Centimos,
				Moneda:   expediente.Analisis.CostePrevisto.Moneda,
			}
		}
		detalle.Analisis = &ports.AnalisisOperativoRRHH{
			ModalidadClave:    expediente.Analisis.ModalidadClave,
			CategoriaRef:      expediente.Analisis.CategoriaRef,
			CausaClave:        expediente.Analisis.CausaClave,
			PeriodoInicio:     expediente.Analisis.Periodo.Inicio,
			PeriodoFin:        expediente.Analisis.Periodo.Fin,
			PorcentajeJornada: expediente.Analisis.PorcentajeJornada,
			ResultadoRC:       expediente.Analisis.ValidacionRC.Resultado,
			CostePrevisto:     coste,
			FuenteCosteRef:    expediente.Analisis.FuenteCosteRef,
			Observaciones:     expediente.Analisis.Observaciones,
		}
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, existe := o.detalles[expediente.Referencia]; existe {
		return application.ErrConsultaRRHHNoDisponible
	}
	o.pagina.Expedientes = append(o.pagina.Expedientes, resumen)
	sort.Slice(o.pagina.Expedientes, func(i, j int) bool {
		primero, segundo := o.pagina.Expedientes[i], o.pagina.Expedientes[j]
		if !primero.ActualizadoEn.Equal(segundo.ActualizadoEn) {
			return primero.ActualizadoEn.After(segundo.ActualizadoEn)
		}
		return primero.ExpedienteRef > segundo.ExpedienteRef
	})
	if o.pagina.GeneradaEn.Before(expediente.ActualizadoEn) {
		o.pagina.GeneradaEn = expediente.ActualizadoEn
	}
	o.detalles[expediente.Referencia] = detalle
	return nil
}

type consultorCuadroContratacionTemporalDesarrollo struct {
	origen *origenConsultasContratacionTemporalDesarrollo
}

func (c *consultorCuadroContratacionTemporalDesarrollo) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	if ctx == nil {
		return ports.PaginaCuadroRRHH{}, application.ErrSolicitudConsultaRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	if c == nil || c.origen == nil {
		return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	c.origen.mu.RLock()
	defer c.origen.mu.RUnlock()
	if c.origen.autoridad != AutoridadNoAutoritativa {
		return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	expedientes := make([]ports.ResumenExpedienteRRHH, 0, len(c.origen.pagina.Expedientes))
	if solicitud.Cursor() == "" {
		for _, resumen := range c.origen.pagina.Expedientes {
			coincide := (solicitud.Texto() == "" || strings.HasPrefix(resumen.NumeroVisible, solicitud.Texto())) &&
				(solicitud.EstadoClave() == "" || solicitud.EstadoClave() == resumen.EstadoClave) &&
				(solicitud.FaseClave() == "" || solicitud.FaseClave() == resumen.FaseClave)
			if coincide {
				expedientes = append(expedientes, resumen)
			}
		}
	}
	if len(expedientes) > int(solicitud.Limite()) {
		return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	return ports.PaginaCuadroRRHH{
		GeneradaEn:  c.origen.pagina.GeneradaEn,
		Expedientes: expedientes,
	}, nil
}

type consultorDetalleContratacionTemporalDesarrollo struct {
	origen *origenConsultasContratacionTemporalDesarrollo
}

func (c *consultorDetalleContratacionTemporalDesarrollo) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	if ctx == nil {
		return ports.DetalleExpedienteRRHH{}, application.ErrSolicitudConsultaRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if c == nil || c.origen == nil {
		return ports.DetalleExpedienteRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	c.origen.mu.RLock()
	defer c.origen.mu.RUnlock()
	detalle, existe := c.origen.detalles[solicitud.ExpedienteRef()]
	if c.origen.autoridad != AutoridadNoAutoritativa || !existe ||
		solicitud.VersionObservada() != 0 &&
			solicitud.VersionObservada() != detalle.Resumen.Version {
		return ports.DetalleExpedienteRRHH{}, application.ErrConsultaRRHHNoObservable
	}
	return detalle.Clonar(), nil
}
