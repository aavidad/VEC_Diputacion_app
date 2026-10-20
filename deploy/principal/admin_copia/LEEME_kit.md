# Kit y configuración de arranque ADMIN

`kit.py` prepara un kit propio desde una revisión Git limpia. El conductor le pasa la revisión completa, el identificador de sistema (SID) de la copia PostgreSQL, el puerto ADMIN y la entrada publicada del catálogo de unidades. La herramienta vuelve a comprobar `HEAD` antes y después de extraer las fuentes. Solo admite archivos regulares de `cmd`, `internal`, `web`, `config`, `locales`, `pruebas`, `go.mod` y `go.sum`; rechaza enlaces, rutas inseguras y archivos excesivos. Guarda una huella SHA-256 por fuente.

```sh
python3 deploy/principal/admin_copia/kit.py \
  --repo /ruta/candidata --revision REVISION_SHA40 --sid-copia SID_COTEJADO \
  --salida /ruta/privada/kit-nuevo --go /ruta/go \
  --modcache /ruta/home/go/pkg/mod --gocache /dev/shm/vec/go-build \
  --encargo-ref encargo:f1-131 --puerto-admin 18443 \
  --catalogo-entrada-unidad CLAVE_PUBLICADA
```

El modo normal compila sin red los nueve ejecutables del kit y deja `source-manifest.json`, `build-config.json`, `manifiesto-kit.json` y `contrato-copia-local-admin-v1.json`. El contrato identifica revisión, SID, puerto, conjunto 7 y huellas. `--sin-compilar` sirve para comprobar la extracción; produce el estado `incompleto_sin_binarios_ni_actos` y no permite preparar la configuración. El arranque H10 consume cinco CLI ADMIN y `helper-hmac`. La prueba de raíz H, el gobierno de usuarios y `vec-admin` quedan en el kit para el conductor de la copia; este preparador no los ejecuta. El helper sale del literal `HMAC_GO` de `arranque.py` en la misma revisión.

Una vez cotejados el respaldo ADMIN, el material de identidad y la CA PostgreSQL local, se prepara la configuración privada:

```sh
python3 deploy/principal/admin_copia/configuracion.py \
  --kit /ruta/privada/kit-nuevo --revision REVISION_SHA40 --sid-copia SID_COTEJADO \
  --puerto-admin 18443 --instancia ensayo1 --contenedor contenedor-copia \
  --puerto-pg 55441 --backup-admin /ruta/privada/preimagen.dump \
  --material /ruta/privada/material --ca-pg /ruta/privada/ca-pg.pem \
  --catalogo-entrada-unidad CLAVE_PUBLICADA --salida /ruta/privada/arranque-nuevo
```

`configuracion.py` coteja los manifiestos, los nueve binarios, los catálogos que usa H10 y el árbol fuente antes de copiarlo. Genera `config.json`, conexiones privadas, las órdenes locales para fijar las contraseñas de LOGIN ya previstos y `contrato-arranque.json`. No crea aprobaciones externas: las rutas quedan pendientes para el circuito autorizado. Los valores por defecto son 90 días para organización, 89 para personas y unidad, 88 para perfiles y 400 para certificados X.509. `--dias-ca 397` conserva la otra duración admitida; los cuatro plazos de autoridad se pueden ajustar dentro de 60–90 días sin invertir su orden. Los planes técnicos conservan dos horas.

El conductor comprueba el SID real de PostgreSQL y el respaldo antes de llamar a estas herramientas. `guardar_preimagen.py` vuelve a cotejar el SID antes de `pg_dump`. `preparar_fixture.py` solo genera el SQL HZ37 ligado al SID recibido y a las huellas del respaldo HZ37 exacto y de la preimagen ADMIN; la revisión y ejecución del SQL son pasos posteriores. El arranque por etapas usa `arranque.py --config /ruta/privada/arranque-nuevo/config.json --etapa comprobar` y las etapas descritas en [LEEME.md](LEEME.md). Preparar el kit o la configuración no acredita actos, permisos, recorrido en navegador ni reinicio.
