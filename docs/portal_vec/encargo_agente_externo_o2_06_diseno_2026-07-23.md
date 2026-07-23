# Encargo aislado O2-06A — diseño ejecutable del adaptador y reconciliación

## Mandato

Lee este documento completo y ejecuta el encargo sin ampliar alcance. Lee antes
`AGENTS.md`, `ORQUESTACION_AGENTES.md`,
`docs/instruccion_direccion_2026-07-18.md`,
`docs/portal_vec/diseno_transaccion_atomica_alta_o2_05_2026-07-23.md`,
`docs/portal_vec/confianza_y_capacidad_vec_ad_3_o2_05_2026-07-23.md` y las
filas O2-05/O2-06 del tablero.

Esta es una tarea de diseño técnico verificable, no una implementación
anticipada. O2-05 está congelando todavía la firma SQL. No programes contra
una interfaz cambiante ni edites el worktree de O2-05.

Son obligatorias todas las reglas del proyecto: diseño y documentación en
castellano coherente; claves i18n para errores públicos; arquitectura hexagonal
y conectores intercambiables; denegación por defecto; neutralidad
web/escritorio/CLI/MCP; ausencia de secretos y datos personales; seguridad de
Administración pública y todas las puertas de `AGENTS.md`. Los términos
técnicos universales pueden conservarse, pero no se admite mezclar castellano
e inglés en el dominio.

## Subagentes obligatorios

El agente principal crea dos revisores de solo lectura:

1. especialista PostgreSQL/pgx: transacciones, errores `40001`/`40P01`,
   cancelación, `COMMIT` indeterminado, pool, roles y reinicio;
2. especialista de seguridad/hexagonal: capacidad VEC-AD-3, segregación,
   minimización, replay, recibos, pruebas y neutralidad de canal.

El principal es el único editor. Los revisores entregan GO/NO-GO y evidencias.

## Worktree

```bash
cd /home/alberto/Trabajo/VEC_Diputacion_app
git worktree list
test ! -e .worktrees/ct-o2-06-diseno
git worktree add .worktrees/ct-o2-06-diseno \
  -b agent/ct-o2-06-diseno feature/contratacion-temporal
cd .worktrees/ct-o2-06-diseno
```

Si ya existe, detente. No uses `/tmp` ni rutas externas para el worktree.

## Objetivo

Entregar el diseño exacto y directamente implementable de O2-06 para que,
cuando O2-05 obtenga GO, un programador pueda construir el adaptador Go sin
reabrir decisiones ni descubrir tarde una frontera ausente.

Debes determinar:

- datos exactos disponibles en `OrdenConfirmarAlta`;
- materiales adicionales necesarios para emitir/consumir VEC-AD-3 sin hacer
  exportables capacidades internas;
- propietario y transporte del emisor segregado;
- firma SQL prevista y mapeo tipo Go↔PostgreSQL;
- política de transacción y sesión;
- reconciliación de resultado indeterminado;
- clasificación pública de errores;
- recibo mínimo y su validación;
- composición posterior O2-07;
- matriz completa de pruebas unitarias y PostgreSQL real.

## Escritura permitida

Crea únicamente:

```text
docs/portal_vec/diseno_adaptador_reconciliacion_o2_06_2026-07-23.md
docs/portal_vec/revisiones/o2_06a_revision_postgresql_2026-07-23.md
docs/portal_vec/revisiones/o2_06a_revision_seguridad_2026-07-23.md
```

No modifiques Go, SQL, scripts, configuración, tablero, README ni manuales.
Puedes ejecutar pruebas y hacer experimentos descartables fuera del repositorio,
pero no versionar código de producción.

## Fuentes obligatorias

Inspecciona, al menos:

- `internal/modules/contrataciontemporal/ports/alta.go`;
- `internal/modules/contrataciontemporal/application/registro_solicitud.go`;
- `internal/modules/contrataciontemporal/adapters/postgres/preparacion_alta.go`;
- `internal/vec/ports/capacidad_atestacion_autorizacion_v3.go`;
- `internal/vec/adapters/seguridad/confianzaatestacion/capacidad_v3_*`;
- migraciones O2-03/O2-05 ya integradas;
- candidato O2-05 solo después de que Dirección comunique un SHA estable.

No leas cambios sin commit del productor O2-05 ni dependas de rutas privadas.

## Decisiones que el documento debe cerrar

1. **Frontera hexagonal:** nombre, responsabilidad y dependencias del adaptador
   de `TransaccionAltas`; aplicación no importa pgx ni confianza concreta.
2. **Emisión segregada:** cómo obtiene la capacidad breve sin reunir
   credenciales emisora y ejecutora, sin serialización general ni fallback
   local.
3. **Material probatorio:** de dónde proceden decisión, motivo, contexto,
   VEC-AD-3, COSE, prueba, raíz SPKI, HMAC y efecto canónico; qué falta hoy en
   la orden y cómo incorporarlo sin autoridad fabricable.
4. **Sesión SQL:** rol exclusivo, `search_path`, UTC, límites de sentencia,
   bloqueo e inactividad, solo `EXECUTE`, sin DML/`SET ROLE`.
5. **Reintentos:** únicamente `40001`/`40P01`, transacción nueva, retroceso
   acotado y presupuesto global.
6. **Resultado indeterminado:** algoritmo de reconciliación por todos los
   alias HMAC gobernados, decisión/correlación, actor/perfil/organización y
   huella de efecto. Nunca repetir ciegamente un `COMMIT`.
7. **Reinicio:** ninguna dependencia de memoria del proceso, txid, WAL o reloj
   del cliente.
8. **Recibo:** campos, copia defensiva, comprobación antes y después del
   `COMMIT`, replay exacto y adulteración.
9. **Errores:** catálogo estable y redactado; ninguna causa SQL, identidad,
   clave o dato personal cruza la frontera.
10. **Pruebas:** éxito, replay, concurrencia, rotación, revocación, expiración,
    timeout antes/después del `COMMIT`, respuesta perdida, reinicio, recibo
    adulterado, ACL, rollback y cadenas.

Incluye diagramas de secuencia para éxito, `40001`, `COMMIT` indeterminado y
reconciliación tras reinicio. Incluye tabla de archivos futuros y write-set
disjunto para la implementación.

## Puertas y entrega

Verifica enlaces y referencias, `git diff --check`, ausencia de rutas privadas
y Gitleaks si está disponible. Commits pequeños en castellano; no empujes.

Entrega SHA, dictámenes de los dos revisores, decisiones cerradas, bloqueos
reales y la declaración:

**O2-06A es diseño; no implementa, integra ni habilita producción.**
