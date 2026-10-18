# Cabecera después de una fiscalización favorable

F1-74 corrige la indicación de siguiente paso de la ficha de Contratación
temporal. El perfil de RRHH carece de la capacidad de fiscalizar, pero eso no
significa que el resultado esté pendiente. La vista consulta el resultado
explícito del detalle autorizado antes de mostrar que espera a Intervención.

## Activación

No hay SQL ni configuración nuevos. Se instala la aplicación con sus recursos
web de la misma versión. Se renueva la cadena de importadores de la ficha, la
entrada del portal, la precarga y la caché pública; no basta con sustituir un
JavaScript aislado.

Se reutilizan la consulta autorizada del expediente, el adaptador de detalle,
la tarjeta de siguiente paso y los catálogos español e inglés. No cambian el
estado del expediente, sus actuaciones, los permisos ni las rutas de escritura.

## Comprobación en copia local

El caso de aceptación sigue siendo `docs/producto/FLUJO_APP.md`, apartado 4.0,
pasos 8 y 9. El expediente de la semilla CT09 tiene fiscalización favorable en
v6, con su recibo; no se registra otra para comprobar esta corrección.

1. Abrir la ficha con el certificado de RRHH. El detalle autorizado debe incluir
   `fiscalizacion.resultado_clave` favorable o favorable con observaciones, su
   fecha y una versión coherente con la ficha.
2. La tarjeta indica «Siguiente paso: llamamiento» y explica que la fiscalización
   está registrada. La fase administrativa continúa en Fiscalización y el
   historial conserva las seis actuaciones originales.
3. Comprobar la misma lectura a 1440 y 390 píxeles y con ambos idiomas. No se
   ofrece un botón genérico que pueda llevar a otro trámite.
4. Reiniciar aplicación y PostgreSQL. Abrir de nuevo: mismo resultado, versión
   e historia, sin registrar ninguna actuación para recuperar la pantalla.

Las pruebas focales conservan la espera cuando falta el resultado, incluso con
un hito de fiscalización. Un desfavorable no se presenta como favorable. Una
carga, un error, una actualización pendiente o un resultado indeterminado no
anuncian un trámite disponible; un expediente terminado conserva su estado.

Esta corrección no acredita la selección desde Bolsa ni el envío de un aviso.
Sus permisos y dependencias siguen comprobándose en el apartado correspondiente.
