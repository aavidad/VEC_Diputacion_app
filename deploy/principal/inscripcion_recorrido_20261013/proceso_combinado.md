# Inscripción del aspirante en el proceso combinado

Con `VEC_PORTAL_PROCESO` sin definir, el único `vec-server` atiende también
«Inscribirme» y «Mis solicitudes». Reutiliza el montaje del portal externo y
la sesión del candidato en Mi Bolsa externa; RRHH conserva su bandeja y su sesión
interna. El proceso `interno` separado no abre estas conexiones externas.

CONFIG NUEVA: en el proceso combinado se exige
`VEC_EXTERNO_PREFLIGHT_V3_DATABASE_URL` con el LOGIN
`vec_externo_preflight_v3_desarrollo`, y el material del consumidor
`inscripcion_bolsa` en `$VEC_MATERIAL_PORTAL_EXTERNO_DIR/externo/v3/`.
CONFIG NUEVA: `VEC_MATERIAL_PORTAL_EXTERNO_DIR=/vec-material-externo`, material
privado externo distinto del interno y con idempotencia propia.
No hay SQL nueva. La opción del directorio externo permite conservar su
idempotencia y sus perfiles propios dentro del proceso combinado.

Dirección mantiene las siguientes opciones en el entorno privado del proceso:

| Opción | Valor o conexión |
| --- | --- |
| `VEC_PORTAL_PROCESO` | Sin definir, para el proceso combinado |
| `VEC_BOLSA_INSCRIPCIONES_ENABLED` | `true` |
| `VEC_MATERIAL_PORTAL_EXTERNO_DIR` | `/vec-material-externo`, mount privado propio |
| `VEC_EXTERNO_BOLSA_INSCRIPCIONES_LECTOR_DATABASE_URL` | LOGIN `vec_bolsa_inscripciones_lector` |
| `VEC_EXTERNO_BOLSA_DATABASE_URL` | LOGIN `vec_externo_bolsa_desarrollo` |
| `VEC_EXTERNO_PREFLIGHT_V3_DATABASE_URL` | LOGIN `vec_externo_preflight_v3_desarrollo`, sólo lectura |
| `VEC_EXTERNO_AUTORIZACION_FUENTE_DATABASE_URL` | LOGIN externo existente, rol `vec_autorizacion_fuente_externa` |
| `VEC_EXTERNO_AUTORIZACION_REGISTRO_DATABASE_URL` | LOGIN externo existente, rol `vec_autorizacion_registro_externo` |
| `VEC_EXTERNO_AUTORIZACION_MOTIVOS_DATABASE_URL` | LOGIN externo existente, rol `vec_autorizacion_motivos_externos` |

Los DSN usan TLS verificado. Ninguno se sustituye por el LOGIN interno de
RRHH. Se conservan el lector de RRHH, su fichero de identidad y los permisos
publicados según `LEEME.md` y `gobierno_inscripcion.md`.

Prepare [la base externa](base_material_externo.md) en un directorio privado del
host, antes de montarlo en `/vec-material-externo`. Con el entorno interno,
publique después los consumidores mediante `vec-server preparar-portal-externo
--material-externo "$MATERIAL_EXTERNO" --consumidores
mi_bolsa,portal_candidato,preferencias_bolsas,inscripcion_bolsa`, omitiendo
`portal_candidato` si sus acciones están apagadas y conservando los otros
consumidores ya publicados. El destino es distinto del material interno.
No copie claves internas al externo ni guarde material o DSN en Git.

La raíz despacha Mi Bolsa, Preferencias de Bolsa, convocatorias abiertas y
solicitudes propias a sus montajes externos, fuera de la frontera interna de
CT. La composición y provisión internas de Mi Bolsa se omiten en combinado. Cada lectura
y presentación conserva las comprobaciones y la auditoría del constructor
existente. Una dependencia ausente impide el arranque; con inscripción apagada
no se abre el preflight. Los pools se cierran al fallar el arranque o al detener
el servidor.

Dirección comprueba en Chrome el recorrido final: el aspirante abre la
convocatoria v2, presenta su solicitud y recupera el recibo; RRHH ve esa misma
solicitud en «Solicitudes y admisión». La prueba focal de composición no
sustituye ese recorrido ni acredita sus tiempos HTTP.
