# Ensayo local de un conjunto

`vec-copias-ejecutar` conecta la ventana CS04, los archivos CS05, el destino
cifrado CS03, el registro CS07 y las restauraciones CS06 para un conjunto
sintético propio. Sólo acepta una copia válida cuando ambos ensayos contrastan
el contenido y el binario archivado supera las sondas de salud y sesión
mTLS autorizada. El corte actual conserva los bloqueos encontrados; todavía
no acredita un conjunto completo restaurado ni habilita restauración productiva.

El ejecutor lee un único JSON privado fuera de Git, regular, propiedad del usuario y modo `0600`.
El directorio de trabajo debe ser propio, modo `0700`, bajo `/var/tmp/vec-cs11-*`.
Use nombres Docker `vec-cs11-src-` seguidos de 8 a 32 letras minúsculas o cifras.
El nombre debe estar libre antes de `preparar`.

1. Obtenga un commit limpio y construya **una vez** su `vec-server` archivado
   desde un clon local independiente. Use `go build -buildvcs=true`. Compruebe
   `vcs.revision`, `vcs.modified=false`, tamaño y SHA256 sobre ese mismo archivo
   con `go version -m`, `stat` y `sha256sum`. Si faltan metadatos Git, consiga un
   descriptor autenticado que vincule commit y bytes; no copie un commit escrito
   a mano al inventario.
2. Seleccione una imagen PostgreSQL 18.4 ya instalada y obtenga su ID real con
   `docker image inspect --format '{{.Id}}' postgres:18.4`. Guarde los 64 dígitos
   posteriores a `sha256:` como `ImagenSHA256`. Cree `pgcluster/data`, `logica`,
   `fisica` y `fuentes` bajo el directorio privado.
3. Genere material sintético con `scripts/generar_credenciales_desarrollo.sh`
   hacia una subcarpeta privada propia. Empaquételo en un TAR regular cuya raíz
   sea `contenido/`. Prepare archivos regulares para el binario, activos web,
   catálogo personal, material, configuración y ficheros. Copie los activos
   desde el mismo commit del binario. Registre incluso el conjunto vacío de
   ficheros; no sustituya componentes ausentes por una marca de éxito.
4. Escriba `origen` en el JSON privado con `Contenedor`, `ImagenSHA256`,
   `RaizPropia`, `PGDATA`, `RaizLogica`, `Base` y `Usuario`. `PGDATA` debe estar
   dentro de un subdirectorio propio `pgcluster/data`. Ejecute
   `vec-copias-ejecutar -accion preparar -config RUTA_PRIVADA`. Crea un cluster
   sin red y aplica el SQL sintético fijo de este adaptador. No repita esta
   acción sobre un cluster existente.
5. Añada a `origen` las cinco rutas de herramientas **dentro de la imagen**
   (`postgres`, `psql`, `pg_dump`, `pg_dumpall`, `pg_restore`), `Fuentes` con
   ID/tipo/ruta de los seis archivos reales, y `TiempoMaximo` en nanosegundos.
   Ejecute `-accion medir` y guarde su JSON con `umask 077`. El resultado mide
   los bytes de herramientas, esquema y archivos; no concede validez a la copia.
6. Prepare `InventarioEsperado` y `Politica` CS01 con esas mediciones y el
   commit observado del binario. Para este ejercicio, el único módulo es
   `administracion`, con el SHA256 del esquema medido y cero migraciones porque
   el SQL sintético no instala migraciones. La política admite exactamente ese
   runtime y esa release. Declare `AmbitoCompleto` solo para el cluster dedicado
   del ejercicio; no use una base compartida.
7. Configure `runtime` como ruta a otro JSON privado `0600` con
   `perfil: "sesion_sintetica"`, `binario_id: "fisica:vec-server"`,
   `material_id: "fisica:material"`,
   `catalogo_personal_id: "fisica:catalogo_personal"`, puerto local propio y
   usuario PostgreSQL del ensayo físico. `entorno` debe estar vacío. La CLI
   monta el material y el catálogo archivados; no hereda DSN ni SMTP del host.
8. Complete el JSON de ejecución con `peticion`, `fisica`, `logica`, `contraste`,
   `captura_fisica`, directorios separados de almacén/catálogo/registro/diario,
   límites y `clave_maestra_fichero` apuntando al archivo privado de 32 bytes
   generado en el paso 3. La CLI borra sus copias temporales de esa clave.
   El usuario bootstrap lógico debe ser distinto de los roles incluidos en
   `globals.sql`. Ejecute
   `-accion revisar-origen`; sólo `observada` permite avanzar. Finalmente,
   ejecute `-accion copiar` con una clave de operación y conjunto nuevas.

`no_comprobable` deja el registro y los artefactos privados disponibles para
diagnóstico. Si la operación quedó pendiente tras publicar, recupere la misma
clave y conjunto para conciliarla; el estado final se consulta con esa misma
clave. Una operación `no_valida` permanece en la historia. Tras corregir su
causa, use una clave y un conjunto nuevos sin borrar el registro anterior. Si
quedaron volcados lógicos de ese intento, asigne también un directorio lógico
nuevo y conserve el anterior como evidencia. Los códigos de
salida son 0 para la acción comprobada, 1 para un bloqueo operativo y 2 para
entrada inválida. Los mensajes para personas están en los catálogos
`web/static/textos/{es,en}/copias_ejecucion.json`.

En el corte del 1 de octubre, el binario `vec-server` del commit limpio
`09b74b412757b1c67aa5b9f18d0e1534af03addf` no arrancó en este perfil de
sesión: su composición exigió conexiones y autoridades PostgreSQL separadas de
Contratación temporal. El primer arranque aislado también descubrió que el UID
del ejecutor necesita una entrada propia en `/etc/passwd` para que
`user.Current()` compruebe el directorio personal. El adaptador debe resolver
ambas condiciones y repetir las sondas antes de afirmar un conjunto válido.
El contraste de la fuente detectó también `otras_bases_no_inventariadas`: la
base de mantenimiento `postgres` requiere inventario junto con las demás bases
del cluster. La extensión multibase sigue pendiente. La operación local del
intento quedó `no_valida`, sin componentes publicados, y la fuente permaneció
intacta. La autoridad ADMIN vigente y la plataforma de sustitución también
siguen siendo dependencias de la restauración operativa.

## Recuperación del 4 de octubre

Se recuperaron los archivos propios del ejecutor sobre la API y los verificadores
actuales, sin sustituir CS06 ni borrar sus avances multibase. Las dependencias
lógicas y de abandono conservan las ramas de las PR #392 y #405.

La lectura de sesión sintética no acredita una consulta nominal de los datos
restaurados. El resultado de arranque distingue esa evidencia y el observador
exige que el proveedor la haya comprobado. El runtime de sesión disponible
permanece sin esa comprobación y no puede cerrar una copia como válida.

Este trabajo sigue en preparación. Falta conciliar la interrupción entre el diario
y su testigo, adaptar todos los consumidores al inventario multibase y conectar
identidad, consumo y auditoría comunes. No habilita copias ni restauraciones
operativas desde Administración. Los borradores SQL de K se conservan en su fuente;
no forman parte de esta recuperación.
