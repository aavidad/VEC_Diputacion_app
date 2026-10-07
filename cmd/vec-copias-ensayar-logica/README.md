# Ensayo lógico local con PostgreSQL

La CLI intenta restaurar un volcado lógico y sus roles en un PostgreSQL nuevo
y aislado. Acepta únicamente archivos que el operador haya confirmado como
sintéticos. No recoge datos de un servidor ni acepta un DSN o un contenedor
existente como destino.

Antes de inicializar PostgreSQL compara la imagen local, las versiones de
`postgres`, `psql` y `pg_restore`, y las versiones de origen y herramienta
que contiene el volcado. Una versión distinta o desconocida detiene el ensayo.
También comprueba la versión efectiva del servidor y que no escuche TCP antes
de restaurar. No descarga imágenes.

Este perfil usa `pg_restore --create`: la muestra debe proceder de una base
propia de ensayo, distinta de `postgres`, `template0` y `template1`. Esas bases
ya existen al inicializar el destino; no se borran para admitir un volcado.

Prepare un archivo de configuración local con estos campos:

| Campo | Valor que debe aportar el operador |
| --- | --- |
| `imagen_sha256` | ID SHA256 de una imagen PostgreSQL 18 ya disponible en Docker local, sin el prefijo `sha256:`. |
| `version_postgresql` | Versión exacta de origen y destino, por ejemplo `18.4`. |
| `usuario_bootstrap` | Usuario sintético nuevo, con prefijo `cs06_`, distinto de todos los roles del volcado global. |
| `limite_archivo_bytes` | Límite positivo por archivo, hasta 1 GiB. |
| `cpus` | Entre uno y ocho núcleos. |
| `memoria_bytes` | Memoria del contenedor, entre 64 MiB y 16 GiB. |
| `tiempo_limite_segundos` | Límite del ensayo, entre uno y 1800 segundos; la retirada de recursos tiene un plazo separado. |

La solicitud tiene `sintetica: true` y dos objetos, `dump` y `globals`, cada uno
con `ruta` y `sha256`. El volcado debe estar en formato custom de `pg_dump`;
los globals deben proceder de `pg_dumpall --globals-only`, con sus propietarios
y permisos. Para las muestras de prueba se usa `--no-role-passwords`: los roles
sintéticos no tienen contraseña. No añada datos reales, credenciales ni claves.
Las huellas comprueban los bytes aportados, no acreditan su autenticidad.

```sh
GOCACHE=/dev/shm/go-build go run -p 8 ./cmd/vec-copias-ensayar-logica \
  -configuracion /ruta/local/configuracion.json \
  -solicitud /ruta/local/solicitud.json \
  -catalogo web/static/textos/es/copias-ensayo-logico.json
```

El resultado indica `restauracion_logica_completada` o
`restauracion_logica_fallida`, su etapa, la comprobación que falló y si se
retiraron los recursos. Nunca habilita una restauración operativa. Los mensajes
proceden del catálogo seleccionado y no contienen datos del SQL ni rutas de
archivos. El código de salida es cero para el ensayo completado, uno si falla
y dos ante errores de argumentos, JSON o catálogo.

La ejecución usa el socket Docker local fijo, `--network none`, `--pull never`,
contenedores sin puertos publicados, raíz de solo lectura, límites de recursos
y PGDATA propio en `/dev/shm`. Copia las entradas a un directorio privado y
las monta en modo de solo lectura. No monta el hogar, secretos ni `docker.sock`
dentro del contenedor. Los comandos son argumentos cerrados, sin un shell del
operador. El volcado puede ejecutar SQL: este ensayo requiere una fuente
sintética conocida y revisada, no convierte SQL ajeno en contenido seguro.

La restauración conserva propietarios, ACL y restricciones. No usa
`--no-owner`, `--no-acl`, filtros de objetos ni desactivación de restricciones.
Un error aborta el ensayo. Éxito, fallo, límite de tiempo o cancelación por
SIGINT/SIGTERM retiran exclusivamente
los contenedores y datos creados por esta operación.

## Comprobación con las muestras incluidas

Obtenga el ID sin prefijo de su imagen local PostgreSQL 18.4. Con ese ID, ejecute:

```sh
VEC_CS06L_IMAGEN_SHA256=ID_SHA256_LOCAL GOCACHE=/dev/shm/go-build \
  go test -p 8 -count=1 -run TestEnsayoLogico \
  ./internal/modules/administracion/adapters/ensayologicopg
```

Las pruebas crean su propia fuente sintética, generan un volcado real y
demuestran restauración correcta, versión incompatible, ausencia de un rol
necesario para una ACL y una CHECK cuya función cambió después de insertar
datos. Sin la variable de imagen, esas pruebas de Docker se omiten: no se
presentan como un ensayo realizado.

Esta pieza comprueba que una restauración lógica acaba sin errores. Faltan
comparación de contenido, recuentos, esquema, roles, ACL, secuencias y objetos
grandes, restauración física, arranque de VEC, autenticidad, custodia de claves
y autorizaciones administrativas. No permite declarar válida una copia completa.
