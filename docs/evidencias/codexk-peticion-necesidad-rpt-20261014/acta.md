# Necesidad publicada en la petición del centro

El centro elige un puesto y una categoría de la RPT publicada. El servidor comprueba esa elección y conserva su contenido en la petición. Otra identidad ratifica los mismos datos; RRHH utiliza el alta existente para crear un único expediente.

Reutiliza el catálogo y el normalizador de Alta v3, su verificador RPT, ServicioPeticionCentro, el registro y la entrega existentes, el formulario de Alta y el cliente público RPT de Personal. El cliente envía una elección, nunca una instantánea. Los reintentos y la ratificación recuperan el contenido ya conservado. La entrega puede utilizar la publicación ratificada aunque haya cambiado el catálogo activo.

## SQL y configuración

CT215 reconstruye las dos funciones instaladas de presentación y entrega, desde sus cuerpos de la copia 13b. Conserva firmas, permisos, historia y autorización. Lista: `deploy/principal/lista_sql_codexk_peticion_necesidad_rpt_20261014.txt`. No ejecutar DOWN ni repetir una instalación con historia.

CONFIG NUEVA: ninguna. Utiliza `VEC_CT_NECESIDADES_ALTA_SOURCE_PATH`, `VEC_RPT_CATALOGO_PATH` y la fuente de organización existentes. Los valores de jornada y las causas permanecen en su catálogo.

UP SHA256: `4457ba04513c7daf09e49feeb5024eeea40ef754d9575e7246288bd5f06910c2`. Una instalación final en copia local PostgreSQL 18.4 terminó en COMMIT; posthashes de las dos funciones `f414c20f…` y `954e2750…`, permisos owner/ejecutor intactos y `search_path=pg_catalog,pg_temp`. Sin SQL en cidonia. El DOWN conserva los cuerpos históricos, pero mantiene `search_path=pg_catalog,pg_temp` al restaurarlos; una guarda comprueba sus metadatos y su cuerpo. Esta corrección recibió GO independiente sobre el archivo SHA256 `c7c9ba16a9497559f4e0b0084c9ab591b7d1d7b49acf5319cafd7a83110e751b`. No se ejecutó DOWN ni se repitió el ensayo del UP.

## Recorrido local

Copia de la principal 13b, con datos de ficción, PostgreSQL propio limitado a 2 GB, sin reinicio automático, HTTPS local y Chrome del sistema con certificados privados fuera de Git. Lucía Fernández Castillo presenta y Manuel García Robles ratifica, con identidades y perfiles distintos. Se seleccionan Administrativo, centro 520 y puesto publicado 9.

| Operación | HTTP | Tiempo | Base |
| --- | --- | --- | --- |
| Presentación desde pantalla | 200 | 97,1 ms servidor | 51,4 ms, 21 consultas |
| Ratificación desde pantalla | 200 | 127,8 ms servidor | 67,9 ms, 26 consultas |
| Alta RRHH desde pantalla | 200 | 175,2 ms navegador | Sin captura separada |
| Reintento tras reinicio, API mTLS de Playwright | 200 | 66,3 ms servidor | 43,3 ms, 13 consultas |

Petición `peticion:centro:b55fa7db-70c8-403a-b9a3-a60770904fdc`, versiones 1 y 2. Recibos de presentación `recibo:peticion-centro:150491d4-ff3f-46a2-9cb4-91fde8b480be` y ratificación `recibo:peticion-centro:cdaca9e8-c414-4ba8-aef8-9154b0709f26`.

Expediente `2026/215`, referencia `expediente:ct:bb6a184cdd909b44351ceb1faef3efbdacd9289bb1219f1a8e47fe5a738c9125`, versión 1. Recibo `recibo:ct-alta:aa62ce40acc707d4a4dd0c19b7873ec9c965425706119014420b5388f89244b4`, fecha `2026-10-13T07:13:04.986776Z`, auditoría `aud_ct_41232374b9a1c8fcd0ec794048f47fa8` y evento `evt_ct_79cd268fa1ccea4cc8af6851422e1cbb`.

Tras reiniciar aplicación y base, Chrome recupera la entrega y abre la ficha del mismo expediente. El reintento API devuelve los mismos ocho campos del recibo. Permanecen 87 altas, 16 reservas y 16 confirmaciones de entrega: no añade otra alta. La instantánea conservada tiene SHA256 `ab744fd8c7b1c1b2a536e516cb6ec7816c1a2d3c18bd568fc983a3242fcf27ce`, igual al de su publicación.

El primer intento RRHH con el número `2026/CT-000215` devolvió 422: el dato incumplía el patrón MOAD publicado. Se corrigió a `2026/215` sobre la misma petición y reserva, sin cambiar código ni permisos.

## Pantalla y comprobaciones

El formulario usa los paneles y controles comunes de Alta. La búsqueda RPT conserva lo escrito durante la espera y descarta resultados de una selección anterior. Cabecera y pie permanecen dentro del lienzo; el contenido y las tablas se desplazan internamente. Chrome comprobó 1440×900 y 390×844 sin desbordamiento global. Las sesiones del centro y ratificador no registraron errores de consola; almacenamiento local y de sesión vacíos. Al abrir el portal general aparecieron dos errores de configuración del kit: preferencias apagadas (404) y certificado del service worker. No se atribuye cero errores a todo el portal.

Capturas y respuestas completas permanecen en el acta privada local, fuera de Git: recibo del centro en escritorio/móvil, entrega RRHH y ficha tras reinicio. La revisión visual tomó como referencia `fotos/image004.png`.

Focales Go de cuatro paquetes, guarda de presentación nueva v3 y recuperación v2, Node de centro/Alta, i18n, gofmt y vecsilencio verdes. SQL recibió GO independiente en `82c46596d`; backend sensible GO en `26d65fe4d`, mantenido en `20091ac15` tras renombrar únicamente una UUID pública de prueba; corrección de pérdida de datos UI GO en `b472acf63`. Semgrep local sin hallazgos. Gosec no encontró problemas nuevos: conserva un G304 de lectura de configuración en una línea sin modificar y reporta carga parcial de 22 paquetes.

La cadena compartida de caché queda conciliada en una sola cohorte, con 6/6 comprobaciones focales. La CI queda pendiente al abrir la PR. Este corte crea el expediente CT; la incorporación en Personal, el cese y la vuelta a Bolsa siguen en la cola de K.
