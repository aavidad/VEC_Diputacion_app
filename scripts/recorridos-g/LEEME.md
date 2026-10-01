# Recorridos preparados de Dietas

`--plan` muestra diez escenarios y sus dependencias. No abre Chrome, no lee
credenciales ni contacta con VEC:

```sh
node scripts/recorridos-g/recorrer.mjs --plan
```

La matriz procede de la ficha D1–D9 y del montaje actual de Dietas. Los ocho
escenarios ejecutables requieren un servidor sintético autorizado, AUT27
instalada, una relación acreditada por Personal, permisos nominales, las
migraciones de Dietas necesarias y el módulo montado. La ruta de Personal debe
responder con al menos una relación válida. Las decisiones exigen además que
la fuente de competencias acredite la etapa exacta. El informe PDF D9 y el
catálogo competente de rectificación quedan identificados como pendientes de
contrato o montaje: no se pueden escoger para ejecutar. La tabla de tarifas
actual es provisional; una cifra calculada no acredita una liquidación real.

El guion utiliza las comprobaciones de rutas privadas y origen del runner F.
Chrome usa `/usr/bin/google-chrome` y Playwright local, sin descargar otro
navegador. Solo acepta HTTPS con dirección numérica de loopback, certificado
cliente sintético y validación TLS. Los archivos de configuración, certificado,
clave y resultados se guardan fuera de Git. La carpeta de salida debe ser nueva
y su padre privado. No use datos ni credenciales reales.

La configuración privada contiene esta forma. Dirección sustituye los
marcadores por valores sintéticos ya autorizados y acredita cada dependencia
antes de marcarla `true`:

```json
{
  "version": 1,
  "sintetico": true,
  "entorno_controlado": true,
  "commit_servido": "HASH_COMPLETO_DEL_SERVIDOR",
  "origen": "https://127.0.0.1:PUERTO",
  "idioma": "es",
  "dependencias": {
    "AUT27": true,
    "Personal": true,
    "Dietas_SQL": true,
    "Montaje": true,
    "OSRM": true,
    "Tarifas": true,
    "Competencias": true,
    "Criterio_retorno_RRHH": true
  },
  "identidades": {
    "empleado": { "certificado": "/RUTA_PRIVADA/empleado.crt", "clave": "/RUTA_PRIVADA/empleado.key" },
    "administrativo": { "certificado": "/RUTA_PRIVADA/administrativo.crt", "clave": "/RUTA_PRIVADA/administrativo.key" },
    "responsable": { "certificado": "/RUTA_PRIVADA/responsable.crt", "clave": "/RUTA_PRIVADA/responsable.key" },
    "rrhh": { "certificado": "/RUTA_PRIVADA/rrhh.crt", "clave": "/RUTA_PRIVADA/rrhh.key" },
    "intervencion": { "certificado": "/RUTA_PRIVADA/intervencion.crt", "clave": "/RUTA_PRIVADA/intervencion.key" }
  },
  "casos": {
    "empleado_solicitud": {
      "pasos": [
        { "tipo": "click", "selector": "[data-dietas-abrir-nueva-comision]" },
        { "tipo": "fill", "selector": "[name=\"fecha_inicio\"]", "valor": "AAAA-MM-DD" },
        { "tipo": "fill", "selector": "[name=\"hora_inicio\"]", "valor": "HH:MM" },
        { "tipo": "fill", "selector": "[name=\"fecha_fin\"]", "valor": "AAAA-MM-DD" },
        { "tipo": "fill", "selector": "[name=\"hora_fin\"]", "valor": "HH:MM" },
        { "tipo": "fill", "selector": "[name=\"motivo\"]", "valor": "Motivo sintético" },
        { "tipo": "select", "selector": "[name=\"origen_codigo\"]", "valor": "CODIGO_CATALOGO" },
        { "tipo": "select", "selector": "[name=\"destino_codigo\"]", "valor": "OTRO_CODIGO_CATALOGO" },
        { "tipo": "ruta", "selector": "[data-dietas-calcular-ruta]" },
        { "tipo": "efecto", "selector": "[data-dietas-borrador-guardar]", "metodo": "POST", "ruta": "/api/vec/dietas/comisiones", "estado_esperado": "borrador", "modo": "nuevo" }
      ]
    }
  }
}
```

Cada caso lleva pasos propios en ese JSON privado. Los pasos admitidos son
`visible`, `click`, `fill`, `select`, `ruta` y `efecto`. `ruta` comprueba la
respuesta del mediador OSRM; `efecto` pulsa un botón y exige
la petición, recibo, versión y estado declarados. Por defecto `modo` es
`nuevo`: exige HTTP 201 y `repeticion:false`. Solo `modo:"recuperacion"`
acepta HTTP 200 y `repeticion:true`, para recuperar una intención sintética
conservada con la misma clave. Cada paso admite una sola petición de negocio;
una segunda se bloquea antes de salir a la red y corta el resultado. En las
decisiones del circuito, el paso declara `etapa` y `decision:"aprobar"`; el
cuerpo HTTP debe coincidir con ambos. Revisión, autorización, liquidación y
fiscalización comprueban exclusivamente su siguiente estado de aprobación. La
devolución no se ejecuta con estos cuatro casos: `retorno_reenvio` parte de
una devolución sintética preparada y autorizada fuera de este guion. El guion
solo admite los efectos de `casos.json` durante el paso que los espera. Las
referencias de una comisión existente se toman de un escenario sintético
preparado y recuperable; no se generan datos nuevos para superar un fallo. La
secuencia de retorno exige primero `PUT` con estado `borrador` y después `POST
…/enviar` con estado `enviado_pendiente_revision`.

```sh
export VEC_PLAYWRIGHT_MODULE=/RUTA_LOCAL/playwright/index.mjs
node scripts/recorridos-g/recorrer.mjs \
  --config /RUTA_PRIVADA/dietas.json --caso empleado_solicitud \
  --salida /RUTA_PRIVADA/dietas-solicitud-antes --escrituras-sinteticas si
```

Cada caso se ejecuta una sola vez por intención preparada. El guion visita
1440 y 390 px: escribe solo en escritorio y vuelve a abrir la vista en móvil.
Comprueba idioma, ausencia de desbordamiento global, errores JavaScript,
cookies, almacenamiento web, respuestas API fallidas y peticiones externas.
`resultado.json` guarda las referencias opacas de comisión y recibo, fecha
exacta, versión, repetición, estado y huella en un archivo privado `0600`.
No guarda cuerpos HTTP, nombres ni certificados. Código `0` indica que
**ese** caso sintético terminó; `2`, que no arrancó por entradas o puertas;
`1`, que Chrome empezó y el recorrido se cortó. `servidor_instalado_verificado`
y `reinicio_verificado` permanecen `false`: Dirección contrasta después el
recibo, la historia y la ausencia de duplicados tras reiniciar aplicación y
PostgreSQL. No se afirma firma, pago ni conformidad global de accesibilidad.

Prueba focal sin servidor ni credenciales:

```sh
node --test scripts/recorridos-g/recorrer.test.mjs
semgrep --config p/javascript --metrics off --disable-version-check scripts/recorridos-g
```

Este LEEME prepara la ejecución técnica; el manual de RRHH corresponde al
corte integrado y recorrido por Dirección.
