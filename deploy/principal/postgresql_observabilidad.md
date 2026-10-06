# Observabilidad de PostgreSQL en la principal (propuesta, sin aplicar)

Esta lista prepara PostgreSQL para que Sistemas vea qué consultas y funciones
consumen el tiempo cuando el portal va lento. Nada de esto está aplicado en la
principal. Es configuración del servidor: no crea ni cambia tablas ni
funciones de VEC.

Ficheros:

- `postgresql_observabilidad.conf`: los parámetros, para incluirlos desde
  `postgresql.conf`.
- `consultas_observabilidad.sql`: consultas de solo lectura sobre las
  estadísticas (las más costosas, las funciones más lentas, quién bloquea a
  quién, conexiones por rol, transacciones abiertas sin actividad).

Se ensayó el 06/10/2026 en un PostgreSQL 18.4 desechable (la misma versión que
la principal): el fichero carga sin errores tras reiniciar, las bibliotecas se
añaden con el paso 2, la extensión se crea, las consultas se ejecutan, la
vuelta atrás deja el servidor como estaba y el registro de consultas lentas no
contiene los valores de los parámetros.

## Qué hace cada bloque

| Parámetro | Para qué |
| --- | --- |
| `pg_stat_statements` | Suma por consulta: llamadas, tiempo total, media, máximo, filas y lecturas de disco. Es lo primero que se mira. |
| `track_functions = 'all'` | Tiempo por función en `pg_stat_user_functions`. VEC hace casi todo dentro de funciones, así que aquí se ve cuál es la lenta. |
| `track_io_timing` | Separa el tiempo de disco del de CPU. Cuesta poco en un servidor moderno. |
| `log_min_duration_statement = 300ms` | Escribe en el registro de PostgreSQL cada consulta que pasa de 300 ms, el mismo umbral que el registro de acceso de VEC. |
| `log_parameter_max_length = 0` | Impide que esas líneas lleven los valores de `$1`, `$2`…, que pueden ser datos personales. |
| `log_error_verbosity = terse` | Quita de los errores el detalle (`DETAIL`), las pistas y el contexto, que pueden repetir valores de las filas. |
| `log_lock_waits` | Avisa de las esperas por bloqueo de más de un segundo. |
| `auto_explain` | Escribe el plan de las consultas muy lentas. Queda cargado pero apagado (ver abajo). |

## Cómo aplicarlo

Con la copia de seguridad previa habitual y en una ventana con poco uso:

1. Copiar `postgresql_observabilidad.conf` al directorio de datos como
   `vec_observabilidad.conf`, propiedad del usuario `postgres` y modo 0600, y
   añadir al final de `postgresql.conf`:
   `include_if_exists = 'vec_observabilidad.conf'`.
2. Añadir las dos bibliotecas a las que ya se precargan, sin quitar ninguna:
   ```sql
   SHOW shared_preload_libraries;   -- por ejemplo: pg_cron
   ALTER SYSTEM SET shared_preload_libraries = 'pg_cron', 'pg_stat_statements', 'auto_explain';
   ```
   Cada biblioteca va entre comillas por separado: escrita como una sola
   cadena (`'a,b'`) PostgreSQL busca un fichero con ese nombre y no arranca.
   Si el valor actual está vacío, quedan solo `'pg_stat_statements',
   'auto_explain'`. Sustituir la lista sin más dejaría de cargar extensiones
   que el servidor ya usa.
3. Reiniciar PostgreSQL.
4. Como superusuario, en la base de la aplicación:
   `CREATE EXTENSION IF NOT EXISTS pg_stat_statements;`
5. Dar lectura de estadísticas al rol de Sistemas que vaya a consultarlas:
   `GRANT pg_read_all_stats TO <rol de Sistemas>;`. No hace falta ningún
   permiso sobre las tablas de VEC.
6. Comprobar: `SHOW shared_preload_libraries;` (deben seguir las de antes),
   `SHOW track_functions;`, `SHOW log_error_verbosity;` y que
   `SELECT count(*) FROM pg_stat_statements;` responde.

Vuelta atrás: quitar la línea `include_if_exists`, devolver
`shared_preload_libraries` a su valor anterior (`ALTER SYSTEM SET …` o
`ALTER SYSTEM RESET shared_preload_libraries` si no tenía ninguno), reiniciar
y, si se quiere, `DROP EXTENSION pg_stat_statements;`.

## El registro de PostgreSQL lleva datos personales

El registro de PostgreSQL se trata siempre como información con datos
personales: acceso restringido a Sistemas, retención corta y nunca fuera del
servidor sin las mismas garantías que la base de datos. Aunque esta
configuración quita los valores de los parámetros (`log_parameter_max_length
= 0`) y el detalle de los errores (`log_error_verbosity = terse`), el texto de
una consulta, un mensaje de error o un plan pueden contener valores.

## auto_explain bajo demanda

Un plan puede mostrar valores concretos de la consulta (por ejemplo, en un
filtro). Por eso `auto_explain` queda apagado y se enciende solo mientras se
investiga un caso.

Para encenderlo durante una investigación:

```sql
ALTER SYSTEM SET auto_explain.log_min_duration = '1s';
SELECT pg_reload_conf();
-- reproducir la petición lenta y leer el plan en el registro de PostgreSQL
ALTER SYSTEM RESET auto_explain.log_min_duration;
SELECT pg_reload_conf();
```

`log_analyze` con `log_timing = off` da filas reales sin el coste de medir el
tiempo de cada nodo. `log_nested_statements = on` hace falta porque las
consultas de VEC se ejecutan dentro de funciones.

## Coste

`pg_stat_statements` y `track_functions` añaden un coste pequeño por consulta
(del orden de un pequeño porcentaje en cargas con muchas consultas cortas). Si
la prueba de carga previa al hito muestra una diferencia apreciable, se puede
bajar `track_functions` a `'pl'` o quitar `track_io_timing` sin reiniciar.
