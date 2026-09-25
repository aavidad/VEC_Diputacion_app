\set ON_ERROR_STOP on
-- Catálogo de permisos SINTÉTICO para desarrollo y demostración a RRHH. Los
-- nombres y cuantías siguen la configuración visible en la aplicación actual
-- y están SIN CONFIRMAR: los fija RRHH al responder la duda 41 (catálogo,
-- unidad, circuito J-A/A, justificante). Cada fila lleva sintetico=true y la
-- fuente «fuente:sintetica:wcronos-a-confirmar-duda-41». Cuando RRHH publique
-- el catálogo real se añadirá una versión nueva que cierre ésta; nunca se
-- editan filas. Requiere cronos_v1 000008. Se puede reejecutar sin duplicar.
-- Horas en minutos; días en días.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_cronos_v1.permiso_catalogo
 (version_ref,permiso_ref,nombre,orden,vigente_desde,vigente_hasta,unidad,computo,circuito,minimo,maximo_solicitud,maximo_mensual,maximo_anual,
  justificante_exigido,solicitable,sintetico,fuente_ref,publicada_en)
SELECT 'catalogo:cronos:'||v.clave||':sintetico-1','permiso:cronos:'||v.clave,v.nombre,v.orden,
  '2026-01-01 00:00:00'::timestamp AT TIME ZONE 'Europe/Madrid',NULL,v.unidad,v.computo,v.circuito,v.minimo,v.max_solicitud,v.max_mensual,v.max_anual,
  v.justificante,v.solicitable,true,'fuente:sintetica:wcronos-a-confirmar-duda-41',clock_timestamp()
FROM (VALUES
 ('asistencia-examenes','Asistencia a exámenes',10,'hora','laborables','J-A',15,450,NULL,NULL,true,true),
 ('asuntos-propios','Asuntos propios',20,'dia','laborables','A',1,NULL,NULL,6,false,true),
 ('bolsa-dias-trienios','Bolsa de días por trienios',30,'dia','laborables','A',1,NULL,NULL,2,false,false),
 ('bolsa-dias-antiguedad','Bolsa de días de vacaciones por años de servicio',40,'dia','laborables','A',1,NULL,NULL,2,false,false),
 ('bolsa-horaria-conciliacion','Bolsa horaria por conciliación',50,'hora','laborables','A',15,NULL,NULL,1800,false,true),
 ('compensacion-festivos','Compensación de festivos',60,'dia','laborables','A',1,NULL,NULL,28,false,false),
 ('compensacion-tiempo-ocio','Compensación de tiempo y ocio libre',70,'dia','laborables','A',1,NULL,NULL,15,false,false),
 ('formacion-iaap-inap','Curso de formación agrupada IAAP-INAP',80,'hora','laborables','J-A',15,NULL,NULL,5400,true,true),
 ('embarazo-maternidad','Embarazo y baja maternal',90,'dia','naturales','A',1,NULL,NULL,112,true,false),
 ('enfermedad-grave-2','Enfermedad grave u hospitalización de familiar de 2.º grado',100,'dia','laborables','J-A',1,4,NULL,NULL,true,true),
 ('enfermedad-grave-1','Enfermedad grave u hospitalización de familiar de 1.er grado',110,'dia','laborables','J-A',1,5,NULL,NULL,true,true),
 ('enfermedad-sin-baja','Enfermedad sin baja',120,'dia','laborables','A',1,NULL,NULL,4,true,false),
 ('fallecimiento-2-distinta-localidad','Fallecimiento de familiar de 2.º grado en distinta localidad',130,'dia','laborables','J-A',1,4,NULL,NULL,true,true),
 ('fallecimiento-1','Fallecimiento de familiar de 1.er grado',140,'dia','laborables','J-A',1,3,NULL,NULL,true,true),
 ('fallecimiento-1-distinta-localidad','Fallecimiento de familiar de 1.er grado en distinta localidad',150,'dia','laborables','J-A',1,5,NULL,NULL,true,true),
 ('fallecimiento-2','Fallecimiento de familiar de 2.º grado',160,'dia','laborables','J-A',1,2,NULL,NULL,true,true),
 ('formacion-externa','Formación externa',170,'hora','laborables','J-A',15,NULL,NULL,3600,true,true),
 ('gestion-servicio','Gestión de servicio',180,'hora','laborables','A',15,NULL,NULL,NULL,false,true),
 ('horas-medico','Horas de médico',190,'hora','laborables','J-A',15,180,NULL,NULL,true,true),
 ('horas-sindicales','Horas sindicales',200,'hora','laborables','A',15,NULL,3600,NULL,false,true),
 ('miercoles-santo-corpus','Miércoles de Semana Santa y Corpus',210,'dia','laborables','A',1,NULL,NULL,1,false,false),
 ('nacimiento','Nacimiento de hijo o hija',220,'dia','naturales','A',1,NULL,NULL,7,true,false),
 ('trabajo-no-presencial','Trabajo no presencial',230,'dia','laborables','A',1,NULL,NULL,31,false,false),
 ('traslado-domicilio','Traslado de domicilio',240,'dia','naturales','J-A',1,1,NULL,1,true,true),
 ('vacaciones','Vacaciones',250,'dia','laborables','J-A',1,NULL,NULL,22,false,false)
) AS v(clave,nombre,orden,unidad,computo,circuito,minimo,max_solicitud,max_mensual,max_anual,justificante,solicitable)
ON CONFLICT (version_ref) DO NOTHING;
COMMIT;
-- El calendario laboral y su asignación a cada persona sintética se publican
-- aparte (duda 43): sin asignación, la pantalla dice que falta el calendario
-- y los permisos en días laborables no se pueden solicitar.
