# Seguir una petición lenta en VEC

Guía para Sistemas y para quien desarrolla. El registro técnico es distinto
de la auditoría de uso de datos: no guarda quién hizo la petición, ni valores
de la ruta, ni consulta, cabeceras o cuerpos.

## Dónde mirar

Cada petición a `vec-server`, `vec-admin` o `vec-publico` deja una línea JSON
con `"msg":"peticion"` en la salida de errores del proceso (`podman logs
<contenedor>`). Ejemplo de una prueba con PostgreSQL 18.4 y datos sintéticos:

```json
{"time":"2026-10-06T12:10:57.44+02:00","level":"WARN","msg":"peticion","servicio":"vec-server","superficie":"interno","entorno":"desarrollo","version":"54ac9a518abc","correlacion":"dcc9372a9fe22c7257a8eddf75f4603c","metodo":"GET","ruta":"/api/vec/bolsa/{bolsa}/participaciones","estado":200,"duracion_ms":406.5,"bytes":0,"bd_consultas":31,"bd_ms":403.1,"bd_espera_ms":3.4,"lenta":true,"consulta_mas_lenta":"vec_bolsa.listar_participaciones","consulta_mas_lenta_ms":401.6}
```

- `level`: `INFO`; `WARN` si es lenta; `ERROR` si respondió 5xx o se
  interrumpió.
- `ruta`: la plantilla de la ruta. Donde iba una referencia sale `{bolsa}` o
  `{valor}`. Un 4xx sin plantilla sale como `{sin_plantilla}`.
- `duracion_ms`: lo que tardó el servidor.
- `bd_consultas` y `bd_ms`: consultas a PostgreSQL y tiempo dentro de ellas.
  `bd_espera_ms`: tiempo esperando una conexión libre del pool.
- `lenta`: más de 300 ms o más de 20 consultas. Muchas consultas cortas
  suelen ser una consulta por fila en el código. `consulta_mas_lenta` es la
  función de PostgreSQL que más tardó.
- `bd_error`: último error de base de datos. `bd_` y el código de PostgreSQL
  (`bd_57014` consulta cancelada por tiempo, `bd_53300` demasiadas
  conexiones), o `conexion_plazo_vencido` si no llegó a conseguir conexión.
- `cancelada`: `cliente` si quien llamó cortó antes; `plazo` si venció un
  plazo del servidor.
- `version`: la revisión de Git del binario. Si sale `desconocida`, se
  compiló sin la marca (ver «Compilar con la revisión»).

Si el portal tarda minutos y acaba en 503, lo primero es buscar esta forma:
cero consultas, todo el tiempo en `bd_espera_ms` y
`bd_error: conexion_plazo_vencido`. Es un pool agotado.

```json
{"level":"ERROR","msg":"peticion","ruta":"/api/vec/ct/expedientes/{ref}","estado":503,"duracion_ms":200.4,"bd_consultas":0,"bd_ms":0,"bd_espera_ms":200.4,"bd_error":"conexion_plazo_vencido"}
```

## Paso a paso

1. Las lentas o fallidas de la última hora, agrupadas por ruta:
   ```sh
   podman logs --since 1h <contenedor> 2>&1 | grep '"msg":"peticion"' \
     | jq -r 'select(.lenta or .estado >= 500) | [.ruta, .estado, .duracion_ms, .bd_consultas, .bd_espera_ms, .bd_error] | @tsv' \
     | sort | uniq -c | sort -rn | head
   ```
2. Si todas las rutas tienen `bd_espera_ms` alto, el problema es el pool o
   PostgreSQL. Si es una ruta, mirar su `consulta_mas_lenta`.
3. Con la `correlacion` de una línea se encuentran las incidencias técnicas de
   esa misma petición: `podman logs <contenedor> 2>&1 | grep <correlacion>`.
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
  ruta.
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
- No se miden el pool acreditado de cobertura O4-05 de Contratación temporal,
  que rechaza por diseño cualquier trazador, ni el pool público de Bolsa.
- `vec-interno` todavía no escribe línea de acceso.
