package cobertura

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"time"

	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func huellaConfirmacionOperacionDecisionCobertura(
	orden OrdenOperacionDecisionCobertura,
) (string, error) {
	if orden.validar() != nil {
		return "", ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	canon := nuevoCanonOperacionDecisionCobertura()
	canon.texto(esquemaHuellaConfirmacionOperacionDecisionCobertura)
	if orden.datos.concesion != nil {
		datos := orden.datos.concesion
		resumen, _ := datos.resumen.Datos()
		decision := datos.agregadoSiguiente.DecisionesCobertura[len(datos.agregadoSiguiente.DecisionesCobertura)-1]
		referenciaC1, errReferenciaC1 :=
			datos.preparacion.preparacionC1.Referencia()
		huellaC1, errHuellaC1 :=
			datos.preparacion.preparacionC1.HuellaSHA256()
		validaHastaC1, errValidaHastaC1 :=
			datos.preparacion.preparacionC1.ValidaHasta()
		numeroOrdenesC1, huellaOrdenesC1, errOrdenesC1 :=
			identidadOrdenesC1ConfirmacionOperacionDecisionCobertura(
				datos.preparacion.preparacionC1,
				datos.efectoEn,
			)
		if errReferenciaC1 != nil || errHuellaC1 != nil ||
			errValidaHastaC1 != nil || errOrdenesC1 != nil {
			return "",
				ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
		}
		canon.texto("concedida")
		escribirIdentidadReservaConfirmacionOperacionDecisionCobertura(
			canon,
			datos.preparacion.reserva,
		)
		canon.texto(referenciaC1)
		canon.texto(huellaC1)
		canon.texto(
			datos.preparacion.preparacionC1.preparadaEn.
				Format(time.RFC3339Nano),
		)
		canon.texto(validaHastaC1.Format(time.RFC3339Nano))
		canon.entero(numeroOrdenesC1)
		canon.texto(huellaOrdenesC1)
		escribirResumenConfirmacionOperacionDecisionCobertura(canon, resumen)
		canon.texto(decision.Referencia)
		canon.texto(decision.HuellaSHA256)
		canon.entero(datos.agregadoSiguiente.Version)
		canon.texto(datos.efectoEn.Format(time.RFC3339Nano))
		canon.texto(datos.validaHasta.Format(time.RFC3339Nano))
	} else {
		datos := orden.datos.denegacion
		resumen, _ := datos.resumen.Datos()
		canon.texto("denegada")
		escribirReservaMinimaPruebaDenegacionOperacionDecisionCobertura(
			canon,
			datos.prueba.reserva,
		)
		canon.texto(datos.prueba.huellaSHA256)
		escribirResumenConfirmacionOperacionDecisionCobertura(canon, resumen)
		canon.texto(datos.validaHasta.Format(time.RFC3339Nano))
	}
	material, err := canon.resultado()
	if err != nil {
		return "", ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func escribirIdentidadReservaConfirmacionOperacionDecisionCobertura(
	canon *canonOperacionDecisionCobertura,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
) {
	canon.texto(reserva.AgregadoAnterior.OrganizacionRef)
	canon.texto(reserva.AgregadoAnterior.Referencia)
	canon.entero(reserva.AgregadoAnterior.Version)
	canon.texto(reserva.AnalisisRef)
	canon.texto(reserva.AnalisisHuellaSHA256)
	canon.texto(reserva.TokenPropietarioSHA256)
	canon.texto(reserva.ReservaRef)
	canon.texto(reserva.ReciboRef)
	canon.texto(reserva.ActuacionRef)
	canon.texto(reserva.AuditoriaRef)
	canon.texto(reserva.EventoRef)
	canon.texto(reserva.CorrelacionVECRef)
	canon.texto(reserva.DecisionVECRef)
	canon.entero(reserva.RevisionCercadoAnterior)
	canon.entero(reserva.RevisionCercado)
	canon.texto(reserva.AmbitoIdempotenciaHMAC)
	canon.texto(reserva.HuellaSemanticaHMAC)
	canon.texto(reserva.ObservadaEnDB.Format(time.RFC3339Nano))
	canon.texto(reserva.PropiedadHasta.Format(time.RFC3339Nano))
}

func identidadOrdenesC1ConfirmacionOperacionDecisionCobertura(
	preparacion PreparacionConjuntosViasCobertura,
	instante time.Time,
) (uint64, string, error) {
	ordenes, err := preparacion.OrdenesPendientesEn(instante)
	if err != nil || len(ordenes) == 0 ||
		uint64(len(ordenes)) >
			MaximoEnteroSeguroOperacionDecisionCobertura {
		return 0, "",
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	acumulada := sha256.New()
	var cantidad [8]byte
	binary.BigEndian.PutUint64(cantidad[:], uint64(len(ordenes)))
	_, _ = acumulada.Write(cantidad[:])
	for _, orden := range ordenes {
		resumen, errResumen := orden.ResumenPendienteEn(instante)
		if errResumen != nil {
			return 0, "",
				ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
		}
		canon := nuevoCanonOperacionDecisionCobertura()
		escribirResumenOrdenC1ConfirmacionOperacionDecisionCobertura(
			canon,
			resumen,
		)
		material, errCanon := canon.resultado()
		if errCanon != nil {
			return 0, "",
				ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
		}
		huella := sha256.Sum256(material)
		_, _ = acumulada.Write(huella[:])
	}
	return uint64(len(ordenes)), hex.EncodeToString(acumulada.Sum(nil)), nil
}

func escribirResumenOrdenC1ConfirmacionOperacionDecisionCobertura(
	canon *canonOperacionDecisionCobertura,
	resumen puertosct.ResumenOrdenConsumoCobertura,
) {
	canon.texto("VEC-CT-IDENTIDAD-ORDEN-C1-CONFIRMACION-C3-V1")
	canon.texto(resumen.PeticionRef)
	canon.texto(resumen.OrganizacionRef)
	canon.texto(resumen.ExpedienteRef)
	canon.entero(resumen.VersionExpediente)
	canon.texto(resumen.Catalogo.Referencia)
	canon.entero(resumen.Catalogo.Version)
	canon.texto(resumen.Catalogo.HuellaSHA256)
	canon.texto(string(resumen.ViaClave))
	canon.entero(uint64(resumen.OrdenComprobacion))
	if resumen.ComprobacionObligatoria {
		canon.texto("obligatoria")
	} else {
		canon.texto("opcional")
	}
	canon.texto(string(resumen.Comprobacion.Clave))
	canon.texto(string(resumen.Comprobacion.Resultado))
	canon.texto(resumen.Comprobacion.FuenteRef)
	canon.texto(resumen.Comprobacion.ReciboRef)
	canon.texto(resumen.Comprobacion.EvaluadaEn.Format(time.RFC3339Nano))
	canon.texto(string(resumen.ProcedenciaClave))
	canon.texto(resumen.DefinicionFuenteRef)
	canon.texto(resumen.CategoriaRef)
	canon.texto(resumen.Periodo.Inicio.Format(time.RFC3339Nano))
	canon.texto(resumen.Periodo.Fin.Format(time.RFC3339Nano))
	canon.texto(resumen.SolicitadaEn.Format(time.RFC3339Nano))
	canon.texto(resumen.EmitidaEn.Format(time.RFC3339Nano))
	canon.texto(resumen.ValidaHasta.Format(time.RFC3339Nano))
	canon.texto(resumen.HuellaPeticionSHA256)
	canon.texto(resumen.HuellaResultadoSHA256)
	canon.texto(resumen.HuellaRespuestaSHA256)
	canon.texto(resumen.AutoridadRef)
	canon.entero(uint64(resumen.Generacion))
	canon.texto(resumen.ReciboRespuestaRef)
	canon.texto(resumen.VerificadorRef)
	canon.texto(resumen.PublicadorCatalogoRef)
}

func escribirResumenConfirmacionOperacionDecisionCobertura(
	canon *canonOperacionDecisionCobertura,
	resumen puertosvec.DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3,
) {
	canon.texto(resumen.DecisionRef)
	canon.texto(resumen.DecisionHuellaSHA256)
	canon.texto(resumen.CodigoProbatorio)
	if resumen.Concedida {
		canon.texto("concedida")
	} else {
		canon.texto("denegada")
	}
	canon.texto(resumen.EmitidaEn.Format(time.RFC3339Nano))
	canon.texto(resumen.ValidaHasta.Format(time.RFC3339Nano))
}

func huellaSolicitudReconciliacionOperacionDecisionCobertura(
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
) (string, error) {
	datos, err := solicitud.CoordenadasPrimarias()
	if err != nil || solicitud.datos == nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			solicitud.datos.huellaOrdenSHA256,
		) {
		return "", ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	canon := nuevoCanonOperacionDecisionCobertura()
	canon.texto("VEC-CT-SOLICITUD-RECONCILIACION-DECISION-COBERTURA-C3-V1")
	canon.texto(datos.OrganizacionRef)
	canon.texto(datos.ExpedienteRef)
	canon.entero(datos.VersionExpediente)
	canon.texto(datos.ReservaRef)
	canon.texto(datos.ReciboRef)
	canon.texto(datos.CorrelacionVECRef)
	canon.texto(datos.DecisionVECRef)
	canon.entero(datos.RevisionCercado)
	canon.texto(solicitud.datos.huellaOrdenSHA256)
	material, err := canon.resultado()
	if err != nil {
		return "", ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}
