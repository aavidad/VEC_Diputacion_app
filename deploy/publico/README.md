# Instalación de la consulta pública

Este paquete prepara `cmd/vec-publico` y su proyección PostgreSQL en una instancia
dedicada. El proceso anónimo recibe un lector de doce vistas públicas y las huellas
externas de la publicación. RRHH conserva su instancia, procesos y permisos.

Una instalación fuera del clon privado requiere la aprobación final del operador.
Los comandos siguientes preparan el material para esa aprobación. El ensayo usa
datos sintéticos y recursos propios; no contacta con cidonia ni con la principal.

## Requisitos

- PostgreSQL 18, cliente `psql` 18, Python 3.11 o posterior y TLS verificado.
- Instancia nueva que contenga solo la base pública y las tres bases de PostgreSQL.
  El esquema `public` debe estar vacío de cualquier objeto, incluidos funciones y
  tipos. Solo se admite la extensión `plpgsql`, sin precargas ni event triggers.
- Un DBA con contraseña SCRAM, una CA y un certificado con el nombre del servidor.
  `pg_hba.conf` exige `hostssl` y `scram-sha-256` para los LOGIN públicos y rechaza
  las conexiones sin TLS. El publicador tiene su propia credencial.
- Dos contraseñas distintas, aleatorias, de 32 a 128 caracteres `A–Z`, `a–z`,
  `0–9`, `_` o `-`, entregadas por ficheros privados. Nunca se ponen en argumentos.
- Binario aprobado de `vec-publico`, binario del publicador existente `vec-server`
  y material de publicación revisado. La salida del publicador fija el ancla V3.

## Preparar la base

Guarde la configuración y los secretos fuera de Git, en un directorio `0700`.
Cada fichero privado debe pertenecer al usuario que ejecuta el instalador, ser
regular, tener un solo enlace y permisos `0600`. Las rutas son absolutas.

El fichero de servicio libpq contiene estas opciones y ninguna más:

```ini
[publico_dba]
host=nombre-del-servidor
port=5432
dbname=vec_bolsa_publica
user=postgres
sslmode=verify-full
sslrootcert=/ruta/privada/ca.crt
require_auth=scram-sha-256
```

El fichero de contraseña DBA sigue el formato de `pgpass`. El JSON privado del
instalador declara los nombres de fichero y la identidad esperada de la instancia:

```json
{
  "service_file": "/ruta/privada/servicio",
  "pass_file": "/ruta/privada/pgpass",
  "service": "publico_dba",
  "database": "vec_bolsa_publica",
  "system_identifier": "IDENTIFICADOR_DE_PG_CONTROL_SYSTEM",
  "reader_password_file": "/ruta/privada/lector",
  "publisher_password_file": "/ruta/privada/publicador"
}
```

Obtenga y contraste `system_identifier` con el inventario de la instancia propia.
El instalador comprueba conexión TLS, versión, identidad e inventario antes de
cambiar ACL. El primer comando hace el preflight; el segundo instala en el clon
autorizado o, tras aprobación, en la instancia externa dedicada:

```bash
python3 deploy/publico/aprovisionar.py --config /ruta/privada/instalacion.json
python3 deploy/publico/aprovisionar.py --config /ruta/privada/instalacion.json --aplicar-instancia-dedicada
```

El instalador verifica las SHA256 fijadas de `roles_up.sql`, `000001` y `000002`.
Reúne sus cuerpos en una sola conexión y transacción, tras comprobar sus límites
exactos y retirar únicamente los `BEGIN` y `COMMIT` exteriores. Un fallo intermedio
revierte roles, objetos, ACL y comentario de la base. No edita los archivos SQL.

En esta instancia vacía retira los privilegios de `PUBLIC` sobre sus cuatro bases.
Crea `vec_publico_login` con una única membresía lectora heredada, sin `SET ROLE`
ni administración. Guarda la versión y la huella de estructura y ACL en el
comentario de la base. La reejecución compara esa huella, incluidos ajustes por
base y rol, y acredita las dos contraseñas con conexiones SCRAM reales. Una
deriva o contraseña diferente se rechaza. La reejecución no rota secretos ni
reaplica migraciones. No hay procedimiento `DOWN` sobre una publicación conservada.

El instalador desactiva el registro y los muestreos de sentencias en su conexión
antes de empezar la transacción que contiene las contraseñas, incluidos los
árboles de depuración `debug_print_parse`, `debug_print_rewritten` y
`debug_print_plan`. Fija `password_encryption=scram-sha-256` dentro de la
transacción, aunque el DBA herede `md5`. Sus errores no
imprimen SQL, DSN ni respuestas de `psql`.

## Publicar y preparar el proceso

La publicación usa el CLI existente, con su configuración privada y la credencial
del publicador. Mantenga el DSN fuera de terminales y registros. La invocación es:

```text
vec-server publicar-proyeccion-publica --proyeccion-v2 PROYECCION.json --manifiesto-v2 MANIFIESTO.json --bolsas-v1 BOLSAS.json
```

Conserve el ancla de su salida junto al material revisado. No la aprenda de la
base pública. La proyección y el bloque B10 se publican juntos por V3; el proceso
anónimo carece de esta credencial. Una nueva publicación necesita ancla nueva y
reinicio con esa configuración, o el despliegue blue/green ya descrito en
[`bolsa_publica/README.md`](../postgresql/bolsa_publica/README.md).

Prepare el artefacto en un directorio nuevo. El empaquetador copia el binario
aprobado, los guiones de arranque y solo los archivos de `web/publico.manifest`; registra sus SHA256 y el
commit de los recursos web y los guiones. Coteja cada recurso y guion con el blob
del commit fijado antes de crear el destino y rechaza cualquier modificación local.
Los bytes que copia proceden de esos blobs. La procedencia del binario debe acompañar al artefacto
aprobado; copiarlo no demuestra que se compiló desde ese commit.

```bash
python3 deploy/publico/empaquetar.py --binary /ruta/aprobada/vec-publico --destination /ruta/nueva/artefacto
```

Para un corte UI separado, `--web-source /ruta/al/worktree-ui --web-commit SHA`
selecciona un commit de recursos distinto del commit de los guiones. También
exige que esos recursos estén limpios. Ambos hashes quedan en `artefacto.json`.

El JSON privado del runtime contiene `binary` (ruta absoluta al binario),
`binary_sha256` y `web_sha256` del artefacto, `environment` y `check`.
`environment` admite exactamente estas variables:

```text
VEC_HTTP_ADDR
VEC_TLS_CERT_FILE
VEC_TLS_KEY_FILE
VEC_BOLSA_PUBLICA_DATABASE_URL
VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256
VEC_BOLSA_CATEGORIES_CATALOG_ID
VEC_BOLSA_CATEGORIES_CATALOG_VERSION
VEC_BOLSA_CATEGORIES_CATALOG_SHA256
VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256
```

El DSN usa `vec_publico_login`, la base dedicada y solo `sslmode=verify-full` y
`sslrootcert` como parámetros. Las cuatro referencias del catálogo y el ancla
proceden de la publicación revisada. `check` contiene `origin` HTTPS, `ca_file`,
`convocatoria` (identificador público existente) y `bolsa` (referencia pública
existente). La clave TLS y el JSON son privados.

```bash
bash deploy/publico/arrancar.sh --config /ruta/privada/runtime.json
bash deploy/publico/comprobar.sh --config /ruta/privada/runtime.json
```

El arranque coteja binario y web, valida TLS y entrega al proceso un entorno nuevo
con las nueve variables públicas, perfil `produccion` y autenticación `disabled`.
El adaptador existente acredita las ACL efectivas, las doce vistas y las huellas
antes de escuchar. La comprobación exige `200` en página, readiness, listas y
detalles públicos, `404` en las rutas internas enumeradas y ausencia de cookies.
No sigue redirecciones ni desactiva la verificación TLS.

La unidad `vec-publico.service` sirve de plantilla para una instalación revisada
en `/opt/vec/publico` y `/etc/vec/publico`, con usuario propio y artefacto de solo
lectura. No crea el usuario, instala archivos ni habilita servicios. Antes de
usarla, adapte las rutas, prepare el JSON `0600` para ese usuario y apruebe el
destino. El proceso público debe poder alcanzar únicamente su proyección y los
canales HTTP admitidos por la configuración de red.

## Ensayo local

Con Docker, OpenSSL y la imagen PostgreSQL 18.4 ya descargada:

```bash
bash deploy/publico/ensayo.sh /ruta/vec-publico /ruta/vec-server
```

Para incluir navegador, prepare un directorio privado de capturas y añada
`VEC_PUBLICO_CAPTURAS=/ruta/privada/capturas` al comando. Requiere Playwright,
`certutil` y `/usr/bin/google-chrome`. Crea un almacén NSS propio con la CA del
ensayo, usa respuestas reales y comprueba 1440 y 390 px. Un recorrido de
convocatorias que no pueda renderizarse hace fallar esa comprobación aunque
las consultas HTTP hayan devuelto `200`.

El autor de un corte UI puede usar `VEC_PUBLICO_WEB_REPO` y
`VEC_PUBLICO_WEB_COMMIT` juntos para ensayar sus recursos ya confirmados con
estos guiones revisados. El runner conserva ambos commits en el artefacto y
crea sus propios servicios y material; no necesita compartir un clon.

El guion crea un contenedor y una red propios, publica PostgreSQL solo en
loopback y elimina sus claves, datos y procesos al terminar. La red Docker del
ensayo permite salida; esta prueba de desarrollo no acredita aislamiento de red
productivo. Los binarios se entregan ya compilados. La imagen se fija por digest.

Prueba rechazo de objetos previos, rollback ante un error en `000002`, instalación,
replay, contraseña incompatible y deriva de ACL y ajustes por base. Comprueba que
las contraseñas no aparecen en el log aun con muestreos y árboles de depuración
activados, y que los dos LOGIN usan SCRAM aunque el DBA herede `md5`.
Publica material sintético por el
CLI real y compara las respuestas tras reiniciar PostgreSQL y aplicación. El
historial debe conservar exactamente una publicación. La huella de respuestas
del escenario inicial es
`eb15756b46af6756986582a7e475443f5621256e031ccce51e39637fdfc6698c`.

Este corte instala la consulta anónima. La fuente de producción, su clasificación,
el circuito del candidato, entrega de avisos y conformidad normativa requieren sus
propias comprobaciones. La unidad systemd es una plantilla, no un servicio
instalado. No se ha ejecutado ninguna instalación externa.

La simultaneidad con el proceso interno no está acreditada. El ensayo adicional
sobre una copia fría de H1 no llegó a `/livez`: su inventario de permisos y tipos
no coincide con la restauración privada disponible. Esta última se rechazó en
`ROLLBACK`, sin confirmar una restauración ni instalar migraciones. Hace falta
una copia interna compatible y revisada para completar esa prueba. Este paquete
no declara H7 operativo ni sustituye las comprobaciones de H6.

El recorrido Chrome original detectó que la página cargaba `contrato-v1.js`
frente a respuestas V2. Tras incorporar `main@7bebfc9a`, el manifiesto público
incluye `contrato-v2.js` y los recursos que utiliza la página actual. El
empaquetador coteja esos archivos con el commit web indicado. Este cambio de
fuentes no acredita por sí solo un nuevo recorrido Chrome de convocatorias.

El runtime admite en la conexión del lector únicamente `sslmode=verify-full`
y un `sslrootcert` absoluto, cada opción una sola vez y con valor. Rechaza
parámetros vacíos o repetidos antes de entregar el DSN a PostgreSQL.

En el recorrido Chrome anterior, la lista B10 y su detalle respondieron `200`
tras el reinicio a 1440 y 390 px, sin errores JavaScript, cookies,
almacenamiento web ni desbordamiento global. El detalle móvil conservó el
desplazamiento dentro de la tabla. Esa evidencia sólo cubre B10 con los
recursos de aquel corte; las convocatorias V2 actuales necesitan otro recorrido.
