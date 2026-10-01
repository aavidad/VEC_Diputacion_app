# Ensayo de estado registrado al corte

La CLI lee una instantánea sintética y devuelve el estado registrado de cada persona
seleccionada y un resumen de esos estados. La etiqueta procede del catálogo del idioma.
El ejercicio cubre una parte de C11 y el recuento neutro de C10; no implementa un listado
de absentismos, incidencias administrativas ni solicitudes pendientes.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-cronos-presencia \
  --snapshot data/demo/cronos/presencia-equipo.json \
  --snapshot-sha256 3bff2e709dbc868843cc3d1f9e235a70cf2e384ca2b22a70bc3621cfc92a6c36 \
  --textos web/static/textos/es/cronos-presencia-ensayo.json \
  --textos-sha256 c793612dc99b24ae48e1bd0b7e39f4ca0c838f05a868be37f076f67eb04c1304 \
  --idioma es
```

Para inglés, use `web/static/textos/en/cronos-presencia-ensayo.json`, huella
`8888482d73e1a524a0e5d548eb5504239785cb3b80fe4144d06ecfe5a2761bc2` e idioma `en`.
El idioma debe coincidir con el archivo cargado. La zona es un dato de la instantánea;
el ejemplo utiliza Europe/Madrid. Se rechaza `Local`, porque depende del equipo. Los instantes son UTC y conservan precisión de microsegundos.

El ejemplo contiene seis personas y devuelve este resumen:

```json
{
  "total": 6,
  "entradas_registradas": 1,
  "pausas_registradas": 1,
  "salidas_registradas": 1,
  "indeterminado": 3
}
```

Las tres personas con estado indeterminado representan falta de marcajes, una secuencia
ambigua y cobertura incompleta. Cada resultado indeterminado incluye `motivo`:

| Motivo | Por qué queda indeterminado |
| --- | --- |
| `cobertura_incompleta` | La fuente no cubre todo el periodo hasta el corte. |
| `sin_marcajes` | La cobertura es completa, pero no hay marcajes. |
| `secuencia_ambigua` | Hay instantes iguales o una transición incoherente. |

El consumidor obtiene el mensaje y la acción de `textos[persona.motivo]` en la salida.
Las tres claves están en los catálogos español e inglés; Go devuelve códigos, no frases.
La cobertura incompleta tiene prioridad, incluso sin marcajes o con una secuencia ambigua.
Los estados determinados omiten `motivo`. Los datos inválidos siguen rechazando toda la
instantánea: no se presentan como un estado indeterminado.

Una entrada abierta es `entrada_registrada`; no acredita
presencia física. Los estados no calculan trabajo, minutos, saldo ni absentismo.

La selección `personas_ref` contiene entre 1 y 100 referencias únicas y coincide exactamente
con las personas de la instantánea. Cada persona admite hasta 100 marcajes. Los resultados
siguen el orden de la selección y los recuentos suman su tamaño. La selección delimita el
ensayo; no concede acceso a personas reales.

`completa_hasta_corte` es obligatorio: expresa que la fuente sintética incluye la secuencia
completa desde el inicio de la fecha hasta el corte. No acredita autorización ni fiabilidad
de una fuente real. Sin cobertura o sin marcajes, el estado queda indeterminado. También
queda indeterminado ante transiciones incoherentes o marcajes simultáneos. Los hechos no se
reescriben para recuperar un estado determinado. Una marca fuera de la fecha o posterior
al corte rechaza toda la instantánea; una marca exactamente en el corte sí se incluye.
Los turnos que atraviesan medianoche requieren un contrato posterior y aquí no se resuelven.

La salida conserva referencia y versión de instantánea y fuente, sus bytes mediante SHA-256,
organización, fecha, zona, corte y cobertura por persona. Cada marcaje de entrada tiene su
referencia, versión y fuente exactas. Las referencias del ensayo llevan `demo:`; este
prefijo y `demostracion` no demuestran por sí mismos que un archivo carezca de datos reales.
Use exclusivamente fixtures sintéticos revisados. Las huellas identifican bytes y no son
firma, auditoría ni autorización.

Los archivos son regulares y de hasta 1 MiB. Se rechazan enlaces en el fichero final,
FIFO, directorios, claves duplicadas, campos desconocidos, exceso de tamaño y JSON adicional.
El comando no admite motivos o tipos de permiso, documentos, edad ni coordenadas.
Una entrada inválida termina con código 2, sin resultado en stdout, y escribe en stderr:

```json
{"error":"entrada_invalida"}
```

Los textos para personas están en `web/static/textos/{idioma}/cronos-presencia-ensayo.json`.
El diagnóstico estable es un código técnico; no reproduce rutas ni datos de entrada.

No hay conexión a PostgreSQL, HTTP, permisos, Personal real ni exportación de información
real. D/Personal y las rutas Cronos actuales conservan su cierre. Una lectura real exigirá
resolver las relaciones vigentes en el servidor, una concesión nominal con finalidad,
campos y ámbito exactos, y auditoría de lectura y denegación. La privacidad de agregados de
grupos pequeños sigue pendiente; el ensayo no fija un umbral. Los criterios de calendario
de equipo y días sin marcaje orientan esta separación, sin aprobar reglas de RRHH.
