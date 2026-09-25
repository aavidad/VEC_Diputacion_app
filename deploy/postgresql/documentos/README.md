# Documentos comunes B5: persistencia

Este módulo conserva metadatos, numeración interna, referencias a originales
inmutables, instantánea de conservación, preparaciones de notificación,
auditoría y outbox. Los bytes originales permanecen en `AlmacenObjetos` y el
adaptador comprueba su recibo antes de confirmar el documento. El número
`VEC-AAAA-N` no es un asiento de registro general.

## Orden de instalación

AD3-60 y AD3-62 no dependen de AD3-53–59, 61, 69, 70 ni 80: se ensayaron
instaladas antes y después de ellas sobre el núcleo AD3 real.

1. `roles_up.sql` como DBA, una sola vez.
2. `../autorizacion_atestada_v3/migraciones/000060_documentos_comunes.up.sql` por su migrador.
3. `migraciones/000001_documentos_comunes.up.sql` por el migrador documental.
4. `../autorizacion_atestada_v3/migraciones/000062_replay_documentos_comunes.up.sql`.
5. `migraciones/000002_replay_autorizado.up.sql`.
6. `migraciones/000003_custodia_externa.up.sql`, en la misma ventana y antes
   de la primera alta: su precondición exige que la numeración interna no se
   haya usado, porque el registro de identificadores parte vacío.
7. Crear fuera del repositorio el LOGIN de aplicación con **solo** la membresía
   `vec_documentos_ejecutor`; no conceder propiedad, migración ni acceso a tablas.

AD3-60 y AD3-62 no se han instalado nunca en ninguna base: el 25/09/2026 se
ampliaron en su sitio con la acción `documentos.externo.registrar`. La AD3-61
de `main` es de Dietas (competencias y rectificación) y no guarda relación
con este módulo.

No reaplicar ni ejecutar `DOWN` sobre bases con historia. Cada fachada exige
transacción `SERIALIZABLE` y material V3 nominal. El alta inicial consume la
decisión en el mismo `COMMIT` que estado, auditoría y outbox. Un replay exacto
puede recibir `consumo_nuevo=false` solo mientras sigan vigentes capacidad,
clave, configuración, raíz y decisión; se cotejan principal, decisión y
auditoría originales, preimagen y recibo objeto. No crea otro documento ni
outbox. El material V3 original caduca como máximo a los cinco segundos: una
recuperación posterior exige una **decisión V3 fresca**, ligada a la misma
preimagen, recurso, principal y clave idempotente. Esta decisión produce su
propio consumo y auditoría de autorización; el documento, número, recibo y
outbox originales permanecen intactos. La consulta exige consumo nuevo y
registra el acceso antes de resolver el objeto por su referencia y versión
inmutables.

## Custodia externa

Cuando el original ya lo custodia otro sistema (justificantes de Dietas o de
Bolsa, correos, un gestor documental), VEC registra solo su referencia opaca
en el custodio (`custodio_id` + `custodia_ref`), la huella SHA-256 y los
metadatos gobernados: tipo documental, versión y política de conservación.
MIME y tamaño son opcionales porque no todos los custodios los declaran. La
tabla `referencia_externa` no tiene columnas de objeto: no se sube contenido
y la descarga sigue limitada a originales custodiados por VEC. Un mismo
identificador no puede nombrar a la vez un original y una referencia externa
(`identificador_documental`). La lista v2 devuelve ambos tipos con el campo
`custodia` (`vec` o `externa`), con el mismo cursor.

## Ensayos

`probar_integracion_pg18.sh` crea y destruye su propio contenedor PostgreSQL
18.4. Comprueba `ROLLBACK`/`COMMIT`, RLS forzada y ACL. Prueba el replay
inmediato con el mismo material dentro de su TTL y, después de reiniciar y
dejarlo caducar, una decisión sintética nueva con idénticos efecto y clave:
devuelve el mismo recibo, mantiene un documento y un outbox, y registra dos
consumos de autorización distintos. Hace lo mismo con un registro de custodia
externa (replay; clave reutilizada, identificador compartido con un original,
referencia con ruta y finalidad ajena rechazados) y con la lista v2 paginada
sobre ambas custodias. Al final ejecuta el repositorio Go (pgx) contra esa base
con el LOGIN ejecutor para cotejar preimagen, proyección y lista v2
(`VEC_DOCUMENTOS_SIN_GO=1` lo omite). Corre sobre una **preimagen sintética** AD3-50 seguida
de AD3-51/52/60/62 reales, y sustituye en esa base las fachadas AD3 por
recibos sintéticos: no acredita COSE.

`probar_cadena_real_pg18.sh BASE.dump ROLES.sql [main-60|60-main]` restaura
una base VEC sintética con el núcleo AD3 real (la misma preimagen del ensayo
Dietas 000008) e instala encima la cadena Dietas, AD3-53/54/70/80 y las
migraciones documentales en los dos órdenes posibles. Comprueba cada
exclusión del núcleo una sola vez, la fachada AD3-60 real con el LOGIN exacto
hasta la clave de capacidad (alta y externa), el rechazo con dos grupos o con
login ajeno, RLS/ACL y la persistencia tras reiniciar. Tampoco acredita COSE:
la base no tiene clave publicada para `vec_documentos.operacion.v1`.

`estado_firma` permanece `pendiente_proveedor` e inmutable. La integración con
AutoFirmaV2 requiere recibo verificable del verificador y otro consumidor V3
nominal; la referencia del objeto firmado o un resultado booleano no habilitan
la transición. La preparación de notificación no declara envío ni entrega.
