package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const tiempoMaximoAsignacion = 15 * time.Second

type SolicitudAsignarUnidad struct {
	AutenticacionRef  string
	SesionRef         string
	PerfilRef         string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	UnidadRef         string
	ResponsableRef    string
}

type SolicitudReasignarUnidad struct {
	AutenticacionRef        string
	SesionRef               string
	PerfilRef               string
	OrganizacionRef         string
	ExpedienteRef           string
	VersionEsperada         uint64
	ClaveIdempotencia       string
	UnidadRef               string
	ResponsableRef          string
	MotivoReasignacionClave domain.ClaveCatalogo
	Observaciones           string
}

type ServicioAsignacion struct {
	contextos     ports.ResolutorContextoAutorizacionAltaV3
	ambitos       ports.SelladorAmbitoAsignacion
	huellas       ports.DerivadorHuellaAsignacion
	consultas     ports.ConsultorAsignacionIdempotente
	preparaciones ports.PreparadorAsignacionIdempotente
	destinos      ports.ResolutorDestinoAsignacion
	politicas     ports.ResolutorPoliticaAsignacion
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2
	autorizador   puertosvec.AutorizadorSolicitudLigadaV3
	reloj         ports.Reloj
	transaccion   ports.TransaccionAsignaciones
}

func NuevoServicioAsignacion(
	contextos ports.ResolutorContextoAutorizacionAltaV3,
	ambitos ports.SelladorAmbitoAsignacion,
	huellas ports.DerivadorHuellaAsignacion,
	consultas ports.ConsultorAsignacionIdempotente,
	preparaciones ports.PreparadorAsignacionIdempotente,
	destinos ports.ResolutorDestinoAsignacion,
	politicas ports.ResolutorPoliticaAsignacion,
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	reloj ports.Reloj,
	transaccion ports.TransaccionAsignaciones,
) (*ServicioAsignacion, error) {
	if dependenciaNula(contextos) || dependenciaNula(ambitos) ||
		dependenciaNula(huellas) || dependenciaNula(consultas) ||
		dependenciaNula(preparaciones) ||
		dependenciaNula(destinos) || dependenciaNula(politicas) ||
		dependenciaNula(correlaciones) || dependenciaNula(autorizador) ||
		dependenciaNula(reloj) || dependenciaNula(transaccion) {
		return nil, ErrServicioAsignacionInvalido
	}
	return &ServicioAsignacion{
		contextos:     contextos,
		ambitos:       ambitos,
		huellas:       huellas,
		consultas:     consultas,
		preparaciones: preparaciones,
		destinos:      destinos,
		politicas:     politicas,
		correlaciones: correlaciones,
		autorizador:   autorizador,
		reloj:         reloj,
		transaccion:   transaccion,
	}, nil
}

func (s *ServicioAsignacion) Asignar(
	ctx context.Context,
	solicitud SolicitudAsignarUnidad,
) (ports.ReciboAsignacion, error) {
	return s.ejecutar(ctx, datosSolicitudAsignacion{
		operacion:         ports.OperacionRegistrarAsignacion,
		autenticacionRef:  solicitud.AutenticacionRef,
		sesionRef:         solicitud.SesionRef,
		perfilRef:         solicitud.PerfilRef,
		organizacionRef:   solicitud.OrganizacionRef,
		expedienteRef:     solicitud.ExpedienteRef,
		versionEsperada:   solicitud.VersionEsperada,
		claveIdempotencia: solicitud.ClaveIdempotencia,
		unidadRef:         solicitud.UnidadRef,
		responsableRef:    solicitud.ResponsableRef,
	})
}

func (s *ServicioAsignacion) Reasignar(
	ctx context.Context,
	solicitud SolicitudReasignarUnidad,
) (ports.ReciboAsignacion, error) {
	return s.ejecutar(ctx, datosSolicitudAsignacion{
		operacion:          ports.OperacionRegistrarReasignacion,
		autenticacionRef:   solicitud.AutenticacionRef,
		sesionRef:          solicitud.SesionRef,
		perfilRef:          solicitud.PerfilRef,
		organizacionRef:    solicitud.OrganizacionRef,
		expedienteRef:      solicitud.ExpedienteRef,
		versionEsperada:    solicitud.VersionEsperada,
		claveIdempotencia:  solicitud.ClaveIdempotencia,
		unidadRef:          solicitud.UnidadRef,
		responsableRef:     solicitud.ResponsableRef,
		motivoReasignacion: solicitud.MotivoReasignacionClave,
		observaciones:      solicitud.Observaciones,
	})
}

type datosSolicitudAsignacion struct {
	operacion          ports.TipoOperacionAsignacion
	autenticacionRef   string
	sesionRef          string
	perfilRef          string
	organizacionRef    string
	expedienteRef      string
	versionEsperada    uint64
	claveIdempotencia  string
	unidadRef          string
	responsableRef     string
	motivoReasignacion domain.ClaveCatalogo
	observaciones      string
}

func (s *ServicioAsignacion) ejecutar(
	ctx context.Context,
	solicitud datosSolicitudAsignacion,
) (ports.ReciboAsignacion, error) {
	if s == nil || ctx == nil || !s.dependenciasValidas() {
		return ports.ReciboAsignacion{}, ErrServicioAsignacionInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAsignacion{}, err
	}
	ctxOperacion, cancelar := context.WithTimeout(ctx, tiempoMaximoAsignacion)
	defer cancelar()
	instanteInicial := instanteCanonico(s.reloj.Ahora())
	if solicitud.validar(instanteInicial) != nil {
		return ports.ReciboAsignacion{}, ErrSolicitudAsignacionInvalida
	}

	contextoSolicitud := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: solicitud.autenticacionRef,
		SesionRef:        solicitud.sesionRef,
		PerfilRef:        solicitud.perfilRef,
	}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(
		ctxOperacion,
		contextoSolicitud,
	)
	if err != nil {
		return ports.ReciboAsignacion{},
			clasificarFalloAsignacion(ctxOperacion, ErrAsignacionDenegada)
	}
	instanteContexto := instanteCanonico(s.reloj.Ahora())
	if contexto.ValidarPara(contextoSolicitud, instanteContexto) != nil {
		return ports.ReciboAsignacion{}, ErrAsignacionDenegada
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return ports.ReciboAsignacion{}, ErrAsignacionDenegada
	}

	material := solicitud.material(vinculo.PrincipalID, vinculo.PerfilActivoRef)
	ambitos, err := s.ambitos.SellarAmbitoAsignacion(
		ctxOperacion,
		ports.SolicitudSellarAmbitoIdempotencia{
			ClaveIdempotencia: solicitud.claveIdempotencia,
			OrganizacionRef:   solicitud.organizacionRef,
			ActorRef:          vinculo.PrincipalID,
			PerfilRef:         vinculo.PerfilActivoRef,
		},
	)
	if err != nil {
		return ports.ReciboAsignacion{},
			clasificarFalloAsignacion(ctxOperacion, ports.ErrPersistenciaAsignacionNoDisponible)
	}
	huellas, err := s.huellas.DerivarHuellaAsignacion(
		ctxOperacion,
		material,
	)
	if err != nil {
		return ports.ReciboAsignacion{},
			clasificarFalloAsignacion(ctxOperacion, ports.ErrPersistenciaAsignacionNoDisponible)
	}
	preparar := solicitudPrepararAsignacion(
		solicitud,
		material,
		ambitos,
		huellas,
	)
	consulta, err := ports.NuevaSolicitudConsultarAsignacionIdempotente(
		preparar,
	)
	if err != nil {
		return ports.ReciboAsignacion{},
			ports.ErrPreparacionAsignacionInvalida
	}
	estado, encontrado, err := s.consultas.ConsultarAsignacion(
		ctxOperacion,
		consulta,
	)
	if err != nil {
		return ports.ReciboAsignacion{},
			clasificarFalloAsignacion(ctxOperacion, err)
	}
	if err := ctxOperacion.Err(); err != nil {
		return ports.ReciboAsignacion{}, err
	}
	if encontrado {
		preparacion, err := estado.PreparacionPara(consulta)
		if err != nil || preparacion.ValidarPara(preparar) != nil {
			return ports.ReciboAsignacion{},
				ErrResultadoAsignacionNoConfiable
		}
		return s.reconciliar(
			ctxOperacion,
			solicitud,
			material,
			contextoSolicitud,
			contexto,
			preparar,
			preparacion,
			estado,
		)
	}
	if !estado.EsCero() {
		return ports.ReciboAsignacion{},
			ErrResultadoAsignacionNoConfiable
	}
	preparacion, err := s.preparaciones.PrepararAsignacion(
		ctxOperacion,
		preparar,
	)
	if err != nil {
		return ports.ReciboAsignacion{},
			clasificarFalloAsignacion(ctxOperacion, err)
	}
	if preparacion.ValidarPara(preparar) != nil {
		return ports.ReciboAsignacion{},
			ports.ErrPreparacionAsignacionInvalida
	}
	if preparacion.Estado == ports.PreparacionAsignacionConfirmada {
		return ports.ReciboAsignacion{},
			ErrResultadoAsignacionNoConfiable
	}
	return s.confirmar(
		ctxOperacion,
		solicitud,
		material,
		contextoSolicitud,
		contexto,
		preparar,
		preparacion,
	)
}
