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
BEGIN, SET, RELEASE y COMMIT fallidos devuelven configuración incompleta,
sin datos ni reintento. Un 42501 de QUERY sin envelope y acuse confirmados
tampoco se presenta como denegación funcional.

SQL devuelve referencias opacas y las coordenadas/HMAC del acto inicial exacto.
`NuevoPostgreSQLConFuenteADMIN` exige el proveedor original de SujetoID, CuentaID
y CuentaOrdinariaID, y el seudonimizador existente. Coteja las tres huellas y
sus coordenadas, sin derivar identificadores desde PersonaRef ni desde digests.
El constructor antiguo conserva su firma y deniega sin esa fuente.

El lector opcional de archivo privado exige SHA aprobado, objeto cerrado,
versión 1 y un máximo de 32 KiB. Reutiliza el protocolo privado existente:
ruta absoluta limpia, padre propio 0700, sin Git ni enlaces en sus ancestros,
raíz `os.Root` y archivo regular propio 0600 que no sea un enlace.
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
sesión por cuenta. Tras cotejar el acuse revalida configuración, cuenta, sesión
y ventanas del vínculo dentro de la misma subtransacción. El evento original
permanece intacto. El adaptador no reinvoca un vínculo ante COMMIT incierto.

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

El ensayo PostgreSQL del replay sigue pendiente. Con un vínculo real, debe
retener el cerrojo `vec:is16:sesion:<sesion_ref>`, iniciar el replay exacto y
comprobar que espera. Si una ventana de configuración, sesión o vínculo vence
durante esa espera, al liberar el cerrojo no puede devolver el recibo permitido.
Se comparan después las huellas del evento y vínculo originales, sin cambiar
el reloj ni renovar fuentes para superar el caso.

Las preimágenes originales del ejercicio se han perdido: el plan y el material
HMAC conservados sólo contienen referencias y digests. Dirección autorizó para
la retoma un nuevo juego sintético por el circuito oficial de fuentes,
titularidad, huellas y CAS, con sus originales guardados en un fixture privado.
Todavía no se ha generado. El proveedor actual permanece cerrado.

## Resultado del ensayo del 4 de octubre de 2026

El código `2d24894cf56b2a08aa55a45439b2b1c4e387c95f` y SQL SHA256
`2f88dafdcf8f2a6038cb42bc5879bc5b143f23e240184b6c040352e721bd5c92`
recibieron dos GO exactos. Las pruebas focales y race, vet y Semgrep local pasan.

AD192 se instaló con un COMMIT y sus dos vectores del productor terminaron
correctamente. Core, ACL y los otros CHECK permanecieron idénticos; se
conservaron las 6.254 filas de auditoría, su huella completa y la cabeza de cadena.
Ese ensayo acredita instalación y ACL, no un append positivo de sesión real.

IS16 quedó sin instalar. El preflight encontró dos interfaces v1 ausentes:

```text
IS14: resolver_identidad_admin_perfiles_propietaria_v1(
  text,text,text,text,text,timestamptz,timestamptz)
CA31: listar_admin_preperfil_propietaria_v1(text,text,text)
```

Se lanzó el UP pese a esos dos resultados negativos. La guarda lo rechazó con
`preimagen incompatible` antes de crear rol o tablas y la conexión revirtió la
transacción. Después se comprobó que los objetos IS16 seguían ausentes y que
historia y metadatos eran idénticos. CA36 no se instaló ni se intentó.

El clon quedó en una copia fría privada con dump, ACL globales y manifiesto;
todos los archivos del tar se cotejaron por SHA256 con el cluster detenido.
El contenedor y sus datos temporales se retiraron. Actas, configuración,
fixtures, fuentes privadas y fríos anteriores se conservan fuera de Git.
La retoma requiere coordinar la restauración y resolver estas dependencias;
no repetir AD192 ni crear fachadas sustitutivas.
