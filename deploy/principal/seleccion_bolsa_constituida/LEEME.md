# Selección de Contratación desde una Bolsa constituida

RRHH conserva el botón de selección de la ficha CT. La emisión usa la Bolsa constituida, su orden vigente y el servicio B7 existente. La solicitud debe conservar una necesidad RPT publicada; los expedientes anteriores sin ese vínculo requieren una solicitud nueva.

## Configuración de cidonia

CONFIG NUEVA: `VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH=/vec-incorporacion/cliente_m10_ct.json`.

Dirección crea el archivo privado en el directorio de configuración que ya monta como `/vec-incorporacion`, con propietario del proceso y permisos 0600. No se guarda en Git. El organismo debe coincidir con `personal_b2.organismo_ref` del `servidor.json` activo:

```json
{
  "version": 1,
  "organismo_ref": "org_0e4b1d56ba2e24fb8e4808cbc7abb660",
  "origen": "https://127.0.0.1:8443",
  "directorio": "/vec-material",
  "autoridad_ca": "ca/ca.crt",
  "certificado_cliente": "mtls/cliente.crt",
  "clave_cliente": "mtls/cliente.key",
  "maximo_bytes_pagina": 1048576
}
```

El certificado debe pertenecer al actor RRHH y permitir consultar la organización histórica de Personal. El directorio y las rutas son los del contenedor de la aplicación; no se copian claves al repositorio. Dirección añade la variable al lanzador privado y reinicia cuando haya completado la instalación y la provisión.

La ausencia o el rechazo de esta configuración mantiene cerrado este paso con 503 y permite arrancar el resto de la aplicación.

## Instalación y permisos

Instalar únicamente las migraciones pendientes, en el orden de `deploy/principal/lista_sql_codexk_origen_ct_nativo_20261013.txt`. AD252 y AD259 pueden estar instaladas; no repetirlas ni ejecutar DOWN.

Admitir la fuente completa `deploy/principal/fuentes/bolsa_origen_ct_rrhh_v1.json` y publicar la concesión nominal de `bolsa.origen_ct.consultar`, con sus 38 campos exactos, por la autoridad ADMIN existente. AUT89 no concede permisos al instalarse. Los actos de B7 conservan su decisión V3; la consulta de origen comprueba la sesión vigente y confirma su apunte nominal antes de entregar datos.

Para continuar hacia la aceptación, el recorrido invoca el acto existente `POST /api/vec/contratacion-temporal/expedientes/vinculos-bolsa` después de seleccionar. Aporta `expediente_ref`, la versión CT actual, `bolsa_ref`, `llamamiento_ref`, `recibo_emision_ref` y una `clave_idempotencia` UUID v4 conservada para los reintentos. Este acto comprueba la emisión real de B7; el recibo de selección por sí solo no lo sustituye.

## Comprobación local acreditada

Las migraciones finales se instalaron en una copia PostgreSQL 18.4. El binario normal arrancó con TLS en loopback en 1,47 s. La restauración del dump necesitó reponer las ACL de la base y de los tipos fila; eso pertenece a la copia y no es una migración para cidonia. El kit local no incluye la configuración del portal externo y ese portal permaneció apagado.

La selección nominal, el alta en Personal y el cese desde Chrome siguen pendientes de acreditar. El arranque no demuestra esos efectos ni un envío de correo. Una interrupción después del envío SMTP y antes de registrar su resultado puede repetir el envío al recuperar; se conserva la misma operación local.
