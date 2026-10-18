# Confirmar una aceptación desde la ficha de Bolsa

Tarea F1-71. Estado: preparado; ensayo y activación pendientes. CONFIG NUEVA: ninguna. La función reutiliza el montaje de Bolsa, la sesión certificada, el servicio de situación y el material nominal común.

## Preparación

1. Partir de una copia PostgreSQL 18 de la principal, con propietarios y ACL completos. Usar la lista única `lista_sql_main_tras_hz37.txt`: las 85 migraciones de main tras F1-12, AD288 de R y después B128. No reaplicar SQL instalado ni omitir sus preimágenes.
2. R aporta AD288 congelada en `a32a1d45ba80e5ff5b4ff7a8eaca6608fbaa743d` (#1176) con el consumidor `registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada`, diez argumentos de material V3, acción `bolsa.respuesta_revision.resolver`, audiencia `vec_bolsa_llamamientos.respuesta_revision.resolver.v1`, recurso de participación y tipo `respuesta_revision_bolsa`. Registro, vigencia y consumo se cotejan en la transacción de B128. El contrato está en `docs/plan_modulos/bolsa_respuestas_en_revision_contrato.md`.
3. Instalar B128 una sola vez, después de AD288, desde la lista única de la candidata conjunta. Añade el acto inmutable y su evento pendiente; el cambio conserva la traza B34. Su SHA-256 es `d0f4ecb2f495c8306f197c1541b7fa1356ad9733825e132fbf3806753d596312`.
4. Admitir la fuente `fuentes/bolsa_confirmacion_aceptacion_rrhh_v1.json` dentro del catálogo completo de Administración. La fuente tiene garantía alta, una acción ordinaria y dimensiones `unidad_ref` y `ambito_ref`; su vigencia de datos es de 90 días. Preparar una propuesta y un cierre con dos administradores distintos mediante el [acto genérico de Bolsa](habilitacion_rol_bolsa_rrhh_generico_20261013.md). Conservar rol, asignación, ámbitos y concesiones anteriores. El archivo de fuente no concede permisos por sí solo.
5. Preparar el origen de consumo nominal para el LOGIN propio de Bolsa, audiencia y operación anteriores, proceso `vec-server` y canal `interna_corporativa`, por el procedimiento técnico vigente. No se crea ese origen al arrancar. Mantener LOGIN, credenciales y material fuera de Git.
6. Con los escritores detenidos, instalar la configuración común y el binario exactos y reiniciar con vigilancia. La adopción conserva la autoridad publicada y sus ámbitos; no publica otra semilla. El proveedor nuevo es opcional: su ausencia deja solo este acto no disponible. Una modificación del rol exige volver a cargar el montaje mediante reinicio; una instantánea revocada u obsoleta debe ser rechazada, nunca dar permiso.

## Un solo recorrido de aceptación

Usar datos ficticios y un llamamiento directo B17 con respuesta B30 `acepta` pendiente de reflejar. Reutilizar la semilla o preparar el dato en la copia, sin atribuirle un acto o una firma reales. Desde Bolsas, abrir la ficha, pulsar «Confirmar aceptación», escribir motivo, revisar y confirmar. Anotar HTTP201, recibo, resolución y paso a pendiente de incorporación. La respuesta original permanece guardada.

Si un envío devuelve 401 o 403, la ficha y el listado se retiran. Solo una nueva lectura autorizada permite recuperar la intención con su cuerpo y clave originales. Una caída de red conserva esa intención para consultar o reintentar.

Repetir la misma operación: HTTP200 y mismo recibo, motivo y fecha. Cambiar el motivo o la fecha con esa clave debe devolver409 sin otro efecto. Reiniciar aplicación y PostgreSQL; comprobar situación, marca retirada y recuperación del recibo. Otra participación, llamamiento ajeno o permiso revocado deben quedar denegados y auditados. El guion `ensayo_confirmacion_directa_f1_06.sql` comprueba el resultado; no fabrica datos ni permisos.

Comprobar también arranque y consultas existentes cuando falta el consumidor, el origen o el material de este acto. Portal y Bolsa deben seguir disponibles; confirmar aceptación falla de forma aislada. Si falla la vuelta al montaje, restaurar binario y configuración anteriores y reiniciar; conservar las migraciones y la historia. No hay DOWN.

## Límites

Solo aceptación directa. Renuncia simple, oferta publicada y justificación admitida o rechazada esperan sus contratos o la integración de #1147. Una referencia y una huella documentales no acreditan custodia. El evento guardado queda pendiente; no se presenta como aviso enviado ni como incorporación realizada. No hay datos ni servicios corporativos en este ensayo.

Pendientes: integración de AD288 de R, ensayo conjunto, actos de activación y revisiones sobre la cabeza final. Las focales Go y web acreditan el comportamiento de sus componentes; no sustituyen el recorrido de este guion.
