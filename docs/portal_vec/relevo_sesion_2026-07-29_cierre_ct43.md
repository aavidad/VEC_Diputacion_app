# Relevo de sesión: cierre de CT-000043

Fecha de corte: 29 de julio de 2026, zona horaria Europe/Madrid.

## Directorio y rama vigentes

La única rama integradora es:

```text
rama: integracion/ct-o4-04e-20260726
worktree: .worktrees/ct-stable-docs
```

No programar en el directorio raíz histórico. No añadir, modificar ni eliminar
el Word de RRHH sin seguimiento. No borrar `/tmp/.git` ni tocar producción o
worktrees ajenos.

## Estado confirmado

Se cerraron dos dependencias consecutivas del camino crítico:

| Capacidad | Commit productor | Commit integrado | Revisión |
| --- | --- | --- | --- |
| Prueba causal VEC-AD-3 `000006` | `a6a57c0` | `b879f63` | doble `GO` |
| Prueba durable CT `000043` | `4d38e9c` | `a1a09b9` | doble `GO` |

El commit VEC-AD-3 quedó publicado con la ejecución GitHub
`30440628360` completamente verde. El commit funcional CT-000043 se publicó y
disparó la ejecución `30447949053`; comprobar su conclusión y la ejecución
posterior del commit documental antes de considerar cerrado el relevo remoto.

No se desplegó en producción y no se usaron datos personales reales.

## VEC-AD-3 `000006`

La autoridad VEC serializa la producción de la prueba con las tres clases de
revocación mediante el checkpoint de gobierno:

- si la prueba obtiene primero el lock, la revocación espera al `COMMIT`;
- si la revocación confirma primero, la prueba con snapshot obsoleto falla con
  `40001`;
- el rollback de la revocación no deja un falso rechazo;
- `55P03` o `40P01` no se aceptan como sustitutos del orden causal.

Las dos funciones nominales devuelven exactamente ocho valores autoritativos:

```text
decision_ref
efecto_ref
huella_efecto_sha256
consumo_huella_sha256
auditoria_ref
auditoria_huella_sha256
consumida_en
revalidada_en
```

El runner PostgreSQL 18.4, la regresión histórica, Go, carrera, ShellCheck,
Gitleaks, tamaños y dos revisiones independientes quedaron verdes.

## CT-000043

CT-000043 instala:

- tipos privados de entrada y salida;
- contenido tipado de cuadro y detalle;
- tabla de prueba y Recibo RRHH V2 de solo anexado;
- relaciones compuestas con acceso, identidad, alcance y prueba VEC;
- primitiva privada de cierre;
- guardias de retirada para las columnas añadidas al registro de acceso;
- RLS forzada, ACL mínima y catálogo semántico;
- avance atómico de barreras `22/6 → 23/7`.

El cierre recalcula dentro de PostgreSQL el contenido, resultado, conjunto
material y los 38 campos del Recibo V2. Compara los ocho valores autoritativos
de VEC-AD-3 y mantiene ligadas las diez piezas originales al mismo consumo.
No expone fachada ni concede `EXECUTE` al consultor RRHH.

La huella semántica acreditada es:

```text
e8a4cbadc41fb73d4381dff9b8aa20a19093ce53a97058af39312957906473a3
```

La revisión detallada y el inventario están en:

```text
docs/portal_vec/revisiones/
o4_05_revision_ct_000043_prueba_durable_recibo_rrhh_2026-07-29.md
```

### Evidencia

El runner:

```bash
./deploy/postgresql/contratacion_temporal/\
probar_o4_05_prueba_resultado_recibo_rrhh_pg18_4.sh
```

terminó verde en dos ciclos consecutivos del productor y en las dos
reproducciones independientes. Incluye:

- CT-000039 a CT-000043 y AUT-000006;
- omisión y alteración de los seis componentes;
- `UP`, reentrada, `DOWN`, reentrada y segundo `UP`;
- cuadro, detalle, replay y rollback;
- `40001`, `40P01`, `55P03` y `57014`;
- concurrencia con un único ganador;
- claves foráneas cruzadas e inmutabilidad;
- revocación viva;
- retirada segura frente a ACL, metadatos, índices, restricciones,
  estadísticas, publicaciones, herencia y dependencias;
- evidencia durable que impide retirar trazabilidad.

Las puertas Go normales y con carrera, `go vet`, compilación, grafos,
manifiestos, TLS, `govulncheck`, Bash, ShellCheck, tamaño y Gitleaks quedaron
verdes. Para Go se usa:

```bash
TMPDIR="$HOME/.cache" go test ./...
```

El marcador externo `/tmp/.git` es preexistente y no debe borrarse.

## Métricas oficiales

| Ámbito | Estado |
| --- | --- |
| Contratación temporal | `21/46`, 46 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

O4-05 no incrementa todavía su contador porque CT-000043 es una primitiva
privada. Falta la vertical productiva completa.

## Continuación exacta

El siguiente corte es CT-000044, motor privado y atómico de consultas RRHH:

1. crear tipos privados de salida de cuadro y detalle;
2. consumir una capacidad VEC nueva y revalidar identidad;
3. materializar una sola colección para respuesta y canon;
4. implementar detalle actual o por versión;
5. implementar cuadro *as-of*, filtros posteriores y orden estable;
6. bloquear, consumir y emitir cursores sin persistir el token;
7. llamar a CT-000043 dentro de la misma transacción;
8. conservar `40001`, `40P01`, `55P03` y `57014`;
9. avanzar barreras `23/7 → 24/8`;
10. crear runner PostgreSQL 18.4 y obtener doble `GO`.

Conjunto de archivos previsto:

```text
deploy/postgresql/contratacion_temporal/
  migraciones/000044_motor_consultas_rrhh.up.sql
  migraciones/000044_motor_consultas_rrhh.down.sql
  migraciones/000044_componentes/
  pruebas_sql/o405_motor_consultas_rrhh.sql
  probar_o4_05_motor_consultas_rrhh_pg18_4.sh
```

No modificar CT-000043 durante CT-000044. Después siguen:

```text
CT-000045 fachadas autorizadas
→ adaptador Go/PostgreSQL
→ composición raíz, identidad/PDP y TLS
→ E2E navegador/API/aplicación/PostgreSQL/Recibo RRHH
→ E2E equivalente desde transporte no web
```

## Reglas que permanecen

- castellano coherente e i18n; sólo vocabulario técnico normalizado;
- arquitectura hexagonal y puertos intercambiables;
- denegación por defecto;
- cero cookies y cero almacenamiento web como autoridad;
- web, escritorio, CLI y MCP comparten casos de uso;
- trazabilidad completa y datos minimizados;
- PostgreSQL real, concurrencia y retirada segura;
- productor distinto de revisor e integrador;
- commits pequeños, documentación simultánea y Gitleaks antes de publicar;
- producción permanece bloqueada hasta cerrar código, E2E, TLS y
  conformidades formales.
