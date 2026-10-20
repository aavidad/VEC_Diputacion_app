# Selección de Contratación desde una Bolsa constituida

RRHH conserva el botón de selección de la ficha CT. La emisión usa la Bolsa constituida, su orden vigente y el servicio B7 existente. La solicitud debe conservar una necesidad RPT publicada; los expedientes anteriores sin ese vínculo requieren una solicitud nueva.

## Configuración de la copia local

CONFIG NUEVA: ninguna. La selección consulta Organización histórica de Personal por un puerto en el mismo proceso. Activar los selectores existentes `VEC_ORGANIZACION_HISTORICA_CONSULTA_ENABLED=true` y `VEC_ORGANIZACION_HISTORICA_MATERIAL_DIR` según la [guía de Organización histórica](../organizacion_historica_combinado_20261015.md). Cada técnico de RRHH necesita una concesión nominal para consultar los puestos con su propio perfil, organismo y unidad. La cuenta y el perfil corresponden a la persona que inicia la petición.

También debe estar configurada `VEC_RPT_CATALOGO_PATH`, con la ruta del catálogo RPT publicado que ya utiliza el alta. Se conserva su valor actual del lanzador privado; no se sustituye por otro catálogo.

El archivo privado `auditoria-intentos.json` debe indicar `proceso: "vec-rrhh"` y `canal: "interna_corporativa"`. Su LOGIN de auditoría necesita una fila con esos mismos valores en `vec_autorizacion_atestada_v3.configuracion_runtime_intentos`, provisionada por su autoridad existente. Esta tabla conserva historia: si el LOGIN actual tiene otro proceso o canal, Dirección provisiona otro LOGIN y su archivo DSN; no modifica la fila anterior. El lector reutiliza el pool de auditoría de CT.

La ausencia o el rechazo del montaje de Organización histórica mantiene cerrado este paso y permite arrancar el resto de la aplicación. Dirección reinicia la copia tras completar la provisión y el material privado.

## Instalación y permisos

Instalar únicamente las migraciones pendientes, en el orden de `deploy/principal/lista_sql_codexk_origen_ct_nativo_20261013.txt`. AD252 y AD259 ya están instaladas y quedan fuera de la lista. No repetirlas ni ejecutar DOWN.

Admitir la fuente completa `deploy/principal/fuentes/bolsa_origen_ct_rrhh_v1.json` y publicar la concesión nominal de `bolsa.origen_ct.consultar`, con sus 38 campos exactos, por la autoridad ADMIN existente. AUT89 no concede permisos al instalarse. Los actos de B7 conservan su decisión V3; la consulta de origen comprueba la sesión vigente y confirma su apunte nominal antes de entregar datos.

Para continuar hacia la aceptación, el recorrido invoca el acto existente `POST /api/vec/contratacion-temporal/expedientes/vinculos-bolsa` después de seleccionar. Aporta `expediente_ref`, la versión CT actual, `bolsa_ref`, `llamamiento_ref`, `recibo_emision_ref` y una `clave_idempotencia` UUID v4 conservada para los reintentos. Este acto comprueba la emisión real de B7; el recibo de selección por sí solo no lo sustituye.

## Comprobación local acreditada

Las migraciones finales se instalaron en una copia PostgreSQL 18.4. El binario normal arrancó con TLS en loopback en 1,47 s. La restauración del dump necesitó reponer las ACL de la base y de los tipos fila; eso pertenece a la copia y no es una migración para cidonia. El kit local no incluye la configuración del portal externo y ese portal permaneció apagado.

La selección nominal, el alta en Personal y el cese desde Chrome siguen pendientes de acreditar. El arranque no demuestra esos efectos ni un envío de correo. Una interrupción después del envío SMTP y antes de registrar su resultado puede repetir el envío al recuperar; se conserva la misma operación local.

## Petición del centro pendiente

La petición del centro debe elegir la necesidad publicada de la RPT, validada por el lector existente. El formulario actual guarda el puesto como texto libre y la entrega no conserva esa necesidad. Ampliar la presentación, la ratificación y su pantalla corresponde al siguiente corte de K, según dirección del 13/10/2026 a las 21:05; véase dudas162. El guion completo #1070 conserva `FALTA` antes de crear otro expediente.
