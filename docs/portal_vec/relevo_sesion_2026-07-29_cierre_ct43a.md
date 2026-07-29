# Relevo de sesión: cierre técnico de CT-000043A

Fecha de corte: 29 de julio de 2026, zona horaria Europe/Madrid.

## Rama vigente

```text
rama: integracion/ct-o4-04e-20260726
worktree: .worktrees/ct-stable-docs
HEAD funcional local: 35cba6afc76fa444e6af8ec521570fbbfbf7e4fa
```

No programar en el directorio raíz histórico, no modificar el Word de RRHH sin
seguimiento, no borrar `/tmp/.git` y no tocar producción.

## CT-000043A

El corrector resuelve una incompatibilidad descubierta antes de CT-000044:
Go y el contrato funcional permiten `version_observada=0` para la primera
carga, pero CT-000043 exigía igualdad literal con la versión positiva.

La solución:

- conserva el cero en el canon y en la capacidad VEC;
- exige una versión materializada positiva;
- registra esa versión en acceso, prueba y Recibo;
- mantiene `N = N` y rechaza `N ≠ actual`;
- no avanza las barreras `23/7`;
- no crea tablas, permisos, fachadas ni otra autoridad;
- restaura byte a byte la función anterior en su `DOWN`.

Commits:

| Papel | Commit |
| --- | --- |
| Productor | `d2ba87e200a9f104c412e50ced98c55983afcec2` |
| Integrado | `35cba6afc76fa444e6af8ec521570fbbfbf7e4fa` |

El productor ejecutó dos ciclos completos. Dos revisores independientes
repitieron la puerta en PostgreSQL 18.4 y emitieron `GO`; dirección la volvió
a ejecutar sobre el commit integrado. La evidencia está en:

```text
docs/portal_vec/revisiones/
o4_05_revision_ct_000043a_detalle_version_actual_2026-07-29.md
```

## Estado remoto anterior

Las ejecuciones GitHub de CT-000043 quedaron completamente verdes:

```text
30447949053
30448509464
```

CT-000043A está integrado localmente en el momento de redactar este relevo.
Antes de declararlo publicado se debe confirmar este corte documental, enviar
la rama y comprobar la nueva ejecución GitHub.

## CT-000044 en curso

Worktree:

```text
.worktrees/ct44-motor-consultas-20260729
```

Estado sin confirmar:

- `030_materializacion_detalle.sql`: prueba PostgreSQL 18.4 y `GO`
  independiente;
- `040_materializacion_cuadro.sql`: funcional, admite el corte cero solo en la
  primera página vacía, conserva el límite de inactividad y tiene `GO`
  independiente local;
- `010` y `020`: base privada válida, pendiente de las correcciones del motor;
- `050`: debe incorporar un punto de control causal por familia y no usar una
  única fila global que serialice familias independientes;
- principales `UP/DOWN`, motor exterior, runner integral y barreras `24/8`:
  pendientes.

Orden obligatorio del motor:

```text
validar forma
→ consumir VEC
→ revalidar Identidad
→ bloquear el punto de control causal de la familia
→ releer familia, cursor, revocación y consumo
→ fijar corte
→ materializar una sola colección
→ cerrar prueba CT-000043A
→ persistir efectos del cursor
→ COMMIT exterior único
```

La carrera de revocación debe acreditar revocación primero, motor primero,
rollback y dos familias sin cerrojo local común. El token en claro solo puede
existir como variable y salida transitoria.

## Métricas oficiales

| Ámbito | Estado |
| --- | --- |
| Contratación temporal | `21/46`, 46 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

CT-000043A corrige una primitiva privada y no incrementa el contador. Tampoco
se contabilizan los componentes locales de CT-000044 hasta cerrar el recorrido.

## Continuación

1. confirmar y publicar este corte documental;
2. comprobar la ejecución GitHub de CT-000043A;
3. integrar el detalle y el cuadro ya revisados en el motor CT-000044;
4. implementar y revisar el control causal, el cursor y el motor privado
   CT-000044;
5. crear `UP/DOWN`, runner PostgreSQL 18.4 y obtener doble `GO`;
6. continuar con CT-000045, adaptador Go, composición raíz, TLS y E2E;
7. mantener castellano, i18n, arquitectura hexagonal, denegación
   predeterminada, cero cookies y trazabilidad completa.
