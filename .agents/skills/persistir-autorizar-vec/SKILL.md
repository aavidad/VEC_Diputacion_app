---
name: persistir-autorizar-vec
description: Implementa SQL e identidad/autorización de VEC conservando historia, permisos y recibos. Usar al modificar migraciones, funciones PostgreSQL, transacciones o concesiones V3.
---

# Persistir y autorizar VEC

Inspeccionar migraciones, funciones, roles y puertos existentes antes de editar.
Separar preparación de código, revisión e instalación efectiva: un archivo SQL
confirmado en Git no demuestra que la base lo haya recibido.

1. Identificar la operación nominal y sus datos de entrada, autoridad, escritura,
   lectura y recuperación. Resolver actor y permisos desde la frontera confiable.
2. Mantener juntas las escrituras que constituyen el efecto: versión, actuación,
   recibo, auditoría y outbox según el contrato existente. Conservar idempotencia.
3. Añadir una migración nueva cuando proceda. No modificar una migración instalada,
   reaplicarla ni ejecutar DOWN sobre el historial conservado.
4. Verificar las precondiciones y la autorización vigente también en recuperación.
   Una concesión histórica o una referencia opaca por sí solas no conceden acceso.
5. Preservar publicaciones, definiciones, huellas y referencias originales. No cambiar
   una definición inmutable para hacer que una transición nueva parezca autorizada.
6. Preparar la comprobación focal y pedir dos revisiones independientes sobre el mismo
   contenido. Corregir los hallazgos con el menor parche que resuelva su causa.
7. Entregar migración/funciones afectadas, precondiciones, dependencias, validación y
   recuperación prevista. Instalar sólo por el canal autorizado y comprobar el recibo.

Usar datos sintéticos. No imprimir credenciales, material de sesión o secretos en
logs, ejemplos o documentación. Respetar los permisos recibidos también al delegar;
una operación denegada no se envía a otro agente o canal para eludir su control.
