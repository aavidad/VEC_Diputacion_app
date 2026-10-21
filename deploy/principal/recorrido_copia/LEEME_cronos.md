# Cronos de la persona empleada en la copia (F3-01, corte C1)

Enciende la parte de Cronos que ya está en main y en el volcado HZ37: jornada
y saldo, movimientos, comunicar un olvido de fichaje, permisos y su solicitud.
La resolución por jefatura y RRHH, las comunicaciones a RRHH, el calendario
con fuente, los informes y los justificantes son cortes posteriores.

## Orden

Sobre una copia creada con `copia_local.py crear --recorrido-completo` desde
la misma revisión limpia:

```sh
python3 deploy/principal/copia_local_cronos.py --equipo <letra> --instancia <instancia>
```

La orden prepara la etapa y reinicia la aplicación de la copia. Repetirla
coteja lo preparado: no crea otras cuentas, filas ni asignaciones. Cuando
Dietas termine con `copia_local.py` y `conductor.py`, la misma etapa pasará a
llamarse desde `crear --recorrido-completo --cronos` (corte C1b).

## Qué prepara

- Ocho conexiones propias de Cronos, una por campo del fichero privado y con
  un solo grupo cada una (`INHERIT TRUE`, `SET FALSE`, sin administración),
  TLS verificado y 90 días de vigencia.
- Las ocho filas técnicas de origen de consumo (AD172) para la conexión que
  ejecuta Cronos. No conceden permisos de negocio.
- La persona técnica adicional de la copia, Antonio Ruiz Lozano, pasa a ser la
  persona empleada de Cronos. Es una **semilla técnica de copia de desarrollo,
  no gobierno**: un perfil propio, su empleado en Personal, la asignación del
  rol sintético `rol:cronos_empleado_sintetico:v1` limitada a ese empleado, y
  el mismo calendario y jornada sintéticos que la persona empleada del volcado.
  El acto real de Administración y de Personal queda pendiente.
- `material/identidad/cronos-empleado.json` (modo 0600) con su cuenta, las
  conexiones y los motivos del catálogo `motivos_cronos_v1` que publica CRN18.
  El motivo de denegación es una entrada distinta de las demás.
  Ese catálogo es **sintético, de desarrollo**: no se instala en producción
  tal cual. Antes de una instalación real hay que retirarlo o dejar CRN18
  fuera de la lista, y publicar el catálogo que apruebe RRHH. Repetir CRN18
  sobre una base que ya lo tiene no falla: termina sin efectos.
- `VEC_CRONOS_EMPLEADO_ENABLED=true`. La resolución y las notificaciones siguen
  apagadas porque faltan sus migraciones.

El menú del portal no ofrece Cronos (decisión vigente en
`portal-modulos-coordinador.js`). Las pantallas se abren con
`/portal-empleado/#cronos` y `/portal-empleado/#cronos-permisos`.

## Comprobado en una copia local (21/10/2026)

Con el certificado de la persona empleada, en Chrome: jornada, saldo,
movimientos, disponibilidad remota y permisos responden 200; la solicitud de un
día de asuntos propios y la comunicación de un olvido responden 201 con su
recibo. Tras reiniciar la aplicación y PostgreSQL se recuperan la misma
solicitud (versión 1) y el mismo olvido. Otra persona (la técnica de RRHH)
recibe 403 en las rutas de Cronos.
