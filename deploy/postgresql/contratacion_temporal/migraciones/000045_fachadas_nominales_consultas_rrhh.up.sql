\set ON_ERROR_STOP on
\set ct000045_aplicar_acl true
\set ct000045_avanzar_barrera true
-- CT-000045: dos fachadas nominales; ningún componente se aplica aislado.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 24
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 8
 FOR UPDATE;

\ir 000045_componentes/010_guardas_frontera.sql
\ir 000045_componentes/020_fachada_cuadro.sql
\ir 000045_componentes/030_fachada_detalle.sql
\ir 000045_componentes/090_acl_catalogo.sql
\ir 000045_componentes/095_avance_barreras.sql

COMMIT;
