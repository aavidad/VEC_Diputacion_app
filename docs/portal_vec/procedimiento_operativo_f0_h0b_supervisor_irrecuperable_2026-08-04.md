# Procedimiento operativo F0-H0b: supervisor o autoridad pidfd irrecuperable

Fecha: 4 de agosto de 2026.

Estado: vinculante para C4b-2. No habilita producción ni sustituye las puertas
automáticas. Su finalidad es impedir una recuperación insegura cuando el
proceso ya no conserva una autoridad pidfd demostrable.

## Ámbito

Se aplica exclusivamente al arnés probatorio F0-H0b en un worker dedicado y
desechable. Se activa ante cualquiera de estos hechos:

- el supervisor Go termina por SIGKILL, abort o fallo sin recibo terminal;
- fallan permanentemente las dos referencias duplicadas del pidfd;
- `PIDFD_SIGNAL_PROCESS_GROUP` deja de funcionar después de autoprueba verde;
- vence `fin_drenaje` tras KILL o queda un adoptado no recolectable;
- se pierde la relación de paternidad, grupo, sesión o evidencia del caso;
- el host, kernel o daemon impide acreditar la postausencia.

No se aplica a errores controlados ordinarios: estos deben cerrar mediante la
máquina terminal, recibo 65 y residuos cero.

## Precondición de ejecución acreditable

Una ejecución que pretenda acreditar H0 debe correr en un worker exclusivo que
pueda reprovisionarse desde una imagen conocida. El sistema de CI conserva por
identificador de trabajo el log completo y el commit probado. Una ejecución
local sin log persistido sirve para diagnóstico, pero no cierra la puerta H0.

El arnés emite a stderr una línea canónica `INCIDENTE_F0_H0B_V1` con datos no
personales: commit, caso, identidad aleatoria, instante UTC, kernel, Go,
contenedor/digest acreditados, última transición, estado de ambos pidfd,
deadline y presencia o ausencia de ACK/recibo. El invocador debe conservar ese
log fuera de `/tmp` y del worker antes de reprovisionarlo.

## Respuesta automática

1. Enclavar incidente 65 y prohibir el caso siguiente.
2. No emitir recibo verde, no afirmar residuos cero y no desarmar la identidad.
3. Escribir y vaciar la línea canónica en el log del job.
4. Preservar `ruta_caso_m38` y la raíz global como cuarentena hasta que el
   sistema de CI haya recogido la evidencia; no borrar una ruta dudosa.
5. Si la identidad Docker continúa acreditada, registrar su inspección. Su
   retirada exacta puede intentarse para reducir daño, pero no convierte el
   worker en reutilizable ni sustituye la reprovisión.
6. Terminar el job y marcar el worker como no programable.

## Recuperación segura

La única recuperación autorizada es reprovisionar o reiniciar el worker
dedicado desde la imagen conocida y volver a ejecutar las puertas completas.
La nueva ejecución debe acreditar antes de H0:

- ausencia de procesos, grupos y contenedores propios del worker anterior;
- ausencia de temporales `vec-f0-h0-*` anteriores;
- imagen, kernel, Go y commit esperados;
- autoprueba ABI y operativa pidfd completamente verdes.

En un worker que no sea desechable, Sistemas debe aislar el host y decidir su
reinicio controlado. No se reanuda el pipeline en ese host por observación
manual de que «parece limpio».

## Acciones prohibidas

- `kill PID`, `kill -PGID`, `pkill`, `killall` o búsqueda por nombre;
- abrir un pidfd nuevo usando un PID conservado en el log;
- adoptar un proceso, grupo, contenedor o temporal por coincidencia parcial;
- borrar la cuarentena antes de recoger evidencia;
- devolver verde, continuar la matriz o reutilizar el worker sin reprovisión;
- registrar secretos, contraseñas, tickets funcionales o datos personales.

Los números PID/PGID del incidente son únicamente evidencia diagnóstica: tras
perder el pidfd podrían haberse reciclado y nunca autorizan una actuación.

## Evidencia mínima de cierre

El incidente queda operativamente cerrado solo con:

1. log canónico conservado fuera del worker y su SHA-256;
2. identificador del job, commit y motivo de cuarentena;
3. constancia de reprovisión/reinicio del worker;
4. inventario posterior limpio;
5. repetición verde de autopruebas y H0 en el worker nuevo.

Hasta reunir los cinco puntos, F0-H0b y producción permanecen en `NO-GO`.
