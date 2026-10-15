package ficheros

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"vec-diputacion-granada/internal/vec/ports"
)

// RecuperarOperacionSubidaLocal sólo consulta el índice durable de una
// operación previamente autorizada. La nueva decisión V3 autoriza mirar ese
// índice, pero el recibo devuelto conserva fecha, decisión y efecto originales.
func (a *Almacen) RecuperarOperacionSubidaLocal(ctx context.Context,
	s ports.SolicitudRecuperarOperacionSubidaLocal) (ports.ResultadoOperacionObjeto, bool, error) {
	var vacio ports.ResultadoOperacionObjeto
	if a == nil || ctx == nil || ctx.Err() != nil || s.Validar() != nil {
		return vacio, false, ports.ErrSolicitudAlmacenInvalida
	}
	ahora := a.reloj.Ahora().UTC()
	if s.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, ahora) != nil {
		return vacio, false, ports.ErrAutorizacionAlmacenInvalida
	}
	p, err := s.Contexto.Proyeccion()
	if err != nil || p.OperacionRef == "" || p.CargaRef != p.OperacionRef || p.RecursoRef != p.OperacionRef ||
		(s.Fase == ports.FaseAlmacenSubidaLocalRecuperarEscritura &&
			(p.PasoRef != ports.PasoAlmacenRecuperarEscrituraSubidaLocal || s.ClaveIdempotencia != p.OperacionRef+":escribir")) ||
		(s.Fase == ports.FaseAlmacenSubidaLocalRecuperarPromocion &&
			(p.PasoRef != ports.PasoAlmacenRecuperarPromocionSubidaLocal || s.ClaveIdempotencia != p.OperacionRef+":promover")) {
		return vacio, false, ports.ErrAutorizacionAlmacenInvalida
	}
	efecto, err := ports.HuellaEfectoRecuperacionSubidaLocal(s, p.OperacionRef)
	if err != nil || p.EfectoRef != efecto {
		return vacio, false, ports.ErrAutorizacionAlmacenInvalida
	}
	desbloquear, err := a.bloquear()
	if err != nil {
		return vacio, false, err
	}
	defer desbloquear()
	if ctx.Err() != nil {
		return vacio, false, ctx.Err()
	}
	if s.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, a.reloj.Ahora().UTC()) != nil {
		return vacio, false, ports.ErrAutorizacionAlmacenInvalida
	}
	registro, existe, err := a.cargarIdempotencia(s.ClaveIdempotencia)
	if err != nil || !existe {
		return vacio, false, err
	}
	if registro.ResultadoOriginal == nil {
		return vacio, false, ports.ErrIntegridadObjetoAlmacen
	}
	original := *registro.ResultadoOriginal
	if original.Validar() != nil || original.Evidencia.ReintentoIdempotente ||
		original.Evidencia.OperacionRef != p.OperacionRef || original.Evidencia.CargaRef != p.CargaRef ||
		original.Evidencia.RecursoRef != p.RecursoRef || original.Evidencia.ModuloID != p.ModuloID ||
		original.Evidencia.AccionNegocio != p.AccionNegocio ||
		original.Evidencia.CorrelacionRef != p.CorrelacionRef || original.Evidencia.Finalidad != p.Finalidad ||
		original.Evidencia.Clasificacion != p.Clasificacion ||
		original.Evidencia.SujetoSeudonimoHMAC != p.SujetoSeudonimoHMAC ||
		original.Evidencia.HuellaSolicitudHMAC != p.HuellaSolicitudHMAC ||
		original.Objeto.MIME != s.MIME || original.Objeto.Tamano != s.Tamano ||
		original.Objeto.HuellaSHA256 != s.HuellaSHA256 ||
		original.Objeto.Objeto.Referencia != registro.Referencia || original.Objeto.Objeto.Version != registro.Version {
		return vacio, false, ports.ErrIntegridadObjetoAlmacen
	}
	if s.Fase == ports.FaseAlmacenSubidaLocalRecuperarEscritura {
		if original.Evidencia.Accion != ports.AccionAlmacenEscribir ||
			original.Objeto.Zona != ports.ZonaAlmacenCuarentena || original.Evidencia.FundamentoRef != "" ||
			original.Evidencia.EfectoRef != efectoOperacionSubidaLocal(p.OperacionRef,
				ports.FaseAlmacenSubidaLocalEscribir, ports.ReferenciaObjetoAlmacen{}) {
			return vacio, false, ports.ErrIntegridadObjetoAlmacen
		}
	} else {
		if original.Evidencia.Accion != ports.AccionAlmacenPromover ||
			original.Objeto.Zona != ports.ZonaAlmacenAdmitida ||
			original.Evidencia.FundamentoRef != s.EvidenciaAnalisisRef ||
			original.Evidencia.EfectoRef != efectoOperacionSubidaLocal(p.OperacionRef,
				ports.FaseAlmacenSubidaLocalPromover, s.Origen) || original.Objeto.Objeto == s.Origen {
			return vacio, false, ports.ErrIntegridadObjetoAlmacen
		}
		origen, err := a.cargarMetadatos(s.Origen)
		if err != nil || origen.Zona != string(ports.ZonaAlmacenCuarentena) ||
			origen.HuellaSHA256 != s.HuellaSHA256 || origen.Tamano != s.Tamano || origen.MIME != s.MIME {
			return vacio, false, ports.ErrIntegridadObjetoAlmacen
		}
		if _, err := a.contenidoVerificado(origen); err != nil {
			return vacio, false, err
		}
	}
	meta, err := a.cargarMetadatos(original.Objeto.Objeto)
	if err != nil || meta.Referencia != original.Objeto.Objeto.Referencia ||
		meta.Version != original.Objeto.Objeto.Version || meta.ConectorID != original.Objeto.ConectorID ||
		meta.Zona != string(original.Objeto.Zona) || meta.MIME != original.Objeto.MIME ||
		meta.Tamano != original.Objeto.Tamano || meta.HuellaSHA256 != original.Objeto.HuellaSHA256 ||
		meta.EvidenciaCreacionRef != original.Objeto.EvidenciaCreacionRef ||
		!meta.AlmacenadoEn.Equal(original.Objeto.AlmacenadoEn) || meta.Eliminado || meta.Inmovilizado {
		return vacio, false, ports.ErrIntegridadObjetoAlmacen
	}
	if _, err := a.contenidoVerificado(meta); err != nil {
		return vacio, false, err
	}
	return original, true, nil
}

func efectoOperacionSubidaLocal(id string, fase ports.FaseAlmacenSubidaLocal, obj ports.ReferenciaObjetoAlmacen) string {
	suma := sha256.Sum256([]byte("vec.documentos.subida.local.efecto.v1\x00" + id + "\x00" +
		string(fase) + "\x00" + obj.Referencia + "\x00" + obj.Version))
	return "efecto-carga:sha256:" + hex.EncodeToString(suma[:])
}

var _ ports.RecuperadorOperacionSubidaLocal = (*Almacen)(nil)
