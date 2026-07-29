package postgres

const (
	consultaCuadroRRHHPostgreSQL = `
SELECT contenido_canonico,
       cursor_siguiente,
       esquema,
       acceso_ref,
       secuencia::bigint,
       anterior_sha256,
       huella_sha256,
       vinculo_identidad_huella_sha256,
       alcance_huella_sha256,
       registrada_en,
       auditoria_vec_ref,
       auditoria_vec_huella_sha256,
       consumo_vec_huella_sha256,
       contenido_huella_sha256,
       resultado_huella_sha256,
       cursor_huella_sha256,
       generada_en,
       expediente_ref,
       version_expediente::bigint,
       total,
       recibo_sello_sha256
  FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(
       ROW($1::text, $2::text, $3::text)::
           vec_contratacion_temporal.alcance_consulta_rrhh_v1,
       ROW($4::text, $5::text, $6::text, $7::smallint, $8::text)::
           vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
       $9::bytea,
       $10::bytea,
       $11::bytea,
       $12::bytea,
       $13::numeric,
       $14::numeric,
       $15::bytea,
       $16::bytea,
       $17::bytea,
       $18::bytea
  )`

	consultaDetalleRRHHPostgreSQL = `
SELECT contenido_canonico,
       esquema,
       acceso_ref,
       secuencia::bigint,
       anterior_sha256,
       huella_sha256,
       vinculo_identidad_huella_sha256,
       alcance_huella_sha256,
       registrada_en,
       auditoria_vec_ref,
       auditoria_vec_huella_sha256,
       consumo_vec_huella_sha256,
       contenido_huella_sha256,
       resultado_huella_sha256,
       cursor_huella_sha256,
       generada_en,
       expediente_ref,
       version_expediente::bigint,
       total,
       recibo_sello_sha256
  FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
       ROW($1::text, $2::text, $3::text)::
           vec_contratacion_temporal.alcance_consulta_rrhh_v1,
       ROW($4::text, $5::numeric)::
           vec_contratacion_temporal.consulta_detalle_rrhh_v1,
       $6::bytea,
       $7::bytea,
       $8::bytea,
       $9::bytea,
       $10::numeric,
       $11::numeric,
       $12::bytea,
       $13::bytea,
       $14::bytea,
       $15::bytea
  )`
)
