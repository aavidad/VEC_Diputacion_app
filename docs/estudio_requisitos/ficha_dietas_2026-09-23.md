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

## Estado al 25 de septiembre de 2026

Contrastado con `origin/main` = `2ad54ce4` (PR #36, #40, #47 y #49) y con
`web/static/portal-empleado/modulos/dietas/INTEGRACION.md`. «Formal» aplica la definición
de terminado del consenso de hoja de ruta del 24/09.

**Formal: 0/9.** Dietas está inactiva en la principal (no hay despliegue ni barrido) y el
recorrido navegador → PostgreSQL con reinicio «corresponde al corte integrado y se documenta
fuera de este archivo» (`INTEGRACION.md`, «Comprobación»): no hay tal evidencia en el
repositorio.

**Técnico: 3/9** (D2, D4, D5). **Uso real en la principal: 0/9** (inactiva; Dietas
000001–000008 instaladas, 000009–000011 y AD3-75/81 pendientes; AD3-81 está en el PR #54,
abierto).

| Id | Técnico (máximo en código y pruebas) | Límite concreto |
| --- | --- | --- |
| D1 | Parcial | Identidad común y V3 sí; ninguna bandeja acredita a nadie hasta la fuente de competencia (PR #40; dudas 40 y 46) |
| D2 | Sí | Alta v1, documento v2, edición, borrado lógico y envío con recibo (000001, 000006, 000007) |
| D3 | Parcial | Tramos por grupo solo para España con tabla provisional; otro país deshabilita guardar (`INTEGRACION.md`, «Ruta y mapa») |
| D4 | Sí | OSRM interna, hasta ocho rutas con diez paradas, ajustes motivados; importe por km provisional |
| D5 | Sí | Tipo de catálogo, fecha, importe y justificante por referencia y huella (000009, PR #49); dudas 49 y 50 |
| D6 | Parcial | Circuito y devolución en SQL y Go (000007, 000008, 000010, 000011, AD3-75/80); `clienteCircuito` no se compone en el portal; vuelta tras devolución pendiente (duda 51) |
| D7 | Parcial | Asignación vía relaciones de Personal; rectificación D7c en código, `clienteRectificacion` no compuesto; sin catálogo competente, confirmar queda deshabilitado |
| D8 | Parcial | Bandejas por papel en código (PR #40), sin componer en el portal |
| D9 | No | No hay informe ni PDF del documento liquidado |
