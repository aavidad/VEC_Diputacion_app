# Operación local de VEC

Corte del 20/10/2026, contrastado con `origin/main@3ef7dee0b`.
El trabajo y los recorridos se hacen en copias locales propias. cidonia está
congelada; una futura instalación en un servidor nuevo corresponde a dirección.
Este procedimiento no autoriza cambiar servicios compartidos.

La receta detallada está en
[LEEME_copia_local.md](../../deploy/principal/LEEME_copia_local.md), y su
implementación en [copia_local.py](../../deploy/principal/copia_local.py).
Antes de usarla, compruebe las opciones de esa misma revisión:

```sh
python3 deploy/principal/copia_local.py --help
```

## Preparar una instancia propia

Se necesitan Docker, Python 3, OpenSSL, las herramientas Go y sus dependencias
ya disponibles, y las imágenes locales indicadas en la receta. El creador no
descarga imágenes ni herramientas para compilar. PostgreSQL es 18; el guion
usa `postgres:18.4`. Para módulos y correo local requiere también Mailpit.
El navegador es el Chrome del sistema, manejado con Playwright.

Use una rama propia, confirmada en Git y con el árbol limpio. El guion conserva
la revisión, las huellas y la ruta fuente: cambiar de revisión y arrancar la
misma instancia no es una actualización admitida; se crea otra instancia.
Reserve puertos propios y un nombre que no use otro equipo.

Compruebe `free -g`, `df -h /dev/shm` y `df -h /home`. No trabaje
con menos de 10 GB libres en disco, según la orden del 20/10. Con menos de 40 GB disponibles,
retire copias y temporales propios sin uso y avise a dirección; el sistema
conserva 32 GB. La guarda del creador rechaza menos de 34 GiB: ese umbral del
código no sustituye la orden de avisar por debajo de 40 GB.
No se borra ni se para una copia ajena.

El material de referencia se obtiene de dirección y se lee sin modificarlo.
`VEC_COPIA_ORIGEN` apunta a la carpeta que contiene el volcado y sus anexos de
permisos de base y tipos. Sus huellas deben corresponder a la misma referencia.
Nunca se publican ese material, contraseñas, conexiones ni certificados en Git.

## Crear, comprobar y recuperar

Desde la raíz de la carpeta de trabajo, con los puertos previamente reservados
para esta instancia:

```sh
export VEC_COPIA_ORIGEN=/ruta/privada/referencia
python3 deploy/principal/copia_local.py crear --equipo d --instancia operacion \
  --puerto-pg 55491 --puerto-app 18491 \
  --lista-sql deploy/principal/lista_sql_main_tras_hz37.txt
python3 deploy/principal/copia_local.py comprobar --equipo d --instancia operacion
```

Sustituya la letra, el nombre, los puertos y la ruta privada por los asignados.
`crear` restaura, instala la lista una vez, genera material propio, compila y
arranca. Sin `--modulos`, comprueba un arranque con las funciones opcionales
apagadas; no prepara por sí sola el recorrido de Contratación y Bolsa.
La lista es explícita: no suponga que omitirla instala los cambios de main.

Para Contratación y Bolsa, la receta añade `--modulos` y un puerto SMTP
propio. Requiere los productores privados de roles y datos identificados por
`VEC_COPIA_ROLES_SCRIPT` y `VEC_COPIA_DATOS_SCRIPT`.
`--portal-externo` requiere `--modulos` y usa el preparador del aspirante
versionado en `deploy/principal/portal_copia/`. `--sin-inscripcion` mantiene
la inscripción apagada. `--segundo-rrhh` prepara otra persona técnica.
`--administracion` requiere su productor privado y puerto propio.
No copie un kit armado ni ejecute material privado ajeno: use las fuentes
versionadas y la referencia que dirección haya autorizado para su copia.

Las órdenes de ciclo de vida conservan los puertos y las marcas del estado:

```sh
python3 deploy/principal/copia_local.py parar --equipo d --instancia operacion
python3 deploy/principal/copia_local.py arrancar --equipo d --instancia operacion
python3 deploy/principal/copia_local.py reiniciar --equipo d --instancia operacion
python3 deploy/principal/copia_local.py comprobar --equipo d --instancia operacion
```

`parar` detiene los procesos propios y PostgreSQL. El contenedor usa `--rm`;
los datos de su volumen permanecen en `/dev/shm/vec/pg/d-operacion/pgdata`.
`arrancar` recrea el contenedor y comprueba revisión, binario y configuración.
`reiniciar` hace parada y arranque de aplicación y base, y de los componentes
propios preparados, como correo, Administración o verificador de firmas.
No edite a mano un fichero sellado para evitar una comprobación de huella.

Si falla la preparación, conserve los registros privados de esa instancia,
comunique el error y siga la recuperación de su receta. Una copia incompleta
no se marca como preparada ni se reutiliza con una revisión distinta.
No reaplique migraciones instaladas: para otro ensayo extraiga una copia nueva.

## Firma y Personal

Estas órdenes necesitan una copia ya creada con módulos y material privado
propio, preparado según los apartados correspondientes de la receta:

```sh
python3 deploy/principal/copia_local.py firma --equipo d --instancia operacion \
  --manifiesto-firma /ruta/privada/manifiesto-firma.json
python3 deploy/principal/copia_local.py personal --equipo d --instancia operacion \
  --config-personal /ruta/privada/servidor-personal.json
```

El manifiesto de firma describe sus etapas y el verificador con su huella.
No basta con generar certificados: hacen falta cargos, asignaciones,
admisiones, plan y verificación. Personal prepara sus conexiones, catálogos y
capacidades gobernadas; no crea una incorporación, alta o cese por SQL.
Las fuentes de preparación que sigan pendientes de integración no se dan
por disponibles en main. Compruebe el tablero y la receta de la revisión usada.

## Qué acredita la copia

Un portal que responde 200 acredita el arranque. Para acreditar una actuación
se comprueban su petición, resultado, recibo y auditoría. Para acreditar
persistencia se repiten las lecturas después de reiniciar aplicación y
PostgreSQL y se cotejan recibos, fechas y ausencia de duplicados.
El caso único y sus variantes están en
[FLUJO_APP §4.0](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).
El hito fijado por dirección el 20/10 pide un expediente completo con reinicio
antes del 6 de noviembre; si no se alcanza, se recalcula la fecha de fin de
fase 1, prevista para la segunda quincena de noviembre. El cierre exige además
las condiciones del [plan](../producto/PLAN_IMPLEMENTACION.md).

Todo dato es ficticio. Las semillas restauran o preparan datos y autoridad de
desarrollo; no prueban una aprobación oficial ni un acto de Administración.
El inventario «Lo que la copia siembra y no es autoritativo» corresponde a
FLUJO_APP §6.7, preparado en la revisión del tablero del 20/10; mientras esa
revisión no esté integrada, consulte su cambio pendiente y el código de la
copia usada. No se presenta como estado ya integrado.

Las semillas pueden incluir entradas iniciales de organización y unidad,
asignaciones ficticias, ámbitos y organizaciones legados, publicaciones de
bases y admisiones técnicas externas. Algunas se están sustituyendo por vías
gobernadas. El inventario debe distinguir lo retirado en main de lo pendiente:
no se mantiene aquí una segunda tabla de tareas. La semilla de Plantilla,
plaza y cobertura tampoco acredita una vacante real.

Un recibo local no confirma el correo corporativo, GINPIX ni Firmadoc. La
configuración y las reglas aún pendientes de RRHH usan valores provisionales
configurables; no se presentan como una decisión institucional definitiva.

## Guardar evidencia y borrar

Guiones, fuentes y evidencia necesaria se conservan en la carpeta de estado
propia en disco; PGDATA y carpetas de trabajo desechables van en `/dev/shm/vec`.
La memoria se pierde al apagar el equipo. Haga commit y push de los documentos
y guiones sin secretos al menos cada 30 minutos mientras haya trabajo nuevo,
y deje ESTADO en el canal. Una cabeza congelada esperando CI o revisión solo
necesita ese ESTADO, según la aclaración de dirección del 20/10 a las 12:22.

Cuando ya no necesite su instancia y haya guardado su evidencia:

```sh
python3 deploy/principal/copia_local.py borrar --equipo d --instancia operacion
```

La orden para los procesos, valida propiedad y destino y borra los datos de
esa copia. No toca el respaldo de referencia. Las ramas y carpetas de trabajo
se retiran después de que dirección integre, con árbol limpio y sin usuarios.

## Preparar la siguiente instalación

El servidor nuevo necesita una candidata identificada por revisión y huellas,
SQL en el orden de su lista, configuración, actos de Administración y datos
preparados. Antes de instalar se ensaya el conjunto completo en una copia:
arranque con y sin la configuración nueva, caso 4.0, variantes y reinicio.
Se conserva un guion de activación y vuelta atrás del mismo conjunto.

Dirección e Informática deben concretar alojamiento, certificados, secretos,
conexiones e integraciones. No se trasladan las credenciales ficticias de una
copia como identidad corporativa. Los requisitos de datos reales, seguridad,
accesibilidad y operación siguen en [TRD §17](../producto/TRD.md#17-lo-que-falta-para-producción).
Este documento no declara listo el servidor ni describe pasos para cidonia.

## Guías temáticas

- [Seguir una petición lenta](observabilidad_tecnica.md).
- [Registros técnicos](registros_tecnicos.md).
- [Verificar auditoría](verificar_auditoria.md).
- [Auditar intentos nominales](auditoria_intentos_nominales.md).
- [Validar paquetes de colores](temas_paquetes.md).
- [Selectores de CT: informe histórico del 13/10](codexh_selectores_ct_20261013.md).
