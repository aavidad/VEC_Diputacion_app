# Inscripción del aspirante en el proceso combinado

Con `VEC_PORTAL_PROCESO` sin definir, el único `vec-server` atiende también
«Inscribirme» y «Mis solicitudes». Reutiliza el montaje del portal externo y
la sesión externa de Preferencias; RRHH conserva su bandeja y su sesión
interna. El proceso `interno` separado no abre estas conexiones externas.

CONFIG NUEVA: en el proceso combinado se exige
`VEC_EXTERNO_PREFLIGHT_V3_DATABASE_URL` con el LOGIN
`vec_externo_preflight_v3_desarrollo`, y el material del consumidor
`inscripcion_bolsa` en `$VEC_DEVELOPMENT_MATERIAL_DIR/externo/v3/`.
No hay nuevas variables ni SQL en esta PR: son las dependencias existentes
del montaje externo, ahora alcanzable desde la raíz combinada.

Dirección mantiene las siguientes opciones en el entorno privado del proceso:

| Opción | Valor o conexión |
| --- | --- |
| `VEC_PORTAL_PROCESO` | Sin definir, para el proceso combinado |
| `VEC_BOLSA_INSCRIPCIONES_ENABLED` | `true` |
| `VEC_USUARIOS_PREFERENCIAS_ENABLED` | `true`, con la sesión externa ya compuesta |
| `VEC_EXTERNO_BOLSA_INSCRIPCIONES_LECTOR_DATABASE_URL` | LOGIN `vec_bolsa_inscripciones_lector` |
| `VEC_EXTERNO_BOLSA_DATABASE_URL` | LOGIN `vec_externo_bolsa_desarrollo` |
| `VEC_EXTERNO_PREFLIGHT_V3_DATABASE_URL` | LOGIN `vec_externo_preflight_v3_desarrollo`, sólo lectura |
| `VEC_EXTERNO_AUTORIZACION_FUENTE_DATABASE_URL` | LOGIN externo existente, rol `vec_autorizacion_fuente_externa` |
| `VEC_EXTERNO_AUTORIZACION_REGISTRO_DATABASE_URL` | LOGIN externo existente, rol `vec_autorizacion_registro_externo` |
| `VEC_EXTERNO_AUTORIZACION_MOTIVOS_DATABASE_URL` | LOGIN externo existente, rol `vec_autorizacion_motivos_externos` |

Los DSN usan TLS verificado. Ninguno se sustituye por el LOGIN interno de
RRHH. Se conservan el lector de RRHH, su fichero de identidad y los permisos
publicados según `LEEME.md` y `gobierno_inscripcion.md`.

El preparador existente exige un directorio externo distinto del interno.
Preparar allí el consumidor mediante `vec-server preparar-portal-externo
--material-externo "$VEC_MATERIAL_PORTAL_EXTERNO_DIR" --consumidores
inscripcion_bolsa`. Mantener en ese inventario los otros consumidores externos
que ya se utilicen. Antes de arrancar el proceso combinado, copiar sólo el
subárbol privado `externo/v3/` preparado a
`$VEC_DEVELOPMENT_MATERIAL_DIR/externo/v3/`, conservando los permisos privados
(directorios 0700 y ficheros 0600). No copiar material interno al directorio
externo ni guardar material o DSN en Git.

La raíz despacha únicamente las rutas de convocatorias abiertas y solicitudes
propias al montaje externo, fuera de la frontera interna de CT. Cada lectura
y presentación conserva las comprobaciones y la auditoría del constructor
existente. Una dependencia ausente impide el arranque; con inscripción apagada
no se abre el preflight. Los pools se cierran al fallar el arranque o al detener
el servidor.

Dirección comprueba en Chrome el recorrido final: el aspirante abre la
convocatoria v2, presenta su solicitud y recupera el recibo; RRHH ve esa misma
solicitud en «Solicitudes y admisión». La prueba focal de composición no
sustituye ese recorrido ni acredita sus tiempos HTTP.
