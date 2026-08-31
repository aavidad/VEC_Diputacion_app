# VEC-DOC-AUTORIDAD-OBJETO-ESPERADO-V1

Fecha: 31 de agosto de 2026.

## Capacidad y limite

Este corte incorpora una capacidad opaca en `internal/vec/ports` que autoriza
la pareja exacta `objeto_ref` y `objeto_version` esperada al registrar un efecto
documental V4. No modifica SQL, migraciones, producto, composicion, identidad,
UI ni otros modulos.

El puerto `AlmacenObjetos.Escribir` determina y devuelve la referencia y la
version despues de escribir. No dispone de reserva previa ni de una identidad
de objeto determinista publicada. Por ello el modelo vigente no puede crear un
compromiso honesto de esa pareja antes de la escritura sin cambiar la autoridad
del almacen y todos sus adaptadores.

La autoridad reutilizada es `ReciboEscrituraObjetoMaterialV2`: liga mediante
atestacion verificable la instantanea material del objeto, los hechos estables
de contexto, el perfil de capacidades homologado, el plan material publicado y
una referencia durable original. Este corte no introduce claves ni algoritmos
criptograficos nuevos.

V2 sigue siendo un contrato sin atestador, registro de referencias ni diario
durable productivos. La nueva capacidad cierra la forma tipada que necesitaba
el registro SQL, pero no declara operativo el flujo ni sustituye esos
adaptadores pendientes.

## Invariante

`AutoridadObjetoEsperadoDocumentalV1` solo se construye cuando se cumplen a la
vez estas condiciones:

1. la declaracion V4 y el recibo material V2 son nominalmente validos;
2. la instantanea atestada coincide exactamente con el resultado V4 en
   conector, referencia, version, zona, MIME, tamano, SHA-256 de contenido,
   evidencia de creacion, instante, retencion, inmovilizacion y estado;
3. modulo, acciones, recurso, operacion, carga, efecto y clasificacion del
   recibo coinciden con el contexto V4, y el efecto coincide tambien con el
   vinculo de ejecucion consumido;
4. la atestacion del recibo se verifica mediante
   `VerificadorAtestacionMaterialAlmacenV2`; y
5. la referencia durable se reconstruye desde la identidad material sin
   referencia y se coteja mediante `VerificadorReferenciaReciboMaterialV2`.

La huella autocontenida de `EvidenciaOperacionAlmacen`, el contenido guardado o
un JSON aportado por el ejecutor no satisfacen ninguna de las dos ultimas
condiciones. Tampoco pueden construir ni serializar genericamente la autoridad.

`PrepararRegistro` es la unica apertura prevista hacia el futuro adaptador SQL.
Repite en vivo las verificaciones criptografica y de referencia durable antes
de entregar copias defensivas de la pareja objeto/version y del material de
verificacion. Una revocacion, una sustitucion o una cancelacion fallan cerradas.

## Frontera criptografica

El contrato V2 vigente admite HMAC-SHA-256 o COSE Sign1. Una clave HMAC sigue
perteneciendo a la autoridad criptografica configurada: no se copia al valor,
no se entrega al adaptador SQL y no se almacena en PostgreSQL. La proyeccion
solo contiene referencia y version de clave, dominio, algoritmo, codigo y bytes
canonicos del recibo.

Si el registro SQL necesita verificacion autonoma fuera del proceso que posee
el verificador V2, el siguiente corte debera usar una autoridad publica ya
gobernada (por ejemplo, la modalidad COSE prevista por V2) o un registro durable
de recibos previamente verificados. No se autoriza trasladar una clave HMAC a
PostgreSQL ni degradar esta condicion a una igualdad de JSON o SHA-256.

## Siguiente corte SQL

El adaptador SQL debe recibir la capacidad opaca, invocar `PrepararRegistro`
con las autoridades V2 configuradas y derivar `objeto_ref` y `objeto_version`
exclusivamente de su proyeccion. Debe cruzar ademas `huella_plan_sha256` y
`huella_efecto_sha256` con los compromisos P0-1 ya disponibles, conservar la
atomicidad con estado, auditoria y outbox, y cerrar por rol/ACL la funcion de
confirmacion para que el ejecutor no pueda invocarla con valores libres.
La activacion productiva seguira bloqueada hasta disponer de los adaptadores y
la recuperacion durable V2 ya pendientes.

Queda expresamente prohibido en ese corte:

- aceptar la pareja desde `contenido_guardado` o evidencia JSON;
- tratar la proyeccion, que es deliberadamente no autoritativa, como entrada
  publica reconstruible;
- inventar una clave compartida con PostgreSQL; o
- omitir la reverificacion de la atestacion y de la referencia durable.
