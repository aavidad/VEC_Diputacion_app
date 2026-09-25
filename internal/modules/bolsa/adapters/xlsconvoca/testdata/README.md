# Fixtures XLS sintéticos de Convoca

Este directorio no contiene exportaciones reales ni datos de personas reales.
Los CSV de `fuentes_sinteticas/` son la fuente legible y auditable de cuatro
libros BIFF8 de prueba:

- `resumen.xls`: cabecera literal de ocho columnas, dos filas válidas y una
  fila con total incoherente;
- `detalle.xls`: cabecera literal de doce columnas, dos méritos válidos y una
  fila con documento enmascarado inválido;
- `formula.xls`: una celda fórmula que el staging debe rechazar;
- `cabecera_desconocida.xls`: una cabecera parecida, pero no autorizada.

Libros del formato real `convoca:v2` y del formato T17 `convoca:v1` con el
nombre de hoja real, generados con `scripts/paquete_ejemplo/generar.py`
(`--cabeceras convoca` y `--cabeceras vec`, una bolsa de cuatro candidaturas):

- `convoca_v2_resumen.xls`, `convoca_v2_detalle.xls`: cabeceras literales de
  CONVOCA en las hojas `grupo de méritos (Tribunal) (1)` y `méritos (1)`;
- `convoca_v1_hoja_real_resumen.xls`, `convoca_v1_hoja_real_detalle.xls`:
  cabeceras T17 con esos mismos nombres de hoja.

Libros de una fila generados con
`fuentes_sinteticas/generar_negativos_convoca_v2.py`: `convoca_v2_nfd.xls`
(cabeceras reales en NFD, se aceptan) y, rechazados, `convoca_v2_hoja_sufijo_2.xls`,
`convoca_v2_hoja_ajena.xls`, `convoca_v2_hoja_cruzada.xls`,
`convoca_v1_hoja_cruzada.xls` y `convoca_mezcla_formatos.xls`.

Los nombres, documentos enmascarados y méritos son deliberadamente sintéticos.
No se copiaron filas, metadatos ni bytes de los ficheros inspeccionados para
definir T17. Los binarios se generaron con LibreOffice usando el filtro
`MS Excel 97`; las pruebas solo requieren los binarios ya incluidos.

Huellas SHA-256 del corte inicial:

```text
b4bb0d12efe5f1a0a353ac62ac1553bd41d828ea74de7cc1f503d6d2dd82425b  cabecera_desconocida.xls
c76345615f3bd7f189cdbb3b5ee419dbfb658d7c48d0931873a4f09aa12f8d30  detalle.xls
6388c9ec5638f3c8143940c5d5b9454f92d18a5302701e14052fff74c71f031f  formula.xls
2ad88853ca3190bb22c323f96c29a9ea700f03f24274797bd6e137dc6c8fd0e9  resumen.xls
```
