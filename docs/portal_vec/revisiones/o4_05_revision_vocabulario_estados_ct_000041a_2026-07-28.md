# O4-05: revisión del vocabulario de estados CT-000041A

Fecha: 28 de julio de 2026

Ámbito: migración PostgreSQL `000041`, vocabulario técnico de publicación RRHH

Estado: `GO` técnico independiente; CT `000041` y producción siguen abiertas

## Alcance cerrado

Los commits `7ae4122`, `74230e6` y `676191b` incorporan:

- los seis estados exactos `pendiente`, `en_curso`, `espera_externa`,
  `completado`, `incidencia` y `cancelado`;
- rechazo de mayúsculas, espacios, acentos, sufijos y valores inventados;
- aceptación del valor vacío únicamente como filtro de consulta, nunca como
  estado persistido;
- barreras de esquema `20/4 → 21/5`;
- manifiesto inmutable del vocabulario;
- instalación, reentrada y reversión transaccionales;
- reversión bloqueada si existe una publicación con un estado nuevo.

La migración no normaliza silenciosamente. El valor almacenado y el filtro
deben coincidir exactamente con el catálogo gobernado.

## Hallazgos y correcciones

La primera revisión obtuvo `NO-GO` por tres defectos:

1. un disparador homónimo reducido a `UPDATE` podía engañar la comprobación
   basada solo en nombre y estado;
2. la reversión podía destruir objetos derivados, permisos y restricciones
   homónimas alteradas;
3. la prueba de normalización no ejercitaba realmente las variantes
   inválidas.

El corrector sustituyó las listas parciales por una huella canónica del
catálogo estructural. Incluye relación, propietario, RLS, permisos, opciones,
columnas, valores por defecto, restricciones, índices, disparadores y sus
funciones, políticas, reglas, herencia, publicaciones, estadísticas,
etiquetas, comentarios, tipo compuesto de fila y array, secuencias asociadas
y TOAST.

Una segunda revisión obtuvo `NO-GO` de aceptación porque tres ataques, aunque
detectados por la huella, no estaban fijados como regresiones exactas. El
commit `676191b` añadió:

- disparador homónimo del manifiesto limitado a `BEFORE UPDATE`;
- disparador heredado de publicación homónimo limitado a `BEFORE UPDATE`
  antes del `up`;
- política homónima `propietario_total` con comando, rol y expresiones
  alterados;
- permiso `USAGE` real sobre el tipo compuesto de fila;
- rechazo nativo de PostgreSQL al permiso directo sobre su tipo array.

Todos se rechazan sin efectos parciales y se restaura el estado exacto antes
de continuar la matriz.

## Reversión segura

El `down` usa bloqueo asesor y exclusivo, exige barreras exactas y compara las
huellas estructurales esperadas. Rechaza:

- columnas, índices, disparadores, políticas o permisos futuros;
- función de inmutabilidad sustituida por una versión inocua;
- restricción homónima cambiada por `CHECK (true)` o no validada;
- secuencia `OWNED BY` sin valor por defecto;
- comentario o permiso del tipo compuesto;
- opciones TOAST;
- dependencias futuras y barreras posteriores;
- cualquier fila publicada con `incidencia`, `cancelado`,
  `espera_externa` o los demás estados no admitidos por la versión anterior.

La retirada usa `RESTRICT`; nunca `CASCADE`.

## Puertas reproducidas

Dirección y revisión independiente ejecutaron la matriz completa con:

```text
postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296
```

Quedaron verdes:

- instalación, reentrada y ciclo `up → down → up`;
- ataques estructurales y semánticos;
- publicación con estado nuevo;
- reversión y rollback exactos;
- `bash -n`;
- ShellCheck;
- `git diff --check`;
- Gitleaks sobre los tres commits: 107,85 KiB, cero hallazgos.

Líneas finales:

| Fichero | Líneas |
| --- | ---: |
| migración `up` | 728 |
| migración `down` | 644 |
| prueba SQL | 290 |
| runner | 313 |
| auxiliar adversarial | 373 |

Todos permanecen por debajo de 800 líneas.

## Dictamen y siguiente corte

La reauditoría final emitió `GO` sin hallazgos pendientes. El vocabulario
CT-000041A queda cerrado y no debe reabrirse salvo defecto reproducible.

El contrato probatorio restante se divide, conforme al
[documento de coordinación](../coordinacion_ct_000041b_postgresql_recibo_rrhh_000042_000043_2026-07-28.md),
en:

- `000042`: cánones y huellas PostgreSQL;
- `000043`: prueba durable y cierre interno del Recibo V2;
- `000044`: futuro motor transaccional propietario;
- `000045`: futuras fachadas nominales.

Por tratarse de un cierre parcial de CT `000041`, se mantienen:

- Contratación temporal: `20/46`, 43 %;
- O4-05: `3/5`;
- Bolsa: `1/14`, 7 %;
- producción: `NO-GO`.
