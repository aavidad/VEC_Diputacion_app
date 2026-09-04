package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	funcionConfirmarFiscalizacion        = "vec_contratacion_temporal.confirmar_fiscalizacion_v1"
	esquemaConfirmarFiscalizacion        = "vec.contratacion-temporal.confirmar-fiscalizacion.v1"
	audienciaConfirmarFiscalizacionV1    = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
	maximoIntentosConfirmarFiscalizacion = 3
	maximoCargaConfirmarFiscalizacion    = 3 * 1024 * 1024
)

var _ ports.TransaccionFiscalizaciones = (*TransaccionFiscalizacionesPostgreSQL)(nil)

type proveedorMaterialConfirmacionFiscalizacion interface {
	ProveerMaterialConfirmacionFiscalizacion(
		context.Context,
		ports.OrdenConfirmarFiscalizacion,
	) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type TransaccionFiscalizacionesPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor proveedorMaterialConfirmacionFiscalizacion
}

func NuevaTransaccionFiscalizacionesPostgreSQL(
	pool *pgxpool.Pool,
	proveedor proveedorMaterialConfirmacionFiscalizacion,
) (*TransaccionFiscalizacionesPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	return &TransaccionFiscalizacionesPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

type politicaConfirmarFiscalizacionV1 struct {
	DefinicionRef          string               `json:"definicion_ref"`
	DefinicionVersion      uint64               `json:"definicion_version"`
	DefinicionHuellaSHA256 string               `json:"definicion_huella_sha256"`
	Accion                 domain.ClaveCatalogo `json:"accion"`
	Finalidad              domain.ClaveCatalogo `json:"finalidad"`
	UnidadFiscalizadoraRef string               `json:"unidad_fiscalizadora_ref"`
	EvaluadaEn             time.Time            `json:"evaluada_en"`
	ValidaHasta            time.Time            `json:"valida_hasta"`
}

type autorizacionConfirmarFiscalizacionV1 struct {
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

type operacionConfirmarFiscalizacionV1 struct {
	Esquema                string                               `json:"esquema"`
	Operacion              string                               `json:"operacion"`
	ReservaRef             string                               `json:"reserva_ref"`
	Referencias            referenciasPrepararFiscalizacionV1   `json:"referencias"`
	AmbitoIdempotenciaHMAC string                               `json:"ambito_idempotencia_hmac"`
	HuellaPeticionHMAC     string                               `json:"huella_peticion_hmac"`
	OrganizacionRef        string                               `json:"organizacion_ref"`
	ExpedienteRef          string                               `json:"expediente_ref"`
	VersionAnterior        uint64                               `json:"version_anterior"`
	ActorRef               string                               `json:"actor_ref"`
	PerfilRef              string                               `json:"perfil_ref"`
	Resultado              domain.ResultadoFiscalizacion        `json:"resultado"`
	Observaciones          string                               `json:"observaciones"`
	ExpedienteAnterior     domain.Expediente                    `json:"expediente_anterior"`
	ExpedienteSiguiente    domain.Expediente                    `json:"expediente_siguiente"`
	Actuacion              domain.Actuacion                     `json:"actuacion"`
	Politica               politicaConfirmarFiscalizacionV1     `json:"politica"`
	Autorizacion           autorizacionConfirmarFiscalizacionV1 `json:"autorizacion"`
	InstanteEfecto         time.Time                            `json:"instante_efecto"`
}

type entradasConfirmarFiscalizacion struct {
	contenido       []byte
	capacidad       []byte
	decision        []byte
	motivo          []byte
	contextoActor   []byte
	personaVersion  int64
	perfilVersion   int64
	payloadVECAD3   []byte
	sobreCOSESign1  []byte
	evidencia       []byte
	raizPublicaSPKI []byte
}

func (e *entradasConfirmarFiscalizacion) borrar() {
	if e == nil {
		return
	}
	for _, contenido := range [][]byte{
		e.contenido, e.capacidad, e.decision, e.motivo, e.contextoActor,
		e.payloadVECAD3, e.sobreCOSESign1, e.evidencia, e.raizPublicaSPKI,
	} {
		borrarBytes(contenido)
	}
}

func (t *TransaccionFiscalizacionesPostgreSQL) ConfirmarFiscalizacion(
	ctx context.Context,
	orden ports.OrdenConfirmarFiscalizacion,
) (ports.ReciboFiscalizacion, error) {
	if ctx == nil || t == nil || dependenciaNula(t.pool) ||
		dependenciaNula(t.proveedor) ||
		validarOrdenConfirmarFiscalizacion(orden, orden.InstanteEfecto) != nil {
		return ports.ReciboFiscalizacion{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboFiscalizacion{}, err
	}
	material, err := t.proveedor.ProveerMaterialConfirmacionFiscalizacion(ctx, orden)
	if err != nil {
		return ports.ReciboFiscalizacion{}, errorDependenciaFiscalizacion(ctx)
	}
	entradas, err := prepararEntradasConfirmarFiscalizacion(orden, material)
	if err != nil {
		return ports.ReciboFiscalizacion{}, err
	}
	defer entradas.borrar()

	for intento := 1; intento <= maximoIntentosConfirmarFiscalizacion; intento++ {
		recibo, causa := t.confirmarEnTransaccion(ctx, orden, entradas)
		if causa == nil {
			return recibo, nil
		}
		if ctx.Err() != nil {
			return ports.ReciboFiscalizacion{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(causa) ||
			intento == maximoIntentosConfirmarFiscalizacion {
			return ports.ReciboFiscalizacion{},
				normalizarErrorConfirmacionFiscalizacion(ctx, causa)
		}
	}
	return ports.ReciboFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
}

func prepararEntradasConfirmarFiscalizacion(
	orden ports.OrdenConfirmarFiscalizacion,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (entradasConfirmarFiscalizacion, error) {
	if material.ValidarEstructura() != nil {
		return entradasConfirmarFiscalizacion{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	solicitud, errSolicitud := orden.Evidencia.SolicitudV3.Datos()
	confirmacion, errConfirmacion := orden.Evidencia.ConfirmacionV3.Datos()
	vinculo, errVinculo := solicitud.VinculoAutenticacionActor.Datos()
	huellaRecurso, errRecurso := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if errSolicitud != nil || errConfirmacion != nil || errVinculo != nil ||
		errRecurso != nil || resumen.ValidarEstructura() != nil ||
		resumen.DecisionRef() != confirmacion.DecisionRef ||
		resumen.DecisionHuellaSHA256() != confirmacion.DecisionHuellaSHA256 ||
		resumen.ContextoRef() != vinculo.RegistroContextoRef ||
		resumen.ContextoHuellaSHA256() != vinculo.ContextoActorHuellaSHA256 ||
		resumen.Operacion() != solicitud.Accion ||
		resumen.EfectoRef() != orden.Preparacion.Material.ExpedienteRef ||
		resumen.EfectoHuellaSHA256() != huellaRecurso ||
		resumen.AudienciaConsumo() != audienciaConfirmarFiscalizacionV1 ||
		!capacidadBreveContenidaEnConcesion(resumen, confirmacion) {
		return entradasConfirmarFiscalizacion{}, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	contenido, err := codificarOperacionConfirmarFiscalizacion(orden)
	if err != nil {
		return entradasConfirmarFiscalizacion{}, err
	}
	return entradasConfirmarFiscalizacion{
		contenido: contenido,
		capacidad: material.CapacidadCanonica(), decision: material.DecisionCanonica(),
		motivo: material.MotivoCanonico(), contextoActor: material.ContextoActorCanonico(),
		personaVersion: int64(material.PersonaVersion()),
		perfilVersion:  int64(material.PerfilVersion()),
		payloadVECAD3:  material.PayloadVECAD3(), sobreCOSESign1: material.SobreCOSESign1(),
		evidencia: material.EvidenciaVerificacion(), raizPublicaSPKI: material.RaizPublicaSPKI(),
	}, nil
}

func codificarOperacionConfirmarFiscalizacion(
	orden ports.OrdenConfirmarFiscalizacion,
) ([]byte, error) {
	if validarOrdenConfirmarFiscalizacion(orden, orden.InstanteEfecto) != nil ||
		len(orden.ExpedienteSiguiente.Actuaciones) == 0 {
		return nil, ports.ErrPreparacionFiscalizacionInvalida
	}
	autorizacion, err := nuevaAutorizacionConfirmarFiscalizacion(orden)
	if err != nil {
		return nil, err
	}
	p := orden.Politica
	material := orden.Preparacion.Material
	operacion := operacionConfirmarFiscalizacionV1{
		Esquema:                esquemaConfirmarFiscalizacion,
		Operacion:              ports.OperacionRegistrarResultadoFiscalizacion,
		ReservaRef:             orden.Preparacion.Referencias.ReservaRef,
		Referencias:            nuevasReferenciasPrepararFiscalizacionV1(orden.Preparacion.Referencias),
		AmbitoIdempotenciaHMAC: orden.Preparacion.AmbitoIdempotenciaHMAC,
		HuellaPeticionHMAC:     orden.Preparacion.HuellaPeticionHMAC,
		OrganizacionRef:        material.OrganizacionRef, ExpedienteRef: material.ExpedienteRef,
		VersionAnterior: material.VersionExpediente,
		ActorRef:        material.ActorRef, PerfilRef: material.PerfilRef,
		Resultado: material.Resultado, Observaciones: material.Observaciones,
		ExpedienteAnterior:  orden.Preparacion.Expediente,
		ExpedienteSiguiente: orden.ExpedienteSiguiente,
		Actuacion:           orden.ExpedienteSiguiente.Actuaciones[len(orden.ExpedienteSiguiente.Actuaciones)-1],
		Politica: politicaConfirmarFiscalizacionV1{
			DefinicionRef: p.DefinicionRef, DefinicionVersion: p.DefinicionVersion,
			DefinicionHuellaSHA256: p.DefinicionHuellaSHA256,
			Accion:                 p.Accion, Finalidad: p.Finalidad,
			UnidadFiscalizadoraRef: p.UnidadFiscalizadoraRef,
			EvaluadaEn:             p.EvaluadaEn, ValidaHasta: p.ValidaHasta,
		},
		Autorizacion: autorizacion, InstanteEfecto: orden.InstanteEfecto,
	}
	contenido, err := json.Marshal(operacion)
	if err != nil || len(contenido) == 0 ||
		len(contenido) > maximoCargaConfirmarFiscalizacion {
		return nil, fmt.Errorf("%w: proyeccion JSON rechazada",
			ports.ErrPreparacionFiscalizacionInvalida)
	}
	return contenido, nil
}

func nuevaAutorizacionConfirmarFiscalizacion(
	orden ports.OrdenConfirmarFiscalizacion,
) (autorizacionConfirmarFiscalizacionV1, error) {
	solicitud, errSolicitud := orden.Evidencia.SolicitudV3.Datos()
	decisionCanonica, errDecision := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(
		orden.Evidencia.DecisionV3,
	)
	motivoCanonico, errMotivo := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
		solicitud.ReferenciaMotivo,
	)
	confirmacion, errConfirmacion := orden.Evidencia.ConfirmacionV3.Datos()
	huellaContexto, errContexto := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	instantanea := orden.Evidencia.Contexto.Resultado.Contexto.Instantanea
	if errSolicitud != nil || errDecision != nil || errMotivo != nil ||
		errConfirmacion != nil || errContexto != nil {
		return autorizacionConfirmarFiscalizacionV1{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	return autorizacionConfirmarFiscalizacionV1{
		DecisionCanonicaHex: hex.EncodeToString(decisionCanonica),
		MotivoCanonicoHex:   hex.EncodeToString(motivoCanonico),
		PersonaVersion:      instantanea.PersonaVersion, PerfilVersion: instantanea.PerfilVersion,
		DecisionRef:          confirmacion.DecisionRef,
		DecisionHuellaSHA256: confirmacion.DecisionHuellaSHA256,
		PrincipalID:          instantanea.PersonaRef,
		PerfilActivoRef:      instantanea.PerfilActivoRef,
		Accion:               solicitud.Accion, RecursoRef: solicitud.Recurso.Referencia,
		ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad:                   solicitud.Finalidad,
	}, nil
}

func (t *TransaccionFiscalizacionesPostgreSQL) confirmarEnTransaccion(
	ctx context.Context,
	orden ports.OrdenConfirmarFiscalizacion,
	entradas entradasConfirmarFiscalizacion,
) (ports.ReciboFiscalizacion, error) {
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return ports.ReciboFiscalizacion{}, err
	}
	defer revertirTransaccion(tx)
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		return ports.ReciboFiscalizacion{}, err
	}
	var ahora time.Time
	if err = tx.QueryRow(ctx,
		`SELECT date_trunc('microseconds', clock_timestamp())`,
	).Scan(&ahora); err != nil {
		return ports.ReciboFiscalizacion{}, err
	}
	ahora = normalizarInstantePostgreSQL(ahora)
	if validarOrdenConfirmarFiscalizacion(orden, ahora) != nil {
		return ports.ReciboFiscalizacion{}, fmt.Errorf(
			"%w: vigencia precommit", ports.ErrPreparacionFiscalizacionInvalida,
		)
	}
	var reciboJSON string
	err = tx.QueryRow(ctx, `SELECT recibo_json::text FROM `+
		funcionConfirmarFiscalizacion+`($1::jsonb,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		entradas.contenido, entradas.capacidad, entradas.decision, entradas.motivo,
		entradas.contextoActor, entradas.personaVersion, entradas.perfilVersion,
		entradas.payloadVECAD3, entradas.sobreCOSESign1, entradas.evidencia,
		entradas.raizPublicaSPKI,
	).Scan(&reciboJSON)
	if err != nil {
		return ports.ReciboFiscalizacion{}, err
	}
	recibo, err := decodificarReciboFiscalizacion(reciboJSON)
	if err != nil || recibo.ValidarParaPreparacion(orden.Preparacion) != nil ||
		recibo.RegistradaEn.Before(orden.InstanteEfecto) {
		return ports.ReciboFiscalizacion{}, ports.ErrResultadoFiscalizacionNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ReciboFiscalizacion{}, err
	}
	return recibo, nil
}

func validarOrdenConfirmarFiscalizacion(
	orden ports.OrdenConfirmarFiscalizacion,
	instante time.Time,
) error {
	p := orden.Preparacion
	anterior := p.Expediente
	material := p.Material
	solicitudPolitica := ports.SolicitudResolverPoliticaFiscalizacion{
		OrganizacionRef: material.OrganizacionRef, ExpedienteRef: material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente,
		ActorRef:          material.ActorRef, PerfilRef: material.PerfilRef,
		Resultado: material.Resultado, Observaciones: material.Observaciones,
		FaseActual: anterior.FaseActual, EstadoActual: anterior.EstadoActual,
		Instante: orden.Politica.EvaluadaEn,
	}
	if anterior.Asignacion != nil {
		solicitudPolitica.UnidadAsignadaRef = anterior.Asignacion.UnidadRef
		solicitudPolitica.ResponsableAsignadoRef = anterior.Asignacion.ResponsableRef
	}
	if anterior.InformeJuridico != nil {
		solicitudPolitica.InformeJuridicoRef = anterior.InformeJuridico.InformeRef
		solicitudPolitica.DocumentoInformeRef = anterior.InformeJuridico.DocumentoRef
	}
	if p.Estado != ports.PreparacionFiscalizacionPreparada ||
		p.ReciboConfirmado != nil || p.Referencias.ValidarPara(material.Resultado) != nil ||
		material.Validar() != nil || anterior.Validar() != nil ||
		anterior.Referencia != material.ExpedienteRef ||
		anterior.OrganizacionRef != material.OrganizacionRef ||
		anterior.Version != 5 || anterior.Asignacion == nil ||
		anterior.InformeJuridico == nil || anterior.Fiscalizacion != nil ||
		!ports.SelloHMACSHA256Valido(p.AmbitoIdempotenciaHMAC) ||
		!ports.SelloHMACSHA256Valido(p.HuellaPeticionHMAC) ||
		orden.ExpedienteSiguiente.Validar() != nil ||
		!domain.InstanteUTCCanonico(orden.InstanteEfecto) ||
		!domain.InstanteUTCCanonico(instante) || instante.Before(orden.InstanteEfecto) ||
		orden.InstanteEfecto.Before(anterior.ActualizadoEn) ||
		orden.Politica.ValidarPara(solicitudPolitica, instante) != nil ||
		validarAutorizacionFiscalizacion(orden, instante) != nil {
		return ports.ErrPreparacionFiscalizacionInvalida
	}
	fase, estado := destinoFiscalizacionPostgreSQL(material.Resultado)
	retornoRef := ""
	if material.Resultado == domain.FiscalizacionDesfavorable {
		retornoRef = p.Referencias.RetornoRef
	}
	esperado, err := anterior.RegistrarFiscalizacion(
		5,
		domain.DatosRegistrarFiscalizacion{
			FiscalizacionRef:       p.Referencias.FiscalizacionRef,
			Resultado:              material.Resultado,
			UnidadFiscalizadoraRef: orden.Politica.UnidadFiscalizadoraRef,
			Observaciones:          material.Observaciones,
			FiscalizadaEn:          orden.InstanteEfecto, RetornoRef: retornoRef,
		},
		domain.DatosActuacion{
			AccionClave: domain.AccionRegistrarFiscalizacion,
			ActorRef:    material.ActorRef, UnidadRef: orden.Politica.UnidadFiscalizadoraRef,
			ReciboRef: p.Referencias.ReciboRef, RealizadaEn: orden.InstanteEfecto,
			FaseDestino: fase, EstadoDestino: estado,
			Observaciones: material.Observaciones,
			DocumentosRef: []string{anterior.InformeJuridico.DocumentoRef},
		},
	)
	if err != nil || !reflect.DeepEqual(esperado, orden.ExpedienteSiguiente) {
		return ports.ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

func validarAutorizacionFiscalizacion(
	orden ports.OrdenConfirmarFiscalizacion,
	instante time.Time,
) error {
	solicitud, errSolicitud := orden.Evidencia.SolicitudV3.Datos()
	vinculoSolicitud, errVinculo := solicitud.VinculoAutenticacionActor.Datos()
	vinculoContexto, errContexto := orden.Evidencia.Contexto.Vinculo.Datos()
	concedida, _, errDecision := orden.Evidencia.DecisionV3.Resultado()
	huellaDecision, errHuella := dominiovec.HuellaSHA256DecisionAutorizacionV3(
		orden.Evidencia.DecisionV3,
	)
	confirmacion, errConfirmacion := orden.Evidencia.ConfirmacionV3.Datos()
	material := orden.Preparacion.Material
	recurso := solicitud.Recurso
	huellaObservaciones := sha256.Sum256([]byte(material.Observaciones))
	anterior := orden.Preparacion.Expediente
	if errSolicitud != nil || errVinculo != nil || errContexto != nil ||
		errDecision != nil || errHuella != nil || errConfirmacion != nil ||
		!concedida || orden.Evidencia.Contexto.Resultado.Validar() != nil ||
		orden.Evidencia.Contexto.Vinculo.ValidarPara(orden.Evidencia.Contexto.Resultado) != nil ||
		!orden.Evidencia.Contexto.Vinculo.VigenteEn(instante, orden.Evidencia.Contexto.Resultado) ||
		orden.Evidencia.DecisionV3.ValidarPara(orden.Evidencia.SolicitudV3) != nil ||
		!reflect.DeepEqual(vinculoSolicitud, vinculoContexto) ||
		vinculoSolicitud.PrincipalID != material.ActorRef ||
		vinculoSolicitud.PerfilActivoRef != material.PerfilRef ||
		solicitud.ReferenciaMotivo != orden.Politica.MotivoAutorizacion ||
		solicitud.Accion != ports.AccionRegistrarFiscalizacion ||
		solicitud.Finalidad != string(orden.Politica.Finalidad) ||
		recurso.Referencia != material.ExpedienteRef ||
		recurso.ModuloID != ports.ModuloContratacion ||
		recurso.Tipo != ports.TipoRecursoFiscalizacion ||
		len(recurso.Ambitos) != 4 || len(recurso.Atributos) != 13 ||
		recurso.Ambitos["organizacion_ref"] != material.OrganizacionRef ||
		recurso.Ambitos["expediente_ref"] != material.ExpedienteRef ||
		recurso.Ambitos["fase_previa"] != string(anterior.FaseActual) ||
		recurso.Ambitos["estado_previo"] != string(anterior.EstadoActual) ||
		recurso.Atributos["version_expediente"] != "5" ||
		recurso.Atributos["resultado"] != string(material.Resultado) ||
		recurso.Atributos["observaciones_huella_sha256"] != hex.EncodeToString(huellaObservaciones[:]) ||
		recurso.Atributos["informe_juridico_ref"] != anterior.InformeJuridico.InformeRef ||
		recurso.Atributos["documento_informe_ref"] != anterior.InformeJuridico.DocumentoRef ||
		recurso.Atributos["unidad_asignada_ref"] != anterior.Asignacion.UnidadRef ||
		recurso.Atributos["responsable_asignado_ref"] != anterior.Asignacion.ResponsableRef ||
		recurso.Atributos["unidad_fiscalizadora_ref"] != orden.Politica.UnidadFiscalizadoraRef ||
		recurso.Atributos["politica_ref"] != orden.Politica.DefinicionRef ||
		recurso.Atributos["politica_version"] != strconv.FormatUint(orden.Politica.DefinicionVersion, 10) ||
		recurso.Atributos["politica_huella_sha256"] != orden.Politica.DefinicionHuellaSHA256 ||
		recurso.Atributos["ambito_idempotencia_hmac"] != orden.Preparacion.AmbitoIdempotenciaHMAC ||
		recurso.Atributos["huella_peticion_hmac"] != orden.Preparacion.HuellaPeticionHMAC ||
		confirmacion.DecisionHuellaSHA256 != huellaDecision ||
		!orden.Evidencia.ConfirmacionV3.DentroDeVentanaEn(instante) {
		return ports.ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

func destinoFiscalizacionPostgreSQL(
	resultado domain.ResultadoFiscalizacion,
) (domain.ClaveFase, domain.EstadoOperativo) {
	if resultado == domain.FiscalizacionDesfavorable {
		return domain.FaseSubsanacionUnidad, domain.EstadoIncidencia
	}
	return domain.FaseFiscalizacion, domain.EstadoEnCurso
}

func decodificarReciboFiscalizacion(contenido string) (ports.ReciboFiscalizacion, error) {
	var recibo ports.ReciboFiscalizacion
	if decodificarJSONEstricto([]byte(contenido), &recibo) != nil {
		return ports.ReciboFiscalizacion{}, ports.ErrResultadoFiscalizacionNoConfiable
	}
	recibo.RegistradaEn = normalizarInstantePostgreSQL(recibo.RegistradaEn)
	return recibo, nil
}

func normalizarErrorConfirmacionFiscalizacion(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, ports.ErrClaveIdempotenciaUsada) ||
		errors.Is(causa, ports.ErrPreparacionFiscalizacionInvalida) ||
		errors.Is(causa, ports.ErrResultadoFiscalizacionNoConfiable) {
		return causa
	}
	return ports.ErrPersistenciaFiscalizacionNoDisponible
}
