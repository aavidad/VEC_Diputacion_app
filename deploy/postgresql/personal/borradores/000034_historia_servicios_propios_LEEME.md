# Historia de servicios propios: preparación SQL

Personal34 y AD180 preparan la lectura de revisiones de servicios del empleado
vinculado al actor. La historia permanece en Personal17; no se crea otra tabla
histórica ni se consulta la ficha RRHH con su permiso. Personal25 conserva su
contrato amplio anterior; esta pieza acota primero los servicios.

AD180 coteja tres huellas POST175: definición y cuerpo del núcleo, y CHECK de
audiencias. Comprueba el CHECK antes de reconstruir el núcleo. Personal34
exige la fachada y su permiso exclusivo para Personal. La captura de la
postimagen es una precondición medida; no equivale al ensayo de AD180/P34.
Ninguna de las dos migraciones está instalada por esta entrega.

La postimagen causal medida tras AD175 contiene una vez cada marca de
extensión: condición de suite, exclusión de `bolsa_llamamiento` y rama de
`servicios_certificados_propios`. No contiene el perfil de historia. AD180
usa las huellas de definición
`abb1fbc89278ba9d27fec7e7e4e2e4003c47dfafd6caa1f81cb82ea8ec1b1f13`,
de cuerpo `0bb3c0965a84cd10c29e0d8821cc2ebc2861c108b0f817a900501b083b0ea250`
y de CHECK `1a0e8f3d3f3e054184c594c6e6dc121d8a07e7ea01dcbe204f76b9ea55421222`.
El CHECK usa condiciones `OR`; el formato
antiguo `ANY (ARRAY[...])` no corresponde. La postimagen se conserva fuera
de Git en la caché local de Dirección. Antes de instalar faltan las dos
revisiones exactas y el ensayo PostgreSQL 18 de AD180 seguido de Personal34.

Personal34 depende de los helpers de Personal16, las historias de Personal17,
la instantánea de catálogo de Personal20, la ficha de Personal22 y el consumidor
AD180. No llama a la fachada CRN11 de Personal26 ni al consumidor AD181. Se
retira su comprobación de existencia: imponía una dependencia histórica circular
sin aportar autoridad a esta lectura. Las comprobaciones de empleado, firma,
concesión propia, roles, permisos y consumo permanecen.

No ejecutar esta lista como instalador ni reaplicar migraciones conservadas.

La fachada devuelve todas las revisiones conocidas que solapan el intervalo
de efectos solicitado. Más de 200 revisiones provoca 54000, sin datos truncados.
El periodo prestado, la vigencia del registro y el conocimiento son distintos.
La cobertura queda no_acreditada mientras no exista una fuente que la acredite.
Acto y fuente son referencias; no permiten descargar ni acreditar documentos.

Consumo, lectura y auditoría se confirman juntos. El recibo es la PK aud_v3_
del nuevo consumo común, ligada a su huella; no es otro UUID sin conservar.
No se garantiza recuperar la respuesta ni el corte a partir de esa huella.
El adaptador valida el JSON y su evidencia antes de COMMIT. Tras rollback,
los errores o denegaciones se registran por el puerto común de intentos.

Campos permitidos exactos, en orden: cobertura,corte,evidencia,revisiones.
La acción y audiencia propias v1 y las claves anidadas cerradas evitan ampliar
la proyección implícitamente. Las pruebas SQL preparadas no están ejecutadas.
Faltan la preimagen causal, dos revisiones finales y el ensayo PostgreSQL 18
tras AD175: lectura propia, referencia ajena, revocación, exceso, fallo del
registrador y conservación de la cadena común tras reinicio. Dirección instala.
