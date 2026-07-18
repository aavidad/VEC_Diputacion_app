package documental

import (
	"strings"
	"testing"
	"time"
)

func hmacV3Prueba(clave, digito string) string {
	return "hmac-sha256:" + clave + ":" + strings.Repeat(digito, 64)
}

func vinculoV3Prueba() DatosVinculoActivacionV3 {
	return DatosVinculoActivacionV3{
		ReservaRef: "reserva:uno", IndiceIdempotenciaHMAC: hmacV3Prueba("indice", "a"),
		HuellaSolicitudHMAC:    hmacV3Prueba("solicitud", "b"),
		HuellaEntradaHMAC:      hmacV3Prueba("entrada", "c"),
		HuellaManifiestoSHA256: strings.Repeat("d", 64), EfectoManifiestoRef: "efecto:uno",
		HuellaPlanManifiestoSHA256: strings.Repeat("e", 64),
		OrdenConsumoDurableV4Ref:   "efecto:uno", DecisionRef: "decision:uno",
		EfectoDecisionRef: "efecto:uno", EsquemaHuellaDecision: "esquema:decision",
		EsquemaHuellaDecisionEsperado: "esquema:decision",
		HuellaDecisionSHA256:          strings.Repeat("f", 64),
		HuellaPlanDecisionSHA256:      strings.Repeat("e", 64),
	}
}

func tokenV3Prueba(vinculo DatosVinculoActivacionV3) DatosTokenCercadoV3 {
	huella := vinculo.HuellaSHA256()
	return DatosTokenCercadoV3{
		Valor: "token:uno", Secuencia: 7,
		HuellaVinculoEstableSHA256: huella, HuellaVinculoEsperadoSHA256: huella,
		HuellaVinculoInternoSHA256: huella,
		HuellaVinculoCercadoSHA256: HuellaVinculoCercadoV3(7, huella),
		ClaveAtestacionRef:         "clave:atestacion", RevisionClave: 3,
		MACAtestacion:         append([]byte{1}, make([]byte, TamanoFirmaHMACSHA256V3-1)...),
		EvidenciaOperacionRef: "evidencia:token", ClaveHuellaEntradaHMAC: "entrada",
	}
}

func TestVinculoYTokenV3FijanDominiosYOrdenDePreimagen(t *testing.T) {
	t.Parallel()

	vinculo := vinculoV3Prueba()
	if !vinculo.Validar() || !SHA256HexadecimalValido(vinculo.HuellaSHA256()) {
		t.Fatal("se rechazo el vinculo canonico")
	}
	reutilizaClave := vinculo
	reutilizaClave.HuellaSolicitudHMAC = hmacV3Prueba("indice", "b")
	if reutilizaClave.Validar() {
		t.Fatal("se acepto reutilizacion de clave HMAC entre dominios")
	}

	token := tokenV3Prueba(vinculo)
	if !token.Validar() {
		t.Fatal("se rechazo el token canonico")
	}
	esperado := SerializarCamposV3([]string{
		esquemaAtestacionTokenCercadoV3, token.Valor, Uint64Decimal(token.Secuencia),
		token.HuellaVinculoEstableSHA256, token.HuellaVinculoCercadoSHA256,
		AlgoritmoHMACSHA256V3, AudienciaTokenCercadoV3, ContextoTokenCercadoV3,
		token.ClaveAtestacionRef, Uint64Decimal(token.RevisionClave), token.EvidenciaOperacionRef,
	})
	if !BytesIguales(token.MensajeAtestacion(), esperado) {
		t.Fatal("cambio el orden de campos de la atestacion de cercado")
	}
	alterado := token
	alterado.MACAtestacion = make([]byte, TamanoFirmaHMACSHA256V3)
	if alterado.Validar() {
		t.Fatal("se acepto una MAC nominal enteramente nula")
	}
}

func pruebaAtestacionV3(audiencia, contexto, evidencia string) DatosPruebaAtestacionDespachoV3 {
	mensaje := []byte("mensaje canonico")
	sobre := append([]byte{1}, make([]byte, TamanoFirmaHMACSHA256V3-1)...)
	return DatosPruebaAtestacionDespachoV3{
		Algoritmo: AlgoritmoHMACSHA256V3, Audiencia: audiencia, Contexto: contexto,
		ClaveGestionadaRef: "clave:atestacion", RevisionClaveGestionada: 3,
		EvidenciaOperacionRef: evidencia, MensajeCanonico: mensaje, SobreCriptografico: sobre,
		HuellaMensajeSHA256: HuellaBytesSHA256(mensaje), HuellaSobreSHA256: HuellaBytesSHA256(sobre),
	}
}

func TestPruebaAtestacionV3CierraPerfilesYAlteraciones(t *testing.T) {
	t.Parallel()

	prueba := pruebaAtestacionV3(AudienciaInicioEfectoV3, ContextoInicioEfectoV3, "evidencia:inicio")
	if !prueba.Validar() || !SHA256HexadecimalValido(prueba.HuellaSHA256()) {
		t.Fatal("se rechazo prueba nominal canonica")
	}
	perfilCruzado := prueba
	perfilCruzado.Contexto = ContextoReclamacionDespachoV3
	if perfilCruzado.Validar() {
		t.Fatal("se acepto audiencia y contexto cruzados")
	}
	mensajeAlterado := prueba
	mensajeAlterado.MensajeCanonico = []byte("otro")
	if mensajeAlterado.Validar() {
		t.Fatal("se acepto una preimagen distinta de su huella")
	}
}

func inicioV3Prueba() DatosAtestacionInicioEfectoV3 {
	return DatosAtestacionInicioEfectoV3{
		InicioRef: "inicio:uno", ReservaRef: "reserva:uno",
		HuellaVinculoEstableSHA256: strings.Repeat("a", 64), SecuenciaCercado: 7,
		HuellaVinculoCercadoSHA256: strings.Repeat("b", 64),
		OrdenConsumoDurableV4Ref:   "orden:uno", VersionInicioCAS: 2,
		AuditoriaInicioRef: "auditoria:inicio", OutboxInicioRef: "outbox:inicio",
		ClaveAtestacionRef: "clave:atestacion", RevisionClave: 3,
		EvidenciaOperacionRef: "evidencia:inicio",
		IniciadoEn:            time.Date(2026, 7, 18, 10, 0, 0, 123_456_000, time.UTC),
	}
}

func TestInicioYReciboV3ConservanPreimagenYUnicidad(t *testing.T) {
	t.Parallel()

	inicio := inicioV3Prueba()
	if !inicio.Validar() || len(inicio.Bytes()) == 0 {
		t.Fatal("se rechazo preimagen de inicio valida")
	}
	prueba := pruebaAtestacionV3(AudienciaInicioEfectoV3, ContextoInicioEfectoV3, inicio.EvidenciaOperacionRef)
	recibo := DatosReciboInicioEfectoV3{
		InicioRef: inicio.InicioRef, ReservaRef: inicio.ReservaRef,
		HuellaVinculoEstableSHA256: inicio.HuellaVinculoEstableSHA256,
		SecuenciaCercado:           inicio.SecuenciaCercado,
		HuellaVinculoCercadoSHA256: inicio.HuellaVinculoCercadoSHA256,
		OrdenConsumoDurableV4Ref:   inicio.OrdenConsumoDurableV4Ref,
		VersionInicioCAS:           inicio.VersionInicioCAS, AuditoriaInicioRef: inicio.AuditoriaInicioRef,
		OutboxInicioRef: inicio.OutboxInicioRef, EvidenciaOperacionRef: inicio.EvidenciaOperacionRef,
		AtestacionValida: prueba.Validar(), HuellaAtestacionSHA256: prueba.HuellaSHA256(),
		IniciadoEn: inicio.IniciadoEn,
	}
	if !recibo.Validar() || !SHA256HexadecimalValido(recibo.HuellaSHA256()) {
		t.Fatal("se rechazo recibo de inicio valido")
	}
	duplicado := recibo
	duplicado.OutboxInicioRef = duplicado.InicioRef
	if duplicado.Validar() {
		t.Fatal("se aceptaron referencias con finalidades duplicadas")
	}
}

func solicitudReclamacionV3Prueba() DatosSolicitudReclamacionV3 {
	reclamada := time.Date(2026, 7, 18, 10, 1, 0, 0, time.UTC)
	return DatosSolicitudReclamacionV3{
		ReclamacionRef: "reclamacion:uno", InicioRef: "inicio:uno",
		OutboxRef: "outbox:inicio", ConsumidorRef: "consumidor:uno",
		ReclamadaEn: reclamada, ExpiraEn: reclamada.Add(DuracionMaximaReclamacionDespachoV3),
	}
}

func TestReclamacionYMaterialV3CierranVentanaYLigaduras(t *testing.T) {
	t.Parallel()

	solicitud := solicitudReclamacionV3Prueba()
	if solicitud.Validar() != nil {
		t.Fatal("se rechazo el borde inclusivo de la ventana de reclamacion")
	}
	fuera := solicitud
	fuera.ExpiraEn = fuera.ExpiraEn.Add(time.Microsecond)
	if fuera.Validar() == nil {
		t.Fatal("se acepto una reclamacion fuera de ventana")
	}

	perfilCercado := pruebaAtestacionV3(AudienciaTokenCercadoV3, ContextoTokenCercadoV3, "evidencia:token")
	perfilInicio := pruebaAtestacionV3(AudienciaInicioEfectoV3, ContextoInicioEfectoV3, "evidencia:inicio")
	perfilReclamacion := pruebaAtestacionV3(
		AudienciaReclamacionDespachoV3, ContextoReclamacionDespachoV3, "evidencia:reclamacion",
	)
	vinculos := VinculosMaterialDespachoV3{
		InicioRef: "inicio:uno", AtestacionInicioRef: "evidencia:inicio",
		ReclamacionRef: "reclamacion:uno", AtestacionReclamacionRef: "evidencia:reclamacion",
		OrdenConsumoDurableV4Ref: "orden:uno", HuellaOrdenDespachoSHA256: strings.Repeat("1", 64),
		HuellaReciboInicioSHA256:   strings.Repeat("2", 64),
		HuellaVinculoEstableSHA256: strings.Repeat("3", 64),
		HuellaVinculoCercadoSHA256: strings.Repeat("4", 64),
		SecuenciaCercado:           7, VersionInicioCAS: 2, VersionReclamacionCAS: 3,
	}
	material := DatosMaterialDespachoV3{
		Vinculos:         vinculos,
		Cercado:          PerfilMaterialDespachoV3{true, perfilCercado.Audiencia, "clave:atestacion", 3, perfilCercado.HuellaSHA256()},
		Inicio:           PerfilMaterialDespachoV3{true, perfilInicio.Audiencia, "clave:atestacion", 3, perfilInicio.HuellaSHA256()},
		Reclamacion:      PerfilMaterialDespachoV3{true, perfilReclamacion.Audiencia, "clave:atestacion", 3, perfilReclamacion.HuellaSHA256()},
		ClaveEsperadaRef: "clave:atestacion", RevisionEsperada: 3,
		HuellaOrdenEsperadaSHA256:   vinculos.HuellaOrdenDespachoSHA256,
		HuellaReciboEsperadaSHA256:  vinculos.HuellaReciboInicioSHA256,
		HuellaVinculoEsperadaSHA256: vinculos.HuellaVinculoEstableSHA256,
		HuellaCercadoEsperadaSHA256: vinculos.HuellaVinculoCercadoSHA256,
		SecuenciaEsperada:           7, VersionInicioEsperada: 2, VersionReclamacionEsperada: 3,
		HuellaInicioEsperadaSHA256:      perfilInicio.HuellaSHA256(),
		HuellaReclamacionEsperadaSHA256: perfilReclamacion.HuellaSHA256(),
	}
	material.Mensaje = material.Bytes()
	material.HuellaMensajeSHA256 = HuellaBytesSHA256(material.Mensaje)
	if !material.Validar() {
		t.Fatal("se rechazo el material ligado")
	}
	alterado := material
	alterado.VersionInicioEsperada++
	if alterado.Validar() {
		t.Fatal("se acepto una version CAS esperada distinta")
	}
}
