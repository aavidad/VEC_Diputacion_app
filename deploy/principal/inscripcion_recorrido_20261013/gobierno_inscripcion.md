# Publicar los permisos de inscripción

La inscripción amplía dos roles existentes con las ocho entradas de
`inscripcion_bolsa_actos_v1.json`. Cada propuesta conserva las concesiones y
asignaciones anteriores, y liga las preimágenes actuales. Dos personas ADMIN
distintas proponen y aprueban; el arranque no publica permisos.

## RRHH

Reutilizar el montaje B103 de `vec-admin` y
`VEC_ADMIN_VERSION_BOLSA_B103_CONFIG_FILE`, con su LOGIN exclusivo. Publicar
primero la fuente mediante el circuito normal de catálogo. Después leer la
cabeza vigente de `tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo` y
preparar las selecciones exactas en este orden:

1. `accion:bolsa.inscripcion.rrhh.motivos`.
2. `accion:bolsa.inscripcion.rrhh.decidir`.
3. `accion:bolsa.inscripcion.rrhh.incorporar`.

Las propuestas y cierres reutilizan respectivamente
`/api/admin/perfiles/v1/gobierno-version-bolsa-b103/propuestas` y `/cierres`.
El cuerpo de propuesta mantiene `operacion_ref`, referencia/versión/huella
del catálogo, `base_ref`, `base_huella_sha256`, revisión/huella del control,
las tres `selecciones`, las `asignaciones`, `motivo` y `referencia_acto`.
Cada selección conserva `entrada_ref`, `entrada_version` y
`entrada_huella_sha256` de la publicación admitida.

RRHH conserva `unidad_ref` y `ambito_ref`. Presentar pertenece al aspirante;
no se añade al rol RRHH. Un cierre incierto se recupera con la misma referencia
y material, sin otra propuesta ni otras asignaciones.

## Aspirante

Reutilizar el montaje externo de gobierno y
`VEC_ADMIN_VERSION_BOLSA_EXTERNA_CONFIG_FILE`. Su LOGIN es diferente del de
B103 y conserva la membresía exclusiva
`vec_admin_version_rol_bolsa_externa_ejecutor`. El rol es
`candidato_bolsa_portal_historial_propio_desarrollo`; leer su cabeza actual,
que puede incluir ya las preferencias, y seleccionar en este orden:

1. `accion:bolsa.inscripcion.convocatorias.listar`.
2. `accion:bolsa.inscripcion.convocatoria.consultar`.
3. `accion:bolsa.inscripcion.propias.listar`.
4. `accion:bolsa.inscripcion.propia.consultar`.
5. `accion:bolsa.inscripcion.presentar`.

Las rutas son
`/api/admin/perfiles/v1/gobierno-version-bolsa-externa/propuestas` y `/cierres`.
Mantener el formato de propuesta/cierre del gobierno externo existente,
incluida la huella del plan esperado calculada desde el dominio y la misma
lectura de la cabeza. Conservar el único ámbito `candidato_ref` de cada
asignación. El perfil de Antonio se conserva; no crear otra identidad.

Las ventanas y los campos de cada concesión proceden de la fuente admitida.
No mezclar las selecciones de RRHH, preferencias y aspirante en un mismo plan.
Si cambia el catálogo, la cabeza, el control o una asignación, preparar un plan
nuevo con las preimágenes actuales antes del primer acto.

## Material de presentación y arranque

Desde el proceso interno, con su material privado vigente, preparar el
consumidor externo que ya conoce `vec-server`:

```sh
vec-server preparar-portal-externo \
  --material-externo "$VEC_MATERIAL_PORTAL_EXTERNO_DIR" \
  --consumidores inscripcion_bolsa
```

El directorio es el material privado del portal externo; esta variable es un
argumento del guion, no una opción nueva de servidor. La audiencia registrada
es `vec_bolsa_llamamientos.inscripcion.presentar.v1`. RRHH reutiliza las
audiencias internas `vec_bolsa_llamamientos.inscripcion.revisar.v1` y
`vec_bolsa_llamamientos.inscripcion.incorporar.v1` del preparador de material
CT existente. Sus LOGIN de ejecución son los de Bolsa ya instalados.

Después de los cierres y de la configuración indicada en `LEEME.md`, dirección
enciende las dos superficies y prueba: Antonio presenta; RRHH abre la misma
solicitud, revisa sus requisitos y registra la admisión. La incorporación
conserva la selección de acta, participación y recibo; una admisión por sí
sola no constituye otra bolsa ni fabrica un acta.
