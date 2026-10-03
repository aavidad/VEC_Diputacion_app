# Comprobar un material local del plan de firma

`vec-plan-firma-validar` comprueba un fichero preparado para el kit privado de
gobierno del plan de firma CT. Recibe la ruta local y el SHA-256 esperado de
los bytes exactos:

```sh
go run ./cmd/vec-plan-firma-validar /ruta/privada/material.json <sha256-esperado>
```

Con material válido emite un JSON con SHA-256, operación, ID del catálogo,
versión, revisión y `material_validado_sin_autorizacion`. El estado solo
acredita una comprobación local del fichero. La herramienta no publica el
catálogo, no obtiene permisos, no consume una decisión V3 y no registra
auditoría ni outbox.

El fichero tiene un límite de 4 MiB. Sus trece claves y tipos deben coincidir
exactamente con el contrato; se rechazan variantes de mayúsculas, valores
`null` indebidos y claves repetidas. Las trazas y eventos deben conservar los
bytes canónicos del modelo común. La comprobación nunca sustituye los bytes
del fichero por una representación nueva.

Guarde el fichero original con acceso privado para repetir la operación con
los mismos bytes y la misma clave. Cada intento de gobierno necesitará una
autorización nueva por el circuito correspondiente. Esta herramienta no
modifica el fichero ni prepara una autorización. Use únicamente datos
sintéticos mientras el circuito no esté admitido.

Un rechazo sale con código 1 y un error de argumentos o escritura de salida
con código 2. La salida de error solo indica `material_rechazado`: no incluye
la ruta, el contenido ni los datos de actor del fichero.
