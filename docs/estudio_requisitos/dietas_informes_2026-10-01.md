# Informes de Dietas: preparación sintética — 1 de octubre de 2026

RRHH puede revisar una vista de informes por persona, unidad y periodo con un
paquete sintético. La exportación y la impresión esperan el permiso nominal
con auditoría de D. Este corte no consulta expedientes reales ni acredita una
liquidación, una fiscalización o un pago.

## Criterios del ejemplo

Los importes se suman en céntimos enteros y por moneda. Se conserva la unidad
vinculada a cada documento; no se sustituye por la adscripción actual de la
persona. El periodo toma la fecha de inicio de la comisión, indicada en los
datos del ejemplo. Ese criterio permite revisar el filtro, pero no es una
regla contable aprobada por RRHH.

El informe usa importes conservados. No vuelve a calcular comisiones antiguas
con el catálogo actual ni reparte el mismo total entre varios meses. Las
etiquetas distinguen estados sintéticos y no acreditan efectos administrativos.

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
