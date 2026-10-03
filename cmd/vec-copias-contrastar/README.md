# Contrastar inventarios de restauración

Esta CLI captura evidencia lógica de PostgreSQL 18 y compara dos inventarios.
Comprueba el contenido de cada tabla, el esquema, roles, permisos, extensiones,
privilegios por defecto, secuencias y objetos grandes. Una celda distinta produce
un resultado diferente aunque ambas tablas tengan el mismo número de filas.

La pieza aporta el contraste lógico de CS06. La verificación completa exige
restauración física y lógica, ficheros, enlaces documentales y arranque del
binario archivado. Un resultado `igual` no valida por sí solo una copia ni
autoriza una recuperación. No conecta ADMIN ni añade permisos de producto.

## Comparar archivos

Compile `./cmd/vec-copias-contrastar`. Las rutas siguientes se proporcionan
mediante configuración privada; los archivos de inventario contienen evidencia
confidencial y deben conservarse protegidos fuera de Git.

```bash
vec-copias-contrastar --modo comparar \
  --esperado "$inventario_origen" --observado "$inventario_restaurado" \
  --catalogo web/static/textos/es/copias_contraste.json
```

Repita la comparación del origen contra cada restauración física y lógica.
El origen y sus componentes deben proceder de la misma ventana de captura.
Comparar solamente las dos restauraciones no demuestra que conserven el origen.

La salida contiene estado y motivos estructurados, sin nombres de tablas,
roles, datos, rutas o errores del proveedor. Los motivos identifican el objeto
por una referencia opaca. El catálogo inglés está en `textos/en`.

| Salida | Significado |
| --- | --- |
| `0` | Captura completa o inventarios iguales. |
| `1` | Diferencia o evidencia no comprobable. |
| `2` | Error de argumentos, configuración, lectura o escritura. |

`--ayuda --catalogo ARCHIVO` muestra la ayuda del idioma elegido. La lectura
rechaza campos desconocidos, alias de nombres, claves repetidas, UTF-8 inválido,
JSON adicional, archivos no
regulares y tamaños superiores a 32 MiB por inventario o 1 MiB de configuración.

## Capturar PostgreSQL

```bash
vec-copias-contrastar --modo capturar \
  --configuracion "$configuracion_privada" \
  --catalogo web/static/textos/es/copias_contraste.json > "$inventario_privado"
```

La configuración JSON requiere `dsn`, `version_postgresql`,
`tiempo_maximo_segundos`, `max_filas`, `max_bytes` y `max_objetos`.
La versión exacta se contrasta con el servidor; el lector admite PostgreSQL 18.
El tiempo máximo es diez minutos; los presupuestos de filas, bytes y objetos
abarcan toda la captura. La evidencia serializada también se limita a 32 MiB.
No guarda ni imprime la conexión. Las credenciales pertenecen al canal de
ensayo y no se toman de la aplicación instalada.

Para comprobar objetos grandes, active `objetos_grandes_semanticos` y declare
`referencias_objetos_grandes` como lista de objetos con `esquema`, `tabla` y
`columna`. Solo se admiten columnas PostgreSQL `oid` declaradas explícitamente.
Un valor distinto de cero y de `NULL` debe apuntar a un objeto grande existente.
Una referencia ausente, desconocida o no declarada devuelve `no_comprobable`.
Sin activar este modo, la presencia de objetos grandes mantiene ese resultado.

El identificador del objeto grande es una identidad lógica que conserva
`pg_dump`; se contrasta junto con las referencias declaradas. El lector sella
los bytes de `lo_get`, su propietario y ACL. Los huecos se leen como ceros;
la distribución en páginas y el tamaño de cada fragmento no intervienen.

El canal directo usa `pgx/v5`, fijado en `go.mod`, con una transacción
`REPEATABLE READ READ ONLY`, representación temporal UTC y lectura sin filtros
RLS. Una réplica con reproducción de WAL pausada permite observar la exclusión
de escritores. En una primaria, la lectura directa conserva evidencia pero
devuelve `no_comprobable` si no tiene un canal que acredite la exclusión.
Una opción declarada en un archivo no acredita esa condición.

Los ejecutores de restauraciones pueden utilizar `Lector.CapturarEjecutor`
con un puerto de ejecución PostgreSQL y un observador de exclusión. El ejecutor
inspecciona los recursos propios, la imagen y la red del entorno aislado antes
y después de la captura. El sello de la evidencia debe permanecer idéntico.
Ese canal permite leer restauraciones primarias aisladas antes de limpiarlas.
Los límites y la base proceden de la configuración externa del ejecutor.

Para inventariar varias bases, el ejecutor configura `BasesInventariadas` con
los nombres de todas las bases del cluster, incluidas las plantillas y la base
de mantenimiento. El lector comprueba el conjunto antes y después de capturarlo.
Una base ausente, adicional o con propiedades cambiantes impide declarar la
evidencia completa. Las claves de tablas, secuencias y objetos grandes incluyen
la base para separar objetos homónimos. El orden de configuración no cambia la
huella. Los presupuestos de filas, bytes y objetos son comunes al conjunto.

Las referencias de objetos grandes incluyen `Base` en este modo. El lector
solo aplica a cada base sus referencias declaradas. Una base no conectable
queda incompleta con el motivo `base_no_conectable_sin_evidencia`; el lector
conserva sus propiedades y ACL y no activa conexiones ni la da por vacía.
`CapturarEjecutorConFuente` admite evidencia de infraestructura vinculada a la
misma base, versión, propiedades y ventana de exclusión. La implementación del
proveedor físico de plantillas pertenece a otra pieza.

La CLI reconoce `bases_inventariadas` y el campo `base` de las referencias.
Su canal DSN carece del observador externo necesario para multibase, por lo
que rechaza la captura antes de conectar con el diagnóstico de captura del
catálogo. El ejecutor de restauración utiliza el puerto anterior; la CLI puede
comparar sus inventarios offline.

## Formato y alcance

`Snapshot` versión 1 contiene versión exacta de PostgreSQL, `completo`, motivos
y objetos con clase, clave privada, cantidad y SHA256. Las ocho clases tienen
inventario obligatorio, incluso cuando no hay secuencias u objetos grandes.
Tablas, secuencias y objetos grandes incluyen entradas individuales y una raíz
ordenada que permite detectar inventarios truncados o incoherentes.

Las filas se codifican con tipo, nombre de columna y valor, conservando `NULL`,
vacío y duplicados. Se ordenan antes de calcular SHA256; el orden físico de
inserción no interviene. Los nombres y propietarios sustituyen los OID internos
en los metadatos. Los objetos grandes se comparan por contenido y metadatos,
conservando objetos idénticos duplicados.

Este lector admite árboles sintéticos simples. Las funciones, tipos propios,
vistas, triggers, políticas RLS, tablas externas y otras estructuras avanzadas
no cubiertas producen `no_comprobable`. No acredita todavía el inventario
completo del esquema de VEC. Un objeto desconocido nunca se omite para afirmar
igualdad. Los OID que identifican otros objetos internos siguen fuera del
alcance admitido; no se reinterpretan como referencias a objetos grandes.

Los archivos offline son declaraciones privadas: SHA256 prueba integridad del
contenido, pero no identifica al emisor. Su procedencia y autenticación deben
conservarse en el manifiesto protegido de la copia. El contraste no conserva
claves, contraseñas de roles, cookies ni una autoridad de permisos paralela.

## Comprobación

Las pruebas focales del dominio cubren diferencias con igual recuento,
orden de objetos, clases ausentes, raíces incoherentes, duplicados, formato
desconocido y cambios en secuencias, ACL y esquema. Las de la CLI comprueban
los estados, mensajes en ambos idiomas y errores sin datos privados.
El ensayo PostgreSQL utiliza exclusivamente recursos sintéticos propios;
no aplica migraciones ni consulta principal, cidonia o bases conservadas.

El ensayo del 1 de octubre de 2026 sobre PostgreSQL 18.4 comprobó igualdad
tras reordenar filas y recrear una tabla con OID nuevo. Detectó diferencias de
celda, CHECK, secuencia, ACL, opción de rol y privilegios por defecto. También
comprobó nombres con `$n`, captura de contenido LO con vínculos pendientes,
estructuras avanzadas y presupuesto de bytes. El contenedor propio se eliminó.

Las regresiones de revisión conservan ACL predeterminadas vacías como reglas
explícitas y rechazan FK con triggers internos desactivados. Los nombres de
campo JSON deben coincidir exactamente con el formato publicado.

La huella ACL incluye los permisos de parámetros de PostgreSQL y lenguajes,
con otorgante, destinatario, privilegio y opción de concesión. Las regresiones
`TestContrasteACLPG18/permiso_parametro` y `permiso_lenguaje` comprueban que
otorgar SET sobre un parámetro o retirar USAGE de un lenguaje produce
`diferente`, y que restablecer esos permisos recupera `igual`.
El propietario de cada lenguaje se sella también cuando su ACL está vacía.
La regresión `propietario_lenguaje_acl_vacia` comprueba ese cambio y su reversión.

Para repetir las pruebas que no abren PostgreSQL:

```bash
GOCACHE="$HOME/.cache/go-build" go test -race -p 8 \
  ./internal/modules/administracion/domain/contrastecopias \
  ./internal/modules/administracion/application/contrastecopias \
  ./internal/modules/administracion/ports/contrastecopias \
  ./internal/modules/administracion/adapters/contrastecopias \
  ./cmd/vec-copias-contrastar
```

Los tests PostgreSQL optativos conservan el contrato del ensayo histórico
(nombre, etiqueta de propietario y digest de imagen). Sin
`VEC_CS06_CONTENEDOR_ENSAYO` se omiten. Una nueva ejecución necesita adaptar
ese contrato a un contenedor sintético propio con `--memory 2g`, datos en disco
bajo la carpeta de estado y `--rm`; nunca se reutiliza una base conservada.
Esta retoma conserva la evidencia del ensayo histórico; las regresiones ACL
siguientes se ejecutan por separado en un recurso desechable nuevo.

Las regresiones ACL usan `VEC_CS06_CONTENEDOR_ACL_ENSAYO`. El responsable
prepara un contenedor PostgreSQL 18.4 propio, sin red, etiquetado
`vec.cs06.owner=contraste-acl`, con `--memory 2g`, datos sintéticos en disco
y `--rm`. Se ejecutan con `go test -p 8 -run '^TestContrasteACLPG18$' -v`
sobre el paquete adaptador; el contenedor y sus datos se retiran al terminar.

El ensayo adicional de objetos grandes usó `pg_dump` y `psql` de PostgreSQL
18.4 sobre el contenedor propio. El contenido y las referencias siguieron
iguales tras restaurar, aunque los huecos se materializaron en más páginas.
Cambiar bytes, propietario, ACL, presencia o referencia produjo diferencias.
Las referencias huérfanas o sin declaración impidieron afirmar igualdad.
La caché compartida de Go agotó su cuota antes de compilar; el ensayo verde
usó una caché privada fuera de Git. El contenedor propio se eliminó.

La regresión multibase opcional usa `VEC_CS06_CONTENEDOR_MULTIBASE_ENSAYO`
y el mismo aislamiento y etiqueta de propietario que el ensayo ACL. Prepara
bases y nombres de personas sintéticos, restaura solo la base auxiliar y
comprueba separación de objetos homónimos, huellas ACL y límites comunes.
Mantiene `template0` como no comprobable sin proveedor: este ensayo no acredita
la verificación completa del cluster ni el arranque de la aplicación.

El ensayo multibase del 3 de octubre de 2026 pasó sobre PostgreSQL 18.4:
contenido por base, restauración auxiliar, permiso de parámetro, propietario de
lenguaje con ACL vacía, orden y límite global de objetos. Las pruebas focales
sin PostgreSQL cubren también límites comunes de filas y bytes y el rechazo
del canal DSN multibase. Se retiraron el contenedor propio y sus datos en disco.
