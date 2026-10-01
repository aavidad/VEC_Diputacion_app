# Comprobar versiones de una copia

CS01 compara un manifiesto de copia, el inventario del destino y una política
de versiones. Trabaja con JSON local por entrada estándar y escribe el resultado
por salida estándar. Puede usar el catálogo de textos que se le indique.

```sh
GOCACHE=/dev/shm/go-build go build -p 8 -o /tmp/vec-copias-comprobar ./cmd/vec-copias-comprobar
/tmp/vec-copias-comprobar -catalogo web/static/textos/es/copias-seguridad.json < cmd/vec-copias-comprobar/testdata/compatible.json
/tmp/vec-copias-comprobar -catalogo web/static/textos/es/copias-seguridad.json < cmd/vec-copias-comprobar/testdata/incompatible.json
/tmp/vec-copias-comprobar -catalogo web/static/textos/en/copias-seguridad.json < cmd/vec-copias-comprobar/testdata/no-comprobable.json
rm /tmp/vec-copias-comprobar
```

El primer ejemplo declara una copia completa de una versión anterior a la del
destino. Puede ser compatible porque se recuperarían también sus binarios y todos
sus ficheros. El segundo cambia la versión exacta de PostgreSQL; el tercero declara
incompleto el inventario de destino. Todos los datos, huellas y referencias de
estos ejemplos son sintéticos. No hay claves ni contenido de copias reales.

| Código de salida | Resultado |
| --- | --- |
| 0 | Versiones declaradas compatibles. |
| 1 | Versiones declaradas incompatibles. |
| 2 | Información insuficiente o inventario inválido. |
| 3 | Entrada, argumentos, catálogo o salida inválidos. |
| 4 | No se ha podido escribir el diagnóstico del fallo. |

La salida mantiene siempre `habilita_restauracion: false`,
`autenticidad: no_comprobada` y `verificacion_restauracion: no_comprobada`.
Un resultado `compatible` solo describe las versiones declaradas. No acredita que
existan los ficheros, que estén cifrados, que el manifiesto sea auténtico ni que se
haya probado una restauración. Una declaración `valida` en la entrada tampoco lo
acredita. El programa no abre conexiones ni ejecuta comandos, SQL o restauraciones.

## Entrada

`testdata/compatible.json` contiene el contrato completo de formato 1:

- `manifiesto`: referencias opacas de copia, operación, solicitante, motivo y
  política; fechas UTC, tamaño total, inventario, componentes, consistencia,
  referencias de protección y verificación física/lógica.
- `destino`: versión PostgreSQL, digest del runtime, plataforma, herramientas y
  extensiones; ámbito completo del cluster, bases y almacenes; release, binarios,
  ficheros, esquema esperado y conjunto de migraciones instalado.
- `politica`: catálogo explícito de runtimes y releases admitidos, y releases
  revocadas. Los valores de los ejemplos no son una política operativa.
- `modo`: `conjunto_completo`. Los demás modos, incluido `solo_base`, bloquean.

Cada módulo registra su huella de esquema y **todas** las migraciones con ID y
SHA256. La comparación trata listas como conjuntos. El orden no cambia el resultado;
un duplicado, una huella ausente o un módulo omitido impiden dar compatibilidad.
Una migración con el mismo número y distinto contenido bloquea.

Los nombres de campos distinguen mayúsculas. Todos los campos son obligatorios,
incluidas las listas vacías. Se rechazan campos desconocidos, claves repetidas
en cualquier nivel, `null`, JSON adicional, tamaños fuera de rango, entrada de más
de 4 MiB y anidamiento de más de 32 niveles. Estas son cotas de transporte de la
CLI; no establecen una retención, una frecuencia ni una versión PostgreSQL admitida.

Las referencias son identificadores lógicos de hasta 128 caracteres, sin rutas.
El motivo y la identidad se expresan por referencia, nunca por texto libre, DNI,
correo o credenciales. Los errores de lectura no reproducen la entrada ni rutas.
Los estados y acciones se localizan desde los catálogos de datos, sin textos
administrativos dentro del código Go.

## Qué se compara

Antes de copiar, `CompararInventarios(esperado, observado)` permite a CS02
contrastar el descriptor instalado con su observación. Detecta binarios y ficheros
distintos, esquema incoherente y migraciones omitidas o alteradas. No necesita
fabricar un manifiesto de copia para esa operación.

Antes de preparar una recuperación, `CompararVersiones` compara el conjunto
completo que quedaría instalado. La base debe coincidir exactamente con el esquema
y las migraciones esperados por el binario archivado. La política vigente debe
admitir esa release completa y el runtime PostgreSQL exacto. Una release revocada
bloquea aunque figure también en la lista de admitidas.

El destino puede declarar otra release actual porque se sustituiría por la archivada.
Debe declarar la misma versión de PostgreSQL, runtime, herramientas, extensiones y
plataforma, además de todas las bases y almacenes del ámbito. Una sustitución de
cluster compartido o una restauración solo de base requieren otro procedimiento;
la CLI las bloquea cuando no está declarado el ámbito completo autorizado.

Cada bloqueo identifica la clave comparada y sus dos valores seguros. Para conjuntos
se muestran huellas canónicas; para migraciones, las dos huellas SQL. Las claves
puntuales de módulo/migración usan índices del orden canónico por ID. No se imprimen
nombres de tablas o roles, contenido ni huellas aisladas de configuración secreta.

`HuellaInventario` sella la declaración: SHA256 de valores encuadrados por longitud,
con colecciones ordenadas por identificador. Los consumidores deben validar antes
el inventario para rechazar duplicados. Una huella de integridad no demuestra
autenticidad; esa comprobación pertenece al adaptador de protección de CS03.

## Comprobación y siguientes piezas

```sh
GOCACHE=/dev/shm/go-build go test -p 8 -race ./internal/modules/administracion/domain/copias ./cmd/vec-copias-comprobar
GOCACHE=/dev/shm/go-build go vet -p 8 ./internal/modules/administracion/domain/copias ./cmd/vec-copias-comprobar
```

Quedan pendientes CS02 (bytes e inventario observado), CS03 (cifrado y autenticidad),
las capturas y restauraciones físicas/lógicas aisladas de CS04–06, y los permisos,
registro, doble control, copia previa y pantallas ADMIN de las piezas siguientes.
CS01 no monta rutas, no reserva migraciones y no toca servidores.
