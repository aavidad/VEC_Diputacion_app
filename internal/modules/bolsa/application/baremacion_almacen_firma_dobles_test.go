package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"sync"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type almacenBaremacionPrueba struct {
	ahora                time.Time
	ultima               puertosvec.SolicitudEscribirObjeto
	firmable             puertosvec.SolicitudEscribirObjeto
	resultado            puertosvec.ResultadoOperacionObjeto
	invocacionesEscribir int
	invocacionesRetener  int
	escrituras           int
	retenciones          int
	errorRetencion       error
}

func (a *almacenBaremacionPrueba) Capacidades(context.Context) (puertosvec.CapacidadesAlmacenObjetos, error) {
	return puertosvec.CapacidadesAlmacenObjetos{
		ConectorID: "almacen-cifrado-prueba", EscrituraEnFlujo: true, LecturaEnFlujo: true,
		ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true, Retencion: true,
		BloqueoLegal: true, CifradoEnTransito: true, CifradoEnReposo: true, CifradoPorObjeto: true,
		TamanoMaximoObjeto: 64 << 20, PreservaObjetoOriginal: true,
	}, nil
}

func (a *almacenBaremacionPrueba) Escribir(
	_ context.Context,
	s puertosvec.SolicitudEscribirObjeto,
) (puertosvec.ResultadoOperacionObjeto, error) {
	a.invocacionesEscribir++
	if s.Validar() != nil {
		return puertosvec.ResultadoOperacionObjeto{}, puertosvec.ErrSolicitudAlmacenInvalida
	}
	contexto, err := s.Contexto.Proyeccion()
	if err != nil {
		return puertosvec.ResultadoOperacionObjeto{}, puertosvec.ErrSolicitudAlmacenInvalida
	}
	_, _ = io.Copy(io.Discard, s.Contenido)
	a.escrituras++
	a.ultima = s
	if a.escrituras == 1 {
		a.firmable = s
	}
	sufijo := strings.Repeat("x", a.escrituras)
	objeto := puertosvec.ReferenciaObjetoAlmacen{Referencia: "objeto:decision:" + sufijo, Version: "version:" + sufijo}
	evidencia := puertosvec.EvidenciaOperacionAlmacen{
		Referencia: "evidencia:custodia:" + sufijo, ConectorID: "almacen-cifrado-prueba",
		EsquemaContexto: contexto.Esquema, AccionNegocio: contexto.AccionNegocio,
		Accion: contexto.AccionTecnica, EfectoRef: contexto.EfectoRef,
		HuellaPlanEfectoSHA256: contexto.HuellaPlanEfectoSHA256, PasoRef: contexto.PasoRef,
		HuellaDecisionSHA256: contexto.HuellaDecisionSHA256, Objeto: objeto,
		OperacionRef: contexto.OperacionRef, CorrelacionRef: contexto.CorrelacionRef,
		AutorizacionRef: contexto.AutorizacionRef, Finalidad: contexto.Finalidad,
		Clasificacion: contexto.Clasificacion, RealizadaEn: a.ahora,
		CargaRef: contexto.CargaRef, SujetoSeudonimoHMAC: contexto.SujetoSeudonimoHMAC,
		RecursoRef: contexto.RecursoRef, ModuloID: contexto.ModuloID,
		HuellaSolicitudHMAC: contexto.HuellaSolicitudHMAC,
	}
	resultado := puertosvec.ResultadoOperacionObjeto{
		Objeto: puertosvec.ObjetoAlmacenado{
			Objeto: objeto, ConectorID: "almacen-cifrado-prueba", Zona: s.Zona, MIME: s.MIME,
			Tamano: s.Tamano, HuellaSHA256: s.HuellaSHA256,
			EvidenciaCreacionRef: evidencia.Referencia, AlmacenadoEn: a.ahora,
		},
		Evidencia: evidencia,
	}
	a.resultado = resultado
	return resultado, resultado.Validar()
}

func (*almacenBaremacionPrueba) Abrir(context.Context, puertosvec.SolicitudAbrirObjeto) (puertosvec.LecturaObjetoAlmacen, error) {
	return puertosvec.LecturaObjetoAlmacen{}, puertosvec.ErrObjetoAlmacenNoEncontrado
}
func (*almacenBaremacionPrueba) Promover(context.Context, puertosvec.SolicitudPromoverObjeto) (puertosvec.ResultadoOperacionObjeto, error) {
	return puertosvec.ResultadoOperacionObjeto{}, puertosvec.ErrTransicionZonaAlmacenNoPermitida
}
func (a *almacenBaremacionPrueba) AplicarRetencion(_ context.Context, s puertosvec.SolicitudRetenerObjeto) (puertosvec.ResultadoOperacionObjeto, error) {
	a.invocacionesRetener++
	if s.ValidarEn(a.ahora) != nil || a.resultado.Objeto.Objeto != s.Objeto {
		return puertosvec.ResultadoOperacionObjeto{}, puertosvec.ErrSolicitudAlmacenInvalida
	}
	contexto, err := s.Contexto.Proyeccion()
	if err != nil {
		return puertosvec.ResultadoOperacionObjeto{}, puertosvec.ErrSolicitudAlmacenInvalida
	}
	a.retenciones++
	if a.errorRetencion != nil {
		return puertosvec.ResultadoOperacionObjeto{}, a.errorRetencion
	}
	objeto := a.resultado.Objeto
	objeto.RetenidoHasta = s.Hasta
	evidencia := puertosvec.EvidenciaOperacionAlmacen{
		Referencia: "evidencia:retencion:firmado:1", ConectorID: objeto.ConectorID,
		EsquemaContexto: contexto.Esquema, AccionNegocio: contexto.AccionNegocio,
		Accion: contexto.AccionTecnica, EfectoRef: contexto.EfectoRef,
		HuellaPlanEfectoSHA256: contexto.HuellaPlanEfectoSHA256, PasoRef: contexto.PasoRef,
		HuellaDecisionSHA256: contexto.HuellaDecisionSHA256, Objeto: objeto.Objeto,
		OperacionRef: contexto.OperacionRef, CorrelacionRef: contexto.CorrelacionRef,
		AutorizacionRef: contexto.AutorizacionRef, Finalidad: contexto.Finalidad,
		Clasificacion: contexto.Clasificacion, RealizadaEn: a.ahora,
		CargaRef: contexto.CargaRef, SujetoSeudonimoHMAC: contexto.SujetoSeudonimoHMAC,
		RecursoRef: contexto.RecursoRef, ModuloID: contexto.ModuloID,
		HuellaSolicitudHMAC: contexto.HuellaSolicitudHMAC, FundamentoRef: s.PoliticaRef,
	}
	resultado := puertosvec.ResultadoOperacionObjeto{Objeto: objeto, Evidencia: evidencia}
	a.resultado = resultado
	return resultado, resultado.ValidarRetencion(s, puertosvec.ObjetoAlmacenado{
		Objeto: objeto.Objeto, ConectorID: objeto.ConectorID, Zona: objeto.Zona, MIME: objeto.MIME,
		Tamano: objeto.Tamano, HuellaSHA256: objeto.HuellaSHA256,
		EvidenciaCreacionRef: objeto.EvidenciaCreacionRef, AlmacenadoEn: objeto.AlmacenadoEn,
	})
}
func (*almacenBaremacionPrueba) Inmovilizar(context.Context, puertosvec.SolicitudInmovilizarObjeto) (puertosvec.ResultadoOperacionObjeto, error) {
	return puertosvec.ResultadoOperacionObjeto{}, puertosvec.ErrSolicitudAlmacenInvalida
}
func (*almacenBaremacionPrueba) LevantarInmovilizacion(context.Context, puertosvec.SolicitudLevantarInmovilizacionObjeto) (puertosvec.ResultadoOperacionObjeto, error) {
	return puertosvec.ResultadoOperacionObjeto{}, puertosvec.ErrSolicitudAlmacenInvalida
}
func (*almacenBaremacionPrueba) Eliminar(context.Context, puertosvec.SolicitudEliminarObjeto) (puertosvec.EvidenciaOperacionAlmacen, error) {
	return puertosvec.EvidenciaOperacionAlmacen{}, puertosvec.ErrSolicitudAlmacenInvalida
}

type firmadorBaremacionPrueba struct {
	mu        sync.Mutex
	ahora     time.Time
	solicitud puertosbolsa.SolicitudPrepararFirmaInteractiva
	pendiente bool
}

func (f *firmadorBaremacionPrueba) PrepararFirmaInteractiva(
	_ context.Context,
	s puertosbolsa.SolicitudPrepararFirmaInteractiva,
) (puertosbolsa.SesionFirmaInteractiva, error) {
	if s.Validar() != nil {
		return puertosbolsa.SesionFirmaInteractiva{}, puertosbolsa.ErrFirmaInteractivaNoDisponible
	}
	f.mu.Lock()
	f.solicitud = s
	f.mu.Unlock()
	carga, _ := puertosbolsa.NuevaCargaProtegida([]byte("lanzamiento-autofirma"))
	sesion := puertosbolsa.SesionFirmaInteractiva{
		SesionRef: "sesion:firma:1", Estado: puertosbolsa.EstadoSesionFirmaPreparada,
		CargaLanzamiento: carga, MIMELanzamiento: "application/json", Documento: s.Documento,
		PoliticaFirmaRef: s.Politica.Referencia, PoliticaFirmaVersion: s.Politica.Version,
		HuellaPoliticaSHA256: s.Politica.HuellaSHA256, EvidenciaPreparacionRef: "evidencia:preparacion:firma:1",
		HuellaEvidenciaSHA256: huellaBaremacionPrueba("7"), PreparadaEn: f.ahora, ExpiraEn: s.ExpiraEn,
	}
	return sesion, nil
}

func (f *firmadorBaremacionPrueba) ConsultarFirmaInteractiva(
	_ context.Context,
	s puertosbolsa.SolicitudConsultarFirmaInteractiva,
) (puertosbolsa.ConsultaFirmaInteractiva, error) {
	if s.Validar() != nil {
		return puertosbolsa.ConsultaFirmaInteractiva{}, puertosbolsa.ErrFirmaInteractivaNoDisponible
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.pendiente {
		return puertosbolsa.ConsultaFirmaInteractiva{
			SesionRef: s.SesionRef, Estado: puertosbolsa.EstadoSesionFirmaPendiente,
			EvidenciaConsultaRef:  "evidencia:consulta:firma:pendiente",
			HuellaEvidenciaSHA256: huellaBaremacionPrueba("8"), ConsultadaEn: f.ahora,
		}, nil
	}
	documento := f.solicitud.Documento
	binarioFirmado := contenidoDocumentoFirmadoBaremacionPrueba()
	huellaBinario := sha256.Sum256(binarioFirmado)
	artefacto := puertosbolsa.ArtefactoFirma{
		ProcesoRef: documento.ProcesoRef, SolicitudRef: documento.SolicitudRef, SujetoRef: documento.SujetoRef,
		BaremacionMeritoRef: documento.BaremacionMeritoRef, DecisionRef: documento.DecisionRef,
		VersionBaremacion: documento.VersionBaremacion, SesionFirmaRef: s.SesionRef,
		EvidenciaFirmaInteractivaRef:     "evidencia:firma:interactiva:1",
		HuellaEvidenciaInteractivaSHA256: huellaBaremacionPrueba("9"),
		DocumentoFirmable:                documento.Objeto.Objeto, HuellaDocumentoFirmableSHA256: documento.HuellaDocumentoSHA256,
		EvidenciaCustodiaRef: documento.EvidenciaCustodia.Referencia, FirmaRef: "firma:decision:1",
		HuellaFirmaSHA256: huellaBaremacionPrueba("a"), DocumentoFirmadoRef: "documento:firmado:1",
		HuellaDocumentoSHA256: hex.EncodeToString(huellaBinario[:]), HuellaContenidoSHA256: documento.HuellaContenidoSHA256,
		PoliticaFirmaRef: s.PoliticaFirmaRef, PoliticaFirmaVersion: s.PoliticaFirmaVersion,
		HuellaPoliticaFirmaSHA256: s.HuellaPoliticaSHA256, FirmanteRef: s.FirmanteRef,
		PerfilFirmanteClave: s.PerfilFirmanteClave, FirmadaEn: f.ahora,
	}
	return puertosbolsa.ConsultaFirmaInteractiva{
		SesionRef: s.SesionRef, Estado: puertosbolsa.EstadoSesionFirmaCompletada, Artefacto: &artefacto,
		EvidenciaConsultaRef:  "evidencia:consulta:firma:1",
		HuellaEvidenciaSHA256: huellaBaremacionPrueba("c"), ConsultadaEn: f.ahora,
	}, nil
}

type recuperadorBinarioBaremacionPrueba struct {
	ahora                 time.Time
	invocacionesRecuperar int
	llamadas              int
}

func (r *recuperadorBinarioBaremacionPrueba) RecuperarBinarioFirmado(
	_ context.Context,
	s puertosbolsa.SolicitudRecuperarBinarioFirmado,
) (puertosbolsa.BinarioFirmadoRecuperado, error) {
	r.invocacionesRecuperar++
	if s.Validar() != nil {
		return puertosbolsa.BinarioFirmadoRecuperado{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.llamadas++
	contenido := contenidoDocumentoFirmadoSegunReferenciaBaremacionPrueba(s.DocumentoFirmadoRef)
	return puertosbolsa.BinarioFirmadoRecuperado{
		DocumentoFirmadoRef: s.DocumentoFirmadoRef, HuellaDocumentoSHA256: s.HuellaDocumentoSHA256,
		MIME: "application/pdf", Tamano: int64(len(contenido)), Contenido: io.NopCloser(strings.NewReader(string(contenido))),
		EvidenciaRecuperacionRef: "evidencia:recuperacion:documento:firmado:1",
		HuellaEvidenciaSHA256:    huellaBaremacionPrueba("b"), RecuperadoEn: instanteBaremacionPrueba,
	}, nil
}

func contenidoDocumentoFirmadoBaremacionPrueba() []byte {
	return []byte("%PDF-1.7\ndocumento-decision-baremacion-firmado")
}

func contenidoDocumentoFirmadoSelladoBaremacionPrueba() []byte {
	return []byte("%PDF-1.7\ndocumento-decision-baremacion-firmado-pades-t")
}

func contenidoDocumentoFirmadoLongevoBaremacionPrueba() []byte {
	return []byte("%PDF-1.7\ndocumento-decision-baremacion-firmado-pades-lta")
}

func contenidoDocumentoFirmadoSegunReferenciaBaremacionPrueba(referencia string) []byte {
	switch {
	case strings.HasSuffix(referencia, ":pades-lta"):
		return contenidoDocumentoFirmadoLongevoBaremacionPrueba()
	case strings.HasSuffix(referencia, ":pades-t"):
		return contenidoDocumentoFirmadoSelladoBaremacionPrueba()
	default:
		return contenidoDocumentoFirmadoBaremacionPrueba()
	}
}

type validadorBaremacionPrueba struct {
	mu       sync.Mutex
	ahora    time.Time
	llamadas int
}

func (v *validadorBaremacionPrueba) ValidarFirmaServidor(
	_ context.Context,
	s puertosbolsa.SolicitudValidarFirmaServidor,
) (puertosbolsa.ValidacionFirmaServidor, error) {
	if s.Validar() != nil {
		return puertosbolsa.ValidacionFirmaServidor{}, puertosbolsa.ErrValidacionFirmaNoConcluyente
	}
	v.mu.Lock()
	v.llamadas++
	numero := v.llamadas
	v.mu.Unlock()
	comprobaciones := make([]puertosbolsa.ComprobacionFirma, 0, len(s.Politica.ComprobacionesObligatorias))
	for indice, clave := range s.Politica.ComprobacionesObligatorias {
		comprobaciones = append(comprobaciones, puertosbolsa.ComprobacionFirma{
			Clave: clave, Estado: puertosbolsa.EstadoComprobacionSuperada,
			EvidenciaRef:          "evidencia:validacion:" + strings.Repeat("x", indice+1),
			HuellaEvidenciaSHA256: huellaBaremacionPrueba("e"),
		})
	}
	resultado := puertosbolsa.ValidacionFirmaServidor{
		Estado: puertosbolsa.EstadoValidacionFirmaValida, Artefacto: s.Artefacto,
		ValidacionRef:          "validacion:firma:" + strings.Repeat("v", numero),
		HuellaValidacionSHA256: huellaBaremacionPrueba("d"), FirmanteVerificadoRef: s.FirmanteEsperadoRef,
		PerfilVerificadoClave: s.PerfilEsperadoClave, PerfilFirmaVerificadoClave: s.PerfilFirmaEsperadoClave,
		SelloTiempoVerificadoRef:                s.SelloTiempoEsperadoRef,
		HuellaSelloTiempoVerificadaSHA256:       s.HuellaSelloTiempoEsperadaSHA256,
		AumentoLongevidadVerificadoRef:          s.AumentoLongevidadEsperadoRef,
		HuellaAumentoLongevidadVerificadaSHA256: s.HuellaAumentoLongevidadEsperadaSHA256,
		Comprobaciones:                          comprobaciones, ValidadaEn: v.ahora,
	}
	return resultado, nil
}
