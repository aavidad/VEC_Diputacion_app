# Instalación H9

Estado del paquete: ensayo SQL, arranque, reinicio y calidad completos sobre `main@39ec9858d`. Contiene 39 migraciones pendientes y un soporte de roles; las exclusiones están en `inventario.md`. Claude revisa y ejecuta la instalación. No se ha escrito en cidonia.

Binario SHA256: `16b8d71f19dc34db4d4bac35e69179526cdcea128ebef7da06f5e8b6f4248922`. Copia fría ensayada: `a1f56a65d53c6aaa20ab9f9b753f08ce80d390178ae03d6926072722ae192ce4`. `SHA256SUMS` contiene las huellas de cada SQL, web y catálogo; el SHA del manifiesto queda registrado en el canal de coordinación.

Este guion se ejecuta localmente como `openclaw`. El kit contiene `sql.list`, las SQL pendientes en orden causal, `consultas_preimagen.sql`, `bin/vec-server`, `web/` completa (incluido `web/static/`) y `locales/`. `SHA256SUMS` cubre todos los archivos del kit salvo el propio manifiesto. Solo admite archivos regulares y directorios. Las SQL deben tener un único `BEGIN;` y un `COMMIT;` final, en líneas independientes.

Los paquetes preparados que contengan `NO_INSTALAR` siguen pendientes de aprobación. El marcador forma parte del manifiesto y el guion los rechaza antes de parar servicios. No se elimina para forzar la instalación: dirección regenera y verifica el kit después de acreditar SQL y arranque.

1. Dirección fija el commit, revisa el kit y ensaya su SQL en el clon. Comprueba que la lista excluye las migraciones instaladas. Conserva por separado el SHA256 del manifiesto y la configuración privada.
2. Prepara fuera del kit un archivo de configuración propio, con permiso `0600`. Incluye las variables de la tabla. Sus tres ejecutables de mantenimiento deben estar revisados y bloquear el tráfico, los escritores externos y las tareas funcionales durante la ventana. Una pausa de proxy compartido no basta para cerrar todos los escritores.
3. Ejecuta `bash instalar.sh /ruta/al/kit /ruta/externa/config.sh SHA256_DEL_MANIFIESTO`. Guarda la salida en un archivo privado. El guion no usa SSH ni modifica el proxy compartido.
4. Tras `H9-OK`, verifica el recorrido autorizado en navegador. Conserva la copia y la relación `aplicadas.list`. Después de abrir tráfico, cualquier recuperación exige conciliar las escrituras posteriores.

| Variable privada | Contenido |
| --- | --- |
| `APP`, `PG` | Nombres de los contenedores locales inventariados. |
| `PGDATA`, `PGCONF`, `PGHBA` | Rutas absolutas de los datos y configuración de PostgreSQL. |
| `ART`, `CONF` | Directorios del binario de la aplicación y su configuración externa. |
| `WEB_ROOT`, `LOCALES_ROOT` | Directorios activos de web y catálogos. Se copian, comprueban y sustituyen completos. Todos los orígenes de copia deben ser disjuntos. |
| `PGDATA_RUNTIME_PATH` | Ruta inventariada de los datos dentro del contenedor PostgreSQL. |
| `WEB_RUNTIME_PATH`, `LOCALES_RUNTIME_PATH`, `BIN_RUNTIME_PATH` | Rutas dentro de la aplicación. Por defecto: `/app/web`, `/app/locales`, `/usr/local/bin/vec-server`. |
| `BACKUP_ROOT` | Directorio privado con espacio para copia fría, artefacto preparado y postimagen. |
| `EXTRA_COPIA` | Array Bash opcional de otros archivos o árboles necesarios para recuperar. Los orígenes deben ser disjuntos. |
| `RUNTIME_MOUNTS_SHA` | Huella obligatoria de los montajes inventariados de PostgreSQL y aplicación, en ese orden. Véase el algoritmo siguiente. |
| `PREIMAGEN_DB_SHA` | SHA256 de la salida de las consultas deterministas del inventario. Incluyen esquema, ACL, roles y datos; se ejecutan con `psql -X -q -At -v ON_ERROR_STOP=1 -U postgres -d postgres -f -`, transacción de solo lectura. |
| `PREIMAGEN_ART_SHA`, `PREIMAGEN_CONF_SHA`, `PREIMAGEN_WEB_SHA`, `PREIMAGEN_LOCALES_SHA` | Huellas de los cuatro árboles mediante el algoritmo siguiente. |
| `MANTENIMIENTO_CERRAR` | Ejecutable externo, sin argumentos, que cierra toda entrada de escrituras funcionales. Debe ser idempotente. |
| `MANTENIMIENTO_COMPROBAR` | Ejecutable externo que devuelve cero únicamente mientras el cierre sigue vigente, también con la aplicación arrancada. |
| `MANTENIMIENTO_ABRIR` | Ejecutable externo que abre el tráfico tras el arranque confirmado. Si falla o abre parcialmente, queda prohibido restaurar automáticamente la copia fría. |

Para `RUNTIME_MOUNTS_SHA`, toma las dos líneas JSON de `podman inspect -f '{{json .Mounts}}' "$PG" "$APP"`. Ordena los montajes de cada contenedor por `(Destination, Source, Type)` y sus listas `Options`. Calcula SHA256 del array de ambos contenedores serializado con `json.dumps(sort_keys=True, separators=(',', ':'))`, sin salto final. Las rutas y la huella del inventario se conservan fuera de Git. El guion comprueba además el montaje bind con el prefijo de destino más largo: cada ruta interna debe corresponder a su origen privado de binario, web, catálogos o PostgreSQL.

La huella de árbol usa SHA256. Ordena todos los descendientes por ruta relativa POSIX. Por cada entrada incorpora `ruta + NUL + modo_octal + NUL`, seguido del SHA256 binario del contenido para archivos o los bytes `directory` para directorios. El modo es `st_mode & 0o777`; la raíz queda excluida. Rechaza enlaces y archivos especiales. El inventario y el guion deben usar este mismo algoritmo.

El guion cierra mantenimiento y para la aplicación antes de comprobar la preimagen. Para PostgreSQL, copia en frío y coteja todos los orígenes; no admite WAL enlazado ni tablespaces externos. Arranca únicamente PostgreSQL. Por cada SQL ejecuta una transacción con `ROLLBACK` y, si termina bien, la misma SQL con `COMMIT`. Cambia el binario en `ART` y la web y los catálogos en `WEB_ROOT` y `LOCALES_ROOT`, con la aplicación parada y espera un mensaje nuevo `vec server listening` antes de abrir tráfico. El mantenimiento debe impedir también las tareas funcionales de la aplicación durante esa comprobación. Si web y catálogos proceden de un worktree, sustituye solo esos dos directorios; no ejecuta `git checkout` ni cambia su HEAD. El manifiesto del kit identifica el contenido desplegado.

Si falla SQL o arranque, para ambos servicios, conserva una postimagen, repone la copia fría y verifica sus huellas. PostgreSQL vuelve a arrancar; después el guion arranca la aplicación anterior bajo mantenimiento, exige un mensaje nuevo de escucha y abre el tráfico. Si tampoco arranca la anterior, la para y mantiene el cierre. La instalación termina con error aunque recupere el servicio anterior. Si la recuperación falla, mantiene ambos servicios bajo mantenimiento y conserva la copia. No ejecuta `DOWN` ni continúa con otras SQL.

`bash prueba_guion.sh` comprueba con dobles locales: fallo de SQL tras una migración confirmada, fallo de arranque, instalación correcta, manifiesto alterado y preimagen distinta. También comprueba aplicación activa o mantenimiento perdido entre migraciones, y rechaza una recuperación cuando fallan la consulta, la huella de base o la de artefacto. Verifica también la sustitución de archivos web fuera de `static` y el rechazo de una configuración que sea archivo. `bash prueba_guion.sh --web-config` ejecuta estos casos y verifica la web realmente servida en una ruta distinta del artefacto, su recuperación tras fallo de arranque o SQL, el rechazo de un origen ajeno al montaje y una huella de web recuperada distinta. Estas pruebas no acreditan instalación, PostgreSQL real ni navegador.

`bash prueba_guion.sh --blocked-kit` comprueba únicamente el rechazo del marcador y que servicios, datos y artefactos permanecen intactos.

`bash prueba_guion.sh --rollback` comprueba la vuelta automática al servicio anterior tras fallo SQL o arranque, y que permanece cerrado si tampoco arranca la versión anterior.
