# Estado comprobable de la petición de RRHH

**Corte:** `origin/main@7f1ecea2fd9f8912d255a80e74da84c69e46b978` y PR abiertas consultadas el 29/09/2026, 23:57 CEST. Esta tabla es un inventario de evidencia; no sustituye el [checklist de RRHH](../../PIDEN_RRHH_CHECKLIST.md) ni acredita una prueba nueva en navegador, una instalación SQL o un despliegue.

**Fuentes:** las cuatro fotografías de `/home/alberto/Trabajo/piden_VEC/`, su [transcripción](../estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md), el seguimiento local `/home/alberto/Trabajo/VEC_Diputacion_app/fotos/2026-09-25 Seguimiento app de gestión Departamento.docx` (fuera de Git) y las filas 1.01–5.08 del checklist. Los puntos **5.01–5.08 son el bloque 5 de ese checklist**, que transcribe el seguimiento y sus cinco imágenes; no he encontrado ocho ficheros Word independientes con esos números. La numeración contiene **68 filas** (60 de las fotos y ocho del bloque 5), aunque la cabecera del checklist dice 67: su suma de «siete» puntos para el bloque 5 omite una de las ocho filas. Para los puntos 1.01–4.15 la fuente de estado detallado es el checklist, comprobado aquí frente al árbol y a las PR actuales.

En las tablas, **M** significa código presente en el corte exacto de `origin/main` indicado arriba; «M parcial» precisa una capacidad incompleta. **PR #n@hash** significa candidata abierta, sin integrar; **rama@hash** es trabajo en curso, sin PR acreditada. «Hecho en M» solo tiene el alcance descrito en la última columna. Un borrador no es un acto firmado, un aviso no acredita entrega y un resultado en un clon no acredita instalación en la principal. El [estado general](../../ESTADO_PROYECTO.md) y la [especificación](../../ESPECIFICACIONES_AGENTES.md) mantienen esos límites.

## Fotografías: histórico, estados y consulta personal

| Punto | Situación verificable | Límite o siguiente comprobación |
| --- | --- | --- |
| 1.01 Contratos anteriores | M: consulta propia paginada y Bolsa 000044. | Solo historia conservada en VEC; fuente de contratos anteriores pendiente (duda 39). |
| 1.02 Llamamientos anteriores | M: histórico propio «Mi Bolsa». | Los contactos anteriores a VEC requieren fuente. |
| 1.03 Renuncias | M: proyección propia y ficha RRHH. | Respuesta a oferta y resolución administrativa son decisiones distintas. |
| 1.04 Sanciones | M: acto, recurso, motivo e historia. | Catálogo de ejemplo pendiente de RRHH (duda 62). |
| 1.05 Cambios de estado | M: valores anterior y nuevo en ficha autorizada. | Datos protegidos se muestran minimizados. |
| 1.06 Correos e intentos | M: intento, resultado y recibo por persona. | «Enviado» no acredita entrega corporativa. |
| 1.07 Documentos generados | M: descargas versionadas de CT y Documentos. | Los borradores no son documentos firmados. |
| 1.08 Disponible | M: situación de Bolsa y vistas. | Estado de la fuente gobernada, no inferido por el menú. |
| 1.09 Trabajando | M: situación de Bolsa y vistas. | Misma condición. |
| 1.10 No disponible | M: situación de Bolsa y vistas. | Misma condición. |
| 1.11 Pendiente de incorporación | M: situación de Bolsa y vistas. | No acredita incorporación en Personal. |
| 1.12 Renuncia | M: situación de Bolsa y operación con recibo. | No equivale a resolución CT. |
| 1.13 Excluido | M: operación y justificante de Bolsa. | Consecuencia sujeta a acto y regla aplicable. |
| 1.14 Disponible desde fecha | M: fecha y orden vigente de Bolsa. | El mero vencimiento no crea una actuación nueva. |
| 1.15 Indisponibilidad +5 meses | M parcial: regla versionada y ensayo PostgreSQL 18. | Falta recorrido navegador → cese → Bolsa y reinicio en servidor aislado; regla definitiva pendiente de RRHH (duda 64). |
| 1.16 Acumulación +9 meses | M parcial: regla por modalidad y mismo ensayo. | Falta el mismo recorrido y reinicio; no aplicar a modalidad desconocida. |
| 1.17 Acceso personal seguro | M: «Mi Bolsa» deriva titular del certificado y del servidor. PR #143@553357c26b y #147@f19e841700 preparan separación del proceso externo; #168@43a43670b2 corrige su arranque; #178@7f59492ea9 reúne «Mi Bolsa» externa en borrador. | Identidad institucional y composición externa definitiva pendientes; DNI + clave de la foto no se adopta. |
| 1.18 Bolsas, posición y estado propios | M: consulta con ámbito V3 de la persona. PR #170@af8b7f9cf9 prepara la lista pública desde B10. | No aceptar identidad libre del navegador; confirmar el recorrido en proceso externo separado. |
| 1.19 Último llamamiento, contratos y disponibilidad | M: «Mi Bolsa» e histórico paginado. | Historia previa a VEC depende de la fuente corporativa. |
| 1.20 Lista pública de integrantes | M: proyección pública minimizada; PR #170@af8b7f9cf9 en integración externa (Codex-B). | Validar qué campos individuales son publicables; no exponer DNI ni identidad completa por la fotografía. |

## Fotografías: bolsas, cobertura y estructura

| Punto | Situación verificable | Límite o siguiente comprobación |
| --- | --- | --- |
| 2.01 Bolsas y candidaturas | M: lista y ficha interna de Bolsa. | Gestión básica en VEC; no implica reglas aprobadas. |
| 2.02 Llamamientos según orden | M: propuesta ordenada y confirmación humana. | No se acredita adjudicación automática sin control; parámetros de ejemplo pendientes (dudas 13–14). |
| 2.03 Contratos | M: expediente CT coordina propuesta y formalización. | No acredita alta laboral corporativa ni firma. |
| 2.04 Cese | M: operación durable de desarrollo y recibo; recorrido de copia de ensayo documentado el 26/09. | No acredita baja en GINPIX ni acto eficaz. |
| 2.05 Reincorporación | M parcial: CT130/134 y Bolsa 000046, con código de retorno. Rama `trabajo/ct-fases-2-8-sembrado-20260929`@c4dd9ae15 (Codex-A) diagnostica generalización. | Falta recorrido de la ruta completa con aplicación y recuperación; hoy la incorporación de ejemplo depende de un expediente concreto. |
| 2.06 Reglas configurables | M: catálogos versionados; PR #162@f3eb063132 (lectura completa), #167@19945be66f (texto claro), #169@4a8cb1c0a9 (carga aislada), #174@4e431161af (almacén de plazos). Codex-D continúa resolutor, rutas y pantalla. | Que exista catálogo no acredita que RRHH pueda cambiar todas las reglas y fechas desde la aplicación; valores de ejemplo pendientes de aprobación. |
| 2.07 Portal del candidato | M: «Mi Bolsa» con certificado de desarrollo y AD3-112 integrado por #151, merge `ccc4474e4`. PR #143/#147 y dependientes #156/#157/#159/#178@7f59492ea9 (Codex-B) separan autoridades externas. | Proveedor de identidad institucional y recorrido exterior pendientes. |
| 2.08 Cuadro de responsables | M: estadísticas operativas de Bolsa. | No equivale a informe oficial de dirección. |
| 2.09 Estadísticas por bolsa y estado | M: agregados en vista Bolsa. | No se acredita explotación externa ni exportación de datos personales. |
| 2.10 Word y PDF | M: generación y descarga de borradores. | Plantillas y firma final se verifican aparte. |
| 2.11 Correo | M: adaptador y recibo de intento para llamamiento. | Falta acreditar SMTP y entrega corporativos (dudas 10 y 45). |
| 2.12 SMS y mensajería | Sin empezar como envío integrado; M solo registra intento manual de otros canales. | Dependen de canal corporativo y viabilidad normativa (dudas 3 y 35). |
| 2.13 Auditoría completa | M parcial: consulta segregada y vista legible. | Falta lectura positiva con certificado y recuperación tras reinicio en principal; decidir campos y motivos (duda 67). |
| 2.14 Datos de la bolsa | M: referencia, categoría, vigencia, resolución y orden versionado. | Las categorías editables de RPT están en diseño de catálogo común, dueño Codex-D; no hay maestro durable aprobado. |
| 2.15 Datos de candidatura y contacto | M: Persona común, participación y contacto protegido. PR #160@281e39dae1 añade «Mi ficha» (Codex-B). | No usar DNI como clave técnica ni duplicar persona por bolsa. |
| 2.16 Situación y observaciones | M: posición, estado, fecha y actuaciones autorizadas. | No hay campo libre público universal. |

## Fotografías: control, comunicaciones y avisos

| Punto | Situación verificable | Límite o siguiente comprobación |
| --- | --- | --- |
| 3.01 Bolsas activas | M: estadísticas de Bolsa. | Indicador operativo. |
| 3.02 Candidatos por bolsa | M: agregado `por_bolsa`. | Indicador operativo. |
| 3.03 Disponibles, trabajando, excluidos, no disponibles | M: agregado `por_estado`. | Depende de estados y fechas vigentes. |
| 3.04 Destinatarios por estado y orden | M: selección de elegibles para una oferta. | Campaña informativa general por estados necesita finalidad y permiso propios (duda 35). |
| 3.05 Correo personalizado masivo | M: plantilla, vista previa, personalización y adaptador SMTP. | Relay de desarrollo no acredita envío corporativo ni entrega. |
| 3.06 Oferta web y plazas por orden | M: PR #125, merge `c4adaf10b`, incorpora plazas, respuesta y propuesta por orden con prueba en clon. PR #173@fbf0ac8a65 (Codex-A) trata el plazo inicial. | No instalada en la principal según checklist; las reglas de ejemplo y la doble validación esperan RRHH (dudas 66 y 75). |
| 3.07 Llamamiento directo si no se cubre | M: PR #125, merge `c4adaf10b`, transición por plaza y acto propio. | Falta política aprobada sobre horario y segundo ciclo telefónico; no equivale a llamada ni aviso externo. |
| 3.08 Aviso por salto de orden | M: aviso interno de Bolsa. | No produce resolución automática. |
| 3.09 Aviso de tres años | M: cálculo sobre historia disponible. | Años anteriores a VEC exigen fuente de Personal (duda 39). |

## Fotografías: documentos y auditoría

| Punto | Situación verificable | Límite o siguiente comprobación |
| --- | --- | --- |
| 4.01 Contrato laboral | M: plantilla y descarga DOCX/PDF. | Borrador, sin firma ni eficacia. |
| 4.02 Nombramiento | M: plantilla y descarga DOCX/PDF. | Borrador, sin nombramiento eficaz. |
| 4.03 Toma de posesión | M: borrador condicionado al expediente. | No acredita posesión realizada. |
| 4.04 Cese documental | M: borrador tras registrar cese. | No acredita baja corporativa. |
| 4.05 Modificación de nombramiento | M: borrador condicionado a la operación. | Eficacia administrativa pendiente. |
| 4.06 Informes | M: informe y catálogo de borradores. | Modelos definitivos sujetos a aprobación. |
| 4.07 Resoluciones | M: borrador de resolución. PR #155@e0f3afbc11 → #164@7d5e75bda5 → #165@20141e4c21 → #176@c314f63326 (Codex-D) preparan custodia del firmado. | La cadena no está integrada; no llamar firmada a la resolución de M. |
| 4.08 Otras plantillas | M parcial: catálogo y funciones CT131/133/135/137. | Falta acreditación de alta, publicación y descarga por identidades V3 reales tras reinicio; duda 68. |
| 4.09 Coste por categoría | M: cálculo y campo de borrador con fuente o «sin calcular». | No inventar importe de nómina ni atribuir fuente corporativa. |
| 4.10 Datos GINPIX o alternativa | M: campos versionados del expediente como alternativa para borrador. | Conexión automática GINPIX sin empezar por falta de contrato corporativo (duda 39). |
| 4.11 Autor e instante de cambio | M parcial: consulta con segundos y actor técnico. | Nombre de persona, fuente nominal, lectura positiva y reinicio pendientes (dudas 67 y 71). |
| 4.12 Antes, después y motivo | M parcial: vista minimiza datos protegidos. | Motivos CT antiguos no constan y RRHH debe decidir exposición (duda 67). |
| 4.13 Documento o expediente relacionado | M parcial: relación y recibo cuando existen. | Falta lectura positiva autorizada y recuperación tras reinicio. |
| 4.14 IP o equipo | M: decisión explícita de omitirlos por minimización. | La fotografía los pide solo si la política lo permite (duda 36). |
| 4.15 Historia sin alteración invisible | M parcial: historia y lectura segregada. PR #177@88512e9956 comprueba y documenta el 409 sin efectos de subsanaciones anteriores al perfil fijo, producido por CT92/#166. | Falta contrastar cada acción del recorrido CT/Bolsa con auditoría tras reinicio (E06; duda 67). El recibo histórico interno de subsanación no se vuelve a mostrar. |

## Seguimiento del Departamento: puntos 5.01–5.08

| Punto | Situación verificable | Límite o siguiente comprobación |
| --- | --- | --- |
| 5.01 Uso intuitivo | M parcial: shell, pasos y ayuda «?». Codex-M tiene PR #180@a647c932c7 (índice y capturador) y borradores #181@4fc01ef0a7 / #182@eaf7877bfb (guiones de recorridos). | Los guiones siguen sin ejecutar contra VEC; faltan recorrido completo CT/Bolsa, teclado, zoom y valoración de RRHH. No hay manual terminado. |
| 5.02 Nombre «Peticiones de personal temporal» | Hecho en M: PR #106, merge `175d5255a`, y #112, merge `805590de3`; el checklist acredita Chrome en la principal. | Repaso de i18n del shell pendiente: «Avisos» aún en castellano en inglés. |
| 5.03 Primera pantalla con CT, Bolsa y SAE | M parcial: PR #106 `175d5255a` y #158 `d1d587a15` dan cuadro y lista CT. PR #172@e8948d1ab7 (Codex-A) añade plazos vencidos visibles. | La tarjeta SAE sigue sin datos; el 503 de la lista se corrigió en código, sin nuevo recorrido atribuido aquí. |
| 5.04 Dos vías y listas de documentos/datos | Hecho como lista **de ejemplo** en M: PR #114, merge `3af40dde1`. | SAE solo informa; RRHH debe confirmar documentos y campos. |
| 5.05 Ofertas al SAE | Sin empezar como operación durable; M tiene maqueta informativa. | Duda 72: datos, canal y selección. No afirmar envío al SAE. |
| 5.06 Estado de firma, AutoFirma y Firmadoc | M parcial: borradores, indicador y lectura de Documentos (#152 `be0e5a391`, #153 `949365f2d`). PR #155@e0f3afbc11 → #164@7d5e75bda5 → #165@20141e4c21 → #176@c314f63326 (Codex-D). Dirección comunica GO sensible y de usabilidad del código/UI de #176 a las 23:40. | Siguen pendientes el recorrido de la cadena con V3/COSE reales, recuperación tras reinicio y tratamiento comprobado de objetos huérfanos. SQL y binario deben instalarse juntos; nada de esta cadena está integrado ni instalado en la principal. Firmadoc no tiene API admitida ni envío activo. No hay firma legal acreditada. |
| 5.07 Error de «Nueva petición» | Hecho en M: PR #117, merge `f68e5d035`; #158, merge `d1d587a15`, corrige además la lista RRHH. | El checklist aún describe #117 como pendiente, pero Git confirma su merge. Repetir Chrome sobre el corte actual; revisar el 404 de borradores al abrir Expediente. |
| 5.08 Preferencias, correos e imagen | M parcial: **5.08a** en PR #113, merge `fab12e430`, con lectura/guardado en principal según checklist. **5.08b/c** en ramas `origin/trabajo/usuarios-508b-correos-20260929`@f89b4e9d9 y `origin/trabajo/usuarios-508c-imagen-20260929`@6a92e7546; Codex-B prepara separación externa en el borrador #179@c063352297. | El borrador #179 no cierra b/c: falta recorrido entre procesos y reclaveado gobernado del correo histórico. Elegir avisos no prueba que se envíen; correo activo verificado y foto custodiada no se declaran terminados. Duda DPD 73. |

Las PR de soporte abiertas en este corte conservan estas dependencias:

| PR | Aporte | Pendiente |
| --- | --- | --- |
| #177@88512e9956 | Transición 409 sin efectos para subsanaciones anteriores a #166. | Revisión final y CI; no reexpone recibo interno antiguo. |
| #178@7f59492ea9, borrador | Proceso externo de «Mi Bolsa» y portal del candidato. | Permisos externos completos, provisión y recorrido. |
| #179@c063352297, borrador | Material y datos separados para correos e imagen externos. | Recorrido entre procesos y reclaveado gobernado. |
| #180@a647c932c7 | Índice y capturador para manuales. | No contiene manuales terminados ni capturas de VEC. |
| #181@4fc01ef0a7 y #182@eaf7877bfb, borradores | Guiones Playwright para portales, Bolsa, Área personal, CT e Intervención. | No ejecutados contra VEC: falta clon H3–H5 y material mTLS atribuibles. |

## Huecos que determinan el siguiente reparto

1. **Recorrido y evidencia:** Codex-M tiene los guiones #181/#182, aún sin ejecución. Debe devolver a Dirección el primer corte exacto de cada proceso cuando exista el clon. Esta tabla no afirma un E2E nuevo. Faltan, entre otros, cese → disponibilidad, reincorporación, auditoría positiva tras reinicio y oferta por plazas en la principal.
2. **Firma y documentos:** #176 obtuvo GO sensible y de usabilidad para el código/UI; Codex-D debe acreditar V3/COSE reales, reinicio y objetos huérfanos antes de integrar la cadena #155/#164/#165/#176. Firmadoc requiere contrato de Informática y decisión RRHH sobre documentos y orden ([petición técnica](../estudio_requisitos/peticion_informatica_integraciones_2026-09-23.md)).
3. **Reglas y categorías:** plazos editables y RPT gobernada siguen en trabajo de Codex-D; Codex-A consume la categoría de esa autoridad para aceptación, renuncia e incorporación. Una regla de ejemplo no es una regla legal aprobada.
4. **Portal y persona candidata:** Codex-B lleva la separación externa, «Mi Bolsa» y las piezas 5.08b/c. Las PR abiertas no acreditan la composición completa ni el proveedor de identidad real.
5. **Decisiones externas:** SAE (duda 72), SMS/mensajería, SMTP corporativo, GINPIX, política pública de datos y reglas de tiempo siguen sin fuente o aprobación suficiente. Mantener sus pantallas o conectores apagados hasta recibirla.

**Dudas concretas para RRHH y Sistemas:** ¿qué documentos y datos exige cada vía Bolsa/SAE (5.04), qué operación y canal componen una oferta SAE (5.05), qué documentos y orden pasan por Firmadoc (5.06), qué reglas definitivas gobiernan +5/+9 meses y los intentos de contacto (1.15–1.16, 3.07), qué motivos y datos pueden verse en auditoría (4.11–4.13) y qué datos individuales pueden publicarse en la lista de Bolsa (1.20)? Las respuestas deben registrarse en `dudas.md` y en el catálogo/versiones de su autoridad; no se infieren de los ejemplos.
