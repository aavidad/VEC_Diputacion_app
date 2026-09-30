# Separación de portales: composición y material propios

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

## Situación de partida, 29 de septiembre de 2026

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
  Cada uno tiene su propia idempotencia, clave TLS y autoridad certificadora.
  El KMS y el sellado de tiempo bajo `kms/` y `tsa/` pertenecen al interno.
  El externo solo admite su lista positiva (identidad del candidato,
  `usuarios-preferencias-externa.json`, `mtls/candidato.crt`, `externo/…`,
  más TLS, idempotencia, `ca/ca.crt`, manifiesto y
  `desarrollo.env`). El interno rechaza esos ficheros del externo. Ninguno
  admite nada bajo `ca/` salvo `ca/ca.crt`; bajo `tls/` solo admite
  `tls/servidor.crt` y `tls/servidor.key`. El interno admite en `kms/` solo la
  clave maestra y los pares de atestación y revalidación; en `tsa/`, solo
  `clave-hmac.bin`. El externo rechaza todos esos ficheros antes de leerlos.
  Rechaza claves privadas de cliente, `*.pem`, `*.p12`, `*.password`, otros
  `*.key` y enlaces simbólicos.
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

### Lista pública de bolsas en el externo

B10 consulta la proyección pública gobernada en una base dedicada. El proceso
externo usa exclusivamente el login `vec_externo_bolsa_publica_consulta` y la
variable `VEC_EXTERNO_BOLSA_PUBLICA_DATABASE_URL`. El login recibe esta única
membresía, después de instalar las migraciones públicas 000001 y 000002:

```sql
GRANT vec_bolsa_publica_consulta TO vec_externo_bolsa_publica_consulta
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
```

La [receta de aprovisionamiento](../../deploy/postgresql/bolsa_publica/README.md#login-lector-del-proceso-externo-b10)
contiene la creación del login, sus ajustes y la entrega privada de la
contraseña. No se concede acceso a tablas internas ni a roles de gobierno.
No reaplique SQL instalado. La activación requiere el manifiesto SHA-256 de
la proyección y las huellas gobernadas de categorías. Sin activación B10,
la lista sigue ausente; una configuración parcial impide el arranque.

Las consultas del proceso externo mantienen el mTLS del candidato. Para la
entrada anónima se utiliza `cmd/vec-publico`, separado del Área personal y
con su propio login lector. Servir una ruta `/api/publico/` desde el proceso
externo no la convierte en una entrada sin certificado.

### Comprobación al desplegar

Cada proceso por sí solo no puede saber si su clave es la misma que la del
otro. Por eso hay un subcomando para quien despliega, que ve los dos lados:

```text
vec-server comprobar-separacion-portales \
  --material-interno DIR --material-externo DIR \
  [--entorno-interno GUION] [--entorno-externo GUION]
```

Falla si los dos directorios comparten autoridad certificadora, incluso cuando
el formato PEM difiere. Compara la huella SHA-256 del certificado DER y rechaza
un certificado de CA ausente o inválido. El manifiesto externo debe usar
versión 2 y declarar `huella_ca_interna_sha256`, calculada sobre el DER de la
CA interna actual, como 64 caracteres hexadecimales en minúsculas. La
comprobación coteja esa declaración con el certificado del directorio interno.
También falla si algún secreto (KMS,
sellado, idempotencia, clave TLS, `externo/*.bin`) tiene el mismo contenido en
los dos directorios, si un mismo usuario de
PostgreSQL aparece en las conexiones de ambos (variables del guion y cadenas
dentro de los JSON del material) o si alguna conexión no lleva usuario
explícito. Solo imprime recuentos y el nombre del elemento repetido.

## Código, ensayo y trabajo pendiente

La composición separada incluye estas capacidades, sin cambiar el modo
combinado cuando `VEC_PORTAL_PROCESO` está vacío:

1. La comprobación de entorno y material precede a las lecturas y conexiones.
   Cada proceso recibe sus credenciales, identidad, TLS e idempotencia.
2. El interno mantiene la composición de RRHH. El externo compone el Área
   personal, «Mis preferencias», correos, imagen, «Mi bolsa» y el portal del
   candidato con sus conexiones y perfiles propios. Cada capacidad se activa
   con su configuración completa; una configuración parcial impide el arranque.
3. El externo no carga identidad de RRHH, KMS, TSA ni conexiones internas.
   Exige una persona candidata registrada y una CA propia. Las rutas internas
   devuelven 404 en este proceso.
4. La preparación explícita conserva una raíz V3 externa propia y entrega
   solo las claves de las audiencias habilitadas. La publicación requiere
   aprobación y preimagen. El servidor externo nunca publica permisos por
   petición: coteja el material con su preflight nominal de solo lectura,
   mediante TLS `verify-full`.
5. El externo puede exportar sus alias sin conexiones. En el proceso interno,
   `preparar-portal-externo --seudonimos FICHERO --cuentas cta_…` comprueba la
   provisión de las cuentas autorizadas en su invocación aprobada, después de
   publicar V3. La propuesta pendiente no acredita esos alias. La herramienta
   no crea cuentas ni sustituye su provisión por huella y CAS. Los alias
   permanecen en el espacio `vec.identidad.desarrollo.externo`.
6. La lista pública usa la proyección gobernada de Bolsa con un lector propio.
   La entrada sin certificado se sirve desde `cmd/vec-publico`; el Área
   personal conserva el mTLS del candidato. Ambas entradas mantienen cerradas
   las rutas internas.

El corte anterior de preferencias se recorrió con ambos procesos en un clon:
GET 200 y PUT 201 en el externo, 404 en el interno y denegación de acceso a
esquemas internos con los logins externos. Para P2, dirección conserva el
ensayo PostgreSQL 18 de AD3-123 y la publicación y recuperación de una
configuración externa con 17 claves, sin duplicados y con el gobierno interno
intacto. Estos resultados no acreditan una instalación en la principal ni el
recorrido completo de todas las capacidades después de reunir las ramas.

Quedan la revisión del conjunto final, el recorrido completo con ambos
procesos tras reunir #178, #179 y P2, y la preparación del despliegue aprobado.
El circuito de avisos se entrega aparte: la composición de correos no acredita
por sí sola envío, recepción ni entrega de una comunicación de Bolsa.

## Preparación del despliegue

1. Preparar un directorio exclusivo para el externo con CA, certificado y
   clave TLS propios, identidad del candidato, configuración de Usuarios e
   idempotencia. La clave privada de la CA y las de los clientes quedan fuera
   de ambos procesos. Cada directorio lleva su marca `portal-proceso.json` v1;
   el manifiesto interno conserva v4.
2. Para el correo externo, conservar los mismos 32 bytes de su semilla en
   `usuarios/correos-externos-semilla.bin`, mediante el traslado controlado
   previsto. No copiar al externo el KMS ni el sellado interno. La preparación
   V3 no genera esta semilla ni cambia los sobres cifrados existentes.
3. Crear los logins externos y sus concesiones mediante el lote SQL aprobado;
   declarar las conexiones propias y el TLS `verify-full`. Instalar solo las
   migraciones nuevas; no reaplicar SQL instalado ni revertir su historia.
4. Ejecutar desde el lado interno `vec-server preparar-portal-externo`, sin
   `--seudonimos`, para obtener primero la propuesta y el manifiesto externo
   v2 con la huella pública DER de la CA interna. El paso conserva el
   manifiesto interno y la raíz propia preparada; aún no publica el gobierno.
5. Con ese material, exportar los alias y provisionar las cuentas, personas y
   perfiles externos por sus herramientas aprobadas, huella y CAS. El
   [procedimiento del candidato](../../cmd/vec-provisionar-candidato-externo/README.md)
   fija las fases y sus identidades técnicas.
6. Repetir la preparación con `--seudonimos` y la lista `--cuentas` autorizada
   para obtener la propuesta; esa respuesta pendiente no acredita la provisión
   de los alias. Revisarla e invocar la publicación con sus
   `--aprobacion-sha256` y `--preimagen-sha256`. Esta invocación publica V3 y
   después coteja la provisión anterior, sin realizarla en su lugar. Si el
   cotejo falla, V3 ya está publicado: corregir la provisión y repetir con la
   misma aprobación y preimagen. El replay recupera la publicación existente
   sin duplicarla y permite completar el cotejo.
7. Ejecutar `vec-server comprobar-separacion-portales` con ambos directorios y
   guiones. Debe devolver `separacion_portales=correcta`: comprueba CA distintas,
   la huella interna declarada, secretos distintos y logins sin cruces.
8. Ensayar en el clon los dos procesos a la vez y recorrer las capacidades
   personales en Chrome, con recuperación y antecedentes conservados.
9. Tras la revisión de dirección, preparar procesos o contenedores separados,
   cada uno con su material, guion y puerto aprobados. El interno usa
   `VEC_PORTAL_PROCESO=interno`; el externo, `VEC_PORTAL_PROCESO=externo`.
   Inventariar sus conexiones y límites de pools frente a `max_connections`.
   Este documento no acredita que el despliegue ya se haya ejecutado.

## Riesgos y límites

- Mientras no haya un gestor de claves real, las claves siguen dentro de cada
  proceso. Esto limita el daño a un portal, no lo evita.
- La preparación explícita crea y conserva una raíz de atestación V3 propia
  del externo. Su semilla permanece en su material; el gobierno recibe la clave
  pública y publica su configuración por huella y control de versión. La
  configuración interna mantiene su raíz y sus referencias. Las claves de
  audiencias del material externo se limitan a las capacidades habilitadas.
- Una rotación exige la preimagen aprobada y reiniciar el proceso externo para
  cargar la nueva raíz. Un proceso que conserve la anterior deniega el nuevo
  material; no adopta la raíz interna como sustituta. La instalación y el
  recorrido con los dos procesos se comprueban aparte del ensayo del código.
- El ensayo usa una instancia PostgreSQL compartida, con identidades técnicas
  y almacenes de población separados. Eso no acredita aislamiento de máquinas
  ni el despliegue de una base aparte.
- La clasificación de ficheros y rutas usa listas positivas. El material
  externo nuevo requiere una ruta nominal admitida; si falta, el arranque
  externo lo rechaza.

### Preflight del gobierno externo

El preflight externo usa exclusivamente el login
`vec_externo_preflight_v3_desarrollo` (AD3-112), con TLS `verify-full`, también
en loopback y en los ensayos locales. Rechaza conexiones sin TLS, sin
verificación del nombre del servidor o con alternativas de conexión inseguras
antes de abrir el pool. No cambia los permisos ni el SQL instalado.

Pruebas focales del material y la configuración de conexión:

```sh
GOCACHE=/dev/shm/go-build TMPDIR=/tmp go test -p 32 \
  ./internal/app/separacionportales ./internal/app/bootstrap \
  -run 'Test(PreflightV3PortalExterno|ComprobarSeparacion|ProcesoExterno|ProcesoSeparadoSolo|MaterialPropio|MaterialSin)'
```

Estas pruebas verifican rechazo y configuración local; el recorrido con los
dos procesos y PostgreSQL se comprueba por separado.

El arranque externo exige ese manifiesto v2 y rechaza una huella interna
ausente, mal formada o igual a su propia CA. Lee solo su directorio: la
preparación del operador entrega la huella pública y la comprobación de
despliegue contrasta ambos directorios. El `manifiesto.json` interno conserva su
versión 4. La marca `portal-proceso.json` sigue en versión 1 en ambos procesos.

El manifiesto es JSON sin firma. Su confianza procede del aprovisionamiento
controlado y del cotejo de despliegue. Las comprobaciones de fichero regular,
enlaces y permisos protegen la lectura, pero no acreditan por sí solas quién
lo preparó. Cada renovación de CA requiere repetir la preparación explícita
y el cotejo antes del arranque.

### Material del correo externo

Usuarios conserva su semilla de 32 bytes en
`usuarios/correos-externos-semilla.bin`, dentro del material externo. Solo se
admite ese fichero nominal; el interno lo rechaza y el externo sigue
rechazando todo `kms/` y `tsa/`. La comprobación de separación incluye esta
semilla al detectar contenidos repetidos entre ambos directorios.

El traslado controlado conserva los mismos bytes de la semilla existente,
con fichero regular y permisos 0600. Mantiene las derivaciones, referencias
y sobres cifrados anteriores. El preparador de la raíz V3 no crea ni renueva
esta semilla.
