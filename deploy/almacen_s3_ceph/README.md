# Perfil reproducible del almacén S3 compatible con Ceph RGW

Este perfil prepara dos buckets exclusivos y ejecuta el contrato real del
adaptador. No convierte por sí solo una instalación en apta para producción:
Sistemas debe validar además copias, restauración, alta disponibilidad,
rotación de credenciales, KMS, monitorización, capacidad y procedimientos de
incidente sobre la versión concreta de Ceph desplegada.

## Invariantes exigidos

- TLS verificado; nunca se admite HTTP ni `InsecureSkipVerify`.
- El proceso no hereda proxies, no sigue redirecciones y solo puede conectar
  con los CIDR declarados en `VEC_CEPH_REDES_PERMITIDAS`; no se admiten listas
  universales ni prefijos de red no canonicos.
- Buckets diferentes para cuarentena y documentos admitidos.
- Versionado y Object Lock habilitados en ambos buckets.
- Ninguna retención predeterminada en cuarentena. El adaptador necesita borrar
  una carga fallida por su `VersionId` exacto.
- Cada documento admitido nace con retención `COMPLIANCE` explícita en el mismo
  `PutObject`; no existe una ventana sin WORM.
- SHA-256 real declarado y comprobado. El ETag no se usa como huella.
- Cifrado por objeto `AES256` o `aws:kms`. Elegir KMS solo después de configurar
  y probar el proveedor de claves de RGW.
- Las claves de objetos y marcadores son opacas y no contienen DNI, nombre,
  correo ni identificadores de expediente aportados por el usuario.
- Eliminación funcional denegada por defecto. Las eliminaciones técnicas de la
  sonda, compensación y carga temporal siempre señalan una versión exacta.

## Preparación

1. Crear un usuario técnico RGW exclusivo y entregar sus credenciales mediante
   el gestor de secretos; no guardarlas en `.env` ni en el repositorio.
2. Exportar las variables de [.env.example](.env.example). Para
   `provisionar_buckets.sh`, AWS CLI usa `AWS_ACCESS_KEY_ID`,
   `AWS_SECRET_ACCESS_KEY` y, si corresponde, `AWS_SESSION_TOKEN`.
3. Configurar `AWS_CA_BUNDLE` cuando RGW use una autoridad corporativa.
4. Ejecutar `./provisionar_buckets.sh` sobre buckets de prueba vacíos y
   exclusivos. El script no cambia una regla de retención predeterminada
   existente sin que AWS CLI lo acepte expresamente.
5. Ejecutar `./probar_contrato.sh`.

La sonda escribe claves técnicas aleatorias sin datos personales. Elimina el
objeto usado para probar legal hold, pero deja dos versiones pequeñas bajo la
retención configurada (origen y destino) para demostrar que un `DELETE` real es
rechazado. Se deben retirar mediante una regla de ciclo de vida cuando expire la
retención; no se usa bypass de `GOVERNANCE`.

## Permisos mínimos a concretar en la política RGW

El principal de ejecución necesita, limitado a estos dos buckets y a los
prefijos `vec/`: consulta de bucket, versionado y Object Lock; `PutObject`,
`GetObject`, `GetObjectVersion`, `DeleteObjectVersion`, lectura/escritura de
retención y lectura/escritura de legal hold. No necesita listar ni leer otros
buckets. La compatibilidad exacta de acciones IAM debe comprobarse en la
versión de Ceph objetivo; la sonda cierra el arranque si alguna capacidad se
anuncia pero no funciona.

La cuenta operativa de la aplicación no debe tener permisos administrativos de
bucket, cambiar políticas, desactivar versionado, alterar Object Lock ni borrar
buckets. El aprovisionamiento debe usar otra identidad separada.

## Criterio de aceptación

No etiquetar el conector como «producción» hasta conservar como evidencia:

- versión y configuración de Ceph RGW;
- salida satisfactoria de `probar_contrato.sh`;
- prueba de restauración desde copia inmutable;
- prueba de rotación/revocación de credenciales y clave de derivación;
- prueba de KMS si se usa `aws:kms`;
- alertas de fallo de sonda, integridad, capacidad y limpieza temporal;
- revisión de política IAM y de los periodos jurídicos de retención.

Referencias oficiales: [Ceph RGW Object Operations](https://docs.ceph.com/en/latest/radosgw/s3/objectops/),
[Ceph RGW Bucket Operations](https://docs.ceph.com/en/latest/radosgw/s3/bucketops/),
[AWS SDK for Go v2: endpoints](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-endpoints.html)
y [AWS S3: presigned URLs y checksums](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html).
