# Encender inscripción y admisión en Bolsa

Dirección carga estos datos y prepara el acceso para las pantallas «Inscribirme»
y «Mis solicitudes» del aspirante, y «Solicitudes y admisión» de RRHH. Las
pantallas y el núcleo ya están en main (#965/#969/#970/#971/#985/#987).
Este corte aporta su configuración. Dirección revisa el diff y prueba la web.

## Orden de instalación

1. Comprobar B96/B97/B99, CA38, CC11 y BC9 en la historia instalada; no
   reaplicar las migraciones de las listas de inscripción del 10/10.
2. Ejecutar `scripts/operacion/inscripcion_bolsa/preparar_accesos.sh --aplicar`
   con `PGSERVICE` y `PGPASSFILE` privados. Crea sólo los LOGIN ausentes,
   valida su única membresía y no fija contraseñas. El DBA configura éstas por
   el gestor privado de secretos.
3. Cargar las dos SQL de datos de `lista_sql_inscripcion_datos_20261013.txt`
   en el orden indicado. La primera publica los tres motivos V2; la segunda
   reutiliza categorías RPT y publica tres catálogos CC1, dos convocatorias y tres solicitudes, con
   historia y recibos. Las versiones ya existentes deben tener los mismos bytes.
4. Tramitar `inscripcion_bolsa_actos_v1.json` mediante el gobierno ADMIN
   vigente: catálogo, propuesta y cierre con los dos ADMIN. Reutilizar el
   circuito de catálogo del kit B103 de dirección y los dos especializadores existentes
   descritos en `gobierno_inscripcion.md`. Las tres acciones RRHH van por
   B103 y las cinco del aspirante por el circuito de Bolsa externa. Ambos conservan las acciones históricas. No se utiliza la ruta genérica
   de creación de roles para esta ampliación. La fuente contiene cuatro lecturas
   propias, la lectura RRHH de motivos y los actos presentar/decidir/incorporar;
   SHA-256 de `entradas_canon`:
   `07bf028039044c91a23f09a2d5d5fe0d2af254e2fe85aa7fbbacea7c473487fe`.
   Las otras tres lecturas RRHH están en la fuente B103 existente.
   Admitir el catálogo no asigna permisos: conservar los roles, ámbitos y
   asignaciones nominales, avanzándolos por sus actos normales de gobierno.
5. Preparar `identidad/inscripcion-bolsa-rrhh.json` fuera de Git con el
   generador siguiente. Las cuentas y las seis DSN se toman del material RRHH
   vigente, con el mismo certificado, sujeto, cuenta y perfil. El generador
   exige archivos privados propios y guarda el resultado con modo `0600`.

```sh
python3 scripts/operacion/inscripcion_bolsa/preparar_accesos_rrhh.py \
  --material-dir "$VEC_DEVELOPMENT_MATERIAL_DIR" \
  --identidad "$VEC_INSCRIPCION_RRHH_CUENTAS_FILE" \
  --conexiones "$VEC_INSCRIPCION_RRHH_CONEXIONES_FILE" \
  --motivos "$VEC_INSCRIPCION_RRHH_MOTIVOS_FILE"
```

Copiar `motivos_autorizacion_v1.json` al material privado, con modo `0600`,
para la última entrada. Reutiliza el catálogo que exige la presentación
compilada; decidir e incorporar tienen motivos distintos. Las variables
`VEC_INSCRIPCION_RRHH_*_FILE` son argumentos de este guion, no nuevas opciones
que deba interpretar el servidor.

## CONFIG NUEVA

En ambos procesos: `VEC_BOLSA_INSCRIPCIONES_ENABLED=true`.

| Proceso | Variable | LOGIN | Único grupo heredado |
| --- | --- | --- | --- |
| Portal externo | `VEC_EXTERNO_BOLSA_INSCRIPCIONES_LECTOR_DATABASE_URL` | `vec_bolsa_inscripciones_lector` | `vec_bolsa_llamamientos_lector_inscripciones` |
| vec-server RRHH | `VEC_BOLSA_INSCRIPCIONES_RRHH_LECTOR_DATABASE_URL` | `vec_bolsa_inscripciones_rrhh_lector` | `vec_bolsa_llamamientos_lector_inscripciones_rrhh` |
| Portal externo | `VEC_EXTERNO_BOLSA_DATABASE_URL` | `vec_externo_bolsa_desarrollo` | `vec_bolsa_llamamientos_portal_externo` |

Las DSN privadas apuntan a la base principal por TLS y tienen cuentas distintas.
No sustituir los LOGIN por nombres parecidos: las fachadas comprueban
`session_user`. RRHH reutiliza `vec_bolsa_llamamientos_desarrollo` para decidir
e incorporar. Hacen falta el material V3 de las dos audiencias internas y el de
`vec_bolsa_llamamientos.inscripcion.presentar.v1` en el portal externo,
preparados por los comandos existentes de cada raíz. La sesión del aspirante
requiere Área personal/Preferencias activa e identidad certificada.

## Recorrido y dependencia de la incorporación

La convocatoria de Administrativo/a está abierta hasta el 30/11/2026;
la de Operario/a está cerrada. RRHH debe ver una solicitud pendiente, una
admitida y una rechazada, y pulsar el recuento para abrir su lista filtrada.
El aspirante identificado presenta en la abierta y recupera su recibo. RRHH
localiza esa misma solicitud, revisa los requisitos y registra su decisión.
Cada lectura nominal se audita por la función B96 existente.

La incorporación B97 necesita categoría idéntica, acta conservada, participación
y vínculo persona→candidato acreditado. Los datos de esta PR no crean esos
vínculos. El usuario del recorrido «candidato» ya tiene vínculo activo; dirección
conserva su identidad. Debe elegirse su participación en un acta admisible:
la carga no crea otra identidad ni añade integrantes a un acta existente. La política `asociacion.manual.acta` no suple esas condiciones.
La carga de datos tampoco sustituye el futuro botón de publicación RRHH sobre
`ConfirmarPublicacion`.
