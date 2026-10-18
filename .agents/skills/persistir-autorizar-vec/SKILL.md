---
name: persistir-autorizar-vec
description: Implementa SQL e identidad/autorización de VEC conservando historia, permisos y recibos. Usar al modificar migraciones, funciones PostgreSQL, transacciones o concesiones V3.
---

# Persistir y autorizar VEC

Las reglas están en `docs/producto/ESQUEMA_BACKEND.md` (base de datos) y
`docs/producto/TRD.md` (identidad, autorización y efectos). Esta skill es el orden de
trabajo. Inspeccionar migraciones, funciones, roles y puertos existentes antes de editar.
Un archivo SQL confirmado en Git no demuestra que la base lo haya recibido.

1. Identificar la operación nominal: datos de entrada, autoridad, escritura, lectura y
   recuperación. Actor y permisos salen de la frontera confiable
   (`docs/producto/TRD.md` §5 Identidad y sesión, §6 Autorización).
2. Escribir el efecto completo en una transacción con idempotencia y versión esperada:
   `docs/producto/TRD.md` §7 y `docs/producto/ESQUEMA_BACKEND.md` §5 Contrato de
   transacción y §6 Historia, versiones e idempotencia.
3. Si hace falta SQL nuevo, reservar el número y escribir la migración con su cabecera y
   guarda de preimagen; nunca reaplicar ni `DOWN` sobre historia:
   `docs/producto/ESQUEMA_BACKEND.md` §9.2, §9.3 y §9.5. Roles, ACL y RLS: §4.
4. Consumir la decisión V3 antes de tocar filas, también al recuperar una operación:
   `docs/producto/ESQUEMA_BACKEND.md` §7.1 y `docs/producto/TRD.md` §6.
5. Conservar publicaciones, definiciones, huellas y referencias originales: una clave
   publicada no cambia de significado (`docs/producto/TRD.md` §6, «Perfiles y
   concesiones»; `docs/producto/ESQUEMA_BACKEND.md` §9.3, «Reconstruir una función
   instalada»).
6. Ensayar en el clon, pedir las revisiones independientes que tocan (dos si afecta a
   identidad, autorización o criptografía) sobre el mismo contenido y corregir con el
   menor parche que resuelva la causa: `docs/producto/ESQUEMA_BACKEND.md` §9.6, skills
   `ensayar-sql` y `revisar-sql-vec`, agente `revisor-sql-vec`.
7. Entregar migración y funciones afectadas, precondiciones, dependencias, validación y
   recuperación prevista. Instala Dirección por el canal autorizado y comprueba el recibo
   (`docs/producto/ESQUEMA_BACKEND.md` §9.6, paso 4).

Datos sintéticos y nada de secretos en registros, ejemplos ni documentación:
`docs/producto/TRD.md` §8 y §11. Respetar los permisos recibidos también al delegar: una
operación denegada no se envía a otro agente o canal para eludir su control.
