# Contrato del núcleo V3 para la subida local

Precondición de `lista_sql_codexo_subida_documental_20261014.txt`: el equipo C instala y revisa AD273 antes de AUT95, D20 y RUM7. Este equipo no instala ni reconstruye el núcleo V3. La propuesta anterior de AD270 y su prueba está fuera de Git, en el borrador privado que recibió dirección.

El acto del aspirante usa el perfil M4 `persona_meritos_cartera` y su asignación externa vigente. La fuente `meritos_cartera_propia_v2.json` publica cinco acciones: las tres anteriores más `meritos.cartera.comprobante.subir_propio` y `meritos.cartera.comprobante.descargar_propio`. Ambas nuevas requieren `modulo_id=meritos`, `tipo_recurso=hecho`, ámbito exacto `persona_ref`, finalidad `custodiar_evidencia_meritos`, motivo del catálogo existente `motivos_autorizacion_meritos_cartera` y obligaciones `[]`. Subir concede exactamente `[contenido,estado,recibo]`; descargar, `[contenido]`. Las tres concesiones anteriores conservan sus campos y la obligación `auditar`; la lectura nueva de cartera añade `evidencias` solo en el rol de cinco acciones.

El consumidor de datos usa exclusivamente el LOGIN `vec_externo_documentos_datos`, miembro heredado del único grupo `vec_documentos_subida_datos`, sin SET ni ADMIN y sin herencia hacia otros grupos. AD273 debe crear el grupo y el consumidor `vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)` con retorno de siete campos: `decision_ref,efecto_ref,huella_efecto_sha256,consumo_huella_sha256,auditoria_ref,consumida_en,consumo_nuevo`. D20 lo llama en la misma transacción serializable, exige `consumo_nuevo=true` y efecto exacto. La autoridad de las decisiones sigue siendo M4, con registro y revalidación viva M4; AUT95 añade solo este LOGIN nominal a los helpers de registro, sesión y revalidación ya existentes. No se conceden funciones de registro directamente al grupo de datos.

AD273 debe aceptar únicamente las dos audiencias nuevas `vec_documentos.subida_local.acto_modulo.v1` para subir y consulta propia por carga y `vec_documentos.subida_local.descarga_propia.v1` para consulta por documento y apertura. La referencia del recurso PDP es `carga_id` salvo primera consulta por `documento_id`; ambos tienen forma `ref:<64 hex>`. El efecto se liga al SHA-256 del JSON compacto `{"ambitos":{"persona_ref":"<persona>"},"atributos":{"preimagen_sha256":"<sha256 de bytes de preimagen>"}}`. La decisión de almacén usa otra huella y no debe servir para D20. C consolida el núcleo desde el cuerpo instalado en HZ37, conservando sus permisos, consumo, auditoría y toda la familia Mayan histórica. La publicación de confianza externa amplía la lista positiva de 23 a 25 audiencias sin retirar ninguna.

La fachada D20 y el lector de cartera dependen además de que AD273 adapte `registrar_lectura_cartera_meritos_v1` para los ocho campos anteriores y los nueve nuevos, manteniendo `cotejar_cartera_meritos_actual_v1` como autoridad viva. El grupo propio de frontera es `vec_documentos_frontera_externa`; su LOGIN `vec_externo_documentos_frontera` queda separado del de datos. D20 publica las seis fachadas de estado y la auditoría de frontera con estos grupos. Dietas y Cronos quedan cerrados hasta publicar sus actos, autorías y funciones nominales de propiedad; la rama instalada de este corte acepta solo Méritos.

## Preparación de Dietas y Cronos para el siguiente corte

El contrato de puertos, datos, funciones del dueño y límites de lo que existe
está en [`contrato_subida_comun.md`](contrato_subida_comun.md). Los
identificadores Go preparados son `dietas.justificante.subir_propio` y
`dietas.justificante.descargar_propio`, con finalidad
`custodiar_justificante_dietas`, y `cronos.justificacion.subir_propio` y
`cronos.justificacion.descargar_propio`, con finalidad
`custodiar_justificacion_cronos`. No son concesiones publicadas ni quedan
habilitados por AD273, AUT95 o D20 tal como están.

Cuando Dietas y Cronos publiquen fuente, perfil, motivo y función nominal de
titularidad, C añadirá al núcleo consolidado AD273 únicamente las audiencias
y ramas exactas de esos actos. Cada acción conserva su identidad,
superficie, finalidad, campos y obligación propios; el consumidor común no
convierte el acto de Méritos en un permiso para los otros módulos. Dietas16
ya define `vec_dietas.validar_propietario_subida_documental_v1(text,bigint,text,text)`
y espera la fachada Documentos de seis parámetros. La función Cronos y la
proyección de original admitido siguen pendientes. Ninguna de estas piezas
forma parte del ensayo de Méritos de esta candidata.
