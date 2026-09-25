package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"reflect"

	baseapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const limiteOriginal = 16 << 20

type Servicio struct {
	Repositorio      ports.Repositorio
	Almacen          vecports.AlmacenObjetos
	Politicas        vecports.ResolutorPoliticaConservacionDocumental
	Reloj            vecports.Reloj
	ContextosLectura ports.FabricaContextoLectura
}

func dependenciaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func (s *Servicio) disponible() bool {
	return s.registroDisponible() && !dependenciaNula(s.Almacen)
}

// registroDisponible basta para operaciones que no tocan bytes: registrar una
// referencia externa y listar. El almacén solo se exige al custodiar o leer.
func (s *Servicio) registroDisponible() bool {
	return s != nil && !dependenciaNula(s.Repositorio) &&
		!dependenciaNula(s.Politicas) && !dependenciaNula(s.Reloj)
}

func efectoLigado(a ports.AutorizacionV3, preimagen []byte, err error) bool {
	return err == nil && a.Material.ResumenCapacidad().EfectoHuellaSHA256() == ports.HuellaEfectoV3(preimagen)
}

// AltaGenerado custodia los bytes por el puerto existente y solo confirma
// metadatos tras verificar su recibo. Ante fallo posterior, el objeto puede
// quedar huerfano y debe reconciliarse; nunca se anuncia como documento.
func (s *Servicio) AltaGenerado(ctx context.Context, in ports.AltaGenerado) (domain.Documento, error) {
	if !s.disponible() || ctx == nil || ctx.Err() != nil ||
		!domain.ReferenciaOpacaValida(in.ID) || !domain.ReferenciaOpacaValida(in.ClaveIdempotencia) ||
		!domain.IdentificadorTecnicoValido(in.ModuloID) || !domain.ReferenciaOpacaValida(in.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(in.TipoRef) || in.Version == 0 || !domain.MIMEValido(in.MIME) ||
		len(in.Contenido) == 0 || len(in.Contenido) > limiteOriginal ||
		in.SolicitudPolitica.Validar() != nil ||
		in.SolicitudPolitica.ExpedienteRef() != in.ExpedienteRef ||
		in.SolicitudPolitica.TipoDocumentalRef() != in.TipoRef {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	ahora := s.Reloj.Ahora()
	if in.Autorizacion.ValidarPara(ports.AccionAlta, ahora) != nil {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	politica, err := baseapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(ctx, s.Politicas, s.Reloj, in.SolicitudPolitica)
	if err != nil {
		return domain.Documento{}, err
	}
	suma := sha256.Sum256(in.Contenido)
	huella := hex.EncodeToString(suma[:])
	persistente := ports.AltaPersistente{
		ID: in.ID, ClaveIdempotencia: in.ClaveIdempotencia,
		ModuloID: in.ModuloID, ExpedienteRef: in.ExpedienteRef,
		TipoRef: in.TipoRef, Version: in.Version, MIME: in.MIME,
		HuellaSHA256: huella, Tamano: int64(len(in.Contenido)),
		Politica: politica, Autorizacion: in.Autorizacion,
	}
	preimagen, err := persistente.PreimagenAlta()
	if err != nil || in.Autorizacion.Material.ResumenCapacidad().EfectoHuellaSHA256() != ports.HuellaEfectoV3(preimagen) {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	solicitud := vecports.SolicitudEscribirObjeto{
		Contexto: in.ContextoAlmacen, ClaveIdempotencia: in.ClaveIdempotencia,
		Zona: vecports.ZonaAlmacenAdmitida, MIME: in.MIME,
		Tamano: int64(len(in.Contenido)), HuellaSHA256: huella,
		Contenido: bytes.NewReader(in.Contenido),
	}
	if solicitud.Validar() != nil || in.ContextoAlmacen.ValidarParaEn(vecports.AccionAlmacenEscribir, ahora) != nil {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	capacidades, err := s.Almacen.Capacidades(ctx)
	if err != nil {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	if !capacidades.Retencion ||
		(politica.Politica().Proteccion() == vecports.ProteccionPoliticaConservacionDocumentalBloqueada && !capacidades.BloqueoLegal) {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	objeto, err := s.Almacen.Escribir(ctx, solicitud)
	if err != nil {
		return domain.Documento{}, err
	}
	if objeto.ValidarEscritura(solicitud, capacidades) != nil ||
		!custodiaSatisfacePolitica(objeto, capacidades, politica.Politica()) {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	persistente.Objeto = objeto
	documento, err := s.Repositorio.ConfirmarAlta(ctx, persistente)
	if err != nil {
		return domain.Documento{}, err
	}
	if documento.Validar() != nil || documento.Custodia != domain.CustodiaVEC ||
		documento.ID != in.ID || documento.Version != in.Version ||
		documento.ModuloID != in.ModuloID || documento.ExpedienteRef != in.ExpedienteRef ||
		documento.TipoRef != in.TipoRef || documento.MIME != in.MIME ||
		documento.HuellaSHA256 != huella || documento.Tamano != int64(len(in.Contenido)) ||
		documento.ObjetoRef != objeto.Objeto.Objeto.Referencia ||
		documento.ObjetoVersion != objeto.Objeto.Objeto.Version ||
		documento.PoliticaRef != in.SolicitudPolitica.PoliticaRef() ||
		documento.VersionPolitica != in.SolicitudPolitica.VersionPolitica() ||
		documento.HuellaPoliticaSHA256 != hex.EncodeToString(in.SolicitudPolitica.HuellaPoliticaSHA256()) ||
		documento.Proteccion != string(politica.Politica().Proteccion()) ||
		!documento.ConservacionHasta.Equal(politica.Politica().ConservacionHasta()) ||
		documento.EstadoPolitica != ports.EstadoPolitica(politica.Politica()) ||
		documento.EstadoFirma != domain.EstadoFirmaPendienteProveedor {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	return documento, nil
}

// custodiaSatisfacePolitica exige, con politica aprobada, retencion del
// proveedor hasta el plazo. Con politica provisional exige lo contrario: que
// el conector no haya fijado retencion ni inmovilizado el objeto, porque ese
// efecto seria irreversible con plazos que la autoridad aun no ha aprobado.
func custodiaSatisfacePolitica(
	objeto vecports.ResultadoOperacionObjeto,
	capacidades vecports.CapacidadesAlmacenObjetos,
	politica vecports.PoliticaConservacionDocumental,
) bool {
	if politica.Validar() != nil || !capacidades.Retencion {
		return false
	}
	if politica.Provisional() {
		return objeto.Objeto.RetenidoHasta.IsZero() && !objeto.Objeto.Inmovilizado &&
			politica.Proteccion() == vecports.ProteccionPoliticaConservacionDocumentalOrdinaria
	}
	if objeto.Objeto.RetenidoHasta.IsZero() ||
		objeto.Objeto.RetenidoHasta.Before(politica.ConservacionHasta()) {
		return false
	}
	if politica.Proteccion() == vecports.ProteccionPoliticaConservacionDocumentalBloqueada &&
		(!capacidades.BloqueoLegal || !objeto.Objeto.Inmovilizado) {
		return false
	}
	return true
}

// RegistrarExterno anota un original custodiado por otro sistema con su
// referencia, huella y metadatos gobernados. No lee ni escribe bytes: la
// custodia, la integridad del contenido y su disponibilidad siguen siendo del
// custodio. El efecto consume V3 en la misma transaccion que metadatos,
// auditoria y outbox.
func (s *Servicio) RegistrarExterno(ctx context.Context, in ports.AltaExterna) (domain.Documento, error) {
	if !s.registroDisponible() || ctx == nil || ctx.Err() != nil ||
		!domain.ReferenciaOpacaValida(in.ID) || !domain.ReferenciaOpacaValida(in.ClaveIdempotencia) ||
		!domain.IdentificadorTecnicoValido(in.ModuloID) || !domain.ReferenciaOpacaValida(in.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(in.TipoRef) || in.Version == 0 || in.Tamano < 0 ||
		(in.MIME != "" && !domain.MIMEValido(in.MIME)) || in.Custodia.Validar() != nil ||
		in.SolicitudPolitica.Validar() != nil ||
		in.SolicitudPolitica.ExpedienteRef() != in.ExpedienteRef ||
		in.SolicitudPolitica.TipoDocumentalRef() != in.TipoRef {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	if in.Autorizacion.ValidarPara(ports.AccionRegistrarExterno, s.Reloj.Ahora()) != nil ||
		in.Autorizacion.RecursoRef != in.ID || in.Autorizacion.AmbitoRef != in.ExpedienteRef {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	politica, err := baseapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(ctx, s.Politicas, s.Reloj, in.SolicitudPolitica)
	if err != nil {
		return domain.Documento{}, err
	}
	persistente := ports.AltaExternaPersistente{
		ID: in.ID, ClaveIdempotencia: in.ClaveIdempotencia,
		ModuloID: in.ModuloID, ExpedienteRef: in.ExpedienteRef,
		TipoRef: in.TipoRef, Version: in.Version, MIME: in.MIME, Tamano: in.Tamano,
		Custodia: in.Custodia, Politica: politica, Autorizacion: in.Autorizacion,
	}
	preimagen, err := persistente.PreimagenExterna()
	if !efectoLigado(in.Autorizacion, preimagen, err) {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	documento, err := s.Repositorio.ConfirmarReferenciaExterna(ctx, persistente)
	if err != nil {
		return domain.Documento{}, err
	}
	if documento.Validar() != nil || documento.Custodia != domain.CustodiaExterna ||
		documento.ID != in.ID || documento.Version != in.Version ||
		documento.ModuloID != in.ModuloID || documento.ExpedienteRef != in.ExpedienteRef ||
		documento.TipoRef != in.TipoRef || documento.MIME != in.MIME || documento.Tamano != in.Tamano ||
		documento.HuellaSHA256 != in.Custodia.HuellaSHA256 || documento.CustodiaExternaRef != in.Custodia ||
		documento.PoliticaRef != in.SolicitudPolitica.PoliticaRef() ||
		documento.VersionPolitica != in.SolicitudPolitica.VersionPolitica() ||
		documento.HuellaPoliticaSHA256 != hex.EncodeToString(in.SolicitudPolitica.HuellaPoliticaSHA256()) ||
		documento.Proteccion != string(politica.Politica().Proteccion()) ||
		!documento.ConservacionHasta.Equal(politica.Politica().ConservacionHasta()) ||
		documento.EstadoPolitica != ports.EstadoPolitica(politica.Politica()) ||
		documento.EstadoFirma != domain.EstadoFirmaPendienteProveedor {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	return documento, nil
}

func (s *Servicio) ListarExpediente(ctx context.Context, in ports.ConsultaExpediente) (ports.PaginaDocumentos, error) {
	if !s.registroDisponible() || ctx == nil || ctx.Err() != nil ||
		!domain.ReferenciaOpacaValida(in.ExpedienteRef) || in.Limite == 0 || in.Limite > 100 ||
		(in.Cursor != "" && !domain.ReferenciaValida(in.Cursor)) ||
		in.Autorizacion.ValidarPara(ports.AccionListar, s.Reloj.Ahora()) != nil {
		return ports.PaginaDocumentos{}, ports.ErrSolicitudInvalida
	}
	preimagen, errPreimagen := in.PreimagenListar()
	if !efectoLigado(in.Autorizacion, preimagen, errPreimagen) {
		return ports.PaginaDocumentos{}, ports.ErrSolicitudInvalida
	}
	pagina, err := s.Repositorio.ListarExpediente(ctx, in)
	if err != nil {
		return ports.PaginaDocumentos{}, err
	}
	if len(pagina.Items) > int(in.Limite) ||
		(pagina.SiguienteCursor != "" && !domain.ReferenciaValida(pagina.SiguienteCursor)) {
		return ports.PaginaDocumentos{}, ports.ErrCapacidadNoDisponible
	}
	for _, d := range pagina.Items {
		if d.Validar() != nil || d.ExpedienteRef != in.ExpedienteRef {
			return ports.PaginaDocumentos{}, ports.ErrCapacidadNoDisponible
		}
	}
	return pagina, nil
}

func (s *Servicio) DescargarOriginal(ctx context.Context, in ports.ConsultaDocumento) (ports.Original, error) {
	if !s.disponible() || ctx == nil || ctx.Err() != nil ||
		!domain.ReferenciaOpacaValida(in.DocumentoID) || in.Version == 0 ||
		in.Autorizacion.ValidarPara(ports.AccionDescargar, s.Reloj.Ahora()) != nil {
		return ports.Original{}, ports.ErrSolicitudInvalida
	}
	preimagen, errPreimagen := in.PreimagenDescargar()
	if !efectoLigado(in.Autorizacion, preimagen, errPreimagen) {
		return ports.Original{}, ports.ErrSolicitudInvalida
	}
	// Sin autoridad de lectura del almacén no se consume la concesión V3:
	// la descarga no llegaría a producirse.
	if dependenciaNula(s.ContextosLectura) {
		return ports.Original{}, ports.ErrCapacidadNoDisponible
	}
	d, err := s.Repositorio.Obtener(ctx, in)
	if err != nil {
		return ports.Original{}, err
	}
	if d.Validar() != nil || d.ID != in.DocumentoID || d.Version != in.Version ||
		!d.Descargable() || d.Tamano > limiteOriginal {
		return ports.Original{}, ports.ErrOriginalNoDisponible
	}
	contextoAlmacen, err := s.ContextosLectura.ContextoLecturaOriginal(ctx, d, in.Autorizacion)
	if err != nil {
		return ports.Original{}, ports.ErrCapacidadNoDisponible
	}
	solicitud := vecports.SolicitudAbrirObjeto{
		Contexto: contextoAlmacen,
		Objeto:   vecports.ReferenciaObjetoAlmacen{Referencia: d.ObjetoRef, Version: d.ObjetoVersion},
		Zona:     vecports.ZonaAlmacenAdmitida, Limite: d.Tamano,
	}
	if solicitud.Validar() != nil || contextoAlmacen.ValidarParaEn(vecports.AccionAlmacenLeer, s.Reloj.Ahora()) != nil {
		return ports.Original{}, ports.ErrSolicitudInvalida
	}
	lectura, err := s.Almacen.Abrir(ctx, solicitud)
	if err != nil {
		return ports.Original{}, err
	}
	if lectura.Contenido != nil {
		defer lectura.Contenido.Close()
	}
	if lectura.ValidarContra(solicitud) != nil ||
		lectura.Objeto.HuellaSHA256 != d.HuellaSHA256 ||
		lectura.Objeto.MIME != d.MIME || lectura.Objeto.Tamano != d.Tamano {
		return ports.Original{}, ports.ErrOriginalNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(lectura.Contenido, d.Tamano+1))
	if err != nil || int64(len(contenido)) != d.Tamano {
		return ports.Original{}, ports.ErrOriginalNoDisponible
	}
	suma := sha256.Sum256(contenido)
	if hex.EncodeToString(suma[:]) != d.HuellaSHA256 {
		return ports.Original{}, ports.ErrOriginalNoDisponible
	}
	return ports.Original{Contenido: contenido, MIME: d.MIME, HuellaSHA256: d.HuellaSHA256}, nil
}

func (s *Servicio) PrepararNotificacion(ctx context.Context, in ports.PreparacionNotificacion) (domain.NotificacionPreparada, error) {
	if !s.disponible() || ctx == nil || ctx.Err() != nil ||
		!domain.ReferenciaOpacaValida(in.ID) || !domain.ReferenciaOpacaValida(in.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(in.DocumentoID) || !domain.ReferenciaOpacaValida(in.DestinatarioRef) ||
		!domain.IdentificadorTecnicoValido(in.Canal) || in.Version == 0 ||
		in.Autorizacion.ValidarPara(ports.AccionPrepararNotificacion, s.Reloj.Ahora()) != nil {
		return domain.NotificacionPreparada{}, ports.ErrSolicitudInvalida
	}
	preimagen, errPreimagen := in.PreimagenPreparar()
	if !efectoLigado(in.Autorizacion, preimagen, errPreimagen) {
		return domain.NotificacionPreparada{}, ports.ErrSolicitudInvalida
	}
	r, err := s.Repositorio.ConfirmarPreparacion(ctx, in)
	if err != nil {
		return domain.NotificacionPreparada{}, err
	}
	if r.Validar() != nil || r.ID != in.ID || r.DocumentoID != in.DocumentoID ||
		r.Version != in.Version || r.DestinatarioRef != in.DestinatarioRef || r.Canal != in.Canal {
		return domain.NotificacionPreparada{}, ports.ErrCapacidadNoDisponible
	}
	return r, nil
}
