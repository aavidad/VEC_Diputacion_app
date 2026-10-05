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
| 2 | **Decidido por dirección (05/10): sí.** Grupo técnico dedicado para el gobierno del plan con una sola pertenencia, como las ramas de administración, en lugar del runtime de Contratación temporal (hoy dos LOGIN). | Dirección con K | Activar el gobierno. | Decisión escrita; si es grupo propio, AD nuevo que lo exija en la rama del núcleo. |
| 3 | Fuente nominal de sesión, perfil y certificado para emitir las decisiones V3 de firma y de recuperación con actores reales. | K | Recorrido de navegador con dos firmas. | Decisión V3 real emitida y consumida en el clon, sin actor ficticio. |
| 4a | **Hecho (#723):** la composición R5 (`componerFirmasR5`) entrega a las dos vías el decorador `RegistroConPlanV2` (descriptor del plan + emisor exterior + CT176) y nunca el registro directo de CT172; el adaptador PostgreSQL real cumple la interfaz que lo exige. | Firmas (E) | — | Prueba `TestComposicionFirmasR5RegistraSiempreConPlanCT176`. |
| 4b | Decorador Go que añade al recurso de gobierno la huella de estado, revisión y SHA del material, y kit que conserva el material exacto para reintentos del gobierno del plan. | Firmas (E) | Activar el gobierno del plan desde la aplicación. | Pruebas del decorador y del kit; reintento con los mismos bytes y la misma clave. |
| 4c | Fuentes reales para la composición R5 en la raíz (descriptor del plan con selector central, emisor V3 nominal): hoy `componerFirmasR5` sólo se ejerce en pruebas. | Firmas (E), depende de 3 | Activar R5. | Raíz compone R5 con dependencias reales en el clon. |
| 5 | **Tarea de K/administración (decidido 05/10):** sustituir las escrituras directas del bootstrap de desarrollo por fachadas gobernadas o por un LOGIN por uso; después, retirar la herencia de `vec_ad3_o207_gobierno`. Evidencia medida en el clon de main: ese LOGIN es el pool de gobierno de desarrollo (`VEC_CT_GOBIERNO_DATABASE_URL`) y pertenece con INHERIT y SET a `vec_autorizacion_propietario`, `vec_contexto_actor_v1_propietario`, `vec_autorizacion_motivos_proyector` y `vec_contratacion_temporal_gobernador`, y con SET a `vec_autorizacion_atestada_v3_migrador` y `vec_identidad_sesiones_v1_propietario`. El bootstrap escribe directamente en `vec_autorizacion.version_rol`, `asignacion_perfil`, `control_vigencia_version_rol*` y `control_catalogo_politicas`, y en las tablas de `vec_contexto_actor_v1`, y hace `SET LOCAL ROLE vec_autorizacion_propietario` (`contratacion_temporal_desarrollo.go:1466`, `contratacion_temporal_perfiles_centro_desarrollo.go:73`). Por eso una migración que corte la herencia rompería el arranque. Hoy la RLS de AUT32 exige `CURRENT_USER = vec_autorizacion_propietario`: sin SET ROLE ese LOGIN lee 0 filas; con SET ROLE las vería. | K/administración | Reduce la superficie de lectura del canon nominal. | Bootstrap sin escrituras directas ni SET ROLE a propietarios; migración que retire la herencia con ensayo de arranque en el clon. |
| 6 | Recorrido completo: navegador → dos firmas → mismo PDF verificado V2 → justificante → reinicio, con auditoría propia de cada descarga. | Firmas (E) | Depende de 1, 3 y 4. | Captura y recibos iguales antes y después de reiniciar aplicación y PostgreSQL. |

Detalles menores que quedan anotados en los documentos de cada migración: el CAS de CC7 usa SQLSTATE 40001, que un reintentador genérico repetiría; las fechas del canon del plan las aporta el material y sólo se valida su orden.
