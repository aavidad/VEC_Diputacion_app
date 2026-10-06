-- Consultas de diagnóstico para Sistemas. Solo leen estadísticas.
-- Se ejecutan con un rol que tenga pg_read_all_stats (ver
-- postgresql_observabilidad.md). Ninguna lee tablas de VEC.

-- 1. Las 20 consultas que más tiempo total consumen (pg_stat_statements).
SELECT queryid,
       calls,
       round(total_exec_time::numeric, 1)                 AS total_ms,
       round(mean_exec_time::numeric, 2)                  AS media_ms,
       round(max_exec_time::numeric, 1)                   AS max_ms,
       rows,
       shared_blks_read,
       left(regexp_replace(query, '\s+', ' ', 'g'), 160)  AS consulta
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 20;

-- 2. Las 20 más lentas de media con al menos 50 llamadas.
SELECT queryid, calls,
       round(mean_exec_time::numeric, 2) AS media_ms,
       round(stddev_exec_time::numeric, 2) AS desviacion_ms,
       left(regexp_replace(query, '\s+', ' ', 'g'), 160) AS consulta
FROM pg_stat_statements
WHERE calls >= 50
ORDER BY mean_exec_time DESC
LIMIT 20;

-- 3. Funciones de VEC por tiempo total (track_functions = all).
SELECT schemaname || '.' || funcname AS funcion,
       calls,
       round(total_time::numeric, 1) AS total_ms,
       round(self_time::numeric, 1)  AS propio_ms,
       round((total_time / NULLIF(calls, 0))::numeric, 2) AS media_ms
FROM pg_stat_user_functions
ORDER BY total_time DESC
LIMIT 30;

-- 4. Qué está haciendo ahora cada conexión que no está ociosa, la más
--    antigua primero. wait_event dice si espera disco, bloqueo o cliente.
SELECT pid,
       usename                                AS rol,
       application_name,
       state,
       wait_event_type,
       wait_event,
       now() - xact_start                     AS en_transaccion,
       now() - query_start                    AS en_consulta,
       left(regexp_replace(query, '\s+', ' ', 'g'), 120) AS consulta
FROM pg_stat_activity
WHERE backend_type = 'client backend'
  AND state <> 'idle'
ORDER BY query_start NULLS LAST;

-- 5. Quién bloquea a quién.
SELECT bloqueada.pid                     AS pid_esperando,
       bloqueada.usename                 AS rol_esperando,
       now() - bloqueada.query_start     AS esperando_desde,
       bloqueadora.pid                   AS pid_bloquea,
       bloqueadora.usename               AS rol_bloquea,
       bloqueadora.state                 AS estado_bloquea,
       now() - bloqueadora.xact_start    AS transaccion_bloquea
FROM pg_stat_activity AS bloqueada
JOIN LATERAL unnest(pg_blocking_pids(bloqueada.pid)) AS b(pid) ON true
JOIN pg_stat_activity AS bloqueadora ON bloqueadora.pid = b.pid;

-- 6. Conexiones por rol y estado, frente al máximo del servidor.
SELECT usename AS rol, state, count(*) AS conexiones,
       current_setting('max_connections')::int AS maximo_servidor
FROM pg_stat_activity
WHERE backend_type = 'client backend'
GROUP BY usename, state
ORDER BY conexiones DESC;

-- 7. Transacciones abiertas sin hacer nada (retienen conexiones y bloqueos).
SELECT pid, usename AS rol, now() - xact_start AS abierta_desde,
       now() - state_change AS ociosa_desde
FROM pg_stat_activity
WHERE state = 'idle in transaction'
ORDER BY xact_start;

-- 8. Para empezar una medición limpia (por ejemplo, antes de una prueba de
--    carga). Borra las estadísticas acumuladas; solo con permiso.
-- SELECT pg_stat_statements_reset();
-- SELECT pg_stat_reset();
