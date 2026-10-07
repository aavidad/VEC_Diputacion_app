# Almacenar y comprobar un componente sintético

Esta CLI prueba el destino local de CS03 sin arrancar VEC ni PostgreSQL. Guarda
un componente acotado y su manifiesto dentro de un sobre JWE autenticado. No
captura un cluster, restaura datos ni concede permisos de Administración.

Requiere Linux, Go y los catálogos `web/static/textos/{es,en}/copias_destino.json`.
Ejecuta los ejemplos desde la raíz del repositorio. Usa solo ficheros sintéticos.
La raíz de destino debe existir, pertenecer al usuario que ejecuta la CLI y tener
permisos `0700`. El fichero de configuración y la clave deben tener permisos `0600`.
Los componentes y sus antecesores no pueden ser enlaces simbólicos. Tampoco se
aceptan archivos con enlaces duros, dispositivos o tuberías.

Prepara fuera de Git una configuración JSON con estos campos. Las rutas del ejemplo
son marcadores que debes sustituir por tus rutas locales:

```json
{
  "raiz": "/RUTA/PRIVADA/destino",
  "clave_fichero": "/RUTA/PRIVADA/kms/clave-maestra.bin",
  "clave_ref": "clave:copias:ensayo",
  "clave_version": "v1",
  "maximo_claro_bytes": 65536,
  "tiempo_maximo_segundos": 30
}
```

La clave contiene exactamente 32 bytes del material de desarrollo externo. La CLI
no genera claves al arrancar. Deriva la misma subclave dedicada que el proveedor
KMS de desarrollo cargado en VEC, mediante su derivación existente. El identificador
y la versión deben coincidir al recuperar. El algoritmo admitido en este adaptador
es JWE `A256KW` con `A256GCM`, implementado por la biblioteca ya instalada `go-jose`.
La custodia, rotación y política operativas quedan pendientes de Sistemas/Seguridad.

El límite y el tiempo del ejemplo son configurables; no son una política de retención
ni valores aprobados. `maximo_claro_bytes` limita el paquete JSON que contiene el
manifiesto y el componente codificados en base64, incluyendo su sobre de datos. El
límite técnico superior de este adaptador es 1 GiB. El proceso trabaja en memoria;
no sirve para cifrar PGDATA de gran tamaño. El siguiente corte debe componer varios
componentes limitados y comprobar el conjunto completo antes de declararlo válido.

Prepara `contenido.txt` y `manifiesto.json` sintéticos. Guarda el componente:

```sh
go run ./cmd/vec-copias-almacenar -sintetico \
  -idioma es -catalogo web/static/textos/es/copias_destino.json \
  -config /RUTA/PRIVADA/config.json -accion almacenar \
  -entrada /RUTA/PRIVADA/contenido.txt -manifiesto /RUTA/PRIVADA/manifiesto.json \
  -conjunto conjunto:ensayo -componente componente:datos -posicion 1 \
  > /RUTA/PRIVADA/resultado.json
```

Extrae el objeto `referencia` del resultado a `referencia.json`:

```sh
python3 -c 'import json,sys; json.dump(json.load(sys.stdin)["referencia"], sys.stdout)' \
  < /RUTA/PRIVADA/resultado.json > /RUTA/PRIVADA/referencia.json
```

Comprueba el componente en otra ejecución. Para mensajes en inglés, usa
`-idioma en -catalogo web/static/textos/en/copias_destino.json`:

```sh
go run ./cmd/vec-copias-almacenar -sintetico \
  -idioma es -catalogo web/static/textos/es/copias_destino.json \
  -config /RUTA/PRIVADA/config.json -accion comprobar \
  -referencia /RUTA/PRIVADA/referencia.json
```

El resultado incluye solo referencias opacas, tamaño y huella del cifrado. El vínculo
autenticado incluye conjunto, componente, posición, huella del manifiesto y
referencia/versión de la clave. El manifiesto detallado y el contenido quedan cifrados.
No se interpretan rutas incluidas en el manifiesto. La referencia no acredita permisos,
estado de restauración ni que estén presentes todos los componentes del conjunto.

La publicación usa un temporal privado, sincroniza sus bytes y publica el nombre
final de forma atómica y exclusiva. Después sincroniza el directorio. Repetir una
referencia existente falla; no sobrescribe. Un fallo después de publicar puede dejar
un objeto completo sin recibo: requiere reconciliación del orquestador. Si el
proceso cae entre publicar el enlace y retirar el temporal, ambos nombres
pueden conservar el mismo archivo. La comprobación rechazará ese objeto por tener
más de un enlace. Un custodio debe comprobar que el temporal privado y el objeto
son el mismo archivo, retirar solo ese enlace temporal y volver a comprobar la
referencia y autenticación; no basta ignorar la guarda. La prueba focal simula este
estado y acredita recuperación tras retirar el enlace. Este corte no automatiza esa
conciliación. No se promete exactamente una vez ni recuperación de un recibo perdido.

Para retirar el componente sintético, cambia `-accion comprobar` por `-accion borrar`.
El borrado exige comprobar antes la referencia y autenticar el contenido. La política
de retención, aprobación y auditoría corresponden al orquestador; este adaptador no
las sustituye. La raíz privada debe tener un único custodio confiable: otro proceso
hostil con el mismo usuario queda fuera de esta frontera de almacenamiento.

Pruebas focales:

```sh
GOCACHE=/dev/shm/go-build go test -race -p 8 \
  ./internal/modules/administracion/adapters/destinocopias \
  ./cmd/vec-copias-almacenar
GOCACHE=/dev/shm/go-build go test -race -p 8 ./internal/app/bootstrap \
  -run TestCopiasUsaProveedorCargadoYSubclaveDedicada
```

Se comprueban recuperación al reabrir el destino, alteración y truncado con huellas
recalculadas, clave o versión incorrecta, vínculos cruzados, límites, enlaces,
publicación concurrente y borrado seguro por referencia. Son ensayos sintéticos;
CS03 integral y las revisiones criptográficas independientes siguen pendientes.
