\set ON_ERROR_STOP on
-- Solo lectura, ANTES de instalar B100 en la principal: cuántas personas por
-- bolsa quedarán fuera de turno al aplicarla (respuesta de Mi bolsa sin una
-- situación registrada y en vigor después). Mismo criterio que
-- respuestas_portal_sin_reflejar_v1.
BEGIN READ ONLY;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT q.bolsa_ref, q.respuesta, count(*)
  FROM (SELECT DISTINCT ON (r.participacion_ref) r.bolsa_ref, r.participacion_ref, r.respuesta
          FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
         WHERE r.respondida_en <= now()
           AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
                            WHERE s.participacion_ref = r.participacion_ref
                              AND s.registrada_en >= r.respondida_en AND s.registrada_en <= now()
                              AND s.desde <= now())
         ORDER BY r.participacion_ref, r.respondida_en DESC, r.respuesta_ref DESC) q
 GROUP BY q.bolsa_ref, q.respuesta
 ORDER BY q.bolsa_ref, q.respuesta;
ROLLBACK;
