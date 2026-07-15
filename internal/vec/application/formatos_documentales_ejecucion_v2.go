package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrEjecucionFormatoDocumentalCerrada = errors.New("vec: ejecucion de formato documental cerrada")
	ErrLimiteFormatoDocumentalExcedido   = errors.New("vec: limite de formato documental excedido")
)

const techoAbsolutoEjecucionDocumentalV2 uint64 = 256 * 1024 * 1024

var referenciaBorradorDocumentalValida = regexp.MustCompile(`^[a-z][a-z0-9._:-]{0,255}$`)
var idClaveHMACDocumentalV2Valido = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)

// OpcionesEjecucionFormatoDocumentalV2 obliga a declarar el techo local. Cero
// no activa un valor positivo por defecto. El limite efectivo sera el minimo
// de este techo, el perfil y las homologaciones de los componentes.
type OpcionesEjecucionFormatoDocumentalV2 struct {
	TechoInstitucionalBytes uint64
}

// ServicioEjecucionFormatoDocumentalV2 es el puente en sombra que ejecuta una
// resolucion sin exponer interfaces de componentes. Todavia no sustituye al
// generador PDF/DOCX heredado ni autoriza una superficie HTTP, CLI o MCP.
type ServicioEjecucionFormatoDocumentalV2 struct {
	catalogoPerfiles    ports.CatalogoPerfilesDocumentales
	situaciones         ports.RegistroSituacionesOperativasPerfilDocumental
	componentes         ports.RegistroComponentesDocumentalesAtestados
	renderizador        ports.EjecutorRenderizadoDocumental
	verificador         ports.EjecutorValidacionConformidadDocumental
	generadorReferencia ports.GeneradorReferenciaBorradorDocumental
	selladorDatos       ports.SelladorDatosDocumento
	reloj               ports.Reloj
	techoInstitucional  uint64
}

func NuevoServicioEjecucionFormatoDocumentalV2(
	catalogoPerfiles ports.CatalogoPerfilesDocumentales,
	situaciones ports.RegistroSituacionesOperativasPerfilDocumental,
	componentes ports.RegistroComponentesDocumentalesAtestados,
	renderizador ports.EjecutorRenderizadoDocumental,
	verificador ports.EjecutorValidacionConformidadDocumental,
	generadorReferencia ports.GeneradorReferenciaBorradorDocumental,
	selladorDatos ports.SelladorDatosDocumento,
	reloj ports.Reloj,
	opciones OpcionesEjecucionFormatoDocumentalV2,
) (*ServicioEjecucionFormatoDocumentalV2, error) {
	dependencias := []any{
		catalogoPerfiles, situaciones, componentes, renderizador,
		verificador, generadorReferencia, selladorDatos, reloj,
	}
	for _, dependencia := range dependencias {
		if dependenciaDocumentalGobernadaNula(dependencia) {
			return nil, ErrEjecucionFormatoDocumentalCerrada
		}
	}
	if opciones.TechoInstitucionalBytes == 0 ||
		opciones.TechoInstitucionalBytes > techoAbsolutoEjecucionDocumentalV2 {
		return nil, ErrEjecucionFormatoDocumentalCerrada
	}
	return &ServicioEjecucionFormatoDocumentalV2{
		catalogoPerfiles: catalogoPerfiles, situaciones: situaciones,
		componentes: componentes, renderizador: renderizador,
		verificador: verificador, generadorReferencia: generadorReferencia,
		selladorDatos: selladorDatos, reloj: reloj,
		techoInstitucional: opciones.TechoInstitucionalBytes,
	}, nil
}

// DatosEvidenciaRenderizadoDocumentalV2 es el sobre restaurable del corte en
// sombra. Su SHA-256 detecta alteraciones accidentales, pero no es una firma ni
// una MAC: produccion exigira anclaje durable y atestacion criptografica.
type DatosEvidenciaRenderizadoDocumentalV2 struct {
	Consulta                ports.ConsultaFormatoDocumental
	DescriptorPerfil        ports.DescriptorPerfilDocumental
	SituacionOperativa      domain.SituacionOperativaPerfilDocumental
	ComponenteRender        ports.DescriptorComponenteDocumentalAtestado
	ComponenteVerificador   ports.DescriptorComponenteDocumentalAtestado
	TechoInstitucionalBytes uint64
	LimiteEfectivoBytes     uint64
	HuellaEntradaHMAC       string
	BorradorRef             string
	HuellaSalidaSHA256      string
	TamanoSalida            uint64
	GeneradoEn              time.Time
	HuellaEvidenciaSHA256   string
}

type EvidenciaRenderizadoDocumentalV2 struct {
	datos DatosEvidenciaRenderizadoDocumentalV2
}

func nuevaEvidenciaRenderizadoDocumentalV2(
	datos DatosEvidenciaRenderizadoDocumentalV2,
) (EvidenciaRenderizadoDocumentalV2, error) {
	datos.HuellaEvidenciaSHA256 = huellaDatosEvidenciaRenderizadoV2(datos)
	evidencia := EvidenciaRenderizadoDocumentalV2{datos: datos}
	if evidencia.Validar() != nil {
		return EvidenciaRenderizadoDocumentalV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return evidencia, nil
}

func RestaurarEvidenciaRenderizadoDocumentalV2(
	datos DatosEvidenciaRenderizadoDocumentalV2,
) (EvidenciaRenderizadoDocumentalV2, error) {
	evidencia := EvidenciaRenderizadoDocumentalV2{datos: datos}
	if evidencia.Validar() != nil {
		return EvidenciaRenderizadoDocumentalV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return evidencia, nil
}

func (e EvidenciaRenderizadoDocumentalV2) Validar() error {
	d := e.datos
	perfil := d.DescriptorPerfil.Perfil()
	if d.Consulta.Validar() != nil || d.DescriptorPerfil.Validar() != nil ||
		!d.DescriptorPerfil.Coincide(d.Consulta) || d.SituacionOperativa.Validar() != nil ||
		!consultaSituacionDesdeDescriptorV2(d.DescriptorPerfil).Coincide(d.SituacionOperativa) ||
		!d.SituacionOperativa.AutorizaEjecucion(perfil, d.DescriptorPerfil.Revision()) ||
		d.ComponenteRender.Validar() != nil || d.ComponenteVerificador.Validar() != nil ||
		d.ComponenteRender.Componente().Rol() != domain.RolComponenteRenderizador ||
		d.ComponenteVerificador.Componente().Rol() != domain.RolComponenteVerificador ||
		!d.ComponenteRender.IndependienteDe(d.ComponenteVerificador) ||
		!d.ComponenteRender.Coincide(consultaComponenteDesdeDescriptorV2(
			d.DescriptorPerfil, domain.RolComponenteRenderizador,
		)) || !d.ComponenteVerificador.Coincide(consultaComponenteDesdeDescriptorV2(
		d.DescriptorPerfil, domain.RolComponenteVerificador,
	)) || d.TechoInstitucionalBytes == 0 ||
		d.TechoInstitucionalBytes > techoAbsolutoEjecucionDocumentalV2 ||
		d.LimiteEfectivoBytes != minimoUint64V2(
			d.TechoInstitucionalBytes, perfil.MaximoBytes(),
			d.ComponenteRender.MaximoBytes(), d.ComponenteVerificador.MaximoBytes(),
		) || !huellaHMACAplicacionV2Valida(d.HuellaEntradaHMAC) ||
		!referenciaBorradorAplicacionV2Valida(d.BorradorRef) ||
		!esSHA256AplicacionV2(d.HuellaSalidaSHA256) || d.TamanoSalida == 0 ||
		d.TamanoSalida > d.LimiteEfectivoBytes || d.GeneradoEn.IsZero() ||
		d.GeneradoEn.Location() != time.UTC || !esSHA256AplicacionV2(d.HuellaEvidenciaSHA256) ||
		huellaDatosEvidenciaRenderizadoV2(d) != d.HuellaEvidenciaSHA256 {
		return ErrEjecucionFormatoDocumentalCerrada
	}
	return nil
}

func (e EvidenciaRenderizadoDocumentalV2) Datos() (
	DatosEvidenciaRenderizadoDocumentalV2,
	error,
) {
	if e.Validar() != nil {
		return DatosEvidenciaRenderizadoDocumentalV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return e.datos, nil
}

func (e EvidenciaRenderizadoDocumentalV2) HuellaSHA256() (string, error) {
	if e.Validar() != nil {
		return "", ErrEjecucionFormatoDocumentalCerrada
	}
	return e.datos.HuellaEvidenciaSHA256, nil
}

// ArtefactoBorradorPreFirmaV2 solo puede ser creado por el ejecutor V2. Los
// bytes permanecen privados y cada lectura devuelve una copia. Marcar o firmar
// producira otro artefacto; este valor nunca se modifica in situ.
type ArtefactoBorradorPreFirmaV2 struct {
	referencia   string
	contenido    []byte
	perfilRef    domain.ReferenciaPerfilDocumental
	digestPerfil string
	mime         string
	huella       string
	generadoEn   time.Time
	evidencia    EvidenciaRenderizadoDocumentalV2
}

func (a ArtefactoBorradorPreFirmaV2) Validar() error {
	datos, err := a.evidencia.Datos()
	if err != nil || !referenciaBorradorAplicacionV2Valida(a.referencia) ||
		len(a.contenido) == 0 || !esSHA256AplicacionV2(a.huella) ||
		huellaBytesAplicacionV2(a.contenido) != a.huella || a.perfilRef.Validar() != nil ||
		!esSHA256AplicacionV2(a.digestPerfil) || strings.TrimSpace(a.mime) == "" ||
		a.generadoEn.IsZero() || a.generadoEn.Location() != time.UTC ||
		datos.BorradorRef != a.referencia || datos.HuellaSalidaSHA256 != a.huella ||
		datos.TamanoSalida != uint64(len(a.contenido)) ||
		datos.DescriptorPerfil.Perfil().Referencia() != a.perfilRef ||
		datos.DescriptorPerfil.Perfil().DigestSHA256() != a.digestPerfil ||
		datos.DescriptorPerfil.Perfil().MIME() != a.mime || !datos.GeneradoEn.Equal(a.generadoEn) {
		return ErrEjecucionFormatoDocumentalCerrada
	}
	return nil
}

func (a ArtefactoBorradorPreFirmaV2) Referencia() (string, error) {
	if a.Validar() != nil {
		return "", ErrEjecucionFormatoDocumentalCerrada
	}
	return a.referencia, nil
}

func (a ArtefactoBorradorPreFirmaV2) Contenido() ([]byte, error) {
	if a.Validar() != nil {
		return nil, ErrEjecucionFormatoDocumentalCerrada
	}
	return append([]byte(nil), a.contenido...), nil
}

func (a ArtefactoBorradorPreFirmaV2) Evidencia() (
	EvidenciaRenderizadoDocumentalV2,
	error,
) {
	if a.Validar() != nil {
		return EvidenciaRenderizadoDocumentalV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return a.evidencia, nil
}

func (a ArtefactoBorradorPreFirmaV2) HuellaSHA256() (string, error) {
	if a.Validar() != nil {
		return "", ErrEjecucionFormatoDocumentalCerrada
	}
	return a.huella, nil
}

func (s *ServicioEjecucionFormatoDocumentalV2) RenderizarBorrador(
	ctx context.Context,
	consulta ports.ConsultaFormatoDocumental,
	contenido domain.ContenidoDocumento,
) (ArtefactoBorradorPreFirmaV2, error) {
	if ctx == nil || ctx.Err() != nil || s == nil ||
		dependenciaDocumentalGobernadaNula(s.catalogoPerfiles) ||
		dependenciaDocumentalGobernadaNula(s.situaciones) ||
		dependenciaDocumentalGobernadaNula(s.componentes) ||
		dependenciaDocumentalGobernadaNula(s.renderizador) ||
		dependenciaDocumentalGobernadaNula(s.verificador) ||
		dependenciaDocumentalGobernadaNula(s.generadorReferencia) ||
		dependenciaDocumentalGobernadaNula(s.selladorDatos) ||
		dependenciaDocumentalGobernadaNula(s.reloj) || consulta.Validar() != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	descriptor, err := s.resolverDescriptorPerfilV2(ctx, consulta)
	if err != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	perfil := descriptor.Perfil()
	situacion, err := s.leerSituacionActualV2(ctx, descriptor)
	if err != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	render, err := s.resolverComponenteV2(ctx, descriptor, domain.RolComponenteRenderizador)
	if err != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	verificador, err := s.resolverComponenteV2(ctx, descriptor, domain.RolComponenteVerificador)
	if err != nil || !render.IndependienteDe(verificador) {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	limite := minimoUint64V2(
		s.techoInstitucional, perfil.MaximoBytes(), render.MaximoBytes(), verificador.MaximoBytes(),
	)
	limiteContenido, contenidoCanonico, contenidoSeguro, err :=
		prepararContenidoNeutralV2(contenido, limite)
	if err != nil || limiteContenido == 0 {
		limpiarBytesAplicacionV2(contenidoCanonico)
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	huellaEntrada, err := s.selladorDatos.SellarDatos(ctx, contenidoCanonico)
	limpiarBytesAplicacionV2(contenidoCanonico)
	if err != nil || ctx.Err() != nil || !huellaHMACAplicacionV2Valida(huellaEntrada) {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	referencia, err := s.generadorReferencia.NuevaReferenciaBorradorDocumental(ctx)
	if err != nil || !referenciaBorradorAplicacionV2Valida(referencia) || ctx.Err() != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	// Relectura inmediatamente anterior al efecto. Una revision historica no
	// sustituye la proyeccion operativa actual.
	preEfecto, err := s.leerSituacionActualV2(ctx, descriptor)
	if err != nil || preEfecto != situacion ||
		s.revalidarDescriptorPerfilV2(ctx, consulta, descriptor) != nil ||
		s.revalidarComponenteV2(ctx, descriptor, render) != nil ||
		s.revalidarComponenteV2(ctx, descriptor, verificador) != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	escritor := nuevoEscritorLimitadoDocumentalV2(limite)
	if err := s.renderizador.Renderizar(
		ctx, render, perfil, contenidoSeguro, limite, escritor,
	); err != nil || ctx.Err() != nil {
		escritor.CerrarSinContenido()
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	salida, err := escritor.CerrarYCopiar()
	if err != nil || len(salida) == 0 || uint64(len(salida)) > limite {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	huellaSalida := huellaBytesAplicacionV2(salida)
	// El documento puede contener datos personales. Antes de entregarlo al
	// verificador externo se vuelve a comprobar que su atestacion, el perfil y
	// la publicacion continuan siendo exactamente los autorizados.
	antesVerificacion, err := s.leerSituacionActualV2(ctx, descriptor)
	if err != nil || antesVerificacion != preEfecto ||
		s.revalidarDescriptorPerfilV2(ctx, consulta, descriptor) != nil ||
		s.revalidarComponenteV2(ctx, descriptor, verificador) != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	// El verificador recibe una copia desechable. Se comprueba ademas que no la
	// haya mutado durante la llamada; nunca recibe el slice autoritativo.
	copiaVerificacion := append([]byte(nil), salida...)
	if err := s.verificador.ValidarConformidad(
		ctx, verificador, perfil, copiaVerificacion, limite,
	); err != nil || ctx.Err() != nil || !bytes.Equal(copiaVerificacion, salida) ||
		huellaBytesAplicacionV2(salida) != huellaSalida {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	postEfecto, err := s.leerSituacionActualV2(ctx, descriptor)
	if err != nil || postEfecto != antesVerificacion ||
		s.revalidarDescriptorPerfilV2(ctx, consulta, descriptor) != nil ||
		s.revalidarComponenteV2(ctx, descriptor, render) != nil ||
		s.revalidarComponenteV2(ctx, descriptor, verificador) != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	generadoEn := s.reloj.Ahora().UTC()
	if generadoEn.IsZero() {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	evidencia, err := nuevaEvidenciaRenderizadoDocumentalV2(
		DatosEvidenciaRenderizadoDocumentalV2{
			Consulta: consulta, DescriptorPerfil: descriptor, SituacionOperativa: postEfecto,
			ComponenteRender: render, ComponenteVerificador: verificador,
			TechoInstitucionalBytes: s.techoInstitucional,
			LimiteEfectivoBytes:     limite, HuellaEntradaHMAC: huellaEntrada,
			BorradorRef: referencia, HuellaSalidaSHA256: huellaSalida,
			TamanoSalida: uint64(len(salida)), GeneradoEn: generadoEn,
		},
	)
	if err != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	borrador := ArtefactoBorradorPreFirmaV2{
		referencia: referencia, contenido: salida,
		perfilRef: perfil.Referencia(), digestPerfil: perfil.DigestSHA256(), mime: perfil.MIME(),
		huella: huellaSalida, generadoEn: generadoEn, evidencia: evidencia,
	}
	if borrador.Validar() != nil {
		return ArtefactoBorradorPreFirmaV2{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return borrador, nil
}

func (s *ServicioEjecucionFormatoDocumentalV2) resolverDescriptorPerfilV2(
	ctx context.Context,
	consulta ports.ConsultaFormatoDocumental,
) (ports.DescriptorPerfilDocumental, error) {
	descriptores, err := s.catalogoPerfiles.BuscarDescriptoresPerfilDocumental(ctx, consulta)
	if err != nil || ctx.Err() != nil || len(descriptores) != 1 ||
		descriptores[0].Validar() != nil || !descriptores[0].Coincide(consulta) {
		return ports.DescriptorPerfilDocumental{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return descriptores[0], nil
}

func (s *ServicioEjecucionFormatoDocumentalV2) revalidarDescriptorPerfilV2(
	ctx context.Context,
	consulta ports.ConsultaFormatoDocumental,
	esperado ports.DescriptorPerfilDocumental,
) error {
	actual, err := s.resolverDescriptorPerfilV2(ctx, consulta)
	if err != nil || actual != esperado {
		return ErrEjecucionFormatoDocumentalCerrada
	}
	return nil
}

func (s *ServicioEjecucionFormatoDocumentalV2) leerSituacionActualV2(
	ctx context.Context,
	descriptor ports.DescriptorPerfilDocumental,
) (domain.SituacionOperativaPerfilDocumental, error) {
	consulta := consultaSituacionDesdeDescriptorV2(descriptor)
	situaciones, err := s.situaciones.BuscarSituacionesOperativasActuales(ctx, consulta)
	if err != nil || ctx.Err() != nil || len(situaciones) != 1 ||
		!consulta.Coincide(situaciones[0]) ||
		!situaciones[0].AutorizaEjecucion(descriptor.Perfil(), descriptor.Revision()) {
		return domain.SituacionOperativaPerfilDocumental{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return situaciones[0], nil
}

func (s *ServicioEjecucionFormatoDocumentalV2) resolverComponenteV2(
	ctx context.Context,
	descriptor ports.DescriptorPerfilDocumental,
	rol domain.RolComponenteDocumental,
) (ports.DescriptorComponenteDocumentalAtestado, error) {
	consulta := consultaComponenteDesdeDescriptorV2(descriptor, rol)
	componentes, err := s.componentes.BuscarComponentesDocumentalesAtestados(ctx, consulta)
	if err != nil || ctx.Err() != nil || len(componentes) != 1 ||
		componentes[0].Validar() != nil || !componentes[0].Coincide(consulta) {
		return ports.DescriptorComponenteDocumentalAtestado{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return componentes[0], nil
}

func (s *ServicioEjecucionFormatoDocumentalV2) revalidarComponenteV2(
	ctx context.Context,
	descriptor ports.DescriptorPerfilDocumental,
	esperado ports.DescriptorComponenteDocumentalAtestado,
) error {
	actual, err := s.resolverComponenteV2(ctx, descriptor, esperado.Componente().Rol())
	if err != nil || actual != esperado {
		return ErrEjecucionFormatoDocumentalCerrada
	}
	return nil
}

func consultaSituacionDesdeDescriptorV2(
	descriptor ports.DescriptorPerfilDocumental,
) ports.ConsultaSituacionOperativaActual {
	return ports.ConsultaSituacionOperativaActual{
		PublicacionRef: descriptor.PublicacionRef(), PerfilRef: descriptor.Perfil().Referencia(),
		DigestPerfil: descriptor.Perfil().DigestSHA256(), RevisionCatalogo: descriptor.Revision(),
	}
}

func consultaComponenteDesdeDescriptorV2(
	descriptor ports.DescriptorPerfilDocumental,
	rol domain.RolComponenteDocumental,
) ports.ConsultaComponenteDocumentalAtestado {
	return ports.ConsultaComponenteDocumentalAtestado{
		Rol: rol, DescriptorPerfilRef: descriptor.Referencia(),
		PublicacionRef: descriptor.PublicacionRef(), PerfilRef: descriptor.Perfil().Referencia(),
		DigestPerfil: descriptor.Perfil().DigestSHA256(), RevisionCatalogo: descriptor.Revision(),
	}
}

func prepararContenidoNeutralV2(
	contenido domain.ContenidoDocumento,
	limite uint64,
) (uint64, []byte, domain.ContenidoDocumento, error) {
	if limite == 0 || len(contenido.Parrafos) > 100_000 ||
		(!utf8.ValidString(contenido.Titulo) || strings.ContainsRune(contenido.Titulo, '\x00')) {
		return 0, nil, domain.ContenidoDocumento{}, ErrEjecucionFormatoDocumentalCerrada
	}
	tamano := uint64(len(contenido.Titulo))
	if tamano == 0 && len(contenido.Parrafos) == 0 {
		return 0, nil, domain.ContenidoDocumento{}, ErrEjecucionFormatoDocumentalCerrada
	}
	if tamano > limite {
		return 0, nil, domain.ContenidoDocumento{}, ErrLimiteFormatoDocumentalExcedido
	}
	// Solo se copian primero las cabeceras de cadena, con un maximo fijo de
	// 100.000 elementos. Los bytes potencialmente grandes no se duplican hasta
	// haber validado el limite efectivo completo.
	parrafos := make([]string, len(contenido.Parrafos))
	copy(parrafos, contenido.Parrafos)
	for _, parrafo := range parrafos {
		if !utf8.ValidString(parrafo) || strings.ContainsRune(parrafo, '\x00') {
			return 0, nil, domain.ContenidoDocumento{}, ErrEjecucionFormatoDocumentalCerrada
		}
		longitud := uint64(len(parrafo))
		if longitud > ^uint64(0)-tamano-8 {
			return 0, nil, domain.ContenidoDocumento{}, ErrEjecucionFormatoDocumentalCerrada
		}
		tamano += longitud + 8
		if tamano > limite {
			return 0, nil, domain.ContenidoDocumento{}, ErrLimiteFormatoDocumentalExcedido
		}
	}
	copia := domain.ContenidoDocumento{
		Titulo:   strings.Clone(contenido.Titulo),
		Parrafos: make([]string, len(parrafos)),
	}
	for indice, parrafo := range parrafos {
		copia.Parrafos[indice] = strings.Clone(parrafo)
	}
	canonico := serializarContenidoNeutralV2(copia)
	if len(canonico) == 0 {
		return 0, nil, domain.ContenidoDocumento{}, ErrEjecucionFormatoDocumentalCerrada
	}
	return tamano, canonico, copia, nil
}

func serializarContenidoNeutralV2(contenido domain.ContenidoDocumento) []byte {
	salida := make([]byte, 0, len(contenido.Titulo)+len(contenido.Parrafos)*24+128)
	salida = anexarCampoCanonicoBytesV2(
		salida, "esquema", "vec.contenido-documental-neutral.v2",
	)
	salida = anexarCampoCanonicoBytesV2(salida, "titulo", contenido.Titulo)
	salida = anexarCampoCanonicoBytesV2(
		salida, "numero_parrafos", strconv.Itoa(len(contenido.Parrafos)),
	)
	for indice, parrafo := range contenido.Parrafos {
		salida = anexarCampoCanonicoBytesV2(
			salida, "parrafo_"+strconv.Itoa(indice), parrafo,
		)
	}
	return salida
}

func anexarCampoCanonicoBytesV2(destino []byte, clave, valor string) []byte {
	destino = strconv.AppendInt(destino, int64(len(clave)), 10)
	destino = append(destino, ':')
	destino = append(destino, clave...)
	destino = append(destino, '=')
	destino = strconv.AppendInt(destino, int64(len(valor)), 10)
	destino = append(destino, ':')
	destino = append(destino, valor...)
	return append(destino, '\n')
}

func limpiarBytesAplicacionV2(contenido []byte) {
	for indice := range contenido {
		contenido[indice] = 0
	}
}

func huellaDatosEvidenciaRenderizadoV2(d DatosEvidenciaRenderizadoDocumentalV2) string {
	perfil := d.DescriptorPerfil.Perfil()
	revision := d.DescriptorPerfil.Revision()
	situacion := d.SituacionOperativa
	render := d.ComponenteRender
	verificador := d.ComponenteVerificador
	return huellaCanonicaAplicacionV2([]string{
		"vec.evidencia-renderizado-documental.v2", d.Consulta.Identidad.Identificador(),
		d.Consulta.PerfilRef.Identificador(), strconv.FormatUint(d.Consulta.PerfilRef.Version(), 10),
		d.Consulta.DigestPerfilSHA256, strconv.FormatUint(d.Consulta.RevisionCatalogo.Numero(), 10),
		d.Consulta.RevisionCatalogo.HuellaSHA256(), d.DescriptorPerfil.Referencia(),
		d.DescriptorPerfil.PublicacionRef(), perfil.DigestSHA256(),
		strconv.FormatUint(revision.Numero(), 10), revision.HuellaSHA256(),
		situacion.PublicacionRef(), strconv.FormatUint(situacion.RevisionOperativa(), 10),
		string(situacion.Estado()), situacion.HuellaSHA256(),
		render.Referencia(), render.DigestDeclaracionSHA256(),
		verificador.Referencia(), verificador.DigestDeclaracionSHA256(),
		strconv.FormatUint(d.TechoInstitucionalBytes, 10),
		strconv.FormatUint(d.LimiteEfectivoBytes, 10), d.HuellaEntradaHMAC,
		d.BorradorRef, d.HuellaSalidaSHA256, strconv.FormatUint(d.TamanoSalida, 10),
		d.GeneradoEn.Format(time.RFC3339Nano),
	})
}

func huellaCanonicaAplicacionV2(valores []string) string {
	calculador := sha256.New()
	for _, valor := range valores {
		_, _ = calculador.Write([]byte(strconv.Itoa(len(valor))))
		_, _ = calculador.Write([]byte{':'})
		_, _ = calculador.Write([]byte(valor))
		_, _ = calculador.Write([]byte{'\n'})
	}
	return hex.EncodeToString(calculador.Sum(nil))
}

func huellaBytesAplicacionV2(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

func esSHA256AplicacionV2(valor string) bool {
	if len(valor) != 64 || valor != strings.TrimSpace(valor) || valor != strings.ToLower(valor) {
		return false
	}
	decodificado, err := hex.DecodeString(valor)
	return err == nil && len(decodificado) == sha256.Size
}

func huellaHMACAplicacionV2Valida(valor string) bool {
	if len(valor) == 0 || len(valor) > 512 || valor != strings.TrimSpace(valor) ||
		strings.ContainsRune(valor, '*') {
		return false
	}
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" ||
		!idClaveHMACDocumentalV2Valido.MatchString(partes[1]) ||
		len(partes[2]) != sha256.Size*2 || partes[2] != strings.ToLower(partes[2]) {
		return false
	}
	decodificada, err := hex.DecodeString(partes[2])
	return err == nil && len(decodificada) == sha256.Size
}

func referenciaBorradorAplicacionV2Valida(valor string) bool {
	return referenciaBorradorDocumentalValida.MatchString(valor) &&
		!strings.ContainsRune(valor, '*')
}

func minimoUint64V2(valores ...uint64) uint64 {
	if len(valores) == 0 {
		return 0
	}
	minimo := valores[0]
	for _, valor := range valores[1:] {
		if valor < minimo {
			minimo = valor
		}
	}
	return minimo
}

type escritorLimitadoDocumentalV2 struct {
	mu       sync.Mutex
	limite   uint64
	buffer   bytes.Buffer
	cerrado  bool
	excedido bool
}

func nuevoEscritorLimitadoDocumentalV2(limite uint64) *escritorLimitadoDocumentalV2 {
	return &escritorLimitadoDocumentalV2{limite: limite}
}

func (e *escritorLimitadoDocumentalV2) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cerrado || e.excedido {
		return 0, io.ErrClosedPipe
	}
	actual := uint64(e.buffer.Len())
	if actual > e.limite || uint64(len(p)) > e.limite-actual {
		e.excedido = true
		return 0, ErrLimiteFormatoDocumentalExcedido
	}
	return e.buffer.Write(p)
}

func (e *escritorLimitadoDocumentalV2) CerrarYCopiar() ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cerrado = true
	if e.excedido || e.buffer.Len() == 0 || uint64(e.buffer.Len()) > e.limite {
		e.buffer.Reset()
		return nil, ErrLimiteFormatoDocumentalExcedido
	}
	contenido := append([]byte(nil), e.buffer.Bytes()...)
	e.buffer.Reset()
	return contenido, nil
}

func (e *escritorLimitadoDocumentalV2) CerrarSinContenido() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cerrado = true
	e.buffer.Reset()
}
