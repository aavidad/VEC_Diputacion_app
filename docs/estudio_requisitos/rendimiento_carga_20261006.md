# Aguante del portal público ante miles de personas (6 de octubre de 2026)

Este informe recoge lo que se ha medido en local, lo que se ha corregido y lo
que hace falta para que el portal público y las áreas personales aguanten
picos de miles de personas conectadas a la vez. Todas las pruebas se han hecho
en el equipo de desarrollo, contra una copia con datos sintéticos. No se ha
tocado cidonia ni ningún servicio externo.

## Contexto

La lentitud de la mañana del 6 de octubre en cidonia se debía sobre todo a un
bucle de 20 contenedores que consumía la CPU. Retirado, la sesión responde en
0,09 s y la lista interna de bolsas en 2,6 s (otro encargo la está
corrigiendo). Este informe no mide cidonia: mide cuánto aguanta el código
del portal público con picos de miles de personas.

## Objetivo

La regla de rendimiento del 6 de octubre pide que cada lectura responda en el
servidor en menos de 300 ms en el 95 % de los casos (p95) con volumen real, y
que la aplicación aguante miles de personas a la vez. Para el portal público se
ha usado esta vara de medir:

| Usuarios a la vez | Lecturas públicas (p95) | Errores |
|---|---|---|
| 100 | menos de 50 ms | 0 % |
| 1.000 | menos de 300 ms | menos de 0,1 % |
| 3.000 | menos de 1 s | menos de 1 % |

## Cómo se ha medido

El laboratorio está en `scripts/carga/` y se puede repetir:

```sh
scripts/carga/laboratorio_publico.sh preparar   # PostgreSQL 18.4, datos y servidor
go run ./scripts/carga/cliente -ca ~/.cache/vec-carga-publico/tls/ca.crt \
    -usuarios 100,500,1000,3000 -duracion 30s -pausa 1s \
    /api/publico/bolsa/bolsas '/api/publico/bolsa/bolsas/bolsa:carga-007/lista' \
    /api/publico/bolsa/convocatorias /api/publico/bolsa/categorias
docker stats vec-carga-publico-pg   # CPU de PostgreSQL, en otra terminal
scripts/carga/laboratorio_publico.sh retirar    # lo borra todo
```

- PostgreSQL 18.4 en un contenedor con 2 GB de memoria, datos en disco y
  `max_connections=100`, como la principal. Lleva instalada la proyección
  pública de Bolsa (`deploy/postgresql/bolsa_publica`) con 40 bolsas y 84.000
  posiciones sintéticas (entre 200 y 4.000 por bolsa) y las convocatorias de
  las pruebas de integración.
- El servidor es el binario real `cmd/vec-publico`, en modo producción, con
  TLS y la misma configuración que exige su composición. Se limitó a 4
  núcleos (y PostgreSQL a 4 núcleos) para parecerse a un servidor modesto.
- El cliente de carga (`scripts/carga/cliente`, un solo fichero corto) abre
  una conexión TLS propia por cada usuario, como haría un navegador distinto,
  recorre las rutas que se le pasan y da percentiles y errores por ruta. Solo
  admite destinos locales. La CPU y las conexiones se miraron con `docker
  stats`, `ps` y `pg_stat_activity`.
- Escenario «api»: cada usuario pide la relación de bolsas, una página de la
  lista de una bolsa, las convocatorias y las categorías, y espera entre 0,5 y
  1,5 s antes de repetir. Es un uso muy intenso: 1.000 usuarios así generan
  unas 3.000 peticiones por segundo, lo que harían unas 15.000 personas que
  consultan a ritmo normal (una página cada 5 s).
- Escenario «pagina-bolsa»: la visita completa a `/bolsa/` sin caché del
  navegador (HTML, 4 hojas de estilo, 4 scripts, textos y las 4 lecturas),
  repetida cada 2 a 6 s.

Las cifras se tomaron con una primera versión del cliente que traía estos dos
recorridos y medía también CPU y conexiones. En el repositorio queda la
versión mínima: se le pasan las rutas del recorrido como argumentos.

## Resultados antes de los cambios

Escenario «api», 4 núcleos para el servidor y 4 para PostgreSQL:

| Usuarios | Peticiones/s | Errores | Bolsas y lista (p95) | Convocatorias (p95) | CPU de PostgreSQL |
|---|---|---|---|---|---|
| 100 | 396 | 0,02 % | 4 ms | 4 ms | 111 % |
| 500 | 1.934 | 25 % | 28 ms | 18 ms (la mitad, rechazadas) | 414 % |
| 1.000 | 2.869 | 49 % | 285 ms | casi todas rechazadas con 429 | 416 % |
| 3.000 | 2.869 | 50 % | 1,8 s | casi todas rechazadas con 429 | 414 % |

Los errores son respuestas 429 («Inténtelo de nuevo en unos instantes») de
convocatorias y categorías: esas dos rutas solo admiten 6 operaciones a la vez
en todo el proceso y rechazaban al instante la séptima.

Con `pg_stat_statements` se vio en qué se iba PostgreSQL:

- El 73 % del tiempo era la comprobación de permisos que el adaptador público
  ejecuta cada vez que toma una conexión del pool (2,5 ms cada vez). Casi todo
  ese tiempo era preguntar por el permiso de ejecución de las 3.347 funciones
  internas de PostgreSQL antes de descartarlas por esquema.
- El 12 % era la lista de una bolsa: para servir una página de 50 posiciones
  se leían y validaban todas (2.088 filas de media, hasta 4.000).

## Cambios hechos (una PR cada uno)

1. Comprobación de permisos más barata. La consulta hace las mismas
   comprobaciones, pero primero descarta las funciones de los esquemas de
   PostgreSQL y después pregunta el permiso. Pasa de 2,5 ms a 0,6 ms. Se sigue
   ejecutando en cada préstamo de conexión. En el laboratorio, conceder un
   permiso de ejecución ajeno o crear una función en `public` hace que la
   siguiente petición falle con 503, igual que antes, y al retirarlo vuelve el
   200.
2. Lista de una bolsa por páginas. PostgreSQL entrega solo el tramo pedido.
   Antes comprueba, en la misma lectura, que las posiciones de la bolsa siguen
   siendo exactamente 1..total (recuento, mínimo y máximo sobre la clave
   primaria), así que la garantía de integridad es la misma que cuando se leía
   la lista entera. La búsqueda por documento sigue leyendo la lista completa.
   Se compararon 320 respuestas antes y después: idénticas byte a byte. Si
   falta una posición de la bolsa, cualquier página da 503, como antes.
3. Estáticos comprimidos. Los ficheros de texto (JS, CSS, JSON, HTML y SVG)
   se sirven con gzip cuando el navegador lo acepta. La versión comprimida se
   guarda en memoria (tope de 64 MiB) y se rehace si el fichero cambia. No se
   comprimen las respuestas de la API: solo contenido igual para todos y sin
   secretos, así que no abre ataques del tipo BREACH. La política de caché y
   las cabeceras de seguridad no cambian.
4. Espera breve antes de rechazar. Si los 6 cupos de convocatorias y
   categorías están ocupados, la petición espera su turno hasta 500 ms (dentro
   del plazo que ya tenía) antes de responder 429. Los topes de memoria no
   cambian: siguen siendo como mucho 6 operaciones a la vez. Como mucho 1.024
   peticiones esperan a la vez; a partir de ahí el 429 es inmediato, como
   antes.

Una revisión independiente dio GO a los cuatro, sin fallos graves. Sus
mejoras menores ya están incorporadas: tipos de fichero comprimibles
limitados a los que Go conoce, no recomprimir lo que no gana y el tope de
peticiones en espera. Quedan dos propuestas sin hacer: una prueba PostgreSQL
de la lista paginada en el arnés de integración (que hoy publica bolsas
vacías) y reescribir del mismo modo la comprobación de secuencias, que
depende del orden que elija el planificador.

## Resultados después de los cambios

Mismo escenario y mismos recursos:

| Usuarios | Peticiones/s | Errores | Bolsas y lista (p95) | Convocatorias (p95) | CPU de PostgreSQL |
|---|---|---|---|---|---|
| 100 | 398 | 0 % | 2 ms | 2,8 ms | 58 % |
| 500 | 1.988 | 0 % | 3,5 ms | 4,8 ms | 308 % |
| 1.000 | 2.780 a 2.999 | 0 % (una de cuatro tandas: 1,6 %) | 6 a 15 ms | 217 a 251 ms (esa tanda: 500 ms) | 413 % |
| 3.000 | 6.832 | 49 % | 101 ms | rechazadas tras 500 ms de espera | 417 % |

- A 500 usuarios todo responde en menos de 5 ms y sin errores.
- A 1.000 usuarios se cumple el objetivo en tres de las cuatro tandas
  medidas: cero errores, bolsas y listas por debajo de 15 ms y convocatorias
  entre 217 y 251 ms. En la cuarta, el 1,6 % de convocatorias y categorías
  esperó 500 ms y recibió 429. PostgreSQL está al límite con 4 núcleos.
- A 3.000 usuarios la relación de bolsas y las listas responden en unos
  100 ms, pero PostgreSQL está saturado y convocatorias y categorías vuelven
  a rechazar.
- El equipo tenía otros trabajos en marcha durante las mediciones; conviene
  repetirlas en una máquina sin más carga.

Visita completa a `/bolsa/` con 1.000 usuarios:

| | Antes | Después |
|---|---|---|
| Datos enviados | 54 MB/s | 15 MB/s |
| Peso de una visita sin caché | unos 218 KB | unos 60 KB |
| Memoria del servidor | 369 MB | 127 MB |
| CPU del servidor | 204 % | 151 % |
| Errores | 0,1 % | 0,02 % |

### Qué pasa con más núcleos

Con 8 núcleos para PostgreSQL y el pool de 6 conexiones, PostgreSQL no pasa
de 4,4 núcleos: el límite pasa a ser el pool. En una prueba solo local, con el
pool y los cupos subidos a 16, PostgreSQL usó los 8 núcleos, las listas
siguieron por debajo de 25 ms a 3.000 usuarios y los rechazos de
convocatorias bajaron del 43 % al 22 %. Ese cambio no se ha subido: los 6
cupos están calculados para respuestas de hasta 32 MiB cada una (192 MiB en
total) y subirlos exige revisar ese presupuesto de memoria.

## Portal interno: arranque

No se ha podido levantar el portal interno completo en local (necesita la
composición entera, con identidad, autorización y unos 14 accesos a la base).
Sí se ha medido lo que descarga al abrirse:

- `portal-empleado/index.html` carga, contando las importaciones de módulos,
  132 ficheros de estilo y código: 1,6 MB sin comprimir y 415 KB con gzip.
  Con el cambio 3 se envía la versión comprimida.
- Dos de cada tres importaciones de módulos no llevan `?v=` y se revalidan en
  cada carga (respuesta 304). Con HTTP/2 es asumible, pero son más de cien
  peticiones por persona al abrir el portal. Poner versión a todas las
  importaciones dejaría la segunda visita casi sin peticiones.

## Mi bolsa, ofertas y aceptación

Tampoco se ha podido medir de extremo a extremo: el proceso externo exige su
propio material de identidad, certificado por persona y toda la autorización
V3. Revisando el código y con una prueba aislada en PostgreSQL aparecen dos
límites claros.

Pools de 2 conexiones. El proceso externo abre Mi bolsa y la comprobación
previa de autorización con `MaxConns = 2`. Con 2 conexiones, si cada consulta
tarda 10 ms, el proceso atiende como mucho unas 200 por segundo; el resto
espera en cola.

Una sola cadena de auditoría para toda la aplicación. Cada operación con
autorización V3 (y con la auditoría total, también cada lectura de Mi bolsa)
actualiza la misma fila `control_cadena_auditoria` para encadenar su registro.
Todas esas transacciones pasan en fila india por esa fila hasta confirmar, y
en SERIALIZABLE las que chocan fallan y se repiten. Una prueba con `pgbench`
que reproduce ese patrón (actualizar la cabeza, insertar el registro y 5 ms de
trabajo más en la transacción) dio:

| Patrón | 10 clientes | 50 clientes | 90 clientes |
|---|---|---|---|
| Cabeza única, SERIALIZABLE | 160 op/s, 29 % fallan tras 20 intentos | 164 op/s, 71 % fallan | 164 op/s, 81 % fallan |
| Cabeza única, READ COMMITTED | | 156 op/s, 0 % fallan, 320 ms de media | |
| Sin cabeza común | | 7.850 op/s, 6 ms de media | |

Es decir: mientras exista una cabeza única, toda la aplicación queda por
debajo de unas 160 operaciones auditadas por segundo, sea cual sea el
servidor. La aplicación reintenta hasta 30 veces con esperas aleatorias, lo
que reduce los fallos pero alarga la respuesta. Para miles de personas
entrando a la vez en Mi bolsa tras un aviso, este es el primer límite.

## Lo que queda por decidir (necesita tu visto bueno)

1. Cadena de auditoría. Mantener el valor de prueba sin cabeza única: cada
   operación inserta su registro sin bloquear a las demás y un proceso de
   sellado los encadena en orden cada pocos segundos (ya existe un sello
   periódico), o varias cadenas en paralelo. Es un cambio de SQL y de
   seguridad, con su revisión.
2. Comprobación de permisos del portal público. Aun optimizada, sigue
   siendo el 80 % del trabajo de PostgreSQL público. Se puede hacer al abrir
   cada conexión y, además, cada pocos segundos en segundo plano, cerrando el
   pool si falla. Detectaría un permiso indebido con unos segundos de retraso
   en vez de en la petición siguiente. No se ha probado ni medido: desactivarla,
   aunque fuera en el laboratorio, es rebajar una guarda.
3. Caché corta de lecturas públicas. Las bolsas y convocatorias publicadas
   solo cambian al publicar. Una caché de 10 a 30 s en Caddy o en una CDN
   quitaría casi toda la carga de PostgreSQL en los picos, a cambio de que una
   publicación o una retirada tarde ese tiempo en verse.
4. Pools configurables. Que el tamaño de los pools del portal externo
   (ahora 2) y del público (6, junto con sus cupos y su presupuesto de
   memoria) se fije en la configuración del despliegue, no en el código.

## Lo que necesita el servidor real

- Procesos separados. El portal público (`vec-publico`) y el externo en su
  propio proceso y, si se puede, en otra máquina que el interno, para que un
  pico de aspirantes no frene a RRHH.
- PostgreSQL de la proyección pública aparte (ya está diseñado así) con al
  menos 8 núcleos, y una réplica de solo lectura si se quieren varios
  servidores públicos. La proyección es de solo lectura, así que una réplica
  en streaming sirve sin cambios.
- Conexiones. Con `max_connections=100` hay que sumar los pools de todos
  los procesos. Varios pools de Contratación temporal no fijan tamaño en el
  código y, si el DSN no lleva `pool_max_conns`, toman
  el valor por defecto de pgx (el mayor entre 4 y el número de núcleos), así
  que en un servidor de 32 núcleos cada uno podría abrir 32 conexiones. Hay
  que fijar tamaños explícitos o poner PgBouncer delante. PgBouncer en
  modo transacción necesita una adaptación: los adaptadores envían al conectar parámetros como
  `statement_timeout` o `default_transaction_read_only` y comprueban su valor
  en cada préstamo; habría que pasarlos a `ALTER ROLE ... SET` y probarlo antes.
- Ajustes de PostgreSQL: `shared_buffers` en torno al 25 % de la memoria,
  `effective_cache_size` al 70 %, `pg_stat_statements` y `track_io_timing`
  activos para ver qué consume, y `max_connections` acorde a los pools.
- Delante del portal público, Caddy (ya preparado en la PR #236) con
  compresión gzip/zstd para las respuestas de la API, y una CDN o caché para
  los estáticos con `?v=` (ya se sirven como inmutables).
- `GOMAXPROCS` igual a los núcleos asignados a cada contenedor.
- Prueba de carga antes de cada hito, con este mismo laboratorio, y una
  prueba de Mi bolsa y aceptación de extremo a extremo cuando el proceso
  externo pueda levantarse en local.
