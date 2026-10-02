# Preparación del recorrido de firma R5

Estos lectores preparan un ensayo sintético sobre un clon PostgreSQL sin red externa. No instalan SQL, publican cargos ni convierten un PDF de prueba en evidencia del expediente.

El director prepara fuera de Git un directorio propio 0700 con `kit.json`, `runtime/runtime-config.json`, el binario estático `runtime/vec-server`, material nominal compatible con la base y PKI de prueba. Las claves, contraseñas PKCS12, token, DSN y registros privados tienen permisos 0600.

```sh
node scripts/recorridos/firma_r5/probar_preparacion.mjs check /ruta/privada/kit
node scripts/recorridos/firma_r5/probar_preparacion.mjs plan-grxfirma /ruta/privada/kit
node scripts/recorridos/firma_r5/probar_preparacion.mjs plan-app /ruta/privada/kit
```

`check` informa dependencias pendientes. `plan-*` imprime argumentos sin valores de configuración ni credenciales y no inicia servicios. El kit fija el propietario, contenedor PostgreSQL, imagen local, fuente, SHA256 del binario de VEC y ruta/SHA256 del binario de GrxFirma. Se verifica que PostgreSQL esté activo con `NetworkMode=none` antes de compartir su espacio de red.

Las acciones `start-app` y `start-grxfirma` requieren `runtime/arranque-aprobado.json`, emitido por el director con `owner`, `pg_container_id`, `config_sha256` de `kit.json`, `runtime_config_sha256`, `binary_sha256` y `material_nominal_verificado=true`. Este recibo controla únicamente el arranque del ensayo; no concede permisos VEC ni acredita una firma. El clon debe ofrecer PostgreSQL TLS en 127.0.0.1:5432 dentro de su espacio de red.

El wrapper utiliza una imagen local, raíz de lectura, capacidades retiradas, usuario propio y límites de procesos, CPU, memoria y archivos. GrxFirma recibe únicamente el ancla/CRL públicas, su token sintético y su directorio propio. El servidor VEC recibe el material nominal de ejecución; se rechazan claves de CA y claves/PKCS12 de clientes dentro de esa proyección. No se monta el directorio personal del operador.

Para parar, `stop-app` o `stop-grxfirma` cotejan el propietario y la ruta del kit antes de actuar sobre el contenedor. El contenedor y los datos se conservan; no se elimina PostgreSQL.

El lector de navegador reutiliza el pin y la guardia CDP de `propuesta_firma`. Necesita Chrome del sistema, Playwright 1.60.0 y confianza TLS de la CA sintética preparada en el perfil aislado. Debe ejecutarse en el mismo espacio de red del clon.

```sh
python3 scripts/recorridos/firma_r5/preflight_navegador.py \
  --source /ruta/checkout/candidato \
  --certificado /ruta/privada/cliente.pem \
  --clave /ruta/privada/cliente.key \
  --salida /ruta/privada/preflight-navegador.json
```

Comprueba `/livez`, abre Contratación en el portal, observa HTTP, errores JavaScript, cookies, almacenamiento y desbordamiento en 1440/390 px. Bloquea otros orígenes, redirecciones y WebSocket. Nunca pulsa Firmar. Su salida declara `firma_ejecutada=false` y `recibo_acreditado=false`.

El recorrido completo espera los contratos y montaje R5. Debe descargar el original custodiado, conservar referencia/versión/SHA256, firmar resolución en orden con dos cargos nominales, verificar los dos PDF y consultar ambos recibos. Tras reiniciar los mismos servicios, se comparan referencias, fechas, versión, huellas e historia. Un dump lógico se coteja antes de las lecturas que añaden auditoría; los marcadores variables del dump PostgreSQL 18 impiden comparar su hash bruto.
