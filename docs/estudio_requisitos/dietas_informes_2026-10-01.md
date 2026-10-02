# Informes de Dietas: preparación sintética — 1 de octubre de 2026

RRHH puede revisar una vista de informes por persona, unidad y periodo con un
paquete sintético. La exportación y la impresión esperan el permiso nominal
con auditoría de D. Este corte no consulta expedientes reales ni acredita una
liquidación, una fiscalización o un pago.

## Criterios del ejemplo

Los importes se suman en céntimos enteros y por moneda. Se conserva la unidad
vinculada a cada documento; no se sustituye por la adscripción actual de la
persona. El catálogo de ejemplo elige la fecha de inicio de la comisión. Ese criterio
permite revisar el filtro; no fija una regla contable aprobada por RRHH.

El informe usa importes conservados. No vuelve a calcular comisiones antiguas
con el catálogo actual ni reparte el mismo total entre varios meses. Las
etiquetas distinguen estados sintéticos y no acreditan efectos administrativos.

## Configuración del ensayo

`data/catalogos/dietas/informes-ejemplo-v1.json` elige la fecha del periodo,
los estados y los conceptos incluidos. Conserva referencia, versión e historia
sintética. La vista recibe ese catálogo y los registros por separado; rechaza
referencias o versiones que no coincidan. Una historia de ejemplo no acredita
publicación ni auditoría administrativa.

La configuración inicial conserva ocho registros y 414,20 euros. Una variante
por fecha de liquidación, solo con estado liquidado y manutención, incluye dos
registros y 42,50 euros. Los importes originales se conservan; la vista rotula
el resultado como «Importe incluido».

El gobierno operativo del catálogo, su autorización y su auditoría durable
siguen esperando el contrato nominal. La pregunta sobre fecha, unidad, estados
e importes se remite a RRHH en `dudas.md` durante el turno de E.

## Fuente y decisión

[Odoo Expenses analysis](https://www.odoo.com/documentation/19.0/applications/finance/expenses/expenses_analysis.html)
organiza el análisis por empleados, categorías y periodos.
[Workday Expense Management security](https://doc.workday.com/admin-guide/en-us/financial-management/expenses/expense-management/set-up-security-for-expense-management.html)
separa el acceso a informes de otras capacidades de gastos.

La inferencia para VEC, acordada por Codex-G y la revisión de arquitectura, es
una tabla filtrable con resumen agregado. La lectura de informes y su exportación
necesitarán contratos distintos. No se prepara un constructor general de
informes ni un generador de descargas antes de recibir esa autoridad.

## Límite de integración

La vista previa tiene su propio punto de entrada y recibe el JSON sintético
como dato. Queda fuera de los manifiestos productivos y no añade enlaces al
portal. El montaje compartido espera el turno de G; el consumidor nominal
espera la fuente autorizada de D. Los manuales de RRHH siguen esperando su
confirmación.
