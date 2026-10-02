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

Para un conjunto de bases, configure `BasesInventariadas` en el adaptador como
lista explícita de todos los nombres observados en `pg_database`, incluidas
las bases de mantenimiento y las plantillas. Las referencias a objetos grandes
deben indicar también su `Base`. Una base omitida, repetida o desconocida impide
afirmar que se ha capturado el conjunto completo. No se excluye `postgres` por
considerarla una base de mantenimiento.

`CapturarEjecutorConFuente` captura las bases conectables bajo la misma ventana
de exclusión y conserva propietarios, ACL y propiedades de todas ellas. Comparte
los límites de filas, bytes y objetos; prefija las identidades con la base y
ordena el conjunto antes de sellarlo.

Para una base no conectable, `FuenteBaseNoConectable` debe observar y validar
su material lógico conservado y la procedencia auténtica de su inicialización.
La evidencia se liga a propiedades actuales, versión de PostgreSQL y ventana.
Sin esa evidencia, el resultado es `no_comprobable`. El lector no activa
`template0`, no escribe en el origen y no genera una huella de fábrica supuesta.

`NuevoFuentePlantillaFisica` permite obtener esa evidencia desde el runtime
aislado de una captura física del mismo punto. Recibe un runtime verificado
que vincula artefacto, imagen, operación, conjunto y ventana. Consulta el censo
y los metadatos originales, crea un clon técnico mediante un canal cerrado,
lee su contenido en `REPEATABLE READ READ ONLY` y lo retira antes del censo final.
Las propiedades y ACL pertenecen a la base original, no se deducen del clon.

El runtime conserva solo dos operaciones técnicas de escritura: crear y retirar
su clon. Debe reconciliar una creación confirmada si se pierde la respuesta y
limpiar su base antes de devolver el error. El proveedor rechaza una base
preexistente y no la retira. La autenticación del manifiesto físico corresponde
al constructor y al contexto verificado del ejecutor; SHA256 por sí solo no la
acredita.

El canal directo de esta CLI conserva el alcance de una sola base. Un conjunto
declarado mediante `bases_inventariadas` requiere el ejecutor y la fuente tipada;
el canal DSN lo rechaza antes de abrir la conexión. La comparación offline admite
los inventarios de conjunto producidos por ese ejecutor.

## Formato y alcance

`Snapshot` versión 2 contiene versión exacta de PostgreSQL, `completo`, motivos
y objetos con clase, clave privada, cantidad y SHA256. Las ocho clases tienen
inventario obligatorio, incluso cuando no hay secuencias u objetos grandes.
Tablas, secuencias y objetos grandes incluyen entradas individuales y una raíz
ordenada que permite detectar inventarios truncados o incoherentes.

Las filas se codifican con tipo, nombre de columna y valor, conservando `NULL`,
vacío y duplicados. Se ordenan antes de calcular SHA256; el orden físico de
inserción no interviene. Los nombres y propietarios sustituyen los OID internos
en los metadatos. Los objetos grandes se comparan por contenido y metadatos,
conservando objetos idénticos duplicados.

El perfil 2 captura definiciones de funciones y procedimientos, atributos de
rutinas, tipos enumerados, dominios, compuestos y arrays. Conserva políticas RLS,
triggers internos y de usuario, constraints, comentarios, collations, definiciones
de estadísticas extendidas y búsqueda textual, lenguajes, extensiones y sus
miembros admitidos. Las referencias usan nombres y firmas. El orden de etiquetas
de un enum y la posición viva de columnas se conservan sin sellar OID ni números
internos que cambian al restaurar.

Los compuestos se leen por campos y los arrays por dimensiones, límites y
subíndices ordenados. Un compuesto NULL se distingue de un compuesto cuyos
campos son todos NULL. El lector exige una cuenta técnica con lectura completa,
usando `row_security=off`; conserva y contrasta las políticas y los indicadores
RLS sin alterarlos. Captura el contenido almacenado de las vistas materializadas.
Para una vista ordinaria captura su definición, sin ejecutar la consulta ni sus
funciones.

Los tipos opacos con conversores propios, casts propios, columnas generadas
virtuales, tablas externas, herencia/particiones y métodos de acceso no admitidos
mantienen `no_comprobable`. La guarda de conversores precede a los decompiladores
y a la lectura de valores: una transacción de solo lectura no basta para contener
los efectos externos de una función C. Un objeto desconocido nunca se omite para
afirmar igualdad. Los OID internos no se reinterpretan como referencias LO.

PostgreSQL sella juntos los verificadores almacenados de todos los roles. El
cliente recibe un digest de conjunto que se incorpora al agregado privado de
roles; no recibe contraseñas ni verificadores individuales y no los publica en
los motivos. Los inventarios permanecen privados y deben conservarse dentro del
conjunto autenticado y cifrado.

El formato de campos no cambia. La versión 1 sigue admitida para comparar dos
inventarios legados, pero una comparación entre versiones 1 y 2 devuelve
`no_comprobable`. No mezcle perfiles ni releases del verificador; la fuente de
una base no conectable debe aportar el perfil 2 para una captura actual.

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

Para repetir las pruebas que no abren PostgreSQL:

```bash
GOCACHE=/dev/shm/go-build go test -race -p 8 \
  ./internal/modules/administracion/domain/contrastecopias \
  ./internal/modules/administracion/application/contrastecopias \
  ./internal/modules/administracion/ports/contrastecopias \
  ./internal/modules/administracion/adapters/contrastecopias \
  ./cmd/vec-copias-contrastar
```

El test PostgreSQL optativo necesita preparar previamente su contenedor
sintético exclusivo. Sin `VEC_CS06_CONTENEDOR_ENSAYO` se omite; una prueba
unitaria verde no demuestra una nueva ejecución del ensayo PostgreSQL.

El ensayo adicional de objetos grandes usó `pg_dump` y `psql` de PostgreSQL
18.4 sobre el contenedor propio. El contenido y las referencias siguieron
iguales tras restaurar, aunque los huecos se materializaron en más páginas.
Cambiar bytes, propietario, ACL, presencia o referencia produjo diferencias.
Las referencias huérfanas o sin declaración impidieron afirmar igualdad.
La caché compartida de Go agotó su cuota antes de compilar; el ensayo verde
usó una caché privada fuera de Git. El contenedor propio se eliminó.

El ensayo multibase capturó `postgres`, `template0`, `template1` y `vec_cs11`.
Una restauración con nombre y `--create` conservó la igualdad. Cambiar una celda
o ACL de mantenimiento, o el contenido de `template1`, produjo diferencias.
Una base adicional, evidencia ausente y presupuesto insuficiente bloquearon
el resultado completo. Las cuatro bases necesitaron un presupuesto explícito
de 32 MiB en el fixture; los límites del código de producción no aumentaron.

El proveedor de prueba observó una copia técnica de `template0` en el motor
antes de la ventana y la retiró antes de capturar el conjunto. Consultó las
propiedades y ACL de la base original. Este ejercicio controlado no entrega un
proveedor de inicialización para el circuito real; sin esa fuente, la base no
conectable sigue como `no_comprobable`. No se activó ni escribió `template0`.

El ensayo del proveedor físico detuvo limpiamente un PostgreSQL 18.4 propio,
archivó su PGDATA completo y restauró el archivo en otro runtime aislado con
solo lectura por defecto. Dos capturas completas coincidieron. Las alteraciones
de procedencia, imagen o ventana se rechazaron antes de crear el clon; también
se comprobó la protección de bases preexistentes y la limpieza tras fallo de
lectura o respuesta de creación perdida. Ambos contenedores y el archivo
temporal se retiraron.

Ese ensayo acredita la custodia local del archivo observado y la ejecución
técnica con PostgreSQL. La autenticación criptográfica CS03 y su integración
con el runtime físico real siguen pendientes en el puente de composición.
