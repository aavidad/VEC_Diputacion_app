# PostgreSQL de contratación temporal

Estado: persistencia O2-05, confirmación O3-04, registro durable de accesos
RRHH O4-05/C2-B, publicación global C2-C e infraestructura de cursor C2-D1
probados en PostgreSQL 18.4. Los tres últimos cortes disponen de revisión
independiente, pero aún faltan funciones exteriores, adaptador Go y
composición completa. No es una autorización de producción.

## Alcance del corte

La migración `000001_preparacion_altas` crea:

- identidad inmutable de cada reserva;
- versiones de solo adición;
- puntero actual separado;
- función cerrada `preparar_alta_v1(jsonb)`;
- idempotencia semántica por HMAC;
- referencias estables ante reintentos y concurrencia;
- privilegio runtime exclusivamente `EXECUTE`.

La migración aditiva `000002_rotacion_hmac` conserva la historia v1 y añade:

- alias inmutables de ámbito y huella por reserva canónica;
- función cerrada `preparar_alta_v2(jsonb)`;
- política explícita de generaciones obligatorias;
- restricciones genéricas `/vN`;
- revocación runtime de v1 y concesión exclusiva de v2;
- rechazo de colisiones o convergencia entre dos reservas.
- límite propio de bloqueo en la función v2;
- precondiciones ambientales de sentencia e inactividad transaccional.

La clave idempotente original no cruza hacia PostgreSQL. El adaptador solicita
a `SelladorAmbitoIdempotencia` una HMAC ligada a organización, actor y perfil,
y solo persiste esa HMAC. Reutilizar el ámbito con otra huella de petición,
organización, actor o perfil devuelve conflicto y no crea otro expediente.

## Matriz de generaciones

La política instalada por `000002` es:

| Posición | Generación | Estado | Obligación durante la ventana |
| --- | ---: | --- | --- |
| 0 | 2 | activa | Siempre presente y primera. |
| 1 | 1 | retenida | Siempre presente hasta cerrar formalmente la ventana. |

El llavero de aplicación admite una activa y hasta tres retenidas, ordenadas
de mayor a menor y sin repetir generación, referencia ni sello. Los dos
dominios —ámbito idempotente y huella de petición— deben presentar la misma
matriz. Si el conector no puede usar una generación retenida, la operación
falla antes de PostgreSQL. Si un cliente omite una generación exigida, la
función v2 falla cerrada.

Antes de cada invocación, la misma transacción debe configurar
`statement_timeout` entre 1 ms y 15 s e
`idle_in_transaction_session_timeout` entre 1 ms y 20 s. La función valida
ambos límites al comenzar y deniega valores ausentes, nulos, mal formados o
superiores. No los instala mediante `proconfig`: el límite de sentencia debe
estar armado antes de iniciar el `SELECT`, y el límite de inactividad debe
seguir vigente después de que la función retorne. `lock_timeout=2s` sí es un
límite propio de `preparar_alta_v2`.

El adaptador reutiliza los mismos sellos y referencias candidatas, pero abre
una transacción nueva hasta un máximo de tres intentos exclusivamente ante
`40001` —fallo de serialización— o `40P01` —interbloqueo—. Cualquier otro
SQLSTATE falla sin reintento. La cancelación o el vencimiento del contexto
detienen la secuencia, mientras que un `COMMIT` confirmado se devuelve como
éxito durable sin una comprobación tardía que lo convierta en fallo.

Retirar v1 requiere una migración posterior, inventario de vida máxima de las
claves idempotentes, agotamiento probado de esa vida, copia y restauración
ensayadas, y aprobación de Seguridad. Nunca se cambia la tabla de política a
mano ni se elimina la clave histórica durante la ventana.

La reserva no concede permisos ni confirma un alta. Las migraciones aditivas
`000003_expediente_confirmacion_atestada` y
`000004_integridad_agregado_alta` y
`000005_funcion_confirmar_alta_atestada` añaden el agregado durable, versión y
actuación iniciales, consumo VEC-AD-3, auditoría y outbox en un único
`COMMIT`. La identidad `confirmacion_agregado_alta`, su huella portable y las
FK diferibles ligan reserva confirmada, expediente, versión, actuación,
auditoría y outbox. No dependen de `txid`, WAL ni estado de la conexión.

El consumidor informa internamente si acaba de consumir la capacidad o si la
recuperó. Un replay nunca inserta ni completa piezas: solo devuelve el recibo
si el reconciliador acredita todas las filas, refs, canones, huellas,
instantes, eslabones y marcador. Si el efecto es posterior a otros efectos,
valida predecesor, sucesor inmediato y cabeza vigente; el marcador conserva
además la prueba durable del eslabón creado originalmente. Al instalar
`000005`, el runtime pierde `preparar_alta_v2` y recibe solo
`confirmar_alta_atestada_v1`. La autorización genérica tampoco queda expuesta
al LOGIN de la aplicación.

La entrada final usa bytes canónicos de esquemas explícitos y versionados; no
serializa por reflexión el agregado. El resultado contiene solo referencias,
número visible, versión, instante y huellas. Ninguna clave idempotente,
secreto HMAC ni dato personal se devuelve.

Las migraciones aditivas `000006_expediente_integral_versionado`,
`000007_preparacion_operaciones_analisis`,
`000008_efectos_expediente_integral`,
`000009_contrato_confirmacion_analisis`,
`000010_validadores_confirmacion_analisis`,
`000011_transicion_confirmacion_analisis` y
`000012_confirmacion_operacion_analisis` implementan la persistencia O3-04:

- materializan el alta O2 como versión 1 de una historia integral única y de
  solo adición;
- separan esa historia del puntero actual empleado por CAS;
- reservan las operaciones de análisis con HMAC generacional sin guardar la
  clave idempotente original;
- permiten replay exacto y consulta de un recibo ya confirmado;
- reservan las tablas inmutables de actuaciones, consumo de fuentes,
  consumo de decisión, auditoría encadenada y outbox para la transacción final;
- verifican los bytes canónicos de la decisión VEC V3 y de cada respuesta de
  fuente, incluido el sello HMAC y la publicación gobernada del motivo;
- revalidan con el reloj de PostgreSQL la decisión, el contexto del recurso y
  la vigencia de las fuentes;
- aplican CAS al puntero vigente y publican versión 2, actuación, consumos,
  auditoría, outbox y recibo en un solo `COMMIT`;
- devuelven en un replay exacto el recibo durable original sin reparar ni
  completar piezas;
- mantienen RLS, ACL de mínimo privilegio, filas inmutables y reversión
  destructiva protegida.

El LOGIN de aplicación solo recibe `EXECUTE` sobre
`confirmar_operacion_analisis_v1(jsonb)`. No puede leer ni escribir las tablas
subyacentes ni invocar directamente las funciones auxiliares.

## Consultas internas RRHH O4-05/C2-B

El delta `roles_consultor_rrhh_up.sql` crea un grupo técnico `NOLOGIN` de
mínimo privilegio. Las cuentas `LOGIN` son nominativas, se aprovisionan fuera
del repositorio y solo pueden tener una arista directa hacia ese grupo, con
`ADMIN FALSE`, `INHERIT TRUE` y `SET FALSE`. Se rechazan grupos adicionales,
roles privilegiados, herencia saliente desde el LOGIN y cadenas transitivas
que permitan usarlo como puente.

La migración `000036_registro_accesos_rrhh_o4_05` exige la barrera 15 de
`000035`, instala C2-B y eleva de forma atómica la barrera global a 16. Añade:

- un control de versión propio;
- una cabeza única y bloqueada para serializar la cadena;
- un registro de accesos de solo adición con RLS forzada;
- la función interna `registrar_acceso_rrhh_interno_v1(jsonb)`, sin
  `EXECUTE` para el consultor ni para `PUBLIC`;
- recibos con referencia, secuencia, huella anterior, huella propia e instante.

El registro conserva referencias opacas, perfiles, finalidad, dominio de
consulta y huellas autoritativas de decisión, capacidad, consumo, auditoría y
resultado. No almacena documentos, material probatorio duplicado, claves,
secretos ni datos de respuesta. Cuadro y detalle tienen listas positivas
distintas. Un no resultado de detalle no puede declarar versión.

La reversión `000036.down` solo admite la barrera 16, falla si existe historia
y restaura la barrera 15 tras retirar todos los objetos. Mientras C2-B está
instalado, `000035.down` queda bloqueado incluso si el registro está vacío.
Después se retira el grupo mediante `roles_consultor_rrhh_down.sql`, una vez
revocadas todas sus cuentas nominativas.

C2-B no crea cursor ni función de lectura. Sus accesos preexistentes tampoco
dependen de la publicación global añadida después.

## Publicación global RRHH O4-05/C2-C

La migración aditiva `000037_publicacion_global_rrhh` exige la barrera global
16 y `control_migracion_consultas_rrhh` v1, y eleva la barrera a 17 en el
mismo `COMMIT`. No altera `expediente_version_integral`: crea una proyección
1:1 de solo adición y un único `control_publicacion_rrhh`.

El backfill toma un bloqueo que impide nuevas inserciones en la historia,
valida y extrae los campos indexables del resumen RRHH, y asigna el
`corte_base` por `expediente_ref COLLATE "C", version`. Ese orden solo hace
reproducible la base; no se presenta como orden histórico de confirmación.

Después del backfill, un disparador `AFTER INSERT` con autoridad del
propietario:

- bloquea el singleton hasta que termina la transacción exterior;
- asigna `ultimo_corte + 1`, con máximo `2^53-1`;
- proyecta la referencia y versión exactas, organización, número visible,
  flujo, fase, estado, centro, categoría, modalidad, unidad, instantes y
  huella del agregado;
- revierte ordinal y proyección si la transacción exterior hace `ROLLBACK`.

La fila bloqueada serializa a los escritores hasta `COMMIT`; por ello los
ordinales posteriores a `corte_base` no dependen de una secuencia adelantada
ni de empates de reloj. La tabla usa RLS forzada, política exclusiva del
propietario, ACL denegadas e inmutabilidad ante `UPDATE`, `DELETE` y
`TRUNCATE`.

`000037.down` solo vuelve de 17 a 16 si no existen publicaciones posteriores
al corte base, divergencias ni dependencias, cursores o fachadas futuras.
Admite accesos C2-B anteriores porque no usan la proyección, y nunca emplea
`CASCADE` ni una bandera de destrucción. El
[informe de revisión](../../../docs/portal_vec/revisiones/o4_05_revision_publicacion_global_rrhh_2026-07-26.md)
conserva la evidencia exacta.

## Instalación

Para una instalación nueva, `roles_up.sql` crea todos los grupos técnicos.
Para actualizar una base que ya tiene `000001`–`000027`, no se vuelve a
ejecutar ese bootstrap: el DBA aplica
`roles_confirmador_cobertura_up.sql`, instala como propietario de Autorización
`migraciones_autorizacion/000006_wrapper_contexto_exacto_cobertura_o4_04e.up.sql`,
ejecuta las migraciones `000028`–`000034` y, por último, concede únicamente el
grupo confirmador al LOGIN nominativo del TCB. La reversión usa el orden
inverso: revocar membresías del LOGIN, ejecutar los `down` `000034`–`000028`,
instalar el `down` de Autorización `000006` y aplicar
`roles_confirmador_cobertura_down.sql`.

La membresía nominativa se concede sin administración ni `SET` transitivo:
`GRANT vec_contratacion_temporal_confirmador_cobertura TO <login> WITH ADMIN
FALSE, INHERIT TRUE, SET FALSE`. El delta admite reintentos con barrera 7–14,
pero antes de la barrera final 14 exige que el grupo no tenga miembros.
La credencial TCB es dedicada: esa debe ser su única arista directa en
`pg_auth_members`. No puede acumular grupos de ContextoActor, Autorización,
ejecución, gobierno, migración ni propiedad; esos usos requieren pools y
credenciales separados. La fachada de confirmación y el lector primario
comprueban en cada llamada tanto la arista exacta como sus tres opciones.

Las migraciones `000028`, `000030` y `000033` superan el objetivo orientativo
de 500 líneas, pero permanecen por debajo del límite duro de 800. Se conservan
como unidades porque cada una instala y valida en una sola transacción atómica
un conjunto inseparable de funciones, ACL, dependencias y avance de barrera;
dividirlas introduciría estados intermedios instalables.

1. Ejecutar `roles_up.sql` como DBA.
2. Aprovisionar una cuenta nominativa de migración miembro únicamente de
   `vec_contratacion_temporal_migrador`.
3. Ejecutar, por orden, `000001_preparacion_altas.up.sql` y
   `000002_rotacion_hmac.up.sql` con dicha cuenta.
4. Instalar el consumidor
   `deploy/postgresql/autorizacion_atestada_v3` y su gobierno.
5. Instalar como propietario de autorización la frontera estrecha
   `migraciones_autorizacion/000001_revalidacion_analisis_v3.up.sql`.
6. Ejecutar `000003_expediente_confirmacion_atestada.up.sql`,
   `000004_integridad_agregado_alta.up.sql` y
   `000005_funcion_confirmar_alta_atestada.up.sql`, por ese orden.
7. Ejecutar `000006_expediente_integral_versionado.up.sql`,
   `000007_preparacion_operaciones_analisis.up.sql` y
   `000008_efectos_expediente_integral.up.sql`, por ese orden.
8. Ejecutar `000009_contrato_confirmacion_analisis.up.sql`,
   `000010_validadores_confirmacion_analisis.up.sql`,
   `000011_transicion_confirmacion_analisis.up.sql` y
   `000012_confirmacion_operacion_analisis.up.sql`, por ese orden.
9. Aprovisionar la cuenta de la aplicación como miembro únicamente de
   `vec_contratacion_temporal_ejecutor`.

Para añadir exclusivamente O4-05/C2-B sobre una base en barrera 15:

1. ejecutar `roles_consultor_rrhh_up.sql` como DBA;
2. ejecutar `000036_registro_accesos_rrhh_o4_05.up.sql` como propietario;
3. aprovisionar después el LOGIN nominativo del consultor con la membresía
   exacta descrita arriba.

Para añadir después la publicación global C2-C sobre la barrera 16:

1. comprobar que `control_migracion_consultas_rrhh` permanece en v1;
2. ejecutar `000037_publicacion_global_rrhh.up.sql` como propietario;
3. comprobar el singleton, la proyección 1:1 y la barrera global 17 antes de
   habilitar cualquier consumidor futuro.

Para añadir después la infraestructura de cursores C2-D1:

1. ejecutar `roles_cursor_rrhh_up.sql` como DBA;
2. ejecutar `000038_cursores_cuadro_rrhh.up.sql` con capacidad de asumir el
   propietario NOLOGIN;
3. comprobar las barreras global 18 y de consultas 2, RLS forzada y ausencia
   de ACL directas para el runtime.

La retirada se realiza en orden inverso: primero `000038...down.sql` y después
`roles_cursor_rrhh_down.sql` como DBA. Solo se admite sin historia ni
dependencias posteriores.

Ejemplo sin credenciales incrustadas:

```bash
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000001_preparacion_altas.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000002_rotacion_hmac.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones_autorizacion/000001_revalidacion_analisis_v3.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000003_expediente_confirmacion_atestada.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000004_integridad_agregado_alta.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000005_funcion_confirmar_alta_atestada.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000006_expediente_integral_versionado.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000007_preparacion_operaciones_analisis.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000008_efectos_expediente_integral.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000009_contrato_confirmacion_analisis.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000010_validadores_confirmacion_analisis.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000011_transicion_confirmacion_analisis.up.sql
psql -X --set ON_ERROR_STOP=1 \
  --file deploy/postgresql/contratacion_temporal/migraciones/000012_confirmacion_operacion_analisis.up.sql
```

El repositorio no contiene DSN, usuarios LOGIN, contraseñas ni secretos HMAC.
Estos se suministran mediante el gestor de secretos y la configuración del
entorno.

## Prueba reproducible

La prueba de integración no instala PostgreSQL en el equipo. Usa un contenedor
efímero, sin puertos publicados, sin red y con el directorio de datos en
`tmpfs`. La imagen está fijada por resumen criptográfico:

```bash
./deploy/postgresql/contratacion_temporal/probar_integracion_docker.sh
./deploy/postgresql/autorizacion_atestada_v3/probar_integracion_o2_05.sh
./deploy/postgresql/contratacion_temporal/probar_o4_05_registro_accesos_pg18_4.sh
./deploy/postgresql/contratacion_temporal/probar_o4_05_publicacion_global_rrhh_pg18_4.sh
./deploy/postgresql/contratacion_temporal/probar_o4_05_cursores_rrhh_pg18_4.sh
```

El ensayo valida instalación con cuenta de migración, separación de roles,
privilegio mínimo del ejecutor, alta v1, rotación v1→v2 con el mismo
expediente, recibo, versión, auditoría, evento e instante de confirmación, alta
nativa v2, conflicto semántico, ausencia de una generación histórica, bloqueo
directo limitado por la propia función y límites conductuales de sentencia e
inactividad transaccional, además de ocho sesiones concurrentes con ambas
generaciones. El ensayo conserva y valida el resumen de `pgbench`: exige
`16/16` transacciones procesadas y cero fallidas. También comprueba
inmutabilidad, rechazo del rollback ordinario con historia y limpieza
destructiva autorizada. Las
identidades LOGIN y la clave usadas existen únicamente dentro del contenedor
efímero y no son credenciales de ningún entorno.

El segundo runner integra ContextoActor, autorización nominal V3, gobierno de
confianza, VEC-AD-3 y contratación. Comprueba el `COMMIT` único, repetición,
concurrencia, rotaciones y revocaciones, fallo inyectado, ACL, retirada
protegida y reconciliación negativa de cada pieza. Retira o altera de forma
aislada reserva, expediente, versión, actuación, auditoría, outbox, marcador,
refs, huellas y cadenas; cada replay debe fallar sin escribir y el runner
restaura exactamente los casos antes de continuar. La guía detallada está en
`deploy/postgresql/autorizacion_atestada_v3/README.md`.

Ese mismo runner comprueba además la instalación aditiva de `000006` a
`000012`, el materializado de la versión integral inicial, la preparación
reservada/reutilizada/conflictiva de análisis y la confirmación O3 completa.
La prueba positiva verifica versión 2, actuación, consumo de fuentes y
decisión, auditoría, outbox, recibo y replay exacto. También comprueba la
imposibilidad de leer o escribir directamente las tablas, el privilegio
runtime exclusivo de la función final y la retirada protegida.

El tercer runner fija PostgreSQL 18.4 por resumen, usa un contenedor efímero
sin puertos publicados y verifica el ciclo `15→16→15`, los órdenes de
instalación y retirada, ACL/RLS, topologías y puentes hostiles, tipos y límites
JSON, cadena inmutable, ocho escritores concurrentes y bloqueo de `000035.down`
con C2-B vacío o con historia.

El cuarto runner fija PostgreSQL 18.4 exacto por resumen y ejecuta las
dependencias reales `000001`–`000036`. Verifica instalación vacía y backfill
no vacío, barrera `16→17→16`, un único ganador entre migraciones concurrentes,
orden `COLLATE "C"`, extracción y cotejo del canon, RLS/ACL, índices e
inmutabilidad. Comprueba además primer ordinal `corte_base+1`, `ROLLBACK` sin
fantasma y con reutilización del ordinal, dos escritores bloqueados hasta el
primer `COMMIT`, ocho escritores sin huecos, safe-down con accesos C2-B
anteriores y rechazo de publicaciones posteriores, dependencias y
divergencias. El contenedor no publica red ni puertos y se elimina al salir.

El quinto runner añade `000038` y su delta DBA. Verifica la identidad exacta
de `pgcrypto`, privilegio mínimo, carreras de instalación y retirada,
encadenamiento de páginas 2 y 3, consumo único, revocación, RLS e
inmutabilidad. Ataca además restricciones y disparadores homónimos,
disparadores RI, reglas, columnas, índices, políticas, ACL y propietarios. La
reversión solo pasa con catálogo exacto e historia vacía.

## Reversión protegida

La reversión normal solo funciona sin historia. Destruir historia exige un
procedimiento formal, copia verificada y la confirmación explícita:

```text
vec.confirmar_destruccion_contratacion_temporal=
DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE
```

Esa variable nunca debe formar parte de una receta ordinaria de despliegue.
