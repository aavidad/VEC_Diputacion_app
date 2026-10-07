# Identidad interna ordinaria sintética

Esta consola prepara y aplica una persona nueva con una cuenta ordinaria en una organización existente. No crea credenciales, perfiles, permisos, cuentas administrativas ni relaciones de empleo. El resultado no permite iniciar sesión en el portal.

Todos los archivos de entrada y salida privados deben estar fuera de Git, en un directorio propio `0700`, con permisos `0600`. Las rutas deben ser absolutas y no contener enlaces. El catálogo ES/EN es público. No pases DSN, contraseñas ni material HMAC en argumentos.

```text
vec-provisionar-identidad-interna plan --fuente /ruta-privada/fuente.json --plan /ruta-privada/plan.json --textos cmd/vec-provisionar-identidad-interna/catalogos/es.json
vec-provisionar-identidad-interna apply --plan /ruta-privada/plan.json --conexion /ruta-privada/conexion.json --aprobacion /ruta-privada/aprobacion.json --acuse /ruta-privada/acuse-1.json --textos cmd/vec-provisionar-identidad-interna/catalogos/es.json --timeout 30s
vec-provisionar-identidad-interna reconcile --plan /ruta-privada/plan.json --conexion /ruta-privada/conexion.json --aprobacion /ruta-privada/aprobacion.json --acuse /ruta-privada/acuse-2.json --textos cmd/vec-provisionar-identidad-interna/catalogos/es.json --timeout 30s
```

`plan` funciona sin conexión. La fuente sigue `PlanIdentidadInternaSinteticaV1`; el archivo generado contiene `plan` y `huella_plan_sha256`. Solo admite `desarrollo` y `sintetico_declarado`. La vigencia del plan no puede superar 24 horas y la referencia de operación admite hasta 128 caracteres. El canon se limita a 64 KiB. La organización lleva versión esperada positiva y `organizacion.procedencia_huella_sha256`, la huella SHA256 de su procedencia vigente; la persona nueva lleva versión cero. Las evidencias incluyen referencias y huellas, nunca claves HMAC.

El DBA prepara la preimagen mediante `vec_autorizacion.preimagen_identidad_interna_sintetica_v1(jsonb,text)` con el plan y el material HMAC privado, y configura la aprobación por el circuito privado. El material HMAC permanece en la configuración privada del DBA; nunca forma parte del plan. La consola no concede permisos para leer esa preimagen. La aprobación privada contiene `huella_plan_sha256`, `preimagen_sha256`, `configuracion_sha256` y `aprobacion_ref`.

La configuración privada del DBA exige `declaracion_aprobada='fuente_maestra_sintetica_persona_cuenta_ordinaria_titularidad'`. La aprobación externa debe acreditar la fuente, versión y huella para la persona, su cuenta ordinaria y la titularidad, exclusivamente en el clon de desarrollo identificado. Esta declaración pertenece al circuito del DBA y no se añade al plan del cliente. `alcance_fuente` permanece en `sintetico_declarado`: no acredita empleo, perfil, nivel de garantía de identidad ni certificado.

La conexión privada contiene `dsn`, `permitir_socket_desarrollo` (debe ser falso), `host_esperado`, `puerto_esperado`, `base_esperada`, `login_esperado` y las mismas cuatro propiedades de aprobación. Solo acepta TCP loopback con TLS verificado y sin destinos alternativos. El LOGIN nominal y sus permisos para la fachada deben existir: la consola no los crea ni eleva el rol.

`apply` llama a `provisionar_identidad_interna_sintetica_v1(text,text)` con el canon y la huella aprobada. `reconcile` llama a `recuperar_identidad_interna_sintetica_v1(text,text)` con la operación original y la misma huella. Ambos usan una transacción SERIALIZABLE de lectura y escritura con zona UTC. Cotejan destino, aprobación y huellas localmente y contra el recibo; SQL conserva la autoridad sobre esos datos.

La consola confirma también los intentos denegados cuando la fachada devuelve su auditoría. Un error de COMMIT deja el resultado indeterminado: recupera con `reconcile`, el mismo plan y otro archivo de acuse. No se reintenta automáticamente. Si el COMMIT termina pero falla la escritura del acuse, `reconcile` permite recuperar el recibo durable.

Códigos de salida: `0`, resultado preparado o confirmado; `1`, entrada o operación rechazada; `2`, catálogo o resultado local incompleto, incluido COMMIT indeterminado. El acuse privado conserva la respuesta validada de la autoridad; la salida de consola solo contiene diagnóstico y estado.
