# Cómo migrar un módulo a catálogos de textos

Regla: **ningún texto visible en el código**. Los textos viven en
`web/static/textos/<idioma>/<modulo>.json`; los idiomas, el idioma por defecto
y su localización Intl, en `web/static/textos/idiomas.json`. El código no nombra
idiomas (`"es"`, `"en-GB"`, `…_ES`). Patrón completo: Cronos
(`web/static/portal-empleado/modulos/cronos/`).

## Pasos

1. **Volcar los diccionarios a datos**, una sección por fichero `i18n*.js`
   (las secciones evitan choques de claves entre ficheros):

   ```sh
   node scripts/extraer_textos.mjs <modulo> es \
     general=web/static/portal-empleado/modulos/<modulo>/i18n.js#MENSAJES_<X>_ES \
     otra=web/static/portal-empleado/modulos/<modulo>/i18n-otra.js#MENSAJES_<Y>_ES
   ```

   Repetir con `en` y las constantes `…_EN` si existen. Si el módulo solo tenía
   castellano, crear `textos/en/<modulo>.json` con las **mismas claves y
   variables `{…}`** traducidas al inglés (no copiar el castellano).

2. **Sustituir cada diccionario** por su sección (ruta relativa a `comun/`):

   ```js
   import { cargarTextos } from "../../../comun/textos.js";

   export const MENSAJES_<X> = (await cargarTextos("<modulo>")).seccion("general");
   ```

   Borrar las constantes `…_EN` y las elecciones `IDIOMA_ACTUAL === "en" ? … : …`.
   Los traductores existentes (`crearTraductor…`) se conservan tal cual.

3. **Quitar el idioma de los nombres** en todo `web/static` (código y pruebas):

   ```sh
   grep -rlE 'MENSAJES_<X>_ES\b' web/static | xargs sed -i -E 's/\bMENSAJES_<X>_ES\b/MENSAJES_<X>/g'
   ```

4. **Localización**: sustituir `"es-ES"`/`"en-GB"` por `LOCALIZACION_ACTUAL`
   (`import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";`). Una
   localización solo de cálculo (p. ej. `formatToParts` para AAAA-MM-DD) se
   marca así: `new Intl.DateTimeFormat(/* localización técnica */ "en-CA", …)`.

5. **Manifiestos**: añadir `static/textos/es/<modulo>.json` y
   `static/textos/en/<modulo>.json` a `web/interno.manifest` y
   `web/produccion.manifest` (solo a los que ya incluyen el módulo), **justo
   antes de la primera línea del propio módulo** para no chocar con otras ramas.
   `comun/textos.js`, `textos/idiomas.json`, la ruta `/textos/` del servidor y
   el verificador de JSON ya están preparados.

6. **Versiones de caché**: renovar el `?v=` de todo lo cambiado en su grafo:

   ```sh
   python3 scripts/renovar_versiones_cache.py 20260929-i18n-<modulo>-v1
   ```

   No toca `portal.js` ni `index.html`: si avisa de ellos, dejarlo anotado en
   la PR (se renuevan al integrar). Los JSON no llevan `?v=` (se sirven
   `no-store`).

7. **Lista de legado**: borrar las líneas de los ficheros migrados en
   `web/static/comun/textos-legado.test-helper.mjs`.

8. **Pruebas** (todas en verde):

   ```sh
   node --test $(find web/static -name '*.test.mjs')
   bash scripts/comprobar_tamano_ficheros.sh
   ```

   `textos-catalogos.test.mjs` exige mismas claves y variables por idioma, y
   que la carpeta `portal-empleado/modulos/<modulo>/` no tenga diccionarios,
   idiomas ni localizaciones fijos. Las pruebas que comprobaban textos
   castellanos siguen valiendo: en Node el idioma es el de por defecto.

## Casos especiales

- **Consulta pública de bolsas** (`web/static/bolsa/`): va en
  `web/publico.manifest`; añadir allí también `static/comun/idioma.js`,
  `static/comun/textos.js` y `static/textos/idiomas.json`, y ampliar la lista
  de compartidos permitidos en `scripts/verificar_manifiestos_superficies_web.sh`.
- **Contratación temporal, `portal-i18n*.js`, `portal.js`, `index.html`, Mis
  preferencias**: no migrar hasta que se cierren sus ramas abiertas.

## Añadir un idioma

1. Añadirlo a `web/static/textos/idiomas.json` (`codigo`, `nombre` propio,
   `localizacion` Intl).
2. Copiar `web/static/textos/es/` a `web/static/textos/<codigo>/` y traducir
   los valores (no las claves ni las variables `{…}`).
3. Añadir los nuevos JSON a los manifiestos y pasar las pruebas.

El selector de idioma del portal se rellena solo desde el índice. Mientras
queden módulos sin migrar, esos módulos se verán en el idioma por defecto.
