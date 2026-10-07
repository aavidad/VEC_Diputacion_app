# Accesos de VEC como aplicación web en puestos Windows

Nota para Sistemas · 2 de octubre de 2026

El corte PWA prevé tres entradas distintas. Use la URL HTTPS autorizada de cada entorno, con estos caminos: `/portal-empleado/` para el portal del empleado, `/area-personal/` para el área personal y `/administracion-perfiles/` para la administración. Esta última es una **superficie administrativa privada con mTLS y permiso explícito**; no debe publicarse como parte del portal ordinario ni suponerse accesible por estar instalado un acceso directo. En el entorno de pruebas, su origen previsto es `admin.cidonia.cloud`, sujeto a la configuración de acceso vigente. **Distribuya esta entrada solo después de comprobar que la ruta está publicada en su superficie privada y que el canal mTLS y la autorización funcionan.** La instalación de una aplicación web solo crea una forma de abrir la URL: no identifica al usuario, no concede permisos y no firma documentos.

## Despliegue en Chrome y Edge

En puestos gestionados, configure `WebAppInstallForceList` en la directiva del navegador que se use. Cree una entrada por cada URL que deba recibir ese colectivo. El ejemplo siguiente muestra **solo la estructura**; sustituya `<origen-autorizado>` por el origen HTTPS exacto, sin redirecciones, y no asigne la entrada de administración a puestos sin autorización:

```json
[{"url":"https://<origen-autorizado>/portal-empleado/","default_launch_container":"window","create_desktop_shortcut":true}]
```

Chrome y Edge admiten la clave `url` y la apertura en `window`. La directiva instala la aplicación sin intervención del usuario y puede crear el acceso directo de escritorio en Windows. Google pide el JSON compacto en una sola línea para GPO y recomienda una URL que no redirija. Compruebe la directiva aplicada en `chrome://policy` o `edge://policy` y pruebe el acceso con un usuario de cada ámbito. [Google Chrome Enterprise](https://support.google.com/chrome/a/answer/9367354?hl=es-es) · [Microsoft Edge](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-policies/webappinstallforcelist)

Si se prefiere un acceso directo Windows sin instalación administrada, el destino de Chrome puede usar `chrome.exe --app=https://<origen-autorizado>/portal-empleado/`; cambie solo la URL para las demás entradas. Este modo abre una ventana de aplicación, pero **no registra por sí mismo una PWA administrada**. El ejecutable se localiza en la instalación aprobada del puesto. [Código de Chromium: opción `--app`](https://chromium.googlesource.com/chromium/src/+/refs/tags/137.0.7114.1/chrome/common/chrome_switches.cc)

Para accesos directos creados por Sistemas, use como icono el recurso público de VEC `/pwa/icons/vec.ico` cuando esté desplegado en el origen correspondiente. Verifique primero que responde en ese entorno y copie el `.ico` a una ubicación administrada y estable del puesto o recurso compartido: Windows necesita una ruta de archivo para el icono de un `.lnk`. No introduzca la URL de un entorno en otro.

## Firefox para Windows

Desde Firefox 143 en Windows (150 si procede de Microsoft Store), el usuario puede instalar un sitio como aplicación web desde el botón de la barra de direcciones; Firefox crea una entrada en Inicio y la barra de tareas. Mozilla no documenta actualmente en su [referencia de políticas empresariales](https://firefox-admin-docs.mozilla.org/reference/policies/) una directiva de instalación forzosa de aplicaciones web comparable a `WebAppInstallForceList`. Por tanto, no se propone una política de ese nombre para Firefox. [Mozilla: aplicaciones web en Windows](https://support.mozilla.org/en-US/kb/web-apps-firefox-windows)

Para una distribución centralizada sencilla, use la preferencia **Accesos directos** de GPO para publicar un `.lnk` de tipo «Objeto del sistema de archivos». Indique por separado la ruta de destino `<ruta-aprobada>\firefox.exe` y los argumentos `--new-window "https://<origen-autorizado>/portal-empleado/"`. Adapte el camino y la URL para cada entrada; use el mismo `.ico` local si ya está disponible. El acceso directo abre una ventana normal de Firefox y funciona como alternativa cuando la instalación manual de aplicación web no conviene. [Microsoft: campos de accesos directos de GPO](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-gppref/51d5a9e5-1ea9-44c0-9ede-e686d9012980) · [Mozilla: parámetro `--new-window`](https://firefox-source-docs.mozilla.org/browser/CommandLineParameters.html)

## Certificado y comprobación

En Chrome y Edge de Windows, compruebe que el certificado de cliente autorizado está disponible para el usuario en el almacén de Windows o en su dispositivo criptográfico; en Firefox, compruebe el certificado en el perfil o la carga desde el almacén del sistema mediante `security.osclientcerts.autoload`, según la configuración corporativa. Esta comprobación concierne al **certificado de cliente**: la confianza en la CA del servidor es distinta. [Microsoft: almacenes de Windows](https://learn.microsoft.com/en-us/windows-hardware/drivers/install/certificate-stores) · [Mozilla: certificados de cliente del sistema](https://firefox-admin-docs.mozilla.org/reference/policies/preferences/)

Antes de distribuir, pruebe cada entrada desde un puesto y perfil representativos: apertura de la URL correcta, icono, selección del certificado, respuesta del portal y denegación de administración para una identidad sin concesión. Un acceso directo instalado no acredita por sí solo que la frontera mTLS ni el permiso estén operativos. Esta nota no instala directivas, certificados ni servicios.
