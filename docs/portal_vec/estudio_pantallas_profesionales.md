# Estudio de pantallas profesionales para VEC

Fecha de consulta: 2026-06-19.

Este documento define que debe mostrar cada opcion de menu de VEC, tomando como
referencia aplicaciones profesionales de RRHH, nomina, control horario y gastos.
El objetivo no es inventariar campos aislados, sino definir pantallas operativas:
que trabajo resuelven, que informacion debe estar visible, que acciones admite
cada estado, que sistemas toca y cuando una operacion se considera terminada.

## Fuentes consultadas

- Factorial, Time tracking & Absences: estructura de fichaje, ausencias,
  aprobaciones, politicas, saldos y turnos.
  https://help.factorialhr.com/en_US/time-tracking-absences
- Sesame HR, Time Tracking: fichaje web/app/kiosco, geolocalizacion, panel del
  empleado, trabajo remoto y acceso a vacaciones/nomina.
  https://www.sesamehr.com/time-tracking/
- Personio, Absence Management: calculo de saldos de vacaciones y visibilidad de
  dias tomados/restantes.
  https://www.personio.com/product/absence-management/
- Sage HR & Payroll Self Service: recibos de nomina, saldos de permisos,
  solicitudes de vacaciones, gastos y datos personales/bancarios.
  https://play.google.com/store/apps/details?id=za.co.pastelpayroll.selfservice
- SAP SuccessFactors Employee Central Payroll: nomina continua, alertas,
  retroactividad y procesos fuera de ciclo.
  https://www.sap.com/products/hcm/employee-central-payroll/features.html
- SAP Payroll Control Center: proceso extremo a extremo, analitica de causa raiz,
  auditoria y trazabilidad de quien hizo que y cuando.
  https://learning.sap.com/courses/sap-successfactors-payroll-control-center/running-payroll-using-the-payroll-control-center-pcc-
- FreeBalance Civil Service Management: RRHH y nomina especificos de gobierno,
  ciclo completo del empleado publico, control de masa salarial y presupuesto.
  https://www.freebalance.com/en/products/civil-service-management/
- Workday HCM para sector publico: perfil 360 de empleado con historial, puesto,
  permisos, compensacion, rendimiento y pago.
  https://www.workday.com/content/dam/web/en-us/documents/datasheets/datasheet-human-capital-management-for-government.pdf
- SAP Concur / Emburse / Rydoo: informes de gasto, kilometraje, per diem,
  calculadora de ruta, politica, aprobaciones y reembolso.
  https://help.chromeriver.com/hc/en-us/articles/15295572336781-Add-Mileage-Expenses-to-a-Report
  https://www.rydoo.com/expense/per-diem-management/
  https://learning.sap.com/courses/working-with-primary-configuration-in-concur-expense-professional-edition/understanding-workflows

## Advertencia sobre referencias profesionales

No se debe copiar composicion visual propietaria, identidad de marca, nombres
comerciales internos, orden exacto de paneles, microcopys, iconografia o
maquetacion reconocible de los productos consultados. Las referencias se usan
solo para extraer patrones de informacion y flujo que son comunes en software
profesional: estado legal visible, bandejas de trabajo, validaciones previas,
recibos, trazabilidad, aprobaciones por rol, detalle lateral, filtros densos,
control de errores y cierre auditable.

## Principios de UI y separacion de dominios

- `Tablero VEC` muestra solo vista agregada: volumen global, pendientes,
  alertas cruzadas y accesos a trabajo pendiente.
- Cada modulo muestra informacion propia de su dominio. Ejemplo: `Nominas` no
  usa "personas activas" como KPI principal; usa periodo, incidencias,
  borradores, cierre, pagos y conceptos.
- Las relaciones cruzadas se muestran como referencias, no como datos mezclados.
  Ejemplo: Bolsa consume `certificate_ref` de Personal; Dietas envia
  `expense_ref` a Nomina; Cronos envia `absence_ref` o `reduction_ref`.
- Toda accion legal o economica debe dejar `audit_ref`, actor, fecha/hora,
  estado anterior, estado nuevo, origen del dato y recibo visible.
- Las pantallas de trabajo usan cabecera compacta, contadores utiles, filtros,
  tabla/lista principal, detalle lateral y timeline. No se plantean como
  landing pages ni como tarjetas decorativas.

## Modelo comun de pantalla

Cada pantalla de modulo debe resolver el mismo contrato visual y funcional:

1. Cabecera: modulo, unidad/ambito, periodo activo, rol del usuario, ultimo
   refresco y acciones primarias.
2. Resumen: contadores propios del menu, no KPIs globales del sistema.
3. Filtros: busqueda, unidad, empleado/candidato, estado, periodo, plazo,
   responsable y solo pendientes.
4. Lista principal: tabla densa con ordenacion, seleccion, acciones por fila,
   acciones por lote y estados con texto mas icono.
5. Detalle lateral: resumen del registro, datos maestros, documentos, validaciones,
   referencias cruzadas, historial y acciones disponibles para el estado actual.
6. Estados tecnicos: carga, vacio, filtrado sin resultados, error recuperable,
   borrador, pendiente, bloqueado, cerrado y exportado.
7. Cierre: cada flujo termina con estado persistido, recibo o referencia,
   auditoria y siguiente accion clara.

## Flujo transversal VEC

1. Captura o importacion: el usuario crea, importa o recibe un registro desde
   otro modulo.
2. Validacion: el sistema revisa datos obligatorios, coherencia temporal,
   permisos, duplicados, normativa y referencias externas.
3. Bandeja de trabajo: el registro queda asignado a empleado, responsable, RRHH,
   intervencion, nomina, Bolsa o administracion segun reglas.
4. Decision: se aprueba, rechaza, requiere subsanacion, devuelve al origen,
   recalcula, firma, publica o liquida.
5. Cierre: se genera recibo, referencia de expediente, documento, asiento,
   notificacion, exportacion o evento para otro modulo.
6. Auditoria: se puede reconstruir quien hizo que, cuando, desde que pantalla,
   con que datos de entrada y que resultado produjo.

## Personal

Flujo operativo del modulo: alta o cambio de empleado, vinculacion con puesto,
situacion administrativa, antiguedad, servicios prestados y certificados. El
resultado debe alimentar Nominas, Cronos, Bolsa, Documentos y Auditoria sin que
esas pantallas dupliquen el maestro de Personal.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `personal.dashboard` | Priorizar trabajo de RRHH por unidad, plazo e impacto. | Altas/bajas, expedientes incompletos, cambios de puesto, certificados pendientes, incidencias por unidad, alertas de datos maestros. | Alta empleado, abrir expediente, asignar responsable, generar certificado, filtrar por unidad. | Normal, con alertas, bloqueado por dato maestro, cerrado de periodo. | Nominas, Cronos, Bolsa, Documentos, Auditoria. | Permisos por unidad, periodo vigente, duplicados de empleado, datos minimos de identidad. | Cada cola enlaza a su pantalla origen y los contadores cuadran con filtros activos. |
| `personal.expedientes` | Mantener la ficha 360 del empleado publico. | Identificacion, regimen, grupo/subgrupo, puesto, unidad, centro, situacion, fechas alta/baja, contacto, datos bancarios si procede, historico. | Crear/editar expediente, anexar resolucion, bloquear dato cerrado, ver historial, emitir cambio a modulos. | Borrador, activo, pendiente de validacion, incompleto, baja, bloqueado, historico. | Nominas para pago, Cronos para horario, Documentos para evidencias, Auditoria. | DNI/NIF, empleado unico, fechas coherentes, campos obligatorios por regimen, cambios con fecha efecto. | Ficha guardada con version, `employee_ref`, auditoria y eventos de actualizacion emitidos. |
| `personal.puestos` | Gestionar RPT/plaza/puesto y su cobertura. | Codigo RPT, plaza, puesto, nivel, complemento destino, especifico, unidad, centro coste, perfil horario, cobertura. | Crear/editar puesto, versionar RPT, asociar empleado, cambiar unidad, cerrar cobertura. | Vacante, ocupado, reservado, amortizado, pendiente de publicar, historico. | Nominas para conceptos, Cronos para perfil horario, Bolsa para categoria, Auditoria. | Solape de ocupacion, vigencia de RPT, centro coste valido, categoria compatible. | Puesto queda versionado, sin solapes y con efecto propagado a modulos consumidores. |
| `personal.situaciones` | Registrar situaciones administrativas con efectos operativos. | Activo, baja, excedencia, servicios especiales, comision, interinidad, contrato, fecha efecto, resolucion. | Registrar situacion, corregir fecha, adjuntar resolucion, generar incidencia de nomina, cerrar situacion. | Propuesta, validada, vigente, finalizada, anulada, pendiente de resolucion. | Nominas, Cronos, Documentos, Auditoria. | Fechas sin huecos incoherentes, situacion compatible con regimen, resolucion obligatoria si aplica. | Situacion vigente calculada, historial cerrado y efecto enviado a Nominas/Cronos. |
| `personal.antiguedad` | Calcular y justificar trienios y servicios reconocidos. | Periodos reconocidos, trienios, fecha proximo trienio, servicios externos, interrupciones, fuente, impacto retributivo. | Recalcular, revisar periodo, excluir tramo, enviar a nomina, emitir certificado. | Calculado, pendiente de revision, reconocido, enviado a nomina, impugnado, historico. | Nominas para trienios, Bolsa para meritos, Documentos para resoluciones, Auditoria. | Solapes, periodos no computables, fecha de reconocimiento, topes y fuente documental. | Calculo tiene recibo, version, justificacion y referencia enviada al consumidor. |
| `personal.servicios` | Consolidar servicios prestados para certificados, meritos y trienios. | Periodos, cuerpo/categoria, unidad, jornada, computable bolsa, computable trienio, origen, interrupciones. | Consolidar periodos, corregir solapes, certificar, exportar, marcar no computable. | Borrador, consolidado, pendiente de documento, certificado, rectificado. | Bolsa, Nominas, Documentos, Auditoria. | Continuidad temporal, duplicados, jornada valida, categoria normalizada, fuente obligatoria. | Servicios quedan consolidados con `service_ref` y disponibles para certificados/meritos. |
| `personal.certificados` | Generar certificados reutilizables por empleado y por modulo. | Tipo certificado, empleado, datos incluidos, version, estado firma/CSV, destino, fecha emision, validez. | Generar, previsualizar, firmar, revocar, descargar, enviar a modulo consumidor. | Borrador, pendiente de firma, firmado, enviado, revocado, caducado. | Documentos/firma, Bolsa, empleado, Auditoria. | Plantilla vigente, datos cerrados, firmante autorizado, CSV unico, destino permitido. | Certificado firmado o revocado con `certificate_ref`, CSV y auditoria accesible. |

## Nominas

Flujo operativo del modulo: preparar periodo, recibir incidencias, validar datos
maestros y conceptos, calcular, resolver alertas, cerrar, publicar recibos,
exportar pago y permitir revisiones fuera de ciclo. Las fuentes SAP Payroll
Control Center, SAP SuccessFactors y Sage sirven como referencia de control de
ciclo, recibos y alertas, no de composicion visual.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `personal.nominas` | Controlar el ciclo de nomina por periodo y unidad. | Periodo, estado ciclo, empleados incluidos/excluidos, bruto/neto agregado, borradores, recibos, atrasos, fuera de ciclo. | Abrir periodo, precalcular, recalcular, cerrar, publicar recibos, exportar pago. | Preparacion, calculando, con errores, validado, cerrado, publicado, rectificado. | Personal, Cronos, Dietas, banco, contabilidad, Documentos, Auditoria. | Empleados activos, datos bancarios, conceptos vigentes, incidencias bloqueantes, totales contra periodo anterior. | Periodo cerrado con recibos publicados, exportacion generada y `payroll_run_ref`. |
| `personal.retribuciones` | Mantener conceptos economicos y tablas aplicables. | Sueldo, trienios, destino, especifico, productividad, gratificaciones, pagas extra, IRPF, SS, embargos, vigencia. | Crear concepto, versionar tabla, simular impacto, aprobar cambio, auditar cambio. | Borrador, pendiente de aprobacion, vigente, sustituido, anulado, bloqueado por cierre. | Personal puestos/situaciones, Nominas, Auditoria. | Vigencia sin solapes, concepto permitido por regimen, importes y porcentajes validos, autorizacion. | Concepto versionado con fecha efecto y simulacion archivada si afecto a calculo. |
| `personal.incidencias` | Resolver alertas que impiden o modifican el calculo de nomina. | Incidencias de Cronos, Dietas, cambios de puesto, atrasos, bajas/altas, errores de datos maestros, severidad. | Resolver, devolver a origen, ignorar con motivo, recalcular empleado, bloquear cierre. | Nueva, asignada, pendiente de origen, corregida, ignorada justificada, bloqueante, cerrada. | Cronos, Dietas, Personal, Documentos, Auditoria. | Incidencia con origen valido, responsable, efecto economico, motivo obligatorio si se ignora. | Incidencia queda cerrada o devuelta con motivo; el recalculo no mantiene errores bloqueantes. |
| `personal.integraciones` | Vigilar entradas y salidas tecnicas del ciclo de nomina. | Entradas Cronos/Dietas/Personal, salidas banco/contabilidad/recibos, payload, lote, errores, reintentos. | Reprocesar, conciliar, ver payload, bloquear cierre, descargar lote, marcar resuelto. | Pendiente, recibido, procesado, error, reprocesado, conciliado, bloqueado. | Cronos, Dietas, Bolsa si aplica, contabilidad, banco, firma, Auditoria. | Esquema valido, idempotencia, totales de control, lote no duplicado, permisos de reproceso. | Integracion queda conciliada o bloqueada con causa y no deja estados ambiguos. |

## Cronos

Flujo operativo del modulo: fichaje, calendario, permisos, vacaciones, reducciones
63/64, incidencias y saldos. Cronos debe distinguir jornada planificada,
presencia real, ausencia autorizada y efecto economico. Los patrones vienen de
Factorial, Sesame y Personio.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `cronos.dashboard` | Mostrar la situacion diaria y las colas de tiempo. | Dia actual, saldo mes, fichajes sin cierre, permisos pendientes, vacaciones, teletrabajo, incidencias por unidad. | Fichar, justificar, abrir calendario, aprobar, cerrar mes si autorizado. | Al dia, con incidencias, pendiente de aprobacion, mes cerrado, error de fichaje. | Personal para horario/puesto, Nominas para incidencias, Auditoria. | Rol, empleado activo, perfil horario vigente, periodo no cerrado. | Cada contador abre una cola filtrada y los saldos coinciden con el detalle. |
| `cronos.fichajes` | Registrar y corregir presencia diaria con trazabilidad. | Entrada, salida, pausas, ubicacion/canal, teletrabajo, origen, motivo, firma/auditoria, dispositivo. | Fichar entrada/salida, alta manual, corregir, justificar, bloquear dia cerrado. | Abierto, completo, incompleto, corregido, pendiente de aprobacion, rechazado, cerrado. | Personal, Auditoria, Nominas si genera incidencia. | Secuencia temporal, duplicados, ubicacion si exigida, permisos para correccion, dia no cerrado. | Jornada queda cerrada o justificada con `time_entry_ref` y auditoria. |
| `cronos.horarios` | Definir perfiles horarios aplicables a puestos/personas. | Perfil, tramos entrada/salida, tramo obligatorio, flexibilidad, unidad, puesto, vigencia, excepciones. | Crear perfil, editar, asignar a puesto, clonar, marcar sin flexibilidad, cerrar vigencia. | Borrador, vigente, pendiente de aprobacion, sustituido, historico. | Personal puestos, Cronos saldos, Auditoria. | Vigencia sin solapes, horas teoricas, compatibilidad con puesto, aprobacion de RRHH. | Perfil publicado y asociado sin romper saldos ya cerrados. |
| `cronos.incidencias` | Resolver desviaciones de jornada. | Sin salida, defecto/exceso, fuera de perfil, solape permiso, ausencia injustificada, severidad, empleado, fecha. | Justificar, aprobar, rechazar, pedir subsanacion, enviar a nomina si afecta. | Nueva, asignada, subsanacion, aprobada, rechazada, enviada a nomina, cerrada. | Personal, Nominas, Documentos justificantes, Auditoria. | Motivo obligatorio, documento requerido, impacto economico, no duplicidad para misma fecha. | Incidencia cerrada con decision y, si aplica, `payroll_incident_ref`. |
| `cronos.permisos` | Solicitar y aprobar permisos con saldo y normativa visibles. | Permiso, maximo, minimo, solicitado, restante, justificacion, fecha/hora, responsable. | Solicitar, aprobar, denegar, modificar, adjuntar justificante, cancelar. | Borrador, solicitado, aprobado, denegado, subsanacion, disfrutado, cancelado. | Personal, Documentos, Nominas si tiene efecto, Auditoria. | Saldo suficiente, plazo, solape, justificante obligatorio, responsable competente. | Permiso queda aprobado/denegado y actualiza calendario y saldo. |
| `cronos.vacaciones` | Gestionar vacaciones con saldo y cobertura de unidad. | Calendario, saldo anual, dias tomados/restantes, solicitudes, solapes, bloqueos por unidad. | Solicitar, aprobar, denegar, mover periodo, bloquear fechas, exportar calendario. | Planificada, solicitada, aprobada, denegada, disfrutada, cancelada, bloqueada. | Personal, Cronos saldos, Auditoria. | Saldo, fechas laborables, solape minimo de unidad, periodo permitido, cierre mensual. | Calendario y saldo quedan actualizados con recibo de decision. |
| `cronos.reducciones` | Aplicar reducciones por edad 63/64 con efecto horario y economico. | Empleado, edad, fecha efecto, reduccion 1h/2h, perfil resultante, resolucion, impacto nomina. | Validar RRHH, aplicar al cuadrante, recalcular saldo, enviar incidencia a nomina, anular. | Detectada, pendiente de RRHH, vigente, enviada a nomina, anulada, historica. | Personal fecha nacimiento/situacion, Nominas, Documentos, Auditoria. | Edad y fecha efecto, compatibilidad con jornada, resolucion, no duplicidad. | Reduccion vigente en calendario y enviada a nomina si afecta retribucion. |
| `cronos.aprobaciones` | Centralizar decisiones de responsables sobre tiempo. | Permisos, incidencias, vacaciones, cambios horario, prioridad, plazo, solicitante, impacto. | Aprobar lote, rechazar, pedir subsanacion, reasignar, ver detalle. | Pendiente, vencida, aprobada, rechazada, subsanacion, reasignada. | Cronos, Documentos, Notificaciones internas, Auditoria. | Competencia del aprobador, lote homogeneo, motivo de rechazo, documentos. | Bandeja sin pendientes seleccionados y decisiones emitidas con auditoria. |
| `cronos.saldos` | Cerrar y explicar saldos diarios, mensuales y anuales. | Teoricas, trabajadas, permisos, vacaciones, teletrabajo, exceso/defecto, saldo anterior, cierre. | Ver detalle fecha, recalcular, exportar, cerrar mes, reabrir con permiso. | Abierto, calculado, con incidencias, cerrado, reabierto, exportado. | Fichajes, permisos, Nominas, Auditoria. | Dias completos, incidencias resueltas, periodo no duplicado, autorizacion para reabrir. | Mes cerrado con totales firmes y exportacion disponible. |

## Dietas

Flujo operativo del modulo: solicitud de comision, autorizacion, justificantes,
kilometraje, dietas, aprobacion, liquidacion y envio a nomina/contabilidad. Los
patrones vienen de SAP Concur, Emburse/Chrome River y Rydoo.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `dietas.dashboard` | Priorizar gastos y comisiones por riesgo, importe y plazo. | Comisiones pendientes, gastos fuera politica, km a validar, liquidaciones, importes por unidad, errores de justificante. | Nueva comision, aprobar, liquidar, exportar, filtrar fuera de politica. | Normal, con alertas, pendiente de aprobacion, listo para liquidar, bloqueado. | Personal, Nominas, contabilidad, Documentos, Auditoria. | Empleado activo, unidad, permiso de gasto, periodo abierto. | Colas enlazan a registros accionables y totales cuadran con liquidaciones. |
| `dietas.comisiones` | Autorizar desplazamientos antes de gasto o liquidacion. | Empleado, motivo, origen/destino, fechas/horas, autorizacion, centro coste, vehiculo, estado. | Crear comision, aprobar, cancelar, modificar, vincular justificante, duplicar viaje. | Borrador, solicitada, autorizada, rechazada, en viaje, cerrada, cancelada. | Personal, Cronos ausencias, Documentos, Auditoria. | Fechas, motivo, centro coste, solape con ausencia, aprobador competente. | Comision queda autorizada o rechazada con `travel_ref`. |
| `dietas.kilometraje` | Calcular y justificar kilometraje reembolsable. | Fecha, origen, destino, paradas, ida/vuelta, km calculados, km ajustados, tarifa, motivo, vehiculo. | Calcular ruta, ajustar km con motivo, validar, rechazar, asociar a comision. | Calculado, ajustado, pendiente de validacion, validado, rechazado, liquidado. | Mapa provincial, comisiones, Nominas/contabilidad, Auditoria. | Ruta valida, tarifa vigente, ajuste justificado, duplicidad, maximos por politica. | Kilometraje validado con `mileage_ref` y listo para liquidacion. |
| `dietas.mapa_provincia` | Mantener referencia de rutas y distancias provinciales. | Municipio origen/destino, distancia, tiempo estimado, ruta preferente, peajes si aplica, vigencia. | Seleccionar ruta, anadir parada, actualizar tabla, marcar ruta excepcional. | Vigente, pendiente de revision, sustituida, excepcional, historica. | Kilometraje, Admin catalogos, Auditoria. | Municipios normalizados, distancia positiva, vigencia, no duplicidad de ruta. | Ruta publicada y disponible para calculos nuevos sin alterar liquidaciones cerradas. |
| `dietas.dietas` | Calcular manutencion, alojamiento y per diem segun politica. | Media dieta, dieta completa, alojamiento, manutencion, horario salida/llegada, regla aplicada, importe. | Calcular dieta, aplicar excepcion, pedir justificante, rechazar concepto. | Borrador, calculada, fuera politica, pendiente de justificante, aprobada, liquidada. | Comisiones, Documentos, Nominas/contabilidad, Auditoria. | Horarios, pernocta, importes maximos, justificante si aplica, no duplicidad. | Dieta aprobada con regla visible y preparada para liquidacion. |
| `dietas.justificantes` | Capturar y validar evidencias de gasto. | Ticket, factura, autorizacion, certificado asistencia, hash/CSV, OCR, importe, proveedor, estado. | Subir, validar, rechazar, solicitar subsanacion, vincular gasto, descargar. | Recibido, OCR pendiente, valido, rechazado, subsanacion, duplicado, archivado. | Documentos, comisiones, liquidaciones, Auditoria. | Formato, hash duplicado, importe legible, fecha compatible, documento obligatorio. | Justificante queda aceptado o rechazado con motivo y `document_ref`. |
| `dietas.aprobaciones` | Decidir gastos segun politica y responsabilidad. | Politica, importe, km, justificantes, historial aprobacion, desviaciones, responsable. | Aprobar, rechazar, escalar, devolver, aprobar lote, ver evidencia. | Pendiente, vencida, aprobada, rechazada, escalada, subsanacion. | Dietas, Documentos, Notificaciones internas, Auditoria. | Aprobador competente, motivo rechazo, politica, documento valido, importe dentro de limite. | Decision emitida con recibo y siguiente estado de liquidacion. |
| `dietas.liquidaciones` | Preparar pago/reembolso y conciliacion. | Lote, empleado, importe, conceptos, estado pago, enlace nomina/contabilidad, errores, fecha. | Liquidar, exportar, enviar a nomina, enviar a contabilidad, conciliar pago, reabrir. | Preparada, exportada, enviada, pagada, conciliada, error, reabierta. | Nominas, contabilidad, banco si aplica, Auditoria. | Conceptos aprobados, empleado pagable, lote no duplicado, totales de control. | Lote conciliado o enviado con `expense_ref` y estado de pago visible. |

## Bolsa

Flujo operativo del modulo: convocatoria, solicitudes, documentos, meritos,
autobaremo, subsanaciones/alegaciones, listados provisionales y definitivos. El
modulo consume certificados y servicios de Personal, pero conserva su propio
expediente de Bolsa.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `bolsa.dashboard` | Priorizar expedientes de convocatoria y plazos. | Convocatorias, solicitudes, subsanaciones, certificados pendientes, listados, alegaciones, vencimientos. | Abrir expediente, publicar listado, exportar, asignar revisores, filtrar por plazo. | Abierta, con subsanaciones, en baremacion, pendiente de publicacion, cerrada. | Personal certificados, Documentos, Notificaciones, Auditoria. | Permisos por convocatoria, calendario, datos de convocatoria vigentes. | Cada contador enlaza a cola accionable y alerta plazos reales. |
| `bolsa.convocatorias` | Configurar y versionar procesos selectivos o bolsas. | Bases, plazas/categorias, requisitos, fechas, baremo, tribunal/responsables, version, estado publicacion. | Crear, versionar, publicar, suspender, cerrar, duplicar convocatoria. | Borrador, publicada, en plazo, suspendida, cerrada, historica. | Documentos, Notificaciones, Admin catalogos, Auditoria. | Fechas, baremo completo, version firmada, requisitos obligatorios, permisos de publicacion. | Convocatoria publicada o cerrada con version y documento oficial. |
| `bolsa.solicitudes` | Revisar admision de candidatos y expediente documental. | Candidato, estado, documentos, certificado servicios, meritos, plazo, requerimientos, canal de entrada. | Revisar, requerir, admitir, excluir, pedir subsanacion, fusionar duplicado. | Borrador, presentada, en revision, admitida, excluida, subsanacion, desistida. | Documentos, Personal certificados, Notificaciones, Auditoria. | Identidad, requisitos, documentos obligatorios, plazo, duplicidad de solicitud. | Solicitud queda admitida/excluida o en subsanacion con motivo notificado. |
| `bolsa.meritos` | Validar meritos y calcular puntos aplicables. | Merito, fuente, certificado Personal, evidencia, categoria, puntos declarados, puntos aplicados, tope. | Validar, rechazar, recalcular, pedir evidencia, aplicar tope, bloquear merito. | Declarado, pendiente, validado, rechazado, topado, impugnado, cerrado. | Personal servicios/certificados, Documentos, Autobaremo, Auditoria. | Fuente, fechas, duplicidad, categoria, tope, documento, correspondencia con convocatoria. | Merito queda validado/rechazado con puntos y justificacion. |
| `bolsa.autobaremo` | Simular y fijar baremacion segun reglas de convocatoria. | Reglas, puntos declarados, puntos aplicados, topes, avisos, desglose por categoria, recibo calculo. | Simular, recalcular, cerrar baremo, explicar diferencia, exportar. | Simulado, con avisos, pendiente de revision, cerrado provisional, cerrado definitivo. | Meritos, Convocatorias, Auditoria. | Reglas vigentes, todos los meritos resueltos, topes, redondeos, version de baremo. | Calculo queda cerrado con `score_ref`, version de reglas y desglose reproducible. |
| `bolsa.alegaciones` | Resolver impugnaciones y subsanaciones con trazabilidad. | Alegacion, item impugnado, plazo, documentos, propuesta resolucion, tecnico, estado, notificacion. | Admitir tramite, resolver, estimar, desestimar, pedir informe, notificar. | Presentada, en estudio, pendiente de informe, estimada, desestimada, notificada, cerrada. | Documentos, Notificaciones, Meritos/Listados, Auditoria. | Plazo, legitimacion, item existente, documento, motivo de resolucion. | Alegacion notificada y efecto aplicado al expediente/listado si procede. |
| `bolsa.listados` | Generar, revisar y publicar listados provisionales/definitivos. | Tipo listado, ranking, posicion, puntos, exclusion, firma, publicacion, CSV, version, reclamaciones. | Generar, validar, firmar, publicar, retirar, exportar, abrir plazo de alegaciones. | Borrador, validado, firmado, publicado, retirado, definitivo, historico. | Autobaremo, Documentos/firma, Notificaciones, Auditoria. | Todas las solicitudes resueltas, baremos cerrados, orden estable, firma, permisos. | Listado publicado con version, CSV, fecha y enlace a expediente de publicacion. |

## Documentos

Flujo operativo del modulo: registrar documentos, validar metadatos, gestionar
versiones, firma/CSV, vinculacion con expedientes y archivo. Debe servir como
repositorio transversal sin invadir la logica de Personal, Cronos, Dietas o
Bolsa.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `documentos.repositorio` | Encontrar y gobernar evidencias por expediente, empleado o modulo. | Documento, tipo, propietario, modulo origen, expediente, version, hash, CSV, fecha, estado, confidencialidad. | Subir, vincular, descargar, reemplazar version, archivar, revocar enlace. | Borrador, recibido, validado, firmado, archivado, revocado, caducado. | Personal, Cronos, Dietas, Bolsa, firma, Auditoria. | Formato, tamano, hash duplicado, metadatos obligatorios, permisos de acceso. | Documento queda guardado con `document_ref`, hash y permisos correctos. |
| `documentos.firma_csv` | Controlar firma, CSV y verificabilidad de documentos. | Documento, firmante, certificado, CSV, sello de tiempo, estado firma, errores, fecha. | Enviar a firma, reintentar, validar CSV, revocar, descargar justificante. | Pendiente, firmado, error firma, CSV valido, CSV invalido, revocado. | AutofirmaV2/firma, archivo, modulos consumidores, Auditoria. | Firmante autorizado, documento cerrado, certificado valido, CSV unico, sello temporal. | Documento verificable con CSV o error bloqueante visible y trazado. |
| `documentos.plantillas` | Mantener plantillas oficiales usadas por certificados y resoluciones. | Plantilla, modulo, version, campos, firmantes, vigencia, ultima prueba, estado. | Crear version, probar mezcla, publicar, retirar, clonar. | Borrador, en prueba, publicada, retirada, historica. | Personal certificados, Bolsa listados, Dietas resoluciones, Auditoria. | Campos requeridos, version sin solapes, prueba con datos anonimizados, permisos. | Plantilla publicada con version y prueba de generacion correcta. |

## Aprobaciones

Flujo operativo del modulo: una bandeja transversal para responsables, RRHH,
intervencion, nomina y administracion. No sustituye la pantalla origen; muestra
lo necesario para decidir y siempre enlaza al detalle completo del modulo.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `aprobaciones.bandeja` | Decidir trabajo pendiente de varios modulos por rol. | Tipo, modulo, solicitante, importe/dias/puntos, plazo, prioridad, resumen, evidencias, responsable. | Aprobar, rechazar, pedir subsanacion, reasignar, abrir origen, aprobar lote. | Pendiente, vencida, aprobada, rechazada, subsanacion, reasignada, bloqueada. | Cronos, Dietas, Bolsa, Personal, Documentos, Auditoria. | Competencia del aprobador, accion permitida por estado, motivo de rechazo, lote compatible. | Decision registrada en origen y bandeja actualizada sin duplicar estado. |
| `aprobaciones.reglas` | Configurar quien aprueba que, con umbrales y suplencias. | Regla, modulo, unidad, umbral, rol, suplente, vigencia, prioridad, excepciones. | Crear regla, probar ruta, publicar, desactivar, versionar. | Borrador, en prueba, vigente, sustituida, desactivada. | Admin usuarios/roles, modulos origen, Auditoria. | Solapes de reglas, rol existente, suplente valido, prueba de enrutamiento. | Regla vigente enruta casos nuevos y mantiene historico para casos abiertos. |

## Auditoria

Flujo operativo del modulo: reconstruir hechos, no operar negocio. Debe permitir
investigar cambios, decisiones, integraciones y documentos sin depender de logs
tecnicos opacos.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `auditoria.eventos` | Consultar eventos funcionales por actor, expediente, modulo y fecha. | Evento, actor, rol, fecha/hora, modulo, entidad, estado anterior/nuevo, IP/canal, `audit_ref`. | Filtrar, abrir entidad, exportar, marcar investigado, comparar payload. | Registrado, investigado, exportado, retenido, error de integridad. | Todos los modulos, observabilidad, archivo si aplica. | Integridad de cadena, permisos, rango temporal, minimizacion de datos sensibles. | Evento localizado/exportado con evidencia suficiente y sin alterar el registro. |
| `auditoria.trazabilidad` | Reconstruir linea temporal de un expediente o documento. | Timeline, decisiones, documentos, integraciones, recibos, errores, reintentos, responsables. | Ver detalle, descargar paquete, comparar versiones, abrir pantalla origen. | Completa, con huecos, retenida, exportada, bloqueada por permisos. | Documentos, Aprobaciones, modulos negocio, Admin. | Entidad existente, permisos por ambito, referencias resueltas, datos sensibles protegidos. | Timeline explica el estado actual y cada salto tiene evento fuente. |

## Admin

Flujo operativo del modulo: configurar usuarios, roles, catalogos, integraciones,
parametros, monitorizacion y permisos. Admin no debe resolver expedientes de
negocio; solo habilita el funcionamiento controlado del sistema.

| Pantalla | Objetivo operativo | Datos visibles | Acciones | Estados | Integraciones | Validaciones | Criterio de terminado |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `admin.usuarios_roles` | Administrar acceso por rol, unidad y modulo. | Usuario, mecanismo auth, roles, unidades, permisos, suplencias, ultimo acceso, estado. | Alta usuario, asignar rol, suspender, activar, revisar permisos, exportar matriz. | Activo, suspendido, pendiente, caducado, bloqueado, historico. | Kerberos/SSO/certificado, todos los modulos, Auditoria. | Separacion de funciones, rol valido, unidad existente, permisos de admin, caducidad. | Usuario queda con permisos efectivos calculados y auditados. |
| `admin.catalogos` | Mantener listas maestras comunes. | Catalogo, codigo, descripcion, vigencia, modulo consumidor, version, estado, uso. | Crear entrada, editar, versionar, desactivar, importar, validar impacto. | Borrador, vigente, sustituido, desactivado, con dependencias. | Personal, Cronos, Dietas, Bolsa, Nominas, Auditoria. | Codigo unico, vigencia, dependencias abiertas, formato, permisos. | Catalogo publicado sin romper registros existentes y con version historica. |
| `admin.integraciones` | Configurar y supervisar conectores externos. | Sistema, endpoint, credencial/alias, ultimo envio, lote, errores, reintentos, SLA, estado. | Probar conexion, pausar, reintentar lote, rotar credencial, ver payload anonimizado. | Activa, degradada, pausada, error, pendiente credencial, conciliada. | Banco, contabilidad, firma, notificacion, archivo, modulos negocio. | Credencial valida, esquema, idempotencia, permisos, limite de reintentos. | Conector operativo o pausado con causa y sin colas ocultas. |
| `admin.monitorizacion` | Vigilar salud funcional del sistema y trabajos en segundo plano. | Jobs, colas, tiempos, errores, ultimo exito, modulos afectados, alertas, capacidad. | Reintentar job, pausar cola, descargar diagnostico, abrir incidencia, silenciar alerta. | Saludable, degradado, caido, recuperando, en mantenimiento. | Observabilidad, Auditoria, integraciones, todos los modulos. | Permisos de operacion, ventana de mantenimiento, impacto, no duplicar trabajos. | Alerta queda resuelta, escalada o documentada con accion siguiente. |

## Criterios de calidad para cualquier pantalla VEC

- Debe responder que requiere accion ahora, que esta bloqueado, por quien y por
  que plazo.
- Debe separar datos maestros, estado del flujo, evidencias y auditoria.
- Debe mostrar acciones solo si son validas para el rol y estado actual.
- Debe tener estados de vacio, carga, error, filtrado, seleccion, guardado,
  cierre y bloqueo.
- Debe poder explicar el dato: origen, ultima actualizacion, modulo emisor y
  referencia.
- Debe permitir volver del detalle al listado sin perder filtros ni seleccion.
- Debe proteger acciones irreversibles con confirmacion que muestre entidad,
  efecto, fecha y estado resultante.
- Debe dejar una referencia persistente: `employee_ref`, `time_entry_ref`,
  `expense_ref`, `certificate_ref`, `document_ref`, `score_ref`, `audit_ref` o
  equivalente.
