# Dietas: continuación del módulo

Plan de 7 de octubre de 2026. Dietas se reanuda para completar la solicitud, la
justificación, la decisión y la liquidación de comisiones de servicio. La principal
de cidonia queda congelada en este encargo: este plan no acredita instalación,
activación ni uso allí. Solo se trabajará con datos sintéticos hasta las
autorizaciones formales.

## Fuentes y decisiones que mandan

- [Real Decreto 462/2002, texto consolidado](https://www.boe.es/buscar/act.php?id=BOE-A-2002-10337): distingue dietas, gastos de viaje, grupos, tramos y justificación. El texto consolidado es informativo; el acto aplicable debe fijarse antes de publicar una regla.
- [Orden HFP/793/2023](https://www.boe.es/buscar/doc.php?id=BOE-A-2023-16462): revisó el kilometraje de automóvil y motocicleta. No convertir esos importes en constantes del programa.
- [Bases de ejecución de la Diputación de Granada de 2026, actualizadas el 5 de octubre](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/economia-y-atencion-al-alcalde/.galleries/DIPUTACION-Delegaciones-Economia-y-Patrimonio-Economia/DIPUTACION-Delegaciones-Economia-y-Patrimonio-Economia-Presupuestos/DIPUTACION-Delegaciones-Economia-y-Patrimonio-Economia-Presupuesto-Diputacion/Presupuesto-Diputacion-Ejercicio-2026/Ejercicio-2026-Presupuesto-General/BASES-DE-EJECUCION-DEL-PRESUPUESTO-2026-TRAS-ENMIENDAS-Y-RECTIFICACION-Y-ANEXOS-acualizadas-05-10-2026.pdf), art. 30: exigen autorización previa del Diputado salvo excepciones, cuenta justificativa y documentación tras el servicio. Clasifican grupos y prevén casos de alojamiento excepcionales.
- [Convenio colectivo publicado por la Diputación, art. 47](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/CONVENIO-2006.pdf): referencia para el personal laboral. RRHH debe confirmar vigencia, ámbito y relación con las bases de 2026 antes de activar cuantías.
- [Ficha D1–D9](../estudio_requisitos/ficha_dietas_2026-09-23.md), [especificaciones](../../ESPECIFICACIONES_AGENTES.md), [contrato modular](../portal_vec/contrato_modulos_vec.md) y [estudio integral](../estudio_requisitos/analisis_integral_rrhh.md) fijan el alcance técnico. Las dudas abiertas están en [dudas.md](../../dudas.md): 24–26, 40, 44, 46, 49–51, 60, 104–108, 118 y 122. Se responderán allí, sin abrir otra lista.

Las bases clasifican al personal en tres grupos, mientras el art. 47 del
convenio laboral remite al grupo 2. RRHH debe fijar la regla aplicable por
colectivo y fecha. Además, las bases remiten para alojamiento al art. 12.4
del RD 462/2002; su art. 10.3 trata el gasto efectivamente realizado y
justificado. Se conserva la
[duda 104](../../dudas.md) y se bloquea la liquidación automática del alojamiento
hasta que RRHH determine cómo aplicar ambas fuentes. Las excepciones y el plazo
de la cuenta también requieren decisión expresa; el programa no deducirá
pérdida de derechos ni otra consecuencia del retraso.

## Inventario real y límite de lo existente

| Pieza | Situación al planificar |
| --- | --- |
| `internal/modules/dietas/{domain,ports,application,adapters}` y `internal/app/bootstrap/dietas_*.go` | Hay comisiones, borradores, cálculo provisional, rutas, circuito, autorización V3 y composición. La fuente de competencia usa perfiles fijos centrales; la escritura de asignación D7 sigue cerrada sin catálogo competente. |
| `deploy/postgresql/dietas_borradores/migraciones/000001`–`000011` | Hay definiciones y ensayos. La ficha de septiembre solo acredita 000001–000008 instaladas entonces en cidonia. Revalidar historia antes de cualquier instalación posterior; no reaplicar ni ejecutar DOWN sobre una base con historia. |
| `deploy/postgresql/dietas_borradores/borradores/000012_liquidacion_economica.md` | Es diseño sin SQL ejecutable. Enumera postimagen y dependencias; no registra una liquidación. |
| `web/static/portal-empleado/modulos/dietas/` | Existen borrador propio, circuito, mapa OSRM y vistas de catálogo e informes. `INTEGRACION.md` separa lo compuesto de clientes de rectificación aún no inyectados. Las vistas de catálogo e informes usan paquetes de ejemplo; sus CSV/PDF no son exportaciones nominales autorizadas. |
| `application/preparacionliquidacion`, `simulaciondevengo` e `informeperiodo` | Preparan instantáneas sintéticas sin efectos. El informe trabaja con registros importados, no con lectura nominal de PostgreSQL. Una propuesta no equivale a acto de liquidación. |
| `application/custodia_justificantes_comision.go` | Registra referencias y huellas de custodia externa; no recibe los bytes del justificante ni acredita que el fichero se haya conservado. |

El último recuento documentado, del 25 de septiembre, es **0/9 formal,
3/9 técnico y 0/9 de uso real**. Ese recuento es histórico: las piezas posteriores
se inventarían arriba sin asignarles cierre o uso nuevo. Dirección actualizará
el estado transversal cuando haya evidencia revisada.

## Recorrido que se debe terminar

1. La persona acreditada abre una comisión vinculada a su relación y unidad a
   fecha. Introduce motivo, destino, horas, itinerario y gastos; conserva borrador
   y consulta el recibo de cada versión. Personal aporta relación y adscripción
   mediante su proyección autorizada; Dietas conserva el expediente propio.
2. La preparación calcula manutención por tramos, alojamiento cuando se admita
   su regla y kilometraje por ruta OSRM interna. Muestra fecha de vigencia,
   procedencia, grupo, país y versión de cada tarifa. Otros transportes y gastos
   usan tipos y topes publicados. Un dato o regla ausente deja la propuesta
   pendiente, con motivo visible, sin sustituirlos por una cifra anterior.
3. La persona adjunta los justificantes exigidos a cada gasto mediante el
   adaptador de Documentos: límites de tamaño, tipos y número, comprobación
   antivirus y custodia con huella y referencia opaca. Si falla la inspección o
   la custodia, el envío queda pendiente. La referencia externa actual sigue
   siendo solo una declaración hasta contrastar el fichero.
4. El envío pasa por revisión administrativa, autorización, liquidación RRHH
   y fiscalización según el circuito aprobado. Cada perfil fijo necesita
   asignación nominal central, ámbito y vigencia. La persona solicitante no
   decide su propio expediente. Devoluciones, correcciones, reenvíos y
   rectificaciones añaden versión y motivo; conservan decisiones anteriores.
5. La liquidación toma la versión exacta de comisión, justificantes y catálogo
   publicado; registra líneas admitidas y rechazadas, total, acto, versión,
   recibo e historia. El PDF se genera desde esa instantánea. Fiscalización
   y aprobación se muestran con sus actos propios, sin etiquetarlas como pago.
6. Una exportación para el sistema económico exige política de campos, permiso
   distinto y acuse del receptor. Su reintento usa la misma clave y se concilia
   sin duplicados. Solo una confirmación de Tesorería o del sistema económico
   competente permite mostrar «pagado», incluso ante pagos parciales o anticipos.
7. Las bandejas por etapa y los informes por persona, centro, unidad y periodo
   consultan datos propios autorizados y paginados. Los resúmenes por semana,
   mes y año distinguen solicitado, autorizado, liquidado y pagado según fuente.
   La exportación de cada informe tiene autorización, campos y registro propios.

## Autoridades, datos y rendimiento

Dietas posee comisión, líneas, decisiones y liquidación. Personal posee persona,
relación y adscripción; Documentos posee los originales que acepte custodiar;
la autoridad común posee identidad, concesiones y auditoría. Los intercambios
usan puertos, referencias opacas y contratos versionados. Este plan no encarga
cambios en Personal, ADMIN, firmas ni Contratación temporal.

Las reglas y cuantías se publicarán en catálogos de datos con norma o acuerdo,
órgano, aprobación, alcance por colectivo/grupo/país/vehículo/concepto,
vigencia y huella. La comisión conserva la versión aplicada. No se introducen
idiomas ni textos visibles en Go o JavaScript: los catálogos i18n comunes dan
avisos, ayudas y documentos. La interfaz prioriza escritorio y conserva acceso
por teclado y móvil, con estados de carga, error, denegación y recuperación.

Cada lectura y exportación nominal se autoriza para acción, recurso, finalidad,
ámbito, campos y perfil exactos; se registra antes de entregar datos. En cada
efecto, consumo de permiso, estado, historia, auditoría nominal y outbox se
confirman en una transacción. La clave repetida recupera el recibo; contenido
distinto con la misma clave da conflicto. El fallo de identidad, auditoría,
antivirus o sistema externo cierra la operación sin simular éxito.

Las bandejas agregan por lotes y disponen de índices por etapa, unidad, periodo
y orden de continuación. No habrá consulta ni decisión V3 por fila (N+1).
Objetivos medidos con volumen realista: lecturas servidor p95 <300 ms, base
<100 ms y pantalla útil <1 s. Cada ruta nueva aporta `EXPLAIN ANALYZE`, tiempo
de ruta o prueba del número fijo de consultas. La carga local debe medir
centenares y miles de conexiones sin quitar las comprobaciones de seguridad.

## Cortes pequeños y dependencia de cada uno

| Orden | Entrega comprobable | Dependencia |
| --- | --- | --- |
| DIE-01 | El preparador existente consume la versión exacta del catálogo de ejemplo ya disponible y devuelve propuesta reproducible con fuente, vigencia, grupo, país, vehículo, huella y aviso de no liquidable. Pruebas con dos versiones, fecha frontera y tarifa ausente. | Puede avanzarse con datos sintéticos. RRHH validará ámbito y fuentes antes de publicar una tarifa. Primer PR solo del dueño Dietas, sin SQL ni rutas nuevas. |
| DIE-02 | Envío propio conserva comisión y justificantes referenciados, versiones y recibo recuperable; los bytes pasan por adaptador de Documentos con límites y antivirus. | Contrato del custodio de Documentos y política de tipos, conservación y subida; dudas 49/50/60. |
| DIE-03 | Bandeja y decisión por perfil fijo, etapa y unidad exactas, con devolución y reenvío histórico. | Competencias centrales publicadas y contrato de acto de autorización; dudas 25/40/46/51/105/122. |
| DIE-04 | Liquidación económica durable y rectificación enlazada; PDF de la instantánea exacta. | Catálogo admitido, DIE-02/03, postimagen de `000012` y orden SQL acordado. Reserva de migración fuera de Git, ensayo en clon y dos revisiones independientes. |
| DIE-05 | Fiscalización con acto y recibo propio; informe nominal paginado y exportación separada por política de campos. | DIE-04; RRHH e Intervención concretan actos, ámbitos y duda 108/118. |
| DIE-06 | Lote de salida económica con acuse, rechazo y conciliación; reintentos sin doble pago. | Contrato y responsable del sistema receptor; duda 26. La confirmación de pago se consume como hecho externo acreditado. |
| DIE-07 | Recorrido navegador → API → autorización → PostgreSQL → recibo, negativos, reintento y recuperación tras reinicio. Revisión independiente de seguridad, SQL y usabilidad antes de integrar. | Cortes anteriores, textos y configuración aprobados. Dirección integra y verifica la principal cuando levante la congelación. |

Cada corte tendrá un archivo o grupo de archivos de propietario único y commit
autónomo. Los cambios de SQL, HTTP, identidad o datos personales reciben revisión
focal con `security-audit` y Semgrep local; SQL necesita además ensayo en el
clon y revisión SQL independiente. No se sube código a un auditor externo.

**Primer encargo ejecutable:** DIE-01. Conectar el preparador existente al catálogo
de ejemplo versionado de Dietas y devolver una propuesta con procedencia
y tres pruebas de frontera. Cierre: el mismo documento y versión producen la
misma huella e importes; una tarifa ausente o una vigencia incompatible impide
preparar la propuesta. No se cambia el circuito ni se marca la comisión pagada.
