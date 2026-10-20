# Organización histórica de RRHH con rol publicado por dos administradores

La consulta usa el perfil de la misma cuenta de RRHH, tanto desde Personal
como desde el llamamiento de Contratación. El rol se publica mediante AUT60;
la fábrica no inserta su definición ni su control.

## Activación en la copia local

1. Crear una copia nueva con la lista SQL completa, `--modulos`,
   `--rrhh-original` y `--administracion`. Añadir `--segundo-rrhh` para
   comprobar otra cuenta. En este momento la aplicación arranca sin la
   consulta OH. No se ha creado su rol ni su asignación.
2. Preparar Administración con el kit autorizado y el conjunto 7, según
   [su procedimiento](admin7_roles_nuevos_f1_106_20261020.md). Instalar AD304
   una sola vez, después de las migraciones de la lista principal.
3. Admitir en AUT58 la fuente
   `fuentes/personal_organizacion_historica_rrhh_v1.json`, conservando el
   catálogo y el censo vigentes. Conservar el recibo de la admisión y retirar
   el acceso técnico temporal utilizado para aplicarla.
4. Seleccionar el perfil de Administración de la aplicación. La primera
   persona administradora presenta la propuesta mediante
   `POST /api/admin/perfiles/v1/gobierno-roles/propuestas`. Selecciona la
   entrada admitida `accion:personal.organizacion_historica.consultar`, con
   su versión y huella, y el rol `consulta_organizacion_historica_rrhh`.
   El servidor obtiene la concesión del catálogo; la petición no contiene
   permisos libres. Guardar la referencia y la huella de la propuesta.
5. Otra persona administradora aprueba esa propuesta mediante
   `POST /api/admin/perfiles/v1/gobierno-roles/cierres`. Guardar el recibo
   y repetir con la misma operación y contenido para comprobarlo.
6. Ejecutar `deploy/principal/copia_local.sh organizacion-historica
   --equipo <equipo> --instancia <instancia>`. La orden exige el rol exacto
   y el acto confirmado, prepara los datos ficticios de identidad y
   asignación y monta la consulta con el material de esa copia. Reinicia la
   aplicación. No repetir una preparación interrumpida sin conciliarla.

Los actos se ejecutan con certificados nominales de personas distintas en
las rutas administrativas existentes. Esta entrega no añade un formulario
de propuesta y aprobación.

## Alcance de los datos ficticios

La asignación conserva los ámbitos `organismo_ref` y `unidad_clave` del
consumidor. Solo concede la consulta del organismo del caso 4.0 y del centro
520 durante 89 días. La identidad, los vínculos y esa asignación son datos
de copia declarados como semilla; no acreditan un otorgamiento gobernado.
La asignación por Administración queda pendiente, según la decisión de
dirección del 20/10 a las 11:26. El lote ordinario usa otras dimensiones y
no se presenta como sustituto.

CONFIG NUEVA: ninguna del portal. Se reutilizan el montaje de gobierno de
roles y los selectores OH existentes; se cambia su material con el proceso
parado y se conserva el entorno anterior para volver atrás. Sin publicación
y preparación OH, las demás funciones siguen arrancando.

## Comprobación pendiente

En una copia nueva: ausencia inicial del rol, propuesta y cierre por dos
personas, rechazo del cierre por el proponente, recuperación del mismo
recibo, dos técnicas con 200 en el centro 520 y 403 fuera. Reiniciar la
aplicación, Administración y PostgreSQL y cotejar los mismos actos y datos.
