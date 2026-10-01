# Preparar hechos del Registro Único de Méritos

Esta utilidad revisa un JSON sintético y exporta sus hechos, versiones y comprobaciones pendientes. Rechaza referencias o fechas inválidas, versiones discontinuas, duplicados del hecho original, estados incoherentes y campos desconocidos. Cada hecho se vincula a una referencia de Persona; un aspirante externo no necesita referencia de empleado.

```sh
go run ./cmd/vec-meritos-preparar < data/ejemplos/meritos/preparacion.json > /tmp/meritos-preparados.json
```

La salida conserva todas las versiones, los estados afirmados por la entrada y las referencias de Documentos. `vigencia_en_corte` compara fechas civiles inclusivas con la fecha de corte aportada; no interpreta el hito de cumplimiento de unas bases. Los motivos de revisión son referencias opacas.

Los nombres del ejemplo son ficticios. `preparacion_sintetica` es el único alcance aceptado. La utilidad no consulta bases, personas ni documentos; no registra hechos, acredita revisiones, verifica firmas, asigna puntos ni concede derechos. Incluso las instantáneas que afirman estar acreditadas conservan pendientes la correspondencia de Persona, procedencia, Documentos y autoridad revisora. Asistencia y superación de un curso son hechos distintos.

RUM01 aporta el modelo y esta comprobación local. Declaración y rectificación autorizadas, historia durable, lectura propia y entrega desde Formación corresponden a los cortes posteriores del plan. Antes de consumir datos reales hacen falta los contratos y las autorizaciones de sus propietarios.

El comando lee como máximo 1 MiB desde la entrada estándar y escribe únicamente en la salida estándar. Ante error devuelve código 1, una clave en la salida de errores y ningún resultado de preparación. No incluye el contenido recibido en el mensaje de error.
