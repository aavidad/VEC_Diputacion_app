package gobiernoreglasbaremo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"time"
	vd "vec-diputacion-granada/internal/vec/domain"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioGobiernoV3 struct {
	repositorio ports.RepositorioGobiernoReglasBaremoV3
	consulta    ports.ConsultaGobiernoReglasBaremoV3
	proveedor   ports.ProveedorMaterialGobiernoReglasV3
	ahoraUTC    func() time.Time
}

func NuevoServicioGobiernoV3(r ports.RepositorioGobiernoReglasBaremoV3, c ports.ConsultaGobiernoReglasBaremoV3, p ports.ProveedorMaterialGobiernoReglasV3, reloj func() time.Time) (*ServicioGobiernoV3, error) {
	if dependenciaCatalogoGobiernoV3Nula(r) || dependenciaCatalogoGobiernoV3Nula(c) || dependenciaCatalogoGobiernoV3Nula(p) || reloj == nil {
		return nil, ErrGobiernoV3NoDisponible
	}
	return &ServicioGobiernoV3{r, c, p, reloj}, nil
}

func (s *ServicioGobiernoV3) instante(ctx context.Context, c CredencialesGobiernoV3) (time.Time, error) {
	if s == nil || s.ahoraUTC == nil || ctx == nil || ctx.Err() != nil {
		return time.Time{}, ErrGobiernoV3NoDisponible
	}
	ahora := s.ahoraUTC().UTC().Truncate(time.Microsecond)
	if !instanteUTCMicrosegundos(ahora) {
		return time.Time{}, ErrGobiernoV3NoDisponible
	}
	if err := c.validar(ahora); err != nil {
		return time.Time{}, err
	}
	return ahora, nil
}

func (s *ServicioGobiernoV3) proveer(ctx context.Context, c CredencialesGobiernoV3, solicitud ports.SolicitudMaterialGobiernoReglasV3, ahora time.Time) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	// Copias para impedir que el broker altere el material que después se persiste.
	copia := solicitud
	copia.Campos = append([]string{}, solicitud.Campos...)
	copia.MaterialCanonico = bytes.Clone(solicitud.MaterialCanonico)
	copia.Recurso = clonarRecurso(solicitud.Recurso)
	v3, err := s.proveedor.ProveerMaterialGobiernoReglasV3(ctx, c.vinculo, copia)
	if err != nil {
		if errors.Is(err, ErrGobiernoV3Prohibido) || errors.Is(err, ErrGobiernoV3NoAutenticado) {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
		}
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrGobiernoV3NoDisponible
	}
	ahora = s.ahoraUTC().UTC().Truncate(time.Microsecond)
	if c.validar(ahora) != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrGobiernoV3NoAutenticado
	}
	resumen := v3.ResumenCapacidad()
	huella, eh := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	motivo, em := c.actor.RepresentacionCanonicaVinculadaV2()
	if v3.ValidarEstructura() != nil || eh != nil || em != nil || resumen.Operacion() != solicitud.Accion || resumen.EfectoRef() != solicitud.Recurso.Referencia || resumen.EfectoHuellaSHA256() != huella || resumen.AudienciaConsumo() != solicitud.Audiencia ||
		ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) || v3.PersonaVersion() != c.actor.Instantanea.PersonaVersion || v3.PerfilVersion() != c.actor.Instantanea.PerfilVersion || !bytes.Equal(v3.ContextoActorCanonico(), motivo) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrGobiernoV3NoDisponible
	}
	// El motivo del agregado/consulta debe coincidir con el atestado por V3.
	if shaGobiernoV3(v3.MotivoCanonico()) != resumen.MotivoHuellaSHA256() {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrGobiernoV3NoDisponible
	}
	esperado, err := motivoCanonicoGobiernoV3(solicitud.Motivo)
	if err != nil || !bytes.Equal(esperado, v3.MotivoCanonico()) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrGobiernoV3NoDisponible
	}
	return v3, nil
}

func (s *ServicioGobiernoV3) GuardarAltaBorrador(ctx context.Context, c CredencialesGobiernoV3, p PeticionAltaBorradorV3) (ports.ResultadoAltaBorradorReglasV3, error) {
	var vacio ports.ResultadoAltaBorradorReglasV3
	ahora, err := s.instante(ctx, c)
	if err != nil {
		return vacio, err
	}
	huella, err := huellaNegocioAltaV3(c.actor.PersonaRef, p)
	if err != nil {
		return vacio, err
	}
	version, err := reglas.NuevaVersionGobernadaReglasBaremo(p.Conjunto, c.actor.PersonaRef, p.Motivo, ahora)
	if err != nil {
		return vacio, ErrGobiernoV3PeticionInvalida
	}
	canon, sha, estado, err := canonizarVersionResultado(version)
	if err != nil {
		return vacio, ErrGobiernoV3PeticionInvalida
	}
	motivo, err := motivoGobiernoV3(p.Motivo)
	if err != nil {
		return vacio, err
	}
	selector := ports.SelectorGobiernoReglasV3{Identidad: p.Conjunto.Identidad(), Estado: estado}
	_, solicitud, err := prepararMaterialGobiernoV3(c, selector, operacionAltaGobiernoV3, motivo, canon, p.ClaveOperacion, huella, ahora)
	if err != nil {
		return vacio, err
	}
	v3, err := s.proveer(ctx, c, solicitud, ahora)
	if err != nil {
		return vacio, err
	}
	orden := ports.OrdenAltaBorradorReglasV3{MaterialCanonico: bytes.Clone(solicitud.MaterialCanonico), HuellaMaterialSHA256: shaGobiernoV3(solicitud.MaterialCanonico), VersionCanonica: bytes.Clone(canon), HuellaVersionSHA256: sha, EstadoPropuesto: estado, ClaveOperacion: p.ClaveOperacion, HuellaSolicitudSHA256: huella, Autorizacion: v3}
	resultado, err := s.repositorio.ConfirmarAltaBorrador(ctx, orden)
	if err != nil {
		return vacio, err
	}
	if validarAccesoGobiernoV3(resultado.Acceso, v3, s.ahoraUTC().UTC().Truncate(time.Microsecond)) != nil {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	original, err := validarReciboGobiernoV3(resultado.Recibo, s.ahoraUTC().UTC().Truncate(time.Microsecond))
	if err != nil || resultado.Recibo.ClaveOperacion != p.ClaveOperacion || resultado.Recibo.HuellaSolicitudSHA256 != huella || original.CreadaPor() != c.actor.PersonaRef {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	conjunto, err := original.Conjunto()
	if err != nil {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	estable, err := huellaNegocioAltaV3(original.CreadaPor(), PeticionAltaBorradorV3{Conjunto: conjunto, Motivo: original.MotivoCreacion(), ClaveOperacion: p.ClaveOperacion})
	if err != nil || estable != huella {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if !resultado.Replay {
		if !bytes.Equal(resultado.Recibo.VersionCanonica, canon) || resultado.Recibo.ConsumoOriginal != resultado.Acceso {
			return vacio, ports.ErrConfirmacionReglasBaremoInvalida
		}
	} else if resultado.Acceso.DecisionRef == resultado.Recibo.ConsumoOriginal.DecisionRef || resultado.Acceso.AuditoriaRef == resultado.Recibo.AuditoriaRef {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	resultado.Recibo.VersionCanonica = bytes.Clone(resultado.Recibo.VersionCanonica)
	return resultado, nil
}

func (s *ServicioGobiernoV3) ConsultarExacta(ctx context.Context, c CredencialesGobiernoV3, p PeticionConsultaExactaV3) (ports.ResultadoConsultaGobiernoReglasV3, error) {
	var vacio ports.ResultadoConsultaGobiernoReglasV3
	orden, v3, _, err := s.ordenConsulta(ctx, c, p.Selector, p.Motivo, operacionConsultaGobiernoV3, "", "")
	if err != nil {
		return vacio, err
	}
	resultado, err := s.consulta.ObtenerExacta(ctx, orden)
	if err != nil {
		return vacio, err
	}
	if validarAccesoGobiernoV3(resultado.Acceso, v3, s.ahoraUTC().UTC().Truncate(time.Microsecond)) != nil || validarVersionSelectorGobiernoV3(resultado.VersionCanonica, p.Selector) != nil {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	resultado.VersionCanonica = bytes.Clone(resultado.VersionCanonica)
	return resultado, nil
}

func (s *ServicioGobiernoV3) RecuperarRecibo(ctx context.Context, c CredencialesGobiernoV3, p PeticionRecuperarReciboV3) (ports.ResultadoRecuperacionGobiernoReglasV3, error) {
	var vacio ports.ResultadoRecuperacionGobiernoReglasV3
	if !claveGobiernoV3Valida(p.ClaveOperacion) || !huellaSHA256Valida(p.HuellaSolicitudSHA256) {
		return vacio, ErrGobiernoV3PeticionInvalida
	}
	orden, v3, _, err := s.ordenConsulta(ctx, c, p.Selector, p.Motivo, operacionRecuperarGobiernoV3, p.ClaveOperacion, p.HuellaSolicitudSHA256)
	if err != nil {
		return vacio, err
	}
	resultado, err := s.consulta.RecuperarRecibo(ctx, orden)
	if err != nil {
		return vacio, err
	}
	if validarAccesoGobiernoV3(resultado.Acceso, v3, s.ahoraUTC().UTC().Truncate(time.Microsecond)) != nil {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if !resultado.Existe {
		if resultado.Recibo.ReciboRef != "" || len(resultado.Recibo.VersionCanonica) != 0 {
			return vacio, ports.ErrConfirmacionReglasBaremoInvalida
		}
		return ports.ResultadoRecuperacionGobiernoReglasV3{Acceso: resultado.Acceso}, nil
	}
	original, err := validarReciboGobiernoV3(resultado.Recibo, s.ahoraUTC().UTC().Truncate(time.Microsecond))
	if err != nil || validarVersionSelectorGobiernoV3(resultado.Recibo.VersionCanonica, p.Selector) != nil || resultado.Recibo.ClaveOperacion != p.ClaveOperacion || resultado.Recibo.HuellaSolicitudSHA256 != p.HuellaSolicitudSHA256 || resultado.Acceso.DecisionRef == resultado.Recibo.ConsumoOriginal.DecisionRef {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	conjunto, err := original.Conjunto()
	if err != nil {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	h, err := huellaNegocioAltaV3(original.CreadaPor(), PeticionAltaBorradorV3{Conjunto: conjunto, Motivo: original.MotivoCreacion(), ClaveOperacion: p.ClaveOperacion})
	if err != nil || h != p.HuellaSolicitudSHA256 {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	resultado.Recibo.VersionCanonica = bytes.Clone(resultado.Recibo.VersionCanonica)
	return resultado, nil
}

func (s *ServicioGobiernoV3) ordenConsulta(ctx context.Context, c CredencialesGobiernoV3, selector ports.SelectorGobiernoReglasV3, motivo vd.ReferenciaEntradaCatalogo, operacion, clave, huella string) (ports.OrdenConsultaGobiernoReglasV3, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, time.Time, error) {
	var vacio ports.OrdenConsultaGobiernoReglasV3
	var sinV3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	ahora, err := s.instante(ctx, c)
	if err != nil {
		return vacio, sinV3, time.Time{}, err
	}
	_, solicitud, err := prepararMaterialGobiernoV3(c, selector, operacion, motivo, nil, clave, huella, ahora)
	if err != nil {
		return vacio, sinV3, time.Time{}, err
	}
	v3, err := s.proveer(ctx, c, solicitud, ahora)
	if err != nil {
		return vacio, sinV3, time.Time{}, err
	}
	return ports.OrdenConsultaGobiernoReglasV3{Selector: selector, MaterialCanonico: bytes.Clone(solicitud.MaterialCanonico), HuellaMaterialSHA256: shaGobiernoV3(solicitud.MaterialCanonico), ClaveOperacion: clave, HuellaSolicitudSHA256: huella, Autorizacion: v3}, v3, ahora, nil
}

func validarAccesoGobiernoV3(a ports.EvidenciaAccesoGobiernoReglasV3, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) error {
	r := v3.ResumenCapacidad()
	if !evidenciaGobiernoV3Valida(a) || a.DecisionRef != r.DecisionRef() || a.DecisionHuellaSHA256 != r.DecisionHuellaSHA256() || a.EfectoRef != r.EfectoRef() || a.EfectoHuellaSHA256 != r.EfectoHuellaSHA256() || a.ConsumidaEn.Before(r.EmitidaEn()) || !a.ConsumidaEn.Before(r.ExpiraEn()) || a.ConsumidaEn.After(ahora) {
		return ports.ErrConfirmacionReglasBaremoInvalida
	}
	return nil
}
func evidenciaGobiernoV3Valida(a ports.EvidenciaAccesoGobiernoReglasV3) bool {
	return referenciaReciboGobiernoV3(a.DecisionRef) && huellaSHA256Valida(a.DecisionHuellaSHA256) && referenciaReciboGobiernoV3(a.EfectoRef) && huellaSHA256Valida(a.EfectoHuellaSHA256) && huellaSHA256Valida(a.ConsumoHuellaSHA256) && referenciaReciboGobiernoV3(a.AuditoriaRef) && instanteUTCMicrosegundos(a.ConsumidaEn)
}
func validarReciboGobiernoV3(r ports.ReciboAltaBorradorReglasV3, ahora time.Time) (reglas.VersionGobernadaReglasBaremo, error) {
	var vacio reglas.VersionGobernadaReglasBaremo
	if !referenciaReciboGobiernoV3(r.ReciboRef) || !referenciaReciboGobiernoV3(r.TransaccionRef) || !referenciaReciboGobiernoV3(r.AuditoriaRef) || !referenciaReciboGobiernoV3(r.OutboxRef) || !claveGobiernoV3Valida(r.ClaveOperacion) || !huellaSHA256Valida(r.HuellaSolicitudSHA256) || r.Estado.Validar() != nil || !evidenciaGobiernoV3Valida(r.ConsumoOriginal) || r.AuditoriaRef != r.ConsumoOriginal.AuditoriaRef || !instanteUTCMicrosegundos(r.ConfirmadaEn) || r.ConfirmadaEn.After(ahora) || r.ConfirmadaEn.Before(r.ConsumoOriginal.ConsumidaEn) {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	v, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(r.VersionCanonica, r.Estado.HuellaEstadoSHA256())
	if err != nil || v.Estado() != reglas.EstadoReglasBaremoBorrador || v.Revision() != 1 || v.CreadaEn().After(r.ConfirmadaEn) {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	estado, err := v.VinculoEstado()
	if err != nil || !estado.CoincideExactamenteCon(r.Estado) {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	return v, nil
}
func validarVersionSelectorGobiernoV3(canon []byte, s ports.SelectorGobiernoReglasV3) error {
	v, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(canon, s.Estado.HuellaEstadoSHA256())
	if err != nil {
		return ports.ErrConfirmacionReglasBaremoInvalida
	}
	estado, err := v.VinculoEstado()
	conjunto, ec := v.Conjunto()
	if err != nil || ec != nil || !estado.CoincideExactamenteCon(s.Estado) || v.Estado() != reglas.EstadoReglasBaremoBorrador {
		return ports.ErrConfirmacionReglasBaremoInvalida
	}
	a, n := conjunto.Identidad(), s.Identidad
	if a.Referencia() != n.Referencia() || a.Version() != n.Version() || a.ConvocatoriaRef() != n.ConvocatoriaRef() || a.ExpedienteRef() != n.ExpedienteRef() {
		return ports.ErrConfirmacionReglasBaremoInvalida
	}
	return nil
}
func shaGobiernoV3(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func dependenciaCatalogoGobiernoV3Nula(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}
func motivoCanonicoGobiernoV3(m vd.ReferenciaEntradaCatalogo) ([]byte, error) {
	return vd.RepresentacionCanonicaMotivoAutorizacionV2(m)
}
