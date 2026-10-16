# Confirmar una aceptación desde la ficha de Bolsa

Tarea F1-06, primer corte. Estado: preparado; ensayo y activación pendientes. CONFIG NUEVA: ninguna. La función reutiliza el montaje de Bolsa, la sesión certificada, el servicio de situación y el material nominal común.

## Preparación

1. Partir de una copia PostgreSQL 18 de la principal, con propietarios y ACL completos. Instalar la cadena común congelada por dirección y la política global de #1155 (CAT14, AD283, B122, B126 y B127). No reaplicar SQL instalado ni omitir sus preimágenes.
2. R prepara AD288 con el consumidor `registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada`, diez argumentos de material V3, acción `bolsa.respuesta_revision.resolver`, audiencia `vec_bolsa_llamamientos.respuesta_revision.resolver.v1`, recurso de participación y tipo `respuesta_revision_bolsa`. Registro, vigencia y consumo se cotejan en la transacción de B128. El contrato está en `docs/plan_modulos/bolsa_respuestas_en_revision_contrato.md`.
3. Instalar B128 una sola vez con `lista_sql_codexs_aceptacion_directa_f1_06_20261016.txt`. Añade el acto inmutable y su evento pendiente; el cambio conserva la traza B34. La lista solo contiene B128: la cadena común y #1155 son precondiciones explícitas, no SQL que se deba reinstalar.
4. Admitir la fuente `fuentes/bolsa_confirmacion_aceptacion_rrhh_v1.json` dentro del catálogo completo de Administración. La fuente tiene garantía alta, una acción ordinaria y dimensiones `unidad_ref` y `ambito_ref`; su vigencia de datos es de 90 días. Preparar una propuesta y un cierre con dos administradores distintos mediante el [acto genérico de Bolsa](habilitacion_rol_bolsa_rrhh_generico_20261013.md). Conservar rol, asignación, ámbitos y concesiones anteriores. El archivo de fuente no concede permisos por sí solo.
5. Preparar el origen de consumo nominal para el LOGIN propio de Bolsa, audiencia y operación anteriores, proceso `vec-server` y canal `interna_corporativa`, por el procedimiento técnico vigente. No se crea ese origen al arrancar. Mantener LOGIN, credenciales y material fuera de Git.
6. Con los escritores detenidos, instalar la configuración común y el binario exactos y reiniciar con vigilancia. La adopción conserva la autoridad publicada y sus ámbitos; no publica otra semilla. El proveedor nuevo es opcional: su ausencia deja solo este acto no disponible. Una modificación del rol exige volver a cargar el montaje mediante reinicio; una instantánea revocada u obsoleta debe ser rechazada, nunca dar permiso.

## Un solo recorrido de aceptación

Usar datos ficticios y un llamamiento directo B17 con respuesta B30 `acepta` pendiente de reflejar. Reutilizar la semilla o preparar el dato en la copia, sin atribuirle un acto o una firma reales. Desde Bolsas, abrir la ficha, pulsar «Confirmar aceptación», escribir motivo, revisar y confirmar. Anotar HTTP201, recibo, resolución y paso a pendiente de incorporación. La respuesta original permanece guardada.

Repetir la misma operación: HTTP200 y mismo recibo, motivo y fecha. Cambiar el motivo o la fecha con esa clave debe devolver409 sin otro efecto. Reiniciar aplicación y PostgreSQL; comprobar situación, marca retirada y recuperación del recibo. Otra participación, llamamiento ajeno o permiso revocado deben quedar denegados y auditados. El guion `ensayo_confirmacion_directa_f1_06.sql` comprueba el resultado; no fabrica datos ni permisos.

Comprobar también arranque y consultas existentes cuando falta el consumidor, el origen o el material de este acto. Portal y Bolsa deben seguir disponibles; confirmar aceptación falla de forma aislada. Si falla la vuelta al montaje, restaurar binario y configuración anteriores y reiniciar; conservar las migraciones y la historia. No hay DOWN.

## Límites

Solo aceptación directa. Renuncia simple, oferta publicada y justificación admitida o rechazada esperan sus contratos o la integración de #1147. Una referencia y una huella documentales no acreditan custodia. El evento guardado queda pendiente; no se presenta como aviso enviado ni como incorporación realizada. No hay datos ni servicios corporativos en este ensayo.

Pendientes: AD288 de R, ensayo conjunto, actos de activación y revisiones sobre la cabeza final. Las focales Go y web acreditan el comportamiento de sus componentes; no sustituyen el recorrido de este guion.
