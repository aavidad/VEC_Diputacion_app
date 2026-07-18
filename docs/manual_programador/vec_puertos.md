# Nucleo VEC: puertos

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/vec/ports`

> Contratos hexagonales del nucleo VEC: autorizacion, auditoria, documental y almacen.

### Constantes

```go
const (
	AccionAlmacenEscribir               = almacencanonico.AccionEscribir
	AccionAlmacenLeer                   = almacencanonico.AccionLeer
	AccionAlmacenPrepararCargaDirecta   = almacencanonico.AccionPrepararCargaDirecta
	AccionAlmacenConfirmarCargaDirecta  = almacencanonico.AccionConfirmarCargaDirecta
	AccionAlmacenAbandonarCargaDirecta  = almacencanonico.AccionAbandonarCargaDirecta
	AccionAlmacenPromover               = almacencanonico.AccionPromover
	AccionAlmacenAplicarRetencion       = almacencanonico.AccionAplicarRetencion
	AccionAlmacenInmovilizar            = almacencanonico.AccionInmovilizar
	AccionAlmacenLevantarInmovilizacion = almacencanonico.AccionLevantarInmovilizacion
	AccionAlmacenEliminar               = almacencanonico.AccionEliminar
	AccionAlmacenAnalizarContenido      = almacencanonico.AccionAnalizarContenido
)
```

Las acciones del puerto forman una lista positiva cerrada. Son operaciones
tecnicas, no permisos de negocio, y nunca se infieren de la ruta, el rol o
la finalidad. Una autorizacion para una accion no habilita ninguna otra.

```go
const (
	ZonaAlmacenCuarentena = almacencanonico.ZonaCuarentena
	ZonaAlmacenAdmitida   = almacencanonico.ZonaAdmitida
)
const (
	MetodoCargaDirectaPUT  = almacencanonico.MetodoCargaDirectaPUT
	MetodoCargaDirectaPOST = almacencanonico.MetodoCargaDirectaPOST
)
const (
	ModuloFuentesAutoridad                                                          = "vec"
	TipoRecursoFuenteAutoridad                                                      = "fuente_autoridad_versionada"
	AccionConsultarFuenteAutoridadInterna                                           = "vec.fuentes_autoridad.consultar_interna"
	FinalidadConsultaInternaFuenteAutoridad                                         = "gobierno_fuentes_autoridad"
	CampoConsultaInternaFuenteAutoridad                                             = "fuente_autoridad"
	AtributoMotivoCatalogoIDConsultaAutoridad                                       = "motivo_catalogo_id"
	AtributoMotivoCatalogoVersionConsultaAutoridad                                  = "motivo_catalogo_version"
	AtributoMotivoCatalogoHuellaConsultaAutoridad                                   = "motivo_catalogo_huella_sha256"
	AtributoMotivoEntradaClaveConsultaAutoridad                                     = "motivo_entrada_clave"
	ResultadoConsultaFuenteEncontrada              ResultadoConsultaFuenteAutoridad = "encontrada"
	ResultadoConsultaFuenteNoEncontrada            ResultadoConsultaFuenteAutoridad = "no_encontrada"
)
const (
	PrefijoReferenciaSolicitudFuenteAutoridad = "solicitud:fuente_autoridad:"
	PrefijoReferenciaOperacionFuenteAutoridad = "operacion:fuente_autoridad:"
)
const (
	EsquemaContextoOperacionAlmacenV1 = "vec.almacen.contexto-operacion.v1"

	// Acciones de negocio de lista positiva. Se duplican deliberadamente en
	// este puerto estable para no importar la capa de aplicacion ni el modulo
	// bolsa desde el nucleo VEC.
	AccionNegocioPrepararCargaDocumental     = "vec.documentos.carga.preparar"
	AccionNegocioConfirmarCargaDocumental    = "vec.documentos.carga.confirmar"
	AccionNegocioAnalizarCargaDocumental     = "vec.documentos.carga.analizar"
	AccionNegocioPromoverCargaDocumental     = "vec.documentos.carga.promover"
	AccionNegocioCustodiarDecisionBaremacion = "bolsa.decision.custodiar"
	AccionNegocioCustodiarDocumentoFirmado   = "bolsa.decision.firma.documento.custodiar"
	AccionNegocioRetenerDocumentoFirmado     = "bolsa.decision.firma.documento.retener"

	// Atributos que deben formar parte del RecursoAutorizable evaluado por el
	// PDP. Asi una decision no puede emplearse para acuñar capacidades sobre
	// otra operacion, carga, objeto o efecto despues de ser emitida.
	AtributoAlmacenOperacionRef        = "almacen_operacion_ref"
	AtributoAlmacenCargaRef            = "almacen_carga_ref"
	AtributoAlmacenClasificacion       = "almacen_clasificacion"
	AtributoAlmacenSujetoSeudonimoHMAC = "almacen_sujeto_seudonimo_hmac"
	AtributoAlmacenHuellaSolicitudHMAC = "almacen_huella_solicitud_hmac"
	AtributoAlmacenEfectoRef           = "almacen_efecto_ref"
	AtributoAlmacenObjetoRef           = "almacen_objeto_ref"
	AtributoAlmacenObjetoVersion       = "almacen_objeto_version"
	// Este atributo debe estar en el recurso antes de consultar al PDP.
	AtributoAlmacenHuellaManifiestoSHA256 = "almacen_manifiesto_generacion_sha256"
)
const (
	PasoAlmacenPrepararCargaDirecta  = almacencanonico.PasoPrepararCargaDirecta
	PasoAlmacenAbandonarCargaDirecta = almacencanonico.PasoAbandonarCargaDirecta
	PasoAlmacenConfirmarCargaDirecta = almacencanonico.PasoConfirmarCargaDirecta
	PasoAlmacenLeerParaAnalisis      = almacencanonico.PasoLeerParaAnalisis
	PasoAlmacenAnalizarContenido     = almacencanonico.PasoAnalizarContenido
	PasoAlmacenPromover              = almacencanonico.PasoPromover
	PasoAlmacenCustodiarDecision     = almacencanonico.PasoCustodiarDecision
	PasoAlmacenCustodiarFirmado      = almacencanonico.PasoCustodiarFirmado
	PasoAlmacenRetenerFirmado        = almacencanonico.PasoRetenerFirmado
)
const (
	EsquemaSolicitudVinculadaAutorizacionEjecucionDocumentalV4  = "vec.documentos.autorizacion-ejecucion.solicitud-vinculada.v4"
	EsquemaSolicitudAplicacionAutorizacionEjecucionDocumentalV4 = "vec.documentos.autorizacion-ejecucion.solicitud-aplicacion.v4"

	// AccionEjecutarPlanDocumentalV4 es deliberadamente una unica accion
	// positiva. No se acepta una accion aportada libremente ni un comodin.
	AccionEjecutarPlanDocumentalV4 = "vec.documentos.ejecucion.ejecutar_plan_v4"

	AtributoAutorizacionDocumentalEfectoRef        = "ejecucion_documental_efecto_ref"
	AtributoAutorizacionDocumentalHuellaPlanSHA256 = "ejecucion_documental_huella_plan_sha256"
)
const (
	AccionCrearCatalogoConfigurable      = "vec.catalogos.crear"
	AccionActualizarCatalogoConfigurable = "vec.catalogos.actualizar"
	AccionPublicarCatalogoConfigurable   = "vec.catalogos.publicar"
	AccionRetirarCatalogoConfigurable    = "vec.catalogos.retirar"
)
const (
	AccionProtegerCodigoCotejo         = "vec.documentos.cotejo.custodia.proteger"
	AccionRecuperarCodigoCotejo        = "vec.documentos.cotejo.custodia.recuperar"
	AccionEliminarCodigoCotejoHuerfano = "vec.documentos.cotejo.custodia.eliminar_huerfano"
)
const (
	EsquemaManifiestoEjecucionDocumentalV3 = "vec.documentos.manifiesto-ejecucion.v3"
	EsquemaEvidenciaRenderizadoV3          = "vec.documentos.evidencia-renderizado.v3"
	AlgoritmoSelloEvidenciaHMACSHA256V3    = documentalcanonico.AlgoritmoHMACSHA256V3
	AudienciaSelloEvidenciaRenderizadoV3   = documentalcanonico.AudienciaSelloEvidenciaRenderizadoV3
	AudienciaAtestacionTokenCercadoV3      = documentalcanonico.AudienciaTokenCercadoV3
	AudienciaAtestacionInicioEfectoV3      = documentalcanonico.AudienciaInicioEfectoV3
	AudienciaAtestacionReclamacionV3       = documentalcanonico.AudienciaReclamacionDespachoV3
	AudienciaComprobacionOrdenDespachoV3   = documentalcanonico.AudienciaComprobacionOrdenDespachoV3
	ContextoAtestacionTokenCercadoV3       = documentalcanonico.ContextoTokenCercadoV3
	ContextoAtestacionInicioEfectoV3       = documentalcanonico.ContextoInicioEfectoV3
	ContextoAtestacionReclamacionV3        = documentalcanonico.ContextoReclamacionDespachoV3
	ContextoComprobacionOrdenDespachoV3    = documentalcanonico.ContextoComprobacionOrdenDespachoV3
)
const (
	// EsquemaHuellaDecisionAutorizacionReforzadaV1 identifica tanto el dominio
	// criptografico como el formato canonico. Cambiar el significado o los
	// campos de la representacion exige publicar otro esquema.
	EsquemaHuellaDecisionAutorizacionReforzadaV1 = "vec.autorizacion.decision.reforzada.v1.autenticacion-actor"
	// EsquemaHuellaDecisionAutorizacionReforzadaV2 añade los compromisos
	// versionados de solicitud completa y motivo verificable por separado.
	EsquemaHuellaDecisionAutorizacionReforzadaV2 = "vec.autorizacion.decision.reforzada.v2.solicitud-ligada"
)
const (
	EsquemaManifiestoGeneracionDocumentalV1 = "vec.documentos.manifiesto-generacion.v1"
)
const (
	EsquemaCanonizacionEntradaNeutralDocumentalV1 = documentalcanonico.EsquemaCanonizacionEntradaNeutralV1
	EsquemaPruebaEscrituraAlmacenDocumentalV1     = "vec.documentos.prueba-escritura-almacen.v1"
	EsquemaPruebaEscrituraAlmacenDocumentalV2     = "vec.documentos.prueba-escritura-almacen.v2"
)
const (
	CanalAuditoriaCobroInterno           = pagoscanonicos.CanalAuditoriaCobroInterno
	CanalAuditoriaCobroPasarela          = pagoscanonicos.CanalAuditoriaCobroPasarela
	CanalAuditoriaCobroProcesoAutomatico = pagoscanonicos.CanalAuditoriaCobroProcesoAutomatico
)
const (
	EventoCobroOrdenCreada                    = pagoscanonicos.EventoCobroOrdenCreada
	EventoCobroOperacionEnviada               = pagoscanonicos.EventoCobroOperacionEnviada
	EventoCobroResultadoPendiente             = pagoscanonicos.EventoCobroResultadoPendiente
	EventoCobroResultadoDesconocido           = pagoscanonicos.EventoCobroResultadoDesconocido
	EventoCobroConfirmado                     = pagoscanonicos.EventoCobroConfirmado
	EventoCobroRechazado                      = pagoscanonicos.EventoCobroRechazado
	EventoCobroCancelado                      = pagoscanonicos.EventoCobroCancelado
	EventoCobroCaducado                       = pagoscanonicos.EventoCobroCaducado
	EventoCobroConciliado                     = pagoscanonicos.EventoCobroConciliado
	EventoCobroDevolucionSolicitada           = pagoscanonicos.EventoCobroDevolucionSolicitada
	EventoCobroDevolucionResultadoPendiente   = pagoscanonicos.EventoCobroDevolucionResultadoPendiente
	EventoCobroDevolucionResultadoDesconocido = pagoscanonicos.EventoCobroDevolucionResultadoDesconocido
	EventoCobroDevolucionRechazada            = pagoscanonicos.EventoCobroDevolucionRechazada
	EventoCobroDevuelto                       = pagoscanonicos.EventoCobroDevuelto
	EventoCobroDevolucionConciliada           = pagoscanonicos.EventoCobroDevolucionConciliada
	EventoCobroIncidenciaDetectada            = pagoscanonicos.EventoCobroIncidenciaDetectada
	EventoCobroEvidenciaAdicional             = pagoscanonicos.EventoCobroEvidenciaAdicional
)
const (
	EsquemaPerfilCapacidadesAlmacenMaterialV2 = recibomaterial.EsquemaPerfil
	EsquemaInstantaneaObjetoMaterialV2        = recibomaterial.EsquemaInstantanea
	EsquemaReciboEscrituraObjetoMaterialV2    = recibomaterial.EsquemaRecibo

	VersionEsquemaMaterialAlmacenV2 uint16 = recibomaterial.EsquemaVersion
)
const EsquemaHuellaRecursoBaseCargaDocumentalV1 = "vec.carga-documental.recurso-base.v1"
const MaximoVersionesPaginaFuenteAutoridad uint16 = 100
const MetodoHandoffCobroPOSTFormulario = pagoscanonicos.MetodoHandoffCobroPOSTFormulario
const VentanaMaximaFrescuraContextoActorV1 = 5 * time.Second
```

VentanaMaximaFrescuraContextoActorV1 acota el tiempo entre el instante
solicitado y la confirmacion durable. No es una gracia de vigencia: tanto
el adaptador como el servicio deben exigir ademas que perfil y referencias
sigan activos en el instante autoritativo de su comprobacion.

```go
const VigenciaMaximaAtestacionActoFuenteAutoridad = 5 * time.Minute
```

### Variables

```go
var (
	ErrSolicitudAlmacenInvalida              = almacencanonico.ErrSolicitudAlmacenInvalida
	ErrObjetoAlmacenNoEncontrado             = errors.New("vec: objeto de almacen no encontrado")
	ErrIntegridadObjetoAlmacen               = errors.New("vec: integridad del objeto de almacen no valida")
	ErrLimiteObjetoAlmacenExcedido           = errors.New("vec: limite del objeto de almacen excedido")
	ErrIdempotenciaAlmacenReutilizada        = errors.New("vec: idempotencia de almacen reutilizada para otra operacion")
	ErrCapacidadAlmacenNoDisponible          = errors.New("vec: capacidad de almacen no disponible")
	ErrTransicionZonaAlmacenNoPermitida      = errors.New("vec: transicion de zona de almacen no permitida")
	ErrRetencionObjetoAlmacenVigente         = errors.New("vec: retencion del objeto de almacen vigente")
	ErrObjetoAlmacenInmovilizado             = errors.New("vec: objeto de almacen inmovilizado")
	ErrObjetoAlmacenEliminado                = errors.New("vec: objeto de almacen eliminado")
	ErrSesionCargaDirectaNoValida            = errors.New("vec: sesion de carga directa no valida")
	ErrConfirmacionCargaDirectaNoDisponible  = errors.New("vec: confirmacion de carga directa no disponible")
	ErrInstruccionesCargaDirectaNoValidas    = almacencanonico.ErrInstruccionesCargaDirectaNoValidas
	ErrSerializacionCargaDirectaProhibida    = errors.New("vec: serializacion accidental de carga directa prohibida")
	ErrSelladoIdempotenciaCargaNoDisponible  = errors.New("vec: sellado de idempotencia de carga no disponible")
	ErrReciboCargaDirectaNoValido            = almacencanonico.ErrReciboCargaDirectaNoValido
	ErrReciboCargaDirectaNoDisponible        = errors.New("vec: verificacion del recibo de carga directa no disponible")
	ErrAtestacionReciboCargaDirectaNoValida  = errors.New("vec: atestacion de consumo de carga directa no valida")
	ErrRegistroReciboCargaDirectaConflicto   = errors.New("vec: indice de recibo de carga directa ya registrado")
	ErrConsumoReciboCargaDirectaDenegado     = errors.New("vec: consumo de recibo de carga directa denegado")
	ErrSerializacionReciboCargaProhibida     = almacencanonico.ErrSerializacionReciboCargaProhibida
	ErrSerializacionSeudonimizacionProhibida = almacencanonico.ErrSerializacionSeudonimizacionProhibida
	ErrSeudonimizacionAlmacenNoDisponible    = almacencanonico.ErrSeudonimizacionAlmacenNoDisponible
)
var (
	ErrCapacidadAnalisisContenidoNoDisponible = errors.New("vec: capacidad de analisis de contenido no disponible")
	ErrSolicitudAnalisisContenidoInvalida     = errors.New("vec: solicitud de analisis de contenido invalida")
	ErrResultadoAnalisisContenidoInvalido     = errors.New("vec: resultado de analisis de contenido invalido")
	ErrAnalizadorContenidoNoDisponible        = errors.New("vec: analizador de contenido no disponible")
)
var (
	ErrSolicitudFirmaAtestacionInvalida = errors.New("vec: solicitud de firma de atestacion invalida")
	ErrFirmaAtestacionNoDisponible      = errors.New("vec: firma de atestacion no disponible")
	ErrResultadoFirmaAtestacionInvalido = errors.New("vec: resultado de firma de atestacion invalido")
)
var (
	ErrSolicitudComprobacionActoAutoridadInvalida = errors.New("vec: solicitud de comprobacion de acto de autoridad invalida")
	ErrAtestacionActoAutoridadInvalida            = errors.New("vec: atestacion de acto de autoridad invalida")
	ErrActoFuenteAutoridadNoDisponible            = errors.New("vec: acto de fuente de autoridad no disponible")
	ErrAtestacionActoAutoridadConsumida           = errors.New("vec: atestacion de acto de autoridad consumida")
	ErrSerializacionAtestacionActoAutoridad       = errors.New("vec: serializacion de atestacion de acto de autoridad prohibida")
)
var (
	ErrConsultaFuenteAutoridadInvalida = errors.New("vec: consulta de fuente de autoridad invalida")
	ErrFuenteAutoridadNoEncontrada     = errors.New("vec: fuente de autoridad no encontrada")
)
var (
	ErrConsultaInternaFuenteAutoridadInvalida = errors.New("vec: consulta interna gobernada de fuente de autoridad invalida")
	ErrReciboConsultaFuenteAutoridadInvalido  = errors.New("vec: recibo de consulta de fuente de autoridad invalido")
	ErrSerializacionGobiernoFuenteAutoridad   = errors.New("vec: serializacion de gobierno de fuente de autoridad prohibida")
)
var (
	ErrEstadoFuenteAutoridadInvalido        = errors.New("vec: estado exacto de fuente de autoridad invalido")
	ErrOperacionFuenteAutoridadInvalida     = errors.New("vec: operacion de fuente de autoridad invalida")
	ErrOperacionFuenteAutoridadNoEncontrada = errors.New("vec: operacion de fuente de autoridad no encontrada")
	ErrSerializacionOperacionAutoridad      = errors.New("vec: serializacion de operacion de fuente de autoridad prohibida")
)
var (
	ErrReferenciaGeneradaFuenteAutoridadInvalida = errors.New("vec: referencia generada de fuente de autoridad invalida")
	ErrGeneracionReferenciaFuenteAutoridad       = errors.New("vec: no se pudo generar una referencia de fuente de autoridad")
	ErrColisionReferenciaFuenteAutoridad         = errors.New("vec: colision de referencia de fuente de autoridad")
	ErrSerializacionReferenciaAutoridad          = errors.New("vec: serializacion de referencia de fuente de autoridad prohibida")
)
var (
	ErrAsignacionPerfilNoEncontrada     = errors.New("vec: asignacion de perfil no encontrada")
	ErrVersionRolNoEncontrada           = errors.New("vec: version de rol no encontrada")
	ErrDecisionAutorizacionNoEncontrada = errors.New("vec: decision de autorizacion no encontrada")
	ErrVersionAutorizacionYaExiste      = errors.New("vec: version de autorizacion ya existe")
	ErrSecuenciaVersionInvalida         = errors.New("vec: secuencia de version de autorizacion invalida")
	ErrFuenteAutorizacionNoDisponible   = errors.New("vec: fuente de autorizacion no disponible")
	ErrRegistroDecisionNoDisponible     = errors.New("vec: registro de decisiones no disponible")
	ErrRegistroDenegacionNoDisponible   = errors.New("vec: registro de denegaciones no disponible")
	ErrInstantaneaAutorizacionObsoleta  = errors.New("vec: instantanea de autorizacion obsoleta")
)
var (
	// ErrAutorizacionAlmacenInvalida expresa siempre una denegacion cerrada:
	// ausencia, ambiguedad, caducidad o cualquier dato no reconocido tienen el
	// mismo resultado y nunca se convierten en una concesion parcial.
	ErrAutorizacionAlmacenInvalida = errors.New("vec: autorizacion de almacen invalida")
	// ErrSerializacionContextoAlmacenProhibida evita que la capacidad o su
	// proyeccion interna crucen accidentalmente una frontera HTTP, de mensajes
	// o de persistencia generica.
	ErrSerializacionContextoAlmacenProhibida = almacencanonico.ErrSerializacionContextoAlmacenProhibida
)
var (
	// ErrAutorizacionEjecucionDocumentalV4Invalida representa una denegacion
	// cerrada. Ausencia, ambiguedad, caducidad o cualquier discrepancia producen
	// el mismo error y nunca una concesion parcial.
	ErrAutorizacionEjecucionDocumentalV4Invalida = errors.New("vec: autorizacion de ejecucion documental v4 invalida")
	// ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida impide usar una
	// solicitud estructural o su proyeccion transaccional como credencial de red.
	ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida = errors.New("vec: serializacion de autorizacion de ejecucion documental v4 prohibida")
)
var (
	ErrRepositorioCargasNoDisponible        = errors.New("vec: repositorio de cargas documentales no disponible")
	ErrReservaCargaDocumentalInvalida       = errors.New("vec: reserva de carga documental invalida")
	ErrReservaCargaDocumentalOcupada        = errors.New("vec: reserva de carga documental ocupada")
	ErrCargaDocumentalNoEncontrada          = errors.New("vec: carga documental no encontrada")
	ErrConflictoVersionCargaDocumental      = errors.New("vec: conflicto de version de carga documental")
	ErrConfirmacionCargaDocumentalInvalida  = errors.New("vec: confirmacion de carga documental invalida")
	ErrManifiestoPreparacionNoEncontrado    = errors.New("vec: manifiesto de preparacion de carga directa no encontrado")
	ErrConfirmacionCargaDocumentalPendiente = errors.New("vec: confirmacion de carga documental pendiente de reconciliacion")
	ErrDecisionPreparacionCargaNoDisponible = errors.New("vec: decision de preparacion de carga no disponible")
	ErrDecisionPreparacionCargaYaConsumida  = errors.New("vec: decision de preparacion de carga ya consumida")
	ErrRecursoBaseCargaDocumentalInvalido   = errors.New("vec: recurso base de carga documental invalido")
	ErrSerializacionTokenReservaProhibida   = errors.New("vec: serializacion de token de reserva prohibida")
)
var (
	ErrCatalogoNoEncontrado        = errors.New("vec: catalogo configurable no encontrado")
	ErrVersionCatalogoYaExiste     = errors.New("vec: version de catalogo ya existe")
	ErrRevisionCatalogoEnConflicto = errors.New("vec: revision de catalogo en conflicto")
	ErrSecuenciaCatalogoInvalida   = errors.New("vec: secuencia de catalogo invalida")
)
var (
	ErrFuenteContextoActorNoDisponible             = errors.New("vec: fuente de contexto de actor no disponible")
	ErrResolutorRegistroContextoActorNoDisponible  = errors.New("vec: resolutor y registro de contexto de actor no disponible")
	ErrGeneradorOperacionContextoActorNoDisponible = errors.New("vec: generador de operacion de contexto de actor no disponible")
	ErrSolicitudRegistroContextoActorV1Invalida    = errors.New("vec: solicitud de registro de contexto de actor v1 invalida")
	ErrConfirmacionRegistroContextoActorV1Invalida = errors.New("vec: confirmacion de registro de contexto de actor v1 invalida")
)
var (
	ErrPoliticaCotejoNoEncontrada               = errors.New("vec: politica de cotejo no encontrada")
	ErrVersionPoliticaCotejoYaExiste            = errors.New("vec: version de politica de cotejo ya existente")
	ErrRevisionPoliticaCotejoConflicto          = errors.New("vec: revision de politica de cotejo en conflicto")
	ErrSecuenciaPoliticaCotejoInvalida          = errors.New("vec: secuencia de politica de cotejo invalida")
	ErrCodigoCotejoNoEncontrado                 = errors.New("vec: codigo de cotejo no encontrado")
	ErrCodigoCotejoYaExiste                     = errors.New("vec: codigo de cotejo ya existente")
	ErrDocumentoConCodigoCotejo                 = errors.New("vec: version documental ya vinculada a un codigo de cotejo")
	ErrIndiceCodigoCotejoYaExiste               = errors.New("vec: indice de codigo de cotejo ya existente")
	ErrIndicesCodigoCotejoAmbiguos              = errors.New("vec: varios codigos coinciden con los indices de cotejo")
	ErrRevisionCodigoCotejoConflicto            = errors.New("vec: revision de codigo de cotejo en conflicto")
	ErrClaveIdempotenciaCotejoInvalida          = errors.New("vec: clave de idempotencia de cotejo invalida")
	ErrClaveIdempotenciaCotejoReutilizada       = errors.New("vec: clave de idempotencia de cotejo reutilizada")
	ErrEmisionCodigoCotejoEnCurso               = errors.New("vec: emision de codigo de cotejo en curso")
	ErrReservaCodigoCotejoNoValida              = errors.New("vec: reserva de codigo de cotejo no valida")
	ErrSerializacionTokenReservaCotejoProhibida = errors.New("vec: serializacion del token de reserva de cotejo prohibida")
	ErrMaterialCodigoCotejoInvalido             = errors.New("vec: material de codigo de cotejo invalido")
	ErrValorCodigoCotejoNoDisponible            = errors.New("vec: valor protegido del codigo de cotejo no disponible")
	ErrSerializacionCodigoCotejoProhibida       = errors.New("vec: serializacion del secreto de cotejo prohibida")
	ErrEvidenciaEmisionNoEncontrada             = errors.New("vec: evidencia de emision no encontrada")
	ErrContextoCustodiaCotejoInvalido           = errors.New("vec: contexto de custodia de codigo de cotejo invalido")
	ErrSerializacionContextoCotejoProhibida     = errors.New("vec: serializacion del contexto de custodia de cotejo prohibida")
)
var (
	ErrPlantillaDocumentoNoEncontrada              = errors.New("vec: plantilla documental no encontrada")
	ErrVersionPlantillaYaExiste                    = errors.New("vec: la version de plantilla ya existe")
	ErrDocumentoNoEncontrado                       = errors.New("vec: documento no encontrado")
	ErrDocumentoYaExiste                           = errors.New("vec: documento ya existe")
	ErrContenidoDocumentoNoEncontrado              = errors.New("vec: contenido documental no encontrado")
	ErrHuellaContenidoNoCoincide                   = errors.New("vec: la huella del contenido no coincide")
	ErrLimiteLecturaExcedido                       = errors.New("vec: limite de lectura documental excedido")
	ErrDocumentoLogicoNoEncontrado                 = errors.New("vec: documento logico no encontrado")
	ErrRepresentacionNoEncontrada                  = errors.New("vec: representacion documental no encontrada")
	ErrClaveIdempotenciaInvalida                   = errors.New("vec: clave de idempotencia documental invalida")
	ErrClaveIdempotenciaReutilizada                = errors.New("vec: clave de idempotencia reutilizada para otra solicitud")
	ErrGeneracionDocumentoEnCurso                  = errors.New("vec: generacion documental en curso")
	ErrReservaDocumentoNoValida                    = errors.New("vec: reserva documental no valida")
	ErrSerializacionTokenReservaDocumentoProhibida = errors.New("vec: serializacion del token de reserva documental prohibida")
	ErrDecisionAutorizacionConsumida               = errors.New("vec: decision de autorizacion ya consumida por otro efecto")
)
var (
	ErrReservaEfectoGeneracionDocumentalInvalida = errors.New("vec: reserva de efecto documental invalida")
	ErrPasoGeneracionDocumentalIndeterminado     = errors.New("vec: resultado de paso documental indeterminado")
)
var (
	ErrCompromisoEjecucionDocumentalInvalido  = errors.New("vec: compromiso de ejecucion documental invalido")
	ErrSobreReciboEjecucionDocumentalInvalido = errors.New("vec: sobre de recibo de ejecucion documental invalido")
	ErrIdentidadEjecucionDocumentalInvalida   = errors.New("vec: identidad de ejecucion documental invalida")
	ErrReciboEjecucionDocumentalInvalido      = errors.New("vec: recibo de ejecucion documental invalido")
)
var (
	ErrManifiestoEjecucionDocumentalV3Invalido = errors.New("vec: manifiesto de ejecucion documental v3 invalido")
	ErrReservaEjecucionDocumentalV3Invalida    = errors.New("vec: reserva de ejecucion documental v3 invalida")
	ErrTokenCercadoDocumentalV3Invalido        = documentalcanonico.ErrTokenCercadoDocumentalV3Invalido
	ErrTransicionEjecucionDocumentalV3Invalida = errors.New("vec: transicion de ejecucion documental v3 invalida")
	ErrReconciliacionDocumentalV3Invalida      = documentalcanonico.ErrReconciliacionDocumentalV3Invalida
	ErrSelloEvidenciaDocumentalV3Invalido      = documentalcanonico.ErrSelloEvidenciaDocumentalV3Invalido
	ErrSerializacionSecretoDocumentalV3        = documentalcanonico.ErrSerializacionSecretoDocumentalV3
	ErrReciboInicioDocumentalV3Invalido        = errors.New("vec: recibo durable de inicio documental v3 invalido")
	ErrOrdenDespachoDocumentalV3Invalida       = documentalcanonico.ErrOrdenDespachoDocumentalV3Invalida
)
var (
	// ErrEvidenciaUsoDecisionAutorizacionInvalida se devuelve siempre que no
	// pueda demostrarse de forma positiva que una evidencia representa una
	// decision reforzada, concedida, completa y vigente.
	ErrEvidenciaUsoDecisionAutorizacionInvalida = errors.New("vec: evidencia de uso de decision de autorizacion invalida")
	// ErrSerializacionEvidenciaUsoAutorizacionProhibida evita que la capacidad
	// opaca termine por accidente en codecs de transporte, trazas o HTTP.
	ErrSerializacionEvidenciaUsoAutorizacionProhibida = errors.New("vec: serializacion de evidencia de uso de autorizacion prohibida")
)
var (
	ErrDefinicionFlujoNoEncontrada      = errors.New("vec: definicion de flujo no encontrada")
	ErrVersionDefinicionFlujoYaExiste   = errors.New("vec: version de definicion de flujo ya existe")
	ErrRevisionDefinicionFlujoConflicto = errors.New("vec: revision de definicion de flujo en conflicto")
	ErrSecuenciaDefinicionFlujoInvalida = errors.New("vec: secuencia de definicion de flujo invalida")
	ErrInstanciaFlujoNoEncontrada       = errors.New("vec: instancia de flujo no encontrada")
	ErrInstanciaFlujoYaExiste           = errors.New("vec: instancia de flujo ya existe")
	ErrEntidadConInstanciaFlujo         = errors.New("vec: la entidad ya tiene una instancia para esta definicion")
	ErrRevisionInstanciaFlujoConflicto  = errors.New("vec: revision de instancia de flujo en conflicto")
	ErrDecisionReglaFlujoNoEncontrada   = errors.New("vec: decision de regla de flujo no encontrada")
	ErrDecisionReglaFlujoYaExiste       = errors.New("vec: decision de regla de flujo ya existe")
	ErrEvaluadorReglaFlujoNoDisponible  = errors.New("vec: evaluador de regla de flujo no disponible")
	ErrAprobacionFlujoNoVerificada      = errors.New("vec: aprobacion de flujo no verificada")
)
var (
	ErrConsultaFormatoDocumentalInvalida        = errors.New("vec: consulta de formato documental invalida")
	ErrDescriptorFormatoDocumentalInvalido      = errors.New("vec: descriptor de formato documental invalido")
	ErrCatalogoFormatosDocumentalesNoDisponible = errors.New("vec: catalogo de formatos documentales no disponible")
	ErrFormatoDocumentalNoResuelto              = errors.New("vec: formato documental no resuelto")
	ErrRenderizadorDocumentalNoDisponible       = errors.New("vec: renderizador documental no disponible")
	ErrMetadatoInstitucionalDocumentalInvalido  = errors.New("vec: metadato institucional documental invalido")
	ErrSituacionOperativaDocumentalInvalida     = errors.New("vec: situacion operativa documental invalida")
	ErrComponenteDocumentalAtestadoInvalido     = errors.New("vec: componente documental atestado invalido")
	ErrPoliticaInstitucionalDocumentalInvalida  = errors.New("vec: politica institucional documental invalida")
)
var (
	ErrManifiestoGeneracionDocumentalInvalido = errors.New("vec: manifiesto de generacion documental invalido")
	ErrSerializacionManifiestoProhibida       = errors.New("vec: serializacion de manifiesto documental prohibida")
)
var (
	ErrEntradaNeutralDocumentalInvalida         = errors.New("vec: entrada neutral documental invalida")
	ErrSumideroSalidaDocumentalInvalido         = errors.New("vec: sumidero de salida documental invalido")
	ErrLimiteSalidaDocumentalExcedido           = errors.New("vec: limite de salida documental excedido")
	ErrBloqueSalidaDocumentalExcedido           = errors.New("vec: bloque de salida documental excedido")
	ErrSumideroSalidaDocumentalCerrado          = errors.New("vec: sumidero de salida documental cerrado")
	ErrSalidaDocumentalVacia                    = errors.New("vec: salida documental vacia")
	ErrEscrituraSalidaDocumentalIncompleta      = errors.New("vec: escritura de salida documental incompleta")
	ErrPruebaEscrituraAlmacenInvalida           = errors.New("vec: prueba de escritura en almacen invalida")
	ErrSerializacionMaterialDocumentalProhibida = errors.New("vec: serializacion de material documental protegido prohibida")
)
var (
	ErrPasarelaCobroNoDisponible          = errors.New("vec: pasarela de cobro no disponible")
	ErrCapacidadPasarelaCobroNoDisponible = pagoscanonicos.ErrCapacidadPasarelaCobroNoDisponible
	ErrSolicitudOperacionCobroInvalida    = pagoscanonicos.ErrSolicitudOperacionCobroInvalida
	ErrInicioOperacionCobroInvalido       = pagoscanonicos.ErrInicioOperacionCobroInvalido
	ErrReferenciaOperacionCobroInvalida   = pagoscanonicos.ErrReferenciaOperacionCobroInvalida
	ErrNotificacionCobroInvalida          = pagoscanonicos.ErrNotificacionCobroInvalida
	ErrSolicitudDevolucionCobroInvalida   = pagoscanonicos.ErrSolicitudDevolucionCobroInvalida
	ErrSolicitudConciliacionCobroInvalida = pagoscanonicos.ErrSolicitudConciliacionCobroInvalida
	ErrResultadoPasarelaCobroInvalido     = pagoscanonicos.ErrResultadoPasarelaCobroInvalido
	ErrIdempotenciaCobroReutilizada       = errors.New("vec: idempotencia de cobro reutilizada con otros datos")
	ErrResultadoPasarelaCobroConflictivo  = errors.New("vec: resultados de pasarela de cobro incompatibles")
	ErrOrdenCobroNoEncontrada             = errors.New("vec: orden de cobro no encontrada")
	ErrOrdenCobroYaExiste                 = errors.New("vec: orden de cobro ya existente")
	ErrVersionOrdenCobroConflicto         = errors.New("vec: version de orden de cobro en conflicto")
	ErrHuellaOrdenCobroConflicto          = errors.New("vec: huella de orden de cobro en conflicto")
	ErrReservaOrdenCobroInvalida          = errors.New("vec: reserva de orden de cobro invalida")
	ErrReservaOrdenCobroCaducada          = errors.New("vec: reserva de orden de cobro caducada")
	ErrControlAutorizacionCobroConflicto  = errors.New("vec: control autoritativo de autorizacion de cobro en conflicto")
	ErrControlLiquidacionCobroConflicto   = errors.New("vec: control autoritativo de liquidacion en conflicto")
	ErrMutacionOrdenCobroInvalida         = pagoscanonicos.ErrMutacionOrdenCobroInvalida
	ErrNotificacionCobroYaConsumida       = errors.New("vec: notificacion de cobro ya consumida")
	ErrNotificacionCobroCaducada          = errors.New("vec: notificacion de cobro caducada")
)
var (
	ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida   = errors.New("vec: preimagen de recurso de autorizacion de ejecucion documental v4 invalida")
	ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida = errors.New("vec: serializacion general de preimagen de recurso de autorizacion v4 prohibida")
)
var (
	ErrSobreCriptograficoDocumentalCrudoV4Invalido  = errors.New("vec: sobre criptografico documental crudo v4 invalido")
	ErrPruebaCriptograficaDocumentalCrudaV4Invalida = errors.New("vec: prueba criptografica documental cruda v4 invalida")
	ErrSerializacionPruebaCriptograficaCrudaV4      = errors.New("vec: serializacion de prueba criptografica documental cruda v4 prohibida")
)
var (
	// ErrReciboEscrituraObjetoMaterialV2NoValido es deliberadamente opaco:
	// un dato ausente, un cruce de contexto y una prueba criptografica falsa
	// producen la misma denegacion cerrada.
	ErrReciboEscrituraObjetoMaterialV2NoValido = recibomaterial.ErrReciboNoValido
	ErrAtestacionMaterialAlmacenV2NoValida     = recibomaterial.ErrAtestacionNoValida
	ErrSerializacionMaterialAlmacenV2Prohibida = recibomaterial.ErrSerializacionProhibida
)
var (
	ErrRetoConsultaReconciliacionDocumentalV4Invalido = errors.New(
		"vec: reto de consulta de reconciliacion documental v4 invalido",
	)
	ErrConsultaReconciliacionDocumentalV4Invalida = errors.New(
		"vec: consulta de reconciliacion documental v4 invalida",
	)
	ErrRespuestaCrudaReconciliacionDocumentalV4Invalida = errors.New(
		"vec: respuesta cruda de reconciliacion documental v4 invalida",
	)
	ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida = errors.New(
		"vec: proyeccion de intento cas de reconciliacion documental v4 invalida",
	)
	ErrSerializacionReconciliacionDocumentalV4 = errors.New(
		"vec: serializacion generica de reconciliacion documental v4 prohibida",
	)
)
var ErrEjecucionDocumentalAtestadaV4NoDisponible = errors.New(
	"vec: ejecucion documental atestada v4 no disponible",
)
var ErrGeneracionReferenciaAutorizacionV2 = errors.New(
	"vec: no se pudo generar una referencia de autorizacion V2",
)
var ErrInteropNotEnabledV0 = errors.New("vec interop port not enabled v0")
var ErrOrdenRegistroAutorizacionSolicitudLigadaV2Invalida = errors.New(
	"vec: orden de registro de autorizacion ligada a solicitud invalida",
)
var ErrRevalidacionAutenticacionActorNoDisponible = errors.New("vec: revalidacion de autenticacion y actor no disponible")
var ErrSerializacionAtestacionAutorizacionV2Prohibida = errors.New(
	"vec: serializacion generica de atestacion de autorizacion V2 prohibida",
)
var ErrSerializacionTokenReservaCobroProhibida = errors.New(
	"vec: serializacion de token de reserva de cobro prohibida",
)
```

### Funciones

```go
func BytesCanonicosConfiguracionOrigenPasarelaCobro(o OrigenPasarelaCobroPublicado) ([]byte, error)
```

BytesCanonicosConfiguracionOrigenPasarelaCobro fija el origen, las rutas y
los campos publicados. Las listas son conjuntos y se ordenan en copias para
que su orden accidental no cambie la huella.

```go
func CalcularHuellaConfiguracionOrigenPasarelaCobro(o OrigenPasarelaCobroPublicado) (string, error)
func HuellaCumplimientosVaciosAutorizacionEjecucionDocumentalV4() string
```

HuellaCumplimientosVaciosAutorizacionEjecucionDocumentalV4 acompana a la
lista de obligaciones vacia; un mapa no vacio se deniega.

```go
func HuellaObligacionesVaciasAutorizacionEjecucionDocumentalV4() string
```

HuellaObligacionesVaciasAutorizacionEjecucionDocumentalV4 fija el unico
valor admitido mientras el flujo V4 no persista evidencia tipada de
cumplimiento y revocacion de obligaciones.

```go
func HuellaRecursoBaseCargaDocumental(recurso domain.RecursoAutorizable) (string, error)
```

HuellaRecursoBaseCargaDocumental fija el recurso ABAC anterior a cualquier
enriquecimiento tecnico. Incluye referencia, modulo, tipo y el contexto
canonico completo de ambitos y atributos. Una clave reservada no se elimina
silenciosamente: se rechaza para impedir que una entrada inyectada quede
blanqueada al calcular la huella.

```go
func MarcadorMetadatoInstitucionalNulo(marcador MarcadorMetadatoInstitucionalDocumental) bool
func MensajeCanonicoAtestacionInicioEfectoDocumentalV3(
	solicitud SolicitudIniciarEfectoDocumentalV3,
	inicioRef string,
	versionInicioCAS uint64,
	auditoriaInicioRef, outboxInicioRef, evidenciaOperacionRef string,
) ([]byte, error)
func MensajeCanonicoAtestacionReclamacionDespachoDocumentalV3(
	recibo ReciboInicioEfectoDocumentalV3Nominal,
	solicitud SolicitudReclamarOrdenDespachoDocumentalV3,
	versionReclamacionCAS uint64,
	auditoriaReclamacionRef, evidenciaOperacionRef string,
) ([]byte, error)
func PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
	resultado ResultadoConsultaFuenteAutoridad,
	estado ReferenciaEstadoFuenteAutoridad,
) (domain.AuditEntry, error)
```

PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad fija el outcome y
el snapshot exacto en la entrada que la barrera debe encadenar y persistir.
ID, secuencia y envolvente de integridad siguen vacios hasta el registro.

```go
func RecursoAutorizableConsultaInternaFuenteAutoridad(
	selector SelectorVersionFuenteAutoridad,
	motivo domain.ReferenciaEntradaCatalogo,
) (domain.RecursoAutorizable, error)
func ReferenciaMotivoConsultaFuenteAutoridadValida(
	referencia domain.ReferenciaEntradaCatalogo,
) bool
```

ReferenciaMotivoConsultaFuenteAutoridadValida restringe el identificador de
la entrada a un valor opaco. La semantica y cualquier etiqueta legible viven
exclusivamente en el catalogo gobernado, no en ordenes, decisiones ni logs.

```go
func RenderizadorDocumentalNulo(renderizador RenderizadorDocumentalPorPerfil) bool
func RepresentacionCanonicaDecisionAutorizacionReforzadaV1(
	decision domain.DecisionAutorizacion,
) ([]byte, error)
```

RepresentacionCanonicaDecisionAutorizacionReforzadaV1 devuelve el unico
perfil JSON comprometido por la huella de una decision reforzada. Esta
proyeccion estrecha existe para que los adaptadores duraderos no repliquen
el orden de conjuntos ni el formato UTC de microsegundo fijo.

A diferencia de NuevaEvidenciaUsoDecisionAutorizacion, no acredita que las
obligaciones hayan sido cumplidas ni convierte la decision en una capacidad
consumible. Por ello admite decisiones validas con obligaciones y solo debe
usarse para persistencia, cotejo o firma de la representacion.

```go
func RepresentacionCanonicaDecisionAutorizacionReforzadaV2(
	decision domain.DecisionAutorizacion,
) ([]byte, error)
```

RepresentacionCanonicaDecisionAutorizacionReforzadaV2 devuelve la proyeccion
apta para persistencia y cotejo. No contiene Motivo ni la referencia de
catalogo en claro, sino sus compromisos de integridad. El consumidor durable
debe releer la misma version publicada y cotejar su huella: este documento
no acredita por si solo la procedencia del catalogo.

```go
func ValidarManifiestoPreparacionParaConfirmacion(
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
	carga domain.CargaDocumental,
	contexto ContextoOperacionAlmacen,
	recursoBase domain.RecursoAutorizable,
) error
```

ValidarManifiestoPreparacionParaConfirmacion cruza los hechos persistidos
de la preparacion con la capacidad nueva de confirmar. No reconstruye el
contexto anterior ni interpreta DecisionRef como autoridad.

```go
func ValidarRecursoBaseManifiestoPreparacionCargaDocumental(
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
	carga domain.CargaDocumental,
	recursoBase domain.RecursoAutorizable,
) error
```

ValidarRecursoBaseManifiestoPreparacionCargaDocumental comprueba antes del
PDP que el contexto ABAC completo sigue siendo exactamente el preparado.
Referencia, modulo y tipo se contrastan ademas de la huella para que esta no
pueda emplearse como sustituto ambiguo de la identidad del recurso.

```go
func ValidarResultadoCargaDirectaConManifiesto(
	resultado ResultadoOperacionObjeto,
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
	carga domain.CargaDocumental,
	confirmacion SolicitudConfirmarCargaDirecta,
	capacidades CapacidadesAlmacenObjetos,
	recursoBase domain.RecursoAutorizable,
) error
```

ValidarResultadoCargaDirectaConManifiesto valida la respuesta contra hechos
historicos durables y la capacidad de confirmacion actual. Deliberadamente
no recibe ni fabrica una SolicitudPrepararCargaDirecta.

```go
func VerificarCapacidadesAlmacen(capacidades CapacidadesAlmacenObjetos, requisitos RequisitosAlmacenObjetos) error
func VerificarCapacidadesAnalizadorContenido(
	capacidades CapacidadesAnalizadorContenido,
	requisitos RequisitosAnalizadorContenido,
) error
func VincularRecursoGeneracionDocumental(
	recurso domain.RecursoAutorizable,
	manifiesto ManifiestoGeneracionDocumental,
	vinculos VinculosOperacionAlmacen,
) (domain.RecursoAutorizable, error)
```

VincularRecursoGeneracionDocumental crea la copia exacta que debe enviarse
al PDP. Rechaza atributos reservados preexistentes para impedir que el
llamador o una entrada externa sombreen los vinculos calculados.

### Tipos

```go
type AccionProyectadaReconciliacionDocumentalV4 string

const (
	// Una firma autentica al declarante, pero la respuesta V4 no contiene un
	// resultado remoto completo ni una prueba de inexistencia. Por ello ningun
	// estado permite confirmar o abandonar: los tres solo proyectan evidencia.
	AccionReconciliacionDocumentalV4RegistrarSoloEvidencia AccionProyectadaReconciliacionDocumentalV4 = "registrar_solo_evidencia"
)
func (a AccionProyectadaReconciliacionDocumentalV4) Valida() bool

type ActivacionEjecucionDocumentalV3Nominal struct {
	Token    TokenCercadoEjecucionDocumentalV3Nominal
	Repetida bool
}

func (a ActivacionEjecucionDocumentalV3Nominal) ValidarContra(s SolicitudActivarEjecucionDocumentalV3) error

type AlgoritmoAtestacionMaterialAlmacenV2 string
```

AlgoritmoAtestacionMaterialAlmacenV2 identifica el formato que el
verificador debe comprobar. La forma valida nunca sustituye la verificacion.

```go
const (
	AlgoritmoAtestacionMaterialHMACSHA256 AlgoritmoAtestacionMaterialAlmacenV2 = "hmac-sha-256"
	AlgoritmoAtestacionMaterialCOSESign1  AlgoritmoAtestacionMaterialAlmacenV2 = "cose-sign1"
)
type AlmacenContenidoDocumento interface {
	GuardarContenido(context.Context, SolicitudGuardarContenido) (ContenidoDocumentoGuardado, error)
	LeerContenido(context.Context, SolicitudLeerContenido) (ContenidoDocumentoLeido, error)
}
```

AlmacenContenidoDocumento es implementable por S3 compatible, filesystem
cifrado o un gestor documental. La referencia devuelta debe ser opaca y
estable; las URL temporales solo se crean al descargar.

```go
type AlmacenObjetos interface {
	Capacidades(context.Context) (CapacidadesAlmacenObjetos, error)
	Escribir(context.Context, SolicitudEscribirObjeto) (ResultadoOperacionObjeto, error)
	Abrir(context.Context, SolicitudAbrirObjeto) (LecturaObjetoAlmacen, error)
	Promover(context.Context, SolicitudPromoverObjeto) (ResultadoOperacionObjeto, error)
	AplicarRetencion(context.Context, SolicitudRetenerObjeto) (ResultadoOperacionObjeto, error)
	Inmovilizar(context.Context, SolicitudInmovilizarObjeto) (ResultadoOperacionObjeto, error)
	LevantarInmovilizacion(context.Context, SolicitudLevantarInmovilizacionObjeto) (ResultadoOperacionObjeto, error)
	Eliminar(context.Context, SolicitudEliminarObjeto) (EvidenciaOperacionAlmacen, error)
}
```

AlmacenObjetos es el puerto estable del nucleo. Filesystem, S3, una cabina,
un gestor documental o una nube privada se conectan detras de este contrato.
No expone rutas, buckets ni operaciones de listado.

```go
type AnalizadorContenido interface {
	Capacidades(context.Context) (CapacidadesAnalizadorContenido, error)
	Analizar(context.Context, SolicitudAnalizarContenido) (ResultadoAnalisisContenido, error)
}

type AtestacionActoFuenteAutoridad struct {
	// Has unexported fields.
}

func NuevaAtestacionActoFuenteAutoridad(
	solicitud SolicitudComprobarActoFuenteAutoridad,
	datos DatosAtestacionActoFuenteAutoridad,
) (AtestacionActoFuenteAutoridad, error)

func (a AtestacionActoFuenteAutoridad) DatosParaConsumo() (
	DatosAtestacionActoFuenteAutoridad,
	error,
)

func (a AtestacionActoFuenteAutoridad) Evidencia() (
	domain.EvidenciaActoFuenteAutoridad,
	error,
)

func (a AtestacionActoFuenteAutoridad) Format(estado fmt.State, _ rune)

func (a AtestacionActoFuenteAutoridad) GoString() string

func (*AtestacionActoFuenteAutoridad) GobDecode([]byte) error

func (AtestacionActoFuenteAutoridad) GobEncode() ([]byte, error)

func (a AtestacionActoFuenteAutoridad) LogValue() slog.Value

func (AtestacionActoFuenteAutoridad) MarshalBinary() ([]byte, error)

func (AtestacionActoFuenteAutoridad) MarshalJSON() ([]byte, error)

func (AtestacionActoFuenteAutoridad) MarshalText() ([]byte, error)

func (AtestacionActoFuenteAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (AtestacionActoFuenteAutoridad) String() string

func (*AtestacionActoFuenteAutoridad) UnmarshalBinary([]byte) error

func (*AtestacionActoFuenteAutoridad) UnmarshalJSON([]byte) error

func (*AtestacionActoFuenteAutoridad) UnmarshalText([]byte) error

func (*AtestacionActoFuenteAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (a AtestacionActoFuenteAutoridad) ValidarPara(
	solicitud SolicitudComprobarActoFuenteAutoridad,
	instante time.Time,
) error

type AtestacionAutorizacionV1 struct {
	// Has unexported fields.
}
```

AtestacionAutorizacionV1 agrupa la cabecera preseleccionada y el resultado
opaco del firmante. No otorga permiso por si sola: registro y consumidor
deben verificarla y revalidar la decision en su propia transaccion.

```go
func NuevaAtestacionAutorizacionV1(
	solicitud SolicitudFirmaAtestacionAutorizacionV1,
	resultado ResultadoFirmaAtestacionAutorizacionV1,
) (AtestacionAutorizacionV1, error)

func (a AtestacionAutorizacionV1) Cabecera() (domain.CabeceraAtestacionAutorizacionV1, error)

func (a AtestacionAutorizacionV1) Resultado() (ResultadoFirmaAtestacionAutorizacionV1, error)

func (a AtestacionAutorizacionV1) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV1,
) error

type AtestacionAutorizacionV2 struct {
	// Has unexported fields.
}
```

AtestacionAutorizacionV2 conserva juntos el mensaje exacto y la salida del
firmante. Sigue siendo evidencia nominal hasta superar perfil de confianza,
vigencia, revocacion, revalidacion y consumo unico en la transaccion final.

```go
func NuevaAtestacionAutorizacionV2(
	solicitud SolicitudFirmaAtestacionAutorizacionV2,
	resultado ResultadoFirmaAtestacionAutorizacionV2,
) (AtestacionAutorizacionV2, error)

func (b AtestacionAutorizacionV2) Format(estado fmt.State, _ rune)

func (b AtestacionAutorizacionV2) GoString() string

func (*AtestacionAutorizacionV2) GobDecode([]byte) error

func (AtestacionAutorizacionV2) GobEncode() ([]byte, error)

func (b AtestacionAutorizacionV2) LogValue() slog.Value

func (AtestacionAutorizacionV2) MarshalBinary() ([]byte, error)

func (AtestacionAutorizacionV2) MarshalCBOR() ([]byte, error)

func (AtestacionAutorizacionV2) MarshalJSON() ([]byte, error)

func (AtestacionAutorizacionV2) MarshalText() ([]byte, error)

func (AtestacionAutorizacionV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (AtestacionAutorizacionV2) MarshalYAML() (any, error)

func (a AtestacionAutorizacionV2) Resultado() (ResultadoFirmaAtestacionAutorizacionV2, error)

func (a AtestacionAutorizacionV2) Solicitud() (SolicitudFirmaAtestacionAutorizacionV2, error)

func (AtestacionAutorizacionV2) String() string

func (*AtestacionAutorizacionV2) UnmarshalBinary([]byte) error

func (*AtestacionAutorizacionV2) UnmarshalCBOR([]byte) error

func (*AtestacionAutorizacionV2) UnmarshalJSON([]byte) error

func (*AtestacionAutorizacionV2) UnmarshalText([]byte) error

func (*AtestacionAutorizacionV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*AtestacionAutorizacionV2) UnmarshalYAML(func(any) error) error

func (a AtestacionAutorizacionV2) Validar() error

func (a AtestacionAutorizacionV2) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV2,
) error

type AtestacionCriptograficaMaterialAlmacenV2 struct {
	// Has unexported fields.
}
```

AtestacionCriptograficaMaterialAlmacenV2 liga algoritmo, clave versionada,
dominio y huella del mensaje. El codigo se copia al entrar y al salir.

```go
func NuevaAtestacionCriptograficaMaterialAlmacenV2(
	solicitud SolicitudAtestarMaterialAlmacenV2,
	algoritmo AlgoritmoAtestacionMaterialAlmacenV2,
	claveRef string,
	claveVersion uint32,
	codigo []byte,
) (AtestacionCriptograficaMaterialAlmacenV2, error)

func (AtestacionCriptograficaMaterialAlmacenV2) Format(e fmt.State, _ rune)

func (AtestacionCriptograficaMaterialAlmacenV2) GoString() string

func (AtestacionCriptograficaMaterialAlmacenV2) LogValue() slog.Value

func (AtestacionCriptograficaMaterialAlmacenV2) MarshalBinary() ([]byte, error)

func (AtestacionCriptograficaMaterialAlmacenV2) MarshalJSON() ([]byte, error)

func (AtestacionCriptograficaMaterialAlmacenV2) MarshalText() ([]byte, error)

func (AtestacionCriptograficaMaterialAlmacenV2) String() string

func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalBinary([]byte) error

func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalJSON([]byte) error

func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalText([]byte) error

type AtestacionCrudaReconciliacionDocumentalV4 struct {
	// Has unexported fields.
}

func NuevaAtestacionCrudaReconciliacionDocumentalV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (AtestacionCrudaReconciliacionDocumentalV4, error)

func (p AtestacionCrudaReconciliacionDocumentalV4) Format(estado fmt.State, _ rune)

func (p AtestacionCrudaReconciliacionDocumentalV4) GoString() string

func (p AtestacionCrudaReconciliacionDocumentalV4) LogValue() slog.Value

func (AtestacionCrudaReconciliacionDocumentalV4) MarshalBinary() ([]byte, error)

func (AtestacionCrudaReconciliacionDocumentalV4) MarshalJSON() ([]byte, error)

func (AtestacionCrudaReconciliacionDocumentalV4) MarshalText() ([]byte, error)

func (p AtestacionCrudaReconciliacionDocumentalV4) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
)

func (AtestacionCrudaReconciliacionDocumentalV4) String() string

func (*AtestacionCrudaReconciliacionDocumentalV4) UnmarshalBinary([]byte) error

func (*AtestacionCrudaReconciliacionDocumentalV4) UnmarshalJSON([]byte) error

func (*AtestacionCrudaReconciliacionDocumentalV4) UnmarshalText([]byte) error

func (p AtestacionCrudaReconciliacionDocumentalV4) ValidarSintaxis() error

type AtestadorMaterialAlmacenV2 interface {
	AtestarMaterialAlmacenV2(
		context.Context,
		SolicitudAtestarMaterialAlmacenV2,
	) (AtestacionCriptograficaMaterialAlmacenV2, error)
}

type AtributosEventoSalidaCobro = pagoscanonicos.AtributosEventoSalidaCobro

type AuditStore interface {
	AppendAudit(context.Context, domain.AuditEntry) (domain.AuditEntry, error)
	ListAudit(context.Context, string) ([]domain.AuditEntry, error)
}

type Autorizador interface {
	Exigir(context.Context, domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error)
}
```

Autorizador es el unico puerto que deben consumir los casos de uso. Exigir
devuelve ErrAutorizacionDenegada para cualquier resultado no concedido.

```go
type AutorizadorSolicitudLigadaV2 interface {
	ExigirSolicitudLigadaV2(
		context.Context,
		domain.SolicitudAutorizacionLigadaV2,
	) (domain.DecisionAutorizacion, error)
}
```

AutorizadorSolicitudLigadaV2 es deliberadamente distinto de Autorizador.
Impide que un flujo nuevo acepte por accidente una decision historica V1.

```go
type CabeceraCargaDirecta = almacencanonico.CabeceraCargaDirecta
```

CabeceraCargaDirecta es una condicion puntual de la carga, no una credencial
general. Se mantiene dentro de InstruccionesCargaDirecta hasta la revelacion
deliberada para impedir que aparezca en registros.

```go
type CampoHandoffCobro = pagoscanonicos.CampoHandoffCobro

type CanalAuditoriaCobro = pagoscanonicos.CanalAuditoriaCobro

type CapacidadesAlmacenObjetos = almacencanonico.Capacidades
```

CapacidadesAlmacenObjetos permite validar el perfil de despliegue al
arrancar. Declarar una capacidad no basta: cada conector productivo debe
superar sus pruebas de contrato y de recuperacion.

```go
type CapacidadesAnalizadorContenido struct {
	ConectorID             string
	VersionConector        int
	AnalisisEnFlujo        bool
	CanalAutenticado       bool
	CifradoEnTransito      bool
	IdentidadMutua         bool
	ActualizacionFirmas    bool
	DetectaMalware         bool
	DetectaContenidoActivo bool
	TamanoMaximo           int64
}
```

CapacidadesAnalizadorContenido permite conectar ICAP, una API corporativa,
un proceso aislado o cualquier motor futuro sin introducir el producto en
el nucleo. MCP no es el transporte de seguridad: el analisis debe ser una
integracion autenticada, determinista y observable entre servicios.

```go
type CapacidadesPasarelaCobro = pagoscanonicos.CapacidadesPasarelaCobro

type CargaHandoffCobro = pagoscanonicos.CargaHandoffCobro

func NuevaCargaHandoffCobro(campos []CampoHandoffCobro, permitidos []string) (CargaHandoffCobro, error)

type CatalogoFormatosDocumentales interface {
	BuscarDescriptoresFormatoDocumental(
		context.Context,
		ConsultaFormatoDocumental,
	) ([]DescriptorFormatoDocumental, error)
}
```

CatalogoFormatosDocumentales devuelve todas las coincidencias, nunca "la
primera". El servicio de aplicacion exige cardinalidad exactamente uno para
detectar duplicados o fuentes contradictorias.

```go
type CatalogoOrigenesPasarelaCobro interface {
	ObtenerOrigenPublicado(context.Context, string, int) (OrigenPasarelaCobroPublicado, error)
}

type CatalogoPerfilesDocumentales interface {
	BuscarDescriptoresPerfilDocumental(
		context.Context,
		ConsultaFormatoDocumental,
	) ([]DescriptorPerfilDocumental, error)
}

type CatalogoPlantillasDocumento interface {
	ObtenerPlantilla(context.Context, string, int) (domain.PlantillaDocumento, error)
	ListarPlantillas(context.Context, string) ([]domain.PlantillaDocumento, error)
}
```

CatalogoPlantillasDocumento es deliberadamente de solo lectura. Toda alta o
transicion pasa por el repositorio de gobierno con autorizacion, auditoria y
outbox; ninguna importacion puede insertar una publicada por esta via.

```go
type CatalogoPoliticasCotejo interface {
	ObtenerPoliticaCotejo(context.Context, string, int) (domain.PoliticaCotejo, error)
	ListarVersionesPoliticaCotejo(context.Context, string) ([]domain.PoliticaCotejo, error)
}
```

CatalogoPoliticasCotejo solo permite leer versiones concretas. El nucleo no
selecciona implicitamente «la ultima», porque eso cambiaria el significado
de un documento ya emitido.

```go
type CatalogoPoliticasInstitucionalesDocumentales interface {
	BuscarPoliticasInstitucionalesExactas(
		context.Context,
		ConsultaPoliticaInstitucionalDocumental,
	) ([]PoliticaInstitucionalDocumentalAtestada, error)
}

type CertAuthPort interface {
	Challenge(context.Context) (domain.AuthChallenge, error)
	Verify(context.Context, domain.AuthChallengeResponse) (domain.Principal, error)
}

type ClaseDeteccionContenido string

const (
	ClaseDeteccionMalware         ClaseDeteccionContenido = "malware"
	ClaseDeteccionContenidoActivo ClaseDeteccionContenido = "contenido_activo"
	ClaseDeteccionArchivoDanado   ClaseDeteccionContenido = "archivo_danado"
	ClaseDeteccionPolitica        ClaseDeteccionContenido = "politica_seguridad"
)
func (c ClaseDeteccionContenido) Valida() bool

type ClaveAplicacionAutorizacionEjecucionDocumentalV4 struct {
	DecisionRef      string
	HuellaPlanSHA256 string
	EfectoRef        string
}
```

ClaveAplicacionAutorizacionEjecucionDocumentalV4 liga de forma exacta la
decision, el plan y el efecto. No es autoridad. El adaptador duradero debera
comprobar la terna y reclamar UNIQUE(DecisionRef) dentro del mismo COMMIT
que aplique el efecto de negocio.

```go
type ClaveConsumoConsultaReconciliacionDocumentalV4 struct {
	ConsultaRef          string
	HuellaRetoSHA256     string
	HuellaConsultaSHA256 string
}
```

ClaveConsumoConsultaReconciliacionDocumentalV4 es comparable y apta para
indices UNIQUE. El registro durable debe imponer unicidad independiente
tanto sobre ConsultaRef como sobre HuellaRetoSHA256, ademas de conservar la
huella completa de consulta. La estructura no concede autoridad.

```go
func (c ClaveConsumoConsultaReconciliacionDocumentalV4) ValidarContra(
	consulta ConsultaReconciliacionDocumentalV4,
) error

type CommonRegistryPort interface {
	RegisterEntry(context.Context, InteropRequest) (InteropResult, error)
}

type ComprobadorActosFuentesAutoridad interface {
	ComprobarActo(
		context.Context,
		SolicitudComprobarActoFuenteAutoridad,
	) (AtestacionActoFuenteAutoridad, error)
}
```

ComprobadorActosFuentesAutoridad debe verificar documento, representacion,
huella, firmas, competencia y procedencia de la atestacion. El repositorio
vuelve a leer el registro y consume el token en la transaccion de gobierno.

```go
type ComprobanteConsumoReciboCargaDirecta struct {
	// Has unexported fields.
}
```

ComprobanteConsumoReciboCargaDirecta no contiene el recibo ni la sesion.
Su atestacion es un HMAC opaco de segunda fase. La mera forma valida no
acredita nada: la fabrica de confirmacion exige siempre un verificador
criptografico independiente antes de crear una solicitud para el conector.

```go
func NuevoComprobanteConsumoReciboCargaDirecta(
	solicitud SolicitudConsumirReciboCargaDirecta,
	resultado ResultadoConsumoReciboCargaDirecta,
	atestacionHMAC string,
) (ComprobanteConsumoReciboCargaDirecta, error)
```

NuevoComprobanteConsumoReciboCargaDirecta acepta exclusivamente el HMAC
opaco emitido despues del consumo durable. No lo verifica por si mismo:
el verificador real es obligatorio en NuevaSolicitudConfirmarCargaDirecta.

```go
func (c ComprobanteConsumoReciboCargaDirecta) Format(estado fmt.State, _ rune)

func (ComprobanteConsumoReciboCargaDirecta) GoString() string

func (c ComprobanteConsumoReciboCargaDirecta) LogValue() slog.Value

func (ComprobanteConsumoReciboCargaDirecta) MarshalJSON() ([]byte, error)

func (ComprobanteConsumoReciboCargaDirecta) MarshalText() ([]byte, error)

func (c ComprobanteConsumoReciboCargaDirecta) RevelarParaVerificacion() (
	indiceHMAC, grupoHMAC, vinculoHMAC, evidenciaConsumoRef, intencionRef, huellaIntencionHMAC string,
	registradoEn, consumidoEn, expiraEn, validaHasta time.Time,
	atestacionHMAC string,
	err error,
)

func (ComprobanteConsumoReciboCargaDirecta) String() string

type CompromisoEjecucionComponenteDocumental struct {
	// Has unexported fields.
}
```

CompromisoEjecucionComponenteDocumental liga una invocacion irrepetible con
el perfil, su publicacion/revision y el componente exacto. En renderizado
la huella y el tamano de salida todavia son desconocidos y deben ser cero;
en las verificaciones son obligatorios.

```go
func NuevoCompromisoEjecucionComponenteDocumental(
	operacionRef string,
	reto [32]byte,
	operacion OperacionComponenteDocumental,
	descriptorPerfil DescriptorPerfilDocumental,
	situacionOperativa domain.SituacionOperativaPerfilDocumental,
	descriptorComponente DescriptorComponenteDocumentalAtestado,
	ordenConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
	borradorRef, huellaContenidoNeutralHMAC, huellaDocumentoSHA256 string,
	tamanoDocumento, limiteBytes uint64,
	vigencia time.Duration,
) (CompromisoEjecucionComponenteDocumental, error)
```

NuevoCompromisoEjecucionComponenteDocumental acepta el comando nominal que
el servicio de aplicacion obtuvo tras verificar por KMS y consumir por CAS.
El valor no es autoritativo ni sustituye la composicion: handlers no deben
poseer este constructor junto con el despachador.

```go
func (c CompromisoEjecucionComponenteDocumental) BorradorRef() string

func (c CompromisoEjecucionComponenteDocumental) ConsumoDecisionCercado() ConsumoDecisionEjecucionDocumentalV3

func (c CompromisoEjecucionComponenteDocumental) DescriptorComponente() DescriptorComponenteDocumentalAtestado

func (c CompromisoEjecucionComponenteDocumental) DescriptorPerfil() DescriptorPerfilDocumental

func (c CompromisoEjecucionComponenteDocumental) EfectoRef() string

func (c CompromisoEjecucionComponenteDocumental) EmitidoEn() time.Time

func (c CompromisoEjecucionComponenteDocumental) ExpiraEn() time.Time

func (c CompromisoEjecucionComponenteDocumental) Format(estado fmt.State, _ rune)

func (c CompromisoEjecucionComponenteDocumental) GoString() string

func (c CompromisoEjecucionComponenteDocumental) HuellaContenidoNeutralHMAC() string

func (c CompromisoEjecucionComponenteDocumental) HuellaDocumentoSHA256() string

func (c CompromisoEjecucionComponenteDocumental) HuellaPlanSHA256() string

func (c CompromisoEjecucionComponenteDocumental) HuellaSHA256() (string, error)

func (c CompromisoEjecucionComponenteDocumental) HuellaVinculoCercadoSHA256() string

func (c CompromisoEjecucionComponenteDocumental) LimiteBytes() uint64

func (c CompromisoEjecucionComponenteDocumental) LogValue() slog.Value

func (c CompromisoEjecucionComponenteDocumental) ManifiestoCercado() ManifiestoEjecucionDocumentalV3

func (CompromisoEjecucionComponenteDocumental) MarshalBinary() ([]byte, error)

func (CompromisoEjecucionComponenteDocumental) MarshalJSON() ([]byte, error)

func (CompromisoEjecucionComponenteDocumental) MarshalText() ([]byte, error)

func (c CompromisoEjecucionComponenteDocumental) Operacion() OperacionComponenteDocumental

func (c CompromisoEjecucionComponenteDocumental) OperacionRef() string

func (c CompromisoEjecucionComponenteDocumental) ReservaRef() string

func (c CompromisoEjecucionComponenteDocumental) Reto() [32]byte

func (c CompromisoEjecucionComponenteDocumental) SecuenciaCercado() uint64

func (c CompromisoEjecucionComponenteDocumental) SituacionOperativa() domain.SituacionOperativaPerfilDocumental

func (CompromisoEjecucionComponenteDocumental) String() string

func (c CompromisoEjecucionComponenteDocumental) TamanoDocumento() uint64

func (*CompromisoEjecucionComponenteDocumental) UnmarshalBinary([]byte) error

func (*CompromisoEjecucionComponenteDocumental) UnmarshalJSON([]byte) error

func (*CompromisoEjecucionComponenteDocumental) UnmarshalText([]byte) error

func (c CompromisoEjecucionComponenteDocumental) Validar() error

func (c CompromisoEjecucionComponenteDocumental) VigenteEn(instante time.Time) bool

func (c CompromisoEjecucionComponenteDocumental) VinculoActivacion() VinculoEstableActivacionDocumentalV3

type CondicionCASReconciliacionDocumentalV4 struct {
	EstadoEsperado           EstadoEjecucionDocumentalV3
	VersionEsperada          uint64
	SecuenciaCercadoEsperada uint64
}

func (c CondicionCASReconciliacionDocumentalV4) ValidarContra(
	consulta ConsultaReconciliacionDocumentalV4,
) error

type ConectorEjecucionDocumentalAtestadaV4 interface {
	EjecutarDocumentalAtestadoV4(
		context.Context,
		SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
		domain.CabeceraAtestacionAutorizacionV1,
		SobreCriptograficoDocumentalCrudoV4,
	) (ResultadoConectorEjecucionDocumentalAtestadaV4, error)
}
```

ConectorEjecucionDocumentalAtestadaV4 es el puerto de salida del nucleo.
PostgreSQL, Oracle u otro conector homologado implementan el consumo
atomico, pero el caso de uso no conoce su motor, transporte ni credenciales.

La solicitud vinculada y el sobre son valores opacos y de valor cero
invalido. El conector debe volver a verificar la autorizacion y confirmar el
efecto, la auditoria y el outbox en una unica frontera transaccional.

```go
type ConfirmacionRegistroContextoActorV1 struct {
	OperacionRef           string
	RegistroContextoRef    string
	Contexto               domain.ContextoActor
	RepresentacionCanonica []byte
	HuellaSHA256           string
	ResueltoEnAutoritativo time.Time
}
```

ConfirmacionRegistroContextoActorV1 es el recibo exacto que el adaptador
recupera de la persistencia y devuelve unicamente despues de un COMMIT
confirmado o reconciliado. RepresentacionCanonica y HuellaSHA256 son los
valores almacenados, no una reconstruccion oportunista posterior al commit.

```go
func (c ConfirmacionRegistroContextoActorV1) ValidarPara(
	solicitud SolicitudResolucionRegistroContextoActorV1,
) error
```

ValidarPara comprueba el eco exacto de la solicitud y liga el recibo a los
mismos bytes que compromete Contexto. No demuestra por si solo que exista la
fila: esa autoridad pertenece al puerto durable y a su reconciliacion.

```go
type ConfirmacionTransicionCargaDocumental struct {
	VersionEsperada      int
	HuellaAnteriorSHA256 string
	Carga                domain.CargaDocumental
	Auditoria            domain.AuditEntry
	Evento               domain.Event
}

func InstantaneaConfirmacionTransicionCargaDocumental(
	confirmacion ConfirmacionTransicionCargaDocumental,
) ConfirmacionTransicionCargaDocumental
```

InstantaneaConfirmacionTransicionCargaDocumental corta todos los alias
mutables antes de consultar el estado autoritativo. El repositorio debe
obtener esta copia antes del bloqueo/transaccion, validarla contra la
version leida dentro de ese bloqueo y persistir exactamente la misma copia.

```go
func NuevaConfirmacionTransicionCargaDocumental(
	anterior, siguiente domain.CargaDocumental,
	auditoria domain.AuditEntry,
	evento domain.Event,
) (ConfirmacionTransicionCargaDocumental, error)

func (c ConfirmacionTransicionCargaDocumental) ValidarContra(anterior domain.CargaDocumental) error

type ConsultaCatalogosConfigurables interface {
	ObtenerCatalogo(context.Context, string, int) (domain.CatalogoConfigurable, error)
	ListarVersionesCatalogo(context.Context, string) ([]domain.CatalogoConfigurable, error)
}

type ConsultaCitaFuenteAutoridad interface {
	ObtenerPorCita(context.Context, domain.CitaFuenteAutoridad) (domain.FuenteAutoridadVersionada, error)
}
```

ConsultaCitaFuenteAutoridad resuelve solo citas exactas; el puerto no
completa una lista vacia de preceptos ni sustituye su fuente.

```go
type ConsultaComponenteDocumentalAtestado struct {
	Rol                 domain.RolComponenteDocumental
	DescriptorPerfilRef string
	PublicacionRef      string
	PerfilRef           domain.ReferenciaPerfilDocumental
	DigestPerfil        string
	RevisionCatalogo    domain.RevisionCatalogoFormatosDocumentales
}

func (c ConsultaComponenteDocumentalAtestado) Validar() error

type ConsultaDefinicionesFlujo interface {
	ObtenerDefinicionFlujo(context.Context, string, int) (domain.DefinicionFlujo, error)
	ObtenerDefinicionFlujoPorReferencia(context.Context, string) (domain.DefinicionFlujo, error)
	ListarVersionesDefinicionFlujo(context.Context, string) ([]domain.DefinicionFlujo, error)
}

type ConsultaEjecucionDocumentalV3 struct {
	ReservaRef             string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
}

func (c ConsultaEjecucionDocumentalV3) Format(estado fmt.State, _ rune)

func (c ConsultaEjecucionDocumentalV3) GoString() string

func (c ConsultaEjecucionDocumentalV3) LogValue() slog.Value

func (ConsultaEjecucionDocumentalV3) MarshalBinary() ([]byte, error)

func (ConsultaEjecucionDocumentalV3) MarshalJSON() ([]byte, error)

func (ConsultaEjecucionDocumentalV3) MarshalText() ([]byte, error)

func (ConsultaEjecucionDocumentalV3) String() string

func (*ConsultaEjecucionDocumentalV3) UnmarshalBinary([]byte) error

func (*ConsultaEjecucionDocumentalV3) UnmarshalJSON([]byte) error

func (*ConsultaEjecucionDocumentalV3) UnmarshalText([]byte) error

func (c ConsultaEjecucionDocumentalV3) Validar() error

type ConsultaFormatoDocumental struct {
	Identidad          domain.IdentidadSintacticaDocumental
	PerfilRef          domain.ReferenciaPerfilDocumental
	DigestPerfilSHA256 string
	RevisionCatalogo   domain.RevisionCatalogoFormatosDocumentales
}
```

ConsultaFormatoDocumental fija todos los ejes gobernados. El digest del
perfil impide reutilizar el mismo ID/version con otra especificacion y la
revision incluye numero y huella de la instantanea completa del catalogo.

```go
func (c ConsultaFormatoDocumental) Validar() error

type ConsultaHistoriaFuentesAutoridad interface {
	ListarVersiones(
		context.Context,
		ConsultaPaginaHistoriaFuenteAutoridad,
	) (PaginaHistoriaFuenteAutoridad, error)
}
```

ConsultaHistoriaFuentesAutoridad pagina las versiones por orden ascendente.
Una pagina vacia es distinta de una consulta invalida.

```go
type ConsultaInstanciasFlujo interface {
	ObtenerInstanciaFlujo(context.Context, string) (domain.InstanciaFlujo, error)
}

type ConsultaInternaGobernadaFuentesAutoridad interface {
	ConsultarVersionExacta(
		context.Context,
		SolicitudConsultaInternaGobernadaFuenteAutoridad,
	) (ResultadoConsultaInternaFuenteAutoridad, error)
}
```

ConsultaInternaGobernadaFuentesAutoridad es una barrera transaccional, no
una lectura DAO. En una unica transaccion debe releer y validar la decision
durable, consumirla exactamente una vez, ejecutar la lectura exacta, fijar
el resultado de auditoria a encontrada o no_encontrada, encadenar y firmar
esa auditoria y emitir el recibo. Resultado y Recibo solo se construyen
y devuelven despues del COMMIT. La ausencia nunca se devuelve como
ErrFuenteAutoridadNoEncontrada ni sigue un camino sin consumo y auditoria.

```go
type ConsultaMetadatosFuenteCatalogos interface {
	ObtenerMetadatosFuenteCatalogos(context.Context) (MetadatosFuenteCatalogos, error)
}

type ConsultaOperacionesFuentesAutoridad interface {
	ObtenerOperacion(context.Context, SelectorOperacionFuenteAutoridad) (OperacionFuenteAutoridad, error)
}

type ConsultaPaginaHistoriaFuenteAutoridad struct {
	FuenteID     string
	DesdeVersion uint64
	Limite       uint16
}

func (s ConsultaPaginaHistoriaFuenteAutoridad) Validar() error

type ConsultaPoliticaInstitucionalDocumental struct {
	Institucion        domain.ReferenciaInstitucionalDocumento
	PerfilRef          domain.ReferenciaPerfilDocumental
	ManifiestoRef      string
	RequiereURIPublica bool
}

func (c ConsultaPoliticaInstitucionalDocumental) Validar() error

type ConsultaReconciliacionDocumentalV4 struct {
	// Has unexported fields.
}
```

ConsultaReconciliacionDocumentalV4 conserva una instantanea defensiva y el
compromiso del payload exacto que debera firmar el sistema consultado.

```go
func NuevaConsultaReconciliacionDocumentalV4(
	datos DatosConsultaReconciliacionDocumentalV4,
) (ConsultaReconciliacionDocumentalV4, error)

func (c ConsultaReconciliacionDocumentalV4) ClaveConsumoUnico() (
	ClaveConsumoConsultaReconciliacionDocumentalV4,
	error,
)

func (c ConsultaReconciliacionDocumentalV4) Datos() (
	DatosConsultaReconciliacionDocumentalV4,
	error,
)

func (c ConsultaReconciliacionDocumentalV4) Format(estado fmt.State, _ rune)

func (c ConsultaReconciliacionDocumentalV4) GoString() string

func (c ConsultaReconciliacionDocumentalV4) HuellaMensajeSHA256() (string, error)

func (c ConsultaReconciliacionDocumentalV4) LogValue() slog.Value

func (ConsultaReconciliacionDocumentalV4) MarshalBinary() ([]byte, error)

func (ConsultaReconciliacionDocumentalV4) MarshalJSON() ([]byte, error)

func (ConsultaReconciliacionDocumentalV4) MarshalText() ([]byte, error)

func (c ConsultaReconciliacionDocumentalV4) MensajeCanonico() ([]byte, error)
```

MensajeCanonico es la unica salida binaria intencionada. Todos sus campos,
incluida la version, llevan longitud big-endian para impedir ambiguedades de
concatenacion y cambios de representacion.

```go
func (ConsultaReconciliacionDocumentalV4) String() string

func (*ConsultaReconciliacionDocumentalV4) UnmarshalBinary([]byte) error

func (*ConsultaReconciliacionDocumentalV4) UnmarshalJSON([]byte) error

func (*ConsultaReconciliacionDocumentalV4) UnmarshalText([]byte) error

func (c ConsultaReconciliacionDocumentalV4) ValidarSintaxis() error
```

ValidarSintaxis comprueba invariantes, no la aleatoriedad ni la unicidad
durable de ConsultaRef y reto.

```go
type ConsultaReferenciaFuenteAutoridad interface {
	ObtenerPorReferencia(context.Context, domain.ReferenciaFuenteAutoridad) (domain.FuenteAutoridadVersionada, error)
}
```

ConsultaReferenciaFuenteAutoridad resuelve una referencia ya ligada a la
huella exacta. Una discrepancia de huella se trata como no encontrada.

```go
type ConsultaSituacionOperativaActual struct {
	PublicacionRef   string
	PerfilRef        domain.ReferenciaPerfilDocumental
	DigestPerfil     string
	RevisionCatalogo domain.RevisionCatalogoFormatosDocumentales
}
```

ConsultaSituacionOperativaActual obliga al registro a devolver la proyeccion
actual exacta. Una entrada historica no satisface el contrato.

```go
func (c ConsultaSituacionOperativaActual) Coincide(
	situacion domain.SituacionOperativaPerfilDocumental,
) bool

func (c ConsultaSituacionOperativaActual) Validar() error

type ConsultaVersionFuenteAutoridad interface {
	ObtenerVersion(context.Context, SelectorVersionFuenteAutoridad) (domain.FuenteAutoridadVersionada, error)
}
```

ConsultaVersionFuenteAutoridad nunca elige la version mas reciente.
La implementacion debe devolver una copia canonica del agregado.

```go
type ConsumidorPrivadoOrdenDespachoDocumentalV3 interface {
	ReleerYConsumirOrdenDespachoDocumentalV3(
		context.Context,
		SolicitudComprobarOrdenDespachoDocumentalV3,
		ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
	) (EstadoCrudoOrdenDespachoDocumentalV3, error)
}
```

ConsumidorPrivadoOrdenDespachoDocumentalV3 relee y consume por CAS la orden
reclamada dentro de una unica transaccion que incrementa VersionConsumoCAS,
inmoviliza un ConsumoDespachoRef UNIQUE y escribe auditoria y outbox. Debe
ejecutar Resultado.ValidarPara(Solicitud) antes de mutar la fila y repetir
esa correlacion dentro de la transaccion, junto con la relectura durable.
No existe una operacion publica read-then-use: el segundo consumo o un
resultado KMS perteneciente a otra solicitud deben fallar sin ejecutar el
CAS.

```go
type ConsumidorReciboCargaDirecta interface {
	ConsumirReciboCargaDirecta(
		context.Context,
		SolicitudConsumirReciboCargaDirecta,
	) (ComprobanteConsumoReciboCargaDirecta, error)
}
```

ConsumidorReciboCargaDirecta verifica MAC/caducidad y marca el recibo como
usado en una unica operacion atomica duradera.

```go
type ConsumoDecisionEjecucionDocumentalV3 struct {
	DecisionRef           string
	EfectoRef             string
	EsquemaHuellaDecision string
	HuellaDecisionSHA256  string
	HuellaPlanSHA256      string
}
```

ConsumoDecisionEjecucionDocumentalV3 se reclama con UNIQUE(DecisionRef) en
la misma transaccion que activa la reserva. Abandono o caducidad no liberan
nunca esa referencia para otra ejecucion.

```go
func (c ConsumoDecisionEjecucionDocumentalV3) ValidarContra(
	manifiesto ManifiestoEjecucionDocumentalV3,
) error

type ConsumoDecisionPreparacionCargaDocumentalV1 struct {
	DecisionRef            string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	EsquemaHuellaDecision  string
	HuellaDecisionSHA256   string
}
```

ConsumoDecisionPreparacionCargaDocumentalV1 es la tupla durable que impide
reutilizar una misma DecisionRef para otro efecto o repetir el mismo.
El repositorio la reclama con restriccion UNIQUE dentro de Reservar, antes
de crear una sesion remota, y consume esa misma reclamacion en el commit
atomico de agregado, manifiesto, auditoria y outbox. Abandono o caducidad no
liberan DecisionRef para otra reclamacion.

```go
func ConsumoDecisionDesdeContextoPreparacionCargaDocumental(
	contexto ContextoOperacionAlmacen,
) (ConsumoDecisionPreparacionCargaDocumentalV1, error)
```

ConsumoDecisionDesdeContextoPreparacionCargaDocumental permite reclamar la
decision antes de crear una sesion remota. La reclamacion no concede ninguna
capacidad: solo reserva de forma fail-closed la misma tupla que mas tarde
debera aparecer en el manifiesto y consumirse en el commit final.

```go
func ConsumoDecisionDesdeManifiestoPreparacionCargaDocumental(
	manifiesto domain.ManifiestoPreparacionCargaDirectaV1,
) (ConsumoDecisionPreparacionCargaDocumentalV1, error)

func (c ConsumoDecisionPreparacionCargaDocumentalV1) Validar() error

type ContenidoDocumentoGuardado struct {
	ReferenciaLogica   string
	Referencia         string
	Version            string
	ConectorID         string
	Zona               ZonaAlmacen
	MIME               string
	HuellaSHA256       string
	Tamano             int64
	EvidenciaOperacion EvidenciaOperacionAlmacen
}

func (g ContenidoDocumentoGuardado) ValidarContra(s SolicitudGuardarContenido) error

type ContenidoDocumentoLeido struct {
	Contenido          []byte
	ConectorID         string
	Zona               ZonaAlmacen
	HuellaSHA256       string
	Tamano             int64
	EvidenciaOperacion EvidenciaOperacionAlmacen
}

type ContenidoNotificacionCobroUnico = pagoscanonicos.ContenidoNotificacionCobroUnico

type ContextoEliminarCodigoCotejoHuerfano struct {
	// Has unexported fields.
}

func NuevoContextoEliminarCodigoCotejoHuerfano(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	codigoRef, clasificacion, proteccionRef, evidenciaRef, motivo string,
	verificadaEn time.Time,
) (ContextoEliminarCodigoCotejoHuerfano, error)
```

NuevoContextoEliminarCodigoCotejoHuerfano exige una decision distinta del
PDP y, ademas, una cuenta privilegiada sobre la superficie administrativa.
El flujo interactivo de emision no dispone de esa autoridad: la futura
limpieza debe ejecutarla un worker tecnico interno expresamente cableado.

```go
func (ContextoEliminarCodigoCotejoHuerfano) MarshalJSON() ([]byte, error)

func (ContextoEliminarCodigoCotejoHuerfano) MarshalText() ([]byte, error)

func (ContextoEliminarCodigoCotejoHuerfano) String() string

type ContextoOperacionAlmacen struct {
	// Has unexported fields.
}
```

ContextoOperacionAlmacen es una capacidad opaca e inmutable. Su valor cero y
cualquier valor reconstruido por serializacion son invalidos.

```go
func NuevoContextoAnalizarCargaDocumentalAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error)

func NuevoContextoConfirmarCargaDirectaAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error)

func NuevoContextoCustodiarDecisionBaremacionAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error)

func NuevoContextoCustodiarDocumentoFirmadoAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error)

func NuevoContextoGeneracionDocumentalAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	manifiesto ManifiestoGeneracionDocumental,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error)
```

NuevoContextoGeneracionDocumentalAlmacen es la unica fabrica que admite
una accion de negocio configurada. La accion procede exclusivamente del
PermisoGenerar de la plantilla publicada comprometida por el manifiesto;
nunca se acepta como parametro libre.

```go
func NuevoContextoPrepararCargaDirectaAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error)

func NuevoContextoPromoverCargaDocumentalAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error)

func NuevoContextoRetenerDocumentoFirmadoAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error)

func (c ContextoOperacionAlmacen) DerivarPaso(pasoRef PasoOperacionAlmacen) (ContextoOperacionAlmacen, error)
```

DerivarPaso solo selecciona un paso del plan comprometido al emitir la
capacidad. No acepta una accion tecnica ni permite ampliar el plan.

```go
func (c ContextoOperacionAlmacen) EvidenciaAutorizacion() (EvidenciaUsoDecisionAutorizacion, error)
```

EvidenciaAutorizacion devuelve deliberadamente la capacidad que el adaptador
duradero debe revalidar y consumir de forma unica en la misma transaccion
que DecisionRef -> (EfectoRef, HuellaPlanEfectoSHA256).

```go
func (c ContextoOperacionAlmacen) Format(estado fmt.State, _ rune)

func (c ContextoOperacionAlmacen) GoString() string

func (c ContextoOperacionAlmacen) LogValue() slog.Value

func (ContextoOperacionAlmacen) MarshalJSON() ([]byte, error)

func (ContextoOperacionAlmacen) MarshalText() ([]byte, error)

func (c ContextoOperacionAlmacen) Proyeccion() (ProyeccionContextoOperacionAlmacen, error)

func (ContextoOperacionAlmacen) String() string

func (*ContextoOperacionAlmacen) UnmarshalJSON([]byte) error

func (*ContextoOperacionAlmacen) UnmarshalText([]byte) error

func (c ContextoOperacionAlmacen) ValidarEn(instante time.Time) error

func (c ContextoOperacionAlmacen) ValidarParaEn(accionTecnica string, instante time.Time) error

type ContextoProtegerCodigoCotejo struct {
	// Has unexported fields.
}
```

Los tres contextos son capacidades opacas e incompatibles. El valor cero es
invalido y una decision concedida para reservar nunca puede rellenarlos ni
reinterpretarse como permiso de custodia.

```go
func NuevoContextoProtegerCodigoCotejo(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	codigoRef, clasificacion, claveIdempotencia, indiceCodigoHMAC string,
	verificadaEn time.Time,
) (ContextoProtegerCodigoCotejo, error)

func (ContextoProtegerCodigoCotejo) MarshalJSON() ([]byte, error)

func (ContextoProtegerCodigoCotejo) MarshalText() ([]byte, error)

func (ContextoProtegerCodigoCotejo) String() string

type ContextoRecuperarCodigoCotejo struct {
	// Has unexported fields.
}

func NuevoContextoRecuperarCodigoCotejo(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	codigoRef, clasificacion, proteccionRef, indiceCodigoHMACEsperado string,
	verificadaEn time.Time,
) (ContextoRecuperarCodigoCotejo, error)

func (ContextoRecuperarCodigoCotejo) MarshalJSON() ([]byte, error)

func (ContextoRecuperarCodigoCotejo) MarshalText() ([]byte, error)

func (ContextoRecuperarCodigoCotejo) String() string

type CustodiaCodigoCotejo struct {
	ProteccionRef string
	ConectorID    string
	EvidenciaRef  string
}

type CustodiaNotificacionesCobro interface {
	Custodiar(context.Context, SolicitudCustodiarNotificacionCobro, io.Reader) (NotificacionCobro, error)
	ConsumirUnaVez(context.Context, NotificacionCobro) (ContenidoNotificacionCobroUnico, error)
	Descartar(context.Context, NotificacionCobro, string) error
}

type DataIntermediationPort interface {
	QueryData(context.Context, InteropRequest) (InteropResult, error)
}

type DatosAtestacionActoFuenteAutoridad struct {
	Evidencia                      domain.EvidenciaActoFuenteAutoridad
	RevisionEsperada               uint64
	HuellaEstadoEsperadoSHA256     string
	VerificadorRef                 string
	RegistroAtestacionRef          string
	HuellaRegistroAtestacionSHA256 string
	TokenConsumoRef                string
	EmitidaEn                      time.Time
	ValidaHasta                    time.Time
	// Has unexported fields.
}
```

DatosAtestacionActoFuenteAutoridad es la vista que el repositorio relee
antes de consumir TokenConsumoRef. Las referencias acreditan registros
externos; este contrato no implementa ni simula criptografia.

```go
func (d DatosAtestacionActoFuenteAutoridad) Format(estado fmt.State, _ rune)

func (d DatosAtestacionActoFuenteAutoridad) GoString() string

func (*DatosAtestacionActoFuenteAutoridad) GobDecode([]byte) error

func (DatosAtestacionActoFuenteAutoridad) GobEncode() ([]byte, error)

func (d DatosAtestacionActoFuenteAutoridad) LogValue() slog.Value

func (DatosAtestacionActoFuenteAutoridad) MarshalBinary() ([]byte, error)

func (DatosAtestacionActoFuenteAutoridad) MarshalJSON() ([]byte, error)

func (DatosAtestacionActoFuenteAutoridad) MarshalText() ([]byte, error)

func (DatosAtestacionActoFuenteAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (DatosAtestacionActoFuenteAutoridad) String() string

func (*DatosAtestacionActoFuenteAutoridad) UnmarshalBinary([]byte) error

func (*DatosAtestacionActoFuenteAutoridad) UnmarshalJSON([]byte) error

func (*DatosAtestacionActoFuenteAutoridad) UnmarshalText([]byte) error

func (*DatosAtestacionActoFuenteAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

type DatosConsultaReconciliacionDocumentalV4 struct {
	ConsultaRef              string
	Reto                     RetoConsultaReconciliacionDocumentalV4
	ReservaRef               string
	EfectoRef                string
	HuellaPlanSHA256         string
	EstadoEsperado           EstadoEjecucionDocumentalV3
	VersionEsperada          uint64
	SecuenciaCercadoEsperada uint64
	EmitidaEn                time.Time
	ExpiraEn                 time.Time
}
```

DatosConsultaReconciliacionDocumentalV4 son datos declarativos. La consulta
resultante sigue sin acreditar frescura, origen servidor ni autorizacion:
el servicio interno debe generarla y persistir sus claves UNIQUE.

```go
type DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3 struct {
	HuellaSolicitudSHA256      string
	HuellaMaterialCrudoSHA256  string
	ComprobacionRef            string
	Algoritmo                  string
	Audiencia                  string
	Contexto                   string
	ClaveGestionadaRef         string
	RevisionClaveGestionada    uint64
	EvidenciaOperacionRef      string
	HuellaAtestacionSHA256     string
	ComprobadaEn               time.Time
	HuellaResultadoCrudoSHA256 string
}
```

DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3 permite que un
adaptador externo persista y enlace la respuesta nominal del KMS. Los campos
son material crudo restaurable: no prueban por si solos que la comprobacion
criptografica haya ocurrido ni autorizan a ejecutar un efecto.

```go
func (d DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) Format(
	estado fmt.State,
	_ rune,
)

func (d DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) GoString() string

func (d DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) LogValue() slog.Value

func (DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error)

func (DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error)

func (DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) MarshalText() ([]byte, error)

func (DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) String() string

func (*DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error

func (*DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error

func (*DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) UnmarshalText([]byte) error

type DatosEvidenciaRenderizadoDocumentalV3 struct {
	Esquema                       string
	ReservaRef                    string
	IndiceIdempotenciaHMAC        string
	HuellaSolicitudHMAC           string
	Manifiesto                    ManifiestoEjecucionDocumentalV3
	SecuenciaCercado              uint64
	HuellaVinculoSHA256           string
	ClaveAtestacionCercadoRef     string
	HuellaMACCercadoSHA256        string
	EvidenciaAtestacionCercadoRef string
	VerificacionCercadoRef        string
	VerificadoCercadoEn           time.Time
	ConsumoDecision               ConsumoDecisionEjecucionDocumentalV3
	Resultado                     ResultadoEfectoRenderizadoDocumentalV3Crudo
	Recibos                       HuellasRecibosEjecucionDocumentalV3
	GeneradoEn                    time.Time
	ConfirmadoEn                  time.Time
	ReconciliacionRef             string
	HuellaReconciliacionSHA256    string
	ReconciliacionConsultadaEn    time.Time
	VerificacionReconciliacionRef string
	ReconciliacionVerificadaEn    time.Time
}

func (d DatosEvidenciaRenderizadoDocumentalV3) Format(estado fmt.State, _ rune)

func (d DatosEvidenciaRenderizadoDocumentalV3) GoString() string

func (d DatosEvidenciaRenderizadoDocumentalV3) LogValue() slog.Value

func (DatosEvidenciaRenderizadoDocumentalV3) MarshalBinary() ([]byte, error)

func (DatosEvidenciaRenderizadoDocumentalV3) MarshalJSON() ([]byte, error)

func (DatosEvidenciaRenderizadoDocumentalV3) MarshalText() ([]byte, error)

func (DatosEvidenciaRenderizadoDocumentalV3) String() string

func (*DatosEvidenciaRenderizadoDocumentalV3) UnmarshalBinary([]byte) error

func (*DatosEvidenciaRenderizadoDocumentalV3) UnmarshalJSON([]byte) error

func (*DatosEvidenciaRenderizadoDocumentalV3) UnmarshalText([]byte) error

func (d DatosEvidenciaRenderizadoDocumentalV3) Validar() error

type DatosEvidenciaUsoDecisionAutorizacion struct {
	EsquemaHuella        string
	HuellaDecisionSHA256 string
	Decision             domain.DecisionAutorizacion
	VerificadaEn         time.Time
	// Has unexported fields.
}
```

DatosEvidenciaUsoDecisionAutorizacion es una proyeccion defensiva para
el adaptador duradero que vaya a consumir la decision dentro de la misma
transaccion que el efecto de negocio. No es una autorizacion serializable ni
permite reconstruir EvidenciaUsoDecisionAutorizacion.

Decision siempre se devuelve como copia profunda y en orden canonico.
El adaptador debe volver a comprobar en su propia transaccion la decision
registrada, su huella, la configuracion vigente y la identidad del efecto.

```go
func (d DatosEvidenciaUsoDecisionAutorizacion) Format(estado fmt.State, _ rune)

func (d DatosEvidenciaUsoDecisionAutorizacion) GoString() string

func (*DatosEvidenciaUsoDecisionAutorizacion) GobDecode([]byte) error

func (DatosEvidenciaUsoDecisionAutorizacion) GobEncode() ([]byte, error)

func (d DatosEvidenciaUsoDecisionAutorizacion) LogValue() slog.Value

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalBinary() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalCBOR() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalJSON() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalText() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalXML(*xml.Encoder, xml.StartElement) error

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalYAML() (any, error)

func (d DatosEvidenciaUsoDecisionAutorizacion) RepresentacionCanonica() ([]byte, error)
```

RepresentacionCanonica devuelve una copia de los bytes exactos sobre los que
se calculo HuellaDecisionSHA256. Es una salida deliberada para adaptadores
duraderos: les permite cotejar la decision registrada sin reimplementar ni
divergir del formato privado del nucleo.

La proyeccion completa sigue bloqueando codecs y formateo; este metodo no
convierte la evidencia opaca en una capacidad reconstruible.

```go
func (DatosEvidenciaUsoDecisionAutorizacion) String() string

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalBinary([]byte) error

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalCBOR([]byte) error

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalJSON([]byte) error

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalText([]byte) error

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalYAML(func(any) error) error

type DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 struct {
	EsquemaHuella        string
	HuellaDecisionSHA256 string
	Decision             domain.DecisionAutorizacion
	VerificadaEn         time.Time
	// Has unexported fields.
}
```

DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 es la proyeccion
defensiva y no reconstruible que un adaptador durable necesita para cotejar
una decision V2. No es asignable al contrato historico V1.

```go
func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) Format(estado fmt.State, _ rune)

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GoString() string

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GobDecode([]byte) error

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GobEncode() ([]byte, error)

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) LogValue() slog.Value

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalBinary() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalCBOR() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalJSON() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalText() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalYAML() (any, error)

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) RepresentacionCanonica() ([]byte, error)

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) String() string

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalBinary([]byte) error

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalCBOR([]byte) error

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalJSON([]byte) error

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalText([]byte) error

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalYAML(func(any) error) error

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) ValidarMotivo(
	motivo domain.ReferenciaEntradaCatalogo,
) error
```

ValidarMotivo coteja la referencia completa ya resuelta por la frontera
confiable. La existencia y vigencia del catalogo deben revalidarse en la
transaccion durable; esta huella acredita integridad, no procedencia.

```go
type DatosManifiestoEjecucionDocumentalV3 struct {
	Esquema               string
	Consulta              ConsultaFormatoDocumental
	DescriptorPerfil      DescriptorPerfilDocumental
	SituacionOperativa    domain.SituacionOperativaPerfilDocumental
	ComponenteRender      DescriptorComponenteDocumentalAtestado
	ComponenteVerificador DescriptorComponenteDocumentalAtestado
	ComponenteSemantico   DescriptorComponenteDocumentalAtestado
	BorradorRef           string
	EfectoRef             string
	HuellaEntradaHMAC     string
	LimiteEfectivoBytes   uint64
	HuellaPlanSHA256      string
}
```

DatosManifiestoEjecucionDocumentalV3 compromete una resolucion completa
y carente de datos personales directos. Las referencias son opacas: no se
admiten URL, rutas, comodines, direcciones de correo ni identificadores DNI.

```go
func (d DatosManifiestoEjecucionDocumentalV3) Format(estado fmt.State, _ rune)

func (d DatosManifiestoEjecucionDocumentalV3) GoString() string

func (d DatosManifiestoEjecucionDocumentalV3) LogValue() slog.Value

func (DatosManifiestoEjecucionDocumentalV3) MarshalBinary() ([]byte, error)

func (DatosManifiestoEjecucionDocumentalV3) MarshalJSON() ([]byte, error)

func (DatosManifiestoEjecucionDocumentalV3) MarshalText() ([]byte, error)

func (DatosManifiestoEjecucionDocumentalV3) String() string

func (*DatosManifiestoEjecucionDocumentalV3) UnmarshalBinary([]byte) error

func (*DatosManifiestoEjecucionDocumentalV3) UnmarshalJSON([]byte) error

func (*DatosManifiestoEjecucionDocumentalV3) UnmarshalText([]byte) error

type DatosMutacionOrdenCobro struct {
	Orden     domain.OrdenCobro
	Auditoria RegistroAuditoriaCobro
	Evento    EventoSalidaCobro
}

type DatosOperacionFuenteAutoridad struct {
	OperacionRef           string
	Solicitud              domain.SolicitudTransicionFuenteAutoridadV1
	EstadoEsperado         ReferenciaEstadoFuenteAutoridad
	Estado                 EstadoOperacionFuenteAutoridad
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	ResolucionRef          string
	PreparadaEn            time.Time
	ActualizadaEn          time.Time
	// Has unexported fields.
}
```

DatosOperacionFuenteAutoridad es una proyeccion interna reconstruible. Los
conectores persisten los bytes canonicos de Solicitud, no una copia de sus
parametros. Atestacion y resolucion son referencias a registros durables.

```go
func (d DatosOperacionFuenteAutoridad) Format(estado fmt.State, _ rune)

func (d DatosOperacionFuenteAutoridad) GoString() string

func (*DatosOperacionFuenteAutoridad) GobDecode([]byte) error

func (DatosOperacionFuenteAutoridad) GobEncode() ([]byte, error)

func (d DatosOperacionFuenteAutoridad) LogValue() slog.Value

func (DatosOperacionFuenteAutoridad) MarshalBinary() ([]byte, error)

func (DatosOperacionFuenteAutoridad) MarshalJSON() ([]byte, error)

func (DatosOperacionFuenteAutoridad) MarshalText() ([]byte, error)

func (DatosOperacionFuenteAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (DatosOperacionFuenteAutoridad) String() string

func (*DatosOperacionFuenteAutoridad) UnmarshalBinary([]byte) error

func (*DatosOperacionFuenteAutoridad) UnmarshalJSON([]byte) error

func (*DatosOperacionFuenteAutoridad) UnmarshalText([]byte) error

func (*DatosOperacionFuenteAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

type DatosOrdenDespachoDocumentalV3Nominal struct {
	ReciboInicio             DatosReciboInicioEfectoDocumentalV3Nominal
	HuellaReciboInicioSHA256 string
	ReclamacionRef           string
	ConsumidorRef            string
	VersionReclamacionCAS    uint64
	AuditoriaReclamacionRef  string
	AtestacionReclamacion    PruebaCrudaAtestacionDespachoDocumentalV3
	ReclamadaEn              time.Time
	ExpiraEn                 time.Time
}

func (d DatosOrdenDespachoDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (d DatosOrdenDespachoDocumentalV3Nominal) GoString() string

func (d DatosOrdenDespachoDocumentalV3Nominal) LogValue() slog.Value

func (DatosOrdenDespachoDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (DatosOrdenDespachoDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (DatosOrdenDespachoDocumentalV3Nominal) MarshalText() ([]byte, error)

func (DatosOrdenDespachoDocumentalV3Nominal) String() string

func (*DatosOrdenDespachoDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*DatosOrdenDespachoDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*DatosOrdenDespachoDocumentalV3Nominal) UnmarshalText([]byte) error

type DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2 struct {
	Decision         domain.DecisionAutorizacion
	ReferenciaMotivo domain.ReferenciaEntradaCatalogo
	// Has unexported fields.
}
```

DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2 es la copia
defensiva que recibe el adaptador durable. La referencia permite releer
en su propia transaccion el catalogo exacto y cotejar version, huella y
entrada.

```go
func (b DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) Format(estado fmt.State, _ rune)

func (b DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) GoString() string

func (*DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) GobDecode([]byte) error

func (DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) GobEncode() ([]byte, error)

func (b DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) LogValue() slog.Value

func (DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalBinary() ([]byte, error)

func (DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalCBOR() ([]byte, error)

func (DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalJSON() ([]byte, error)

func (DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalText() ([]byte, error)

func (DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalXML(*xml.Encoder, xml.StartElement) error

func (DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalYAML() (any, error)

func (DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) String() string

func (*DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalBinary([]byte) error

func (*DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalCBOR([]byte) error

func (*DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalJSON([]byte) error

func (*DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalText([]byte) error

func (*DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalYAML(func(any) error) error

type DatosReciboConsultaInternaFuenteAutoridad struct {
	TransaccionRef                 string
	Selector                       SelectorVersionFuenteAutoridad
	Resultado                      ResultadoConsultaFuenteAutoridad
	Estado                         ReferenciaEstadoFuenteAutoridad
	DecisionRef                    string
	HuellaDecisionSHA256           string
	AuditoriaRef                   string
	AuditoriaSecuencia             int64
	AuditoriaAlgoritmoIntegridad   string
	AuditoriaEncadenadoAnteriorRef string
	AuditoriaFirmaRef              string
	AuditoriaConfirmada            domain.AuditEntry
	AuditoriaHuellaEntradaSHA256   string
	HuellaCompromisoReciboSHA256   string
	ConfirmadaEn                   time.Time
	// Has unexported fields.
}
```

DatosReciboConsultaInternaFuenteAutoridad es una proyeccion interna. La
entrada confirmada queda minimizada y su copia defensiva permite comprobar
el registro firmado que ya existe tras el COMMIT, no solo una referencia.

```go
func (b DatosReciboConsultaInternaFuenteAutoridad) Format(estado fmt.State, _ rune)

func (b DatosReciboConsultaInternaFuenteAutoridad) GoString() string

func (*DatosReciboConsultaInternaFuenteAutoridad) GobDecode([]byte) error

func (DatosReciboConsultaInternaFuenteAutoridad) GobEncode() ([]byte, error)

func (b DatosReciboConsultaInternaFuenteAutoridad) LogValue() slog.Value

func (DatosReciboConsultaInternaFuenteAutoridad) MarshalBinary() ([]byte, error)

func (DatosReciboConsultaInternaFuenteAutoridad) MarshalCBOR() ([]byte, error)

func (DatosReciboConsultaInternaFuenteAutoridad) MarshalJSON() ([]byte, error)

func (DatosReciboConsultaInternaFuenteAutoridad) MarshalText() ([]byte, error)

func (DatosReciboConsultaInternaFuenteAutoridad) MarshalXML(*xml.Encoder, xml.StartElement) error

func (DatosReciboConsultaInternaFuenteAutoridad) MarshalYAML() (any, error)

func (DatosReciboConsultaInternaFuenteAutoridad) String() string

func (*DatosReciboConsultaInternaFuenteAutoridad) UnmarshalBinary([]byte) error

func (*DatosReciboConsultaInternaFuenteAutoridad) UnmarshalCBOR([]byte) error

func (*DatosReciboConsultaInternaFuenteAutoridad) UnmarshalJSON([]byte) error

func (*DatosReciboConsultaInternaFuenteAutoridad) UnmarshalText([]byte) error

func (*DatosReciboConsultaInternaFuenteAutoridad) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*DatosReciboConsultaInternaFuenteAutoridad) UnmarshalYAML(func(any) error) error

type DatosReciboEjecucionComponenteDocumentalNominal struct {
	HuellaCompromisoSHA256     string
	OperacionRef               string
	Operacion                  OperacionComponenteDocumental
	DescriptorPerfil           DescriptorPerfilDocumental
	SituacionOperativa         domain.SituacionOperativaPerfilDocumental
	DescriptorComponente       DescriptorComponenteDocumentalAtestado
	ReservaRef                 string
	EfectoRef                  string
	HuellaPlanSHA256           string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	DecisionRef                string
	EsquemaHuellaDecision      string
	HuellaDecisionSHA256       string
	InicioEfectoRef            string
	OutboxInicioRef            string
	ReclamacionDespachoRef     string
	ConsumoDespachoRef         string
	OutboxConsumoRef           string
	VersionInicioCAS           uint64
	VersionReclamacionCAS      uint64
	VersionConsumoCAS          uint64
	HuellaOrdenDespachoSHA256  string
	ComprobacionKMSRef         string
	ConsumidaEn                time.Time
	BorradorRef                string
	HuellaContenidoNeutralHMAC string
	HuellaDocumentoSHA256      string
	TamanoDocumento            uint64
	LimiteBytes                uint64
	CompromisoEmitidoEn        time.Time
	CompromisoExpiraEn         time.Time
	ReciboRef                  string
	Resultado                  ResultadoEjecucionComponenteDocumental
	HuellaSalidaSHA256         string
	TamanoSalida               uint64
	Identidad                  IdentidadEjecucionComponenteDocumental
	EmitidoEn                  time.Time
	HuellaSobreCOSESHA256      string
	HuellaReciboSHA256         string
}
```

DatosReciboEjecucionComponenteDocumentalNominal es una proyeccion segura
para evidencia. Nunca expone CompromisoEjecucionComponenteDocumental,
TokenCercadoEjecucionDocumentalV3Nominal, su MAC ni el valor secreto de
cercado.

```go
func (d DatosReciboEjecucionComponenteDocumentalNominal) Format(estado fmt.State, _ rune)

func (d DatosReciboEjecucionComponenteDocumentalNominal) GoString() string

func (d DatosReciboEjecucionComponenteDocumentalNominal) LogValue() slog.Value

func (DatosReciboEjecucionComponenteDocumentalNominal) MarshalBinary() ([]byte, error)

func (DatosReciboEjecucionComponenteDocumentalNominal) MarshalJSON() ([]byte, error)

func (DatosReciboEjecucionComponenteDocumentalNominal) MarshalText() ([]byte, error)

func (DatosReciboEjecucionComponenteDocumentalNominal) String() string

func (*DatosReciboEjecucionComponenteDocumentalNominal) UnmarshalBinary([]byte) error

func (*DatosReciboEjecucionComponenteDocumentalNominal) UnmarshalJSON([]byte) error

func (*DatosReciboEjecucionComponenteDocumentalNominal) UnmarshalText([]byte) error

type DatosReciboInicioEfectoDocumentalV3Nominal struct {
	InicioRef                  string
	ReservaRef                 string
	HuellaVinculoEstableSHA256 string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	OrdenConsumoDurableV4Ref   string
	VersionInicioCAS           uint64
	AuditoriaInicioRef         string
	OutboxInicioRef            string
	AtestacionInicio           PruebaCrudaAtestacionDespachoDocumentalV3
	IniciadoEn                 time.Time
}
```

DatosReciboInicioEfectoDocumentalV3Nominal es la proyeccion nominal del
COMMIT que hizo el CAS activa -> iniciada. Puede persistirse como columnas,
pero una copia construida por el llamador nunca concede autoridad de
despacho.

```go
func (d DatosReciboInicioEfectoDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (d DatosReciboInicioEfectoDocumentalV3Nominal) GoString() string

func (d DatosReciboInicioEfectoDocumentalV3Nominal) LogValue() slog.Value

func (DatosReciboInicioEfectoDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (DatosReciboInicioEfectoDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (DatosReciboInicioEfectoDocumentalV3Nominal) MarshalText() ([]byte, error)

func (DatosReciboInicioEfectoDocumentalV3Nominal) String() string

func (*DatosReciboInicioEfectoDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*DatosReciboInicioEfectoDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*DatosReciboInicioEfectoDocumentalV3Nominal) UnmarshalText([]byte) error

type DatosSalidaObservadaDocumental struct {
	HuellaSHA256 string
	Tamano       uint64
	LimiteBytes  uint64
}
```

DatosSalidaObservadaDocumental es una proyeccion no autoritativa. La salida
opaca que la origina solo puede fabricarla SumideroLimitadoSalidaDocumental.

```go
type DatosSelloEvidenciaDocumentalV3Crudos struct {
	Algoritmo             string
	ClaveID               string
	Audiencia             string
	HuellaMensajeSHA256   string
	Firma                 []byte
	EvidenciaOperacionRef string
	FirmadoEn             time.Time
}

func (d DatosSelloEvidenciaDocumentalV3Crudos) Format(estado fmt.State, _ rune)

func (d DatosSelloEvidenciaDocumentalV3Crudos) GoString() string

func (d DatosSelloEvidenciaDocumentalV3Crudos) LogValue() slog.Value

func (DatosSelloEvidenciaDocumentalV3Crudos) MarshalBinary() ([]byte, error)

func (DatosSelloEvidenciaDocumentalV3Crudos) MarshalJSON() ([]byte, error)

func (DatosSelloEvidenciaDocumentalV3Crudos) MarshalText() ([]byte, error)

func (DatosSelloEvidenciaDocumentalV3Crudos) String() string

func (*DatosSelloEvidenciaDocumentalV3Crudos) UnmarshalBinary([]byte) error

func (*DatosSelloEvidenciaDocumentalV3Crudos) UnmarshalJSON([]byte) error

func (*DatosSelloEvidenciaDocumentalV3Crudos) UnmarshalText([]byte) error

type DeclaracionEscrituraAlmacenDocumental struct {
	// Has unexported fields.
}
```

DeclaracionEscrituraAlmacenDocumental es una copia opaca de la preparacion
exacta que debe firmar el conector. Sus campos no son reconstruibles desde
un DTO y su validez sintactica no concede autoridad.

```go
func NuevaDeclaracionEscrituraAlmacenDocumental(
	preparacion PreparacionEscrituraAlmacenDocumentalV4Nominal,
) (DeclaracionEscrituraAlmacenDocumental, error)

func (d DeclaracionEscrituraAlmacenDocumental) Capacidades() (
	CapacidadesAlmacenObjetos,
	error,
)

func (d DeclaracionEscrituraAlmacenDocumental) EvidenciaOperacion() (
	EvidenciaOperacionAlmacen,
	error,
)

func (d DeclaracionEscrituraAlmacenDocumental) Format(estado fmt.State, _ rune)

func (d DeclaracionEscrituraAlmacenDocumental) GoString() string

func (d DeclaracionEscrituraAlmacenDocumental) LogValue() slog.Value

func (DeclaracionEscrituraAlmacenDocumental) MarshalBinary() ([]byte, error)

func (DeclaracionEscrituraAlmacenDocumental) MarshalJSON() ([]byte, error)

func (DeclaracionEscrituraAlmacenDocumental) MarshalText() ([]byte, error)

func (d DeclaracionEscrituraAlmacenDocumental) Objeto() (ObjetoAlmacenado, error)

func (d DeclaracionEscrituraAlmacenDocumental) Politica() (
	VinculoPoliticaInmutabilidadDocumental,
	error,
)

func (DeclaracionEscrituraAlmacenDocumental) String() string

func (*DeclaracionEscrituraAlmacenDocumental) UnmarshalBinary([]byte) error

func (*DeclaracionEscrituraAlmacenDocumental) UnmarshalJSON([]byte) error

func (*DeclaracionEscrituraAlmacenDocumental) UnmarshalText([]byte) error

func (d DeclaracionEscrituraAlmacenDocumental) Validar() error

func (d DeclaracionEscrituraAlmacenDocumental) ValidarContraEjecucion(
	ordenConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
	salida SalidaObservadaDocumental,
) error

func (d DeclaracionEscrituraAlmacenDocumental) ValidarContraSalida(
	salida SalidaObservadaDocumental,
) error

func (d DeclaracionEscrituraAlmacenDocumental) VinculoEjecucion() (
	VinculoEjecucionEscrituraAlmacenDocumental,
	error,
)

type DeclaracionRepresentacionGeneracionDocumental struct {
	ReferenciaLogica  string
	ClaveIdempotencia string
	Formato           domain.FormatoDocumento
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
}
```

DeclaracionRepresentacionGeneracionDocumental contiene los metadatos de
los bytes ya renderizados. No concede acceso. La fabrica los canoniza y los
compromete antes de solicitar una decision al PDP.

```go
type DeclaracionRespuestaReconciliacionDocumentalV4 struct {
	ConsultaRef          string
	HuellaConsultaSHA256 string
	HuellaRetoSHA256     string
	ReservaRef           string
	EfectoRef            string
	HuellaPlanSHA256     string
	// Estos tres valores son un eco firmado de la condicion CAS enviada por
	// el servidor. No representan una lectura remota ni demuestran por si solos
	// el estado local o la existencia/ausencia del efecto.
	EstadoReservaEsperadoEco    EstadoEjecucionDocumentalV3
	VersionReservaEsperadaEco   uint64
	SecuenciaCercadoEsperadaEco uint64
	Resultado                   EstadoResultadoReconciliacionDocumentalV4
	HuellaEfectoAplicadoSHA256  string
	TamanoEfectoAplicado        uint64
	RespondidaEn                time.Time
}
```

DeclaracionRespuestaReconciliacionDocumentalV4 refleja lo que
afirma el componente remoto. Sigue siendo entrada hostil hasta que
application/internal verifique el COSE, la clave, el algoritmo, la audiencia
y este payload exacto.

```go
type DescriptorComponenteDocumentalAtestado struct {
	// Has unexported fields.
}
```

DescriptorComponenteDocumentalAtestado es una declaracion de valor emitida
por el registro/broker. El ejecutable no declara su ID ni su digest.
La verificacion criptografica de la atestacion corresponde al adaptador del
broker antes de construir este valor.

```go
func NuevoDescriptorComponenteDocumentalAtestado(
	referencia string,
	consulta ConsultaComponenteDocumentalAtestado,
	componente domain.ReferenciaComponenteDocumental,
	dominioConfianzaRef, brokerRef, atestacionBrokerRef, huellaAtestacionBrokerSHA256 string,
	maximoBytes uint64,
) (DescriptorComponenteDocumentalAtestado, error)

func (d DescriptorComponenteDocumentalAtestado) AtestacionBrokerRef() string

func (d DescriptorComponenteDocumentalAtestado) BrokerRef() string

func (d DescriptorComponenteDocumentalAtestado) Coincide(
	consulta ConsultaComponenteDocumentalAtestado,
) bool

func (d DescriptorComponenteDocumentalAtestado) Componente() domain.ReferenciaComponenteDocumental

func (d DescriptorComponenteDocumentalAtestado) Consulta() ConsultaComponenteDocumentalAtestado

func (d DescriptorComponenteDocumentalAtestado) DigestDeclaracionSHA256() string

func (d DescriptorComponenteDocumentalAtestado) DominioConfianzaRef() string

func (d DescriptorComponenteDocumentalAtestado) HuellaAtestacionBrokerSHA256() string

func (d DescriptorComponenteDocumentalAtestado) IndependienteDe(
	otro DescriptorComponenteDocumentalAtestado,
) bool
```

IndependienteDe exige segregacion real: rol, ID/version, artefacto,
homologacion y dominio de confianza diferentes. Renombrar el mismo binario
bajo otra funcion no crea una barrera independiente.

```go
func (d DescriptorComponenteDocumentalAtestado) MaximoBytes() uint64

func (d DescriptorComponenteDocumentalAtestado) Referencia() string

func (d DescriptorComponenteDocumentalAtestado) Validar() error

type DescriptorFormatoDocumental struct {
	// Has unexported fields.
}
```

DescriptorFormatoDocumental no contiene codigo, comandos, rutas, URL ni
configuracion libre. Vincula un perfil inmutable a una revision concreta y a
una unica version de conector homologado.

```go
func NuevoDescriptorFormatoDocumental(
	referencia string,
	perfil domain.PerfilFormatoDocumental,
	revision domain.RevisionCatalogoFormatosDocumentales,
	conector domain.ReferenciaConectorDocumental,
) (DescriptorFormatoDocumental, error)

func (d DescriptorFormatoDocumental) Coincide(c ConsultaFormatoDocumental) bool

func (d DescriptorFormatoDocumental) Conector() domain.ReferenciaConectorDocumental

func (d DescriptorFormatoDocumental) Perfil() domain.PerfilFormatoDocumental

func (d DescriptorFormatoDocumental) Referencia() string

func (d DescriptorFormatoDocumental) Revision() domain.RevisionCatalogoFormatosDocumentales

func (d DescriptorFormatoDocumental) Validar() error

type DescriptorPerfilDocumental struct {
	// Has unexported fields.
}
```

DescriptorPerfilDocumental es la declaracion de catalogo V2. No contiene
ejecutores ni componentes: solo enlaza el perfil inmutable con la
publicacion operativa que debe releerse antes de cada efecto.

```go
func NuevoDescriptorPerfilDocumental(
	referencia, publicacionRef string,
	perfil domain.PerfilFormatoDocumental,
	revision domain.RevisionCatalogoFormatosDocumentales,
) (DescriptorPerfilDocumental, error)

func (d DescriptorPerfilDocumental) Coincide(c ConsultaFormatoDocumental) bool

func (d DescriptorPerfilDocumental) Perfil() domain.PerfilFormatoDocumental

func (d DescriptorPerfilDocumental) PublicacionRef() string

func (d DescriptorPerfilDocumental) Referencia() string

func (d DescriptorPerfilDocumental) Revision() domain.RevisionCatalogoFormatosDocumentales

func (d DescriptorPerfilDocumental) Validar() error

type DespachadorComponentesDocumentalesAtestados interface {
	// Cada metodo acepta un compromiso nominal ligado al consumo CAS. Este puerto
	// solo puede estar en el servicio de aplicacion precompuesto, que verifica KMS,
	// consume y despacha dentro de la misma llamada. Nunca se entrega a handlers.
	Renderizar(
		context.Context,
		CompromisoEjecucionComponenteDocumental,
		domain.PerfilFormatoDocumental,
		domain.ContenidoDocumento,
		io.Writer,
	) (SobreReciboEjecucionDocumentalCrudo, error)
	ValidarEstructura(
		context.Context,
		CompromisoEjecucionComponenteDocumental,
		domain.PerfilFormatoDocumental,
		[]byte,
	) (SobreReciboEjecucionDocumentalCrudo, error)
	VerificarSemantica(
		context.Context,
		CompromisoEjecucionComponenteDocumental,
		domain.PerfilFormatoDocumental,
		domain.ContenidoDocumento,
		[]byte,
	) (SobreReciboEjecucionDocumentalCrudo, error)
}
```

DespachadorComponentesDocumentalesAtestados es transporte hacia procesos
aislados; no es por si mismo un ejecutor homologado. El recibo firmado por
la carga de trabajo es obligatorio para aceptar cualquier resultado.

```go
type DeteccionContenido struct {
	Clase    ClaseDeteccionContenido
	Codigo   string
	FirmaRef string
}
```

DeteccionContenido conserva codigos normalizados, nunca la salida cruda del
motor, contenido, nombres originales ni datos personales.

```go
func (d DeteccionContenido) Validar() error

type DocumentArchivePort interface {
	ArchiveDocument(context.Context, InteropRequest) (InteropResult, error)
}

type EUDIWalletPort interface {
	VerifyPresentation(context.Context, InteropRequest) (InteropResult, error)
}

type EjecutorExtraccionMetadatoInstitucional interface {
	// Debe analizar el documento completo y fallar ante metadato ausente,
	// duplicado o escondido en comentarios/zero-width/canales no autorizados.
	ExtraerYValidarMetadatoInstitucional(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		[]byte,
		uint64,
	) (ResultadoExtraccionMetadatoInstitucional, error)
}

type EjecutorMarcadoInstitucionalDocumental interface {
	IncorporarMetadatoAntesFirma(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		[]byte,
		string,
		domain.MarcaInstitucionalDocumento,
		uint64,
		io.Writer,
	) error
}

type EjecutorRenderizadoDocumental interface {
	Renderizar(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		domain.ContenidoDocumento,
		uint64,
		io.Writer,
	) error
}
```

Los ejecutores son contratos por operacion. No exponen identidad propia y
nunca se devuelven desde un resultado de aplicacion.

```go
type EjecutorValidacionConformidadDocumental interface {
	ValidarConformidad(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		[]byte,
		uint64,
	) error
}

type EjecutorVerificacionSemanticaDocumental interface {
	VerificarEquivalenciaSemantica(
		context.Context,
		DescriptorComponenteDocumentalAtestado,
		domain.PerfilFormatoDocumental,
		[]byte,
		[]byte,
		uint64,
	) error
}

type EmisorReciboCargaDirecta interface {
	EmitirReciboCargaDirecta(
		context.Context,
		SolicitudEmitirReciboCargaDirecta,
	) (ReciboCargaDirecta, error)
}

type EntradaNeutralDocumentalNominal struct {
	// Has unexported fields.
}
```

EntradaNeutralDocumentalNominal es opaca e inmutable. Puede contener datos
personales, por lo que no ofrece serializacion generica ni representaciones
de depuracion que revelen el contenido.

```go
func NuevaEntradaNeutralDocumentalNominal(
	preparacion PreparacionEntradaNeutralDocumentalNominal,
	huellaHMACDeclarada string,
) (EntradaNeutralDocumentalNominal, error)
```

NuevaEntradaNeutralDocumentalNominal asocia una HMAC declarada a la
preparacion ya fijada, sin verificarla criptograficamente. Ni esta fabrica
ni la salida nominal de un conector conceden por si solas capacidad de uso.

```go
func (e EntradaNeutralDocumentalNominal) CanonicalizacionRef() (string, error)

func (e EntradaNeutralDocumentalNominal) Contenido() (domain.ContenidoDocumento, error)

func (e EntradaNeutralDocumentalNominal) ContenidoCanonico() ([]byte, error)

func (e EntradaNeutralDocumentalNominal) Format(estado fmt.State, _ rune)

func (e EntradaNeutralDocumentalNominal) GoString() string

func (e EntradaNeutralDocumentalNominal) HuellaHMACDeclarada() (string, error)
```

HuellaHMACDeclarada es una vinculacion nominal pendiente de comprobacion
por el servicio de aplicacion mediante su conector criptografico privado.
Su formato valido no demuestra que se haya calculado con una clave confiable
ni habilita por si solo ningun efecto.

```go
func (e EntradaNeutralDocumentalNominal) LogValue() slog.Value

func (EntradaNeutralDocumentalNominal) MarshalBinary() ([]byte, error)

func (EntradaNeutralDocumentalNominal) MarshalJSON() ([]byte, error)

func (EntradaNeutralDocumentalNominal) MarshalText() ([]byte, error)

func (EntradaNeutralDocumentalNominal) String() string

func (e EntradaNeutralDocumentalNominal) Tamano() (uint64, error)

func (*EntradaNeutralDocumentalNominal) UnmarshalBinary([]byte) error

func (*EntradaNeutralDocumentalNominal) UnmarshalJSON([]byte) error

func (*EntradaNeutralDocumentalNominal) UnmarshalText([]byte) error

func (e EntradaNeutralDocumentalNominal) Validar() error

type EstadoAnalisisContenido string

const (
	EstadoAnalisisContenidoLimpio        EstadoAnalisisContenido = "limpio"
	EstadoAnalisisContenidoMalicioso     EstadoAnalisisContenido = "malicioso"
	EstadoAnalisisContenidoSospechoso    EstadoAnalisisContenido = "sospechoso"
	EstadoAnalisisContenidoNoConcluyente EstadoAnalisisContenido = "no_concluyente"
	EstadoAnalisisContenidoError         EstadoAnalisisContenido = "error"
)
func (e EstadoAnalisisContenido) Valido() bool

type EstadoControlLiquidacionCobro string
```

EstadoControlLiquidacionCobro es deliberadamente mas estrecho que el
catalogo funcional de liquidaciones. En el commit de un alta solo existe un
estado positivo: cualquier otro valor, incluido uno futuro, deniega.

```go
const EstadoControlLiquidacionCobroExigible EstadoControlLiquidacionCobro = "exigible"
type EstadoCrudoOrdenDespachoDocumentalV3 struct {
	// Has unexported fields.
}
```

EstadoCrudoOrdenDespachoDocumentalV3 es la relectura nominal del registro
privado posterior a la reclamacion. Nunca concede autoridad por si solo.

```go
func NuevoEstadoCrudoOrdenDespachoDocumentalV3(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	resultado ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
	estadoRef, auditoriaRef, consumoRef, outboxConsumoRef string,
	versionConsumoCAS uint64,
	consumidaEn time.Time,
) (EstadoCrudoOrdenDespachoDocumentalV3, error)

func (e EstadoCrudoOrdenDespachoDocumentalV3) Format(estado fmt.State, _ rune)

func (e EstadoCrudoOrdenDespachoDocumentalV3) GoString() string

func (e EstadoCrudoOrdenDespachoDocumentalV3) LogValue() slog.Value

func (EstadoCrudoOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error)

func (EstadoCrudoOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error)

func (EstadoCrudoOrdenDespachoDocumentalV3) MarshalText() ([]byte, error)

func (EstadoCrudoOrdenDespachoDocumentalV3) String() string

func (*EstadoCrudoOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error

func (*EstadoCrudoOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error

func (*EstadoCrudoOrdenDespachoDocumentalV3) UnmarshalText([]byte) error

func (e EstadoCrudoOrdenDespachoDocumentalV3) ValidarPara(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	resultado ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
) error

type EstadoEjecucionDocumentalV3 string

const (
	EstadoEjecucionDocumentalV3Preparada           EstadoEjecucionDocumentalV3 = "preparada"
	EstadoEjecucionDocumentalV3Activa              EstadoEjecucionDocumentalV3 = "activa"
	EstadoEjecucionDocumentalV3EfectoIniciado      EstadoEjecucionDocumentalV3 = "efecto_iniciado"
	EstadoEjecucionDocumentalV3Indeterminada       EstadoEjecucionDocumentalV3 = "indeterminada"
	EstadoEjecucionDocumentalV3Confirmada          EstadoEjecucionDocumentalV3 = "confirmada"
	EstadoEjecucionDocumentalV3AbandonadaSinEfecto EstadoEjecucionDocumentalV3 = "abandonada_sin_efecto"
)
func (e EstadoEjecucionDocumentalV3) PuedeTransicionarA(siguiente EstadoEjecucionDocumentalV3) bool

func (e EstadoEjecucionDocumentalV3) Valido() bool

type EstadoInmovilizacionObjetoMaterialV2 string
```

EstadoInmovilizacionObjetoMaterialV2 evita representar el bloqueo legal
con un booleano ambiguo. La ausencia y cualquier estado desconocido son
invalidos.

```go
const (
	EstadoInmovilizacionMaterialNoAplicada EstadoInmovilizacionObjetoMaterialV2 = "no_inmovilizado"
	EstadoInmovilizacionMaterialAplicada   EstadoInmovilizacionObjetoMaterialV2 = "inmovilizado"
)
type EstadoObjetoMaterialV2 string
```

EstadoObjetoMaterialV2 es una lista positiva cerrada. Este primer corte solo
emite recibos para objetos activos y no eliminados.

```go
const EstadoObjetoMaterialActivo EstadoObjetoMaterialV2 = "activo"
type EstadoOperacionFuenteAutoridad string

const (
	EstadoOperacionFuenteAutoridadPendiente  EstadoOperacionFuenteAutoridad = "pendiente"
	EstadoOperacionFuenteAutoridadAtestada   EstadoOperacionFuenteAutoridad = "atestada"
	EstadoOperacionFuenteAutoridadConfirmada EstadoOperacionFuenteAutoridad = "confirmada"
	EstadoOperacionFuenteAutoridadCancelada  EstadoOperacionFuenteAutoridad = "cancelada"
	EstadoOperacionFuenteAutoridadExpirada   EstadoOperacionFuenteAutoridad = "expirada"
	EstadoOperacionFuenteAutoridadObsoleta   EstadoOperacionFuenteAutoridad = "obsoleta"
)
func (e EstadoOperacionFuenteAutoridad) Terminal() bool

func (e EstadoOperacionFuenteAutoridad) Valido() bool

type EstadoPasoDuraderoGeneracionDocumental struct {
	PasoRef               PasoOperacionAlmacen
	HuellaPasoSHA256      string
	Estado                EstadoPasoEfectoGeneracionDocumental
	Objeto                ReferenciaObjetoAlmacen
	ConectorID            string
	EvidenciaOperacionRef string
	IncidenteRef          string
}
```

EstadoPasoDuraderoGeneracionDocumental es una proyeccion no autoritativa
de la reserva. Un paso confirmado conserva tambien el conector exacto para
que un replay pueda reconstruir la misma auditoria sin inventar metadatos.
Un paso indeterminado no contiene un objeto asumido: requiere reconciliacion
expresa antes de confirmar o volver a ejecutar.

```go
type EstadoPasoEfectoGeneracionDocumental string

const (
	EstadoPasoEfectoDocumentalReservado     EstadoPasoEfectoGeneracionDocumental = "reservado"
	EstadoPasoEfectoDocumentalConfirmado    EstadoPasoEfectoGeneracionDocumental = "confirmado"
	EstadoPasoEfectoDocumentalIndeterminado EstadoPasoEfectoGeneracionDocumental = "indeterminado"
)
type EstadoResultadoReconciliacionDocumentalV3 string

const (
	ResultadoReconciliacionDocumentalV3AplicadoExacto EstadoResultadoReconciliacionDocumentalV3 = "aplicado_exacto"
	ResultadoReconciliacionDocumentalV3NoAplicado     EstadoResultadoReconciliacionDocumentalV3 = "no_aplicado_atestado"
	ResultadoReconciliacionDocumentalV3Desconocido    EstadoResultadoReconciliacionDocumentalV3 = "desconocido"
	ResultadoReconciliacionDocumentalV3Conflictivo    EstadoResultadoReconciliacionDocumentalV3 = "conflictivo"
)
func (e EstadoResultadoReconciliacionDocumentalV3) Valido() bool

type EstadoResultadoReconciliacionDocumentalV4 string

const (
	EstadoResultadoReconciliacionDocumentalV4Aplicado    EstadoResultadoReconciliacionDocumentalV4 = "aplicado"
	EstadoResultadoReconciliacionDocumentalV4NoAplicado  EstadoResultadoReconciliacionDocumentalV4 = "no_aplicado"
	EstadoResultadoReconciliacionDocumentalV4Desconocido EstadoResultadoReconciliacionDocumentalV4 = "desconocido"
)
func (e EstadoResultadoReconciliacionDocumentalV4) Valido() bool

type EtapaMetadatoInstitucionalDocumental string

const EtapaMetadatoInstitucionalAntesFirma EtapaMetadatoInstitucionalDocumental = "antes_firma"
type EvaluadorReglasFlujo interface {
	EvaluarReglaFlujo(context.Context, SolicitudEvaluarReglaFlujo) (domain.DecisionReglaFlujo, error)
}
```

EvaluadorReglasFlujo resuelve ReglaRef y obtiene los hechos mediante
referencias internas. La solicitud no transporta payloads personales para
evitar que estos terminen accidentalmente en trazas o colas.

```go
type EventStore interface {
	PublishEvent(context.Context, domain.Event) error
	ListEvents(context.Context, []string) ([]domain.Event, error)
}

type EventoSalidaCobro = pagoscanonicos.EventoSalidaCobro

func NuevoEventoSalidaCobro(orden domain.OrdenCobro) (EventoSalidaCobro, error)
```

NuevoEventoSalidaCobro deriva el mensaje completo, incluido un identificador
determinista ligado a orden, version, secuencia y huella del ultimo hecho.
No recibe ningun campo semantico del llamador.

```go
type EvidenciaOperacionAlmacen struct {
	Referencia             string
	ConectorID             string
	EsquemaContexto        string
	AccionNegocio          string
	Accion                 string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	HuellaManifiestoSHA256 string
	HuellaPasoSHA256       string
	PasoRef                PasoOperacionAlmacen
	HuellaDecisionSHA256   string
	Objeto                 ReferenciaObjetoAlmacen
	OperacionRef           string
	CorrelacionRef         string
	AutorizacionRef        string
	Finalidad              string
	Clasificacion          string
	RealizadaEn            time.Time
	CargaRef               string
	SujetoSeudonimoHMAC    string
	RecursoRef             string
	ModuloID               string
	HuellaSolicitudHMAC    string
	FundamentoRef          string
	// ReintentoIdempotente distingue una respuesta repetida de la evidencia
	// que creo realmente el objeto. La accion no cambia ni se relaja.
	ReintentoIdempotente bool
}
```

EvidenciaOperacionAlmacen es el recibo tecnico que el caso de uso incorpora
a la auditoria probatoria. No contiene rutas, URL, nombres ni datos de la
persona interesada.

```go
func (e EvidenciaOperacionAlmacen) Validar() error

func (e EvidenciaOperacionAlmacen) ValidarEliminacion(
	solicitud SolicitudEliminarObjeto,
	anterior ObjetoAlmacenado,
) error
```

ValidarEliminacion mantiene la eliminacion como operacion privilegiada
separada: exige aprobacion exacta y rechaza bloqueo o retencion vigentes en
el instante acreditado por el conector.

```go
type EvidenciaUsoDecisionAutorizacion struct {
	// Has unexported fields.
}
```

EvidenciaUsoDecisionAutorizacion es una capacidad opaca e inmutable dentro
del proceso. Su valor cero no es valido y sus campos no pueden rellenarse
con un literal desde otro paquete.

Esta evidencia solo fija la decision que debe consumirse. No sustituye el
consumo atomico con el efecto de negocio ni acredita que una transaccion
de base de datos haya llegado a COMMIT. Su huella no es una firma ni una
atestacion del PDP: por si sola tampoco demuestra que el autorizador no haya
sido suplantado. Esa procedencia exige cableado interno confiable y, cuando
corresponda, una atestacion separada verificable por el adaptador duradero.

```go
func NuevaEvidenciaUsoDecisionAutorizacion(
	decision domain.DecisionAutorizacion,
	verificadaEn time.Time,
) (EvidenciaUsoDecisionAutorizacion, error)
```

NuevaEvidenciaUsoDecisionAutorizacion crea una evidencia exclusivamente
a partir de una decision reforzada, concedida y vigente en verificadaEn.
El instante procede del reloj confiable del servidor y debe estar en UTC con
precision de microsegundo, igual que la persistencia PostgreSQL prevista.

Mientras no exista una prueba tipada del cumplimiento de obligaciones,
una decision que las contenga se deniega. Una futura ampliacion debe
incorporarlas de forma positiva; nunca puede ignorarlas silenciosamente.

```go
func (e EvidenciaUsoDecisionAutorizacion) Datos() (DatosEvidenciaUsoDecisionAutorizacion, error)
```

Datos devuelve una copia profunda de la proyeccion necesaria para el
adaptador. La comprobacion estructural no implica vigencia actual: antes de
un efecto debe usarse ValidarEn y, en produccion, revalidarse y consumirse
la decision dentro de la transaccion que confirma dicho efecto.

```go
func (e EvidenciaUsoDecisionAutorizacion) Format(estado fmt.State, _ rune)

func (e EvidenciaUsoDecisionAutorizacion) GoString() string

func (*EvidenciaUsoDecisionAutorizacion) GobDecode([]byte) error

func (EvidenciaUsoDecisionAutorizacion) GobEncode() ([]byte, error)

func (e EvidenciaUsoDecisionAutorizacion) LogValue() slog.Value

func (EvidenciaUsoDecisionAutorizacion) MarshalBinary() ([]byte, error)

func (EvidenciaUsoDecisionAutorizacion) MarshalCBOR() ([]byte, error)

func (EvidenciaUsoDecisionAutorizacion) MarshalJSON() ([]byte, error)
```

La capacidad opaca no se serializa. Solo Datos permite una extraccion
deliberada, defensiva y tipada dentro de un adaptador de salida.

```go
func (EvidenciaUsoDecisionAutorizacion) MarshalText() ([]byte, error)

func (EvidenciaUsoDecisionAutorizacion) MarshalXML(*xml.Encoder, xml.StartElement) error

func (EvidenciaUsoDecisionAutorizacion) MarshalYAML() (any, error)

func (EvidenciaUsoDecisionAutorizacion) String() string

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalBinary([]byte) error

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalCBOR([]byte) error

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalJSON([]byte) error

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalText([]byte) error

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalYAML(func(any) error) error

func (e EvidenciaUsoDecisionAutorizacion) ValidarEn(instante time.Time) error
```

ValidarEn vuelve a comprobar alcance temporal usando un instante
efectivo del servidor. Se rechaza tambien un reloj anterior a la primera
verificacion: un retroceso temporal nunca recupera una capacidad ya emitida.

```go
type EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 struct {
	// Has unexported fields.
}
```

EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 es una capacidad opaca
exclusiva de efectos V2. No existe conversion desde V1 ni constructor desde
bytes o una proyeccion historica.

```go
func NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
	decision domain.DecisionAutorizacion,
	verificadaEn time.Time,
) (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error)

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) Datos() (
	DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	error,
)

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) Format(estado fmt.State, _ rune)

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GoString() string

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GobDecode([]byte) error

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GobEncode() ([]byte, error)

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) LogValue() slog.Value

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalBinary() ([]byte, error)

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalCBOR() ([]byte, error)

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalJSON() ([]byte, error)

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalText() ([]byte, error)

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalYAML() (any, error)

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) String() string

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalBinary([]byte) error

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalCBOR([]byte) error

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalJSON([]byte) error

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalText([]byte) error

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalYAML(func(any) error) error

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) ValidarEn(instante time.Time) error

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) ValidarMotivo(
	motivo domain.ReferenciaEntradaCatalogo,
) error

type ExpectativaAutorizacionEjecucionDocumentalV4 struct {
	// DecisionEsperada debe proceder del registro/resolucion confiable del
	// servidor, no de la peticion. Su huella completa se coteja con la evidencia
	// para comprometer tambien asignacion, rol, controles, catalogo, politicas y
	// garantia; no solo los campos funcionales repetidos debajo.
	DecisionEsperada                domain.DecisionAutorizacion
	PrincipalID                     string
	PerfilActivoRef                 string
	AutenticacionRef                string
	SesionRef                       string
	ControlSesionRef                string
	ControlSesionRevision           uint64
	ControlSesionHuellaSHA256       string
	ContextoActorRef                string
	ContextoActorVersion            uint64
	ContextoActorHuellaSHA256       string
	Recurso                         domain.RecursoAutorizable
	Finalidad                       string
	CorrelacionRef                  string
	EfectoRef                       string
	HuellaPlanSHA256                string
	CamposPermitidosEsperados       []string
	ObligacionesEsperadas           []string
	CumplimientosObligacionesPorRef map[string]string
}
```

ExpectativaAutorizacionEjecucionDocumentalV4 contiene los valores resueltos
por el servidor que deben coincidir exactamente con la decision. No concede
autoridad y tampoco acredita la procedencia criptografica del PDP.

Recurso debe contener al menos un ambito explicito y exactamente los dos
atributos de efecto y plan definidos arriba. Los valores sensibles de ambito
deben llegar tokenizados o protegidos con HMAC antes de construirlo.

```go
type FirmaCrudaEvidenciaDocumentalV4 struct {
	// Has unexported fields.
}

func NuevaFirmaCrudaEvidenciaDocumentalV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (FirmaCrudaEvidenciaDocumentalV4, error)

func (p FirmaCrudaEvidenciaDocumentalV4) Format(estado fmt.State, _ rune)

func (p FirmaCrudaEvidenciaDocumentalV4) GoString() string

func (p FirmaCrudaEvidenciaDocumentalV4) LogValue() slog.Value

func (FirmaCrudaEvidenciaDocumentalV4) MarshalBinary() ([]byte, error)

func (FirmaCrudaEvidenciaDocumentalV4) MarshalJSON() ([]byte, error)

func (FirmaCrudaEvidenciaDocumentalV4) MarshalText() ([]byte, error)

func (p FirmaCrudaEvidenciaDocumentalV4) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
)

func (FirmaCrudaEvidenciaDocumentalV4) String() string

func (*FirmaCrudaEvidenciaDocumentalV4) UnmarshalBinary([]byte) error

func (*FirmaCrudaEvidenciaDocumentalV4) UnmarshalJSON([]byte) error

func (*FirmaCrudaEvidenciaDocumentalV4) UnmarshalText([]byte) error

func (p FirmaCrudaEvidenciaDocumentalV4) ValidarSintaxis() error

type FirmanteAtestacionesAutorizacionV1 interface {
	FirmarAtestacionAutorizacionV1(
		context.Context,
		SolicitudFirmaAtestacionAutorizacionV1,
	) (ResultadoFirmaAtestacionAutorizacionV1, error)
}
```

FirmanteAtestacionesAutorizacionV1 es un puerto de salida. La implementacion
productiva debe usar identidad exclusiva del PDP y una clave no exportable.

```go
type FirmanteAtestacionesAutorizacionV2 interface {
	FirmarAtestacionAutorizacionV2(
		context.Context,
		SolicitudFirmaAtestacionAutorizacionV2,
	) (ResultadoFirmaAtestacionAutorizacionV2, error)
}
```

FirmanteAtestacionesAutorizacionV2 es un puerto deliberadamente distinto de
V1. La implementacion productiva debe usar la identidad exclusiva del PDP y
una clave no exportable aprobada para el perfil VEC-AD-2.

```go
type FirmanteEvidenciasRenderizadoDocumentalV3 interface {
	FirmarEvidenciaRenderizadoDocumentalV3(
		context.Context,
		SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	) (SelloEvidenciaDocumentalV3Nominal, error)
}

type FuenteAutorizacion interface {
	ObtenerInstantaneaAutorizacion(context.Context, string, string) (domain.InstantaneaAutorizacion, error)
}
```

FuenteAutorizacion aporta una unica instantanea coherente de todos los
datos que pueden cambiar el resultado. El perfil se resuelve conjuntamente
con el principal para impedir usar, o siquiera descubrir, el perfil de otra
persona.

```go
type FuenteContextoActor interface {
	BuscarInstantaneasContextoActor(
		context.Context,
		domain.SolicitudContextoActor,
	) ([]domain.InstantaneaContextoActor, error)
}
```

FuenteContextoActor es un puerto heredado para pruebas y migracion. No es
una frontera productiva porque separa la lectura del registro durable.

Devuelve todas las instantaneas que coincidan exactamente con cuenta y
perfil. No debe usar LIMIT 1, precedencia ni perfil por defecto: el servicio
de aplicacion es quien exige una coincidencia unica.

La implementacion devuelve copias defensivas y nunca consulta por DNI,
nombre, correo ni otro dato personal. Cuenta y referencias son
identificadores opacos.

```go
type FuenteEvidenciaEmisionDocumento interface {
	ObtenerEvidenciaEmisionDocumento(context.Context, SolicitudObtenerEvidenciaEmisionDocumento) (domain.EvidenciaEmisionDocumento, error)
}
```

FuenteEvidenciaEmisionDocumento consulta exclusivamente fuentes internas
confiables (firma, validacion, registro y almacenamiento). Nunca acepta
huellas o referencias declaradas por el cliente HTTP.

```go
type GeneradorIDCargaDocumental interface {
	NuevoIDCargaDocumental() (string, error)
}

type GeneradorIDCodigoCotejo interface {
	NuevoIDCodigoCotejo() (string, error)
}

type GeneradorIDDocumento interface {
	NuevoIDDocumento() (string, error)
}

type GeneradorIDInstanciaFlujo interface {
	NuevoIDInstanciaFlujo() (string, error)
}

type GeneradorIDOrdenCobro interface {
	NuevoIDOrdenCobro() (string, error)
	NuevoIDDevolucionCobro() (string, error)
}

type GeneradorOperacionContextoActorV1 interface {
	NuevaReferenciaOperacionContextoActorV1(context.Context) (string, error)
}
```

GeneradorOperacionContextoActorV1 crea una referencia nueva antes de
entrar en la operacion durable. Cada token debe proceder de un CSPRNG,
aportar como minimo 144 bits de entropia y pertenecer al espacio oca_. La
misma invocacion logica conserva el token al reconciliar un COMMIT ambiguo;
una invocacion nueva nunca reutiliza el anterior.

```go
type GeneradorReferenciaBorradorDocumental interface {
	NuevaReferenciaBorradorDocumental(context.Context) (string, error)
}
```

GeneradorReferenciaBorradorDocumental crea una referencia opaca antes del
renderizado. La implementacion productiva debe garantizar unicidad en el
repositorio durable; el valor no puede contener datos personales.

```go
type GeneradorReferenciaDecisionAutorizacion interface {
	NuevaReferenciaDecisionAutorizacion() (string, error)
}
```

GeneradorReferenciaDecisionAutorizacion evita que el caso de uso dependa de
una biblioteca, formato de UUID o proveedor concreto.

```go
type GeneradorReferenciasAutorizacionV2 interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
	NuevaClaveMotivoAutorizacionV2(context.Context) (string, error)
}
```

GeneradorReferenciasAutorizacionV2 crea identificadores opacos para
dos finalidades distintas. La implementacion productiva debe usar
un CSPRNG con al menos 128 bits y nunca derivarlos de identidad,
expediente o texto libre. La correlacion en texto es solo la salida
tecnica del adaptador: la frontera debe entregarla inmediatamente a
domain.GenerarReferenciaCorrelacionAutorizacionV2 y transportar desde
entonces exclusivamente la capacidad nominal resultante.

```go
type GeneradorReferenciasFuentesAutoridad interface {
	NuevaReferenciaSolicitud(context.Context) (ReferenciaSolicitudFuenteAutoridad, error)
	NuevaReferenciaOperacion(context.Context) (ReferenciaOperacionFuenteAutoridad, error)
}
```

GeneradorReferenciasFuentesAutoridad produce namespaces distintos. La forma
no acredita entropia: el adaptador debe obtener al menos 128 bits aleatorios
con un CSPRNG y codificarlos, por ejemplo, como 22 caracteres base64url sin
relleno o 32 hexadecimales. La barrera durable reserva ademas la unicidad y
puede devolver ErrColisionReferenciaFuenteAutoridad.

```go
type GeneradorRetosEjecucionDocumental interface {
	NuevoRetoEjecucionDocumental(context.Context) ([32]byte, error)
}

type GeneradorValorCodigoCotejo interface {
	GenerarValorCodigoCotejo(context.Context) (ValorCodigoCotejoGenerado, error)
}
```

GeneradorValorCodigoCotejo debe usar un CSPRNG y al menos 128 bits de
entropia. El caso de uso vuelve a validar alfabeto, longitud y metadatos.

```go
type GestorCargaDirecta interface {
	PrepararCargaDirecta(context.Context, SolicitudPrepararCargaDirecta) (InstruccionesCargaDirecta, error)
	ConfirmarCargaDirecta(context.Context, SolicitudConfirmarCargaDirecta) (ResultadoOperacionObjeto, error)
	AbandonarCargaDirecta(context.Context, ContextoOperacionAlmacen, string) error
}
```

GestorCargaDirecta es una capacidad opcional. ConfirmarCargaDirecta debe
tratar IntencionConfirmacionRef como clave idempotente y HuellaIntencionHMAC
como identidad exacta de la peticion: una repeticion con la misma pareja
devuelve el mismo efecto; la misma referencia con otra huella falla cerrada.
Tambien debe comprobar el limite temporal en el punto del efecto. Este
contrato no aporta por si solo atomicidad distribuida.

```go
type HuellaPlanMaterialAlmacenV2 struct {
	// Has unexported fields.
}
```

HuellaPlanMaterialAlmacenV2 es una capacidad opaca creada exclusivamente
tras cotejar el plan publicado con todos los hechos estables del recibo.

```go
func (h HuellaPlanMaterialAlmacenV2) Bytes() ([]byte, error)

func (HuellaPlanMaterialAlmacenV2) Format(e fmt.State, _ rune)

func (HuellaPlanMaterialAlmacenV2) GoString() string

func (h HuellaPlanMaterialAlmacenV2) Hexadecimal() (string, error)

func (HuellaPlanMaterialAlmacenV2) LogValue() slog.Value

func (HuellaPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error)

func (HuellaPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error)

func (HuellaPlanMaterialAlmacenV2) MarshalText() ([]byte, error)

func (HuellaPlanMaterialAlmacenV2) String() string

func (*HuellaPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error

func (*HuellaPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error

func (*HuellaPlanMaterialAlmacenV2) UnmarshalText([]byte) error

type HuellasRecibosEjecucionDocumentalV3 struct {
	ReciboRenderRef               string
	HuellaSobreRenderSHA256       string
	HuellaReciboRenderSHA256      string
	ReciboEstructuralRef          string
	HuellaSobreEstructuralSHA256  string
	HuellaReciboEstructuralSHA256 string
	ReciboSemanticoRef            string
	HuellaSobreSemanticoSHA256    string
	HuellaReciboSemanticoSHA256   string
}
```

HuellasRecibosEjecucionDocumentalV3 es la proyeccion persistible de tres
sobres COSE nominalmente correlacionados. No acredita su comprobacion ni
concede autoridad: el registro debe releer y comprobar los recibos mediante
dependencias privadas antes del CAS de confirmacion.

```go
func (h HuellasRecibosEjecucionDocumentalV3) Validar() error

type IdentidadEjecucionComponenteDocumental struct {
	// Has unexported fields.
}
```

IdentidadEjecucionComponenteDocumental identifica la carga de trabajo y la
instancia concreta que firmo el recibo. La medicion debe coincidir despues
con el artefacto homologado del descriptor.

```go
func NuevaIdentidadEjecucionComponenteDocumental(
	cargaTrabajoRef, instanciaProcesoRef, dominioAislamientoRef, claveFirmaRef string,
	huellaClaveFirmaSHA256, huellaMedicionSHA256 string,
) (IdentidadEjecucionComponenteDocumental, error)

func (i IdentidadEjecucionComponenteDocumental) CargaTrabajoRef() string

func (i IdentidadEjecucionComponenteDocumental) ClaveFirmaRef() string

func (i IdentidadEjecucionComponenteDocumental) DominioAislamientoRef() string

func (i IdentidadEjecucionComponenteDocumental) Format(estado fmt.State, _ rune)

func (i IdentidadEjecucionComponenteDocumental) GoString() string

func (i IdentidadEjecucionComponenteDocumental) HuellaClaveFirmaSHA256() string

func (i IdentidadEjecucionComponenteDocumental) HuellaMedicionSHA256() string

func (i IdentidadEjecucionComponenteDocumental) InstanciaProcesoRef() string

func (i IdentidadEjecucionComponenteDocumental) LogValue() slog.Value

func (IdentidadEjecucionComponenteDocumental) MarshalBinary() ([]byte, error)

func (IdentidadEjecucionComponenteDocumental) MarshalJSON() ([]byte, error)

func (IdentidadEjecucionComponenteDocumental) MarshalText() ([]byte, error)

func (IdentidadEjecucionComponenteDocumental) String() string

func (*IdentidadEjecucionComponenteDocumental) UnmarshalBinary([]byte) error

func (*IdentidadEjecucionComponenteDocumental) UnmarshalJSON([]byte) error

func (*IdentidadEjecucionComponenteDocumental) UnmarshalText([]byte) error

func (i IdentidadEjecucionComponenteDocumental) Validar() error

type IdentityPort interface {
	Identify(context.Context, InteropRequest) (InteropResult, error)
}

type InicioOperacionCobro = pagoscanonicos.InicioOperacionCobro

type InstantaneaEjecucionDocumentalV3Nominal struct {
	ReservaRef                 string
	IndiceIdempotenciaHMAC     string
	HuellaSolicitudHMAC        string
	Manifiesto                 ManifiestoEjecucionDocumentalV3
	Estado                     EstadoEjecucionDocumentalV3
	SecuenciaCercado           uint64
	HuellaVinculoSHA256        string
	ConsumoDecision            ConsumoDecisionEjecucionDocumentalV3
	OrdenConsumoDurableV4Ref   string
	Resultado                  ResultadoEfectoRenderizadoDocumentalV3Crudo
	IncidenteRef               string
	EvidenciaRef               string
	HuellaEvidenciaSHA256      string
	EstadoOrigenAbandono       EstadoEjecucionDocumentalV3
	MotivoAbandonoRef          string
	ReconciliacionRef          string
	HuellaReconciliacionSHA256 string
	ActualizadaEn              time.Time
}

func (i InstantaneaEjecucionDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (i InstantaneaEjecucionDocumentalV3Nominal) GoString() string

func (i InstantaneaEjecucionDocumentalV3Nominal) LogValue() slog.Value

func (InstantaneaEjecucionDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (InstantaneaEjecucionDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (InstantaneaEjecucionDocumentalV3Nominal) MarshalText() ([]byte, error)

func (InstantaneaEjecucionDocumentalV3Nominal) String() string

func (*InstantaneaEjecucionDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*InstantaneaEjecucionDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*InstantaneaEjecucionDocumentalV3Nominal) UnmarshalText([]byte) error

func (i InstantaneaEjecucionDocumentalV3Nominal) Validar() error

type InstantaneaObjetoMaterialV2 struct {
	// Has unexported fields.
}
```

InstantaneaObjetoMaterialV2 contiene solo hechos originales del objeto.
No incorpora la evidencia nueva de un reintento ni datos del intento.

```go
func (i InstantaneaObjetoMaterialV2) BytesCanonicos() ([]byte, error)

func (InstantaneaObjetoMaterialV2) Format(e fmt.State, _ rune)

func (InstantaneaObjetoMaterialV2) GoString() string

func (i InstantaneaObjetoMaterialV2) HuellaSHA256() ([sha256.Size]byte, error)

func (InstantaneaObjetoMaterialV2) LogValue() slog.Value

func (InstantaneaObjetoMaterialV2) MarshalBinary() ([]byte, error)

func (InstantaneaObjetoMaterialV2) MarshalJSON() ([]byte, error)

func (InstantaneaObjetoMaterialV2) MarshalText() ([]byte, error)

func (InstantaneaObjetoMaterialV2) String() string

func (*InstantaneaObjetoMaterialV2) UnmarshalBinary([]byte) error

func (*InstantaneaObjetoMaterialV2) UnmarshalJSON([]byte) error

func (*InstantaneaObjetoMaterialV2) UnmarshalText([]byte) error

func (i InstantaneaObjetoMaterialV2) Validar() error

type InstruccionesCargaDirecta struct {
	// Has unexported fields.
}
```

InstruccionesCargaDirecta contiene un secreto de corta duracion. No debe
persistirse, auditarse ni incluirse en trazas, metricas o mensajes.

```go
func NuevasInstruccionesCargaDirecta(
	conectorID, sesionRef string,
	metodo MetodoCargaDirecta,
	destino string,
	cabeceras []CabeceraCargaDirecta,
	emitidaEn, expiraEn time.Time,
	tamanoMaximo int64,
) (InstruccionesCargaDirecta, error)
```

NuevasInstruccionesCargaDirecta solo acepta una concesion corta, limitada a
un destino HTTPS y a un tamano. El conector no puede devolver mapas mutables
ni una credencial utilizable para listar, leer o elegir otro objeto.

```go
func NuevasInstruccionesCargaDirectaParaSolicitud(
	solicitud SolicitudPrepararCargaDirecta,
	conectorID, sesionRef string,
	metodo MetodoCargaDirecta,
	destino string,
	cabeceras []CabeceraCargaDirecta,
	emitidaEn time.Time,
) (InstruccionesCargaDirecta, error)
```

NuevasInstruccionesCargaDirectaParaSolicitud es el constructor que deben
usar los conectores productivos. Ademas de acotar la concesion, deja dentro
del valor opaco una huella de todos los datos de la solicitud para que el
nucleo pueda detectar respuestas cruzadas, repetidas o fabricadas por un
conector defectuoso.

```go
func (i InstruccionesCargaDirecta) Abandonar(
	ctx context.Context,
	gestor GestorCargaDirecta,
	contexto ContextoOperacionAlmacen,
) error
```

Abandonar revoca la concesion sin revelar la referencia de sesion al caso
de uso. Se emplea como compensacion si no puede confirmarse la reserva
transaccional del nucleo.

```go
func (i InstruccionesCargaDirecta) EmitirReciboConfirmacion(
	ctx context.Context,
	solicitud SolicitudPrepararCargaDirecta,
	capacidades CapacidadesAlmacenObjetos,
	emisor EmisorReciboCargaDirecta,
) (ReciboCargaDirecta, error)

func (i InstruccionesCargaDirecta) Format(estado fmt.State, _ rune)

func (InstruccionesCargaDirecta) GoString() string

func (InstruccionesCargaDirecta) MarshalJSON() ([]byte, error)

func (InstruccionesCargaDirecta) MarshalText() ([]byte, error)

func (i InstruccionesCargaDirecta) RevelarParaEntrega() (
	sesionRef string,
	metodo MetodoCargaDirecta,
	destino string,
	cabeceras []CabeceraCargaDirecta,
	expiraEn time.Time,
	tamanoMaximo int64,
	err error,
)
```

RevelarParaEntrega es el unico punto que expone la concesion. El adaptador
HTTP debe copiarla inmediatamente a una respuesta no almacenable y borrar
sus referencias; nunca debe pasar el valor InstruccionesCargaDirecta a un
serializador o registrador generico.

```go
func (i InstruccionesCargaDirecta) SellarVinculoSesion(
	ctx context.Context,
	sellador SelladorVinculoSesionCarga,
) (string, error)
```

SellarVinculoSesion permite persistir solo un HMAC de la referencia de
sesion sin revelarla al caso de uso. El sellador usa una clave exclusiva y
devuelve un identificador versionado, nunca la referencia original.

```go
func (InstruccionesCargaDirecta) String() string

func (i InstruccionesCargaDirecta) Validar() error

func (i InstruccionesCargaDirecta) ValidarContra(capacidades CapacidadesAlmacenObjetos) error
```

ValidarContra impide aceptar un destino que no figure exactamente en el
perfil de despliegue publicado para el mismo conector.

```go
func (i InstruccionesCargaDirecta) ValidarPara(
	solicitud SolicitudPrepararCargaDirecta,
	capacidades CapacidadesAlmacenObjetos,
) error

func (i InstruccionesCargaDirecta) VigenteEn(instante time.Time) bool

type InteropRequest struct {
	Operation string            `json:"operation"`
	Subject   string            `json:"subject,omitempty"`
	Payload   map[string]string `json:"payload,omitempty"`
}

type InteropResult struct {
	Operation string            `json:"operation"`
	Reference string            `json:"reference,omitempty"`
	Status    string            `json:"status"`
	Payload   map[string]string `json:"payload,omitempty"`
}

type InvoicePort interface {
	SubmitInvoice(context.Context, InteropRequest) (InteropResult, error)
}

type LecturaObjetoAlmacen struct {
	Objeto    ObjetoAlmacenado
	Evidencia EvidenciaOperacionAlmacen
	Contenido io.ReadCloser
}

func (l LecturaObjetoAlmacen) ValidarContra(solicitud SolicitudAbrirObjeto) error

type ManifiestoEjecucionDocumentalV3 struct {
	// Has unexported fields.
}
```

ManifiestoEjecucionDocumentalV3 es inmutable y opaco. Su huella SHA-256
identifica el plan; no sustituye el sello criptografico de la evidencia.

```go
func NuevoManifiestoEjecucionDocumentalV3(
	consulta ConsultaFormatoDocumental,
	descriptor DescriptorPerfilDocumental,
	situacion domain.SituacionOperativaPerfilDocumental,
	render, verificador, semantico DescriptorComponenteDocumentalAtestado,
	borradorRef, efectoRef, huellaEntradaHMAC string,
	limiteEfectivoBytes uint64,
) (ManifiestoEjecucionDocumentalV3, error)

func (m ManifiestoEjecucionDocumentalV3) Datos() (DatosManifiestoEjecucionDocumentalV3, error)

func (m ManifiestoEjecucionDocumentalV3) Format(estado fmt.State, _ rune)

func (m ManifiestoEjecucionDocumentalV3) GoString() string

func (m ManifiestoEjecucionDocumentalV3) HuellaSHA256() (string, error)

func (m ManifiestoEjecucionDocumentalV3) LogValue() slog.Value

func (ManifiestoEjecucionDocumentalV3) MarshalBinary() ([]byte, error)

func (ManifiestoEjecucionDocumentalV3) MarshalJSON() ([]byte, error)

func (ManifiestoEjecucionDocumentalV3) MarshalText() ([]byte, error)

func (ManifiestoEjecucionDocumentalV3) String() string

func (*ManifiestoEjecucionDocumentalV3) UnmarshalBinary([]byte) error

func (*ManifiestoEjecucionDocumentalV3) UnmarshalJSON([]byte) error

func (*ManifiestoEjecucionDocumentalV3) UnmarshalText([]byte) error

func (m ManifiestoEjecucionDocumentalV3) Validar() error

type ManifiestoGeneracionDocumental struct {
	// Has unexported fields.
}
```

ManifiestoGeneracionDocumental es un plan opaco e inmutable. Su valor cero
y cualquier intento de reconstruirlo por serializacion son invalidos.
Una generacion simple es exactamente el mismo contrato con un unico paso.

```go
func NuevoManifiestoGeneracionDocumental(
	plantilla domain.PlantillaDocumento,
	representaciones []DeclaracionRepresentacionGeneracionDocumental,
) (ManifiestoGeneracionDocumental, error)

func (m ManifiestoGeneracionDocumental) Format(estado fmt.State, _ rune)

func (m ManifiestoGeneracionDocumental) GoString() string

func (m ManifiestoGeneracionDocumental) LogValue() slog.Value

func (ManifiestoGeneracionDocumental) MarshalJSON() ([]byte, error)

func (ManifiestoGeneracionDocumental) MarshalText() ([]byte, error)

func (m ManifiestoGeneracionDocumental) Proyeccion() (ProyeccionManifiestoGeneracionDocumental, error)

func (ManifiestoGeneracionDocumental) String() string

func (*ManifiestoGeneracionDocumental) UnmarshalJSON([]byte) error

func (*ManifiestoGeneracionDocumental) UnmarshalText([]byte) error

type MarcadorMetadatoInstitucionalDocumental interface {
	PerfilDocumental() domain.ReferenciaPerfilDocumental
	DigestPerfilSHA256() string
	ConectorDocumental() domain.ReferenciaConectorDocumental
	IncorporarMetadatoInstitucional(
		context.Context,
		SolicitudIncorporarMetadatoInstitucional,
	) (ResultadoMetadatoInstitucional, error)
}
```

MarcadorMetadatoInstitucionalDocumental debe usar metadatos estandar del
perfil. No se le permite autocertificar la equivalencia semantica de su
propia salida: esa barrera pertenece a un puerto independiente.

```go
type MaterialCrudoVerificacionOrdenDespachoDocumentalV3 struct {
	// Has unexported fields.
}
```

MaterialCrudoVerificacionOrdenDespachoDocumentalV3 agrupa el mensaje
canonico, tres pruebas HMAC y todos los vinculos durables que debe cotejar
un adaptador. Es material nominal autocreable: verificarlo no concede
autoridad fuera del servicio de aplicacion que ejecuta despues el consumo
CAS.

```go
func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) Format(estado fmt.State, _ rune)

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) GoString() string

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) HuellaSHA256() (string, error)

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) LogValue() slog.Value

func (MaterialCrudoVerificacionOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error)

func (MaterialCrudoVerificacionOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error)

func (MaterialCrudoVerificacionOrdenDespachoDocumentalV3) MarshalText() ([]byte, error)

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) MensajeCanonico() ([]byte, error)

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) Pruebas() (
	cercado, inicio, reclamacion PruebaCrudaAtestacionDespachoDocumentalV3,
	err error,
)

func (MaterialCrudoVerificacionOrdenDespachoDocumentalV3) String() string

func (*MaterialCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error

func (*MaterialCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error

func (*MaterialCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalText([]byte) error

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) Validar() error

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) Vinculos() (
	VinculosCrudosVerificacionOrdenDespachoDocumentalV3,
	error,
)

type MetadatosAuditoriaCobro = pagoscanonicos.MetadatosAuditoriaCobro

type MetadatosComprobacionEvidenciaDocumentalV3Nominal struct {
	// Has unexported fields.
}

func NuevosMetadatosComprobacionEvidenciaDocumentalV3Nominal(
	solicitud SolicitudVerificacionEvidenciaDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (MetadatosComprobacionEvidenciaDocumentalV3Nominal, error)
```

NuevosMetadatosComprobacionEvidenciaDocumentalV3Nominal restaura la salida
nominal del conector. No sustituye la comprobacion criptografica privada del
servicio de aplicacion ni autoriza una confirmacion.

```go
func (r MetadatosComprobacionEvidenciaDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (r MetadatosComprobacionEvidenciaDocumentalV3Nominal) GoString() string

func (r MetadatosComprobacionEvidenciaDocumentalV3Nominal) LogValue() slog.Value

func (MetadatosComprobacionEvidenciaDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (MetadatosComprobacionEvidenciaDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (MetadatosComprobacionEvidenciaDocumentalV3Nominal) MarshalText() ([]byte, error)

func (MetadatosComprobacionEvidenciaDocumentalV3Nominal) String() string

func (*MetadatosComprobacionEvidenciaDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*MetadatosComprobacionEvidenciaDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*MetadatosComprobacionEvidenciaDocumentalV3Nominal) UnmarshalText([]byte) error

func (r MetadatosComprobacionEvidenciaDocumentalV3Nominal) ValidarPara(
	solicitud SolicitudVerificacionEvidenciaDocumentalV3,
) error

type MetadatosComprobacionReconciliacionDocumentalV3Nominal struct {
	// Has unexported fields.
}

func NuevosMetadatosComprobacionReconciliacionDocumentalV3Nominal(
	solicitud SolicitudVerificacionReconciliacionDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (MetadatosComprobacionReconciliacionDocumentalV3Nominal, error)
```

NuevosMetadatosComprobacionReconciliacionDocumentalV3Nominal restaura una
salida nominal del conector. Nunca sustituye la verificacion COSE privada.

```go
func (r MetadatosComprobacionReconciliacionDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (r MetadatosComprobacionReconciliacionDocumentalV3Nominal) GoString() string

func (r MetadatosComprobacionReconciliacionDocumentalV3Nominal) LogValue() slog.Value

func (MetadatosComprobacionReconciliacionDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (MetadatosComprobacionReconciliacionDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (MetadatosComprobacionReconciliacionDocumentalV3Nominal) MarshalText() ([]byte, error)

func (MetadatosComprobacionReconciliacionDocumentalV3Nominal) String() string

func (*MetadatosComprobacionReconciliacionDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*MetadatosComprobacionReconciliacionDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*MetadatosComprobacionReconciliacionDocumentalV3Nominal) UnmarshalText([]byte) error

func (r MetadatosComprobacionReconciliacionDocumentalV3Nominal) ValidarPara(
	solicitud SolicitudVerificacionReconciliacionDocumentalV3,
) error

type MetadatosComprobacionTokenCercadoDocumentalV3Nominal struct {
	// Has unexported fields.
}

func NuevosMetadatosComprobacionTokenCercadoDocumentalV3Nominal(
	solicitud SolicitudVerificacionTokenCercadoDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (MetadatosComprobacionTokenCercadoDocumentalV3Nominal, error)
```

NuevosMetadatosComprobacionTokenCercadoDocumentalV3Nominal solo correlaciona
una solicitud con referencia e instante. Es una fabrica publica autocreable
y, por tanto, estos metadatos son nominales y NO acreditan que se verificara
el MAC. Nunca habilitan el inicio ni otro efecto por si solos.

```go
func (m MetadatosComprobacionTokenCercadoDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (m MetadatosComprobacionTokenCercadoDocumentalV3Nominal) GoString() string

func (m MetadatosComprobacionTokenCercadoDocumentalV3Nominal) LogValue() slog.Value

func (MetadatosComprobacionTokenCercadoDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (MetadatosComprobacionTokenCercadoDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (MetadatosComprobacionTokenCercadoDocumentalV3Nominal) MarshalText() ([]byte, error)

func (MetadatosComprobacionTokenCercadoDocumentalV3Nominal) String() string

func (*MetadatosComprobacionTokenCercadoDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*MetadatosComprobacionTokenCercadoDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*MetadatosComprobacionTokenCercadoDocumentalV3Nominal) UnmarshalText([]byte) error

func (r MetadatosComprobacionTokenCercadoDocumentalV3Nominal) ValidarPara(
	solicitud SolicitudVerificacionTokenCercadoDocumentalV3,
) error

type MetadatosFuenteCatalogos struct {
	Revision      string
	ActualizadaEn time.Time
	Demostracion  bool
	Aviso         string
}
```

MetadatosFuenteCatalogos describe el adaptador de lectura sin revelar
la procedencia interna de cada entrada ni los actores del gobierno del
catalogo. En produccion estos datos pueden proceder del repositorio
publicado; el adaptador de fichero los toma del envoltorio DEMO versionado.
La huella de procedencia declarada no equivale a una firma verificable del
paquete.

```go
type MetodoCargaDirecta = almacencanonico.MetodoCargaDirecta

type MetodoHandoffCobro = pagoscanonicos.MetodoHandoffCobro

type ModuleRegistryStore interface {
	SaveModule(context.Context, domain.ModuleManifest) error
	ListModules(context.Context) ([]domain.ModuleManifest, error)
}

type MutacionOrdenCobro struct {
	// Has unexported fields.
}
```

MutacionOrdenCobro es una unidad opaca para persistir agregado, auditoria y
outbox en una sola transaccion. El constructor copia el agregado y deriva el
evento; el llamador no puede compartir ni sustituir memoria interna.

```go
func NuevaMutacionOrdenCobro(orden domain.OrdenCobro) (MutacionOrdenCobro, error)

func (m MutacionOrdenCobro) Datos() (DatosMutacionOrdenCobro, error)

func (m MutacionOrdenCobro) Format(estado fmt.State, _ rune)

func (m MutacionOrdenCobro) GoString() string

func (MutacionOrdenCobro) MarshalJSON() ([]byte, error)

func (MutacionOrdenCobro) MarshalText() ([]byte, error)

func (MutacionOrdenCobro) String() string

func (*MutacionOrdenCobro) UnmarshalJSON([]byte) error

func (m MutacionOrdenCobro) Validar() error

type NotificacionCobro = pagoscanonicos.NotificacionCobro

type NotificationPort interface {
	SendNotification(context.Context, InteropRequest) (InteropResult, error)
}

type ObjetoAlmacenado = almacencanonico.ObjetoAlmacenado

type OperacionComponenteDocumental string
```

OperacionComponenteDocumental es vocabulario cerrado del protocolo entre el
nucleo y el despachador. Cada operacion tiene un unico rol admisible.

```go
const (
	OperacionRenderizadoDocumental           OperacionComponenteDocumental = "renderizado"
	OperacionValidacionEstructuralDocumental OperacionComponenteDocumental = "validacion_estructural"
	OperacionVerificacionSemanticaDocumental OperacionComponenteDocumental = "verificacion_semantica"
)
func (o OperacionComponenteDocumental) Valida() bool

type OperacionFuenteAutoridad struct {
	// Has unexported fields.
}
```

OperacionFuenteAutoridad evita que un callback reconstruya actor, motivo,
accion o revision a partir de parametros externos.

```go
func NuevaOperacionPendienteFuenteAutoridad(
	operacionRef string,
	solicitud domain.SolicitudTransicionFuenteAutoridadV1,
	esperado ReferenciaEstadoFuenteAutoridad,
) (OperacionFuenteAutoridad, error)

func RehidratarOperacionFuenteAutoridad(
	datos DatosOperacionFuenteAutoridad,
) (OperacionFuenteAutoridad, error)

func (o OperacionFuenteAutoridad) Datos() (DatosOperacionFuenteAutoridad, error)

func (o OperacionFuenteAutoridad) Format(estado fmt.State, _ rune)

func (o OperacionFuenteAutoridad) GoString() string

func (*OperacionFuenteAutoridad) GobDecode([]byte) error

func (OperacionFuenteAutoridad) GobEncode() ([]byte, error)

func (o OperacionFuenteAutoridad) LogValue() slog.Value

func (OperacionFuenteAutoridad) MarshalBinary() ([]byte, error)

func (OperacionFuenteAutoridad) MarshalJSON() ([]byte, error)

func (OperacionFuenteAutoridad) MarshalText() ([]byte, error)

func (OperacionFuenteAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (OperacionFuenteAutoridad) String() string

func (o OperacionFuenteAutoridad) Terminal() bool

func (*OperacionFuenteAutoridad) UnmarshalBinary([]byte) error

func (*OperacionFuenteAutoridad) UnmarshalJSON([]byte) error

func (*OperacionFuenteAutoridad) UnmarshalText([]byte) error

func (*OperacionFuenteAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

type OrdenConsumoReciboCargaDirecta = almacencanonico.OrdenConsumoReciboCargaDirecta
```

OrdenConsumoReciboCargaDirecta no contiene secreto ni una hora de consumo
propuesta por el proceso. RegistradoEn procede del recibo atestado tras el
alta y debe coincidir con el registro; el repositorio decide ConsumidoEn con
su reloj transaccional autoritativo y lo devuelve despues de persistirlo.

```go
type OrdenDespachoDocumentalV3ConsumidaNominal struct {
	// Has unexported fields.
}
```

OrdenDespachoDocumentalV3ConsumidaNominal es el comando restaurable que el
servicio de aplicacion crea despues de verificar por KMS y consumir por CAS.
Su fabrica publica solo coteja forma y correlacion: NO acredita que esas dos
operaciones ocurrieran y nunca concede autoridad por si sola. HTTP, CLI,
MCP y handlers no deben recibirla ni poseer el despachador que la consume.

```go
func NuevaOrdenDespachoDocumentalV3ConsumidaNominal(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	resultado ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
	estado EstadoCrudoOrdenDespachoDocumentalV3,
) (OrdenDespachoDocumentalV3ConsumidaNominal, error)

func (o OrdenDespachoDocumentalV3ConsumidaNominal) DatosOrden() (
	DatosOrdenDespachoDocumentalV3Nominal,
	error,
)

func (o OrdenDespachoDocumentalV3ConsumidaNominal) Format(estado fmt.State, _ rune)

func (o OrdenDespachoDocumentalV3ConsumidaNominal) GoString() string

func (o OrdenDespachoDocumentalV3ConsumidaNominal) LogValue() slog.Value

func (OrdenDespachoDocumentalV3ConsumidaNominal) MarshalBinary() ([]byte, error)

func (OrdenDespachoDocumentalV3ConsumidaNominal) MarshalJSON() ([]byte, error)

func (OrdenDespachoDocumentalV3ConsumidaNominal) MarshalText() ([]byte, error)

func (OrdenDespachoDocumentalV3ConsumidaNominal) String() string

func (*OrdenDespachoDocumentalV3ConsumidaNominal) UnmarshalBinary([]byte) error

func (*OrdenDespachoDocumentalV3ConsumidaNominal) UnmarshalJSON([]byte) error

func (*OrdenDespachoDocumentalV3ConsumidaNominal) UnmarshalText([]byte) error

func (o OrdenDespachoDocumentalV3ConsumidaNominal) ValidarEn(instante time.Time) error

func (o OrdenDespachoDocumentalV3ConsumidaNominal) VinculoActivacion() (
	VinculoEstableActivacionDocumentalV3,
	error,
)

type OrdenDespachoDocumentalV3Nominal struct {
	// Has unexported fields.
}
```

OrdenDespachoDocumentalV3Nominal es la salida restaurable de la reclamacion
CAS. Aunque contenga atestaciones, su fabrica publica nunca la promueve a
autoridad. Solo sirve como entrada cruda al servicio de aplicacion privado.

```go
func NuevaOrdenDespachoDocumentalV3Nominal(
	recibo ReciboInicioEfectoDocumentalV3Nominal,
	solicitud SolicitudReclamarOrdenDespachoDocumentalV3,
	versionReclamacionCAS uint64,
	auditoriaReclamacionRef string,
	atestacionReclamacion PruebaCrudaAtestacionDespachoDocumentalV3,
) (OrdenDespachoDocumentalV3Nominal, error)

func (o OrdenDespachoDocumentalV3Nominal) Datos() (DatosOrdenDespachoDocumentalV3Nominal, error)

func (o OrdenDespachoDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (o OrdenDespachoDocumentalV3Nominal) GoString() string

func (o OrdenDespachoDocumentalV3Nominal) HuellaSHA256() (string, error)

func (o OrdenDespachoDocumentalV3Nominal) LogValue() slog.Value

func (OrdenDespachoDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (OrdenDespachoDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (OrdenDespachoDocumentalV3Nominal) MarshalText() ([]byte, error)

func (OrdenDespachoDocumentalV3Nominal) String() string

func (*OrdenDespachoDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*OrdenDespachoDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*OrdenDespachoDocumentalV3Nominal) UnmarshalText([]byte) error

func (o OrdenDespachoDocumentalV3Nominal) Validar() error

type OrdenRegistroDecisionAutorizacionSolicitudLigadaV2 struct {
	// Has unexported fields.
}
```

OrdenRegistroDecisionAutorizacionSolicitudLigadaV2 es una orden opaca
y nominal. Evita que un registro V2 reciba solo la decision y pierda la
preimagen catalogada necesaria para la revalidacion durable.

```go
func NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
) (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2, error)

func (o OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) Datos() (
	DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
	error,
)

func (b OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) Format(estado fmt.State, _ rune)

func (b OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) GoString() string

func (*OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) GobDecode([]byte) error

func (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) GobEncode() ([]byte, error)

func (b OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) LogValue() slog.Value

func (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalBinary() ([]byte, error)

func (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalCBOR() ([]byte, error)

func (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalJSON() ([]byte, error)

func (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalText() ([]byte, error)

func (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalXML(*xml.Encoder, xml.StartElement) error

func (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) MarshalYAML() (any, error)

func (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) String() string

func (*OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalBinary([]byte) error

func (*OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalCBOR([]byte) error

func (*OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalJSON([]byte) error

func (*OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalText([]byte) error

func (*OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) UnmarshalYAML(func(any) error) error

type OrgDirectoryPort interface {
	LookupOrgUnit(context.Context, InteropRequest) (InteropResult, error)
}

type OrigenPasarelaCobroPublicado = pagoscanonicos.OrigenPasarelaCobroPublicado

type PaginaHistoriaFuenteAutoridad struct {
	Versiones        []domain.FuenteAutoridadVersionada
	HayMas           bool
	SiguienteVersion uint64
}
```

PaginaHistoriaFuenteAutoridad conserva continuidad explicita. Si HayMas
es cierto, SiguienteVersion es la primera version de la pagina siguiente;
no existe cursor implicito ni selector de «ultima» version.

```go
func (p PaginaHistoriaFuenteAutoridad) ClonarPara(
	consulta ConsultaPaginaHistoriaFuenteAutoridad,
) (PaginaHistoriaFuenteAutoridad, error)

func (p PaginaHistoriaFuenteAutoridad) ValidarPara(
	consulta ConsultaPaginaHistoriaFuenteAutoridad,
) error

type PasarelaCobro interface {
	VerificadorPasarelaCobro
	Capacidades(context.Context) (CapacidadesPasarelaCobro, error)
	CrearOperacion(context.Context, SolicitudOperacionCobro) (InicioOperacionCobro, error)
	ConsultarOperacion(context.Context, ReferenciaOperacionCobro) (ResultadoOperacionCobro, error)
	SolicitarDevolucion(context.Context, SolicitudDevolucionCobro) (ResultadoDevolucionCobro, error)
	ConsultarDevolucion(context.Context, ReferenciaDevolucionCobro) (ResultadoDevolucionCobro, error)
	Conciliar(context.Context, SolicitudConciliacionCobro) (ResultadoConciliacionCobro, error)
}
```

PasarelaCobro es el unico contrato remoto del nucleo de cobros. No conoce
proveedores, protocolos ni redes concretas y no sirve para pagos salientes.

```go
type PasoOperacionAlmacen = almacencanonico.PasoOperacionAlmacen
```

PasoOperacionAlmacen identifica un paso previamente declarado por el nucleo.
Aunque su representacion sea texto, ninguna fabrica acepta valores fuera del
plan cerrado asociado a la accion de negocio.

```go
type PerfilCapacidadesAlmacenMaterialV2 struct {
	// Has unexported fields.
}
```

PerfilCapacidadesAlmacenMaterialV2 es una instantanea atestada de las
capacidades materiales relevantes para escritura. Omite de forma expresa
origenes, endpoint, bucket, rutas y cualquier detalle fisico.

```go
func NuevoPerfilCapacidadesAlmacenMaterialV2(
	ctx context.Context,
	referencia string,
	version uint32,
	capacidades CapacidadesAlmacenObjetos,
	atestador AtestadorMaterialAlmacenV2,
	verificador VerificadorAtestacionMaterialAlmacenV2,
	verificadorPublicacion VerificadorPerfilPublicadoMaterialV2,
) (PerfilCapacidadesAlmacenMaterialV2, error)

func (p PerfilCapacidadesAlmacenMaterialV2) BytesCanonicos() ([]byte, error)

func (PerfilCapacidadesAlmacenMaterialV2) Format(e fmt.State, _ rune)

func (PerfilCapacidadesAlmacenMaterialV2) GoString() string

func (p PerfilCapacidadesAlmacenMaterialV2) HuellaSHA256() ([sha256.Size]byte, error)

func (PerfilCapacidadesAlmacenMaterialV2) LogValue() slog.Value

func (PerfilCapacidadesAlmacenMaterialV2) MarshalBinary() ([]byte, error)

func (PerfilCapacidadesAlmacenMaterialV2) MarshalJSON() ([]byte, error)

func (PerfilCapacidadesAlmacenMaterialV2) MarshalText() ([]byte, error)

func (PerfilCapacidadesAlmacenMaterialV2) String() string

func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalBinary([]byte) error

func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalJSON([]byte) error

func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalText([]byte) error

func (p PerfilCapacidadesAlmacenMaterialV2) Validar() error

func (p PerfilCapacidadesAlmacenMaterialV2) VerificarAtestacion(
	ctx context.Context,
	verificador VerificadorAtestacionMaterialAlmacenV2,
) error

func (p PerfilCapacidadesAlmacenMaterialV2) VerificarPublicacion(
	ctx context.Context,
	verificador VerificadorPerfilPublicadoMaterialV2,
) error
```

VerificarPublicacion reconsulta el catalogo autoritativo. Se ejecuta al
crear el perfil y de nuevo antes de cada recibo para que una revocacion
posterior falle cerrada.

```go
type PerfilSelloEvidenciaDocumentalV3 struct {
	Algoritmo string
	ClaveID   string
	Audiencia string
}

func NuevoPerfilSelloEvidenciaHMACSHA256V3(claveID string) (PerfilSelloEvidenciaDocumentalV3, error)

func (p PerfilSelloEvidenciaDocumentalV3) Format(estado fmt.State, _ rune)

func (p PerfilSelloEvidenciaDocumentalV3) GoString() string

func (p PerfilSelloEvidenciaDocumentalV3) LogValue() slog.Value

func (PerfilSelloEvidenciaDocumentalV3) MarshalBinary() ([]byte, error)

func (PerfilSelloEvidenciaDocumentalV3) MarshalJSON() ([]byte, error)

func (PerfilSelloEvidenciaDocumentalV3) MarshalText() ([]byte, error)

func (PerfilSelloEvidenciaDocumentalV3) String() string

func (*PerfilSelloEvidenciaDocumentalV3) UnmarshalBinary([]byte) error

func (*PerfilSelloEvidenciaDocumentalV3) UnmarshalJSON([]byte) error

func (*PerfilSelloEvidenciaDocumentalV3) UnmarshalText([]byte) error

func (p PerfilSelloEvidenciaDocumentalV3) Validar() error

type PoliticaInstitucionalDocumentalAtestada struct {
	// Has unexported fields.
}
```

PoliticaInstitucionalDocumentalAtestada es una entrada positiva de catalogo.
El usuario no proporciona URI: ConstruirMarca la deriva de la base HTTPS
exacta que el catalogo institucional ha permitido.

```go
func NuevaPoliticaInstitucionalDocumentalAtestada(
	referencia string,
	revision uint64,
	consulta ConsultaPoliticaInstitucionalDocumental,
	endpointPublicoRef, baseURIPublicaPermitida, huellaPoliticaSHA256 string,
) (PoliticaInstitucionalDocumentalAtestada, error)

func (p PoliticaInstitucionalDocumentalAtestada) Coincide(
	consulta ConsultaPoliticaInstitucionalDocumental,
) bool

func (p PoliticaInstitucionalDocumentalAtestada) ConstruirMarca(
	documentoUUID string,
	fecha time.Time,
) (domain.MarcaInstitucionalDocumento, error)

func (p PoliticaInstitucionalDocumentalAtestada) Consulta() ConsultaPoliticaInstitucionalDocumental

func (p PoliticaInstitucionalDocumentalAtestada) DigestDeclaracionSHA256() string

func (p PoliticaInstitucionalDocumentalAtestada) EndpointPublicoRef() string

func (p PoliticaInstitucionalDocumentalAtestada) HuellaPoliticaSHA256() string

func (p PoliticaInstitucionalDocumentalAtestada) Referencia() string

func (p PoliticaInstitucionalDocumentalAtestada) Revision() uint64

func (p PoliticaInstitucionalDocumentalAtestada) Validar() error

type PredecesorReciboCargaDirecta = almacencanonico.PredecesorReciboCargaDirecta
```

PredecesorReciboCargaDirecta conserva el enlace tipado que el repositorio
crea al sustituir el recibo activo de un grupo. No es por si mismo un
evento de auditoria: el adaptador durable debera incorporarlo a su registro
transaccional y al outbox cuando estos existan.

```go
type PreimagenRecursoAutorizacionEjecucionDocumentalV4 struct {
	// Has unexported fields.
}
```

PreimagenRecursoAutorizacionEjecucionDocumentalV4 conserva la representacion
completa que permite recomputar el contexto de recurso firmado por VEC-AD-1.
Es opaca, no es una decision y nunca concede autoridad. Solo nace de una
SolicitudAplicacion ya ligada o de la interpretacion estricta de sus bytes
persistidos, que continua siendo material no confiable hasta cotejarlo con
una firma PDP y una raiz historica autentica.

```go
func InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
	serializacion []byte,
	huellaEsperadaSHA256 string,
) (PreimagenRecursoAutorizacionEjecucionDocumentalV4, error)
```

InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4 acepta solo el
formato cerrado, limitado y canonico. El resultado sigue sin ser autoridad.

```go
func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) Format(estado fmt.State, _ rune)

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) GoString() string

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) HuellaAmbitosSHA256() (
	string,
	error,
)

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) HuellaContextoRecursoSHA256() (
	string,
	error,
)

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) HuellaSHA256() (string, error)

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) LogValue() slog.Value

func (PreimagenRecursoAutorizacionEjecucionDocumentalV4) MarshalBinary() ([]byte, error)

func (PreimagenRecursoAutorizacionEjecucionDocumentalV4) MarshalJSON() ([]byte, error)

func (PreimagenRecursoAutorizacionEjecucionDocumentalV4) MarshalText() ([]byte, error)

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) RecursoCanonico() (
	domain.RecursoAutorizable,
	error,
)
```

RecursoCanonico devuelve la preimagen completa mediante copia profunda.
Los valores sensibles deben haber sido tokenizados/HMAC antes de crear el
RecursoAutorizable original.

```go
func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) SerializacionCanonicaParaPersistencia() (
	[]byte,
	error,
)

func (PreimagenRecursoAutorizacionEjecucionDocumentalV4) String() string

func (*PreimagenRecursoAutorizacionEjecucionDocumentalV4) UnmarshalBinary([]byte) error

func (*PreimagenRecursoAutorizacionEjecucionDocumentalV4) UnmarshalJSON([]byte) error

func (*PreimagenRecursoAutorizacionEjecucionDocumentalV4) UnmarshalText([]byte) error

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) Validar() error

type PreparacionCargaDocumentalPersistida struct {
	Carga      domain.CargaDocumental
	Manifiesto domain.ManifiestoPreparacionCargaDirectaV1
}
```

PreparacionCargaDocumentalPersistida es una instantanea coherente. El
repositorio nunca devuelve el agregado y el manifiesto mediante dos lecturas
independientes que puedan observar versiones distintas.

```go
func (p PreparacionCargaDocumentalPersistida) Validar() error

type PreparacionEjecucionDocumentalV3Nominal struct {
	ReservaRef  string
	BorradorRef string
	EfectoRef   string
	Repetida    bool
	Estado      EstadoEjecucionDocumentalV3
}

func (p PreparacionEjecucionDocumentalV3Nominal) ValidarContra(
	s SolicitudPrepararEjecucionDocumentalV3,
) error

type PreparacionEntradaNeutralDocumentalNominal struct {
	// Has unexported fields.
}
```

PreparacionEntradaNeutralDocumentalNominal fija el contenido y su codec
antes de solicitar la HMAC al servicio interno. Es opaca, no autoritativa y
no puede serializarse por mecanismos genericos porque puede contener datos
personales.

```go
func NuevaPreparacionEntradaNeutralDocumentalNominal(
	contenido domain.ContenidoDocumento,
) (PreparacionEntradaNeutralDocumentalNominal, error)

func (p PreparacionEntradaNeutralDocumentalNominal) Contenido() (domain.ContenidoDocumento, error)

func (p PreparacionEntradaNeutralDocumentalNominal) ContenidoCanonico() ([]byte, error)

func (p PreparacionEntradaNeutralDocumentalNominal) Format(estado fmt.State, _ rune)

func (p PreparacionEntradaNeutralDocumentalNominal) GoString() string

func (p PreparacionEntradaNeutralDocumentalNominal) LogValue() slog.Value

func (PreparacionEntradaNeutralDocumentalNominal) MarshalBinary() ([]byte, error)

func (PreparacionEntradaNeutralDocumentalNominal) MarshalJSON() ([]byte, error)

func (PreparacionEntradaNeutralDocumentalNominal) MarshalText() ([]byte, error)

func (PreparacionEntradaNeutralDocumentalNominal) String() string

func (*PreparacionEntradaNeutralDocumentalNominal) UnmarshalBinary([]byte) error

func (*PreparacionEntradaNeutralDocumentalNominal) UnmarshalJSON([]byte) error

func (*PreparacionEntradaNeutralDocumentalNominal) UnmarshalText([]byte) error

func (p PreparacionEntradaNeutralDocumentalNominal) Validar() error

type PreparacionEscrituraAlmacenDocumentalV4Nominal struct {
	// Has unexported fields.
}
```

PreparacionEscrituraAlmacenDocumentalV4Nominal es una instantanea opaca y
no autoritativa. Solo nace tras cotejar la solicitud semantica completa,
el resultado del conector, sus capacidades exactas, los bytes observados,
la ejecucion cercada y la politica. No es un recibo ni confirma el efecto.

```go
func NuevaPreparacionEscrituraAlmacenDocumentalV4Nominal(
	solicitud SolicitudEscribirObjeto,
	resultado ResultadoOperacionObjeto,
	capacidades CapacidadesAlmacenObjetos,
	salida SalidaObservadaDocumental,
	vinculo VinculoEjecucionEscrituraAlmacenDocumental,
	politica VinculoPoliticaInmutabilidadDocumental,
) (PreparacionEscrituraAlmacenDocumentalV4Nominal, error)

func (p PreparacionEscrituraAlmacenDocumentalV4Nominal) Format(estado fmt.State, _ rune)

func (p PreparacionEscrituraAlmacenDocumentalV4Nominal) GoString() string

func (p PreparacionEscrituraAlmacenDocumentalV4Nominal) LogValue() slog.Value

func (PreparacionEscrituraAlmacenDocumentalV4Nominal) MarshalBinary() ([]byte, error)

func (PreparacionEscrituraAlmacenDocumentalV4Nominal) MarshalJSON() ([]byte, error)

func (PreparacionEscrituraAlmacenDocumentalV4Nominal) MarshalText() ([]byte, error)

func (PreparacionEscrituraAlmacenDocumentalV4Nominal) String() string

func (*PreparacionEscrituraAlmacenDocumentalV4Nominal) UnmarshalBinary([]byte) error

func (*PreparacionEscrituraAlmacenDocumentalV4Nominal) UnmarshalJSON([]byte) error

func (*PreparacionEscrituraAlmacenDocumentalV4Nominal) UnmarshalText([]byte) error

func (p PreparacionEscrituraAlmacenDocumentalV4Nominal) Validar() error

type ProtectorCodigoCotejo interface {
	ProtegerCodigoCotejo(context.Context, SolicitudProtegerCodigoCotejo) (CustodiaCodigoCotejo, error)
	RecuperarCodigoCotejo(context.Context, SolicitudRecuperarCodigoCotejo) (RecuperacionCodigoCotejo, error)
	EliminarCodigoCotejoHuerfano(context.Context, SolicitudEliminarCodigoCotejoHuerfano) error
}
```

ProtectorCodigoCotejo representa Vault, KMS/HSM o un servicio equivalente.
Las implementaciones productivas deben cifrar con autenticacion, versionar
claves, llamar ValidarEn con reloj fiable justo antes de cada operacion y
eliminar solo referencias huerfanas con autoridad tecnica expresa.

```go
type ProyeccionAplicacionAutorizacionEjecucionDocumentalV4 struct {
	Esquema                         string
	Clave                           ClaveAplicacionAutorizacionEjecucionDocumentalV4
	EsquemaHuellaDecision           string
	HuellaDecisionSHA256            string
	PerfilActivoRef                 string
	ContextoActorHuellaSHA256       string
	Accion                          string
	RecursoRef                      string
	ModuloID                        string
	TipoRecurso                     string
	HuellaRecursoSHA256             string
	HuellaAmbitosSHA256             string
	Finalidad                       string
	CorrelacionRef                  string
	HuellaCamposPermitidosSHA256    string
	HuellaObligacionesSHA256        string
	HuellaCumplimientosSHA256       string
	VerificadaEn                    time.Time
	VinculadaEn                     time.Time
	SolicitadaEn                    time.Time
	ValidaHasta                     time.Time
	HuellaSolicitudVinculadaSHA256  string
	HuellaSolicitudAplicacionSHA256 string
}
```

ProyeccionAplicacionAutorizacionEjecucionDocumentalV4 es una copia defensiva
y no autoritativa para mapear columnas dentro de la transaccion. No permite
reconstruir la solicitud opaca ni demuestra procedencia del PDP.

```go
func (p ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) Format(estado fmt.State, _ rune)

func (p ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) GoString() string

func (p ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) LogValue() slog.Value

func (ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) MarshalJSON() ([]byte, error)

func (ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) MarshalText() ([]byte, error)

func (ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) String() string

func (*ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) UnmarshalJSON([]byte) error

func (*ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) UnmarshalText([]byte) error

type ProyeccionContextoOperacionAlmacen = almacencanonico.ProyeccionContextoOperacionAlmacen
```

ProyeccionContextoOperacionAlmacen es una copia defensiva para conectores
dentro del proceso. No permite reconstruir la capacidad y tampoco se puede
serializar mediante los codificadores habituales.

```go
type ProyeccionIntentoCASReconciliacionDocumentalV4 struct {
	// Has unexported fields.
}
```

ProyeccionIntentoCASReconciliacionDocumentalV4 describe el registro de
evidencia que podria solicitarse despues de una verificacion criptografica
fresca. V4 no proyecta ninguna transicion de estado. Nunca es una capacidad
ni debe aceptarse directamente por un repositorio.

Queda pendiente el puerto durable que, en una sola transaccion y antes
de COMMIT, debera consumir con UNIQUE ConsultaRef y HuellaRetoSHA256,
comprobar estado='indeterminada' AND version=N AND cercado=S y anexar
evidencia sin confirmar ni abandonar. Una futura version solo podra mutar
estado con un resultado remoto completo o una prueba fuerte de inexistencia.
Este tipo no simula esas garantias.

```go
func ProyectarIntentoCASReconciliacionDocumentalV4(
	consulta ConsultaReconciliacionDocumentalV4,
	respuesta RespuestaCrudaReconciliacionDocumentalV4,
) (ProyeccionIntentoCASReconciliacionDocumentalV4, error)

func (p ProyeccionIntentoCASReconciliacionDocumentalV4) AccionProyectada() (
	AccionProyectadaReconciliacionDocumentalV4,
	error,
)

func (p ProyeccionIntentoCASReconciliacionDocumentalV4) ClaveConsumoUnico() (
	ClaveConsumoConsultaReconciliacionDocumentalV4,
	error,
)

func (p ProyeccionIntentoCASReconciliacionDocumentalV4) CondicionCAS() (
	CondicionCASReconciliacionDocumentalV4,
	error,
)

func (p ProyeccionIntentoCASReconciliacionDocumentalV4) RequiereVerificacionCriptograficaFresca() bool
```

RequiereVerificacionCriptograficaFresca es siempre cierto: ni siquiera una
respuesta "aplicado" puede mutar estado a partir de esta proyeccion cruda.

```go
func (p ProyeccionIntentoCASReconciliacionDocumentalV4) ValidarContra(
	consulta ConsultaReconciliacionDocumentalV4,
	respuesta RespuestaCrudaReconciliacionDocumentalV4,
) error

type ProyeccionManifiestoGeneracionDocumental struct {
	Esquema                string
	PlantillaID            string
	PlantillaVersion       int
	ModuloID               string
	TipoDocumental         string
	HuellaPlantillaSHA256  string
	PermisoGenerar         string
	HuellaManifiestoSHA256 string
	Pasos                  []ProyeccionPasoGeneracionDocumental
}
```

ProyeccionManifiestoGeneracionDocumental es una copia defensiva interna.
PermisoGenerar permite formular la solicitud al PDP, pero no es autoridad:
la fabrica volvera a cotejar la decision con el valor opaco.

```go
func (p ProyeccionManifiestoGeneracionDocumental) Format(estado fmt.State, _ rune)

func (p ProyeccionManifiestoGeneracionDocumental) GoString() string

func (p ProyeccionManifiestoGeneracionDocumental) LogValue() slog.Value

func (ProyeccionManifiestoGeneracionDocumental) MarshalJSON() ([]byte, error)

func (ProyeccionManifiestoGeneracionDocumental) MarshalText() ([]byte, error)

func (ProyeccionManifiestoGeneracionDocumental) String() string

func (*ProyeccionManifiestoGeneracionDocumental) UnmarshalJSON([]byte) error

func (*ProyeccionManifiestoGeneracionDocumental) UnmarshalText([]byte) error

type ProyeccionPasoGeneracionDocumental struct {
	PasoRef           PasoOperacionAlmacen
	ReferenciaLogica  string
	ClaveIdempotencia string
	Formato           domain.FormatoDocumento
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
	HuellaPasoSHA256  string
}

type PruebaCrudaAtestacionDespachoDocumentalV3 struct {
	// Has unexported fields.
}
```

PruebaCrudaAtestacionDespachoDocumentalV3 transporta material suficiente
para que un adaptador criptografico real compruebe una HMAC. Es restaurable
y, por tanto, nunca acredita por si misma que la comprobacion haya ocurrido.

```go
func NuevaPruebaCrudaAtestacionDespachoDocumentalV3(
	algoritmo, audiencia, contexto, claveGestionadaRef string,
	revisionClaveGestionada uint64,
	evidenciaOperacionRef string,
	mensajeCanonico, sobreCriptografico []byte,
) (PruebaCrudaAtestacionDespachoDocumentalV3, error)

func (p PruebaCrudaAtestacionDespachoDocumentalV3) EvidenciaOperacionRef() (string, error)

func (p PruebaCrudaAtestacionDespachoDocumentalV3) Format(estado fmt.State, _ rune)

func (p PruebaCrudaAtestacionDespachoDocumentalV3) GoString() string

func (p PruebaCrudaAtestacionDespachoDocumentalV3) HuellasSHA256() (
	mensaje, sobre string,
	err error,
)

func (p PruebaCrudaAtestacionDespachoDocumentalV3) LogValue() slog.Value

func (PruebaCrudaAtestacionDespachoDocumentalV3) MarshalBinary() ([]byte, error)

func (PruebaCrudaAtestacionDespachoDocumentalV3) MarshalJSON() ([]byte, error)

func (PruebaCrudaAtestacionDespachoDocumentalV3) MarshalText() ([]byte, error)

func (p PruebaCrudaAtestacionDespachoDocumentalV3) MensajeCanonico() ([]byte, error)

func (p PruebaCrudaAtestacionDespachoDocumentalV3) Perfil() (
	algoritmo, audiencia, contexto, claveGestionadaRef string,
	revisionClaveGestionada uint64,
	err error,
)

func (p PruebaCrudaAtestacionDespachoDocumentalV3) SobreCriptografico() ([]byte, error)

func (PruebaCrudaAtestacionDespachoDocumentalV3) String() string

func (*PruebaCrudaAtestacionDespachoDocumentalV3) UnmarshalBinary([]byte) error

func (*PruebaCrudaAtestacionDespachoDocumentalV3) UnmarshalJSON([]byte) error

func (*PruebaCrudaAtestacionDespachoDocumentalV3) UnmarshalText([]byte) error

func (p PruebaCrudaAtestacionDespachoDocumentalV3) Validar() error

type PruebaCrudaEscrituraAlmacen struct {
	// Has unexported fields.
}
```

PruebaCrudaEscrituraAlmacen es solo un sobre opaco pendiente de
verificacion. Construirlo, persistirlo o calcular su SHA-256 no concede
autoridad y nunca permite confirmar una escritura.

```go
func NuevaPruebaCrudaEscrituraAlmacen(
	pruebaRef string,
	declaracion DeclaracionEscrituraAlmacenDocumental,
	sobre SobreCriptograficoDocumentalCrudoV4,
) (PruebaCrudaEscrituraAlmacen, error)

func (p PruebaCrudaEscrituraAlmacen) Declaracion() (
	DeclaracionEscrituraAlmacenDocumental,
	error,
)

func (p PruebaCrudaEscrituraAlmacen) Format(estado fmt.State, _ rune)

func (p PruebaCrudaEscrituraAlmacen) GoString() string

func (p PruebaCrudaEscrituraAlmacen) HuellaMensajeSHA256() (string, error)

func (p PruebaCrudaEscrituraAlmacen) LogValue() slog.Value

func (PruebaCrudaEscrituraAlmacen) MarshalBinary() ([]byte, error)

func (PruebaCrudaEscrituraAlmacen) MarshalJSON() ([]byte, error)

func (PruebaCrudaEscrituraAlmacen) MarshalText() ([]byte, error)

func (p PruebaCrudaEscrituraAlmacen) Mensaje() ([]byte, error)

func (p PruebaCrudaEscrituraAlmacen) PruebaRef() (string, error)

func (p PruebaCrudaEscrituraAlmacen) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
)

func (PruebaCrudaEscrituraAlmacen) String() string

func (*PruebaCrudaEscrituraAlmacen) UnmarshalBinary([]byte) error

func (*PruebaCrudaEscrituraAlmacen) UnmarshalJSON([]byte) error

func (*PruebaCrudaEscrituraAlmacen) UnmarshalText([]byte) error

func (p PruebaCrudaEscrituraAlmacen) ValidarSintaxis() error
```

ValidarSintaxis solo coteja el mensaje nominal y el contenedor crudo.
No interpreta COSE ni concede autoridad para confirmar el efecto remoto.

```go
type PruebaCrudaReciboComponenteDocumentalV4 struct {
	// Has unexported fields.
}

func NuevaPruebaCrudaReciboComponenteDocumentalV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (PruebaCrudaReciboComponenteDocumentalV4, error)

func (p PruebaCrudaReciboComponenteDocumentalV4) Format(estado fmt.State, _ rune)

func (p PruebaCrudaReciboComponenteDocumentalV4) GoString() string

func (p PruebaCrudaReciboComponenteDocumentalV4) LogValue() slog.Value

func (PruebaCrudaReciboComponenteDocumentalV4) MarshalBinary() ([]byte, error)

func (PruebaCrudaReciboComponenteDocumentalV4) MarshalJSON() ([]byte, error)

func (PruebaCrudaReciboComponenteDocumentalV4) MarshalText() ([]byte, error)

func (p PruebaCrudaReciboComponenteDocumentalV4) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
)

func (PruebaCrudaReciboComponenteDocumentalV4) String() string

func (*PruebaCrudaReciboComponenteDocumentalV4) UnmarshalBinary([]byte) error

func (*PruebaCrudaReciboComponenteDocumentalV4) UnmarshalJSON([]byte) error

func (*PruebaCrudaReciboComponenteDocumentalV4) UnmarshalText([]byte) error

func (p PruebaCrudaReciboComponenteDocumentalV4) ValidarSintaxis() error

type PruebaCrudaTokenCercadoDocumentalV4 struct {
	// Has unexported fields.
}

func NuevaPruebaCrudaTokenCercadoDocumentalV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (PruebaCrudaTokenCercadoDocumentalV4, error)

func (p PruebaCrudaTokenCercadoDocumentalV4) Format(estado fmt.State, _ rune)

func (p PruebaCrudaTokenCercadoDocumentalV4) GoString() string

func (p PruebaCrudaTokenCercadoDocumentalV4) LogValue() slog.Value

func (PruebaCrudaTokenCercadoDocumentalV4) MarshalBinary() ([]byte, error)

func (PruebaCrudaTokenCercadoDocumentalV4) MarshalJSON() ([]byte, error)

func (PruebaCrudaTokenCercadoDocumentalV4) MarshalText() ([]byte, error)

func (p PruebaCrudaTokenCercadoDocumentalV4) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
)

func (PruebaCrudaTokenCercadoDocumentalV4) String() string

func (*PruebaCrudaTokenCercadoDocumentalV4) UnmarshalBinary([]byte) error

func (*PruebaCrudaTokenCercadoDocumentalV4) UnmarshalJSON([]byte) error

func (*PruebaCrudaTokenCercadoDocumentalV4) UnmarshalText([]byte) error

func (p PruebaCrudaTokenCercadoDocumentalV4) ValidarSintaxis() error

type ReciboCargaDirecta = almacencanonico.ReciboCargaDirecta
```

ReciboCargaDirecta es el valor opaco y de un uso entregado junto a la
concesion temporal. Es un secreto efimero: solo se revela al transporte y
nunca se persiste en claro, serializa por reflexion ni registra.

```go
func NuevoReciboCargaDirecta(valor string) (ReciboCargaDirecta, error)

type ReciboConsultaInternaFuenteAutoridad struct {
	// Has unexported fields.
}
```

ReciboConsultaInternaFuenteAutoridad acredita que la decision se consumio,
la auditoria firmada y encadenada se registro y, si existia, que snapshot
exacto se leyo. Solo puede salir de la barrera despues de su COMMIT.

```go
func NuevoReciboConsultaInternaFuenteAutoridad(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
	datos DatosReciboConsultaInternaFuenteAutoridad,
) (ReciboConsultaInternaFuenteAutoridad, error)

func (r ReciboConsultaInternaFuenteAutoridad) Datos() (
	DatosReciboConsultaInternaFuenteAutoridad,
	error,
)

func (r ReciboConsultaInternaFuenteAutoridad) Format(estado fmt.State, _ rune)

func (r ReciboConsultaInternaFuenteAutoridad) GoString() string

func (*ReciboConsultaInternaFuenteAutoridad) GobDecode([]byte) error

func (ReciboConsultaInternaFuenteAutoridad) GobEncode() ([]byte, error)

func (r ReciboConsultaInternaFuenteAutoridad) LogValue() slog.Value

func (ReciboConsultaInternaFuenteAutoridad) MarshalBinary() ([]byte, error)

func (ReciboConsultaInternaFuenteAutoridad) MarshalCBOR() ([]byte, error)

func (ReciboConsultaInternaFuenteAutoridad) MarshalJSON() ([]byte, error)

func (ReciboConsultaInternaFuenteAutoridad) MarshalText() ([]byte, error)

func (ReciboConsultaInternaFuenteAutoridad) MarshalXML(*xml.Encoder, xml.StartElement) error

func (ReciboConsultaInternaFuenteAutoridad) MarshalYAML() (any, error)

func (ReciboConsultaInternaFuenteAutoridad) String() string

func (*ReciboConsultaInternaFuenteAutoridad) UnmarshalBinary([]byte) error

func (*ReciboConsultaInternaFuenteAutoridad) UnmarshalCBOR([]byte) error

func (*ReciboConsultaInternaFuenteAutoridad) UnmarshalJSON([]byte) error

func (*ReciboConsultaInternaFuenteAutoridad) UnmarshalText([]byte) error

func (*ReciboConsultaInternaFuenteAutoridad) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*ReciboConsultaInternaFuenteAutoridad) UnmarshalYAML(func(any) error) error

func (r ReciboConsultaInternaFuenteAutoridad) ValidarPara(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
) error

type ReciboEjecucionComponenteDocumentalNominal struct {
	// Has unexported fields.
}

func NuevoReciboEjecucionComponenteDocumentalNominal(
	compromiso CompromisoEjecucionComponenteDocumental,
	sobre SobreReciboEjecucionDocumentalCrudo,
	reciboRef string,
	resultado ResultadoEjecucionComponenteDocumental,
	huellaSalidaSHA256 string,
	tamanoSalida uint64,
	identidad IdentidadEjecucionComponenteDocumental,
	emitidoEn time.Time,
) (ReciboEjecucionComponenteDocumentalNominal, error)
```

NuevoReciboEjecucionComponenteDocumentalNominal solo comprueba el contrato
de valor: NO acredita la firma ni la atestacion criptografica y nunca
concede autoridad por si mismo.

```go
func (r ReciboEjecucionComponenteDocumentalNominal) Datos() (
	DatosReciboEjecucionComponenteDocumentalNominal,
	error,
)

func (r ReciboEjecucionComponenteDocumentalNominal) Format(estado fmt.State, _ rune)

func (r ReciboEjecucionComponenteDocumentalNominal) GoString() string

func (r ReciboEjecucionComponenteDocumentalNominal) HuellaSHA256() (string, error)

func (r ReciboEjecucionComponenteDocumentalNominal) IndependienteDe(
	otro ReciboEjecucionComponenteDocumentalNominal,
) bool
```

IndependienteDe exige segregacion tanto declarativa como observada durante
la ejecucion. El broker y el nodo fisico pueden ser compartidos; la carga de
trabajo, proceso, dominio, clave y medicion no.

```go
func (r ReciboEjecucionComponenteDocumentalNominal) LogValue() slog.Value

func (ReciboEjecucionComponenteDocumentalNominal) MarshalBinary() ([]byte, error)

func (ReciboEjecucionComponenteDocumentalNominal) MarshalJSON() ([]byte, error)

func (ReciboEjecucionComponenteDocumentalNominal) MarshalText() ([]byte, error)

func (ReciboEjecucionComponenteDocumentalNominal) String() string

func (*ReciboEjecucionComponenteDocumentalNominal) UnmarshalBinary([]byte) error

func (*ReciboEjecucionComponenteDocumentalNominal) UnmarshalJSON([]byte) error

func (*ReciboEjecucionComponenteDocumentalNominal) UnmarshalText([]byte) error

func (r ReciboEjecucionComponenteDocumentalNominal) Validar() error

func (r ReciboEjecucionComponenteDocumentalNominal) ValidarContra(
	compromiso CompromisoEjecucionComponenteDocumental,
	sobre SobreReciboEjecucionDocumentalCrudo,
) error

type ReciboEscrituraObjetoMaterialV2 struct {
	// Has unexported fields.
}
```

ReciboEscrituraObjetoMaterialV2 liga el objeto original con el perfil de
capacidades y un plan material. No es aun un contrato productivo: sin diario
durable, persistencia y recuperacion el flujo completo sigue NO-GO.

```go
func NuevoReciboEscrituraObjetoMaterialV2(
	ctx context.Context,
	solicitud SolicitudEscribirObjeto,
	resultado ResultadoOperacionObjeto,
	capacidades CapacidadesAlmacenObjetos,
	perfil PerfilCapacidadesAlmacenMaterialV2,
	verificadorPublicacion VerificadorPerfilPublicadoMaterialV2,
	seleccionPlan SeleccionPlanMaterialAlmacenV2,
	verificadorPlan VerificadorPlanMaterialAlmacenV2,
	registroReferencias RegistroReferenciasReciboMaterialV2,
	verificadorReferencia VerificadorReferenciaReciboMaterialV2,
	atestador AtestadorMaterialAlmacenV2,
	verificador VerificadorAtestacionMaterialAlmacenV2,
) (ReciboEscrituraObjetoMaterialV2, error)

func (r ReciboEscrituraObjetoMaterialV2) BytesCanonicos() ([]byte, error)

func (ReciboEscrituraObjetoMaterialV2) Format(e fmt.State, _ rune)

func (ReciboEscrituraObjetoMaterialV2) GoString() string

func (r ReciboEscrituraObjetoMaterialV2) HuellaSHA256() ([sha256.Size]byte, error)

func (r ReciboEscrituraObjetoMaterialV2) Instantanea() (
	InstantaneaObjetoMaterialV2,
	error,
)

func (ReciboEscrituraObjetoMaterialV2) LogValue() slog.Value

func (ReciboEscrituraObjetoMaterialV2) MarshalBinary() ([]byte, error)

func (ReciboEscrituraObjetoMaterialV2) MarshalJSON() ([]byte, error)

func (ReciboEscrituraObjetoMaterialV2) MarshalText() ([]byte, error)

func (ReciboEscrituraObjetoMaterialV2) String() string

func (*ReciboEscrituraObjetoMaterialV2) UnmarshalBinary([]byte) error

func (*ReciboEscrituraObjetoMaterialV2) UnmarshalJSON([]byte) error

func (*ReciboEscrituraObjetoMaterialV2) UnmarshalText([]byte) error

func (r ReciboEscrituraObjetoMaterialV2) Validar() error

func (r ReciboEscrituraObjetoMaterialV2) VerificarAtestacion(
	ctx context.Context,
	verificador VerificadorAtestacionMaterialAlmacenV2,
) error

type ReciboInicioEfectoDocumentalV3Nominal struct {
	// Has unexported fields.
}
```

ReciboInicioEfectoDocumentalV3Nominal es un recibo durable nominal y
atestado. Lo devuelve el registro en el mismo COMMIT del inicio, pero solo
la posterior relectura y reclamacion CAS de su outbox puede convertirlo en
candidato a despacho. Su fabrica publica restaura datos; no certifica su
procedencia.

```go
func NuevoReciboInicioEfectoDocumentalV3Nominal(
	solicitud SolicitudIniciarEfectoDocumentalV3,
	inicioRef string,
	versionInicioCAS uint64,
	auditoriaInicioRef, outboxInicioRef string,
	atestacionInicio PruebaCrudaAtestacionDespachoDocumentalV3,
) (ReciboInicioEfectoDocumentalV3Nominal, error)

func (r ReciboInicioEfectoDocumentalV3Nominal) Datos() (DatosReciboInicioEfectoDocumentalV3Nominal, error)

func (r ReciboInicioEfectoDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (r ReciboInicioEfectoDocumentalV3Nominal) GoString() string

func (r ReciboInicioEfectoDocumentalV3Nominal) HuellaSHA256() (string, error)

func (r ReciboInicioEfectoDocumentalV3Nominal) LogValue() slog.Value

func (ReciboInicioEfectoDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (ReciboInicioEfectoDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (ReciboInicioEfectoDocumentalV3Nominal) MarshalText() ([]byte, error)

func (ReciboInicioEfectoDocumentalV3Nominal) String() string

func (*ReciboInicioEfectoDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*ReciboInicioEfectoDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*ReciboInicioEfectoDocumentalV3Nominal) UnmarshalText([]byte) error

func (r ReciboInicioEfectoDocumentalV3Nominal) Validar() error

func (r ReciboInicioEfectoDocumentalV3Nominal) ValidarContra(
	solicitud SolicitudIniciarEfectoDocumentalV3,
) error

type RecibosEjecucionDocumentalV3 struct {
	Render      ReciboEjecucionComponenteDocumentalNominal
	Estructural ReciboEjecucionComponenteDocumentalNominal
	Semantico   ReciboEjecucionComponenteDocumentalNominal
}

func (r RecibosEjecucionDocumentalV3) Huellas() (HuellasRecibosEjecucionDocumentalV3, error)

func (r RecibosEjecucionDocumentalV3) ValidarContra(
	manifiesto ManifiestoEjecucionDocumentalV3,
	resultado ResultadoEfectoRenderizadoDocumentalV3Crudo,
	reservaRef string,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	ordenConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
	instante time.Time,
) error

type ReconciliadorEfectosRenderizadoDocumentalV3 interface {
	ConsultarEfectoRenderizadoDocumentalV3(
		context.Context,
		SolicitudConsultarEfectoDocumentalV3,
	) (ResultadoConsultaEfectoDocumentalV3Crudo, error)
}

type RecuperacionCodigoCotejo struct {
	Secreto      SecretoCodigoCotejo
	ConectorID   string
	EvidenciaRef string
}

type ReferenciaDevolucionCobro = pagoscanonicos.ReferenciaDevolucionCobro

type ReferenciaEstadoFuenteAutoridad struct {
	Fuente               domain.ReferenciaFuenteAutoridad
	Revision             uint64
	Estado               domain.EstadoFuenteAutoridad
	HuellaHistoriaSHA256 string
	HuellaEstadoSHA256   string
}
```

ReferenciaEstadoFuenteAutoridad es la precondicion OCC completa. La
referencia de contenido, por si sola, no detecta una transicion de estado.

```go
func EstadoExactoFuenteAutoridad(
	fuente domain.FuenteAutoridadVersionada,
) (ReferenciaEstadoFuenteAutoridad, error)

func (r ReferenciaEstadoFuenteAutoridad) Validar() error

type ReferenciaObjetoAlmacen = almacencanonico.ReferenciaObjetoAlmacen

type ReferenciaOperacionCobro = pagoscanonicos.ReferenciaOperacionCobro

type ReferenciaOperacionFuenteAutoridad struct {
	// Has unexported fields.
}

func NuevaReferenciaOperacionFuenteAutoridad(valor string) (ReferenciaOperacionFuenteAutoridad, error)

func (r ReferenciaOperacionFuenteAutoridad) Format(estado fmt.State, _ rune)

func (r ReferenciaOperacionFuenteAutoridad) GoString() string

func (*ReferenciaOperacionFuenteAutoridad) GobDecode([]byte) error

func (ReferenciaOperacionFuenteAutoridad) GobEncode() ([]byte, error)

func (r ReferenciaOperacionFuenteAutoridad) LogValue() slog.Value

func (ReferenciaOperacionFuenteAutoridad) MarshalBinary() ([]byte, error)

func (ReferenciaOperacionFuenteAutoridad) MarshalJSON() ([]byte, error)

func (ReferenciaOperacionFuenteAutoridad) MarshalText() ([]byte, error)

func (ReferenciaOperacionFuenteAutoridad) MarshalXML(*xml.Encoder, xml.StartElement) error

func (r ReferenciaOperacionFuenteAutoridad) Referencia() (string, error)

func (ReferenciaOperacionFuenteAutoridad) String() string

func (*ReferenciaOperacionFuenteAutoridad) UnmarshalBinary([]byte) error

func (*ReferenciaOperacionFuenteAutoridad) UnmarshalJSON([]byte) error

func (*ReferenciaOperacionFuenteAutoridad) UnmarshalText([]byte) error

func (*ReferenciaOperacionFuenteAutoridad) UnmarshalXML(*xml.Decoder, xml.StartElement) error

type ReferenciaSolicitudFuenteAutoridad struct {
	// Has unexported fields.
}

func NuevaReferenciaSolicitudFuenteAutoridad(valor string) (ReferenciaSolicitudFuenteAutoridad, error)

func (r ReferenciaSolicitudFuenteAutoridad) Format(estado fmt.State, _ rune)

func (r ReferenciaSolicitudFuenteAutoridad) GoString() string

func (*ReferenciaSolicitudFuenteAutoridad) GobDecode([]byte) error

func (ReferenciaSolicitudFuenteAutoridad) GobEncode() ([]byte, error)

func (r ReferenciaSolicitudFuenteAutoridad) LogValue() slog.Value

func (ReferenciaSolicitudFuenteAutoridad) MarshalBinary() ([]byte, error)

func (ReferenciaSolicitudFuenteAutoridad) MarshalJSON() ([]byte, error)

func (ReferenciaSolicitudFuenteAutoridad) MarshalText() ([]byte, error)

func (ReferenciaSolicitudFuenteAutoridad) MarshalXML(*xml.Encoder, xml.StartElement) error

func (r ReferenciaSolicitudFuenteAutoridad) Referencia() (string, error)

func (ReferenciaSolicitudFuenteAutoridad) String() string

func (*ReferenciaSolicitudFuenteAutoridad) UnmarshalBinary([]byte) error

func (*ReferenciaSolicitudFuenteAutoridad) UnmarshalJSON([]byte) error

func (*ReferenciaSolicitudFuenteAutoridad) UnmarshalText([]byte) error

func (*ReferenciaSolicitudFuenteAutoridad) UnmarshalXML(*xml.Decoder, xml.StartElement) error

type RegistroAuditoriaCobro = pagoscanonicos.RegistroAuditoriaCobro

type RegistroComponentesDocumentalesAtestados interface {
	BuscarComponentesDocumentalesAtestados(
		context.Context,
		ConsultaComponenteDocumentalAtestado,
	) ([]DescriptorComponenteDocumentalAtestado, error)
}

type RegistroDecisionesAutorizacion interface {
	RegistrarDecisionSiInstantaneaVigente(context.Context, domain.DecisionAutorizacion) error
}
```

RegistroDecisionesAutorizacion conserva exclusivamente concesiones
ejecutables. Debe ser duradero, de solo adicion y carecer de capacidades
de consulta en produccion. RegistrarDecisionSiInstantaneaVigente debe
comparar y cambiar de forma atomica: antes de insertar revalida que
la asignacion actual y el control del catalogo de politicas coinciden
exactamente con la evidencia de la decision. Cualquier diferencia devuelve
ErrInstantaneaAutorizacionObsoleta y no inserta nada. Una concesion no es
valida hasta completar este registro.

```go
type RegistroDecisionesAutorizacionSolicitudLigadaV2 interface {
	RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
		context.Context,
		OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
	) error
}
```

RegistroDecisionesAutorizacionSolicitudLigadaV2 nunca acepta V1. El nombre
distinto obliga a seleccionar de forma visible el esquema durable.

```go
type RegistroDecisionesReglaFlujo interface {
	RegistrarDecisionReglaFlujo(context.Context, domain.DecisionReglaFlujo, domain.AuditEntry, domain.Event) error
	ObtenerDecisionReglaFlujo(context.Context, string) (domain.DecisionReglaFlujo, error)
}
```

RegistroDecisionesReglaFlujo es de solo adicion. Incluso una denegacion se
registra antes de devolverse para conservar la explicabilidad del intento.

```go
type RegistroDenegacionesAutorizacion interface {
	RegistrarDenegacionAutorizacion(context.Context, domain.DecisionAutorizacion) error
}
```

RegistroDenegacionesAutorizacion conserva el resultado probatorio de
una evaluacion negativa sin convertirlo en una capacidad consumible.
Una denegacion sigue siendo efectiva si este registro falla, pero el fallo
debe propagarse para que operacion y seguridad detecten la perdida de traza.

Este puerto no revalida para conceder ni ofrece lectura al PDP.
Su almacen productivo debe ser append-only y estar separado del registro de
concesiones.

```go
type RegistroDenegacionesAutorizacionSolicitudLigadaV2 interface {
	RegistrarDenegacionAutorizacionSolicitudLigadaV2(
		context.Context,
		OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
	) error
}

type RegistroEfectosGeneracionDocumental interface {
	ReservarEfectoGeneracionDocumental(
		context.Context,
		SolicitudReservarEfectoGeneracionDocumental,
	) (ResultadoReservaEfectoGeneracionDocumental, error)
	ConfirmarPasoGeneracionDocumental(
		context.Context,
		SolicitudConfirmarPasoGeneracionDocumental,
	) error
	MarcarPasoGeneracionDocumentalIndeterminado(
		context.Context,
		SolicitudMarcarPasoGeneracionDocumentalIndeterminado,
	) error
}
```

RegistroEfectosGeneracionDocumental es deliberadamente un puerto distinto
del almacen de objetos. Su adaptador productivo debe ser transaccional y
duradero:
  - Reservar consume una DecisionRef una sola vez y fija (EfectoRef,
    HuellaDecision, HuellaPlan, HuellaManifiesto);
  - Confirmar fija una sola respuesta tecnica para (EfectoRef, PasoRef);
  - MarcarIndeterminado impide reintentos ciegos y fuerza reconciliacion.

No se aporta implementacion en memoria como sustituto productivo: sin este
puerto configurado la generacion con efectos remotos permanece cerrada.

```go
type RegistroEjecucionesDocumentalesV3 interface {
	PrepararEjecucionDocumentalV3(
		context.Context,
		SolicitudPrepararEjecucionDocumentalV3,
	) (PreparacionEjecucionDocumentalV3Nominal, error)
	ActivarEjecucionDocumentalV3AdoptandoOrdenV4(
		context.Context,
		SolicitudActivarEjecucionDocumentalV3,
	) (ActivacionEjecucionDocumentalV3Nominal, error)
	MarcarInicioEfectoDocumentalV3(
		context.Context,
		SolicitudIniciarEfectoDocumentalV3,
	) (ReciboInicioEfectoDocumentalV3Nominal, error)
	ReclamarOrdenDespachoDocumentalV3(
		context.Context,
		SolicitudReclamarOrdenDespachoDocumentalV3,
	) (OrdenDespachoDocumentalV3Nominal, error)
	ConfirmarEjecucionDocumentalV3(context.Context, SolicitudConfirmarEjecucionDocumentalV3) error
	AbandonarEjecucionDocumentalV3(context.Context, SolicitudAbandonarEjecucionDocumentalV3) error
	MarcarEjecucionDocumentalV3Indeterminada(
		context.Context,
		SolicitudMarcarEjecucionDocumentalV3Indeterminada,
	) error
	ObtenerEjecucionDocumentalV3(
		context.Context,
		ConsultaEjecucionDocumentalV3,
	) (InstantaneaEjecucionDocumentalV3Nominal, error)
	AplicarReconciliacionDocumentalV3(context.Context, SolicitudAplicarReconciliacionDocumentalV3) error
}
```

RegistroEjecucionesDocumentalesV3 debe ser durable y transaccional.
Las restricciones UNIQUE de indice HMAC, borrador, efecto y DecisionRef
son permanentes. Confirmacion, manifiesto, evidencia, auditoria y outbox se
confirman juntos; no se ofrece una implementacion en memoria productiva.

ActivarEjecucionDocumentalV3AdoptandoOrdenV4 es la unica frontera de
adopcion. El adaptador PostgreSQL debe abrir una sola transaccion,
bloquear y releer por OrdenConsumoDurableV4Ref la fila V4 autoritativa,
exigir que su orden_ref sea el EfectoRef y cotejar DecisionRef,
HuellaDecisionSHA256, HuellaPlanSHA256, estado pendiente y contexto comun
completos. En el mismo COMMIT debe: insertar una adopcion durable 1:1/UNIQUE
que referencie la orden V4 sin mutarla; guardar en la adopcion sus huellas
autoritativas de aplicacion, orden y contexto; reclamar UNIQUE por orden,
DecisionRef, efecto y reserva; activar la reserva; incrementar el cercado;
y escribir auditoria/outbox. Cualquier cero, cruce, discordancia o replay
revierte todo. Un reintento con la misma intencion estable (reserva, HMAC,
manifiesto, orden, terna y huellas) y otro ActivadaEn recupera el mismo
token con Repetida=true; la marca temporal efimera no crea otra activacion.

MarcarInicioEfectoDocumentalV3 debe, en una sola transaccion, bloquear
y releer el registro V3 activo, su adopcion durable y la orden V4
autoritativa; reconstruir y comparar el VinculoEstableActivacionDocumentalV3
completo; verificar el MAC del token con la clave gestionada indicada por
su referencia (nunca mediante MetadatosComprobacion autocreables); y solo
entonces ejecutar el CAS activa -> iniciada junto con auditoria y outbox.
Cualquier ausencia, cruce o discordancia revierte toda la transaccion y no
inicia el efecto. Devuelve el ReciboInicioEfectoDocumentalV3Nominal creado
en ese mismo COMMIT.

ReclamarOrdenDespachoDocumentalV3 debe bloquear el evento outbox y releer
el recibo de inicio, V3, adopcion y V4. En una sola transaccion ejecuta CAS
pendiente -> reclamada, incrementa VersionReclamacionCAS y escribe auditoria
y atestacion. Una segunda reclamacion, incluso identica, falla cerrada.
La orden devuelta sigue siendo nominal. El servicio de aplicacion privado
debe verificarla una vez por KMS y entregarla al consumo CAS transaccional;
solo entonces crea un comando ConsumidaNominal y lo usa dentro de la misma
llamada.

ConfirmarEjecucionDocumentalV3 y AplicarReconciliacionDocumentalV3 no pueden
confiar en comandos nominales, metadatos publicos ni recibos restaurados.
El registro debe releer inicio, reclamacion, consumo, adopcion V3/V4 y
versiones; verificar criptograficamente COSE/firmas con dependencias
privadas; ejecutar el CAS de estado; y confirmar evidencia, auditoria y
outbox en un solo COMMIT. Abandono e indeterminacion aplican la misma regla
de relectura y CAS cerrado. El sistema permanece NO-GO hasta existir y
probar el servicio application precompuesto que mantenga KMS, consumidor,
despachador y almacen fuera de HTTP, CLI, MCP y modulos funcionales.

```go
type RegistroMarcadoresMetadatoInstitucional interface {
	BuscarMarcadoresMetadatoInstitucional(
		context.Context,
		domain.ReferenciaPerfilDocumental,
		domain.ReferenciaConectorDocumental,
	) ([]MarcadorMetadatoInstitucionalDocumental, error)
}

type RegistroReciboCargaDirecta = almacencanonico.RegistroReciboCargaDirecta
```

RegistroReciboCargaDirecta es la unica informacion propuesta al alta
durable. No contiene una fecha de emision del proceso: el repositorio
elige RegistradoEn dentro de la transaccion y la devuelve en el resultado.
IndiceHMAC deriva del material secreto estable anterior a esa fecha;
VinculoHMAC autentica sus invariantes y la sesion. Ninguno puede sustituirse
por el recibo o por la referencia de sesion.

```go
type RegistroReferenciasReciboMaterialV2 interface {
	ReservarORecuperarReferenciaReciboMaterialV2(
		context.Context,
		SolicitudReservarReferenciaReciboMaterialV2,
	) (ResultadoReferenciaReciboMaterialV2, error)
}

type RegistroRenderizadoresDocumentales interface {
	BuscarRenderizadoresDocumentales(
		context.Context,
		domain.ReferenciaPerfilDocumental,
		domain.ReferenciaConectorDocumental,
	) ([]RenderizadorDocumentalPorPerfil, error)
}
```

RegistroRenderizadoresDocumentales devuelve candidatos exactos. Cero o mas
de uno se cierran en la capa de aplicacion.

```go
type RegistroSituacionesOperativasPerfilDocumental interface {
	// Debe consultar la proyeccion vigente en el origen autoritativo, no una
	// revision historica ni una cache que ignore revocaciones.
	BuscarSituacionesOperativasActuales(
		context.Context,
		ConsultaSituacionOperativaActual,
	) ([]domain.SituacionOperativaPerfilDocumental, error)
}

type RegistryInterconnectPort interface {
	ExchangeRegistryEntry(context.Context, InteropRequest) (InteropResult, error)
}

type Reloj interface {
	Ahora() time.Time
}

type RenderizadorDocumentalPorPerfil interface {
	PerfilDocumental() domain.ReferenciaPerfilDocumental
	DigestPerfilSHA256() string
	ConectorDocumental() domain.ReferenciaConectorDocumental
	Renderizar(context.Context, domain.ContenidoDocumento) ([]byte, error)
	ValidarSalida(context.Context, []byte) error
}
```

RenderizadorDocumentalPorPerfil declara su identidad gobernada; no recibe
comandos ni configuracion procedente del catalogo.

```go
type RenderizadorDocumento interface {
	Formato() domain.FormatoDocumento
	Renderizar(context.Context, domain.ContenidoDocumento) ([]byte, error)
	ValidarSalida(context.Context, []byte) error
}
```

RenderizadorDocumento mantiene el nucleo ajeno a librerias PDF, DOCX o a
servicios de conversion externos.

```go
type RepositorioCargasDocumentales interface {
	Reservar(context.Context, SolicitudReservarCargaDocumental) (ReservaCargaDocumental, error)
	ConfirmarPreparacion(context.Context, SolicitudConfirmarPreparacionCargaDocumental) error
	ConfirmarTransicion(context.Context, ConfirmacionTransicionCargaDocumental) error
	AbandonarReserva(context.Context, TokenReservaCargaDocumental) error
	Obtener(context.Context, string) (domain.CargaDocumental, error)
	ObtenerPreparacion(context.Context, string) (PreparacionCargaDocumentalPersistida, error)
}
```

RepositorioCargasDocumentales aplica idempotencia, control optimista de
concurrencia y la escritura atomica del agregado, auditoria y outbox.
Reservar reclama DecisionRef+efecto+plan+huella antes de cualquier
sesion remota. La reclamacion tiene estados explicitos y nunca
vuelve a quedar disponible tras abandono o expiracion: un reintento
necesita otra decision. ConfirmarPreparacion consume el token,
consume DecisionRef una sola vez con su efecto/plan/huella y fija un
unico manifiesto inmutable, todo en el mismo commit. DecisionRef debe
tener una restriccion UNIQUE: tanto el replay exacto como su cruce con
otro efecto se deniegan. ConfirmarTransicion nunca acepta el token ni
sustituye ese manifiesto. ErrDecisionPreparacionCargaNoDisponible y
ErrDecisionPreparacionCargaYaConsumida son conflictos inequivocos sin
commit parcial; cualquier otro error de ConfirmarPreparacion puede ser una
respuesta ambigua y exige reconciliacion.

```go
type RepositorioCodigosCotejo interface {
	ReservarEmisionCodigoCotejo(context.Context, SolicitudReservarEmisionCodigoCotejo) (ReservaEmisionCodigoCotejo, error)
	ConfirmarReservaCodigoCotejo(context.Context, TokenReservaEmisionCodigoCotejo, string, time.Time, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error
	AbandonarReservaCodigoCotejo(context.Context, TokenReservaEmisionCodigoCotejo) error
	ObtenerCodigoCotejo(context.Context, string) (domain.CodigoCotejo, error)
	ObtenerCodigoCotejoPorDocumento(context.Context, domain.ReferenciaDocumento) (domain.CodigoCotejo, error)
	BuscarCodigoCotejoPorIndices(context.Context, []string) (domain.CodigoCotejo, error)
	ConfirmarActivacionCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarRetiradaCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarSustitucionCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error
	RegistrarConsultaCotejo(context.Context, domain.AuditEntry, domain.Event) error
}
```

RepositorioCodigosCotejo mantiene tres unicidades permanentes: identificador
interno, indice HMAC del CSV y un codigo por version documental. Todas las
mutaciones confirman agregado, auditoria y outbox de forma atomica.

```go
type RepositorioDocumentos interface {
	ConfirmarGeneracion(context.Context, domain.DocumentoGenerado, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ObtenerDocumento(context.Context, string) (domain.DocumentoGenerado, error)
	ListarDocumentosExpediente(context.Context, string) ([]domain.DocumentoGenerado, error)
}
```

RepositorioDocumentos conserva el agregado documental historico. No es el
mecanismo que habilita un efecto remoto: toda generacion nueva debe reservar
primero mediante RegistroEfectosGeneracionDocumental. La confirmacion de
metadatos, auditoria y outbox sigue siendo atomica y la evidencia opaca
nunca autoriza por si sola.

```go
type RepositorioDocumentosLogicos interface {
	ReservarGeneracion(context.Context, SolicitudReservarGeneracionDocumento) (ReservaGeneracionDocumento, error)
	ConfirmarGeneracionLogica(context.Context, TokenReservaGeneracionDocumento, string, time.Time, domain.ResultadoGeneracionDocumento, domain.AuditEntry, domain.Event) error
	AbandonarGeneracion(context.Context, TokenReservaGeneracionDocumento) error
	ObtenerDocumentoLogico(context.Context, domain.ReferenciaDocumento) (domain.DocumentoLogico, error)
	ListarRepresentacionesDocumento(context.Context, domain.ReferenciaDocumento) ([]domain.RepresentacionDocumento, error)
}
```

RepositorioDocumentosLogicos protege la idempotencia funcional y confirma
en una sola transaccion el agregado, sus representaciones, la auditoria y
el outbox. No sustituye la reserva tecnica previa de cada efecto remoto. En
produccion debe implementarse con una restriccion unica por principal+clave
y bloqueo transaccional o mecanismo equivalente.

```go
type RepositorioGobiernoCatalogos interface {
	ConfirmarAltaBorradorCatalogo(context.Context, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarActualizacionBorradorCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarPublicacionCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarRetiradaCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
}
```

RepositorioGobiernoCatalogos confirma cada cambio junto con su auditoria y
outbox. Una version publicada o retirada nunca se sobrescribe.

```go
type RepositorioGobiernoFlujos interface {
	ConfirmarAltaBorradorFlujo(context.Context, domain.DefinicionFlujo, domain.AuditEntry, domain.Event) error
	ConfirmarActualizacionBorradorFlujo(context.Context, string, domain.DefinicionFlujo, domain.AuditEntry, domain.Event) error
	ConfirmarPublicacionFlujo(context.Context, string, domain.DefinicionFlujo, domain.AuditEntry, domain.Event) error
	ConfirmarRetiradaFlujo(context.Context, string, domain.DefinicionFlujo, domain.AuditEntry, domain.Event) error
}
```

RepositorioGobiernoFlujos confirma cada mutacion con su auditoria y evento
en una misma transaccion. Una version publicada o retirada es inmutable.

```go
type RepositorioGobiernoPlantillasDocumento interface {
	ConfirmarAltaBorradorPlantilla(context.Context, domain.PlantillaDocumento, domain.AuditEntry, domain.Event) error
	ConfirmarPublicacionPlantilla(context.Context, string, domain.PlantillaDocumento, domain.AuditEntry, domain.Event) error
}
```

RepositorioGobiernoPlantillasDocumento confirma cada alta o publicacion con
su evidencia y outbox. Una interfaz separada evita que el caso de uso pueda
sobrescribir una plantilla publicada mediante el catalogo de consulta.

```go
type RepositorioGobiernoPoliticasCotejo interface {
	ConfirmarAltaBorradorPoliticaCotejo(context.Context, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarActualizacionBorradorPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarPublicacionPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarRetiradaPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error
}
```

RepositorioGobiernoPoliticasCotejo confirma estado, auditoria y outbox en
una unica transaccion. La huella anterior actua como control optimista.

```go
type RepositorioInstanciasFlujo interface {
	ConfirmarInicioInstanciaFlujo(context.Context, domain.InstanciaFlujo, domain.AuditEntry, domain.Event) error
	ConfirmarTransicionInstanciaFlujo(
		context.Context,
		string,
		domain.InstanciaFlujo,
		domain.CambioEstadoFlujo,
		domain.DecisionReglaFlujo,
		*domain.EvidenciaAprobacionFlujo,
		domain.AuditEntry,
		domain.Event,
	) error
}
```

RepositorioInstanciasFlujo aplica control optimista y confirma estado,
auditoria y outbox de forma atomica. La decision de regla debe estar
registrada previamente y se vuelve a cotejar al confirmar la transicion.

```go
type RepositorioOrdenesCobro interface {
	ReservarCreacion(context.Context, SolicitudReservaOrdenCobro) (ReservaOrdenCobro, error)
	ConfirmarCreacion(context.Context, SolicitudConfirmarCreacionOrdenCobro) error
	AbandonarReservaCreacion(context.Context, SolicitudAbandonarReservaOrdenCobro) error
	ReservarDevolucion(context.Context, SolicitudReservaDevolucionCobro) (ReservaDevolucionCobro, error)
	ConfirmarDevolucion(context.Context, SolicitudConfirmarReservaDevolucionCobro) error
	AbandonarReservaDevolucion(context.Context, SolicitudAbandonarReservaDevolucionCobro) error
	ObtenerOrden(context.Context, string) (domain.OrdenCobro, error)
	ObtenerOrdenPorOperacion(context.Context, ReferenciaOperacionCobro) (domain.OrdenCobro, error)
	ConfirmarTransicion(context.Context, SolicitudConfirmarTransicionOrdenCobro) error
}
```

RepositorioOrdenesCobro debe aplicar cada confirmacion en una unica
transaccion: agregado, auditoria inmutable y outbox, o ninguno. Al confirmar
una creacion obtiene el instante de un reloj transaccional (nunca de la
solicitud), valida EvidenciaAutorizacion en ese instante y consume su
DecisionRef de forma atomica con el efecto. El consumo es una relacion
unica DecisionRef -> (OrdenRef, HuellaEfectoSHA256). Una decision ya
consumida solo puede resolver idempotentemente ese mismo efecto existente;
nunca vuelve a escribirlo. HuellaEfectoSHA256 es el control de concurrencia
de la orden y Mutacion.Validar liga a esa orden la auditoria y el outbox
derivados. Reutilizar la decision para otra orden o huella deniega.
Datos() fija la lista positiva exacta que el adaptador debe comparar contra
sus registros oficiales: decision activa y su huella canonica; asignacion
activa, vigente y su huella; version activa y control de vigencia del rol
(referencia, revision, estado y huellas); revision y huella del catalogo
completo; control de sesion (referencia, revision, huella y vigencia);
y contexto de actor (referencia, version y huella). Una retirada,
revocacion, ausencia, ambiguedad, CAS perdido o control desconocido deniega
sin escrituras. La evidencia no reemplaza estas lecturas ni permite confiar
en una copia anterior a la transaccion.

En la misma transaccion hace ademas CAS de todos los datos de la reserva
y comprueba en el registro oficial el control exacto de liquidacion:
referencia, revision, huella, estado exigible y vigencia. El instante debe
estar dentro de las vigencias de reserva, decision, sesion y liquidacion.
La transicion ordinaria usa comparacion simultanea de version y huella.

Un adaptador cuya liquidacion autoritativa vive en una fuente externa y no
puede ofrecer un bloqueo, fence o CAS que permanezca valido hasta el commit
NO satisface este puerto. Consultarla antes y confiar despues en la copia no
constituye atomicidad y el modulo debe permanecer sin cablear.

```go
type RepositorioRecibosCargaDirecta interface {
	RegistrarReciboCargaDirecta(context.Context, RegistroReciboCargaDirecta) (ResultadoRegistroReciboCargaDirecta, error)
	ConsumirReciboCargaDirecta(context.Context, OrdenConsumoReciboCargaDirecta) (ResultadoConsumoReciboCargaDirecta, error)
}
```

RepositorioRecibosCargaDirecta es un puerto saliente durable. Registrar
conserva la unicidad permanente de IndiceHMAC, fija RegistradoEn con su
reloj durable y, en la misma transaccion, sustituye el recibo activo
anterior del GrupoHMAC. Debe conservar el enlace tipado al indice y
autorizacion del predecesor; el anterior permanece inactivo y no se
reactiva. Consumir exige indice, grupo, vinculo, intencion y huella exactos,
que el recibo siga siendo el activo del grupo, que no se haya consumido
y que la hora transaccional durable cumpla RegistradoEn <= ahora <
min(ExpiraEn, ValidaHasta). En esa misma escritura desactiva el grupo,
crea la intencion pendiente y persiste evidencia y fecha. El resultado
devuelve RegistradoEn y ExpiraEn del registro, nunca fechas propuestas por
el proceso. El puerto no ofrece lectura, listado, reapertura ni borrado de
consumos.

Una colision de alta devuelve ErrRegistroReciboCargaDirectaConflicto.
Toda denegacion de consumo usa ErrConsumoReciboCargaDirectaDenegado.
Los demas errores representan indisponibilidad y el adaptador falla cerrado.

```go
type RepresentationPort interface {
	CheckRepresentation(context.Context, InteropRequest) (InteropResult, error)
}

type RequisitosAlmacenObjetos = almacencanonico.Requisitos

type RequisitosAnalizadorContenido struct {
	AnalisisEnFlujo        bool
	CanalAutenticado       bool
	CifradoEnTransito      bool
	IdentidadMutua         bool
	ActualizacionFirmas    bool
	DetectaMalware         bool
	DetectaContenidoActivo bool
	TamanoMinimo           int64
}

type ReservaCargaDocumental struct {
	Token    TokenReservaCargaDocumental
	Repetida bool
	Carga    domain.CargaDocumental
}
```

ReservaCargaDocumental devuelve exactamente uno de dos resultados:
token nuevo, o agregado previamente confirmado. Una reserva en curso ajena
se comunica como error y nunca se roba silenciosamente.

```go
func (r ReservaCargaDocumental) Validar() error

type ReservaDevolucionCobro struct {
	Token    TokenReservaDevolucionCobro
	Repetida bool
	Orden    *domain.OrdenCobro
}

func (r ReservaDevolucionCobro) Format(estado fmt.State, _ rune)

func (r ReservaDevolucionCobro) GoString() string

func (ReservaDevolucionCobro) MarshalJSON() ([]byte, error)

func (ReservaDevolucionCobro) String() string

func (r ReservaDevolucionCobro) Validar() error

type ReservaEmisionCodigoCotejo struct {
	Token    TokenReservaEmisionCodigoCotejo
	Repetida bool
	Codigo   domain.CodigoCotejo
}
```

ReservaEmisionCodigoCotejo devuelve un token nuevo o el agregado confirmado
anteriormente. Incluso en una repeticion el valor visible se recupera por el
protector; este contrato nunca lo persiste.

```go
type ReservaGeneracionDocumento struct {
	Token     TokenReservaGeneracionDocumento
	Repetida  bool
	Resultado domain.ResultadoGeneracionDocumento
}
```

ReservaGeneracionDocumento tiene dos resultados excluyentes: una reserva
nueva con Token, o el resultado confirmado anteriormente con Repetida=true.

```go
type ReservaOrdenCobro struct {
	Token    TokenReservaOrdenCobro
	Repetida bool
	Orden    *domain.OrdenCobro
}

func (r ReservaOrdenCobro) Format(estado fmt.State, _ rune)

func (r ReservaOrdenCobro) GoString() string

func (ReservaOrdenCobro) MarshalJSON() ([]byte, error)

func (ReservaOrdenCobro) String() string

func (r ReservaOrdenCobro) Validar() error

type ResolutorRegistroContextoActorV1 interface {
	ResolverYRegistrarContextoActorV1(
		context.Context,
		SolicitudResolucionRegistroContextoActorV1,
	) (ConfirmacionRegistroContextoActorV1, error)
}
```

ResolutorRegistroContextoActorV1 es la unica frontera productiva para
resolver y dejar registrada una capacidad de actor. La implementacion debe
ejecutar ambas operaciones en una sola transaccion: no puede leer primero y
registrar despues ni devolver una capacidad si el registro durable falla.

La operacion, la cuenta, el perfil, el metodo y la garantia son entradas
exactas. La implementacion no los normaliza, completa ni sustituye.
SolicitadoEn no reemplaza al reloj autoritativo del adaptador. Tras
adquirir todos los bloqueos, la implementacion debe releer cuenta, perfil,
persona y vinculos; obtener clock_timestamp() despues de esos bloqueos;
exigir que no sea anterior a SolicitadoEn ni lo supere en la ventana maxima;
comprobar toda la vigencia en ese instante; y solo entonces registrar y
confirmar en la misma transaccion. Antes del commit debe conservar los
bytes de RepresentacionCanonicaVinculadaV1 y su huella SHA-256, ligados a la
version resuelta. Una espera que alcance caducidad o exceda la ventana debe
abortar.

operacion_ref es clave idempotente. Ante un resultado de COMMIT ambiguo,
el adaptador consulta de nuevo por esa referencia y solo devuelve la fila
si la solicitud, el recibo, los bytes y la huella coinciden exactamente.
Una colision con otro contenido falla cerrada. Si la ausencia queda
confirmada, la misma invocacion puede reintentarse con la misma referencia;
nunca con una nueva. RegistroContextoRef pertenece a un espacio CSPRNG rca_
independiente.

```go
type RespuestaCrudaReconciliacionDocumentalV4 struct {
	// Has unexported fields.
}

func NuevaRespuestaCrudaReconciliacionDocumentalV4(
	consulta ConsultaReconciliacionDocumentalV4,
	declaracion DeclaracionRespuestaReconciliacionDocumentalV4,
	atestacion AtestacionCrudaReconciliacionDocumentalV4,
) (RespuestaCrudaReconciliacionDocumentalV4, error)

func (r RespuestaCrudaReconciliacionDocumentalV4) AtestacionCruda() (
	AtestacionCrudaReconciliacionDocumentalV4,
	error,
)

func (r RespuestaCrudaReconciliacionDocumentalV4) Declaracion() (
	DeclaracionRespuestaReconciliacionDocumentalV4,
	error,
)

func (r RespuestaCrudaReconciliacionDocumentalV4) Format(estado fmt.State, _ rune)

func (r RespuestaCrudaReconciliacionDocumentalV4) GoString() string

func (r RespuestaCrudaReconciliacionDocumentalV4) HuellaMensajeSHA256() (string, error)

func (r RespuestaCrudaReconciliacionDocumentalV4) LogValue() slog.Value

func (RespuestaCrudaReconciliacionDocumentalV4) MarshalBinary() ([]byte, error)

func (RespuestaCrudaReconciliacionDocumentalV4) MarshalJSON() ([]byte, error)

func (RespuestaCrudaReconciliacionDocumentalV4) MarshalText() ([]byte, error)

func (r RespuestaCrudaReconciliacionDocumentalV4) MensajeCanonico() ([]byte, error)

func (RespuestaCrudaReconciliacionDocumentalV4) String() string

func (*RespuestaCrudaReconciliacionDocumentalV4) UnmarshalBinary([]byte) error

func (*RespuestaCrudaReconciliacionDocumentalV4) UnmarshalJSON([]byte) error

func (*RespuestaCrudaReconciliacionDocumentalV4) UnmarshalText([]byte) error

func (r RespuestaCrudaReconciliacionDocumentalV4) ValidarSintaxisContra(
	consulta ConsultaReconciliacionDocumentalV4,
) error
```

ValidarSintaxisContra no verifica COSE ni convierte la declaracion en hecho.

```go
type ResultadoAnalisisContenido struct {
	Objeto                ReferenciaObjetoAlmacen
	ConectorAlmacenID     string
	HuellaObjetoSHA256    string
	TamanoObjeto          int64
	MIMEDeclarado         string
	MIMEDetectado         string
	ConectorAnalizadorID  string
	VersionConector       int
	MotorRef              string
	VersionMotor          string
	FirmasRef             string
	Estado                EstadoAnalisisContenido
	CodigoResultado       string
	Detecciones           []DeteccionContenido
	BytesAnalizados       int64
	EvidenciaRef          string
	HuellaEvidenciaSHA256 string
	AnalisisIniciadoEn    time.Time
	AnalisisCompletadoEn  time.Time
	CorrelacionRef        string
	AutorizacionRef       string
	Finalidad             string
	Clasificacion         string
}
```

ResultadoAnalisisContenido es evidencia tecnica normalizada y acotada.
Un error o resultado no concluyente nunca equivale a limpio.

```go
func (r ResultadoAnalisisContenido) Validar() error

func (r ResultadoAnalisisContenido) ValidarContra(s SolicitudAnalizarContenido) error

type ResultadoConciliacionCobro = pagoscanonicos.ResultadoConciliacionCobro

type ResultadoConectorEjecucionDocumentalAtestadaV4 struct {
	// Has unexported fields.
}
```

ResultadoConectorEjecucionDocumentalAtestadaV4 es una confirmacion opaca
y no una capacidad reutilizable. Solo conserva referencias operativas;
nunca transporta COSE, payload, identidad personal, secreto ni credencial.

```go
func NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
	ordenRef, estado, auditoriaRef, eventoOutboxRef string,
	registradaEn time.Time,
) (ResultadoConectorEjecucionDocumentalAtestadaV4, error)

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) AuditoriaRef() (string, error)

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) Estado() (string, error)

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) EventoOutboxRef() (string, error)

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) Format(estado fmt.State, _ rune)

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) GoString() string

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) LogValue() slog.Value

func (ResultadoConectorEjecucionDocumentalAtestadaV4) MarshalBinary() ([]byte, error)

func (ResultadoConectorEjecucionDocumentalAtestadaV4) MarshalJSON() ([]byte, error)

func (ResultadoConectorEjecucionDocumentalAtestadaV4) MarshalText() ([]byte, error)

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) OrdenRef() (string, error)

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) RegistradaEn() (time.Time, error)

func (ResultadoConectorEjecucionDocumentalAtestadaV4) String() string

func (*ResultadoConectorEjecucionDocumentalAtestadaV4) UnmarshalBinary([]byte) error

func (*ResultadoConectorEjecucionDocumentalAtestadaV4) UnmarshalJSON([]byte) error

func (*ResultadoConectorEjecucionDocumentalAtestadaV4) UnmarshalText([]byte) error

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) Validar() error

type ResultadoConsultaEfectoDocumentalV3Crudo struct {
	ReservaRef             string
	EfectoRef              string
	SecuenciaCercado       uint64
	HuellaVinculoSHA256    string
	HuellaPlanSHA256       string
	Estado                 EstadoResultadoReconciliacionDocumentalV3
	Resultado              ResultadoEfectoRenderizadoDocumentalV3Crudo
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	SobreAtestacion        SobreAtestacionReconciliacionDocumentalV3Crudo
	ConsultadaEn           time.Time
}

func (r ResultadoConsultaEfectoDocumentalV3Crudo) Format(estado fmt.State, _ rune)

func (r ResultadoConsultaEfectoDocumentalV3Crudo) GoString() string

func (r ResultadoConsultaEfectoDocumentalV3Crudo) LogValue() slog.Value

func (ResultadoConsultaEfectoDocumentalV3Crudo) MarshalBinary() ([]byte, error)

func (ResultadoConsultaEfectoDocumentalV3Crudo) MarshalJSON() ([]byte, error)

func (ResultadoConsultaEfectoDocumentalV3Crudo) MarshalText() ([]byte, error)

func (ResultadoConsultaEfectoDocumentalV3Crudo) String() string

func (*ResultadoConsultaEfectoDocumentalV3Crudo) UnmarshalBinary([]byte) error

func (*ResultadoConsultaEfectoDocumentalV3Crudo) UnmarshalJSON([]byte) error

func (*ResultadoConsultaEfectoDocumentalV3Crudo) UnmarshalText([]byte) error

func (r ResultadoConsultaEfectoDocumentalV3Crudo) ValidarContra(
	s SolicitudConsultarEfectoDocumentalV3,
) error

type ResultadoConsultaFuenteAutoridad string

func (r ResultadoConsultaFuenteAutoridad) Valido() bool

type ResultadoConsultaInternaFuenteAutoridad struct {
	Encontrada bool
	Fuente     domain.FuenteAutoridadVersionada
	Estado     ReferenciaEstadoFuenteAutoridad
	Recibo     ReciboConsultaInternaFuenteAutoridad
	// Has unexported fields.
}

func (r ResultadoConsultaInternaFuenteAutoridad) ClonarPara(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
) (ResultadoConsultaInternaFuenteAutoridad, error)

func (b ResultadoConsultaInternaFuenteAutoridad) Format(estado fmt.State, _ rune)

func (b ResultadoConsultaInternaFuenteAutoridad) GoString() string

func (*ResultadoConsultaInternaFuenteAutoridad) GobDecode([]byte) error

func (ResultadoConsultaInternaFuenteAutoridad) GobEncode() ([]byte, error)

func (b ResultadoConsultaInternaFuenteAutoridad) LogValue() slog.Value

func (ResultadoConsultaInternaFuenteAutoridad) MarshalBinary() ([]byte, error)

func (ResultadoConsultaInternaFuenteAutoridad) MarshalCBOR() ([]byte, error)

func (ResultadoConsultaInternaFuenteAutoridad) MarshalJSON() ([]byte, error)

func (ResultadoConsultaInternaFuenteAutoridad) MarshalText() ([]byte, error)

func (ResultadoConsultaInternaFuenteAutoridad) MarshalXML(*xml.Encoder, xml.StartElement) error

func (ResultadoConsultaInternaFuenteAutoridad) MarshalYAML() (any, error)

func (ResultadoConsultaInternaFuenteAutoridad) String() string

func (*ResultadoConsultaInternaFuenteAutoridad) UnmarshalBinary([]byte) error

func (*ResultadoConsultaInternaFuenteAutoridad) UnmarshalCBOR([]byte) error

func (*ResultadoConsultaInternaFuenteAutoridad) UnmarshalJSON([]byte) error

func (*ResultadoConsultaInternaFuenteAutoridad) UnmarshalText([]byte) error

func (*ResultadoConsultaInternaFuenteAutoridad) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*ResultadoConsultaInternaFuenteAutoridad) UnmarshalYAML(func(any) error) error

func (r ResultadoConsultaInternaFuenteAutoridad) ValidarPara(
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
) error

type ResultadoConsumoReciboCargaDirecta = almacencanonico.ResultadoConsumoReciboCargaDirecta
```

ResultadoConsumoReciboCargaDirecta acredita la escritura condicional del
repositorio. RegistradoEn, ConsumidoEn y ExpiraEn proceden del registro
durable, no de datos propuestos por el proceso. Todos los identificadores
deben coincidir exactamente con la orden.

```go
type ResultadoCrudoVerificacionOrdenDespachoDocumentalV3 struct {
	// Has unexported fields.
}
```

ResultadoCrudoVerificacionOrdenDespachoDocumentalV3 es la respuesta nominal
del conector KMS. Su fabrica publica permite restaurarla y, por eso,
nunca es autoridad aislada.

```go
func NuevoResultadoCrudoVerificacionOrdenDespachoDocumentalV3(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	comprobacionRef, evidenciaOperacionRef string,
	huellaAtestacionSHA256 string,
	comprobadaEn time.Time,
) (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3, error)

func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) DatosCrudos() (
	DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3,
	error,
)
```

DatosCrudos devuelve una copia agrupada para adaptadores KMS y de
persistencia. La operacion solo valida forma nominal; el servicio de
aplicacion conserva la obligacion de llamar ValidarPara y efectuar la
relectura/CAS dentro de su frontera privada.

```go
func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) Format(estado fmt.State, _ rune)

func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) GoString() string

func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) LogValue() slog.Value

func (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error)

func (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error)

func (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) MarshalText() ([]byte, error)

func (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) String() string

func (*ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error

func (*ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error

func (*ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalText([]byte) error

func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) ValidarPara(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
) error

type ResultadoDevolucionCobro = pagoscanonicos.ResultadoDevolucionCobro

type ResultadoEfectoRenderizadoDocumentalV3Crudo struct {
	BorradorRef           string
	EfectoRef             string
	ContenidoRef          string
	ContenidoVersion      string
	ConectorRef           string
	MIME                  string
	HuellaSalidaSHA256    string
	TamanoSalida          uint64
	EvidenciaOperacionRef string
}
```

ResultadoEfectoRenderizadoDocumentalV3Crudo conserva una referencia exacta y
versionada del objeto; nunca una URL temporal ni los bytes del documento.

```go
func (r ResultadoEfectoRenderizadoDocumentalV3Crudo) Format(estado fmt.State, _ rune)

func (r ResultadoEfectoRenderizadoDocumentalV3Crudo) GoString() string

func (r ResultadoEfectoRenderizadoDocumentalV3Crudo) LogValue() slog.Value

func (ResultadoEfectoRenderizadoDocumentalV3Crudo) MarshalBinary() ([]byte, error)

func (ResultadoEfectoRenderizadoDocumentalV3Crudo) MarshalJSON() ([]byte, error)

func (ResultadoEfectoRenderizadoDocumentalV3Crudo) MarshalText() ([]byte, error)

func (ResultadoEfectoRenderizadoDocumentalV3Crudo) String() string

func (*ResultadoEfectoRenderizadoDocumentalV3Crudo) UnmarshalBinary([]byte) error

func (*ResultadoEfectoRenderizadoDocumentalV3Crudo) UnmarshalJSON([]byte) error

func (*ResultadoEfectoRenderizadoDocumentalV3Crudo) UnmarshalText([]byte) error

func (r ResultadoEfectoRenderizadoDocumentalV3Crudo) ValidarContra(
	manifiesto ManifiestoEjecucionDocumentalV3,
) error

type ResultadoEjecucionComponenteDocumental string

const (
	ResultadoRenderizadoDocumentalCorrecto  ResultadoEjecucionComponenteDocumental = "renderizado_correcto"
	ResultadoEstructuraDocumentalConforme   ResultadoEjecucionComponenteDocumental = "estructura_conforme"
	ResultadoSemanticaDocumentalEquivalente ResultadoEjecucionComponenteDocumental = "semantica_equivalente"
)
type ResultadoExtraccionMetadatoInstitucional struct {
	Metadato                domain.MarcaInstitucionalDocumento
	HuellaContenidoSHA256   string
	DigestConformidadSHA256 string
}

func (r ResultadoExtraccionMetadatoInstitucional) ValidarContra(
	perfil domain.PerfilFormatoDocumental,
	contenido []byte,
) error

type ResultadoFirmaAtestacionAutorizacionV1 struct {
	// Has unexported fields.
}
```

ResultadoFirmaAtestacionAutorizacionV1 conserva solo la firma y evidencia
opaca del proveedor. No puede sustituir la cabecera preseleccionada.

```go
func NuevoResultadoFirmaAtestacionAutorizacionV1(
	solicitud SolicitudFirmaAtestacionAutorizacionV1,
	firma []byte,
	evidenciaOperacionRef string,
	firmadaEn time.Time,
) (ResultadoFirmaAtestacionAutorizacionV1, error)

func (r ResultadoFirmaAtestacionAutorizacionV1) EvidenciaOperacionRef() (string, error)

func (r ResultadoFirmaAtestacionAutorizacionV1) Firma() ([]byte, error)

func (r ResultadoFirmaAtestacionAutorizacionV1) FirmadaEn() (time.Time, error)

func (r ResultadoFirmaAtestacionAutorizacionV1) HuellaMensajeSHA256() (string, error)

func (r ResultadoFirmaAtestacionAutorizacionV1) Validar() error

func (r ResultadoFirmaAtestacionAutorizacionV1) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV1,
) error

type ResultadoFirmaAtestacionAutorizacionV2 struct {
	// Has unexported fields.
}
```

ResultadoFirmaAtestacionAutorizacionV2 conserva la salida opaca del
proveedor y la liga a una unica solicitud VEC-AD-2 mediante su huella.
Verificar el perfil criptografico sigue siendo responsabilidad del adaptador
privado de confianza que consuma esta salida.

```go
func NuevoResultadoFirmaAtestacionAutorizacionV2(
	solicitud SolicitudFirmaAtestacionAutorizacionV2,
	firma []byte,
	evidenciaOperacionRef string,
	firmadaEn time.Time,
) (ResultadoFirmaAtestacionAutorizacionV2, error)

func (r ResultadoFirmaAtestacionAutorizacionV2) EvidenciaOperacionRef() (string, error)

func (r ResultadoFirmaAtestacionAutorizacionV2) Firma() ([]byte, error)

func (r ResultadoFirmaAtestacionAutorizacionV2) FirmadaEn() (time.Time, error)

func (b ResultadoFirmaAtestacionAutorizacionV2) Format(estado fmt.State, _ rune)

func (b ResultadoFirmaAtestacionAutorizacionV2) GoString() string

func (*ResultadoFirmaAtestacionAutorizacionV2) GobDecode([]byte) error

func (ResultadoFirmaAtestacionAutorizacionV2) GobEncode() ([]byte, error)

func (r ResultadoFirmaAtestacionAutorizacionV2) HuellaMensajeSHA256() (string, error)

func (b ResultadoFirmaAtestacionAutorizacionV2) LogValue() slog.Value

func (ResultadoFirmaAtestacionAutorizacionV2) MarshalBinary() ([]byte, error)

func (ResultadoFirmaAtestacionAutorizacionV2) MarshalCBOR() ([]byte, error)

func (ResultadoFirmaAtestacionAutorizacionV2) MarshalJSON() ([]byte, error)

func (ResultadoFirmaAtestacionAutorizacionV2) MarshalText() ([]byte, error)

func (ResultadoFirmaAtestacionAutorizacionV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ResultadoFirmaAtestacionAutorizacionV2) MarshalYAML() (any, error)

func (ResultadoFirmaAtestacionAutorizacionV2) String() string

func (*ResultadoFirmaAtestacionAutorizacionV2) UnmarshalBinary([]byte) error

func (*ResultadoFirmaAtestacionAutorizacionV2) UnmarshalCBOR([]byte) error

func (*ResultadoFirmaAtestacionAutorizacionV2) UnmarshalJSON([]byte) error

func (*ResultadoFirmaAtestacionAutorizacionV2) UnmarshalText([]byte) error

func (*ResultadoFirmaAtestacionAutorizacionV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*ResultadoFirmaAtestacionAutorizacionV2) UnmarshalYAML(func(any) error) error

func (r ResultadoFirmaAtestacionAutorizacionV2) Validar() error

func (r ResultadoFirmaAtestacionAutorizacionV2) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV2,
) error

type ResultadoIntentoCASReconciliacionDocumentalV4 string

const (
	ResultadoIntentoCASReconciliacionDocumentalV4Aplicado  ResultadoIntentoCASReconciliacionDocumentalV4 = "aplicado"
	ResultadoIntentoCASReconciliacionDocumentalV4Conflicto ResultadoIntentoCASReconciliacionDocumentalV4 = "conflicto"
)
func (r ResultadoIntentoCASReconciliacionDocumentalV4) SoloPermiteRegistrarEvidencia() bool
```

SoloPermiteRegistrarEvidencia es cierto para todo resultado reconocido: que
el registro haya aplicado su operacion o detectado conflicto no convierte la
afirmacion remota en permiso para confirmar o abandonar el efecto.

```go
func (r ResultadoIntentoCASReconciliacionDocumentalV4) Valido() bool

type ResultadoMetadatoInstitucional struct {
	Contenido            []byte
	HuellaFinalSHA256    string
	HuellaMetadatoSHA256 string
	PerfilRef            domain.ReferenciaPerfilDocumental
	Conector             domain.ReferenciaConectorDocumental
}

func (r ResultadoMetadatoInstitucional) ValidarContra(
	solicitud SolicitudIncorporarMetadatoInstitucional,
) error
```

ValidarContra modela exclusivamente incorporacion embebida mediante el
mecanismo estandar del perfil, por eso los bytes finales deben cambiar.
Un formato sin metadato estandar queda cerrado hasta disponer de un puerto
distinto de manifiesto lateral; nunca se simula con contenido invisible.

```go
type ResultadoOperacionCobro = pagoscanonicos.ResultadoOperacionCobro

type ResultadoOperacionObjeto struct {
	Objeto    ObjetoAlmacenado
	Evidencia EvidenciaOperacionAlmacen
}

func (r ResultadoOperacionObjeto) Validar() error

func (r ResultadoOperacionObjeto) ValidarCargaDirecta(
	preparacion SolicitudPrepararCargaDirecta,
	confirmacion SolicitudConfirmarCargaDirecta,
	capacidades CapacidadesAlmacenObjetos,
) error
```

ValidarCargaDirecta comprueba la respuesta completa del conector contra la
preparacion y la autorizacion de confirmacion. La huella declarada por el
navegador solo se acepta si coincide con la calculada por el almacen sobre
el objeto efectivamente recibido.

```go
func (r ResultadoOperacionObjeto) ValidarEscritura(
	solicitud SolicitudEscribirObjeto,
	capacidades CapacidadesAlmacenObjetos,
) error
```

ValidarEscritura coteja una escritura en flujo, incluida una repeticion
idempotente inequívocamente marcada, con la solicitud exacta.

```go
func (r ResultadoOperacionObjeto) ValidarInmovilizacion(
	solicitud SolicitudInmovilizarObjeto,
	anterior ObjetoAlmacenado,
) error

func (r ResultadoOperacionObjeto) ValidarLevantamientoInmovilizacion(
	solicitud SolicitudLevantarInmovilizacionObjeto,
	anterior ObjetoAlmacenado,
) error

func (r ResultadoOperacionObjeto) ValidarPromocion(
	solicitud SolicitudPromoverObjeto,
	origen ObjetoAlmacenado,
	capacidades CapacidadesAlmacenObjetos,
) error
```

ValidarPromocion impide que una respuesta sustituya el contenido analizado
por otro. En perfiles que preservan el original, la zona admitida debe usar
una referencia distinta y conservar exactamente huella, tamano y MIME.

```go
func (r ResultadoOperacionObjeto) ValidarRetencion(
	solicitud SolicitudRetenerObjeto,
	anterior ObjetoAlmacenado,
) error
```

ValidarRetencion exige que el conector solo modifique la fecha de retencion,
nunca acorte una vigente ni altere los bytes o su custodia.

```go
type ResultadoReferenciaReciboMaterialV2 struct {
	// Has unexported fields.
}
```

ResultadoReferenciaReciboMaterialV2 es una capacidad opaca ligada a la
solicitud exacta. Su forma no acredita durabilidad: la fabrica exige ademas
un verificador autoritativo del registro.

```go
func NuevoResultadoReferenciaReciboMaterialV2(
	solicitud SolicitudReservarReferenciaReciboMaterialV2,
	referencia string,
) (ResultadoReferenciaReciboMaterialV2, error)

func (ResultadoReferenciaReciboMaterialV2) Format(e fmt.State, _ rune)

func (ResultadoReferenciaReciboMaterialV2) GoString() string

func (ResultadoReferenciaReciboMaterialV2) LogValue() slog.Value

func (ResultadoReferenciaReciboMaterialV2) MarshalBinary() ([]byte, error)

func (ResultadoReferenciaReciboMaterialV2) MarshalJSON() ([]byte, error)

func (ResultadoReferenciaReciboMaterialV2) MarshalText() ([]byte, error)

func (ResultadoReferenciaReciboMaterialV2) String() string

func (*ResultadoReferenciaReciboMaterialV2) UnmarshalBinary([]byte) error

func (*ResultadoReferenciaReciboMaterialV2) UnmarshalJSON([]byte) error

func (*ResultadoReferenciaReciboMaterialV2) UnmarshalText([]byte) error

type ResultadoRegistroReciboCargaDirecta = almacencanonico.ResultadoRegistroReciboCargaDirecta
```

ResultadoRegistroReciboCargaDirecta acredita la fecha del alta durable y,
cuando procede, la relacion con el predecesor del mismo grupo sustituido en
la misma transaccion.

```go
type ResultadoReservaEfectoGeneracionDocumental struct {
	ReservaRef             string
	EfectoRef              string
	HuellaDecisionSHA256   string
	HuellaPlanEfectoSHA256 string
	HuellaManifiestoSHA256 string
	Repetida               bool
	Pasos                  []EstadoPasoDuraderoGeneracionDocumental
}

func (r ResultadoReservaEfectoGeneracionDocumental) ValidarContra(
	solicitud SolicitudReservarEfectoGeneracionDocumental,
) error

type ResultadoVerificacionPlanMaterialAlmacenV2 struct {
	// Has unexported fields.
}

func NuevoResultadoVerificacionPlanMaterialAlmacenV2(
	solicitud SolicitudVerificarPlanMaterialAlmacenV2,
	huellaPlanHexadecimal string,
) (ResultadoVerificacionPlanMaterialAlmacenV2, error)

func (ResultadoVerificacionPlanMaterialAlmacenV2) Format(e fmt.State, _ rune)

func (ResultadoVerificacionPlanMaterialAlmacenV2) GoString() string

func (ResultadoVerificacionPlanMaterialAlmacenV2) LogValue() slog.Value

func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error)

func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error)

func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalText() ([]byte, error)

func (ResultadoVerificacionPlanMaterialAlmacenV2) String() string

func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error

func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error

func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalText([]byte) error

type RetoConsultaReconciliacionDocumentalV4 struct {
	// Has unexported fields.
}
```

RetoConsultaReconciliacionDocumentalV4 es un valor nominal, opaco y sin
autoridad. Este paquete solo comprueba tamano y forma. Su generacion
mediante crypto/rand y la prohibicion de reutilizarlo corresponden al
servicio de confianza alojado en application/internal.

```go
func NuevoRetoConsultaReconciliacionDocumentalV4(
	valor []byte,
) (RetoConsultaReconciliacionDocumentalV4, error)

func (r RetoConsultaReconciliacionDocumentalV4) BytesParaProtocolo() ([]byte, error)
```

BytesParaProtocolo entrega una copia exclusivamente para construir o
verificar el payload canonico. Los serializadores generales estan cerrados.

```go
func (r RetoConsultaReconciliacionDocumentalV4) Format(estado fmt.State, _ rune)

func (r RetoConsultaReconciliacionDocumentalV4) GoString() string

func (r RetoConsultaReconciliacionDocumentalV4) HuellaSHA256() (string, error)

func (r RetoConsultaReconciliacionDocumentalV4) LogValue() slog.Value

func (RetoConsultaReconciliacionDocumentalV4) MarshalBinary() ([]byte, error)

func (RetoConsultaReconciliacionDocumentalV4) MarshalJSON() ([]byte, error)

func (RetoConsultaReconciliacionDocumentalV4) MarshalText() ([]byte, error)

func (RetoConsultaReconciliacionDocumentalV4) String() string

func (*RetoConsultaReconciliacionDocumentalV4) UnmarshalBinary([]byte) error

func (*RetoConsultaReconciliacionDocumentalV4) UnmarshalJSON([]byte) error

func (*RetoConsultaReconciliacionDocumentalV4) UnmarshalText([]byte) error

func (r RetoConsultaReconciliacionDocumentalV4) ValidarSintaxis() error
```

ValidarSintaxis no certifica que el reto proceda de un CSPRNG.

```go
type RevalidadorAutenticacionActorV1 = domain.RevalidadorAutenticacionActorV1
```

RevalidadorAutenticacionActorV1 consulta registros autoritativos de sesion,
cuenta, superficie, garantia y controles de revocacion. No debe copiar esos
atributos de una peticion ni devolver una sesion almacenada sin revalidarla.
La comprobacion de sesion y cuentas debe ser atomica en el adaptador.

```go
type SalidaObservadaDocumental struct {
	// Has unexported fields.
}

func (s SalidaObservadaDocumental) Datos() (DatosSalidaObservadaDocumental, error)

func (s SalidaObservadaDocumental) Validar() error

func (s SalidaObservadaDocumental) ValidarContraSolicitudEscribirObjeto(
	solicitud SolicitudEscribirObjeto,
) error
```

ValidarContraSolicitudEscribirObjeto enlaza la observacion local con el
puerto de almacen ya existente. El sumidero no guarda objetos: aporta la
pareja exacta Tamano/HuellaSHA256 con la que SolicitudEscribirObjeto ejecuta
la escritura en flujo y aplica su contexto de autorizacion.

```go
type SecretoCodigoCotejo struct {
	// Has unexported fields.
}
```

SecretoCodigoCotejo impide que el CSV acabe accidentalmente en JSON, texto,
trazas o mensajes de formato. Revelar es la unica salida deliberada para el
renderizador del sello/QR o para el DTO inicial estrictamente autorizado.

```go
func NuevoSecretoCodigoCotejo(valor string) (SecretoCodigoCotejo, error)

func (s SecretoCodigoCotejo) Format(estado fmt.State, _ rune)

func (SecretoCodigoCotejo) GoString() string

func (SecretoCodigoCotejo) MarshalJSON() ([]byte, error)

func (SecretoCodigoCotejo) MarshalText() ([]byte, error)

func (s SecretoCodigoCotejo) Revelar() string

func (SecretoCodigoCotejo) String() string

func (s SecretoCodigoCotejo) Validar() error

type SecureNetworkPort interface {
	CheckConnectivity(context.Context, InteropRequest) (InteropResult, error)
}

type SeleccionPlanMaterialAlmacenV2 struct {
	// Has unexported fields.
}
```

SeleccionPlanMaterialAlmacenV2 identifica un plan publicado. No contiene su
huella: esta solo puede proceder del verificador autoritativo del registro.

```go
func NuevaSeleccionPlanMaterialAlmacenV2(
	referencia string,
	version uint32,
) (SeleccionPlanMaterialAlmacenV2, error)

func (SeleccionPlanMaterialAlmacenV2) Format(e fmt.State, _ rune)

func (SeleccionPlanMaterialAlmacenV2) GoString() string

func (SeleccionPlanMaterialAlmacenV2) LogValue() slog.Value

func (SeleccionPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error)

func (SeleccionPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error)

func (SeleccionPlanMaterialAlmacenV2) MarshalText() ([]byte, error)

func (SeleccionPlanMaterialAlmacenV2) String() string
```

Los metodos permanecen declarados sobre cada tipo para conservar su forma,
tamaño y reflexión. Las funciones comunes solo centralizan el resultado.

```go
func (*SeleccionPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error

func (*SeleccionPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error

func (*SeleccionPlanMaterialAlmacenV2) UnmarshalText([]byte) error

type SelectorOperacionFuenteAutoridad struct {
	OperacionRef string
}
```

SelectorOperacionFuenteAutoridad permite validar una referencia recibida
antes de consultar el repositorio. No concede acceso ni acredita que la
operacion exista.

```go
func (s SelectorOperacionFuenteAutoridad) Validar() error

type SelectorVersionFuenteAutoridad struct {
	FuenteID string
	Version  uint64
}
```

SelectorVersionFuenteAutoridad identifica una fila exacta para tareas de
gobierno. No equivale a una referencia consumible: esta ultima incorpora
tambien la huella del contenido.

```go
func (s SelectorVersionFuenteAutoridad) Validar() error

type SelladorDatosDocumento interface {
	SellarDatos(context.Context, []byte) (string, error)
}
```

SelladorDatosDocumento calcula una huella autenticada (por ejemplo,
HMAC-SHA-256 con clave en KMS). No debe usarse SHA sin clave para campos de
baja entropia como DNI, porque permitiria ataques de diccionario.

```go
type SelladorIdempotenciaCarga interface {
	SellarIdempotenciaCarga(context.Context, SolicitudSellarIdempotenciaCarga) (string, error)
}
```

SelladorIdempotenciaCarga liga una clave de reintento a la operacion y
a todos sus datos exactos mediante una clave exclusiva del servidor.
El navegador nunca aporta directamente la clave aceptada por el almacen.

```go
type SelladorIndiceCodigoCotejo interface {
	SellarIndiceCodigoCotejo(context.Context, SecretoCodigoCotejo) (string, error)
	SellarIndicesConsultaCodigoCotejo(context.Context, SecretoCodigoCotejo) ([]string, error)
}
```

SelladorIndiceCodigoCotejo produce un indice determinista HMAC para buscar
un CSV sin conservarlo ni permitir ataques de diccionario sin la clave.

```go
type SelladorSolicitudCargaDocumental interface {
	SellarSolicitudCargaDocumental(context.Context, []byte) (string, error)
}
```

SelladorSolicitudCargaDocumental usa una clave distinta de la que indexa
reintentos. La huella identifica todos los datos con efecto de la orden sin
persistir la clave aportada por el navegador.

```go
type SelladorSolicitudCobro interface {
	// Cada operacion tiene un metodo distinto: el adaptador no puede elegir un
	// dominio libre ni reutilizar accidentalmente el de otra finalidad.
	SellarIndiceAltaCobro(context.Context, []byte) (string, error)
	SellarHuellaPeticionCobro(context.Context, []byte) (string, error)
	SellarIndiceDevolucionCobro(context.Context, []byte) (string, error)
}

type SelladorSolicitudCotejo interface {
	SellarSolicitudCotejo(context.Context, []byte) (string, error)
}
```

SelladorSolicitudCotejo tiene una clave y ciclo de rotacion independientes
del indice; evita que la idempotencia se convierta en un segundo indice.

```go
type SelladorSolicitudDocumento interface {
	SellarSolicitudDocumento(context.Context, []byte) (string, error)
}
```

SelladorSolicitudDocumento usa una clave criptografica separada y estable
durante la ventana de idempotencia. Separarlo del sellado de datos permite
rotar una clave sin cambiar silenciosamente el significado de otra.

```go
type SelladorVinculoSesionCarga interface {
	SellarVinculoSesionCarga(context.Context, string) (string, error)
}
```

SelladorVinculoSesionCarga usa una clave distinta de la idempotencia.
Su salida debe tener el formato hmac-sha256:<version>:<hex>.

```go
type SelloEvidenciaDocumentalV3Nominal struct {
	// Has unexported fields.
}

func NuevoSelloEvidenciaDocumentalV3Nominal(
	solicitud SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	firma []byte,
	evidenciaOperacionRef string,
	firmadoEn time.Time,
) (SelloEvidenciaDocumentalV3Nominal, error)

func (s SelloEvidenciaDocumentalV3Nominal) Datos() (
	DatosSelloEvidenciaDocumentalV3Crudos,
	error,
)

func (s SelloEvidenciaDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (s SelloEvidenciaDocumentalV3Nominal) GoString() string

func (s SelloEvidenciaDocumentalV3Nominal) LogValue() slog.Value

func (SelloEvidenciaDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (SelloEvidenciaDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (SelloEvidenciaDocumentalV3Nominal) MarshalText() ([]byte, error)

func (SelloEvidenciaDocumentalV3Nominal) String() string

func (*SelloEvidenciaDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*SelloEvidenciaDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*SelloEvidenciaDocumentalV3Nominal) UnmarshalText([]byte) error

func (s SelloEvidenciaDocumentalV3Nominal) ValidarPara(
	solicitud SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
) error

type SeudonimizadorSujetoAlmacen interface {
	SeudonimizarSujetoAlmacen(
		context.Context,
		SolicitudSeudonimizarSujetoAlmacen,
	) (string, error)
}
```

SeudonimizadorSujetoAlmacen debe usar una clave y version propias; no se
reutiliza la clave de sesiones, idempotencia ni huellas de solicitudes.

```go
type SignaturePort interface {
	Sign(context.Context, domain.Principal, domain.SignRequest) (domain.SignReceipt, error)
	VerifySignature(context.Context, string) (domain.SignVerification, error)
}

type SignatureValidationPort interface {
	ValidateSignature(context.Context, InteropRequest) (InteropResult, error)
}

type SobreAtestacionReconciliacionDocumentalV3Crudo struct {
	// Has unexported fields.
}

func NuevoSobreAtestacionReconciliacionDocumentalV3Crudo(
	coseSign1 []byte,
) (SobreAtestacionReconciliacionDocumentalV3Crudo, error)

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) COSESign1() ([]byte, error)

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) Format(estado fmt.State, _ rune)

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) GoString() string

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) HuellaSHA256() (string, error)

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) LogValue() slog.Value

func (SobreAtestacionReconciliacionDocumentalV3Crudo) MarshalBinary() ([]byte, error)

func (SobreAtestacionReconciliacionDocumentalV3Crudo) MarshalJSON() ([]byte, error)

func (SobreAtestacionReconciliacionDocumentalV3Crudo) MarshalText() ([]byte, error)

func (SobreAtestacionReconciliacionDocumentalV3Crudo) String() string

func (*SobreAtestacionReconciliacionDocumentalV3Crudo) UnmarshalBinary([]byte) error

func (*SobreAtestacionReconciliacionDocumentalV3Crudo) UnmarshalJSON([]byte) error

func (*SobreAtestacionReconciliacionDocumentalV3Crudo) UnmarshalText([]byte) error

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) Validar() error

type SobreCriptograficoDocumentalCrudoV4 struct {
	// Has unexported fields.
}
```

SobreCriptograficoDocumentalCrudoV4 transporta los bytes opacos de un
COSE_Sign1. Solo acredita limites y ausencia de alias mutables: no concede
autoridad, no interpreta cabeceras y no verifica ninguna firma.

```go
func NuevoSobreCriptograficoDocumentalCrudoV4(
	coseSign1 []byte,
) (SobreCriptograficoDocumentalCrudoV4, error)

func (s SobreCriptograficoDocumentalCrudoV4) COSESign1() ([]byte, error)
```

COSESign1 entrega una copia para su verificacion local. Es la unica salida
binaria deliberada; los serializadores generales permanecen bloqueados.

```go
func (s SobreCriptograficoDocumentalCrudoV4) Format(estado fmt.State, _ rune)

func (s SobreCriptograficoDocumentalCrudoV4) GoString() string

func (s SobreCriptograficoDocumentalCrudoV4) HuellaSHA256() (string, error)

func (s SobreCriptograficoDocumentalCrudoV4) LogValue() slog.Value

func (SobreCriptograficoDocumentalCrudoV4) MarshalBinary() ([]byte, error)

func (SobreCriptograficoDocumentalCrudoV4) MarshalJSON() ([]byte, error)

func (SobreCriptograficoDocumentalCrudoV4) MarshalText() ([]byte, error)

func (SobreCriptograficoDocumentalCrudoV4) String() string

func (*SobreCriptograficoDocumentalCrudoV4) UnmarshalBinary([]byte) error

func (*SobreCriptograficoDocumentalCrudoV4) UnmarshalJSON([]byte) error

func (*SobreCriptograficoDocumentalCrudoV4) UnmarshalText([]byte) error

func (s SobreCriptograficoDocumentalCrudoV4) ValidarSintaxis() error
```

ValidarSintaxis comprueba exclusivamente el contenedor crudo. Su exito no
significa que el contenido sea COSE valido ni que su firma sea confiable.

```go
type SobreReciboEjecucionDocumentalCrudo struct {
	// Has unexported fields.
}
```

SobreReciboEjecucionDocumentalCrudo conserva un COSE_Sign1 opaco.
La comprobacion criptografica se coordina dentro del servicio de aplicacion
privado; este valor solo impide alias, tamanos ilimitados y sobres vacios
evidentes.

```go
func NuevoSobreReciboEjecucionDocumentalCrudo(coseSign1 []byte) (SobreReciboEjecucionDocumentalCrudo, error)

func (s SobreReciboEjecucionDocumentalCrudo) COSESign1() ([]byte, error)

func (s SobreReciboEjecucionDocumentalCrudo) Format(estado fmt.State, _ rune)

func (s SobreReciboEjecucionDocumentalCrudo) GoString() string

func (s SobreReciboEjecucionDocumentalCrudo) HuellaSHA256() (string, error)

func (s SobreReciboEjecucionDocumentalCrudo) LogValue() slog.Value

func (SobreReciboEjecucionDocumentalCrudo) MarshalBinary() ([]byte, error)

func (SobreReciboEjecucionDocumentalCrudo) MarshalJSON() ([]byte, error)

func (SobreReciboEjecucionDocumentalCrudo) MarshalText() ([]byte, error)

func (SobreReciboEjecucionDocumentalCrudo) String() string

func (*SobreReciboEjecucionDocumentalCrudo) UnmarshalBinary([]byte) error

func (*SobreReciboEjecucionDocumentalCrudo) UnmarshalJSON([]byte) error

func (*SobreReciboEjecucionDocumentalCrudo) UnmarshalText([]byte) error

func (s SobreReciboEjecucionDocumentalCrudo) Validar() error

type SolicitudAbandonarEjecucionDocumentalV3 struct {
	ReservaRef      string
	Manifiesto      ManifiestoEjecucionDocumentalV3
	EstadoEsperado  EstadoEjecucionDocumentalV3
	ConsumoDecision ConsumoDecisionEjecucionDocumentalV3
	MotivoRef       string
	AbandonadaEn    time.Time
}

func (s SolicitudAbandonarEjecucionDocumentalV3) Validar() error

type SolicitudAbandonarReservaDevolucionCobro struct {
	Token               TokenReservaDevolucionCobro
	OrdenRef            string
	DevolucionRef       string
	PrincipalRef        string
	HuellaSolicitudHMAC string
}

func (s SolicitudAbandonarReservaDevolucionCobro) Validar() error

type SolicitudAbandonarReservaOrdenCobro struct {
	Token               TokenReservaOrdenCobro
	OrdenRef            string
	PrincipalRef        string
	HuellaSolicitudHMAC string
}

func (s SolicitudAbandonarReservaOrdenCobro) Validar() error

type SolicitudAbrirObjeto struct {
	Contexto ContextoOperacionAlmacen
	Objeto   ReferenciaObjetoAlmacen
	Zona     ZonaAlmacen
	Limite   int64
}

func (s SolicitudAbrirObjeto) Validar() error

type SolicitudActivarEjecucionDocumentalV3 struct {
	ReservaRef             string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	// OrdenConsumoDurableV4Ref es una referencia nominal, nunca autoridad.
	// El registro debe releer la orden V4 y adoptarla en su propio COMMIT.
	OrdenConsumoDurableV4Ref string
	ActivadaEn               time.Time
}

func (s SolicitudActivarEjecucionDocumentalV3) Format(estado fmt.State, _ rune)

func (s SolicitudActivarEjecucionDocumentalV3) GoString() string

func (s SolicitudActivarEjecucionDocumentalV3) LogValue() slog.Value

func (SolicitudActivarEjecucionDocumentalV3) MarshalBinary() ([]byte, error)

func (SolicitudActivarEjecucionDocumentalV3) MarshalJSON() ([]byte, error)

func (SolicitudActivarEjecucionDocumentalV3) MarshalText() ([]byte, error)

func (SolicitudActivarEjecucionDocumentalV3) String() string

func (*SolicitudActivarEjecucionDocumentalV3) UnmarshalBinary([]byte) error

func (*SolicitudActivarEjecucionDocumentalV3) UnmarshalJSON([]byte) error

func (*SolicitudActivarEjecucionDocumentalV3) UnmarshalText([]byte) error

func (s SolicitudActivarEjecucionDocumentalV3) Validar() error

func (s SolicitudActivarEjecucionDocumentalV3) VinculoEstable() (
	VinculoEstableActivacionDocumentalV3,
	error,
)

type SolicitudAnalizarContenido struct {
	Contexto          ContextoOperacionAlmacen
	Objeto            ReferenciaObjetoAlmacen
	ConectorAlmacenID string
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
	Contenido         io.Reader
}
```

SolicitudAnalizarContenido siempre apunta a la version exacta de un objeto
en cuarentena. El caso de uso abre el objeto desde AlmacenObjetos y entrega
un lector limitado; el navegador nunca invoca directamente al motor.

```go
func (s SolicitudAnalizarContenido) Validar() error

type SolicitudAplicacionAutorizacionEjecucionDocumentalV4 struct {
	// Has unexported fields.
}
```

SolicitudAplicacionAutorizacionEjecucionDocumentalV4 es una peticion opaca y
no autoritativa preparada para el registro atomico posterior. Un adaptador
no debe aceptarla como prueba autosuficiente: la autoridad interna que la
presenta y la evidencia deben revalidarse en la misma operacion duradera.

```go
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) CotejarConDecisionHistoricaAtestacionPDPV1(
	datos domain.DatosDecisionHistoricaAtestacionAutorizacionV1,
	preimagen PreimagenRecursoAutorizacionEjecucionDocumentalV4,
) error
```

CotejarConDecisionHistoricaAtestacionPDPV1 comprueba que la solicitud opaca
viva, la preimagen durable y la proyeccion nominal extraida del payload
VEC-AD-1 describen exactamente la misma aplicacion.

Este cotejo no verifica COSE, no consulta una raiz de confianza y no concede
autoridad. Solo debe invocarlo el caso de uso interno despues de verificar
la firma y antes de entregar la solicitud al registrador tecnico aislado.
Obligaciones permanece cerrado: exige lista y mapa de cumplimientos vacios,
con sus huellas canonicas. No se habilitaran obligaciones no vacias hasta
que el registro persista el mapa, evidencia tipada de cumplimiento y
revocacion.

```go
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) EvidenciaEstructural() (
	EvidenciaUsoDecisionAutorizacion,
	error,
)
```

EvidenciaEstructural devuelve la evidencia defensiva necesaria para que la
capa de aplicacion vuelva a comprobarla. No convierte la solicitud en una
concesion ni demuestra por si sola que la decision proceda del PDP.

```go
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) Format(estado fmt.State, _ rune)

func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) GoString() string

func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) LogValue() slog.Value

func (SolicitudAplicacionAutorizacionEjecucionDocumentalV4) MarshalJSON() ([]byte, error)

func (SolicitudAplicacionAutorizacionEjecucionDocumentalV4) MarshalText() ([]byte, error)

func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) PreimagenRecursoParaEvidenciaDurable() (
	PreimagenRecursoAutorizacionEjecucionDocumentalV4,
	error,
)
```

PreimagenRecursoParaEvidenciaDurable extrae una copia defensiva desde la
solicitud opaca viva. El llamador no puede aportar ni reemplazar el recurso.

```go
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) ProyeccionParaTransaccion() (
	ProyeccionAplicacionAutorizacionEjecucionDocumentalV4,
	error,
)

func (SolicitudAplicacionAutorizacionEjecucionDocumentalV4) String() string

func (*SolicitudAplicacionAutorizacionEjecucionDocumentalV4) UnmarshalJSON([]byte) error

func (*SolicitudAplicacionAutorizacionEjecucionDocumentalV4) UnmarshalText([]byte) error

func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) ValidarContraEn(
	decisionRef, huellaPlanSHA256, efectoRef string,
	instante time.Time,
) error

type SolicitudAplicarReconciliacionDocumentalV3 struct {
	Consulta          SolicitudConsultarEfectoDocumentalV3
	ResultadoConsulta ResultadoConsultaEfectoDocumentalV3Crudo
	TieneConfirmacion bool
	Confirmacion      SolicitudConfirmarEjecucionDocumentalV3
}

func (s SolicitudAplicarReconciliacionDocumentalV3) Validar() error

type SolicitudAtestarMaterialAlmacenV2 struct {
	// Has unexported fields.
}
```

SolicitudAtestarMaterialAlmacenV2 es la unica apertura de los bytes hacia el
adaptador criptografico. Entrega siempre una copia defensiva.

```go
func (SolicitudAtestarMaterialAlmacenV2) Format(e fmt.State, _ rune)

func (SolicitudAtestarMaterialAlmacenV2) GoString() string

func (SolicitudAtestarMaterialAlmacenV2) LogValue() slog.Value

func (SolicitudAtestarMaterialAlmacenV2) MarshalBinary() ([]byte, error)

func (SolicitudAtestarMaterialAlmacenV2) MarshalJSON() ([]byte, error)

func (SolicitudAtestarMaterialAlmacenV2) MarshalText() ([]byte, error)

func (s SolicitudAtestarMaterialAlmacenV2) RevelarParaAtestacion() (
	dominio string,
	mensaje []byte,
	huellaSHA256 [sha256.Size]byte,
	err error,
)
```

RevelarParaAtestacion devuelve el dominio separado, los bytes exactos y su
huella. No revela credenciales ni acepta que el adaptador cambie el mensaje.

```go
func (SolicitudAtestarMaterialAlmacenV2) String() string

func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalBinary([]byte) error

func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalJSON([]byte) error

func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalText([]byte) error

type SolicitudComprobarActoFuenteAutoridad struct {
	Solicitud      domain.SolicitudTransicionFuenteAutoridadV1
	EstadoEsperado ReferenciaEstadoFuenteAutoridad
	ComprobarEn    time.Time
	// Has unexported fields.
}
```

SolicitudComprobarActoFuenteAutoridad contiene la solicitud de dominio
exacta y el snapshot OCC que la produjo. El adaptador recibe este valor
desde aplicacion; nunca lo reconstruye con datos de un callback.

```go
func (s SolicitudComprobarActoFuenteAutoridad) Clonar() (
	SolicitudComprobarActoFuenteAutoridad,
	error,
)

func (s SolicitudComprobarActoFuenteAutoridad) Format(estado fmt.State, _ rune)

func (s SolicitudComprobarActoFuenteAutoridad) GoString() string

func (*SolicitudComprobarActoFuenteAutoridad) GobDecode([]byte) error

func (SolicitudComprobarActoFuenteAutoridad) GobEncode() ([]byte, error)

func (s SolicitudComprobarActoFuenteAutoridad) LogValue() slog.Value

func (SolicitudComprobarActoFuenteAutoridad) MarshalBinary() ([]byte, error)

func (SolicitudComprobarActoFuenteAutoridad) MarshalJSON() ([]byte, error)

func (SolicitudComprobarActoFuenteAutoridad) MarshalText() ([]byte, error)

func (SolicitudComprobarActoFuenteAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (SolicitudComprobarActoFuenteAutoridad) String() string

func (*SolicitudComprobarActoFuenteAutoridad) UnmarshalBinary([]byte) error

func (*SolicitudComprobarActoFuenteAutoridad) UnmarshalJSON([]byte) error

func (*SolicitudComprobarActoFuenteAutoridad) UnmarshalText([]byte) error

func (*SolicitudComprobarActoFuenteAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (s SolicitudComprobarActoFuenteAutoridad) Validar() error

type SolicitudComprobarOrdenDespachoDocumentalV3 struct {
	// Has unexported fields.
}

func NuevaSolicitudComprobarOrdenDespachoDocumentalV3(
	orden OrdenDespachoDocumentalV3Nominal,
	vinculo VinculoEstableActivacionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3Nominal,
) (SolicitudComprobarOrdenDespachoDocumentalV3, error)

func (s SolicitudComprobarOrdenDespachoDocumentalV3) Format(estado fmt.State, _ rune)

func (s SolicitudComprobarOrdenDespachoDocumentalV3) GoString() string

func (s SolicitudComprobarOrdenDespachoDocumentalV3) LogValue() slog.Value

func (SolicitudComprobarOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error)

func (SolicitudComprobarOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error)

func (SolicitudComprobarOrdenDespachoDocumentalV3) MarshalText() ([]byte, error)

func (s SolicitudComprobarOrdenDespachoDocumentalV3) MaterialCrudo() (
	MaterialCrudoVerificacionOrdenDespachoDocumentalV3,
	error,
)

func (s SolicitudComprobarOrdenDespachoDocumentalV3) Mensaje() ([]byte, error)

func (SolicitudComprobarOrdenDespachoDocumentalV3) String() string

func (*SolicitudComprobarOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error

func (*SolicitudComprobarOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error

func (*SolicitudComprobarOrdenDespachoDocumentalV3) UnmarshalText([]byte) error

func (s SolicitudComprobarOrdenDespachoDocumentalV3) Validar() error

type SolicitudConciliacionCobro = pagoscanonicos.SolicitudConciliacionCobro

type SolicitudConfirmarCargaDirecta struct {
	// Has unexported fields.
}
```

SolicitudConfirmarCargaDirecta solo puede construirse mediante la fabrica
verificada. Sus campos privados evitan que un conector reciba un comprobante
estructuralmente valido pero no atestado.

```go
func NuevaSolicitudConfirmarCargaDirecta(
	ctx context.Context,
	contexto ContextoOperacionAlmacen,
	sesionRef string,
	comprobante ComprobanteConsumoReciboCargaDirecta,
	verificador VerificadorAtestacionConsumoReciboCargaDirecta,
) (SolicitudConfirmarCargaDirecta, error)

func (s SolicitudConfirmarCargaDirecta) Format(estado fmt.State, _ rune)

func (SolicitudConfirmarCargaDirecta) GoString() string

func (s SolicitudConfirmarCargaDirecta) LogValue() slog.Value

func (SolicitudConfirmarCargaDirecta) MarshalJSON() ([]byte, error)

func (SolicitudConfirmarCargaDirecta) MarshalText() ([]byte, error)

func (s SolicitudConfirmarCargaDirecta) RevelarParaConector() (
	contexto ContextoOperacionAlmacen,
	sesionRef, intencionRef, huellaIntencionHMAC, evidenciaConsumoRef string,
	registradoEn, consumidoEn, expiraEn, validaHasta time.Time,
	err error,
)
```

RevelarParaConector entrega exclusivamente lo necesario al gestor de carga
directa despues de que la fabrica haya verificado la atestacion.

```go
func (SolicitudConfirmarCargaDirecta) String() string

func (s SolicitudConfirmarCargaDirecta) Validar() error

type SolicitudConfirmarCreacionOrdenCobro struct {
	Token                    TokenReservaOrdenCobro
	OrdenRef                 string
	PrincipalRef             string
	IndiceIdempotenciaHMAC   string
	HuellaSolicitudHMAC      string
	ReservaSolicitadaEn      time.Time
	ReservaExpiraEn          time.Time
	DecisionAutorizacionRef  string
	HuellaDecisionSHA256     string
	DecisionValidaHasta      time.Time
	HuellaEfectoSHA256       string
	EvidenciaAutorizacion    EvidenciaUsoDecisionAutorizacion
	ContextoAutorizacion     domain.ContextoAutorizacionCobro
	SesionRef                string
	HuellaSesionHMAC         string
	SesionValidaHasta        time.Time
	LiquidacionRef           string
	LiquidacionRevision      uint64
	LiquidacionHuellaSHA256  string
	LiquidacionEstado        EstadoControlLiquidacionCobro
	LiquidacionExigibleDesde time.Time
	LiquidacionExigibleHasta time.Time
	Mutacion                 MutacionOrdenCobro
}

func (s SolicitudConfirmarCreacionOrdenCobro) Validar() error

type SolicitudConfirmarEjecucionDocumentalV3 struct {
	ReservaRef             string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	OrdenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal
	Resultado              ResultadoEfectoRenderizadoDocumentalV3Crudo
	Recibos                RecibosEjecucionDocumentalV3
	Evidencia              DatosEvidenciaRenderizadoDocumentalV3
	Sello                  SelloEvidenciaDocumentalV3Nominal
}

func (s SolicitudConfirmarEjecucionDocumentalV3) Validar() error

type SolicitudConfirmarPasoGeneracionDocumental struct {
	ReservaRef string
	Contexto   ContextoOperacionAlmacen
	Guardado   ContenidoDocumentoGuardado
}
```

SolicitudConfirmarPasoGeneracionDocumental confirma el resultado tecnico
exacto de un paso ya reservado. No vuelve a consumir la decision: el
repositorio debe comprobar ReservaRef y la tupla de plan que consumio antes
del efecto remoto, y hacer la transicion de estado de manera condicional.

```go
func (s SolicitudConfirmarPasoGeneracionDocumental) Validar() error

type SolicitudConfirmarPreparacionCargaDocumental struct {
	Token        TokenReservaCargaDocumental
	Confirmacion ConfirmacionTransicionCargaDocumental
	Manifiesto   domain.ManifiestoPreparacionCargaDirectaV1
}
```

SolicitudConfirmarPreparacionCargaDocumental obliga al repositorio a
consumir la reserva y conservar el agregado, la auditoria, el outbox y el
manifiesto historico en una unica transaccion. Persistir cualquiera de esas
piezas por separado no satisface el contrato.

```go
func InstantaneaSolicitudConfirmarPreparacionCargaDocumental(
	solicitud SolicitudConfirmarPreparacionCargaDocumental,
) (SolicitudConfirmarPreparacionCargaDocumental, error)
```

InstantaneaSolicitudConfirmarPreparacionCargaDocumental copia una sola vez
toda la entrada mutable antes de validarla. Los adaptadores deben invocarla
antes de adquirir su bloqueo o transaccion y persistir exclusivamente la
copia devuelta; volver a leer la solicitud original reabriria un TOCTOU.

```go
func (s SolicitudConfirmarPreparacionCargaDocumental) Validar() error

type SolicitudConfirmarReservaDevolucionCobro struct {
	Token                TokenReservaDevolucionCobro
	HuellaSolicitudHMAC  string
	VersionEsperada      int
	HuellaEsperadaSHA256 string
	Mutacion             MutacionOrdenCobro
}

func (s SolicitudConfirmarReservaDevolucionCobro) Validar() error

type SolicitudConfirmarTransicionOrdenCobro struct {
	VersionEsperada      int
	HuellaEsperadaSHA256 string
	Mutacion             MutacionOrdenCobro
}

func (s SolicitudConfirmarTransicionOrdenCobro) Validar() error

type SolicitudConsultaInternaGobernadaFuenteAutoridad struct {
	// Has unexported fields.
}

func NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
	selector SelectorVersionFuenteAutoridad,
	autorizacion EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	auditoria domain.AuditEntry,
	motivoCatalogo domain.ReferenciaEntradaCatalogo,
	correlacion domain.ReferenciaCorrelacionAutorizacionV2,
	solicitadaEn time.Time,
) (SolicitudConsultaInternaGobernadaFuenteAutoridad, error)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Auditoria() (domain.AuditEntry, error)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Autorizacion() (
	EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	error,
)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Correlacion() (
	domain.ReferenciaCorrelacionAutorizacionV2,
	error,
)
```

Correlacion conserva la capacidad nominal mientras la operacion permanece
dentro del nucleo. CorrelacionRef revela el valor solo al adaptador durable.

```go
func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) CorrelacionRef() (string, error)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Format(estado fmt.State, _ rune)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) GoString() string

func (*SolicitudConsultaInternaGobernadaFuenteAutoridad) GobDecode([]byte) error

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) GobEncode() ([]byte, error)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) LogValue() slog.Value

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) MarshalBinary() ([]byte, error)

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) MarshalCBOR() ([]byte, error)

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) MarshalJSON() ([]byte, error)

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) MarshalText() ([]byte, error)

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) MarshalXML(*xml.Encoder, xml.StartElement) error

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) MarshalYAML() (any, error)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) MotivoCatalogo() (
	domain.ReferenciaEntradaCatalogo,
	error,
)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) Selector() (
	SelectorVersionFuenteAutoridad,
	error,
)

func (s SolicitudConsultaInternaGobernadaFuenteAutoridad) SolicitadaEn() (time.Time, error)

func (SolicitudConsultaInternaGobernadaFuenteAutoridad) String() string

func (*SolicitudConsultaInternaGobernadaFuenteAutoridad) UnmarshalBinary([]byte) error

func (*SolicitudConsultaInternaGobernadaFuenteAutoridad) UnmarshalCBOR([]byte) error

func (*SolicitudConsultaInternaGobernadaFuenteAutoridad) UnmarshalJSON([]byte) error

func (*SolicitudConsultaInternaGobernadaFuenteAutoridad) UnmarshalText([]byte) error

func (*SolicitudConsultaInternaGobernadaFuenteAutoridad) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (*SolicitudConsultaInternaGobernadaFuenteAutoridad) UnmarshalYAML(func(any) error) error

type SolicitudConsultarEfectoDocumentalV3 struct {
	ReservaRef             string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	OrdenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal
	SolicitadaEn           time.Time
}

func (s SolicitudConsultarEfectoDocumentalV3) Validar() error

type SolicitudConsumirReciboCargaDirecta struct {
	Contexto  ContextoOperacionAlmacen
	SesionRef string
	Recibo    ReciboCargaDirecta
	// ValidaHasta es el menor limite entre la autorizacion vigente y la
	// sesion de carga. El repositorio debe cotejarlo con su hora durable en
	// la misma escritura condicional que consume el recibo.
	ValidaHasta time.Time
}
```

SolicitudConsumirReciboCargaDirecta se ejecuta antes de confirmar en el
almacen. El adaptador debe verificar la MAC, caducidad y consumo atomico de
un uso. Una consulta repetida siempre falla cerrada.

```go
func (s SolicitudConsumirReciboCargaDirecta) Format(estado fmt.State, _ rune)

func (SolicitudConsumirReciboCargaDirecta) GoString() string

func (s SolicitudConsumirReciboCargaDirecta) LogValue() slog.Value

func (SolicitudConsumirReciboCargaDirecta) MarshalJSON() ([]byte, error)

func (SolicitudConsumirReciboCargaDirecta) MarshalText() ([]byte, error)

func (SolicitudConsumirReciboCargaDirecta) String() string

func (s SolicitudConsumirReciboCargaDirecta) Validar() error

type SolicitudCustodiarNotificacionCobro = pagoscanonicos.SolicitudCustodiarNotificacionCobro

type SolicitudDevolucionCobro = pagoscanonicos.SolicitudDevolucionCobro

type SolicitudEliminarCodigoCotejoHuerfano struct {
	Contexto      ContextoEliminarCodigoCotejoHuerfano
	ProteccionRef string
	EvidenciaRef  string
	Motivo        string
}

func (s SolicitudEliminarCodigoCotejoHuerfano) ValidarEn(instante time.Time) error

type SolicitudEliminarObjeto struct {
	Contexto      ContextoOperacionAlmacen
	Objeto        ReferenciaObjetoAlmacen
	AprobacionRef string
	Motivo        string
}

func (s SolicitudEliminarObjeto) Validar() error

type SolicitudEmitirReciboCargaDirecta struct {
	// Has unexported fields.
}
```

SolicitudEmitirReciboCargaDirecta encapsula la sesion temporal para que
no circule como un campo registrable. El emisor firma tambien el vinculo
inmutable de carga, sujeto seudonimizado, recurso, modulo y solicitud.

```go
func (s SolicitudEmitirReciboCargaDirecta) Format(estado fmt.State, _ rune)

func (SolicitudEmitirReciboCargaDirecta) GoString() string

func (s SolicitudEmitirReciboCargaDirecta) LogValue() slog.Value

func (SolicitudEmitirReciboCargaDirecta) MarshalJSON() ([]byte, error)

func (SolicitudEmitirReciboCargaDirecta) MarshalText() ([]byte, error)

func (s SolicitudEmitirReciboCargaDirecta) RevelarParaEmision() (
	contexto ContextoOperacionAlmacen,
	sesionRef string,
	expiraEn time.Time,
	vinculoSolicitudSHA256 string,
	err error,
)
```

RevelarParaEmision es el unico punto donde el adaptador MAC puede obtener la
sesion. No debe copiarla a errores, trazas o persistencia.

```go
func (SolicitudEmitirReciboCargaDirecta) String() string

func (s SolicitudEmitirReciboCargaDirecta) Validar() error

type SolicitudEscribirObjeto struct {
	Contexto          ContextoOperacionAlmacen
	ClaveIdempotencia string
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
	Contenido         io.Reader
}

func (s SolicitudEscribirObjeto) Validar() error

type SolicitudEvaluarReglaFlujo struct {
	Definicion     domain.DefinicionFlujo
	Instancia      domain.InstanciaFlujo
	Transicion     domain.TransicionFlujoConfigurable
	ActorID        string
	Finalidad      string
	Motivo         string
	CorrelacionRef string
}

type SolicitudFirmaAtestacionAutorizacionV1 struct {
	// Has unexported fields.
}
```

SolicitudFirmaAtestacionAutorizacionV1 es una capacidad inmutable.
La cabecera se selecciona antes de construir el mensaje y el firmante recibe
exactamente esos bytes; no puede devolver otra suite, clave o audiencia.

```go
func NuevaSolicitudFirmaAtestacionAutorizacionV1(
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	decision domain.DecisionAutorizacion,
) (SolicitudFirmaAtestacionAutorizacionV1, error)

func (s SolicitudFirmaAtestacionAutorizacionV1) Cabecera() (domain.CabeceraAtestacionAutorizacionV1, error)

func (s SolicitudFirmaAtestacionAutorizacionV1) HuellaMensajeSHA256() (string, error)

func (s SolicitudFirmaAtestacionAutorizacionV1) Mensaje() ([]byte, error)

func (s SolicitudFirmaAtestacionAutorizacionV1) ReferenciaDecision() (string, error)

func (s SolicitudFirmaAtestacionAutorizacionV1) Validar() error

type SolicitudFirmaAtestacionAutorizacionV2 struct {
	// Has unexported fields.
}
```

SolicitudFirmaAtestacionAutorizacionV2 conserva exactamente el mensaje
VEC-AD-2 que debe recibir el firmante. Es un contrato nominal: ni construir
la solicitud ni obtener sus datos concede autoridad para ejecutar un efecto.

```go
func NuevaSolicitudFirmaAtestacionAutorizacionV2(
	cabecera domain.CabeceraAtestacionAutorizacionV2,
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
) (SolicitudFirmaAtestacionAutorizacionV2, error)

func (s SolicitudFirmaAtestacionAutorizacionV2) Cabecera() (
	domain.CabeceraAtestacionAutorizacionV2,
	error,
)

func (b SolicitudFirmaAtestacionAutorizacionV2) Format(estado fmt.State, _ rune)

func (b SolicitudFirmaAtestacionAutorizacionV2) GoString() string

func (*SolicitudFirmaAtestacionAutorizacionV2) GobDecode([]byte) error

func (SolicitudFirmaAtestacionAutorizacionV2) GobEncode() ([]byte, error)

func (s SolicitudFirmaAtestacionAutorizacionV2) HuellaMensajeSHA256() (string, error)

func (s SolicitudFirmaAtestacionAutorizacionV2) HuellaMotivoCatalogoSHA256() (string, error)

func (s SolicitudFirmaAtestacionAutorizacionV2) HuellaSolicitudLigadaSHA256() (string, error)

func (b SolicitudFirmaAtestacionAutorizacionV2) LogValue() slog.Value

func (SolicitudFirmaAtestacionAutorizacionV2) MarshalBinary() ([]byte, error)

func (SolicitudFirmaAtestacionAutorizacionV2) MarshalCBOR() ([]byte, error)

func (SolicitudFirmaAtestacionAutorizacionV2) MarshalJSON() ([]byte, error)

func (SolicitudFirmaAtestacionAutorizacionV2) MarshalText() ([]byte, error)

func (SolicitudFirmaAtestacionAutorizacionV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (SolicitudFirmaAtestacionAutorizacionV2) MarshalYAML() (any, error)

func (s SolicitudFirmaAtestacionAutorizacionV2) Mensaje() ([]byte, error)

func (s SolicitudFirmaAtestacionAutorizacionV2) ReferenciaDecision() (string, error)

func (SolicitudFirmaAtestacionAutorizacionV2) String() string

func (*SolicitudFirmaAtestacionAutorizacionV2) UnmarshalBinary([]byte) error

func (*SolicitudFirmaAtestacionAutorizacionV2) UnmarshalCBOR([]byte) error

func (*SolicitudFirmaAtestacionAutorizacionV2) UnmarshalJSON([]byte) error

func (*SolicitudFirmaAtestacionAutorizacionV2) UnmarshalText([]byte) error

func (*SolicitudFirmaAtestacionAutorizacionV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*SolicitudFirmaAtestacionAutorizacionV2) UnmarshalYAML(func(any) error) error

func (s SolicitudFirmaAtestacionAutorizacionV2) Validar() error

type SolicitudFirmaEvidenciaRenderizadoDocumentalV3 struct {
	// Has unexported fields.
}

func NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(
	perfil PerfilSelloEvidenciaDocumentalV3,
	datos DatosEvidenciaRenderizadoDocumentalV3,
) (SolicitudFirmaEvidenciaRenderizadoDocumentalV3, error)

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Format(estado fmt.State, _ rune)

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) GoString() string

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) HuellaMensajeSHA256() (string, error)

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) LogValue() slog.Value

func (SolicitudFirmaEvidenciaRenderizadoDocumentalV3) MarshalBinary() ([]byte, error)

func (SolicitudFirmaEvidenciaRenderizadoDocumentalV3) MarshalJSON() ([]byte, error)

func (SolicitudFirmaEvidenciaRenderizadoDocumentalV3) MarshalText() ([]byte, error)

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Mensaje() ([]byte, error)

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Perfil() (PerfilSelloEvidenciaDocumentalV3, error)

func (SolicitudFirmaEvidenciaRenderizadoDocumentalV3) String() string

func (*SolicitudFirmaEvidenciaRenderizadoDocumentalV3) UnmarshalBinary([]byte) error

func (*SolicitudFirmaEvidenciaRenderizadoDocumentalV3) UnmarshalJSON([]byte) error

func (*SolicitudFirmaEvidenciaRenderizadoDocumentalV3) UnmarshalText([]byte) error

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Validar() error

type SolicitudGuardarContenido struct {
	Contexto          ContextoOperacionAlmacen
	ClaveIdempotencia string
	DocumentoID       string
	Zona              ZonaAlmacen
	MIME              string
	HuellaSHA256      string
	Tamano            int64
	Contenido         []byte
}

func (s SolicitudGuardarContenido) Validar() error
```

Validar exige que la solicitud coincida byte a byte con el paso actualmente
seleccionado del manifiesto autorizado. Un contexto de escritura generico,
otro paso del mismo manifiesto o metadatos parcialmente coincidentes se
deniegan; no existe compatibilidad permisiva para la generacion documental.

```go
func (s SolicitudGuardarContenido) ValidarEn(instante time.Time) error
```

ValidarEn debe ejecutarse inmediatamente antes del efecto remoto. Ademas del
manifiesto exacto, revalida la vigencia temporal de la decision.

```go
type SolicitudIncorporarMetadatoInstitucional struct {
	Perfil              domain.PerfilFormatoDocumental
	Conector            domain.ReferenciaConectorDocumental
	Etapa               EtapaMetadatoInstitucionalDocumental
	ContenidoSinFirma   []byte
	HuellaEntradaSHA256 string
	Metadato            domain.MarcaInstitucionalDocumento
}
```

SolicitudIncorporarMetadatoInstitucional solo acepta bytes aun no firmados.
El metadato es normalizado y no renderizado por defecto; el conector debe
incorporarlo mediante el mecanismo estandar del perfil, nunca mediante
zero-width, comentarios secretos o alteracion del significado visible.

```go
func (s SolicitudIncorporarMetadatoInstitucional) Validar() error

type SolicitudIniciarEfectoDocumentalV3 struct {
	VinculoActivacion VinculoEstableActivacionDocumentalV3
	Token             TokenCercadoEjecucionDocumentalV3Nominal
	IniciadoEn        time.Time
}

func (s SolicitudIniciarEfectoDocumentalV3) Format(estado fmt.State, _ rune)

func (s SolicitudIniciarEfectoDocumentalV3) GoString() string

func (s SolicitudIniciarEfectoDocumentalV3) LogValue() slog.Value

func (SolicitudIniciarEfectoDocumentalV3) MarshalBinary() ([]byte, error)

func (SolicitudIniciarEfectoDocumentalV3) MarshalJSON() ([]byte, error)

func (SolicitudIniciarEfectoDocumentalV3) MarshalText() ([]byte, error)

func (SolicitudIniciarEfectoDocumentalV3) String() string

func (*SolicitudIniciarEfectoDocumentalV3) UnmarshalBinary([]byte) error

func (*SolicitudIniciarEfectoDocumentalV3) UnmarshalJSON([]byte) error

func (*SolicitudIniciarEfectoDocumentalV3) UnmarshalText([]byte) error

func (s SolicitudIniciarEfectoDocumentalV3) Validar() error

type SolicitudInmovilizarObjeto struct {
	Contexto      ContextoOperacionAlmacen
	Objeto        ReferenciaObjetoAlmacen
	AprobacionRef string
	Motivo        string
}

func (s SolicitudInmovilizarObjeto) Validar() error

type SolicitudLeerContenido struct {
	Contexto   ContextoOperacionAlmacen
	Referencia string
	Zona       ZonaAlmacen
	Limite     int64
}

type SolicitudLevantarInmovilizacionObjeto struct {
	Contexto      ContextoOperacionAlmacen
	Objeto        ReferenciaObjetoAlmacen
	AprobacionRef string
	Motivo        string
}

func (s SolicitudLevantarInmovilizacionObjeto) Validar() error

type SolicitudMarcarEjecucionDocumentalV3Indeterminada struct {
	ReservaRef             string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	OrdenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal
	IncidenteRef           string
	MarcadaEn              time.Time
}

func (s SolicitudMarcarEjecucionDocumentalV3Indeterminada) Validar() error

type SolicitudMarcarPasoGeneracionDocumentalIndeterminado struct {
	ReservaRef   string
	Contexto     ContextoOperacionAlmacen
	IncidenteRef string
}
```

SolicitudMarcarPasoGeneracionDocumentalIndeterminado se usa cuando no puede
saberse si el almacen aplico el efecto. IncidenteRef es una referencia opaca
a la traza; nunca incorpora el error, una URL ni datos del documento.

```go
func (s SolicitudMarcarPasoGeneracionDocumentalIndeterminado) Validar() error

type SolicitudObtenerEvidenciaEmisionDocumento struct {
	Documento        domain.ReferenciaDocumento
	RepresentacionID string
	SolicitanteID    string
	AutorizacionRef  string
	Finalidad        string
	CorrelacionRef   string
}

type SolicitudOperacionCobro = pagoscanonicos.SolicitudOperacionCobro

type SolicitudPrepararCargaDirecta struct {
	Contexto          ContextoOperacionAlmacen
	ClaveIdempotencia string
	MIME              string
	Tamano            int64
	HuellaSHA256      string
	ExpiraEn          time.Time
}

func (s SolicitudPrepararCargaDirecta) Validar() error

type SolicitudPrepararEjecucionDocumentalV3 struct {
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	SolicitadaEn           time.Time
	ExpiraEn               time.Time
}

func (s SolicitudPrepararEjecucionDocumentalV3) Format(estado fmt.State, _ rune)

func (s SolicitudPrepararEjecucionDocumentalV3) GoString() string

func (s SolicitudPrepararEjecucionDocumentalV3) LogValue() slog.Value

func (SolicitudPrepararEjecucionDocumentalV3) MarshalBinary() ([]byte, error)

func (SolicitudPrepararEjecucionDocumentalV3) MarshalJSON() ([]byte, error)

func (SolicitudPrepararEjecucionDocumentalV3) MarshalText() ([]byte, error)

func (SolicitudPrepararEjecucionDocumentalV3) String() string

func (*SolicitudPrepararEjecucionDocumentalV3) UnmarshalBinary([]byte) error

func (*SolicitudPrepararEjecucionDocumentalV3) UnmarshalJSON([]byte) error

func (*SolicitudPrepararEjecucionDocumentalV3) UnmarshalText([]byte) error

func (s SolicitudPrepararEjecucionDocumentalV3) Validar() error

type SolicitudPromoverObjeto struct {
	Contexto             ContextoOperacionAlmacen
	ClaveIdempotencia    string
	Origen               ReferenciaObjetoAlmacen
	EvidenciaAnalisisRef string
}

func (s SolicitudPromoverObjeto) Validar() error

type SolicitudProtegerCodigoCotejo struct {
	Contexto          ContextoProtegerCodigoCotejo
	ClaveIdempotencia string
	Secreto           SecretoCodigoCotejo
	IndiceCodigoHMAC  string
}

func (s SolicitudProtegerCodigoCotejo) ValidarEn(instante time.Time) error

type SolicitudReclamarOrdenDespachoDocumentalV3 = documentalcanonico.DatosSolicitudReclamacionV3
```

SolicitudReclamarOrdenDespachoDocumentalV3 identifica una unica reclamacion
CAS del evento outbox de inicio. El llamador solo aporta referencias opacas;
nunca un resultado de verificacion ni un token.

```go
type SolicitudRecuperarCodigoCotejo struct {
	Contexto                 ContextoRecuperarCodigoCotejo
	ProteccionRef            string
	IndiceCodigoHMACEsperado string
}

func (s SolicitudRecuperarCodigoCotejo) ValidarEn(instante time.Time) error

type SolicitudReservaDevolucionCobro struct {
	OrdenRef               string
	DevolucionRef          string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	PrincipalRef           string
	SolicitadaEn           time.Time
	ExpiraEn               time.Time
}

func (s SolicitudReservaDevolucionCobro) Validar() error

type SolicitudReservaOrdenCobro struct {
	OrdenRef               string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	PrincipalRef           string
	SolicitadaEn           time.Time
	ExpiraEn               time.Time
}

func (s SolicitudReservaOrdenCobro) Validar() error

type SolicitudReservarCargaDocumental struct {
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	Carga                  domain.CargaDocumental
	DecisionPreparacion    ConsumoDecisionPreparacionCargaDocumentalV1
	SolicitadaEn           time.Time
	ReservaExpiraEn        time.Time
}

func (s SolicitudReservarCargaDocumental) Validar() error

type SolicitudReservarEfectoGeneracionDocumental struct {
	Contexto   ContextoOperacionAlmacen
	Manifiesto ManifiestoGeneracionDocumental
}
```

SolicitudReservarEfectoGeneracionDocumental no permite proponer la tupla
durable como campos libres. El repositorio la extrae de la capacidad y
del manifiesto opacos y consume DecisionRef de forma unica en la misma
transaccion que reserva EfectoRef y todos sus pasos.

```go
func (s SolicitudReservarEfectoGeneracionDocumental) ValidarEn(instante time.Time) error

type SolicitudReservarEmisionCodigoCotejo struct {
	ClaveIdempotencia   string
	PrincipalID         string
	HuellaSolicitudHMAC string
	Documento           domain.ReferenciaDocumento
	Politica            domain.ReferenciaPoliticaCotejo
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
}
```

SolicitudReservarEmisionCodigoCotejo fija el significado de una clave de
idempotencia sin guardar los datos de la orden. La huella debe ser HMAC con
una clave distinta de la empleada para indexar el CSV.

```go
type SolicitudReservarGeneracionDocumento struct {
	ClaveIdempotencia   string
	PrincipalID         string
	HuellaSolicitudHMAC string
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
}
```

SolicitudReservarGeneracionDocumento fija una clave idempotente dentro del
ambito del principal. La huella HMAC vincula todos los datos con efecto sin
persistirlos en claro dentro del control de concurrencia.

```go
type SolicitudReservarReferenciaReciboMaterialV2 struct {
	// Has unexported fields.
}
```

SolicitudReservarReferenciaReciboMaterialV2 identifica el recibo por todos
sus hechos materiales, sin aceptar una referencia propuesta por el llamador.
El registro debe reservarla o recuperar la original atomica y durablemente.

```go
func (SolicitudReservarReferenciaReciboMaterialV2) Format(e fmt.State, _ rune)

func (SolicitudReservarReferenciaReciboMaterialV2) GoString() string

func (s SolicitudReservarReferenciaReciboMaterialV2) HuellaIdentidad() (
	[sha256.Size]byte,
	error,
)

func (SolicitudReservarReferenciaReciboMaterialV2) LogValue() slog.Value

func (SolicitudReservarReferenciaReciboMaterialV2) MarshalBinary() ([]byte, error)

func (SolicitudReservarReferenciaReciboMaterialV2) MarshalJSON() ([]byte, error)

func (SolicitudReservarReferenciaReciboMaterialV2) MarshalText() ([]byte, error)

func (SolicitudReservarReferenciaReciboMaterialV2) String() string

func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalBinary([]byte) error

func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalJSON([]byte) error

func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalText([]byte) error

type SolicitudResolucionRegistroContextoActorV1 struct {
	OperacionRef string
	Contexto     domain.SolicitudContextoActor
	SolicitadoEn time.Time
}
```

SolicitudResolucionRegistroContextoActorV1 fija la identidad de una unica
invocacion durable. SolicitadoEn es una observacion local para acotar
frescura y deriva de reloj; no es el instante autoritativo que debe guardar
el adaptador.

```go
func (s SolicitudResolucionRegistroContextoActorV1) Validar() error

type SolicitudRetenerObjeto struct {
	Contexto    ContextoOperacionAlmacen
	Objeto      ReferenciaObjetoAlmacen
	PoliticaRef string
	Hasta       time.Time
}

func (s SolicitudRetenerObjeto) Validar() error

func (s SolicitudRetenerObjeto) ValidarEn(instante time.Time) error

type SolicitudSellarIdempotenciaCarga = almacencanonico.SolicitudSellarIdempotenciaCarga

type SolicitudSeudonimizarSujetoAlmacen = almacencanonico.SolicitudSeudonimizarSujetoAlmacen
```

SolicitudSeudonimizarSujetoAlmacen mantiene el identificador interno fuera
del contexto, evidencias y conectores de objetos. Solo el sellador local
confiable lo revela durante la operacion HMAC con clave exclusiva.

```go
func NuevaSolicitudSeudonimizarSujetoAlmacen(
	sujetoRef, ambitoRef string,
) (SolicitudSeudonimizarSujetoAlmacen, error)

type SolicitudVerificacionEvidenciaDocumentalV3 struct {
	// Has unexported fields.
}

func NuevaSolicitudVerificacionEvidenciaDocumentalV3(
	firma SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	sello SelloEvidenciaDocumentalV3Nominal,
) (SolicitudVerificacionEvidenciaDocumentalV3, error)

func NuevaSolicitudVerificacionEvidenciaDocumentalV3DesdeDatos(
	firma SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	datos DatosSelloEvidenciaDocumentalV3Crudos,
) (SolicitudVerificacionEvidenciaDocumentalV3, error)
```

NuevaSolicitudVerificacionEvidenciaDocumentalV3DesdeDatos permite restaurar
un sobre persistido, pero no lo convierte en evidencia confiable: el unico
la salida del conector tambien es nominal. Solo el servicio de aplicacion
precompuesto puede cotejarla con relectura durable y CAS dentro de su
llamada.

```go
func (s SolicitudVerificacionEvidenciaDocumentalV3) Format(estado fmt.State, _ rune)

func (s SolicitudVerificacionEvidenciaDocumentalV3) GoString() string

func (s SolicitudVerificacionEvidenciaDocumentalV3) LogValue() slog.Value

func (SolicitudVerificacionEvidenciaDocumentalV3) MarshalBinary() ([]byte, error)

func (SolicitudVerificacionEvidenciaDocumentalV3) MarshalJSON() ([]byte, error)

func (SolicitudVerificacionEvidenciaDocumentalV3) MarshalText() ([]byte, error)

func (s SolicitudVerificacionEvidenciaDocumentalV3) Mensaje() ([]byte, error)

func (s SolicitudVerificacionEvidenciaDocumentalV3) Sello() (DatosSelloEvidenciaDocumentalV3Crudos, error)

func (SolicitudVerificacionEvidenciaDocumentalV3) String() string

func (*SolicitudVerificacionEvidenciaDocumentalV3) UnmarshalBinary([]byte) error

func (*SolicitudVerificacionEvidenciaDocumentalV3) UnmarshalJSON([]byte) error

func (*SolicitudVerificacionEvidenciaDocumentalV3) UnmarshalText([]byte) error

func (s SolicitudVerificacionEvidenciaDocumentalV3) Validar() error

type SolicitudVerificacionReconciliacionDocumentalV3 struct {
	// Has unexported fields.
}

func NuevaSolicitudVerificacionReconciliacionDocumentalV3(
	consulta SolicitudConsultarEfectoDocumentalV3,
	resultado ResultadoConsultaEfectoDocumentalV3Crudo,
) (SolicitudVerificacionReconciliacionDocumentalV3, error)

func (s SolicitudVerificacionReconciliacionDocumentalV3) Format(estado fmt.State, _ rune)

func (s SolicitudVerificacionReconciliacionDocumentalV3) GoString() string

func (s SolicitudVerificacionReconciliacionDocumentalV3) LogValue() slog.Value

func (SolicitudVerificacionReconciliacionDocumentalV3) MarshalBinary() ([]byte, error)

func (SolicitudVerificacionReconciliacionDocumentalV3) MarshalJSON() ([]byte, error)

func (SolicitudVerificacionReconciliacionDocumentalV3) MarshalText() ([]byte, error)

func (s SolicitudVerificacionReconciliacionDocumentalV3) Mensaje() ([]byte, error)

func (s SolicitudVerificacionReconciliacionDocumentalV3) Sobre() ([]byte, error)

func (SolicitudVerificacionReconciliacionDocumentalV3) String() string

func (*SolicitudVerificacionReconciliacionDocumentalV3) UnmarshalBinary([]byte) error

func (*SolicitudVerificacionReconciliacionDocumentalV3) UnmarshalJSON([]byte) error

func (*SolicitudVerificacionReconciliacionDocumentalV3) UnmarshalText([]byte) error

func (s SolicitudVerificacionReconciliacionDocumentalV3) Validar() error

type SolicitudVerificacionTokenCercadoDocumentalV3 struct {
	// Has unexported fields.
}

func NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
	vinculo VinculoEstableActivacionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3Nominal,
) (SolicitudVerificacionTokenCercadoDocumentalV3, error)

func (s SolicitudVerificacionTokenCercadoDocumentalV3) ClaveAtestacionRef() (string, error)

func (s SolicitudVerificacionTokenCercadoDocumentalV3) Format(estado fmt.State, _ rune)

func (s SolicitudVerificacionTokenCercadoDocumentalV3) GoString() string

func (s SolicitudVerificacionTokenCercadoDocumentalV3) LogValue() slog.Value

func (s SolicitudVerificacionTokenCercadoDocumentalV3) MAC() ([]byte, error)

func (SolicitudVerificacionTokenCercadoDocumentalV3) MarshalBinary() ([]byte, error)

func (SolicitudVerificacionTokenCercadoDocumentalV3) MarshalJSON() ([]byte, error)

func (SolicitudVerificacionTokenCercadoDocumentalV3) MarshalText() ([]byte, error)

func (s SolicitudVerificacionTokenCercadoDocumentalV3) Mensaje() ([]byte, error)

func (SolicitudVerificacionTokenCercadoDocumentalV3) String() string

func (*SolicitudVerificacionTokenCercadoDocumentalV3) UnmarshalBinary([]byte) error

func (*SolicitudVerificacionTokenCercadoDocumentalV3) UnmarshalJSON([]byte) error

func (*SolicitudVerificacionTokenCercadoDocumentalV3) UnmarshalText([]byte) error

func (s SolicitudVerificacionTokenCercadoDocumentalV3) Validar() error

type SolicitudVerificarAprobacionFlujo struct {
	ReferenciaAprobacion string
	SolicitanteID        string
	Definicion           domain.DefinicionFlujo
	Instancia            domain.InstanciaFlujo
	Transicion           domain.TransicionFlujoConfigurable
	DecisionRegla        domain.DecisionReglaFlujo
	Finalidad            string
	CorrelacionRef       string
}

type SolicitudVerificarAtestacionMaterialAlmacenV2 struct {
	// Has unexported fields.
}
```

SolicitudVerificarAtestacionMaterialAlmacenV2 impide verificar una firma
descontextualizada o sobre bytes recompuestos por otro componente.

```go
func (SolicitudVerificarAtestacionMaterialAlmacenV2) Format(e fmt.State, _ rune)

func (SolicitudVerificarAtestacionMaterialAlmacenV2) GoString() string

func (SolicitudVerificarAtestacionMaterialAlmacenV2) LogValue() slog.Value

func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalBinary() ([]byte, error)

func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalJSON() ([]byte, error)

func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalText() ([]byte, error)

func (s SolicitudVerificarAtestacionMaterialAlmacenV2) RevelarParaVerificacion() (
	dominio string,
	mensaje []byte,
	algoritmo AlgoritmoAtestacionMaterialAlmacenV2,
	claveRef string,
	claveVersion uint32,
	codigo []byte,
	err error,
)
```

RevelarParaVerificacion es la unica apertura de clave, version, mensaje y
autenticador hacia un verificador homologado.

```go
func (SolicitudVerificarAtestacionMaterialAlmacenV2) String() string

func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalBinary([]byte) error

func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalJSON([]byte) error

func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalText([]byte) error

type SolicitudVerificarPerfilPublicadoMaterialV2 struct {
	// Has unexported fields.
}
```

SolicitudVerificarPerfilPublicadoMaterialV2 separa autenticidad
criptografica de homologacion. Una firma valida sobre capacidades elegidas
por el llamador no convierte el perfil en un perfil publicado.

```go
func (SolicitudVerificarPerfilPublicadoMaterialV2) Format(e fmt.State, _ rune)

func (SolicitudVerificarPerfilPublicadoMaterialV2) GoString() string

func (SolicitudVerificarPerfilPublicadoMaterialV2) LogValue() slog.Value

func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalBinary() ([]byte, error)

func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalJSON() ([]byte, error)

func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalText() ([]byte, error)

func (s SolicitudVerificarPerfilPublicadoMaterialV2) RevelarParaHomologacion() (
	referencia string,
	version uint32,
	conectorLogicoID string,
	huella [sha256.Size]byte,
	canonico []byte,
	err error,
)

func (SolicitudVerificarPerfilPublicadoMaterialV2) String() string

func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalBinary([]byte) error

func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalJSON([]byte) error

func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalText([]byte) error

type SolicitudVerificarPlanMaterialAlmacenV2 struct {
	// Has unexported fields.
}
```

SolicitudVerificarPlanMaterialAlmacenV2 liga el plan a los hechos que el
registro debe reconocer como estables. Operacion, carga y efecto nunca se
aceptan solo porque aparezcan en el contexto V1.

```go
func (SolicitudVerificarPlanMaterialAlmacenV2) Format(e fmt.State, _ rune)

func (SolicitudVerificarPlanMaterialAlmacenV2) GoString() string

func (SolicitudVerificarPlanMaterialAlmacenV2) LogValue() slog.Value

func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error)

func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error)

func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalText() ([]byte, error)

func (s SolicitudVerificarPlanMaterialAlmacenV2) RevelarParaVerificacionPlanMaterial() (
	referencia string,
	version uint32,
	conectorLogicoID, moduloID, accionNegocio, accionTecnica, recursoRef string,
	operacionRef, cargaRef, efectoRef, clasificacion string,
	huellaVinculo [sha256.Size]byte,
	err error,
)
```

RevelarParaVerificacionPlanMaterial entrega una copia estable al registro de
planes. El resultado debe quedar ligado a HuellaVinculo.

```go
func (SolicitudVerificarPlanMaterialAlmacenV2) String() string

func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error

func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error

func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalText([]byte) error

type SolicitudVinculadaAutorizacionEjecucionDocumentalV4 struct {
	// Has unexported fields.
}
```

SolicitudVinculadaAutorizacionEjecucionDocumentalV4 es una comprobacion
estructural, opaca e inmutable. No es una capacidad ni concede autoridad: la
evidencia de la que parte puede construirse desde un DTO publico. Su valor
cero se deniega y no puede reconstruirse desde una proyeccion persistida.

La autoridad de composicion final debe nacer dentro de un conector
homologado que implemente ConectorEjecucionDocumentalAtestadaV4. El nucleo
nunca convierte esta comprobacion estructural en una concesion por si solo.

```go
func NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
	evidencia EvidenciaUsoDecisionAutorizacion,
	expectativa ExpectativaAutorizacionEjecucionDocumentalV4,
	vinculadaEn time.Time,
) (SolicitudVinculadaAutorizacionEjecucionDocumentalV4, error)
```

NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4 estrecha una
evidencia estructural ya concedida. Nunca amplia su accion, actor, recurso,
finalidad, ambitos, campos u obligaciones. vinculadaEn debe proceder del
reloj confiable del servidor. El resultado sigue sin ser autoridad.

EvidenciaUsoDecisionAutorizacion actualmente deniega toda obligacion no
vacia porque aun no existe una prueba tipada de cumplimiento. Este puente
conserva esa garantia: una obligacion esperada o un cumplimiento espurio se
deniegan hasta que el contrato raiz pueda acreditarlos positivamente.

```go
func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) Format(estado fmt.State, _ rune)

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) GoString() string

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) HuellaSHA256() (string, error)

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) LogValue() slog.Value

func (SolicitudVinculadaAutorizacionEjecucionDocumentalV4) MarshalJSON() ([]byte, error)

func (SolicitudVinculadaAutorizacionEjecucionDocumentalV4) MarshalText() ([]byte, error)

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) PrepararSolicitudAplicacionEn(
	instante time.Time,
) (SolicitudAplicacionAutorizacionEjecucionDocumentalV4, error)

func (SolicitudVinculadaAutorizacionEjecucionDocumentalV4) String() string

func (*SolicitudVinculadaAutorizacionEjecucionDocumentalV4) UnmarshalJSON([]byte) error

func (*SolicitudVinculadaAutorizacionEjecucionDocumentalV4) UnmarshalText([]byte) error

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) ValidarEn(instante time.Time) error

type SumideroLimitadoSalidaDocumental struct {
	// Has unexported fields.
}
```

SumideroLimitadoSalidaDocumental observa exactamente los bytes aceptados
por el destino. Cada Write es atomico respecto de otros Write y de Cerrar;
al superar el limite o fallar el destino queda cerrado sin posibilidad de
reanudar una salida parcial.

```go
func NuevoSumideroLimitadoSalidaDocumental(
	destino io.Writer,
	limiteBytes uint64,
) (*SumideroLimitadoSalidaDocumental, error)

func (s *SumideroLimitadoSalidaDocumental) Cerrar() (SalidaObservadaDocumental, error)
```

Cerrar es irreversible e idempotente cuando la primera clausura tuvo exito.
Tras un fallo devuelve siempre ese fallo y nunca fabrica una salida parcial.

```go
func (s *SumideroLimitadoSalidaDocumental) Write(p []byte) (int, error)

type TimestampPort interface {
	Timestamp(context.Context, InteropRequest) (InteropResult, error)
}

type TipoEventoSalidaCobro = pagoscanonicos.TipoEventoSalidaCobro

type TokenCercadoEjecucionDocumentalV3Nominal struct {
	// Has unexported fields.
}
```

TokenCercadoEjecucionDocumentalV3Nominal combina un valor aleatorio con
una secuencia monotona y una MAC restaurable. La secuencia de cercado es
distinta de la secuencia operativa del perfil, aunque la huella liga ambas
de forma inseparable. Su construccion publica solo acredita forma nominal:
el servicio privado debe verificar la MAC con la clave gestionada antes de
cualquier uso.

```go
func NuevoTokenCercadoEjecucionDocumentalV3Nominal(
	valor string,
	secuencia uint64,
	vinculo VinculoEstableActivacionDocumentalV3,
	claveAtestacionRef string,
	revisionClave uint64,
	macAtestacion []byte,
	evidenciaOperacionRef string,
) (TokenCercadoEjecucionDocumentalV3Nominal, error)
```

NuevoTokenCercadoEjecucionDocumentalV3Nominal queda reservado al adaptador
de RegistroEjecucionesDocumentalesV3. Solo construye el sobre: NO autentica
el token. El MAC compromete la huella canonica del vinculo estable; el
registro debe comprobarlo con su clave gestionada antes de cualquier efecto.

```go
func (t TokenCercadoEjecucionDocumentalV3Nominal) Format(estado fmt.State, _ rune)

func (t TokenCercadoEjecucionDocumentalV3Nominal) GoString() string

func (t TokenCercadoEjecucionDocumentalV3Nominal) HuellaVinculoSHA256() string

func (t TokenCercadoEjecucionDocumentalV3Nominal) LogValue() slog.Value

func (TokenCercadoEjecucionDocumentalV3Nominal) MarshalBinary() ([]byte, error)

func (TokenCercadoEjecucionDocumentalV3Nominal) MarshalJSON() ([]byte, error)

func (TokenCercadoEjecucionDocumentalV3Nominal) MarshalText() ([]byte, error)

func (t TokenCercadoEjecucionDocumentalV3Nominal) RevisionClaveGestionada() uint64

func (t TokenCercadoEjecucionDocumentalV3Nominal) Secuencia() uint64

func (TokenCercadoEjecucionDocumentalV3Nominal) String() string

func (*TokenCercadoEjecucionDocumentalV3Nominal) UnmarshalBinary([]byte) error

func (*TokenCercadoEjecucionDocumentalV3Nominal) UnmarshalJSON([]byte) error

func (*TokenCercadoEjecucionDocumentalV3Nominal) UnmarshalText([]byte) error

func (t TokenCercadoEjecucionDocumentalV3Nominal) ValidarPara(
	vinculo VinculoEstableActivacionDocumentalV3,
) error

type TokenReservaCargaDocumental struct {
	// Has unexported fields.
}
```

TokenReservaCargaDocumental es una capacidad efimera y nominal entre el caso
de uso y el repositorio. Su material CSPRNG vive exclusivamente en un cierre
privado e inmutable ligado al dominio de carga documental. Nunca forma parte
del agregado, la auditoria, el outbox, una respuesta HTTP o un mensaje de
error. Los repositorios persisten solo HuellaSHA256 y verifican mediante
CoincideConHuellaSHA256.

```go
func NuevoTokenReservaCargaDocumental() (TokenReservaCargaDocumental, error)

func (t TokenReservaCargaDocumental) CoincideConHuellaSHA256(huella string) bool

func (t TokenReservaCargaDocumental) Format(estado fmt.State, _ rune)

func (t TokenReservaCargaDocumental) GoString() string

func (t TokenReservaCargaDocumental) HuellaSHA256() (string, error)

func (t TokenReservaCargaDocumental) LogValue() slog.Value

func (TokenReservaCargaDocumental) MarshalBinary() ([]byte, error)

func (TokenReservaCargaDocumental) MarshalJSON() ([]byte, error)

func (TokenReservaCargaDocumental) MarshalText() ([]byte, error)

func (TokenReservaCargaDocumental) MarshalXML(*xml.Encoder, xml.StartElement) error

func (TokenReservaCargaDocumental) String() string

func (*TokenReservaCargaDocumental) UnmarshalBinary([]byte) error

func (*TokenReservaCargaDocumental) UnmarshalJSON([]byte) error

func (*TokenReservaCargaDocumental) UnmarshalText([]byte) error

func (*TokenReservaCargaDocumental) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (t TokenReservaCargaDocumental) Valido() bool

type TokenReservaDevolucionCobro struct {
	// Has unexported fields.
}
```

TokenReservaDevolucionCobro no es intercambiable con el token de alta. Su
huella usa otro dominio aun cuando ambos secretos tengan la misma longitud.
La ligadura se realiza al crear el cierre, no al invocarlo.

```go
func NuevoTokenReservaDevolucionCobro() (TokenReservaDevolucionCobro, error)

func (t TokenReservaDevolucionCobro) CoincideConHuellaSHA256(huella string) bool

func (t TokenReservaDevolucionCobro) Format(estado fmt.State, _ rune)

func (t TokenReservaDevolucionCobro) GoString() string

func (t TokenReservaDevolucionCobro) HuellaSHA256() (string, error)

func (t TokenReservaDevolucionCobro) LogValue() slog.Value

func (TokenReservaDevolucionCobro) MarshalBinary() ([]byte, error)

func (TokenReservaDevolucionCobro) MarshalJSON() ([]byte, error)

func (TokenReservaDevolucionCobro) MarshalText() ([]byte, error)

func (TokenReservaDevolucionCobro) MarshalXML(*xml.Encoder, xml.StartElement) error

func (TokenReservaDevolucionCobro) String() string

func (*TokenReservaDevolucionCobro) UnmarshalBinary([]byte) error

func (*TokenReservaDevolucionCobro) UnmarshalJSON([]byte) error

func (*TokenReservaDevolucionCobro) UnmarshalText([]byte) error

func (*TokenReservaDevolucionCobro) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (t TokenReservaDevolucionCobro) Valido() bool

type TokenReservaEmisionCodigoCotejo struct {
	// Has unexported fields.
}
```

TokenReservaEmisionCodigoCotejo es una capacidad efimera, nominal y no
serializable. Su material aleatorio solo vive capturado por un cierre
privado e inmutable, ya ligado a este dominio, y es incompatible por
tipo con cualquier otra reserva del portal. La huella SHA-256 no permite
recuperar una entrada CSPRNG de 256 bits; un almacen puede envolverla con
HMAC si ademas necesita impedir correlacion entre almacenes.

```go
func NuevoTokenReservaEmisionCodigoCotejo() (TokenReservaEmisionCodigoCotejo, error)

func (t TokenReservaEmisionCodigoCotejo) CoincideConHuellaSHA256(huella string) bool

func (t TokenReservaEmisionCodigoCotejo) Format(estado fmt.State, _ rune)

func (t TokenReservaEmisionCodigoCotejo) GoString() string

func (t TokenReservaEmisionCodigoCotejo) HuellaSHA256() (string, error)

func (t TokenReservaEmisionCodigoCotejo) LogValue() slog.Value

func (TokenReservaEmisionCodigoCotejo) MarshalBinary() ([]byte, error)

func (TokenReservaEmisionCodigoCotejo) MarshalJSON() ([]byte, error)

func (TokenReservaEmisionCodigoCotejo) MarshalText() ([]byte, error)

func (TokenReservaEmisionCodigoCotejo) MarshalXML(*xml.Encoder, xml.StartElement) error

func (TokenReservaEmisionCodigoCotejo) String() string

func (*TokenReservaEmisionCodigoCotejo) UnmarshalBinary([]byte) error

func (*TokenReservaEmisionCodigoCotejo) UnmarshalJSON([]byte) error

func (*TokenReservaEmisionCodigoCotejo) UnmarshalText([]byte) error

func (*TokenReservaEmisionCodigoCotejo) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (t TokenReservaEmisionCodigoCotejo) Valido() bool

type TokenReservaGeneracionDocumento struct {
	// Has unexported fields.
}
```

TokenReservaGeneracionDocumento es una capacidad efimera, nominal y
no serializable. Se crea con 256 bits del CSPRNG y no ofrece ninguna
operacion para revelar su material. Los repositorios deben persistir
exclusivamente la huella obtenida con HuellaSHA256 y verificarla mediante
CoincideConHuellaSHA256. SHA-256 es suficiente aqui porque la entrada
tiene 256 bits uniformes, no es un dato humano atacable por diccionario.
Un almacen que deba ocultar tambien correlaciones puede envolver esta
huella con HMAC y una clave gestionada. El cierre privado e inmutable impide
recuperar el material mediante la API de reflexion segura y hace que el tipo
no sea comparable mediante ==.

```go
func NuevoTokenReservaGeneracionDocumento() (TokenReservaGeneracionDocumento, error)

func (t TokenReservaGeneracionDocumento) CoincideConHuellaSHA256(huella string) bool
```

CoincideConHuellaSHA256 realiza la unica comparacion autoritativa del token
contra estado durable. La comparacion de los 32 bytes se hace en tiempo
constante; una huella mal formada se rechaza en cerrado.

```go
func (t TokenReservaGeneracionDocumento) Format(estado fmt.State, _ rune)

func (t TokenReservaGeneracionDocumento) GoString() string

func (t TokenReservaGeneracionDocumento) HuellaSHA256() (string, error)

func (t TokenReservaGeneracionDocumento) LogValue() slog.Value

func (TokenReservaGeneracionDocumento) MarshalBinary() ([]byte, error)

func (TokenReservaGeneracionDocumento) MarshalJSON() ([]byte, error)

func (TokenReservaGeneracionDocumento) MarshalText() ([]byte, error)

func (TokenReservaGeneracionDocumento) MarshalXML(*xml.Encoder, xml.StartElement) error

func (TokenReservaGeneracionDocumento) String() string

func (*TokenReservaGeneracionDocumento) UnmarshalBinary([]byte) error

func (*TokenReservaGeneracionDocumento) UnmarshalJSON([]byte) error

func (*TokenReservaGeneracionDocumento) UnmarshalText([]byte) error

func (*TokenReservaGeneracionDocumento) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (t TokenReservaGeneracionDocumento) Valido() bool

type TokenReservaOrdenCobro struct {
	// Has unexported fields.
}
```

TokenReservaOrdenCobro es una capacidad efimera y nominal para confirmar o
abandonar exclusivamente una reserva de alta. Nunca se serializa ni revela;
la huella con separacion de dominio es el unico material persistible. El
secreto vive solo en un cierre privado e inmutable ligado a dicho dominio.

```go
func NuevoTokenReservaOrdenCobro() (TokenReservaOrdenCobro, error)

func (t TokenReservaOrdenCobro) CoincideConHuellaSHA256(huella string) bool

func (t TokenReservaOrdenCobro) Format(estado fmt.State, _ rune)

func (t TokenReservaOrdenCobro) GoString() string

func (t TokenReservaOrdenCobro) HuellaSHA256() (string, error)

func (t TokenReservaOrdenCobro) LogValue() slog.Value

func (TokenReservaOrdenCobro) MarshalBinary() ([]byte, error)

func (TokenReservaOrdenCobro) MarshalJSON() ([]byte, error)

func (TokenReservaOrdenCobro) MarshalText() ([]byte, error)

func (TokenReservaOrdenCobro) MarshalXML(*xml.Encoder, xml.StartElement) error

func (TokenReservaOrdenCobro) String() string

func (*TokenReservaOrdenCobro) UnmarshalBinary([]byte) error

func (*TokenReservaOrdenCobro) UnmarshalJSON([]byte) error

func (*TokenReservaOrdenCobro) UnmarshalText([]byte) error

func (*TokenReservaOrdenCobro) UnmarshalXML(*xml.Decoder, xml.StartElement) error

func (t TokenReservaOrdenCobro) Valido() bool

type ValidadorReferenciaMotivoAutorizacionV2 interface {
	ValidarReferenciaMotivoAutorizacionV2(
		context.Context,
		domain.ReferenciaEntradaCatalogo,
		time.Time,
	) error
}
```

ValidadorReferenciaMotivoAutorizacionV2 resuelve positivamente una entrada
contra el catalogo publicado. La validacion estructural de una referencia no
demuestra su existencia ni evita que un llamador fabrique una huella.

```go
type ValorCodigoCotejoGenerado struct {
	Secreto          SecretoCodigoCotejo
	EntropiaBits     int
	VersionGenerador string
}

type VerificadorAprobacionesFlujo interface {
	VerificarAprobacionFlujo(context.Context, SolicitudVerificarAprobacionFlujo) (domain.EvidenciaAprobacionFlujo, error)
}
```

VerificadorAprobacionesFlujo impide considerar suficiente una referencia
escrita por el cliente. El adaptador consulta el registro de aprobaciones y
devuelve la evidencia exacta, que el caso de uso vuelve a validar.

```go
type VerificadorAtestacionConsumoReciboCargaDirecta interface {
	VerificarAtestacionConsumoReciboCargaDirecta(
		context.Context,
		ContextoOperacionAlmacen,
		string,
		ComprobanteConsumoReciboCargaDirecta,
	) error
}
```

VerificadorAtestacionConsumoReciboCargaDirecta es una dependencia de
seguridad obligatoria del nucleo. Una implementacion que solo compruebe la
forma o devuelva nil no es apta para produccion: debe verificar con la clave
HMAC exclusiva de atestacion el contexto completo, autorizacion, accion,
sesion, evidencia y fecha durable.

```go
type VerificadorAtestacionMaterialAlmacenV2 interface {
	VerificarAtestacionMaterialAlmacenV2(
		context.Context,
		SolicitudVerificarAtestacionMaterialAlmacenV2,
	) error
}

type VerificadorAtestacionesReconciliacionDocumentalV3 interface {
	VerificarAtestacionReconciliacionDocumentalV3(
		context.Context,
		SolicitudVerificacionReconciliacionDocumentalV3,
	) (MetadatosComprobacionReconciliacionDocumentalV3Nominal, error)
}

type VerificadorCrudoRecibosDocumentales interface {
	VerificarReciboCrudo(
		context.Context,
		CompromisoEjecucionComponenteDocumental,
		SobreReciboEjecucionDocumentalCrudo,
	) (ReciboEjecucionComponenteDocumentalNominal, error)
}
```

VerificadorCrudoRecibosDocumentales es el conector intercambiable que coteja
COSE, confianza, vigencia/revocacion, ventana/reto y clave. Su salida es
nominal y nunca concede por si sola permiso para confirmar un efecto.

```go
type VerificadorEquivalenciaSemanticaDocumental interface {
	VerificarEquivalenciaSemantica(
		context.Context,
		domain.PerfilFormatoDocumental,
		[]byte,
		[]byte,
	) error
}
```

VerificadorEquivalenciaSemanticaDocumental es una dependencia distinta del
marcador. Sin una implementacion homologada independiente, la integracion
queda cerrada; una autocomprobacion del mismo conector no es garantia
productiva suficiente. Esta interfaz aun no modela identidad, digest ni
atestacion del verificador: el bootstrap productivo no debe habilitarla
hasta incorporar y cotejar esas pruebas contra un componente distinto del
marcador.

```go
type VerificadorEvidenciasRenderizadoDocumentalV3 interface {
	VerificarEvidenciaRenderizadoDocumentalV3(
		context.Context,
		SolicitudVerificacionEvidenciaDocumentalV3,
	) (MetadatosComprobacionEvidenciaDocumentalV3Nominal, error)
}
```

VerificadorEvidenciasRenderizadoDocumentalV3 es un conector intercambiable.
Su salida es nominal; solo el servicio precompuesto puede usarla junto con
relectura durable y CAS, sin exponerla como autoridad a handlers.

```go
type VerificadorOrdenDespachoDocumentalV3 interface {
	VerificarOrdenDespachoDocumentalV3(
		context.Context,
		SolicitudComprobarOrdenDespachoDocumentalV3,
	) (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3, error)
}
```

VerificadorOrdenDespachoDocumentalV3 es un puerto intercambiable de KMS.
Devuelve un resultado crudo nominal; no puede promover capacidades.

```go
type VerificadorPasarelaCobro interface {
	VerificarNotificacionCobro(context.Context, NotificacionCobro) (ResultadoOperacionCobro, error)
	VerificarNotificacionDevolucion(context.Context, NotificacionCobro, ReferenciaDevolucionCobro) (ResultadoDevolucionCobro, error)
}
```

VerificadorPasarelaCobro es la unica frontera autorizada para convertir
una recepcion custodiada o una respuesta remota en evidencia verificada.
Sus implementaciones verifican criptografia, audiencia, vigencia y replay
antes de usar las fabricas de dominio *Verificada.

```go
type VerificadorPerfilPublicadoMaterialV2 interface {
	VerificarPerfilPublicadoMaterialV2(
		context.Context,
		SolicitudVerificarPerfilPublicadoMaterialV2,
	) error
}
```

VerificadorPerfilPublicadoMaterialV2 debe consultar el catalogo autoritativo
de perfiles homologados y cotejar referencia, version, huella y bytes
exactos. No puede limitarse a volver a verificar la firma.

```go
type VerificadorPlanMaterialAlmacenV2 interface {
	VerificarPlanMaterialAlmacenV2(
		context.Context,
		SolicitudVerificarPlanMaterialAlmacenV2,
	) (ResultadoVerificacionPlanMaterialAlmacenV2, error)
}

type VerificadorReferenciaReciboMaterialV2 interface {
	VerificarReferenciaReciboMaterialV2(
		context.Context,
		SolicitudReservarReferenciaReciboMaterialV2,
		ResultadoReferenciaReciboMaterialV2,
	) error
}
```

VerificadorReferenciaReciboMaterialV2 debe consultar el mismo registro
durable por la huella estable y confirmar que la referencia devuelta es la
original. Un comprobador de forma no cumple este contrato.

```go
type VinculoEjecucionEscrituraAlmacenDocumental struct {
	// Has unexported fields.
}
```

VinculoEjecucionEscrituraAlmacenDocumental conserva la orden consumida
nominal y su vinculo estable. Solo acredita correlacion estructural:
la autoridad procede del servicio de aplicacion que posee KMS, registro CAS
y conector de almacen; este valor nunca debe cruzar una frontera de entrada.

```go
func NuevoVinculoEjecucionEscrituraAlmacenDocumental(
	ordenConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
) (VinculoEjecucionEscrituraAlmacenDocumental, error)

func (v VinculoEjecucionEscrituraAlmacenDocumental) Format(estado fmt.State, _ rune)

func (v VinculoEjecucionEscrituraAlmacenDocumental) GoString() string

func (v VinculoEjecucionEscrituraAlmacenDocumental) LogValue() slog.Value

func (VinculoEjecucionEscrituraAlmacenDocumental) MarshalBinary() ([]byte, error)

func (VinculoEjecucionEscrituraAlmacenDocumental) MarshalJSON() ([]byte, error)

func (VinculoEjecucionEscrituraAlmacenDocumental) MarshalText() ([]byte, error)

func (VinculoEjecucionEscrituraAlmacenDocumental) String() string

func (*VinculoEjecucionEscrituraAlmacenDocumental) UnmarshalBinary([]byte) error

func (*VinculoEjecucionEscrituraAlmacenDocumental) UnmarshalJSON([]byte) error

func (*VinculoEjecucionEscrituraAlmacenDocumental) UnmarshalText([]byte) error

func (v VinculoEjecucionEscrituraAlmacenDocumental) Validar() error

func (v VinculoEjecucionEscrituraAlmacenDocumental) ValidarContra(
	ordenConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
) error

type VinculoEstableActivacionDocumentalV3 struct {
	ReservaRef               string
	IndiceIdempotenciaHMAC   string
	HuellaSolicitudHMAC      string
	Manifiesto               ManifiestoEjecucionDocumentalV3
	ConsumoDecision          ConsumoDecisionEjecucionDocumentalV3
	OrdenConsumoDurableV4Ref string
}
```

VinculoEstableActivacionDocumentalV3 es el DTO nominal unico que compromete
la intencion estable de activacion. Excluye ActivadaEn deliberadamente
para que un reintento posterior recupere el mismo cercado. Sus campos son
inspeccionables, pero su forma y huella no conceden autoridad: el registro
debe reconstruirlo desde las filas V3/V4 durables y compararlo dentro de la
transaccion que autoriza el efecto.

```go
func (v VinculoEstableActivacionDocumentalV3) Format(estado fmt.State, _ rune)

func (v VinculoEstableActivacionDocumentalV3) GoString() string

func (v VinculoEstableActivacionDocumentalV3) HuellaSHA256() (string, error)

func (v VinculoEstableActivacionDocumentalV3) LogValue() slog.Value

func (VinculoEstableActivacionDocumentalV3) MarshalBinary() ([]byte, error)

func (VinculoEstableActivacionDocumentalV3) MarshalJSON() ([]byte, error)

func (VinculoEstableActivacionDocumentalV3) MarshalText() ([]byte, error)

func (VinculoEstableActivacionDocumentalV3) String() string

func (*VinculoEstableActivacionDocumentalV3) UnmarshalBinary([]byte) error

func (*VinculoEstableActivacionDocumentalV3) UnmarshalJSON([]byte) error

func (*VinculoEstableActivacionDocumentalV3) UnmarshalText([]byte) error

func (v VinculoEstableActivacionDocumentalV3) Validar() error

type VinculoPoliticaInmutabilidadDocumental struct {
	PoliticaRef                string
	Version                    uint64
	HuellaSHA256               string
	Requisitos                 RequisitosAlmacenObjetos
	HuellaRequisitosSHA256     string
	HuellaCapacidadesSHA256    string
	RetencionHasta             time.Time
	ExigeInmovilizacionInicial bool
}
```

VinculoPoliticaInmutabilidadDocumental identifica la version exacta de una
politica gobernada y el perfil de capacidades que esta exige. El nombre se
conserva por compatibilidad conceptual, pero Versionado no se interpreta
como WORM ni como bloqueo legal: Retencion y BloqueoLegal son requisitos
independientes y solo producen esos estados cuando la politica los exige.

Este valor sigue siendo declarativo. La salida del conector de capacidades
tambien es nominal y debe cotejarse en la composicion privada de aplicacion.

```go
func NuevoVinculoPoliticaInmutabilidadDocumental(
	politicaRef string,
	version uint64,
	huellaSHA256 string,
	requisitos RequisitosAlmacenObjetos,
	capacidades CapacidadesAlmacenObjetos,
	retencionHasta time.Time,
	exigeInmovilizacionInicial bool,
) (VinculoPoliticaInmutabilidadDocumental, error)

func (v VinculoPoliticaInmutabilidadDocumental) Validar() error

type VinculosCrudosVerificacionOrdenDespachoDocumentalV3 = documentalcanonico.VinculosMaterialDespachoV3

type VinculosOperacionAlmacen struct {
	OperacionRef        string
	CargaRef            string
	Clasificacion       string
	SujetoSeudonimoHMAC string
	HuellaSolicitudHMAC string
	EfectoRef           string
	ObjetoVinculado     ReferenciaObjetoAlmacen
}
```

VinculosOperacionAlmacen contiene datos no autoritativos que el constructor
coteja uno a uno con el RecursoAutorizable ya evaluado. No es una capacidad.
ObjetoVinculado es obligatorio solo en planes de lectura, promocion o
retencion; en el resto debe ser el valor cero.

```go
type ZonaAlmacen = almacencanonico.Zona
```

ZonaAlmacen separa tecnicamente objetos que aun no son confiables de los que
ya pueden incorporarse a un expediente. No representa un estado de negocio
configurable.
