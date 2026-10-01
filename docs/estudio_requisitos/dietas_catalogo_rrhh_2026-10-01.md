# Catálogo de tarifas de Dietas: revisión de RRHH — 1 de octubre de 2026

Esta vista permite consultar un catálogo de ejemplo, su historia sintética y
preparar una propuesta de cambio en memoria. Registrar el cambio sigue cerrado
hasta recibir el contrato nominal de gobierno. La preparación no modifica la
tarifa ni añade un asiento a la historia.

## Qué debe distinguir la vista

Cada tarifa conserva concepto, grupo, país, moneda, importe, vigencia, versión
y fuente. La aprobación y la vigencia son datos distintos: una fecha futura
no convierte una propuesta en una tarifa aprobada. El ejemplo normativo del
repositorio no sustituye la tabla vigente pendiente de RRHH.

La historia muestra fecha, actor, acción, motivo y versión. Sus nombres son
sintéticos. En el circuito futuro, el actor procederá de la identidad
acreditada; no se aceptará como campo editable del navegador.

La propuesta recoge importe, vigencia, fuente y motivo. Revisarla no produce
un recibo, una publicación ni un acto administrativo. Al cerrar la vista se
pierde la preparación local; no se utiliza almacenamiento del navegador.

## Fuente y decisión

[SAP Concur: Variable Rate Configuration](https://help.sap.com/docs/CONCUR_EXPENSE/bb83754b1c5541808d50c09901e11475/eddfb60bac924296b94084058007bd63.html)
distingue acceso de edición, colectivos, país, moneda y fecha efectiva.
[RD 462/2002, BOE](https://www.boe.es/buscar/act.php?id=BOE-A-2002-10337)
establece ámbitos, grupos y conceptos que debe identificar la fuente normativa.

La inferencia para VEC, acordada por Codex-G y la revisión de arquitectura, es
separar consulta, preparación y publicación. Esta pieza prepara las dos
primeras. No reserva permisos ni inventa endpoints para la tercera.

## Límite de integración

La vista previa tiene su propio punto de entrada y recibe datos sintéticos.
Queda fuera de los manifiestos productivos. El montaje compartido espera el
turno de G; la escritura y su historia oficial esperan el contrato de D y la
persistencia gobernada. No se ha añadido SQL ni se ha cambiado cidonia.
