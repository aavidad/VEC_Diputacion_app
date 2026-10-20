# Acta de rendimiento de lecturas (F1-34), 20 de octubre de 2026

Tarea F1-34 del tablero: lecturas por debajo de 300 ms en el percentil 95
(cuadro de Contratación temporal, lista, ficha, bolsas, candidatos y ficha
del candidato), medidas por HTTP en copia local con el volumen de la
principal. Esta acta sustituye a la del 16/10, que no pudo medir por HTTP
candidatos ni la ficha del candidato.

Resumen. Con una sola persona y las peticiones de una en una, cinco de las
seis pantallas quedan por debajo de 300 ms en el p95; la lista de CT queda
justo debajo (275 ms) y su p99 lo pasa (317 ms). La ficha del candidato falla
por otra causa: el navegador lanza seis peticiones a la vez y entre una cuarta
y una tercera parte de las aperturas recibe un 503 en alguna sección por
interbloqueos en la cadena de autorización. Con diez clientes a la vez con
la misma identidad, las tres pantallas de CT pasan de 700 ms y el cuadro
devuelve 503 en tres de cada cuatro peticiones. Bolsas y candidatos aguantan
la concurrencia. El tiempo se va en la acreditación del pool de motivos, en
el recibo atestado de cada lectura y en los bloqueos de las filas de la
persona; las consultas de datos cuestan poco.

## Copia y método

- Código: rama de la PR #1189 con `origin/main` fusionado
  (`0b936524be29e8836c3bfe82346a10143b72a323`); binario compilado por la copia
  desde el commit de fusión `94c633604`.
- Copia nueva `d-claude-f134`, creada con `deploy/principal/copia_local.py
  crear --modulos --rrhh-original … --lista-sql
  deploy/principal/lista_sql_main_tras_hz37.txt` (124 SQL de `main` posteriores
  a HZ37) y los productores privados de roles y datos de la copia de
  presentación. Sin Administración: ninguna de las seis lecturas la necesitó.
- Volumen de la semilla, sin añadir datos: 86 expedientes de CT, 13 bolsas,
  434 entradas de constitución, 652 situaciones, 19 contactos y 10
  llamamientos emitidos en 6 bolsas. La bolsa más grande tiene 42
  candidatos y es la que se midió (Auxiliar de servicios generales). Es el
  volumen de la principal, no el de producción.
- PostgreSQL 18.4 en contenedor con 2 CPU y 2 GB, `shared_buffers` de 128 MB
  y JIT activado. La aplicación corre fuera del contenedor. En el equipo había
  otras ocho copias vivas y la carga media osciló entre 5 y 10.
- Peticiones con el cliente `scripts/carga/cliente` (mTLS, CA verificada,
  conexión persistente por usuario): 3 calentamientos y 50 muestras en serie
  (`-usuarios 1`), y 10 clientes con 50 muestras cada uno (`-usuarios 10`, 500
  peticiones). Todos usan la identidad de RRHH de la copia, que no tiene otra
  persona con los mismos permisos. Los percentiles son de rango más cercano y
  solo cuentan respuestas 200.
- Cuerpos y rutas copiados del navegador: el Chrome del sistema, con
  Playwright, abrió la lista de candidatos y una ficha, y se registraron sus
  peticiones. La ficha del candidato lanza a la vez seis GET: `sanciones`,
  `contratos`, `operaciones`, `reglas-situacion`,
  `contactos?llamamiento_ref=…` y `datos-contacto`. El cliente mide
  peticiones sueltas, así que la apertura completa se midió con un guion
  aparte que lanza las seis en paralelo y toma el tiempo hasta la última
  respuesta.
- Consultas y tiempo en base por petición: registro técnico de la aplicación
  (`vec.bd.consultas`, `vec.bd.duracion`). Planes: `auto_explain` con
  `log_analyze`, `log_buffers` y `log_nested_statements`, activado solo en una
  pasada aparte de 3 peticiones por pantalla para no alterar los tiempos.
  Tiempo en Go: perfil de CPU y traza de `pprof` en la escucha local de
  diagnóstico (`VEC_DIAGNOSTICO_ESCUCHA`).

## Resultados

Tiempos en milisegundos (p50 / p95 / p99). La columna de 10 a la vez es la
pantalla medida sola con 10 clientes; la ficha del candidato se midió con el
guion de aperturas.

| Pantalla | Peticiones y consultas | En serie | 10 a la vez | Errores con 10 a la vez | ¿p95 por debajo de 300 ms? |
| --- | --- | ---: | ---: | --- | --- |
| Cuadro de CT (`resumen: true`, límite 100) | 1 petición, 27 consultas | 197 / 249 / 275 | 672 / 708 / 732 | 374 de 500 con 503 | Solo en serie |
| Lista de CT (límite 100) | 1 petición, 27 consultas | 193 / 275 / 317 | 1.866 / 2.235 / 3.181 | ninguno | Solo en serie, al límite |
| Ficha de CT (versión 9) | 1 petición, 26 consultas | 134 / 180 / 255 | 741 / 1.777 / 2.752 | 1 de 500 con 404 | Solo en serie |
| Lista de bolsas | 1 petición, 4 consultas | 15 / 21 / 26 | 82 / 90 / 93 | ninguno | Sí |
| Candidatos de una bolsa (50 por página, 42 personas) | 1 petición, 17 consultas | 53 / 85 / 100 | 173 / 194 / 203 | ninguno | Sí |
| Ficha del candidato (apertura completa) | 6 peticiones a la vez, 62 consultas | 104 / 306 / 1.211, con 12 de 50 aperturas con algún 503 | 6.606 / 9.484 / 10.428 | 432 de 500 aperturas con 503 o 403 | No |

Detalle de la ficha del candidato. Cada sección por separado, en serie, tarda
entre 28 y 58 ms de mediana y entre 37 y 88 ms de p95 (`reglas-situacion`,
0,5 ms). `datos-contacto` hace 8 consultas, `reglas-situacion` 1 y las otras
cuatro 13 o 14. Medidas solas con 10 clientes, `sanciones` da 268 ms de p95 y
`datos-contacto` 182 ms. El fallo aparece al combinarlas: en una segunda
pasada de 50 aperturas de una sola persona fallaron 17 (p95 de las completas,
1.088 ms), 18 veces en `datos-contacto` y una en `operaciones`.

Avisos de Bolsa (`/api/vec/bolsa/avisos`, no es una de las seis pantallas):
21 / 23 / 24 ms en serie y 104 / 153 / 158 ms con 10 clientes, sin errores.

En la tanda conjunta de las once lecturas con 10 clientes a la vez, los 503
alcanzaron a todas las rutas con autorización atestada, incluida la lista de
bolsas (283 de 500), que sola no falla.

## Por qué tardan las pantallas que pasan de 100 ms

Cuadro y lista de CT, unos 195 ms por petición. Según la traza de Go, unos
150 ms son espera a PostgreSQL y entre 47 y 62 ms son CPU, sobre todo
decodificar y validar JSON y expresiones regulares de referencias. La espera a
la base se reparte así, según `auto_explain`:

- Entre 61 y 66 ms en la acreditación del pool de motivos de RRHH
  (`acreditarConexionResolucionMotivosRRHH`, en
  `internal/modules/contrataciontemporal/adapters/postgres/acreditacion_pool_resolucion_motivos_rrhh.go:596`).
  En cada petición repite una consulta sobre `pg_roles`, `pg_namespace`,
  `pg_proc` y las ACL, con cuatro bloqueos consultivos, para comprobar que el
  catálogo no ha cambiado. Es la sentencia más cara de la petición y no lee
  ningún dato de expedientes.
- Entre 82 y 86 ms en `consultar_cuadro_rrhh_atestado_v5`, que encadena v4,
  v3, v2 y v1 (unos 10 ms entre envoltorios). Dentro, unos 20 ms son el
  `INSERT` en `prueba_resultado_recibo_rrhh_v2` (el recibo de la lectura) y
  unos 8 ms el consumo de la decisión (`consumir_consulta_rrhh_v3_interna`).
  La lectura de los 86 expedientes es lo de menos.
- Unos 8 ms en el registro del contexto del actor
  (`resolver_y_registrar_contexto_actor_v2`), fuera de la consulta atestada.

El registro técnico solo atribuye 21 ms a la base en estas peticiones; no
parece contar el pool de motivos ni los de autorización.

Ficha de CT, unos 135 ms. La misma acreditación (61 a 64 ms) más
`consultar_detalle_rrhh_atestado_v1`, de 28 a 33 ms, casi todo en
`consultar_detalle_rrhh_con_bolsa_v1` con `numero_personas_detalle_rrhh_ct225`.

Ninguna sentencia de estas tres pantallas hace un recorrido secuencial caro.
El mayor nodo de datos es un `Append` de 22 ms dentro de la acreditación,
sobre tablas de catálogo. No hace falta ningún índice.

## Fallos por concurrencia (para una tarea aparte)

Las lecturas escriben en tablas de la persona (decisión, consumo, uso del
contexto), y dos lecturas simultáneas de la misma persona chocan. Con más
máquina seguiría fallando.

1. Ficha del candidato, una sola persona. Interbloqueo en
   `vec_contexto_actor_v1.proyeccion_cuenta_actual`, la fila de la cuenta.
   `datos-contacto` llama a
   `vec_bolsa_llamamientos.consultar_origen_datos_contacto_participacion_rrhh_v1`,
   que pasa por `vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1`
   y `vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1`
   (`FOR SHARE OF va,ca,pa,xa,vc,oa`, migración 000040 de contexto_actor_v1,
   rehecha en 000042 y 000043). A la vez, las otras secciones escriben en esas
   filas con `vec_autorizacion.registrar_decision_contexto_actor_v3`,
   `acreditar_uso_registro_contexto_actor_v2` y
   `acreditar_uso_registro_contexto_interno_v2`. En 50 aperturas PostgreSQL
   registró 24 `deadlock detected` (esperas de `ShareLock` de transacción y de
   `RowShareLock` sobre la tupla de la cuenta) y 3 fallos de serialización.
   Una de cada tres o cuatro aperturas enseña una sección en error.
2. Cuadro, lista y ficha de CT con varias peticiones a la vez de la misma
   persona. `vec_autorizacion.revalidar_decision_contexto_actor_v3_viva`
   (migración 000007 de autorización) toma `FOR UPDATE OF actual` sobre
   `vec_autorizacion.asignacion_perfil_actual` y la transacción serializable
   falla con 40001 (`could not serialize access due to concurrent update`).
   En la tanda conjunta hubo 543 de estos errores y 126 interbloqueos; la
   aplicación los devuelve como 503 (`etapa=autoridad_contexto` o
   `etapa=sql sqlstate=40001`). Cuando reintenta y acaba bien, como en la
   lista, la mediana sube a 1,9 s.

Recomendación, sin implementarla y con revisión SQL y de seguridad: el mismo
orden de bloqueo en las funciones de contexto y de decisión (o no bloquear la
fila de la cuenta en lecturas), un reintento acotado de 40001 y 40P01 en el
adaptador de lectura con la misma clave, y que la ficha del candidato pida
sus secciones en una sola petición con una sola decisión. Los índices y la
paginación no lo arreglan.

## N+1 de los avisos de Bolsa

Sigue igual que el 16/10. La aplicación usa `consultar_avisos_rrhh_v3`
(Bolsa 000042), que llama a `consultar_avisos_rrhh_v2` (000041), y esta toma
de `consultar_avisos_rrhh_v1` (000020) los saltos de orden. En v1 sigue el
`CROSS JOIN LATERAL vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(l.bolsa_ref,l.emitido_en)`
por cada llamamiento emitido (`000020_avisos_rrhh.up.sql:39`). B139
(`000139_avisos_modalidades.up.sql`) solo cambia el disparador
`crear_aviso_oferta_b129` de las ofertas publicadas y no toca esta lectura.

Plan con los 10 llamamientos de la copia: `Function Scan on
leer_orden_vigente_bolsa_v1 … loops=10`, 2,6 ms por vuelta, 26 de los 38 ms de
`consultar_avisos_rrhh_v3`. Además, v2 solo usa las filas `salto_orden` de v1,
pero v1 calcula también sus avisos de tres años, que v2 descarta y calcula de
nuevo con sus parámetros. El coste crece en línea con los llamamientos
emitidos. Propuesta para una tarea aparte: leer en bloque el orden de todos
los pares de bolsa y fecha de emisión, en una migración nueva de Bolsa con su
revisión SQL.

## Conclusión y qué se recomienda

F1-34 no se puede cerrar todavía. En serie y con el volumen de la principal
cumplen todas las pantallas salvo la apertura completa de la ficha del
candidato, y la lista de CT queda al límite (p95 275 ms, p99 317 ms). Con
concurrencia de la misma persona fallan la ficha del candidato y las tres de
CT.

Recomendaciones, de mayor a menor efecto, sin implementarlas:

1. Concurrencia de la cadena de autorización (apartado anterior). Es la que
   produce errores visibles; se arregla en las consultas y en el orden de los
   bloqueos.
2. Acreditación del pool de motivos: comprobar el catálogo una vez por
   conexión, o cuando cambie una versión de catálogo, en lugar de en cada
   petición. Quitaría unos 63 ms de cada lectura de CT, un tercio del cuadro.
   Toca autorización y necesita revisión de seguridad.
3. Recibo atestado de la lectura: averiguar por qué el `INSERT` en
   `prueba_resultado_recibo_rrhh_v2` tarda 20 ms con 86 expedientes.
4. Paginación del cuadro y la lista de CT por cursor en SQL, como proponía el
   acta del 16/10. Con 86 expedientes no se nota; con miles, sí.
5. N+1 de avisos, con la lectura en bloque descrita arriba.

No se propone ningún índice. Una observación menor: las secciones de la ficha
del candidato esperan unos 10 ms por conexión en cada petición
(`vec.bd.espera_conexion`) aunque vayan de una en una.

## Límites de esta medida

Hay una sola identidad de RRHH, así que los 10 clientes son la misma persona:
representa la ficha del candidato y el uso con varias pestañas, no diez
personas distintas. El volumen es pequeño (86 expedientes y 42 candidatos por
bolsa) y no se añadieron datos. El equipo estaba cargado por otras copias. Los
planes salen de una pasada aparte y no de las mismas peticiones
cronometradas.
