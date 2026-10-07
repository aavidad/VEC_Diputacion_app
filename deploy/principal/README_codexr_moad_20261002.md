# Número de expediente externo de MOAD

Esta entrega requiere la PR de Fin #430. La aplicación conserva el número asignado por MOAD desde la apertura; las nuevas altas lo piden y no lo generan. El catálogo de formato distribuido está en `internal/modules/contrataciontemporal/adapters/numeracion/numero_expediente.json`. Para otro formato, dirección puede configurar `VEC_CT_NUMERO_EXPEDIENTE_SOURCE_PATH` con un catálogo versionado de la misma estructura. No incluir datos personales ni secretos en ese archivo.

Antes de instalar, dirección conserva la copia de la base y sus permisos. Comprueba CT151 y el paquete CT165/166/167, y aplica únicamente el UP nuevo de `lista_sql_codexr_moad_20261002.txt`. Los archivos `.ensayo.sql` y `.positivo.sql` son pruebas del clon; no se ejecutan en la principal. No reaplicar UP instalado ni ejecutar DOWN sobre historia.

Se publica el binario y su grafo web del mismo corte. En una apertura nueva, RRHH introduce el número que ya existe en MOAD, por ejemplo el del catálogo. Un formato incorrecto devuelve un error editable; una repetición con la misma clave y datos conserva el recibo. Una clave confirmada con otro número se rechaza.

Las altas anteriores conservan su número histórico y no se presentan como números MOAD. El control de recuperación de una entrega antigua envía la misma clave sin inventar un número; CT169 obtiene el material original con autorización vigente y el servidor contrasta HMAC y recibo antes del commit. Si el perfil cambió, el perfil anterior sirve como dato para ese contraste, sin otorgar permisos. Omitir el número en una alta nueva de MOAD no permite recuperar otro material.

Validación de desarrollo: revisiones independientes, Go y race/vet globales, Node y controles de calidad por fases, ensayo SQL con datos sintéticos y un doble nominal V3 declarado. No acredita un recorrido integrado con COSE/PDP reales, Chrome y reinicio. Dirección debe comprobar ese recorrido tras instalar en su entorno autorizado; esta entrega no escribe en la principal ni conecta con una API de MOAD.
