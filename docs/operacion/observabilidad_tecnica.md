# Seguir una petición lenta en VEC

Guía para Sistemas y para quien desarrolla. Explica dónde deja VEC el rastro
técnico de cada petición y cómo ir de «el portal va lento» a la consulta o la
función que se lleva el tiempo.

Este registro técnico es distinto de la auditoría de uso de datos. No guarda
quién hizo la petición, ni los valores de la ruta, ni la consulta, ni
cabeceras ni cuerpos. Solo guarda la forma de la ruta, tiempos, tamaños y
códigos cerrados. La auditoría nominal sigue en su propia cadena, con sus
permisos.

## Dónde mirar

| Qué | Dónde | Para qué |
| --- | --- | --- |
| Una línea por petición (`vec.acceso_http.v1`) | Salida de errores del proceso: `podman logs <contenedor>` o el diario del servicio | Saber qué ruta tardó, cuánto, con cuántas consultas y por qué falló |
| Peticiones que siguen abiertas (`vec.peticion_en_curso.v1`) | El mismo sitio | Ver un atasco mientras ocurre, sin esperar a que termine |
| Incidencias técnicas (`vec.incidencia_tecnica.v1`) | Salida estándar, recogida por `vec-registros-tecnicos` | Fallos con código del catálogo; comparten la correlación con la línea de acceso |
| Métricas y perfiles | Escucha de diagnóstico en bucle local, con token | Tendencias por ruta, estado de los pools, CPU y memoria |
| PostgreSQL | Su propio registro y `pg_stat_statements` | Qué consulta concreta tarda y su plan (ver `deploy/principal/postgresql_observabilidad.md`) |

Las líneas son JSON, una por renglón, así que se filtran con `grep` y `jq`.

## La línea de acceso

Ejemplo real de una prueba con PostgreSQL 18.4 y datos sintéticos:

```json
{"esquema":"vec.acceso_http.v1","instante":"2026-10-06T09:44:37.885Z","nivel":"aviso","servicio":"vec-server","superficie":"interno","entorno":"desarrollo","version":"245887825abc","correlacion":"8f530f8aedf04fca39c92a9bd397a98d","metodo":"GET","ruta":"/api/vec/bolsa/{bolsa}/participaciones","estado":200,"duracion_ms":809.3,"bytes":0,"bd_consultas":31,"bd_ms":403.5,"bd_espera_ms":405.6,"lenta":true,"lenta_por":["duracion","consultas"],"desglose":[{"operacion":"vec_bolsa.listar_participaciones","consultas":1,"ms":401.5,"max_ms":401.5},{"operacion":"vec_bolsa.consultar_participacion","consultas":30,"ms":2,"max_ms":0.5}]}
```

Cómo leerla:

- `instante` es la llegada de la petición, en hora UTC. `duracion_ms` es lo que
  tardó el servidor en contestar.
- `ruta` es la plantilla. Donde iba una referencia aparece `{bolsa}` o
  `{valor}`. Si ningún enrutador anotó la plantilla, un 404 sale como
  `{no_encontrada}` y otro 4xx (por ejemplo, una sesión rechazada antes de
  llegar a la ruta) como `{sin_plantilla}`, para no copiar lo que escribió la
  persona.
- `bd_consultas` y `bd_ms` son las consultas a PostgreSQL de esa petición y el
  tiempo que pasaron en ellas. `bd_espera_ms` es el tiempo esperando una
  conexión libre del pool.
- `lenta` se marca al pasar de 300 ms o de 20 consultas. `lenta_por` dice cuál
  de las dos. Con 20 consultas o más casi siempre hay una consulta por fila (el
  problema N+1).
- `desglose` lista las cinco operaciones que más tiempo llevaron. La operación
  es la función o tabla de PostgreSQL, sacada del texto SQL, sin valores.
- `version` es la revisión de Git del binario y `entorno`, el entorno de
  despliegue. Si salen como `desconocida`, el binario se compiló sin la marca
  de revisión (ver «Compilar con la revisión»).

En el ejemplo, la petición tardó 809 ms: unos 400 ms esperando conexión, 401 ms
en `vec_bolsa.listar_participaciones` y treinta llamadas a
`vec_bolsa.consultar_participacion`, que es un N+1.

## Cuando falla

Una respuesta 5xx deja `nivel: error` y, si se conoce, la causa:

```json
{"esquema":"vec.acceso_http.v1","nivel":"error","ruta":"/api/vec/ct/expedientes/{ref}","estado":503,"duracion_ms":200.8,"bd_consultas":0,"bd_ms":0,"bd_espera_ms":200.7,"bd_error":"conexion_plazo_vencido","causa":"plazo_vencido","etapa_fallo":"consulta_expediente","correlacion":"10b1574633ac6cad5538d0943e4a662c"}
```

(Se han quitado campos para que quepa.)

- `bd_error` es el último error de base de datos de la petición: `bd_` seguido
  del código SQLSTATE de PostgreSQL (`bd_57014` es una consulta cancelada por
  tiempo, `bd_53300` demasiadas conexiones, `bd_40001` un conflicto de
  serialización). Si empieza por `conexion_`, el fallo fue al pedir una
  conexión al pool.
- `causa` y `etapa_fallo` aparecen cuando el código anotó el fallo. La causa es
  una clase cerrada (`plazo_vencido`, `cancelada`, `red`, `bd_…`), nunca el
  texto del error.
- `incidencia`, `componente` y `etapa_incidencia` aparecen cuando un adaptador
  declaró una incidencia del catálogo durante la petición.
- `cancelada: cliente` indica que quien llamó cortó la conexión antes de la
  respuesta; `cancelada: plazo`, que venció un plazo del servidor.
- `interrumpida: true` indica un pánico o un corte; la supervisión respondió
  con un 500.

El ejemplo es la firma de un pool agotado: cero consultas, todo el tiempo en
`bd_espera_ms` y `conexion_plazo_vencido`. Es lo primero que hay que descartar
cuando el portal tarda minutos y acaba en 503.

## Peticiones que no terminan

Si una petición sigue abierta a los 10 segundos, el proceso escribe una línea
`vec.peticion_en_curso.v1`, y otra a los 30 s, 90 s, 270 s…

```json
{"esquema":"vec.peticion_en_curso.v1","instante":"2026-10-06T09:44:38.836Z","nivel":"aviso","correlacion":"2ec907a79280fc76a3a87eeca055340a","metodo":"GET","ruta":"/api/vec/atascada","llegada":"2026-10-06T09:44:37.835Z","transcurrido_ms":1000.7,"bd_consultas":0,"bd_ms":0,"bd_espera_ms":3.6,"actividad":"consulta","objeto":"vec_bolsa.lenta","actividad_ms":997.1}
```

`actividad` dice qué está haciendo ahora: `consulta` (con la operación en
`objeto`) o `esperando_conexion` (con el pool, es decir, el rol de base de
datos). Si no hay actividad, el tiempo se va fuera de la base de datos.

## Seguir una petición de principio a fin

1. Localizar las lentas o fallidas de la última hora:
   ```sh
   podman logs --since 1h <contenedor> 2>&1 | grep '"vec.acceso_http.v1"' \
     | jq -c 'select(.lenta or .estado >= 500) | {instante, ruta, estado, duracion_ms, bd_consultas, bd_ms, bd_espera_ms, bd_error, causa, correlacion}'
   ```
2. Ver si se repite en la misma ruta (agrupar por `ruta`) o es general. Si es
   general y `bd_espera_ms` es alto, el problema es el pool o PostgreSQL, no
   una pantalla.
3. Con la `correlacion` de una petición, buscar todo lo demás de esa petición:
   ```sh
   podman logs <contenedor> 2>&1 | grep <correlacion>
   ```
   Salen la línea de acceso, las líneas «en curso» y las incidencias técnicas
   que se emitieron durante ella.
4. Mirar el `desglose`. Si una operación tiene muchas consultas cortas, es un
   N+1 en el código. Si tiene una consulta larga, se pasa a PostgreSQL.
5. En PostgreSQL, buscar esa función en `pg_stat_user_functions` y en
   `pg_stat_statements` (consultas 1 a 3 de
   `deploy/principal/consultas_observabilidad.sql`). Si hace falta el plan,
   encender `auto_explain` un rato, como explica
   `deploy/principal/postgresql_observabilidad.md`.
6. Si hay esperas de conexión, mirar quién ocupa las conexiones (consultas 4 a
   7) y las métricas del pool (siguiente apartado).

## Métricas y perfiles

Cada proceso (`vec-server`, `vec-admin`, `vec-publico`) puede abrir una escucha
de diagnóstico. Está apagada salvo que Sistemas la configure:

- `VEC_DIAGNOSTICO_ESCUCHA=127.0.0.1:9464`: solo se admite una IP de bucle
  local. Con cualquier otra dirección no se abre y queda un aviso en el
  registro.
- `VEC_DIAGNOSTICO_TOKEN_FILE=/ruta/privada/token`: fichero propio, modo 0600,
  de 32 a 4096 caracteres. Cada petición lo presenta como
  `Authorization: Bearer …`.

Desde dentro del contenedor:

```sh
token=$(cat /ruta/privada/token)
curl -s -H "Authorization: Bearer $token" http://127.0.0.1:9464/metrics | grep vec_pool_
curl -s -H "Authorization: Bearer $token" -o cpu.pprof 'http://127.0.0.1:9464/debug/pprof/profile?seconds=30'
```

El perfil se lee después en un equipo con Go: `go tool pprof -top cpu.pprof`.

Lo más útil cuando el portal va lento:

- `vec_pool_conexiones_en_uso` frente a `vec_pool_conexiones_maximas`: si
  coinciden durante un rato, el pool está agotado.
- `vec_pool_prestamos_con_espera_total` y `vec_pool_espera_segundos_total`:
  cuántas veces y cuánto tiempo se esperó por una conexión.
- `vec_http_duracion_segundos` por ruta: para calcular el percentil 95.
- `vec_http_bd_consultas_total` dividido entre `vec_http_peticiones_total`:
  consultas medias por petición de cada ruta.
- `vec_bd_errores_total` por clase.
- `vec_http_en_curso`: peticiones abiertas ahora mismo.
- `/debug/pprof/goroutine?debug=2`: dónde está parado cada hilo de Go, útil si
  el proceso no responde.

## Ajustes

| Variable | Valor por defecto | Qué cambia |
| --- | --- | --- |
| `VEC_TELEMETRIA_LENTA_MS` | 300 | Milisegundos a partir de los que una petición es lenta |
| `VEC_TELEMETRIA_LENTA_CONSULTAS` | 20 | Consultas a partir de las que una petición es lenta |
| `VEC_TELEMETRIA_EN_CURSO_S` | 10 | Segundos hasta la primera línea «en curso» |
| `VEC_ENTORNO` | perfil de ejecución | `desarrollo`, `pruebas`, `presentacion` o `produccion` |

Un valor no válido deja un aviso en el arranque y se usa el valor por defecto.

## Compilar con la revisión

Los guiones de despliegue compilan con `-buildvcs=false`, así que la revisión
se marca a mano:

```sh
go build -buildvcs=false \
  -ldflags "-X vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria.Revision=$(git rev-parse --short=12 HEAD)" \
  -o vec-server ./cmd/vec-server
```

`deploy/principal/desplegar.sh`, `scripts/arrancar_vec_desarrollo.sh` y el
`Dockerfile` (con `--build-arg VEC_REVISION=…`) ya lo hacen.

## Límites

- Una consulta solo cuenta para su petición si el código usa el contexto de la
  petición. Las que van con un contexto propio salen en las métricas del
  proceso, pero no en la línea de acceso.
- El pool acreditado de cobertura O4-05 de Contratación temporal rechaza por
  diseño cualquier trazador y no se mide. Tampoco el pool público de Bolsa
  (`internal/modules/bolsa/adapters/postgrespublico`), pendiente de su equipo.
- `vec-interno` todavía no escribe línea de acceso.
- Si la cola de líneas se llena (por ejemplo, porque el destino está
  bloqueado), las líneas se descartan para no frenar el portal, y una línea
  `vec.acceso_http_descartes.v1` dice cuántas se perdieron.
