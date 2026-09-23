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

En cidonia, con AD3 `000043` y Bolsa `000010` ya instaladas, preparar **fuera de
Git** un candidato sintético distinto de RRHH e Intervención. No ejecutar de
nuevo esas migraciones. Usar la CA que ya está en
`$VEC_DEVELOPMENT_MATERIAL_DIR/ca/`; emitir `mtls/candidato.crt` y
`mtls/candidato.key` con el mismo procedimiento de
`scripts/generar_credenciales_desarrollo.sh` para `cliente`, cambiando el CN y
SAN a `candidato-bolsa-desarrollo` y `urn:vec:desarrollo:candidato-bolsa`.
Mantener certificado, clave, CSR, extensión y contraseña PKCS#12 en el directorio
privado, propietario del servicio, modo `0600`; exportar PKCS#12 para el
navegador del ensayo. La clave y la contraseña no entran en JSON ni en Git.

1. Con acceso de lectura del propietario de Bolsa, elegir **una participación
   importada existente** y anotar `candidato_ref` mediante
   `SELECT candidato_ref, participacion_ref FROM
   vec_bolsa_llamamientos.vinculo_candidato WHERE candidato_ref = '<ref>'`.
   No inventar un `can_` ni importar otra bolsa para esta prueba.
2. Calcular `certificate_sha256` como SHA-256 del DER de
   `mtls/candidato.crt` (`openssl x509 -in ... -outform DER | sha256sum`).
   Elegir un `subject` opaco `per_...` sintético, exclusivo del candidato.
   Calcular las referencias con SHA-256 y los **primeros 16 bytes** en hex:
   `cuenta_ref = "cta_" + SHA256("vec.ct.alta.desarrollo.v1\0" + subject +
   "\0" + certificate_sha256 + "\0cuenta")[:16]`;
   `persona_ref` usa el mismo material y sufijo `\0persona` con prefijo
   `per_`. Elegir `perfil_ref` opaco `prf_...` exclusivo.
3. Crear `identidad/candidato.json` (modo `0600`) con
   `{"version":1,"autoridad":"no_autoritativo","certificate_sha256":"<SHA256 DER>","subject":"<subject>","display_name":"Candidato sintético","roles":["candidato_bolsa"]}`.
   Crear `identidad/bolsa-candidato.json` (modo `0600`) con
   `{"version":1,"autoridad":"no_autoritativo","certificado":"mtls/candidato.crt","identidad":"identidad/candidato.json","sujeto":"<subject>","cuenta_ref":"<cuenta_ref>","persona_ref":"<persona_ref>","perfil_ref":"<perfil_ref>","candidato_ref":"<ref importada>"}`.
4. En la base de desarrollo y mediante el propietario de
   `vec_contexto_actor_v1`, proyectar esa cuenta, persona, perfil, vínculo de
   cuenta a perfil y **un solo** vínculo vigente de tipo `candidato` a la
   referencia importada. La plantilla exacta de columnas, tablas `*_versiones`
   y punteros `*_actual` es
   `deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql`:
   sustituir sus cinco referencias opacas y la procedencia por valores nuevos,
   el tipo/referencia del primer vínculo por `candidato`/`<ref importada>`,
   y **omitir** su segundo vínculo de ejemplo. Usar una transacción, procedencia
   `no_autoritativa`, vigencia que incluya el momento del ensayo, y comprobar
   que no existe otro vínculo `candidato` vigente para esa persona. No insertar
   en el registro `contexto_actor_v1`: el arranque y cada GET lo crean mediante
   el resolutor registrado existente.
5. Reiniciar solo la instancia de desarrollo y consultar
   `GET /api/vec/bolsa/mi-bolsa` con el PKCS#12 del candidato: esperar `200` y
   el aviso de certificado sintético. Repetir con el certificado RRHH (denegado)
   y probar una ruta interna con el candidato (denegada). Si no hay material
   válido o la participación no coincide, el arranque falla cerrado. Cl@ve,
   certificado FNMT y DNIe dependen de la pasarela de Sistemas.
