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

Los dos ejemplos de esta guía son salida real del registro, pero de un
manejador sintético: rutas que existen en `vec-server`, un binario compilado
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
| `vec.bd.consultas` | Consultas de datos a PostgreSQL. No cuenta `BEGIN`, `COMMIT`, `SET` ni `set_config` |
| `vec.bd.duracion` | Tiempo dentro de PostgreSQL, incluidas esas órdenes de control |
| `vec.bd.espera_conexion` | Tiempo esperando una conexión libre del pool |
| `vec.lenta` | Más de 0,3 s o más de 20 consultas. Muchas consultas cortas suelen ser una consulta por fila en el código |
| `vec.bd.consulta_mas_lenta` | En las lentas, la función de PostgreSQL que más tardó y su duración |
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
3. Con `vec.correlacion` se encuentran las incidencias técnicas de esa misma
   petición: `podman logs <contenedor> 2>&1 | grep <correlacion>`.
4. En PostgreSQL, buscar la función en `pg_stat_user_functions` y
   `pg_stat_statements` (consultas 1 a 3 de
   `deploy/principal/consultas_observabilidad.sql`). Quién ocupa las
   conexiones: consultas 4 a 7. Para el plan, `auto_explain` un rato, como
   explica `deploy/principal/postgresql_observabilidad.md`.

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
