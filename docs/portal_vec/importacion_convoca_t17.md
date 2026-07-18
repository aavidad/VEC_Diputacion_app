# T17 — Importación gobernada de exportaciones Convoca

## Estado del corte

El primer corte vertical está implementado y preparado para revisión. Lee XLS
binario real (BIFF8), detecta estrictamente los dos esquemas documentados,
ejecuta staging y validación por fila, calcula la huella SHA-256, aplica
idempotencia atómica por contenido y produce un acta minimizada. El adaptador
PostgreSQL, la autorización del endpoint interno y la composición productiva
quedan expresamente para el siguiente corte.

No se ha usado, copiado ni incorporado ningún fichero real de Convoca. Los
tests emplean libros sintéticos cuya fuente CSV está versionada junto a ellos.

## Encaje arquitectónico

El flujo conserva la arquitectura hexagonal sin crear un puerto global nuevo:

```text
XLS BIFF8 acotado
  -> adaptador xlsconvoca
  -> HojaStaging neutral
  -> caso de uso importacionconvoca
  -> validación de dominio
  -> RepositorioImportaciones local al caso de uso
  -> staging aceptado + acta atómica
```

Las interfaces del lector y del repositorio viven junto al caso de uso. De este
modo no crece `internal/modules/bolsa/ports`, conforme a DEC-051, y el dominio y
la aplicación no dependen de la biblioteca XLS.

## Esquemas cerrados

No basta contar columnas: nombre, capitalización y orden deben coincidir de
forma literal.

1. Resumen por persona, ocho columnas: `DNI/NIE`, `Primer Apellido`, `Segundo
   Apellido`, `Nombre`, `Turno`, `Experiencia`, `Formacion`, `Total`.
2. Detalle por mérito, doce columnas: las cinco primeras anteriores, `Grupo`,
   `Descripcion del grupo`, `Orden grupo`, `Descripcion del merito`, `Puntos
   autobaremacion`, `Puntos tribunal`, `Motivo`.

Una variante con tildes, columnas reordenadas, añadidas o ausentes no se
interpreta heurísticamente como otro esquema.

## Controles implantados

- Solo contenedor OLE2 y BIFF8, exactamente una hoja.
- Máximo 16 MiB, 100.001 filas, 32 columnas y 64 KiB por celda.
- Rechazo explícito de fórmulas, errores, fechas y booleanos en campos que no
  admiten esos tipos.
- Documento únicamente en la forma enmascarada `***NNNN**`.
- Texto NFC, sin controles ni prefijos peligrosos de hoja de cálculo y con
  límites por campo.
- Decimales positivos canonizados sin `float64`; el total del resumen debe ser
  exactamente experiencia más formación.
- `Orden grupo` entero positivo.
- Cancelación por `context.Context` durante lectura y persistencia.
- Recuperación cerrada ante pánico de un parser al recibir un libro hostil.
- Copias defensivas en la frontera de persistencia.
- CAS concurrente por SHA-256: una reimportación devuelve el acta original y
  no duplica filas.

El acta conserva actor opaco, instante UTC, nombre base del fichero, huella,
esquema, conteos y motivos por número de fila/campo/código. Nunca incluye el
valor rechazado, nombres ni documentos.

## Procedencia y efectos

Cada lote queda marcado de forma estructural con:

- fuente `Convoca (exportacion enmascarada)`;
- autoridad `no_autoritativa`;
- `habilita_actos_con_efectos = false`;
- confirmación obligatoria desde el registro corporativo;
- puntos de autobaremación con uso exclusivo `historico_contraste`.

Alterar cualquiera de esos valores invalida el lote. Esta importación no
identifica de manera plena a una persona, no concede puntuación oficial y no
puede habilitar un llamamiento o contrato. Así ejecuta DEC-079: el motor oficial
configurable calcula la puntuación y la identidad se confirma de oficio.

## Dependencia abierta reutilizada

El adaptador usa `github.com/nkiri/xls` fijado en `v0.0.4`: biblioteca MIT,
etiquetada el 16 de marzo de 2026, pura Go, con lectura BIFF8 desde
`io.ReadSeeker` y tipos de celda explícitos. Excelize se descartó porque soporta
formatos OOXML modernos, no el `.xls` binario exigido por T17. La dependencia
queda aislada en el adaptador y puede sustituirse sin modificar dominio ni caso
de uso.

## Pruebas y siguiente corte

El corte prueba ambos esquemas, cabecera alterada, fórmula, formato hostil,
límites, validación parcial, minimización del acta, huella exacta, cancelación,
copias defensivas e idempotencia concurrente con un único ganador.

Pendiente para considerar T17 completo:

1. adaptador PostgreSQL con transacción única para acta y filas de staging;
2. cifrado de las columnas personales y derivación ciega de búsqueda;
3. autorización interna, auditoría institucional y composición en bootstrap;
4. reconciliación posterior con registro corporativo y modelos de solicitud;
5. política de conservación y borrado del fichero original tras el acta.
