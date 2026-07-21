package postgres

import (
	"encoding/json"
	"errors"
	"math"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func serializarConfirmacionBorradorPostgreSQL(
	s gobiernoconvocatorias.SolicitudConfirmacionBorrador,
) (cargaConfirmacionBorradorPostgreSQL, error) {
	if s.Validar() != nil || s.Control.Revision > math.MaxInt64 || s.Control.Cercado > math.MaxInt64 {
		return cargaConfirmacionBorradorPostgreSQL{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	identidad, err := proyectarIdentidadDiarioPostgreSQL(s.Reserva.IdentidadPrimaria)
	if err != nil {
		return cargaConfirmacionBorradorPostgreSQL{}, err
	}
	material, err := json.Marshal(s.Material)
	huellaMaterial, errHuella := s.Material.HuellaSHA256()
	if err != nil || errHuella != nil || huellaBytesDiarioPostgreSQL(material) != huellaMaterial {
		borrarBytesDiarioPostgreSQL(material)
		return cargaConfirmacionBorradorPostgreSQL{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	version, err := s.Version.RepresentacionCanonica()
	datosDecision, errDecision := s.Concesion.Evidencia.Datos()
	decision, errCanonica := datosDecision.RepresentacionCanonica()
	recurso, errRecurso := puertosbolsa.RecursoAutorizableMutacionConvocatoria(s.Material, s.Version)
	contexto, errContexto := serializarContextoRecursoBorradorPostgreSQL(recurso)
	if errors.Join(err, errDecision, errCanonica, errRecurso, errContexto) != nil ||
		huellaBytesDiarioPostgreSQL(decision) != datosDecision.HuellaDecisionSHA256 {
		borrarBytesDiarioPostgreSQL(material, version, decision, contexto)
		return cargaConfirmacionBorradorPostgreSQL{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	prueba, err := json.Marshal(pruebaDecisionBorradorPostgreSQL{
		EsquemaHuella:        datosDecision.EsquemaHuella,
		DecisionRef:          datosDecision.Decision.DecisionRef,
		HuellaDecisionSHA256: datosDecision.HuellaDecisionSHA256,
		VerificadaEn:         datosDecision.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo),
		PrincipalRef:         datosDecision.Decision.PrincipalID,
	})
	if err != nil {
		borrarBytesDiarioPostgreSQL(material, version, decision, contexto)
		return cargaConfirmacionBorradorPostgreSQL{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	aad, errAAD := s.Cifrado.AAD.RepresentacionCanonica()
	huellaAAD, errHuellaAAD := s.Cifrado.AAD.HuellaSHA256()
	perfilEnvoltura, claveRef, versionClave, envuelto, huellaAADE, huellaEnvoltura, errEnvoltura :=
		s.Cifrado.EnvolturaClave.DatosParaPersistencia()
	perfilSobre, nonce, cifrado, huellaAADS, huellaSobre, errSobre :=
		s.Cifrado.SobreCifrado.DatosParaPersistencia()
	esquemaEnvoltura, errEsquemaEnvoltura := s.Cifrado.EnvolturaClave.EsquemaParaPersistencia()
	esquemaSobre, errEsquemaSobre := s.Cifrado.SobreCifrado.EsquemaParaPersistencia()
	if errors.Join(
		errAAD, errHuellaAAD, errEnvoltura, errSobre, errEsquemaEnvoltura, errEsquemaSobre,
	) != nil || perfilEnvoltura != s.PerfilCifrado || perfilSobre != s.PerfilCifrado ||
		huellaAADE != huellaAAD || huellaAADS != huellaAAD {
		borrarCargaConfirmacionParcial(material, version, decision, contexto, prueba, aad, envuelto, nonce, cifrado)
		return cargaConfirmacionBorradorPostgreSQL{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	perfil := proyectarPerfilCifradoBorradorPostgreSQL(s.PerfilCifrado)
	procedencia := proyectarProcedenciaBorradorPostgreSQL(s.Procedencia)
	politica := proyectarPoliticaCifradoBorradorPostgreSQL(s.PoliticaCifrado, perfil, identidad)
	evidenciaPerfil := proyectarEvidenciaPerfilBorradorPostgreSQL(
		s.ResolucionPerfilCifrado.Evidencia, perfil, identidad,
	)
	firmaAtestacion, err := proyectarFirmaEvidenciaBorradorPostgreSQL(s.Cifrado.AtestacionKMS.Firma)
	if err != nil {
		borrarCargaConfirmacionParcial(material, version, decision, contexto, prueba, aad, envuelto, nonce, cifrado)
		return cargaConfirmacionBorradorPostgreSQL{}, err
	}
	atestacion := proyectarAtestacionKMSBorradorPostgreSQL(
		s.Cifrado.AtestacionKMS, perfil, procedencia, firmaAtestacion,
	)
	evidencia, errEvidencia := json.Marshal(evidenciaCifradoBorradorPostgreSQL{
		Esquema: esquemaEvidenciaCifradoPostgreSQLV1, Perfil: perfil,
		Politica: politica, EvidenciaPerfil: evidenciaPerfil, Procedencia: procedencia,
		AAD: aadBorradorPostgreSQL{Esquema: esquemaAADPostgreSQLV1, HuellaSHA256: huellaAAD},
		EnvolturaClave: envolturaClaveBorradorPostgreSQL{
			Esquema: esquemaEnvoltura, ClaveMaestraRef: claveRef, VersionClave: versionClave,
			HuellaAAD: huellaAAD, HuellaEnvolturaSHA256: huellaEnvoltura,
		},
		Sobre: sobreCifradoBorradorPostgreSQL{
			Esquema: esquemaSobre, HuellaAAD: huellaAAD, HuellaSobreSHA256: huellaSobre,
		},
		AtestacionKMS: atestacion,
	})
	confirmacion, errConfirmacion := json.Marshal(confirmacionBorradorPostgreSQL{
		Esquema: esquemaConfirmacionBorradorPostgreSQLV2, Identidad: identidad,
		Revision: s.Control.Revision, Cercado: s.Control.Cercado,
		SelladoMotivo:    proyectarSelladoMotivoBorradorPostgreSQL(s.SelladoMotivo),
		EnvolturaCifrado: map[string]any{},
		ProyeccionLigera: proyectarVersionLigeraBorradorPostgreSQL(
			s.Version, s.Material.EstadoPrincipalNuevo,
		),
		SolicitadaEn: s.SolicitadaEn.UTC().Format(formatoInstanteMicrosegundo),
	})
	if errors.Join(errEvidencia, errConfirmacion) != nil {
		borrarCargaConfirmacionParcial(
			material, version, decision, contexto, prueba, aad, envuelto, nonce, cifrado,
			evidencia, confirmacion,
		)
		return cargaConfirmacionBorradorPostgreSQL{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return cargaConfirmacionBorradorPostgreSQL{
		Confirmacion: confirmacion, Prueba: prueba, Evidencia: evidencia,
		Decision: decision, Contexto: contexto, Material: material, Version: version, AAD: aad,
		MaterialEnvuelto: envuelto, Nonce: nonce, TextoCifrado: cifrado,
	}, nil
}

func borrarCargaConfirmacionParcial(valores ...[]byte) {
	borrarBytesDiarioPostgreSQL(valores...)
}

func proyectarVersionLigeraBorradorPostgreSQL(
	v dominiobolsa.VersionConvocatoriaGobernada,
	estado puertosbolsa.ReferenciaEstadoVersionConvocatoria,
) proyeccionLigeraBorradorPostgreSQL {
	actualizada := v.CreadaEn
	if v.Revision > 1 {
		actualizada = v.UltimaModificacionEn
	}
	return proyeccionLigeraBorradorPostgreSQL{
		ConvocatoriaID: v.ID, Secuencia: v.Secuencia, Referencia: estado.Referencia,
		Revision: v.Revision, HuellaEstadoSHA256: estado.HuellaEstadoSHA256,
		CodigoVersionPublica: v.CodigoVersionPublica,
		IdentificadorPublico: v.Contenido.IdentificadorPublico,
		Titulo:               v.Contenido.Titulo, Tipo: v.Contenido.Tipo,
		Categorias: append([]string(nil), v.Contenido.Categorias...), ExpedienteRef: v.ExpedienteRef,
		OrganizacionRef:  v.AmbitoOrganizativo.OrganizacionRef(),
		UnidadGestionRef: v.AmbitoOrganizativo.UnidadGestionRef(),
		NumeroPlazos:     len(v.Contenido.Plazos), NumeroRequisitos: len(v.Contenido.Requisitos),
		NumeroDocumentos: len(v.Contenido.Documentos), NumeroAyudas: len(v.Contenido.Ayuda),
		CreadaEn:      v.CreadaEn.UTC().Format(formatoInstanteMicrosegundo),
		ActualizadaEn: actualizada.UTC().Format(formatoInstanteMicrosegundo),
	}
}
