# RUM04-P1: consulta propia interna — WIP

Este corte prepara la consulta nominal de un hecho propio y su ficha actual. Está pendiente de revisión y de completar el orden causal de migraciones; no tiene GO de entrega ni acredita una instalación institucional.

El cliente envía únicamente `hecho_ref`. El servidor revalida la sesión, reconstruye el contexto y obtiene la persona del vínculo autenticado. La acción fija es `meritos.hecho.consultar_propio`, la finalidad `consulta_hecho_propio` y la audiencia `vec_meritos.hecho.consultar_propio.v1`. El perfil admite solamente `hecho_actual` y `recibo_consulta`, con obligación de auditar. No exige la condición de empleado para leer el hecho de la persona autenticada.

La función PostgreSQL consume la autorización nominal y recupera la versión actual en una misma transacción serializable. Devuelve una ficha minimizada y un recibo de consulta nuevo después del commit. La ficha excluye las referencias de persona, declarante y actor revisor. Una referencia ajena y una ausente producen la misma respuesta de ausencia tras una autorización positiva. La lectura conserva la historia de negocio; solamente añade la constancia de consulta y la auditoría de autorización.

La provisión del perfil, las concesiones y las claves se realiza separadamente, con huella y CAS. Ninguno de esos valores se acepta en una petición HTTP. El rol técnico propio carece de LOGIN, herencia y privilegios elevados; el login de ejecución tiene una única membresía nominal y no obtiene acceso directo a las tablas.

La pantalla y el receptor loopback están preparados para el ensayo interno con identidades sintéticas y las autoridades reales del clon. El receptor no se monta en la raíz institucional. La superficie externa y la verificación de méritos continúan cerradas. Una ficha consultada no acredita el mérito ni sustituye una revisión o firma.

## Estado comprobado

En un clon privado nuevo, las migraciones nuevas se instalaron una vez tras AD142 y Méritos000001. La consulta nominal real recuperó el hecho propio en versión 3 y una ausencia; la consulta de otra persona al mismo hecho devolvió ausencia. Tras reiniciar PostgreSQL y ejecutar otro proceso, la consulta propia volvió a recuperar la versión 3 y la ausencia. Las cuatro tablas de negocio conservaron sus huellas y sus recuentos: un hecho, tres versiones, tres operaciones y tres eventos de outbox. Los recibos de consulta pasaron de cuatro a seis por las dos nuevas lecturas.

Las pruebas focales de aplicación, adaptadores y HTTP pasaron en Go normal/race/vet, con análisis local de seguridad sin hallazgos. La interfaz pasó sus 22 pruebas con Node20 y los catálogos por idioma. Estas comprobaciones no sustituyen Chrome ni las revisiones independientes pendientes.

## TODO antes de cerrar el corte

- Conservar las reservas propias AD000145 y Méritos000002. El orden central pendiente es AD142 → AD143 (K) → AD144 (G) → AD145 (A). Para esta pieza, el orden de instalación es roles de consulta propia → AD145 → Méritos000002, tras sus prerrequisitos.
- Actualizar la preimagen de AD145 cuando AD143 y AD144 estén publicadas y ensayar el conjunto en un clon nuevo. La candidata actual protege la preimagen posterior a AD142 y no acredita compatibilidad posterior a AD144. No relajar las guardas ni reaplicar UP/DOWN de migraciones con historia.
- Completar el recorrido con Chrome del sistema y la revisión independiente de usabilidad.
- Obtener dos revisiones del hash exacto, incluida SQL, autorización y seguridad, y ejecutar la puerta de calidad de cierre correspondiente.
- Acreditar la auditoría de la denegación SQL que sucede después de emitir una autorización y termina en rollback; este ensayo no la demuestra.
- Preparar la PR funcional sólo después de resolver esos pendientes. La rama actual es WIP y RUM04 permanece en cola, no cerrado.

No se incluyen claves, configuraciones privadas, SQL de provisión ni datos del clon en esta entrega.
