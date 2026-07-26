package postgres

// Acreditación viva completa y llamada comparten sentencia, conexión y
// snapshot. PostgreSQL no evalúa la subconsulta STABLE de la rama CASE falsa.
const consultaSQLRecuperacionResultadoCoberturaO405 = `
	WITH acreditacion(
	  oid_funcion,usuario_sesion,usuario_efectivo,tls_activo,primaria,
	  login_seguro,grupo_seguro,membresia_directa,membresia_total,
	  login_sin_autoridad,grupo_sin_autoridad,privilegios_exactos,
	  funcion_exacta,propietario_exacto,seguridad_exacta,
	  configuracion_exacta,firma_retorno_exactos,lenguaje_probin_exactos,
	  prosrc_exacto,definicion_exacta
	) AS MATERIALIZED (` +
	consultaAcreditacionPoolRecuperacionCoberturaO405 + `
	),
	manifiesto AS MATERIALIZED (
		SELECT oid_funcion,
		       COALESCE(
		         oid_funcion=$12::oid
		         AND usuario_sesion<>''
		         AND usuario_sesion=usuario_efectivo
		         AND tls_activo=$13::boolean
		         AND primaria
		         AND login_seguro
		         AND grupo_seguro
		         AND membresia_directa
		         AND membresia_total
		         AND login_sin_autoridad
		         AND grupo_sin_autoridad
		         AND privilegios_exactos
		         AND funcion_exacta
		         AND propietario_exacto
		         AND seguridad_exacta
		         AND configuracion_exacta
		         AND firma_retorno_exactos
		         AND lenguaje_probin_exactos
		         AND prosrc_exacto
		         AND definicion_exacta,
		         false
		       ) AS acreditado
		  FROM acreditacion
	),
	resultado AS MATERIALIZED (
		SELECT manifiesto.acreditado,
		       CASE WHEN manifiesto.acreditado THEN (
		         SELECT llamada.resultado_json::text
		           FROM vec_contratacion_temporal
		             .recuperar_resultado_propio_decision_cobertura_o405_v1(
		               $14::jsonb
		             ) AS llamada
		       ) END AS contenido
		  FROM manifiesto
	)
	SELECT CASE
	         WHEN acreditado
	          AND pg_catalog.octet_length(contenido)<=$15::bigint
	         THEN contenido
	       END,
	       CASE WHEN acreditado THEN
	         COALESCE(
	           pg_catalog.octet_length(contenido),
	           0
	         )::bigint
	       ELSE 0::bigint END,
	       acreditado
	  FROM resultado`
