package reglas

// Identificadores de los catálogos de reglas y claves de sus entradas. Los
// módulos consumidores usan estas constantes en lugar de repetir literales.
const (
	CatalogoBolsa                = "vec.bolsa.reglas"
	ModuloBolsa                  = "bolsa"
	CatalogoContratacionTemporal = "vec.contratacion_temporal.reglas"
	ModuloContratacionTemporal   = "contratacion_temporal"
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
	// BolsaPrefijoSanciones agrupa las consecuencias de una sanción: cada
	// entrada con este prefijo es una consecuencia que RRHH puede resolver,
	// de modo que añadir otra no exige cambiar código.
	BolsaPrefijoSanciones = "b24.sancion."
	BolsaEstadosRecurso   = "b24.recurso_estados"
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
)
