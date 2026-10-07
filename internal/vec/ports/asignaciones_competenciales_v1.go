package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrFuenteAsignacionesCompetencialesV1NoDisponible = errors.New("vec: fuente de asignaciones competenciales v1 no disponible")

// LectorAsignacionesCompetencialesV1 resuelve una unica asignacion nominal.
// La implementacion consulta las autoridades de identidad, Personal y RBAC,
// exige autorizacion V3 vigente para esta lectura y audita exito o denegacion.
// Comprueba certificado, cargo, acto/delegacion y cobertura exacta; ausencia,
// ambiguedad o dependencia caida deniega. No crea personas, permisos ni cargos.
// Coteja PersonaRef con la huella EXACTA del certificado del dictamen PAdES,
// no con el certificado mTLS del consultante. PerfilFirmanteRef es el perfil
// esperado del circuito; el perfil activo nominal se resuelve en el servidor,
// sin exigir una sesion online del firmante del PDF. Unidad/Puesto/Ambito y el
// enlace ocupante proceden de Personal, nunca de un mapa libre o del cargo.
// AccionLectura/FinalidadLectura pertenecen al consultante y son distintas de
// AccionCompetencial/FinalidadCompetencial del firmante: no se intercambian.
// El proveedor/PDP comprueba tambien la competencia funcional del firmante,
// restricciones, campos, obligaciones y alcance de su delegacion sobre este
// recurso. Autorizar la lectura del consultante nunca basta para acreditarla.
// La evidencia devuelta no concede acceso ni acredita por si sola una firma.
type LectorAsignacionesCompetencialesV1 interface {
	LeerAsignacionCompetencialV1(context.Context, domain.SolicitudAsignacionCompetencialV1) (domain.EvidenciaAsignacionCompetencialV1, error)
}

// RevalidadorAsignacionesCompetencialesTransaccionV1 solo se implementa sobre
// la misma transaccion durable que aplica el efecto. La composicion liga la
// instancia a esa transaccion: no abre ni confirma una transaccion propia.
// Revalida fuentes actuales, certificado, actor, asignacion, cargo y catalogo
// de motivo y compara las referencias, versiones y huellas de la evidencia
// bajo bloqueo/CAS. Si algo cambio o falta, impide el efecto completo.
// Mantiene la proteccion frente a cese, retirada o revocacion concurrentes
// hasta el COMMIT: releer en READ COMMITTED y liberar la proteccion no basta.
// ComprobadaEn es un reloj de lectura, no parte de la huella idempotente del
// material de firma. El consumidor aplica su propio reloj autoritativo.
// Se invoca tambien en replay junto al consumo de autorizacion V3 actual;
// devolver nil no sustituye ese consumo, auditoria, historia ni recibo.
// El contrato no expone conexiones SQL ni transportes al dominio.
type RevalidadorAsignacionesCompetencialesTransaccionV1 interface {
	RevalidarAsignacionCompetencialV1(context.Context, domain.SolicitudAsignacionCompetencialV1, domain.EvidenciaAsignacionCompetencialV1) error
}
