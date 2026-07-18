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
