# Cuenta y vínculo de sesión ADMIN — IS16

IS16 conecta la cuenta privilegiada nominativa con una sesión real de IS y con
la selección vigente de CA31. No concede perfiles ni cambia el núcleo V2.
Requiere IS14/15, CA31 y la familia de auditoría AD192. CA36 consume después el
vínculo desde su propietario, con otro LOGIN y pool.

La migración está reservada. Esta entrega prepara SQL, adaptador y pruebas;
no acredita instalación, recuperación PostgreSQL ni recorrido HTTP. Sólo se
instala tras las dos revisiones del mismo candidato y el ensayo de Dirección.

## Contratos y permisos

El grupo `vec_identidad_sesiones_v1_admin_perfiles_runtime` tiene cuatro
ejecutables: acreditación, resolución de cuenta, vínculo y registro del fallo
de fuente privada. Sólo recibe USAGE de IS y CONNECT; no recibe tablas. El
LOGIN, proceso, entorno, host, audiencia, espacio de identidad y vigencia se
configuran fuera de Git. El grupo y el LOGIN de CA36 son distintos.

Las fachadas de resolución y vínculo reciben la observación mTLS confiable y
referencias de evento/correlación generadas en el servidor. Devuelven un objeto
cerrado `{estado, datos}` y un acuse AD192 de cinco campos. Go valida referencia,
secuencia, huella, correlación e instante antes del COMMIT o de usar los datos.
La denegación se confirma antes de devolver el error funcional. Un acuse
inválido, un fallo de transacción o un COMMIT incierto no devuelven éxito.

SQL devuelve referencias opacas y las coordenadas/HMAC del acto inicial exacto.
`NuevoPostgreSQLConFuenteADMIN` exige el proveedor original de SujetoID, CuentaID
y CuentaOrdinariaID, y el seudonimizador existente. Coteja las tres huellas y
sus coordenadas, sin derivar identificadores desde PersonaRef ni desde digests.
El constructor antiguo conserva su firma y deniega sin esa fuente.

El lector opcional de archivo privado exige SHA aprobado, objeto cerrado,
versión 1, archivo regular sin acceso de grupo/otros y un máximo de 32 KiB.
El archivo lo entrega el productor original. No hay ejemplo de identidad real
ni generación alternativa de preimágenes. Cuenta y fuente redactan JSON,
formato y logs para impedir que salgan identificadores o material SQL.

Si el proveedor privado o un HMAC fallan, Go revierte el SAVEPOINT que contiene
la resolución positiva y su auditoría. Fuera de ese SAVEPOINT, dentro de la
misma transacción, `rechazar_fuente_cuenta_admin_v1` registra el error AD192.
Un fallo de ese acuse o del COMMIT sigue cerrado.

## Vínculo exacto y recuperación

El vínculo `vis_` usa 128 bits CSPRNG y conserva versión, SHA, autenticación,
sesión, persona, ambas cuentas, perfil, certificado/CA, política, revisión de
selección, control de sesión y fuente. Es inmutable. El replay sólo admite el
mismo plan, referencia, evento, operador y acuse original; nunca elige otra
sesión por cuenta. El adaptador no reinvoca un vínculo ante COMMIT incierto.

La fachada propietaria para CA36 es:

```text
cotejar_vinculo_sesion_admin_v1(
  referencia text, version numeric, huella_sha256 text,
  autenticacion_ref text, sesion_ref text, cuenta_ref text,
  perfil_ref text, solicitado_en timestamptz, metodo_observado text
) -> jsonb
```

Devuelve exactamente doce claves: `referencia`, `version`, `huella_sha256`,
`autenticacion_ref`, `sesion_ref`, `persona_ref`, `cuenta_ref`, `perfil_ref`,
`seleccion_revision`, `fuente_ref`, `fuente_sha256` y `vigente_hasta`.
La ausencia o una autoridad no vigente devuelven NULL. El método debe coincidir
con la sesión real, certificado o DNIe; no basta con pertenecer al mismo conjunto.
Sólo el owner CA recibe EXECUTE, nunca el LOGIN ni el grupo CA.

En SERIALIZABLE coteja también CA31 mediante su fachada. En READ COMMITTED
coteja IS certificado/CA/CRL/política/cuentas/sesión/control actuales; CA36
coteja su selección CA31 actual bajo sus propios cerrojos. Ese resultado IS
por sí solo no acredita un contexto recuperado. CA36 conserva registro,
enlace y acuse original exactos; la recuperación no añade auditoría.

`ContextoConVinculoSesionADMIN` y `VinculoSesionADMINDeContexto` transmiten la
ligadura tipada antes de `CrearVinculoAutenticacionActorV2ConResultado`, sin
ampliar puertos generales ni aceptar datos de la petición HTTP.

## Comprobaciones y dependencia pendiente

Las pruebas focales Go cubren discrepancias de las tres huellas, coordenadas y
fuente; acuses ajenos/incompletos/futuros; JSON con claves repetidas o ajenas;
y redacción de identificadores. Usan dobles aislados del contrato, no una
sesión productiva. El vector SQL comprueba ACL y ausencia bajo ambas transacciones;
su ejecución queda pendiente del ensayo autorizado.

Falta localizar el archivo o productor original de las preimágenes del ejercicio
privado actual. El plan de fuentes y el material HMAC conservados sólo contienen
referencias y digests. Sin ese artefacto el proveedor permanece cerrado; no se
declara la sesión conectada ni se regeneran identidades.
