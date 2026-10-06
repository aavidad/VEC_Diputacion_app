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
| 2 | **Decidido por dirección (05/10): sí; preparado en AD200** (al activar, el DBA crea el LOGIN y sus cuatro filas de origen de consumo, ver `PLAN_NOMINAL_FIRMA_AD177.md`). Grupo técnico dedicado para el gobierno del plan con una sola pertenencia, como las ramas de administración, en lugar del runtime de Contratación temporal (hoy dos LOGIN). | Dirección con K | Activar el gobierno. | Decisión escrita; si es grupo propio, AD nuevo que lo exija en la rama del núcleo. |
| 3 | Fuente nominal de sesión, perfil y certificado para emitir las decisiones V3 de firma y de recuperación con actores reales. | K | Recorrido de navegador con dos firmas. | Decisión V3 real emitida y consumida en el clon, sin actor ficticio. |
| 4a | **Hecho (#723):** la composición R5 (`componerFirmasR5`) entrega a las dos vías el decorador `RegistroConPlanV2` (descriptor del plan + emisor exterior + CT176) y nunca el registro directo de CT172; el adaptador PostgreSQL real cumple la interfaz que lo exige. | Firmas (E) | — | Prueba `TestComposicionFirmasR5RegistraSiempreConPlanCT176`. |
| 4b | Decorador Go que añade al recurso de gobierno la huella de estado, revisión y SHA del material, y kit que conserva el material exacto para reintentos del gobierno del plan. | Firmas (E) | Activar el gobierno del plan desde la aplicación. | Pruebas del decorador y del kit; reintento con los mismos bytes y la misma clave. |
| 4c | Fuentes reales para la composición R5 en la raíz (descriptor del plan con selector central, emisor V3 nominal): hoy `componerFirmasR5` sólo se ejerce en pruebas. | Firmas (E), depende de 3 | Activar R5. | Raíz compone R5 con dependencias reales en el clon. |
| 5 | **Tarea de K/administración (decidido 05/10):** sustituir las escrituras directas del bootstrap de desarrollo por fachadas gobernadas o por un LOGIN por uso; después, retirar la herencia de `vec_ad3_o207_gobierno`. Evidencia medida en el clon de main: ese LOGIN es el pool de gobierno de desarrollo (`VEC_CT_GOBIERNO_DATABASE_URL`) y pertenece con INHERIT y SET a `vec_autorizacion_propietario`, `vec_contexto_actor_v1_propietario`, `vec_autorizacion_motivos_proyector` y `vec_contratacion_temporal_gobernador`, y con SET a `vec_autorizacion_atestada_v3_migrador` y `vec_identidad_sesiones_v1_propietario`. El bootstrap escribe directamente en `vec_autorizacion.version_rol`, `asignacion_perfil`, `control_vigencia_version_rol*` y `control_catalogo_politicas`, y en las tablas de `vec_contexto_actor_v1`, y hace `SET LOCAL ROLE vec_autorizacion_propietario` (`contratacion_temporal_desarrollo.go:1466`, `contratacion_temporal_perfiles_centro_desarrollo.go:73`). Por eso una migración que corte la herencia rompería el arranque. Hoy la RLS de AUT32 exige `CURRENT_USER = vec_autorizacion_propietario`: sin SET ROLE ese LOGIN lee 0 filas; con SET ROLE las vería. | K/administración | Reduce la superficie de lectura del canon nominal. | Bootstrap sin escrituras directas ni SET ROLE a propietarios; migración que retire la herencia con ensayo de arranque en el clon. |
| 6 | Recorrido completo: navegador → dos firmas → mismo PDF verificado V2 → justificante → reinicio, con auditoría propia de cada descarga. | Firmas (E) | Depende de 1, 3 y 4. | Captura y recibos iguales antes y después de reiniciar aplicación y PostgreSQL. |

Detalles menores que quedan anotados en los documentos de cada migración: el CAS de CC7 usa SQLSTATE 40001, que un reintentador genérico repetiría; las fechas del canon del plan las aporta el material y sólo se valida su orden.

## Qué falta para 4c (medido el 05/10 sobre main)

`componerFirmasR5` necesita diez dependencias. Sólo el registro durable (CT172/CT176, `postgres.RegistroFirmasVerificadasPostgreSQL`) y el verificador GrxFirma (`validadorautofirma.Cliente`) tienen ya implementación real lista. Del emisor V3 (`firmaemisorv2.Emisor`), el autorizador nominal y el original hay piezas, pero no completas. Sin implementación fuera de pruebas: el selector central del descriptor (`plannominal.SelectorCentralDescriptorFirmaV2`), la comprobación de la publicación del plan (`plannominal.PublicacionAutorizada`), la competencia del firmante, el PDF anterior custodiado (`ports.FuentePDFFirmaAnterior`) y la política de firmantes (`ports.FuentePoliticaMismaPersonaEnPasos`, corte 4c-1 en #739). Tampoco están registradas las rutas de escritura (registro VEC, registro externo, preflight y original).

Cortes, en orden de dependencia:

| Corte | Qué | Depende de |
| --- | --- | --- |
| 4c-1 | Política de firmantes desde el circuito del catálogo (`reglas.CircuitoFirma.PermiteMismaPersonaEnPasos`). | — |
| 4c-2 | El montaje conserva el verificador como `VerificadorFirmasDocumento`; sin verificador no se compone R5. | — |
| 4c-3 | Fachada CT de lectura que compruebe la publicación del plan sobre `leer_plan_nominal_firma_v1` (hoy sólo la ejecuta el propietario CT) y `PublicacionAutorizada` en Go. | Plan publicado (gobierno, tarea 3) |
| 4c-4 | Fachada de lectura de la selección central y la competencia (CA25, Personal29, AUT32/AUT35) para el ejecutor CT antes del PDP, y `SelectorCentralDescriptorFirmaV2` y `FuenteCompetenciaFirmante` en Go. | Cargos (ver abajo) |
| 4c-5 | Emisor V3 de escritura: audiencias `firma_vec.v2` y `firma_externa.v2` (decisión interior y exterior), concesiones del perfil y fuente nominal que acepte esas rutas. | 4c-3, 4c-4 |
| 4c-6 | Original firmable: implementar `almacen.AutorizacionesDocumentosOriginalCT` y componer su cadena. | — |
| 4c-7 | PDF anterior: lectura del firmado custodiado en Documentos con su concesión de descarga. | 4c-6 |
| 4c-8 | Composición en la raíz y las cuatro rutas, con su autoridad de canal y sus entradas en la lista de rutas y de transportes mTLS. | todos |

## Cargos de quien firma

La competencia del firmante (AUT35) exige tres fuentes nominales de la misma persona: el certificado firmante vinculado (CA25, publicado por AD165 con la acción `administracion.certificados.nominal.publicar`), el cargo y su enlace de ejercicio en Personal (Personal29, publicados por AD166 con `personal.cargo_competencial.publicar`) y una asignación activa del perfil cuyo `rol_id` es el del paso del plan. AUT35 no lee las tablas `cargo_ct_*` de AD160.

- Certificado y cargo en Personal: el SQL existe y Rol7 ya tiene las dos acciones, pero vec-admin no tiene emisor ni llamada para ellas.
- Asignación del perfil de cargo: AD164 dejó cerradas las fachadas `*_plan_cargo_ct_v1` hasta tener auditoría común nominal y categoría Aplicación, y `vec-cargos-ct` necesita decisiones `administracion.perfiles.{proponer,aprobar,otorgar,recibo.consultar}` que no están en el catálogo nominal de Rol7. Abrirlas exige rol nuevo, rama del núcleo, audiencias, reapertura con auditoría común y emisor en vec-admin.
- Además no hay versiones de rol publicadas para los cargos del plan, ni un circuito gobernado para publicar roles ordinarios (hoy los publica el bootstrap de desarrollo; ver tarea 5).

Corte mínimo (aprobado por dirección el 05/10): publicar las versiones de rol de los cargos, registrarlas como asignables (AUT49) y asignarlas con el lote ordinario de Administración (AD190/AUT44, pantalla A8), y añadir a vec-admin el emisor y la llamada de AD165 y AD166. Eso sustituye el doble control de AD164 por el control del lote. Se construye sin esperar a RRHH, con la regla de quién asigna configurable; la pregunta 143 de `dudas.md` queda abierta.

## Gobierno del plan: bloqueo de ámbitos (hallado el 05/10)

AD177 y `plannominal.RecursoGobiernoPlanFirma` fijan el recurso de gobierno sin ámbitos (`"ambitos":{}` en la huella de contexto). El PDP común exige que el recurso tenga exactamente las dimensiones de la asignación (`AsignacionPerfil.Cubre`), y una asignación siempre tiene al menos un ámbito; la del administrador con Rol7 tiene organización y unidad, y el catálogo nominal de Rol7 declara esas dos dimensiones para `vec.catalogos.*`. Resultado: cualquier decisión de gobierno del plan se deniega con `ambito_no_autorizado` antes de llegar a AD177. Ningún ensayo lo había detectado porque nadie emitía todavía esa decisión.

Aprobado por dirección el 05/10: que el recurso de gobierno lleve `organizacion_ref` y `unidad_ref` de la asignación del administrador, y una migración nueva que sustituya `registrar_y_confirmar_gobierno_plan_firma_v1` para recibirlos, cotejarlos con la asignación de la decisión consumida (fachada AUT) y calcular con ellos la huella de contexto. El material del kit (13 claves) y CC7 no cambian.

## Corte 4c-3: publicación vigente del plan (CC8 y CT178)

El descriptor del plan (`plannominal.Fuente`) relee en cada firma el catálogo del plan y pide a `PublicacionAutorizada` que confirme que es la publicación vigente. `leer_plan_nominal_firma_v1` (CC7) no sirve antes del PDP porque exige un consumo de firma de la misma transacción. CC8 añade `comprobar_publicacion_plan_nominal_firma_v1(id, versión, SHA)`, de sólo lectura, con las mismas comprobaciones de publicación que CC7 y sin devolver el documento: sólo fecha y revisión. Lo ejecuta el propietario CT; CT178 lo ofrece al ejecutor CT. En Go, `postgres.PublicacionPlanFirmaPostgreSQL` coteja además la fecha y la revisión con el catálogo leído (la fecha de publicación debe tener precisión de microsegundo: PostgreSQL redondea y una fecha más fina no coincidiría nunca). La comprobación definitiva sigue en CT176 con el pin y el consumo.
