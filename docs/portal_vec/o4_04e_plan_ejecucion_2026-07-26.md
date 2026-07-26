# Plan de ejecución O4-04E: confirmación durable de cobertura

**Fecha:** 26 de julio de 2026  
**Estado:** en ejecución  
**Rama de integración:** `integracion/ct-o4-04e-20260726`

## Propósito

Cerrar la confirmación de una decisión de cobertura de Contratación temporal
como una única operación durable, atómica y reconciliable. El resultado debe
probar la decisión VEC, el gobierno aplicado, los consumos C1, la transición
del expediente, la decisión C2, la auditoría, el evento de salida y el recibo
terminal.

O4-04E no incluye la API ni la pantalla. Esas superficies pertenecen a O4-05
y solo se conectarán cuando la transacción crítica haya superado sus puertas
de PostgreSQL 18 y revisión independiente.

## Línea base integrada

La línea completa de Contratación temporal se ha fusionado sobre la principal
actual, conservando los avances de Bolsa, seguridad y composición. La
convergencia pasa:

- `go test ./...`;
- `go vet ./...`;
- comprobación de diferencias sin errores de formato.

La fusión no aumenta por sí sola el porcentaje funcional: mantiene
Contratación temporal en 18 de 46 tareas hasta que E sea productiva y revisada.

## Hallazgos previos obligatorios

Antes del adaptador PostgreSQL deben cerrarse tres huecos:

1. El fragmento C1 debe transportar, mediante un tipo nominal y copias
   defensivas, las siete pruebas canónicas que exige la persistencia.
2. La reconciliación debe empujar coordenadas y huella de orden a una sesión
   primaria opaca. No se añadirá un captador público de la huella.
3. Desde el instante en que se intenta confirmar en PostgreSQL, cualquier
   error sin recibo válido es ambiguo y obliga a reconciliar. Nunca autoriza un
   reintento ciego.

La relación terminal existente impone este orden final:

`recibo → versión terminal → marcador → puntero de reserva`.

El puntero de reserva será la última escritura funcional.

## Cortes paralelos

| Corte | Alcance | Estado |
| --- | --- | --- |
| E-GO-1 | Contratos Go: pruebas C1, lectura push opaca y ambigüedad | Cerrado y revisado |
| E-SQL-1 | Enlace probatorio VEC, cánones y tablas durables | Cerrado, revisado y subido |
| E-GO-2 | Ejecutor PostgreSQL y reconciliador primario | Cerrado, revisado y subido |
| E-SQL-2 | Función exterior atómica, ramas concedida/denegada y lector fuerte | En curso; pendiente de E-SQL-1 para probar |
| E-PG18 | Integración, replay, concurrencia, fallos y ACL/RLS | Pendiente |
| E-REV | Revisión independiente y cierre documental | Pendiente |

Los cortes modifican superficies distintas. No se aceptará un corte porque
compile de forma aislada: todos se revisarán sobre esta misma rama integrada.

## Reglas de la función exterior

- La transacción de escritura será `SERIALIZABLE` y no tendrá reintentos
  automáticos.
- La función recibirá una carga cerrada y no aceptará un resultado VEC
  construido por Go.
- La decisión VEC se registrará mediante el envoltorio privado existente y en
  la misma transacción.
- La concesión revalidará el gobierno, consumirá C1, aplicará CAS al agregado
  y persistirá C2 antes de cerrar la prueba terminal.
- La denegación no podrá crear consumos C1, versión positiva del agregado,
  actuación ni decisión C2.
- Un replay exacto devolverá la prueba terminal existente incluso si
  posteriormente han caducado vigencias.
- Una respuesta SQL o de `COMMIT` perdida se resolverá únicamente mediante
  lectura primaria `SERIALIZABLE READ ONLY`.
- El rol de ejecución solo podrá invocar la función exterior y el lector
  fuerte. Las primitivas B, C y D continuarán privadas.

## Puertas de aceptación

1. Vectores cruzados Go/SQL para todos los materiales canónicos.
2. Concesión y denegación reales en PostgreSQL 18.
3. Replay exacto y rechazo de una colisión mutada.
4. Límites C1 de 1 y 512 aceptados; 0 y 513 rechazados.
5. Carreras de reserva, agregado, C1, gobierno y decisión VEC.
6. Reversión total tras fallos inyectados después de cada escritura.
7. Reconciliación tras pérdida de respuesta SQL y de `COMMIT`.
8. Reinicio de PostgreSQL y replay posterior.
9. ACL, RLS, `FORCE RLS`, propietario `NOLOGIN` y mínimo privilegio.
10. `go test -race`, `go vet`, manifiestos, límites de tamaño y revisión
    independiente sin hallazgos bloqueantes.

## Paso siguiente

Al cerrar O4-04E comenzará O4-05:

`navegador RRHH → API interna → caso de uso → función O4-04E → recibo`.

La interfaz conservará la pantalla administrativa existente y sustituirá el
adaptador de presentación por el puerto productivo, sin introducir estado de
sesión web ni acoplar la vista a PostgreSQL.
