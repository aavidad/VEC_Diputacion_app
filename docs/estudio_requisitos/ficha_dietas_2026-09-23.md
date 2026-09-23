# Ficha de requisitos de VEC Dietas — 23 de septiembre de 2026

Fuente: 5 capturas del «Portal Interno» actual, apartado «Dietas y gastos de locomoción»
(PHP 5), `fotos/dietas/`, tomadas el 20/09/2026. Decisión de alcance del 16/09: Dietas **se
crea de nuevo** con tres puntos de vista (empleado que solicita y justifica, responsable que
autoriza, RRHH que liquida); solo se reutiliza la cartografía interna (OSRM y teselas). No se
reproducen datos personales de las capturas.

## Lo que hace hoy la aplicación

- **Accesos por papel**: RRHH, Intervención, Administrativo, Empleado y Responsable de centro.
- El empleado ve sus **documentos pendientes de completar** (identificador, fecha de
  apertura; editar, borrar, enviar), el **centro y la unidad** a los que está asignado y
  **a quién se enviará** el documento (administrativo y responsable), con opción de corregir
  esa asignación antes de crear uno nuevo.
- **Nuevo documento** con tres bloques y un importe total acumulado:
  - **Dietas**: fecha y hora de inicio y fin, motivo, localidades, país; «calcular dietas»
    devuelve los tramos (tipo de dieta, intervalo, importe) que el empleado acepta.
  - **Kilometraje**: vehículo propio sí/no; salida y llegada (provincia y ciudad); km
    calculados más un ajuste; varias rutas; otros medios de transporte con gasto
    justificado.
  - **Otros gastos**: motivo e importe.
- **Control de documentos**: pendientes de revisar por el administrativo del servicio;
  búsqueda por fechas.

## Requisitos para VEC

| Id | Requisito | Notas |
| --- | --- | --- |
| D1 | Identidad común de VEC; papeles de empleado, administrativo del servicio, responsable de centro, RRHH e Intervención; **sin contraseñas propias**. | Autorización positiva V3. |
| D2 | **Comisión de servicio**: documento del empleado con motivo, fechas y horas, localidades y país. Borrador editable hasta enviarlo. | |
| D3 | **Cálculo de dietas** por tramos (manutención y alojamiento) según fechas, horas y destino, con tabla de importes **versionada** por grupo y país. | Norma de referencia: RD 462/2002; cuantías **a confirmar por RRHH**, rotuladas provisionales. |
| D4 | **Kilometraje** con vehículo propio: ruta origen–destino calculada con la **OSRM interna** (Granada + 15 km), ajuste manual justificado, varias rutas; importe por km versionado. | La cartografía ya existe en la principal. |
| D5 | **Otros medios y otros gastos** con justificante (referencia y huella; el documento en su custodia). | Patrón del justificante de Bolsa B8. |
| D6 | **Circuito**: envío → revisión del administrativo → autorización del responsable → liquidación por RRHH → fiscalización por Intervención; devolución con motivo en cada paso; recibo e historia de solo adición. | |
| D7 | **Asignación** de centro, unidad, administrativo y responsable de cada empleado, corregible con registro. | Referencias de Personal, no copia de sus datos. |
| D8 | Bandejas por papel (pendientes de revisar, autorizar, liquidar, fiscalizar) y búsqueda por fechas. | Lienzo R10. |
| D9 | Informe y PDF del documento liquidado. | |

## Primer corte propuesto

D2 + D3 + D4 para **el empleado**: crear una comisión de servicio con dietas calculadas y
kilometraje por la OSRM interna, importe total y envío con recibo; después D6 con la
revisión y la autorización. Tablas de importes rotuladas provisionales.

## Pendiente de RRHH

Tabla de cuantías vigente por grupo y país; importe por km; reglas de tramos horarios; quién
revisa y autoriza en cada centro; integración de la liquidación con la nómina.
