package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"

	baseapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// limiteFirmado coincide con el de los borradores de firma de CT.
const limiteFirmado = 1 << 20

// CustodiarFirmado custodia el PDF firmado y verificado de un expediente.
// Orden: política → preimagen → V3 del autorizador ligada a ella → concesión
// de almacén V3 ligada a esa decisión → escritura verificada → confirmación
// SQL, que consume la V3 en la misma transacción que documento, firmado,
// auditoría y outbox. Si la confirmación falla, el objeto queda huérfano y se
// reconcilia; nunca se anuncia como documento. Una recuperación tras perder la
// respuesta pide otra V3 y recibe el documento original (Documentos 000009).
func (s *Servicio) CustodiarFirmado(ctx context.Context, in ports.CustodiaFirmado, autorizador ports.AutorizadorCustodiaFirmado) (domain.Documento, error) {
	if s == nil || dependenciaNula(s.RepositorioCustodia) || dependenciaNula(s.ContextosCustodia) ||
		dependenciaNula(autorizador) ||
		dependenciaNula(s.Almacen) || dependenciaNula(s.Politicas) || dependenciaNula(s.Reloj) ||
		ctx == nil || ctx.Err() != nil ||
		!domain.ReferenciaOpacaValida(in.ID) || !domain.ReferenciaOpacaValida(in.ClaveIdempotencia) ||
		!domain.IdentificadorTecnicoValido(in.ModuloID) || !domain.ReferenciaExpedienteModuloValida(in.ModuloID, in.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(in.TipoRef) || !domain.ReferenciaOpacaValida(in.FirmaOperacionRef) ||
		!domain.HuellaValida(in.HuellaOriginalSHA256) || in.Version == 0 ||
		len(in.Contenido) < 5 || len(in.Contenido) > limiteFirmado || !bytes.HasPrefix(in.Contenido, []byte("%PDF-")) ||
		!s.tipoReservadoFirmado(in.TipoRef) ||
		in.SolicitudPolitica.Validar() != nil ||
		in.SolicitudPolitica.ExpedienteRef() != in.ExpedienteRef ||
		in.SolicitudPolitica.TipoDocumentalRef() != in.TipoRef {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	suma := sha256.Sum256(in.Contenido)
	huella := hex.EncodeToString(suma[:])
	if huella == in.HuellaOriginalSHA256 {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	politica, err := baseapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(ctx, s.Politicas, s.Reloj, in.SolicitudPolitica)
	if err != nil {
		return domain.Documento{}, err
	}
	persistente := ports.CustodiaFirmadoPersistente{
		ID: in.ID, ClaveIdempotencia: in.ClaveIdempotencia, ModuloID: in.ModuloID,
		ExpedienteRef: in.ExpedienteRef, TipoRef: in.TipoRef, Version: in.Version,
		HuellaSHA256: huella, Tamano: int64(len(in.Contenido)),
		HuellaOriginalSHA256: in.HuellaOriginalSHA256, FirmaOperacionRef: in.FirmaOperacionRef,
		Politica: politica,
	}
	preimagen, err := persistente.PreimagenCustodia()
	if err != nil {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	autorizacion, err := autorizador.AutorizarCustodiaFirmado(ctx, preimagen, in.ID, in.ExpedienteRef)
	if err != nil {
		return domain.Documento{}, err
	}
	if autorizacion.ValidarPara(ports.AccionCustodiarFirmado, s.Reloj.Ahora()) != nil ||
		autorizacion.RecursoRef != in.ID || autorizacion.AmbitoRef != in.ExpedienteRef ||
		!efectoLigado(autorizacion, preimagen, nil) ||
		autorizacion.Material.ResumenCapacidad().Operacion() != ports.AccionCustodiarFirmado {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	persistente.Autorizacion = autorizacion
	contexto, err := s.ContextosCustodia.ContextoCustodiaFirmado(ctx, persistente)
	if err != nil {
		// El productor distingue una denegación confirmada de una fuente caída.
		// La fábrica conserva ambas causas; no borrar aquí la del PDP.
		return domain.Documento{}, err
	}
	proyeccion, err := contexto.Proyeccion()
	if err != nil || proyeccion.AccionNegocio != vecports.AccionNegocioCustodiarDocumentoFirmadoExpediente ||
		proyeccion.CargaRef != in.ID || proyeccion.EfectoRef != in.ID {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	solicitud := vecports.SolicitudEscribirObjeto{
		Contexto: contexto, ClaveIdempotencia: claveAlmacenCustodia(in.ClaveIdempotencia, autorizacion),
		Zona: vecports.ZonaAlmacenAdmitida, MIME: ports.MIMEDocumentoFirmado,
		Tamano: int64(len(in.Contenido)), HuellaSHA256: huella, Contenido: bytes.NewReader(in.Contenido),
	}
	if solicitud.Validar() != nil || contexto.ValidarParaEn(vecports.AccionAlmacenEscribir, s.Reloj.Ahora()) != nil {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	capacidades, err := s.Almacen.Capacidades(ctx)
	if err != nil || !capacidades.Retencion ||
		(politica.Politica().Proteccion() == vecports.ProteccionPoliticaConservacionDocumentalBloqueada && !capacidades.BloqueoLegal) {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	objeto, err := s.Almacen.Escribir(ctx, solicitud)
	if err != nil {
		return domain.Documento{}, err
	}
	// La versión y la referencia del objeto deben poder leerse después como
	// documento: si no, la fila quedaría confirmada e irrecuperable.
	if objeto.ValidarEscritura(solicitud, capacidades) != nil ||
		!domain.ReferenciaValida(objeto.Objeto.Objeto.Referencia) ||
		!domain.VersionObjetoValida(objeto.Objeto.Objeto.Version) ||
		!custodiaSatisfacePolitica(objeto, capacidades, politica.Politica()) {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	persistente.Objeto = objeto
	documento, err := s.RepositorioCustodia.ConfirmarCustodiaFirmado(ctx, persistente)
	if err != nil {
		return domain.Documento{}, err
	}
	// Una recuperación devuelve el documento original: su objeto puede ser
	// otro (el de este intento queda huérfano) y su conservación anterior.
	mismoObjeto := documento.ObjetoRef == objeto.Objeto.Objeto.Referencia &&
		documento.ObjetoVersion == objeto.Objeto.Objeto.Version
	if documento.Validar() != nil || documento.Custodia != domain.CustodiaVEC ||
		documento.ID != in.ID || documento.Version != in.Version ||
		documento.ModuloID != in.ModuloID || documento.ExpedienteRef != in.ExpedienteRef ||
		documento.TipoRef != in.TipoRef || documento.MIME != ports.MIMEDocumentoFirmado ||
		documento.HuellaSHA256 != huella || documento.Tamano != int64(len(in.Contenido)) ||
		documento.PoliticaRef != in.SolicitudPolitica.PoliticaRef() ||
		documento.VersionPolitica != in.SolicitudPolitica.VersionPolitica() ||
		documento.HuellaPoliticaSHA256 != hex.EncodeToString(in.SolicitudPolitica.HuellaPoliticaSHA256()) ||
		documento.Proteccion != string(politica.Politica().Proteccion()) ||
		documento.ConservacionHasta.After(politica.Politica().ConservacionHasta()) ||
		(mismoObjeto && !documento.ConservacionHasta.Equal(politica.Politica().ConservacionHasta())) ||
		documento.EstadoPolitica != ports.EstadoPolitica(politica.Politica()) ||
		documento.EstadoFirma != domain.EstadoFirmaPendienteProveedor {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	return documento, nil
}

// claveAlmacenCustodia liga la idempotencia del almacén a la decisión V3 de
// este intento. El almacén asocia su clave a la concesión exacta, y una
// recuperación tras perder la respuesta llega con otra decisión: con la clave
// del documento, el almacén la rechazaría y la custodia ya confirmada no se
// podría recuperar nunca. Con otra decisión se escribe otro objeto, que queda
// huérfano si SQL ya tenía el documento (se devuelve el original). Repetir con
// la misma decisión falla cerrado en el almacén antes de escribir, porque cada
// intento obtiene una concesión de almacén nueva: toda recuperación pide otra V3.
func claveAlmacenCustodia(clave string, a ports.AutorizacionV3) string {
	suma := sha256.Sum256([]byte("vec.documentos.custodia_firmado.almacen.v1\x00" + clave +
		"\x00" + a.Material.ResumenCapacidad().DecisionRef()))
	return "ref:" + hex.EncodeToString(suma[:])
}

// tipoReservadoFirmado consulta la reserva del catálogo de conservación. Sin
// ese catálogo ningún tipo cuenta como reservado (la custodia se deniega).
func (s *Servicio) tipoReservadoFirmado(tipoRef string) bool {
	r, ok := s.Politicas.(ports.ReservaTiposFirmado)
	return ok && r.CustodiaFirmadoReservada(tipoRef)
}
