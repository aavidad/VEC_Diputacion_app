# Teclado y zoom de Convoca público

Este guion añade una comprobación que no cubre el recorrido integrado de Convoca:
Tab y Shift+Tab, nombres accesibles calculados por Chrome, foco visible sin
obstrucción y zoom **nativo** al 200 %. Consulta listado y ficha pública en
español e inglés a 1440 y 390 px, y a 1440 px con zoom al 200 %: doce superficies.
Reutiliza la auditoría, el ajuste de zoom y las guardas del runner F.

El plan no abre Chrome, lee credenciales ni contacta con un servidor:

```sh
node scripts/recorridos/convoca_accesibilidad/recorrer.mjs --plan
```

Después de instalar y autorizar el servidor público con datos sintéticos,
Dirección crea un JSON privado fuera de Git. Esta es su forma:

```json
{
  "version": 1,
  "sintetico": true,
  "entorno_controlado": true,
  "origen": "https://127.0.0.1:PUERTO",
  "commit_servido": "HASH_COMPLETO_VERIFICADO_POR_DIRECCION",
  "identificador_publico": "IDENTIFICADOR_SINTETICO_DE_LA_PRIMERA_PAGINA"
}
```

El archivo debe ser propio, regular, sin enlaces y privado (0600). La salida
debe ser nueva y su carpeta padre propia y privada (0700). El identificador
debe estar publicado en la primera página del listado. Chrome verifica TLS;
su confianza en la CA pública del servidor debe estar preparada antes.
No se admiten identidades, certificados cliente ni cabeceras de autenticación:
esta superficie es anónima. Use `/usr/bin/google-chrome` y Playwright local.

```sh
export VEC_PLAYWRIGHT_MODULE=/RUTA_PLAYWRIGHT_EXISTENTE/index.mjs
node scripts/recorridos/convoca_accesibilidad/recorrer.mjs \
  --config /RUTA_PRIVADA/convoca.json --salida /RUTA_PRIVADA/SALIDA_NUEVA
```

El guion permite únicamente GET al mismo origen HTTPS de loopback numérico;
rechaza redirecciones, Set-Cookie, WebSocket y otras operaciones. Abre la ficha
con Tab y Enter. Usa el validador V2 montado en la página y exige fuente
gobernada y coincidencia del identificador. No inicia una solicitud ni descarga
documentos. Las lecturas pueden registrar auditoría del servidor.

El zoom usa un perfil temporal propio para la página interna de ajustes de
Chrome. Los contextos de lectura son separados. Verifica ancho CSS, densidad y
escala para distinguir zoom nativo de una emulación. Cierra Chrome y elimina
el perfil al terminar. `resultado.json` conserva contadores, estados y commit
declarado; omite URL, referencias, cuerpos, textos de página y credenciales.
Código 0 indica que terminaron las comprobaciones enumeradas; 1 indica un corte.

Esto no acredita instalación, persistencia, inscripción, firma ni conformidad
WCAG global. No comprueba contraste, orden lógico de lectura, controles con
flechas ni lector de pantalla. El auditor común no incluye `summary`; su
revisión sigue siendo manual. No sustituye el recorrido funcional ya existente
en `scripts/recorridos/convoca_integrado.py` ni el baremador de B.

Para comprobar la herramienta sin servidor, use un entorno aislado sin red:

```sh
node --test scripts/recorridos/convoca_accesibilidad/recorrer.test.mjs
VEC_F_CONVOCA_SALIDA=/RUTA_PRIVADA/NUEVO_ENSAYO \
  node scripts/recorridos/convoca_accesibilidad/recorrer.test.mjs --ensayar-chrome
```

El ensayo entrega bytes sintéticos mediante Playwright, sin arrancar servicios
ni abrir conexiones HTTP. Reutiliza los DTO del fixture de Convoca y el
validador V2 real. El HTML mínimo del ensayo es una fixture de la herramienta;
su resultado no afirma haber recorrido la pantalla VEC ni el servidor instalado.
El informe lo identifica como `prueba_guion` y mantiene esas acreditaciones en
falso. El operador retira su salida después de conservar el resultado necesario.
