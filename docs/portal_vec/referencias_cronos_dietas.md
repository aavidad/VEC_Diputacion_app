# Referencias para modulos Cronos y Dietas

Fecha de consulta: 2026-06-19.

## Cronos: control horario, permisos y vacaciones

- BOE, RDL 8/2019 / Estatuto de los Trabajadores art. 34.9: el registro diario
  debe incluir inicio y finalizacion de jornada. En VEC esto implica que Cronos
  no puede ser solo una tabla de fichajes; debe conservar movimientos, saldos e
  incidencias por dia.
  https://www.boe.es/buscar/doc.php?id=BOE-A-2019-3481
- Guia MITES de registro de jornada: refuerza que deben registrarse inicio y fin
  de jornada, y que conviene tratar interrupciones, jornadas partidas y personal
  desplazado.
  https://www.mites.gob.es/ficheros/ministerio/GuiaRegistroJornada.pdf
- Factorial Help Center: los productos profesionales separan fichaje, ausencias,
  aprobaciones, configuracion de politicas, bloqueos y turnos.
  https://help.factorialhr.com/en_US/time-tracking-absences
- Sesame HR Time Off: gestion profesional de vacaciones/ausencias con saldos,
  aprobaciones, calendarios compartidos, sincronizacion y reportes.
  https://www.sesamehr.com/time-off-management/

## Reglas internas a modelar en Cronos Diputacion

- Perfiles horarios por puesto: flexible administrativo, turno fijo, atencion
  directa a personas mayores u otros puestos sin flexibilidad por cobertura.
- Saldos: horas teoricas, trabajadas, teletrabajo, exceso/defecto diario y
  acumulado mensual.
- Permisos/licencias: tabla tipo Cronos con permiso, solicitar, maximo, minimo,
  solicitado y restante.
- Reducciones por prejubilacion: regla parametrizable validada por RRHH:
  63 anos = 1 hora menos diaria; 64 anos = 2 horas menos diarias.

## Dietas: kilometraje, medias dietas y dietas completas

- BOE, RD 462/2002 sobre indemnizaciones por razon del servicio: marco base para
  dietas, desplazamientos e indemnizaciones del sector publico.
  https://www.boe.es/buscar/act.php?id=BOE-A-2002-10337
- BOE, Orden HFP/792/2023: actualizacion relacionada con importes de uso de
  vehiculo particular en el marco del RD 462/2002.
  https://www.boe.es/buscar/doc.php?id=BOE-A-2023-16461
- Factorial Expenses: formularios separados para gasto normal, kilometraje y per
  diem reducen errores y facilitan cumplimiento.
  https://help.factorialhr.com/en_US/modeule-setup-admins-hr-finance/configure-expenses-forms-regular-per-diem-mileage
- Emburse / Chrome River: uso de calculadora integrada con Google Maps para
  kilometraje, importes bloqueados por tarifa organizativa y datos de ruta.
  https://help.chromeriver.com/hc/en-us/articles/15295572336781-Add-Mileage-Expenses-to-a-Report
- Rydoo: calculo de per diem con tarifas oficiales y detalles del viaje; gestion
  de kilometraje con mapa/ruta y aprobaciones.
  https://www.rydoo.com/expense/per-diem-management/

## Implicacion para VEC

VEC no debe mezclar dominios. Debe exponer una bandeja comun y auditoria comun,
pero Cronos, Dietas y Bolsa conservan modulo, permisos, reglas y almacenamiento
propios. Las relaciones cruzadas deben hacerse por referencias: empleado,
comision, justificante, expediente, auditoria y evento.
