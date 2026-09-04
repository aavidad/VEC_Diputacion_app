package postgres

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaConfirmarInformeJuridico        = "vec.contratacion-temporal.confirmar-informe-juridico.v1"
	maximoCargaConfirmacionInformeJuridico = 3 * 1024 * 1024
)

type operacionConfirmarInformeJuridicoV1 struct {
	Esquema                string                                  `json:"esquema"`
	Operacion              string                                  `json:"operacion"`
	ReservaRef             string                                  `json:"reserva_ref"`
	Referencias            referenciasPrepararInformeJuridicoV1    `json:"referencias"`
	AmbitoIdempotenciaHMAC string                                  `json:"ambito_idempotencia_hmac"`
	HuellaPeticionHMAC     string                                  `json:"huella_peticion_hmac"`
	OrganizacionRef        string                                  `json:"organizacion_ref"`
	ExpedienteRef          string                                  `json:"expediente_ref"`
	VersionAnterior        uint64                                  `json:"version_anterior"`
	ActorRef               string                                  `json:"actor_ref"`
	PerfilRef              string                                  `json:"perfil_ref"`
	ExpedienteAnterior     domain.Expediente                       `json:"expediente_anterior"`
	ExpedienteSiguiente    domain.Expediente                       `json:"expediente_siguiente"`
	Actuacion              domain.Actuacion                        `json:"actuacion"`
	Configuracion          configuracionConfirmarInformeJuridicoV1 `json:"configuracion"`
	Borrador               domain.EstadoBorradorInformeJuridico    `json:"borrador"`
	Documento              ports.DocumentoInformeJuridico          `json:"documento"`
	Autorizacion           autorizacionConfirmarInformeJuridicoV1  `json:"autorizacion"`
	InstanteEfecto         time.Time                               `json:"instante_efecto"`
}

type configuracionConfirmarInformeJuridicoV1 struct {
	Plantilla             domain.ReferenciaPlantillaInformeJuridico   `json:"plantilla"`
	ReferenciasNormativas []domain.ReferenciaNormativaInformeJuridico `json:"referencias_normativas"`
	Anexos                []domain.AnexoDocumentalInformeJuridico     `json:"anexos"`
	DefinicionRef         string                                      `json:"definicion_ref"`
	DefinicionVersion     uint64                                      `json:"definicion_version"`
	DefinicionHuella      string                                      `json:"definicion_huella_sha256"`
	Accion                domain.ClaveCatalogo                        `json:"accion"`
	Finalidad             domain.ClaveCatalogo                        `json:"finalidad"`
	UnidadEjecutoraRef    string                                      `json:"unidad_ejecutora_ref"`
	EvaluadaEn            time.Time                                   `json:"evaluada_en"`
	ValidaHasta           time.Time                                   `json:"valida_hasta"`
}

type autorizacionConfirmarInformeJuridicoV1 struct {
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

func codificarOperacionConfirmarInformeJuridico(
	orden ports.OrdenConfirmarInformeJuridico,
) ([]byte, error) {
	if validarOrdenConfirmarInformeJuridico(orden, orden.InstanteEfecto) != nil ||
		len(orden.ExpedienteSiguiente.Actuaciones) == 0 {
		return nil, ports.ErrPreparacionInformeJuridicoInvalida
	}
	autorizacion, err := nuevaAutorizacionConfirmarInformeJuridico(orden)
	if err != nil {
		return nil, err
	}
	configuracion := orden.Configuracion
	operacion := operacionConfirmarInformeJuridicoV1{
		Esquema: esquemaConfirmarInformeJuridico, Operacion: "preparar",
		ReservaRef:             orden.Preparacion.Referencias.ReservaRef,
		Referencias:            nuevasReferenciasPrepararInformeJuridicoV1(orden.Preparacion.Referencias),
		AmbitoIdempotenciaHMAC: orden.Preparacion.AmbitoIdempotenciaHMAC,
		HuellaPeticionHMAC:     orden.Preparacion.HuellaPeticionHMAC,
		OrganizacionRef:        orden.Preparacion.Material.OrganizacionRef,
		ExpedienteRef:          orden.Preparacion.Material.ExpedienteRef,
		VersionAnterior:        orden.Preparacion.Material.VersionExpediente,
		ActorRef:               orden.Preparacion.Material.ActorRef,
		PerfilRef:              orden.Preparacion.Material.PerfilRef,
		ExpedienteAnterior:     orden.Preparacion.Expediente,
		ExpedienteSiguiente:    orden.ExpedienteSiguiente,
		Actuacion:              orden.ExpedienteSiguiente.Actuaciones[len(orden.ExpedienteSiguiente.Actuaciones)-1],
		Configuracion: configuracionConfirmarInformeJuridicoV1{
			Plantilla:             configuracion.Plantilla,
			ReferenciasNormativas: configuracion.ReferenciasNormativas,
			Anexos:                configuracion.Anexos, DefinicionRef: configuracion.DefinicionRef,
			DefinicionVersion: configuracion.DefinicionVersion,
			DefinicionHuella:  configuracion.DefinicionHuella,
			Accion:            configuracion.Accion, Finalidad: configuracion.Finalidad,
			UnidadEjecutoraRef: configuracion.UnidadEjecutoraRef,
			EvaluadaEn:         configuracion.EvaluadaEn, ValidaHasta: configuracion.ValidaHasta,
		},
		Borrador: orden.Borrador.Estado(), Documento: orden.Documento,
		Autorizacion: autorizacion, InstanteEfecto: orden.InstanteEfecto,
	}
	contenido, err := json.Marshal(operacion)
	if err != nil || len(contenido) == 0 ||
		len(contenido) > maximoCargaConfirmacionInformeJuridico {
		return nil, fmt.Errorf(
			"%w: proyeccion JSON rechazada",
			ports.ErrPreparacionInformeJuridicoInvalida,
		)
	}
	return contenido, nil
}

func nuevaAutorizacionConfirmarInformeJuridico(
	orden ports.OrdenConfirmarInformeJuridico,
) (autorizacionConfirmarInformeJuridicoV1, error) {
	solicitud, errSolicitud := orden.Evidencia.SolicitudV3.Datos()
	decisionCanonica, errDecision :=
		dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(
			orden.Evidencia.DecisionV3,
		)
	motivoCanonico, errMotivo :=
		dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
			solicitud.ReferenciaMotivo,
		)
	confirmacion, errConfirmacion := orden.Evidencia.ConfirmacionV3.Datos()
	huellaContexto, errContexto :=
		solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	instantanea := orden.Evidencia.Contexto.Resultado.Contexto.Instantanea
	if errSolicitud != nil || errDecision != nil || errMotivo != nil ||
		errConfirmacion != nil || errContexto != nil {
		return autorizacionConfirmarInformeJuridicoV1{},
			ports.ErrPreparacionInformeJuridicoInvalida
	}
	return autorizacionConfirmarInformeJuridicoV1{
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
