# Coordinación C2.2-A: organización corporativa versionada

Fecha: 31 de julio de 2026.

Estado: **contrato de implementación; no acredita todavía la capacidad**.

## Resultado único

C2.2-A añade a ContextoActor una proyección organizativa de primera clase,
versionada y sin datos semánticos. Entrega únicamente su historia, el puntero
actual y las garantías de integridad necesarias para C2.2-B. No publica, no
revoca, no selecciona actores y no autoriza consultas RRHH.

La fuente normativa técnica es la
[decisión C2.2](decision_c2_2_organizacion_y_vinculo_corporativo_2026-07-30.md).
Ante una discrepancia prevalece esa decisión y la implementación se detiene.

## Dependencias y numeración

La migración reservada es:

```text
deploy/postgresql/contexto_actor_v1/migraciones/
  000003_organizacion_corporativa_v1.up.sql
  000003_organizacion_corporativa_v1.down.sql
```

Requiere una instalación exacta y completa de `000001` y `000002`. No cambia
sus objetos. `000004` queda reservada para C2.2-B y no puede comenzar hasta el
GO independiente de C2.2-A.

El orden de barreras es invariable:

```text
A compartida = vec_contexto_actor_v1:migracion:acreditacion_uso:v2
B exclusiva  = vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1
```

Tanto el alta como la retirada toman `A SHARED → B EXCLUSIVE`, antes de
bloquear relaciones. Así `000002 down` no atraviesa `000003` y una futura
`000004` podrá usar `A SHARED → B SHARED → C EXCLUSIVE` sin ciclos.

## Objetos exactos

`000003 up` crea, y no adopta, estos cinco objetos nominales:

1. función `vec_contexto_actor_v1.organizacion_ref_valida(text)`;
2. tabla `vec_contexto_actor_v1.organizacion_versiones`;
3. tabla `vec_contexto_actor_v1.organizacion_actual`;
4. tipos fila automáticos de ambas tablas;
5. índices, restricciones, políticas, triggers y relaciones TOAST internas
   derivadas que se enumeran o acreditan abajo.

No crea roles, secuencias, vistas, API, funciones de escritura, extensiones ni
un segundo contador de generación.

### Identificador nominal

`organizacion_ref_valida(text)` es SQL, `IMMUTABLE`, no `SECURITY DEFINER` y
fija `search_path = pg_catalog`. Devuelve verdadero exclusivamente para bytes
ASCII que cumplan:

```regex
^org_[a-z0-9]{16,80}$
```

No reutiliza `referencia_valida`; no normaliza, translitera ni convierte una
referencia libre. `PUBLIC`, runtime, selector y cualquier consumidor carecen
de `EXECUTE`. La gramática no puede demostrar por sí sola que un sufijo carece
de significado: la fuente gobernada debe generar valores opacos y C2.3 deberá
rechazar una procedencia que no acredite esa generación; no se inventa aquí un
clasificador lingüístico falible.

### Historia `organizacion_versiones`

Columnas, en este orden:

| Columna | Tipo | Contrato |
| --- | --- | --- |
| `organizacion_ref` | `text NOT NULL` | validador nominal `org_` |
| `version` | `numeric(20,0) NOT NULL` | `1..18446744073709551615` |
| `procedencia_ref` | `text NOT NULL` | procedencia existente exacta |
| `procedencia_version` | `numeric(20,0) NOT NULL` | versión comprometida |
| `procedencia_huella_sha256` | `text NOT NULL` | 64 hex minúsculas |
| `procedencia_autoridad` | `text NOT NULL` | sólo `autoridad_maestra_acreditada` |
| `estado` | `text NOT NULL` | sólo `activo` o `revocado` |
| `vigente_desde` | `timestamptz(6) NOT NULL` | UTC finito |
| `vigente_hasta` | `timestamptz(6) NOT NULL` | UTC finito y posterior |

Restricciones exactas:

- `organizacion_versiones_pk`, primaria
  `(organizacion_ref, version)`;
- `organizacion_versiones_procedencia_uq`, alternativa completa
  `(organizacion_ref, version, procedencia_ref,
  procedencia_version, procedencia_huella_sha256,
  procedencia_autoridad)`, necesaria para la futura FK de C2.2-B;
- `organizacion_versiones_procedencia_fk`, FK `MATCH FULL` de los cuatro
  campos de procedencia a la clave alternativa exacta de `procedencias`;
- `organizacion_versiones_ref_ck`, validador nominal;
- `organizacion_versiones_version_ck`, rango `uint64`;
- `organizacion_versiones_procedencia_ck`, `procedencia_valida` completa;
- `organizacion_versiones_autoridad_ck`, igualdad exacta con
  `autoridad_maestra_acreditada`;
- `organizacion_versiones_estado_ck`, estado cerrado;
- `organizacion_versiones_vigente_desde_ck` y
  `organizacion_versiones_vigente_hasta_ck`, instantes finitos;
- `organizacion_versiones_ventana_ck`, ventana civil estricta
  `[vigente_desde, vigente_hasta)`.

No existen índices de negocio adicionales: los únicos son
`organizacion_versiones_pk` y `organizacion_versiones_procedencia_uq`, creados
por sus restricciones y con esos mismos nombres. La relación TOAST y su índice
interno, si PostgreSQL los materializa, se descubren desde `reltoastrelid` y
deben tener forma, propietario, ACL y dependencias nativas exactas; nunca se
aceptan por nombre o por una cuenta global fija.

La historia usa exactamente:

- `historia_inmutable`, `BEFORE UPDATE OR DELETE FOR EACH ROW`, con
  `rechazar_mutacion_historia()`;
- `historia_no_truncable`, `BEFORE TRUNCATE FOR EACH STATEMENT`, con
  `rechazar_truncado()`.

No existe borrado lógico mutable: una revocación futura será una nueva versión
append-only y el puntero podrá avanzar hasta ella.

### Puntero `organizacion_actual`

Contiene sólo `organizacion_ref text NOT NULL` y
`version numeric(20,0) NOT NULL`. `organizacion_actual_pk` es la primaria por
referencia; `organizacion_actual_version_ck` exige el rango `uint64`; y
`organizacion_actual_version_fk` es una FK `MATCH FULL` de
`(organizacion_ref, version)` a la primaria histórica. El único índice es
`organizacion_actual_pk`. Puede apuntar a una versión revocada; expresa la
última versión conocida, no permiso.

Reutiliza exactamente los tres triggers comunes de `000002`:

- `puntero_actual_no_truncable_v2`, `BEFORE TRUNCATE`, ejecuta
  `rechazar_truncado()`;
- `serializar_mutacion_punteros_actuales_v2`, `BEFORE INSERT OR UPDATE OR
  DELETE`, ejecuta la función homónima;
- `avanzar_generacion_punteros_actuales_v2`, `AFTER INSERT OR UPDATE OR
  DELETE`, ejecuta la función homónima.

Todos son de sentencia, no llevan argumentos ni tablas de transición y quedan
habilitados en modo ordinario. No se añade otro trigger al puntero.

## Propiedad, RLS y privilegios

Los cinco objetos y todos sus dependientes pertenecen únicamente a
`vec_contexto_actor_v1_propietario`.

Ambas tablas tienen `ENABLE ROW LEVEL SECURITY` y
`FORCE ROW LEVEL SECURITY`. Cada una posee una sola política llamada
`acceso_propietario_exacto`, `PERMISSIVE`, `FOR ALL`, limitada al propietario,
con `USING` y `WITH CHECK` que exigen que `current_user` sea exactamente el
propietario.

No se concede ningún privilegio. Se revoca expresamente todo sobre función,
tablas y tipos fila a `PUBLIC` y `vec_contexto_actor_v1_runtime`. El selector
corporativo, sus LOGIN, PDP, Autorización y Contratación temporal conservan
cero acceso directo y efectivo. C2.2-A no altera el `USAGE` preexistente del
esquema ni las funciones runtime de `000001`.

## Alta atómica y fallo cerrado

El `up` es un único documento SQL transaccional y autónomo, ejecutable sin
metacomandos por `pgx.Conn.Exec`. Primero sólo comprueba superusuario, versión
y codificación; después toma `A SHARED → B EXCLUSIVE`. A continuación bloquea
`procedencias` y `control_generacion_punteros_actuales_v2` en `SHARE` y toma
también `SHARE` sobre los mismos catálogos enumerados para el `down`, hasta el
`COMMIT`. Sólo después de tener esa fotografía inmóvil reacredita:

- superusuario, una versión compatible `>= 18.0 y < 19.0`, UTF-8 y la zona
  transaccional fijada por la propia migración a UTC;
- roles y propietario base presentes;
- esquema, `procedencias`, funciones de validación/inmutabilidad y control de
  generación de `000002` presentes y con propietario esperado;
- ausencia total de cualquier objeto nominal de `000003`.

La reacreditación posterior a los locks y la creación son contiguas. Después
hace `SET LOCAL ROLE` al propietario, crea todos los objetos —las escrituras
propias de la sesión pueden ampliar sus locks catalogales—, cierra ACL y
confirma otra vez las postcondiciones exactas. Una discrepancia revierte todo;
no hay reentrada permisiva ni `IF NOT EXISTS`. Las barreras coordinan
migraciones VEP; los locks de relaciones y catálogos cierran además el
`ALTER ROLE`, `ALTER FUNCTION`, ACL o DDL concurrente de un superusuario.

## Retirada gobernada

El `down` es también un único documento SQL transaccional, autónomo, sin
`CASCADE`, sin metacomandos y con `lock_timeout` y `statement_timeout` finitos.
Está denegado salvo que el GUC de sesión tenga exactamente:

```text
vec.confirmar_retirada_organizacion_corporativa_v1
= RETIRAR_ORGANIZACION_CORPORATIVA_V1
```

La confirmación no permite destruir evidencia. El orden completo es:

1. validar únicamente GUC, superusuario y versión, sin tomar decisiones sobre
   un catálogo todavía mutable;
2. tomar `A SHARED → B EXCLUSIVE`;
3. tomar `ACCESS EXCLUSIVE`, primero sobre `organizacion_actual` y después
   sobre `organizacion_versiones`;
4. tomar `SHARE` hasta `COMMIT` sobre `pg_authid`, `pg_auth_members`,
   `pg_db_role_setting`, `pg_class`, `pg_attribute`, `pg_index`,
   `pg_namespace`, `pg_language`, `pg_collation`, `pg_proc`, `pg_type`,
   `pg_default_acl`, `pg_description`, `pg_seclabel`, `pg_init_privs`,
   `pg_depend`, `pg_shdepend`, `pg_constraint`, `pg_trigger`, `pg_policy`,
   `pg_publication`, `pg_publication_namespace`, `pg_publication_rel`,
   `pg_subscription_rel` y `pg_statistic_ext`;
5. reacreditar GUC, ejecutor, filas y manifiesto completo después de obtener
   todos los bloqueos, e iniciar inmediatamente los drops explícitos.

Con esa fotografía inmóvil, la retirada:

- rechaza cualquier fila en historia o puntero;
- inmoviliza hasta `COMMIT` los catálogos que acrediten estructura, ACL,
  políticas, comentarios y dependencias;
- exige el manifiesto exacto de función, tablas, tipos, columnas,
  restricciones, índices, RLS, políticas, cinco triggers y relaciones TOAST;
- rechaza concesiones, propietarios, comentarios, etiquetas, publicaciones,
  estadísticas, herencias o dependencias ajenas;
- rechaza consumidores posteriores, incluida `000004`;
- elimina explícitamente triggers, políticas, tablas y función en orden
  inverso, siempre con semántica `RESTRICT`;
- deja `000001` y `000002` byte y catálogo-equivalentes a su estado anterior.

Cualquier error revierte el documento completo. Una carrera DDL o DML espera
los mismos bloqueos o provoca rechazo/cancelación; nunca permite inventariar un
estado y retirar otro.

La certificación se ejecuta sobre PostgreSQL 18.4 exacto. Admitir 18.x en la
migración expresa compatibilidad dentro de la versión mayor, no certifica una
menor futura: cada actualización 18.x requiere repetir la matriz antes de
autorizarla operativamente.

## Matriz de aceptación PostgreSQL 18.4

El runner autónomo se llamará:

```text
deploy/postgresql/contexto_actor_v1/
  probar_organizacion_corporativa_v1_pg18_4.sh
```

Usará la imagen PostgreSQL 18.4 fijada por digest, un contenedor efímero y
limpieza en `EXIT`, `INT` y `TERM`. Debe acreditar al menos:

1. instalación `000001 → 000002 → 000003`, atomicidad y rechazo de reentrada;
2. nombres, tipos, orden de columnas, typmods, FKs, checks, índices,
   propietarios, RLS forzada, política exacta, ACL de tabla/columna/tipo y
   cuerpo/configuración de la función, incluida la forma derivada de TOAST;
3. límites de `org_`: 16/80 válidos y `NULL`, vacío, cortos, largos,
   mayúsculas, guiones, subrayados, Unicode, espacios y controles ASCII
   rechazados;
4. versiones extremas, huella, autoridad, estados, instantes no finitos,
   precisión y ventanas inválidas;
5. inserción de versiones activa y revocada, avance del puntero y aumento de
   la generación común;
6. rechazo de `UPDATE`, `DELETE` y `TRUNCATE` de historia y de `TRUNCATE` del
   puntero, conservando datos y generación;
7. cero acceso para runtime, selector y LOGIN sintéticos, incluso mediante
   columnas o tipos;
8. `000002 down` rechazado mientras existe `000003`, sin deriva;
9. `000003 down` rechazado sin opt-in, con opt-in incorrecto, con filas, ACL,
   objetos o consumidores hostiles y ante `000004` sintética;
10. retirada vacía exacta `000003 down → 000003 up`, sin alterar `000002`;
11. carreras reales de instalación/retirada, DDL, inserción y avance de
    puntero, con espera acotada, ausencia de interbloqueo y rollback íntegro;
12. lectura de los bytes completos del `down` y ejecución sin preprocesado
    mediante `pgx.Conn.Exec`, con cancelación y saneamiento de la conexión;
13. ausencia de contenedores, ficheros o procesos residuales.

El runner no llama a otros runners. Tras su GO, `probar_integracion.sh` lo
invocará directa y exactamente una vez al final de sus pruebas propias.

## Minitareas, dependencias y write-sets

Los write-sets son exclusivos mientras una minitarea está activa. A2 depende
de A1 y modifica el mismo runner únicamente después de cerrar y confirmar A1;
no existe edición concurrente del fichero compartido.

| Minitarea | Dependencia y escritura | Entrega |
| --- | --- | --- |
| A0 | este documento | contrato revisado y commit documental |
| A1 | tras A0; los dos SQL `000003` y runner focal | migración reversible y casos 1–11 y 13 verdes en el mismo commit autónomo; se justifica el trío porque alta, retirada y prueba son un único contrato reversible |
| A2 | tras A1; `internal/vec/adapters/contextoactor/postgres/retirada_organizacion_corporativa_v1_integracion_test.go` nuevo y ampliación secuencial del runner | caso 12 y después matriz completa 1–13 verdes; no cambia Go productivo |
| A3 | tras A2; `probar_integracion.sh` y README del módulo | composición directa una vez, calidad global verde y corte técnico candidato |
| A4 | tras A3; sólo documento nuevo de revisión | revisión independiente del corte técnico compuesto y GO con `P0=P1=P2=0`; cualquier corrección vuelve al productor como minitarea focal con su prueba y exige nueva revisión |
| A5 | tras A4; sólo documentos transversales de estado y relevo, nunca el documento de revisión | estado sincronizado y publicación del corte aprobado |

Productor y revisor no pueden ser la misma persona o agente. Sólo A2 añade una
prueba Go de integración; ninguna minitarea modifica Go productivo ni los
módulos de Identidad, PDP, Autorización, Contratación o Bolsa. Los commits
serán pequeños, en castellano y con documentación asociada.

## Fuera de alcance y criterio de cierre

Quedan fuera: datos reales, AD, nómina, RPT, denominación, CIF, jerarquías,
unidades, centros, importadores, API de publicación/revocación, selección,
recibo, autorización y web. No se crean fixtures que puedan confundirse con
autoridad productiva.

C2.2-A sólo se cierra cuando A1, A2, A3, A4 y A5 estén integradas, todas las
puertas sean verdes, una revisión independiente del corte compuesto declare
`GO` con `P0=P1=P2=0`, el árbol esté limpio y el corte se haya publicado. Hasta
entonces C2.2-B sigue bloqueada y las métricas funcionales permanecen sin
cambios.
