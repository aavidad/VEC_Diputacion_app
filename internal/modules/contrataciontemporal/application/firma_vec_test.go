package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorVecPrueba struct {
	certificadoCanal  string
	sinCanal          bool
	capacidadInvalida bool
	audiencia         string
	vistas            []ports.MaterialFirmaVec
}

func (d *autorizadorVecPrueba) AutorizarFirmaVec(_ context.Context, m ports.MaterialFirmaVec) (ports.CapacidadFirmaVec, error) {
	d.vistas = append(d.vistas, m)
	if d.sinCanal || d.certificadoCanal != m.CertificadoHuella {
		return ports.CapacidadFirmaVec{}, errors.New("canal sellado no acredita el certificado")
	}
	if d.capacidadInvalida {
		return ports.CapacidadFirmaVec{}, nil
	}
	r, err := RecursoFirmaVec(m)
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	contexto, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	hasta := instanteFirmaPrueba.Add(time.Minute)
	audiencia := ports.AudienciaFirmaVecV3
	if d.audiencia != "" {
		audiencia = d.audiencia
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision-firma-vec-001", strings.Repeat("a", 64), strings.Repeat("a", 64), "contexto-firma-vec-001", strings.Repeat("a", 64),
		ports.AccionRegistrarFirmaVec, r.Referencia, contexto, audiencia, hasta.Add(-5*time.Second), hasta,
	)
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte{7}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte("payload"), []byte("cose"), []byte("evidencia"), raiz,
	)
	if err != nil {
		return ports.CapacidadFirmaVec{}, err
	}
	return ports.TransportarMaterialFirmaVec(material), nil
}

type registroVecPrueba struct {
	firmas               []ports.FirmaRegistrada
	firmantesPrincipales []string
	registrado           []ports.MaterialFirmaVec
	consultas            []ports.MaterialConsultaFirmasR5
	lecturas             []ports.LecturaFirmasR5
	historiaRevision     uint64
	historiaHuella       string
	versionActual        uint64
}

func (d *registroVecPrueba) ConsultarFirmasAutorizadas(_ context.Context, m ports.MaterialConsultaFirmasR5, c ports.CapacidadConsultaFirmasR5) (ports.LecturaFirmasR5, error) {
	if err := ValidarCapacidadConsultaFirmasR5(c, m); err != nil {
		return ports.LecturaFirmasR5{}, err
	}
	d.consultas = append(d.consultas, m)
	firmas := make([]ports.FirmaRegistrada, 0, len(d.firmas))
	for _, firma := range d.firmas {
		if firma.Documento == m.Documento {
			firmas = append(firmas, firma)
		}
	}
	exacta := false
	for _, firma := range firmas {
		if firma.Documento == m.Documento && firma.ExpedienteVersion == m.VersionExpediente &&
			firma.ClaveIdempotencia == m.ClaveIdempotencia {
			firmas = []ports.FirmaRegistrada{firma}
			exacta = true
			break
		}
	}
	if !exacta && m.VersionExpediente < d.versionActual {
		firmas = nil
	}
	for i := range firmas {
		for j, registrada := range d.firmas {
			if registrada.ClaveIdempotencia == firmas[i].ClaveIdempotencia {
				firmas[i].CoincideFirmanteCandidato = j < len(d.firmantesPrincipales) &&
					d.firmantesPrincipales[j] == m.FirmantePrincipalCandidatoRef
				break
			}
		}
	}
	coincide, acreditada := separacionGlobalFirmasR5Prueba(d.firmas, d.firmantesPrincipales, m, exacta)
	lectura := ports.LecturaFirmasR5{Firmas: firmas, HistoriaRevision: d.historiaRevision, HistoriaHuella: d.historiaHuella,
		CoincideFirmanteEnOtroPaso: coincide, HistoriaSeparacionAcreditada: acreditada}
	d.lecturas = append(d.lecturas, lectura)
	return lectura, nil
}

func (d *registroVecPrueba) RegistrarFirmaVec(_ context.Context, m ports.MaterialFirmaVec, c ports.CapacidadFirmaVec) (ports.ReciboFirmaDocumento, error) {
	if err := ValidarCapacidadFirmaVec(c, m); err != nil {
		return ports.ReciboFirmaDocumento{}, err
	}
	h, _ := m.HuellaSHA256()
	for _, previo := range d.registrado {
		if previo.ClaveIdempotencia == m.ClaveIdempotencia {
			hPrevio, _ := previo.HuellaSHA256()
			if hPrevio != h {
				return ports.ReciboFirmaDocumento{}, ports.ErrClaveFirmaDocumentoUsada
			}
			return reciboFirmaVecPrueba(m, h, true), nil
		}
	}
	if m.HistoriaRevision != d.historiaRevision || m.HistoriaHuella != d.historiaHuella {
		return ports.ReciboFirmaDocumento{}, ports.ErrFirmaDocumentoEnConflicto
	}
	d.registrado = append(d.registrado, m)
	recibo := reciboFirmaVecPrueba(m, h, false)
	d.firmas = append(d.firmas, ports.FirmaRegistrada{Via: m.Via, FirmaRef: recibo.FirmaRef, ReciboRef: recibo.ReciboRef,
		Documento: m.Documento, Secuencia: m.Secuencia, ExpedienteVersion: m.VersionExpediente,
		CatalogoRef: m.CatalogoRef, CatalogoHuella: m.CatalogoHuella, PasoRef: m.PasoRef, PasoOrden: m.PasoOrden,
		Resultado: domain.ResultadoFirmaFirmado, OriginalHuella: m.OriginalHuella, FirmadoHuella: m.FirmadoHuella,
		OriginalRef: m.OriginalRef, OriginalVersion: m.OriginalVersion, FirmantePrincipalAcreditado: true,
		HistoriaRevision: m.HistoriaRevision, HistoriaHuella: m.HistoriaHuella,
		ClaveIdempotencia: m.ClaveIdempotencia, DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion})
	d.firmantesPrincipales = append(d.firmantesPrincipales, m.FirmantePrincipalRef)
	d.historiaRevision++
	d.historiaHuella = h
	return recibo, nil
}

func reciboFirmaVecPrueba(m ports.MaterialFirmaVec, huellaSolicitud string, repetido bool) ports.ReciboFirmaDocumento {
	return ports.ReciboFirmaDocumento{FirmaRef: "firma:vec:" + strconv.Itoa(m.Secuencia),
		ReciboRef: "recibo:firma:vec:" + strconv.Itoa(m.Secuencia), Secuencia: m.Secuencia,
		Resultado: domain.ResultadoFirmaFirmado, ExpedienteVersion: m.VersionExpediente, ActorRef: m.FirmantePrincipalRef,
		PerfilRef: m.PerfilFirmanteRef, RegistradaEn: instanteFirmaPrueba, SolicitudHuella: huellaSolicitud, YaRegistrada: repetido,
		DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion}
}

func servicioFirmaVecPrueba(t *testing.T) (*ServicioFirmaVec, *registroVecPrueba, *originalExternoPrueba, *competenciaExternaPrueba, *autorizadorVecPrueba, *custodioPrueba) {
	t.Helper()
	original := &originalExternoPrueba{contenido: []byte("%PDF-1.7 original vec sintetico")}
	base, err := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}, verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada})
	if err != nil {
		t.Fatal(err)
	}
	custodio := &custodioPrueba{}
	if err := base.ComponerOriginalAutorizado(original); err != nil {
		t.Fatal(err)
	}
	if err := base.ComponerCustodia(custodio, map[string]string{"informe_definitivo": tipoCustodiaPrueba}); err != nil {
		t.Fatal(err)
	}
	registro := &registroVecPrueba{historiaHuella: strings.Repeat("1", 64), versionActual: 7}
	consulta, competencia := &autorizadorConsultaFirmasR5Prueba{}, &competenciaExternaPrueba{}
	autorizador := &autorizadorVecPrueba{certificadoCanal: strings.Repeat("f", 64)}
	s, err := NuevoServicioFirmaVec(base, registro, consulta, autorizador, competencia)
	if err != nil {
		t.Fatal(err)
	}
	return s, registro, original, competencia, autorizador, custodio
}

func solicitudFirmaVecPrueba(original []byte, clave string) SolicitudFirmaVec {
	return SolicitudFirmaVec{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "informe_definitivo", PasoOrden: 1, OriginalRef: "ref:" + strings.Repeat("2", 64), OriginalVersion: 7,
		PDFFirmado: append(bytes.Clone(original), []byte(" firma-vec-pades-sintetica")...), ClaveIdempotencia: clave}
}

func TestFirmaVecRegistraR5ConCertificadoCanalSellado(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
	sol := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-000001")
	r, err := s.Firmar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	if len(registro.registrado) != 1 || len(competencia.vistas) != 1 || len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 ||
		len(registro.consultas) != 1 || registro.consultas[0].FirmantePrincipalCandidatoRef != r.Material.FirmantePrincipalRef ||
		r.Material.Via != ports.ViaFirmaCertificadoVEC || r.Material.CertificadoHuella != autorizador.certificadoCanal ||
		r.Recibo.ActorRef != r.Material.FirmantePrincipalRef || r.Custodiado.HuellaSHA256 != huella(sol.PDFFirmado) {
		t.Fatalf("registro R5 incompleto: %+v", r)
	}
}

func TestFirmaVecDeniegaConsultaAntesDeLeerHistoria(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
	consulta := s.consulta.(*autorizadorConsultaFirmasR5Prueba)
	consulta.err = errors.New("consulta denegada de prueba")

	_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-consulta-denegada"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(consulta.vistas) != 1 ||
		consulta.vistas[0].FirmantePrincipalCandidatoRef != "per_firmante_sintetico_001" || len(registro.consultas) != 0 ||
		len(registro.registrado) != 0 || len(competencia.vistas) != 1 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("la consulta denegada alcanzó historia o efectos: %v", err)
	}
}

func TestFirmaVecRechazaCapacidadDeConsultaLigadaAOtroCandidato(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
	consulta := s.consulta.(*autorizadorConsultaFirmasR5Prueba)
	consulta.alterar = func(m *ports.MaterialConsultaFirmasR5) {
		m.FirmantePrincipalCandidatoRef = "per_candidato_adulterado_999"
	}

	_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-candidato-adulterado"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(consulta.vistas) != 1 || len(registro.consultas) != 0 ||
		len(registro.registrado) != 0 || len(competencia.vistas) != 1 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("capacidad ligada a otro candidato alcanzó historia o efectos: %v", err)
	}
}

func TestFirmaVecDosPasosConservanOriginalYSeparanFirmantes(t *testing.T) {
	s, registro, original, competencia, autorizadorFirma, custodio := servicioFirmaVecPrueba(t)
	primera := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-paso-001")
	primera.PDFFirmado = append(bytes.Clone(original.contenido), []byte(" visto-bueno-pades-sintetico")...)
	r1, err := s.Firmar(context.Background(), primera)
	if err != nil {
		t.Fatal(err)
	}

	s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada, firmanteByte: 'd'}
	segunda := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-paso-002")
	segunda.PasoOrden = 2
	segunda.PDFFirmado = append(bytes.Clone(original.contenido), []byte(" resolucion-pades-sintetica")...)
	autorizador := s.autorizador.(*autorizadorVecPrueba)
	autorizador.certificadoCanal = strings.Repeat("d", 64)
	r2, err := s.Firmar(context.Background(), segunda)
	if err != nil {
		t.Fatal(err)
	}
	registro.historiaRevision++
	registro.historiaHuella = strings.Repeat("8", 64)
	registro.versionActual = 8
	cabezaActualRevision, cabezaActualHuella := registro.historiaRevision, registro.historiaHuella
	r2Repetido, err := s.Firmar(context.Background(), segunda)
	if err != nil {
		t.Fatal(err)
	}
	versionDistinta := segunda
	versionDistinta.VersionExpediente++
	_, err = s.Firmar(context.Background(), versionDistinta)
	if !errors.Is(err, ports.ErrClaveFirmaDocumentoUsada) {
		t.Fatalf("la clave histórica aceptó otra versión de expediente: %v", err)
	}
	claveNueva := versionDistinta
	claveNueva.ClaveIdempotencia = "clave-firma-vec-nueva-version"
	_, err = s.Firmar(context.Background(), claveNueva)
	if !errors.Is(err, ErrPasoFirmaNoPendiente) {
		t.Fatalf("una clave nueva reutilizó el paso completo: %v", err)
	}
	filasPasoDos := 0
	for _, firma := range registro.firmas {
		if firma.PasoOrden == 2 {
			filasPasoDos++
		}
	}

	if r1.Material.OriginalRef != r2.Material.OriginalRef ||
		r1.Material.OriginalVersion != r2.Material.OriginalVersion ||
		r1.Material.OriginalHuella != r2.Material.OriginalHuella ||
		r1.Material.FirmadoHuella == r2.Material.FirmadoHuella ||
		r1.Material.CertificadoHuella == r2.Material.CertificadoHuella ||
		r1.Material.FirmanteRef == r2.Material.FirmanteRef ||
		r1.Material.FirmantePrincipalRef == r2.Material.FirmantePrincipalRef ||
		r1.Material.CargoFirmante != "Técnico" || r2.Material.CargoFirmante != "Jefatura" ||
		!r2Repetido.Recibo.YaRegistrada || r2Repetido.Recibo.ReciboRef != r2.Recibo.ReciboRef ||
		r2Repetido.Recibo.SolicitudHuella != r2.Recibo.SolicitudHuella ||
		r2Repetido.Material.OriginalHuella != r2.Material.OriginalHuella ||
		r2Repetido.Material.HistoriaRevision != r2.Material.HistoriaRevision ||
		r2Repetido.Material.HistoriaHuella != r2.Material.HistoriaHuella ||
		registro.historiaRevision != cabezaActualRevision || registro.historiaHuella != cabezaActualHuella ||
		len(registro.registrado) != 2 || filasPasoDos != 1 || len(registro.consultas) != 5 ||
		registro.consultas[2].ClaveIdempotencia != segunda.ClaveIdempotencia ||
		registro.consultas[1].PasoOrden != 2 || registro.consultas[1].CatalogoHuella != r2.Material.CatalogoHuella ||
		registro.consultas[1].FirmantePrincipalCandidatoRef != r2.Material.FirmantePrincipalRef ||
		len(registro.lecturas[2].Firmas) != 1 || registro.lecturas[2].Firmas[0].Secuencia != 2 ||
		len(registro.lecturas[4].Firmas) != 2 || len(competencia.vistas) != 5 ||
		len(autorizadorFirma.vistas) != 3 || len(custodio.ordenes) != 3 {
		t.Fatalf("los dos pasos VEC no conservaron el original o no separaron firmantes: p1=%+v p2=%+v", r1.Material, r2.Material)
	}
	consulta := s.consulta.(*autorizadorConsultaFirmasR5Prueba)
	consulta.err = errors.New("concesión de lectura revocada")
	if _, err := s.Firmar(context.Background(), segunda); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) ||
		len(registro.consultas) != 5 || len(registro.registrado) != 2 || len(autorizadorFirma.vistas) != 3 || len(custodio.ordenes) != 3 {
		t.Fatalf("la revocación permitió leer o repetir el efecto: %v", err)
	}
}

func TestFirmaVecMismaPersonaEnPasosRequierePoliticaVersionada(t *testing.T) {
	for _, permite := range []bool{false, true} {
		t.Run(strconv.FormatBool(permite), func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
			if _, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-vec-separacion-p1")); err != nil {
				t.Fatal(err)
			}
			competencia.alterar = func(e *ports.EvidenciaCompetenciaFirmante) {
				e.FirmantePrincipalRef = "per_firmante_sintetico_001"
				e.ActoCompetenciaRef = "acto:competencia:jefatura:002"
			}
			s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoFirmaVerificada, firmanteByte: 'd'}
			autorizador.certificadoCanal = strings.Repeat("d", 64)
			segunda := solicitudFirmaVecPrueba(original.contenido, "clave-vec-separacion-p2")
			segunda.PasoOrden = 2
			if permite {
				if err := s.ComponerPoliticaMismaPersonaEnPasos(politicaMismaPersonaAplicacionPrueba{}); err != nil {
					t.Fatal(err)
				}
			}
			r, err := s.Firmar(context.Background(), segunda)
			if !permite {
				if !errors.Is(err, ports.ErrMismaPersonaEnOtroPasoR5) || len(registro.registrado) != 1 ||
					len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 ||
					!registro.lecturas[1].CoincideFirmanteEnOtroPaso || !registro.lecturas[1].HistoriaSeparacionAcreditada {
					t.Fatalf("misma persona pasó sin política: %v", err)
				}
				return
			}
			if err != nil || len(registro.registrado) != 2 || r.Material.FirmantePrincipalRef != "per_firmante_sintetico_001" ||
				r.Material.PerfilFirmanteRef != "perfil:ct:jefatura" || r.Material.CargoFirmante != "Jefatura" ||
				r.Material.ActoCompetenciaRef != "acto:competencia:jefatura:002" || len(autorizador.vistas) != 2 {
				t.Fatalf("política no conservó competencia del paso 2: %+v %v", r.Material, err)
			}
		})
	}
}

func TestFirmaVecLegadoEnOtroDocumentoNoAcreditaSeparacion(t *testing.T) {
	s, registro, original, _, autorizador, custodio := servicioFirmaVecPrueba(t)
	legada := firmaLegadaPasoUnoPrueba(original.contenido)
	legada.Documento = "resolucion"
	registro.firmas = append(registro.firmas, legada)
	registro.firmantesPrincipales = append(registro.firmantesPrincipales, "")
	_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-vec-legado-otro-documento"))
	if !errors.Is(err, ports.ErrMismaPersonaEnOtroPasoR5) || len(registro.registrado) != 0 ||
		len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 ||
		len(registro.lecturas[0].Firmas) != 0 || registro.lecturas[0].HistoriaSeparacionAcreditada {
		t.Fatalf("historia global legada acreditó separación: %v", err)
	}
}

func TestFirmaVecConsultaV3NoAceptaOtroPasoOCatalogo(t *testing.T) {
	for nombre, alterar := range map[string]func(*ports.MaterialConsultaFirmasR5){
		"paso":     func(m *ports.MaterialConsultaFirmasR5) { m.PasoOrden++ },
		"catalogo": func(m *ports.MaterialConsultaFirmasR5) { m.CatalogoHuella = strings.Repeat("9", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, _, autorizador, custodio := servicioFirmaVecPrueba(t)
			consulta := s.consulta.(*autorizadorConsultaFirmasR5Prueba)
			consulta.alterar = alterar
			_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-vec-consulta-otro-"+nombre))
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(registro.consultas) != 0 ||
				len(registro.registrado) != 0 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("capacidad V3 de otro %s alcanzó la historia o el efecto: %v", nombre, err)
			}
		})
	}
}

func TestFirmaVecAntecedenteLegadoNoAcreditaPasoR5(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
	registro.firmas = append(registro.firmas, firmaLegadaPasoUnoPrueba(original.contenido))
	sol := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-paso-legacy")
	sol.PasoOrden = 2

	_, err := s.Firmar(context.Background(), sol)
	if !errors.Is(err, ports.ErrAntecedenteFirmaR5NoAcreditado) || len(registro.registrado) != 0 ||
		len(competencia.vistas) != 1 || len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("la firma legada acreditó indebidamente el paso R5: %v", err)
	}
}

func TestFirmaVecPasoDosRechazaOtroOriginalAunqueConserveLosBytes(t *testing.T) {
	variantes := map[string]func(*SolicitudFirmaVec){
		"referencia": func(s *SolicitudFirmaVec) { s.OriginalRef = "ref:" + strings.Repeat("9", 64) },
		"version":    func(s *SolicitudFirmaVec) { s.OriginalVersion++ },
	}
	for nombre, variar := range variantes {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
			if _, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-original-p1")); err != nil {
				t.Fatal(err)
			}
			segunda := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-original-p2")
			segunda.PasoOrden = 2
			variar(&segunda)

			_, err := s.Firmar(context.Background(), segunda)
			if !errors.Is(err, ports.ErrAntecedenteFirmaR5NoAcreditado) || len(registro.registrado) != 1 ||
				len(competencia.vistas) != 2 || len(autorizador.vistas) != 1 || len(custodio.ordenes) != 1 {
				t.Fatalf("otro original con los mismos bytes heredó el antecedente R5: %v", err)
			}
		})
	}
}

func TestFirmaVecRechazaPDFODictamenNoAcreditadoSinEfectos(t *testing.T) {
	casos := map[string]func(*ServicioFirmaVec, *originalExternoPrueba, *SolicitudFirmaVec){
		"PDF desligado del original": func(_ *ServicioFirmaVec, _ *originalExternoPrueba, sol *SolicitudFirmaVec) {
			sol.PDFFirmado = []byte("%PDF-1.7 firmado sin el original autorizado")
		},
		"dictamen de integridad negativo": func(s *ServicioFirmaVec, _ *originalExternoPrueba, _ *SolicitudFirmaVec) {
			s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoIntegridadNoValida}
		},
		"firmante criptografico indeterminado": func(s *ServicioFirmaVec, _ *originalExternoPrueba, _ *SolicitudFirmaVec) {
			s.base.verificador = verificadorExternoPrueba{motivo: docports.MotivoFirmanteNoIdentificado}
		},
	}
	for nombre, preparar := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
			sol := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-dictamen-001")
			preparar(s, original, &sol)
			_, err := s.Firmar(context.Background(), sol)
			if !errors.Is(err, ErrFirmaNoVerificada) || len(registro.registrado) != 0 || len(competencia.vistas) != 0 ||
				len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("PDF o dictamen no acreditado produjo efectos: %v", err)
			}
		})
	}
}

func TestFirmaVecRechazaOriginalAjenoONoDisponible(t *testing.T) {
	casos := map[string]struct {
		preparar func(*originalExternoPrueba)
		err      error
	}{
		"fuente no disponible": {
			preparar: func(d *originalExternoPrueba) { d.err = errors.New("fuente sintetica no disponible") },
			err:      ports.ErrFuenteOriginalFirmaNoDisponible,
		},
		"referencia ajena": {
			preparar: func(d *originalExternoPrueba) {
				d.alterar = func(o *ports.OriginalFirmaAutorizado) { o.Solicitud.OriginalRef = "documento:ajeno" }
			},
			err: ports.ErrOriginalFirmaNoAutorizado,
		},
		"huella ajena": {
			preparar: func(d *originalExternoPrueba) {
				d.alterar = func(o *ports.OriginalFirmaAutorizado) { o.HuellaSHA256 = strings.Repeat("0", 64) }
			},
			err: ports.ErrOriginalFirmaNoAutorizado,
		},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
			caso.preparar(original)
			_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-original-001"))
			if !errors.Is(err, caso.err) || len(registro.registrado) != 0 || len(competencia.vistas) != 0 ||
				len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("original no autorizado produjo efectos: %v", err)
			}
		})
	}
}

func TestFirmaVecRechazaCompetenciaNoCoincidente(t *testing.T) {
	casos := map[string]func(*ports.EvidenciaCompetenciaFirmante){
		"cargo":     func(e *ports.EvidenciaCompetenciaFirmante) { e.CargoFirmante = "Otro cargo" },
		"perfil":    func(e *ports.EvidenciaCompetenciaFirmante) { e.PerfilFirmanteRef = "perfil:ajeno" },
		"solicitud": func(e *ports.EvidenciaCompetenciaFirmante) { e.Solicitud.PasoRef = "paso:ajeno" },
		"vigencia":  func(e *ports.EvidenciaCompetenciaFirmante) { e.Vigente = false },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
			competencia.alterar = alterar
			_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-competencia-001"))
			if !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) || len(registro.registrado) != 0 ||
				len(autorizador.vistas) != 0 || len(custodio.ordenes) != 0 {
				t.Fatalf("competencia no coincidente produjo efectos: %v", err)
			}
		})
	}
}

func TestFirmaVecRechazaCapacidadV3DeOtraAudienciaSinCustodiar(t *testing.T) {
	s, registro, original, _, autorizador, custodio := servicioFirmaVecPrueba(t)
	autorizador.audiencia = ports.AudienciaFirmaExternaV3
	_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-capacidad-001"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(autorizador.vistas) != 1 ||
		len(registro.registrado) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("capacidad V3 de otra audiencia produjo efectos: %v", err)
	}
}

func TestFirmaVecRechazaCertificadoDistintoAlCanalSellado(t *testing.T) {
	s, registro, original, _, autorizador, custodio := servicioFirmaVecPrueba(t)
	autorizador.certificadoCanal = strings.Repeat("d", 64)
	_, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-canal-001"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(autorizador.vistas) != 1 ||
		len(registro.registrado) != 0 || len(custodio.ordenes) != 0 {
		t.Fatalf("certificado ajeno al canal produjo efectos: %v", err)
	}
}

func TestFirmaVecReplayRecuperaMismoMaterialYRechazaConflicto(t *testing.T) {
	s, registro, original, competencia, autorizador, custodio := servicioFirmaVecPrueba(t)
	competencia.avanzarComprobacion = true
	sol := solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-replay-001")
	primero, err := s.Firmar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	registro.historiaRevision++
	registro.historiaHuella = strings.Repeat("9", 64)
	cabezaAvanzadaRevision, cabezaAvanzadaHuella := registro.historiaRevision, registro.historiaHuella
	repetido, err := s.Firmar(context.Background(), sol)
	hPrimero, errHuellaPrimero := primero.Material.HuellaSHA256()
	hRepetido, errHuellaRepetido := repetido.Material.HuellaSHA256()
	if err != nil || errHuellaPrimero != nil || errHuellaRepetido != nil || hPrimero != hRepetido ||
		!repetido.Recibo.YaRegistrada || repetido.Recibo.Secuencia != primero.Recibo.Secuencia ||
		len(registro.registrado) != 1 || len(custodio.ordenes) != 2 || len(competencia.competenciasVistas) != 2 ||
		competencia.competenciasVistas[0] != "2026-10-02T10:00:00Z" || competencia.competenciasVistas[1] != "2026-10-02T10:00:01Z" {
		t.Fatalf("replay identico no recuperado: %+v, %v", repetido, err)
	}
	if repetido.Material.HistoriaRevision != primero.Material.HistoriaRevision ||
		repetido.Material.HistoriaHuella != primero.Material.HistoriaHuella ||
		registro.historiaRevision != cabezaAvanzadaRevision || registro.historiaHuella != cabezaAvanzadaHuella {
		t.Fatalf("el replay no conservó la cabeza original: primero=%+v replay=%+v", primero.Material, repetido.Material)
	}
	conflicto := primero.Material
	conflicto.ClaveIdempotencia = "clave-firma-vec-cabeza-obsoleta"
	conflicto.DocumentoCustodiaRef = ports.DocumentoCustodiaRef(conflicto.OrganizacionRef, conflicto.ExpedienteRef, conflicto.ClaveIdempotencia)
	capacidadConflicto, err := autorizador.AutorizarFirmaVec(context.Background(), conflicto)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registro.RegistrarFirmaVec(context.Background(), conflicto, capacidadConflicto); !errors.Is(err, ports.ErrFirmaDocumentoEnConflicto) {
		t.Fatalf("una clave nueva con cabeza obsoleta no produjo conflicto: %v", err)
	}

	sol.PDFFirmado = append(bytes.Clone(sol.PDFFirmado), 'x')
	if _, err := s.Firmar(context.Background(), sol); !errors.Is(err, ports.ErrClaveFirmaDocumentoUsada) || len(registro.registrado) != 1 {
		t.Fatalf("replay con material distinto admitido: %v", err)
	}
}

func TestMaterialFirmaVecCanonicoLigaOriginalPDFYPasoSinDeclaracionPortafirmas(t *testing.T) {
	s, _, original, _, _, _ := servicioFirmaVecPrueba(t)
	r, err := s.Firmar(context.Background(), solicitudFirmaVecPrueba(original.contenido, "clave-firma-vec-canon-001"))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := r.Material.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	comprobarCanonFirmaR5Prueba(t, canon, 43, false)
	for _, fragmento := range [][]byte{
		[]byte(`"Via":"certificado_vec"`),
		[]byte(`"OriginalRef":"` + r.Material.OriginalRef + `"`),
		[]byte(`"OriginalVersion":7`),
	} {
		if !bytes.Contains(canon, fragmento) {
			t.Fatalf("el canon VEC no contiene %s: %s", fragmento, canon)
		}
	}
	for _, ausente := range [][]byte{[]byte("ReferenciaPortafirmasDeclarada"), []byte("FechaPortafirmasDeclarada")} {
		if bytes.Contains(canon, ausente) {
			t.Fatalf("el canon VEC contiene una declaración de Portafirmas: %s", canon)
		}
	}

	base, err := r.Material.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	variantes := map[string]func(*ports.MaterialFirmaVec){
		"referencia original": func(m *ports.MaterialFirmaVec) { m.OriginalRef = "ref:" + strings.Repeat("3", 64) },
		"version original":    func(m *ports.MaterialFirmaVec) { m.OriginalVersion++ },
		"PDF firmado":         func(m *ports.MaterialFirmaVec) { m.FirmadoHuella = strings.Repeat("1", 64) },
		"paso":                func(m *ports.MaterialFirmaVec) { m.PasoOrden++ },
	}
	for nombre, variar := range variantes {
		t.Run(nombre, func(t *testing.T) {
			m := r.Material
			variar(&m)
			h, err := m.HuellaSHA256()
			if err != nil || h == base {
				t.Fatalf("la variación no quedó ligada a la huella: %s, %v", h, err)
			}
		})
	}
}
