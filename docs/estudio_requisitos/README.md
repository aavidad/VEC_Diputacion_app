# Estudio de requisitos del portal transversal de RRHH

Estado: **documentación de trabajo; todavía no es el pliego ni la especificación final**.

Fecha de corte actual: 16 de julio de 2026.

## Orden de las fuentes

1. Necesidades expresadas por RRHH y decisiones de la Diputación.
2. `Peticion.pdf`, pliegos y evidencias de `convoca_dipgra`.
3. Normativa vigente, categorización, análisis de riesgos y criterios de los órganos
   competentes.
4. Procedimientos reales levantados con las unidades que los ejecutan.
5. Patrones útiles de otras administraciones.
6. Prototipos y documentación técnica actual, que deberán adaptarse a las decisiones
   anteriores.

## Documentos incorporados

- [Transcripción y lectura técnica de la petición de RRHH](peticion_rrhh_transcripcion_y_lectura.md).
- [Baremacion configurable, jornada y servicios obtenidos de oficio](baremacion_configurable_jornada_y_datos_de_oficio.md).
- [Integracion del Baremador de puestos singularizados y otros procesos](integracion_baremador_concursos_provision.md).
- [Modelo historico de RPT, plazas, puestos, ocupaciones y vacantes](modelo_historico_rpt_plazas_puestos_y_vacantes.md).
- [Brechas para un producto profesional, seguro y trazable](brechas_para_producto_profesional.md).
- [Sistema de diseño, plantillas y temas visuales](sistema_diseno_y_temas.md).
- [Ayuda, documentación, audio y asistente](ayuda_documentacion_audio_y_asistente.md).
- [Arquitectura de despliegue, contenedores y escalado](despliegue_contenedores_y_escalado.md).
- [Manual 00 para Sistemas: preparación e instalación de la plataforma](manual_sistemas_preparacion_plataforma.md).
- [Seguridad y despliegue del módulo Cronos](seguridad_y_despliegue_cronos.md).
- [Acceso interno de técnicos, RRHH y administración](acceso_interno_tecnicos_administracion.md).
- [Matriz inicial de perfiles, roles y ambitos](../portal_vec/matriz_roles_y_ambitos.md).
- [Método de desarrollo asistido con Orquesta V06](metodo_desarrollo_orquestado_v06.md).
- [Inventario de manuales públicos descargados](../referencias_portales_aapp/README.md).
- [Comparativa y composición recomendada](../referencias_portales_aapp/comparativa_y_composicion_recomendada.md).
- [Decisión de espacios y persistencia PostgreSQL/Oracle](../referencias_portales_aapp/decision_espacios_y_persistencia.md).
- [Registro cronologico de decisiones, alternativas y motivos](../portal_vec/registro_decisiones.md).
- [Dominio de Bolsa, autobaremacion y revision firmada](../portal_vec/dominio_y_autobaremacion.md).
- [Versiones gobernadas de convocatoria de Bolsa](../portal_vec/convocatorias_gobernadas.md).
- [Firma, CSV, QR y cotejo documental](../portal_vec/firma_csv_qr_y_cotejo.md).
- [Almacen documental seguro e intercambiable](../portal_vec/almacen_documental_seguro.md).
- [Pagos, tasas y conciliacion](../portal_vec/pagos_tasas_y_conciliacion.md).
- [Seguridad, privilegios y seguridad por filas en PostgreSQL](../portal_vec/seguridad_persistencia_postgresql.md).
- [Atestacion criptografica y registro durable de decisiones](../portal_vec/atestacion_criptografica_decisiones.md).

El `README` actua como indice de la memoria viva. El registro de decisiones
explica el por que; las especificaciones de capacidad describen el que y el
como; las pruebas y el codigo acreditan que parte esta ya implantada. En el
documento final no se confundira una decision, un prototipo y una capacidad
productiva verificada.

## Decisiones ya incorporadas

- Portal genérico de RRHH; Bolsa es el primer módulo, no toda la aplicación.
- Persona canónica y datos reutilizables con procedencia, versión y autorización.
- Portal externo, portal del empleado, espacio del responsable y área interna de RRHH
  separados, aunque compartan núcleo.
- Una sesión tiene un único perfil; los privilegios no se suman.
- Arquitectura hexagonal y conectores sustituibles.
- PostgreSQL inicial; Oracle futuro mediante adaptador y migraciones propios.
- Configuración funcional gobernada para la variabilidad ordinaria.
- API, web, CLI, MCP y futuro escritorio como adaptadores de los mismos casos de uso.
- Máxima trazabilidad, seguridad desde el diseño y preparación para ENS categoría ALTA,
  sin declarar conformidad antes de su categorización y auditoría.
- Documentación y manuales mantenidos durante el desarrollo.
- Sistema de diseño central con temas heredables y acentos opcionales por módulo.
- Preferencias visuales por persona dentro de variantes accesibles aprobadas, sin CSS
  arbitrario ni almacenamiento de diagnósticos.
- Una fuente de ayuda gobernada generará HTML accesible, manuales, PDF, audio, búsqueda y
  corpus del asistente, evitando mantener contenidos contradictorios.
- Cronos será un enclave exclusivamente interno de Mulhacén, sin rutas, artefactos, datos,
  mapas, conectores o credenciales compartidos con el portal público.
- Módulo lógico y contenedor no son equivalentes: se separará el despliegue por frontera de
  seguridad, escalado, fallo o ciclo de vida.
- Producción se prepara para clúster público e interno separados, NGINX sobre Gateway API y
  escalado HPA/KEDA; la plataforma exacta será validada por Sistemas.
- La primera implantación con datos reales será austera: Bolsa en tres VM —NGINX, servicios
  y datos— con Podman Quadlet o Docker Compose v2; dos VM solo como excepción con riesgo
  aceptado y una VM únicamente sin datos reales.
- Kubernetes, broker, S3 distribuido, alta disponibilidad y autoescalado se aplazan, no se
  descartan. Se mantienen imágenes OCI, servicios sin estado y puertos sustituibles para
  migrar sin reescribir Bolsa.
- En la fase austera PostgreSQL aporta trabajos duraderos y bandeja de salida; los
  documentos pueden usar un volumen dedicado detrás del puerto de almacenamiento. Copias
  externas, restauración, auditoría remota, cuarentena, antivirus y controles ENS/RGPD no
  se aplazan.
- El Manual 00 de Sistemas y sus anexos de producto/versión son una puerta previa al
  despliegue de la aplicación.
- Orquesta V06 queda estudiada pero temporalmente suspendida por indicacion del
  responsable mientras se repara. El trabajo se paraleliza con subagentes
  directos separados; un resultado de agente no equivale a codigo integrado
  hasta su revision y pruebas globales.
- El autobaremo ciudadano y el baremo administrativo son resultados distintos.
  El candidato no puede autovalidar estados y el resultado oficial solo computa
  decisiones tecnicas eficaces.
- La aceptacion o rechazo de un merito/documento se expresa mediante decisiones
  firmadas e inmutables. Una revision inspectora crea una rectificacion
  append-only, recalcula y conserva todas las versiones; nunca sobrescribe ni
  borra el pasado.
- Se adopta una composicion de patrones oficiales: merito y puntuacion
  declarada/validada de VEC, historico de SACYL, cierre colegiado del INAP y
  cotejo/alegaciones de bases locales; se añaden firma por decision, decimal
  fijo y doble control configurable por riesgo.
- Las bases de cada convocatoria gobiernan una version exacta del baremo. RRHH
  utilizara reglas declarativas, simuladas y firmadas; jornada, solapes,
  conversiones y redondeos no tienen valores juridicos ocultos por defecto.
- Los servicios internos se obtendran de oficio desde Personal. Nomina se usa
  como reconciliacion, no como unica fuente, y una indisponibilidad nunca se
  convierte en cero puntos.

## Trabajo pendiente antes de redactar las especificaciones definitivas

- Talleres con RRHH sobre tramitación, baremación, llamamientos, contratos y excepciones.
- Inventario de sistemas corporativos y responsables de cada dato.
- Validación con Sistemas, Seguridad, Secretaría/Asesoría, Archivo y DPD.
- Categorización ENS, análisis de riesgos y evaluación de impacto.
- Matriz completa de roles, ámbitos, incompatibilidades y delegaciones.
- Catálogo de procedimientos, documentos, firmas, registros, notificaciones y plazos.
- Volumetría, disponibilidad, RTO/RPO y presupuesto operativo.
- Inventario de hipervisor, redes, balanceo/WAF, almacenamiento, identidad, PKI, registro,
  secretos, SIEM, copias y segundo CPD para completar el libro ejecutable de instalación.
- Decisión RKE2/OpenShift/plataforma corporativa y conector de autoaprovisionamiento de
  nodos, si la infraestructura permite crearlos por API.
- EIPD, negociación y diseño definitivo de Cronos/GPS, incluida la elección entre APN
  privada en tiempo real o carga diferida sin tránsito exterior.
- Rebaselado de los documentos existentes de `docs/portal_vec`, varios de los cuales aún
  describen un único portal que cambia por rol.
