package application

import (
	"context"
	"errors"
	"slices"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type aumentadorNoInvocadoPAdESTPrueba struct{ llamadas int }

func (a *aumentadorNoInvocadoPAdESTPrueba) AumentarFirma(
	context.Context,
	puertosbolsa.SolicitudAumentarFirma,
) (puertosbolsa.ResultadoAumentoFirma, error) {
	a.llamadas++
	return puertosbolsa.ResultadoAumentoFirma{}, puertosbolsa.ErrAumentoFirmaNoDisponible
}

type selladorNoOpPAdESTPrueba struct {
	base          selladorTiempoActivoBaremacionPrueba
	llamadas      int
	sustituciones int
}

func (s *selladorNoOpPAdESTPrueba) SellarTiempoFirma(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudSellarTiempoFirma,
) (puertosbolsa.SelloTiempoFirma, error) {
	s.llamadas++
	resultado, err := s.base.SellarTiempoFirma(ctx, solicitud)
	if err == nil {
		s.sustituciones++
		resultado.ArtefactoSellado = resultado.ArtefactoOrigen
	}
	return resultado, err
}

type validadorPerfilBEnPAdESTPrueba struct {
	base          *validadorBaremacionPrueba
	llamadas      int
	sustituciones int
	perfiles      []string
}

func (v *validadorPerfilBEnPAdESTPrueba) ValidarFirmaServidor(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudValidarFirmaServidor,
) (puertosbolsa.ValidacionFirmaServidor, error) {
	v.llamadas++
	v.perfiles = append(v.perfiles, solicitud.PerfilFirmaEsperadoClave)
	resultado, err := v.base.ValidarFirmaServidor(ctx, solicitud)
	if err == nil && solicitud.PerfilFirmaEsperadoClave == puertosbolsa.PerfilFirmaPAdESBaselineT {
		v.sustituciones++
		resultado.PerfilFirmaVerificadoClave = puertosbolsa.PerfilFirmaPAdESBaselineB
	}
	return resultado, err
}

type validadorSelloAjenoPAdESTPrueba struct {
	base          *validadorBaremacionPrueba
	llamadas      int
	sustituciones int
	perfiles      []string
}

func (v *validadorSelloAjenoPAdESTPrueba) ValidarFirmaServidor(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudValidarFirmaServidor,
) (puertosbolsa.ValidacionFirmaServidor, error) {
	v.llamadas++
	v.perfiles = append(v.perfiles, solicitud.PerfilFirmaEsperadoClave)
	resultado, err := v.base.ValidarFirmaServidor(ctx, solicitud)
	if err == nil && solicitud.PerfilFirmaEsperadoClave == puertosbolsa.PerfilFirmaPAdESBaselineT {
		v.sustituciones++
		resultado.SelloTiempoVerificadoRef = "sello-tiempo:ajeno"
		resultado.HuellaSelloTiempoVerificadaSHA256 = huellaBaremacionPrueba("a")
	}
	return resultado, err
}

type validadorAumentoAjenoPAdESLTAPrueba struct {
	base          *validadorBaremacionPrueba
	llamadas      int
	sustituciones int
	perfiles      []string
}

func (v *validadorAumentoAjenoPAdESLTAPrueba) ValidarFirmaServidor(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudValidarFirmaServidor,
) (puertosbolsa.ValidacionFirmaServidor, error) {
	v.llamadas++
	v.perfiles = append(v.perfiles, solicitud.PerfilFirmaEsperadoClave)
	resultado, err := v.base.ValidarFirmaServidor(ctx, solicitud)
	if err == nil && solicitud.PerfilFirmaEsperadoClave == puertosbolsa.PerfilFirmaPAdESBaselineLTA {
		v.sustituciones++
		resultado.AumentoLongevidadVerificadoRef = "evidencia:aumento:ajena"
		resultado.HuellaAumentoLongevidadVerificadaSHA256 = huellaBaremacionPrueba("a")
	}
	return resultado, err
}

type efectosPersistentesPAdESPrueba struct {
	recuperaciones   int
	escrituras       int
	retenciones      int
	reservas         int
	confirmaciones   int
	abandonos        int
	intentosAbandono int
}

func efectosPersistentesPAdES(entorno *entornoBaremacionPrueba) efectosPersistentesPAdESPrueba {
	return efectosPersistentesPAdESPrueba{
		recuperaciones:   entorno.recuperador.invocacionesRecuperar,
		escrituras:       entorno.almacen.invocacionesEscribir,
		retenciones:      entorno.almacen.invocacionesRetener,
		reservas:         entorno.repositorio.reservas,
		confirmaciones:   entorno.repositorio.confirmaciones,
		abandonos:        entorno.repositorio.abandonos,
		intentosAbandono: entorno.repositorio.intentosAbandono,
	}
}

func configurarPoliticaPAdESTPrueba(entorno *entornoBaremacionPrueba) {
	politica := entorno.politicas.politica
	politica.PerfilFirmaClave = puertosbolsa.PerfilFirmaPAdESBaselineT
	politica.RequiereSelloTiempo = true
	politica.PoliticaSelloTiempoRef = "politica-sello-tiempo"
	politica.PoliticaSelloTiempoVersion = 2
	politica.HuellaPoliticaSelloTiempoSHA256 = huellaBaremacionPrueba("3")
	entorno.politicas.politica = politica
}

func configurarPoliticaPAdESLTAPrueba(entorno *entornoBaremacionPrueba) {
	configurarPoliticaPAdESTPrueba(entorno)
	politica := entorno.politicas.politica
	politica.PerfilFirmaClave = puertosbolsa.PerfilFirmaPAdESBaselineLTA
	politica.RequiereAumentoLongevidad = true
	politica.NivelAumentoClave = "pades_lta"
	politica.PoliticaLongevidadRef = "politica-longevidad"
	politica.PoliticaLongevidadVersion = 3
	politica.HuellaPoliticaLongevidadSHA256 = huellaBaremacionPrueba("4")
	entorno.politicas.politica = politica
}

func TestServicioBaremacionMaterializaValidaYCustodiaPAdESBaselineT(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	configurarPoliticaPAdESTPrueba(entorno)
	sellador := &selladorTiempoActivoBaremacionPrueba{ahora: instanteBaremacionPrueba}
	aumentador := &aumentadorNoInvocadoPAdESTPrueba{}
	entorno.servicio.selladorTiempo = sellador
	entorno.servicio.aumentadorFirma = aumentador
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	orden := ordenFinalizarBaremacionPrueba(preparada, "pades-t")
	orden.ClaveIdempotenciaSello = "idempotencia:sello:pades-t"

	resultado, err := entorno.servicio.FinalizarFirma(context.Background(), orden)
	if err != nil {
		t.Fatalf("FinalizarFirma() PAdES-T error = %v", err)
	}
	if resultado.SelloTiempo == nil || resultado.ValidacionTrasSello == nil || resultado.Aumento != nil {
		t.Fatalf("cadena PAdES-T incompleta: sello=%v validacion_t=%v aumento=%v",
			resultado.SelloTiempo != nil, resultado.ValidacionTrasSello != nil, resultado.Aumento != nil)
	}
	artefactoT := resultado.SelloTiempo.ArtefactoSellado
	if entorno.validador.llamadas != 2 || sellador.llamadas != 1 || aumentador.llamadas != 0 ||
		entorno.recuperador.llamadas != 1 ||
		resultado.ValidacionTrasSello.PerfilFirmaVerificadoClave != puertosbolsa.PerfilFirmaPAdESBaselineT ||
		resultado.ValidacionFinal.ValidacionRef != resultado.ValidacionTrasSello.ValidacionRef ||
		resultado.DocumentoFirmado.DocumentoFirmadoRef != artefactoT.DocumentoFirmadoRef ||
		resultado.DocumentoFirmado.HuellaDocumentoSHA256 != artefactoT.HuellaDocumentoSHA256 ||
		resultado.DocumentoFirmado.Objeto.HuellaSHA256 != artefactoT.HuellaDocumentoSHA256 ||
		entorno.almacen.ultima.HuellaSHA256 != artefactoT.HuellaDocumentoSHA256 ||
		entorno.almacen.ultima.HuellaSHA256 == resultado.SelloTiempo.ArtefactoOrigen.HuellaDocumentoSHA256 ||
		resultado.Decision.Firma.PerfilFirmaAlcanzadoClave != puertosbolsa.PerfilFirmaPAdESBaselineT {
		t.Fatalf("no se custodio y acredito exactamente la revision T: %+v", resultado)
	}
	if artefactoT.DocumentoFirmadoRef == resultado.SelloTiempo.ArtefactoOrigen.DocumentoFirmadoRef ||
		artefactoT.HuellaDocumentoSHA256 == resultado.SelloTiempo.ArtefactoOrigen.HuellaDocumentoSHA256 {
		t.Fatal("PAdES-T reutilizo la referencia o la huella del PDF Baseline-B")
	}
}

func TestServicioBaremacionRechazaSelloPAdESTNoOpAntesDeCustodiar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	configurarPoliticaPAdESTPrueba(entorno)
	sellador := &selladorNoOpPAdESTPrueba{base: selladorTiempoActivoBaremacionPrueba{ahora: instanteBaremacionPrueba}}
	entorno.servicio.selladorTiempo = sellador
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	efectosPrevios := efectosPersistentesPAdES(entorno)
	orden := ordenFinalizarBaremacionPrueba(preparada, "pades-t-no-op")
	orden.ClaveIdempotenciaSello = "idempotencia:sello:pades-t-no-op"

	if _, err := entorno.servicio.FinalizarFirma(context.Background(), orden); !errors.Is(err, ErrResultadoBaremacionNoConfiable) {
		t.Fatalf("error sin centinela de resultado no confiable: %v", err)
	}
	if sellador.llamadas != 1 || sellador.sustituciones != 1 {
		t.Fatalf("el wrapper no acredito la etapa de sello: llamadas=%d sustituciones=%d",
			sellador.llamadas, sellador.sustituciones)
	}
	if efectos := efectosPersistentesPAdES(entorno); efectos != efectosPrevios {
		t.Fatalf("el sello no-op produjo efectos: antes=%+v despues=%+v", efectosPrevios, efectos)
	}
}

func TestServicioBaremacionRechazaPerfilBDeclaradoParaPAdESTAntesDeCustodiar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	configurarPoliticaPAdESTPrueba(entorno)
	entorno.servicio.selladorTiempo = &selladorTiempoActivoBaremacionPrueba{ahora: instanteBaremacionPrueba}
	validadorHostil := &validadorPerfilBEnPAdESTPrueba{base: entorno.validador}
	entorno.servicio.validadorFirma = validadorHostil
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	efectosPrevios := efectosPersistentesPAdES(entorno)
	orden := ordenFinalizarBaremacionPrueba(preparada, "pades-t-perfil-b")
	orden.ClaveIdempotenciaSello = "idempotencia:sello:pades-t-perfil-b"

	if _, err := entorno.servicio.FinalizarFirma(context.Background(), orden); !errors.Is(err, ErrResultadoBaremacionNoConfiable) {
		t.Fatalf("error sin centinela de resultado no confiable: %v", err)
	}
	esperados := []string{puertosbolsa.PerfilFirmaPAdESBaselineB, puertosbolsa.PerfilFirmaPAdESBaselineT}
	if validadorHostil.llamadas != 2 || validadorHostil.sustituciones != 1 ||
		!slices.Equal(validadorHostil.perfiles, esperados) {
		t.Fatalf("el wrapper no acredito etapas B,T exactas: llamadas=%d sustituciones=%d perfiles=%v",
			validadorHostil.llamadas, validadorHostil.sustituciones, validadorHostil.perfiles)
	}
	if efectos := efectosPersistentesPAdES(entorno); efectos != efectosPrevios {
		t.Fatalf("el perfil incorrecto produjo efectos: antes=%+v despues=%+v", efectosPrevios, efectos)
	}
}

func TestServicioBaremacionRechazaAtestacionTDeOtroSelloAntesDeCustodiar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	configurarPoliticaPAdESTPrueba(entorno)
	sellador := &selladorTiempoActivoBaremacionPrueba{ahora: instanteBaremacionPrueba}
	entorno.servicio.selladorTiempo = sellador
	validadorHostil := &validadorSelloAjenoPAdESTPrueba{base: entorno.validador}
	entorno.servicio.validadorFirma = validadorHostil
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	efectosPrevios := efectosPersistentesPAdES(entorno)
	orden := ordenFinalizarBaremacionPrueba(preparada, "pades-t-sello-ajeno")
	orden.ClaveIdempotenciaSello = "idempotencia:sello:pades-t-ajeno"

	if _, err := entorno.servicio.FinalizarFirma(context.Background(), orden); !errors.Is(err, ErrResultadoBaremacionNoConfiable) {
		t.Fatalf("error sin centinela de resultado no confiable: %v", err)
	}
	esperados := []string{puertosbolsa.PerfilFirmaPAdESBaselineB, puertosbolsa.PerfilFirmaPAdESBaselineT}
	if validadorHostil.llamadas != 2 || validadorHostil.sustituciones != 1 ||
		!slices.Equal(validadorHostil.perfiles, esperados) || sellador.llamadas != 1 {
		t.Fatalf("el wrapper no acredito etapas B,T exactas: llamadas=%d sustituciones=%d perfiles=%v",
			validadorHostil.llamadas, validadorHostil.sustituciones, validadorHostil.perfiles)
	}
	if efectos := efectosPersistentesPAdES(entorno); efectos != efectosPrevios {
		t.Fatalf("la sustitucion de sello produjo efectos: antes=%+v despues=%+v", efectosPrevios, efectos)
	}
}

func TestServicioBaremacionRechazaAtestacionLTADeOtroAumentoAntesDeCustodiar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	configurarPoliticaPAdESLTAPrueba(entorno)
	sellador := &selladorTiempoActivoBaremacionPrueba{ahora: instanteBaremacionPrueba}
	aumentador := &aumentadorActivoBaremacionPrueba{ahora: instanteBaremacionPrueba}
	entorno.servicio.selladorTiempo = sellador
	entorno.servicio.aumentadorFirma = aumentador
	validadorHostil := &validadorAumentoAjenoPAdESLTAPrueba{base: entorno.validador}
	entorno.servicio.validadorFirma = validadorHostil
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	efectosPrevios := efectosPersistentesPAdES(entorno)
	orden := ordenFinalizarBaremacionPrueba(preparada, "pades-lta-aumento-ajeno")
	orden.ClaveIdempotenciaSello = "idempotencia:sello:pades-lta-ajeno"
	orden.ClaveIdempotenciaAumento = "idempotencia:aumento:pades-lta-ajeno"

	if _, err := entorno.servicio.FinalizarFirma(context.Background(), orden); !errors.Is(err, ErrResultadoBaremacionNoConfiable) {
		t.Fatalf("error sin centinela de resultado no confiable: %v", err)
	}
	esperados := []string{
		puertosbolsa.PerfilFirmaPAdESBaselineB,
		puertosbolsa.PerfilFirmaPAdESBaselineT,
		puertosbolsa.PerfilFirmaPAdESBaselineLTA,
	}
	if validadorHostil.llamadas != 3 || validadorHostil.sustituciones != 1 ||
		!slices.Equal(validadorHostil.perfiles, esperados) || sellador.llamadas != 1 || aumentador.llamadas != 1 {
		t.Fatalf("el wrapper no acredito etapas B,T,LTA exactas: llamadas=%d sustituciones=%d perfiles=%v",
			validadorHostil.llamadas, validadorHostil.sustituciones, validadorHostil.perfiles)
	}
	if efectos := efectosPersistentesPAdES(entorno); efectos != efectosPrevios {
		t.Fatalf("la sustitucion de aumento produjo efectos: antes=%+v despues=%+v", efectosPrevios, efectos)
	}
}
