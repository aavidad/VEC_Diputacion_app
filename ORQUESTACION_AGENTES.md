# Orden de trabajo para agentes paralelos

**Proyecto:** VEC Diputación de Granada

**Rama de integración:** `vec-orquesta-20260619`

**Tablero de dirección:** [ESTADO_PROYECTO.md](ESTADO_PROYECTO.md)

**Prioridad absoluta:** terminar una Bolsa real, no acumular contratos aislados.

## Jerarquía de dirección

| Nivel | Responsabilidad | Puede integrar |
| --- | --- | --- |
| Director principal | Divide el producto, resuelve dependencias, asigna agentes, mantiene el tablero y decide qué entrega entra | Sí, después de revisión y pruebas. |
| Supervisores nativos | Tres áreas estables: seguridad/cumplimiento; persistencia/calidad; funcional/web. Inspeccionan a los programadores y reproducen sus pruebas | No por sí solos; emiten GO/NO-GO. |
| Programadores Codex de Orquesta | Un equipo por vertical, con `CODEX_HOME`, worktree, write-set y presupuesto propios; pueden usar subagentes dentro de su encargo | No; solo entregan commits o parches en su rama. |
| Revisor cruzado | Agente distinto del autor que busca defectos, fugas, deuda, falsos E2E y desviaciones de alcance | No; emite GO/NO-GO y pruebas. |

Cada programador de Orquesta queda asignado a un supervisor nativo. Un mismo
supervisor puede controlar varios equipos con write-sets disjuntos, pero no
programa en sus ramas. Ningún autor revisa o cierra su propia entrega.

## Instrucción corta para un agente nuevo

El operador debe asignar un identificador de trabajo de este documento. El
mensaje mínimo es:

```text
Lee AGENTS.md, ORQUESTACION_AGENTES.md y ESTADO_PROYECTO.md completos.
Toma exclusivamente el trabajo <IDENTIFICADOR>. Trabaja en un worktree propio,
cumple sus límites y entrega commit, pruebas, riesgos y revisión solicitada.
```

Un agente no elige «el siguiente trabajo libre» por su cuenta: dos procesos
arrancados a la vez podrían reclamarlo. Dirección u operador asignan un
identificador distinto a cada agente.

## Lectura obligatoria y autoridad

Antes de editar, por este orden:

1. `AGENTS.md` y las instrucciones de mayor prioridad de su entorno.
2. Este documento completo.
3. `ESTADO_PROYECTO.md`.
4. `docs/instruccion_direccion_2026-07-18.md`.
5. La entrada técnica y decisiones indicadas en la ficha del trabajo.

Si dos documentos se contradicen, manda el más reciente y específico. El
agente informa de la contradicción y no inventa un puente temporal.

## Forma segura de trabajar en paralelo

Los agentes manuales no deben usar el árbol principal, que puede contener
cambios sin commit. Cada uno usa una rama y un worktree exclusivos dentro de
la carpeta ignorada `.worktrees/` de este proyecto. Está prohibido crear
worktrees de VEC en `/tmp`, en `~/Trabajo/.worktrees` o dentro de otro
proyecto:

```bash
cd /home/usuario/Trabajo/VEC_Diputacion_app
git fetch --all --prune
git worktree add .worktrees/<IDENTIFICADOR> \
  -b agente/<IDENTIFICADOR> vec-orquesta-20260619
```

Antes de crearlo se comprueban `git worktree list` y `df -h`. No se crea un
segundo worktree para el mismo identificador. El agente no fusiona, no hace
`cherry-pick` sobre la rama principal y no cambia la visibilidad del repositorio.
Entrega uno o varios commits pequeños; dirección revisa e integra.

Después de integrar y solo tras comprobar que no queda trabajo útil:

```bash
git worktree remove .worktrees/<IDENTIFICADOR>
git branch -d agente/<IDENTIFICADOR>
```

No usar `--force` ni borrar un worktree sucio.

## Reglas no negociables

- Arquitectura hexagonal; adaptadores intercambiables; ninguna dependencia
  concreta entra en el núcleo.
- Denegación por defecto y privilegio mínimo. Lo no concedido expresamente se
  rechaza.
- No crear ficheros nuevos en `*/ports` mientras siga vigente DEC-051. Si un
  contrato falta, se informa antes de ampliar la superficie.
- No introducir datos personales, exportaciones reales, claves, certificados,
  tokens, rutas privadas del usuario ni secretos en Git, fixtures o logs.
- No usar adaptadores falsos en una ruta que se presente como real. Los datos
  sintéticos se marcan como DEMO y quedan fuera de producción.
- No abrir acceso directo a tablas para evitar una autorización, una función
  gobernada o un recibo.
- Código y documentación en castellano coherente, salvo términos técnicos de
  uso universal. No mezclar idiomas en nombres de dominio.
- Reutilizar software libre mantenido cuando resuelva correctamente la
  capacidad. Antes de añadir una dependencia: licencia compatible, actividad,
  vulnerabilidades, superficie y motivo documentados.
- Documentar el código y actualizar el manual en el mismo trabajo.
- Preservar cambios ajenos. Nunca `git reset --hard`, `git checkout --`,
  rebase destructivo ni limpieza general del árbol.
- `T02` está aparcado. No reabrirlo ni ampliar `internal/vec/ports` como forma
  de esquivar el trabajo funcional.

## Trabajos ya ocupados

No iniciar otra implementación de estas fichas.

| Identificador | Trabajo | Propietario actual | Archivos o zona reservada | Estado |
| --- | --- | --- | --- | --- |
| `BORRADORES-TRANSACCION` | Guardado PostgreSQL/KMS de borradores | Dirección + revisor independiente | `deploy/postgresql/bolsa_convocatorias/**` | En revisión. |
| `BORRADORES-HMAC` | Identificadores seguros y rotación | Agente Codex + revisor | `config/**`, `internal/app/bootstrap/*desarrollo*`, generador de credenciales | Corrigiendo cobertura. |
| `IMPORTADOR-CONVOCA` | Importación gobernada de dos hojas XLS | Agente Codex | Nueva zona del importador y pruebas sintéticas; no tocar T20 | En desarrollo. |
| `DIRECCION-INTEGRACION` | Tablero, revisión final y commits | Agente director | `ESTADO_PROYECTO.md`, `README.md`, integración | Activo permanentemente. |

## Trabajos libres para agentes manuales

### `AUDITORIA-DURABLE` — primera tarea libre

**Objetivo:** llevar a PostgreSQL la evidencia probatoria de Bolsa que hoy solo
existe en memoria o en verticales aisladas, sin conectarla todavía a datos
reales.

**Leer:** entrada T12 de
`docs/autoprogramacion_orquesta_pendientes_2026-07-16.md`, EIPD y decisiones de
auditoría/recibos citadas allí.

**Alcance inicial:** inventario verificable, diseño de transacción y primer
adaptador durable que conserve cadena, recibo, outbox y puntos de recuperación
con privilegios mínimos. Reutilizar contratos existentes; ningún puerto nuevo.
Pruebas PostgreSQL reales, reinicio y corrupción/duplicado. No tocar
`deploy/postgresql/bolsa_convocatorias/**` mientras esté reservado.

**Terminado cuando:** existe una vertical durable reproducible, saneada y
revisable; no cuando solo se añade DDL o documentación.

### `REGISTRO-ACCESOS` — análisis paralelo; código tras congelar su enlace

**Objetivo:** registrar cada acceso a datos personales con actor opaco,
expediente, finalidad, base de autorización, instante y versión de reglas.

**Leer:** entrada T13 y documentación de cumplimiento. Puede realizar el
preflight y preparar pruebas ahora. No debe escribir una integración que dependa
de decisiones aún no congeladas por `AUDITORIA-DURABLE`.

**Límites:** nada de DNI, nombre, correo o motivo clínico en el registro; no
registrar causas técnicas sensibles; consulta gobernada y retención definida
por configuración.

### `AYUDA-BOT-PUBLICO` — libre e independiente

**Objetivo:** primer bot de ayuda limitado a información pública de Bolsa, sin
acceso a expedientes ni datos personales.

**Alcance:** preparar corpus gobernado y versionado desde bases, FAQ y manuales
públicos; recuperación con citas de procedencia; respuesta cerrada si no hay
fuente; API detrás de interfaz intercambiable. No seleccionar proveedor de IA
ni enviar documentación a servicios externos. Tests de fuga y prompt injection.

**Terminado cuando:** el recorrido usa solo corpus público sintético o ya
publicado y demuestra que ninguna ruta interna puede alcanzarse.

### `FORMATOS-SALIDA` — libre tras preflight

**Objetivo:** cerrar un caso de uso real de generación documental a través del
registro de formatos existente, sin fijar PDF/DOCX en el núcleo.

**Alcance:** seleccionar un documento no personal de Bolsa, plantilla
versionada, salida elegible y recibo; reutilizar adaptadores existentes.
No tocar firma/custodia real ni crear otro registro de formatos. Si el contrato
actual no permite el recorrido, entregar mapa exacto antes de editar.

### `ACCESIBILIDAD-UAT` — libre, sin alterar lógica de negocio

**Objetivo:** automatizar la comprobación de teclado, foco, contraste, zoom,
estados vacío/carga/error y audio/transcripción de las pantallas disponibles.

**Alcance:** pruebas y correcciones acotadas en recursos web que no estén
reservados por `BORRADORES-TRANSACCION`. No rediseñar ni añadir datos de
presentación. Entregar además guion de prueba manual para RRHH.

## Trabajos bloqueados o que no deben duplicarse todavía

| Trabajo | Bloqueo real | Se desbloquea cuando |
| --- | --- | --- |
| Lector Go de borradores | Falta lectura SQL gobernada V2 y proyección cifrada completa | Se cierre el contrato de autorización/lectura inmediatamente posterior a la transacción. |
| Conexión web definitiva de borradores | Depende del lector, confirmador y autorización anteriores | Los adaptadores Go pasan revisión. |
| Datos reales de Convoca | Faltan durabilidad probatoria y registro de accesos | `AUDITORIA-DURABLE` y `REGISTRO-ACCESOS` estén integrados. |
| Publicación o retirada oficial | Requiere aprobación firmada y dependencias autoritativas (DEC-091) | Exista el flujo de aprobación/firma y se componga. |
| Firma, registro, antivirus, correo, Telegram o pagos productivos | Falta proveedor corporativo y autorización de Sistemas | Puede avanzarse contra interfaces de desarrollo, pero no declararse producción. |
| Contratos y ceses | Falta modelo propio y relaciones con RRHH/nómina | Dirección abra ese frente después del ciclo principal de selección. |

## Pruebas mínimas para cualquier entrega

Según el alcance, el agente ejecuta y registra:

```bash
gofmt -w <ficheros-go-propios>
go test <paquetes-afectados> -count=1
go test -race <paquetes-con-concurrencia> -count=1
go test ./...
go vet ./...
git diff --check
```

Además: integración real de PostgreSQL si toca persistencia; pruebas del
generador si toca credenciales; barrido de secretos y datos personales; límites
de tamaño; `bash -n` para shell. Una prueba omitida debe figurar como limitación,
nunca como verde.

## Entrega obligatoria del agente

El mensaje final contiene exactamente la información necesaria para revisión:

```text
Trabajo: <IDENTIFICADOR>
Estado: GO / NO-GO / bloqueado
Commit(s): <hash>
Archivos: <lista acotada>
Qué funciona de extremo a extremo: <evidencia>
Pruebas ejecutadas: <comandos y resultado>
Pruebas no ejecutadas: <motivo>
Seguridad y datos: <comprobaciones>
Limitaciones reales: <sin ocultarlas>
Cambio propuesto para ESTADO_PROYECTO.md: <fila y estado>
Revisión independiente solicitada: sí/no
```

El autor no actualiza `ESTADO_PROYECTO.md` ni declara cerrada una capacidad.
Dirección lo hace después de revisar e integrar el commit.
