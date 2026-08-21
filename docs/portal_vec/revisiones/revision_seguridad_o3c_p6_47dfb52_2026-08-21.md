# Revisión independiente de seguridad O3C-P6 — `47dfb52`

Tarea: `O3C-P6`, revisión de seguridad independiente del candidato exacto
`47dfb52`.

Estado: **NO-GO**.

## Alcance y write-set

La revisión fue read-only sobre fuentes, pruebas, autoridad y material del
candidato. El único fichero creado por esta revisión es esta acta Markdown;
no se modificaron fuentes, pruebas, contratos, ledgers canónicos ni estado
transversal.

## Hallazgo bloqueante S-01 — atestación de selectores sujeta a sustitución TOCTOU

`ejecutar_aislado` crea `atestacion-selectores.tsv` y entrega su pathname al
proceso probado mediante `O3C_P5_ATESTACION`. Tras esperar al líder, el
conductor llama a `validar_atestaciones_selectores`, que vuelve a resolver ese
pathname y únicamente comprueba que el objeto final sea fichero regular no
symlink, de propietario `$EUID` y modo `0600`, y después lee su contenido.

No se conserva ni se compara un inode/dispositivo (ni una huella de bytes)
entre la creación previa al `exec` y la lectura posterior. Por tanto, un
proceso descendiente que permanezca vivo durante la ventana de validación, o
cualquier proceso del mismo UID con acceso al contenedor `0700`, puede retirar
el fichero y sustituirlo por otro regular `0600` antes de la validación. El
`kill -0` del grupo no constituye una barrera de concurrencia ni congela el
pathname; además la comprobación de residuos y la retirada del runtime ocurren
después de validar la atestación.

Esto permite que una atestación fabricada sea aceptada como evidencia del
selector ejecutado. El filtro de nombres/campos, el propietario/mode y el
marcador de FD no corrigen la identidad temporal del objeto leído.

## Criterio de cierre exigido

Antes de entregar el pathname al child debe capturarse una identidad estable
del fichero (dispositivo+inode, y preferiblemente tamaño/huella); tras el
retorno, debe revalidarse esa identidad y el contenido desde un descriptor
abierto de forma segura, rechazando cualquier sustitución, enlace, cambio de
propietario/mode o deriva. La revisión posterior debe repetir las pruebas
focales y las puertas de seguridad del contrato.

## Comprobaciones adicionales

La inspección confirmó que el candidato contiene preconteo/limpieza del
`TMPDIR`, comprobaciones de UID/mode/inode del directorio temporal, cierre de
FD ambiental mediante marcador, revalidación de destino y padre antes de
`mv -n -T`, conservación de inode origen/destino y `SHA256SUMS` que incluye
`tmpdir_selectores.tsv` y `utilidades.tsv`. Esas comprobaciones no mitigan
S-01, por lo que no se emite GO.

Commit de acta: pendiente de commit local único por el agente revisor.
