# Un proceso para cada portal: diseño y primer corte

Fecha: 29 de septiembre de 2026. Fase 1, paso 1 del estudio
[datos personales de las inscripciones](../estudio_requisitos/datos_personales_inscripciones_rgpd_2026-09-29.md)
(apartados 4.2, 5.2 y 8). Datos siempre sintéticos. No toca cidonia.

## Qué se quiere conseguir

El portal externo (Área personal de aspirantes y consulta pública de bolsas)
estará expuesto a Internet. Si alguien toma el control del proceso que lo
atiende, no debe poder leer datos de trabajadores. Para eso el proceso externo
tiene que arrancar sin las contraseñas de base de datos del interno y sin sus
claves de cifrado, y al revés. Separar solo tablas o esquemas no basta mientras
un único proceso tenga todas las credenciales y todas las claves.

## Cómo está hoy

- En la principal corre un único `vec-server` con la composición de
  desarrollo (doble llave), escuchando en `127.0.0.1:18443` dentro del pod
  `vec-desarrollo-20260906`; el contenedor del túnel lo publica en
  `127.0.0.1:8443`. Ese proceso atiende a la vez el portal de RRHH y del
  empleado, el Área personal, «Mi bolsa» y la consulta pública.
- Credenciales de base de datos que recibe ese proceso:
  - 15 variables `VEC_*_DATABASE_URL` que carga `arrancar_app.sh` desde
    `material/arrancar-local.sh` (el guion exige exactamente 15), más
    `VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL` leída de un fichero aparte.
  - Otras conexiones dentro de ficheros JSON del material:
    `identidad/cronos-empleado.json`, `dietas-comisiones.json`,
    `documentos.json`, `usuarios-preferencias-interna.json` y
    `usuarios-preferencias-externa.json`. Usuarios ya usa usuarios de base de
    datos distintos para cada superficie (`vec_pref508a_i_*` y
    `vec_pref508a_e_*`).
- Material de claves: un único directorio (`/vec-material`) con la clave
  maestra del cifrado (`kms/clave-maestra.bin`), las del sellado de tiempo y la
  idempotencia, el certificado y la clave TLS del servidor, las identidades de
  RRHH, Intervención, centros y candidato, y también la clave privada de la
  autoridad certificadora (`ca/ca.key`) y las claves privadas de los clientes
  (`mtls/*.key`, `*.p12`). El servidor nunca lee estas últimas, pero el
  arranque exigía que estuvieran; con ellas cualquiera que comprometa el
  proceso podría emitir certificados de acceso.
- «Mi bolsa» y el portal del candidato se componen dentro de la composición de
  Contratación temporal y usan sus conexiones de identidad y autorización
  (`VEC_CT_REGISTRO_IDENTIDAD`, `VEC_CT_REVALIDACION_IDENTIDAD`,
  `VEC_CT_CONTEXTO_ACTOR`…) y la de llamamientos de Bolsa.
- `cmd/vec-interno` y `cmd/vec-publico` son raíces antiguas más pequeñas; no se
  despliegan y no tienen Área personal.

## Diseño

### Dos modos del mismo binario

Se elige con `VEC_PORTAL_PROCESO`:

| Valor | Qué atiende | Qué comprueba al arrancar |
| --- | --- | --- |
| vacío | todo, como hasta ahora | nada nuevo; solo impide usar un material ya separado |
| `interno` | RRHH, empleado y consulta pública | que no haya conexiones ni material del externo |
| `externo` | Área personal y consulta pública | que no haya conexiones, secretos ni material del interno |

Por qué un binario con dos modos y no dos binarios: lo que protege es que cada
proceso no tenga credenciales ni claves del otro, no que el código esté
ausente. Un solo artefacto evita duplicar la compilación, el ensayo en el clon
y el despliegue. Si más adelante se quiere que el binario externo ni siquiera
contenga el código interno, bastará con una raíz `cmd/vec-externo` que llame a
la misma composición externa.

### Qué recibe cada proceso

- Conexiones: las del externo llevan siempre el prefijo `VEC_EXTERNO_` y
  usuarios de PostgreSQL propios. En su entorno, el proceso externo rechaza
  cualquier otra conexión, aunque tenga un nombre raro (también reconoce una
  cadena de conexión por su valor). El interno rechaza cualquier
  `VEC_EXTERNO_*`. Las conexiones escritas dentro de los JSON del material no
  las puede juzgar cada proceso solo: las compara la comprobación de
  despliegue, que además exige usuario explícito en todas.
- Ninguno de los dos admite credenciales que pgx lee por su cuenta
  (`PGPASSWORD`, `PGUSER`, `PGPASSFILE`, `PGSERVICE`, `PGSSLKEY`… ni `~/.pgpass` o
  `~/.postgresql/postgresql.key`).
- Material: cada proceso tiene su propio directorio con un fichero
  `portal-proceso.json` (`{"version":1,"portal":"interno"}` o `"externo"`).
  Cada uno tiene su propia clave maestra, sellado de tiempo, idempotencia y
  clave TLS. El externo solo admite su lista positiva (identidad del candidato,
  `usuarios-preferencias-externa.json`, `mtls/candidato.crt`, `externo/…`,
  más TLS, KMS, TSA, idempotencia, `ca/ca.crt`, manifiesto y
  `desarrollo.env`). El interno rechaza esos ficheros del externo. Ninguno
  admite nada bajo `ca/` salvo `ca/ca.crt`, ningún `*.p12` ni `*.password`,
  ningún `*.key` fuera de `tls/` y `kms/`, ni enlaces simbólicos.
- El proceso externo tampoco admite secretos ni custodias del interno aunque
  no sean conexiones: la custodia de CONVOCA, el token del validador de firma,
  el fichero de incorporación de CT o cualquier variable `VEC_*` con aspecto
  de secreto que no esté clasificada.

### Qué rutas sirve cada proceso

- Externo, lista positiva: `/area-personal`, `/bolsa`, `/verificar`,
  `/acceso`, `/api/publico/…`, `/api/vec/bolsa/mi-…`, `/api/vec/bolsa/mis-…`,
  `/api/vec/bolsa/area-personal`, `/api/vec/usuarios/area-personal/…`,
  `/api/vec/usuarios/contacto-propio`, `/api/vec/personas/mi-perfil/…`,
  `/api/vec/personas/mis-preferencias/…`, recursos comunes (`/comun/`,
  `/textos/`, `/locales/`, `/assets/`) y estado (`/livez`, `/readyz`,
  `/healthz`). Todo lo demás, 404.
- Interno: todo menos las rutas propias del Área personal (las de la primera
  mitad de la lista anterior), que dan 404.
- Un valor de portal mal escrito cierra todas las rutas.

### Comprobación al desplegar

Cada proceso por sí solo no puede saber si su clave es la misma que la del
otro. Por eso hay un subcomando para quien despliega, que ve los dos lados:

```text
vec-server comprobar-separacion-portales \
  --material-interno DIR --material-externo DIR \
  [--entorno-interno GUION] [--entorno-externo GUION]
```

Falla si algún secreto (KMS, sellado, idempotencia, clave TLS, `externo/*.bin`)
tiene el mismo contenido en los dos directorios, si un mismo usuario de
PostgreSQL aparece en las conexiones de ambos (variables del guion y cadenas
dentro de los JSON del material) o si alguna conexión no lleva usuario
explícito. Solo imprime recuentos y el nombre del elemento repetido.

## Qué hace este corte y qué falta

Hecho (sin cambiar nada si `VEC_PORTAL_PROCESO` no se configura):

1. `internal/app/separacionportales`: clasificación y comprobaciones de
   entorno, material y separación entre los dos procesos.
2. Arranque: la comprobación va antes de leer material o abrir conexiones. En
   modo separado ya no se exigen las claves privadas de la CA ni de clientes.
3. Rutas: filtro por portal en la superficie integrada.
4. `vec-server comprobar-separacion-portales`. El proceso externo no ejecuta
   ninguna tarea (importar CONVOCA, constituir bolsa…), solo el servidor.
5. El proceso interno ya puede arrancar solo: pasa la separación y sigue la
   composición de siempre.
6. El proceso externo arranca con su propia composición (primer corte): mTLS
   con la identidad de la persona candidata como única identidad, consulta
   pública de convocatorias y categorías, y los ficheros del Área personal.
   No carga identidad de RRHH, clave maestra ni conexiones. Sin persona
   candidata en su material no arranca.
7. «Mis preferencias» del Área personal funciona en el proceso externo:
   - el lado interno ejecuta `vec-server preparar-portal-externo`: con el rol
     de gobierno publica (idempotente) las claves de las audiencias externas y
     deja en el material externo solo esas claves derivadas y la raíz de
     atestación (`externo/v3/`), nunca la clave base ni la maestra;
   - el proceso externo coteja ese material y lee la configuración de
     confianza vigente con su propio login de solo lectura (AD3-112,
     `VEC_EXTERNO_PREFLIGHT_V3_DATABASE_URL`); nunca publica;
   - usa los logins de su configuración de Usuarios
     (`identidad/usuarios-preferencias-externa.json`) y su propia clave de
     idempotencia, con un espacio de seudónimos propio
     (`vec.identidad.desarrollo.externo`). Los alias de sus cuentas los calcula
     él (`vec-server exportar-seudonimos-portal-externo`, sin conexiones) y los
     registra el lado interno (`preparar-portal-externo --seudonimos FICHERO
     --cuentas cta_…`) solo para las cuentas que el operador autoriza, nunca
     para cuentas privilegiadas ni de la superficie corporativa.
   Recorrido real en un clon propio con los dos procesos a la vez: GET 200 y
   PUT 201 de preferencias en el externo, el interno responde 404 a esa ruta y
   los logins externos no pueden leer ningún esquema interno.

Pendiente, en este orden:

1. Capacidades personales en el proceso externo que faltan, en este orden:
   correos e imagen de la superficie externa (necesitan una subclave de
   cifrado propia); «Mi bolsa» y el portal del candidato; y la lista pública
   de bolsas desde la proyección pública (esquema `bolsa_publica`, aún no
   instalado en la principal). El gobierno
   de autorización de esas capacidades (publicar audiencias, perfil y
   motivos) lo hace el lado interno con un paso de preparación; el proceso
   externo solo recibe claves derivadas para sus audiencias y usuarios de
   PostgreSQL propios, nunca el rol de gobierno ni la clave maestra.
2. Usuarios de PostgreSQL propios del externo para lo que hoy comparte con
   RRHH: identidad, contexto de actor y autorización del candidato, y lectura
   de «Mi bolsa». Es SQL: revisión SQL y ensayo en el clon.
3. Usuarios (preferencias, correos, foto): componer una sola superficie por
   proceso. Hoy la composición exige las dos, así que el proceso interno no
   arranca con `VEC_USUARIOS_PREFERENCIAS_ENABLED=true` sin el fichero del
   externo. Coordinar con F1.3.
4. Con la composición externa: que toda variable de ruta (`*_PATH`, `*_DIR`,
   `*_FILE`) del proceso externo apunte dentro de su material o esté en una
   lista admitida. Hoy el externo no compone nada y en cidonia irá en su
   propio contenedor, pero la comprobación debe existir antes de encenderlo.
5. Datos cifrados que leen los dos lados (el contacto de la participación en
   Bolsa): con claves distintas, el proceso externo pedirá esos datos por un
   puerto autorizado del módulo dueño. Encaja con el módulo Aspirantes (F1.5).

## Despliegue en cidonia (cuando esté lo pendiente)

1. Generar un segundo directorio de material para el externo con su propia
   CA de servidor, clave maestra, sellado e idempotencia; mover a él la
   identidad del candidato y `usuarios-preferencias-externa.json`. Añadir la
   marca `portal-proceso.json` a cada directorio y retirar del interno
   `ca/ca.key`, `ca/serie` y las claves y `.p12` de clientes (guardarlas fuera
   del servidor).
2. Crear los usuarios de PostgreSQL del externo con sus migraciones y
   declarar sus conexiones como `VEC_EXTERNO_*` en un guion propio.
3. Ejecutar `vec-server comprobar-separacion-portales` con los dos directorios
   y los dos guiones. Debe decir `separacion_portales=correcta`.
4. Ensayar los dos procesos en un clon (`vec-clon-*`) y hacer el recorrido en
   Chrome de los dos portales.
5. En la principal: poner `VEC_PORTAL_PROCESO=interno` en `arrancar_app.sh`,
   quitar `VEC_BOLSA_PORTAL_CANDIDATO_ENABLED` y montar solo el material
   interno. Añadir al pod un contenedor nuevo (por ejemplo
   `vec-aplicacion-externa`) con el mismo binario, `VEC_PORTAL_PROCESO=externo`,
   su guion, solo el material externo y otro puerto (por ejemplo
   `127.0.0.1:18444`); el túnel lo publica aparte. Deben ser contenedores
   distintos: dos procesos en el mismo contenedor verían los dos directorios.
6. Contador de conexiones: el guion interno sigue contando sus 15 conexiones
   hasta que salgan las del candidato; el externo tiene su propio contador de
   `VEC_EXTERNO_*`. Cada conexión es un grupo de pgx que abre hasta
   max(4, número de CPU) sesiones, así que el externo suma sesiones en el mismo
   PostgreSQL: revisar `max_connections` antes de encenderlo.

## Riesgos y límites

- Mientras no haya un gestor de claves real, las claves siguen dentro de cada
  proceso. Esto limita el daño a un portal, no lo evita.
- El proceso externo guarda la semilla de la raíz de atestación V3, la misma
  que usa el interno (como ya hace vec-interno). No le sirve para actuar sobre
  datos internos: cada consumo exige además la clave HMAC de su audiencia, que
  el externo solo tiene para las suyas. Una raíz propia del externo exige que
  el gobierno V3 admita varias raíces por configuración; queda para después.
- Ambos procesos usan la misma instancia de PostgreSQL. La separación por
  esquemas por población (opción B) y, si la categorización ENS lo pide, una
  base aparte (opción C) vienen después.
- La clasificación de ficheros y rutas es una lista en el código. Una ruta o un
  fichero nuevo del Área personal hay que añadirlo ahí; si se olvida, el
  proceso externo lo rechaza (falla cerrado) y el interno lo sirve como hoy.
