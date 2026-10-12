# Alta en Personal y cese con vuelta a Bolsa — 12/10/2026

**Entorno:** Chrome del sistema y PostgreSQL 18.4 en una copia propia de la principal. Todos los datos son inventados. No se tocó cidonia. La cabeza final de Personal46 se instaló una sola vez en esta copia; no se ejecutó `DOWN` ni se reaplicó SQL con historia.

## Recorrido comprobado con Personal46 final

| Paso | Respuesta y efecto observado |
| --- | --- |
| Alta desde el expediente CT `2026/94014` | Confirmación HTTP 200. CT devolvió `recibo:incorporacion-personal-b2:99aac398-2c24-4713-81b2-b98caece10eb`; Personal conservó un plan, una ejecución, una relación y una ocupación, con terminal RPT. |
| Cese desde el formulario CT | HTTP 201 y `recibo:ct-seguimiento:35cf8f90aa3a5b9d478eaa5d367972d72f3ff767a7761e3f812126327b9c5241`. La fecha de fin configurada para esta copia es el 20/11/2026. |
| Finalización en Personal | La relación y **la misma ocupación** tienen revisión 2 `finalizada`, hasta 20/11/2026. Sus revisiones 1 siguen en la historia. Recibos `perrec_d5dc440a24864762aef42f8d1161ada9` (relación) y `perrec_6d5a1a9a2fb0417abab96c9a1576c60f` (ocupación); no se creó otro identificador de ocupación. La ficha HTTP 200 devuelve versión 4. |
| Vuelta a Bolsa | Tras un primer intento asíncrono fallido (`23503`), el relevo recuperó la misma operación sin duplicar: un contrato de cese, una restricción y una auditoría. Recibo `recibo:bolsa:cese:0d7d835ce8da5fa05d2abeba2b60164577c99b74054852278b6dbf5d93d72e25`; **disponible desde 20/04/2027** según la política de ejemplo v2 cargada solo en el clon. |
| Reinicio y repetición | Reiniciados aplicación y PostgreSQL sin ejecutar SQL de nuevo. Seguimiento CT HTTP 200 y ficha Personal HTTP 200 mantuvieron recibos, fecha y revisiones. El mismo POST de cese devolvió HTTP 201 y el mismo recibo; los recuentos e identificadores antes/después son idénticos. |

Personal46 conserva la definición instalada de Personal45 salvo la rama nominal de cierre de ocupación. Preimagen `pg_get_functiondef` SHA256 `9ba1f3c54a9140a2dfe6dca5ee6a55637f6a0deafb4e8dbbff88a4e82be2d258`; postimagen final `35c1b1a3bfbd49d35176c365cd02e5bd58f77b6fa55df704a5494c54d2cd2afb`. La prueba SQL transaccional terminó con `P46_OK`: cierre, repetición, conservación de historia y **19 rechazos**, incluidos selectores ajenos frente a ausentes. Su doble V3 existe solo dentro de `ROLLBACK`; el recorrido Chrome utilizó la autorización real del clon.

La segunda copia partió del PGDATA anterior a Personal46. Para reproducir las funciones de CT ya corregidas en la primera copia, se alineó **una vez** la definición literal del validador CT205 de `e43ee05c…` a `23b1bad8…` tras cotejar ambas huellas; CT206, CT207 y Personal46 se instalaron una vez. Esta alineación es preparación del clon, no ensayo de la migración CT205 ni instrucción de despliegue.

La lista Personal con 25 filas de 1.001 empleados respondió GET 200 en 52–54 ms de servidor. La ficha respondió GET 200 en 139 ms de Chrome y 97 ms de servidor, con 13 consultas fijas; a 1440 px no hubo scroll de documento y a 390 px no hubo desbordamiento horizontal global. Los selectores nuevos de historia Personal usan índices por relación y ocupación, con una búsqueda y dos bloques en caché cada uno (0,030 y 0,028 ms en el primer clon). El GET CT del plan B2 con 32 vacantes tardó 590–610 ms y 100 consultas; queda comunicado a su equipo como problema de rendimiento ajeno a Personal46.

## SQL y configuración para la principal

Cotejar primero qué está instalado y aplicar **solo lo pendiente**, en este orden: Personal45 de #1011; CT205 de #1049; CT206 de #1050; CT207 de #1052; Personal46 de esta PR. La lista ejecutable focal es [`lista_sql_codexk_personal46_20261012.txt`](../../../deploy/principal/lista_sql_codexk_personal46_20261012.txt). Personal46 exige la postimagen Personal45 y aborta ante otra definición.

**CONFIG NUEVA de Personal46: ninguna.** Para activar el recorrido completo, la instalación ya prevista requiere `VEC_PERSONAL_B2_GOBIERNO_ENABLED=true`, `VEC_CT_INCORPORACION_ACREDITADA_ENABLED=true` y `VEC_BOLSA_CESE_CT_ENABLED=true`, con sus LOGIN/provisiones nominales existentes. En el JSON privado `personal_b2.cese_fecha_efecto` debe fijarse según la decisión de RRHH; esta copia usa `primer_dia_sin_relacion` solo para el ensayo. La lectura de la lista necesita la terna AD172 ya inventariada `vec_ct_personal_b2_consultas / vec_personal.registro_empleado.empleados.v1 / personal.registro_empleado.empleados.consultar / interna_corporativa`. La política de ejemplo de Bolsa que mapea `vacante` quedó integrada en #1057; su publicación en cidonia corresponde a Dirección y no a esta rama.

## Capturas y límites observados

- [Cese confirmado en CT](cese-ct-confirmado-1440.png).
- [Ficha: relación finalizada](ficha-cese-historia-1440.png) y [periodo de la ocupación](ficha-cese-periodos-1440.png).
- [Ficha móvil, inicio de tabla](ficha-ocupaciones-390.png) y [periodo tras desplazamiento interno](ficha-ocupaciones-periodo-390.png).

Tras el cese, el panel de incorporación CT muestra «El expediente o la selección han cambiado» incluso al refrescar y deja de ofrecer el enlace a la ficha. La lista RRHH permite abrirla; la tabla de ocupaciones muestra ambas revisiones, pero carece de columna de estado y exige desplazamiento interno horizontal para ver el periodo. Dirección y M tienen avisados estos detalles. El fichero elegido en el formulario de cese aportó su huella; este recorrido no acredita custodia del documento, firma oficial ni envío externo.
