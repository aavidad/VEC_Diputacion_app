# VEC-DOC-AUTORIDAD-OBJETO-ESPERADO-V1

Fecha: 31 de agosto de 2026.

## Capacidad y limite

Este corte incorpora una capacidad opaca en `internal/vec/ports` que autoriza
la pareja exacta `objeto_ref` y `objeto_version` esperada al registrar un efecto
documental V4. La pareja se deriva exclusivamente de la `instantanea` atestada
y referenciada del recibo material V2; la declaracion V4 solo se coteja y no es
fuente de esos dos valores. No modifica SQL, migraciones, producto, composicion,
identidad, UI ni otros modulos.

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

La capacidad conserva copias defensivas del recibo y de la declaracion V4. El
clon focal V4 alcanza el contexto, sus pasos, las capacidades y toda la rama
mutable del vinculo de ejecucion, incluido `vinculoActivacion` y los datos
privados de su manifiesto; no modifica las autoridades fuente.

La huella canonica V2 excluye deliberadamente la atestacion. Por ello la
capacidad conserva ademas una copia separada de su material exacto y aplica un
sello SHA-256 con dominio propio sobre los bytes canonicos V2, algoritmo,
referencia y version de clave, dominio y codigo completo —HMAC o sobre
COSE_Sign1—. Este sello no cambia la huella ni la semantica V2.

`Validar` vuelve a comprobar ambos sellos, el contexto V4 completo y, de forma
individual, efecto, huella del plan, huella del manifiesto, referencia y huella
del paso. `PrepararRegistro` repite el mismo cierre antes y despues de cada uno
de los dos verificadores vivos y antes de proyectar; una mutacion durante
cualquiera de esas fronteras devuelve el error opaco y una proyeccion cero. La
proyeccion entrega la huella de la declaracion, no sus bytes ni una via para
reconstruirla.

La huella autocontenida de `EvidenciaOperacionAlmacen`, el contenido guardado o
un JSON aportado por el ejecutor no satisfacen ninguna de las dos ultimas
condiciones. Tampoco pueden construir ni serializar genericamente la autoridad.

`PrepararRegistro` es la unica apertura prevista hacia el futuro adaptador SQL.
Repite en vivo las verificaciones criptografica y de referencia durable antes
de entregar copias defensivas de la pareja objeto/version y del material de
verificacion. La cancelacion durante cualquiera de ambos verificadores devuelve
una proyeccion cero y el mismo error opaco. Una revocacion, una sustitucion o
una cancelacion fallan cerradas.

## Frontera criptografica

El contrato V2 vigente admite HMAC-SHA-256 o COSE Sign1; la prueba focal recorre
ambas modalidades sin fijar HMAC en la capacidad. Una clave HMAC sigue
perteneciendo a la autoridad criptografica configurada: no se copia al valor,
no se entrega al adaptador SQL y no se almacena en PostgreSQL. La proyeccion
solo contiene referencia y version de clave, dominio, algoritmo, codigo y bytes
canonicos del recibo.

Las regresiones focales alteran las copias originales del recibo y del
vinculo/manifiesto despues de construir la autoridad, y alteran el material
sellado durante cada verificador vivo. Tambien conservan la cancelacion con
proyeccion cero en ambas fronteras y los recorridos HMAC y COSE_Sign1.

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
