# Entorno privado de la principal para `main@48e64f84`

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
