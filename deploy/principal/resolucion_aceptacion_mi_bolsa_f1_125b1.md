# Consumidor de resolución de Mi Bolsa

Primer bloque del segundo corte de F1-125. Reutiliza el acto de revisión de
respuesta de RRHH y añade la variante portal al contrato HTTP existente.
No añade configuración ni permisos.

La activación SQL CT249 va en el bloque siguiente. Sin ella, el lector no
devuelve la marca de disponibilidad nativa: el formulario no se ofrece.
El POST portal devuelve el error catalogado de servicio no disponible si
la operación SQL falta; no devuelve un recibo parcial. Las consultas y la
vía de correo conservan su contrato.

Con ambos bloques instalados, Antonio acepta desde Mi Bolsa. RRHH revisa
respuesta y plazo y confirma desde la ficha; recibe un recibo CT distinto
del justificante de Mi Bolsa. Una petición incierta conserva la misma clave
al reintentar. La marca de origen del primer corte no habilita la escritura:
la emisión, participación y respuesta se acreditan de nuevo en el acto.

Antes de presentar ambos bloques para integración, comprobar juntos el 201,
el replay 200 con las mismas referencias tras reinicio y el arranque sin
CT249. La recuperación del recibo al reabrir la ficha queda para el tercer
corte. La guía describe la activación; no sustituye su evidencia.
