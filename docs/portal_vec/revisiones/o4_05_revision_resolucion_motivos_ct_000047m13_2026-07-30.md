# Revisión final de resolución nominal de motivos CT-000047M1.3

Fecha: 30 de julio de 2026.

## Resultado

**GO técnico independiente**.

| Elemento | Valor |
| --- | --- |
| Base estable previa | `e0e3d49` |
| Commit productor | `d196b23` |
| Commit integrado | `281f52b` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Garantías verificadas

La migración `000010` expone exclusivamente dos resoluciones nominales:

```text
resolver_motivo_cuadro_rrhh_v1(timestamptz)
resolver_motivo_detalle_rrhh_v1(timestamptz)
```

No admite selectores de catálogo, entrada, organización, acción, finalidad ni
clase. Cada fachada deriva el actor de `session_user`, exige el `LOGIN`
resolutor con topología exacta, conserva el orden de bloqueos
`000008 → 000009 → 000010` y devuelve cero o una referencia V2 completa.

Quedaron acreditados:

- vigencia histórica y actual del catálogo y de su entrada;
- retirada local y retirada global V2;
- motivos distintos para cuadro y detalle;
- referencias ausentes, futuras, expiradas o temporalmente inválidas;
- RLS forzada, políticas, restricciones, disparadores, funciones y ACL
  fundamentales;
- denegación a `PUBLIC`, roles ajenos y membresías o otorgantes distintos;
- ausencia de lectura o DML directo para el resolutor;
- `search_path` hostil, homónimos y sobrecargas;
- carreras causales con las retiradas nominal y V2;
- reversión segura, atómica y sin `CASCADE`;
- conservación íntegra de evidencia, tablas y funciones fundamentales.

## Hallazgo y corrección

La primera matriz completa era verde, pero una contrarrevisión detectó que el
runner no ejercitaba de forma explícita la deriva de RLS y política exigida por
la coordinación. El SQL ya la rechazaba.

El candidato final añade dos venenos independientes:

1. deshabilitar RLS;
2. sustituir la política del propietario por una política permisiva.

En ambos casos la instalación debe fallar y dejar cero fachadas. Después se
restaura el fundamento exacto y el manifiesto estricto permite continuar. El
primer dictamen favorable anterior quedó retirado y ambos revisores evaluaron
de nuevo el hash corregido.

## Evidencia reproducible

| Fichero | SHA-256 | Líneas |
| --- | --- | ---: |
| `000010_resolucion_vinculaciones_motivo_consultas_rrhh.up.sql` | `91fd4a53108bba45f9ca28f8ee28e9aabd431d7fc3197ae04a9b281acb973e17` | 789 |
| `000010_resolucion_vinculaciones_motivo_consultas_rrhh.down.sql` | `e58c0b09d81b8ff6505fab48261c233e916742d868fbca9f2d6dfa5fcd0b7a47` | 472 |
| `probar_resolucion_vinculaciones_motivo_consultas_rrhh_000010_pg18_4.sh` | `a70082b00b4ed327b85624216a1492a2890026723487e685834260039f55f4de` | 476 |

- productor: PostgreSQL 18.4, tres ejecuciones completas verdes;
- primer revisor: PostgreSQL 18.4, tres ejecuciones completas verdes sobre el
  candidato final;
- segundo revisor: revisión estática independiente del hash final;
- dirección: nueva ejecución completa verde tras integrar en la rama estable;
- `bash -n`, ShellCheck, `git diff/show --check` y límites: verdes;
- Gitleaks: cinco commits del lote y cero hallazgos;
- ningún contenedor temporal quedó activo.

## Límites conservados

M1.3 no implementa el pool ni el adaptador Go M2, la autoridad/PDP
corporativa, la composición raíz, TLS/mTLS viva ni el E2E HTTP/web. Tampoco
publica motivos reales ni sustituye las conformidades de RRHH, DPD, Jurídico,
Sistemas y DBA. Producción permanece en `NO-GO`.
