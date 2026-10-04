# Catálogo de textos de incidencias técnicas

Los archivos de `textos/<idioma>/incidencias_tecnicas.json` contienen exactamente
`esquema`, `version_catalogo` y `plantillas`. Las hojas son cadenas, conforme al
catálogo i18n común. `esquema` vale `"1"`; `version_catalogo` es la representación
decimal exacta de `domain.VersionCatalogoIncidenciasTecnicas`, también como cadena.
No se admiten números JSON, signos, ceros iniciales, decimales, espacios ni otras
revisiones. El lector exige los 13 códigos actuales y textos fijos sin sustituciones.

El límite de lectura es 1 MiB. Un catálogo ambiguo, incompleto o de otra revisión
se rechaza antes de crear el almacén. Este formato externo nuevo todavía no está
instalado: la revisión debe expresarse como cadena en la configuración entregada.
El JSONL de incidencias y resultados conserva sus esquemas v1 y sus mismos campos;
la revisión del catálogo no se añade a cada registro.
