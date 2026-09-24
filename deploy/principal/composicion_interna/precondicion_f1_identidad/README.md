# Precondición de F1/Identidad 000004 en una base con historia

Este procedimiento acota el endurecimiento previo al selector corporativo F1 y a
Identidad `000004`. No se ejecutó contra la base principal. Las fuentes `.up.sql`
se leen por sus bytes canónicos, se despojan únicamente de `BEGIN` y `COMMIT`
externos y se ejecutan dentro de **una** transacción junto con el cambio de ACL.
No se invoca `DOWN`, ni se reaplica una migración instalada. El script falla si
ya encuentra selector o fachada.

## Preimagen y condiciones de entrada

Usar un acceso de superusuario PostgreSQL 18 autorizado, sin contraseñas en
argumentos. `--pg-container` usa `docker exec -i` contra un contenedor indicado;
sin esa opción usa `psql` local y el servicio PostgreSQL configurado por el
operador. `--report` es material privado con permisos `0600`; contiene nombres
de roles y objetos, nunca filas ni secretos.

```bash
SCRIPT=deploy/principal/composicion_interna/precondicion_f1_identidad/precondicion.py
python3 "$SCRIPT" --database "$BASE" --admin-user "$ADMIN" \
  --pg-container "$CONTENEDOR" --report "$MATERIAL/inventario.json"
```

El inventario guarda ACL PUBLIC de bases, esquemas, relaciones, funciones y
tipos; enumera las membresías que tocan propietario, migrador y runtime de
Contexto. El SHA256 informado fija esta preimagen. Antes del ensayo deben
existir exactamente **187** tipos de fila con USAGE de PUBLIC y exactamente
cinco membresías en ese subgrafo: la canónica
`vec_contexto_actor_v1_migrador → vec_contexto_actor_v1_propietario` y cuatro
concesiones de LOGIN de desarrollo, otorgadas por `postgres`, heredables,
sin `SET` ni administración. Cualquier diferencia exige revisar una nueva
preimagen y adaptar el cambio mediante revisión independiente; el script no
amplía el alcance automáticamente.

La transacción retira las cuatro concesiones de desarrollo y USAGE PUBLIC de
los 187 tipos de fila, ejecuta el selector y la fachada, restaura las cuatro
concesiones con sus opciones, compara todo el subgrafo con la preimagen y
comprueba privilegios efectivos del selector y del consumidor antes de COMMIT.
El primer recorrido termina con ROLLBACK y compara otra vez el inventario.

```bash
python3 "$SCRIPT" --mode rollback --database "$BASE" --admin-user "$ADMIN" \
  --pg-container "$CONTENEDOR" --report "$MATERIAL/ensayo.json" \
  --expected-inventory-sha256 "$INVENTARIO_SHA256"
```

## Restauración comprobada: condición previa al COMMIT

El [comentario del PR 29](https://github.com/aavidad/VEC_Diputacion_app/pull/29#issuecomment-5823727557)
documenta que una fila de `prueba_resultado_recibo_rrhh_v2` incumple su CHECK
actual al restaurar. Por tanto, el volcado disponible **no acredita una
restauración recuperable**. Con la evidencia actual, el script **no permite
COMMIT en la principal**: no existe acta válida de restauración para su
preimagen ni ensayo ROLLBACK completo. La reparación exacta se hará en la
autoridad CT después de diagnosticar la fila y el CHECK en el clon. No omitir
esa tabla, no deshabilitar CHECK, no usar
`session_replication_role`, no editar el dump ni borrar o reescribir el recibo
para hacer que `pg_restore` termine con cero. Eso destruiría la prueba de
recuperación y podría alterar historia.

Diagnosticar en la fuente mediante superusuario autorizado, en transacciones de
solo lectura, sin emitir contenido ni referencias originales:

```bash
python3 "$SCRIPT" --mode diagnose-check --database "$BASE" \
  --admin-user "$ADMIN" --pg-container "$CONTENEDOR" \
  --report "$MATERIAL/checks.json"
```

El informe contiene cada CHECK, su estado `convalidated`, el número de filas
que evalúan `FALSE` y hasta 20 huellas SHA256 de `acceso_ref`. Si la expresión
depende de una función que cambió después de registrar el recibo, comparar la
versión de función y la evidencia histórica con la fuente de la migración.
Resolver la discrepancia en la **autoridad CT** mediante un delta versionado y
revisado que preserve bytes, huellas, recibos e historia. La corrección exacta
depende del CHECK que falle y de la preimagen observada; este script no la
infiere ni la modifica. Repetir diagnóstico y obtener cero filas inválidas.

Después, repetir el inventario y el ensayo ROLLBACK sobre la preimagen ya
corregida. Hacer una copia nueva coherente del clúster, incluidos roles globales,
ACL, tipos de fila, datos, funciones y objetos; conservar huellas y recuentos
de la fuente. Restaurarla en PostgreSQL 18 aislado y vacío con `ON_ERROR_STOP`
y `pg_restore --exit-on-error`, sin filtrar errores. Comparar inventario de
roles/ACL/tipos, migraciones, recuentos y recibos/huellas antes de declarar
`verified=true`; repetir el diagnóstico CHECK en el destino y verificar cero
filas inválidas. El [manual de sistemas](../../../../docs/manual_sistemas/README.md)
describe el orden de globales y base. Secretos y claves se reponen por el canal
privado; la recuperación de aplicación exige también la prueba de lectura de
recibos y denegaciones tras reinicio, sin reemitir efectos.

Un acta privada revisada debe registrar, al menos:

```json
{
  "verified": true,
  "source_database": "nombre_de_base",
  "source_inventory_sha256": "sha256_del_inventario_revisado",
  "restore_exit_zero": true,
  "check_validated": true,
  "backup_sha256": "sha256_del_volcado_nuevo",
  "restore_manifest_sha256": "sha256_del_acta_completa_privada"
}
```

Los booleanos son **atestaciones del ensayo real**, no valores que genera el
script. El procedimiento solo coteja los campos mínimos y exige el acta
privada; Dirección y dos revisores E10 deben validar su evidencia. Si no se
puede restaurar exactamente la historia, no ejecutar el COMMIT.

```bash
python3 "$SCRIPT" --mode commit --database "$BASE" --admin-user "$ADMIN" \
  --pg-container "$CONTENEDOR" --report "$MATERIAL/confirmacion.json" \
  --expected-inventory-sha256 "$INVENTARIO_SHA256" \
  --rollback-report "$MATERIAL/ensayo.json" \
  --restoration-evidence "$MATERIAL/restauracion-verificada.json"
```

La copia y la restauración comprobada deben corresponder a la misma preimagen;
si cambia el inventario, repetirlas. Tras COMMIT, el informe confirma que las
cuatro membresías volvieron exactamente, no queda USAGE PUBLIC en tipos fila y
selector/fachada existen. Una versión exacta modificada del procedimiento o de
sus migraciones reabre las dos revisiones E10.

## Ensayo disponible

El diagnóstico y rechazo por preimagen se ejecutaron en PostgreSQL 18.4
desechable `vec_precondicion_v3` del contenedor
`vec-composicion-v3-pg18-20260925`: inventario JSON correcto; un CHECK
sintético dependiente de función detectó 1 fila inválida al cambiar la
definición; el ensayo de migraciones se rechazó porque faltan los roles reales,
sin crear selector ni fachada. Esto comprueba la detección y el cierre por
preimagen, pero no acredita el ROLLBACK completo sobre el clon ni la
restauración del volcado de la principal.

Como prueba positiva **solo sintética**, se creó una función CHECK `v2` que
acepta la fila histórica sin modificarla, se sustituyó el CHECK en una
transacción, se generó un dump custom del esquema y se restauró tras borrar
ese esquema en la misma base desechable. `pg_restore --exit-on-error` terminó
con código 0. Antes y después había 1 fila y la huella agregada fue idéntica:
`9b36d7a935b1535fd975ae737a0251127a54b6d221eabeb9e9ce9d6700ae78f4`.
El dump sintético tuvo SHA256
`70fb90424a296e2e4ebcf6b7ad9eee760b8f5d46809efc53abd6c1a96a6b86c2`;
ambos CHECK restaurados dieron 0 filas inválidas. Este ensayo muestra una vía
de corrección que conserva la fila; no determina que el defecto real tenga esa
causa ni sustituye la restauración integral de globals, ACL, secretos e historia.
