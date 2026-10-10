# Gobierno de usuarios de Administración

Este comando publica las dos claves que usa Administración para listar y
consultar usuarios. Se ejecuta en la principal de desarrollo, en tres fases:
preparar, aplicar y verificar. No crea el LOGIN técnico ni la configuración
aprobada: eso lo hace el DBA con la migración 000188 ya instalada. Los detalles
de la autoridad están en
`deploy/postgresql/autorizacion_atestada_v3/GOBIERNO_USUARIOS_AD188.md`.

```sh
GOCACHE=$HOME/.cache/go-build go build -p 4 \
  -o /ruta/privada/vec-gobierno-usuarios-admin ./cmd/vec-gobierno-usuarios-admin
```

## Configuración privada

Es un JSON propio con permisos 0600, en una carpeta propia 0700 fuera de
cualquier repositorio. No admite campos desconocidos ni claves repetidas:

```json
{
  "directorio_material": "/ruta/privada/material",
  "ruta_configuracion_hmac": "/ruta/privada/hmac.json",
  "archivo_semilla_raiz": "/ruta/privada/semilla-raiz",
  "dsn_lectura": "host=/run/postgresql user=LECTOR dbname=BASE",
  "dsn_operador": "host=/run/postgresql user=OPERADOR dbname=BASE",
  "salida": "/ruta/privada/gobierno-usuarios-AAAAMMDD",
  "horas_validez_claves": 2
}
```

- Preparar usa las tres rutas de material, `dsn_lectura`, `salida` y
  `horas_validez_claves` (de 1 a 24).
- Aplicar usa `dsn_operador` y `salida`. El usuario de `dsn_operador` no puede
  ser el mismo que el de `dsn_lectura`.
- Verificar usa `dsn_lectura` y `salida`.

### LOGIN de `dsn_lectura` (migración 000235)

Preparar y verificar leen solo por dos funciones de solo lectura,
`instantanea_gobierno_admin_lectura_v1` y `cadena_gobierno_admin_lectura_v1`,
dentro de una transacción READ ONLY. Con la 000235 instalada, el usuario de
`dsn_lectura` no tiene que ser superusuario: basta un LOGIN miembro solo del
grupo `vec_autorizacion_atestada_v3_lector_gobierno`. Ese grupo solo puede
conectarse y ejecutar esas dos funciones; no lee tablas, no escribe y no tiene
TEMP ni CREATE, así que vec-server arranca aunque el LOGIN empiece por `vec_`.
El DBA lo crea así (contraseña como verificador SCRAM, nunca en claro en el
registro de PostgreSQL):

```sql
CREATE ROLE vec_adm_lectura_gob_AAAAMMDD LOGIN INHERIT NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD 'SCRAM-SHA-256$…'
  VALID UNTIL 'AAAA-MM-DD 00:00:00+00';
GRANT vec_autorizacion_atestada_v3_lector_gobierno TO vec_adm_lectura_gob_AAAAMMDD
  WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
```

No debe ser miembro de ningún otro grupo; en particular, nunca del grupo
operador de aplicar. Esta versión de la CLI exige la 000235 instalada: sin
ella, preparar y verificar fallan siempre, también con superusuario.

Las conexiones van por socket local, por la dirección de bucle o por TLS con
verificación del servidor. La carpeta del socket debe existir y no puede tener
escritura para cualquiera (por ejemplo, `/tmp` no vale). El único parámetro de
sesión admitido es `application_name`. La carpeta de
salida debe existir, vacía, con permisos 0700. Ningún fichero se sobrescribe.

## Procedimiento

1. Preparar. Lee la configuración vigente, deriva las dos claves y escribe en
   la carpeta de salida `material.json`, `configuracion-material.json`,
   `plan.json` y `aprobacion-candidata.json`. Las claves dependen de la
   secuencia de la nueva configuración, que crece en cada publicación, así que
   cada renovación diaria obtiene un `clave_id` y un secreto nuevos (AD188
   rechaza los ya publicados). Repetir la preparación sin haber aplicado da las
   mismas claves.

   ```sh
   vec-gobierno-usuarios-admin -fase preparar -config /ruta/privada/config.json \
     -textos /ruta/app/web/static/textos/es/admin-gobierno-usuarios.json -timeout 30s
   ```

   La consola muestra las huellas del plan, del material y de la preimagen, y
   la hora de caducidad del plan (dos horas como máximo). Si falla a medias,
   borre la carpeta entera, porque puede contener `material.json` con las
   claves secretas, y empiece en otra vacía.

2. El DBA crea un LOGIN técnico exclusivo, miembro solo de
   `vec_gobierno_usuarios_admin_operador`, e inserta una fila en
   `config_gobierno_usuarios_admin_v1` con ese `identidad_login`, las tres
   huellas de `aprobacion-candidata.json` y una ventana `vigente_desde` /
   `vigente_hasta` que cubra la aplicación, con `entorno = 'desarrollo'`.

3. Aplicar, con un nombre de acuse nuevo (solo el nombre, sin carpetas):

   ```sh
   vec-gobierno-usuarios-admin -fase aplicar -config /ruta/privada/config.json \
     -textos ... -timeout 30s -acuse acuse-aplicar.json
   ```

   El comando rehace el material con la configuración guardada y comprueba que
   coincide byte a byte con el preparado. Después envía el plan en una
   transacción SERIALIZABLE. El acuse se reserva antes de enviar y solo se
   conserva si la base confirma COMMIT. Si el proceso se corta, puede quedar
   un acuse vacío (0 bytes): no vale como recibo y se puede borrar.

4. Repetir aplicar con otro acuse (`-acuse acuse-replay.json`). La base
   devuelve el mismo recibo y solo añade un intento nuevo. Así se comprueba la
   recuperación, y es también lo que hay que hacer si COMMIT quedó en duda o si
   el acuse no llegó a guardarse. Nunca vuelva a preparar en ese caso. La
   recuperación solo funciona antes de que caduque el plan; después, consulte
   al DBA el estado real antes de hacer nada más.

5. Verificar. Recalcula el tramo de auditoría del gobierno de usuarios y deja
   el informe en `verificacion-cadena-<fecha>.json` dentro de la salida:

   ```sh
   vec-gobierno-usuarios-admin -fase verificar -config /ruta/privada/config.json \
     -textos ... -timeout 30s
   ```

## Conjunto de capacidades (AD198)

Con `"conjunto_capacidades": 1` en la configuración, el comando publica en la
misma operación las dos claves de usuarios y la del lote ordinario de perfiles,
con la migración 000198. Sin el campo, o con 0, sigue usando AD188 y publica
solo las dos de usuarios. El LOGIN técnico de un conjunto es miembro de
`vec_gobierno_capacidades_admin_operador`, y su fila aprobada va en
`config_gobierno_capacidades_admin_v1` con el número de conjunto. El
procedimiento está en
`deploy/postgresql/autorizacion_atestada_v3/GOBIERNO_CAPACIDADES_AD198.md`.

## Conjunto 5 y overlay de versión de Bolsa (AD234)

Con `"conjunto_capacidades": 5` la publicación diaria incluye, además de las
seis claves del conjunto 4, las dos que usa vec-admin para proponer y cerrar
una versión nueva del rol de Bolsa (`vec_autorizacion.versionar_rol_bolsa.propuesta.v1`
y `.cierre.v1`). Requiere la migración 000234 instalada. El LOGIN técnico del
día es miembro de `vec_gobierno_capacidades_admin_operador` y su fila en
`config_gobierno_capacidades_admin_v1` lleva `conjunto_version = 5`.

Después de aplicar (y de la repetición), la fase `version-bolsa` escribe en la
misma carpeta de salida el fichero que lee vec-admin en
`VEC_ADMIN_VERSION_BOLSA_CONFIG_FILE`. No conecta con la base. Necesita en la
configuración privada un bloque más:

```json
"version_bolsa": {
  "pool_gobierno": "/ruta/privada/pools/version-bolsa-ejecutor.json",
  "pool_catalogo": "/ruta/privada/pools/catalogo-acciones-lector.json",
  "motivos": {
    "vec_autorizacion.versionar_rol_bolsa.propuesta.v1": { "catalogo_id": "…", "catalogo_version": 1, "catalogo_huella_sha256": "…", "entrada_clave": "motivo_…" },
    "vec_autorizacion.versionar_rol_bolsa.cierre.v1": { "catalogo_id": "…", "catalogo_version": 1, "catalogo_huella_sha256": "…", "entrada_clave": "motivo_…" }
  }
}
```

Los dos pools son los ficheros de conexión de los LOGIN B1 de vec-admin
(ejecutor `vec_admin_version_rol_bolsa_ejecutor` y lector del catálogo
`vec_admin_catalogo_acciones_lector`), distintos de los de cualquier otro
overlay y fuera de la carpeta del día. Los motivos son las entradas publicadas
del catálogo de motivos que usa vec-admin.

```sh
vec-gobierno-usuarios-admin -fase version-bolsa -config /ruta/privada/config.json \
  -textos ... -timeout 30s -acuse acuse-aplicar.json
```

El acuse es el que guardó aplicar: la fase comprueba que la base confirmó
COMMIT de ese mismo plan y material, y que el conjunto incluye las dos
audiencias. Escribe `version-bolsa-propuesta.bin` y `version-bolsa-cierre.bin`
(cada clave en su propio fichero 0600, que ningún otro overlay usa) y por
último `version-bolsa.json` (0600), con la raíz y la configuración del día y
exactamente esas dos entradas. No sobrescribe nada: cada día va en su carpeta.
Después se reinicia vec-admin apuntando a ese `version-bolsa.json`.

## Salida

Por consola solo sale un JSON con `codigo`, `mensaje` y `limite`, tomados del
catálogo (castellano o inglés), y en preparar las huellas. Nunca muestra DSN,
rutas, errores SQL ni secretos. Sin catálogo válido solo emite
`catalogo_no_disponible`.

| Código | Significado |
|---|---|
| 0 | Hecho: preparación lista, claves confirmadas con acuse guardado o cadena verificada. |
| 1 | Rechazo o fallo sin efecto pendiente, o rechazo/error de la base ya registrado. |
| 2 | Resultado indeterminado (COMMIT no confirmado), COMMIT confirmado sin acuse guardado, o catálogo no disponible. |

Ante un 2, mire `confirmado` y `acuse_guardado` y repita aplicar con otro
acuse.
