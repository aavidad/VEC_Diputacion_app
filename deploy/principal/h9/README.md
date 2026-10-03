# Instalación H9

Este guion se ejecuta localmente como `openclaw`. El kit contiene `sql.list`, las SQL pendientes en orden causal, `consultas_preimagen.sql`, `bin/vec-server`, `web/` completa (incluido `web/static/`) y, cuando se usan, `locales/`. `SHA256SUMS` cubre todos los archivos del kit salvo el propio manifiesto. Solo admite archivos regulares y directorios. Las SQL deben tener un único `BEGIN;` y un `COMMIT;` final, en líneas independientes.

1. Dirección fija el commit, revisa el kit y ensaya su SQL en el clon. Comprueba que la lista excluye las migraciones instaladas. Conserva por separado el SHA256 del manifiesto y la configuración privada.
2. Prepara fuera del kit un archivo de configuración propio, con permiso `0600`. Incluye las variables de la tabla. Sus tres ejecutables de mantenimiento deben estar revisados y bloquear el tráfico, los escritores externos y las tareas funcionales durante la ventana. Una pausa de proxy compartido no basta para cerrar todos los escritores.
3. Ejecuta `bash instalar.sh /ruta/al/kit /ruta/externa/config.sh SHA256_DEL_MANIFIESTO`. Guarda la salida en un archivo privado. El guion no usa SSH ni modifica el proxy compartido.
4. Tras `H9-OK`, verifica el recorrido autorizado en navegador. Conserva la copia y la relación `aplicadas.list`. Después de abrir tráfico, cualquier recuperación exige conciliar las escrituras posteriores.

| Variable privada | Contenido |
| --- | --- |
| `APP`, `PG` | Nombres de los contenedores locales inventariados. |
| `PGDATA`, `PGCONF`, `PGHBA` | Rutas absolutas de los datos y configuración de PostgreSQL. |
| `ART`, `CONF` | Directorios del artefacto de la aplicación y su configuración externa. |
| `BACKUP_ROOT` | Directorio privado con espacio para copia fría, artefacto preparado y postimagen. |
| `EXTRA_COPIA` | Array Bash opcional de otros archivos o árboles necesarios para recuperar. Los orígenes deben ser disjuntos. |
| `RUNTIME_MOUNTS_SHA` | Huella opcional de los montajes inventariados de PostgreSQL y aplicación, en ese orden. Véase el algoritmo siguiente. |
| `PREIMAGEN_DB_SHA` | SHA256 de la salida de las consultas deterministas del inventario. Incluyen esquema, ACL, roles y datos; se ejecutan con `psql -X -q -At -v ON_ERROR_STOP=1 -U postgres -d postgres -f -`, transacción de solo lectura. |
| `PREIMAGEN_ART_SHA`, `PREIMAGEN_CONF_SHA` | Huellas de los árboles mediante el algoritmo siguiente. |
| `MANTENIMIENTO_CERRAR` | Ejecutable externo, sin argumentos, que cierra toda entrada de escrituras funcionales. Debe ser idempotente. |
| `MANTENIMIENTO_COMPROBAR` | Ejecutable externo que devuelve cero únicamente mientras el cierre sigue vigente, también con la aplicación arrancada. |
| `MANTENIMIENTO_ABRIR` | Ejecutable externo que abre el tráfico tras el arranque confirmado. Si falla o abre parcialmente, queda prohibido restaurar automáticamente la copia fría. |

Para `RUNTIME_MOUNTS_SHA`, toma las dos líneas JSON de `podman inspect -f '{{json .Mounts}}' "$PG" "$APP"`. Ordena los montajes de cada contenedor por `(Destination, Source, Type)` y sus listas `Options`. Calcula SHA256 del array de ambos contenedores serializado con `json.dumps(sort_keys=True, separators=(',', ':'))`, sin salto final. Las rutas y la huella del inventario se conservan fuera de Git.

La huella de árbol usa SHA256. Ordena todos los descendientes por ruta relativa POSIX. Por cada entrada incorpora `ruta + NUL + modo_octal + NUL`, seguido del SHA256 binario del contenido para archivos o los bytes `directory` para directorios. El modo es `st_mode & 0o777`; la raíz queda excluida. Rechaza enlaces y archivos especiales. El inventario y el guion deben usar este mismo algoritmo.

El guion cierra mantenimiento y para la aplicación antes de comprobar la preimagen. Para PostgreSQL, copia en frío y coteja todos los orígenes; no admite WAL enlazado ni tablespaces externos. Arranca únicamente PostgreSQL. Por cada SQL ejecuta una transacción con `ROLLBACK` y, si termina bien, la misma SQL con `COMMIT`. Cambia binario, web y catálogos con la aplicación parada y espera un mensaje nuevo `vec server listening` antes de abrir tráfico. El mantenimiento debe impedir también las tareas funcionales de la aplicación durante esa comprobación.

Si falla SQL o arranque, para ambos servicios, conserva una postimagen, repone la copia fría y verifica sus huellas. PostgreSQL vuelve a arrancar; la aplicación queda parada y el mantenimiento cerrado. Dirección revisa el fallo antes de arrancar el artefacto anterior y abrir tráfico. Si la recuperación falla, mantiene ambos servicios bajo mantenimiento y conserva la copia. No ejecuta `DOWN` ni continúa con otras SQL.

`bash prueba_guion.sh` comprueba con dobles locales: fallo de SQL tras una migración confirmada, fallo de arranque, instalación correcta, manifiesto alterado y preimagen distinta. También comprueba aplicación activa o mantenimiento perdido entre migraciones, y rechaza una recuperación cuando fallan la consulta, la huella de base o la de artefacto. Verifica también la sustitución de archivos web fuera de `static` y el rechazo de una configuración que sea archivo. `bash prueba_guion.sh --web-config` ejecuta estos casos y comprueba coincidencia o cambio de montajes. Estas pruebas no acreditan instalación, PostgreSQL real ni navegador.
