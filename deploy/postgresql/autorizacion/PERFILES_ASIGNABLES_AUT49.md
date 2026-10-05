# Perfiles asignables — AUT49 y AD196

AUT49 permite registrar versiones de rol ordinarias, ya publicadas, como
perfiles que la persona administradora podrá asignar después desde la
pantalla. Instalar AUT49 no registra ningún perfil, no concede permisos y no
crea asignaciones. AD196 añade a la auditoría común los dos tipos de registro
de esta operación.

## Qué comprueba

Por cada perfil del plan:

- la versión del rol existe, está publicada y su huella es la del plan;
- su control de vigencia actual está habilitado, con la revisión y la huella
  del plan;
- no es administración ni Sistemas; ninguna versión del mismo rol es sensible,
  fija, de otra clase administrable ni tiene asignaciones externas;
- no es de Intervención ni de fiscalización (por identificador, nombre,
  módulo, acción o finalidad) ni de aspirantes o usuarios externos (rol
  `candidato_…`, identificador con «extern», módulo `aspirantes`);
- ninguna concesión es del módulo `administracion`;
- los ámbitos fijos son válidos (al menos uno, normalmente `organizacion_ref`)
  y no incluyen `unidad_ref`; cuando `unidad_requerida` es verdadero, la unidad
  la aporta cada asignación;
- la vigencia es finita y termina en el futuro; la duración propuesta está
  entre 60 segundos y un año;
- no estaba registrado antes.

Estas exclusiones son una defensa técnica por nombres. La clasificación
positiva es la aprobación: Alberto revisa la lista exacta del plan antes de
firmar su huella.

Hasta que se responda la pregunta 141 de `dudas.md`, ningún plan incluye
`rol:organizacion_preparacion` (modifica la estructura organizativa de
Personal) ni `rol:entrega-peticion-rrhh-fijo` (entrega peticiones de centro y
crea solicitudes de Contratación). El operador los quita al preparar el plan y
quien aprueba rechaza cualquier plan que los contenga. No se excluyen en SQL
porque la decisión puede ser incluirlos; si la respuesta es que no, una
migración posterior los añadirá a las exclusiones.

El plan es el texto canónico de `jsonb` (sin claves repetidas), tiene de 1 a 32 perfiles sin repetir, caduca como mucho un día después
de prepararse y su huella debe coincidir con la que aprobó Alberto. Cualquier
fallo deja la operación sin efecto.

## Perfiles que podrá dar el lote de la pantalla

El efecto del lote (AUT44) sólo asigna un perfil registrado con la audiencia
`vec_autorizacion.administracion_perfiles.lote_ordinario.v1` y con
`ambitos_fijos` exactamente igual a
`[{"clave":"organizacion_ref","valores":["<organización configurada en vec-admin>"]}]`.
La unidad la añade cada asignación. Un perfil registrado con otros ámbitos fijos
queda fuera del lote (42501); para usarlo hay que registrar otra versión del rol.

## Procedimiento

1. **Plan.** El operador prepara `plan.json` con una consulta de sólo lectura
   sobre `version_rol` y `control_vigencia_version_rol` (modelo en el ensayo de
   `docs/plan_modulos/administracion.md`). Formato:
   `{"esquema":"vec.admin.perfiles-asignables.plan.v1","operacion_ref":"rpa_…","preparado_en":"…Z","caduca_en":"…Z","perfiles":[{"version_rol_ref","version_rol_sha256","control_revision","control_sha256","unidad_requerida","ambitos_fijos","vigente_desde","vigente_hasta","duracion_propuesta_segundos"}]}`.
   El fichero no lleva salto de línea final; su SHA256 es la huella del plan.
2. **Aprobación.** Alberto aprueba esa huella por escrito. El texto de la
   aprobación queda en un fichero privado 0600 y se anota su SHA256.
3. **DBA.** Como superusuario: crea un LOGIN nuevo para esta operación, lo hace
   miembro de `vec_admin_perfiles_asignables_ejecutor` con
   `WITH INHERIT TRUE, SET FALSE, ADMIN FALSE` y sin otros permisos, e inserta la
   fila de `vec_autorizacion.config_perfiles_asignables_admin_v1` con el LOGIN,
   la huella del plan, la referencia y la huella de la aprobación, entorno
   `desarrollo` y una ventana corta (dos horas; la tabla no admite más de un día).
4. **Aplicar.** Con ese LOGIN, por TLS `verify-full`:

   ```sql
   \set plan `cat plan.json`
   BEGIN ISOLATION LEVEL SERIALIZABLE;
   SET LOCAL timezone='UTC';
   SET LOCAL statement_timeout='30s';
   SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan','<huella del plan>');
   COMMIT;
   ```

   La respuesta trae `estado` (`permitido`, `denegado` o `error`), el recibo y la
   referencia del intento auditado. No se muestran causas detalladas. La
   fachada espera como mucho 5 s por un bloqueo; el tope total de 30 s lo pone
   el `SET LOCAL statement_timeout` de la sesión (el de la función no actúa en
   una llamada anidada). Si dos
   operadores coinciden, PostgreSQL puede abortar con 40001 sin respuesta: se
   repite el mismo paso, que es idempotente.
5. **Replay.** Repetir el mismo paso devuelve el mismo recibo con
   `replay: true` y añade sólo un intento a la auditoría. Sirve para recuperar
   un COMMIT dudoso.
6. **Cierre.** Terminada la ventana, `ALTER ROLE <login> NOLOGIN`. La fila de
   configuración es inmutable; otro plan necesita otro LOGIN.

## Lo que queda registrado

- `rol_administrable_exacto_v1`: una fila `ordinario` por perfil, inmutable.
- `registro_perfiles_asignables_admin_v1`: el plan exacto, el LOGIN, el recibo
  y la referencia de auditoría.
- Auditoría común: un registro `perfiles_asignables_admin` con la huella del
  plan, la de la aprobación, la operación y la huella de la lista; y un `intento_perfiles_asignables_admin`
  por cada llamada (registrado, replay, denegado o error).

Un perfil registrado no se borra. Para dejar de ofrecerlo se retira su versión
de rol por el control de vigencia, que ya hace que `resolver_rol_administrable_v1`
deje de devolverla.
