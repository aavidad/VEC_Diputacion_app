# Revisión C2.2-S0.2: cierre de la retirada de `000002`

Fecha: 31 de julio de 2026.

Estado: **GO técnico local**, con `P0=0`, `P1=0` y `P2=0` en las
revisiones independientes finales. Este cierre no habilita producción, datos
reales ni una nueva puerta funcional; C2.2-A y C2.2-B continúan pendientes.

## Alcance integrado

| Corte | Commit estable | Resultado |
| --- | --- | --- |
| Corrector de integridad | `d133680` | Serializa inventario y retirada frente a cambios catalogales concurrentes. |
| Regresión del corrector | `25ea01e` | Acredita `000001` y `000002`, roles, lenguaje, colación y ajustes por base. |
| S0.2c | `a721a5f` | Concurrencia, cancelación, preservación, `RESTRICT` y tres ciclos. |
| S0.2b | `b146f10`, `e5e237a` | Estructura, ACL, consumidores y topología completa de roles. |
| S0.2d | `dcc2151` | Composición directa de los tres runners desde el runner principal. |

## Hallazgos cerrados

La auditoría preventiva reabrió S0.1 y S0.2a al reproducir una carrera entre
el inventario y los `DROP`: un cambio soportado de rol, comentario o ACL podía
confirmarse durante esa ventana. La misma revisión reprodujo cambios de
lenguaje y colación que alteraban el manifiesto de `000002`.

El corrector mantiene hasta `COMMIT` bloqueos `SHARE` sobre los catálogos
canonizados. La lista se comprobó con DDL PostgreSQL 18.4 real: los locks de
las relaciones cubren restricciones, triggers, políticas, reglas y valores
predeterminados; `pg_depend` y `pg_shdepend` cubren extensiones, operadores y
dependencias. Se añadieron únicamente los locks catalogales necesarios,
incluidos roles, membresías, ajustes, lenguaje y colación.

La retirada acepta LOGIN externos como miembros directos de runtime solo con
la forma exacta `ADMIN FALSE`, `INHERIT TRUE`, `SET FALSE`, sin atributos
administrativos, ajustes ni membresías adicionales. Rechaza miembros externos
de propietario o migrador, roles del módulo como miembros u otorgantes y
ajustes globales o de la base actual. Un ajuste de otra base del mismo clúster
no produce un falso rechazo.

La primera versión de S0.2b recibió `NO-GO`, `P1=1`, porque no ejercitaba
todas esas ramas. `e5e237a` añadió las regresiones separadas y la postcondición
que conserva el LOGIN legítimo tras retirar `000002`; la re-revisión terminó
con `P0=P1=P2=0`.

## Evidencia reproducida

- PostgreSQL `18.4` exacto por imagen fijada por digest.
- Runner principal completo, incluido S0.2b, S0.2c y el corrector: verde.
- Runner catalogal del corrector: verde en productor, dirección y revisor.
- S0.2b: dos ejecuciones de productor y una del revisor.
- S0.2c: dos ejecuciones de productor y una del revisor.
- Cancelación, `lock_timeout`, cola de ACL, rollback, reentrada, OID nuevos,
  consumidores externos, huellas físicas y lógicas y retirada final.
- `bash -n`, ShellCheck, `git diff --check`, tamaños y Gitleaks: verdes.
- Tamaños: `000001.down.sql` 789, `000002.down.sql` 799 y runner principal
  787 líneas; ningún fichero supera 800.
- Cero temporales, procesos o contenedores residuales tras las pruebas.

S0.2d obtuvo revisión independiente `GO`, `P0=P1=P2=0`: el runner principal
llama una sola vez a cada runner autónomo, ninguno delega en otro y cualquier
fallo corta la composición y activa la limpieza.

## Frontera de la garantía

La garantía cubre DDL, ACL, comentarios y dependencias soportados por
PostgreSQL 18.4 durante una ventana de mantenimiento exclusiva. Los locks
serializan el intervalo inventario–retirada; no prometen impedir una operación
nueva después de `COMMIT`. El tráfico migratorio debe permanecer detenido en
esa ventana. DML directo de superusuario contra `pg_catalog` supone compromiso
total de la base y permanece fuera del contrato.

La siguiente minitarea es C2.2-A, historia y puntero de organización
corporativa. C2.2-B empezará únicamente después de cerrar A.
