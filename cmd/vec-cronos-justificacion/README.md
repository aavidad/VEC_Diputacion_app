# Ensayo de justificación de permisos

Este programa prepara en memoria el anexo de un justificante y su revisión. Usa dos expedientes inventados: uno termina en «aceptada» y otro en «rechazada». El resultado muestra estados y versiones simulados, junto con un aviso claro. Ninguna operación se registra.

Desde la raíz del repositorio:

```sh
escenario=data/demo/cronos/justificacion-escenario.json
politica=data/demo/cronos/justificacion-politica.json
textos=web/static/textos/es/cronos-justificacion-ensayo.json
go run ./cmd/vec-cronos-justificacion \
  -escenario "$escenario" -escenario-sha256 "$(sha256sum "$escenario" | cut -d ' ' -f 1)" \
  -politica "$politica" -politica-sha256 "$(sha256sum "$politica" | cut -d ' ' -f 1)" \
  -textos "$textos" -textos-sha256 "$(sha256sum "$textos" | cut -d ' ' -f 1)"
```

Para ver el texto inglés, cambie solo la ruta `textos` por `web/static/textos/en/cronos-justificacion-ensayo.json`. El programa comprueba la huella exacta de cada archivo, limita su tamaño y rechaza enlaces y archivos especiales. Si hay un error, devuelve código de salida 2 y un diagnóstico JSON mínimo sin reproducir la entrada.

Es un ensayo sintético sin SQL, HTTP, servicio en ejecución, autorización, custodia ni persistencia. Sus estados y versiones no son recibos ni prueban que una persona haya presentado un documento o que RRHH lo haya revisado.
