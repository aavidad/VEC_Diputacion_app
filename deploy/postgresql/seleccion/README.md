# Selección — PostgreSQL

Esquema `vec_seleccion` del módulo Selección («Convoca integrado», fase 1):
publicación versionada de convocatorias y solicitudes de participación
(borrador, presentación con justificante interno, historia, auditoría de
accesos de RRHH y outbox). Selección no lee tablas de otros módulos.

## Orden de instalación

1. `roles_up.sql` (superusuario, una vez): grupos `vec_seleccion_propietario`,
   `vec_seleccion_migrador` y `vec_seleccion_ejecutor`, todos NOLOGIN.
2. `GRANT vec_seleccion_ejecutor TO <LOGIN ejecutor de Bolsa>;` — única
   membresía nueva: vec-server reutiliza la conexión de Bolsa.
3. AD3-89 y AD3-90 (`autorizacion_atestada_v3/migraciones/000089…`, `000090…`).
4. `migraciones/000001_solicitudes_participacion.up.sql`.

Detección:

| Paso | Expresión |
|---|---|
| roles | `SELECT count(*)=3 FROM pg_roles WHERE rolname IN ('vec_seleccion_propietario','vec_seleccion_migrador','vec_seleccion_ejecutor')` |
| AD3-89 | `SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_solicitud_propia_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL` |
| AD3-90 | `SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL` |
| Selección 000001 | `SELECT to_regclass('vec_seleccion.solicitud') IS NOT NULL` |

Los DOWN se aplican en orden inverso y solo sin historia (sin solicitudes ni
accesos; sin claves de capacidad de las audiencias de AD3-89/90).

## Ensayo

`probar_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP` restaura el volcado sintético en
PostgreSQL 18.4 desechable (datos en `/dev/shm`, `--rm`, sin red) y prueba
ROLLBACK sin rastro, UP, doble UP rechazado, ida y vuelta DOWN/UP con el
núcleo AD3 idéntico, ACL con roles reales, el recorrido completo con dobles
de las fachadas V3 (`pruebas_sql/`) y el reinicio de PostgreSQL.
