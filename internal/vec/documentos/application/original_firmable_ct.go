package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	baseapp "vec-diputacion-granada/internal/vec/application"
	canonicodoc "vec-diputacion-granada/internal/vec/canonico/documental"
	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func datosReservaOriginalFirmable(r ports.ReservaOriginalFirmable) (canonicodoc.DatosReservaOriginal, error) {
	if r.Politica.Validar() != nil {
		return canonicodoc.DatosReservaOriginal{}, ports.ErrSolicitudInvalida
	}
	p := r.Politica.Politica()
	s := p.Solicitud()
	if s.ExpedienteRef() != r.ExpedienteRef || s.TipoDocumentalRef() != r.TipoRef {
		return canonicodoc.DatosReservaOriginal{}, ports.ErrSolicitudInvalida
	}
	return canonicodoc.DatosReservaOriginal{
		ID: r.ID, ClaveIdempotencia: r.ClaveIdempotencia, ModuloID: r.ModuloID,
		ExpedienteRef: r.ExpedienteRef, TipoRef: r.TipoRef, Version: r.Version,
		MIME: r.MIME, HuellaSHA256: r.HuellaSHA256, Tamano: r.Tamano,
		PoliticaRef: s.PoliticaRef(), VersionPolitica: s.VersionPolitica(),
		HuellaPoliticaSHA256: hex.EncodeToString(s.HuellaPoliticaSHA256()),
		Proteccion:           string(p.Proteccion()), ConservacionHasta: p.ConservacionHasta(),
		EstadoPolitica: ports.EstadoPolitica(p),
	}, nil
}

func PreimagenReservaOriginalFirmable(r ports.ReservaOriginalFirmable) ([]byte, error) {
	d, err := datosReservaOriginalFirmable(r)
	if err != nil {
		return nil, err
	}
	preimagen, err := d.Preimagen()
	if err != nil {
		return nil, ports.ErrSolicitudInvalida
	}
	return preimagen, nil
}

func ValidarIntentoOriginalFirmable(i ports.IntentoOriginalFirmable, r ports.ReservaOriginalFirmable) error {
	d, err := datosReservaOriginalFirmable(r)
	if err != nil || !i.ValidoContra(d) {
		return ports.ErrCapacidadNoDisponible
	}
	return nil
}

func PreimagenConfirmacionOriginalFirmable(c ports.ConfirmacionOriginalFirmable) ([]byte, error) {
	preimagen, err := (canonicodoc.DatosConfirmacionOriginal{Intento: c.Intento, Objeto: c.Objeto}).Preimagen()
	if err != nil {
		return nil, ports.ErrSolicitudInvalida
	}
	return preimagen, nil
}

func ValidarAutorizacionOriginalFirmable(a ports.AutorizacionV3, accion string, ahora time.Time) error {
	resumen := a.Material.ResumenCapacidad()
	d := canonicodoc.DatosAutorizacionOriginal{
		MaterialValido: a.Material.ValidarEstructura() == nil && resumen.ValidarEstructura() == nil,
		Accion:         a.Accion, Finalidad: a.Finalidad, RecursoRef: a.RecursoRef,
		AmbitoRef: a.AmbitoRef, PrincipalID: a.PrincipalID, PerfilActivoRef: a.PerfilActivoRef,
		CorrelacionRef: a.CorrelacionRef, Audiencia: resumen.AudienciaConsumo(),
		Operacion: resumen.Operacion(), EfectoRef: resumen.EfectoRef(),
		EmitidaEn: resumen.EmitidaEn(), ExpiraEn: resumen.ExpiraEn(), Ahora: ahora,
	}
	if a.Accion != accion || !d.Valida() {
		return ports.ErrSolicitudInvalida
	}
	return nil
}

func nuevoObjetoOriginalFirmable(i ports.IntentoOriginalFirmable, resultado vecports.ResultadoOperacionObjeto) (ports.ObjetoOriginalFirmable, error) {
	if resultado.Validar() != nil || resultado.Objeto.HuellaSHA256 != i.HuellaSHA256 {
		return ports.ObjetoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	recibo, err := json.Marshal(resultado.Evidencia)
	if err != nil {
		return ports.ObjetoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	var retencion *time.Time
	if !resultado.Objeto.RetenidoHasta.IsZero() {
		v := resultado.Objeto.RetenidoHasta.UTC()
		retencion = &v
	}
	o := ports.ObjetoOriginalFirmable{
		ClaveAlmacenRef: i.ClaveAlmacenRef,
		ObjetoRef:       resultado.Objeto.Objeto.Referencia, ObjetoVersion: resultado.Objeto.Objeto.Version,
		ConectorRef: resultado.Objeto.ConectorID, ReciboObjetoRef: resultado.Evidencia.OperacionRef,
		ReciboObjetoHuellaSHA256: canonicodoc.HuellaReciboObjetoOriginal(recibo),
		RetenidoHasta:            retencion, Inmovilizado: resultado.Objeto.Inmovilizado,
		MIME: resultado.Objeto.MIME, Tamano: resultado.Objeto.Tamano,
		HuellaSHA256: resultado.Objeto.HuellaSHA256,
	}
	if !o.Valido() {
		return ports.ObjetoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	return o, nil
}

// CustodiarOriginalFirmable reserva la identidad y la huella antes de escribir
// el objeto. Cada intento pendiente usa la clave durable emitida por SQL; un
// fallo entre almacén y confirmación deja un objeto huérfano reconciliable,
// pero no permite sustituir la reserva ni confirmar otro material.
func (s *Servicio) CustodiarOriginalFirmable(
	ctx context.Context,
	in ports.OrdenCustodiarOriginalFirmable,
	autoridad ports.AutorizarOriginalFirmable,
) (ports.IntentoOriginalFirmable, error) {
	if !s.disponible() || dependenciaNula(autoridad) || ctx == nil || ctx.Err() != nil ||
		!domain.ReferenciaOpacaValida(in.ID) || !domain.ReferenciaOpacaValida(in.ClaveIdempotencia) ||
		!domain.IdentificadorTecnicoValido(in.ModuloID) || !domain.ReferenciaExpedienteModuloValida(in.ModuloID, in.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(in.TipoRef) || in.Version == 0 || in.Version > 9007199254740991 ||
		in.MIME != "application/pdf" || len(in.Contenido) < 8 || len(in.Contenido) > 1<<20 ||
		!bytes.HasPrefix(in.Contenido, []byte("%PDF-")) || in.SolicitudPolitica.Validar() != nil ||
		in.SolicitudPolitica.ExpedienteRef() != in.ExpedienteRef ||
		in.SolicitudPolitica.TipoDocumentalRef() != in.TipoRef || !s.tipoReservadoOriginalCT(in.TipoRef) {
		return ports.IntentoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	repositorio, ok := s.Repositorio.(ports.RepositorioOriginalFirmable)
	if !ok || dependenciaNula(repositorio) {
		return ports.IntentoOriginalFirmable{}, ports.ErrCapacidadNoDisponible
	}
	politica, err := baseapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(ctx, s.Politicas, s.Reloj, in.SolicitudPolitica)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	suma := sha256.Sum256(in.Contenido)
	huella := hex.EncodeToString(suma[:])
	reserva := ports.ReservaOriginalFirmable{
		ID: in.ID, ClaveIdempotencia: in.ClaveIdempotencia, ModuloID: in.ModuloID,
		ExpedienteRef: in.ExpedienteRef, TipoRef: in.TipoRef, Version: in.Version,
		MIME: in.MIME, HuellaSHA256: huella, Tamano: int64(len(in.Contenido)), Politica: politica,
	}
	preimagenReserva, err := PreimagenReservaOriginalFirmable(reserva)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	reserva.Autorizacion, err = autoridad.AutorizarReservaOriginal(ctx, preimagenReserva, in.ID, in.ExpedienteRef)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	if reserva.Autorizacion.RecursoRef != in.ID || reserva.Autorizacion.AmbitoRef != in.ExpedienteRef ||
		ValidarAutorizacionOriginalFirmable(reserva.Autorizacion, ports.AccionReservarOriginalFirmable, s.Reloj.Ahora()) != nil ||
		reserva.Autorizacion.Material.ResumenCapacidad().EfectoHuellaSHA256() != ports.HuellaEfectoV3(preimagenReserva) {
		return ports.IntentoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	intento, err := repositorio.ReservarOriginalFirmable(ctx, reserva)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	if ValidarIntentoOriginalFirmable(intento, reserva) != nil {
		return ports.IntentoOriginalFirmable{}, ports.ErrCapacidadNoDisponible
	}
	if intento.Estado == "confirmado" {
		return intento, nil
	}
	contextoAlmacen, err := autoridad.ContextoEscrituraOriginal(ctx, reserva, intento)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	escritura := vecports.SolicitudEscribirObjeto{
		Contexto: contextoAlmacen, ClaveIdempotencia: intento.ClaveAlmacenRef,
		Zona: vecports.ZonaAlmacenAdmitida, MIME: in.MIME, Tamano: int64(len(in.Contenido)),
		HuellaSHA256: huella, Contenido: bytes.NewReader(in.Contenido),
	}
	if escritura.Validar() != nil || contextoAlmacen.ValidarParaEn(vecports.AccionAlmacenEscribir, s.Reloj.Ahora()) != nil {
		return ports.IntentoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	capacidades, err := s.Almacen.Capacidades(ctx)
	if err != nil || !capacidades.Retencion ||
		(politica.Politica().Proteccion() == vecports.ProteccionPoliticaConservacionDocumentalBloqueada && !capacidades.BloqueoLegal) {
		return ports.IntentoOriginalFirmable{}, ports.ErrCapacidadNoDisponible
	}
	objeto, err := s.Almacen.Escribir(ctx, escritura)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	if objeto.ValidarEscritura(escritura, capacidades) != nil ||
		!custodiaSatisfacePolitica(objeto, capacidades, politica.Politica()) {
		return ports.IntentoOriginalFirmable{}, ports.ErrCapacidadNoDisponible
	}
	objetoConfirmacion, err := nuevoObjetoOriginalFirmable(intento, objeto)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	confirmacion := ports.ConfirmacionOriginalFirmable{Intento: intento, Objeto: objetoConfirmacion}
	preimagenConfirmacion, err := PreimagenConfirmacionOriginalFirmable(confirmacion)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	confirmacion.Autorizacion, err = autoridad.AutorizarConfirmacionOriginal(ctx, preimagenConfirmacion, in.ID, in.ExpedienteRef)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	if confirmacion.Autorizacion.RecursoRef != in.ID || confirmacion.Autorizacion.AmbitoRef != in.ExpedienteRef ||
		ValidarAutorizacionOriginalFirmable(confirmacion.Autorizacion, ports.AccionConfirmarOriginalFirmable, s.Reloj.Ahora()) != nil ||
		confirmacion.Autorizacion.Material.ResumenCapacidad().EfectoHuellaSHA256() != ports.HuellaEfectoV3(preimagenConfirmacion) {
		return ports.IntentoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	documento, err := repositorio.ConfirmarOriginalFirmable(ctx, confirmacion)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	if documento.Validar() != nil || documento.ID != in.ID || documento.ModuloID != in.ModuloID ||
		documento.ExpedienteRef != in.ExpedienteRef || documento.TipoRef != in.TipoRef ||
		documento.Version != in.Version || documento.HuellaSHA256 != huella ||
		documento.MIME != in.MIME || documento.Tamano != int64(len(in.Contenido)) ||
		documento.ObjetoRef != objeto.Objeto.Objeto.Referencia ||
		documento.ObjetoVersion != objeto.Objeto.Objeto.Version {
		return ports.IntentoOriginalFirmable{}, ports.ErrCapacidadNoDisponible
	}
	intento.Estado = "confirmado"
	return intento, nil
}

func (s *Servicio) tipoReservadoOriginalCT(tipoRef string) bool {
	if s == nil {
		return false
	}
	r, ok := s.Politicas.(ports.ReservaTiposOriginalCT)
	return ok && r.CustodiaOriginalCTReservada(tipoRef)
}

// DescargarOriginalConDocumento devuelve el contenido exacto y sus metadatos
// durables en una sola operación V3. Es genérico: no interpreta el módulo ni
// el tipo documental. La consulta SQL consume la concesión una vez; llamar
// después a Repositorio.Obtener con la misma concesión sería rechazado.
func (s *Servicio) DescargarOriginalConDocumento(ctx context.Context, in ports.ConsultaDocumento) (ports.Original, domain.Documento, error) {
	vacio := ports.Original{}
	documentoVacio := domain.Documento{}
	if !s.disponible() || ctx == nil || ctx.Err() != nil ||
		!domain.ReferenciaOpacaValida(in.DocumentoID) || in.Version == 0 ||
		in.Autorizacion.ValidarPara(ports.AccionDescargar, s.Reloj.Ahora()) != nil {
		return vacio, documentoVacio, ports.ErrSolicitudInvalida
	}
	preimagen, err := in.PreimagenDescargar()
	if !efectoLigado(in.Autorizacion, preimagen, err) {
		return vacio, documentoVacio, ports.ErrSolicitudInvalida
	}
	if dependenciaNula(s.ContextosLectura) {
		return vacio, documentoVacio, ports.ErrCapacidadNoDisponible
	}
	d, err := s.Repositorio.Obtener(ctx, in)
	if err != nil {
		return vacio, documentoVacio, err
	}
	if d.Validar() != nil || d.ID != in.DocumentoID || d.Version != in.Version ||
		!d.Descargable() || d.Tamano > limiteOriginal {
		return vacio, documentoVacio, ports.ErrOriginalNoDisponible
	}
	contextoAlmacen, err := s.ContextosLectura.ContextoLecturaOriginal(ctx, d, in.Autorizacion)
	if err != nil {
		return vacio, documentoVacio, fmt.Errorf("%w: %w", ports.ErrCapacidadNoDisponible, err)
	}
	solicitud := vecports.SolicitudAbrirObjeto{
		Contexto: contextoAlmacen,
		Objeto:   vecports.ReferenciaObjetoAlmacen{Referencia: d.ObjetoRef, Version: d.ObjetoVersion},
		Zona:     vecports.ZonaAlmacenAdmitida, Limite: d.Tamano,
	}
	if solicitud.Validar() != nil || contextoAlmacen.ValidarParaEn(vecports.AccionAlmacenLeer, s.Reloj.Ahora()) != nil {
		return vacio, documentoVacio, ports.ErrSolicitudInvalida
	}
	lectura, err := s.Almacen.Abrir(ctx, solicitud)
	if err != nil {
		return vacio, documentoVacio, err
	}
	if lectura.Contenido != nil {
		defer lectura.Contenido.Close()
	}
	if lectura.ValidarContra(solicitud) != nil || lectura.Objeto.HuellaSHA256 != d.HuellaSHA256 ||
		lectura.Objeto.MIME != d.MIME || lectura.Objeto.Tamano != d.Tamano {
		return vacio, documentoVacio, ports.ErrOriginalNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(lectura.Contenido, d.Tamano+1))
	if err != nil || int64(len(contenido)) != d.Tamano {
		return vacio, documentoVacio, ports.ErrOriginalNoDisponible
	}
	suma := sha256.Sum256(contenido)
	if hex.EncodeToString(suma[:]) != d.HuellaSHA256 {
		return vacio, documentoVacio, ports.ErrOriginalNoDisponible
	}
	return ports.Original{Contenido: contenido, MIME: d.MIME, HuellaSHA256: d.HuellaSHA256}, d, nil
}
