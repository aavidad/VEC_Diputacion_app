package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type DependenciasOriginalSubidaLocalRRHH struct {
	Registro       docports.RegistroOriginalSubidaLocalRRHH
	Autoridad      docports.AutoridadActoSubidaLocalRRHHV3
	Almacen        vecports.AlmacenObjetos
	Seudonimizador vecports.SeudonimizadorSujetoAlmacen
	Reloj          vecports.Reloj
}

type ServicioOriginalSubidaLocalRRHH struct {
	d DependenciasOriginalSubidaLocalRRHH
}

func NuevoServicioOriginalSubidaLocalRRHH(d DependenciasOriginalSubidaLocalRRHH) (*ServicioOriginalSubidaLocalRRHH, error) {
	if dependenciaNula(d.Registro) || dependenciaNula(d.Autoridad) || dependenciaNula(d.Almacen) ||
		dependenciaNula(d.Seudonimizador) || dependenciaNula(d.Reloj) {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	return &ServicioOriginalSubidaLocalRRHH{d: d}, nil
}

func (s *ServicioOriginalSubidaLocalRRHH) AbrirOriginalRRHH(ctx context.Context,
	q docports.SolicitudOriginalSubidaLocalRRHH) (io.ReadCloser, error) {
	if s == nil || ctx == nil || ctx.Err() != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	actor, err := q.ActorRegistradoV2.Clonar()
	if err != nil || q.VinculoV2.ValidarPara(actor) != nil {
		return nil, docports.ErrSubidaLocalDenegada
	}
	q.ActorRegistradoV2 = actor
	pre, err := docports.PreimagenConsultaSubidaLocalRRHH(q)
	if err != nil {
		return nil, err
	}
	autorizacion, err := s.d.Autoridad.AutorizarConsultaRRHH(ctx, pre, q)
	if err != nil {
		return nil, err
	}
	if autorizacion.Accion != docports.AccionBolsaJustificanteRenunciaDescargarRRHH ||
		autorizacion.CargaID != q.DocumentoID || autorizacion.RecursoRef != q.RecursoRef ||
		autorizacion.Finalidad != "custodiar_justificante_renuncia_bolsa" ||
		autorizacion.CorrelacionRef != q.CorrelacionRef {
		return nil, docports.ErrSubidaLocalDenegada
	}
	i, err := s.d.Registro.ConsultarAdmitidoRRHH(ctx, docports.ConsultaPersistenciaSubidaLocalRRHH{
		SolicitudOriginalSubidaLocalRRHH: q, AutorizacionV3: autorizacion})
	if err != nil {
		return nil, err
	}
	if !instantaneaOriginalRRHHValida(i, q) {
		return nil, docports.ErrSubidaLocalDenegada
	}
	obj := i.Admitido.Objeto.Objeto
	peticionSeudonimo, err := vecports.NuevaSolicitudSeudonimizarSujetoAlmacen(i.PersonaRef, "subida_local:"+i.Resultado.CargaID)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	seudonimo, err := s.d.Seudonimizador.SeudonimizarSujetoAlmacen(ctx, peticionSeudonimo)
	if err != nil || !strings.HasPrefix(seudonimo, "hmac-sha256:") {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	v := vecports.VinculosOperacionAlmacen{OperacionRef: i.Resultado.CargaID,
		CargaRef: i.Resultado.CargaID, Clasificacion: i.Clasificacion,
		SujetoSeudonimoHMAC: seudonimo, HuellaSolicitudHMAC: i.HuellaSolicitudHMAC,
		EfectoRef:       efectoSubidaLocal(i.Resultado.CargaID, vecports.FaseAlmacenSubidaLocalDescargar, obj),
		ObjetoVinculado: obj}
	contexto, err := s.d.Autoridad.AutorizarAlmacenRRHH(ctx, q, i, v)
	if err != nil {
		return nil, err
	}
	return abrirBytesOriginalSubidaLocal(ctx, s.d.Almacen, s.d.Reloj, contexto, obj,
		i.HuellaPDFSHA256, i.Tamano)
}

func instantaneaOriginalRRHHValida(i docports.InstantaneaSubidaLocal,
	q docports.SolicitudOriginalSubidaLocalRRHH) bool {
	if i.Resultado.CargaID == "" || (q.CargaID != "" && i.Resultado.CargaID != q.CargaID) ||
		i.Resultado.ComprobanteRef != q.DocumentoID || i.HuellaPDFSHA256 != q.HuellaPDFSHA256 ||
		i.ModuloID != "bolsa" || i.TipoRecurso != "justificante_renuncia" ||
		i.Finalidad != "custodiar_justificante_renuncia_bolsa" ||
		i.RecursoRef != q.RecursoRef || i.PersonaRef != q.PersonaTitularRef ||
		i.Resultado.Estado != docports.EstadoSubidaLocalAdmitida ||
		i.Resultado.DocumentoVersion < 1 || i.Resultado.ReciboRef == "" ||
		i.Resultado.RegistradoEn.IsZero() || i.Version < 5 ||
		i.Clasificacion == "" || i.HuellaSolicitudHMAC == "" ||
		i.MIME != "application/pdf" || i.Tamano < 1 || i.Tamano > limitePDFSubidaLocal ||
		i.Analisis == nil || i.Analisis.Validar() != nil ||
		i.Analisis.Estado != vecports.EstadoAnalisisContenidoLimpio ||
		i.Analisis.HuellaObjetoSHA256 != i.HuellaPDFSHA256 ||
		i.Admitido == nil || i.Admitido.Validar() != nil ||
		i.Admitido.Objeto.Zona != vecports.ZonaAlmacenAdmitida ||
		i.Admitido.Objeto.HuellaSHA256 != i.HuellaPDFSHA256 ||
		i.Admitido.Objeto.Tamano != i.Tamano || i.Admitido.Objeto.MIME != i.MIME {
		return false
	}
	_, err := hex.DecodeString(i.HuellaPDFSHA256)
	return len(i.HuellaPDFSHA256) == sha256.Size*2 && err == nil
}

func abrirBytesOriginalSubidaLocal(ctx context.Context, almacen vecports.AlmacenObjetos,
	reloj vecports.Reloj, contexto vecports.ContextoOperacionAlmacen,
	obj vecports.ReferenciaObjetoAlmacen, huella string, tamano int64) (io.ReadCloser, error) {
	if contexto.ValidarParaEn(vecports.AccionAlmacenLeer, reloj.Ahora().UTC()) != nil {
		return nil, docports.ErrSubidaLocalDenegada
	}
	sol := vecports.SolicitudAbrirObjeto{Contexto: contexto, Objeto: obj,
		Zona: vecports.ZonaAlmacenAdmitida, Limite: limitePDFSubidaLocal}
	lectura, err := almacen.Abrir(ctx, sol)
	if err != nil {
		return nil, err
	}
	defer lectura.Contenido.Close()
	if lectura.ValidarContra(sol) != nil || lectura.Objeto.HuellaSHA256 != huella ||
		lectura.Objeto.Tamano != tamano || lectura.Objeto.MIME != "application/pdf" {
		return nil, docports.ErrSubidaLocalDenegada
	}
	contenido, err := io.ReadAll(io.LimitReader(lectura.Contenido, limitePDFSubidaLocal+1))
	if err != nil || int64(len(contenido)) != tamano {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	suma := sha256.Sum256(contenido)
	if hex.EncodeToString(suma[:]) != huella {
		return nil, docports.ErrSubidaLocalDenegada
	}
	return io.NopCloser(bytes.NewReader(contenido)), nil
}

var _ docports.LectorOriginalSubidaLocalRRHH = (*ServicioOriginalSubidaLocalRRHH)(nil)
