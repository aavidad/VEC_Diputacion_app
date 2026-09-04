package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const tiempoMaximoFiscalizacion = 15 * time.Second

var (
	ErrServicioFiscalizacionesInvalido = errors.New(
		"contratacion temporal: servicio de fiscalizaciones invalido",
	)
	ErrSolicitudFiscalizacionInvalida = errors.New(
		"contratacion temporal: solicitud de fiscalizacion invalida",
	)
	ErrFiscalizacionDenegada             = ports.ErrAutorizacionDenegada
	ErrResultadoFiscalizacionNoConfiable = errors.New(
		"contratacion temporal: resultado de fiscalizacion no confiable",
	)
)

type SolicitudRegistrarResultadoFiscalizacion struct {
	AutenticacionRef  string
	SesionRef         string
	PerfilRef         string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	Resultado         domain.ResultadoFiscalizacion
	Observaciones     string
}

func (s SolicitudRegistrarResultadoFiscalizacion) Validar() error {
	if (ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: s.AutenticacionRef,
		SesionRef:        s.SesionRef,
		PerfilRef:        s.PerfilRef,
	}).Validar() != nil || !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		s.VersionEsperada != 5 ||
		!ports.ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		ports.ValidarResultadoFiscalizacion(s.Resultado, s.Observaciones) != nil {
		return ErrSolicitudFiscalizacionInvalida
	}
	return nil
}

type ServicioFiscalizaciones struct {
	contextos     ports.ResolutorContextoAutorizacionAltaV3
	ambitos       ports.SelladorAmbitoFiscalizacion
	huellas       ports.DerivadorHuellaFiscalizacion
	preparaciones ports.PreparadorFiscalizacionIdempotente
	politicas     ports.ResolutorPoliticaFiscalizacion
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2
	autorizador   puertosvec.AutorizadorSolicitudLigadaV3
	reloj         ports.Reloj
	transaccion   ports.TransaccionFiscalizaciones
}

func NuevoServicioFiscalizaciones(
	contextos ports.ResolutorContextoAutorizacionAltaV3,
	ambitos ports.SelladorAmbitoFiscalizacion,
	huellas ports.DerivadorHuellaFiscalizacion,
	preparaciones ports.PreparadorFiscalizacionIdempotente,
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	reloj ports.Reloj,
	transaccion ports.TransaccionFiscalizaciones,
) (*ServicioFiscalizaciones, error) {
	politicas, compatible := contextos.(ports.ResolutorPoliticaFiscalizacion)
	dependencias := []any{
		contextos, ambitos, huellas, preparaciones,
		correlaciones, autorizador, reloj, transaccion,
	}
	for _, dependencia := range dependencias {
		if dependenciaNula(dependencia) {
			return nil, ErrServicioFiscalizacionesInvalido
		}
	}
	if !compatible || dependenciaNula(politicas) {
		return nil, ErrServicioFiscalizacionesInvalido
	}
	return &ServicioFiscalizaciones{
		contextos: contextos, ambitos: ambitos, huellas: huellas,
		preparaciones: preparaciones, politicas: politicas,
		correlaciones: correlaciones, autorizador: autorizador,
		reloj: reloj, transaccion: transaccion,
	}, nil
}

func (s *ServicioFiscalizaciones) Registrar(
	ctx context.Context,
	solicitud SolicitudRegistrarResultadoFiscalizacion,
) (ports.ReciboFiscalizacion, error) {
	if s == nil || ctx == nil || solicitud.Validar() != nil {
		return ports.ReciboFiscalizacion{}, ErrSolicitudFiscalizacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboFiscalizacion{}, err
	}
	ctxOperacion, cancelar := context.WithTimeout(ctx, tiempoMaximoFiscalizacion)
	defer cancelar()

	contextoSolicitud := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: solicitud.AutenticacionRef,
		SesionRef:        solicitud.SesionRef,
		PerfilRef:        solicitud.PerfilRef,
	}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(
		ctxOperacion, contextoSolicitud,
	)
	if err != nil || contexto.ValidarPara(
		contextoSolicitud, instanteCanonico(s.reloj.Ahora()),
	) != nil {
		return ports.ReciboFiscalizacion{}, ErrFiscalizacionDenegada
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return ports.ReciboFiscalizacion{}, ErrFiscalizacionDenegada
	}
	material := ports.MaterialHuellaFiscalizacion{
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		VersionExpediente: solicitud.VersionEsperada,
		ActorRef:          vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef,
		Resultado: solicitud.Resultado, Observaciones: solicitud.Observaciones,
	}
	ambitos, err := s.ambitos.SellarAmbitoFiscalizacion(
		ctxOperacion,
		ports.SolicitudSellarAmbitoIdempotencia{
			ClaveIdempotencia: solicitud.ClaveIdempotencia,
			OrganizacionRef:   solicitud.OrganizacionRef,
			ActorRef:          material.ActorRef, PerfilRef: material.PerfilRef,
		},
	)
	if err != nil {
		return ports.ReciboFiscalizacion{}, clasificarFalloFiscalizacion(ctxOperacion, err)
	}
	huellas, err := s.huellas.DerivarHuellaFiscalizacion(ctxOperacion, material)
	if err != nil {
		return ports.ReciboFiscalizacion{}, clasificarFalloFiscalizacion(ctxOperacion, err)
	}
	preparar := ports.SolicitudPrepararFiscalizacion{
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		AmbitosHMAC:       ambitos, HuellasPeticionHMAC: huellas, Material: material,
	}
	preparacion, err := s.preparaciones.PrepararFiscalizacion(ctxOperacion, preparar)
	if err != nil {
		return ports.ReciboFiscalizacion{}, clasificarFalloFiscalizacion(ctxOperacion, err)
	}
	if preparacion.ValidarPara(preparar) != nil {
		return ports.ReciboFiscalizacion{}, ErrResultadoFiscalizacionNoConfiable
	}
	if preparacion.Estado == ports.PreparacionFiscalizacionConfirmada {
		return *preparacion.ReciboConfirmado, nil
	}

	instantePolitica := instanteCanonico(s.reloj.Ahora())
	solicitudPolitica := nuevaSolicitudPoliticaFiscalizacion(
		material, preparacion.Expediente, instantePolitica,
	)
	politica, err := s.politicas.ResolverPoliticaFiscalizacion(
		ctxOperacion, solicitudPolitica,
	)
	if err != nil || politica.ValidarPara(solicitudPolitica, instantePolitica) != nil {
		return ports.ReciboFiscalizacion{}, ErrFiscalizacionDenegada
	}
	solicitudV3, err := s.nuevaSolicitudAutorizacion(
		ctxOperacion, contexto, material, preparacion, politica,
		preparacion.AmbitoIdempotenciaHMAC,
		preparacion.HuellaPeticionHMAC,
	)
	if err != nil {
		return ports.ReciboFiscalizacion{}, ErrFiscalizacionDenegada
	}
	decisionV3, confirmacionV3, err := s.autorizador.ExigirSolicitudLigadaV3(
		ctxOperacion, solicitudV3, contexto.Resultado,
	)
	instanteEfecto := instanteCanonico(s.reloj.Ahora())
	if err != nil || politica.ValidarPara(solicitudPolitica, instanteEfecto) != nil ||
		!autorizacionV3ValidaEn(solicitudV3, decisionV3, confirmacionV3, instanteEfecto) {
		return ports.ReciboFiscalizacion{}, ErrFiscalizacionDenegada
	}

	faseDestino, estadoDestino := destinoFiscalizacion(material.Resultado)
	retornoRef := ""
	if material.Resultado == domain.FiscalizacionDesfavorable {
		retornoRef = preparacion.Referencias.RetornoRef
	}
	datos := domain.DatosRegistrarFiscalizacion{
		FiscalizacionRef:       preparacion.Referencias.FiscalizacionRef,
		Resultado:              material.Resultado,
		UnidadFiscalizadoraRef: politica.UnidadFiscalizadoraRef,
		Observaciones:          material.Observaciones,
		FiscalizadaEn:          instanteEfecto, RetornoRef: retornoRef,
	}
	expedienteSiguiente, err := preparacion.Expediente.RegistrarFiscalizacion(
		material.VersionExpediente,
		datos,
		domain.DatosActuacion{
			AccionClave: domain.AccionRegistrarFiscalizacion,
			ActorRef:    material.ActorRef, UnidadRef: politica.UnidadFiscalizadoraRef,
			ReciboRef:   preparacion.Referencias.ReciboRef,
			RealizadaEn: instanteEfecto, FaseDestino: faseDestino,
			EstadoDestino: estadoDestino, Observaciones: material.Observaciones,
			DocumentosRef: []string{preparacion.Expediente.InformeJuridico.DocumentoRef},
		},
	)
	if err != nil {
		return ports.ReciboFiscalizacion{}, ErrResultadoFiscalizacionNoConfiable
	}
	recibo, err := s.transaccion.ConfirmarFiscalizacion(
		ctxOperacion,
		ports.OrdenConfirmarFiscalizacion{
			Preparacion: preparacion, Politica: politica,
			ExpedienteSiguiente: expedienteSiguiente,
			Evidencia: ports.EvidenciaAutorizacionFiscalizacion{
				Contexto: contexto, SolicitudV3: solicitudV3,
				DecisionV3: decisionV3, ConfirmacionV3: confirmacionV3,
			},
			InstanteEfecto: instanteEfecto,
		},
	)
	if err != nil {
		return ports.ReciboFiscalizacion{}, clasificarFalloFiscalizacion(ctxOperacion, err)
	}
	if recibo.ValidarParaPreparacion(preparacion) != nil {
		return ports.ReciboFiscalizacion{}, ErrResultadoFiscalizacionNoConfiable
	}
	return recibo, nil
}

// RegistrarResultado conserva la frontera esperada por el adaptador HTTP;
// Registrar es el nombre canónico del caso de uso.
func (s *ServicioFiscalizaciones) RegistrarResultado(
	ctx context.Context,
	solicitud SolicitudRegistrarResultadoFiscalizacion,
) (ports.ReciboFiscalizacion, error) {
	return s.Registrar(ctx, solicitud)
}

func nuevaSolicitudPoliticaFiscalizacion(
	material ports.MaterialHuellaFiscalizacion,
	expediente domain.Expediente,
	instante time.Time,
) ports.SolicitudResolverPoliticaFiscalizacion {
	return ports.SolicitudResolverPoliticaFiscalizacion{
		OrganizacionRef:   material.OrganizacionRef,
		ExpedienteRef:     material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente,
		ActorRef:          material.ActorRef, PerfilRef: material.PerfilRef,
		Resultado: material.Resultado, Observaciones: material.Observaciones,
		FaseActual: expediente.FaseActual, EstadoActual: expediente.EstadoActual,
		UnidadAsignadaRef:      expediente.Asignacion.UnidadRef,
		ResponsableAsignadoRef: expediente.Asignacion.ResponsableRef,
		InformeJuridicoRef:     expediente.InformeJuridico.InformeRef,
		DocumentoInformeRef:    expediente.InformeJuridico.DocumentoRef,
		Instante:               instante,
	}
}

func (s *ServicioFiscalizaciones) nuevaSolicitudAutorizacion(
	ctx context.Context,
	contexto ports.ContextoAutorizacionAltaV3,
	material ports.MaterialHuellaFiscalizacion,
	preparacion ports.PreparacionFiscalizacion,
	politica ports.PoliticaFiscalizacion,
	ambitoActivo string,
	huellaActiva string,
) (dominiovec.SolicitudAutorizacionLigadaV3, error) {
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx, s.correlaciones,
	)
	if err != nil {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	huellaObservaciones := sha256.Sum256([]byte(material.Observaciones))
	anterior := preparacion.Expediente
	return dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo:          politica.MotivoAutorizacion,
			Accion:                    ports.AccionRegistrarFiscalizacion,
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: material.ExpedienteRef,
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoFiscalizacion,
				Ambitos: map[string]string{
					"organizacion_ref": material.OrganizacionRef,
					"expediente_ref":   material.ExpedienteRef,
					"fase_previa":      string(anterior.FaseActual),
					"estado_previo":    string(anterior.EstadoActual),
				},
				Atributos: map[string]string{
					"version_expediente":          strconv.FormatUint(material.VersionExpediente, 10),
					"resultado":                   string(material.Resultado),
					"observaciones_huella_sha256": hex.EncodeToString(huellaObservaciones[:]),
					"informe_juridico_ref":        anterior.InformeJuridico.InformeRef,
					"documento_informe_ref":       anterior.InformeJuridico.DocumentoRef,
					"unidad_asignada_ref":         anterior.Asignacion.UnidadRef,
					"responsable_asignado_ref":    anterior.Asignacion.ResponsableRef,
					"unidad_fiscalizadora_ref":    politica.UnidadFiscalizadoraRef,
					"politica_ref":                politica.DefinicionRef,
					"politica_version":            strconv.FormatUint(politica.DefinicionVersion, 10),
					"politica_huella_sha256":      politica.DefinicionHuellaSHA256,
					"ambito_idempotencia_hmac":    ambitoActivo,
					"huella_peticion_hmac":        huellaActiva,
				},
			},
			Finalidad: string(politica.Finalidad), Correlacion: correlacion,
		},
	)
}

func destinoFiscalizacion(
	resultado domain.ResultadoFiscalizacion,
) (domain.ClaveFase, domain.EstadoOperativo) {
	if resultado == domain.FiscalizacionDesfavorable {
		return domain.FaseSubsanacionUnidad, domain.EstadoIncidencia
	}
	return domain.FaseFiscalizacion, domain.EstadoEnCurso
}

func clasificarFalloFiscalizacion(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, ports.ErrClaveIdempotenciaUsada) {
		return ports.ErrClaveIdempotenciaUsada
	}
	if errors.Is(causa, ErrFiscalizacionDenegada) {
		return ErrFiscalizacionDenegada
	}
	if errors.Is(causa, ports.ErrResultadoFiscalizacionNoConfiable) {
		return ErrResultadoFiscalizacionNoConfiable
	}
	return ports.ErrPersistenciaFiscalizacionNoDisponible
}
