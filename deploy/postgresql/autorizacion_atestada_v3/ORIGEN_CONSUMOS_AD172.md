# Origen de consumos confirmados: AD172

AD172 añade proceso y canal a los nuevos asientos del núcleo MUTACIÓN que
alcanza su bloque de auditoría. Usa la corriente `auditoria_consumo_v3` y su
cabeza actuales, dentro de la transacción de autorización y negocio.
Las ramas delegadas a otros núcleos RRHH o exteriores requieren su propio corte.

El proceso procede de configuración técnica del DBA. El canal procede de
`vinculo_autenticacion_actor.superficie`, cuya autoridad revalida el núcleo
antes de escribir. Una fila de configuración no concede permisos de negocio.

La familia nueva es `consumo_confirmado_v2`, con `version_consumo=2`.
Los históricos mantienen `consumo_confirmado`, versión nula y proceso/canal
nulos. Los intentos y las familias de AD171 conservan sus condiciones y huellas.
No hay actualización de filas históricas ni disparador que rehaga huellas.

## Huella y ABI

La firma, los once argumentos y los siete resultados de
`consumir_decision_mutacion_v3_interna` permanecen iguales. También permanecen
el cálculo de `consumo_huella_sha256` y la referencia `aud_v3_<32 hex>`.
Proceso y canal se comprometen en el eslabón nuevo; no se atribuyen al material
V3 firmado original.

Orden de `encuadrar_mac` del eslabón:

1. `consumo_confirmado_v2`
2. `2`
3. secuencia decimal
4. anterior SHA256
5. decisión
6. efecto
7. huella del efecto
8. huella del consumo
9. proceso
10. canal

Cada campo usa longitud en octetos UTF-8, dos puntos, valor y salto de línea.
La huella SHA256 resultante se escribe en la fila y en la cabeza bajo el bloqueo
actual. Un rollback conserva ambas y los efectos de negocio anteriores.

La proyección `RegistroConsumoOrigenV2` contiene las coordenadas existentes de
`RegistroCadenaV3`, `tipo_registro`, `version_consumo`, `proceso` y `canal`.
`CotejarConsumoOrigenV2` valida un asiento; no valida por sí solo una cadena ni
autentica el checkpoint. El dispatch mixto y el parser CLI siguen perteneciendo
al corte #522 de K; este commit no los modifica.

## Preparación e instalación

La lista causal del corte añade exclusivamente AD172 a un checkpoint frío
compatible. Requiere AD171 y sus dependencias. El manifiesto
`pruebas_sql/ad172_postimagenes.json` fija dos variantes físicas exactas:
A postAD153 y K postAD168. Una combinación A/K o cualquier definición distinta
requiere comprobar y revisar otra postimagen; no se acepta por sus anclas.

La migración compara definición, fuente, ACL y configuración del núcleo,
comprueba la postimagen prevista y revierte en memoria los tres bloques del
parche para demostrar que el resto permanece idéntico. Conserva metadatos y
dependencias. La reejecución se rechaza; no incluye DOWN.

Tras ensayo y revisión, el DBA configura fuera de Git una fila exacta en
`configuracion_origen_consumos_v1`: LOGIN existente, audiencia de consumo,
operación, proceso y canal permitido. La clave es la terna LOGIN/audiencia/
operación; no admite comodines ni cambios o borrado posteriores. La aplicación
no puede leer o escribir esa tabla. Cambiar el origen de una terna exige otro
LOGIN gobernado o una migración nueva revisada. No se crean identidades,
asignaciones ni concesiones durante este corte.

La ausencia de configuración o un canal distinto rechazan el consumo nuevo
antes de sus escrituras. El replay conserva el asiento original: ejecuta las
guardas y el cotejo completo de material existentes y retorna antes de consultar
la configuración nueva. No renueva la procedencia histórica ni añade un asiento.

## Condición de cierre

El ensayo estructural del 03/10 pasó en PostgreSQL 18.4, en un único contenedor
sin red, con límite de 2 GB y datos en disco. Partió de la copia fría A post153
SHA256 `0795367fc1430bd7185ae3e01edbc2a7740ce997ea48738d3fa6f6937328aab3`.
Su acta acredita H7 → h7kit/h8kit → AD155/B77/Convoca5 →
Personal26/AD149 → AD161 → AD145 → AD153/BC8. Sobre una copia se instalaron
roles de intentos, CA26, IS13, AD169, AD171 y AD172, una vez cada una.
No se modificó el checkpoint original ni se escribió en la principal.

La prueba `pruebas_sql/ad172_origen_consumos.sql` comprueba la configuración
exacta y sus rechazos, permisos efectivos, RLS, inmutabilidad, familia histórica,
nueva familia y mezclas inadmisibles, además del vector SHA256 del ABI.
Todo el material sintético de prueba se revierte con ROLLBACK. Una fachada
temporal permite observar el resolver dentro del propietario; no reemplaza
el núcleo ni simula un consumo positivo.

Las 6.240 filas previas y la cabeza mantuvieron sus huellas; el reinicio
conservó el mismo resultado. La reejecución de AD172 fue rechazada con
SQLSTATE `55000` sin cambios. La postimagen del núcleo coincide con A post153
en `pruebas_sql/ad172_postimagenes.json`, incluido el cotejo del ABI, ACL y
dependencias que exige la migración.

Faltan el consumo firmado positivo, replay v1/v2 y rollback de negocio,
dos revisiones independientes del commit final y la integración del verificador.
La selección de fuente vigente del cold no encontró ningún par que cumpliera
simultáneamente CA26 e IS13; la fecha vigente por sí sola no acredita identidad.
No se renovaron fuentes ni se inventó una sesión para sortear ese rechazo.
Las pruebas estructurales y del resolver no acreditan esas operaciones ni
la cobertura de ramas delegadas.

Antes del despliegue, preparar las ternas exactas de configuración junto con
el DBA: un consumo nuevo sin su fila quedará denegado. Este corte no publica
una función de provisión ni habilita escritura desde la aplicación.
