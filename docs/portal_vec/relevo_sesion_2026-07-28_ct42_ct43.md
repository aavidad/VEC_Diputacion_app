# Relevo de sesión: cierre CT-000042 e inicio CT-000043

Fecha de corte: 28 de julio de 2026, zona horaria Europe/Madrid.

## Actualización vigente de dirección — 31 de julio de 2026

El último corte publicado completamente verde es `fc923d9` en
`integracion/ct-o4-04e-20260726`, con la ejecución GitHub `30626845298`.
C2.1b, S0.1 y S0.2 están cerradas. C2.2-A superó A0–A4 y la revisión final
en `6017606`, P0=P1=P2=0; A5 todavía debe publicarla y obtener CI verde.
El trabajo activo es A5, publicación y CI;
C2.2-B sigue bloqueada hasta ese cierre. Las métricas no se inflan por este
trabajo compartido: Contratación `24/46` (52 %), O4-05 `3/5`, Bolsa
productiva `1/14` (7 %) y producción `NO-GO`.

Esta actualización prevalece sobre los cortes históricos del resto del
documento. La dirección detallada vigente está en el relevo de Contratación y
en la decisión C2.2.

## Actualización vigente de dirección — 30 de julio de 2026

La rama y el worktree siguen siendo los indicados, pero el corte vigente ya no
es CT-000043. CT-000047C1 y CT-000047C2.1a están integradas. El último commit
técnico verificado es `e8c950c`, el cierre documental de C2.1a es `808522d` y
el corte publicado usado como base es `cc6041e`.

C2.1a obtuvo `GO` funcional y de seguridad final, `P0=P1=P2=0`, después de
cerrar la autoridad implícita de `PUBLIC` sobre tipos y distinguir por catálogo
los arrays automáticos de los tipos base reales. El siguiente trabajo es
C2.1b, fachada mínima de Identidad; después siguen la selección/registro
corporativos, PDP y composición raíz, TLS/mTLS y el E2E completo. Las
métricas oficiales no cambian todavía porque no se ha cerrado una capacidad
funcional completa: Contratación `24/46` (52 %), primera vertical `5/10`
(50 %), O4-05 `3/5`, Bolsa productiva `1/14` (7 %) y producción `NO-GO`.

La ejecución GitHub `30527303065`, correspondiente al cierre publicado de
C2.1a, terminó completamente verde. C2.1b conserva el borrador `e8883df`,
todavía sin runner, `GO` ni integración; su dependencia ACL está en una rama
candidata pendiente de revisión. El cuerpo que sigue conserva el estado
histórico del 28 de julio y no debe usarse como orden vigente; prevalecen el
relevo del 29 de julio y el mapa actual.

## Resumen ejecutivo

El trabajo confirmado y publicado está en:

```text
rama: integracion/ct-o4-04e-20260726
worktree: /home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-stable-docs
HEAD local/remoto: e8d2afcca8aac5a3aaf32f4a8aef5da06c8f5fb5
```

Dos commits nuevos quedaron publicados:

| Commit | Contenido |
| --- | --- |
| `0a3d72a` | CT-000042: cánones PostgreSQL de resultados y Recibo RRHH V2, vectores Go/SQL, runner y pruebas adversariales. |
| `e8d2afc` | Espera verificable del servidor PostgreSQL definitivo en la puerta de ContextoActor V3. |

Gitleaks no encontró secretos. El autor y confirmador de ambos commits es
`aavidad <avidad@dipgra.es>`.

La ejecución GitHub anterior, `30396784705`, terminó completamente verde. La
ejecución del corte actual es `30399280237`; al congelar la sesión tenía cuatro
puertas verdes y `puerta-calidad` aún en ejecución. El siguiente agente debe
consultar su conclusión antes de integrar otro corte:

```bash
gh run view 30399280237 --json status,conclusion,url,jobs
```

## CT-000042: cerrado

CT-000042 obtuvo `GO` independiente final. Se acreditaron:

- PostgreSQL 18.4 fijado por digest;
- instalación atómica por seis componentes;
- 21 vectores materiales y cánones Go/SQL byte a byte;
- 38 omisiones y 38 sustituciones del Recibo V2;
- cuadro sin cursor y con cursor, detalle y costes;
- rechazo de nulos parciales, arrays hostiles, importes y monedas inválidos;
- RFC3339Nano y DER SPKI Ed25519 canónicos;
- ACL, propietario, `search_path`, catálogo y huella estructural;
- homónimos, sobrecargas, cuerpo, propietario, `proconfig` y dependencia;
- base clonada con OID coincidente;
- barrera futura, reentrada y `up → down → up`;
- Go focal y global, `go vet`, Bash, ShellCheck, formato, tamaños y Gitleaks.

La rama productora local
`agent/ct42-sql-postgresql-20260728` conserva el commit `ed8a4cd`, equivalente
al `0a3d72a` integrado. No hace falta volver a integrarlo ni publicarlo.

## CT-000043: WIP congelado, no revisado

Se creó desde `e8d2afc`:

```text
rama: agent/ct43-prueba-durable-20260728
worktree: /home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct43-prueba-durable-20260728
```

Al congelar la sesión hay cinco archivos sin seguimiento y ningún commit:

```text
deploy/postgresql/contratacion_temporal/migraciones/
  000043_prueba_resultado_recibo_rrhh.up.sql
  000043_componentes/010_tipos_cierre.sql
  000043_componentes/020_relaciones_y_prueba.sql
  000043_componentes/030_primitiva_cierre.sql
  000043_componentes/090_acl_catalogo_y_barrera.sql
```

Estos archivos son borradores parciales. No tienen `down`, runner final,
matriz adversarial completa, verificación independiente ni autorización para
confirmarse. No asumir que el SQL es correcto por estar escrito.

El contrato fijado para CT-000043 es:

- migración de barreras `22/6` a `23/7`;
- tabla append-only `prueba_resultado_recibo_rrhh_v2`;
- claves foráneas compuestas exactas al registro, vínculo y alcance;
- RLS forzada y ACL cerrada;
- rechazo de `UPDATE`, `DELETE` y `TRUNCATE`;
- primitiva interna `SECURITY DEFINER`;
- derivación de cánones, huellas y recibo desde fuentes persistidas y material
  original;
- safe-down semántico que nunca borra evidencia para facilitar la reversión.

La fuente normativa técnica obligatoria es:

`docs/portal_vec/coordinacion_ct_000041b_postgresql_recibo_rrhh_000042_000043_2026-07-28.md`.

## Continuación exacta

1. Comprobar el resultado final de GitHub `30399280237`. Si falla, leer el log
   exacto antes de editar o volver a publicar.
2. Entrar en el worktree CT-000043 y leer completamente la coordinación.
3. Auditar los cinco borradores contra CT-000039 a CT-000042 y los contratos
   VEC-AD-3. Corregir, no completar por inercia.
4. Implementar el `down` no destructivo y dividir componentes si cualquier
   archivo supera 800 líneas.
5. Crear runner PostgreSQL 18.4 por digest con instalación, reentrada,
   `up → down → up`, ACL/RLS, DML hostil, FKs cruzadas, rollback, carreras,
   revocación, evidencia que bloquea `down`, catálogo y safe-down.
6. Completar la matriz adversarial íntegra definida en la coordinación.
7. Ejecutar puertas PostgreSQL, Go focal/global, carrera aplicable, `go vet`,
   Bash, ShellCheck, formato, `git diff --check`, tamaños y Gitleaks.
8. Obtener revisión independiente del árbol estable. No confirmar con un
   `NO-GO`.
9. Tras `GO`, confirmar localmente, integrar mediante `cherry-pick` en
   `integracion/ct-o4-04e-20260726`, actualizar tablero/relevo y hacer un único
   envío consolidado.
10. Continuar en orden con CT-000044, CT-000045, adaptador PostgreSQL Go,
    composición raíz, matriz TLS viva y E2E HTTP.

## Métricas que no deben inflarse

El corte oficial permanece:

| Ámbito | Estado |
| --- | --- |
| Contratación temporal | `20/46`, 43 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

CT-000042 es un cierre probatorio interno de O4-05; no constituye por sí solo
una nueva capacidad funcional E2E. La métrica solo aumenta al cerrar una pieza
completa del recorrido.

## Brecha restante

Camino crítico inmediato de Contratación:

```text
CT-000043 durable
→ CT-000044 motor interno
→ CT-000045 fachadas autorizadas
→ adaptador Go/PostgreSQL
→ composición raíz + identidad/PDP + TLS
→ navegador/API/aplicación/PostgreSQL/recibo E2E
```

Después siguen las tareas abiertas de alta E2E, formulario de análisis,
asignación e informes, fiscalización, llamamiento/formalización, incorporación
y GINPIX, seguimiento, cierre, conservación y expurgo.

Bolsa conserva como brechas principales el recorrido moderno de borradores,
panel interno, baremación, llamamientos, candidatura/alegaciones,
comunicaciones, documentos/firma y composición de identidad, auditoría y
almacenamiento.

## Estado de otros directorios

El directorio raíz está en la rama histórica `vec-orquesta-20260619`, commit
`14cbcad`. Contiene sin seguimiento el Word enviado por RRHH:

`Pantalla de procedimiento de gestión de contratación y gestión de bolsas.docx`

Es una fuente del usuario. No añadirla, modificarla ni eliminarla sin una
decisión expresa.

No se tocó el servidor de producción ni se realizó despliegue en esta sesión.
No deben eliminarse worktrees antiguos durante la continuación: antes hace
falta un inventario específico porque algunos conservan historia de trabajo.
