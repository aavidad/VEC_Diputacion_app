# Acta de rendimiento de lecturas (F1-34), 16 de octubre de 2026

Tarea F1-34 del tablero: lecturas por debajo de 300 ms en el percentil 95
(cuadro de Contratación temporal, lista, ficha, bolsas, candidatos y ficha del
candidato) y base común por debajo de 100 ms, medidas en copia local con el
volumen de la principal.

Resumen: en la base de datos todas las lecturas medibles quedan muy por debajo
del objetivo (p95 entre 0,1 y 15 ms en serie). Lo que acerca el cuadro y la
lista de CT a los 300 ms no son sus consultas de datos, sino la cadena de
identidad y autorización que se repite en cada petición, y el trabajo en Go
que la acompaña. Candidatos y ficha del candidato no tienen todavía medida
HTTP; por su número de consultas por apertura son las de más riesgo.

## Copia y método

Copia `y-claude-f134` creada con `deploy/principal/copia_local.sh crear`
desde el volcado HZ37 (principal del 14/10), con las listas
`candidata_hz39/lista_sql.txt`, `candidata_hz39/lista_sql_firma_vigencias.txt`,
`candidata_hz39/personal/lista_sql.txt` y
`lista_sql_codexs_varias_plazas_20261015.txt`, en ese orden, sobre
`origin/main` en `cf5f348a4`. Después se aplicaron a mano, una vez y sin
DOWN, las demás migraciones de `main` posteriores a `44218865a`: AD266,
AD282, AD285, CT226, CT232 y el bloque del justificante de
`lista_sql_codexo_justificante_limpio_20261015.txt` (sin AD263 ni AD265,
que ya traía la lista 1). Entraron todas salvo B119, D23, B123, B124, D26,
B125, D27 y B130, que exigen antes los LOGIN documentales que crea el DBA y
fallaron en su comprobación previa. Ninguna de ellas define funciones que
lean las seis pantallas medidas.

Volumen de la principal en la copia: 86 expedientes de CT con 217
publicaciones, 13 bolsas constituidas, 434 entradas de constitución,
340 situaciones de participación, 10 llamamientos emitidos y ningún vínculo
de emisión CT con Bolsa. Es poco volumen: estas cifras dicen cuánto cuesta
hoy la principal, no cómo crecerá con miles de expedientes.

PostgreSQL 18.4 en contenedor con 2 CPU y 2 GB, `shared_buffers` de 128 MB,
JIT activado y `ANALYZE` hecho tras restaurar. El equipo tenía otras copias y
compilaciones en marcha (carga media de 8). Cada lectura se repitió 50 veces
en serie (`pgbench -c 1 -t 50`) y 50 veces por cliente con 10 clientes en
paralelo (`-c 10 -t 50`, 500 transacciones). Mediana y p95 salen de la
latencia de cada transacción. Con 10 clientes sobre 2 CPU, el límite de CPU
del contenedor produce paradas de unos 80 ms; por eso las cifras en paralelo
de lecturas de 1 ms saltan a 40 u 85 ms en el p95. Las cifras en serie son
las que miden la consulta.

Las fachadas de lectura de CT y de Bolsa con material VEC-AD-3 no se pueden
llamar a mano: cada llamada consume una autorización de un solo uso, firmada
y ligada a la sesión. Se midieron sus consultas internas de datos por
separado, como propietario del módulo y dentro de una transacción que se
revierte. Las lecturas de Bolsa sin VEC-AD-3 se llamaron con un LOGIN de
medición creado solo en la copia (`medicion_bolsa`, miembro únicamente de
`vec_bolsa_llamamientos_ejecutor`; `medicion_recuperador`, miembro
únicamente de `vec_bolsa_importacion_convoca_recuperador`). Los planes se
sacaron con `auto_explain` (`log_analyze`, `log_buffers`,
`log_nested_statements`) para ver las sentencias dentro de cada función.

Medición HTTP: no fue viable en el tiempo fijado. La copia del guion arranca
sin conexiones nominales ni módulos (F0-28, primer corte) y no existe en el
equipo una copia arrancada que sirva CT y Bolsa de RRHH: las de `codexb-f141`
sirven el portal externo y su lectura modificaría su auditoría. Prepararla
exige el kit de administradores, la confianza ADMIN y los LOGIN nominales
(apartados 4 y 5 del LEEME). Para la lista de bolsas sí se midió el manejador
Go real contra la copia, sin TLS ni autorización, que es como funciona esa
ruta (autorización nominal en memoria). Las cifras HTTP de las demás
pantallas son las de las actas del 8 y el 12 de octubre, sobre otras copias
con 5.071 expedientes.

## Resultados

| Pantalla | Consultas por petición | Datos en base: mediana / p95 en serie | p95 con 10 en paralelo | p95 HTTP conocido | ¿Cumple? | Causa principal |
| --- | --- | ---: | ---: | --- | --- | --- |
| Base común (sesión, módulos) | ninguna a base en la lectura | - | - | 0,23 y 0,32 ms (12/10) | Sí | En memoria |
| Cuadro de CT (`resumen: true`) | 12 a 29, más 1 o 2 de calendarios por año | totales 0,28 / 0,40 ms; resumen 1,00 / 1,55 ms; página 0,39 / 0,67 ms | 0,56; 41; 1,2 ms | 278,8 ms (12/10, 5.071 exp.) | Al límite | Identidad, perfil y gobierno V3 en cada petición; en Go el resto |
| Lista de CT (límite 100) | igual que el cuadro | página de 100: 0,46 / 0,71 ms | 5,1 ms | 298,2 ms (12/10) | Al límite | Igual; además la página agrega todo el corte en memoria |
| Ficha de CT | igual, sin calendarios | detalle y agregado: 3,12 / 3,78 ms | 85 ms (CPU) | 140,5 ms (12/10) | Sí | Cadena de autorización |
| Lista de bolsas | 4 (más 1 o 2 de avisos) | 13,2 / 14,5 ms; avisos 12,4 / 15,3 ms | 97 y 102 ms | manejador real: 16,8 / 18,9 ms en serie; 86 / 99 ms con 10 a la vez | Sí | Avisos: una lectura del orden por cada llamamiento (N+1 en SQL) |
| Candidatos de una bolsa | unas 20 con una página de contactos; 9 más por cada 100 contactos | datos 5,5 / 10,2 ms; importación 1,0 / 1,7 ms; contactos 0,09 / 0,14 ms | 89; 32; 0,14 ms | sin medida (503 el 12/10) | Sin medir | Lee la bolsa entera y pagina en Go; un consumo V3 por cada 100 contactos |
| Ficha del candidato | 5 a 8 peticiones, 10 a 12 consultas cada una (60 a 90 por apertura) | datos de las secciones: 0,74 / 1,09 ms | 5,3 ms | sin medida | Sin medir, riesgo alto | Cada sección repite sesión, gobierno V3 y consumo propio |

Las tres pantallas de CT y la lista de bolsas no tienen N+1 por fila en Go.
En SQL sí hay uno en los avisos de Bolsa. Las funciones
`motivo_relacion_oferta_v1` y `eventos_relacion_oferta_v1` no están en `main`:
vienen en la PR #1144 (B118) y se llaman fila a fila dentro de
`leer_orden_contextual_oferta_v1` (`000118_criterios_contextuales_ofrecimiento.up.sql:428`
y `:447`), que lee el nuevo llamamiento, no ninguna de estas seis pantallas.

## Por qué tarda cada una

Cuadro, lista y ficha de CT. Con 5.071 expedientes, el acta del 8/10 midió
194 ms en el servidor y 23 ms acumulados en base (27 consultas). Hoy, con el
volumen de la principal, las consultas de datos suman menos de 2 ms (cuadro)
y 4 ms (ficha). El resto es la cadena común: registrar la sesión,
revalidarla, resolver el contexto del actor, acreditar el pool de motivos,
resolver el motivo, leer la confianza V3 (4 o 5 consultas, en cada
operación) y registrar la decisión antes de la lectura atestada. En la vía de
perfil dinámico del kit sintético se añaden unas 18 consultas más, 8 de ellas
`INSERT` que escriben de verdad en cada petición
(`internal/app/bootstrap/autoridad_postgresql_desarrollo.go:134` y
`:376-690`). El cálculo de firmas y plazos en Go no se ha perfilado.

Con más volumen pesa además la página: `materializar_cuadro_rrhh_v1` reúne
todas las filas filtradas en un arreglo y luego lo corta al límite
(`deploy/postgresql/contratacion_temporal/migraciones/000044_componentes/040_materializacion_cuadro.sql:214`,
cuerpo reescrito por CT187). Con 86 expedientes son 0,7 ms; con 5.170
publicaciones el plan del módulo midió 204,5 ms de p95 para v5
(`docs/plan_modulos/contratacion_temporal.md:68`).

Lista de bolsas. Una transacción de lectura con cuatro funciones. La más
cara es `leer_resumen_situaciones_bolsas_v1` (5 ms en caliente). Los avisos
cuestan más que la lista: `consultar_avisos_rrhh_v1` llama a
`leer_orden_vigente_bolsa_v1` una vez por llamamiento emitido con
`CROSS JOIN LATERAL`
(`deploy/postgresql/bolsa_llamamientos/migraciones/000020_avisos_rrhh.up.sql:39`).
El plan lo muestra: 10 llamadas de 1,45 ms, 14,5 de los 16,9 ms de esa
sentencia. Crece en línea con los llamamientos emitidos.

Candidatos. `respuestaCandidatos`
(`internal/app/bootstrap/bolsa_rrhh_desarrollo.go:576`) recibe la bolsa entera
y filtra y pagina en memoria aunque la pantalla pida 50. Los contactos se leen
en un bucle de páginas de 100 (`bolsa_rrhh_desarrollo.go:415-432`), y cada
página repite admisión de la bolsa, lectura del gobierno V3, registro de la
decisión y una transacción serializable con consumo de un solo uso. Además,
`salidaCandidata` recorre todos los contactos por cada candidata
(`bolsa_rrhh_desarrollo.go:718-731`): coste N por C en Go.

Ficha del candidato. `abrirFicha`
(`web/static/portal-empleado/portal-bolsas-api.js:841-861`) lanza entre 5 y 8
peticiones: sanciones, operaciones, reglas, contratos, datos de contacto,
contactos, reincorporaciones y solicitudes documentales. Cada una repite la
sesión (S1 a S3), la admisión de la bolsa, el gobierno V3, el registro de la
decisión y un consumo propio; tres de ellas añaden además auditoría síncrona
de intento. Las consultas de datos de todas las secciones suman 1 ms. En
`ListarContactosParticipacion` se resuelve dos veces el mismo contexto
(`internal/modules/bolsa/application/contacto_participacion.go:113` y `:116`).

## Propuestas

No se ha aplicado ningún cambio de SQL ni de código. Ninguna propuesta quita
permisos ni auditoría ni guarda en caché datos sujetos a permisos.

1. Avisos de Bolsa: sustituir la llamada por fila a
   `leer_orden_vigente_bolsa_v1` en `000020_avisos_rrhh.up.sql:39` por una
   lectura en bloque del orden de todas las bolsas con llamamientos emitidos
   (una función que devuelva el orden de un conjunto de pares bolsa y corte).
   Hoy son 14,5 ms con 10 llamamientos; en una migración nueva de Bolsa, con
   su revisión SQL.
2. Candidatos: llevar el filtro y la página a SQL y leer solo las
   participaciones de la página (`bolsa_rrhh_desarrollo.go:576`); pedir los
   contactos de esas participaciones en una sola llamada con una sola
   decisión V3 en lugar de una por cada 100 (`:415-432`); y agrupar los
   contactos por participación en un mapa antes de `salidaCandidata`
   (`:718-731`).
3. Ficha del candidato: una única decisión y un único consumo para abrir la
   ficha, con una lectura agregada de las secciones de solo consulta
   (sanciones, operaciones, contratos, reincorporaciones). Si se mantienen
   peticiones separadas, reutilizar la sesión resuelta en lugar de repetir
   S1 a S3 en cada una. Quitar la segunda resolución de contexto de
   `contacto_participacion.go:116`.
4. Gobierno V3: la publicación de confianza está versionada e inmutable por
   punto de control. Se puede memorizar por punto de control y leer solo el
   puntero en cada operación, en vez de 4 o 5 consultas
   (`internal/app/bootstrap/contratacion_temporal_confianza_renovable_desarrollo.go:410-470`).
   Es un catálogo versionado, no un dato con permisos. Toca autorización:
   necesita revisión de seguridad.
5. Cuadro y lista de CT: limitar la página en SQL con orden por
   `actualizado_en` y `expediente_ref` y cursor por clave, en lugar de
   agregar todo el corte en un arreglo
   (`000044_componentes/040_materializacion_cuadro.sql:214`). No cambia nada
   con el volumen de la principal; es lo que más pesa con 5.000 expedientes.
6. Vía de perfil dinámico del kit: evitar los 8 `INSERT` por petición
   (`autoridad_postgresql_desarrollo.go:376-690`) reutilizando la instantánea
   publicada mientras siga vigente. Solo afecta a desarrollo, pero es la vía
   con la que se han hecho las mediciones HTTP.
7. Posible fuga de memoria: `instantaneasPorSolicitud`
   (`internal/app/bootstrap/contratacion_temporal_alta_instantaneas_desarrollo.go:206-210`)
   añade una entrada por petición con la correlación en la clave y no se
   encontró dónde se borra. Conviene confirmarlo con un perfil de memoria.

Índices: con este volumen ninguna consulta medida hace un recorrido
secuencial costoso; los mayores son de 434 filas en memoria. No se propone
ningún índice nuevo.

## Qué falta para cerrar F1-34

La medición HTTP en una copia con CT y Bolsa de RRHH compuestas, con 50
peticiones en serie y 10 en paralelo y el número de consultas de cada
petición en el registro técnico, sobre todo para candidatos y ficha del
candidato, que no tienen ninguna medida válida. También perfilar el tiempo en
Go del cuadro de CT, que es la parte desconocida de sus 280 ms.
