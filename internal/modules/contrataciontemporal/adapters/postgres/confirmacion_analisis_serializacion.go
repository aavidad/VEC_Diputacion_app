package postgres

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const esquemaConfirmarAnalisis = "vec.contratacion-temporal.confirmar-operacion-analisis.v1"

type operacionConfirmarAnalisisV1 struct {
	Esquema               string                      `json:"esquema"`
	ReservaRef            string                      `json:"reserva_ref"`
	ReciboRef             string                      `json:"recibo_ref"`
	Operacion             ports.TipoOperacionAnalisis `json:"operacion"`
	OrganizacionRef       string                      `json:"organizacion_ref"`
	ExpedienteRef         string                      `json:"expediente_ref"`
	VersionAnterior       uint64                      `json:"version_anterior"`
	ActorRef              string                      `json:"actor_ref"`
	PerfilRef             string                      `json:"perfil_ref"`
	ArtefactoRef          string                      `json:"artefacto_ref"`
	ArtefactoHuellaSHA256 string                      `json:"artefacto_huella_sha256"`
	AmbitoRaizHMAC        string                      `json:"ambito_raiz_hmac"`
	HuellaSemanticaHMAC   string                      `json:"huella_semantica_hmac"`
	AmbitoConsultaHMAC    string                      `json:"ambito_consulta_hmac"`
	HuellaConsultaHMAC    string                      `json:"huella_consulta_hmac"`
	AliasesConsulta       []aliasConsultaAnalisisV1   `json:"aliases_consulta"`
	ExpedienteAnterior    domain.Expediente           `json:"expediente_anterior"`
	ExpedienteSiguiente   domain.Expediente           `json:"expediente_siguiente"`
	Actuacion             domain.Actuacion            `json:"actuacion"`
	Fuentes               fuentesConfirmarAnalisisV1  `json:"fuentes"`
	Autorizacion          autorizacionAnalisisV1      `json:"autorizacion"`
	Politica              politicaConfirmarAnalisisV1 `json:"politica"`
}

type aliasConsultaAnalisisV1 struct {
	Generacion uint32 `json:"generacion"`
	AmbitoHMAC string `json:"ambito_hmac"`
}

type fuentesConfirmarAnalisisV1 struct {
	ConjuntoHuellaSHA256 string                     `json:"conjunto_huella_sha256"`
	PruebaCanonicaHex    string                     `json:"prueba_canonica_hex"`
	RC                   fuenteConfirmarAnalisisV1  `json:"rc"`
	Coste                *fuenteConfirmarAnalisisV1 `json:"coste"`
}

type fuenteConfirmarAnalisisV1 struct {
	Tipo                  ports.TipoRespuestaFuenteAnalisis `json:"tipo"`
	PeticionRef           string                            `json:"peticion_ref"`
	RespuestaHuellaSHA256 string                            `json:"respuesta_huella_sha256"`
	AutoridadRef          string                            `json:"autoridad_ref"`
	Generacion            uint32                            `json:"generacion"`
	ReciboRespuestaRef    string                            `json:"recibo_respuesta_ref"`
	SelloRespuestaHMAC    string                            `json:"sello_respuesta_hmac"`
	VerificadorRef        string                            `json:"verificador_ref"`
	MaterialHuellaSHA256  string                            `json:"material_huella_sha256"`
	EmitidaEn             time.Time                         `json:"emitida_en"`
	ValidaHasta           time.Time                         `json:"valida_hasta"`
	VerificadaEn          time.Time                         `json:"verificada_en"`
	Publicacion           *publicacionFuenteAnalisisV1      `json:"publicacion"`
}

type publicacionFuenteAnalisisV1 struct {
	PublicadorRef         string    `json:"publicador_ref"`
	PublicacionRef        string    `json:"publicacion_ref"`
	ReciboVerificacionRef string    `json:"recibo_verificacion_ref"`
	HuellaSolicitudSHA256 string    `json:"huella_solicitud_sha256"`
	VerificadaEn          time.Time `json:"verificada_en"`
}

type autorizacionAnalisisV1 struct {
	DecisionCanonicaHex         string `json:"decision_canonica_hex"`
	MotivoCanonicoHex           string `json:"motivo_canonico_hex"`
	PersonaVersion              uint64 `json:"persona_version"`
	PerfilVersion               uint64 `json:"perfil_version"`
	DecisionRef                 string `json:"decision_ref"`
	DecisionHuellaSHA256        string `json:"decision_huella_sha256"`
	PrincipalID                 string `json:"principal_id"`
	PerfilActivoRef             string `json:"perfil_activo_ref"`
	Accion                      string `json:"accion"`
	RecursoRef                  string `json:"recurso_ref"`
	ContextoRecursoHuellaSHA256 string `json:"contexto_recurso_huella_sha256"`
	Finalidad                   string `json:"finalidad"`
}

type politicaConfirmarAnalisisV1 struct {
	DefinicionRef            string `json:"definicion_ref"`
	Version                  uint64 `json:"version"`
	HuellaSHA256             string `json:"huella_sha256"`
	Accion                   string `json:"accion"`
	Finalidad                string `json:"finalidad"`
	FasePrevia               string `json:"fase_previa"`
	EstadoPrevio             string `json:"estado_previo"`
	UnidadRef                string `json:"unidad_ref"`
	MotivoRectificacionClave string `json:"motivo_rectificacion_clave"`
	ExigeActorDistinto       bool   `json:"exige_actor_distinto"`
}

func nuevaOperacionConfirmarAnalisis(
	orden ports.OrdenConfirmarOperacionAnalisis,
) (operacionConfirmarAnalisisV1, error) {
	evidencia, err := orden.Datos()
	if err != nil {
		return operacionConfirmarAnalisisV1{},
			ports.ErrOrdenOperacionAnalisisInvalida
	}
	preparacion, err := evidencia.Preparacion.DatosPara(
		evidencia.SolicitudPreparacion,
	)
	fuentes, errFuentes := nuevasFuentesConfirmarAnalisis(
		evidencia.OrdenConsumoFuentes,
	)
	autorizacion, errAutorizacion := nuevaAutorizacionConfirmarAnalisis(
		evidencia,
	)
	aliases, errAliases := nuevosAliasesConsultaAnalisis(
		evidencia.SolicitudPreparacion,
	)
	switch {
	case err != nil:
		return operacionConfirmarAnalisisV1{}, fmt.Errorf(
			"%w: preparación no proyectable",
			ports.ErrOrdenOperacionAnalisisInvalida,
		)
	case errFuentes != nil:
		return operacionConfirmarAnalisisV1{}, fmt.Errorf(
			"%w: fuentes no proyectables",
			ports.ErrOrdenOperacionAnalisisInvalida,
		)
	case errAutorizacion != nil:
		return operacionConfirmarAnalisisV1{}, fmt.Errorf(
			"%w: autorización no proyectable",
			ports.ErrOrdenOperacionAnalisisInvalida,
		)
	case errAliases != nil:
		return operacionConfirmarAnalisisV1{}, fmt.Errorf(
			"%w: alias de consulta no proyectable",
			ports.ErrOrdenOperacionAnalisisInvalida,
		)
	case len(evidencia.ExpedienteSiguiente.Actuaciones) == 0:
		return operacionConfirmarAnalisisV1{}, fmt.Errorf(
			"%w: actuación ausente",
			ports.ErrOrdenOperacionAnalisisInvalida,
		)
	}
	indiceActuacion := len(evidencia.ExpedienteSiguiente.Actuaciones) - 1
	actuacion := evidencia.ExpedienteSiguiente.Actuaciones[indiceActuacion]
	motivoRectificacion :=
		string(evidencia.SolicitudPolitica.MotivoRectificacionClave)
	if motivoRectificacion == "" {
		motivoRectificacion = ports.ValorMotivoRectificacionNoAplica
	}
	return operacionConfirmarAnalisisV1{
		Esquema:    esquemaConfirmarAnalisis,
		ReservaRef: preparacion.ReservaRef, ReciboRef: preparacion.ReciboRef,
		Operacion:       preparacion.Operacion,
		OrganizacionRef: preparacion.OrganizacionRef,
		ExpedienteRef:   preparacion.ExpedienteRef,
		VersionAnterior: preparacion.VersionExpediente,
		ActorRef:        preparacion.ActorRef, PerfilRef: preparacion.PerfilRef,
		ArtefactoRef:          preparacion.ArtefactoRef,
		ArtefactoHuellaSHA256: preparacion.ArtefactoHuellaSHA256,
		AmbitoRaizHMAC:        preparacion.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:   preparacion.HuellaSemanticaHMAC,
		AmbitoConsultaHMAC:    preparacion.AmbitoConsultaHMAC,
		HuellaConsultaHMAC:    preparacion.HuellaConsultaHMAC,
		AliasesConsulta:       aliases,
		ExpedienteAnterior:    evidencia.ExpedienteAnterior,
		ExpedienteSiguiente:   evidencia.ExpedienteSiguiente,
		Actuacion:             actuacion, Fuentes: fuentes,
		Autorizacion: autorizacion,
		Politica: politicaConfirmarAnalisisV1{
			DefinicionRef:            evidencia.Politica.DefinicionRef,
			Version:                  evidencia.Politica.Version,
			HuellaSHA256:             evidencia.Politica.HuellaSHA256,
			Accion:                   string(evidencia.Politica.Accion),
			Finalidad:                string(evidencia.Politica.Finalidad),
			FasePrevia:               string(evidencia.Politica.FasePrevia),
			EstadoPrevio:             string(evidencia.Politica.EstadoPrevio),
			UnidadRef:                evidencia.Politica.UnidadRef,
			MotivoRectificacionClave: motivoRectificacion,
			ExigeActorDistinto:       evidencia.Politica.ExigeActorDistinto,
		},
	}, nil
}

func nuevasFuentesConfirmarAnalisis(
	orden ports.OrdenConsumoConjuntoFuentesAnalisisO3,
) (fuentesConfirmarAnalisisV1, error) {
	datos, err := orden.Datos()
	prueba, errPrueba := orden.PruebaCanonica()
	rc, errRC := nuevaFuenteConfirmarAnalisis(datos.OrdenRC)
	if err != nil || errPrueba != nil || errRC != nil {
		return fuentesConfirmarAnalisisV1{},
			ports.ErrOrdenOperacionAnalisisInvalida
	}
	fuentes := fuentesConfirmarAnalisisV1{
		ConjuntoHuellaSHA256: datos.HuellaSHA256,
		PruebaCanonicaHex:    hex.EncodeToString(prueba),
		RC:                   rc,
	}
	if datos.OrdenCoste != nil {
		coste, errCoste := nuevaFuenteConfirmarAnalisis(
			*datos.OrdenCoste,
		)
		if errCoste != nil {
			return fuentesConfirmarAnalisisV1{},
				ports.ErrOrdenOperacionAnalisisInvalida
		}
		fuentes.Coste = &coste
	}
	return fuentes, nil
}

func nuevaFuenteConfirmarAnalisis(
	orden ports.OrdenConsumoRespuestaFuenteAnalisis,
) (fuenteConfirmarAnalisisV1, error) {
	datos, err := orden.Datos()
	confirmacion, errConfirmacion := datos.ConfirmacionRespuesta.Datos()
	if err != nil || errConfirmacion != nil {
		return fuenteConfirmarAnalisisV1{},
			ports.ErrOrdenOperacionAnalisisInvalida
	}
	fuente := fuenteConfirmarAnalisisV1{
		Tipo: datos.Tipo, PeticionRef: datos.PeticionRef,
		RespuestaHuellaSHA256: datos.HuellaRespuestaSHA256,
		AutoridadRef:          datos.Atestacion.Metadatos.AutoridadRef,
		Generacion:            datos.Atestacion.Metadatos.Generacion,
		ReciboRespuestaRef:    datos.Atestacion.Metadatos.ReciboRef,
		SelloRespuestaHMAC:    confirmacion.SelloRespuestaHMAC,
		VerificadorRef:        confirmacion.VerificadorRef,
		MaterialHuellaSHA256:  confirmacion.HuellaMaterialSHA256,
		EmitidaEn:             confirmacion.EmitidaEn,
		ValidaHasta:           confirmacion.ValidaHasta,
		VerificadaEn:          confirmacion.VerificadaEn,
	}
	if datos.ConfirmacionPublicacion != nil {
		publicacion, errPublicacion := datos.ConfirmacionPublicacion.Datos()
		if errPublicacion != nil {
			return fuenteConfirmarAnalisisV1{},
				ports.ErrOrdenOperacionAnalisisInvalida
		}
		fuente.Publicacion = &publicacionFuenteAnalisisV1{
			PublicadorRef:         publicacion.PublicadorRef,
			PublicacionRef:        publicacion.PublicacionRef,
			ReciboVerificacionRef: publicacion.ReciboVerificacionRef,
			HuellaSolicitudSHA256: publicacion.HuellaSolicitudSHA256,
			VerificadaEn:          publicacion.VerificadaEn,
		}
	}
	return fuente, nil
}

func nuevaAutorizacionConfirmarAnalisis(
	evidencia ports.EvidenciaOrdenConfirmarOperacionAnalisis,
) (autorizacionAnalisisV1, error) {
	solicitud, errSolicitud := evidencia.SolicitudV3.Datos()
	decisionCanonica, errDecision :=
		dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(
			evidencia.DecisionV3,
		)
	motivoCanonico, errMotivo :=
		dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
			solicitud.ReferenciaMotivo,
		)
	confirmacion, errConfirmacion := evidencia.ConfirmacionV3.Datos()
	huellaContexto, errContexto :=
		solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	instantanea := evidencia.ContextoAutorizacion.Resultado.Contexto.Instantanea
	if errSolicitud != nil || errDecision != nil || errMotivo != nil ||
		errConfirmacion != nil || errContexto != nil {
		return autorizacionAnalisisV1{},
			ports.ErrOrdenOperacionAnalisisInvalida
	}
	return autorizacionAnalisisV1{
		DecisionCanonicaHex:  hex.EncodeToString(decisionCanonica),
		MotivoCanonicoHex:    hex.EncodeToString(motivoCanonico),
		PersonaVersion:       instantanea.PersonaVersion,
		PerfilVersion:        instantanea.PerfilVersion,
		DecisionRef:          confirmacion.DecisionRef,
		DecisionHuellaSHA256: confirmacion.DecisionHuellaSHA256,
		PrincipalID:          instantanea.PersonaRef,
		PerfilActivoRef:      instantanea.PerfilActivoRef,
		Accion:               solicitud.Accion, RecursoRef: solicitud.Recurso.Referencia,
		ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad:                   solicitud.Finalidad,
	}, nil
}

func nuevosAliasesConsultaAnalisis(
	solicitud ports.SolicitudPrepararOperacionAnalisis,
) ([]aliasConsultaAnalisisV1, error) {
	coleccion, err := solicitud.IdentidadConsulta.AmbitosIdempotencia()
	if err != nil {
		return nil, ports.ErrOrdenOperacionAnalisisInvalida
	}
	datos, err := coleccion.Datos()
	if err != nil {
		return nil, ports.ErrOrdenOperacionAnalisisInvalida
	}
	aliases := make([]aliasConsultaAnalisisV1, 0, len(datos.Retenidos)+1)
	aliases = append(aliases, aliasConsultaAnalisisV1{
		Generacion: datos.Activo.Generacion,
		AmbitoHMAC: datos.Activo.Valor,
	})
	for _, retenido := range datos.Retenidos {
		aliases = append(aliases, aliasConsultaAnalisisV1{
			Generacion: retenido.Generacion,
			AmbitoHMAC: retenido.Valor,
		})
	}
	return aliases, nil
}

func codificarOperacionConfirmarAnalisis(
	orden ports.OrdenConfirmarOperacionAnalisis,
) ([]byte, error) {
	operacion, err := nuevaOperacionConfirmarAnalisis(orden)
	if err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(operacion)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: proyección JSON rechazada: %v",
			ports.ErrOrdenOperacionAnalisisInvalida,
			err,
		)
	}
	if len(contenido) == 0 || len(contenido) > 2*1024*1024 {
		return nil, ports.ErrOrdenOperacionAnalisisInvalida
	}
	return contenido, nil
}

func decodificarReciboConfirmacionAnalisis(
	contenido string,
) (ports.ReciboOperacionAnalisis, error) {
	var recibo ports.ReciboOperacionAnalisis
	if decodificarJSONEstricto([]byte(contenido), &recibo) != nil ||
		recibo == (ports.ReciboOperacionAnalisis{}) {
		return ports.ReciboOperacionAnalisis{},
			ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	recibo.ConfirmadaEn = normalizarInstantePostgreSQL(recibo.ConfirmadaEn)
	return recibo, nil
}

func errorConfirmacionAnalisis(causa error) error {
	var postgres *pgconn.PgError
	if errors.As(causa, &postgres) && postgres.Code == "23505" {
		return ports.ErrConjuntoFuentesAnalisisYaConsumido
	}
	return ports.ErrPersistenciaOperacionAnalisisNoDisponible
}
