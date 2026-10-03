# AD178: preparación de recuperación R5 nominal

AD178 está reservada y sigue siendo un borrador bloqueado. Prepara la recuperación
R5 de 48 campos solicitada por E. No instala gobierno ni lectura del plan y no
crea perfiles humanos, LOGIN, grupos técnicos, membresías o concesiones nominales.

## Contrato preparado

El clasificador fijo es `recuperacion_firmas_r5_ct_v2`. Usa el mismo runtime
técnico CT que `consulta_firmas_r5_ct_v2`: la comprobación existente de
`vec_contratacion_temporal_ejecutor` permanece intacta. La clasificación técnica
no concede a una persona permiso para recuperar un canon. El núcleo conserva su
cotejo de identidad, contexto, decisión nominal y autorización viva.

| Coordenada | Valor exacto |
| --- | --- |
| Acción | `contratacion_temporal.documento.firmas_r5_v2.recuperar` |
| Audiencia | `vec_contratacion_temporal.firmas_r5.recuperar.v2` |
| Módulo | `contratacion_temporal` |
| Tipo de recurso | `expediente_contratacion_temporal` |
| Finalidad | `gestionar_contratacion_temporal` |
| Superficie | `interna_corporativa` |
| Recurso | `ExpedienteRef`, igual a `efecto_ref` |
| Huella del efecto | Huella del contexto autorizado: ámbito `organizacion_ref` y atributo `material_sha256` |
| Obligaciones | Array vacío |

Los campos son exactamente `CamposRecuperacionFirmasV2()` de E, contrato
`d9dd19fd3`: los 44 de consulta R5 más `CanonNominal`, `CanonNominalRef`,
`CanonNominalSHA256` y `MaterialRootSHA256`, ordenados. La lista completa está
en `pruebas_sql/ad178_referencia_post173.json` y en las dos guardas SQL del
delta/fachada. La consulta de 44 campos conserva su acción, audiencia y cuerpo.

La nueva fachada es:

```text
vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(
  p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
  p_persona_version numeric, p_perfil_version numeric,
  p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
)
RETURNS TABLE (
  decision_ref text, efecto_ref text, huella_efecto_sha256 text,
  consumo_huella_sha256 text, auditoria_ref text,
  consumida_en timestamptz, consumo_nuevo boolean
)
```

El núcleo recibe once argumentos al anteponer el clasificador fijo. La fachada
exige una autorización nueva en una transacción SERIALIZABLE de escritura.
Un consumo anterior de la misma decisión se rechaza; una lectura repetida requiere
otra decisión nominal. Sólo el propietario AUT y el propietario técnico CT reciben
EXECUTE sobre la fachada. No se concede acceso directo al núcleo ni SELECT a sus
tablas. La ACL final se comprueba dentro de la migración.

La fachada consume y audita; no devuelve el canon histórico. K debe aportar la
lectura histórica autorizada que conserve los bytes originales. Recuperar una
firma anterior no exige ni inventa un cargo actual de su firmante. E conserva el
cotejo del canon recuperado contra la firma/PDF registrados y la raíz original.

## Delta y cuarentena de instalación

La migración añade una sola condición de recuperación junto al bloque de consulta
R5 V2 y una audiencia al CHECK de `clave_capacidad_version`. No altera la
clasificación runtime, firma V2, cálculo del consumo, replay del núcleo,
configuración de origen AD172, eslabón nominal AD173 ni filas históricas.

Las seis huellas finales de definición/fuente/CHECK, antes y después del delta,
siguen a NULL. La primera guarda aborta con SQLSTATE `55000`:

```text
AD178: PARO clave=pre_post_aprobadas actual=NULL
esperado=POST175_176_medida_y_POST178_revisada
```

El PARO ocurre antes del DDL y las concesiones. Después de medir la base final,
la migración compara ambas preimágenes del núcleo, propietario, configuración,
ACL y dependencias, exige una única ancla, coteja las postimágenes y revierte el
delta en memoria para demostrar la conservación del resto del cuerpo. El CHECK
de audiencias también requiere pre/postimagen exacta; no se acepta por nombre.

La referencia offline procede de la captura real POST173 conservada por dirección:
definición `6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9`,
fuente `bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c`.
Una inserción y su reversión exacta dejan los 110 clasificadores anteriores y
añaden únicamente el de recuperación. La referencia reconstruida produce
definición `f7baf39764debe75cf88b5e24271a676bc80b014e14d946f81b50f1e69967ffc`
y fuente `7bf0c49974edf7898575c9fc7f41e78962ce10a8f16edeaf3feb8a6e25a595ea`.
Esas huellas son referencia de texto, no pre/postimágenes de instalación sobre
las futuras AD175/176. No se han trasladado a las guardas de la migración.

## Gobierno y lectura pendientes

`gobierno_plan_nominal_firma_ct` queda reservado en el contrato de E. AD177 fija
las acciones `vec.catalogos.crear/actualizar/publicar/retirar`, audiencia
`vec_catalogos_configurables.plan_nominal_firma.gobierno.v1`, módulo CT, recurso
`catalogo_configurable`, finalidad `gestionar_contratacion_temporal`, campos y
obligaciones vacíos. CC7 liga el efecto a `catalogo_id:version`, SHA del material,
estado y revisión esperados, y compara actor/perfil con el consumo común.

Falta el grupo técnico autorizado para ese gobierno. AD178 no reutiliza el runtime
CT general para conceder administración del catálogo ni incorpora una rama de
gobierno vacía. La plantilla histórica AD147 es de Cronos y no es una autorización
para este plan CT. El grupo y la fachada se completarán en un corte revisado.

`consulta_plan_nominal_firma_ct` sigue pendiente de acción, audiencia, finalidad,
recurso y campos exactos de E. Tampoco tiene una rama aceptada. La revalidación
privada del pin en COMMIT CT de CC7 consume la firma vigente; no convierte al
firmante en administrador ni acredita una lectura administrativa nueva.

El orden causal es AD173 → AD174/175/176 reancladas → delta L → AD177 → CC7.
AD177 exige la postimagen del delta L, pese a tener un número menor. El borrador
actual de AD178 cubre sólo recuperación48 y por sí solo no habilita el gobierno
que necesita AD177/CC7. La lista causal entregada advierte de esta cuarentena.

## Aceptación preparada y límites

`pruebas_sql/ad178_recuperacion_contrato.sql` prepara la comprobación de ABI/ACL y
cinco rechazos: ausencia de cada campo nuevo y uso de sólo 44 campos. Usa entradas
sintéticas negativas; no fabrica un consumo V3 válido ni escribe historia. Se
ejecutará únicamente en el clon después de completar y revisar AD178.

Antes de integración faltan la base causal final y sus seis huellas, el grupo de
gobierno y contrato de lectura, la fachada histórica nominal de K, el fixture
firmado vigente, ensayo PostgreSQL y dos revisiones independientes del hash final.
El ensayo debe conservar historia/ACL/firma V2/consulta44, comprobar denegación
ante cualquier coordenada distinta, el asiento común y rollback sin efectos.

No se ejecutaron Go, PostgreSQL, SQL en principal/cidonia, DOWN ni reaplicaciones.
La comprobación hecha hasta ahora es la reconstrucción offline de texto sobre
POST173; no acredita instalación, consumo firmado ni recuperación histórica.
Semgrep local comprobó las dos SQL con tres reglas: cero hallazgos. Las dos listas
de 48 campos coinciden con el contrato Go de E y `git diff --check` está limpio.
