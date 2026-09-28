---
name: revisar-sql-vec
description: Revisa migraciones SQL de VEC (PostgreSQL 18) antes de pedir su integración; lista de fallos típicos de PL/pgSQL, ACL, reconstrucción de funciones y tablas ausentes en la principal. Usar en toda minitarea que añada o cambie SQL.
---

# Revisar SQL de VEC

En Codex el ensayo sobre el clon de la principal lo ejecuta Dirección: incluye en la rama `deploy/principal/lista_sql_<rama>.txt` con las SQL nuevas en orden causal. Antes de avisar «PR: lista», repasa a mano el punto 3.

1. Localiza las SQL nuevas o cambiadas de la rama (git diff contra origin/main) y su orden causal (lista `deploy/principal/lista_sql_*.txt` si existe; si no, por módulo y número).
2. Ensáyalas con `~/.claude/hooks/vec/ensayar_sql_clon.sh <lista.txt> <raiz>` (extrae la raíz con `git archive <hash> | tar -x` en /dev/shm). Debe terminar en ENSAYO-OK.
3. Revisa además a mano:
   - `CASE … THEN` dentro de la condición de un `IF` de PL/pgSQL sin paréntesis (corta la condición en el primer THEN).
   - Alias usados que no existen en el FROM (p. ej. `to_jsonb(p)` sin `pg_proc p`).
   - Reconstrucción de funciones con `replace(pg_get_functiondef(...))`: la marca debe existir una sola vez en el cuerpo REAL de la principal (tiene consumidores de Dietas, Personal, Cronos…).
   - Tablas o políticas que pueden no existir en la principal (Bolsa 000001/000002 no están instaladas): comprobar existencia antes de LOCK/ALTER.
   - `REVOKE … FROM PUBLIC` en toda función y tipo nuevo; `CREATE POLICY … TO <rol>`, nunca PUBLIC; `SECURITY DEFINER` con `search_path` fijo.
   - Idempotencia, `pg_advisory_xact_lock`, `lock_timeout` y `statement_timeout`.
4. Informe breve: ENSAYO-OK o FALLO con la salida, y hallazgos (bloquea / revisar / menor) con fichero:línea y corrección concreta.
