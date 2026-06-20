# Modulo Personal/Nominas para VEC

Fecha de consulta: 2026-06-19.

## Objetivo

Construir `vec.module.personal` como modulo independiente de VEC para expediente
de empleado, puestos, situaciones administrativas, antiguedad, servicios
prestados, certificados, nomina e incidencias retributivas.

Cronos, Dietas y Bolsa no deben recalcular datos maestros. Deben consumir
referencias normalizadas del modulo Personal:

- `employee_ref`
- `position_ref`
- `administrative_status_ref`
- `seniority_ref`
- `service_period_ref`
- `payroll_period_ref`
- `certificate_ref`

## Normativa base a modelar

- TREBEP, RDL 5/2015: retribuciones basicas y complementarias, sueldo,
  trienios, pagas extraordinarias, funcionarios interinos, personal laboral,
  retribuciones diferidas e indemnizaciones.
  https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719
- RDL 14/2025: incremento retributivo del sector publico para 2026.
  https://www.boe.es/diario_boe/txt.php?id=BOE-A-2025-24445
- Secretaria de Estado de Presupuestos y Gastos: tablas anuales de retribuciones
  del personal funcionario, haberes reguladores, derechos pasivos y mutualidades.
  https://www.sepg.pap.hacienda.gob.es/sitios/sepg/es-ES/CostesPersonal/EstadisticasInformes/paginas/retribucionespersonalfuncionario.aspx
- Orden PRE/390/2002: captacion automatizada de datos de retribuciones desde los
  procesos mensuales de nomina del sector publico estatal. Exige pensar en
  ficheros de intercambio/auditoria, no solo pantalla.
  https://www.boe.es/buscar/doc.php?id=BOE-A-2002-3880

## Referencias de producto profesional

- SAP SuccessFactors Employee Central Payroll: calculo automatizado, control
  continuo de nomina, alertas, anomalías, retroactividad y fuera de ciclo.
  https://www.sap.com/products/hcm/employee-central-payroll.html
  https://www.sap.com/products/hcm/employee-central-payroll/features.html
- Cegid/Meta4 Peoplenet: solucion integrada de payroll, personal y time
  management para grandes organizaciones.
  https://www.cegid.com/global/meta4-es-joins-cegid/
- FreeBalance Civil Service Management: enfoque especifico para administraciones
  publicas, control de masa salarial, presupuesto y ciclo de vida del empleado
  publico.
  https://www.freebalance.com/es/productos/gestion-de-la-funcion-publica/
- Sage HR/Payroll self-service: autoservicio para nominas, saldos de permisos,
  reclamaciones de viajes y datos personales/bancarios segun permisos.
  https://apps.apple.com/na/app/sage-hr-payroll-self-service/id1054882197

## Modelo funcional minimo serio

1. Maestro de empleado:
   identificacion, regimen, grupo/subgrupo, puesto, unidad, centro, IBAN,
   situacion administrativa, fecha alta/baja, jornada y perfil horario.
2. Puestos/RPT:
   puesto, nivel, complemento destino, complemento especifico, unidad organica,
   centro de coste, cobertura y compatibilidad con flexibilidad horaria.
3. Antiguedad y servicios prestados:
   periodos reconocidos, trienios, servicios en Diputacion, servicios externos,
   interrupciones, certificado automatico y uso por Bolsa/meritos.
4. Conceptos de nomina:
   sueldo, trienios, complemento destino, especifico, productividad,
   gratificaciones, pagas extra, atrasos, dietas integrables si procede,
   reintegros, anticipos, embargos, IRPF y cotizaciones.
5. Motor de calculo:
   borrador mensual, pagas extra, atrasos/retroactividad, fuera de ciclo,
   regularizaciones, recalculo por cambios de puesto/situacion y comparativa con
   nomina anterior.
6. Cierre:
   prevalidacion, incidencias, aprobacion, firma/publicacion de recibos,
   exportacion bancaria/contable, ficheros para intercambio y auditoria.
7. Autoservicio:
   consulta de recibos, certificado de servicios prestados, certificado de
   retenciones si procede, datos personales/bancarios y reclamacion de nomina.

## Gates de calidad entidad publica

- `gate-nomina-trazable`: cada concepto calculado tiene fuente normativa,
  periodo, version de tabla, actor y referencia de dato maestro.
- `gate-retroactividad`: un cambio con efecto anterior genera atrasos o
  regularizacion, nunca pisa una nomina cerrada.
- `gate-cierre`: una nomina cerrada es inmutable; correcciones van por
  incidencia/regularizacion.
- `gate-certificados`: servicios prestados y antiguedad se calculan desde
  periodos normalizados y producen certificado con CSV/firma cuando existan
  adaptadores documentales.
- `gate-integracion-cronos`: reducciones de jornada y ausencias validadas entran
  como incidencias, no como importes manuales.
- `gate-integracion-dietas`: dietas/liquidaciones aprobadas entran con referencia
  de comision y justificante, no como texto libre.
- `gate-integracion-bolsa`: Bolsa consume certificados/servicios prestados por
  referencia, no duplica ni reinterpreta antiguedad.
