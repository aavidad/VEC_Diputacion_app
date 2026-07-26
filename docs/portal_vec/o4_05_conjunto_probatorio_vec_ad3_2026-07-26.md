# O4-05: conjunto probatorio completo VEC-AD-3

Fecha: 26 de julio de 2026.

## Resultado

La consulta interna de Contratación temporal ya dispone de una frontera común
para transportar, sin reinterpretaciones parciales, todo el material que el
consumidor PostgreSQL VEC-AD-3 debe verificar y consumir.

Una única exportación fija estas diez entradas:

1. capacidad canónica;
2. decisión canónica;
3. motivo canónico;
4. contexto de actor canónico;
5. versión de persona;
6. versión de perfil;
7. carga útil VEC-AD-3;
8. sobre COSE Sign1;
9. evidencia de verificación;
10. raíz pública DER-SPKI Ed25519.

La misma instantánea incluye el resumen tipado y no autoritativo de la
capacidad. Así, Contratación temporal puede cotejar acción, recurso, efecto,
audiencia y ventana sin analizar el JSON privado ni recibir un segundo
exportador susceptible de mezclar una capacidad A con un resumen B.

## Frontera hexagonal

El contrato neutral vive en
`internal/vec/ports/material_consumo_autorizacion_atestada_v3.go`. Devuelve un
valor concreto con campos privados, huella interna, accesores deterministas y
copias defensivas. Un valor concreto evita tanto los nulos tipados como un
objeto hostil cuyos accesores cambien de resultado entre llamadas.
La huella SHA-256 validada del conjunto se expone únicamente para ligar
clonados, órdenes y recibos sin duplicar el canon en cada módulo consumidor.

El constructor estructural común:

- aplica los límites exactos aceptados por la función PostgreSQL;
- exige versiones representables exactamente en el perfil SQL;
- acepta únicamente una raíz DER-SPKI Ed25519 canónica de 44 bytes;
- bloquea JSON, XML, texto, binario, gob, CBOR y YAML;
- redacta `fmt` y `slog`;
- no concede autorización ni verifica por sí solo la criptografía.

La implementación productiva vive en
`internal/vec/adapters/seguridad/confianzaatestacion/material_consumo_v3.go`.
Solo se construye desde solicitud, decisión, motivo, contexto, atestación,
prueba, capacidad y raíz V3 nominales. Antes de exportar vuelve a ligar:

- decisión, motivo y contexto;
- versiones de persona y perfil;
- carga útil, COSE y evidencia;
- identificador, versión, huella y ventana de la raíz;
- configuración de confianza;
- operación, efecto, audiencia y ventanas de la capacidad;
- resumen tipado y bytes originales.

## Autoridad y consumo

La exportación común es transporte, no autoridad. PostgreSQL conserva la
obligación de verificar dentro de la misma transacción:

- canon y huellas;
- MAC de la capacidad;
- firma COSE;
- gobierno, rotación y revocación;
- audiencia y operación exactas;
- vigencia;
- consumo único;
- lectura de la proyección y registro durable del acceso.

La función común existente
`registrar_y_consumir_decision_v3_atestada` está perfilada exclusivamente para
confirmar un alta. No se ampliará con comodines. O4-05 deberá crear fachadas
nominales separadas para cuadro y detalle, con acciones y audiencias cerradas,
sin reducir las garantías de alta.

## Verificación

La revisión independiente obtuvo `GO` sin hallazgos pendientes. La matriz
adversaria cubre:

- cruces A/B de capacidad, atestación, prueba y raíz;
- mutación de las diez entradas, de las dos versiones y del resumen;
- raíz distinta, X25519 del mismo tamaño, RSA, DER adulterado y 44 bytes
  hostiles;
- límites inferiores y superiores del contrato SQL;
- mutación de buffers de entrada y de copias devueltas;
- serialización, deserialización, formato y registros;
- valores cero y transporte estructural no autoritativo.

Quedaron verdes las pruebas normales sin caché, la carrera focal, `go vet`,
`gofmt` y `git diff --check`. Los ficheros productivos quedan por debajo de
500 líneas y las pruebas por debajo de 800.

## Alcance pendiente

Este corte desbloquea la migración de `CapacidadConsultaRRHH`, pero no cierra
la proyección PostgreSQL, la identidad corporativa, la API, la raíz de
composición, la fuente web ni el E2E. Por ello Contratación temporal conserva
19 de 46 tareas verificadas y O4-05 mantiene tres de cinco hitos internos.
