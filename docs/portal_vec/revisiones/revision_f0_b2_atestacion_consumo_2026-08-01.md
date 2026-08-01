# Revisión F0-B2: atestación y consumo de fuente

Fecha: 1 de agosto de 2026.

## Resultado

F0-B2 obtuvo dos revisiones independientes finales `GO`, ambas con
`P0=P1=P2=0`. El commit productor `be0d03a` se integró en la rama estable
como `7519063`.

El corte añade exclusivamente los dos componentes
`060_atestacion_consumo.sql`. No publica una función de inserción, una
fachada, un rol ni una migración `000007` instalable.

## Huellas finales

```text
M060: 471efc13b7acd82b3f79150a0cad82e93d138418b8923c4e5f11ff54a9fe68fb
      137 líneas
T060: aea7b03000f0955c43b988f648e1845c211e5696403dd6b096dc22291e434e3c
      799 líneas
```

## Evidencia reproducida

El productor completó tres ejecuciones finales literales
`probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh --etapa B2` sobre
PostgreSQL 18.4 por digest. Ambos revisores repitieron la etapa y dirección
volvió a ejecutarla después de integrar. Todas terminaron con `ROLLBACK`,
línea base exacta y cero residuos atribuibles.

La matriz acredita:

- dos relaciones `PERMANENT` minimizadas, de cuatro y seis columnas;
- PK por capacidad, evento estable, nonce y operación únicos;
- consumo dependiente mediante PK/FK 1:1, `MATCH FULL`, no diferible y
  `NO ACTION`;
- canon de 512..32.768 octetos, huella SHA-256 derivada e instante UTC finito;
- rechazo de `UPDATE`, `DELETE` y `TRUNCATE`;
- RLS habilitada y forzada, política propietaria y ACL cerrada;
- tipos compuestos y arrays derivados ligados. PostgreSQL no permite
  conceder privilegios al array por separado: la prueba concede `USAGE` al
  compuesto y acredita la propagación efectiva al array;
- columnas, índices, restricciones, disparadores propios y cuatro
  disparadores RI exactos;
- dos árboles TOAST, dependencias, publicaciones, extensiones, vistas, FK
  entrantes, estadísticas y tablespaces adversos.

Las revisiones mataron mutantes externos de FK ausente, ligadura canon–huella
tautológica, validador temporal eliminado, ACL hostil compuesto→array,
publicación, pertenencia a extensión, vista y FK entrantes, estadística
extendida y traslado de heap/TOAST. Los mutantes internos cubren además RLS,
índices, disparadores, restricciones, atributos caídos, almacenamiento,
compresión y opciones TOAST.

La primera instrumentación confundió el ACL predeterminado del array con un
privilegio configurable. PostgreSQL 18.4 confirmó `0LP01` para un
`GRANT/REVOKE` directo sobre el array; la versión final inventaría los cuatro
tipos, exige `typacl` nula en los arrays y comprueba el privilegio efectivo
derivado. El test se reestructuró para conservar 799 líneas legibles, sin
compactación artificial.

`git diff --check`, límites y Gitleaks quedaron verdes. El rango publicado no
contiene fugas. La ejecución CI `30719044843` está en curso sobre `7519063`;
este informe no la declarará verde hasta que termine.

## Continuación

B2, junto con C1 y A4, alimentará C2. No cambia métricas ni habilita
producción. C1 permanece en corrección tras un `NO-GO` independiente por
cronología y cobertura adversarial.
