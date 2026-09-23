# Ficha de requisitos de VEC Cronos — 23 de septiembre de 2026

Fuente: 12 capturas de la aplicación actual «Portal del empleado de WCronos» (PHP 5),
`fotos/cronos/`, tomadas el 20/09/2026. Decisión de alcance del 16/09: Cronos **se reescribe**
dentro de VEC; la aplicación actual es la fuente de pantallas, reglas y datos. Esta ficha no
reproduce datos personales de las capturas: todo dato de ejemplo en VEC es sintético.
Seguridad y despliegue: `seguridad_y_despliegue_cronos.md` (enclave interno).

## Lo que hace hoy la aplicación

- **Acceso** por tarjeta y contraseña, con dos organizaciones: Diputación y Patronato
  Provincial de Turismo.
- **Saldo horario**: hoy (resumen y detalle), semana, mes, año y periodo a elegir.
- **Movimientos** (fichajes): hoy, semana, mes, año, periodo; **olvidos de marcaje**;
  **absentismos**; **calendario anual**.
- **Permisos y licencias**: resumen con catálogo por tipo (máximo, mínimo, solicitado y
  restante; en días u horas; días laborables o naturales), quién lo autoriza (marcas «A» y
  «J-A»), solicitud; listado de concedidos del año con fechas, días y horas; pendientes de
  justificar; solicitados pendientes de conceder.
- **Notificaciones** del empleado a RRHH: tipo, fecha a la que se refiere, mensaje de hasta
  512 caracteres y documento adjunto; consulta y pendientes.
- **Mensajes**: avisos de resolución («Se autoriza a … el permiso … desde … hasta …»),
  archivables; consulta histórica.
- **Incidencias / acumulados**, configuración, impresión en PDF, favoritos y ayuda.

Catálogo de permisos visto en la aplicación (valores de configuración, no personales; **a
confirmar por RRHH**): asistencia a exámenes, asuntos propios (6 días), bolsas de días por
trienios y por años de servicio, bolsa horaria por conciliación (30 h), compensación de
festivos, compensación horaria por horas extra, compensación de sábados, compensación de
tiempo y ocio, contrato de relevo, formación (IAAP-INAP y externa), embarazo y maternidad,
enfermedad grave u hospitalización de familiar (1.º y 2.º grado, con más días fuera de la
localidad), enfermedad sin baja, fallecimiento de familiar (1.º y 2.º grado, con más días
fuera de la localidad), gestión de servicio, horas de médico (3 h), horas sindicales (bolsa
mensual), miércoles de Semana Santa y Corpus, nacimiento, progenitor distinto de la madre
biológica, trabajo no presencial, traslado de domicilio (1 día) y vacaciones (22 días).

## Requisitos para VEC

| Id | Requisito | Notas |
| --- | --- | --- |
| C1 | Identidad común de VEC (certificado, después Cl@ve/DNIe): **sin tarjeta y contraseña**. Organización (Diputación o Patronato) como ámbito de la persona. | Decisión de seguridad; sin segunda autoridad de identidad. |
| C2 | **Registro de jornada**: fichajes de entrada y salida con origen (terminal, portal, corrección), append-only y con recibo. | RD-ley 8/2019: registro diario de jornada. |
| C3 | **Saldo horario** por día, semana, mes, año y periodo, derivado de fichajes, jornada teórica y permisos. | Reproducible; sin recalcular en escritura. |
| C4 | **Calendario laboral** anual por organización y centro (festivos, jornadas especiales). | Reutilizar `calendario_habil_laboral_historico.md`. |
| C5 | **Olvidos de marcaje**: solicitud de corrección con motivo, validada por el responsable; el fichaje original no se borra. | Historia de solo adición. |
| C6 | **Catálogo de permisos** versionado: unidad (días u horas), cómputo (laborables o naturales), máximo anual o mensual, mínimo, quién autoriza, justificante exigido. | Valores iniciales: los de la aplicación actual, rotulados «a confirmar por RRHH». |
| C7 | **Solicitud de permiso** por el empleado con saldo restante visible; **resolución** por quien corresponda (responsable y/o RRHH), con motivo, recibo y aviso al empleado. | Circuito de autorización V3 existente. |
| C8 | **Justificación** posterior de permisos que la exigen (documento con referencia y huella; el documento no entra en VEC si hay custodia). | Mismo patrón que el justificante de Bolsa B8. |
| C9 | **Notificaciones** del empleado a RRHH (tipo, fecha, texto ≤ 512, adjunto) y **mensajes** de resolución archivables. | |
| C10 | **Absentismos e incidencias**: listado por periodo para el empleado y agregados para RRHH. | |
| C11 | Vistas de **responsable** (su equipo: presencia, solicitudes pendientes) y de **RRHH** (todas, informes). | |
| C12 | Exportación e impresión en PDF de saldos, movimientos y permisos. | |

## Primer corte propuesto

C2 + C3 (hoy y semana) + C6 + C7 para **el empleado**: fichar, ver su saldo y solicitar un
permiso del catálogo que resuelve su responsable, con recibo. Datos sintéticos, catálogo
rotulado provisional. Después C5, C8, C9 y las vistas de responsable y RRHH.

## Pendiente de RRHH

Catálogo y cuantías definitivas; jornada teórica por colectivo; quién autoriza cada permiso;
origen real de los fichajes (terminales) y su integración.
