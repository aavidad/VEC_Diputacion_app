# Alta en Personal y cese con vuelta a Bolsa — 12/10/2026

**Entorno:** Chrome del sistema y PostgreSQL 18.4 en un contenedor propio con 2 GiB. La base procede de una copia fría de la principal; todos los datos de este recorrido son inventados. No se modificó cidonia. Las migraciones de este recorrido se instalaron una vez, sin `DOWN` ni reaplicación.

## Recorrido comprobado

| Paso | Respuesta y efecto observado |
| --- | --- |
| Alta desde el expediente CT `2026/94014` | Confirmación HTTP 200. CT devolvió `recibo:incorporacion-personal-b2:8ea5f26c-c52f-4170-9d28-e1c2024d0375`; Personal conservó un plan, una ejecución, una relación y una ocupación, con su terminal RPT. El enlace del recibo abrió la ficha. |
| Cese desde el formulario CT | HTTP 201 y `recibo:ct-seguimiento:ec02dbbf5cef9bed3804b55b8c9dd1ec32dece745df5a53bfc56c5807c520c55`. La fecha de fin configurada para esta copia es el 20/11/2026. |
| Finalización en Personal | La relación y **la misma ocupación** tienen revisión 2 `finalizada`, hasta 20/11/2026. Sus revisiones 1 siguen en la historia. Recibos `perrec_ffa740fb6bf344028c5ad141cf250e03` (relación) y `perrec_ac96d6dfa02c4328b31a74bb1a0042b5` (ocupación); una sola ocupación opaca, sin crear otro `ocu_*`. La ficha HTTP 200 devuelve versión 4. |
| Vuelta a Bolsa | Un contrato de cese, una restricción y una auditoría nominal. Recibo `recibo:bolsa:cese:ee752992548c9e401633900b0c01ab5b1497fc8116e9dd1b45cc2874ee23e303`; **disponible desde 20/04/2027** según la política de ejemplo v2 cargada únicamente en el clon. |
| Reinicio y repetición | Reiniciados aplicación y PostgreSQL sin ejecutar SQL de nuevo. Seguimiento CT HTTP 200 y ficha Personal HTTP 200 mantuvieron recibos, fecha y revisiones. El mismo POST de cese devolvió HTTP 201 y el mismo recibo; los recuentos e identificadores antes/después son idénticos. |

Personal46 conserva la definición instalada de Personal45 salvo una rama nominal de cierre de ocupación. Preimagen `pg_get_functiondef` SHA256 `9ba1f3c54a9140a2dfe6dca5ee6a55637f6a0deafb4e8dbbff88a4e82be2d258`; postimagen `9ee3dbb80a80ee8599d9cdd27cfa173ea93e2c3f6fce5ccca948b81365be9b49`. La prueba SQL transaccional terminó con `P46_OK`: cierre, repetición, conservación de historia y 12 rechazos; el doble V3 de esa prueba existe solo dentro de `ROLLBACK`. El recorrido Chrome utilizó la autorización V3 real del clon.

Los dos selectores nuevos de historia Personal usan índices: `Index Scan Backward` por relación y ocupación, una búsqueda y dos bloques en caché cada uno; EXPLAIN midió 0,030 y 0,028 ms. En Chrome, ficha GET 200 tardó 53 ms y lista GET 200 tardó 102 ms; el servidor registró 60 ms para la lista, por debajo de 100 ms. El POST de repetición, que recupera dos actos nominales, tardó 669 ms. A 1440 px no hubo scroll de documento; a 390 px no hubo desbordamiento horizontal global.

## SQL y configuración para la principal

Cotejar primero qué está instalado y aplicar **solo lo pendiente**, en este orden: Personal45 de #1011; CT205 de #1049; CT206 de #1050; CT207 de #1052; Personal46 de esta PR. La lista ejecutable focal es [`lista_sql_codexk_personal46_20261012.txt`](../../../deploy/principal/lista_sql_codexk_personal46_20261012.txt). Personal46 exige la postimagen Personal45 y aborta ante otra definición.

**CONFIG NUEVA de Personal46: ninguna.** Para activar el recorrido completo, la instalación ya prevista requiere `VEC_PERSONAL_B2_GOBIERNO_ENABLED=true`, `VEC_CT_INCORPORACION_ACREDITADA_ENABLED=true` y `VEC_BOLSA_CESE_CT_ENABLED=true`, con sus LOGIN/provisiones nominales existentes. En el JSON privado `personal_b2.cese_fecha_efecto` debe fijarse según la decisión de RRHH; esta copia usa `primer_dia_sin_relacion` solo para el ensayo. La lectura de la lista necesita la terna AD172 ya inventariada `vec_ct_personal_b2_consultas / vec_personal.registro_empleado.empleados.v1 / personal.registro_empleado.empleados.consultar / interna_corporativa`; faltaba en el clon y se añadió allí directamente. La política de ejemplo de Bolsa que mapea `vacante` queda a cargo de Codex-I y Dirección; no se publica desde esta rama.

## Capturas y límites observados

- [Recibo de alta CT](alta-ct-recibo-1440.png).
- [Ficha: relación finalizada](ficha-cese-historia-1440.png) y [periodo de la ocupación](ficha-cese-periodos-1440.png).
- [Ficha en móvil](ficha-ocupaciones-390.png).

Tras el cese, el panel de incorporación CT muestra «El expediente o la selección han cambiado» incluso al refrescar y deja de ofrecer el enlace a la ficha. La lista RRHH sí abre la ficha una vez provisionada su terna AD172. La tabla de ocupaciones muestra ambas revisiones, pero carece de columna de estado y exige desplazamiento interno horizontal para ver el periodo; Dirección tiene avisados estos dos detalles de interfaz. El fichero escogido en el formulario de cese aportó su huella; este recorrido no acredita custodia del documento, firma oficial ni envío externo.
