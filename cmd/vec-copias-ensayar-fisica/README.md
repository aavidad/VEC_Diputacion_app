# Ensayo físico local de copias sintéticas

Esta CLI recupera PostgreSQL en un contenedor desechable sin salida a la red.
Acepta el formato de captura CS05: un TAR por componente, con raíz `contenido`.
El primer componente debe ser `fisica:pgdata`, de tipo `base_fisica`.
Los TAR se reciben junto con sus SHA256 desde el índice protegido de la copia.
Un SHA256 suelto no acredita la autenticidad del conjunto.

La CLI devuelve el estado de la restauración física y de su limpieza. Por defecto,
contraste y arranque VEC quedan `no_comprobable`; `habilita_restauracion` siempre
es `false`. Una restauración PostgreSQL correcta no convierte la copia en válida.
La composición del verificador completo debe aportar un observador antes de la
limpieza, contrastar el contenido y arrancar el binario archivado con su perfil.

## Uso

Compile con la versión Go admitida por el repositorio y las dependencias ya
instaladas. No hace falta acceso a una base existente.

```sh
go build -o vec-copias-ensayar-fisica ./cmd/vec-copias-ensayar-fisica
./vec-copias-ensayar-fisica \
  --configuracion configuracion-sintetica.json \
  --solicitud solicitud-sintetica.json \
  --catalogo web/static/textos/es/copias_ensayo_fisico.json
```

Configuración de ejemplo; la imagen debe estar instalada localmente y su digest
exacto debe estar admitido por la política del ensayo:

```json
{
  "imagen_sha256": "DIGEST_HEXADECIMAL_DE_64_CARACTERES",
  "version_postgresql": "18.4",
  "usuario_bootstrap": "cs06_fixture",
  "limite_archivo_bytes": 134217728,
  "limite_extraido_bytes": 134217728,
  "limite_entradas": 10000,
  "cpus": 1,
  "memoria_bytes": 536870912,
  "tiempo_limite_segundos": 180
}
```

Solicitud de ejemplo con archivos sintéticos:

```json
{
  "sintetica": true,
  "componentes": [
    {
      "id": "fisica:pgdata",
      "tipo": "base_fisica",
      "tar": {"ruta": "pgdata-sintetico.tar", "sha256": "SHA256_DEL_TAR"}
    }
  ]
}
```

`usuario_bootstrap` debe existir en el cluster capturado. No se crean roles ni se
aplican migraciones al recuperarlo. El código de salida es `0` si PostgreSQL se ha
recuperado y los recursos se han retirado, `1` si falla el ensayo y `2` si no se
puede leer una entrada o producir la salida.

## Aislamiento y límites

La imagen se referencia por digest y se comprueba su versión antes de enviar SQL.
PGDATA debe ser de PostgreSQL 18, estar parado limpiamente, no contener señales
de recuperación ni un proceso activo y tener `pg_tblspc` vacío. Los tablespaces
externos y los enlaces no están admitidos en este corte.

Antes de extraer se verifican todos los TAR y sus límites acumulados. Se rechazan
rutas absolutas, `..`, duplicados, enlaces simbólicos y duros, dispositivos, FIFO y
modos especiales. La extracción escribe sólo dentro de una raíz privada nueva.
Se conservan los archivos originales y sus huellas; la copia de ensayo usa
configuración PostgreSQL nueva, sin conexiones, replicación o archivado del origen.

La raíz temporal se genera bajo `/var/tmp`, con permisos `0700`. Opcionalmente,
`raiz_temporal` puede señalar una carpeta propia privada directamente bajo
`/var/tmp`; no se admiten enlaces ni bases temporales compartidas elegidas por entrada.
Este corte evita `/dev/shm`: una prueba real detectó cuota agotada al crear PGDATA,
aunque el sistema aún mostraba espacio libre. No modifica cachés ni carpetas ajenas.

Docker usa endpoint Unix local fijo, red `none`, usuario sin privilegios, raíz de
sólo lectura, capacidades retiradas y límites de CPU, memoria, procesos y tiempo.
No se montan bases principales ni sockets del host dentro del contenedor.
El ensayo retira y comprueba sólo los contenedores y archivos que él mismo creó.

## Observación y arranque archivado

Los puertos de `ports/ensayofisicopg` sirven para ambos ensayos. El observador
recibe ejecutores vivos y un sello de aislamiento calculado mediante inspección
real de Docker y parámetros PostgreSQL. Puede releer el sello antes y después.
El lector de contraste no recibe un DSN ni el nombre de un contenedor elegidos
por el usuario.

Los componentes extraídos incluyen SHA256 y tamaño del archivo original cuando
la raíz es un archivo regular. Los directorios requieren contraste de su árbol;
no tienen una huella de contenido ficticia. El tipo `material` admite un TAR interior
validado con los mismos límites acumulados. `RutaMaterial` apunta al árbol extraído
y `RutaInterna` conserva el archivo original para contrastar sus bytes.

El ejecutor archivado permite arrancar un único proceso por ID de componente,
consultarlo por HTTP o HTTPS en loopback y detenerlo. Las sondas mantienen TLS y
mTLS, no siguen redirecciones, no usan proxy ni cookies y acotan el cuerpo.
Sólo entregan hasta 16 KiB del cuerpo al observador, en memoria, para comprobar
la respuesta autorizada; el resultado final no debe incluirlo.

El auxiliar es el propio binario de la CLI, copiado a la raíz exclusiva y montado
de sólo lectura. Cualquier CLI que componga estos puertos debe llamar a
`ManejarModoInterno` antes de parsear sus argumentos. No se descarga una sonda ni
se ejecuta un script del host.

La configuración y las credenciales sintéticas de VEC deben proceder del perfil
archivado compatible, con rutas de conexión sustituidas por el clon y despachos
bloqueados. El observador debe probar salud y consulta autorizada reales. Si no
puede hacerlo, devuelve `no_comprobable`; una salida de proceso no basta.

## Comprobación acreditada en este corte

La prueba real usa una fuente PostgreSQL 18.4 creada para el ensayo, la detiene,
recupera sus datos físicos y comprueba una fila. También recupera ejecutables
archivados y arranca un testigo HTTPS sintético: exige certificado cliente,
rechaza la consulta sin autorización y admite la consulta autorizada. Comprueba
que no sigue una redirección y que el proceso se detiene. El testigo prueba el
mecanismo; no es VEC ni acredita un permiso funcional de VEC.

Las pruebas del ensayo lógico comprueban que el callback consulta la base
recuperada antes de la limpieza y que un fallo del callback retira sus recursos.
Ambos ensayos mantienen `habilita_restauracion=false`.

```sh
VEC_CS06F_IMAGEN_SHA256=DIGEST_LOCAL \
VEC_CS06L_IMAGEN_SHA256=DIGEST_LOCAL \
go test -race ./internal/modules/administracion/adapters/ensayofisicopg \
  ./internal/modules/administracion/adapters/ensayologicopg \
  ./cmd/vec-copias-ensayar-fisica
```

Sin esas variables, las pruebas de PostgreSQL real se omiten de forma explícita.
Este corte no ejecuta el kit del hito 6, no toca cidonia ni instala SQL real.

## Puente de template0 y NSS

`NuevoRuntimePlantillas` sólo admite un entorno creado por la restauración física
real. Comprueba el anclaje privado del runtime, vuelve a leer el TAR y compara su
SHA256 con el contexto que la aplicación revalida después de CS03. Ese contexto
conserva operación, conjunto, ventana, manifiesto, artefacto e imagen. El adaptador
no crea otra firma ni convierte esos datos en una autorización administrativa.

El nombre del clon se genera dentro del puente. La operación técnica sólo admite
`template0` y ese nombre: no activa conexiones en la base original. La sesión de
CREATE, marca de propiedad y DROP cambia temporalmente su propia lectura; la
configuración global sigue en lectura. La retirada compara OID, propietario y
marca, conserva bases preexistentes y rechaza una identidad sustituida o renombrada.
Una respuesta perdida tras CREATE se concilia con el mismo nombre y retira el clon
propio antes de devolver el fallo. La memoria y el registro privados de este puente
no sustituyen al catálogo durable CS07 ni permiten recuperar una operación productiva.

Ambos ensayos montan passwd/group sintéticos de sólo lectura para el UID/GID sin
privilegios que ejecuta Docker. El directorio personal es el tmpfs `/tmp` del ensayo.
No se lee ni se monta passwd/group del host y no se cambia el usuario a root.

La prueba usa PostgreSQL 18.4 físico real: lee el clon, retira sólo su recurso,
conserva una base preexistente y confirma los metadatos originales de `template0`.
La pérdida de respuesta se inyecta después de un CREATE real confirmado, no mediante
un doble que dé por hecha la creación. El testigo HTTPS consulta NSS dentro del
contenedor; sigue sin ser un arranque de VEC ni una prueba de autenticación CS03.
