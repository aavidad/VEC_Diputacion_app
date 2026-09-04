package postgres

import (
	"context"
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
	funcionConfirmarInformeJuridico        = "vec_contratacion_temporal.confirmar_informe_juridico_v1"
	maximoIntentosConfirmarInformeJuridico = 3
	// La clave publicada VEC-AD-3 pertenece al consumidor transaccional de CT.
	audienciaConfirmarInformeJuridicoV1 = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
)

var _ ports.TransaccionInformesJuridicos = (*TransaccionInformesJuridicosPostgreSQL)(nil)

type TransaccionInformesJuridicosPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor proveedorMaterialConfirmacionInformeJuridico
}

type proveedorMaterialConfirmacionInformeJuridico interface {
	ProveerMaterialConfirmacionInformeJuridico(
		context.Context,
		ports.OrdenConfirmarInformeJuridico,
	) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func NuevaTransaccionInformesJuridicosPostgreSQL(
	pool *pgxpool.Pool,
	proveedor proveedorMaterialConfirmacionInformeJuridico,
) (*TransaccionInformesJuridicosPostgreSQL, error) {
	return nuevaTransaccionInformesJuridicosPostgreSQL(pool, proveedor)
}

func nuevaTransaccionInformesJuridicosPostgreSQL(
	pool iniciadorTransacciones,
	proveedor proveedorMaterialConfirmacionInformeJuridico,
) (*TransaccionInformesJuridicosPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	return &TransaccionInformesJuridicosPostgreSQL{
		pool: pool, proveedor: proveedor,
	}, nil
}

type entradasConfirmarInformeJuridico struct {
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

func (e *entradasConfirmarInformeJuridico) borrar() {
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

func (t *TransaccionInformesJuridicosPostgreSQL) ConfirmarInformeJuridico(
	ctx context.Context,
	orden ports.OrdenConfirmarInformeJuridico,
) (ports.ReciboInformeJuridico, error) {
	if ctx == nil || t == nil || dependenciaNula(t.pool) ||
		dependenciaNula(t.proveedor) ||
		validarOrdenConfirmarInformeJuridico(orden, orden.InstanteEfecto) != nil {
		return ports.ReciboInformeJuridico{},
			ports.ErrPreparacionInformeJuridicoInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboInformeJuridico{}, err
	}
	material, err := t.proveedor.ProveerMaterialConfirmacionInformeJuridico(ctx, orden)
	if err != nil {
		return ports.ReciboInformeJuridico{},
			errorDependenciaInformeJuridico(ctx)
	}
	entradas, err := prepararEntradasConfirmarInformeJuridico(orden, material)
	if err != nil {
		return ports.ReciboInformeJuridico{}, err
	}
	defer entradas.borrar()

	for intento := 1; intento <= maximoIntentosConfirmarInformeJuridico; intento++ {
		recibo, causa := t.confirmarInformeJuridicoEnTransaccion(ctx, orden, entradas)
		if causa == nil {
			return recibo, nil
		}
		if ctx.Err() != nil {
			return ports.ReciboInformeJuridico{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(causa) ||
			intento == maximoIntentosConfirmarInformeJuridico {
			return ports.ReciboInformeJuridico{},
				normalizarErrorConfirmacionInformeJuridico(ctx, causa)
		}
	}
	return ports.ReciboInformeJuridico{},
		ports.ErrPersistenciaInformeJuridicoNoDisponible
}

func prepararEntradasConfirmarInformeJuridico(
	orden ports.OrdenConfirmarInformeJuridico,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (entradasConfirmarInformeJuridico, error) {
	if material.ValidarEstructura() != nil {
		return entradasConfirmarInformeJuridico{},
			ports.ErrPreparacionInformeJuridicoInvalida
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
		resumen.AudienciaConsumo() != audienciaConfirmarInformeJuridicoV1 ||
		!capacidadBreveContenidaEnConcesion(resumen, confirmacion) {
		return entradasConfirmarInformeJuridico{},
			ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	contenido, err := codificarOperacionConfirmarInformeJuridico(orden)
	if err != nil {
		return entradasConfirmarInformeJuridico{}, err
	}
	return entradasConfirmarInformeJuridico{
		contenido: contenido,
		capacidad: material.CapacidadCanonica(), decision: material.DecisionCanonica(),
		motivo: material.MotivoCanonico(), contextoActor: material.ContextoActorCanonico(),
		personaVersion: int64(material.PersonaVersion()),
		perfilVersion:  int64(material.PerfilVersion()),
		payloadVECAD3:  material.PayloadVECAD3(), sobreCOSESign1: material.SobreCOSESign1(),
		evidencia: material.EvidenciaVerificacion(), raizPublicaSPKI: material.RaizPublicaSPKI(),
	}, nil
}

func (t *TransaccionInformesJuridicosPostgreSQL) confirmarInformeJuridicoEnTransaccion(
	ctx context.Context,
	orden ports.OrdenConfirmarInformeJuridico,
	entradas entradasConfirmarInformeJuridico,
) (ports.ReciboInformeJuridico, error) {
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return ports.ReciboInformeJuridico{}, err
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
		return ports.ReciboInformeJuridico{}, err
	}
	var ahora time.Time
	if err = tx.QueryRow(ctx,
		`SELECT date_trunc('microseconds', clock_timestamp())`,
	).Scan(&ahora); err != nil {
		return ports.ReciboInformeJuridico{}, err
	}
	ahora = normalizarInstantePostgreSQL(ahora)
	if validarOrdenConfirmarInformeJuridico(orden, ahora) != nil {
		return ports.ReciboInformeJuridico{},
			fmt.Errorf("%w: vigencia precommit",
				ports.ErrPreparacionInformeJuridicoInvalida)
	}

	var reciboJSON string
	err = tx.QueryRow(ctx, `SELECT recibo_json::text FROM `+
		funcionConfirmarInformeJuridico+`($1::jsonb,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		entradas.contenido, entradas.capacidad, entradas.decision, entradas.motivo,
		entradas.contextoActor, entradas.personaVersion, entradas.perfilVersion,
		entradas.payloadVECAD3, entradas.sobreCOSESign1, entradas.evidencia,
		entradas.raizPublicaSPKI,
	).Scan(&reciboJSON)
	if err != nil {
		return ports.ReciboInformeJuridico{}, err
	}
	recibo, err := decodificarReciboInformeJuridico(reciboJSON)
	if err != nil || recibo.ValidarParaPreparacion(orden.Preparacion) != nil ||
		recibo.HuellaDocumentoSHA256 != orden.Documento.HuellaDocumentoSHA256 ||
		recibo.HuellaBorradorSHA256 != orden.Borrador.HuellaSHA256() ||
		recibo.ContenidoDesarrollo != orden.Documento.ContenidoDesarrollo {
		return ports.ReciboInformeJuridico{},
			ports.ErrResultadoInformeJuridicoNoConfiable
	}
	confirmacion, err := orden.Evidencia.ConfirmacionV3.Datos()
	if err != nil || recibo.ConcesionV3DecisionRef != confirmacion.DecisionRef ||
		recibo.ConfirmadaEn.Before(orden.InstanteEfecto) {
		return ports.ReciboInformeJuridico{},
			ports.ErrResultadoInformeJuridicoNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ReciboInformeJuridico{}, err
	}
	return recibo, nil
}

func validarOrdenConfirmarInformeJuridico(
	orden ports.OrdenConfirmarInformeJuridico,
	instante time.Time,
) error {
	p := orden.Preparacion
	anterior := p.Expediente
	material := p.Material
	solicitudDocumento := ports.SolicitudGenerarDocumentoInformeJuridico{
		DocumentoRef: p.Referencias.DocumentoRef,
		Borrador:     orden.Borrador,
	}
	solicitudConfiguracion := ports.SolicitudResolverConfiguracionInformeJuridico{
		OrganizacionRef: material.OrganizacionRef, ExpedienteRef: material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente, ActorRef: material.ActorRef,
		PerfilRef: material.PerfilRef, FaseActual: anterior.FaseActual,
		EstadoActual: anterior.EstadoActual, Instante: orden.Configuracion.EvaluadaEn,
	}
	if anterior.Asignacion != nil {
		solicitudConfiguracion.UnidadAsignadaRef = anterior.Asignacion.UnidadRef
	}
	if p.Estado != ports.PreparacionInformeJuridicoReservada ||
		p.ReciboConfirmado != nil || p.Referencias.Validar() != nil ||
		material.Validar() != nil || anterior.Validar() != nil ||
		anterior.Referencia != material.ExpedienteRef ||
		anterior.OrganizacionRef != material.OrganizacionRef ||
		anterior.Version != material.VersionExpediente ||
		anterior.Asignacion == nil || anterior.InformeJuridico != nil ||
		!ports.SelloHMACSHA256Valido(p.AmbitoIdempotenciaHMAC) ||
		!ports.SelloHMACSHA256Valido(p.HuellaPeticionHMAC) ||
		orden.Borrador.Validar() != nil ||
		!configuracionInformeJuridicoCoincideConBorrador(orden.Configuracion, orden.Borrador) ||
		orden.Documento.ValidarPara(solicitudDocumento) != nil ||
		orden.ExpedienteSiguiente.Validar() != nil ||
		!domain.InstanteUTCCanonico(orden.InstanteEfecto) ||
		!domain.InstanteUTCCanonico(instante) ||
		instante.Before(orden.InstanteEfecto) ||
		orden.InstanteEfecto.Before(anterior.ActualizadoEn) ||
		orden.Configuracion.ValidarPara(solicitudConfiguracion, instante) != nil ||
		validarAutorizacionInformeJuridico(orden, instante) != nil {
		return ports.ErrPreparacionInformeJuridicoInvalida
	}
	informe := domain.InformeJuridicoEmitido{
		Borrador: orden.Borrador.Estado(), InformeRef: p.Referencias.InformeRef,
		DocumentoRef:          orden.Documento.DocumentoRef,
		VersionDocumento:      orden.Documento.VersionDocumento,
		HuellaDocumentoSHA256: orden.Documento.HuellaDocumentoSHA256,
		EmitidoEn:             orden.InstanteEfecto,
	}
	esperado, err := anterior.RegistrarInformeJuridico(
		material.VersionExpediente,
		informe,
		domain.DatosActuacion{
			AccionClave: domain.AccionEmitirInformeJuridico,
			ActorRef:    material.ActorRef, UnidadRef: orden.Configuracion.UnidadEjecutoraRef,
			ReciboRef: p.Referencias.ReciboRef, RealizadaEn: orden.InstanteEfecto,
			FaseDestino: domain.FaseInformeJuridico, EstadoDestino: domain.EstadoEnCurso,
			DocumentosRef: []string{orden.Documento.DocumentoRef},
		},
	)
	if err != nil || !reflect.DeepEqual(esperado, orden.ExpedienteSiguiente) {
		return ports.ErrPreparacionInformeJuridicoInvalida
	}
	return nil
}

func configuracionInformeJuridicoCoincideConBorrador(
	configuracion ports.ConfiguracionInformeJuridico,
	borrador domain.BorradorInformeJuridico,
) bool {
	estado := borrador.Estado()
	return estado.Plantilla == configuracion.Plantilla &&
		reflect.DeepEqual(
			estado.ReferenciasNormativas,
			configuracion.ReferenciasNormativas,
		) &&
		reflect.DeepEqual(estado.Anexos, configuracion.Anexos)
}

func validarAutorizacionInformeJuridico(
	orden ports.OrdenConfirmarInformeJuridico,
	instante time.Time,
) error {
	solicitud, errSolicitud := orden.Evidencia.SolicitudV3.Datos()
	vinculoSolicitud, errVinculo := solicitud.VinculoAutenticacionActor.Datos()
	vinculoContexto, errContexto := orden.Evidencia.Contexto.Vinculo.Datos()
	concedida, _, errDecision := orden.Evidencia.DecisionV3.Resultado()
	huellaDecision, errHuella :=
		dominiovec.HuellaSHA256DecisionAutorizacionV3(orden.Evidencia.DecisionV3)
	confirmacion, errConfirmacion := orden.Evidencia.ConfirmacionV3.Datos()
	material := orden.Preparacion.Material
	recurso := solicitud.Recurso
	if errSolicitud != nil || errVinculo != nil || errContexto != nil ||
		errDecision != nil || errHuella != nil || errConfirmacion != nil ||
		!concedida || orden.Evidencia.Contexto.Resultado.Validar() != nil ||
		orden.Evidencia.Contexto.Vinculo.ValidarPara(
			orden.Evidencia.Contexto.Resultado) != nil ||
		!orden.Evidencia.Contexto.Vinculo.VigenteEn(
			instante, orden.Evidencia.Contexto.Resultado) ||
		orden.Evidencia.DecisionV3.ValidarPara(orden.Evidencia.SolicitudV3) != nil ||
		!reflect.DeepEqual(vinculoSolicitud, vinculoContexto) ||
		vinculoSolicitud.PrincipalID != material.ActorRef ||
		vinculoSolicitud.PerfilActivoRef != material.PerfilRef ||
		solicitud.ReferenciaMotivo != orden.Configuracion.MotivoAutorizacion ||
		solicitud.Accion != ports.AccionEmitirInformeJuridico ||
		solicitud.Finalidad != string(orden.Configuracion.Finalidad) ||
		recurso.Referencia != material.ExpedienteRef ||
		recurso.ModuloID != ports.ModuloContratacion ||
		recurso.Tipo != ports.TipoRecursoInformeJuridico ||
		len(recurso.Ambitos) != 4 || len(recurso.Atributos) != 10 ||
		recurso.Ambitos["organizacion_ref"] != material.OrganizacionRef ||
		recurso.Ambitos["expediente_ref"] != material.ExpedienteRef ||
		recurso.Ambitos["fase_previa"] != string(orden.Preparacion.Expediente.FaseActual) ||
		recurso.Ambitos["estado_previo"] != string(orden.Preparacion.Expediente.EstadoActual) ||
		recurso.Atributos["version_expediente"] != strconv.FormatUint(material.VersionExpediente, 10) ||
		recurso.Atributos["configuracion_ref"] != orden.Configuracion.DefinicionRef ||
		recurso.Atributos["configuracion_version"] !=
			strconv.FormatUint(orden.Configuracion.DefinicionVersion, 10) ||
		recurso.Atributos["configuracion_huella_sha256"] != orden.Configuracion.DefinicionHuella ||
		recurso.Atributos["plantilla_ref"] != orden.Configuracion.Plantilla.PlantillaRef ||
		recurso.Atributos["plantilla_version"] !=
			strconv.FormatUint(orden.Configuracion.Plantilla.Version, 10) ||
		recurso.Atributos["plantilla_huella_sha256"] != orden.Configuracion.Plantilla.HuellaSHA256 ||
		recurso.Atributos["borrador_huella_sha256"] != orden.Borrador.HuellaSHA256() ||
		recurso.Atributos["ambito_idempotencia_hmac"] !=
			orden.Preparacion.AmbitoIdempotenciaHMAC ||
		recurso.Atributos["huella_peticion_hmac"] != orden.Preparacion.HuellaPeticionHMAC ||
		confirmacion.DecisionHuellaSHA256 != huellaDecision ||
		!orden.Evidencia.ConfirmacionV3.DentroDeVentanaEn(instante) {
		return ports.ErrPreparacionInformeJuridicoInvalida
	}
	return nil
}

func decodificarReciboInformeJuridico(
	contenido string,
) (ports.ReciboInformeJuridico, error) {
	var recibo ports.ReciboInformeJuridico
	if decodificarJSONEstricto([]byte(contenido), &recibo) != nil {
		return ports.ReciboInformeJuridico{},
			ports.ErrResultadoInformeJuridicoNoConfiable
	}
	recibo.ConfirmadaEn = normalizarInstantePostgreSQL(recibo.ConfirmadaEn)
	return recibo, nil
}

func normalizarErrorConfirmacionInformeJuridico(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, ports.ErrPreparacionInformeJuridicoInvalida) ||
		errors.Is(causa, ports.ErrResultadoInformeJuridicoNoConfiable) {
		return causa
	}
	return ports.ErrPersistenciaInformeJuridicoNoDisponible
}
