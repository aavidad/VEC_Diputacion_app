# Coordinación CT-000045: fachadas nominales de consulta RRHH

Fecha: 29 de julio de 2026.

## Autoridad y alcance

Base exacta:

```text
c3e93ba2222098e5deadc4d5b803b1fbdeed7cea
```

Rama productora:

```text
agent/ct45-fachadas-20260729
```

Worktree relativo:

```text
.worktrees/ct45-fachadas-20260729
```

CT-000045 expone el motor privado CT-000044 mediante dos fachadas SQL
nominales. No implementa todavía el adaptador Go, la composición raíz, TLS,
HTTP, web, E2E ni conformidad organizativa. No modifica migraciones cerradas.

El productor solo puede crear:

```text
deploy/postgresql/contratacion_temporal/migraciones/
  000045_fachadas_nominales_consultas_rrhh.up.sql
  000045_fachadas_nominales_consultas_rrhh.down.sql
  000045_componentes/
deploy/postgresql/contratacion_temporal/pruebas_sql/o405_ct45_*
deploy/postgresql/contratacion_temporal/
  probar_o4_05_fachadas_nominales_ct45_pg18_4.sh
docs/portal_vec/revisiones/o4_05_revision_ct_000045_*.md
```

Solo dirección actualiza tableros, porcentajes, relevos o documentación
transversal después de integrar un candidato con `GO`.

## Frontera exterior

Se crean exactamente dos funciones, sin selector genérico:

```text
vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1
vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1
```

Cada función recibe doce argumentos `IN`, en este orden:

1. `alcance_consulta_rrhh_v1`;
2. `consulta_cuadro_rrhh_v1` o `consulta_detalle_rrhh_v1`;
3. `capacidad_canonica bytea`;
4. `decision_canonica bytea`;
5. `motivo_canonico bytea`;
6. `contexto_actor_canonico bytea`;
7. `persona_version numeric`;
8. `perfil_version numeric`;
9. `payload_vec_ad_3 bytea`;
10. `sobre_cose_sign_1 bytea`;
11. `evidencia_verificacion bytea`;
12. `raiz_publica_spki bytea`.

Los tipos CT40 de alcance y consulta son contratos nominales, no autoridades.
El futuro adaptador liga únicamente escalares PostgreSQL integrados y
construye `ROW(...)::tipo` en el texto SQL. No liga estructuras Go ni necesita
registrar un códec de compuestos.

Esta invocación sin `USAGE` sobre los tipos quedó reproducida mediante
`Parse/Bind/Execute` en PostgreSQL 18.4. La prueba CT-000045 deberá incorporarla
como regresión. La seguridad no depende de ocultar la forma del tipo: depende
de la capacidad VEC nueva, la identidad viva, el contrato nominal y el
`EXECUTE` mínimo.

No se aceptan como argumentos libres actor, perfil, sesión, rol, organización
alternativa, acción, módulo, audiencia, finalidad o tipo de recurso.

## Constantes por fachada

| Ligadura | Cuadro | Detalle |
| --- | --- | --- |
| Acción | `contratacion_temporal.cuadro.consultar` | `contratacion_temporal.expediente.consultar` |
| Audiencia | `vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1` | `vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1` |
| Módulo | `contratacion_temporal` | `contratacion_temporal` |
| Tipo de recurso | `cuadro_rrhh_contratacion_temporal` | `expediente_contratacion_temporal` |
| Finalidad | `gestion_operativa_contratacion_temporal` | `tramitacion_expediente_contratacion_temporal` |
| Dominio | `vec.contratacion_temporal.consulta_rrhh.cuadro.v1` | `vec.contratacion_temporal.consulta_rrhh.detalle.v1` |

Cada constante se fija en la fachada y vuelve a cotejarse en CT-000044. Un
cruce de capacidad, consulta, organización, ámbito o expediente falla cerrado.

## Salida minimizada

La fachada invoca exactamente una vez su motor y expande el resultado en
argumentos `OUT`. No devuelve `resultado_motor_*`, material VEC, identidad,
sesión ni tipos privados CT44.

La salida exterior contiene:

- `contenido_canonico bytea`, obtenido de la misma materialización usada por
  CT-000043; no se permite una segunda lectura;
- `cursor_siguiente text` solo en cuadro; en detalle no existe cursor;
- los diecinueve campos escalares de
  `resultado_cierre_prueba_rrhh_v2`:
  `esquema`, `acceso_ref`, `secuencia`, `anterior_sha256`,
  `huella_sha256`, `vinculo_identidad_huella_sha256`,
  `alcance_huella_sha256`, `registrada_en`, `auditoria_vec_ref`,
  `auditoria_vec_huella_sha256`, `consumo_vec_huella_sha256`,
  `contenido_huella_sha256`, `resultado_huella_sha256`,
  `cursor_huella_sha256`, `generada_en`, `expediente_ref`,
  `version_expediente`, `total` y `recibo_sello_sha256`.

El cursor claro solo puede vivir en memoria y en la respuesta provisional. No
se persiste, registra ni incluye en argumentos de procesos.

## Transacción y propiedades

PostgreSQL no permite que una función inicie su propia transacción. El futuro
adaptador abre:

```sql
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
```

La fachada comprueba el aislamiento y que la transacción no sea de solo
lectura antes de invocar el motor. Las dos funciones son:

```text
SECURITY DEFINER
VOLATILE
PARALLEL UNSAFE
no STRICT
no LEAKPROOF
search_path = pg_catalog
row_security = on
timezone = UTC
lock_timeout = 1s
statement_timeout = 4s
idle_in_transaction_session_timeout = 6s
```

El orden atómico permanece:

```text
validación estructural pura
→ comprobación de frontera y transacción
→ construcción privada del material VEC
→ una única invocación nominal al motor
→ consumo VEC antes de leer
→ identidad viva y ligaduras exactas
→ una sola materialización
→ registro, prueba y efectos de cursor
→ salida provisional
→ validación Go
→ COMMIT exterior
→ entrega al cliente
```

Un error o rollback anterior al `COMMIT` deja ausentes consumo, acceso, prueba
y efectos de cursor. Un resultado ambiguo no permite reusar la capacidad.

## Errores públicos

Se preservan sin normalizar:

```text
40001
40P01
55P03
57014
```

Todo otro rechazo de ejecución devuelve exactamente:

```text
SQLSTATE 42501
consulta RRHH rechazada
```

Ausente, ajeno, denegado, capacidad caducada o revocada y versión incorrecta
son indistinguibles por código, mensaje y estado durable. `55000` se reserva a
instalación o retirada incompatible y nunca constituye un oráculo funcional.

## Roles y privilegios

Las funciones pertenecen al `NOLOGIN`
`vec_contratacion_temporal_propietario`. El grupo
`vec_contratacion_temporal_consultor_rrhh` conserva únicamente:

- `CONNECT` a la base;
- `USAGE` del esquema;
- `EXECUTE` sin opción de concesión sobre las dos firmas exactas.

No recibe lectura o DML, secuencias, `USAGE` sobre tipos CT40–CT44 ni
`EXECUTE` sobre motores, cierres, cánones, identidad, VEC o registradores.

El `LOGIN` de aplicación debe ser nominativo, no privilegiado, con una única
membresía directa al grupo, `INHERIT=true`, sin `ADMIN`, `SET ROLE`,
`CREATEROLE`, `CREATEDB`, `REPLICATION`, `BYPASSRLS` ni membresías puente.
Una comprobación transitiva mediante `pg_has_role` no basta.

## Migración y retirada

CT-000045 exige exactamente:

```text
control_migracion_cobertura_o4 = 24
control_migracion_consultas_rrhh = 8
```

El `UP` avanza de forma atómica a `25/9`. El `DOWN` exige `25/9`, retira
primero las fachadas con `DROP ... RESTRICT` y vuelve a `24/8` sin borrar
accesos, recibos, cursores o historia.

La instalación debe ser atómica, reentrable solo mediante error controlado y
tener huella semántica estable. Omisión, mutación, dependencia entrante,
deriva de ACL/propietario/configuración o barrera futura provocan rollback
completo. CT-000045 bloquea el `DOWN` de CT-000044 mientras permanezca
instalada.

## Componentes previstos

```text
000045_componentes/010_guardas_frontera.sql
000045_componentes/020_fachada_cuadro.sql
000045_componentes/030_fachada_detalle.sql
000045_componentes/090_acl_catalogo.sql
000045_componentes/095_avance_barreras.sql
```

Cada fichero queda por debajo de 800 líneas. El `UP` incluye componentes de
forma verificable y no duplica su lógica.

## Matriz mínima PostgreSQL 18.4

El runner fija PostgreSQL 18.4 por digest y UTF-8, y acredita:

1. `UP`, reentrada rechazada, `DOWN`, segunda retirada rechazada y
   `UP → DOWN → UP`;
2. huella idéntica en tres instalaciones frescas;
3. omisión y mutación individual de cada componente sin objeto ni barrera
   parcial;
4. firmas, modos `IN/OUT`, nombres, tipos, propiedades, propietario,
   comentarios y dependencias exactos;
5. `PUBLIC`, roles y `LOGIN` no autorizados rechazados; propietario,
   superusuario y migrador con `SET ROLE` se catalogan como excepciones
   administrativas y no se confunden con runtime;
6. grupo consultor `NOLOGIN` y `LOGIN` runtime autorizado solo en las dos
   fachadas; motores privados inaccesibles;
7. llamada mediante binds escalares y `ROW(...)::tipo`, sin `USAGE` CT40–CT44;
8. topología de membresía directa y rechazo de puentes o membresías extra;
9. rechazo de `READ COMMITTED`, `REPEATABLE READ` y `READ ONLY`;
10. éxito solo en `SERIALIZABLE READ WRITE` y rollback exterior sin efectos;
11. mutación individual de las seis constantes, ámbito, consulta, identidad
    y cada una de las diez piezas VEC;
12. igualdad exterior de ausente, ajeno, denegado y versión incorrecta;
13. capacidad nueva de un solo uso; dos llamadas concurrentes con la misma
    capacidad dejan un ganador; replay, cruce, caducidad y revocación;
14. cursor falso, consumido, caducado, revocado o ligado a otros filtros,
    ámbito o identidad;
15. dos continuaciones concurrentes con un ganador;
16. revocación antes del motor, motor antes de revocación, rollback de
    revocación, dos capacidades iguales y familias independientes;
17. cancelación, timeout, interbloqueo y serialización sin estado parcial;
18. consumo antes de cualquier lectura, materialización única, corte estable,
    orden total y límite exactos; `contenido_canonico` coincide byte a byte
    con CT42 y su SHA-256 con `contenido_huella_sha256`;
19. `search_path` hostil, `row_security=off`, homónimos, sobrecargas y derivas;
20. barrera futura `26/10`, dependencia futura y safe-down de CT44;
21. token y material VEC ausentes de tablas, logs, errores, temporales y
    `argv`.

Los secretos sintéticos se transportan por entrada estándar o entorno
transitorio. Se prohíben `psql --set=secreto`, `docker --env X=valor`,
`grep -F "$secreto"` y equivalentes. Los comprobadores de ausencia fallan
cerrados si el valor está vacío o el buscador devuelve error.

## Separación de responsabilidades

- productor: DDL, runner, pruebas y evidencia del candidato;
- revisor independiente: reproduce PostgreSQL 18.4 y emite `GO` o `NO-GO`;
- integrador: aplica commits uno a uno, repite puertas, actualiza estado y
  publica.

Producción permanece en `NO-GO` aunque CT-000045 termine. Después se requieren
adaptador PostgreSQL Go, composición raíz, TLS/mTLS viva, web definitiva, E2E
HTTP y conformidades RRHH, DPD y Sistemas.
