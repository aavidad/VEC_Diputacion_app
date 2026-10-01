# Ejemplo PDF del saldo de Cronos

Desde la raíz del repositorio:

```sh
TMPDIR=/tmp GOCACHE=/dev/shm/go-build go run ./cmd/vec-cronos-informe-saldo \
  web/static/textos/es/cronos-informe-saldo.json \
  internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json > /tmp/cronos-saldo-ejemplo.pdf
```

El PDF muestra el saldo de Carmen Molina Ortega, persona sintética. Lleva una
marca visible de ejemplo y conserva «No disponible» cuando falta un valor.
El programa exige `demo: true` y emite el documento completo después de validar
la salida del renderer común. Un fallo previo deja stdout vacío.

C12 está preparado, pendiente de autoridad productiva. La CLI no usa el caso de
uso `ExportarSaldoPropio`, no consulta servicios ni confirma auditoría. Ese caso
exige una fuente nominal propia y un registro durable que revalide el permiso
de exportación y confirme su consumo y auditoría antes de devolver bytes. No
hay adaptador de esas autoridades, ruta HTTP ni descarga habilitada.

El catálogo inglés queda preparado. El renderer común fija `es-ES` en el PDF;
si el catálogo pide otro idioma, la preparación falla sin devolver bytes.
Falta extender el renderer común para seleccionar idioma y validar PDF/UA.
La muestra permite comprobar legibilidad; no acredita accesibilidad PDF/UA,
instalación, permiso, auditoría durable, firma ni validez administrativa.

Gosec G304 y G703 se justifican en las dos aperturas de ficheros: las rutas
las elige el operador local de esta CLI, con sus propios permisos. No hay
entrada HTTP, cuenta de servicio ni acceso a un servidor. El contenido debe
cumplir el esquema cerrado y estar marcado como sintético antes de emitir bytes.
