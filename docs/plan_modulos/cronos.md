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
| Fichaje y corrección C2/C5 | `internal/modules/cronos/{domain,application,ports,adapters}`, marcaje remoto, solicitud de olvido, historia y recibos; SQL `cronos_v1` 000001–000008. | Terminales sin integración acreditada; CRN11 y su consumidor AD149 precisan reanclaje a H9, composición y prueba nominal. La aprobación y aplicación de la corrección no están cerradas. |
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
incluye como dependencia instalada. El módulo Personal y su proyección de
adscripción pertenecen al equipo T; Cronos recibirá solo un contrato opaco
autorizado y no editará sus tablas ni sus archivos.

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
Se subdivide si excede el presupuesto de una minitarea de `AGENTS.md`. Los
archivos de autorización común, Calendarios, Personal, Documentos y auditoría
pertenecen a sus equipos; la composición compartida se reserva antes de editar.

| Orden | Corte y propietario | Resultado comprobable; dependencia |
| --- | --- | --- |
| 1 | **C5a, Cronos + autoridad V3:** reanclar CRN11/AD149 a la preimagen H9, sin reaplicar migraciones; L/K conserva el consumidor compartido. | Consulta propia y recuperación de un olvido con recibo único, denegación ajena y error cerrado. Primero cotejar #578/AD181 y acordar el contrato histórico con T; ensayo PostgreSQL 18 en clon y dos revisiones SQL/identidad. |
| 2 | **C5b, Cronos:** decisión de jefatura, validación RRHH y aplicación de corrección como asientos nuevos. | Marcaje original intacto; motivo, versión, actores y recibos consultables tras reinicio. No concede por ausencia de responsable ni por perfil propio. |
| 3 | **C4/C3a, Calendarios → Cronos:** consumir versión oficial y cuadrante publicado junto a adscripción histórica de T. | Una fecha y un turno que cruza medianoche se explican con fuente y versión; falta de fuente da `no determinable`. El ensayo `ensayo_calendario_historico` sirve de vector, no de autoridad. |
| 4 | **C6/C7, Cronos + K/L:** provisionar perfiles fijos nominales, ámbito de jefatura y RRHH, y reconciliar AD57→Cronos9→AD58→Cronos10 con H9. | Solicitud, doble resolución y aviso con recibos recuperables; revocación, conflicto de funciones y falta de jefatura deniegan o dejan pendiente. Catálogo aprobado antes de aplicar derechos. |
| 5 | **C8, Documentos → Cronos:** enlazar original custodiado, huella y versión de justificante; revisar y registrar resultado. | Documento ajeno o huella distinta rechazada; aportar referencia no equivale a justificar. CRN14 espera Documentos12/AD148 y AD146 compatibles. |
| 6 | **C3b, Cronos:** adoptar catálogo de efectos y calcular saldo con jornada, asientos y permisos resueltos. | Día/semana/mes/periodo reproducibles con versión histórica; corrección posterior genera nuevo resumen, sin reescribir el anterior. CAT5/AD147, CRN13 y AD146 preceden a CRN12; reanclar a H9. |
| 7 | **C10/C11, Cronos:** consultas de incidencias, presencia y agregados por equipo vigente. | Jefatura ve solo datos mínimos de su equipo; ausencia de fichaje no se llama absentismo probado. Consultas paginadas y en lote. |
| 8 | **C12 y cierre mensual, Cronos + Documentos/L:** bloquear periodo con versión y recibo, permitir rectificación trazada, emitir informe autorizado. | Mes cerrado reproducible tras reinicio; ninguna exportación por permiso de mera lectura. Campos y tipos esperan la respuesta 118 y autorización nominal específica. |

El **primer corte ejecutable es C5a**. Su entrega es una cadena CRN11 compatible
con H9, consumo V3 exacto, lectura propia y replay del mismo recibo en el clon.
Se detiene antes de aplicación de correcciones y antes de activar Cronos. Si
AD181 sigue en borrador, T entrega primero el contrato histórico mínimo; el
equipo Cronos puede preparar el cotejo de preimagen sin escribir en Personal.

## Puerta de cada corte

Identidad y perfil vienen del servidor; autorización positiva por acción,
persona, organización, unidad y fecha se consume en la transacción con
estado, auditoría nominal común, historia y outbox. Un reintento recupera el
mismo efecto. La lectura y el rechazo dejan auditoría sin revelar motivos de
salud o actividad sindical a la jefatura. Ningún consumidor hace una consulta
ni una decisión V3 por fila: filtros y permisos se resuelven en lote, con
índices y paginación. Medir `EXPLAIN ANALYZE` y ruta con volumen realista:
**base <100 ms**, **lectura servidor p95 <300 ms** y pantalla útil <1 s.

Pruebas focales Go, Node y SQL real desechable; revisión independiente de
usabilidad si cambia una pantalla, y dos revisiones independientes de SQL,
identidad, autorización o datos personales. Antes de integrar: puerta de
calidad, `git diff --check`, Semgrep local de lo cambiado y recorrido Chrome
con datos sintéticos, denegaciones, reinicio de aplicación/PostgreSQL y recibo
idéntico. Solo Dirección integra y publica. La salida a producción con datos
reales requiere las autorizaciones formales y el enclave interno descritos en
[seguridad y despliegue](../estudio_requisitos/seguridad_y_despliegue_cronos.md).
