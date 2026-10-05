# Mantenimiento del perfil fijo para el plan de firma — AUT51

AUT51 prepara el paso del perfil fijo de Aplicación de Rol6 a Rol7. Rol7 es Rol6
más cuatro concesiones: `vec.catalogos.crear`, `actualizar`, `publicar` y
`retirar`. Son las que necesita el gobierno del plan nominal de firma (AD177).
Instalar la migración no publica Rol7 ni cambia asignaciones. Eso lo hace
después `vec-mantener-admin-fijo` con un plan aprobado de versión 3, igual que
AUT42 hizo con Rol5 y AUT45 con Rol6.

## Qué concede y qué no

Las cuatro concesiones van con módulo `contratacion_temporal`, tipo
`catalogo_configurable`, finalidad `gestionar_contratacion_temporal`, garantía
`alto` y sin campos ni obligaciones. Coinciden con lo que comprueba AD177. Su
catálogo está en `data/catalogos/administracion/acciones_plan_firma_v1.json`.

La categoría de Aplicación (AUT48) sólo acredita la concesión exacta. Una acción
general de catálogos con otro módulo o con otro tipo de recurso no pasa: lo
comprueba la prueba positiva del clon. Además, AD177 exige la audiencia
`vec_catalogos_configurables.plan_nominal_firma.gobierno.v1`, de modo que una
decisión para otro catálogo no se consume en el gobierno del plan.

## La puerta del lote

`acreditar_perfil_aplicacion_lote_ordinario_v1` (AUT45) aceptaba sólo Rol6.
Con Rol7 publicada, el lote ordinario (AD190/AUT44) habría dejado de acreditar.
AUT51 la reescribe como AUT48 hizo con las otras cuatro: acepta la versión que
señala la asignación actual, siempre que esa versión contenga la concesión del
lote exacta, en el rol y en el catálogo, con su obligación de auditar. Antes de
sustituirla, la migración comprueba que la puerta instalada es exactamente la de
AUT45 (huella del cuerpo).

## Operación

Igual que AUT45: LOGIN técnico propio con una sola pertenencia al grupo
`vec_admin_mantenimiento_plan_firma_ejecutor`, configuración externa vigente del
DBA y huellas del plan, catálogo y preimagen. Se registran la confirmación y cada
intento en la auditoría común (AD183). El replay devuelve el mismo recibo; un
acuse de Rol6 no sirve para Rol7. Las asignaciones pasan de la revisión 3 a la 4
sin tocar personas, perfiles, ámbitos ni el Hasta.

## Instalación

`deploy/principal/lista_sql_claude_aut51_plan_firma_20261005.txt`, una vez y sin
DOWN, después de AUT45 y AUT48. Luego, con la ventana de la aprobación, preparar
el plan versión 3 y aplicarlo con la CLI. Si el programador de administración ha
instalado antes otra versión del rol, el plan detecta la preimagen distinta y se
para sin escribir.

## Ensayo del 5 de octubre de 2026

Clon desechable de la copia fría H10-30 con las listas de main hasta AD177
(AD194/IS16/CA36, AUT47, AD193, AUT48, AD195 y Personal36, AD196 y AUT49,
AD178 y AD177), PostgreSQL 18.4 con 2 GB, arranque 2+1 real y vec-admin de la
rama. Resultado:

| Paso | Resultado |
| --- | --- |
| Rol5 → Rol6 (AUT45) por la CLI, y replay | confirmado; replay con el mismo recibo |
| Huella de la puerta del lote antes de AUT51 | coincide con la de AUT45 (`f6cbcb22…`) |
| AUT51 (SHA256 `f23e8dd6…`) y su prueba estructural | salida 0; `AUT51-ESTRUCTURA-OK`; la poscondición de la puerta del lote pasa |
| Puerta del lote nueva con Rol6, antes de publicar Rol7 | acredita una decisión real de vec-admin en Rol6 |
| Segunda aplicación de AUT51 | se para en `dependencias` sin cambios |
| Rol6 → Rol7 por la CLI (plan versión 3), y replay | confirmado; replay con el mismo recibo |
| Lista y ficha de usuarios en vec-admin con Rol7 | 200 para las dos personas |
| Prueba positiva con una decisión real de vec-admin | la puerta del lote acredita Rol7 y no Rol6; la categoría acredita `vec.catalogos.publicar` y rechaza otro módulo, otro tipo y otra acción; AD177 acepta el gobierno sin sustituir la categoría |
| Aprobación divergente desde el LOGIN operador | `denegado` con su intento en la auditoría común |
| Reinicio de PostgreSQL y vec-admin | usuarios en 200; replay de Rol7 con el mismo recibo |

La prueba de AUT48 sigue pasando. El catálogo de Rol7 conserva las entradas de
Rol6 en su revisión siguiente y añade las cuatro de catálogos en la revisión 1.

## Límite que hay que respetar en el futuro

Con Rol7, la categoría de Aplicación acredita `vec.catalogos.*` sobre cualquier
`catalogo_configurable` de Contratación temporal: AUT48 no mira qué catálogo es.
Hoy sólo lo consume AD177, que exige la audiencia y el esquema del plan de firma.
Cualquier fachada futura de catálogos que use esta categoría tiene que exigir su
propia audiencia y su `catalogo_id`; si no, el administrador de Aplicación podría
gobernar otros catálogos del módulo.
