# Ejemplo PDF del saldo de Cronos

Desde la raíz del repositorio:

```sh
TMPDIR=/tmp GOCACHE=/dev/shm/go-build go run ./cmd/vec-cronos-informe-saldo \
  web/static/textos/es/cronos-informe-saldo.json \
  internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json > /tmp/cronos-saldo-ejemplo.pdf
```

El PDF muestra el saldo de Carmen Molina Ortega, persona sintética. Lleva una
marca visible de ejemplo y conserva «No disponible» cuando falta un valor.
El programa exige `demo: true` y emite el documento completo después de validar
la salida del renderer común. Un fallo previo deja stdout vacío. `stderr` sólo emite un código técnico estable;
nunca imprime el mensaje original del renderer ni rutas o datos del documento.

C12 está preparado, pendiente de autoridad productiva. La CLI no usa el caso de
uso `ExportarSaldoPropio`, no consulta servicios ni confirma auditoría. Ese caso
exige una fuente nominal propia y un registro durable que revalide el permiso
de exportación y confirme su consumo y auditoría antes de devolver bytes. No
hay adaptador de esas autoridades, ruta HTTP ni descarga habilitada.

Los catálogos castellano e inglés generan sus textos con el idioma correspondiente
en el PDF: `es-ES` o `en-GB`. La muestra inglesa ya se ha ensayado.
El PDF etiquetado y la validación PDF/UA siguen pendientes.
La muestra permite comprobar legibilidad; no acredita accesibilidad PDF/UA,
instalación, permiso, auditoría durable, firma ni validez administrativa.

Gosec G304 y G703 se justifican en las dos aperturas de ficheros: las rutas
las elige el operador local de esta CLI, con sus propios permisos. No hay
entrada HTTP, cuenta de servicio ni acceso a un servidor. El contenido debe
cumplir el esquema cerrado y estar marcado como sintético antes de emitir bytes.

## Ejemplo de movimientos

El mismo programa genera una lista de fichajes de ensayo:

```sh
TMPDIR=/tmp GOCACHE=/dev/shm/go-build go run ./cmd/vec-cronos-informe-saldo \
  --vista=movimientos \
  web/static/textos/es/cronos-informe-movimientos.json \
  cmd/vec-cronos-informe-saldo/testdata/movimientos.json > /tmp/cronos-movimientos-ejemplo.pdf
```

Para inglés, use `web/static/textos/en/cronos-informe-movimientos.json` como
catálogo. El modo de saldo conserva los dos argumentos del primer ejemplo.
Cada vista exige su propio esquema; intercambiar sus catálogos o entradas falla
antes de escribir el PDF.

La entrada de movimientos contiene `demo: true`, nombre sintético, periodo,
zona horaria, completitud declarada y lista explícita de fichajes. Cada fila
solo admite instante UTC, movimiento y origen opcional. Son los campos de
`ports.MarcajeDia` que la consulta existente proyecta de `FuenteSaldo.Marcajes`;
el origen procede de `TipoOrigen`, nunca se deduce del canal ni de referencias.
No admite programación, libro de saldo, motivos, documentos ni identificadores.

El documento ordena los hechos por instante UTC sin alterar la entrada ni
eliminar coincidencias: hora y movimiento no identifican un evento. Muestra
fecha, hora con desfase UTC y origen declarado o «Sin verificar». El ejemplo
incluye el cambio de hora de Madrid para distinguir las dos horas locales
02:30. Conserva el aviso de fuente incompleta. Una lista vacía no acredita una
ausencia laboral. No calcula jornada, saldo, ausencias ni pares de fichajes.

Esta vista solo prepara ejemplos sintéticos. No consulta registros reales,
consume autorizaciones ni confirma auditoría. El permiso nominal, los tipos y
campos exportables aprobados y la política de exportación siguen pendientes
(duda 118 de RRHH y DPD). No habilita descarga, impresión ni una ruta HTTP.
El PDF etiquetado y la validación PDF/UA siguen pendientes.
