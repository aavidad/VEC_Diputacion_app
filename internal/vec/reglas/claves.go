package reglas

// Identificadores de los catálogos de reglas y claves de sus entradas. Los
// módulos consumidores usan estas constantes en lugar de repetir literales.
const (
	CatalogoBolsa                = "vec.bolsa.reglas"
	ModuloBolsa                  = "bolsa"
	CatalogoContratacionTemporal = "vec.contratacion_temporal.reglas"
	ModuloContratacionTemporal   = "contratacion_temporal"
	// CatalogoBolsaRolesSegregacion lista las operaciones de Bolsa que valida
	// una segunda persona (duda 6 de RRHH).
	CatalogoBolsaRolesSegregacion = "vec.bolsa.roles_segregacion"
	BolsaSegundaPersona           = "s01.segunda_persona"
	// CatalogoCircuitoFirmaCT agrupa los pasos de firma de los documentos
	// de Contratación temporal; cada entrada es un paso.
	CatalogoCircuitoFirmaCT = "vec.contratacion_temporal.circuito_firma"
	// MunicipioSedeDiputacion es la sede para el cómputo de plazos cuando el
	// consumidor no conoce otra (Granada, código INE 18087).
	MunicipioSedeDiputacion = "municipio:ine:18087"
)

// Reglas de Bolsa.
const (
	BolsaOrdenPrelacion               = "b01.orden_prelacion"
	BolsaIntentosContacto             = "b02.intentos_contacto"
	BolsaSeparacionIntentos           = "b02.separacion_intentos"
	BolsaProcesosSinContacto          = "b03.procesos_sin_contacto"
	BolsaFranjaLlamadas               = "b04.franja_llamadas"
	BolsaPlazoRespuesta               = "b05.plazo_respuesta"
	BolsaCorreoNoAbrePlazo            = "b06.correo_no_abre_plazo"
	BolsaFueraDePlazo                 = "b07.fuera_de_plazo"
	BolsaSinRespuestaBaja             = "b08.sin_respuesta_baja"
	BolsaSiguienteCandidato           = "b09.siguiente_candidato"
	BolsaPlazoPublicacion             = "b10.plazo_publicacion"
	BolsaAcreditarRenunciaJustificada = "b11.acreditar_renuncia_justificada"
	BolsaPeriodoMatrimonio            = "b11.periodo_matrimonio"
	BolsaRenunciaNoJustificada        = "b12.renuncia_no_justificada"
	BolsaRenunciaNombramientoEnCurso  = "b13.renuncia_nombramiento_en_curso"
	BolsaReposicionGeneral            = "b14.reposicion_general"
	BolsaReposicionAcumulacionTareas  = "b14.reposicion_acumulacion_tareas"
	BolsaRecuperaPosicion             = "b15.recupera_posicion"
	BolsaPrestaServicios              = "b16.presta_servicios"
	BolsaAvisoEncadenamiento          = "b17.aviso_encadenamiento"
	BolsaPausaVoluntaria              = "b18.pausa_voluntaria"
	BolsaVacanteDuracionMaxima        = "b19.vacante_duracion_maxima"
	BolsaSAEDuracionMaxima            = "b20.sae_duracion_maxima"
	BolsaPlazoDocumentacion           = "b21.plazo_documentacion"
	BolsaDocumentosIncorporacion      = "b22.documentos_incorporacion"
	BolsaPlazoIncorporacion           = "b23.plazo_incorporacion"
	BolsaConsecuencias                = "b24.consecuencias"
	BolsaVigencia                     = "b25.vigencia_bolsa"
	BolsaAgotamiento                  = "b26.agotamiento"
	BolsaContactoOrigenConvoca        = "b29.contacto_origen_convoca"
	// BolsaPrefijoSanciones agrupa las consecuencias de una sanción: cada
	// entrada con este prefijo es una consecuencia que RRHH puede resolver,
	// de modo que añadir otra no exige cambiar código.
	BolsaPrefijoSanciones = "b24.sancion."
	BolsaEstadosRecurso   = "b24.recurso_estados"
	// BolsaEstadosRecursoRevocatorios lista los estados del recurso que
	// revierten la sanción (readmisión); sin la entrada ninguno la revierte.
	BolsaEstadosRecursoRevocatorios = "b24.recurso_revierte"
	// BolsaCamposPortal es la lista de datos de «Mi bolsa» (duda 17).
	BolsaCamposPortal = "b29.campos_mi_bolsa"
	// BolsaPortalCandidato fija el modo de las acciones propias del candidato
	// (dudas 3 y 18): quién valida, qué contacto abre el plazo y desde qué
	// situaciones se admite cada solicitud.
	BolsaPortalCandidato = "b29.portal_candidato"
)

// Reglas de Bolsa que se consultan por prefijo: cada entrada es una opción
// del catálogo y añadir otra no exige cambiar el código.
const (
	// AtributoEfecto es el efecto de una consecuencia «b24.sancion.*» sobre
	// la situación (ninguna, pausar, excluir). Las consecuencias con efecto
	// «excluir» son también las causas de baja definitiva (art. 11) que RRHH
	// elige al excluir: una sola fuente para sanciones y bajas.
	AtributoEfecto = "efecto"
	// BolsaPrefijoTransicionesSituacion + situación de origen es una lista con
	// los destinos admitidos desde ella. Sin entrada rige la tabla compilada.
	// Bolsa publica al arrancar la tabla resultante como política de la base
	// de datos (migración 000032 de bolsa_llamamientos), que es la que decide.
	BolsaPrefijoTransicionesSituacion = "b28.transiciones."
	// AtributoModalidades es la lista de modalidades de nombramiento a las que
	// se aplica una regla de reposición distinta de la general.
	AtributoModalidades = "modalidades"
)

// Entradas del paquete de ejemplo con esos prefijos.
const (
	BolsaTransicionesRenuncia   = BolsaPrefijoTransicionesSituacion + "renuncia"
	BolsaTransicionesDisponible = BolsaPrefijoTransicionesSituacion + "disponible"
)

// Reglas de Contratación temporal.
const (
	CTPlazoAnalisis               = "c01.plazo_analisis"
	CTPlazoInformes               = "c02.plazo_informes"
	CTPlazoFiscalizacion          = "c03.plazo_fiscalizacion"
	CTPlazoSubsanacion            = "c04.plazo_subsanacion"
	CTMotivosRectificacion        = "c05.motivos_rectificacion"
	CTAltaSeguridadSocial         = "c06.alta_seguridad_social"
	CTJornadaCompleta             = "c07.jornada_completa"
	CTDuracionAcumulacionTareas   = "c08.acumulacion_tareas"
	CTDuracionProgramasTemporales = "c08.programas_temporales"
	CTDuracionVacante             = "c08.vacante"
	CTDuracionSustitucion         = "c08.sustitucion"
	CTDuracionCircunstancias      = "c08.circunstancias_produccion"
	CTModificacionFaseRetorno     = "c09.modificacion_fase_retorno"
	CTCierreExpediente            = "c10.cierre_expediente"
	CTCausasCese                  = "c11.causas_cese"
	// CTAcreditacionIncorporacion fija el documento que acredita la
	// incorporación: «valor» es el tipo general y «valor_<modalidad>» el de
	// una modalidad concreta; «roles_confirman» lista los perfiles del
	// centro que la confirman (duda 11).
	CTAcreditacionIncorporacion = "c12.acreditacion_incorporacion"
)

// Atributos de las reglas de Contratación temporal.
const (
	// AtributoCierreSinCese en c10: con el valor «admitido» se ofrece el
	// cierre administrativo sin cese; con cualquier otro, no.
	AtributoCierreSinCese = "cierre_sin_cese"
	// AtributoRolesConfirmanIncorporacion en c12: perfiles del centro que
	// confirman la incorporación, separados por comas.
	AtributoRolesConfirmanIncorporacion = "roles_confirman"
	// PrefijoValorModalidad en una regla de lista: «valor_<modalidad>».
	PrefijoValorModalidad = "valor_"
)
