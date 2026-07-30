# Revisión final de publicación y retirada de motivos CT-000047M1.2

Fecha: 30 de julio de 2026.

## Resultado

**GO técnico independiente**.

| Elemento | Valor |
| --- | --- |
| Base estable | `71e961a` |
| Candidato final | `1e965af` |
| Commits integrados | `7d7f16c`, `a04c5a2`, `1e1bd51`, `0a564e2` |
| P0 | 0 |
| P1 | 0 |
| P2 | 3 no bloqueantes |

La integración conserva cuatro commits pequeños: capacidad inicial,
validación exacta de fundamentos, ampliación del arnés y orden contractual de
bloqueos.

## Garantías verificadas

La migración `000009` publica o retira de forma nominal las vinculaciones de
motivo de cuadro y detalle RRHH. No resuelve motivos ni expone un selector
libre.

Quedaron acreditados:

- historial de solo adición y checkpoint por clase de consulta;
- cuatro fachadas nominales para publicar o retirar cuadro y detalle;
- actor derivado de `session_user`, sin identidad aportada por parámetros;
- replay independiente del actor y cronología estricta;
- exclusión temporal entre una misma entrada V2 usada por cuadro y detalle;
- prueba de catálogo y entrada V2 derivada de PostgreSQL y comprobada en el
  instante del evento y en el actual;
- retirada local posible aunque el catálogo V2 ya se haya retirado;
- RLS forzada, propietario único, ACL mínima y denegación predeterminada;
- bloqueo ordenado `000008 → 000009`;
- retirada segura sin `CASCADE`, conservando cualquier evidencia.

## Hallazgos y correcciones

El primer revisor reprodujo que una clave foránea fundamental recompuesta con
`ON DELETE CASCADE` conservaba nombre y columnas y era aceptada. La corrección
compara bidireccionalmente diez restricciones, sus definiciones y los
veintiocho disparadores internos de integridad referencial.

Una segunda contrarrevisión detectó que dos bloqueos advisory se habían
compactado en una sola lista `SELECT`. PostgreSQL no garantiza el orden de
evaluación de sus expresiones. `0a564e2` los separa en dos sentencias y
restablece contractualmente `000008 → 000009`.

El arnés también comprueba de forma aislada privilegios predeterminados de
tabla y función, la semántica real del tipo fila implícito de PostgreSQL 18,
ACL hostiles de tabla, columna, función y tipo, huella de las ocho funciones,
barrera causal y una dependencia SQL real que obliga a `DROP RESTRICT`.

## Evidencia reproducida

- productor: PostgreSQL 18.4, tres ejecuciones finales verdes;
- primer revisor: PostgreSQL 18.4, tres ejecuciones finales verdes y caso
  adicional de política `000008` degradada;
- segunda contrarrevisión: cuatro ejecuciones del candidato anterior y
  verificación exacta del corrector final;
- dirección: una ejecución del candidato anterior y otra del árbol integrado;
- `bash -n`, ShellCheck, `git diff/show --check` y límites: verdes;
- Gitleaks: cuatro commits y cero hallazgos;
- ningún contenedor temporal quedó activo.

## P2 no bloqueantes

- El veneno dinámico ejercita una clave foránea representativa, no cada una de
  las diez restricciones.
- La carrera acredita colisión opaca `23505`, pero no fuerza una rama interna
  concreta del manejador `unique_violation`.
- Puede añadirse una prueba viva específica con un `LOGIN` sin pertenencia al
  rol proyector; la topología y las ACL ya se verifican estructuralmente.

## Límites conservados

M1.2 no implementa resolución `000010`, adaptador Go M2, composición raíz,
TLS/mTLS ni E2E. Tampoco autoriza catálogos reales ni sustituye las
conformidades de RRHH, DPD, Jurídico, Sistemas y DBA.
