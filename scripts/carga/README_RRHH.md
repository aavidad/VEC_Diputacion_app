# Medir las lecturas internas de Bolsa y CT

El cliente de `scripts/carga/cliente` admite las consultas internas HTTPS con
certificado de cliente. Usa el mismo banco que el portal público: cada usuario
conserva su conexión y recorre los casos en orden. Solo conecta a loopback,
verifica el certificado del servidor y no sigue redirecciones.

Antes de medir, prepara una copia **local** PostgreSQL 18 con datos sintéticos.
El contenedor debe llevar `--rm --restart=no --memory 2g`, con PGDATA en disco.
No uses las credenciales ni la dirección de cidonia. El banco no instala SQL,
no concede permisos y no crea expedientes ni aspirantes.

Reutiliza CT187 y las pruebas de Bolsa global B94 para comprobar el volumen y
las consultas. El acta de CT del 8 de octubre conserva una medida HTTPS con
5.071 expedientes; el laboratorio público genera 84.000 posiciones. Esas
cargas no cubren por sí solas todas las lecturas internas ni todos los estados.
La muestra del banco debe tener miles de expedientes y aspirantes, con fases
variadas, plazos vencidos, urgencias, incidencias y filtros sin resultados.
Registra en el acta los recuentos por estado, el commit del servidor, las SQL
instaladas, los recursos asignados y los EXPLAIN de las consultas lentas.

## Casos y ejecución

Guarda fuera de Git un JSON como este, con referencias de la copia local:

```json
[
  {
    "nombre": "cuadro-ct-100",
    "metodo": "POST",
    "ruta": "/api/vec/contratacion-temporal/cuadro/consultas",
    "cuerpo": {"filtros": {"texto": ""}, "paginacion": {"limite": 100}},
    "presupuesto_ms": 300
  },
  {
    "nombre": "bolsas",
    "metodo": "GET",
    "ruta": "/api/vec/bolsa/bolsas",
    "presupuesto_ms": 300
  },
  {
    "nombre": "sesion",
    "metodo": "GET",
    "ruta": "/api/vec/session",
    "presupuesto_ms": 100
  }
]
```

Añade la lista CT, ficha CT, candidatos y ficha del candidato, módulos y
preferencias con sus rutas y contratos vigentes. Para POST solo se admiten
`/api/vec/contratacion-temporal/cuadro/consultas` y
`/api/vec/contratacion-temporal/expedientes/consultas`. Copia los cuerpos de
lectura que usa el navegador de la copia local. Si el contrato necesita una
sesión previa, prepárala con el circuito existente; el cliente no la inventa.

```sh
GOCACHE=$HOME/.cache/go-build go run -p 6 ./scripts/carga/cliente \
  -url https://localhost:18551 \
  -ca /ruta/privada/ca.crt \
  -cert /ruta/privada/cliente.crt -key /ruta/privada/cliente.key \
  -casos /ruta/privada/lecturas.json \
  -usuarios 1 -muestras 30 -calentamientos 3 -pausa 0s
```

La salida es una tabla Markdown p50/p95/p99 por caso. Los percentiles usan
nearest rank y solo incluyen HTTP 200 cuyo cuerpo se haya recibido completo.
Los errores, incluidos los del calentamiento, quedan contados por código o
clase; no se imprimen cuerpos, certificados ni claves. Si no hay éxitos,
la tabla muestra `NaN`, no una latencia cero. El proceso termina con código 1
si hay errores, faltan muestras o p95 alcanza el presupuesto indicado.
El código 2 indica una configuración inválida.

Compara también límites 1/50/100 para detectar consultas por fila. Para una
segunda serie con concurrencia usa, por ejemplo, `-usuarios 5,20 -muestras 30`;
cada usuario hace 30 vueltas. La misma identidad representa concurrencia de
transporte, no veinte identidades distintas. Las lecturas pueden añadir
asientos a la auditoría real: comprueba el incremento y el número de consultas
por petición en la copia local. No atribuyas esa evidencia a cidonia.

Los argumentos antiguos de rutas GET del laboratorio público siguen
funcionando. `-muestras 0` conserva el modo de carga por `-duracion`.

## Reutiliza

- `scripts/carga/cliente/main.go`: conexión por usuario, tanda y percentiles.
- `scripts/carga/laboratorio_publico.sh`: volumen público y PostgreSQL aislado.
- `deploy/postgresql/contratacion_temporal/pruebas_sql/ct187_rendimiento_listas.sql`:
  paridad y 30 muestras SQL de página, totales, resumen y peticiones.
- `internal/app/bootstrap/bolsa_rrhh_global_test.go`:
  `TestBolsaGlobalHTTPP95Clon`, lectura global y páginas con 2.000 participaciones.
- `docs/portal_vec/pruebas_rendimiento_cuadro_ct_20261008.md`: acta HTTPS previa.

El protocolo de 3 calentamientos, 30 muestras y mTLS también conserva el del
arnés privado de CT/Bolsa de Codex-S del 7 de octubre. Sus rutas privadas y
material criptográfico quedan fuera del repositorio.

CONFIG NUEVA: ninguna para la aplicación. Los argumentos pertenecen solo al
cliente local de medida.
