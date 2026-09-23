# Entorno privado de la principal para `main@48e64f84`

## Activar Dietas

Con `main` desplegada, ejecutar **como el usuario del servicio** en cidonia:

```bash
VEC_DEVELOPMENT_MATERIAL_DIR=<material privado> python3 deploy/principal/preparar_dietas_desarrollo.py
```

El script es idempotente (fechas de la política fijadas en su estado privado) y ensaya cada paso con `ROLLBACK` antes de confirmarlo:
instala `04_dietas_migraciones.sh` si falta; crea ocho LOGIN nominales con
credenciales privadas; da a la identidad sintética del certificado cliente de
la demo (la de Bolsa/B-BACK) un perfil Dietas, una relación y asignación de
Personal rotuladas provisionales y su rol V3; escribe
`identidad/dietas-comisiones.json` y `dietas-r1d.env` (OSRM limitado a Granada
y a su IP `/32`), añade `--env-file` a las tres variantes de `arrancar_app.sh`
con copia `*.antes-dietas-r1d` y rearranca.

**No publica gobierno V3.** En desarrollo hay un único gobierno: Contratación
publica la configuración vigente con una sola raíz y la renueva cada día. Dietas
deriva sus tres claves de capacidad por audiencia bajo esa raíz, como Bolsa
(`internal/app/bootstrap/dietas_material_ct_desarrollo.go`). Publicar otra
configuración movería el puntero único y dejaría sin firma a CT y Bolsa.

Si el contenedor OSRM no es inequívoco, el script se detiene antes de escribir y
pide `VEC_DIETAS_OSRM_URL=http://IP:PUERTO`.

Volver atrás: restaurar las tres copias `*.antes-dietas-r1d` sobre
`arrancar_app.sh` y sus variantes y rearrancar. La historia SQL, la relación
sintética y el material privado se conservan; no ejecutar migraciones `DOWN`.
También siguen vigentes el rol y la asignación RBAC de Dietas y los ocho LOGIN;
sin el selector activo no se usan (desarrollo).


El inventario canónico de `config/` contiene 18 variables `VEC_*_DATABASE_URL`.
La principal ya tiene las once identidades de Contratación/Bolsa llamamientos y
`VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL`. Faltan estas seis en
`material/arrancar-local.sh`; el contador exacto pasa de **12 a 18**:

```bash
VEC_BOLSA_PUBLICA_DATABASE_URL='postgresql://vec_bolsa_publica_consulta_desarrollo@localhost:5432/postgres?sslmode=verify-full&sslrootcert=/ruta/privada/postgresql/ca.crt'
VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL='postgresql://vec_bolsa_importacion_convoca_desarrollo@localhost:5432/postgres?sslmode=verify-full&sslrootcert=/ruta/privada/postgresql/ca.crt'
VEC_BOLSA_BORRADORES_EJECUTOR_CONSULTA_DATABASE_URL='postgresql://vec_bolsa_convocatorias_ejecutor_consulta_desarrollo@localhost:5432/postgres?sslmode=verify-full&sslrootcert=/ruta/privada/postgresql/ca.crt'
VEC_BOLSA_BORRADORES_PROYECTOR_GOBIERNO_DATABASE_URL='postgresql://vec_bolsa_convocatorias_proyector_gobierno_desarrollo@localhost:5432/postgres?sslmode=verify-full&sslrootcert=/ruta/privada/postgresql/ca.crt'
VEC_BOLSA_BORRADORES_VERIFICADOR_RECIBO_DATABASE_URL='postgresql://vec_bolsa_convocatorias_verificador_recibo_desarrollo@localhost:5432/postgres?sslmode=verify-full&sslrootcert=/ruta/privada/postgresql/ca.crt'
VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL='postgresql://vec_b2_auditoria_frontera_desarrollo@localhost:5432/postgres?sslmode=verify-full&sslrootcert=/ruta/privada/postgresql/ca.crt'
```

Son ejemplos sin contraseña. Deben conservar el host/puerto/base y el patrón TLS
de las doce conexiones privadas existentes. Cada URL usa un LOGIN diferente y
el rol nominal indicado por su nombre; no se reutiliza una credencial. El nuevo
LOGIN de auditoría se crea mediante `01_roles.sql`; su contraseña solo entra por
`VEC_BOLSA_AUDITORIA_FRONTERA_LOGIN_PASSWORD` al ejecutar el despliegue.

Para activar B-BACK/B2 también se exige `VEC_BOLSA_BORRADORES_ENABLED=true` y el
fichero privado `identidad/bolsa-bback.json` bajo
`VEC_DEVELOPMENT_MATERIAL_DIR`, modo `0600`, propietario `openclaw`. Ejemplo JSON
válido para las doce bolsas sintéticas constituidas:

```json
{
  "version": 2,
  "autoridad": "no_autoritativo",
  "sujeto": "per_0123456789abcdef0123456789abcdef",
  "certificado_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "perfil_ref": "prf_0123456789abcdef0123456789abcdef",
  "unidad_ref": "unidad:desarrollo:rrhh",
  "ambito_ref": "ambito:desarrollo:bolsa",
  "bolsas_ref": [
    "bolsa:demo:ctpd-auxilar-de-enfermaria",
    "bolsa:demo:administrativo",
    "bolsa:demo:trabajador-social",
    "bolsa:demo:operario",
    "bolsa:demo:oficial-de-servicios-multiples",
    "bolsa:demo:tecnico-de-gestion",
    "bolsa:demo:auxiliar-administrativo",
    "bolsa:demo:educador",
    "bolsa:demo:psicologo",
    "bolsa:demo:enfermera-o",
    "bolsa:demo:encargado",
    "bolsa:demo:auxiliar-servicios-generales"
  ]
}
```

El ejemplo es sintético y no se copia literalmente: `sujeto`,
`certificado_sha256` y `perfil_ref` deben ser los tres valores correlacionados
con la identidad de desarrollo ya publicada. Ninguno se obtiene del navegador.
No se versionan el fichero efectivo, contraseñas, certificados ni DSN reales.

## Lo que exigió de verdad el despliegue en cidonia (23/09/2026)

- **Conexiones.** Con Bolsa llamamientos configurada, el arranque solo exige
  `VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL` (LOGIN
  `vec_b2_auditoria_frontera_desarrollo`); las otras cinco de la lista anterior
  son de funciones opcionales (consulta pública separada, borradores de
  convocatoria) y su ausencia no impide arrancar. En cidonia el contador pasó de
  12 a **13**, no a 18.
- **`perfil_ref` no es libre.** El arranque de B-BACK lo compara con el perfil
  que deriva de la identidad (`bolsa_borrador_identidad_desarrollo.go`), así que
  un valor inventado lo rechaza con «material criptográfico de desarrollo
  inválido». Se calcula así, con `sujeto` y `certificado_sha256` del mismo
  manifiesto:

  ```text
  perfil_ref = "prf_" + hex(SHA-256("vec.ct.alta.desarrollo.v1\0" + sujeto
                                    + "\0" + certificado_sha256
                                    + "\0perfil-bolsa-bback-v1")[0:16])
  ```

- **`bolsas_ref`** son las referencias vigentes de
  `vec_bolsa_llamamientos.bolsa_constituida` en esa base; en cidonia, las doce
  importadas de CONVOCA (`bolsa:<categoría>:2026-09-17`).
- **Réplica atrasada.** Si la base está por detrás de AD3 `000046` / Bolsa
  `000014`, primero `00_puesta_al_dia.sh` (ensayo con `ROLLBACK`, luego
  `COMMIT`), después `01_roles.sql`, el ensamblador `02_migraciones.sh` y Bolsa `000018`.
- **Diagnóstico.** Si B-BACK no monta, el registro de arranque indica ahora el
  fichero y la línea de la comprobación que falló, encadenando la causa.

## D3-B11: acceso personal de Bolsa en desarrollo

En cidonia, tras instalar las migraciones autorizadas, ejecutar **como `openclaw`**:

```bash
bash deploy/principal/preparar_candidato_desarrollo.sh
```

El script usa el material de `${XDG_STATE_HOME:-$HOME/.local/state}/vec-diputacion/desarrollo`,
`vec-postgresql-20260906` y la base `postgres` por defecto. Se pueden ajustar
`VEC_DEVELOPMENT_MATERIAL_DIR`, `VEC_PRINCIPAL_POSTGRES_CONTAINER`, `VEC_CANDIDATE_DATABASE`,
`VEC_CANDIDATE_DATABASE_USER`, `VEC_CANDIDATE_BOLSA_REF` y
`VEC_CANDIDATE_BASE_URL`. Emite el certificado y el PKCS#12 con la CA existente;
elige una participación importada de bolsa vigente; calcula las referencias
opacas y escribe los dos manifiestos privados tras ensayar la proyección SQL
con `ROLLBACK` y aplicarla con `COMMIT`. Una intención privada conserva la
selección ante un fallo entre COMMIT y manifiesto; cada transacción comprueba
la vigencia de esa participación bajo bloqueo de la tabla de bolsas. Repetirlo con el mismo material y
proyección no duplica filas ni sobrescribe ficheros. Ante material parcial,
referencias distintas o vínculo candidato previo distinto, se detiene.

Al terminar imprime el `podman restart` de la aplicación y el `curl` mTLS
exactos para `GET /api/vec/bolsa/mi-bolsa`. Ese GET y los controles negativos
con RRHH y con una ruta interna siguen siendo comprobaciones del operador en
cidonia; el script no reinicia servicios ni declara el HTTP probado. El
certificado es sintético y no sustituye Cl@ve, FNMT ni DNIe, que dependen de la
pasarela de Sistemas. No reaplicar AD3 `000043` ni Bolsa `000010`.

### D3-B11-D: vínculos de constituciones anteriores a Bolsa 000008

Bolsa `000021` añade una función propietaria de relleno. El paquete
`02_migraciones.sh` la ensambla desde su fuente canónica. En una base donde
`000016`/`000017` ya tienen historia, instalar **solo** `000021`: primero
reemplazar el `COMMIT;` final por `ROLLBACK;` y ejecutar con `psql -v
ON_ERROR_STOP=1`, después ejecutar el `.up.sql` intacto. No reaplicar las
migraciones antiguas ni ejecutar el `.down.sql` si hay vínculos.

El staging de CONVOCA guarda las filas cifradas; la referencia `can_*` exige
el mismo material KMS que utilizó `constituir-bolsa`. Por ello SQL no descifra
ni recibe la clave: `rellenar-vinculos-bolsa` usa el recuperador acreditado de
CONVOCA, `DerivadorCandidatoHMAC` y el número de fila guardado en la
constitución. Solo transmite pares `{fila_numero,candidato_ref}` a la función
de Bolsa. Esta comprueba que el acta ya existe, que se entregan **todas** sus
filas una sola vez y que cualquier vínculo previo coincide. No modifica
constituciones, situaciones ni vínculos existentes. Si el staging fue
expurgado o falta la clave KMS original, se detiene: no se infieren referencias.

Como administrador de la base, enumerar las actas ya constituidas con su
huella y categoría desde la importación (solo metadatos, sin staging):

```sql
SELECT l.huella_fichero_sha256, l.categoria_ref
FROM vec_bolsa_importacion_convoca.lote l
WHERE EXISTS (
  SELECT 1 FROM vec_bolsa_llamamientos.constitucion c
  WHERE c.acta_ref=l.acta_ref
)
ORDER BY l.categoria_ref, l.huella_fichero_sha256;
```

Con `VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL` de **recuperación** (LOGIN
miembro solo de `vec_bolsa_importacion_convoca_recuperador`) y la conexión
administrativa existente `VEC_PRINCIPAL_ADMIN_DATABASE_URL`, por cada pareja
ejecutar el binario fuera del servidor web en el entorno privado de desarrollo.
La función 000021 no concede `EXECUTE` al ejecutor ordinario de Bolsa: el
comando usa `SET LOCAL ROLE vec_bolsa_llamamientos_propietario` dentro de la
transacción administrativa. La primera llamada ensaya en `ROLLBACK`; la
segunda confirma. Se repite para las doce actas y se comprueba que un segundo
`--aplicar` informa `nuevos=0`. No se pasan nombres ni documentos por CLI.

```bash
vec-server rellenar-vinculos-bolsa --huella '<sha256_importado>' --categoria '<categoria_ref>'
vec-server rellenar-vinculos-bolsa --huella '<sha256_importado>' --categoria '<categoria_ref>' --aplicar
bash deploy/principal/preparar_candidato_desarrollo.sh
```

`preparar_candidato_desarrollo.sh` sigue siendo una operación posterior y
separada. Instalar 000021 y rellenar los vínculos no enciende por sí mismo
«Mi bolsa» ni declara una sesión de candidato probada.
