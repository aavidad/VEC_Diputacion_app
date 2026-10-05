# Informes de Dietas: preparación sintética — 1 de octubre de 2026

RRHH puede revisar una vista de informes por persona, unidad y periodo con un
paquete sintético. Puede filtrar también por situación del ejemplo. Las opciones
proceden de las situaciones incluidas en su configuración; el filtro no amplía
ese conjunto. Lista, recuento y subtotales usan la misma selección.

La situación elegida se aplica con los demás filtros y se conserva al recargar.
Si una nueva configuración deja de incluirla, el resultado queda vacío hasta
cambiar o quitar el filtro. La exportación y la impresión esperan el permiso nominal
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

## Muestra en CSV

La misma selección se puede sacar a una hoja de cálculo desde la terminal:

```sh
go run -p 2 ./cmd/vec-dietas --informe-periodo-csv \
  --textos web/static/textos/es/dietas-informes-csv.json \
  --configuracion data/catalogos/dietas/informes-ejemplo-v1.json \
  --unidad unidad-demo-01 --desde 2026-09-01 --hasta 2026-09-30 \
  < data/demo/dietas/informes.json > /tmp/dietas-informes.csv
```

Los filtros `--persona`, `--unidad`, `--situacion`, `--desde` y `--hasta`
son los de la vista y se pueden omitir. La selección se hace en el código Go
del módulo (`internal/modules/dietas/application/informeperiodo`) con las
mismas comprobaciones que la vista, más límites de tamaño y un esquema cerrado:
la configuración de ejemplo debe coincidir en referencia y versión, y la suma
de los conceptos debe dar el total de cada comisión. El caso de la vista con fecha de liquidación, solo liquidado y
manutención entre el 5 y el 20 de septiembre da los mismos dos registros y
42,50 euros.

Cada fila lleva referencia, versión, persona, unidad, situación, la fecha que
elige la configuración, un importe por concepto incluido y el importe incluido.
Los importes van en céntimos enteros, así la hoja no depende del separador
decimal. No salen las referencias internas de persona ni de unidad. Los textos
que una hoja interpretaría como fórmula llevan un apóstrofo delante. Para
inglés se usa el catálogo de `textos/en/`. Si algo falla, la orden termina
con código 2 y el fichero solo contiene un código de error en JSON; conviene
mirar el código de salida antes de abrir el `.csv`.

Es una muestra local con datos sintéticos. El botón Exportar de la vista sigue
desactivado y la exportación nominal, con permiso propio y auditoría, sigue
pendiente.

## Muestra en PDF

La misma selección sale también en PDF con el generador de documentos común:

```sh
go run -p 2 ./cmd/vec-dietas --informe-periodo-pdf \
  --textos web/static/textos/es/dietas-informes-pdf.json \
  --configuracion data/catalogos/dietas/informes-ejemplo-v1.json \
  --desde 2026-09-01 --hasta 2026-09-30 \
  < data/demo/dietas/informes.json > /tmp/dietas-informes.pdf
```

Admite los mismos filtros que el CSV. El documento empieza con el aviso de
ejemplo sintético y explica qué fecha cuenta para el período. Luego muestra el
período pedido, el número de informes y el importe incluido, el subtotal de
cada concepto y una línea por comisión con referencia, versión, persona,
unidad, situación, fecha e importe. Termina recordando que los importes son
los conservados y que el documento no acredita liquidación, fiscalización ni
pago. Los importes llevan el formato del idioma: `1.234,56 €` en castellano y
`€1,234.56` en inglés. Todos los textos están en
`textos/{es,en}/dietas-informes-pdf.json`, y un nombre que contenga llaves se
imprime tal cual, sin tomarse por plantilla.

Igual que en Cronos, el PDF etiquetado y la validación PDF/UA siguen
pendientes. La muestra sirve para revisar contenido y legibilidad; no habilita
impresión ni descarga en la vista.

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
