# Cronos: preparar el motivo de auditoría del vínculo caducado

La PR #861 incorporó la guarda y el registro común del rechazo. Para arrancar
Cronos empleado falta una entrada gobernada para
`auditoria_intentos.motivo_denegado`. Se propone el texto de catálogo
**«Vínculo de empleado no vigente»**. V y Dirección deben aprobar la entrada,
su clave y su publicación. Este documento no acredita que estén aprobadas ni
instaladas.

## Publicación por la autoridad del catálogo

V debe obtener de la fuente maestra la versión vigente, sus entradas positivas,
la huella y el último evento confirmado. La [versión de catálogo](../../internal/vec/domain/catalogos.go)
es consecutiva y conserva historia. La versión ya publicada no se modifica:
se prepara una nueva instantánea **completa**, con las entradas positivas
conservadas y la nueva entrada de rechazo. La clave nueva sigue el patrón
`motivo_` más 32 caracteres hexadecimales; su vigencia se declara con fecha UTC
y precisión de microsegundos. La huella corresponde a toda la nueva versión,
no a esa entrada aislada.

La autoridad maestra confirma primero esa versión y su evento. Su proyector
exclusivo transmite el mismo catálogo, huella, evento y secuencia a
[`vec_autorizacion.publicar_motivos_autorizacion_v2`](../../deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql).
El checkpoint exige una secuencia contigua y un replay idéntico. Si maestro y
proyección comparten base, la publicación ocurre en la misma transacción; si
están en bases distintas, se aplica el circuito de invalidación previo definido
en la [persistencia de Autorización](../../deploy/postgresql/autorizacion/README.md).
La proyección SQL no sustituye la aprobación ni la fuente maestra. La
composición administrativa durable de `ServicioCatalogos` aún no está
acreditada para este corte; V debe identificar el canal operativo autorizado.
No se reaplica ni revierte la migración `000003` con historia.

## Configuración privada después de publicar

En `identidad/cronos-empleado.json`, bajo el director privado de material, se
añade la referencia publicada en `auditoria_intentos.motivo_denegado`. Sus
cuatro campos son exactamente `catalogo_id` (texto), `catalogo_version`
(entero), `catalogo_huella_sha256` (64 caracteres hexadecimales) y
`entrada_clave` (`motivo_` más 32 hexadecimales). El ID es el mismo que el de
`motivos.saldo`. La entrada de rechazo tiene una clave distinta de las
positivas activas.

Las referencias positivas de `motivos` se actualizan a la publicación que
resulte, con la versión y huella emitidas por la autoridad; cada acción
conserva su clave positiva. No se copian coordenadas de pruebas ni se calculan
huellas a mano. La cuenta, el proceso y el canal del registrador común se
configuran por el material privado de `auditoria-intentos.json` y deben pasar
el preflight de AD169. Ninguno de esos ficheros privados entra en Git.

## Comprobación y condición de término

La entrega revisable conserva la evidencia del acto de la fuente maestra, la
proyección confirmada con evento y secuencia, y las cuatro coordenadas de cada
referencia que se instalará. Con datos sintéticos, se comprueba que el
validador de motivos resuelve positivamente la nueva entrada y las positivas,
que Cronos arranca con la configuración nueva y que un vínculo caducado
produce 403 **tras** el acuse de auditoría común. Si el registro falta o falla,
la misma operación devuelve 503 sin datos ni efecto en el repositorio de
negocio. La prueba local no equivale a publicación ni a uso real.

Hasta que V confirme la entrada y el canal de publicación, Cronos empleado
permanece cerrado. La aprobación pendiente sigue en su circuito y no detiene
las partes independientes de otros módulos.
