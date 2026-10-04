# AD191: acuse del gobierno de usuarios

AD188 fallaba con SQLSTATE `55000` antes de leer la configuración de origen: la variable `a record` ocultaba el alias SQL `a`. AD191 renombra únicamente ese registro a `v_confirmacion`, el destino de su consulta y su conversión al recibo. El diagnóstico con las tres sustituciones devolvió `permitido` y un recibo real dentro de una transacción revertida.

La migración conserva la firma, el OID, el propietario, las ACL y toda la metadata de `pg_proc` salvo `prosrc`. Exige la definición instalada medida antes de cambiarla y verifica la definición candidata y la fuente resultante. No añade tablas, permisos, claves ni auditorías; las publicaciones posteriores mantienen la transacción y la auditoría común de AD188. Nunca se reaplica AD188 ni se ejecuta DOWN.

Dependencia: AD188 instalada, definición SHA256 `f2d7076ec1aaee4c4070f3caabbfc03b4ea725f55553ff525f0935e49421dc39`, fuente SHA256 `29ac3718c0b255641ccf68994ff07147cdd3e1c006a5969d9c73d664faa57e7e`. Postimagen: definición `aadaaa7f9a1029564e4c02587394a175ab4a5dd45c67526d50ea034f8b6ef973`, fuente `4dc9a79e919633b6c9f355e368dfef00c33701cf6ce1dc5ed39def98b4f7ad2e`. La lista causal contiene sólo AD191; no sustituye la instalación de sus dependencias.

Comprobación estructural: `probar_gobierno_usuarios_alias_000191.sql`, sin escrituras durables. Pendiente de dos revisiones e instalación única en el clon. Después: LOGIN técnico real, dos claves y dos punteros, replay, reinicio y negativas, manteniendo la historia y la regresión de Contratación temporal. El firmante DEV del clon procede del circuito existente autorizado; no es el firmante de la principal.
