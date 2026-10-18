# Avisos de ofertas por correo

Tarea F1-18, primer corte: publicación, estado y recuperación de avisos pendientes desde «Gestionar ofertas». El ensayo local está terminado. La integración corresponde a dirección. El reintento expreso de un envío sin confirmación queda para el segundo corte; F1-18 sigue abierta hasta entonces.

## Activación

1. Partir de una copia PostgreSQL 18 de la principal, con propietarios y permisos completos y B117 instalada. Detener los escritores propios. No modificar material ni servicios de otro equipo.
2. Instalar una sola vez `lista_sql_codexs_aviso_oferta_f1_18_20261016.txt`: B129 → B131. Ambas cotejan el cuerpo instalado antes de cambiarlo. No hay DOWN ni se reaplica historia. Activar junto con el binario compatible: un publicador anterior no puede confirmar una oferta nueva sin vincular su aviso.
3. Conservar las conexiones y el material HMAC versionado existentes. No se crean perfiles ni secretos. Cada oferta conserva la generación de su token; retirarla antes de terminar los avisos cierra su despacho.
4. Configurar `VEC_BOLSA_AVISO_OFERTA_ENLACE`: URL HTTPS de Mi Bolsa, con ruta `/area-personal/` y sin credenciales. El transporte SMTP y su confianza reutilizan la configuración existente. La plantilla por idioma vive en `config/bolsa_aviso_oferta_v1.json`; incluye categoría, centro, fecha de inicio y enlace. Omite nombre, posición, expediente y descripción libre.
5. Reiniciar con vigilancia. Sin el enlace o con uno inválido, la app arranca y conserva los pendientes; la pantalla indica qué falta. Corregirlo y reiniciar permite continuar desde «Actualizar estado», sin recargar la página. Para volver atrás se conserva un binario compatible con las migraciones: no se revierte SQL ni se borra historia.

El ensayo conjunto usó el código de reglas de #1155 (`e6f55733d`), requerido por el estado S6 de la copia, y el ajuste de política de #1144 (`6741f833b`), que admite el rol vigente mediante concesiones positivas exactas. Ambos tienen sus propias revisiones y los integra dirección. Esta PR no copia esos cambios. El manifiesto privado conserva la procedencia completa: Go de `6a9367d58`, interfaz de `dd4aed444` y SHA-256 del binario `c1e42fb3633c41d1403846ffd72cec5f47228f42aa561234b85b9ee5de6c81f7`.

El ajuste final `d226d00d4` registra el enlace en el inventario de configuración, como exige su prueba existente. Esa prueba focal pasó. El binario con ese metadato (`fb9110ff85cc77b89e3ae7739068f32d20847ec99c13734a35ace93fed28f3f4`) se reinició con enlace válido, ausente e inválido: página, sesión, bolsas, ofertas y salud de proceso respondieron 200. Tras integrar la principal en `10a0064b8`, la nueva composición con S6 y el ajuste R se compiló en el binario `bbc17704837772564dab3b17c62c8b9c58512145dccfa2b36825247d71335579`: los tres arranques repitieron esas respuestas 200 y la recuperación autorizada conservó recibo, fecha y plazo. La lógica propia del aviso mantiene la versión del recorrido descrito.

## Recorrido comprobado

En Chrome del sistema, publicar desde «Gestionar ofertas» de una bolsa constituida con el buzón propio detenido: respuesta 201, recibo y dieciséis avisos pendientes. Se mantiene abierta la confirmación; la intención solo permanece en memoria, sin almacenamiento web.

Después de reiniciar aplicación y PostgreSQL, arrancar el buzón local y pulsar «Reintentar avisos pendientes»: respuesta 200 con el mismo cuerpo, clave, recibo, fecha y plazo. Quince mensajes fueron aceptados; en el restante el buzón cortó la conexión después de DATA y quedó «Sin confirmación». Otro reinicio y la recuperación de esa operación conservaron los dieciséis mensajes y sus dieciséis identificadores únicos: ningún reenvío automático.

Se comprobó también la configuración ausente e inválida. La página, la sesión, las bolsas y la salud de proceso respondieron 200; los avisos permanecieron pendientes. Restaurar el enlace, pulsar «Actualizar estado» y después «Reintentar avisos pendientes» envió los dieciséis de la misma oferta y devolvió el foco al recibo. Se conservaron clave, cuerpo, referencia, fecha y plazo. Las tres ofertas usadas dejaron cuarenta y ocho mensajes y cuarenta y ocho identificadores únicos en total; los cuerpos decodificados contienen el enlace a Mi Bolsa y omiten la descripción libre.

Otra bolsa existente carecía de los ocho contactos de alta: quedaron ocho fallidos, sin llamar al correo. La ausencia conocida de contacto es definitiva; un fallo temporal de consulta conserva el pendiente. No se añadieron contactos ni preferencias para simular un resultado favorable. Se reutilizaron las políticas publicadas por su API y los datos ficticios existentes.

La interfaz final se comprobó en castellano e inglés, a 1440 y 390 píxeles, sin desbordamiento horizontal; el escritorio conserva el desplazamiento en su panel. La ayuda queda plegada tras «?». La recuperación espera la consulta antes de enfocar el recibo; 401/403 retiran los datos y la intención. El registro conserva errores previos de otras superficies y de la conexión durante los reinicios: no se afirma una consola global limpia. La salud conjunta sigue indicando el montaje parcial preexistente.

## Controles y límites

- La publicación autorizada fija destinatarios disponibles y vincula el token en su misma transacción. El correo procede del alta de Bolsa, con esa fuente guardada. La consulta verificada de Usuarios y las personas en activo esperan sus contratos posteriores. Quien queda disponible después puede ver la oferta, pero no se añade a la instantánea de correo.
- La prueba SQL verifica instantánea, vínculo exacto, rechazo de otro actor o token, reclamación antes de la red, resultado y turno tras un fallo temporal. B129 se instaló una vez. El primer POST real revirtió sin oferta ni aviso porque B117 inserta dentro de una subtransacción. B131, instalada una vez, conserva el cierre diferido y coteja el identificador principal guardado al insertar. La regresión con bloque `EXCEPTION` pasó y revirtió sus datos.
- La reclamación confirma la marca incierta antes de la red; la finalización coteja intento y token. Las pruebas focales y la prueba con detección de carreras cubren los cuatro resultados y la recuperación sin duplicar. El presupuesto técnico por petición deja el resto pendiente; no hay otro proceso ni otra bandeja de salida.
- «Enviado» acredita aceptación por el servicio de correo. No acredita entrega ni notificación administrativa, ni cambia el plazo. Un resultado incierto no se reenvía automáticamente. Su reintento expreso, con advertencia de posible duplicado y el mismo identificador de mensaje, corresponde al segundo corte aprobado por dirección.
- Cerrar o recargar la página pierde la clave que el navegador conserva en memoria. El resumen persiste. La recuperación desde el expediente corresponde a F1-16.

Dos revisiones independientes dieron conformidad al cambio sensible y a las correcciones focales posteriores. Semgrep local no encontró hallazgos en lo cambiado. Gosec no señaló líneas nuevas; su ejecución informó de problemas de carga de grupos y avisos en código existente, por lo que no equivale a una auditoría global sin incidencias. La revisión visual independiente y la puerta completa de CI se consignan en la PR.

La definición literal de la proyección instalada ocupa 148 líneas generadas, protegidas por su huella; se excluyen del límite de líneas manuales. Se conserva la cobertura de pruebas.
