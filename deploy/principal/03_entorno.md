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
