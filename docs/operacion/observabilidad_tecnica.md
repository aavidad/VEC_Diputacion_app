# Seguir una petición lenta en VEC

Guía para Sistemas y para quien desarrolla. El registro técnico es distinto
de la auditoría de uso de datos: no guarda quién hizo la petición, ni valores
de la ruta, ni consulta, cabeceras o cuerpos.

## Dónde mirar

Cada petición a `vec-server` o `vec-admin` deja una línea JSON
con `"msg":"http.server.request"` en la salida de errores del proceso
(`podman logs <contenedor>`). Los nombres de campo siguen las convenciones
semánticas de OpenTelemetry, que entienden las herramientas habituales de
registros; los propios de VEC empiezan por `vec.`. Las duraciones van en
segundos.

Los dos ejemplos de esta guía son capturas históricas del registro de un
manejador sintético, anteriores a la lista positiva de alias: rutas que existen en `vec-server`, un binario compilado
con la revisión marcada, PostgreSQL 18.4 desechable con un pool de 2
conexiones y unas funciones de PostgreSQL de prueba (`vec_bolsa.listar_participaciones`
tarda 0,4 s; `vec_bolsa.consultar_participacion` se llama 30 veces, una por
fila). No proceden de la principal.

```json
{"time":"2026-10-06T16:18:06.94+02:00","level":"WARN","msg":"http.server.request","service.name":"vec-server","service.version":"72cf97a1c07e","deployment.environment.name":"desarrollo","vec.superficie":"interno","http.request.method":"GET","url.path":"/api/vec/bolsa/mi-bolsa/historial","http.response.status_code":200,"http.server.request.duration":0.4069,"http.response.body.size":0,"vec.correlacion":"fdc3a21b2ddcc57c2fd7149dade6cf24","vec.bd.consultas":31,"vec.bd.duracion":0.4036,"vec.bd.espera_conexion":0.0033,"vec.lenta":true,"vec.bd.consulta_mas_lenta":"vec_bolsa.listar_participaciones","vec.bd.consulta_mas_lenta.duracion":0.4014}
```

| Campo | Qué es |
| --- | --- |
| `level` | `INFO`; `WARN` si es lenta; `ERROR` si respondió 5xx o se interrumpió |
| `url.path` | El camino con cada tramo que pueda ser un valor cambiado por `{valor}`: solo quedan tramos de minúsculas, guion y guion bajo, y versiones como `v2`. Un 4xx sale como `{oculto}`, porque el camino puede ser lo que escribió la persona. Una palabra suelta en minúsculas sí pasa |
| `http.response.status_code` | Código de respuesta |
| `http.server.request.duration` | Lo que tardó el servidor, en segundos |
| `vec.bd.consultas` | Consultas normales y resultados de datos observados en lotes. No cuenta `BEGIN`, `COMMIT`, `SET` ni `set_config`; los callbacks de lote no prueban cuántos comandos llegó a ejecutar el servidor |
| `vec.bd.duracion` | Tiempo de las operaciones pgx, incluidas las órdenes de control. En lotes incluye el consumo del cliente hasta cerrar el resultado; no es tiempo exclusivo del servidor |
| `vec.bd.espera_conexion` | Tiempo esperando una conexión libre del pool |
| `vec.lenta` | Más de 0,3 s o más de 20 consultas. Muchas consultas cortas suelen ser una consulta por fila en el código |
| `vec.bd.consulta_mas_lenta` | Alias conocido de la consulta normal más lenta y su duración. Una operación desconocida se reduce a su verbo, por ejemplo `select`; un lote no se atribuye a su primer SQL |
| `vec.bd.error` | Último error con la base de datos: `bd_` y el código de PostgreSQL (`bd_57014` cancelada por tiempo, `bd_53300` demasiadas conexiones) o `conexion_plazo_vencido` si no llegó a conseguir conexión. Una consulta de datos correcta posterior lo borra (un error ya manejado no aparece); un ROLLBACK no |
| `error.type` | En un 5xx, `vec.bd.error` si lo hay; si no, el código de estado |
| `vec.cancelada` | `cliente` si quien llamó cortó antes; `plazo` si venció un plazo del servidor |
| `vec.correlacion` | Enlaza la línea con las incidencias técnicas de la misma petición |
| `service.version` | Revisión de Git del binario. `desconocida` si se compiló sin la marca |

Las consultas solo cuentan si el pool tiene el trazador. No lo tienen los pools
acreditados de Contratación temporal (consultas RRHH, resolución de motivos y
cobertura O4-05), que por seguridad rechazan cualquier trazador, ni el pool
público de Bolsa. Las peticiones que usan esos pools salen con
`vec.bd.consultas: 0` aunque consulten.

## Arranque de los procesos

`vec-server` y `vec-admin` escriben un registro JSON `vec.process.startup`
en su salida de errores al terminar la composición. `vec.arranque.duracion`
indica los segundos transcurridos en esa fase. Un
resultado `preparada` significa que el servidor está construido y va a abrir
la escucha; todavía no confirma que acepte conexiones. Si falla la
configuración, la composición o la escucha, se escribe `fallida`, con la fase
y una clase cerrada en `error.type`. El fallo de `vec-server` conserva además
su incidencia técnica común `ARRANQUE_FALLIDO`.

El registro JSON de arranque contiene servicio, versión, entorno y superficie.
No incluye el texto del error, rutas de archivos, credenciales ni DSN. La línea
fatal posterior indica la etapa, el componente que falló y un diagnóstico
saneado. Conserva los mensajes internos conocidos, como «material criptografico
de desarrollo invalido» o
«auditoria.intentos.configuracion_no_disponible». Para PostgreSQL muestra el
SQLSTATE, sin el mensaje de la base; para ficheros y red omite rutas, direcciones
y usuarios. Si el error es texto libre no catalogado, muestra su tipo técnico,
sin copiar el texto. Para encontrar los fallos y sus tiempos:

```sh
podman logs --since 1h <contenedor> 2>&1 | grep '"msg":"vec.process.startup"' \
  | jq -c '{servicio: .["service.name"], fase: .["vec.arranque.fase"], resultado: .["vec.arranque.resultado"], segundos: .["vec.arranque.duracion"], causa: .["error.type"]}'
```

La línea de `escucha` solo aparece si `ListenAndServe` devuelve un error. Si
se necesita acreditar disponibilidad, hay que hacer una petición de salud por
la entrada autorizada y comprobar la respuesta.

Si el portal tarda minutos y acaba en 503, lo primero es buscar esta forma:
cero consultas, todo el tiempo en `vec.bd.espera_conexion` y
`conexion_plazo_vencido`. Es un pool agotado.

```json
{"level":"ERROR","msg":"http.server.request","url.path":"/api/vec/personal/rpt/positions/{valor}","http.response.status_code":503,"http.server.request.duration":0.2004,"vec.bd.consultas":0,"vec.bd.duracion":0,"vec.bd.espera_conexion":0.2004,"vec.bd.error":"conexion_plazo_vencido","error.type":"conexion_plazo_vencido"}
```

(El segundo ejemplo se ha recortado para que quepa.)

## Paso a paso

1. Las lentas o fallidas de la última hora, agrupadas por ruta:
   ```sh
   podman logs --since 1h <contenedor> 2>&1 | grep '"msg":"http.server.request"' \
     | jq -r 'select(.["vec.lenta"] or .["http.response.status_code"] >= 500)
              | [.["url.path"], .["http.response.status_code"], .["error.type"]] | @tsv' \
     | sort | uniq -c | sort -rn | head
   ```
2. Si todas las rutas esperan conexión (`vec.bd.espera_conexion` alto), el
   problema es el pool o PostgreSQL. Si es una ruta, mirar su
   `vec.bd.consulta_mas_lenta`.
3. Si el alias es `select` u otro verbo, consultar los resúmenes de PostgreSQL: el registro de VEC no expone nombres SQL desconocidos. Con `vec.correlacion` se encuentran las incidencias técnicas de esa misma
   petición: `podman logs <contenedor> 2>&1 | grep <correlacion>`.
4. En PostgreSQL, buscar la función en `pg_stat_user_functions` y
   `pg_stat_statements` (consultas 1 a 3 de
   `deploy/principal/consultas_observabilidad.sql`). Quién ocupa las
   conexiones: consultas 4 a 7. Para el plan, `auto_explain` un rato, como
   explica `deploy/principal/postgresql_observabilidad.md`.

## Consultas y lotes de una petición

En una petición lenta o con estado HTTP de error (400 o superior),
`vec.bd.operaciones` resume las consultas normales por alias: `nombre`, `n`,
`total` y `maxima` en segundos, y recuentos `errores` por clase cerrada.
El resumen admite dieciséis nombres y un grupo `otras`; cada operación conserva
cuatro clases de error y otro grupo `otras`. La consulta más lenta mantiene su
medida propia aunque su grupo se haya resumido en `otras`.

Los alias proceden de una lista positiva de operaciones técnicas conocidas.
Una consulta normal sin alias se reduce a un verbo cerrado, como `select`, y suma
`vec.bd.operaciones_desconocidas`. La clasificación usa los primeros 2048 bytes:
es diagnóstica, no un análisis semántico del SQL. Nunca se registran SQL,
argumentos, identificadores dinámicos ni texto libre del error. Con la clasificación actual, la operación de Bolsa del ejemplo histórico se
registra como `select`, porque no está en esa lista. No se ha repetido ese ensayo.

Para los lotes, `vec.bd.lotes` cuenta cierres; `vec.bd.lote.resultados_observados`
cuenta callbacks de resultado y `vec.bd.lote.errores` los fallos observados.
`vec.bd.lote.duracion_hasta_cierre` suma segundos desde el envío hasta el cierre,
incluido el tiempo de consumo del cliente. pgx no ofrece un inicio individual
para cada resultado: no se atribuye el tiempo del lote a una consulta ni se
confunden los callbacks con todos los comandos enviados. Un cierre repetido no
suma otro lote.

Sistemas puede consultar los resúmenes sin mostrar SQL ni parámetros:

```sh
podman logs --since 1h <contenedor> 2>&1 | grep '"msg":"http.server.request"' \
  | jq -c 'select(.["vec.bd.operaciones"] or .["vec.bd.lotes"])
           | {ruta: .["url.path"], correlacion: .["vec.correlacion"],
              operaciones: .["vec.bd.operaciones"], lotes: .["vec.bd.lotes"],
              resultados: .["vec.bd.lote.resultados_observados"],
              segundos_lotes: .["vec.bd.lote.duracion_hasta_cierre"]}'
```

La cobertura depende del contexto de petición y de los pools instrumentados;
los pools acreditados que exigen `Tracer=nil` conservan esa guarda y no quedan
medidos por este trazador. La auditoría nominal mantiene su propia transacción,
actores y cadena; estos resúmenes técnicos no la sustituyen.

## Métricas y perfiles

Con `VEC_DIAGNOSTICO_ESCUCHA=127.0.0.1:9464` el proceso abre una escucha solo
para el bucle local (otra dirección no se admite). Desde dentro del
contenedor:

```sh
curl -s http://127.0.0.1:9464/debug/vars | jq '.vec_bd_pools, .vec_http_en_curso, .vec_http_lentas'
curl -s -o cpu.pprof 'http://127.0.0.1:9464/debug/pprof/profile?seconds=30'
curl -s 'http://127.0.0.1:9464/debug/pprof/goroutine?debug=2' | head -100
```

- `vec_bd_pools`: por rol de base de datos, `maximo`, `en_uso`, `libres`,
  `prestamos_con_espera` y `espera_ms`. Si `en_uso` iguala a `maximo` un rato,
  el pool está agotado.
- `vec_http_peticiones` y `vec_http_ms`: recuento y milisegundos acumulados por
  ruta. `vec_http_lentas` y `vec_http_en_curso`: lentas desde el arranque y
  peticiones abiertas ahora.
- `goroutine?debug=2`: dónde está parado cada hilo de Go si el proceso no
  responde. El perfil de CPU se lee en un equipo con Go:
  `go tool pprof -top cpu.pprof`.

## Ajustes

| Variable | Por defecto | Qué cambia |
| --- | --- | --- |
| `VEC_TELEMETRIA_LENTA_MS` | 300 | Milisegundos para considerar lenta una petición |
| `VEC_TELEMETRIA_LENTA_CONSULTAS` | 20 | Consultas para considerarla lenta |
| `VEC_DIAGNOSTICO_ESCUCHA` | vacía | Escucha de métricas y perfiles (solo bucle local) |
| `VEC_ENTORNO` | perfil de ejecución | `desarrollo`, `pruebas`, `presentacion` o `produccion` |

## Compilar con la revisión

```sh
go build -buildvcs=false \
  -ldflags "-X vec-diputacion-granada/internal/shared/telemetria.Revision=$(git rev-parse --short=12 HEAD)" \
  -o vec-server ./cmd/vec-server
```

`deploy/principal/desplegar.sh`, `scripts/arrancar_vec_desarrollo.sh` y el
`Dockerfile` (con `--build-arg VEC_REVISION=…`) ya lo hacen.

## Límites

- Una consulta cuenta para su petición si el código usa el contexto de la
  petición.
- No se miden los pools acreditados de Contratación temporal ni el pool
  público de Bolsa (ver arriba).
- `vec-interno` y `vec-publico` todavía no escriben línea de acceso. El
  público tiene una lista positiva de dependencias y ampliarla es una decisión
  aparte.
