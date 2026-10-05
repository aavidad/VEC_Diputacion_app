# Firmas de Contratación temporal: inventario y continuación

Estado del 5 de octubre de 2026, sobre `origin/main` con AD195 y AD196 ya fusionadas. La firma se hace con GrxFirma (antes AutofirmaV2) por el protocolo `afirma://` desde el navegador, y la verificación con su validador como servicio aparte. Este plan no da por firmado ningún documento ni por verificado ningún PDF.

## Qué existe

| Pieza | Dónde y estado |
| --- | --- |
| GrxFirma V2, kit de material exacto y driver de recuperación nominal | Ya en main (#558, #579, #588, #656, #657). |
| Pin del plan en el descriptor y servicio de recuperación de 48 campos | En main (#702, #703, antes #568 y #569 de Codex-E). |
| Lector PostgreSQL y API de recuperación, firma con plan, vista de consulta histórica y emisión de la autorización de recuperación | PR #705, #706, #707 y #708, con CI verde y revisión. |
| SQL de la cadena: AD178 y AD177 | PR #697. Preimágenes medidas sobre main posterior a AD195/AD196: núcleo `728dde66` → `2ccd704a`, CHECK de audiencias `26497f11` → `e76428d2`. |
| SQL de recuperación: AUT41 y CT175 | PR #698, sobre #697. Sin huellas del núcleo. |
| SQL del plan: CC7 y CT176 | PR #701, sobre #697. Sin huellas del núcleo. |

Las pruebas de cada migración están junto a ella, en `pruebas_sql/`. Las que acaban en `_positivo_clon.sql` recorren el caso que funciona con filas sintéticas dentro de un ROLLBACK y sólo se ejecutan en un clon desechable.

## Orden de instalación

AD193 → AD195/AD196 → AD178 → AD177 → AUT41 → CT175 → CC7 → CT176, con `lista_sql_claude_firmas_ad178_ad177_20261005.txt`, `lista_sql_claude_firmas_recuperacion_20261005.txt` y `lista_sql_claude_firmas_plan_20261005.txt`. Una sola vez, sin DOWN. AD178 mide el núcleo: si entra antes otra migración que lo reescriba, AD178 se detiene sin tocar nada y hay que remedirla.

## Cola pendiente con dueño

| Orden | Tarea | Dueño propuesto | Bloquea | Cierre verificable |
| --- | --- | --- | --- | --- |
| 1 | Extender `vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1` a `vec.catalogos.{crear,actualizar,publicar,retirar}` **sólo** con `tipo_recurso=catalogo_configurable` y la audiencia `vec_catalogos_configurables.plan_nominal_firma.gobierno.v1`. Si se abre a todo `vec.catalogos.*`, las fachadas generales de catálogos quedarían autorizables. | K (identidad y permisos) | Gobierno del plan (crear, publicar y retirar el plan de quién firma). | Migración AUT nueva con preimagen medida; la prueba positiva de AD177 pasa sin sustituir la categoría Aplicación. |
| 2 | Decidir si el gobierno del plan usa un grupo técnico dedicado con una sola pertenencia, como las ramas de administración, en lugar del runtime de Contratación temporal (hoy dos LOGIN). | Dirección con K | Activar el gobierno. | Decisión escrita; si es grupo propio, AD nuevo que lo exija en la rama del núcleo. |
| 3 | Fuente nominal de sesión, perfil y certificado para emitir las decisiones V3 de firma y de recuperación con actores reales. | K | Recorrido de navegador con dos firmas. | Decisión V3 real emitida y consumida en el clon, sin actor ficticio. |
| 4 | Montaje: que la firma V2 llame a CT176 (`registrar_firma_con_plan_v2`, #706) y no a CT172 directamente. Tras instalar CT176, una firma V2 por CT172 falla al confirmar (23514). Además, el decorador Go que añade al recurso la huella de estado, revisión y SHA del material, y el kit que conserva el material exacto para reintentos del gobierno. | Firmas (E) | Activar R5 y el gobierno del plan desde la aplicación. | Composición raíz con CT176; pruebas de COMMIT incierto y cancelación sin datos entregados. |
| 5 | Retirar la pertenencia del LOGIN `vec_ad3_o207_gobierno` a `vec_autorizacion_propietario` (con INHERIT y SET puede leer AUT32 directamente sin pasar por AUT41), o justificarla. Es anterior a esta cadena; hay que confirmarlo también en la principal. | Dirección con K | Nada inmediato; reduce la superficie de lectura del canon nominal. | Consulta de pertenencias en la principal y, si procede, migración que la retire. |
| 6 | Recorrido completo: navegador → dos firmas → mismo PDF verificado V2 → justificante → reinicio, con auditoría propia de cada descarga. | Firmas (E) | Depende de 1, 3 y 4. | Captura y recibos iguales antes y después de reiniciar aplicación y PostgreSQL. |

Detalles menores que quedan anotados en los documentos de cada migración: el CAS de CC7 usa SQLSTATE 40001, que un reintentador genérico repetiría; las fechas del canon del plan las aporta el material y sólo se valida su orden.
