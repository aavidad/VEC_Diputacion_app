# Copia local de VEC para ensayar y recorrer

El guion `deploy/principal/copia_local.sh` restaura la base de referencia,
aplica las listas SQL indicadas y arranca el portal. Por defecto genera
certificados nuevos; la opción `--rrhh-original` conserva la identidad de RRHH.
Tarea F0-28 del tablero. Sin selectores arranca con las funciones opcionales
apagadas. `--modulos` prepara conexiones nuevas de Contratación y Bolsa y un
buzón local; `--portal-externo` añade la preparación del aspirante desde el
material privado de copia. El gobierno de administradores sigue pendiente.
Un portal que responde 200 acredita el arranque; el recorrido requiere además
las consultas y actuaciones de cada módulo.

Reglas que no cambian:

- Todo es local. Nada de esta receta toca cidonia; solo dirección sube a la
  principal, y únicamente lo ensayado entero en una copia.
- Los datos son ficticios. En tu copia, si un dato, una huella o una fila te
  estorba más de quince minutos, bórralo o recréalo por SQL y sigue (orden de
  Alberto del 14/10). Eso vale para la copia, nunca para el código ni para el
  SQL que va a Git, que no rebajan ninguna guarda.
- Claves, certificados, conexiones, contraseñas y recibos se quedan en el
  material privado de copia (fuera de Git), con modo `0700` en carpetas y `0600`
  en ficheros. En este texto se nombran solo por su función.
- No ejecutes tal cual el lanzador, el JSON o el SQL privado de otro equipo:
  cópialo y cambia puertos, rutas, nombre de contenedor e identificadores. No
  llames a servicios ni a bases de otro equipo.

Lo marcado *(por verificar)* no está acreditado en el canal ni en los
documentos; compruébalo antes de darlo por bueno y corrige esta receta.

## 1. Qué se necesita y límites de memoria

- Un PostgreSQL 18 propio en contenedor (`postgres:18.4`, que es la imagen de
  los ensayos), un `vec-server` compilado desde tu rama y el Chrome del sistema
  manejado con Playwright.
- El material privado de copia: el respaldo de la principal (volcado), las ACL de la base y de los tipos, el kit de los dos
  administradores de copia y los clientes y configuraciones privadas de cada
  módulo. Pídeselo a dirección si no lo tienes.
- Memoria: el sistema conserva siempre 32 GB. Antes de restaurar o compilar,
  `free -g` debe dar 34 GB disponibles o más (32 para el sistema y 2 para la copia); si no, no restaures ni compiles,
  borra copias y temporales tuyos que ya no uses y avisa en el canal. No toques
  copias ajenas.
- Todo el trabajo va en memoria, bajo `/dev/shm/vec`:
  - carpeta de trabajo: `/dev/shm/vec/wt/<rama>`;
  - datos de PostgreSQL: `/dev/shm/vec/pg/<letra>-<asunto>`;
  - `GOCACHE=/dev/shm/vec/go-build` y `TMPDIR=/dev/shm/vec/tmp/<letra>`.
  Lo que está en memoria se pierde al apagar el equipo: commit y push a menudo.
- Una sola copia viva por equipo y una sola compilación a la vez.
- El contenedor de PostgreSQL se lanza siempre con `--rm --restart=no
  --memory 2g`, los datos montados desde `/dev/shm/vec/pg/...` y escuchando
  solo en `127.0.0.1`. `--rm` borra el contenedor al pararlo, pero no los
  datos, que viven en la carpeta montada.
- Al terminar la pieza, borra tu copia, tu carpeta y tus temporales.

`git worktree move` falla al pasar de disco a `/dev/shm` (cambia de
dispositivo). Haz commit y push, copia la carpeta y ejecuta
`git worktree repair`, comprobando que la cabeza y el árbol coinciden.

## 2. Crear y usar una copia con el guion

Requiere Docker, Python 3, OpenSSL, la imagen local `postgres:18.4`,
la herramienta Go declarada en `go.mod` y sus dependencias ya descargadas.
No descarga herramientas ni imágenes para compilar. La variable
`VEC_COPIA_ORIGEN` apunta a la carpeta privada de referencia, solo de lectura.
Debe contener el volcado `principal-hz37-20261014.dump` y sus anexos
`acl-base-20261014.sql` y `acl-tipos-20261014.sql`.

Para montar Contratación y Bolsa, añade `--modulos --puerto-smtp 11081` y
ten instalada también la imagen `axllent/mailpit:v1.27.8`. El buzón escucha
solo en el puerto local elegido, exige STARTTLS y solo acepta direcciones
`@example.test`; no se configura reenvío. Se comprueba el rechazo de otro
dominio antes de arrancar la aplicación. `parar`, `reiniciar` y `borrar`
incluyen el buzón de esa instancia.

La preparación de los datos ficticios usa los productores del kit privado:
`VEC_COPIA_ROLES_SCRIPT`, antes de instalar SQL, y `VEC_COPIA_DATOS_SCRIPT`,
después de crear las credenciales. Para el aspirante se añade
`--portal-externo`, que requiere `--modulos`, usa el preparador versionado en
`portal_copia/`. No necesita un guion externo del equipo. Añade
`--sin-inscripcion` para preparar solo las lecturas y la clasificación; ese
modo nunca publica el selector de inscripción.
Las dos variables de roles y datos apuntan a archivos privados propios, fuera de Git y de la
carpeta de PostgreSQL, sin enlaces ni permisos de grupo o de otros usuarios.
Esos dos productores reciben `--raiz <instancia>` y dejan su registro
en ella. El preparador del portal recibe además el candado que mantiene la
fábrica. Los datos concretos y el material privado de cada copia no se versionan.

El productor previo debe preparar los dos LOGIN que figuran en
`configuracion_origen_consumos_v1` para los orígenes de Contratación y Bolsa:
grupos exactos, membresía exclusiva, `INHERIT TRUE`, `SET FALSE`, sin
administración y con vigencia de 90 días. AD278 comprueba esos datos; no se
debilita su guarda. Las demás conexiones se crean con nombres y contraseñas
nuevos. El registro privado conserva sus valores y la huella del entorno.

La copia prepara también `VEC_BOLSA_RELEVO_NO_INCORPORACION_DATABASE_URL`
con un LOGIN distinto, TLS verificado y pertenencia exclusiva al grupo del
relevo: `INHERIT TRUE, SET TRUE, ADMIN FALSE`, según el apartado de relevo de
`03_entorno.md`. La conexión no activa el seguimiento ni la incorporación:
esos selectores y los catálogos se preparan para el recorrido que los use.

Con `--modulos` la copia enciende además la cancelación del expediente
(`VEC_CT_CANCELACION_ENABLED`, tarea F1-96). Usa el catálogo de reglas de CT
`reglas.v3.json` (regla c20) y el de motivos de cancelación de `data/demo/reglas`;
si el kit de datos ya fijó alguno de los dos, se conserva el suyo. Sus
migraciones (CT 000122 y AD3-87) ya están en el volcado de referencia. Sin
ellas la ficha recibía 404 al consultar la cancelación y no mostraba el apartado.

El segundo RRHH que exige la regla c22 necesita su identidad y perfil
publicados por el circuito de Administración; esta conexión no los concede.

`VEC_CALENDARIOS_DATABASE_URL` usa otro LOGIN, solo con el grupo
`vec_calendarios_lector`. Bolsa consulta el calendario oficial para saber si
una llamada cae en día hábil y para calcular el plazo de las ofertas. Sin esa
conexión, registrar una llamada y ver la lista de contactos respondían 503.

Desde F1-99, `crear --modulos` sin `--portal-externo` prepara además los datos
del caso 4.0 para los pasos 9 y 10 (`copia_local_caso40.py`):

- aplica los tres guiones de `candidata_hz39/datos` (puesto 9/01 del centro
  520, Antonio disponible en la bolsa vigente y cargo de Manuel);
- crea con `preparar_candidato_desarrollo.sh --candidato-ref` una identidad
  nueva de Antonio ligada a su candidatura vigente. La persona del material
  original tiene el permiso de Mi Bolsa ligado a la candidatura de la bolsa
  sustituida y recibía 403. El permiso nuevo lo publica la aplicación al
  arrancar. El certificado queda en `material/mtls/candidato.crt` y se activa
  `VEC_BOLSA_PORTAL_CANDIDATO_ENABLED` junto con `VEC_BOLSA_REGLAS_SOURCE_PATH`
  (`data/catalogos/bolsa/reglas.v7.json`), el catálogo que el servidor exige con
  ese selector. Si el kit ya fijó alguna de las dos, se respeta su valor;
- registra por la API de Bolsa, como la técnica de RRHH, el correo
  `antonio.reyes@example.test` y un teléfono ficticio. El buzón de la copia
  solo acepta ese dominio, así que el aviso del llamamiento llega a él y no
  sale fuera.

Los nombres de las participaciones no se tocan: el nombre que muestra la lista
de la bolsa es el del acta. Lo revisa F1-100.

Tras el productor de datos, el guion coteja los 24 LOGIN internos: opciones
sin privilegios de administración y membresías directas exactas. La conexión
de importación CONVOCA conserva los grupos `vec_bolsa_importacion_convoca_ejecutor`
y `vec_bolsa_importacion_convoca_recuperador`, ambos con `INHERIT TRUE`,
`SET TRUE` y `ADMIN FALSE`, como exige el contrato H1 de
`scripts/recorridos/clon_bolsa_material.py` para importar y recuperar la operación.
Las demás conexiones de peticiones tienen un solo grupo. El gobierno de
arranque necesita seis arcos concretos, definidos en `copia_local_verificacion.py`,
y no admite otros.
Los propietarios y el proyector conservan las opciones exactas de `SET` e
`INHERIT`; no se trasladan a los LOGIN de peticiones. H retirará esta
excepción al completar F0-08 con el contrato nominal de R.

El lector de motivos conserva caducidad nula porque así lo exige su
comprobación instalada. Es una excepción de datos de esta copia; su TLS y
los demás LOGIN nuevos duran 90 días. Se ha comunicado a dirección para
retirarla al adaptar ese contrato. El cotejo rechaza cualquier otra caducidad
nula o vigencia fuera de 60 a 90 días.

El productor del aspirante se ejecuta con la aplicación y el sellado de
auditoría activos. Al terminar se reinician PostgreSQL y la aplicación para
cargar su configuración. La preparación debe publicar identidad, contexto,
asignación y confianza mediante los productores existentes. Las admisiones
técnicas y los datos ficticios de copia no acreditan actos de administración.

Desde la raíz de tu rama:

```sh
deploy/principal/copia_local.sh crear --equipo l --instancia recorrido \
  --puerto-pg 55481 --puerto-app 18481 \
  --lista-sql deploy/principal/candidata_hz39/lista_sql.txt \
  --lista-sql deploy/principal/candidata_hz39/lista_sql_firma_vigencias.txt
```

Elige tu letra, una instancia nueva y dos puertos libres. Se puede repetir
`--lista-sql` para formar un único orden causal; las rutas y las huellas
opcionales se validan antes de restaurar. Se conservan los bytes validados
para aplicar y registrar exactamente la misma SQL. Una SQL repetida o una huella
incorrecta rechazan la lista entera. Solo admite migraciones `.up.sql` bajo
`deploy/postgresql`. Solo admite SQL añadida en la cabeza versionada después del corte de
referencia `44218865a`. Rechaza conservadoramente cualquier SQL que ya
existía en ese corte, aunque quedara sin instalar; esos huecos requieren
un corte específico del módulo, no repetir una receta histórica. Revisa
el orden del apartado 3. La orden `crear`
rechaza un destino existente. Una restauración fallida se conserva para
examinar sus registros; se borra y se crea otra, sin reaplicar SQL.

El resultado vive en `/dev/shm/vec/pg/<equipo>-<instancia>`, con padre `0700`.
El volcado y sus anexos aportan nombres de propietarios y beneficiarios para
crear roles `NOLOGIN`. El guion recupera también los grupos declarados como
`NOLOGIN` en los ficheros `roles*_up.sql` del repositorio cuando aparecen citados en el
esquema o en la lista SQL validada. Deja que cada migración de esa lista cree
los roles que declara por sí misma. La creación comprueba si el rol ya existe;
no añade membresías ni permisos. El guion no abre ni restaura el fichero de roles
globales, sus contraseñas ni sus membresías. La restauración conserva las ACL
de los objetos; después aplica las ACL de la base y los tipos. No convierte
esos roles en identidades operativas de la aplicación.

F0-31: antes de las migraciones, el guion crea también
`vec_externo_documentos_bolsa_datos` y `vec_interno_documentos_bolsa_rrhh`
como `NOLOGIN`, aunque no aparezcan en el respaldo. B119 necesita ambos
nombres. No se añaden contraseñas, membresías ni permisos; la preparación
documental posterior sigue siendo necesaria para usarlos desde el portal.
Si el respaldo ya cita uno de esos nombres, se crea una sola vez.

El generador común crea CA, TLS y certificados de RRHH e Intervención de
90 días. `VEC_DESARROLLO_VIGENCIA_DIAS` permite de 60 a 90 días al invocarlo
por separado; el guion de copia usa 90. Un material existente se verifica,
sin renovar sus certificados ni cambiar sus claves. PostgreSQL usa TLS con
esa CA nueva. La aplicación se inicia con un entorno explícito, sin heredar
conexiones ni selectores de otra copia.

### Copia con la identidad original de RRHH

Para recorrer una asignación histórica ligada al certificado de RRHH, añade
`--rrhh-original <carpeta_material>` a `crear --modulos`. La carpeta indicada
debe contener el material histórico de desarrollo correspondiente al respaldo
HZ37. El guion selecciona las piezas internas de CA, certificados, identidades,
KMS, TSA e idempotencia. Exige también `identidad/bolsa-bback.json` y coteja su
certificado y sujeto con la identidad de RRHH. Incluye, cuando existen, las
identidades de centros, consultas de RRHH, solicitante y ratificador con sus
certificados, claves y paquetes importables, y los dos catálogos públicos. Si
`identidad/centros.json` referencia a `centro-solicitante` y
`centro-ratificador`, ambas identidades y sus certificados se copian y cotejan
antes de publicar el material.
Comprueba que cada pieza seleccionada y sus carpetas pertenecen al usuario,
son privadas y no tienen enlaces. La fuente se abre solo para lectura y el
conjunto seleccionado se publica en un destino nuevo.

```sh
deploy/principal/copia_local.sh crear --equipo l --instancia rrhh-original \
  --puerto-pg 55481 --puerto-app 18481 --puerto-smtp 11081 --modulos \
  --rrhh-original "$VEC_COPIA_MATERIAL_RRHH" \
  --lista-sql deploy/principal/candidata_hz39/lista_sql.txt
```

La CA, el certificado y la clave de cliente, su paquete PKCS#12, la identidad,
las claves KMS y TSA, B-BACK y las demás piezas seleccionadas conservan
exactamente sus bytes. El certificado histórico de cliente puede tener una
vigencia de 397 días:
no se renueva porque la asignación existente está ligada a su huella. Solo el
certificado y la clave del servidor se generan de nuevo por 90 días, firmados
por la CA original. La huella del servidor en el manifiesto de la copia se
calcula a partir del certificado nuevo. PostgreSQL y el portal usan ese servidor.
Las conexiones a PostgreSQL se crean para la copia con `sslmode=verify-full` y
la CA original; nunca se activan las conexiones del material fuente.

El guion deriva `desarrollo.env` en el destino con rutas de la copia. No abre
ni traslada scripts antiguos, respaldos, ficheros `.dsn`, otros `.env` ni
descriptores de conexión de módulos. Si una pieza seleccionada contiene un DSN,
la creación se detiene antes de publicar material. Los productores privados
preparan las conexiones y configuraciones opcionales para esta instancia.
`estado.json` registra `material_modo: rrhh_original`,
que permite a los productores conservar la raíz histórica de autorización.
El perfil de permisos de esta receta requiere la publicación vigente sin carga
de CONVOCA ni B103; no se restablece una raíz de autorización nueva.

### Segunda persona técnica de RRHH para no incorporación

La opción `--segundo-rrhh` se usa solo al crear una copia nueva con `--modulos`.
Prepara en el material privado una segunda persona ficticia, Antonio Ruiz
Lozano, con certificado de cliente propio firmado por la CA de la copia. Su
certificado dura como máximo 90 días; la asignación sintética dura 89. La
identidad, la cuenta y el perfil tienen referencias distintas de las de Carmen.
El perfil original de RRHH conserva su certificado y sus permisos.

```sh
deploy/principal/copia_local.sh crear --equipo l --instancia segundo-rrhh \
  --puerto-pg 55481 --puerto-app 18481 --puerto-smtp 11081 --modulos \
  --rrhh-original "$VEC_COPIA_MATERIAL_RRHH" --segundo-rrhh \
  --lista-sql deploy/principal/lista_sql_main_tras_hz37.txt
```

Usa los productores privados de roles y datos del apartado 2 y elige puertos
libres de tu equipo. El respaldo debe contener la versión publicada
`rol:seguimiento_cese_ct_desarrollo:v2`, su control de vigencia habilitado y
las tablas de asignaciones de autorización. La semilla exige la concesión de
no incorporación de ese rol y que el perfil nuevo no tenga otra asignación.
Si falta una condición, `crear` se detiene y conserva la instancia incompleta
para examinar sus registros; bórrala con la orden propia antes de repetir.

El guion guarda `rrhh-adicional-semilla.json` y su huella en `estado.json`,
registra la asignación sintética en una transacción y activa los catálogos de
seguimiento de Contratación. Esa asignación solo sirve en la copia ficticia:
no acredita publicación por Administración. Es gemela de la de Carmen: mismo
emisor de desarrollo y mismo acto, y cada operación la ciñe al expediente
concreto antes de decidir, igual que con ella. El servidor comprueba la hoja, el
sujeto, el perfil, la asignación vigente y la ruta de no incorporación. Si falta
material o permiso para esa persona, conserva el acceso de la técnica original
y deniega la operación de la segunda. No se instala ninguna migración nueva
por esta opción. Se combina con `--portal-externo` si hace falta.

Lo que falta para recorrer la no incorporación en la web con dos personas:
la segunda hoja solo alcanza el POST de
`/api/vec/contratacion-temporal/no-incorporaciones`. Para confirmar desde la
pantalla, Antonio Ruiz Lozano necesita antes abrir el seguimiento y la ficha
del expediente con la propuesta y su versión, y esa lectura acotada a su
persona llega con la tarea F1-47. Hasta entonces el recorrido web completo del
segundo técnico queda pendiente. No se envía el POST a mano para suplir la
pantalla en ningún recorrido ni en ninguna demostración. Con su certificado, las lecturas
de Contratación, de Bolsa y de reglas y el resto de operaciones responden 403;
solo responden lo común a cualquier persona identificada (sesión, menú,
módulos y catálogos públicos de Personal). F1-47 ampliará a propósito lo que puede leer
(seguimiento y ficha del expediente) cuando llegue.

En el ensayo técnico de esta opción basta comprobar tres cosas: que `crear`
termina con la semilla registrada y su huella en `estado.json`, que el
servidor arranca con la segunda hoja (sin el aviso «segundo técnico de RRHH
sin …» en el registro) y que todo sigue igual tras `reiniciar`. Un GET de
seguimiento o un POST de cese con la segunda hoja deben denegarse. Conserva
los cuerpos y respuestas de esta copia fuera de Git.

Conserva los mismos argumentos de equipo e instancia para estas órdenes:

```sh
deploy/principal/copia_local.sh comprobar --equipo l --instancia recorrido
deploy/principal/copia_local.sh reiniciar --equipo l --instancia recorrido
deploy/principal/copia_local.sh parar --equipo l --instancia recorrido
deploy/principal/copia_local.sh arrancar --equipo l --instancia recorrido
deploy/principal/copia_local.sh borrar --equipo l --instancia recorrido
```

`reiniciar` detiene aplicación y PostgreSQL y los vuelve a crear con los
mismos datos. Comprueba `/livez` y `/portal-empleado/` por HTTPS con CA y
certificado de cliente verificados. Con `--modulos`, comprueba además la
consulta del cuadro de Contratación y el listado de bolsas. La disponibilidad
global de otras funciones opcionales no se deduce de estas comprobaciones.
`arrancar` rechaza una copia incompleta o
una revisión de código distinta; crea otra instancia cuando cambie la rama.
La compilación exige índice y árbol limpios antes y después de generar el
binario. Su SHA-256 queda en el estado; cada arranque lo coteja y exige la
misma fuente limpia, también para los archivos web que se sirven desde ella.
`borrar` comprueba el árbol de trabajo creador, la marca de propiedad y la
etiqueta del contenedor antes
de detener y eliminar únicamente su instancia. No actúa sobre puertos o
contenedores encontrados por coincidencia.

Los registros privados de restauración, SQL, compilación y aplicación y el
diario `sql_aplicado.json` quedan en la instancia. `estado.json` conserva la
revisión del código y los segundos empleados hasta obtener los dos 200.
En el primer ensayo local sin SQL adicional se midieron 72,17 segundos;
con B117 y comprobación de permisos, 64,95 segundos. En ambos se
repitieron los dos 200 después de reiniciar ambos procesos. Falta la prueba
de otro equipo y la preparación automática de los módulos para cerrar F0-28.

### Regenerar el clon de ensayo SQL

La copia antigua de septiembre carece de AD251. Para preparar su reemplazo,
crea una instancia nueva desde el volcado de octubre, con las listas ya
instaladas que deba tener tu base de ensayo. Después:

```sh
deploy/principal/copia_local.sh exportar-clon --equipo l --instancia recorrido \
  --archivo-clon "$VEC_COPIA_CLON_DESTINO"
```

`VEC_COPIA_CLON_DESTINO` es una ruta absoluta nueva, fuera de Git y de la
instancia. La exportación publica el archivo solo al terminar correctamente. Detiene
ambos procesos y empaqueta el directorio
de PostgreSQL en frío; deja la copia parada. El archivo contiene datos y
material privado: no se sube a Git. La extracción también necesita un
contenedor con 2 GiB; comprueba su resultado antes de intentar arrancarlo. Se puede usar como `VEC_CLON_ESTADO` con
el ensayo local que ya acepta esa variable. No se sustituye su clon
compartido ni se cambia el valor predeterminado del ensayo. Se restaura cada
archivo de ensayo en una carpeta propia y se aplica una sola vez la lista
de SQL nueva; para repetir se vuelve a extraer el clon frío.

## 3. Poner la copia al día con las SQL de `main`

Usa esta lista: `deploy/principal/lista_sql_main_tras_hz37.txt`. Trae una
sola vez, con su huella, cada migración añadida a `main` después de
`44218865a`, en el orden en que se fusionaron sus PR y, dentro de cada PR, en
el orden de su propia lista. Se pasa sola, con un único `--lista-sql`; no la
combines con las listas antiguas de cada PR, que repiten migraciones y harían
que el guion rechazara todo. Toda PR que añada migraciones añade sus líneas
al final de esta lista en la misma PR, con la ruta y su SHA-256.

Los dos roles sin LOGIN que necesita B119 los crea ya el guion (F0-31, apartado 2);
no hace falta crearlos a mano.

AD278 (F1-16) necesita además dos usuarios de origen con LOGIN que el guion
todavía no prepara: `vec_ct_o207_runtime` (viene en el volcado sin LOGIN),
miembro de `vec_contratacion_temporal_ejecutor`, y
`vec_bolsa_llamamientos_desarrollo` (no viene en el volcado), miembro de
`vec_bolsa_llamamientos_ejecutor`. Hasta que el guion lo haga, prepáralos en
cuanto existan los roles del volcado y antes de que lleguen las migraciones:
`ALTER ROLE vec_ct_o207_runtime LOGIN`, `CREATE ROLE vec_bolsa_llamamientos_desarrollo LOGIN`
y los dos `GRANT` de pertenencia, sin contraseña.

Lo que sigue explica de dónde sale la lista y cómo se hacía a mano hasta el
18/10. Sirve para comprobarla, no para aplicar otra cosa.

La copia tiene lo que tiene cidonia. Le faltan las migraciones fusionadas
después de `44218865a`. Para saber cuáles son:

```sh
git diff --name-only --diff-filter=A 44218865a origin/main -- \
  'deploy/principal/lista_sql_*' 'deploy/principal/*/lista_sql*'
git diff --name-only --diff-filter=A 44218865a origin/main -- \
  'deploy/postgresql/**/*.up.sql'
```

La primera orden da las listas nuevas; la segunda, todas las migraciones
nuevas. Coteja las dos: una migración que no aparezca en ninguna lista también
falta. A 15/10 son 26 migraciones, que se aplican en este orden:

| Orden | Lista | Qué trae |
| --- | --- | --- |
| 1 | `candidata_hz39/lista_sql.txt` | Las once de la candidata (confianza externa, cargos y plan de firma, organización de la firma, informe firmado, detalle RRHH). No trae las cinco de la firma: van en la lista 1b |
| 1b | `candidata_hz39/lista_sql_firma_vigencias.txt` | CT218, IS20, AUT97, CT219 y CT224 (firma de tres pasos, #1137). Las mismas que `lista_sql_codexh_firma_vigencias_20261014.txt`: aplica solo una de las dos, una vez |
| 2 | ninguna | AD266, secuencia global de confianza ADMIN (#1134). La necesita el gobierno de usuarios ADMIN (apartado 4). P la aplicó después de las once primeras *(por verificar si debe ir antes de CT218)* |
| 3 | `candidata_hz39/personal/lista_sql.txt` | Ocho de Personal y Bolsa nativos (alta y cese desde la bolsa). Exige antes B110 y AD261, que deberían estar en HZ37 *(por verificar)* |
| 4 | `lista_sql_codexs_varias_plazas_20261015.txt` | B117, ofertas con plazo desde la publicación (#1145) |

Corrección del 16/10 (T): la lista 1 tiene once SQL; las cinco de la firma
están solo en la lista 1b y se aplican aparte, sin duplicarlas.
`lista_sql_codexh_firma_tres_pasos_20261013.txt`,
`lista_sql_codexk_personal_bolsa_nativa_20261013.txt` y
`lista_sql_codexp_confianza_externa_20261014.txt` son subconjuntos de las
listas 1 y 3.

Cómo aplicarlas:

- Cada fichero, una vez, con `psql -X -v ON_ERROR_STOP=1`. Comprueba antes su
  SHA-256 cuando la lista lo trae.
- Nunca ejecutes un `DOWN` ni reapliques una migración ya instalada. Si una
  falla porque ya estaba, consérvala y sigue con la siguiente.
- Las SQL de PR sin fusionar (por ejemplo AD282 de #1151) solo se aplican
  cuando ensayas esa PR.

Datos del recorrido, después de las SQL: `lista_sql_codexk_datos_recorrido_20261014.txt`
(centro 520, puesto 9/01, plaza 2026/0033, bolsa de Administrativo, con
`-v organismo=<organismo_ref>`), luego `candidata_hz39/personal/datos_plantilla.sql`
y, para la firma, `candidata_hz39/firma/datos_empleados_firmantes.sql`. Las
condiciones de cada uno están en su `LEEME.md`.

## 4. Administradores de la copia

Los dos administradores de cidonia no salen de cidonia y sus claves no se dan
a ningún equipo. Cada copia usa dos administradores ficticios propios.

1. Copia del material privado el kit de administración de copia y el
   «fixture ADMIN» (`fixture-admin-hz37.sql.in`). El fixture hace que las
   vinculaciones de la base coincidan con ese material: desactiva los
   disparadores de inmutabilidad y borra fuentes, vínculos, outbox, arranque y
   control previo. Solo se ejecuta en tu copia (dirección lo autorizó el 14/10
   a las 22:35). Guarda antes una preimagen de Administración.
2. Sigue `candidata_hz39/admin_copia/LEEME.md`: `preparar_copia.py` y después
   `arranque_principal_h10.py` por etapas (fuentes, unidad, arranque y
   verificación), con los planes y aprobaciones de tu copia. No copies recibos,
   LOGIN, puertos ni SQL con identificadores de otro equipo.
3. `preparar_vec_admin.py` crea los LOGIN y además inserta ocho filas técnicas
   de ejecución. No te saltes esa parte.
4. Con `cmd/vec-gobierno-usuarios-admin` y el conjunto 5 (requiere AD266)
   publica la confianza de gobierno y genera la fase `version-bolsa`.
5. Arranca `vec-admin` con la configuración de usuarios
   (`VEC_ADMIN_USUARIOS_CONFIG_FILE`), que monta su frontera técnica con el
   grupo `vec_admin_frontera_tecnica_ejecutor` de AD189. No uses el compositor
   heredado de `vec-admin` (el que arranca sin esa configuración) y no crees el
   grupo `vec_admin_perfiles_auditoria_ejecutor`: llama a una función que no
   existe (consenso de seguridad del 15/10; arreglo en F0-02).

Renovación diaria. Hasta que entre F0-03 (#1151, vida configurable hasta 90
días en desarrollo), las claves y la confianza de gobierno ADMIN duran 24
horas. Cuando venzan, renovar no es alargar: genera audiencias y claves nuevas
con la vida actual por la vía normal del kit (`preparar`, aprobación,
`aplicar`, `verificar`) y haz la propuesta y el cierre con los dos
administradores dentro de esa ventana. No toques las claves históricas.
La ventana de un día del plan técnico de admisión (AUT61) y la del plan de
publicación de cargos se mantienen: son el plazo para aplicar un acto, no la
vigencia de lo publicado.

Perfiles y asignaciones de la copia:

- Las asignaciones ficticias de perfiles de Contratación temporal y sus
  ámbitos pueden ir como dato de semilla por SQL. Los cargos, los certificados
  y el plan de firma van por actos reales con los dos administradores.
- Contratación temporal fija la unidad `unidad:desarrollo:rrhh`. El plan, los
  cargos y las asignaciones deben cubrir esa unidad; una unidad nueva de la
  copia no basta.
- No insertes asignaciones con dimensiones que ningún consumidor valida (por
  ejemplo centro y categoría): quedarían sin acotar. Acota por organización y
  unidad gestora mediante el lote ordinario. Lo que de verdad exija centro y
  categoría queda pendiente de Administración (F0-06).
- Una asignación retirada no se revive: se otorga una nueva.

## 5. Identidad del aspirante externo

La orden `crear --modulos --portal-externo` usa los productores existentes
para generar identidad, contexto, asignación y confianza del aspirante.
Las admisiones técnicas ausentes del respaldo se generan como **semilla
ficticia de esa copia**. No acreditan un acto de Administración ni F0-21.
El material y los registros quedan bajo la carpeta privada `portal` de la
instancia, y `borrar` los retira junto con ella.

```sh
deploy/principal/copia_local.sh crear --equipo b --instancia aspirante \
  --puerto-pg 55481 --puerto-app 18481 --puerto-smtp 11081 \
  --modulos --portal-externo --sin-inscripcion \
  --rrhh-original "$VEC_COPIA_ORIGEN/copia-20261014/material" \
  --lista-sql deploy/principal/lista_sql_main_tras_hz37.txt
```

Usa los productores de roles y datos del apartado 2 y puertos libres propios.
El preparador conserva el candado de la instancia y coteja contenedor, puerto,
identificador PostgreSQL y binario antes de operar. Guarda el documento y la
referencia de asignación antes de confirmar; si se pierde la respuesta,
reconcilia el mismo efecto por su huella y evento, sin publicar otro.
La reanudación exige la misma revisión y el árbol limpio. La actualización
del entorno conserva un intento privado antes de cambiar sus archivos: si
se interrumpe entre el entorno y su huella, el mismo preparador completa esa
publicación. Una modificación distinta detiene la recuperación.

Una cabeza del rol candidato que exija Administración detiene la semilla con
`requiere_publicacion_por_administracion`; no recupera una versión antigua.
En ese caso corresponde publicar el permiso por la vía gobernada. La sesión
y su auditoría no necesitan AD295; el gobierno ADMIN sí mantiene sus requisitos.
Las lecturas de Mi Bolsa y del historial no acreditan el envío de inscripción:
ese POST y su recibo deben comprobarse en el recorrido que lo consume.

## 6. Organización histórica y lectura de Personal en el servidor combinado

La selección desde la bolsa constituida (#1088) consulta el puesto en la
Organización histórica de Personal. Hasta el 15/10 ese GET solo se montaba en
`vec-interno`; el `vec-server` combinado publicaba la clave pero no servía la
consulta, y la selección fallaba.

- #1150 (pendiente de fusión a 15/10) monta esa consulta en el combinado, solo
  en desarrollo; en producción manda la cápsula de `vec-interno`. Se activa
  con `VEC_ORGANIZACION_HISTORICA_CONSULTA_ENABLED=true` y
  `VEC_ORGANIZACION_HISTORICA_MATERIAL_DIR`, además de
  `VEC_ORGANIZACION_HISTORICA_GOBIERNO_ENABLED=true` para publicar la capacidad.
  Los pasos (perfil con sus seis campos, motivos, material y cuatro conexiones)
  están en el guion `deploy/principal/organizacion_historica_combinado_20261015.md`
  de esa PR. Las fechas de los motivos llevan seis decimales.
- La lectura de Personal para Contratación (M10) sigue
  `candidata_hz39/personal/LEEME.md`. Copia del material privado el cliente
  `cliente_m10_ct.json` y cambia su origen al puerto de tu propio servicio; el
  que trae apunta al de otro equipo. `VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH`
  apunta a esa fuente.
- Unidad gestora y unidad del puesto: el puesto 9/01 de la semilla es de
  `unidad:centro:520` y la unidad gestora de Contratación es la de RRHH. Hasta
  que entre F1-21 (#1153), la selección rechaza ese puesto. No cambies el dato
  para esconderlo.
- Necesidad de la RPT: la solicitud 2026/94011 de la copia no tiene necesidad
  guardada y está en acumulación de tareas, que no admite puesto ni huella. En
  tu copia puedes usar la variante vacante del caso 4.0 con dos personas
  solicitadas, como dato de copia y sin atribuirla a la fiscalización anterior.
- «Llamar a la siguiente persona» desde la ficha del expediente funciona en
  una copia nueva creada con `--modulos` (sin `--portal-externo`) y con la
  firma compuesta (F1-103, 20/10). La selección responde 200 con su recibo y
  llama a la primera persona disponible por el orden vigente; repetirla
  devuelve el mismo llamamiento. Antes daba 409 por tres cosas: faltaban dos
  permisos de la técnica de RRHH (la consulta de Organización histórica y la
  lectura de la bolsa constituida desde Contratación,
  `bolsa.origen_ct.consultar`) y la lectura auditada del origen fallaba en
  cuanto alguien de la bolsa tenía una comprobación de ofrecimiento
  registrada. Esto último lo corrige AD302, que viene en la lista única.
- `crear --modulos` siembra y monta la consulta de Organización histórica.
  Con `--portal-externo` no lo hace, igual que los datos del caso 4.0:
  1. Siembra el rol `consulta_organizacion_historica_rrhh` v1 con la concesión
     de `fuentes/personal_organizacion_historica_rrhh_v1.json` (sustancial,
     seis campos, sin obligaciones). Lo asigna a un perfil nuevo de la técnica
     del certificado `mtls/cliente.crt`, con su vínculo de contexto y su
     vínculo corporativo `consulta_rrhh`, gemelos de los de su perfil de
     Bolsa. Los ámbitos son exactos: el organismo del caso 4.0 y
     `unidad:centro:520`; otra unidad responde 403. Es semilla de copia con
     procedencia `identidad:desarrollo:no-autoritativa`, como la de F0-37, y
     dura 89 días. Su huella queda en `estado.json`
     (`organizacion_historica_semilla_sha256`).
  2. Crea cuatro LOGIN propios (Personal, fuente y registro de autorización y
     motivos), la fila de origen del consumo de Personal (AD172), el catálogo
     de cuatro motivos, el cliente `cliente_m10_ct.json` con el puerto de la
     copia y `VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH`.
  3. Con la aplicación ya arrancada, deriva la clave con
     `vec-preparar-material-interno` (con un LOGIN de gobierno que solo vive
     mientras dura ese paso), escribe `organizacion_historica_combinado.json`,
     enciende la consulta y reinicia.
- Los dos permisos se completan con dos etapas privadas del manifiesto de
  `firma` (§11), con la herramienta de actos de la copia:
  1. Entre la etapa de Administración y la de firmantes, AUT58 admite en el
     catálogo de acciones `fuentes/personal_organizacion_historica_rrhh_v1.json`
     y `fuentes/bolsa_origen_ct_rrhh_v2.json`. La v2 sustituye a la v1, a la
     que le faltan los dos campos que hoy lee la selección
     (`criterios_ofrecimiento` y `entradas[].eventos_relacion`).
  2. Al final, una versión nueva del rol RRHH de Bolsa con
     `bolsa.origen_ct.consultar`, por la vía de
     [`habilitacion_rol_bolsa_rrhh_generico_20261013.md`](habilitacion_rol_bolsa_rrhh_generico_20261013.md):
     el DBA de la copia crea dos LOGIN y sus dos filas de origen AD172,
     `vec-gobierno-usuarios-admin -fase version-bolsa` escribe el overlay
     B103, `vec-admin` se reinicia con `VEC_ADMIN_VERSION_BOLSA_B103_CONFIG_FILE`,
     el primer administrador propone y el segundo cierra. El servidor copia
     la concesión del catálogo; el cliente solo elige la entrada.
  Sin la primera etapa la consulta de Organización histórica responde 403;
  sin la segunda, la selección da 409.
- El orden importa. La admisión AUT58 tiene que ir antes de que la firma
  publique los roles de cargo: esos roles se guardan sin `retirada_en` y,
  una vez publicados, el censo de la admisión no cuadra y la autoridad la
  rechaza. Es un fallo aparte, anotado en F1-103 del tablero.
- La consulta guarda el perfil y su versión por cuenta. Si cambia la versión
  del perfil, recrea la copia: el material preparado ya no sirve.
- Los expedientes 2026/93001 y 2026/94011 no guardan necesidad de la RPT y su
  historia no se puede corregir: las versiones son inmutables. Para llamar
  desde la ficha, recorre el caso 4.0 con la variante vacante (puesto 9,
  plaza 2026/0033) hasta la fiscalización, después de componer la firma: un
  expediente preparado antes se queda en el paso 7b.
- Quedan aparte el acto de Administración que publique la consulta de
  Organización histórica en producción (AUT60, conjunto 7) y la salida
  definitiva de F0-37.

## 7. Arrancar `vec-server`

Base común (`docs/producto/TRD.md`, apartado 4):

- `VEC_HTTP_ADDR` en `127.0.0.1`, TLS con `VEC_TLS_CERT_FILE` y
  `VEC_TLS_KEY_FILE`, e identidad por certificado.
- Doble llave de desarrollo: `VEC_EXECUTION_PROFILE`, `VEC_AUTH_MODE` y
  `VEC_DEVELOPMENT_GUARD`. Sin ella no se componen Bolsa ni Contratación.
- Conexiones nominales de tu copia, con TLS verificado y un LOGIN por
  consumidor. Nunca la conexión de superusuario ni la de otro módulo.
- `config/selectores_dependencias_bolsa_ct_v1.json` lista lo que pide cada
  selector. Al arrancar, el registro avisa de cada dependencia que falta.

Orden de arranque:

1. Primero sin ninguna configuración opcional. El lanzador de S para HZ37
   apaga firma, Documentos, incorporación y ajustes avanzados. Deben responder
   `/livez`, el portal, la sesión y Bolsa.
2. Después enciende una configuración cada vez y reinicia vigilando el
   arranque. Con ella ausente o inválida, la aplicación arranca y solo esa
   función responde 503 o 404.

| Configuración opcional | Dónde está la receta |
| --- | --- |
| Firma de cinco pasos y verificador | `candidata_hz39/firma/LEEME.md` (`identidad/firma-vec.json`, `VEC_FIRMA_VERIFICACION_*`) |
| Personal, alta y cese desde la bolsa | `candidata_hz39/personal/LEEME.md` |
| Organización histórica en el combinado | Guion de #1150 (apartado 6) |
| Ofertas con plazo desde la publicación | `activacion_codexs_ofertas_20261015.md` (`VEC_BOLSA_REGLAS_SOURCE_PATH`) |
| Auditoría de RRHH | `habilitacion_auditoria_rrhh_20261013.md` (tarea F0-01) |

Un fichero que el proceso lee al arrancar no se cambia con la aplicación en
marcha: para, cambia y arranca. Si algo falla, vuelve al binario y la
configuración anteriores. El TRD avisa de que no todos los montajes cumplen
todavía la regla de cerrar solo su función: si un componente tumba el
arranque, anótalo como fallo del producto.

## 8. Comprobar en Chrome

- Chrome del sistema con Playwright, un certificado por perfil.
- Direcciones: `/portal-empleado/` (RRHH e Intervención),
  `/portal-empleado/peticiones-centro/` (centro y ratificador) y
  `/area-personal/` (aspirante).
- El recorrido es el caso 4.0 de `docs/producto/FLUJO_APP.md`. Para el tramo
  de Personal, el orden es 17, 18, 16a, 16b, 19, 20 y 21.
- Por cada paso: respuesta HTTP, recibo y captura a 1440 y 390 píxeles. Sin
  errores de JavaScript, sin cookies ni almacenamiento web y sin
  desbordamiento horizontal.
- Un recibo local no confirma por sí solo el efecto en otro módulo: comprueba
  también la pantalla del consumidor (Mi Bolsa, ficha de Personal).

## 9. Reinicio y recuperación

La composición de Administración añade `--administracion` a `crear --modulos`.
El puerto se elige con `--puerto-admin` y debe estar libre y ser distinto de
los puertos de PostgreSQL, la aplicación y el correo.
Requiere `VEC_COPIA_ADMIN_SCRIPT`, un ejecutor privado preparado para la misma
candidata. Recibe `--raiz`, `--fase preparar|arrancar|comprobar`,
`--candado-fd` y `--puerto-admin`: debe heredar ese descriptor al llamar al kit y conservar sus
comprobaciones de revisión, huellas, etiqueta y sistema PostgreSQL.
La preparación se invoca una sola vez; un resultado incierto se concilia con
el mismo plan y recibo. El reinicio solo arranca y comprueba lo ya preparado.
La parada exige que PID, ejecutable propio e inicio del proceso coincidan.
Si el proceso sigue vivo pero su binario cambió, se detiene con error y
conserva la copia: hay que pararlo a mano antes de `borrar`.

Si se interrumpe la preparación, conserva la instancia y su carpeta `admin`.
No repitas `crear` ni la fase `preparar`. Mantén PostgreSQL y la aplicación
con su sellador activos; Administración debe estar parada. La continuación
nativa del mismo paquete se documenta en `LEEME.continuacion-ad295.md`, del
kit privado (no está en el repositorio):
`continuar_admin_runtime_ad295.py --repo … --kit … --equipo … --instancia …
--sid … --config-gobierno … --acuse-gobierno … --salida-verificacion …
--comprobar` coteja por lectura el mismo gobierno. Después, el comando sin
`--comprobar`, con otra salida nueva, continúa únicamente el runtime que falta.
Usa la revisión, el identificador PostgreSQL y las rutas de esa copia; los
informes nuevos quedan junto al acuse original. Un fallo de cotejo detiene
la continuación y conserva los archivos para diagnóstico.

Probado el 20/10 (F0-28) con un ejecutor privado que reutiliza el kit H6
(conjunto 6) fijado a la revisión de la copia. En `preparar` compila el kit
desde la revisión limpia, prepara Administración y genera el montaje externo
(`preparar_overlay_externo.py`, también del kit privado, y la fase
`version-bolsa-externa` de `vec-gobierno-usuarios-admin`). En `arrancar` lanza `vec-admin` con
`VEC_ADMIN_VERSION_BOLSA_EXTERNA_CONFIG_FILE` y guarda PID e inicio del
proceso. El ejecutor y el kit viven en `~/.local/state`, nunca en Git.
El catálogo `catalogo:bolsa:carga_convoca:b1` v6 de la línea base trae la
fuente de inscripción v1. Desde F1-101 se admite antes la v2 en un catálogo v7
(guía en `candidata_hz39/aspirante/inscripcion_fuente_v2.md`) y el plan del
aspirante toma de ahí sus cinco acciones.

Orden del acto del aspirante, con el Chrome del sistema y un perfil por
certificado (`admin/h10/fuentes/cliente-1` y `cliente-2`):

1. Cada administrador abre `https://admin.localhost:PUERTO/administracion-perfiles/`
   y elige «Administración de la aplicación». La elección sobrevive al reinicio.
2. Leer catálogo, cabeza del rol `candidato_bolsa_portal_historial_propio_desarrollo`,
   su control y la asignación activa, y preparar con el dominio el plan de
   las cinco acciones de inscripción (`huella_plan_esperada`).
3. El administrador 1 envía la propuesta a
   `/api/admin/perfiles/v1/gobierno-version-bolsa-externa/propuestas` (201;
   la repetición da 200 con la misma referencia).
4. El administrador 2 la cierra en `…/cierres` (200, rol en v2 con doce
   concesiones y la asignación movida a v2; la repetición da el mismo recibo).

No hay pantalla para este acto: las dos peticiones salen del propio Chrome
del administrador con su certificado. Playwright verifica el servidor con la
CA de la copia (`NODE_EXTRA_CA_CERTS`) y Chrome con la huella SPKI de su
certificado; no se desactiva la verificación.

Después del cierre, el aspirante lee `/api/vec/bolsa/inscripciones/propias`
con 200. Con las entradas de la v1 (catálogo v6), la lista de convocatorias
abiertas y su ficha responden 403: no traen los campos que la aplicación pide
desde F1-40 y F1-41 (`modalidad_inscripcion` y `solicitudes_propias[]`), y la
autorización exige que coincidan. Con la fuente v2 admitida (F1-101) las dos
responden 200.

El suplemento del segundo RRHH queda aparcado por dirección: el ámbito
antiguo de Contratación necesita una tarea propia para usar organización y
unidad gobernadas.

El preparador del portal externo recibe también `--candado-fd` de la fábrica
y debe conservarlo hasta acabar. La invocación independiente del preparador
adquiere el candado de esa misma instancia antes de comprobar su contenedor.
Se necesita la receta actualizada; una receta antigua que solo acepte
`--raiz` no satisface este contrato.

Usa `reiniciar` del apartado 2 para la copia creada por el guion. La orden
conserva PostgreSQL y su material, y repite las comprobaciones HTTPS.
Después de preparar los módulos, el ensayo del recorrido debe recuperar
sus lecturas y recibos con las mismas claves de operación: mismas fechas y
versiones, sin duplicados. Los dos 200 de arranque no sustituyen esa prueba.

Si se apagó el equipo, `/dev/shm` está vacío: crea otra instancia con el
mismo volcado y las listas necesarias.

## 10. Problemas conocidos

| Síntoma | Causa | Arreglo |
| --- | --- | --- |
| La aplicación se detiene en la comprobación previa de preferencias o de Contratación | Faltan las ACL de la base tras restaurar | Aplicar el fichero de ACL de base del material privado (apartado 2) |
| Errores de permisos sobre tipos de fila | Faltan las ACL de los tipos | Aplicarlas tras restaurar |
| Los administradores del kit no coinciden con ninguna vinculación de la copia | Falta el fixture ADMIN | Aplicar el fixture en tu copia (apartado 4) |
| Propuesta o cierre de gobierno rechazados por claves vencidas | Vida de 24 h hasta F0-03 | Generar claves y audiencias nuevas con el kit y aplicarlas en la ventana |
| `vec-admin` pide `vec_admin_perfiles_auditoria_ejecutor` | Se está usando el compositor heredado | Arrancar con `VEC_ADMIN_USUARIOS_CONFIG_FILE`; no crear el grupo |
| Plan técnico de admisión (AUT61) caducado | Ventana de un día para aplicar | Prepararlo de nuevo y aplicarlo el mismo día |
| Mi Bolsa responde 503 o «instantánea obsoleta» | Admisiones de identidad externa vacías en HZ37 | Restituirlas como dato de copia (apartado 5) |
| La selección desde la bolsa constituida da 404, 409 o 503 | Organización histórica sin montar en el combinado o sin material | #1150 y su material (apartado 6) |
| La selección rechaza el puesto 9/01 | Unidad gestora frente a unidad del puesto | Esperar a F1-21 (#1153); no tocar el dato |
| La selección exige una necesidad de la RPT | La solicitud de la copia no la guardó | Variante vacante del caso 4.0 como dato de copia |
| `crear --modulos` sin `--portal-externo` se detiene en `selectores_bolsa_ct` | El selector del portal del candidato iba sin el catálogo de reglas de Bolsa | Arreglado en F1-109: la copia aporta `VEC_BOLSA_REGLAS_SOURCE_PATH` (apartado 2) |
| Mi Bolsa responde 403 a Antonio | Su permiso estaba ligado a la candidatura de la bolsa sustituida | Arreglado en F1-99: la copia crea su identidad ligada a la candidatura vigente (apartado 2) |
| Registrar una llamada o ver los contactos de una participación da 503 | Faltaba la conexión del calendario oficial | Arreglado en F1-99: `VEC_CALENDARIOS_DATABASE_URL` (apartado 2) |
| El aviso del llamamiento no llega al buzón | La participación no tenía datos de contacto | Arreglado en F1-99: la copia registra un correo `@example.test` (apartado 2) |
| El cliente de Personal (M10) llama a un puerto ajeno | `cliente_m10_ct.json` copiado sin cambiar | Apuntarlo a tu servicio |
| La consulta de firmas da 503 con «autenticación histórica inválida» | Falta IS20 | Aplicar la lista 1b |
| El contenedor de la app no arranca al montar `/app` en solo lectura y `/app/var` en escritura | Falta la carpeta `var/` en tu carpeta de trabajo | Crea `var/` en tu carpeta de trabajo antes de montar (lo vio T el 16/10) |
| Rechazo de una fecha que es el mismo instante | Representación distinta: AUT32 compara texto con el canon nominal; CA25 y los motivos de Organización histórica exigen seis decimales | Generar las fechas en el formato canónico de cada guarda; corregir el dato de la copia |
| Los datos de empleados firmantes piden 400 días de relación cubierta | Las asignaciones de la copia duran 90 días | Ajustar en la copia las relaciones a las vigencias existentes (P usó 89 días por SQL) |
| Firma del paso 3 rechazada por unidad | CT224 solo admite el paso 3 con unidad y paso publicados en el plan vigente | Publicar el plan con esa unidad |
| Con la inscripción encendida, el arranque que sigue a `--portal-externo` se detiene en `inscripcion_rrhh_cuentas_lectores` | La copia no declaraba a la técnica de RRHH en `identidad/consultas-rrhh.json` | Arreglado en F1-90: el preparador escribe el manifiesto. Una copia anterior se crea de nuevo |
| `--portal-externo` se detiene en `fuente_privada_divergente` | El material de origen ya traía un `identidad/consultas-rrhh.json` distinto (otros lectores u otro formato) | Crear la copia con un material sin ese fichero; el preparador no mezcla manifiestos |
| El aspirante abre la inscripción y recibe 403 | Su rol aún no tiene las acciones `bolsa.inscripcion.*` | Actos de Administración de `inscripcion_recorrido_20261013/gobierno_inscripcion.md` (F0-28, F1-29) |
| Tras el acto, las convocatorias abiertas siguen en 403 y las propias dan 200 | El rol se publicó con la fuente v1 (catálogo v6), sin `modalidad_inscripcion` ni `solicitudes_propias[]`, y la autorización exige los mismos campos | Admitir antes la fuente v2 y publicar el rol desde ella (F1-101, apartado 9); en una copia que ya publicó la v1, crear otra copia. No tocar la concesión por SQL |
| La segunda firma del informe da 422 `firma_no_verificada` | GrxFirma 0.0.105 duplica `/Version` al cofirmar y su primera firma bloquea los campos | Firmar las dos veces con GrxFirma 0.0.106 o posterior, en un expediente sin firmas de la 0.0.105 (apartado 11) |
| GrxFirma rechaza el origen de la copia con 403 | Su configuración no tiene permisos 600 o la confianza por decisión está apagada | `chmod 600` a sus ficheros y activar la confianza por decisión con el origen admitido |
| Chrome anota `net::ERR_NETWORK_CHANGED` y la aplicación, un 503 de una petición que el navegador ya cortó | Otro contenedor Docker del equipo arranca o se borra y Chrome corta lo que tiene en curso | Nada que arreglar en la copia. El registro de una firma se repite solo con la misma clave (F1-94); una lectura se repite con «Comprobar» |
| `exportar_admisiones` no devuelve firmantes | La asignación del firmante acaba después de que se retire su certificado | Hacer que la asignación termine en la retirada del certificado (apartado 11) |
| `free -g` por debajo de 34 GB | Copias o cachés acumuladas | No restaurar ni compilar; borrar lo propio y avisar |
| `git worktree move` falla hacia `/dev/shm` | Cambio de dispositivo | Copiar la carpeta y `git worktree repair` |

## 11. Firma compuesta (orden `firma`)

Para que el paso 7b del caso 4.0 funcione, la copia necesita el plan de firma
publicado, los firmantes con certificado y asignación, sus admisiones en
`identidad/firma-vec.json`, la v4 de plantillas, el circuito RRHH v2 y un
verificador GrxFirma propio. Resultado y evidencia en
[`firma_copia_f1_32_20261019.md`](../../docs/estudio_requisitos/firma_copia_f1_32_20261019.md).

Con la copia creada con `--modulos` (si alguna etapa privada necesita la aplicación, arráncala antes):

```sh
python3 deploy/principal/copia_local.py firma --equipo d --instancia <nombre> \
  --manifiesto-firma /ruta/privada/firma.json
```

El manifiesto es privado (`0600`, fuera de Git) y solo admite estas claves:

```json
{
  "version": 1,
  "etapas": [{"nombre": "admin", "orden": ["/ruta/privada/etapa-admin"]}],
  "verificador": {"binario": "/ruta/privada/grxfirma-verificador",
                  "binario_sha256": "<sha256>", "puerto": 63174}
}
```

1. Las etapas son el camino largo de `candidata_hz39/firma/LEEME.md`:
   Administración y sus cuatro montajes de efectos, firmantes, AUT53,
   asignaciones, CA45, empleados, los 14 actos, el plan y `firma-vec.json`.
   Cada etapa es un ejecutable privado (`0700`). Se ejecutan en orden, sin el
   entorno del operador, con `VEC_COPIA_RAIZ`, `VEC_COPIA_NOMBRE`,
   `VEC_COPIA_PUERTO_PG` y `VEC_COPIA_PUERTO_APP`. Su salida queda en
   `firma-etapas/<nombre>.log`. Una etapa terminada no se repite mientras su
   orden no cambie, y si una falla las siguientes no se ejecutan.
2. Publica la v4 de plantillas (F1-86) con `publicar_informe.sh` sobre TLS
   verificado. Si ya estaba publicada, el recibo dice `replay`.
3. Copia el circuito RRHH v2, el catálogo v4 y la política sintética de
   subsanación de reparos de esta revisión a
   `material/identidad/firma-recursos/` y comprueba que `firma-vec.json`
   apunta al plan y a la CRL de esa carpeta y que no está retirado.
4. Instala el verificador en `<copia>/verificador` (binario con su huella, CA
   y TLS de la copia, CRL de clientes y su credencial, que se crea la primera
   vez y se conserva después) y añade al
   entorno de módulos diez variables: las de firma
   (`VEC_CT_FIRMA_VEC_R5_ENABLED`, `VEC_CT_CIRCUITO_RRHH_SOURCE_PATH`,
   `VEC_CT_PLANTILLAS_SOURCE_PATH` y las cinco `VEC_FIRMA_VERIFICACION_*`),
   `VEC_CT_SUBSANACION_POLITICA_FILE` para registrar la subsanación de un
   reparo y `VEC_CT_PLANTILLAS_GOBIERNO_ENABLED` para abrir la ruta
   `#contratacion-temporal/plantillas-rrhh`.
   El circuito RRHH v2 solo monta la consulta que usa la firma: los
   expedientes nuevos siguen naciendo con el flujo de ocho fases y
   pueden pasar el análisis (F1-98).
5. Reinicia la copia. Desde entonces `arrancar` levanta el verificador antes
   que la aplicación (con la CRL vigente de `firma-recursos`) y `parar` (y
   `borrar`) lo para. Solo se para el proceso
   propio: mismo ejecutable y mismo instante de arranque.

Después, aprueba la provisión de los perfiles fijos de RRHH
(`firma_documento`, `lector_entrega_peticion`) como en la presentación y
comprueba que el preflight ofrece `certificado_vec` a la persona del paso.

En copias nuevas (F1-94) el camino entero funciona con `firma` sin pasos a
mano, si las etapas privadas cumplen estas condiciones: las asignaciones de
los firmantes terminan cuando se retira su certificado; la correspondencia
CA45 se publica antes del plan; la verificación del gobierno de
Administración se reintenta hasta que esté disponible; y el kit de
Administración usa la revisión de la copia nueva. Detalle en
[`firma_jefatura_f1_94_20261020.md`](../../docs/estudio_requisitos/firma_jefatura_f1_94_20261020.md).

**Aviso: GrxFirma debe ser la 0.0.106 o posterior.** Es el lanzador de
`afirma://` para firmar en Chrome. Con la 0.0.105 (la instalada en
`/usr/bin`) la segunda firma del informe sale 422, y un informe que el
técnico firmó con ella ya no admite la firma de Jefatura. Cada
persona usa su propia carpeta personal, con `afirma://` permitido en Chrome
para el origen de la copia y la configuración de GrxFirma en `0600`, la
confianza por decisión activada y el origen admitido.
