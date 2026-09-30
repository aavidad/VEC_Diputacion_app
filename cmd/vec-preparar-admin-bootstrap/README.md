# Preparar el plan de los dos primeros administradores

`vec-preparar-admin-bootstrap` prepara un documento privado y muestra **solo su
huella SHA-256**. No conecta con la base, no da perfiles y no acredita por sí
solo que las cuentas o los certificados declarados sean auténticos. El operador
debe cotejar las fuentes por su canal privado y aprobar la huella exacta antes
de cualquier alta. La futura AUT24 tendrá que repetir esas comprobaciones y el
CAS dentro de una única transacción; este programa no invoca AUT24.

La fuente JSON de versión 1 contiene `preparado_en`, `caduca_en`,
`control_continuidad_revision_esperada` igual a 1 y
`bootstrap_estado_esperado` igual a `pendiente`; `rol` con
`version_ref` exacta `rol:administracion_perfiles:v2`, su huella publicada,
`control_revision` y `control_huella_sha256`; `fuente_identidad` y
`fuente_ca_admin` con `referencia`, `version` y `huella_sha256`; y `personas`
con **dos** entradas ordenadas por `persona_ref` ascendente. Cada entrada
incluye `cuenta_ref`, `cuenta_version`, `persona_ref`, `persona_version`,
`perfil_ref` y `vinculo_ref` nuevos, `preimagen_huella_sha256`,
`procedencia` (referencia, versión y huella), `vigente_hasta`, y
`certificado_admin` con `persona_ref`, `cuenta_ref`, `huella_sha256`,
`ca_huella_sha256` y `acreditacion` (referencia, versión y huella). La huella
de CA del certificado debe coincidir con `fuente_ca_admin.huella_sha256`.
Las fechas son UTC, sin fracciones de segundo; la vigencia de cada perfil debe
cubrir al menos la caducidad del plan. No se incluyen nombres, DNI, PEM, claves
ni credenciales. Los identificadores y huellas deben venir de sus autoridades;
el operador no debe inventarlos para superar la validación.

La fuente y el plan viven fuera de Git, en un directorio propio `0700`, con
rutas absolutas sin enlaces. La fuente es un fichero `0600` de hasta 64 KiB.
El programa exige campos exactos y únicos, y rechaza datos adicionales.

```sh
go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json

go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json -cotejar
```

El plan se crea exclusivamente como `0600`. Repetir con el mismo material
devuelve la misma huella sin sobrescribir; un plan diferente se rechaza.
`-cotejar` exige que el plan ya exista. La huella se calcula sobre el JSON
compacto del campo `plan`, en el orden de campos del formato v1, sin incluir
`huella_plan_sha256`; el archivo añade esa huella como campo separado. La
fecha de preparación es parte del material: cambiarla cambia la huella.

AUT23 crea un control global con revisión 1 y estado pendiente. El dominio
exige revisión de continuidad **cero en cada preimagen individual**. Son dos
precondiciones distintas y AUT24 debe cotejar ambas, además de la publicación
real del rol, sus asignaciones, la identidad nominal, la CA y la no revocación.
El archivo ordinario puede ser alterado por su propietario; su huella y el
cotejo detectan cambios, pero no sustituyen la aprobación externa ni un recibo
durable de alta. Hasta ese recibo no hay administradores dados de alta.
