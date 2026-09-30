# Clon local para comprobar recorridos

Este guion reconstruye una copia aislada con datos sintéticos. Usa el archivo del
hito 1, instala únicamente las SQL posteriores del plan revisado y construye el
binario del mismo commit de `main`. Ejecuta el portal interno de forma explícita.
El hito 5 configura la política de ofertas;
no contiene migraciones. No contacta con la principal.

El plan revisado de `a7d9df2b3285b0df6be6bba0bae09331463f0a3d` contiene
38 SQL. Conserva las 33 de `7f1ecea2`, CT147, AD3-114 y CT148; añade
AD3-113 y Documentos9. El orden sigue las dependencias, no el número de migración.
Los recibos anteriores conservan sus posiciones y fechas.

El plan de `1e443463df69dffeaac239f9b7000f48dd1b7bb7` añade una única SQL:
Aspirantes000002, después de Aspirantes000001. Son 39 instalaciones. Al pasar
de la copia anterior a esta fuente, solo se instala esa ampliación. Una fuente
anterior conserva su plan de 38; no se le atribuyen instalaciones posteriores.

El plan de `890b3fe0e9f9e30e249b9dc2d3778971121a8cc2` contiene 43 entradas.
Dirección ha retenido su primera instalación de RPT para corregir la fuente.
El guion solo permite recuperar esa versión si sus 43 recibos exactos ya están
en la copia; no la instala sobre H1 ni sobre un prefijo incompleto. El último
punto reconstruible aprobado es `3be3110e6d4a8389aed5a496c0b6d1bd805b7723`,
con 39 instalaciones. La fuente corregida requiere una revisión y una copia fría
distinta; la versión retenida conserva su historia, sin DOWN ni reaplicación.

El plan H6 aprobado para `5694d2da15e19fa97afecae51e1a30ce21d5fca5`
contiene 41 instalaciones: conserva las primeras 39 y añade CT150 y CT151.
Usa `sql_main_h6.txt`, con huella
`95c3feff3cbd5b95cf0286af74c576d2c551d92b3cb7787b337754d3aeed87fb`.
Las cuatro entradas de RPT retenidas se cotejan como parte del inventario de
la fuente, pero quedan fuera de este plan de ejecución. El binario congelado
de H6 procede de `ab875bb8036af59e9b5ac624d6840b8581178ed2`: conserva
exactamente ese inventario SQL. Ambos hashes se registran por separado.

H6 se prepara desde una copia fría nueva de H1 o continúa un prefijo de ese
plan hasta 39. La copia que ya contiene las 43 instalaciones de RPT se
conserva aparte; no puede convertirse en una copia de 41. No se ejecutan
`DOWN` ni reaplicaciones para cambiar de historia.

Un commit posterior de `main` puede usar este mismo plan si conserva exactamente
el inventario SQL revisado. El guion compara los archivos del commit con la fuente
extraída. Una SQL nueva, modificada o ausente detiene la preparación y exige
revisar el plan. El commit de la aplicación y la referencia del plan quedan
registrados por separado.

Necesita Docker, las imágenes locales `postgres:18.4` y `alpine:3.22`, Python,
OpenSSL, `certutil`, `socat`, Chrome del sistema, Playwright y el compilador indicado por
`go.mod`, con las dependencias descargadas. La compilación usa los 32 núcleos y
la caché compartida `/dev/shm/go-build`. PostgreSQL usa un volumen temporal en
`/dev/shm`, montado con `-v`; el contenedor lleva `--rm` y solo publica en
`127.0.0.1`. El correo de prueba usa la imagen local `axllent/mailpit:v1.27.8`,
una red Docker interna y dos puertos locales. Exige TLS y solo admite destinatarios
del dominio sintético `example.test`.

El archivo `estado-cidonia-20260929-hito1.tgz` y el material sintético del hito 1
deben estar fuera de Git, en el directorio privado de estado `vec-clon`. El
guion conserva los certificados de RRHH e Intervención y las claves que
protegen la historia. Añade solo las identidades que faltan. No copia material
de una persona real ni concede permisos a partir de una petición del navegador.

Los perfiles externos conservan su material fuera del proceso interno. Preparar
sus certificados no habilita sus autoridades. Mientras sus dependencias no estén
en `main`, esos recorridos quedan pendientes; el modo combinado histórico no los
sustituye.

El binario interno se construye sin dependencias dinámicas y se ejecuta en un
contenedor propio. Solo monta su fuente y material interno como lectura, más
cuatro directorios propios de escritura: documentos, imágenes, datos y avisos.
Las claves de cliente y emisión de certificados permanecen fuera de esos
montajes. El usuario del proceso tiene un directorio personal vacío.

El contenedor usa la red del anfitrión para alcanzar PostgreSQL y el correo del
clon. El guion valida esos destinos locales; esta configuración no acredita
aislamiento de red frente a otros servicios del anfitrión.

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
export VEC_RECORRIDOS_REFERENCIA=origin/main
export VEC_RECORRIDOS_ESTADO="$HOME/.local/state/vec-recorridos"
export VEC_RECORRIDOS_CONTENEDOR=vec-recorridos-local
export VEC_RECORRIDOS_PUERTO_PG=55531
export VEC_RECORRIDOS_PUERTO_WEB=18531
export VEC_RECORRIDOS_PUERTO_SMTP=11025
export VEC_RECORRIDOS_PUERTO_CORREO_WEB=18532
bash scripts/recorridos/preparar_clon.sh plan
bash scripts/recorridos/preparar_clon.sh preparar
```

Para repetir el corte H6 congelado, sustituya la referencia por
`ab875bb8036af59e9b5ac624d6840b8581178ed2` y elija un estado privado y un
nombre de contenedor nuevos. El archivo H1 y su material original se
conservan. La preparación no usa la base de 43 instalaciones como origen.

Elija otro nombre y cuatro puertos distintos si están ocupados. El guion rechaza un
contenedor ajeno. El directorio privado tiene permisos `0700`; las claves,
configuración y registros se guardan con permisos `0600`.

El registro de SQL queda en la propia copia, dentro de cada transacción, y en
`sql-journal.json`. Si se pierde el JSON, se recupera desde ese registro. Repetir
el guion no reaplica una migración ni ejecuta `DOWN`. Una fuente o una huella
distinta detienen el montaje. No instala SQL de una PR pendiente.

La preparación acredita los accesos técnicos del archivo del hito 1. Si faltan,
restaura únicamente los permisos nominales comprobados y retira el acceso general
de la base. Conserva la preimagen, comprueba la reversión en una transacción sin
efectos y verifica las conexiones reales con TLS. No concede privilegios para
sortear otra comprobación fallida.

Para actualizar el clon anterior a este corte, detenga primero la aplicación
con `parar`, cambie `VEC_RECORRIDOS_REFERENCIA` y vuelva a ejecutar `preparar`.
La fuente anterior debe ser antecesora de la nueva y el instalador debe conocer
la ampliación. El volumen, la historia y el material privado se conservan.

Si cambia la proyección de la aplicación, añada
`VEC_RECORRIDOS_ROTAR_PROYECCION_INTERNA=true`. Después de comprobar que el
runtime está detenido, el guion archiva la proyección anterior completa, prepara
la nueva y copia sus cuatro directorios de datos. Coteja contenido, permisos y
propiedad antes del arranque. El recibo `rotacion-interna.json` permite continuar
una copia interrumpida; nunca sobrescribe datos nuevos ni vuelve a transferirlos
después de una actualización completada. El archivo anterior no autoriza
arrancar el binario antiguo sobre SQL posteriores.

Si solo cambia el sello de preparación del operador, puede solicitar su
conciliación con `VEC_RECORRIDOS_REFRESCAR_PRUEBA_INTERNA=true`. El guion
comprueba la preimagen y exige iguales fuente, destino, configuración, archivos
y montajes de ejecución. Guarda ambos sellos y rechaza cualquier otro cambio.
La opción no rota claves ni modifica datos o identidades.

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
HTTPS del portal interno. El registro identifica esa superficie y las condiciones
que siguen pendientes; no acredita disponibilidad del portal externo. Un certificado de candidato
puede existir sin una cuenta externa autorizada: ese caso queda bloqueado y no se
reutiliza la identidad de otra persona. La preparación crea un
almacén de confianza privado en `chrome-home`, sin modificar el del usuario.
Los recorridos montan su almacén NSS como lectura mediante `bwrap` y usan
un temporal privado corto para Chrome. El transporte de Playwright usa
`NODE_EXTRA_CA_CERTS` con la CA que indica el registro. Los planes siguen
fuera de Git. Una denegación observada no equivale a un proceso terminado.

El registro distingue el commit del binario de `sql_fuente_aprobada` y conserva
las huellas del plan SQL, del inventario completo y de la configuración.
Su publicación vuelve a cotejar los bytes del binario y exige que la fuente
verificada por el instalador sea la del proceso activo.

## Retirar lo propio

```bash
bash scripts/recorridos/preparar_clon.sh parar
bash scripts/recorridos/preparar_clon.sh retirar
```

`parar` detiene la aplicación, PostgreSQL y el correo propios; conserva el volumen
para continuar más tarde. `retirar` elimina el volumen, las fuentes temporales,
los binarios y los logs del guion, y marca ese estado como retirado. Conserva
material, recibos y capturas para revisión. Para
reconstruir después de retirar, elija un directorio privado nuevo, por ejemplo:

```bash
export VEC_RECORRIDOS_ESTADO="$HOME/.local/state/vec-recorridos-segundo"
bash scripts/recorridos/preparar_clon.sh preparar
```

Así se conservan los recibos y las capturas de la copia anterior sin mezclarlos
con otra base. No borre evidencia que otra persona esté revisando. El clon
compartido se conserva solo mientras tenga recorridos activos.
