# Cronos: continuación por capacidades

Plan de trabajo del 7 de octubre de 2026. La orden de Dirección reanuda los módulos;
cidonia queda congelada. Esta ficha ordena cortes futuros sobre `main@01046e2e`;
no autoriza instalar SQL, habilitar Cronos ni usar datos reales.

## Punto de partida comprobado

La [ficha C1–C12](../estudio_requisitos/ficha_cronos_2026-09-23.md) mantiene
el último cómputo acreditado: **0/12 formal**, 3/12 técnico (C6, C7 y C9) y
0/12 de uso real con barrido documentado. Las fechas y cantidades servidas
de septiembre son referencias históricas, no una medición nueva. El
[inventario post-H9](../estudio_requisitos/cronos_dietas_instalacion_2026-10-04.md)
constata cadenas SQL sin instalar y Cronos oculto por la orden entonces vigente.

| Pieza | Reutilización concreta | Límite actual |
| --- | --- | --- |
| Fichaje y corrección C2/C5 | `internal/modules/cronos/{domain,application,ports,adapters}`, marcaje remoto, solicitud de olvido, historia y recibos; SQL `cronos_v1` 000001–000008. | Terminales sin integración acreditada; CRN11 y su consumidor AD149 histórico precisan compatibilidad con el núcleo común vigente, composición y prueba nominal. La aprobación y aplicación de la corrección no están cerradas. |
| Jornada y saldo C3/C4 | `consulta_saldo`, libro de tiempo, ensayos de calendario histórico y vistas de saldo/calendario. | Jornada teórica y cuantías provisionales; falta adoptar fuente/versiones de Calendarios, adscripción histórica y efecto aprobado de permisos. |
| Permisos y comunicaciones C6–C9 | Catálogo de ejemplo, solicitudes, resolución jefatura→RRHH, avisos, notificaciones y pantallas registradas en `web/static/portal-empleado/modulos/cronos/`. | La cadena AD57→Cronos9→AD58→Cronos10 no está acreditada como instalada en la principal. Justificación C8 carece de enlace documental operativo. |
| Equipo e informes C10–C12 | Bandejas y ensayos de presencia, incidencias, agregados y CSV/PDF sintéticos. | Falta alcance nominal del equipo, política de exportación por campos, auditoría común y recorrido conectado. |

`internal/app/bootstrap/cronos_empleado.go` y `cronos_manejadores.go` son el
punto de composición que debe cotejarse en cada corte. La
[guía de integración](../../web/static/portal-empleado/modulos/cronos/INTEGRACION.md)
explica sus selectores y conserva explícito el recorrido todavía pendiente.
Las pruebas de dominio, CLI y contrato web no equivalen a instalación ni a
recuperación tras reiniciar aplicación y PostgreSQL.

La PR **#578 / AD181** se trata como borrador de lectura histórica para C5:
faltan huellas y comprobación de su estado final. No cierra C5 ni C7, y no se
incluye como dependencia instalada. El equipo V es dueño de AD181, del núcleo
de autorización y de la auditoría común; T entrega la fuente Personal26 y
la proyección de adscripción. U conserva los consumidores y el montaje de
Cronos. El módulo no editará tablas ni archivos de Personal.

## Fuentes y reglas que gobiernan el diseño

El [TREBEP, arts. 47–51](https://www.boe.es/eli/es/rdlg/2015/10/30/5/con)
distingue jornada, permisos y vacaciones del personal funcionario, y remite
para el laboral también a su legislación. Para este colectivo, la Diputación
publica el [Convenio colectivo de 2006](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/CONVENIO-2006.pdf),
sujeto a modificaciones posteriores. También publica el
[Reglamento de tiempo de trabajo de 2010](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/REGLAMENTO-TIEMPO-DE-TRABAJO.pdf),
la [Resolución de jornada y permisos de 2015](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/RESOLUCION-2015-SOBRE-JORNADA-HORARIO-VACACIONES-Y-PERMISOS.pdf)
y sus modificaciones en la [página oficial de RRHH](https://www.dipgra.es/contenidos/normativa-recursos-humanos/).
El [acuerdo del Pleno de 2/09/2026](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/Certificado_pleno_2_09_2026_punto_nA_12_Rev_art_17.4.pdf)
declara nulo el artículo 17.4 del Reglamento: el
[estudio de turnos](../estudio_requisitos/turnos_festivos_y_compensaciones.md)
todavía lo señala como pendiente de resolución y debe contrastarse con ese
acto antes de publicar reglas de vacaciones. No se deducen aquí días,
compensaciones ni competencias individuales de esas fuentes.

`Calendarios` gobierna festivos, centro, fuentes y versiones;
`Personal` acredita relación, colectivo, centro y adscripción histórica;
`Cronos` conserva programación, marcajes, correcciones, permisos y saldos.
El [modelo histórico común](../estudio_requisitos/calendario_habil_laboral_historico.md)
distingue día hábil administrativo, apertura del centro y minutos programados.
Una ausencia de fuente se muestra como **no determinable**, nunca como cero.

Los catálogos de jornada, permisos, cuantías, unidad de cómputo, redondeos,
justificantes, responsables y vigencia guardan versión, fuente, aprobación y
fecha de efecto. Los 25 permisos y cantidades del ejemplo continúan
provisionales. Las cuestiones existentes se siguen en
[dudas.md](../../dudas.md) 38–41, 47–48, 99–103, 118 y 122; no se abre otra
pregunta por los mismos extremos. Para pruebas se usan catálogos y personas
sintéticos. El perfil fijo de resolución se asigna por Administración con
historia y ámbito; no crea por sí solo la competencia administrativa. Se
prohíbe resolver la solicitud propia y repetir la misma persona los pasos de
jefatura y RRHH.

## Minitareas en orden de dependencia

Cada fila es un corte con una responsabilidad observable y criterio de cierre.
Los archivos de autorización común, Calendarios, Personal, Documentos y auditoría
pertenecen a sus equipos; la composición compartida se reserva antes de editar.

| Orden | Corte y propietario | Resultado comprobable; dependencia |
| --- | --- | --- |
| 1 | **C3p, U:** optimizar la consulta de saldo propio existente y preparar su contrato de datos. | Consulta acotada por periodo, paginada donde corresponda, con número fijo de consultas y decisiones V3; `EXPLAIN ANALYZE` y tiempo de ruta en clon local. No cambia reglas ni derechos. |
| 2 | **C5a, V/T → U, bloqueado:** V mide la preimagen exacta del núcleo común en el clon de `main` vigente, incluidos AD207/AD208, y entrega AD181 y auditoría nominal; T entrega Personal26. U adapta consumidor CRN11 y montaje a esos contratos. | Consulta propia y recuperación de un olvido con recibo único, denegación ajena y error cerrado. Ensayo PostgreSQL 18 en clon y dos revisiones SQL/identidad; ninguna migración histórica se reaplica. |
| 3 | **C5b, U:** decisión de jefatura, validación RRHH y aplicación de corrección como asientos nuevos. | Marcaje original intacto; motivo, versión, actores y recibos consultables tras reinicio. No concede por ausencia de responsable ni por perfil propio. |
| 4 | **C4/C3a, Calendarios → U:** consumir versión oficial y cuadrante publicado junto a adscripción histórica de T. | Una fecha y un turno que cruza medianoche se explican con fuente y versión; falta de fuente da `no determinable`. El ensayo `ensayo_calendario_historico` sirve de vector, no de autoridad. |
| 5 | **C6/C7, V → U:** V provisiona perfiles fijos nominales, ámbito de jefatura y RRHH, y reconcilia los consumidores históricos AD57→Cronos9→AD58→Cronos10 con el núcleo común vigente. U monta los casos de uso. | Solicitud, doble resolución y aviso con recibos recuperables; revocación, conflicto de funciones y falta de jefatura deniegan o dejan pendiente. Catálogo aprobado antes de aplicar derechos. |
| 6 | **C8, Documentos/V → U:** enlazar original custodiado, huella y versión de justificante; revisar y registrar resultado. | Documento ajeno o huella distinta rechazada; aportar referencia no equivale a justificar. CRN14 espera Documentos12/AD148 y AD146 compatibles con `main`. |
| 7 | **C3b, V/Calendarios/T → U:** adoptar catálogo de efectos y calcular saldo con jornada, asientos y permisos resueltos. | Día/semana/mes/periodo reproducibles con versión histórica; corrección posterior genera nuevo resumen, sin reescribir el anterior. CAT5/AD147, CRN13 y AD146 preceden a CRN12; medir compatibilidad sobre la cadena común vigente. |
| 8 | **C10/C11, U:** consultas de incidencias, presencia y agregados por equipo vigente. | Jefatura ve solo datos mínimos de su equipo; ausencia de fichaje no se llama absentismo probado. Consultas paginadas y en lote. |
| 9 | **C12 y cierre mensual, Documentos/V → U:** bloquear periodo con versión y recibo, permitir rectificación trazada, emitir informe autorizado. | Mes cerrado reproducible tras reinicio; ninguna exportación por permiso de mera lectura. Campos y tipos esperan la respuesta 118 y autorización nominal específica. |

El **primer corte independiente de U es C3p**: medir la lectura de saldo
propio con datos sintéticos realistas, fijar el número de consultas y corregir
la ruta si supera el presupuesto. C5a sigue bloqueado mientras V no entregue
AD181 y la cadena común medida en el clon exacto, y T no entregue Personal26.
H9 queda como antecedente de diagnóstico; ninguna huella suya sirve para
instalar sobre el núcleo actual. U puede preparar los DTO y pruebas de
contrato de CRN11 mientras espera, sin escribir en Personal ni en autorización.

## Corte C3p preparado: consulta anual de saldo

La construcción del saldo agrupa una vez los marcajes por fecha local. Conserva
el orden de la fuente, el cambio de hora y las listas vacías del contrato. La
prueba focal final y la revisión independiente de seguridad han pasado.

En local, con 10.000 marcajes y 365 días, la mediana de tres ejecuciones del
benchmark de agrupación baja de 219,09 ms a 1,47 ms. El constructor completo
tarda 2,30 ms; estas cifras miden cálculo en memoria, sin base de datos ni
petición HTTP. La PR #831 pasó la puerta completa y quedó integrada en `main`. La medición
de petición y pintado útil sigue abierta; C3p no acredita activar Cronos ni
cerrar la recuperación nominal C5a.

## Puerta de cada corte

Identidad y perfil vienen del servidor; autorización positiva por acción,
persona, organización, unidad y fecha se consume en la transacción con
estado, auditoría nominal común, historia y outbox. Un reintento recupera el
mismo efecto. La lectura y el rechazo dejan auditoría sin revelar motivos de
salud o actividad sindical a la jefatura. Ningún consumidor hace una consulta
ni una decisión V3 por fila: filtros y permisos se resuelven en lote, con
índices y paginación. Medir `EXPLAIN ANALYZE` y ruta con volumen realista:
objetivos de **base <100 ms** y **petición y pintado útil p95 <300 ms
en local**. Son criterios de aceptación pendientes de
medición, no resultados obtenidos por este plan.

Pruebas focales Go, Node y SQL real desechable; revisión independiente de
usabilidad si cambia una pantalla, y dos revisiones independientes de SQL,
identidad, autorización o datos personales. Antes de integrar: puerta de
calidad, `git diff --check`, Semgrep local de lo cambiado y recorrido Chrome
con datos sintéticos, denegaciones, reinicio de aplicación/PostgreSQL y recibo
idéntico. Solo Dirección integra y publica. La salida a producción con datos
reales requiere las autorizaciones formales y el enclave interno descritos en
[seguridad y despliegue](../estudio_requisitos/seguridad_y_despliegue_cronos.md).

## Vigencia del vínculo y auditoría del rechazo

La solicitud de olvido de #851 comprueba que el vínculo del empleado siga
vigente al actuar. Este corte extiende esa comprobación a las consultas y
solicitudes que comparten el mismo helper, y conecta sus rechazos con la
auditoría nominal común. Conserva la identidad histórica acreditada por el
servidor; no toma el actor ni el perfil del cuerpo de la petición.

El rechazo solo responde 403 cuando el registrador confirma el acuse. Si la
auditoría falta o falla, responde 503 sin datos ni efectos en el repositorio de
negocio. El reintento interno conserva la misma orden. Una operación válida
no añade decisiones ni registros por fila.

Cronos activo requiere el registrador común de AD169 y un motivo de rechazo
publicado, vigente y distinto de los motivos positivos, configurado en
`auditoria_intentos.motivo_denegado`. El LOGIN, proceso y canal proceden del
material privado y pasan el preflight común. Este cambio no crea sus permisos,
no instala SQL ni acredita C5a, C5b o un recorrido nominal en PostgreSQL.
