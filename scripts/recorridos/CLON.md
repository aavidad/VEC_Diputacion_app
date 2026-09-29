# Clon local para comprobar recorridos

Este guion reconstruye una copia aislada con datos sintéticos. Usa el archivo del
hito 1, instala únicamente las 34 SQL posteriores de este corte y construye el
binario del mismo commit de `main`. El hito 5 configura la política de ofertas;
no contiene migraciones. No contacta con la principal.

El corte actual es `ff6493cfccb2da4e83c94fa7c59be24c025cb7c9`. La revisión
anterior, `7f1ecea2fd9f8912d255a80e74da84c69e46b978`, tenía 33 SQL; la
actualización añade CT147 y conserva sus recibos. Para comprobar otro commit
hay que revisar la lista y las huellas de `sql_main.txt` primero.
Cambiar el binario sin comprobar su esquema no prepara otro clon válido.

Necesita Docker, las imágenes locales `postgres:18.4` y `alpine:3.22`, Python,
OpenSSL, `certutil`, Chrome del sistema, Playwright y el compilador indicado por
`go.mod`, con las dependencias descargadas. La compilación usa los 32 núcleos y
la caché compartida `/dev/shm/go-build`. PostgreSQL usa un volumen temporal en
`/dev/shm`, montado con `-v`; el contenedor lleva `--rm` y solo publica en
`127.0.0.1`.

El archivo `estado-cidonia-20260929-hito1.tgz` y el material sintético del hito 1
deben estar fuera de Git, en el directorio privado de estado `vec-clon`. El
guion conserva los certificados de RRHH e Intervención y las claves que
protegen la historia. Añade solo las identidades que faltan. No copia material
de una persona real ni concede permisos a partir de una petición del navegador.

El archivo del hito 1 no incluye las cuentas de Usuarios que indican sus JSON
privados. La preparación las concilia con la autoridad de identidad del clon,
manteniendo certificado, sujeto, persona y perfil. La cuenta canónica puede
tener una referencia nueva; no se presenta como recuperación de una fila que
el archivo no contiene. Las asignaciones existentes se cotejan y una revocación
detiene ese perfil.

## Preparar

Desde un checkout que contenga estos guiones:

```bash
git fetch origin
export VEC_RECORRIDOS_REFERENCIA=ff6493cfccb2da4e83c94fa7c59be24c025cb7c9
export VEC_RECORRIDOS_ESTADO="$HOME/.local/state/vec-recorridos"
export VEC_RECORRIDOS_CONTENEDOR=vec-recorridos-local
export VEC_RECORRIDOS_PUERTO_PG=55531
export VEC_RECORRIDOS_PUERTO_WEB=18531
bash scripts/recorridos/preparar_clon.sh plan
bash scripts/recorridos/preparar_clon.sh preparar
```

Elija otro nombre y otros puertos si están ocupados. El guion rechaza un
contenedor ajeno. El directorio privado tiene permisos `0700`; las claves,
configuración y registros se guardan con permisos `0600`.

El registro de SQL queda en la propia copia, dentro de cada transacción, y en
`sql-journal.json`. Si se pierde el JSON, se recupera desde ese registro. Repetir
el guion no reaplica una migración ni ejecuta `DOWN`. Una fuente o una huella
distinta detienen el montaje.

Para actualizar el clon anterior a este corte, detenga primero la aplicación
con `parar`, cambie `VEC_RECORRIDOS_REFERENCIA` y vuelva a ejecutar `preparar`.
La fuente anterior debe ser antecesora de la nueva y el instalador debe conocer
la ampliación. El volumen, la historia y el material privado se conservan.

## Comprobar y recuperar

```bash
bash scripts/recorridos/preparar_clon.sh estado
bash scripts/recorridos/preparar_clon.sh reiniciar
```

Coordine el reinicio con quienes estén usando el clon. Cada recorrido conserva
su petición y sus recibos fuera de Git. Tras un fallo de escritura, inspeccione
el resultado antes de repetir: una respuesta de error puede haber dejado la
operación registrada.

`runtime-manifest.json` liga el binario a su commit y su SHA256.
`material-manifest.json` identifica los perfiles y sus condiciones pendientes.
Preparar certificados no acredita que un perfil esté autorizado. Solo un
recorrido ejecutado acredita sus respuestas y recibos; tampoco acredita firma
legal, envío corporativo ni uso en producción.

Las capturas y los planes se guardan fuera de cualquier repositorio. Los guiones
usan Playwright con `/usr/bin/google-chrome`, a 1440 y 390 px. La CA sintética
debe estar confiada tanto por Chrome como por el transporte de Playwright;
no se desactiva la comprobación TLS.

`READY.json` solo aparece después de comprobar binario, SQL, material y escucha
HTTPS. Incluye las condiciones que siguen pendientes. La preparación crea un
almacén de confianza privado en `chrome-home`, sin modificar el del usuario.
Al invocar un guion, use ese directorio como `HOME` únicamente para su proceso
y `NODE_EXTRA_CA_CERTS` con la CA que indica el registro. Los planes siguen
fuera de Git. Una denegación observada no equivale a un proceso terminado.

## Retirar lo propio

```bash
bash scripts/recorridos/preparar_clon.sh parar
bash scripts/recorridos/preparar_clon.sh retirar
```

`parar` conserva el volumen para continuar más tarde; `retirar` elimina el
volumen propio tras cotejar su registro. No borre capturas o recibos que otra
persona esté revisando. El clon compartido se conserva solo mientras tenga
recorridos activos; se reconstruye con el mismo guion cuando haga falta.
