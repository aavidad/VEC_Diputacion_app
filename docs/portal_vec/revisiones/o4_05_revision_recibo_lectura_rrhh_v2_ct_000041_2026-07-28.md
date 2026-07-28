# O4-05: revisión del Recibo de lectura RRHH V2

Fecha: 28 de julio de 2026

Ámbito: parte Go del contrato probatorio CT `000041`

Estado: `GO` técnico independiente; CT `000041` y producción siguen abiertas

## Alcance verificado

Los commits `0109f48`, `b52e69d` y `0f7d979` incorporan:

- un Recibo de lectura RRHH V2 nominal y opaco;
- vínculo del recibo con el contexto, la capacidad, la orden, el contenido, el
  resultado, el cursor y los hitos temporales exactos;
- discriminador de esquema exacto
  `vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2`;
- validación diferenciada entre conservación histórica y ejecución interna;
- rechazo de V1 en la vía de ejecución interna, sin invalidar su lectura
  histórica;
- límite estricto de 256 KiB, copias defensivas y limpieza de material
  temporal;
- serialización y registro accidental bloqueados para el material probatorio.

El esquema forma parte del canon sellado antes de los siete valores probatorios.
No se acepta vacío, V1, variantes de mayúsculas ni un esquema de otra
operación.

## Hallazgos y corrección

La primera revisión independiente obtuvo `NO-GO` por dos defectos:

1. el detalle podía aceptar en ejecución interna un resultado generado antes
   de la orden si se registraba después;
2. el constructor no exigía el discriminador exacto que devuelve la migración
   `000039`.

El corrector `0f7d979` cerró ambos casos. La prueba adversarial conserva la
capacidad en `T0`, sitúa la orden en `T0+1`, genera el resultado en `T0` y lo
registra en `T0+2`: la validación histórica lo admite, pero la ejecución
interna V2 lo rechaza. El borde exacto, generado en el mismo instante de la
orden, se acepta. Cuadro y detalle aplican la misma frontera temporal.

## Puertas reproducidas

La revisión independiente y la integración reprodujeron:

- pruebas adversariales de antigüedad y esquema repetidas veinte veces;
- `go test -count=1 ./internal/modules/contrataciontemporal/...`;
- `go test -count=1 -race` para `ports` y `domain`;
- `go vet ./internal/modules/contrataciontemporal/...`;
- `git diff --check`;
- Gitleaks sobre los tres commits: 43,91 KiB, cero hallazgos.

Los cuatro ficheros del corte permanecen por debajo de 800 líneas.

## Dictamen y límites

El Recibo de lectura RRHH V2 obtiene `GO` independiente sobre el corrector
final. Este cierre no autoriza a construir el recibo desde el cliente ni a
atribuirle autoridad propia: deberá producirlo el futuro motor transaccional
interno a partir del resultado recalculado por PostgreSQL.

CT `000041` continúa abierta hasta cerrar:

- el vocabulario PostgreSQL de seis estados y su reversión estructural exacta;
- el contrato PostgreSQL que recalcule y selle contenido, resultado y recibo
  en la misma transacción;
- el cruce real Go y PostgreSQL del recibo completo.

Por tanto se mantienen:

- Contratación temporal: `20/46`, 43 %;
- O4-05: `3/5`;
- Bolsa: `1/14`, 7 %;
- producción: `NO-GO`.

## Cierre posterior del vocabulario

El vocabulario PostgreSQL de seis estados se corrigió después de este
dictamen y obtuvo `GO` independiente. La evidencia vigente está en
[la revisión de CT-000041A](o4_05_revision_vocabulario_estados_ct_000041a_2026-07-28.md).
CT `000041` permanece abierta únicamente por el contrato PostgreSQL de
contenido, resultado y recibo y su cruce completo con Go.
