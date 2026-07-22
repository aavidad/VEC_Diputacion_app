# Acceso a la presentación desde un proxy corporativo

## Alcance

Este procedimiento publica únicamente la demostración sintética. No convierte
el artefacto en producción ni habilita identidad, persistencia o datos reales.
El acceso local existente permanece en `127.0.0.1:8081` y se añade una entrada
de borde descartable en una dirección interna concreta.

```text
navegador --HTTPS--> proxy corporativo
                         |
                         | HTTP interno, origen autorizado único
                         v
              IP_SERVIDOR:18081
              entrada-remota-presentacion
                         |
                         | red Docker de borde
                         v
                 proxy-presentacion
                         |
             portal/cartografía/teselas
```

La entrada adicional no se inicia con el perfil `presentacion`: requiere el
perfil explícito `presentacion-remota`. Se enlaza a una IP del servidor, nunca
a `0.0.0.0`, y su Nginx decide por la dirección TCP observada, no por
`X-Forwarded-For`.

## ACL operativa

El repositorio monta `deploy/proxy-local/acceso-remoto-denegar.conf`, que falla
cerrado. Sistemas debe crear fuera del repositorio un fichero de despliegue con
permisos de solo lectura:

```nginx
allow 127.0.0.1;
allow IP_PROXY_AUTORIZADO;
deny all;
```

`127.0.0.1` permite exclusivamente la comprobación de salud dentro del propio
contenedor. La IP concreta del proxy y la topología interna no se incorporan al
repositorio público.

El servidor guarda junto a la ACL un fichero de entorno, también fuera de Git,
que fija todos los valores operativos:

```bash
VEC_PRESENTACION_REMOTE_BIND_ADDRESS=IP_INTERNA_SERVIDOR
VEC_PRESENTACION_REMOTE_PUBLISHED_PORT=18081
VEC_PRESENTACION_REMOTE_ACL_PATH=/RUTA_SEGURA/acceso-remoto.conf
VEC_PRESENTACION_REMOTE_IMAGE=IMAGEN_NGINX_APROBADA_Y_FIJADA
```

El mismo fichero se pasa en todos los comandos para evitar que una recreación
vuelva accidentalmente a los valores locales. En un despliegue offline con
release inmutable:

```bash
docker compose \
  --env-file /RUTA_SEGURA/presentacion-remota.env \
  --project-directory /RUTA_RELEASE \
  -f /RUTA_RELEASE/docker-compose.yml \
  -f /RUTA_RELEASE/compose.offline.yaml \
  --profile presentacion \
  --profile presentacion-remota \
  up --detach --wait
```

La imagen y la ACL tienen valores seguros para desarrollo cuando no se activa
la entrada remota. En el servidor, el fichero anterior es obligatorio y debe
apuntar a una imagen ya cargada y a la ACL aprobada. El servicio conserva
`restart: unless-stopped`.

## Configuración del proxy corporativo

La publicación debe ocupar la raíz de un virtual host HTTPS; no un prefijo como
`/vec/`, porque los activos, las API y las rutas de cotejo son relativas a la
raíz. Configuración Nginx mínima:

```nginx
location / {
    proxy_pass http://IP_INTERNA_SERVIDOR:18081;
    proxy_http_version 1.1;
    proxy_set_header Host 127.0.0.1;
    proxy_set_header Connection "";

    proxy_set_header Authorization "";
    proxy_set_header Proxy-Authorization "";
    proxy_set_header Cookie "";
    proxy_set_header Forwarded "";
    proxy_set_header X-Forwarded-For "";
    proxy_set_header X-Forwarded-Host "";
    proxy_set_header X-Forwarded-Proto "";
    proxy_hide_header Set-Cookie;
}
```

El frontal corporativo termina TLS. La entrada adicional vuelve a vaciar
cookies, credenciales y todas las familias de cabeceras de identidad antes de
acceder al proxy privado. No se debe habilitar `trusted_headers` para la demo.

HTTP entre los dos equipos sólo se admite para esta presentación temporal, sin
datos reales y dentro de una red controlada. Antes de tratar datos reales se
requiere TLS o mTLS entre proxies, identidad aprobada y el despliegue productivo
descrito por Sistemas.

## Cortafuegos y aceptación

Publicar un puerto Docker puede eludir reglas colocadas únicamente en
UFW/INPUT. Sistemas debe autorizar exclusivamente el flujo
`IP_PROXY_AUTORIZADO -> IP_INTERNA_SERVIDOR:18081/TCP` tanto en la ACL de red
como en `DOCKER-USER`/FORWARD, con reglas persistentes. El usuario de despliegue
no modifica el cortafuegos del anfitrión.

La aceptación exige:

1. `ss` muestra el puerto sólo en la IP interna elegida, nunca en `0.0.0.0`.
2. El proxy autorizado obtiene `200` en `/livez` y abre `/presentacion/`.
3. Un origen distinto es rechazado por la ACL de red y por Nginx.
4. Los registros de la entrada muestran como remoto la IP autorizada real.
5. No aparecen `Set-Cookie` y siguen bloqueadas las API privadas.
6. Portal, cartografía, OSRM y teselas superan la prueba rápida existente.

## Retirada

La pieza es descartable y no altera el portal:

```bash
docker compose \
  --env-file /RUTA_SEGURA/presentacion-remota.env \
  --project-directory /RUTA_RELEASE \
  -f /RUTA_RELEASE/docker-compose.yml \
  -f /RUTA_RELEASE/compose.offline.yaml \
  --profile presentacion --profile presentacion-remota \
  stop entrada-remota-presentacion
docker compose \
  --env-file /RUTA_SEGURA/presentacion-remota.env \
  --project-directory /RUTA_RELEASE \
  -f /RUTA_RELEASE/docker-compose.yml \
  -f /RUTA_RELEASE/compose.offline.yaml \
  --profile presentacion --profile presentacion-remota \
  rm --force entrada-remota-presentacion
```

Después se retira la regla de red correspondiente. El acceso privado
`127.0.0.1:8081` y los cinco contenedores de la presentación permanecen sin
cambios.
