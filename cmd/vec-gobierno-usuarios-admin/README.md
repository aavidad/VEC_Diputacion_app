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

Las conexiones van por socket local, por la dirección de bucle o por TLS con
verificación del servidor. No se admiten `role` ni `options`. La carpeta de
salida debe existir, vacía, con permisos 0700. Ningún fichero se sobrescribe.

## Procedimiento

1. Preparar. Lee la configuración vigente, deriva las dos claves y escribe en
   la carpeta de salida `material.json`, `configuracion-material.json`,
   `plan.json` y `aprobacion-candidata.json`.

   ```sh
   vec-gobierno-usuarios-admin -fase preparar -config /ruta/privada/config.json \
     -textos /ruta/app/web/static/textos/es/admin-gobierno-usuarios.json -timeout 30s
   ```

   La consola muestra las huellas del plan, del material y de la preimagen, y
   la hora de caducidad del plan (dos horas como máximo). Si falla a medias,
   no reutilice la carpeta: empiece en otra vacía.

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
   conserva si la base confirma COMMIT.

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
