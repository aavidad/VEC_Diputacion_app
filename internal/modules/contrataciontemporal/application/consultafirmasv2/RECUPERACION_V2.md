# Recuperación de la evidencia nominal de firma

El servicio de recuperación exige una autorización actual con acción y audiencia
propias y los 48 campos exactos. La consulta anterior conserva sus 44 campos.
El contexto procede del proveedor nominal; la solicitud no elige la identidad.

La recuperación conserva el texto original del canon histórico y coteja su
huella mediante el validador común. Comprueba su enlace con el certificado,
fecha, expediente, originales, revisiones PDF y circuito registrados. Una
cancelación durante la proyección devuelve un resultado vacío.

`MaterialRootSHA256` identifica la huella del material de firma V2. La respuesta
actual de registro la calcula mediante `MaterialFirmaVerificadaV2.HuellaSHA256`
y exige que coincida con `ReciboFirmaDocumento.SolicitudHuella`. CT172 conserva
ese dato en `firma_documento_v1.solicitud_huella_sha256`. Este nombre exterior
no identifica la raíz de la capacidad criptográfica V3.

El futuro lector SQL debe recuperar ese valor de la fila original y el canon
por la fachada propietaria de Autorización, ligado a la evidencia CT original.
Debe consumir la autorización y registrar la consulta en la auditoría común
antes de devolver datos, dentro de la misma transacción. Un permiso histórico
del firmante no autoriza al consultante actual; la retirada del cargo tampoco
elimina la evidencia histórica.

Faltan las fachadas SQL de lectura, el consumidor técnico común, el adaptador y
el montaje nominal HTTP. Las pruebas del servicio usan dependencias sintéticas.
La consulta HTTP existente sigue declarando recuperación parcial; este servicio
no acredita un recorrido ni recuperación tras reiniciar.
