# Cartera del aspirante y consulta de títulos por RRHH

El aspirante abre **Mi Bolsa → Títulos y méritos**, registra el nombre, el tipo
y las fechas del mérito y recibe su comprobante. El registro conserva el estado
`declarado`; no acredita el título ni añade puntos. Un fallo de respuesta conserva
la misma operación para reintentarla, también al volver a abrir la pantalla.

RRHH abre **Bolsas → Revisión de méritos**, elige la convocatoria y la solicitud
y consulta la cartera de esa persona. Puede volver a la solicitud para decidir
la admisión con el circuito de inscripción existente. La consulta no verifica
documentos ni modifica el estado del mérito.

## Orden de instalación

AD256 de la PR #1079 debe preceder a AD253. Aplicar después, una sola vez,
[`lista_sql_codexm_meritos_basal_20261013.txt`](lista_sql_codexm_meritos_basal_20261013.txt)
y [`lista_sql_codexm_meritos_revision_rrhh_20261013.txt`](lista_sql_codexm_meritos_revision_rrhh_20261013.txt).
Las dos listas pertenecen a este corte. La antigua lista M4 de I conserva su
valor histórico y no sustituye este orden. Nunca aplicar DOWN ni repetir una
migración instalada.

La copia de ensayo usada es `/var/tmp/cidonia-copia-20261013/principal.sql`.
La instalación SQL no crea cuentas funcionales, asignaciones, motivos ni permisos.
Estos se publican y asignan mediante el gobierno ADMIN existente.

## Configuración del proceso externo

CONFIG NUEVA: `VEC_EXTERNO_MERITOS_OPERACION_DATABASE_URL`, conexión del LOGIN
`vec_externo_meritos_operacion`, con los grupos `vec_meritos_ejecutor` y
`vec_meritos_externo`, ambos `INHERIT TRUE`, `SET FALSE`, `ADMIN FALSE`.
Su contraseña y la conexión completa se conservan fuera de Git, en el material
privado de dirección. Debe usar la misma base que el resto del proceso externo,
TLS `verify-full` y las rutas de CA/certificados de ese proceso.

La cartera heredada M4 requiere también estas conexiones separadas:

| Variable | LOGIN | Grupo técnico |
| --- | --- | --- |
| `VEC_EXTERNO_MERITOS_CARTERA_DATABASE_URL` | `vec_externo_meritos_cartera` | `vec_meritos_cartera_externo` |
| `VEC_EXTERNO_MERITOS_FRONTERA_DATABASE_URL` | `vec_externo_meritos_frontera` | `vec_meritos_frontera_externo` |
| `VEC_EXTERNO_MERITOS_FUENTE_AUTORIZACION_DATABASE_URL` | `vec_externo_meritos_fuente_autorizacion` | `vec_autorizacion_fuente_meritos_externa` |
| `VEC_EXTERNO_MERITOS_REGISTRO_AUTORIZACION_DATABASE_URL` | `vec_externo_meritos_registro_autorizacion` | `vec_autorizacion_registro_meritos_externo` |
| `VEC_EXTERNO_MERITOS_MOTIVOS_AUTORIZACION_DATABASE_URL` | `vec_externo_meritos_motivos_autorizacion` | `vec_autorizacion_motivos_meritos_externo` |
| `VEC_EXTERNO_MERITOS_CONTEXTO_DATABASE_URL` | `vec_externo_meritos_contexto` | `vec_contexto_actor_v1_usuarios_externo` |

Cada LOGIN tiene sólo su grupo, sin privilegios ni ACL propias. Identidad y
revalidación usan las conexiones externas ya existentes; no usar cuentas internas.
La operación nueva no reutiliza el LOGIN de lectura de cartera.

El fichero privado `identidad/meritos-cartera-externa.json`, bajo
`VEC_DEVELOPMENT_MATERIAL_DIR`, contiene `version: 1`, `motivo_retirada`,
`motivo_declaracion` y `cuentas`. Cada cuenta conserva el certificado y la persona
de Usuarios, con otra `cuenta_ref` y otro `perfil_ref` aprobados para Méritos.
No inferir permisos del certificado ni copiar las asignaciones de Usuarios.

`motivo_declaracion` es una referencia publicada al catálogo
`motivos_meritos_declaracion`: `catalogo_id`, `catalogo_version`,
`catalogo_huella_sha256` y `entrada_clave`. `motivo_retirada` usa el catálogo
`motivos_autorizacion_meritos_cartera`. No escribir huellas inventadas en este fichero.

## Fuente de permisos y material

Publicar la fuente
[`fuentes/meritos_cartera_propia_v1.json`](fuentes/meritos_cartera_propia_v1.json)
con el mecanismo de fuentes de ADMIN existente. El rol fijo
`persona_meritos_cartera` contiene, en este orden, las tres concesiones de la fuente:

1. `meritos.cartera.consultar_propia`.
2. `meritos.cartera.retirar_propio`.
3. `meritos.hecho.declarar`.

Las tres usan el ámbito `persona_ref`. La tercera no concede acreditación,
rectificación ni acceso a otra persona. El rol anterior de dos concesiones
conserva lectura y retirada; no permite declarar.

Preparar la publicación del rol y su asignación personal por el circuito ADMIN
actual, con huella y CAS de la lectura vigente. Preparar después el material
con el consumidor existente `meritos_cartera` de `vec-server preparar-portal-externo`.
Ahora incluye las audiencias `vec_meritos.cartera.retirar_propio.v1` y
`vec_meritos.hecho.declarar.v1`; la segunda reutiliza el descriptor de declaración
ya definido por el núcleo. Las claves quedan sólo en el material privado.

## Lectura de RRHH

La revisión se monta junto a las inscripciones de Bolsa ya activadas. Reutiliza
`vec_bolsa_inscripciones_rrhh_lector` y su grupo existente, sin otro DSN.
B111 conserva la persona del detalle únicamente dentro del servidor; `json:"-"`
impide incluirla en la respuesta pública de inscripciones. RUM6 consulta sus
propias tablas y pide a Bolsa que revalide la relación y la autorización en la
misma transacción. AD255 registra la consulta nominal antes de confirmar y
entregar los datos. La web sólo envía la referencia de solicitud.

## Comprobación de dirección

Con una persona aspirante inscrita y sus perfiles aprobados:

1. Registrar una titulación desde Mi Bolsa y comprobar la fecha y el recibo.
2. Recuperar la misma operación: mismo mérito y recibo, sin otra alta.
3. Abrir su solicitud con RRHH, consultar los títulos y volver a la solicitud.
4. Comprobar la auditoría de ambas lecturas; una solicitud ajena no revela datos.
5. Cambiar ES/EN y comprobar que sólo se pide el catálogo del idioma activo.

La primera copia HZ24 no contenía solicitudes, hechos ni el LOGIN de lectura
RRHH: la instalación acreditada no sustituye este recorrido nominal. Dirección
hace la comprobación de pantalla según su orden del 13/10 a las 03:35.
