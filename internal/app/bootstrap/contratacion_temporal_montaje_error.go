package bootstrap

import (
	"errors"
	"log/slog"

	"vec-diputacion-granada/internal/shared/telemetria"
)

// falloMontajeCT conserva la causa sin transportar su mensaje al registro.
// Las etiquetas admitidas proceden únicamente de los puntos de composición.
type falloMontajeCT struct {
	constructor string
	causa       error
}

func (f *falloMontajeCT) Error() string { return etiquetaConstructorMontajeCT(f.constructor) }
func (f *falloMontajeCT) Unwrap() error { return f.causa }

func registrarFalloMontajeCT(fallo error) {
	slog.Warn("contratacion_temporal_no_disponible", "componente", ComponenteFalloArranque(fallo),
		"causa", telemetria.ClaseErrorArranque(fallo), "constructor", constructorFalloMontajeCT(fallo))
}

func marcarFalloMontajeCT(constructor string, err error) error {
	if err == nil {
		return nil
	}
	var previo *falloMontajeCT
	if errors.As(err, &previo) && previo != nil {
		return err
	}
	return &falloMontajeCT{constructor: constructor, causa: err}
}

// constructorFalloMontajeCT entrega solo un valor de cardinalidad cerrada.
func constructorFalloMontajeCT(err error) string {
	var fallo *falloMontajeCT
	if errors.As(err, &fallo) && fallo != nil {
		return etiquetaConstructorMontajeCT(fallo.constructor)
	}
	return "ct.montaje.sin_etiqueta"
}

func etiquetaConstructorMontajeCT(constructor string) string {
	switch constructor {
	case "ajustes_resolutor",
		"alta",
		"analisis",
		"asignacion",
		"auditoria_circuito",
		"auditoria_externa",
		"auditoria_orden_ct",
		"auditoria_rutas",
		"autorizaciones_bases",
		"autorizaciones_firma",
		"borrador_bolsa",
		"campos_mi_bolsa",
		"cancelacion",
		"capacidad_no_compuesta",
		"catalogo",
		"catalogo_fronteras",
		"catalogos_alta",
		"cobertura",
		"comunicaciones",
		"conexiones_autorizacion",
		"configuracion_ajustes",
		"configuracion_analisis",
		"configuracion_auditoria",
		"configuracion_bases",
		"configuracion_documental",
		"configuracion_firma",
		"configuracion_incorporacion",
		"configuracion_plantillas",
		"configurar_emisor_nativo",
		"configurar_necesidades",
		"configurar_operacion_nativa",
		"configurar_verificador_rpt",
		"consulta_circuito",
		"consultas_autorizacion",
		"consultas_rrhh",
		"contexto_reincorporacion",
		"continuidad_incorporacion",
		"definicion_circuito",
		"dependencias",
		"derivacion_nombramiento",
		"descriptores_b2",
		"efectos_bolsa",
		"entrega_peticion",
		"envolver_circuito_firma",
		"envolver_rutas_firma",
		"estadisticas",
		"evento_plazo",
		"ficha_ginpix",
		"firma_admision",
		"firma_almacen_fuente",
		"firma_almacen_registro",
		"firma_autoridad_contexto",
		"firma_autoridad_sesion",
		"firma_catalogo_motivos",
		"firma_catalogo_plan",
		"firma_competencia",
		"firma_competencia_plan",
		"firma_descriptor",
		"firma_documento",
		"firma_dsn_contexto",
		"firma_emisor_consulta",
		"firma_emisor_recuperacion",
		"firma_emisor_vec",
		"firma_fuente_nominal",
		"firma_fuente_plan",
		"firma_fuente_vec",
		"firma_material_consulta",
		"firma_material_lecturas",
		"firma_material_recuperacion",
		"firma_material_registro",
		"firma_motivos",
		"firma_motivos_custodia",
		"firma_motivos_detalle",
		"firma_motivos_original",
		"firma_nominal",
		"firma_plan_publicado",
		"firma_pool_contexto",
		"firma_publicacion_plan",
		"firma_requisitos",
		"firma_requisitos_lecturas",
		"firma_resolutor_contexto",
		"firma_resolutor_plan",
		"firma_resolver_motivo_detalle",
		"firma_retirada",
		"firma_seleccion",
		"firma_selector_central",
		"fiscalizacion",
		"frontera_original",
		"fronteras",
		"fronteras_auditoria",
		"fronteras_baremo",
		"fronteras_bases",
		"fronteras_bolsa",
		"fronteras_documental",
		"fuente_necesidad",
		"fuente_rpt",
		"fuentes_propuesta_nativa",
		"gobierno_baremo",
		"identidad_bolsa",
		"identidad_mi_bolsa",
		"identidad_operativa",
		"incorporacion_config",
		"incorporacion_legada",
		"informe_juridico",
		"informe_subsanacion",
		"lector_flujos",
		"lector_resultado_nativo",
		"lector_visual_circuito",
		"lector_visual_legado",
		"lectura_intervencion",
		"lectura_rpt",
		"lecturas_cuadro",
		"lecturas_descriptor_descarga",
		"lecturas_detalle",
		"lecturas_emisor",
		"lecturas_firma",
		"lecturas_historico",
		"lecturas_lector_circuito",
		"lecturas_manejador_circuito",
		"lecturas_material_circuito",
		"lecturas_material_descarga",
		"lecturas_motivo_circuito",
		"lecturas_motivo_detalle",
		"lecturas_proceso_auditoria",
		"lecturas_requisitos",
		"lecturas_servicio_circuito",
		"llamamiento",
		"manejador_resolucion",
		"mi_bolsa",
		"migracion_urgencia",
		"motivos_rectificacion",
		"numeracion",
		"operacion_bolsa_nativa",
		"organizacion",
		"perfil_emision_bolsa",
		"perfiles_b2",
		"perfiles_circuito_firma",
		"perfiles_fijos",
		"perfiles_incorporacion",
		"peticiones_centro",
		"plantillas_borrador",
		"politica_ofertas",
		"pool_fuente",
		"pool_motivos_preflight",
		"preflight_circuito",
		"preparacion_bases",
		"propuesta",
		"protocolos_incorporacion",
		"recibo_respuesta",
		"recuperador_necesidad",
		"registrador_auditoria",
		"registrador_auditoria_consulta",
		"registro_empleado",
		"reglas_analisis",
		"reglas_portal",
		"repositorio_alta_nativa",
		"repositorio_cese_nativo",
		"requisitos_auditoria",
		"requisitos_auditoria_consulta",
		"requisitos_bolsa_nativa",
		"requisitos_documental",
		"requisitos_efectos_bolsa",
		"requisitos_firma",
		"requisitos_firma_nominal",
		"requisitos_incorporacion_b2",
		"requisitos_incorporacion_configurada",
		"requisitos_incorporacion_legada",
		"requisitos_mi_bolsa",
		"requisitos_plantillas",
		"requisitos_portal",
		"resolucion_formalizacion",
		"respuesta_recibida",
		"resultados_fiscalizacion",
		"rutas",
		"rutas_ajustes",
		"rutas_auditoria",
		"rutas_b2",
		"rutas_bases",
		"rutas_documental",
		"rutas_firmas_r5",
		"rutas_plantillas",
		"seguimiento_cese",
		"seguimiento_incorporacion",
		"segundo_tecnico",
		"selector_auditoria",
		"selector_reincorporacion",
		"servicio_alta_nativa",
		"servicio_cese_nativo",
		"soporte_bolsa",
		"soporte_documental",
		"soporte_plantillas",
		"soportes_auditoria",
		"token_finalizacion",
		"verificador_rpt",
		"vinculo_emision_bolsa":
		return "ct.montaje." + constructor
	}
	return "ct.montaje.sin_etiqueta"
}
