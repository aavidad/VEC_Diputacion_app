package ports

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type escenarioEjecucionDocumentalV3Prueba struct {
	instante            time.Time
	manifiesto          ManifiestoEjecucionDocumentalV3
	preparar            SolicitudPrepararEjecucionDocumentalV3
	consumo             ConsumoDecisionEjecucionDocumentalV3
	token               TokenCercadoEjecucionDocumentalV3
	verificacionCercado ResultadoVerificacionTokenCercadoDocumentalV3
	resultado           ResultadoEfectoRenderizadoDocumentalV3
	recibos             RecibosEjecucionDocumentalV3
	evidencia           DatosEvidenciaRenderizadoDocumentalV3
	sello               SelloEvidenciaDocumentalV3
	verificacion        ResultadoVerificacionEvidenciaDocumentalV3
	confirmacion        SolicitudConfirmarEjecucionDocumentalV3
}

func TestManifiestoEjecucionDocumentalV3EsCanonicoTripleEIdempotente(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	if err := escenario.manifiesto.Validar(); err != nil {
		t.Fatalf("manifiesto valido rechazado: %v", err)
	}
	datos, err := escenario.manifiesto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.LimiteEfectivoBytes != datos.ComponenteSemantico.MaximoBytes() ||
		datos.LimiteEfectivoBytes >= datos.ComponenteRender.MaximoBytes() ||
		datos.LimiteEfectivoBytes >= datos.ComponenteVerificador.MaximoBytes() {
		t.Fatal("el limite efectivo no quedo sujeto al menor de los tres componentes")
	}
	huella := datos.HuellaPlanSHA256
	datos.HuellaPlanSHA256 = strings.Repeat("f", 64)
	if escenario.manifiesto.Validar() != nil {
		t.Fatal("Datos altero el manifiesto opaco")
	}
	datosRestaurados, _ := escenario.manifiesto.Datos()
	restaurado := ManifiestoEjecucionDocumentalV3{datos: &datosRestaurados}
	if restaurado.datos == escenario.manifiesto.datos ||
		!manifiestosEjecucionDocumentalV3Coinciden(restaurado, escenario.manifiesto) ||
		huella != datosRestaurados.HuellaPlanSHA256 {
		t.Fatal("la igualdad semantica dependio de identidad de puntero")
	}

	preparacion := PreparacionEjecucionDocumentalV3{
		ReservaRef: "reserva:documental:v3:001", BorradorRef: datosRestaurados.BorradorRef,
		EfectoRef: datosRestaurados.EfectoRef, Estado: EstadoEjecucionDocumentalV3Preparada,
	}
	if err := preparacion.ValidarContra(escenario.preparar); err != nil {
		t.Fatalf("preparacion exacta rechazada: %v", err)
	}
	cruzada := preparacion
	cruzada.EfectoRef = "efecto:documental:v3:otro"
	if cruzada.ValidarContra(escenario.preparar) == nil {
		t.Fatal("preparacion acepto otra referencia de efecto")
	}

	base, _ := escenario.manifiesto.Datos()
	if _, err := NuevoManifiestoEjecucionDocumentalV3(
		base.Consulta, base.DescriptorPerfil, base.SituacionOperativa,
		base.ComponenteRender, base.ComponenteVerificador, base.ComponenteVerificador,
		base.BorradorRef, base.EfectoRef, base.HuellaEntradaHMAC, base.LimiteEfectivoBytes,
	); !errors.Is(err, ErrManifiestoEjecucionDocumentalV3Invalido) {
		t.Fatalf("componente semantico no independiente aceptado: %v", err)
	}
	if _, err := NuevoManifiestoEjecucionDocumentalV3(
		base.Consulta, base.DescriptorPerfil, base.SituacionOperativa,
		base.ComponenteRender, base.ComponenteVerificador, base.ComponenteSemantico,
		"dni:12345678z", base.EfectoRef, base.HuellaEntradaHMAC, base.LimiteEfectivoBytes,
	); !errors.Is(err, ErrManifiestoEjecucionDocumentalV3Invalido) {
		t.Fatalf("PII evidente aceptada como referencia: %v", err)
	}
	if _, err := NuevoManifiestoEjecucionDocumentalV3(
		base.Consulta, base.DescriptorPerfil, base.SituacionOperativa,
		base.ComponenteRender, base.ComponenteVerificador, base.ComponenteSemantico,
		base.BorradorRef, base.EfectoRef, base.HuellaEntradaHMAC,
		base.ComponenteSemantico.MaximoBytes()+1,
	); !errors.Is(err, ErrManifiestoEjecucionDocumentalV3Invalido) {
		t.Fatalf("techo semantico excedido: %v", err)
	}
}

func TestTokenCercadoV3ExigeVerificacionYNoSeSerializa(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	texto := escenario.token.String() + escenario.token.GoString()
	if strings.Contains(texto, "token:cercado:v3:001") || strings.Contains(texto, "aaaa") {
		t.Fatal("token filtrado por formato")
	}
	if _, err := json.Marshal(escenario.token); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON token: %v", err)
	}
	if _, err := escenario.token.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalText token: %v", err)
	}
	var restaurado TokenCercadoEjecucionDocumentalV3
	if err := json.Unmarshal([]byte(`{}`), &restaurado); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalJSON token: %v", err)
	}
	if err := restaurado.UnmarshalText([]byte("token")); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalText token: %v", err)
	}

	macEntrada := []byte(strings.Repeat("z", 32))
	tokenCopiado, err := NuevoTokenCercadoEjecucionDocumentalV3(
		"token:cercado:v3:copia", 42, "reserva:documental:v3:001", escenario.manifiesto,
		escenario.consumo, "clave:cercado:v3", macEntrada, "evidencia:cercado:v3:copia",
	)
	if err != nil {
		t.Fatal(err)
	}
	macEntrada[0] ^= 0xff
	if tokenCopiado.macAtestacion[0] != 'z' {
		t.Fatal("el token retuvo un alias del MAC de entrada")
	}
	solicitudCopia, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		"reserva:documental:v3:001", escenario.manifiesto, escenario.consumo, tokenCopiado,
	)
	if err != nil {
		t.Fatal(err)
	}
	mensajeCopia, _ := solicitudCopia.Mensaje()
	macCopia, _ := solicitudCopia.MAC()
	mensajeCopia[0] ^= 0xff
	macCopia[0] ^= 0xff
	if solicitudCopia.Validar() != nil || tokenCopiado.macAtestacion[0] != 'z' {
		t.Fatal("los accesores del verificador expusieron memoria interna")
	}

	inicio := SolicitudIniciarEfectoDocumentalV3{
		ReservaRef: "reserva:documental:v3:001", Manifiesto: escenario.manifiesto,
		ConsumoDecision: escenario.consumo, Token: escenario.token,
		VerificacionCercado: escenario.verificacionCercado,
		IniciadoEn:          escenario.instante.Add(time.Second),
	}
	if err := inicio.Validar(); err != nil {
		t.Fatalf("inicio cercado valido: %v", err)
	}
	sinVerificacion := inicio
	sinVerificacion.VerificacionCercado = ResultadoVerificacionTokenCercadoDocumentalV3{}
	if sinVerificacion.Validar() == nil {
		t.Fatal("un token solo estructural habilito el efecto")
	}

	alterado := escenario.token
	alterado.macAtestacion = append([]byte(nil), escenario.token.macAtestacion...)
	alterado.macAtestacion[0] ^= 0xff
	if alterado.ValidarPara(inicio.ReservaRef, escenario.manifiesto, escenario.consumo) != nil {
		t.Fatal("la prueba necesita un sobre estructuralmente valido")
	}
	solicitudAlterada, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		inicio.ReservaRef, escenario.manifiesto, escenario.consumo, alterado,
	)
	if err != nil {
		t.Fatal(err)
	}
	if escenario.verificacionCercado.ValidarPara(solicitudAlterada) == nil {
		t.Fatal("una verificacion anterior autorizo otro MAC")
	}

	consumoCruzado := escenario.consumo
	consumoCruzado.DecisionRef = "decision:documental:v3:otra"
	if escenario.token.ValidarPara(inicio.ReservaRef, escenario.manifiesto, consumoCruzado) == nil {
		t.Fatal("token aceptado para otra DecisionRef")
	}
}

func TestEstadosEjecucionDocumentalV3SonCerrados(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	for _, caso := range []struct {
		desde, hasta EstadoEjecucionDocumentalV3
		admite       bool
	}{
		{EstadoEjecucionDocumentalV3Preparada, EstadoEjecucionDocumentalV3Activa, true},
		{EstadoEjecucionDocumentalV3Activa, EstadoEjecucionDocumentalV3EfectoIniciado, true},
		{EstadoEjecucionDocumentalV3EfectoIniciado, EstadoEjecucionDocumentalV3Indeterminada, true},
		{EstadoEjecucionDocumentalV3Indeterminada, EstadoEjecucionDocumentalV3Confirmada, true},
		{EstadoEjecucionDocumentalV3Confirmada, EstadoEjecucionDocumentalV3Activa, false},
		{"futuro", EstadoEjecucionDocumentalV3Confirmada, false},
	} {
		if caso.desde.PuedeTransicionarA(caso.hasta) != caso.admite {
			t.Fatalf("transicion %s -> %s", caso.desde, caso.hasta)
		}
	}
	base := InstantaneaEjecucionDocumentalV3{
		ReservaRef:             "reserva:documental:v3:001",
		IndiceIdempotenciaHMAC: escenario.preparar.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    escenario.preparar.HuellaSolicitudHMAC,
		Manifiesto:             escenario.manifiesto, ActualizadaEn: escenario.instante,
	}
	preparada := base
	preparada.Estado = EstadoEjecucionDocumentalV3Preparada
	if preparada.Validar() != nil {
		t.Fatal("instantanea preparada valida rechazada")
	}
	abandonada := base
	abandonada.Estado = EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	abandonada.EstadoOrigenAbandono = EstadoEjecucionDocumentalV3Preparada
	abandonada.MotivoAbandonoRef = "motivo:sin:efecto"
	if abandonada.Validar() != nil {
		t.Fatal("abandono desde preparada rechazado")
	}
	abandonada.SecuenciaCercado = 99
	if abandonada.Validar() == nil {
		t.Fatal("abandono acepto cercado arbitrario")
	}

	solicitudAbandonoPreparada := SolicitudAbandonarEjecucionDocumentalV3{
		ReservaRef: "reserva:documental:v3:001", Manifiesto: escenario.manifiesto,
		EstadoEsperado: EstadoEjecucionDocumentalV3Preparada,
		MotivoRef:      "motivo:sin:efecto", AbandonadaEn: escenario.instante.Add(time.Second),
	}
	if solicitudAbandonoPreparada.Validar() != nil {
		t.Fatal("abandono desde preparada valido rechazado")
	}
	solicitudAbandonoActiva := solicitudAbandonoPreparada
	solicitudAbandonoActiva.EstadoEsperado = EstadoEjecucionDocumentalV3Activa
	solicitudAbandonoActiva.ConsumoDecision = escenario.consumo
	solicitudAbandonoActiva.Token = escenario.token
	solicitudAbandonoActiva.VerificacionCercado = escenario.verificacionCercado
	if solicitudAbandonoActiva.Validar() != nil {
		t.Fatal("abandono desde activa valido rechazado")
	}
	sinPruebaCercado := solicitudAbandonoActiva
	sinPruebaCercado.VerificacionCercado = ResultadoVerificacionTokenCercadoDocumentalV3{}
	if sinPruebaCercado.Validar() == nil {
		t.Fatal("abandono desde activa sin verificacion de cercado aceptado")
	}
	abandonadaActiva := base
	abandonadaActiva.Estado = EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	abandonadaActiva.EstadoOrigenAbandono = EstadoEjecucionDocumentalV3Activa
	abandonadaActiva.MotivoAbandonoRef = "motivo:sin:efecto"
	abandonadaActiva.SecuenciaCercado = escenario.token.Secuencia()
	abandonadaActiva.HuellaVinculoSHA256 = escenario.token.HuellaVinculoSHA256()
	abandonadaActiva.ConsumoDecision = escenario.consumo
	if abandonadaActiva.Validar() != nil {
		t.Fatal("instantanea de abandono desde activa rechazada")
	}

	indeterminada := base
	indeterminada.Estado = EstadoEjecucionDocumentalV3Indeterminada
	indeterminada.SecuenciaCercado = escenario.token.Secuencia()
	indeterminada.HuellaVinculoSHA256 = escenario.token.HuellaVinculoSHA256()
	indeterminada.ConsumoDecision = escenario.consumo
	indeterminada.IncidenteRef = "incidente:documental:v3:001"
	if indeterminada.Validar() != nil {
		t.Fatal("instantanea indeterminada valida rechazada")
	}
	indeterminada.Resultado = escenario.resultado
	if indeterminada.Validar() == nil {
		t.Fatal("indeterminada acepto resultado asumido")
	}

	abandonadaIndeterminada := indeterminada
	abandonadaIndeterminada.Resultado = ResultadoEfectoRenderizadoDocumentalV3{}
	abandonadaIndeterminada.Estado = EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	abandonadaIndeterminada.EstadoOrigenAbandono = EstadoEjecucionDocumentalV3Indeterminada
	abandonadaIndeterminada.MotivoAbandonoRef = "motivo:reconciliado:sin:efecto"
	abandonadaIndeterminada.ReconciliacionRef = "atestacion:reconciliacion:v3:sin:efecto"
	abandonadaIndeterminada.HuellaReconciliacionSHA256 = strings.Repeat("9", 64)
	if abandonadaIndeterminada.Validar() != nil {
		t.Fatal("abandono reconciliado no conservo incidente y evidencia negativa")
	}

	datosSello, _ := escenario.sello.Datos()
	confirmada := base
	confirmada.Estado = EstadoEjecucionDocumentalV3Confirmada
	confirmada.SecuenciaCercado = escenario.token.Secuencia()
	confirmada.HuellaVinculoSHA256 = escenario.token.HuellaVinculoSHA256()
	confirmada.ConsumoDecision = escenario.consumo
	confirmada.Resultado = escenario.resultado
	confirmada.EvidenciaRef = datosSello.EvidenciaOperacionRef
	confirmada.HuellaEvidenciaSHA256 = datosSello.HuellaMensajeSHA256
	confirmada.ReconciliacionRef = "atestacion:reconciliacion:v3:001"
	confirmada.HuellaReconciliacionSHA256 = strings.Repeat("8", 64)
	if confirmada.Validar() != nil {
		t.Fatal("instantanea confirmada por reconciliacion exacta rechazada")
	}
	confirmada.HuellaReconciliacionSHA256 = ""
	if confirmada.Validar() == nil {
		t.Fatal("instantanea confirmada acepto reconciliacion parcial")
	}
}

func TestConfirmacionV3LigaTresRecibosCercadoYSelloRestaurable(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	if err := escenario.confirmacion.Validar(); err != nil {
		t.Fatalf("confirmacion valida rechazada: %v", err)
	}
	datos, _ := escenario.manifiesto.Datos()
	manifiestoRestaurado := ManifiestoEjecucionDocumentalV3{datos: &datos}
	restaurada := escenario.confirmacion
	restaurada.Manifiesto = manifiestoRestaurado
	restaurada.Evidencia.Manifiesto = manifiestoRestaurado
	if err := restaurada.Validar(); err != nil {
		t.Fatalf("confirmacion restaurada por valor rechazada: %v", err)
	}

	cruzada := escenario.confirmacion
	cruzada.Recibos.Semantico = escenario.recibos.Estructural
	if cruzada.Validar() == nil {
		t.Fatal("confirmacion acepto recibo semantico cruzado")
	}
	evidenciaAlterada := escenario.confirmacion
	evidenciaAlterada.Evidencia.Recibos.HuellaReciboSemanticoSHA256 = strings.Repeat("e", 64)
	if evidenciaAlterada.Validar() == nil {
		t.Fatal("confirmacion acepto huella de recibo manipulada")
	}

	datosSello, err := escenario.sello.Datos()
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), datosSello.Firma...)
	datosSello.Firma[0] ^= 0xff
	datosOtraLectura, _ := escenario.sello.Datos()
	if string(datosOtraLectura.Firma) != string(original) {
		t.Fatal("Datos expuso alias de la firma")
	}
	perfil, _ := NuevoPerfilSelloEvidenciaHMACSHA256V3("clave:evidencia:v3")
	firma, _ := NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(perfil, escenario.evidencia)
	solicitudManipulada, err := NuevaSolicitudVerificacionEvidenciaDocumentalV3DesdeDatos(
		firma, datosSello,
	)
	if err != nil {
		t.Fatal(err)
	}
	if escenario.verificacion.ValidarPara(solicitudManipulada) == nil {
		t.Fatal("resultado de verificacion reutilizado tras manipular la firma")
	}
	var selloCero SelloEvidenciaDocumentalV3
	if err := json.Unmarshal([]byte(`{}`), &selloCero); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalJSON sello: %v", err)
	}
	if err := selloCero.UnmarshalText([]byte("sello")); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("UnmarshalText sello: %v", err)
	}
	if _, err := json.Marshal(escenario.sello); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON sello: %v", err)
	}
	if _, err := escenario.sello.MarshalText(); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalText sello: %v", err)
	}

	cronologia := escenario.evidencia
	cronologia.VerificadoCercadoEn = cronologia.GeneradoEn.Add(time.Second)
	if cronologia.Validar() == nil {
		t.Fatal("evidencia acepto cercado verificado despues de generar")
	}
}

func TestReconciliacionV3ExigeCOSEVerificadoYLigaConfirmacion(t *testing.T) {
	escenario := nuevoEscenarioEjecucionDocumentalV3Prueba(t)
	consulta := SolicitudConsultarEfectoDocumentalV3{
		ReservaRef: "reserva:documental:v3:001", Manifiesto: escenario.manifiesto,
		ConsumoDecision: escenario.consumo, Token: escenario.token,
		VerificacionCercado: escenario.verificacionCercado,
	}
	bytesSobre := []byte("cose-reconciliacion-aplicada-v3")
	sobre, err := NuevoSobreAtestacionReconciliacionDocumentalV3(bytesSobre)
	if err != nil {
		t.Fatal(err)
	}
	bytesSobre[0] ^= 0xff
	coseExpuesto, _ := sobre.COSESign1()
	coseExpuesto[0] ^= 0xff
	coseOtraLectura, _ := sobre.COSESign1()
	if string(coseOtraLectura) != "cose-reconciliacion-aplicada-v3" {
		t.Fatal("el sobre COSE no realizo copias defensivas")
	}
	if _, err := json.Marshal(sobre); !errors.Is(err, ErrSerializacionSecretoDocumentalV3) {
		t.Fatalf("MarshalJSON sobre COSE: %v", err)
	}
	huellaSobre, _ := sobre.HuellaSHA256()
	resultado := ResultadoConsultaEfectoDocumentalV3{
		ReservaRef: consulta.ReservaRef, EfectoRef: escenario.resultado.EfectoRef,
		SecuenciaCercado:    escenario.token.Secuencia(),
		HuellaVinculoSHA256: escenario.token.HuellaVinculoSHA256(),
		HuellaPlanSHA256:    escenario.consumo.HuellaPlanSHA256,
		Estado:              ResultadoReconciliacionDocumentalV3AplicadoExacto,
		Resultado:           escenario.resultado, AtestacionRef: "atestacion:reconciliacion:v3:001",
		HuellaAtestacionSHA256: huellaSobre, SobreAtestacion: sobre,
		ConsultadaEn: escenario.instante.Add(5 * time.Second),
	}
	solicitudVerificacion, err := NuevaSolicitudVerificacionReconciliacionDocumentalV3(consulta, resultado)
	if err != nil {
		t.Fatal(err)
	}
	verificacion, err := NuevoResultadoVerificacionReconciliacionDocumentalV3(
		solicitudVerificacion, "verificacion:reconciliacion:v3:001",
		escenario.instante.Add(6*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := confirmarReconciliacionEjecucionDocumentalV3Prueba(
		t, escenario, resultado, verificacion,
	)
	aplicar := SolicitudAplicarReconciliacionDocumentalV3{
		Consulta: consulta, ResultadoConsulta: resultado, Verificacion: verificacion,
		TieneConfirmacion: true, Confirmacion: confirmacion,
	}
	if err := aplicar.Validar(); err != nil {
		t.Fatalf("indeterminada -> aplicada -> confirmada rechazada: %v", err)
	}
	sinVerificacion := aplicar
	sinVerificacion.Verificacion = ResultadoVerificacionReconciliacionDocumentalV3{}
	if sinVerificacion.Validar() == nil {
		t.Fatal("reconciliacion sin verificacion COSE cambio estado")
	}
	manipulada := resultado
	manipulada.SobreAtestacion.coseSign1 = append([]byte(nil), sobre.coseSign1...)
	manipulada.SobreAtestacion.coseSign1[0] ^= 0xff
	manipulada.SobreAtestacion.huella = huellaBytesFormatoDocumental(manipulada.SobreAtestacion.coseSign1)
	manipulada.HuellaAtestacionSHA256 = manipulada.SobreAtestacion.huella
	solicitudManipulada, err := NuevaSolicitudVerificacionReconciliacionDocumentalV3(consulta, manipulada)
	if err != nil {
		t.Fatal(err)
	}
	if verificacion.ValidarPara(solicitudManipulada) == nil {
		t.Fatal("verificacion de reconciliacion reutilizada con otro sobre")
	}
	cronologia := confirmacion.Evidencia
	cronologia.ReconciliacionConsultadaEn = cronologia.GeneradoEn.Add(-time.Second)
	if cronologia.Validar() == nil {
		t.Fatal("evidencia acepto una reconciliacion anterior a su generacion")
	}
}

func nuevoEscenarioEjecucionDocumentalV3Prueba(t *testing.T) escenarioEjecucionDocumentalV3Prueba {
	t.Helper()
	instante := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	identidad, _ := domain.NuevaIdentidadSintacticaDocumental("pdf")
	perfilRef, _ := domain.NuevaReferenciaPerfilDocumental("pdfa-4", 3)
	capacidades, _ := domain.NuevasCapacidadesPerfilFormatoDocumental(
		domain.CapacidadPerfilRenderizar, domain.CapacidadPerfilMetadatoInstitucional,
	)
	conformidad, err := domain.NuevaReferenciaConformidadDocumental(
		"conformidad:pdfa4:v3", 1, "esquema:pdfa4:v3", "dialecto:pdfa4:v3",
		"canonicalizacion:pdf:v3", "reglas:pdfa4:v3", strings.Repeat("1", 64),
		"politica:documental:v3", strings.Repeat("2", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := domain.NuevoPerfilFormatoDocumentalConforme(
		perfilRef, identidad, "application/pdf", "pdf", "binario",
		capacidades, conformidad, 4*1024*1024,
	)
	if err != nil {
		t.Fatal(err)
	}
	revision, _ := domain.NuevaRevisionCatalogoFormatosDocumentales(21, strings.Repeat("3", 64))
	consulta := ConsultaFormatoDocumental{
		Identidad: identidad, PerfilRef: perfilRef, DigestPerfilSHA256: perfil.DigestSHA256(),
		RevisionCatalogo: revision,
	}
	descriptor, err := NuevoDescriptorPerfilDocumental(
		"descriptor:pdfa4:v3", "publicacion:pdfa4:v3", perfil, revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	situacion, _ := domain.NuevaSituacionOperativaPerfilDocumental(
		descriptor.PublicacionRef(), perfil, revision, 7, domain.EstadoPublicacionPerfilVigente,
	)
	render := descriptorComponenteEjecucionDocumentalV3Prueba(
		t, descriptor, domain.RolComponenteRenderizador, "motor:render:v3", "dominio:render:v3", '4', 2*1024*1024,
	)
	estructural := descriptorComponenteEjecucionDocumentalV3Prueba(
		t, descriptor, domain.RolComponenteValidadorEstructural, "motor:estructural:v3", "dominio:estructural:v3", '7', 2*1024*1024,
	)
	semantico := descriptorComponenteEjecucionDocumentalV3Prueba(
		t, descriptor, domain.RolComponenteVerificadorSemantico, "motor:semantico:v3", "dominio:semantico:v3", 'a', 1024*1024,
	)
	entrada := "hmac-sha256:entrada-v3:" + strings.Repeat("a", 64)
	manifiesto, err := NuevoManifiestoEjecucionDocumentalV3(
		consulta, descriptor, situacion, render, estructural, semantico,
		"borrador:documental:v3:001", "efecto:documental:v3:001", entrada, 1024*1024,
	)
	if err != nil {
		t.Fatal(err)
	}
	preparar := SolicitudPrepararEjecucionDocumentalV3{
		IndiceIdempotenciaHMAC: "hmac-sha256:indice-v3:" + strings.Repeat("b", 64),
		HuellaSolicitudHMAC:    "hmac-sha256:solicitud-v3:" + strings.Repeat("c", 64),
		Manifiesto:             manifiesto, SolicitadaEn: instante, ExpiraEn: instante.Add(10 * time.Minute),
	}
	datosManifiesto, _ := manifiesto.Datos()
	consumo := ConsumoDecisionEjecucionDocumentalV3{
		DecisionRef: "decision:documental:v3:001", EfectoRef: datosManifiesto.EfectoRef,
		EsquemaHuellaDecision: EsquemaHuellaDecisionAutorizacionReforzadaV1,
		HuellaDecisionSHA256:  strings.Repeat("d", 64), HuellaPlanSHA256: datosManifiesto.HuellaPlanSHA256,
	}
	token, err := NuevoTokenCercadoEjecucionDocumentalV3(
		"token:cercado:v3:001", 41, "reserva:documental:v3:001", manifiesto, consumo,
		"clave:cercado:v3", []byte(strings.Repeat("m", 32)), "evidencia:cercado:v3:001",
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudCercado, _ := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		"reserva:documental:v3:001", manifiesto, consumo, token,
	)
	verificacionCercado, err := NuevoResultadoVerificacionTokenCercadoDocumentalV3(
		solicitudCercado, "verificacion:cercado:v3:001", instante.Add(500*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := ResultadoEfectoRenderizadoDocumentalV3{
		BorradorRef: datosManifiesto.BorradorRef, EfectoRef: datosManifiesto.EfectoRef,
		ContenidoRef: "objeto:documental:v3:001", ContenidoVersion: "version:v3:001",
		ConectorRef: "conector:almacen:v3", MIME: perfil.MIME(),
		HuellaSalidaSHA256: strings.Repeat("e", 64), TamanoSalida: 2048,
		EvidenciaOperacionRef: "evidencia:almacen:v3:001",
	}
	recibos := RecibosEjecucionDocumentalV3{
		Render: reciboEjecucionDocumentalV3Prueba(
			t, "render", OperacionRenderizadoDocumental, render, descriptor, situacion,
			manifiesto, consumo, token, verificacionCercado, resultado, instante, 1,
		),
		Estructural: reciboEjecucionDocumentalV3Prueba(
			t, "estructural", OperacionValidacionEstructuralDocumental, estructural,
			descriptor, situacion, manifiesto, consumo, token, verificacionCercado, resultado, instante, 2,
		),
		Semantico: reciboEjecucionDocumentalV3Prueba(
			t, "semantico", OperacionVerificacionSemanticaDocumental, semantico,
			descriptor, situacion, manifiesto, consumo, token, verificacionCercado, resultado, instante, 3,
		),
	}
	huellasRecibos, err := recibos.Huellas()
	if err != nil {
		t.Fatal(err)
	}
	evidencia := DatosEvidenciaRenderizadoDocumentalV3{
		Esquema: EsquemaEvidenciaRenderizadoV3, ReservaRef: "reserva:documental:v3:001",
		IndiceIdempotenciaHMAC: preparar.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    preparar.HuellaSolicitudHMAC, Manifiesto: manifiesto,
		SecuenciaCercado: token.Secuencia(), HuellaVinculoSHA256: token.HuellaVinculoSHA256(),
		ClaveAtestacionCercadoRef:     token.claveAtestacionRef,
		HuellaMACCercadoSHA256:        huellaBytesFormatoDocumental(token.macAtestacion),
		EvidenciaAtestacionCercadoRef: token.evidenciaOperacionRef,
		VerificacionCercadoRef:        verificacionCercado.verificacionRef,
		VerificadoCercadoEn:           verificacionCercado.verificadaEn,
		ConsumoDecision:               consumo, Resultado: resultado, Recibos: huellasRecibos,
		GeneradoEn: instante.Add(2 * time.Second), ConfirmadoEn: instante.Add(3 * time.Second),
	}
	perfilSello, _ := NuevoPerfilSelloEvidenciaHMACSHA256V3("clave:evidencia:v3")
	firma, err := NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(perfilSello, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	sello, err := NuevoSelloEvidenciaDocumentalV3(
		firma, []byte(strings.Repeat("s", 32)), "evidencia:firma:v3:001", instante.Add(3*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudVerificacion, _ := NuevaSolicitudVerificacionEvidenciaDocumentalV3(firma, sello)
	verificacion, err := NuevoResultadoVerificacionEvidenciaDocumentalV3(
		solicitudVerificacion, "verificacion:evidencia:v3:001", instante.Add(4*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := SolicitudConfirmarEjecucionDocumentalV3{
		ReservaRef: evidencia.ReservaRef, Manifiesto: manifiesto, ConsumoDecision: consumo,
		Token: token, VerificacionCercado: verificacionCercado, Resultado: resultado,
		Recibos: recibos, Evidencia: evidencia, Sello: sello, Verificacion: verificacion,
	}
	return escenarioEjecucionDocumentalV3Prueba{
		instante: instante, manifiesto: manifiesto, preparar: preparar, consumo: consumo,
		token: token, verificacionCercado: verificacionCercado, resultado: resultado,
		recibos: recibos, evidencia: evidencia, sello: sello,
		verificacion: verificacion, confirmacion: confirmacion,
	}
}

func descriptorComponenteEjecucionDocumentalV3Prueba(
	t *testing.T,
	descriptor DescriptorPerfilDocumental,
	rol domain.RolComponenteDocumental,
	identificador, dominio string,
	semilla byte,
	maximo uint64,
) DescriptorComponenteDocumentalAtestado {
	t.Helper()
	huella := func(offset byte) string {
		alfabeto := "0123456789abcdef"
		return strings.Repeat(string(alfabeto[(int(semilla-'0')+int(offset))%len(alfabeto)]), 64)
	}
	consulta := consultaComponenteEjecucionDocumentalV3(descriptor, rol)
	componente, err := domain.NuevaReferenciaComponenteDocumental(
		rol, identificador, uint64(semilla), "homologacion:"+identificador,
		huella(0), huella(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := NuevoDescriptorComponenteDocumentalAtestado(
		"descriptor:"+identificador, consulta, componente, dominio,
		"broker:"+identificador, "atestacion:"+identificador, huella(2), maximo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func reciboEjecucionDocumentalV3Prueba(
	t *testing.T,
	nombre string,
	operacion OperacionComponenteDocumental,
	componente DescriptorComponenteDocumentalAtestado,
	descriptor DescriptorPerfilDocumental,
	situacion domain.SituacionOperativaPerfilDocumental,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
	verificacionCercado ResultadoVerificacionTokenCercadoDocumentalV3,
	resultado ResultadoEfectoRenderizadoDocumentalV3,
	instante time.Time,
	semilla byte,
) ReciboEjecucionComponenteDocumentalVerificado {
	t.Helper()
	var reto [32]byte
	reto[0] = semilla
	huellaDocumento := resultado.HuellaSalidaSHA256
	tamano := resultado.TamanoSalida
	if operacion == OperacionRenderizadoDocumental {
		huellaDocumento, tamano = "", 0
	}
	compromiso, err := NuevoCompromisoEjecucionComponenteDocumental(
		"operacion:"+nombre+":v3", reto, operacion, descriptor, situacion, componente,
		"reserva:documental:v3:001", manifiesto, consumo, token, verificacionCercado,
		resultado.BorradorRef, manifiesto.datos.HuellaEntradaHMAC, huellaDocumento,
		tamano, manifiesto.datos.LimiteEfectivoBytes, 5*time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	sobre, err := NuevoSobreReciboEjecucionDocumental([]byte(strings.Repeat(string('k'+semilla), 32)))
	if err != nil {
		t.Fatal(err)
	}
	artefacto := componente.Componente().HuellaArtefactoSHA256()
	identidad, err := NuevaIdentidadEjecucionComponenteDocumental(
		"carga:"+nombre+":v3", "instancia:"+nombre+":v3", componente.DominioConfianzaRef(),
		"clave:"+nombre+":v3", strings.Repeat(string('1'+semilla), 64), artefacto,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultadoRecibo := ResultadoRenderizadoDocumentalCorrecto
	if operacion == OperacionValidacionEstructuralDocumental {
		resultadoRecibo = ResultadoEstructuraDocumentalConforme
	} else if operacion == OperacionVerificacionSemanticaDocumental {
		resultadoRecibo = ResultadoSemanticaDocumentalEquivalente
	}
	recibo, err := NuevoReciboEjecucionComponenteDocumentalVerificado(
		compromiso, sobre, "recibo:"+nombre+":v3", resultadoRecibo,
		resultado.HuellaSalidaSHA256, resultado.TamanoSalida, identidad,
		verificacionCercado.verificadaEn.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	return recibo
}

func confirmarReconciliacionEjecucionDocumentalV3Prueba(
	t *testing.T,
	escenario escenarioEjecucionDocumentalV3Prueba,
	resultado ResultadoConsultaEfectoDocumentalV3,
	verificacion ResultadoVerificacionReconciliacionDocumentalV3,
) SolicitudConfirmarEjecucionDocumentalV3 {
	t.Helper()
	evidencia := escenario.evidencia
	evidencia.ReconciliacionRef = resultado.AtestacionRef
	evidencia.HuellaReconciliacionSHA256 = resultado.HuellaAtestacionSHA256
	evidencia.ReconciliacionConsultadaEn = resultado.ConsultadaEn
	evidencia.VerificacionReconciliacionRef = verificacion.verificacionRef
	evidencia.ReconciliacionVerificadaEn = verificacion.verificadaEn
	evidencia.ConfirmadoEn = verificacion.verificadaEn.Add(time.Second)
	perfil, _ := NuevoPerfilSelloEvidenciaHMACSHA256V3("clave:evidencia:v3")
	firma, err := NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(perfil, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	sello, err := NuevoSelloEvidenciaDocumentalV3(
		firma, []byte(strings.Repeat("r", 32)), "evidencia:firma:reconciliada:v3",
		evidencia.ConfirmadoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, _ := NuevaSolicitudVerificacionEvidenciaDocumentalV3(firma, sello)
	verificacionEvidencia, err := NuevoResultadoVerificacionEvidenciaDocumentalV3(
		solicitud, "verificacion:evidencia:reconciliada:v3", evidencia.ConfirmadoEn.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := escenario.confirmacion
	confirmacion.Evidencia = evidencia
	confirmacion.Sello = sello
	confirmacion.Verificacion = verificacionEvidencia
	return confirmacion
}
