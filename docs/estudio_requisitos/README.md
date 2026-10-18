# Estudio de requisitos del portal de RRHH

Lo vigente de este estudio está ahora en los documentos de producto de
[docs/producto/](../producto/):

- orden de las fuentes, puerta institucional antes de programar y decisiones
  de producto: [PRD §6, «De dónde salen las reglas»](../producto/PRD.md#de-dónde-salen-las-reglas)
  y el resto del [PRD](../producto/PRD.md);
- decisiones técnicas (arquitectura, conectores, seguridad, despliegue):
  [TRD](../producto/TRD.md);
- qué falta y en qué orden: [PLAN_IMPLEMENTACION](../producto/PLAN_IMPLEMENTACION.md).

Esta carpeta guarda los estudios, fichas y actas de cada materia, con sus
citas completas. La versión anterior de este índice, con el estudio de julio de
2026, está en la historia de Git de este fichero.

## Decisiones de despliegue que aún no están en el TRD

Tomadas en julio de 2026 para el paso a datos reales; las valida Sistemas. El
detalle está en [despliegue_contenedores_y_escalado.md](despliegue_contenedores_y_escalado.md)
y en el [Manual 00 para Sistemas](manual_sistemas_preparacion_plataforma.md),
que es la puerta previa a instalar.

- Primera implantación austera: tres máquinas virtuales (NGINX, servicios y
  datos) con Podman Quadlet o Docker Compose. Dos solo como excepción con
  riesgo aceptado; una sola, solo sin datos reales.
- Kubernetes, colas externas, almacén S3 distribuido, alta disponibilidad y
  autoescalado se aplazan, no se descartan. Imágenes OCI, servicios sin estado
  y puertos sustituibles permiten pasar después sin reescribir.
- En esa fase, PostgreSQL lleva los trabajos duraderos y la bandeja de salida.
  Las copias externas, la restauración, la auditoría remota, el antivirus y los
  controles ENS y RGPD no se aplazan.
- PostgreSQL es la base inicial; Oracle sería un adaptador y unas migraciones
  propias si algún día hiciera falta.
- Se prepara para ENS de categoría ALTA, sin declarar conformidad antes de la
  categorización y la auditoría.

Siguen pendientes, además de lo que recoge el PRD §14: volumetría,
disponibilidad, tiempos de recuperación (RTO y RPO), presupuesto operativo e
inventario de la infraestructura de la Diputación (red, balanceo, almacenamiento,
identidad, certificados, copias y segundo centro de datos).

## Índice de la carpeta

**Decisiones, actas y dudas**

- [Decisiones de RRHH sobre Contratación temporal y Bolsa (2/10/2026)](decisiones_rrhh_2026-10-02.md)
- [Acta del recorrido de Contratación temporal y Bolsa (5/10/2026)](acta_recorrido_ct_bolsa_20261005.md)
- [Contraste de las dudas de RRHH con otras aplicaciones](dudas_contraste_otras_apps_2026-09-30.md)
- [Petición de RRHH: transcripción y lectura técnica](peticion_rrhh_transcripcion_y_lectura.md)
- [Petición a Informática sobre integraciones](peticion_informatica_integraciones_2026-09-23.md)
- [Referencia del portal público de CONVOCA](referencia_convoca_publica_2026-09-23.md)

**Normativa y modelo**

- [Matriz normativa de Recursos Humanos](matriz_normativa_rrhh_2026.md)
- [RPT, plazas, puestos, ocupaciones y vacantes](modelo_historico_rpt_plazas_puestos_y_vacantes.md)
- [Calendario hábil y laboral con historia](calendario_habil_laboral_historico.md)
- [Turnos, festivos y compensaciones](turnos_festivos_y_compensaciones.md)
- [Materias económicas, reservadas y relaciones laborales](materias_reservadas_economicas_y_relaciones_laborales.md)
- [Archivo documental de RRHH](archivo_documental_rrhh_relacionado.md)
- [Datos personales de las inscripciones y RGPD](datos_personales_inscripciones_rgpd_2026-09-29.md)
- [Plazos configurables](plazos_configurables_2026-09-30.md)
- [Lo que queda del análisis integral](analisis_integral_rrhh.md)

**Fichas por módulo**

- Base: [fichas B1 a B5](fichas_base_B1_B5_2026-09-25.md),
  [requisitos y oportunidades (S1 y B15)](diseno_s1_b15_requisitos_2026-09-23.md)
- Bolsa y CONVOCA: [adaptación de Bolsa a CONVOCA](ficha_adaptacion_bolsa_convoca_2026-09-16.md),
  [inscripción externa de CONVOCA](convoca_inscripcion_externa_contrato_2026-10-01.md)
- Baremación: [baremación configurable y datos de oficio](baremacion_configurable_jornada_y_datos_de_oficio.md),
  [baremador en concursos y provisión](integracion_baremador_concursos_provision.md),
  [recorrer el baremador en local](recorridos_baremador_local.md),
  [pendientes de la versión 2](pendientes_v2.md)
- Cronos: [ficha](ficha_cronos_2026-09-23.md),
  [seguridad y despliegue](seguridad_y_despliegue_cronos.md)
- Dietas: [ficha](ficha_dietas_2026-09-23.md),
  [inventario](inventario_dietas_2026-10-01.md),
  [catálogo de tarifas](dietas_catalogo_rrhh_2026-10-01.md),
  [liquidación](dietas_liquidacion_2026-10-01.md),
  [informes](dietas_informes_2026-10-01.md),
  [ensayo de devengo](dietas_simulacion_devengo_2026-10-01.md)
- [Cronos y Dietas: qué falta para instalar](cronos_dietas_instalacion_2026-10-04.md)
- Provisión: [ficha](ficha_provision_2026-10-04.md),
  [recorrido y persistencia](provision_recorrido_y_persistencia_2026-10-01.md)
- Otros módulos: [acción social](ficha_accion_social_2026-10-04.md),
  [canal interno de información](ficha_canal_interno_informacion_2026-10-04.md),
  [formación](ficha_formacion_2026-10-04.md),
  [igualdad](ficha_igualdad_2026-10-04.md),
  [incompatibilidades](ficha_incompatibilidades_2026-10-04.md),
  [nóminas](ficha_nominas_2026-10-04.md),
  [prevención de riesgos laborales](ficha_prevencion_riesgos_laborales_2026-10-04.md),
  [régimen disciplinario](ficha_regimen_disciplinario_2026-10-04.md),
  [relaciones sindicales](ficha_relaciones_sindicales_2026-10-04.md)
- [Borrador de certificado de servicios (CER-001)](certificados_borrador_servicios.md)

**Acceso, administración, firma y auditoría**

- [Acceso interno de técnicos, RRHH y administración](acceso_interno_tecnicos_administracion.md)
- [Administración de perfiles](administracion_perfiles_2026-09-30.md)
- [Contrato con el verificador de firma](contrato_verificador_firma_vec.md)
- [Cobertura de auditoría (3/10/2026)](cobertura_auditoria_2026-10-03.md)

**Diseño, ayuda y trabajo en equipo**

- [Sistema de diseño y temas](sistema_diseno_y_temas.md),
  [colores de los temas](temas_colores_fuentes_2026-09-30.md),
  [temas instalables](temas_instalables_2026-10-04.md)
- [Accesibilidad personalizable, ayuda, audio y asistente](ayuda_documentacion_audio_y_asistente.md)
- [Espacio de trabajo colaborativo y agenda](espacio_trabajo_colaborativo_y_agenda.md)

**Plataforma, rendimiento y método**

- [Despliegue, contenedores y escalado](despliegue_contenedores_y_escalado.md)
- [Manual 00 para Sistemas](manual_sistemas_preparacion_plataforma.md)
- [Aguante del portal público ante miles de personas](rendimiento_carga_20261006.md)
- [Brechas para un producto profesional](brechas_para_producto_profesional.md)
- [Método Orquesta V06 (suspendido)](metodo_desarrollo_orquestado_v06.md)

La carpeta [capturas/](capturas/) guarda imágenes de apoyo de las actas.
