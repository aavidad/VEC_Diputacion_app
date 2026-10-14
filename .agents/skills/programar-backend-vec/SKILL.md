---
name: programar-backend-vec
description: Implementa incrementos del backend Go de VEC reutilizando dominio, aplicación, puertos y adaptadores. Usar para expedientes, seguimiento, API o composición de contratación temporal.
---

# Programar backend VEC

Partir del recorrido que necesita RRHH y buscar primero su implementación en el índice
de código (codebase-memory); rg queda para texto y configuración.
Inspeccionar internal/modules/contrataciontemporal/{domain,application,ports,adapters}
y la composición de internal/app. Distinguir un servicio existente de una ruta montada.

Reglas que se aplican y dónde están:

- capas y dependencias, propiedad de los datos (Bolsa y Personal son dueños de sus
  agregados; Contratación temporal no los escribe): `docs/producto/TRD.md` §2 y
  `docs/producto/ESQUEMA_BACKEND.md` §3 y §8.2;
- recibos, versión esperada, idempotencia y recuperación con la operación original:
  `docs/producto/TRD.md` §7 y `docs/producto/ESQUEMA_BACKEND.md` §6;
- anotación administrativa y cambio de estado son operaciones distintas:
  `docs/producto/ESQUEMA_BACKEND.md` §6;
- puertos y adaptadores PostgreSQL, API, errores y recibos HTTP:
  `docs/producto/ESQUEMA_BACKEND.md` §10 y §11.

Pasos:

1. Identificar contrato, propietario de datos, punto de montaje y archivos asignados.
2. Poner las decisiones en dominio o aplicación y los detalles HTTP o SQL en adaptadores.
3. Reutilizar servicios existentes. Extender una pieza con un parche acotado antes de
   introducir otra variante equivalente. Conectar las APIs que consume la interfaz.
4. Conservar referencias, versiones esperadas, recibos y errores nominales. Una consulta
   del seguimiento original no prueba cuál es su estado actual: releer antes de decidir.
5. Ejecutar las pruebas focales del comportamiento modificado y entregar contrato,
   archivos y punto exacto de composición al responsable de integración.

Coordinar con el propietario SQL o de identidad antes de modificar persistencia o
autorización (skill `persistir-autorizar-vec`). Aplicar $orquestar-vec para encargar
trabajo independiente sin invadir sus archivos.
