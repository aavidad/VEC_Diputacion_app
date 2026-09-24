# Aprovisionamiento privado de `vec-interno` (desarrollo)

Este corte se instala únicamente en una base PostgreSQL 18 desechable y, tras
revisión, en cidonia. El material efectivo se guarda en un directorio 0700
fuera del repositorio, propiedad del usuario de servicio. Todos sus ficheros
son 0600. El script no aplica migraciones, no ejecuta `DOWN`, no reinicia
servicios y no publica gobierno V3. En una base con historia, solo se instalan
las nuevas migraciones que falten mediante el procedimiento principal.

## Dependencias y frontera

- Python 3, OpenSSL 3 y, para los once LOGIN, `psql` o Docker/Podman con
  `psql` dentro de un contenedor PostgreSQL 18 ya existente.
- Token PKCS#11 de desarrollo inicializado por el operador, con PIN, clave
  generada *dentro* del token, `CKA_NEVER_EXTRACTABLE`, `CKA_SENSITIVE` y
  `CKA_PRIVATE`. SoftHSM sirve para el ensayo sintético; su almacén y módulo
  son privados y exteriores a Git. El PIN va en un fichero 0600 exterior a
  Git. Un PKCS#12 exportable no acredita esta protección.
- Los roles, funciones y ACL de los módulos se instalan por sus migraciones
  canónicas antes de crear LOGIN. `roles` comprueba PostgreSQL 18 y las once
  funciones/ACL, ensaya `ROLLBACK` y solo entonces confirma. Ningún LOGIN es
  superusuario ni hereda más de un grupo; cada pool usa un LOGIN propio.
- La CA aquí emitida es **solo de desarrollo**. El certificado personal dura
  37 días y el permiso de acceso tiene retirada explícita. No acredita
  garantía alta ni certificación corporativa.

## Operación

Definir `MATERIAL` como directorio exterior al repositorio y `SCRIPT` como
`deploy/principal/composicion_interna/aprovisionar.py`. Los identificadores de
persona, cuenta y perfil se obtienen del aprovisionamiento sintético
único de F1; no se inventa un segundo registro de persona.

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" init-ca --server-name "$NOMBRE_TLS"
```

La raíz `cmd/vec-interno` no abre TLS desde material propiedad del usuario.
Después de crear la CA, el operador root copia solo fullchain/clave de servidor
y CA a una ruta de sistema con todos los antecesores root-owned. El proceso
Go corre sin root y pertenece al grupo indicado. El script comprueba la
cadena, no sobrescribe archivos diferentes y nunca cambia el origen:

```bash
python3 deploy/principal/composicion_interna/instalar_tls.py \
  --material-dir "$MATERIAL" --dest-root "$TLS_ROOT" \
  --service-group "$GRUPO_SERVICIO"
```

`$TLS_ROOT` debe estar bajo una ruta de sistema como `/etc/vec-interno/tls`:
directorios root-owned 0755/0750, cert/CA root:root 0644 y clave
root:grupo-servicio 0440. No vale `/tmp` ni un directorio de usuario para el
TLS que carga Go. En contenedor aislado se probó que un proceso `nobody` lee
los tres archivos y que todos los propietarios/modos coinciden. El cargador
Go rechaza el material anterior user-owned y acepta la copia root-owned.

Para cada persona sintética, con un token y etiqueta **distintos**, crear el
CSR desde la clave no exportable e instalar el certificado en el token usando
la herramienta PKCS#11 del puesto. `create-csr` genera la clave dentro del
token si no existe, exige sus atributos protegidos y escribe el CSR privado:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" create-csr \
  --subject-id "$PERSONA_REF" --token-url "$URL_CLAVE_PKCS11" \
  --pkcs11-module "$MODULO_PKCS11" --p11tool "$P11TOOL" \
  --certtool "$CERTTOOL" --pin-file "$PIN_FILE"
python3 "$SCRIPT" --material-dir "$MATERIAL" issue-person \
  --csr "$MATERIAL/personas/$PERSONA_REF.csr" --subject-id "$PERSONA_REF" \
  --token-url "$URL_CLAVE_PKCS11" --pkcs11-module "$MODULO_PKCS11" \
  --p11tool "$P11TOOL" --pin-file "$PIN_FILE"
```

`issue-person` coteja la clave pública del CSR con la clave protegida del
token, verifica la firma del CSR y emite un certificado `clientAuth` con SAN
opaco. Se niega a compartir clave entre personas. El certificado público está
en `personas/$PERSONA_REF.crt`; la clave personal nunca se escribe en disco por
este script. El operador importa ese certificado público en el token/NSS de
Chrome junto con la CA de desarrollo. El navegador pide el PIN al usarlo.

La clave HMAC de identidad usa otro objeto de token, de 32 bytes generados
dentro del token. El helper PKCS#11 recibe el PIN desde
`$MATERIAL/identidad/hmac.pin` 0600, sin ponerlo en los argumentos del proceso.
Comprueba `CKA_ALWAYS_SENSITIVE`, `CKA_NEVER_EXTRACTABLE`, `CKA_SIGN` y una
firma HMAC-SHA256 real antes de escribir `identidad/hmac.json`:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" hmac-token \
  --pkcs11-module "$MODULO_PKCS11" --token-label "$TOKEN_LABEL" \
  --token-serial "$TOKEN_SERIAL" --object-id-hex "$OBJETO_HMAC_ID" \
  --key-id "$HMAC_CLAVE_ID" --key-version "$HMAC_VERSION" \
  --domain-ref "$DOMINIO_IDH_REF" --identity-space "$ESPACIO_IDENTIDAD_HTTPS" \
  --pin-file "$MATERIAL/identidad/hmac.pin"
```

Con referencias F1 ya creadas y verificadas, registrar el vínculo técnico de
certificado y el selector nominal de perfil. Estos ficheros no conceden permiso:
la petición debe resolver F1 y V3 contra PostgreSQL en vivo.

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" register-person \
  --subject-id "$PERSONA_REF" --account-id "$CUENTA_REF"
python3 "$SCRIPT" --material-dir "$MATERIAL" register-context \
  --account-id "$CUENTA_REF" --profile-id "$PERFIL_REF" \
  --organization-ref "$ORGANIZACION_CT_REF" --unit-ref "$UNIDAD_CT_REF"
```

`identidad/certificados.json` es el registro de admisión vivo. `init-ca`
emite también `identidad/clientes.crl`, CRL X.509 vacía y firmada. La
revocación operativa se hace con `revoke-person`: primero cambia
atómicamente `activo` a `false`, luego revoca el serial en la AC de desarrollo
y publica una CRL nueva. Ante una interrupción, repetir completa el paso y
el registro privado ya deniega. La siguiente petición se rechaza sin reinicio:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" revoke-person --subject-id "$PERSONA_REF"
```

La CRL vence y debe renovarse antes de `NextUpdate` con `refresh-crl`:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" refresh-crl
```

La comprobación del navegador y el servidor usa esta CRL de desarrollo; no
hay OCSP corporativo ni certificación de producción.
Para revocar F1 se añade la nueva versión de vínculo revocado en su autoridad;
no se elimina historia ni se edita una versión anterior.

La política temporal común se instala con la referencia `pga_`, la huella
SHA256 y la fecha UTC **exactas** que consume el evaluador de garantía. La
tabla de Identidad debe existir por la migración nueva autorizada. Repetir con
los mismos valores comprueba la fila; una fila distinta o revocada detiene la
operación y jamás se reactiva automáticamente:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" policy \
  --database "$PG_DATABASE" --admin-user "$PG_ADMIN" \
  --pg-container "$PG_CONTAINER" --container-engine docker \
  --reference "$POLITICA_REF" --fingerprint "$POLITICA_SHA256" \
  --expires-at "$POLITICA_RETIRA_EN" --key-id "$EMISOR_CLAVE_ID"
```

Después de instalar migraciones y verificar las once funciones propietarias:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" roles \
  --database "$PG_DATABASE" --admin-user "$PG_ADMIN" \
  --pg-container "$PG_CONTAINER" --container-engine docker \
  --db-host "$PG_TLS_HOST" --db-port "$PG_TLS_PORT" --db-ca "$PG_CA_FILE"
```

Si `psql` está en el host, omitir `--pg-container`. El host de `--db-host` debe
coincidir con el SAN del certificado PostgreSQL **desde el proceso de la app**.
`--db-ca` debe ser legible en esa misma ruta desde la app; los DSN de
`pools.json` exigen `sslmode=verify-full`. Los errores de SQL se redactan para
no mostrar contraseñas. El estado de credenciales se conserva en
`pools-state.json` antes de la transacción, para repetirla con los mismos LOGIN.

En una base ya restaurada con preimagen canónica CT `000108`, Identidad
`000005`, Personal BASE `000010`–`000011`, AD3 y las ACL nominales intactas, el instalador
de deltas nuevos exige además Identidad `000004` y ausencia exacta de
Contexto `000006`, Identidad `000006`, Personal `000010a`, CT identidad
`000002` y AD3 `000050a`. Lee las fuentes `.up.sql` canónicas,
ensaya la cadena en `ROLLBACK`, aplica idénticos bytes en un único `COMMIT`
y deja un recibo privado SHA256. Repetirlo con historia falla:

```bash
python3 deploy/principal/composicion_interna/instalar_esquema.py \
  --material-dir "$MATERIAL" --database "$PG_DATABASE" \
  --admin-user "$PG_ADMIN" --pg-container "$PG_CONTAINER" \
  --container-engine docker
```

El dump sintético de septiembre restaurado en el clon E2E llega solo a CT67,
sin Personal ni Contexto/Identidad corporativos. El upgrade focal ensayado
alcanzó CT108/AD3-29/50a/Personal6/Contexto3/5/6, omitiendo CT86–96 y
Dietas/Cronos. Personal BASE `000010` todavía no está en ese clon; no se
instala `000010a` sobre él. La migración histórica Contexto `000004` falla por manifiesto
de predecesor con huella distinta incluso en su runner oficial; Identidad
`000006` exige Identidad `000004`, así que este clon no cumple todavía la
preimagen del instalador. No se fuerza ninguna de esas migraciones ni se
declara consulta E2E antes de resolver esa divergencia con la autoridad SQL.

Los cuatro pools de identidad/F1/auditoría se generan con el mismo patrón en
`identidad/pools.json` mediante `identity-roles`. Requieren Identidad `000006`
y CT `000108` ya instaladas en la preimagen, y sus roles propietarios:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" identity-roles \
  --database "$PG_DATABASE" --admin-user "$PG_ADMIN" \
  --pg-container "$PG_CONTAINER" --container-engine docker \
  --db-host "$PG_TLS_HOST" --db-port "$PG_TLS_PORT" --db-ca "$PG_CA_FILE"
```

Los seis LOGIN de autorización, motivos RRHH, consulta y preflight V3 se
preparan con `v3-roles` cuando AD3 `000050a` y CT `000108` consten instaladas.
El sexto se llama exactamente `vec_interno_preflight_v3_desarrollo` y solo
pertenece a `vec_autorizacion_atestada_v3_preflight_interno`. El script deja
`ct_v3_pools.json` 0600 como entrada privada al manifiesto `ct_v3.json`; no
inventa COSE/HMAC, planes, motivos ni gobierno V3. Estos tienen que existir en
la autoridad V3 antes del arranque y superar su sonda nominal:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" v3-roles \
  --database "$PG_DATABASE" --admin-user "$PG_ADMIN" \
  --pg-container "$PG_CONTAINER" --container-engine docker \
  --db-host "$PG_TLS_HOST" --db-port "$PG_TLS_PORT" --db-ca "$PG_CA_FILE"
```

El entorno de arranque se genera fuera de Git y hereda la fecha de retirada
exacta de la fila gobernada. `runtime-env` exige al menos un certificado
activo, CRL presente y el nombre TLS del servidor. El operador fija solo red,
audiencia y emisor acordados:

```bash
python3 "$SCRIPT" --material-dir "$MATERIAL" runtime-env \
  --listen "$ESCUCHA_INTERNA" --allowed-cidrs "$CIDR_INTERNA" \
  --server-name "$NOMBRE_TLS" --tls-root "$TLS_ROOT" \
  --audience "$AUDIENCIA_INTERNA" \
  --issuer "$EMISOR_INTERNO"
```

`arrancar-interno.env` 0600 contiene `VEC_INTERNO_HTTP_ADDR`,
`VEC_INTERNO_HTTP_ALLOWED_CIDRS`, `VEC_INTERNO_TLS_CERT_FILE`,
`VEC_INTERNO_TLS_KEY_FILE`, `VEC_INTERNO_TLS_CLIENT_CA_FILE`,
`VEC_INTERNO_TLS_SERVER_NAME`, `VEC_INTERNO_IDENTITY_AUDIENCE`,
`VEC_INTERNO_IDENTITY_ISSUER`, `VEC_INTERNO_PROXY_TLS_SHA256`,
`VEC_INTERNO_CERT_POLICY_EXPIRES_AT` y `VEC_INTERNO_MATERIAL_DIR`.
Ningún selector legado de perfil/autenticación/almacén/presentación se añade.
El directorio del material se comprueba contra cualquier árbol Git, no solo
este worktree.

El montaje compone `pools.json`, `identidad/certificados.json`,
`identidad/contextos.json`, `identidad/emisor.json`, la configuración TLS y el gobierno F1/V3 privado
en su manifiesto de arranque. `VEC_INTERNO_CERT_POLICY_EXPIRES_AT` coincide
exactamente con `retirar_en`; no se crea gobierno nuevo al arrancar. La
aplicación falla si falta una dependencia o un pool no acredita LOGIN, TLS y
ACL. El destino debe mantenerse en la superficie interna de desarrollo y
presentación aprobada; cidonia no tiene Kerberos corporativo.

## Chrome con token, sin PKCS#12

Para el ensayo se creó un directorio NSS 0700 exterior a Git, se inicializó
con `certutil -N`, se registró el módulo SoftHSM con `modutil`, se importó la
CA pública con `certutil -A` y el certificado personal público en el token con
`p11tool --write --load-certificate`. El `--id` de ese certificado debe ser
**idéntico** al CKA_ID de la clave privada no exportable. `certutil -L -h
"$TOKEN_LABEL"` mostró `u,u,u` solo tras igualar ambos ID; `certutil -K -h`
enumeró la clave dentro del token. No se creó `.p12` ni clave privada en NSS.

Chrome en Linux lee `~/.pki/nssdb` aunque se use `--user-data-dir`. En la
instancia desechable se comprobó esto con `strace` y se montó el NSS privado
solo en el espacio de nombres del proceso Chrome mediante `bwrap --bind`, sin
editar el NSS global. El intento headless quedó esperando el diálogo de PIN.
Se ejecutó después Chrome real en un Xvfb privado, con `--ozone-platform=x11`:
el diálogo `Unlock Security Device` recibió el PIN con `xdotool type --file`
desde un fichero 0600, se seleccionó el certificado personal y la página HTTPS
mostró `200` con `vec-interno-mtls-test`. El servidor de ensayo exigía
`ssl.CERT_REQUIRED` y CA de desarrollo. Es evidencia de
**navegador→mTLS sintético**, todavía no de identidad/F1/V3/CT en
`cmd/vec-interno`. Para repetir el aislamiento, proporcionar `NSS_PRIVADO` y
`NSS_GLOBAL` ya inspeccionados; no se sustituye un perfil compartido:

```bash
"$XVFB" :91 -screen 0 1280x800x24 -nolisten tcp -ac &
XVFB_PID=$!
DISPLAY=:91 \
bwrap --bind / / --dev-bind /dev /dev --proc /proc \
  --bind "$NSS_PRIVADO" "$NSS_GLOBAL" -- \
  google-chrome --user-data-dir="$PERFIL_CHROME_PRIVADO" \
  --no-sandbox --no-first-run --ozone-platform=x11 --new-window "$URL_INTERNA_HTTPS" &
CHROME_PID=$!
# En el diálogo PIN de ese DISPLAY, sin incluir el PIN en argv:
DISPLAY=:91 xdotool type --file "$PIN_SIN_SALTO_FINAL"
# Seleccionar el certificado personal en el diálogo de Chrome.
# Al terminar, detener solamente CHROME_PID y XVFB_PID de este ensayo.
```

El mismo certificado y PIN se usan en el recorrido final con `cmd/vec-interno`.
El perfil privado se conserva fuera de Git y se retira junto con la instancia
desechable cuando termine la revisión.

## Verificaciones locales del script

En un directorio desechable fuera de Git se comprobaron `init-ca`, CSR desde
SoftHSM 2.6.1 con PIN y clave marcada `CKA_NEVER_EXTRACTABLE`,
`issue-person`, repetición idempotente y `register-person`. Después de
`revoke-person`, `openssl verify -crl_check` devolvió `certificate revoked`
y la repetición de revocación fue idempotente. La ejecución de
`roles`/`policy` requiere PostgreSQL 18 con migraciones nuevas y se acredita
por separado; estas comprobaciones locales de OpenSSL no afirman E2E.
