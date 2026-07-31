# Decisión C2.2: organización y vínculo corporativo RRHH

Fecha: 30 de julio de 2026.

Estado: **decisión cerrada; implementación en curso**.

## Problema

La resolución corporativa de ContextoActor debe acreditar una organización
activa y revocable. El esquema vigente conserva cuenta, persona, perfil y
vínculos, pero no posee una autoridad organizativa ni permite revocar una
organización con independencia de un vínculo.

Incluir la organización como una cadena dentro del vínculo impediría
distinguir:

- revocación de la organización;
- revocación de la adscripción de una persona;
- cambio de perfil;
- sustitución de la fuente maestra;
- avance de versión de cualquiera de esos hechos.

También convertiría la referencia libre que todavía usan algunos contratos y
fixtures de Contratación temporal en autoridad, lo que está prohibido.

## Decisión

ContextoActor incorporará una proyección organizativa versionada de primera
clase y un vínculo corporativo separado. Ninguna de las dos será una fuente
maestra: ambas materializarán una procedencia gobernada y acreditada.

```text
organización versionada
        ↓
cuenta + persona + perfil + vínculo ContextoActor
        ↓
vínculo corporativo para interna_corporativa + consulta_rrhh
```

Una organización revocada invalida todos sus usos sin reescribir sus vínculos.
Un vínculo revocado no altera la historia de la organización. El futuro
selector exige una única combinación activa y vigente; no elige la primera ni
suma perfiles.

## Identificador organizativo

La referencia autoritativa:

- usa el prefijo `org_`;
- adopta el contrato interoperable ya publicado por Bolsa:
  `^org_[a-z0-9]{16,80}$`;
- posee un sufijo opaco de 16 a 80 octetos formado exclusivamente por
  minúsculas ASCII y dígitos;
- no acepta mayúsculas, guiones, subrayados ni representaciones Unicode;
- no contiene nombre, CIF, código visible, unidad, provincia ni otro dato
  semántico;
- permanece estable aunque cambien la denominación o los atributos de la
  organización.

ContextoActor añadirá un validador nominal de esa gramática; no reutilizará
`referencia_valida`, cuyo mínimo de 22 octetos y alfabeto más amplio son
incompatibles. La validación en los límites de ambos módulos no crea otra
autoridad: el identificador es el mismo y la procedencia versionada sigue
perteneciendo a ContextoActor.

El código visible, denominación, jerarquía y relaciones con unidades, centros
o RPT pertenecen a fuentes o catálogos gobernados posteriores. No se codifican
en `organizacion_ref`.

Las referencias libres como `organizacion:diputacion-granada` siguen siendo
fixtures o coordenadas legadas no autoritativas. C2.6 deberá sustituirlas por
el recibo corporativo opaco en el recorrido real. No habrá conversión
implícita, doble identidad ni tabla de equivalencias creada por conveniencia.

## Modelo C2.2-A: organización

La historia de organización compromete:

- `organizacion_ref` y versión `numeric(20,0)` en el rango `uint64`;
- procedencia, versión, huella SHA-256 y autoridad exactas;
- estado técnico `activo` o `revocado`;
- ventana civil autoritativa `[vigente_desde, vigente_hasta)`;
- instantes UTC finitos con precisión `timestamptz(6)`.

El puntero actual contiene solo referencia y versión, con clave foránea
completa a la historia. Puede apuntar a una versión revocada: el puntero
expresa la última versión, no una autorización.

Historia y puntero:

- pertenecen al propietario de ContextoActor;
- usan RLS activada y forzada como defensa adicional;
- no conceden acceso a `PUBLIC`, runtime, selector, LOGIN, PDP ni
  Contratación temporal;
- rechazan `UPDATE`, `DELETE` y `TRUNCATE` de la historia;
- usan la generación común de punteros de la migración `000002`.

## Modelo C2.2-B: vínculo corporativo

Cada versión liga de forma inseparable:

- vínculo corporativo y versión;
- cuenta y versión;
- persona y versión;
- perfil y versión;
- vínculo ContextoActor base y versión;
- organización y versión;
- superficie literal `interna_corporativa`;
- uso literal `consulta_rrhh`;
- procedencia completa;
- estado y ventana de vigencia.

El puntero actual es único por:

```text
(cuenta_ref, superficie, uso)
```

y referencia la fila histórica completa. Cero o varias coincidencias se
tratan como la misma denegación. C2.2 no expone esas operaciones: publicación
y revocación pertenecen a C2.3; selección y recibo privado a C2.4; fachada y
reconciliación a C2.5; acreditación nominal a C2.8.

`vinculo_corporativo_ref` usa el prefijo `vcr_` y la gramática existente de
ContextoActor para referencias internas: sufijo de 22 a 128 octetos
`[A-Za-z0-9_-]`.

No bastan claves foráneas separadas a filas que puedan pertenecer a actores
distintos. La migración debe añadir o reutilizar claves alternativas
inmutables y FKs compuestas que prueben conjuntamente:

- que el perfil corresponde a la persona exacta;
- que el vínculo ContextoActor corresponde a esa cuenta, persona y perfil;
- que todas las versiones referidas son las comprometidas por la fila;
- que la organización y versión existen bajo la procedencia indicada.

Si las tablas base no ofrecen una clave compuesta suficiente, C2.2-B debe
añadirla de forma aditiva y acreditarla; no puede sustituirla por consultas
sin una restricción durable. La futura selección y cada reacreditación
compararán además `organizacion_ref` y `organizacion_version` comprometidas
por el vínculo con el puntero organizativo actual, y después comprobarán el
estado y la vigencia de esa versión organizativa.

Historia y puntero de C2.2-B aplican las mismas garantías que C2.2-A:

- RLS activada y forzada, con una única política exacta para el propietario;
- ACL de tabla, columnas y tipos cerradas;
- historia sin `UPDATE`, `DELETE` ni `TRUNCATE`;
- puntero con los tres triggers exactos de generación de `000002`;
- cero acceso para `PUBLIC`, runtime, selector, LOGIN, PDP o Contratación
  temporal.

## Descomposición obligatoria

C2.2 no se implementará como una sola tarea:

| Minitarea | Responsabilidad única | Dependencia | Estado |
| --- | --- | --- | --- |
| C2.2-D0 | Esta decisión de organización, referencias y numeración | Ninguna | Cerrada |
| C2.2-S0.1 | Retirada segura de la base ContextoActor sin `CASCADE` ni pérdida de evidencia | D0 | Cerrada e integrada |
| C2.2-S0.2a | `down` portable de `000002` y ejecución literal mediante `pgx` | S0.1 | Cerrada e integrada |
| C2.2-S0.2b | Runner PostgreSQL 18.4 de estructura, ACL y consumidores | S0.2a | Cerrada e integrada |
| C2.2-S0.2c | Runner PostgreSQL 18.4 de concurrencia, preservación y ciclos | S0.2a | Cerrada e integrada |
| C2.2-S0.2d | Composición de los runners focales | S0.2b + S0.2c | Cerrada e integrada |
| C2.2-A | Historia y puntero de organización | C2.1b + S0.2 | Cerrada y publicada en `54a8cde`; CI `30633248293` verde |
| C2.2-B | Historia y puntero del vínculo corporativo | C2.2-A publicada | Lista; siguiente corte |

Cada productor y cada revisor serán distintos. A y B tendrán migración,
reversión y runner PostgreSQL 18.4 propios.

## Numeración

La reserva queda:

```text
contexto_actor_v1/migraciones/000003  organización corporativa
contexto_actor_v1/migraciones/000004  vínculo corporativo RRHH
contexto_actor_v1/migraciones/000005  publicación y revocación C2.3
contexto_actor_v1/migraciones/000006  recibo y selección privada C2.4
contexto_actor_v1/migraciones/000007  fachada y reconciliación C2.5
contexto_actor_v1/migraciones/000008  acreditación nominal C2.8
```

C2.6 y C2.7 son contratos y adaptadores Go y no consumen migración. C2.9 y
C2.10 conservan sus reservas en Autorización y Contratación temporal. No se
reutiliza un número ni se mezcla más de una responsabilidad.

## Retirada segura S0

S0 se divide para no mezclar dos migraciones históricas.

### S0.1 — base `000001`

- sustituye el `DROP SCHEMA ... CASCADE` del artefacto de retirada por
  inventario exacto, drops explícitos en orden inverso y `RESTRICT`;
- deniega si cualquier tabla histórica contiene filas;
- deniega ante objetos, dependencias, ACL, propietarios o consumidores no
  pertenecientes exactamente a la instalación base vacía;
- ejecuta toda la retirada en una transacción, por lo que cualquier rechazo
  restaura el estado completo;
- solo permite retirar una instalación base vacía, exacta y sin migraciones
  posteriores.

### S0.2 — generación `000002`

- descubre mediante catálogo el trío exacto de cada puntero: el trigger no
  truncable que ejecuta `rechazar_truncado()` de `000001`, el trigger que
  ejecuta `serializar_mutacion_punteros_actuales_v2()` y el que ejecuta
  `avanzar_generacion_punteros_actuales_v2()`; no compara una lista fija de
  quince;
- acredita también la tabla de generación y las dependencias efectivas de las
  dos funciones aportadas por `000002`;
- deniega ante tablas o punteros añadidos por migraciones posteriores;
- recorre ACL efectivas, tipos, dependencias y consumidores nominales;
- usa drops explícitos con `RESTRICT` y rollback total;
- acredita que A y B bloquean su retirada mientras existan.

S0.2 conserva un único `down` SQL autónomo. Puede superar de forma justificada
el objetivo de 500 líneas, pero nunca el tope duro de 800: partirlo mediante
metacomandos `psql` impediría ejecutar sus bytes sin preprocesado desde
`pgx.Conn.Exec`, y concatenar componentes exigiría un empaquetador inexistente.
No se usarán `\ir`, `\if`, `\gset`, funciones temporales ni dependencias del
directorio de trabajo.

La confirmación se comunica mediante el GUC de sesión
`vec.confirmar_retirada_acreditacion_contexto_actor_v2`, con el valor exacto
`RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2`. El adaptador toma una conexión
dedicada, fija el GUC, ejecuta el documento SQL literal y, ante cualquier
error, cancela la transacción y restablece la sesión antes de devolverla. El
propio documento vuelve a validar la confirmación dentro de su transacción.

La implementación de S0.2 se separa en cuatro commits verificables:

1. artefacto `down` portable y prueba de ejecución literal mediante `pgx`,
   cerrado e integrado como S0.2a en `9f96bcf`–`e654312`;
2. runner PostgreSQL 18.4 de estructura, ACL y consumidores;
3. runner PostgreSQL 18.4 de concurrencia, preservación y ciclos;
4. composición de los runners focales desde `probar_integracion.sh`.

S0.1 y S0.2 están cerradas. Una auditoría posterior reabrió la integridad
catalogal de S0.1/S0.2a, reprodujo carreras reales y las corrigió antes de
cerrar b, c y d. La
[revisión completa de S0.2](revisiones/revision_c2_2_s0_2_cierre_2026-07-31.md)
conserva los `NO-GO`, las correcciones, las pruebas y la frontera de amenaza.
C2.2-A queda desbloqueada.

El write-set queda limitado al `down`, `probar_integracion.sh`, los runners
focales y una prueba Go de integración del adaptador ContextoActor. Los runners
no se llaman entre sí; el script de integración es el único lanzador focal.
El orden de bloqueo empieza por la barrera base compartida, continúa con la
barrera de acreditación exclusiva y después toma `ACCESS EXCLUSIVE` sobre la
tabla de control de `000002` y las doce relaciones de `000001`. Los catálogos
acreditados quedan a continuación en `SHARE` hasta el `COMMIT`. Cada rechazo
debe conservar una huella idéntica de catálogo y datos.

VEP continúa en `NO-GO`, no tiene una instalación productiva autorizada ni
datos reales. Por eso se permite corregir ahora los artefactos `down` de
`000001` y `000002` antes de congelar la primera versión desplegable. No se
modifica el esquema creado por sus `up`. Cualquier entorno técnico que hubiese
copiado el artefacto anterior debe reconstruirse desde el corte publicado o
verificar su huella; nunca se sustituye silenciosamente un script aprobado.
Tras la primera versión desplegable, una retirada publicada será inmutable y
cualquier corrección se añadirá como una migración nueva.

## Bloqueos y generación

Los nuevos punteros reutilizan las primitivas y la fila de generación de
`000002`; no crean un segundo reloj, contador o cerrojo.

Orden:

```text
A = vec_contexto_actor_v1:migracion:acreditacion_uso:v2
B = vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1
C = vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1
```

- C2.2-A: `A SHARED → B EXCLUSIVE`;
- C2.2-B: `A SHARED → B SHARED → C EXCLUSIVE`;
- consumidores posteriores:
  `A SHARED → B SHARED → C SHARED → barrera propia`.

La retirada de `000002` debe quedar bloqueada mientras existan triggers o
consumidores nuevos.

## Fuera de alcance

C2.2 no:

- importa Active Directory, nómina, RPT ni datos reales;
- publica o revoca organizaciones mediante una API;
- resuelve candidatos;
- registra el recibo corporativo;
- autoriza con el PDP;
- modifica la organización libre de los contratos Go;
- crea jerarquías, unidades, centros o multitenencia.

La fuente maestra, semántica organizativa, responsables de publicación,
vigencia y conservación requieren aprobación de RRHH, Sistemas y DPD. Su
ausencia mantiene producción en `NO-GO`.
