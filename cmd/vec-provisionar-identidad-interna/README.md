# Identidad interna ordinaria sintética

Esta consola registra una declaración preparatoria de persona, cuenta ordinaria y titularidad en una organización existente. Las nuevas procedencias quedan como `no_autoritativa`. No crea credenciales, perfiles, permisos, cuentas administrativas ni relaciones de empleo. El resultado no permite iniciar sesión en el portal.

Todos los archivos de entrada y salida privados deben estar fuera de Git, en un directorio propio `0700`, con permisos `0600`. Las rutas deben ser absolutas y no contener enlaces. El catálogo ES/EN es público. No pases DSN, contraseñas ni material HMAC en argumentos.

```text
vec-provisionar-identidad-interna plan --fuente /ruta-privada/fuente.json --plan /ruta-privada/plan.json --textos web/static/textos/es/identidad-interna-provision.json
vec-provisionar-identidad-interna apply --plan /ruta-privada/plan.json --conexion /ruta-privada/conexion.json --aprobacion /ruta-privada/aprobacion.json --acuse /ruta-privada/acuse-1.json --textos web/static/textos/es/identidad-interna-provision.json --timeout 30s
vec-provisionar-identidad-interna reconcile --plan /ruta-privada/plan.json --conexion /ruta-privada/conexion.json --aprobacion /ruta-privada/aprobacion.json --acuse /ruta-privada/acuse-2.json --textos web/static/textos/es/identidad-interna-provision.json --timeout 30s
```

`plan` funciona sin conexión. La fuente sigue `PlanIdentidadInternaSinteticaV1`; el archivo generado contiene `plan` y `huella_plan_sha256`. Solo admite `desarrollo` y `sintetico_declarado`. La vigencia del plan no puede superar 24 horas y la referencia de operación admite hasta 128 caracteres. El canon se limita a 64 KiB. La organización lleva versión esperada positiva y `organizacion.procedencia_huella_sha256`, la huella SHA256 de su procedencia vigente; la persona nueva lleva versión cero. Las evidencias incluyen referencias y huellas, nunca claves HMAC.

El DBA prepara la preimagen mediante `vec_autorizacion.preimagen_identidad_interna_sintetica_v1(jsonb,text)` con el plan y el material HMAC privado, y configura la aprobación por el circuito privado. El material HMAC permanece en la configuración privada del DBA; nunca forma parte del plan. La consola no concede permisos para leer esa preimagen. La aprobación privada contiene `huella_plan_sha256`, `preimagen_sha256`, `configuracion_sha256` y `aprobacion_ref`.

La configuración privada del DBA exige `declaracion_aprobada='registro_preparatorio_sintetico_persona_cuenta_titularidad'`. Autoriza la escritura en el clon identificado; no acredita la veracidad de los datos de Persona ni convierte referencias y huellas del plan en fuentes maestras. La admisión posterior requiere una fuente independiente verificable y nuevas versiones por el circuito propietario. Este corte no proporciona un contexto nominal utilizable para provisión, empleo o asignación de perfiles. Conserva recibos e historia sin reescribirlos.

La conexión privada contiene `dsn`, `permitir_socket_desarrollo` (debe ser falso), `host_esperado`, `puerto_esperado`, `base_esperada`, `login_esperado` y las mismas cuatro propiedades de aprobación. Solo acepta TCP loopback con TLS verificado y sin destinos alternativos. El LOGIN nominal y sus permisos para la fachada deben existir: la consola no los crea ni eleva el rol.

El DBA debe retirar `TEMP` de `PUBLIC` en la base de destino y no concederlo al LOGIN de provisión. La fachada comprueba esa condición, pero no sustituye la restricción de privilegios del motor.

`apply` llama a `provisionar_identidad_interna_sintetica_v1(text,text)` con el canon y la huella aprobada. `reconcile` llama a `recuperar_identidad_interna_sintetica_v1(text,text)` con la operación original y la misma huella. Ambos usan una transacción SERIALIZABLE de lectura y escritura con zona UTC. Cotejan destino, aprobación y huellas localmente y contra el recibo; SQL conserva la autoridad sobre esos datos. `--help` muestra los subcomandos y sus opciones sin necesitar el catálogo.

La consola confirma también los intentos denegados cuando la fachada devuelve su auditoría. Si el COMMIT es incierto, conserve **el mismo plan, conexión y aprobación**; ejecute `reconcile` con una ruta de acuse nueva. No vuelva a ejecutar `apply` para averiguar el resultado. Si el COMMIT termina pero falla la escritura del acuse, use esa misma recuperación. Cada consulta de recuperación deja un intento nuevo en la auditoría.

Después de `caduca_en`, `apply` queda denegado incluso con la misma operación. Para leer un recibo ya confirmado, use solo `reconcile` mientras sigan vigentes la configuración privada, el LOGIN y el permiso de lectura de ese circuito.

Códigos de salida: `0` significa ayuda solicitada, plan preparado o recibo permitido y guardado; `1` indica falta de comando, entrada/ruta/acuse inválido o inseguro, operación no confirmada, denegación o error registrado; `2` indica que falta un catálogo válido, no se pudo escribir la salida, el COMMIT es incierto o no se guardó el acuse. Si aparece `catalogo_no_disponible`, revise la ruta `--textos` y el archivo ES/EN antes de repetir el comando. Sin argumentos, el código `1` acompaña a la ayuda estructurada.

En el JSON de consola, `estado=permitido` significa que existe un recibo del registro preparatorio; `estado=denegado` o `error` indica que solo se confirmó su intento. `transaccion_confirmada=true` significa que PostgreSQL confirmó esa respuesta, **no** que haya permitido el alta. `replay=true` significa que se devolvió el recibo original sin crear otra cuenta; la consulta o repetición añade su propio intento. `acuse_guardado=true` indica que el archivo de acuse privado se escribió. `codigo` y `estado` son valores de protocolo estables, incluso cuando el mensaje se muestra en inglés.
