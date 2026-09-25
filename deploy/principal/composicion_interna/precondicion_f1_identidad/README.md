# Precondición de F1/Identidad 000004 en una base con historia

Este procedimiento acota el endurecimiento previo al selector corporativo F1,
ContextoActor `000003` y `000004a`, e Identidad `000004`. Se ensayó en un clon
exacto de la principal (`vec-clon-f1-*`, 25/09/2026) con ROLLBACK y COMMIT; no
se ejecutó contra la base principal. Las fuentes `.up.sql` se leen por sus
bytes canónicos, se despojan únicamente de `BEGIN` y `COMMIT` externos y se
ejecutan dentro de **una** transacción junto con el cambio de ACL. No se invoca
`DOWN`, ni se reaplica una migración instalada. El script falla si ya encuentra
selector o fachada.

ContextoActor `000004` no se instala: su guarda exige una única huella del
manifiesto del predecesor (`bddc5574…`) que no reproduce ninguna base actual
(base nueva `613d1837…`, principal `6fa63aa7…` porque `000005` se instaló antes).
`000004a` es su corrección con la misma definición y una lista cerrada de esas
tres huellas; `000004` conserva sus bytes.

## Preimagen y condiciones de entrada

Usar un acceso de superusuario PostgreSQL 18 autorizado, sin contraseñas en
argumentos. `--pg-container` usa `docker exec -i` (o `podman` con
`--container-engine podman`) contra un contenedor indicado; sin esa opción usa
`psql` local y el servicio PostgreSQL configurado por el operador. `--report`
es material privado con permisos `0600`; contiene nombres de roles y objetos,
nunca filas ni secretos.

```bash
SCRIPT=deploy/principal/composicion_interna/precondicion_f1_identidad/precondicion.py
python3 "$SCRIPT" --database "$BASE" --admin-user "$ADMIN" \
  --pg-container "$CONTENEDOR" --report "$MATERIAL/inventario.json"
```

El inventario guarda ACL PUBLIC de bases, esquemas, relaciones, funciones y
tipos (los arrays automáticos no tienen ACL propia y no se cuentan); enumera
las membresías que tocan propietario, migrador y runtime de Contexto. El
SHA256 informado fija esta preimagen. El número de tipos de fila con USAGE de
PUBLIC lo fija ese inventario revisado (227 en la principal el 25/09/2026); no
puede haber PUBLIC en dominios, enumerados u otros tipos. En el subgrafo debe
haber exactamente cinco membresías: la canónica
`vec_contexto_actor_v1_migrador → vec_contexto_actor_v1_propietario` y cuatro
concesiones de desarrollo otorgadas por `postgres`, heredables y sin
administración (la del LOGIN de gobierno AD3 conserva `SET`; una de las cuatro
es un rol sin LOGIN). Cualquier otra diferencia exige revisar una nueva
preimagen y adaptar el cambio mediante revisión independiente; el script no
amplía el alcance automáticamente.

La transacción retira las cuatro concesiones de desarrollo y USAGE PUBLIC de
los tipos de fila, ejecuta el selector, ContextoActor `000003` y `000004a` e
Identidad `000004`, restaura las cuatro concesiones con sus opciones, compara
todo el subgrafo con la preimagen y comprueba privilegios efectivos del
selector y del consumidor. Captura además el inventario de ACL y punteros
dentro de la misma transacción: antes del COMMIT exige que la postimagen
completa coincida con la preimagen excepto por la retirada de PUBLIC en los
tipos de fila y la aparición del selector/fachada. Si falla cualquiera de
esas comparaciones, PostgreSQL revierte la transacción. El primer recorrido
termina con ROLLBACK y compara otra vez el inventario.

```bash
python3 "$SCRIPT" --mode rollback --database "$BASE" --admin-user "$ADMIN" \
  --pg-container "$CONTENEDOR" --report "$MATERIAL/ensayo.json" \
  --expected-inventory-sha256 "$INVENTARIO_SHA256"
```

## Restauración comprobada: condición previa al COMMIT

**Diagnóstico (25/09/2026, solo lectura sobre la principal).** No es una fila:
son **420** de las 1845 filas de `vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2`,
todas de tipo `detalle` y registradas entre 2026-09-05 y 2026-09-18T00:15Z,
antes de CT `000106`. Esa migración añadió cuatro atributos
(`fiscalizacion_presente`, `fiscalizacion`, `referencia_fiscalizacion`,
`referencia_subsanacion`) al tipo `entrada_detalle_expediente_rrhh_v1` y
redefinió `canon_contenido_detalle_rrhh_v1`, que los exige. PostgreSQL no
revalida filas existentes al cambiar una función de un CHECK: en la principal
`check1` figura `convalidated` pero sus filas antiguas lanzan
`contenido de detalle RRHH inválido` al evaluarlo, de modo que ninguna
restauración lógica puede cargarlas. Las 1255 filas de `cuadro` y las 170 de
`detalle` posteriores cumplen todos los CHECK. El modo `diagnose-check`
evalúa cada fila en su propia subtransacción y cuenta también las que lanzan
excepción.

**Resolución adoptada (orden del operador: documentar o excluir la sonda en la
restauración, sin alterar la principal).** La copia de seguridad conserva la
tabla completa (`pg_dump -Fc`, SHA256 de sus datos en el acta). La restauración
comprobada excluye solo la entrada `TABLE DATA` de esta tabla de la lista de
`pg_restore`, se ejecuta con `--exit-on-error --single-transaction` y después
carga fila a fila, con los CHECK vigentes, las filas que los cumplen; las que no
quedan solo en la copia y el acta registra su número, intervalo, causa y las
huellas SHA256 de su `acceso_ref`. No se deshabilitan CHECK ni disparadores,
no se usa `session_replication_role` ni se edita la copia. La corrección
definitiva (un delta CT que acepte la forma anterior a `000106` sin reescribir
recibos) queda para la autoridad CT.

Además del volcado, `pg_dumpall` omite ACL explícitas que coinciden con
`acldefault`: en relaciones «solo propietario» (110 el 25/09/2026) y en tipos
(68, que en tipos significan **PUBLIC revocado**). La restauración las reaplica
leídas de la fuente, igual que las ACL y ajustes de la base `postgres`. El
contraste posterior compara 13185 líneas de catálogo y datos (esquemas,
relaciones, columnas, restricciones, funciones con su definición, tipos,
políticas, disparadores, roles, membresías con otorgante, ACL por defecto, base,
secuencias y, por tabla, recuento y huella de contenido). Las únicas
diferencias admitidas son la sonda y 30 CHECK cuyo texto cambia solo de
parentización de `AND` al reanalizarse (idénticos sin paréntesis).

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
si cambia el inventario, repetirlas. Las comprobaciones de ACL, membresías,
privilegios efectivos y punteros se cierran **antes de COMMIT**; el informe
posterior es una auditoría adicional. Un acuse de COMMIT seguido de fallo de
lectura o diferencia posterior produce `resultado: indeterminado` y salida 2,
sin describir el efecto como rechazado: detener reintentos y reconciliar en
solo lectura la presencia de selector/fachada, ACL y membresías. Una versión
exacta modificada del procedimiento o de sus migraciones reabre las dos
revisiones E10.

## Procedimiento validado para la principal (no aplicado)

Validado de principio a fin el 25/09/2026 en el clon `vec-clon-f1-*` de
cidonia, creado desde la principal `vec-postgresql-20260906` con la copia y la
restauración anteriores (inventario F1 idéntico en ambas:
`20cb81339997e9f0c987a7883ddb88665ec409dedfcb6de748e5e830334c81c8`). Requiere la
autorización expresa de Alberto; ningún paso se ha ejecutado en la principal.
Todo se hace como `postgres` por el socket del contenedor, en serie, y cada
fichero se ensaya antes con su única línea `COMMIT;` cambiada por `ROLLBACK;`.

1. **Copia y restauración comprobada** en un clon nuevo, según la sección
   anterior; escribir `restauracion-verificada.json` (acta mínima) y el acta
   completa privada. Si la principal cambia después, repetir.
2. **Huellas CT de referencia**: con el binario y la configuración de la
   principal contra el clon, `POST cuadro/consultas` (límite 100) y
   `expedientes/consultas` del primer expediente. En el clon: `200/200`,
   huellas normalizadas `9001f81c…`/`b7354636…`. Tomar la misma pareja en la
   principal antes y después.
3. **Inventario**: `--mode inspect`; el SHA256 debe ser el revisado. Si no,
   parar: la preimagen cambió.
4. **Precondición F1** (una transacción): `--mode rollback` y, con su informe y
   el acta de restauración, `--mode commit`. Aplica selector, ContextoActor
   `000003`, `000004a` e Identidad `000004`, retirando y restaurando las cuatro
   membresías y retirando USAGE de PUBLIC en los 227 tipos fila. En el clon:
   `ensayo revertido` y `confirmado`, postinventario
   `8d1ba3288145da8b59a1cbb8f83df9d5d5527b2d1020092aecfa7bc36225cad3`.
5. **Deltas de composición** con `../instalar_esquema.py` (ROLLBACK y COMMIT
   en una transacción): ContextoActor `000006`, Identidad `000006`, Personal
   `000010a`, CT identidad `000002` y AD3 `000050a`.
6. **Resto, fichero a fichero** (ensayo y COMMIT): ContextoActor `000007`,
   AD3 `000053a` y CT `000109`. AD3 `000053a` es la lectura de configuración
   interna, renumerada porque `000053` ya es de Cronos.
7. **Aprovisionamiento** de `vec-interno` con `../aprovisionar.py` fuera de
   Git (orden probado: `init-ca`, tokens PKCS#11 de HMAC y de la persona,
   `hmac-token`, `create-csr`, `issue-person`, `register-person`; registro F1
   del perfil propio de vec-interno, su vínculo de contexto, la organización
   corporativa `org_…` y el vínculo corporativo `interna_corporativa`/
   `consulta_rrhh` de la persona RRHH ya registrada en F1; `register-context`
   con el ámbito CT `organizacion:…`; `policy`; `alias-hmac` con
   `--corporate-organization-ref`; `roles`, `identity-roles`, `v3-roles`;
   **después** Autorización `000014` (exige el LOGIN de la fuente V3) y
   `vec-publicar-permiso-interno` con `--organizacion` (CT) y
   `--organizacion-corporativa`. El registro F1 del perfil y los vínculos no
   tiene todavía herramienta versionada.
8. **Comprobación**: huellas CT iguales a las del paso 2 con el binario de la
   principal y con el de la rama; arrancar `vec-interno` y consultar el
   seguimiento con el certificado personal; repetir tras reiniciar aplicación
   y PostgreSQL.

| Paso | Fichero (`deploy/postgresql/…`) | SHA256 |
|---|---|---|
| 4 | `contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql` | `d8a94def…fa62b802` |
| 4 | `contexto_actor_v1/migraciones/000003_organizacion_corporativa_v1.up.sql` | `6f6e7682…083901f6` |
| 4 | `contexto_actor_v1/migraciones/000004a_vinculo_corporativo_rrhh_v1.up.sql` | `ca969bc9…a3cabaa0` |
| 4 | `identidad_sesiones_v1/migraciones/000004_revalidacion_contexto_corporativo_rrhh_v1.up.sql` | `83130607…025a3d90` |
| 5 | `contexto_actor_v1/migraciones/000006_vinculos_efectivos_temporales.up.sql` | `e70043d8…57f1e3c0` |
| 5 | `identidad_sesiones_v1/migraciones/000006_politica_certificado_personal_desarrollo.up.sql` | `38ed28d8…6078f46d` |
| 5 | `personal/migraciones/000010a_lectura_incorporacion_certificado_desarrollo.up.sql` | `6c2b308c…fbc6de11` |
| 5 | `contratacion_temporal/migraciones_identidad/000002_consulta_rrhh_certificado_desarrollo.up.sql` | `969d37a0…228ec0a7` |
| 5 | `autorizacion_atestada_v3/migraciones/000050a_preflight_material_interno.up.sql` | `542b6629…d661119c` |
| 6 | `contexto_actor_v1/migraciones/000007_alcance_proyecciones_empleado.up.sql` | `6b201fdd…91bd5b3c` |
| 6 | `autorizacion_atestada_v3/migraciones/000053a_lectura_configuracion_interna.up.sql` | `ce2e2e2c…315d2d0c` |
| 6 | `contratacion_temporal/migraciones/000109_consulta_resumen_seguimiento.up.sql` | `fb18461b…32363526` |
| 7 | `autorizacion/migraciones/000014_perfil_interno_certificado.up.sql` | `1f46c4c8…51e53b4b` |

El núcleo AD3 (`md5` de `consumir_decision_mutacion_v3_interna`) no cambia en
ningún paso (`a5ef3e6f…`). Con el binario actual de la principal, CT sigue
igual tras los pasos 4–7; el binario de la rama también. No aplicar
ContextoActor `000006` antes que `000003`/`000004a` ni instalar `000004`.

## Ensayos sintéticos anteriores

Anteriores al ensayo en el clon exacto descrito arriba; se conservan como
evidencia de las guardas.

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

La comparación transaccional del inventario también se probó en PostgreSQL
18.4 con dos instantáneas sintéticas: una transición exacta emitió
`VERIFICACION_PRECIERRE_OK` dentro de ROLLBACK; una membresía distinta provocó
error antes del cierre. Una prueba focal simula el acuse de COMMIT y un fallo
de lectura posterior y comprueba el informe `indeterminado`.
