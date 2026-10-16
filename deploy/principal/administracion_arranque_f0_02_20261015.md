# Arranque de Administración: F0-02

Este corte cierra el montaje heredado de perfiles. No añade SQL, grupos,
cuentas, datos ni actos de administración. Conserva el montaje de usuarios
`metadatos_v1`, que usa auditoría de frontera técnica.

## Preparación y activación

1. Conservar el binario anterior y la configuración privada de la instalación.
2. Comprobar que el proceso separado `vec-admin` tiene
   `VEC_ADMIN_USUARIOS_CONFIG_FILE`, con un JSON privado válido del modo
   `metadatos_v1`. Reutilizar la configuración aprobada de esa copia, sus
   conexiones nominales y su material vigente. No crear el grupo
   `vec_admin_perfiles_auditoria_ejecutor` ni la función heredada para forzar
   otro arranque.
3. Compilar `go build -p 6 -o /ruta/privada/vec-admin ./cmd/vec-admin` sobre la
   cabeza congelada. Reiniciar solo Administración de forma vigilada.
4. Abrir en Chrome la lista de usuarios y una ficha con el certificado nominal
   autorizado. Comprobar respuesta y apunte de auditoría.
5. En una ejecución separada de la copia, quitar únicamente
   `VEC_ADMIN_USUARIOS_CONFIG_FILE`. Debe terminar con
   `etapa=usuarios_config`, antes de leer los demás ficheros privados, abrir
   conexiones o escuchar. Repetir con un fichero JSON privado inválido: debe
   terminar con la misma etapa antes de abrir conexiones.
6. Restaurar la configuración válida y comprobar de nuevo la lista y la ficha.
   El portal de RRHH es otro proceso y conserva su configuración.

La retirada del montaje heredado no habilita funciones adicionales de
Administración. Un fichero de usuarios ausente o inválido nunca selecciona
otra fuente de autorización.

## Vuelta atrás

Conservar la configuración válida y restaurar el binario anterior si el
arranque aprobado falla. No retirar comprobaciones, cambiar permisos ni crear
el grupo inexistente. La instalación en cidonia corresponde a dirección tras
el ensayo de la candidata completa.
